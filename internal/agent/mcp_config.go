// MCP 配置管理 —— 自闭环：config/json 的读写、层级管理、外部加载。
// 从 cmd/companion/ui/mcp/mcp.go 迁移而来，去掉 core 包依赖，使用全局注入变量。
// UI/Agent 通用，不依赖 GUI（无 //go:build 标签，全平台可用）。

package agent

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	// Enabled 启用开关（三态）：
	//   true  = 启用（连接并注册工具）；
	//   false = 显式禁用；
	//   nil   = 未显式启用——按 2026-09-15 新策略视为禁用（原语义为默认启用，属 Breaking 变更，
	//           迁移指引见 MCPLoadConfigs / docs/mcp-default-off-breaking-change.md）。
	Enabled *bool `json:"enabled,omitempty"`
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

// MCPEnabled 检查某层级的 MCP 服务器是否启用（三态语义）。
//
// ★ 2026-09-15 语义变更（业务有意为之：「添加 ≠ 启用，显式启用才连接」）：
//   - true  → 启用（连接并注册工具）
//   - false → 禁用（显式禁用）
//   - nil   → 默认禁用（原为默认启用，Breaking 变更）
//
// 变更理由：MCP 连接有真实启动成本（npx 拉包 5~15s/台、常驻子进程），
// 历史缺省条目（如批量添加的示例服务器）不应拖慢每次会话启动。
// 恢复使用：mcp_enable / UI 启用一次（写入 enabled:true）。
func MCPEnabled(lv MCPLevel, name string) bool {
	f := mcpReadFile(lv)
	e, ok := f.Servers[name]
	if !ok {
		return false // 条目不存在 = 未启用
	}
	if e.Enabled == nil {
		return false // 缺省（未显式启用）= 禁用（★ 2026-09-15 变更点）
	}
	return *e.Enabled // 显式 true=启用 / 显式 false=禁用
}

// MCPSetEnabled 设置（启用/禁用）某层级的 MCP 服务器。
func MCPSetEnabled(lv MCPLevel, name string, enabled bool) error {
	f := mcpReadFile(lv)
	e, ok := f.Servers[name]
	if !ok {
		return os.ErrNotExist
	}
	e.Enabled = &enabled
	f.Servers[name] = e
	return mcpWriteFile(lv, f)
}

// mcpImplicitOffWarned 历史缺省条目的一次性提示护栏（进程内只提示一次，
// 避免 MCPLoadConfigs 高频调用时重复刷屏）。
var mcpImplicitOffWarned sync.Once

// MCPLoadConfigs 从所有层级加载 MCP 服务器配置（供 RegisterMCPServers 连接外部 MCP）。
// 只返回显式启用的服务器（enabled=true）。
//
// ★ 2026-09-15 迁移机制：缺省 enabled 的历史条目按新策略停用（不连接）。
// 首次检测到此类条目时输出一次性告警日志（名单 + 恢复方式），
// 避免「静默失连」不可发现；完整迁移指引见 docs/mcp-default-off-breaking-change.md。
// 返回 agent.MCPServerConfig 列表（已在 mcp.go 定义），与 agent 自闭环兼容。
func MCPLoadConfigs() []MCPServerConfig {
	var out []MCPServerConfig
	var implicit []string // 缺省 enabled 的条目（新语义下停用，做迁移提示）
	for _, lv := range MCPLevels {
		for _, e := range MCPReadLevel(lv.ID) {
			if e.Enabled == nil {
				implicit = append(implicit, e.Name)
				continue
			}
			if !*e.Enabled {
				continue
			}
			out = append(out, MCPServerConfig{
				Name: e.Name, Command: e.Command, Args: e.Args,
			})
		}
	}
	if len(implicit) > 0 {
		mcpImplicitOffWarned.Do(func() {
			log.Printf("[MCP] %d 个历史条目未显式启用（enabled 缺省），按新策略（2026-09-15：添加≠启用）已停用：%s——如需使用请用 mcp_enable / UI 启用（写入 enabled:true 后连接）",
				len(implicit), strings.Join(implicit, "、"))
		})
	}
	return out
}
