package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"path/filepath"

	"sort"
	"strings"
	"sync"

	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// WorkspaceRoots 工作区所有根目录（多根工作区支持）。
// 由 bridge.go 在初始化 agent 时设置。resolvePath 会检查路径是否在任一根目录内。
// ★ 2026-09-09：生产读取点一律改用 workspaceRootsSnapshot()（实时合并
// core.Folders）——本变量保留为测试注入点与兼容赋值点。
var WorkspaceRoots []string

// workspaceRootsSnapshot 返回当前工作区根实时快照：WorkspaceRoots 变量与
// core.Folders（设置持久层，运行中添加项目即时更新）合并去重，前者优先保序。
// ★ 修复加固：此前所有根归属判定读包级变量 WorkspaceRoots，其值仅在启动/
// 构建 LoopOpts/OnSyncWorkspace 钩子时刷新——HTTP 请求（文件树 /api/fs/list、
// ctx.fs 工具）在钩子同步前到达时，新添加的项目根不在列表内 → resolvePath
// 判定越界 → 文件树展开为空。改读本快照消除时序隐患（core.Folders 是
// add/remove 端点直接改写的真相源，agent→core 单向依赖已存在）。
func workspaceRootsSnapshot() []string {
	roots := make([]string, 0, len(WorkspaceRoots)+len(core.Folders)+1)
	seen := map[string]bool{}
	for _, r := range WorkspaceRoots {
		if r != "" && !seen[r] {
			seen[r] = true
			roots = append(roots, r)
		}
	}
	for _, r := range core.Folders {
		if r != "" && !seen[r] {
			seen[r] = true
			roots = append(roots, r)
		}
	}
	return roots
}

// FileChangeCallback 文件变更回调（可选）。每次写类工具成功修改文件后调用，供外部追踪变更。
// filePath 为工作区相对路径。由 orchestration.go 或外部宿主设置。
var FileChangeCallback func(filePath string)

// ErrRetry 由 OnToolError 钩子返回，指示 Execute 用修改后的 args 重新执行 handler。
// 用于可恢复错误（如 edit 匹配失败→自动降级行号定位重试）。
// 注意：OnToolError 应通过 args（引用）设置重试参数。
var ErrRetry = errors.New("__retry__")

// ToolHandler 工具执行体：收到已解析的 JSON 参数，返回结果文本或 error。
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool 一个已注册工具（名/描述/参数 Schema/执行体 + 元信息）。
// Tool 一个已注册工具（名/描述/参数 Schema/执行体 + 元信息）。
type Tool struct {
	Name             string
	Description      string         // 简短描述（传给 LLM function-calling）
	UsageGuide       string         // ★ 详细使用指导：何时用此工具、注意事项、对比 bash 的优势
	Category         string         // ★ 工具分类：如 "code-search", "git", "file", "web", "debug", "build", "test"
	Parameters       map[string]any // JSON Schema
	Handler          ToolHandler
	SystemTool       bool // ★ 系统内部工具（如历史搜索、委托任务等），不暴露给 LLM agent 选择，前端 UI 隐藏
	ReadOnly         bool // 只读（不改文件系统）——供并行/免审
	RequiresApproval bool // 写类工具：需人工确认（UI 接入时用）
	// DynamicApproval 动态审批判定（可选）：返回 true 时本次调用需走审批门
	// （与 RequiresApproval 为「或」关系；approveFn 为 nil 的放行路径不触发）。
	// 用于按参数/状态动态决定是否拦截，如插件 client 半激活首次装载需人工批准。
	DynamicApproval func(tc ToolCall) bool
	Enabled         bool // ★ 是否启用（默认 true；按工作区配置可关闭）
}

// Registry 工具注册表（并发安全）。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
	order []string // 保持注册顺序，传给 LLM 时稳定

	// 钩子（均可空）：
	//   BeforeTool：执行前调用；返回 proceed=false 则短路——用 override/overrideErr 作结果，不执行 handler。
	//               用途：审批拒绝、缓存命中、参数校验拦截。
	//   AfterTool：执行后调用（无论成败，err 非 nil 表示出错）。不可改结果，仅观察。
	//               用途：统计、日志、耗时监控。
	//   OnToolError：执行出错时调用（AfterTool 之后）。返回 (result, replacedErr) 替换原结果/错误；
	//               返回 ("", nil) 可吞掉错误转为成功（避免连续失败止损误触）。
	//               用途：错误诊断增强、可恢复错误降级。
	BeforeTool  func(ctx context.Context, name string, args map[string]any) (proceed bool, override string, overrideErr error)
	AfterTool   func(ctx context.Context, name string, args map[string]any, result string, err error, duration time.Duration)
	OnToolError func(ctx context.Context, name string, args map[string]any, err error) (result string, replacedErr error)

	// OnToolUpdate 工具执行期间流式更新回调（可选）。工具 handler 在执行过程中调用此钩子
	// 推送中间结果（如 bash 的逐行输出、read 的翻页进度）。
	// callID 为工具调用 ID（空串表示非工具调用场景），partialResult 为当前累积的中间文本。
	// 此钩子不替代最终结果，仅用于流式展示。handler 的返回值仍是正式结果。
	OnToolUpdate func(name string, callID string, partialResult string)
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: map[string]*Tool{}}
}

