// ═══════════════════════════════════════════════════════════════
// ★ 2026-08-16 自根 pkg/toolbin 内嵌（插件自包含；改协议请全局同步 17 个插件）。
// registry.go — 插件独立二进制的轻量工具注册表
//
// ★ 2026-08-16 第四轮：工具实现从 internal/agent 迁出（每组一个 impl 包，
//
//	放各插件目录），独立二进制不再 import agent——注册表/辅助函数下沉本包。
//	本包为二进制侧自持的轻量实现（与宿主 Registry 类型无关，协议即 JSON）。
//
// ═══════════════════════════════════════════════════════════════
package toolbin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// ToolHandler 工具执行体（与宿主一致）。
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// Tool 一个已注册工具（名/描述/参数 Schema/执行体 + 元信息）。
type Tool struct {
	Name             string
	Description      string         // 简短描述（传给 LLM function-calling）
	UsageGuide       string         // 详细使用指导
	Category         string         // 工具分类（如 "git", "file", "web"）
	Parameters       map[string]any // JSON Schema
	Handler          ToolHandler
	SystemTool       bool // 系统内部工具，不暴露给 LLM
	ReadOnly         bool // 只读（不改文件系统）
	RequiresApproval bool // 写类工具
	Enabled          bool // 是否启用（默认 true）
}

// Registry 工具注册表（并发安全）。
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
	order []string // 保持注册顺序
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
		if !t.Enabled {
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

// IsEnabled 返回工具是否已启用。
func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return !ok || t.Enabled
}

// ToolMeta 工具元信息（协议回显/调试用）。
type ToolMeta struct {
	Name        string
	Description string
	ReadOnly    bool
	Enabled     bool
}

// AllToolMeta 全部工具元信息（注册顺序）。
func (r *Registry) AllToolMeta() []ToolMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolMeta, 0, len(r.order))
	for _, n := range r.order {
		t := r.tools[n]
		out = append(out, ToolMeta{Name: t.Name, Description: t.Description, ReadOnly: t.ReadOnly, Enabled: t.Enabled})
	}
	return out
}

// ToolNames 工具名清单（注册顺序）。
func (r *Registry) ToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.order...)
}

// EnabledNames 已启用工具名清单（注册顺序，禁用过滤）。
func (r *Registry) EnabledNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.order))
	for _, n := range r.order {
		if t := r.tools[n]; t.Enabled {
			out = append(out, n)
		}
	}
	return out
}

// Execute 执行工具（JSON 参数 → 文本结果）。
func (r *Registry) Execute(ctx context.Context, name, argsJSON string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	var args map[string]any
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("参数解析失败: %v", err)
		}
	}
	return t.Handler(ctx, args)
}

// ─── JSON Schema 辅助（与宿主一致）──────────────────────────────

// props JSON Schema properties 简写。
type Props map[string]any

func StrProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}
func BoolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}
func IntProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
func ArrProp(items any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

// objSchema 构建对象 Schema（properties + required）。
func ObjSchema(properties Props, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any(properties),
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// ─── 参数提取辅助 ──────────────────────────────────────────────

// argStr 取字符串参数（缺省返回 def）。
func ArgStr(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// argBool 取布尔参数。
func ArgBool(args map[string]any, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// argInt 取整数参数（支持 float64/string，缺省返回 def）。
func ArgInt(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			var out int
			if _, err := fmt.Sscanf(n, "%d", &out); err == nil {
				return out
			}
		}
	}
	return def
}

// argStrSlice 取字符串数组参数。
func ArgStrSlice(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	var out []string
	switch arr := v.(type) {
	case []any:
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	case []string:
		out = arr
	}
	return out
}

// ToolShortDesc 工具简短描述（截断，协议回显用）。
func ToolShortDesc(t *Tool) string {
	if t == nil || t.Description == "" {
		return ""
	}
	d := t.Description
	if len([]rune(d)) > 60 {
		runes := []rune(d)
		return string(runes[:60]) + "…"
	}
	return d
}

// ContainsStr 判断字符串切片是否含目标。
func ContainsStr(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

// HasPrefixAny 判断 s 是否以 prefixes 任一开头。
func HasPrefixAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
