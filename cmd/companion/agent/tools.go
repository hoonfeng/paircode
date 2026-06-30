package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// WorkspaceRoots 工作区所有根目录（多根工作区支持）。
// 由 bridge.go 在初始化 agent 时设置。resolvePath 会检查路径是否在任一根目录内。
var WorkspaceRoots []string

// ToolHandler 工具执行体：收到已解析的 JSON 参数，返回结果文本或 error。
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool 一个已注册工具（名/描述/参数 Schema/执行体 + 元信息）。
type Tool struct {
	Name             string
	Description      string
	Parameters       map[string]any // JSON Schema
	Handler          ToolHandler
	ReadOnly         bool // 只读（不改文件系统）——供并行/免审
	RequiresApproval bool // 写类工具：需人工确认（UI 接入时用）
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

// Definitions 导出全部工具定义（按注册顺序），传给 LLM 作 function-calling。
func (r *Registry) Definitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		defs = append(defs, ToolDefinition{
			Type:     "function",
			Function: FunctionDefinition{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
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
	start := time.Now()
	result, err := t.Handler(ctx, args)
	dur := time.Since(start)
	// AfterTool 钩子：观察（统计/日志），不改结果
	if r.AfterTool != nil {
		r.AfterTool(ctx, name, args, result, err, dur)
	}
	// OnToolError 钩子：错误诊断增强 / 可恢复错误降级（返回 ("",nil) 吞掉错误）
	if err != nil && r.OnToolError != nil {
		result, err = r.OnToolError(ctx, name, args, err)
	}
	return result, err
}

// ─── 核心工具集 ──────────────────────────────────────────────

// RegisterDefaultTools 注册核心工具，全部限定在工作区 root 内（安全底线：禁访问工作区外）。
// read_file / write_file / edit_file / list_files / run_command。
func RegisterDefaultTools(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "read_file",
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
		Description:      "把 content 完整写入 path（覆盖；父目录自动创建）。",
		Parameters:       objSchema(props{"path": strProp("文件路径"), "content": strProp("完整文件内容")}, "path", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			content := argStr(args, "content")
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("已写入 %s（%d 字节）", argStr(args, "path"), len(content)), nil
		},
	})

	r.Register(&Tool{
		Name: "edit_file",
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
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			out, err := ApplyEdit(string(data), EditOptions{
				OldString: argStr(args, "old_string"),
				NewString: argStr(args, "new_string"),
				LineStart: argInt(args, "line_start", 0),
				LineEnd:   argInt(args, "line_end", 0),
			})
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
				return "", err
			}
			return "已编辑 " + argStr(args, "path"), nil
		},
	})

	r.Register(&Tool{
		Name: "multi_edit",
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
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			edits, _ := args["edits"].([]any)
			if len(edits) == 0 {
				return "", fmt.Errorf("edits 不能为空")
			}
			content := string(data)
			for i, it := range edits {
				m, ok := it.(map[string]any)
				if !ok {
					return "", fmt.Errorf("edits[%d] 格式错误", i)
				}
				old, _ := m["old_string"].(string)
				neu, _ := m["new_string"].(string)
				lineStart := 0
				lineEnd := 0
				switch v := m["line_start"].(type) {
				case float64:
					lineStart = int(v)
				case int:
					lineStart = v
				}
				switch v := m["line_end"].(type) {
				case float64:
					lineEnd = int(v)
				case int:
					lineEnd = v
				}
				if old == "" && lineStart <= 0 {
					return "", fmt.Errorf("edits[%d] 必须提供 old_string 或 line_start", i)
				}
				out, err := ApplyEdit(content, EditOptions{
					OldString: old,
					NewString: neu,
					LineStart: lineStart,
					LineEnd:   lineEnd,
				})
				if err != nil {
					return "", fmt.Errorf("edits[%d] 应用失败: %w", i, err)
				}
				content = out
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("已对 %s 应用 %d 处编辑", argStr(args, "path"), len(edits)), nil
		},
	})

	r.Register(&Tool{
		Name:        "list_files",
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
		Description:      "在工作区执行一条 shell 命令并返回输出（同步、120s 超时、UTF-8）。用于构建/测试/查询。",
		Parameters:       objSchema(props{"command": strProp("要执行的命令"), "cwd": strProp("可选工作目录（工作区内，省略=根）")}, "command"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			if strings.TrimSpace(command) == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			dir := root
			if cwd := argStr(args, "cwd"); cwd != "" {
				var err error
				if dir, err = resolvePath(root, cwd); err != nil {
					return "", err
				}
			}
			cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			// chcp 65001 统一 UTF-8 输出（避免中文乱码，同终端面板）。
			c := exec.CommandContext(cctx, "cmd", "/C", "chcp 65001 >nul & "+command)
			c.Dir = dir
			out, err := c.CombinedOutput()
			res := capOutput(string(out), 16000)
			if cctx.Err() == context.DeadlineExceeded {
				res += "\n[超时 120s 已终止]"
			} else if err != nil {
				res += "\n[退出: " + err.Error() + "]"
			}
			return res, nil
		},
	})

	r.Register(&Tool{
		Name:             "move_file",
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
			return fmt.Sprintf("已移动 %s → %s", argStr(args, "from"), argStr(args, "to")), nil
		},
	})

	r.Register(&Tool{
		Name:             "delete_file",
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
			return "已删除 " + argStr(args, "path"), nil
		},
	})

	registerSearchTools(r, root)              // search_content / search_files（见 search.go）
	registerGitTools(r, root)                 // git_status / git_diff / git_log / git_show / git_blame / git_add / ...（见 git.go）
	registerWebTools(r)                       // web_fetch / web_search（联网，见 web.go）
	registerPlanTool(r)                       // update_plan（任务清单，见 plan.go）
	registerShellTools(r, root)               // run_background / read_output / kill_process（后台命令，见 shell.go）
	registerMemoryTools(r, root)              // memory_write/read/list/search（跨会话记忆，见 memory.go）
	registerFindFilesByPatternTool(r, root)   // find_files_by_pattern（glob 查文件，支持 **，见 findfiles.go）
	registerFindSymbolTool(r, root)           // find_symbol（符号定位，见 symbolfinder.go）
	registerFileSymbolTools(r, root)          // get_file_symbols / find_symbol_usages / check_impact / find_circular_deps（见 filesymbol.go）
	registerTaskTools(r, root)                // task_create/update/list/delete/summary（持久化任务追踪，见 task_tools.go）
	registerGoBuildTools(r, root)             // go_build（见 gobuild.go）
	registerRunTestTools(r, root)             // run_test（见 runtest.go）
	registerGoRunTools(r, root)               // go_run（见 gorun.go）
	registerCodeFixTool(r, root)              // code_fix（见 fixer.go）
	registerCodeFormatTool(r, root)          // code_format（见 formatter.go）
	registerProjectInfoTools(r, root)        // project_info_write/read/list/search/delete/explore（项目知识库，见 projectinfo.go）
	registerBinaryTools(r, root)             // inspect_binary / write_binary（二进制读写，见 binary.go）
	registerBinaryRETools(r, root)           // binary_strings/find/patch/info/hash/entropy（二进制正则，见 binary_re.go）
	registerDebugTools(r, root)              // debug_start/stop/breakpoint/continue/next/step_in/step_out/stack/variables/evaluate/status（见 debugtools.go）
}

// ─── 辅助 ────────────────────────────────────────────────────

type props = map[string]any

func strProp(desc string) map[string]any  { return map[string]any{"type": "string", "description": desc} }
func boolProp(desc string) map[string]any { return map[string]any{"type": "boolean", "description": desc} }
func intProp(desc string) map[string]any  { return map[string]any{"type": "integer", "description": desc} }

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
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	full = filepath.Clean(full)
	// 先查 primary root
	rel, err := filepath.Rel(root, full)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return full, nil
	}
	// 再查其他工作区根目录（多根工作区支持）
	for _, wr := range WorkspaceRoots {
		if wr == root {
			continue
		}
		rel, err := filepath.Rel(wr, full)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return full, nil
		}
	}
	return "", fmt.Errorf("路径越界（不允许访问工作区外）: %s", p)
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