// Register 注册一个工具（同名覆盖，顺序不变）。
// ★ 同名覆盖时保留原启用状态：重复注册（如 RegisterDefaultTools 与
//
//	RegisterToolGroups 对同一批内置工具双注册）不得清除 harness 过滤/
//	工具集禁用状态（Enabled=false 保持 false）。首次注册默认启用。
func (r *Registry) Register(t *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, exists := r.tools[t.Name]; exists {
		t.Enabled = old.Enabled // 覆盖：保留原启用状态
	} else {
		if !t.Enabled { // 未显式设置则默认启用
			t.Enabled = true
		}
		r.order = append(r.order, t.Name)
	}
	r.tools[t.Name] = t
}

// Get 取工具。
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Unregister 卸载工具（插件热重载用）。不存在则无操作。
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; !ok {
		return
	}
	delete(r.tools, name)
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}

// SetToolEnabled 启用或禁用指定工具（按工作区配置调用）。
func (r *Registry) SetToolEnabled(name string, enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.tools[name]; ok {
		t.Enabled = enabled
	}
}

// IsEnabled 返回工具是否已启用。
func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.tools[name]; ok {
		return t.Enabled
	}
	return false
}

// EnabledDefinitions 导出已启用工具的定义（按注册顺序），传给 LLM 作 function-calling。
// 只包含 Enabled=true 的工具。禁用工具不暴露给 LLM。
// EnabledDefinitions 导出已启用工具的定义。
// 已委托给 Definitions()，两者行为一致（Definitions 已默认过滤禁用工具）。
func (r *Registry) EnabledDefinitions() []ToolDefinition {
	return r.Definitions()
}

// Names 全部已注册工具名（注册顺序；含禁用工具）。
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.order...)
}

// EnabledNames 已启用工具名（注册顺序；禁用工具不返回）。
// 供 run_code 沙箱注入等场景——禁用工具不得暴露（防绕过过滤）。
func (r *Registry) EnabledNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.order))
	for _, name := range r.order {
		if t := r.tools[name]; t != nil && t.Enabled {
			out = append(out, name)
		}
	}
	return out
}

// UsageGuideText 生成工具使用指南文本（供注入系统提示使用）。
// ★ 工具描述已取消（2026-08-17）：提示词中不再注入工具使用指南——
//
//	工具的名称、用途与参数完全由 tools 参数 schema 提供（随 function-calling
//	下发，天然与注册表一致，不会与提示词文本脱节）。故恒返回空串；
//	保留函数签名与调用点，便于将来按需恢复。
func (r *Registry) UsageGuideText() string {
	return ""
}

// Copy 深拷贝 Registry（含钩子引用）。子 Loop 用副本注册工具，避免污染父表。
func (r *Registry) Copy() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := &Registry{
		tools:        map[string]*Tool{},
		order:        append([]string(nil), r.order...),
		BeforeTool:   r.BeforeTool,
		AfterTool:    r.AfterTool,
		OnToolError:  r.OnToolError,
		OnToolUpdate: r.OnToolUpdate,
	}
	for n, t := range r.tools {
		out.tools[n] = t
	}
	return out
}

// Subset 按工具名白名单过滤返回新 Registry（含钩子）。
// 用于子 agent 的 Tools 白名单裁剪：names 非空时只保留白名单内工具。
// 调用方应先判断白名单是否为空（空=继承父全部，直接用父 Registry 或 Copy）。
func (r *Registry) Subset(names []string) *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := &Registry{
		tools:        map[string]*Tool{},
		BeforeTool:   r.BeforeTool,
		AfterTool:    r.AfterTool,
		OnToolError:  r.OnToolError,
		OnToolUpdate: r.OnToolUpdate,
	}
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	for _, n := range r.order {
		if set[n] {
			out.tools[n] = r.tools[n]
			out.order = append(out.order, n)
		}
	}
	return out
}

// ToolMeta 工具的完整元信息（供前端 UI 展示）。
type ToolMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	UsageGuide  string `json:"usageGuide"`
	Enabled     bool   `json:"enabled"`
	ReadOnly    bool   `json:"readOnly"`
	SystemTool  bool   `json:"systemTool"`
}

