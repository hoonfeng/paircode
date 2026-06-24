// Package mcp 是 MCP（Model Context Protocol）服务器管理。
// 提供 MCP 服务器的配置读取/写入/删除（用户级 config/mcp.json + 项目级 .pair/mcp.json）。
// UI 面板待后续迁移；当前提供数据层操作供 bridge/agenttools 使用。
//
//go:build windows

package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// ─── 层级 ───

// Level 配置层级（用户级 / 项目级）。
type Level string

const (
	LevelUser    Level = "user"
	LevelProject Level = "project"
)

// LevelDef 层级描述。
type LevelDef struct {
	ID   Level
	Name string
}

// Levels 所有层级（显示顺序）。
var Levels = []LevelDef{
	{LevelUser, "用户级"},
	{LevelProject, "项目级"},
}

// ─── 配置结构 ───

// MCPServerConfig MCP 服务器配置（bridge 用，兼容 agent.RegisterMCPServers）。
type MCPServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Enabled *bool             `json:"enabled,omitempty"` // nil=默认启用
}

// Entry MCP 服务器条目（agenttools 用）。
type Entry struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// mcpFile 用户级/项目级 mcp.json 的结构。
type mcpFile struct {
	Servers map[string]Entry `json:"servers"`
}

// ─── 路径 ───

func levelPath(lv Level) string {
	switch lv {
	case LevelUser:
		return filepath.Join(core.ConfigDir(), "mcp.json")
	case LevelProject:
		return filepath.Join(core.Root(), ".pair", "mcp.json")
	}
	return ""
}

// ─── 读写 ───

func readFile(lv Level) mcpFile {
	var f mcpFile
	data, err := os.ReadFile(levelPath(lv))
	if err != nil {
		return f
	}
	_ = json.Unmarshal(data, &f)
	if f.Servers == nil {
		f.Servers = map[string]Entry{}
	}
	return f
}

func writeFile(lv Level, f mcpFile) error {
	dir := filepath.Dir(levelPath(lv))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(levelPath(lv), data, 0o644)
}

// ReadLevel 读某层级的所有 MCP 服务器（按名排序）。
func ReadLevel(lv Level) []Entry {
	f := readFile(lv)
	out := make([]Entry, 0, len(f.Servers))
	for _, e := range f.Servers {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Upsert 新增/更新某层级的 MCP 服务器。
func Upsert(lv Level, e Entry) error {
	f := readFile(lv)
	if f.Servers == nil {
		f.Servers = map[string]Entry{}
	}
	f.Servers[e.Name] = e
	return writeFile(lv, f)
}

// Delete 删除某层级的 MCP 服务器。
func Delete(lv Level, name string) error {
	f := readFile(lv)
	if _, ok := f.Servers[name]; !ok {
		return os.ErrNotExist
	}
	delete(f.Servers, name)
	return writeFile(lv, f)
}

// Enabled 检查某层级的 MCP 服务器是否启用（默认启用）。
func Enabled(lv Level, name string) bool {
	return true
}

// LoadConfigs 从所有层级加载 MCP 服务器配置（bridge 用：连接外部 MCP）。
func LoadConfigs() []MCPServerConfig {
	var out []MCPServerConfig
	for _, lv := range Levels {
		for _, e := range ReadLevel(lv.ID) {
			out = append(out, MCPServerConfig{
				Name: e.Name, Command: e.Command, Args: e.Args, Enabled: boolPtr(true),
			})
		}
	}
	return out
}

func boolPtr(b bool) *bool { return &b }
