package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"path/filepath"
	"regexp"

	"sort"
	"strconv"
	"strings"
	"sync"

	"time"
)

// WorkspaceRoots 工作区所有根目录（多根工作区支持）。
// 由 bridge.go 在初始化 agent 时设置。resolvePath 会检查路径是否在任一根目录内。
var WorkspaceRoots []string

// FileChangeCallback 文件变更回调（可选）。每次写类工具成功修改文件后调用，供外部追踪变更。
// filePath 为工作区相对路径。由 orchestration.go 或外部宿主设置。
var FileChangeCallback func(filePath string)

// ErrRetry 由 OnToolError 钩子返回，指示 Execute 用修改后的 args 重新执行 handler。
// 用于可恢复错误（如 edit_file 匹配失败→自动降级行号定位重试）。
// 注意：OnToolError 应通过 args（引用）设置重试参数。
var ErrRetry = errors.New("__retry__")

// ToolHandler 工具执行体：收到已解析的 JSON 参数，返回结果文本或 error。
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool 一个已注册工具（名/描述/参数 Schema/执行体 + 元信息）。
type Tool struct {
	Name             string
	Description      string // 简短描述（传给 LLM function-calling）
	UsageGuide       string // ★ 详细使用指导：何时用此工具、注意事项、对比 run_command 的优势
	Category         string // ★ 工具分类：如 "code-search", "git", "file", "web", "debug", "build", "test"
	Parameters       map[string]any // JSON Schema
	Handler          ToolHandler
	ReadOnly         bool // 只读（不改文件系统）——供并行/免审
	RequiresApproval bool // 写类工具：需人工确认（UI 接入时用）
	Enabled          bool // ★ 是否启用（默认 true；按工作区配置可关闭）
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
	// 推送中间结果（如 run_command 的逐行输出、read_file 的翻页进度）。
	// callID 为工具调用 ID（空串表示非工具调用场景），partialResult 为当前累积的中间文本。
	// 此钩子不替代最终结果，仅用于流式展示。handler 的返回值仍是正式结果。
	OnToolUpdate func(name string, callID string, partialResult string)

	CommitMessage string // agent 通过 generate_commit_message 工具显式设置的提交信息
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: map[string]*Tool{}}
}
// Register 注册一个工具（同名覆盖，顺序不变）。
func (r *Registry) Register(t *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name]; !exists {
		r.order = append(r.order, t.Name)
	}
	if t.Enabled == false { // 未显式设置则默认启用
		t.Enabled = true
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

// Unregister 卸载工具（Lua 热重载用）。不存在则无操作。
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


// UsageGuideText 生成工具使用指南文本（供注入系统提示使用）。
// 按 Category 分组，展示每个已启用工具的 UsageGuide。
func (r *Registry) UsageGuideText() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type guideEntry struct {
		name     string
		guide    string
		category string
	}
	entries := make([]guideEntry, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if !t.Enabled || t.UsageGuide == "" {
			continue
		}
		entries = append(entries, guideEntry{name, t.UsageGuide, t.Category})
	}
	if len(entries) == 0 {
		return ""
	}
	// 按 Category 分组
	groups := map[string][]guideEntry{}
	var cats []string
	catSet := map[string]bool{}
	for _, e := range entries {
		cat := e.category
		if cat == "" {
			cat = "其他"
		}
		if !catSet[cat] {
			catSet[cat] = true
			cats = append(cats, cat)
		}
		groups[cat] = append(groups[cat], e)
	}
	var b strings.Builder
	b.WriteString("📋 工具使用指南（按分类，请优先使用专用工具而非 run_command）：\n\n")
	for _, cat := range cats {
		ents := groups[cat]
		b.WriteString("### " + cat + "\n")
		for _, e := range ents {
			b.WriteString("- **" + e.name + "**：" + e.guide + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("💡 通用原则：当存在专用工具时，请优先使用专用工具而非 run_command。" +
		"专用工具拥有更精确的参数校验、错误处理和输出格式化，结果更可靠。")
	return b.String()
}

// Copy 深拷贝 Registry（含钩子引用）。子 Loop 用副本注册工具，避免污染父表。
func (r *Registry) Copy() *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := &Registry{
		tools:       map[string]*Tool{},
		order:       append([]string(nil), r.order...),
		BeforeTool:  r.BeforeTool,
		AfterTool:   r.AfterTool,
		OnToolError: r.OnToolError,
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
		tools:       map[string]*Tool{},
		BeforeTool:  r.BeforeTool,
		AfterTool:   r.AfterTool,
		OnToolError: r.OnToolError,
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
			ReadOnly:    t.ReadOnly,
		})
	}
	return metas
}