// AllToolMeta 返回所有工具的元信息列表（含禁用工具），供前端 UI 展示工具开关列表。
func (r *Registry) AllToolMeta() []ToolMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metas := make([]ToolMeta, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		metas = append(metas, ToolMeta{
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
			UsageGuide:  t.UsageGuide,
			Enabled:     t.Enabled,
			ReadOnly:    t.ReadOnly,
			SystemTool:  t.SystemTool,
		})
	}
	return metas
}

// Definitions 导出已启用工具的定义，传给 LLM 作 function-calling。
// 只包含 Enabled=true 的工具。禁用工具不暴露给 LLM。
// ★ 对齐 harness system-prompt orderTools（2026-08-17）：按 name 字典序
//
//	（code-unit，locale 无关）排序后返回——注册顺序依赖装配时序（内置组、
//	磁盘插件、工具集、MCP 的加载顺序），直接下发会随时序漂移导致 tools
//	JSON 逐字节变化，从 tools 处切断 KV 缓存前缀（其后全部历史 miss）。
//	字典序保证跨机器/跨装配时序稳定，缓存前缀可复用。
func (r *Registry) Definitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if !t.Enabled {
			continue // 禁用的工具不暴露给 LLM
		}
		desc := trimToolDesc(t.Description)
		defs = append(defs, ToolDefinition{
			Type:     "function",
			Function: FunctionDefinition{Name: t.Name, Description: desc, Parameters: t.Parameters},
		})
	}
	// ★ 字典序排序（对齐 harness orderTools 的 compareToolNames）：
	//   code-unit 比较，locale 无关，任意机器/任意装配顺序结果一致。
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Function.Name < defs[j].Function.Name
	})
	return defs
}

// trimToolDesc 工具描述截断（rune 安全）。
// 截断目标 ~120 字符；优先在中文句号「。」处断，其次在空格/逗号处断。
// ★ 修复（2026-08-17）：原实现按字节 len(desc[:120]) 截断，多字节 UTF-8
//
//	描述（中文）会切在字符中间产生无效字节，strings.LastIndex 在无效字节
//	上找不到「。」返回 -1，随后 desc[100] 字节级扫描进一步产生乱码截断点
//	（json.Marshal 会把无效序列转义为 \ufffd，工具描述乱码）。
//	现改为 []rune 截断，始终落在字符边界。
func trimToolDesc(desc string) string {
	runes := []rune(desc)
	if len(runes) <= 120 {
		return desc
	}
	head := runes[:120]
	// 在 rune 切片中找最后一个「。」（rune 索引；勿用 strings.LastIndex——
	// 它返回字节索引，与 head[:cut] 的 rune 索引混用会切错位置）
	cut := -1
	for i, r := range head {
		if r == '。' {
			cut = i
		}
	}
	if cut < 60 {
		// 无中文句号（或太靠前）：退到 100 字符内找空格/逗号
		cut = 100
		if cut >= len(head) {
			cut = len(head) - 1
		}
		for cut > 60 && head[cut] != ' ' && head[cut] != ',' {
			cut--
		}
	}
	return strings.TrimSpace(string(head[:cut]))
}

