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
	return t.Handler(ctx, args)
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
		Name:             "edit_file",
		Description:      "把文件中唯一一处 old_string 替换为 new_string（old_string 必须在文件中恰好出现一次）。",
		Parameters:       objSchema(props{"path": strProp("文件路径"), "old_string": strProp("待替换的原文（须唯一）"), "new_string": strProp("替换后的新文")}, "path", "old_string", "new_string"),
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
			old, neu := argStr(args, "old_string"), argStr(args, "new_string")
			if old == "" {
				return "", fmt.Errorf("old_string 不能为空")
			}
			switch strings.Count(string(data), old) {
			case 0:
				return "", fmt.Errorf("未找到 old_string，无法定位编辑点")
			case 1:
				// ok
			default:
				return "", fmt.Errorf("old_string 出现多次，不唯一——请提供更长的上下文")
			}
			out := strings.Replace(string(data), old, neu, 1)
			if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
				return "", err
			}
			return "已编辑 " + argStr(args, "path"), nil
		},
	})

	r.Register(&Tool{
		Name: "multi_edit",
		Description: "对一个文件按顺序应用多处替换（edits：每项 old_string→new_string；每个 old_string 须在「应用到该步时」的内容中恰好出现一次）。" +
			"比多次 edit_file 高效、原子（任一步失败则全部不写）。",
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
							"old_string": strProp("待替换的原文（须唯一）"),
							"new_string": strProp("替换后的新文"),
						},
						"required": []string{"old_string", "new_string"},
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
				if old == "" {
					return "", fmt.Errorf("edits[%d] old_string 不能为空", i)
				}
				switch strings.Count(content, old) {
				case 0:
					return "", fmt.Errorf("edits[%d] 未找到 old_string", i)
				case 1:
					// ok
				default:
					return "", fmt.Errorf("edits[%d] old_string 不唯一——请提供更长上下文", i)
				}
				content = strings.Replace(content, old, neu, 1)
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

	registerSearchTools(r, root) // search_content / search_files（见 search.go）
	registerGitTools(r, root)    // git_status / git_diff / git_log（只读，见 git.go）
	registerWebTools(r)          // web_fetch（联网读，见 web.go）
	registerPlanTool(r)          // update_plan（任务清单，见 plan.go）
	registerShellTools(r, root)  // run_background / read_output / kill_process（后台命令，见 shell.go）
	registerMemoryTools(r, root) // memory_write/read/list/search（跨会话记忆，见 memory.go）
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
func resolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	full = filepath.Clean(full)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("路径越界（不允许访问工作区外）: %s", p)
	}
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