// Definitions 导出已启用工具的定义（按注册顺序），传给 LLM 作 function-calling。
// 只包含 Enabled=true 的工具。禁用工具不暴露给 LLM。
func (r *Registry) Definitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if !t.Enabled {
			continue // 禁用的工具不暴露给 LLM
		}
		desc := t.Description
		if len(desc) > 120 {
			// 在第一个句号或空格处截断
			cut := strings.LastIndex(desc[:120], "。")
			if cut < 60 {
				cut = len(desc[:100])
				for cut > 60 && desc[cut] != ' ' && desc[cut] != ',' {
					cut--
				}
			}
			desc = strings.TrimSpace(desc[:cut])
		}
		defs = append(defs, ToolDefinition{
			Type:     "function",
			Function: FunctionDefinition{Name: t.Name, Description: desc, Parameters: t.Parameters},
		})
	}
	return defs
}

// Execute 解析 JSON 参数并执行工具。参数 JSON 由 LLM 流式拼接而来，可能为空。
// 依次触发 BeforeTool → handler → AfterTool → OnToolError（仅出错时）钩子。
func (r *Registry) Execute(ctx context.Context, name, argsJSON string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	args := map[string]any{}
	if s := strings.TrimSpace(argsJSON); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			return "", fmt.Errorf("参数 JSON 解析失败: %w（原文 %q）", err, argsJSON)
		}
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
	start := time.Now()
	result, err := t.Handler(ctx, args)
	dur := time.Since(start)
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