// Execute 解析 JSON 参数并执行工具。参数 JSON 由 LLM 流式拼接而来，可能为空。
// 依次触发 BeforeTool → handler → AfterTool → OnToolError（仅出错时）钩子。
func (r *Registry) Execute(ctx context.Context, name, argsJSON string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("未知工具: %s（当前工具集未包含该工具。可能原因：未加入工作区工具集 / 插件未装载 / 名称拼写错误。可用 tools.list() 查看当前可用工具；需要启用可用 toolset_edit add_builtin）", name)
	}
	if !t.Enabled {
		// 禁用工具：对齐原「未注册」的不可见语义——agent 不应调用（Definitions 已过滤），
		// 但模型幻觉/手动调用仍可能到达这里，明确报错防绕过。
		return "", fmt.Errorf("工具 %s 已禁用（未加入工作区工具集；可用 toolset_edit add_builtin 或工具集面板启用）", name)
	}
	args := map[string]any{}
	if s := strings.TrimSpace(argsJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			return "", fmt.Errorf("参数 JSON 解析失败: %w（原文 %q）", err, argsJSON)
		}
	}
	// ★ 钩子系统（t1 L2 闭环）：PreToolUse 门——配置钩子（.pair/settings.json /
	//   ~/.pair/settings.json，exit 2/超时拦截）+ 插件注册钩子（ctx.hooks.register）。
	//   拦截时把反馈作为错误回灌 LLM（对齐审批驳回的可见性）。无钩子配置时零开销。
	if blocked, feedback := firePreToolUseHooks(ctx, name, argsJSON); blocked {
		msg := strings.TrimSpace(feedback)
		if msg == "" {
			msg = "PreToolUse 钩子拦截了此工具调用"
		}
		return "", fmt.Errorf("钩子拦截: %s", msg)
	}
	// BeforeTool 钩子：可短路（审批拒绝/缓存命中/校验拦截）
	if r.BeforeTool != nil {
		proceed, override, overrideErr := r.BeforeTool(ctx, name, args)
		if !proceed {
			return override, overrideErr
		}
	}
	// 流式更新回调注入 Context
	if r.OnToolUpdate != nil {
		ctx = WithStreamCallback(ctx, func(name, callID, partial string) {
			r.OnToolUpdate(name, callID, partial)
		})
	}
	// ★ 工具前通知（观察型，非 外部中间件）：agent/pre-tool——工具即将执行
	//   时发出；args 截断防协议风暴，无插件订阅时零开销（emitBridgeEvent
	//   白名单过滤）。注意与 DSH agent/pre-step（LLM 调用前中间件瀑布，
	//   见 dsh_prestep.go）语义区分：pre-tool 是单向通知，pre-step 是
	//   决策回传（可改写输入/拒绝 turn）。
	evArgs := argsJSON
	if len(evArgs) > 2048 {
		evArgs = evArgs[:2048] + "…"
	}
	emitBridgeEvent("agent/pre-tool", map[string]any{"tool": name, "args": evArgs})
	start := time.Now()
	result, err := t.Handler(ctx, args)
	dur := time.Since(start)
	// ★ 钩子系统（t1 L2 闭环）：PostToolUse 观察（配置钩子 + 插件钩子；不拦截）。
	//   在 AfterTool 之前触发（内部钩子语义：工具结果产出后立即通知）。
	firePostToolUseHooks(ctx, name, argsJSON, result, err)
	// AfterTool 钩子：观察（统计/日志），不改结果
	if r.AfterTool != nil {
		r.AfterTool(ctx, name, args, result, err, dur)
	}
	// OnToolError 钩子：错误诊断增强 / 可恢复错误降级
	// 返回 ErrRetry 时用修改后的 args 重试 handler（最多 1 次防无限循环）
	if err != nil && r.OnToolError != nil {
		newResult, newErr := r.OnToolError(ctx, name, args, err)
		if errors.Is(newErr, ErrRetry) {
			// 重试：handler 用 OnToolError 修改过的 args
			result, err = t.Handler(ctx, args)
			if r.AfterTool != nil {
				r.AfterTool(ctx, name, args, result, err, time.Since(start))
			}
		} else {
			result, err = newResult, newErr
		}
	}
	return result, err
}

// ─── 核心工具集 ──────────────────────────────────────────────

