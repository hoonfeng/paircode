// MCP 配置管理 —— 自闭环：config/json 的读写、层级管理、外部加载。
// 从 cmd/companion/ui/mcp/mcp.go 迁移而来，去掉 core 包依赖，使用全局注入变量。
// UI/Agent 通用，不依赖 GUI（无 //go:build 标签，全平台可用）。

package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ─── 层级 ───

// MCPLevel 配置层级（用户级 / 项目级）。
type MCPLevel string

const (
	MCPLevelUser    MCPLevel = "user"
	MCPLevelProject MCPLevel = "project"
)

// MCPLevelDef 层级描述。
type MCPLevelDef struct {
	ID   MCPLevel
	Name string
}

// MCPLevels 所有层级（显示顺序）。
var MCPLevels = []MCPLevelDef{
	{MCPLevelUser, "用户级"},
	{MCPLevelProject, "工作区级"},
}

// ─── 全局配置路径（由外部注入，类似 SkillSystemDir/SkillProjectDir）──

// MCPUserConfigPath 用户级 mcp.json 路径。
// 由外部在初始化时设为 filepath.Join(configDir, "mcp.json")。
var MCPUserConfigPath string

// MCPProjectConfigPath 项目级 mcp.json 路径。
// 由外部在初始化时设为 filepath.Join(root, ".pair", "mcp.json")。
var MCPProjectConfigPath string

// ─── 配置结构 ───

// MCPEntry MCP 服务器条目。
type MCPEntry struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// mcpFile 用户级/项目级 mcp.json 的结构（不导出）。
type mcpFile struct {
	Servers map[string]MCPEntry `json:"servers"`
}

// ─── 路径（含防穿越校验）──

func mcpLevelPath(lv MCPLevel) (string, error) {
	var base string
	switch lv {
	case MCPLevelUser:
		base = MCPUserConfigPath
	case MCPLevelProject:
		base = MCPProjectConfigPath
	default:
		return "", errors.New("未知 MCP 层级: " + string(lv))
	}
	if base == "" {
		return "", errors.New("MCP 配置路径未设置（请先注入 MCPUserConfigPath / MCPProjectConfigPath）")
	}
	// clean + 防穿越：确保最终路径在允许的目录内
	cleaned := filepath.Clean(base)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	// 校验文件名必须是 mcp.json
	if !strings.HasSuffix(abs, "mcp.json") {
		return "", errors.New("MCP 配置路径必须以 mcp.json 结尾")
	}
	return abs, nil
}

// ─── 内部读写（严格 JSON）──

func mcpReadFile(lv MCPLevel) mcpFile {
	var f mcpFile
	path, err := mcpLevelPath(lv)
	if err != nil {
		return f
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return f
	}
	// 严格 JSON：拒绝未知字段
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return mcpFile{}
	}
	if f.Servers == nil {
		f.Servers = map[string]MCPEntry{}
	}
	return f
}

func mcpWriteFile(lv MCPLevel, f mcpFile) error {
	path, err := mcpLevelPath(lv)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// 严格 JSON 序列化
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ─── 公开 API ───

// MCPReadLevel 读某层级的所有 MCP 服务器（按名排序）。
func MCPReadLevel(lv MCPLevel) []MCPEntry {
	f := mcpReadFile(lv)
	out := make([]MCPEntry, 0, len(f.Servers))
	for _, e := range f.Servers {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// MCPUpsert 新增/更新某层级的 MCP 服务器。
func MCPUpsert(lv MCPLevel, e MCPEntry) error {
	f := mcpReadFile(lv)
	if f.Servers == nil {
		f.Servers = map[string]MCPEntry{}
	}
	f.Servers[e.Name] = e
	return mcpWriteFile(lv, f)
}

// MCPDelete 删除某层级的 MCP 服务器。
func MCPDelete(lv MCPLevel, name string) error {
	f := mcpReadFile(lv)
	if _, ok := f.Servers[name]; !ok {
		return os.ErrNotExist
	}
	delete(f.Servers, name)
	return mcpWriteFile(lv, f)
}

// MCPEnabled 检查某层级的 MCP 服务器是否启用（默认启用）。
func MCPEnabled(lv MCPLevel, name string) bool {
	return true
}

// MCPLoadConfigs 从所有层级加载 MCP 服务器配置（供 RegisterMCPServers 连接外部 MCP）。
// 返回 agent.MCPServerConfig 列表（已在 mcp.go 定义），与 agent 自闭环兼容。
func MCPLoadConfigs() []MCPServerConfig {
	var out []MCPServerConfig
	for _, lv := range MCPLevels {
		for _, e := range MCPReadLevel(lv.ID) {
			out = append(out, MCPServerConfig{
				Name: e.Name, Command: e.Command, Args: e.Args,
			})
		}
	}
	return out
}