// RegisterDefaultTools 注册核心工具，全部限定在工作区 root 内（安全底线：禁访问工作区外）。
// read_file / write_file / edit_file / list_files / run_command。
func RegisterDefaultTools(r *Registry, root string) {
	eh := newEditHistory() // ★ v2: 编辑行号偏移追踪器
	bg := &bgRegistry{procs: map[int]*bgProc{}} // 共享后台进程注册表
	r.Register(&Tool{
		Name:        "read_file",
		UsageGuide:  "读取文件内容，限工作区内路径。大文件用 offset+limit 分页读取，避免撑爆上下文。二进制文件会自动拒绝读取，请改用 inspect_binary。比 os.ReadFile 更安全（路径越界拦截+二进制保护）。",
		Description: "读取文件内容。path 为工作区内路径。可选 offset(起始行,1 基)+limit(行数)读片段；省略则读全文(超 2000 行只返回前 2000 行并提示用 offset/limit 翻页)。",
		Parameters:  objSchema(props{"path": strProp("文件路径（工作区内）"), "offset": intProp("可选：起始行号(1 基)"), "limit": intProp("可选：读取行数")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			// 二进制保护：含 NULL 字节视为二进制，拒绝读取并引导 inspect_binary（避免把字节流灌进上下文）
			if strings.IndexByte(string(data), 0) >= 0 {
				return "", fmt.Errorf("「%s」是二进制文件，read_file 不支持读取二进制内容；请用 inspect_binary 工具查看（hexdump/类型嗅探）", argStr(args, "path"))
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
		Name:             "write_file",
		UsageGuide:       "写入文件，父目录自动创建。需审核批准。比 os.WriteFile 更安全（自动快照+路径越界拦截+变更回调）。如需追加内容请先用 read_file 读入再加上新内容后 write_file 覆盖。",
		Description:      "把 content 完整写入 path（覆盖；父目录自动创建）。",
		Parameters:       objSchema(props{"path": strProp("文件路径"), "content": strProp("完整文件内容")}, "path", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
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
		Name: "edit_file",
		UsageGuide: "把文件中唯一一处 old_string 替换为 new_string。内置智能匹配（CRLF 归一化+空白折叠）。匹配失败时优先用 line_start/line_end 行号定位（最可靠）。比手动 read+write 更精确（保留换行风格+行号偏移追踪+codegraph 自动注入）。仅用于小改动（≤5 行），大改动请用 write_file 写整段。",
		Description: "把文件中唯一一处 old_string 替换为 new_string。" +
			"匹配策略（自动）：精确→CRLF归一化（兼容 Windows \\r\\n 文件与 LLM 给的 \\n）→空白折叠（容忍缩进/行尾空白/tab与空格差异）；全部失败时返回带行号上下文的诊断。" +
			"替代方案：用 line_start/line_end 行号定位整段替换（最可靠，old_string 可选作校验）。" +
			"保留文件原换行风格（CRLF 文件替换后仍 CRLF）。",
		Parameters: objSchema(props{
			"path":       strProp("文件路径"),
			"old_string": strProp("待替换原文（须在文件中唯一；line_start>0 时可省略或作校验）"),
			"new_string": strProp("替换后的新文"),
			"line_start": intProp("可选：1 基起始行号，>0 时启用行号定位模式（与 old_string 二选一或并用）"),
			"line_end":   intProp("可选：1 基结束行号（含）；省略或 < line_start 时只替换 line_start 一行"),
		}, "path", "new_string"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			SnapshotBeforeWriteWithTracking(root, p)

			orig, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			origStr := string(orig)
			newStr, err := ApplyEdit(origStr, EditOptions{
				OldString: argStr(args, "old_string"),
				NewString: argStr(args, "new_string"),
				LineStart: argInt(args, "line_start", 0),
				LineEnd:   argInt(args, "line_end", 0),
			})
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(p, []byte(newStr), 0o644); err != nil {
				return "", err
			}
			// ★ v2: 行号偏移追踪 + 编辑后上下文反馈
			oldLC := countLines(origStr)
			newLC := countLines(newStr)
			delta := newLC - oldLC
			ls := argInt(args, "line_start", 0)
			le := argInt(args, "line_end", 0)
			if ls <= 0 {
				ls = 1
			}
			if le <= 0 {
				le = ls
			}
			nl := countLines(argStr(args, "new_string"))
			eh.record(fileEditRecord{Path: p, LineDelta: delta, EditEnd: ls + nl - 1})
		editCtx := editContext(strings.Split(normalizeNewlines(newStr), "\n"), ls, le, nl, delta)
			if FileChangeCallback != nil {
				FileChangeCallback(argStr(args, "path"))
			}
		return fmt.Sprintf("已编辑 %s\n%s", argStr(args, "path"), editCtx), nil
		},
	})

	r.Register(&Tool{
		Name: "multi_edit",
		UsageGuide: "按顺序对一个文件应用多处替换。比多次 edit_file 更高效（原子提交：任一步失败全部回滚）。编辑项较多时用 multi_edit 替代多次 edit_file 调用。",
		Description: "对一个文件按顺序应用多处替换（edits：每项 old_string→new_string 或 line_start/line_end 行号定位）。" +
			"匹配策略同 edit_file（精确→CRLF归一化→空白折叠→诊断）。原子：任一步失败则全部不写。" +
			"比多次 edit_file 高效。保留文件原换行风格。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path": strProp("文件路径"),
				"edits": map[string]any{
					"type":        "array",
					"description": "按顺序应用的替换列表",
					"items": map[string]any{
						"type": "object",
						"properties": props{
							"old_string": strProp("待替换原文（须唯一；line_start>0 时可省略或作校验）"),
							"new_string": strProp("替换后的新文"),
							"line_start": intProp("可选：1 基起始行号，>0 时启用行号定位模式"),
							"line_end":   intProp("可选：1 基结束行号（含）；省略只替换 line_start 一行"),
						},
						"required": []string{"new_string"},
					},
				},
			},
			"required": []string{"path", "edits"},
		},
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			SnapshotBeforeWriteWithTracking(root, p)
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			edits, _ := args["edits"].([]any)
			if len(edits) == 0 {
				return "", fmt.Errorf("edits 不能为空")
			}
			origLC := countLines(string(data))
			content := string(data)
			totalDelta := 0
			lastEditEnd := 0
			for i, it := range edits {
				m, ok := it.(map[string]any)
				if !ok {
					return "", fmt.Errorf("edits[%d] 格式错误", i)
				}
				old, _ := m["old_string"].(string)
				neu, _ := m["new_string"].(string)
				ls := argInt(m, "line_start", 0)
				le := argInt(m, "line_end", 0)
				if old == "" && ls <= 0 {
					return "", fmt.Errorf("edits[%d] 必须提供 old_string 或 line_start", i)
				}
				// ★ v2: 自动补偿行号偏移
				if ls > 0 && totalDelta != 0 && ls > lastEditEnd {
					ls += totalDelta
					if le > 0 {
						le += totalDelta
					}
				}
				out, err := ApplyEdit(content, EditOptions{
					OldString: old,
					NewString: neu,
					LineStart: ls,
					LineEnd:   le,
				})
				if err != nil {
					return "", fmt.Errorf("edits[%d] 应用失败: %w", i, err)
				}
				curDelta := countLines(out) - countLines(content)
				totalDelta += curDelta
				if ls > 0 {
					lastEditEnd = ls + countLines(neu) - 1
				}
				content = out
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
			newLC := countLines(content)
			eh.record(fileEditRecord{Path: p, LineDelta: newLC - origLC, EditEnd: lastEditEnd})
			if FileChangeCallback != nil {
				FileChangeCallback(argStr(args, "path"))
			}
			return fmt.Sprintf("已对 %s 应用 %d 处编辑（累计偏移 %+d 行）", argStr(args, "path"), len(edits), totalDelta), nil
		},
	})

	r.Register(&Tool{
		Name:        "list_files",
		UsageGuide:  "列出工作区目录下的文件和子目录（目录排前）。比 run_command dir /s 更高效（跳过 .git/node_modules、结果结构化排序）。配合 pattern 按通配符过滤文件（如 *.go）。",
		Description: "列出目录下的文件/子目录（目录在前）。path 省略则列工作区根；pattern 可选（如 *.go）。",
		Parameters:  objSchema(props{"path": strProp("目录路径（省略=工作区根）"), "pattern": strProp("可选通配符过滤，如 *.go")}, ),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rel := argStr(args, "path")
			p := root
			if rel != "" {
				var err error
				if p, err = resolvePath(root, rel); err != nil {
					return "", err
				}
			}
			entries, err := os.ReadDir(p)
			if err != nil {
				return "", err
			}
			pattern := argStr(args, "pattern")
			sort.SliceStable(entries, func(i, j int) bool {
				if entries[i].IsDir() != entries[j].IsDir() {
					return entries[i].IsDir()
				}
				return entries[i].Name() < entries[j].Name()
			})
			var b strings.Builder
			for _, e := range entries {
				if pattern != "" && !e.IsDir() {
					if ok, _ := filepath.Match(pattern, e.Name()); !ok {
						continue
					}
				}
				if e.IsDir() {
					b.WriteString(e.Name() + "/\n")
				} else {
					sz := int64(-1)
					if fi, err := e.Info(); err == nil {
						sz = fi.Size()
					}
					fmt.Fprintf(&b, "%s\t%d\n", e.Name(), sz)
				}
			}
			if b.Len() == 0 {
				return "（空目录或无匹配）", nil
			}
			return b.String(), nil
		},
	})
	r.Register(&Tool{
		Name:             "run_command",
		UsageGuide:       "同步执行 shell 命令，120s 超时自动终止（内部后台执行，不阻塞 agent）。适用于构建、编译、测试、文件查询等短命令。禁止用于长期进程（dev server/npm run dev/watch 模式）——请改用 run_background。比直接手动执行更安全（路径越界拦截+输出截断 16KB+UTF-8 编码统一）。",
		Description:      "同步执行一条 shell 命令并返回输出。适用于构建、编译、测试、文件查询等短命令（几秒内完成）。\n禁止用于以下场景（会阻塞 agent）：启动 dev server、npm run dev、go run 启动服务、watch 模式、tcp 监听、任何需保持运行的进程。此类命令请改用 run_background。",
		Parameters:       objSchema(props{"command": strProp("要执行的命令"), "cwd": strProp("可选工作目录（工作区内，省略=根）")}, "command"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			if strings.TrimSpace(command) == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			// ★ 通过 bg.start() 后台启动命令（不阻塞 loop），然后轮询等待完成。
			dir := root
			if cwd := argStr(args, "cwd"); cwd != "" {
				var err error
				if dir, err = resolvePath(root, cwd); err != nil {
					return "", err
				}
			}
			id, err := bg.start(command, dir)
			if err != nil {
				return "", err
			}
			p := bg.get(id)
			if p == nil {
				return "", fmt.Errorf("内部错误：后台进程创建后丢失")
			}
			// 轮询等待完成（带超时和 ctx 中断），不阻塞 loop 线程。
			deadline := time.After(120 * time.Second)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					// 上下文取消时杀死子进程，防止残留
					if p.cmd != nil && p.cmd.Process != nil {
						killProcessTree(p.cmd.Process.Pid)
					}
					out, _, _ := p.snapshot()
					return capOutput(out, 16000) + "\n[已中断: " + ctx.Err().Error() + "]", nil
				case <-deadline:
					killProcessTree(p.cmd.Process.Pid)
					out, _, _ := p.snapshot()
					return capOutput(out, 16000) + "\n[超时 120s 已终止]", nil
				case <-ticker.C:
					out, done, exitErr := p.snapshot()
					if done {
						res := capOutput(out, 16000)
						if exitErr != "" {
							res += "\n[退出: " + exitErr + "]"
						}
						// 清理已完成的进程记录
						bg.mu.Lock()
						delete(bg.procs, id)
						bg.mu.Unlock()
						return res, nil
					}
				}
			}
		},
	})

	r.Register(&Tool{
		Name:             "move_file",
		UsageGuide:       "移动或重命名工作区内的文件/目录。目标父目录自动创建。需审核批准。覆盖 os.Rename 的限制（自动创建目标目录+路径越界拦截+变更通知）。",
		Description:      "把文件/目录从 from 移动或重命名到 to（都在工作区内；目标父目录自动创建）。",
		Parameters:       objSchema(props{"from": strProp("源路径"), "to": strProp("目标路径")}, "from", "to"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			from, err := resolvePath(root, argStr(args, "from"))
			if err != nil {
				return "", err
			}
			to, err := resolvePath(root, argStr(args, "to"))
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return "", err
			}
			if err := os.Rename(from, to); err != nil {
				return "", err
			}
			if FileChangeCallback != nil {
				FileChangeCallback(argStr(args, "from") + " → " + argStr(args, "to"))
			}
			return fmt.Sprintf("已移动 %s → %s", argStr(args, "from"), argStr(args, "to")), nil
		},
	})

	r.Register(&Tool{
		Name:             "delete_file",
		UsageGuide:       "删除工作区内的文件（不可恢复，谨慎）。为安全不删目录（删除目录请用 run_command rmdir）。需审核批准。比直接 os.Remove 更安全（只删文件不删目录+路径越界拦截）。",
		Description:      "删除一个文件（工作区内，不可恢复，谨慎）。为安全不删目录。",
		Parameters:       objSchema(props{"path": strProp("要删除的文件路径")}, "path"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			info, err := os.Stat(p)
			if err != nil {
				return "", err
			}
			if info.IsDir() {
				return "", fmt.Errorf("delete_file 不删目录：%s", argStr(args, "path"))
			}
			if err := os.Remove(p); err != nil {
				return "", err
			}
			if FileChangeCallback != nil {
				FileChangeCallback("(删除) " + argStr(args, "path"))
			}
			return "已删除 " + argStr(args, "path"), nil
		},
	})

	registerSearchTools(r, root)              // search_content / search_files（见 search.go）
	registerGitTools(r, root)                 // git_status / git_diff / git_log / git_show / git_blame / git_add / ...（见 git.go）
	registerWebTools(r)                       // web_fetch / web_search（联网，见 web.go）
	// update_plan 仅在自主模式外层注册（RegisterPlanOnlyTools），非自主模式不暴露
	registerShellTools(r, bg, root)               // run_background / read_output / kill_process（后台命令，见 shell.go）
	registerMemoryTools(r, root)              // memory_write/read/list/search（跨会话记忆，见 memory.go）
	registerVerifyTools(r, root)              // memory_verify / project_info_verify（过期验证，见 verify_tools.go）
	// find_files_by_pattern 已合并到 search_files（增加 language 参数），不再独立注册。
	registerFindSymbolTool(r, root)           // find_symbol（符号定位，见 symbolfinder.go）
	registerFileSymbolTools(r, root)          // list_exported_symbols / get_file_dependencies / check_impact / find_circular_deps（见 filesymbol.go）
	registerTaskTools(r, root)                // task_create/update/list/delete/summary（持久化任务追踪，见 task_tools.go）
	registerProjectInfoTools(r, root)        // project_info_write/read/list/search/delete/explore（项目知识库，见 projectinfo.go）
	registerBinaryTools(r, root)             // inspect_binary / write_binary（二进制读写，见 binary.go）
	registerBinaryRETools(r, root)           // binary_strings/find/patch/info/hash/entropy（二进制正则，见 binary_re.go）
	registerDebugTools(r, root)              // debug_start/stop/breakpoint/continue/next/step_in/step_out/stack/variables/evaluate/status（见 debugtools.go）
	registerVisionTools(r, root)             // image_analyze / image_ocr（图像视觉分析，见 vision.go）
	registerScreenshotTools(r, root)         // screenshot_desktop/window/area/webpage（截图工具，见 screenshot_tool.go）
	registerWebDebugTool(r, root)            // web_debug（网页验证：控制台错误+截图+JS执行+交互+文字提取，见 webdebug.go）
	RegisterBugTools(r, root)                // bug_detect / bug_analyze / bug_fix（BUG 自动检测与修复，见 bugdetect.go + bugfix.go）
	registerOfficeTools(r, root)             // csv_read / csv_write / json_to_table / table_stats / text_report / word_read（见 officetools.go）
	registerLSPTools(r, root)              // lsp_definition / lsp_references / lsp_hover / lsp_diagnostics（见 lsptools.go）
	registerCodeGraphTools(r, root)          // codegraph_build / codegraph_search / codegraph_impact / ...（代码知识图谱，见 codegraph_tools.go + pkg/codegraph）
	registerExtraCodeGraphTools(r, root)     // codegraph_find_by_signature / codegraph_explore（额外工具，见 codegraph_extra.go）
	registerLuaToolTools(r, root)            // lua_tool_list/create/update/delete（Lua 自定义工具管理，见 luatool_tools.go）
	// ── 默认 BeforeTool：edit_file/multi_edit 执行前用 codegraph 注入最新行号 ──
	// codegraph 的符号级行号比 old_string 字符串匹配更可靠（不受 CRLF/空白折叠/行号偏移影响）。
	if r.BeforeTool == nil {
		r.BeforeTool = func(ctx context.Context, name string, args map[string]any) (bool, string, error) {
			if name != "edit_file" && name != "multi_edit" {
				return true, "", nil // 放行
			}
			oldStr, _ := args["old_string"].(string)
			filePath, _ := args["path"].(string)
			if oldStr == "" || filePath == "" {
				return true, "", nil // 放行
			}
			symName := extractGoSymbolName(oldStr)
			if symName == "" {
				return true, "", nil // 放行
			}
			cg, cgErr := getCodeGraph(root)
			if cgErr != nil || cg == nil {
				return true, "", nil // 放行
			}
			for _, e := range cg.SearchEntities(symName) {
				if e.FilePath != filePath {
					continue
				}
				if !strings.Contains(e.Name, symName) {
					continue
				}
				// ★ 找到匹配实体 → 注入最新行号，让 edit_file 用行号定位执行
				args["line_start"] = float64(e.Line)
				if e.EndLine > e.Line {
					args["line_end"] = float64(e.EndLine)
				}
				break
			}
			return true, "", nil // 放行（参数已被修改）
		}
	}

	// ── 默认 OnToolError：edit_file/multi_edit 匹配失败→自动行号定位重试 ──
	// 注意：BeforeTool 已用 codegraph 预注入行号，此处为兜底（codegraph 找不到时）。
	if r.OnToolError == nil {
		r.OnToolError = func(ctx context.Context, name string, args map[string]any, err error) (string, error) {
			if name != "edit_file" && name != "multi_edit" {
				return "", err
			}
			if ls, _ := args["line_start"].(float64); ls > 0 {
				return "", err
			}
			errStr := err.Error()
			if !strings.Contains(errStr, "未找到") && !strings.Contains(errStr, "多次") && !strings.Contains(errStr, "不唯一") {
				return "", err
			}
			re := regexp.MustCompile(`L(\d+):`)
			m := re.FindStringSubmatch(errStr)
			if len(m) < 2 {
				return "", err
			}
			lineNo, _ := strconv.Atoi(m[1])
			if lineNo <= 0 {
				return "", err
			}
			args["line_start"] = float64(lineNo)
			return "", ErrRetry
		}
	}
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

