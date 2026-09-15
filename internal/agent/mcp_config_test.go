// mcp_config_test.go — MCP 启用语义（★ 2026-09-15 Breaking 变更）与迁移行为单测。
//
// 覆盖点：
//  1. enabled 三态：nil=默认禁用（新语义）/ false=禁用 / true=启用（防语义回归）；
//  2. MCPLoadConfigs 只返回显式启用条目（缺省条目不连接）；
//  3. 迁移路径：对历史缺省条目执行 MCPSetEnabled(true) 后即恢复连接。

package agent

import (
	"path/filepath"
	"testing"
)

// setupMCPConfigDir 为单个测试隔离用户级/项目级 mcp.json 到临时目录（测试后恢复）。
func setupMCPConfigDir(t *testing.T) {
	t.Helper()
	oldProjectPath, oldUserPath := MCPProjectConfigPath, MCPUserConfigPath
	t.Cleanup(func() { MCPProjectConfigPath, MCPUserConfigPath = oldProjectPath, oldUserPath })
	user := t.TempDir()
	MCPUserConfigPath = filepath.Join(user, "mcp.json")
	MCPProjectConfigPath = filepath.Join(user, "proj", ".pair", "mcp.json")
}

// TestMCPEnabledTriState enabled 三态语义：nil=默认禁用（新）/ false=禁用 / true=启用。
func TestMCPEnabledTriState(t *testing.T) {
	setupMCPConfigDir(t)

	tr, fa := true, false
	if err := MCPUpsert(MCPLevelUser, MCPEntry{Name: "nil-entry", Command: "echo"}); err != nil {
		t.Fatalf("预置 nil-entry 失败: %v", err)
	}
	if err := MCPUpsert(MCPLevelUser, MCPEntry{Name: "off-entry", Command: "echo", Enabled: &fa}); err != nil {
		t.Fatalf("预置 off-entry 失败: %v", err)
	}
	if err := MCPUpsert(MCPLevelUser, MCPEntry{Name: "on-entry", Command: "echo", Enabled: &tr}); err != nil {
		t.Fatalf("预置 on-entry 失败: %v", err)
	}

	if MCPEnabled(MCPLevelUser, "nil-entry") {
		t.Error("nil（未显式启用）应为禁用——★2026-09-15 语义变更点")
	}
	if MCPEnabled(MCPLevelUser, "off-entry") {
		t.Error("显式 false 应为禁用")
	}
	if !MCPEnabled(MCPLevelUser, "on-entry") {
		t.Error("显式 true 应为启用")
	}
	if MCPEnabled(MCPLevelUser, "missing") {
		t.Error("不存在的条目应为禁用")
	}
}

// TestMCPLoadConfigsOnlyExplicitEnabled 连接列表只含显式启用条目；
// 缺省条目在执行 MCPSetEnabled(true)（迁移动作）后恢复。
func TestMCPLoadConfigsOnlyExplicitEnabled(t *testing.T) {
	setupMCPConfigDir(t)

	tr := true
	if err := MCPUpsert(MCPLevelUser, MCPEntry{Name: "legacy-entry", Command: "echo"}); err != nil {
		t.Fatalf("预置 legacy-entry 失败: %v", err)
	}
	if err := MCPUpsert(MCPLevelUser, MCPEntry{Name: "on-entry", Command: "echo", Enabled: &tr}); err != nil {
		t.Fatalf("预置 on-entry 失败: %v", err)
	}

	cfgs := MCPLoadConfigs()
	if len(cfgs) != 1 || cfgs[0].Name != "on-entry" {
		t.Fatalf("MCPLoadConfigs = %+v，期望仅 on-entry（legacy-entry 缺省 enabled 不连接）", cfgs)
	}

	// 迁移路径：显式启用后立即恢复连接
	if err := MCPSetEnabled(MCPLevelUser, "legacy-entry", true); err != nil {
		t.Fatalf("迁移（显式启用）失败: %v", err)
	}
	cfgs = MCPLoadConfigs()
	if len(cfgs) != 2 {
		t.Fatalf("显式启用后 MCPLoadConfigs 应含 2 条，实际 %d：%+v", len(cfgs), cfgs)
	}
}