// RegisterDefaultTools 注册全部内置工具组（独立宿主/测试/示例用；由内置插件
// 规格统一分发，见 builtin_plugins.go）。★ 宿主进程不调用——改用
// RegisterHostFrameworkTools（工具实现已迁移磁盘插件 .pair/plugins/tool-*）。
//
// ★ 2026-09 工具重构 Phase B：编辑面统一为 apply_patch（codex 语法自由格式
// 补丁）——edit/multi_edit（JSON old_string 匹配+行号追踪）与 move_file/
// delete_file 已移除（Add/Update/Delete/Move to 全覆盖）；read/write 保留。
func registerCoreTools(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "read",
		UsageGuide:  "读取文件内容，限工作区内路径。path 相对「主项目根」解析——多项目工作区访问其他项目请传 project（项目目录名）或绝对路径，勿反复重试相对路径。大文件用 offset+limit 分页读取，避免撑爆上下文。二进制文件会自动拒绝读取，请改用 inspect_binary。比 os.ReadFile 更安全（路径越界拦截+二进制保护）。",
		Description: "读取文件内容。path 为工作区内路径（相对主项目根；跨项目传 project 或绝对路径）。可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示用 offset/limit 翻页)。",
		Parameters:  objSchema(props{"path": strProp("文件路径（工作区内；相对主项目根，跨项目用 project 参数或绝对路径）"), "offset": intProp("可选：起始行号(1 基)"), "limit": intProp("可选：读取行数"), "project": projectSchemaProp()}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePathFor(root, args, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			// 二进制保护：含 NULL 字节视为二进制，拒绝读取并引导 inspect_binary（避免把字节流灌进上下文）
			if strings.IndexByte(string(data), 0) >= 0 {
				return "", fmt.Errorf("「%s」是二进制文件，read 不支持读取二进制内容；请用 inspect_binary 工具查看（hexdump/类型嗅探）", argStr(args, "path"))
			}
			offset, limit := argInt(args, "offset", 0), argInt(args, "limit", 0)
			if offset <= 0 && limit <= 0 { // 全文（超 2000 行截断，提示翻页）
				lines := strings.Split(string(data), "\n")
				if len(lines) > 2000 {
					return strings.Join(lines[:2000], "\n") + fmt.Sprintf("\n…[文件共 %d 行，仅显示前 2000；用 offset/limit 读其余]", len(lines)), nil
				}
				return string(data), nil
			}
			lines := strings.Split(string(data), "\n") // 片段
			start := offset - 1
			if start < 0 {
				start = 0
			}
			if start >= len(lines) {
				return "", fmt.Errorf("offset %d 超出文件行数 %d", offset, len(lines))
			}
			end := len(lines)
			if limit > 0 && start+limit < end {
				end = start + limit
			}
			return strings.Join(lines[start:end], "\n"), nil
		},
	})

	r.Register(&Tool{
		Name:             "write",
		UsageGuide:       "写入文件，父目录自动创建。需审核批准。path 相对「主项目根」解析——多项目工作区写其他项目文件请传 project（项目目录名）或绝对路径。比 os.WriteFile 更安全（自动快照+路径越界拦截+变更回调）。如需追加内容请先用 read 读入再加上新内容后 write 覆盖。",
		Description:      "把 content 完整写入 path（覆盖；父目录自动创建）。path 相对主项目根（跨项目传 project 或绝对路径）。",
		Parameters:       objSchema(props{"path": strProp("文件路径（相对主项目根，跨项目用 project 参数或绝对路径）"), "content": strProp("完整文件内容"), "project": projectSchemaProp()}, "path", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePathFor(root, args, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			SnapshotBeforeWriteWithTracking(root, p) // 修改前自动快照（关联到当前消息）
			content := argStr(args, "content")
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
			if FileChangeCallback != nil {
				FileChangeCallback(argStr(args, "path"))
			}
			return fmt.Sprintf("已写入 %s（%d 字节）", argStr(args, "path"), len(content)), nil
		},
	})

	r.Register(&Tool{
		Name:       "apply_patch",
		UsageGuide: "应用 codex 语法自由格式补丁修改文件（*** Begin Patch / Add File / Update File / Delete File / *** End Patch）。免 JSON 转义、免唯一性/行号依赖——Update 用上下文行定位；编辑文件的主工具（取代 edit/multi_edit）。",
		Description: "应用补丁修改文件：一次调用可含多个文件、四类操作（新增/更新/删除/移动）。" +
			"@@ 段内：空格前缀=上下文行、- 前缀=删除行、+ 前缀=新增行；写前自动快照+变更回调。",
		Parameters:       objSchema(props{"patch": strProp("补丁文本（*** Begin Patch … *** End Patch）")}, "patch"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return ApplyPatchText(root, argStr(args, "patch"))
		},
	})

	r.Register(&Tool{
		Name: "glob",
		// ★ Round3：原 list_files（目录列举）+ search_files（递归查找）合并为 glob 基座
		//   （Go 侧仅测试/归档基座；生产语义以 tool-harness JS 插件为准）。
		//   有 pattern → 递归查找（searchFilesHandler）；仅 path → 目录列举（listFilesHandler）。
		UsageGuide:  "列出工作区目录下的文件和子目录（目录排前），或按通配符递归查找文件。path 相对「主项目根」解析——多项目工作区搜其他项目请传 project（项目目录名）或绝对路径。比 bash dir /s 更高效（自动跳过依赖库/VCS/IDE 运行数据目录、结果结构化排序）。提供 pattern（如 *.go）时递归查找；仅给 path（或空参数）时列目录。",
		Description: "按通配符递归查找文件（pattern 含 / 或 ** 按路径模式，如 internal/**/*.go），返回相对路径列表；pattern 省略则列出 path（省略=主项目根）下的文件/子目录（目录在前，pattern 可选过滤如 *.go）。path 相对主项目根（跨项目传 project 或绝对路径）。自动跳过依赖库（node_modules/vendor/.venv…）、构建产物（dist/build/out/target…）、VCS（.git…）与项目根下的 IDE 运行数据目录（_temp/logs/bin/release/screenshots…）。",
		Parameters: objSchema(props{
			"path":        strProp("目录路径（省略=主项目根；列举模式）或限定子目录（查找模式）；相对主项目根，跨项目用 project 参数或绝对路径"),
			"pattern":     strProp("可选通配符：提供则递归查找文件（如 *.go、internal/**/*.go），省略则列目录"),
			"language":    strProp("可选：查找模式按语言过滤，如 \"go\"、\"typescript\""),
			"max_results": intProp("可选：查找模式结果上限（默认 500）"),
			"project":     projectSchemaProp(),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			if strings.TrimSpace(argStr(args, "pattern")) != "" {
				return searchFilesHandler(root)(ctx, args) // 递归查找
			}
			return listFilesHandler(root)(ctx, args) // 目录列举
		},
	})

	// grep：工作区内正则全文搜索（原 search_content，并入 core 组；生产语义以
	// tool-harness JS 插件为准，Go 侧仅测试/归档基座）
	r.Register(&Tool{
		Name: "grep",
		Description: "在工作区内按正则搜索文件内容，返回匹配的「相对路径:行号: 行文本」。" +
			"pattern 为 RE2 正则；path 限定子目录（省略=主项目根，相对主项目根解析，跨项目传 project 或绝对路径）；glob 按文件名过滤（如 *.go）；" +
			"case_insensitive 忽略大小写；max_results 上限（默认 200）。自动跳过依赖库（node_modules/vendor/.venv…）、构建产物（dist/build/out/target…）、VCS（.git…）、项目根下的 IDE 运行数据目录（_temp/logs/bin/release/screenshots…）与二进制/超大文件。",
		UsageGuide: "搜索文件内容（全文搜索）。path 相对「主项目根」解析——多项目工作区搜其他项目请传 project（项目目录名）或绝对路径。比 bash findstr/grep 更精确（自动跳过依赖库/VCS/IDE 运行数据目录、自动处理编码、结果结构化）。搜索函数/类型定义请优先用 codegraph_search（基于 AST，更精确）。",
		Category:   "代码搜索",
		Parameters: objSchema(props{
			"pattern":          strProp("RE2 正则表达式"),
			"path":             strProp("限定子目录（省略=主项目根；相对主项目根，跨项目用 project 参数或绝对路径）"),
			"glob":             strProp("文件名通配过滤，如 *.go"),
			"case_insensitive": boolProp("忽略大小写"),
			"max_results":      intProp("结果行数上限（默认 200）"),
			"project":          projectSchemaProp(),
		}, "pattern"),
		ReadOnly: true,
		Handler:  searchContentHandler(root),
	})

	registerCodeGraphTools(r, root)      // codegraph_build / codegraph_search / codegraph_relations / ...（代码知识图谱，见 codegraph_tools.go + pkg/codegraph）
	registerExtraCodeGraphTools(r, root) // codegraph_find_by_signature / codegraph_explore（额外工具，见 codegraph_extra.go）

}

// ─── 流式更新支持 ──────────────────────────────────────────────

// streamUpdateKey 上下文键类型，避免 key 冲突。
type streamUpdateKey struct{}

// WithStreamCallback 在 context 中注入流式更新回调，供工具 handler 调用。
func WithStreamCallback(ctx context.Context, fn func(name, callID, partial string)) context.Context {
	return context.WithValue(ctx, streamUpdateKey{}, fn)
}

// StreamUpdate 从 context 中取出流式更新回调并调用（若存在）。
// 工具 handler 在执行过程中调用此函数推送中间结果，不影响最终返回值。
func StreamUpdate(ctx context.Context, name, callID, partial string) {
	if fn, ok := ctx.Value(streamUpdateKey{}).(func(name, callID, partial string)); ok {
		fn(name, callID, partial)
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

type props = map[string]any

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

// objSchema 拼 object 类型的 JSON Schema。

// objSchema 拼 object 类型的 JSON Schema。
func objSchema(properties props, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

func argStr(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func argBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// argInt 取整型参数（JSON 数字 unmarshal 为 float64）；缺省/非数字返回 def。
func argInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return def
}

// argStrSlice 取字符串数组参数（JSON 数组 unmarshal 为 []any）；非数组返回 nil。
func argStrSlice(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// orderedRoots 返回工作区所有根目录，primaryRoot 排最后（供"后加载覆盖"场景使用：
// 同名工具/配置以 primary 项目为准）。
func orderedRoots(primaryRoot string) []string {
	allRoots := workspaceRootsSnapshot()
	roots := make([]string, 0, len(allRoots)+1)
	seen := map[string]bool{primaryRoot: true}
	for _, wr := range allRoots {
		if wr != primaryRoot && !seen[wr] {
			seen[wr] = true
			roots = append(roots, wr)
		}
	}
	roots = append(roots, primaryRoot)
	return roots
}

// resolvePath 把相对/绝对路径解析为工作区内的绝对路径，越界则报错（安全底线）。
// 先检查路径是否在 primary root 下；若不在，再查是否在 WorkspaceRoots（工作区其他根目录）下。
func resolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	full := p
	// ★ 相对路径一律相对于 primary root 解析（读取/写入落点在 root 内），
	//   避免落到进程 cwd（测试/多目录运行时 cwd ≠ 工作区根，会写错位置）。
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	full = filepath.Clean(full)

	// 归属检查：先查 primary root，再查其他工作区根（多根工作区支持）。
	// 路径语法上属于某根 → 确认真实存在于此（避免 `goui/main.go` 被拼成
	// `gou-ide/goui/main.go` 但实际应在 `../goui/`）。
	// ★ 2026-09-09：读 workspaceRootsSnapshot()（实时合并 core.Folders），
	//   运行中添加的项目即刻可解析（此前读包级变量存在同步窗口期）。
	allRoots := workspaceRootsSnapshot()
	roots := make([]string, 0, len(allRoots)+1)
	seen := map[string]bool{root: true}
	roots = append(roots, root)
	for _, wr := range allRoots {
		if !seen[wr] {
			seen[wr] = true
			roots = append(roots, wr)
		}
	}
	for _, r := range roots {
		rel, err := filepath.Rel(r, full)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if pathExists(full) || parentDirExists(r, full) {
				return full, nil
			}
		}
	}

	// 兜底：文件尚未在任何根下创建 → 默认用 primary root（新建文件归宿）。
	// ★ 但必须确认 full 仍在 primary root 内（越界路径一律拒绝）。
	if rel, err := filepath.Rel(root, full); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return full, nil
	}
	// ★ 诊断增强（2026-09）：相对路径在其他工作区项目下存在（同工作区跨项目场景）→
	//   直接给出候选路径与 project 用法，而不是笼统「超出工作区范围」。
	if cands := otherRootCandidates(root, p); len(cands) > 0 {
		parts := make([]string, 0, len(cands))
		for _, c := range cands {
			parts = append(parts, fmt.Sprintf("%s（project=%q）", c.path, filepath.Base(c.root)))
		}
		return "", fmt.Errorf("路径 %q 不在当前项目（%s）内；同工作区其他项目下存在同名路径：%s。跨项目请用 project 参数（如 project=%q）或直接传绝对路径。",
			p, filepath.Base(root), strings.Join(parts, "、"), filepath.Base(cands[0].root))
	}
	return "", fmt.Errorf("路径 %q 超出工作区范围（root: %s）", p, root)
}

// rootPathCandidate 跨项目寻址候选（诊断提示用）。
type rootPathCandidate struct {
	root string // 命中路径所属的工作区项目根
	path string // 该根下真实存在的完整路径
}

// otherRootCandidates 扫其他工作区根，收集「同为该相对路径且真实存在」的候选。
// 仅用于路径解析失败时的诊断提示（引导 LLM 改用 project 参数或绝对路径），
// 不参与任何自动回退解析（避免跨项目误读/误写）。
func otherRootCandidates(root, p string) []rootPathCandidate {
	if filepath.IsAbs(p) {
		return nil
	}
	var out []rootPathCandidate
	for _, r := range workspaceRootsSnapshot() {
		if r == "" || samePath(r, root) {
			continue
		}
		cand := filepath.Join(r, p)
		if pathExists(cand) {
			out = append(out, rootPathCandidate{root: r, path: cand})
		}
	}
	return out
}

// capOutput 截断过长输出（保头 3/4 + 尾 1/4），防工具结果撑爆上下文。
func capOutput(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	head := limit * 3 / 4
	tail := limit - head
	return s[:head] + "\n...[输出截断 " + fmt.Sprint(len(s)-limit) + " 字节]...\n" + s[len(s)-tail:]
}

// pathExists 检查文件/目录是否存在。
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// dirExists 检查目录是否存在。
func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// parentDirExists 检查相对于根目录的父级目录是否存在（新建文件时确认目标目录归宿）。
// 例如 root=F:/syproject/gou-ide, full=F:/syproject/gou-ide/goui/new.go
// → 检查 F:/syproject/gou-ide/goui/ 是否存在
func parentDirExists(root, full string) bool {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return false
	}
	parent := filepath.Dir(rel)
	if parent == "." {
		return true // 直接在根目录创建文件
	}
	absParent := filepath.Join(root, parent)
	fi, err := os.Stat(absParent)
	return err == nil && fi.IsDir()
}

// isSelfHarmCommand 检测命令是否会伤害自身进程（companion 进程本身）。
// 自身项目指包含 cmd/companion 目录的项目（即 companion 自身）。
// 返回非空字符串表示阻止原因，空字符串表示安全。
// 适用场景：agent 迭代自身代码后，LLM 可能尝试杀死旧进程或启动新实例导致自身崩溃。
func isSelfHarmCommand(command, root string) string {
	// 检查是否是 companion 项目根
	companionDir := filepath.Join(root, "cmd", "companion")
	if _, err := os.Stat(companionDir); os.IsNotExist(err) {
		return "" // 不是自身项目，放行
	}

	lower := strings.ToLower(command)

	// 1. 杀死自身进程（taskkill / kill / Stop-Process 等）
	if strings.Contains(lower, "taskkill") {
		if strings.Contains(lower, "companion") || strings.Contains(lower, "/pid") || strings.Contains(lower, "/im") {
			return "⚠️ 禁止杀死自身进程：命令尝试终止 companion 进程，但当前 agent 正运行在 companion 中，杀死自己会导致所有后续动作无法执行。如欲验证修改效果，请使用 run_background 并设置不同端口（如 WEB_PORT=9091）。"
		}
	}
	if strings.Contains(lower, "stop-process") && strings.Contains(lower, "companion") {
		return "⚠️ 禁止杀死自身进程：命令尝试终止 companion 进程（当前 agent 自身）..."
	}
	if (strings.Contains(lower, " pkill ") || strings.Contains(lower, "killall ")) && strings.Contains(lower, "companion") {
		return "⚠️ 禁止杀死自身进程..."
	}

	// 2. 直接运行 companion（端口冲突导致进程异常退出）
	// 排除 "go run" 的编译过程（不在同一个 cmd /C 上下文中）
	if lower == "companion.exe" || lower == "./companion.exe" || lower == "./companion" ||
		lower == "start companion.exe" || lower == `start "" companion.exe` {
		return "⚠️ 禁止直接运行 companion.exe：当前 agent 已在运行中（端口已被占用）。如需测试新版本，请用 run_background 并设置 WEB_PORT=9091 等不同端口。"
	}

	// 3. 在同一个命令中 build + run companion（常见 agent 自我迭代模式）
	hasBuildCompanion := strings.Contains(lower, "go build") &&
		(strings.Contains(lower, "./cmd/companion") || strings.Contains(lower, " cmd/companion"))
	hasRunCompanion := strings.Contains(lower, "companion.exe") || strings.Contains(lower, " && .\\")
	if hasBuildCompanion && hasRunCompanion {
		return "⚠️ 禁止构建并运行自身项目：这会覆盖正在运行的二进制或导致端口冲突。如需测试，请用不同目录+不同端口（如 WEB_PORT=9091）。"
	}

	return ""
}

// isBlockingCommand 检测命令是否为长期进程（会阻塞 bash 120s 超时）。
// 匹配高置信度模式：dev server / watch / 文件监听等。短命令（build/test/install）不命中。
func isBlockingCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	// 精确匹配 npm start（独立启动服务）
	if lower == "npm start" || lower == "yarn start" || lower == "pnpm start" {
		return true
	}
	// npm run / yarn / pnpm + 阻塞子命令
	if matched := hasRunCommandBlockingPattern(lower); matched {
		return true
	}
	// 裸 go run .（运行当前包，通常是服务端）
	if lower == "go run ." || lower == "go run ./..." {
		return true
	}
	// go run ./cmd/xxx（可能是服务入口）
	if strings.HasPrefix(lower, "go run ./cmd/") && !strings.Contains(lower, " ") {
		return true
	}
	// 裸 vite / webpack-dev-server / nodemon
	baseCmd := extractBaseCommand(lower)
	switch baseCmd {
	case "vite", "vitepress", "nodemon", "webpack-dev-server", "live-server",
		"browser-sync", "parcel", "ts-node-dev", "concurrently":
		return true
	}
	// 带 --watch / --serve / watch 标志
	if strings.Contains(lower, "--watch") || strings.Contains(lower, "-w") ||
		strings.Contains(lower, "--serve") || strings.Contains(lower, "watch") {
		return true
	}
	return false
}

