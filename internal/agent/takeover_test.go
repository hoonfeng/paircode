package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBridgeToolTakeover 遗留处置 F2 自动化（2026-08-29）：
// DSH 插件（node-bridge 桥，cordis4 轨）工具与 repo 移植版 goja 插件
// （.pair/plugins/agent-teams）同名时，takeoverConflictingPlugin 自动停用
// 移植版并接管注册——把文档化的手动步骤自动化；非同源冲突保持严格拒绝。
func TestBridgeToolTakeover(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)

	// 装载「repo 移植版」插件（goja 轨，命名 agent-teams），注册一个同名工具
	patch := filepath.Join(dir, "cordis.patch.json")
	content := `{
  "plugins": [
    {
      "purpose": "repo 移植版 agent-teams",
      "code": "return { name: 'agent-teams', apply(ctx, config) { ctx.tools.register({ name: 'agent_teams_create', description: '移植版', execute: () => ({ text: 'port' }) }) } }"
    }
  ]
}`
	if err := os.WriteFile(patch, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := host.LoadCordisPatch(patch); err != nil {
		t.Fatalf("LoadCordisPatch: %v", err)
	}
	if host.State("agent-teams") != PluginRunning {
		t.Fatalf("agent-teams 应 running")
	}
	if !host.HasPluginTool("agent_teams_create") {
		t.Fatalf("移植版工具应已注册")
	}

	b := &nodeBridge{}
	dshtool := &Tool{
		Name:        "agent_teams_create",
		Description: "DSH 版",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return "dsh", nil
		},
	}
	owner := "node-bridge:@nanmicoder/dsh-agent-teams"

	// ① 直接注册 → claimTool 冲突（既有严格拒绝语义仍在）
	if err := host.Context().forPlugin(owner).RegisterTool(dshtool); err == nil {
		t.Fatalf("同名直接注册应冲突")
	}

	// ② 接管：自动停用移植版 + 重试注册成功
	if err := b.takeoverConflictingPlugin(host, owner, dshtool); err != nil {
		t.Fatalf("接管应成功: %v", err)
	}
	if host.State("agent-teams") == PluginRunning {
		t.Fatalf("移植版应已被停用")
	}
	owners := host.PluginToolOwners()
	if owners["agent_teams_create"] != owner {
		t.Fatalf("工具归属应转给 %s，实际 %q", owner, owners["agent_teams_create"])
	}

	// ③ 接管后桥插件再注册无冲突（归属已转）
	if err := host.Context().forPlugin(owner).RegisterTool(dshtool); err != nil {
		t.Fatalf("接管后二次注册应成功: %v", err)
	}

	// ④ 负例：非 dsh- 命名的桥插件不做接管
	bogus := &Tool{Name: "other_tool", Description: "x", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", nil }}
	if err := host.Context().forPlugin("agent-teams").RegisterTool(bogus); err != nil {
		t.Fatalf("占用方已停用，注册应成功: %v", err) // 占位：确保 bogus 无归属者
	}
	if err := b.takeoverConflictingPlugin(host, "node-bridge:some-other-pkg", bogus); err == nil {
		t.Fatalf("非 dsh- 命名应拒绝接管")
	} else if !strings.Contains(err.Error(), "非 dsh- 同源命名") {
		t.Fatalf("拒绝信息不符: %v", err)
	}

	// ⑤ 负例：占用方已是桥插件（node-bridge: 前缀）→ 拒绝接管
	if err := b.takeoverConflictingPlugin(host, owner, dshtool); err == nil {
		t.Fatalf("占用方为桥插件时应拒绝接管")
	} else if !strings.Contains(err.Error(), "非可接管的 goja 插件") {
		t.Fatalf("拒绝信息不符: %v", err)
	}
}

// TestBridgeToolTakeoverPkgShort 桥包短名解析：npm scope + dsh- 前缀剥离。
func TestBridgeToolTakeoverPkgShort(t *testing.T) {
	cases := []struct{ owner, want string }{
		{"node-bridge:@nanmicoder/dsh-agent-teams", "agent-teams"},
		{"node-bridge:dsh-tool-x", "tool-x"},
		{"node-bridge:plain", "plain"}, // 无 dsh- 前缀 → 非同源命名
	}
	for _, c := range cases {
		pkg := c.owner
		if i := strings.LastIndex(pkg, ":"); i >= 0 {
			pkg = pkg[i+1:]
		}
		if i := strings.LastIndex(pkg, "/"); i >= 0 {
			pkg = pkg[i+1:]
		}
		got := strings.TrimPrefix(pkg, "dsh-")
		if got == c.want {
			t.Logf("ok: %s → %s", c.owner, got)
		} else {
			t.Errorf("%s → %q，期望 %q", c.owner, got, c.want)
		}
	}
}