func strProp(desc string) map[string]any  { return map[string]any{"type": "string", "description": desc} }
func boolProp(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }
func intProp(desc string) map[string]any  { return map[string]any{"type": "integer", "description": desc} }

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

// resolvePath 把相对/绝对路径解析为工作区内的绝对路径，越界则报错（安全底线）。
// 先检查路径是否在 primary root 下；若不在，再查是否在 WorkspaceRoots（工作区其他根目录）下。
func resolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	full := p
	full = filepath.Clean(full)

	// 先查 primary root
	rel, err := filepath.Rel(root, full)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// 路径语法上属于 primary root → 确认真实存在于此
		//（避免 `goui/main.go` 被拼成 `gou-ide/goui/main.go` 但实际应在 `../goui/`）
		if pathExists(full) || parentDirExists(root, full) {
			return full, nil
		}
		// 不存在于 primary root → 可能属于其他工作区根，继续查
	}

	// 再查其他工作区根目录（多根工作区支持）
	for _, wr := range WorkspaceRoots {
		if wr == root {
			continue
		}
		rel, err := filepath.Rel(wr, full)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			// 路径语法上属于此工作区根 → 确认真实存在
			if pathExists(full) || parentDirExists(wr, full) {
				return full, nil
			}
		}
	}

	// 兜底：文件尚未在任何根下创建 → 默认用 primary root（新建文件归宿）
	return full, nil
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

// isBlockingCommand 检测命令是否为长期进程（会阻塞 run_command 120s 超时）。
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

// extractGoSymbolName 从 Go 代码片段（old_string）中提取符号名称。
// 用于 edit_file 匹配失败时通过 codegraph 定位符号的最新行号。
// 匹配优先级：方法 > 函数 > 类型 > 变量 > 常量。
func extractGoSymbolName(s string) string {
	if s == "" {
		return ""
	}
	// 去掉可能的前导空白和注释
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") {
		return ""
	}

	// 方法：func (r *Receiver) MethodName(
	re := regexp.MustCompile(`(?m)^func\s+\([^)]*\)\s*(\w+)\s*\(`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}

	// 函数：func FunctionName(
	re = regexp.MustCompile(`(?m)^func\s+(\w+)\s*\(`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}

	// 类型/结构体/接口：type TypeName
	re = regexp.MustCompile(`(?m)^type\s+(\w+)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}

	// 变量：var VarName
	re = regexp.MustCompile(`(?m)^var\s+(\w+)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}

	// 常量：const ConstName
	re = regexp.MustCompile(`(?m)^const\s+(\w+)`)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}

	return ""
}