// hasRunCommandBlockingPattern 检查 npm run / yarn / pnpm 后的子命令是否为阻塞类型。
func hasRunCommandBlockingPattern(cmd string) bool {
	// 提取 run 后的子命令
	subCmd := ""
	if idx := strings.Index(cmd, " run "); idx >= 0 {
		subCmd = cmd[idx+5:]
	} else if strings.HasPrefix(cmd, "yarn ") && !strings.HasPrefix(cmd, "yarn run ") {
		subCmd = cmd[5:]
	}
	if subCmd == "" {
		return false
	}
	subCmd = strings.TrimSpace(subCmd)
	// 移除尾部参数（如 --port 3000）
	if spaceIdx := strings.Index(subCmd, " "); spaceIdx > 0 {
		subCmd = subCmd[:spaceIdx]
	}
	blockingSubCmds := map[string]bool{
		"dev": true, "serve": true, "start": true, "watch": true,
		"develop": true, "server": true, "hot": true, "hmr": true,
		"webpack-dev-server": true, "storybook": true, "docs:dev": true,
	}
	return blockingSubCmds[subCmd]
}

// extractBaseCommand 提取命令行的首命令（去掉路径前缀和参数）。
func extractBaseCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if idx := strings.Index(cmd, " "); idx > 0 {
		cmd = cmd[:idx]
	}
	// 取 basename
	cmd = strings.TrimPrefix(cmd, "./")
	cmd = strings.TrimPrefix(cmd, "npx ")
	return cmd
}
