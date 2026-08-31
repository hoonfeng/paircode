package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestBridgeRepoCoexist 并存与生效方切换（2026-08-31，取消同名工具覆盖）：
// DSH 桥插件（node-bridge 轨）与 repo 移植版 goja 插件（.pair/plugins/agent-teams）
// 同名工具不再互相停用/让位——两版并存，默认 repo 优先生效，桥工具挂起；
// 插件面板可切换生效方；repo 版停用后挂起的桥工具自动接管。
func TestBridgeRepoCoexist(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	origFolders := core.Folders
	core.Folders = []string{dir} // nodeBridgeDir()/patch 读取指向临时工作区
	defer func() { core.Folders = origFolders }()

	// ── repo 移植版（goja 轨，插件名 agent-teams）先注册工具 ──
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

	// ── 假桥（无 node 进程）：已装载 DSH 包 + 注册同名工具 ──
	pkg := "@nanmicoder/dsh-agent-teams"
	owner := "node-bridge:" + pkg
	dshTool := &Tool{
		Name:        "agent_teams_create",
		Description: "DSH 版",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return "dsh", nil
		},
	}
	b := &nodeBridge{
		ready:     true,
		tools:     map[string]*Tool{dshTool.Name: dshTool},
		toolOwner: map[string]string{dshTool.Name: owner},
		specs:     map[string]string{pkg: pkg + "@0.1.14"},
	}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	// ① 桥工具注册冲突 → 不停用 repo 版，记录并存（active=repo）
	if err := host.Context().forPlugin(owner).RegisterTool(dshTool); err == nil {
		t.Fatalf("同名注册应冲突（claimTool 严格语义）")
	}
	if !b.shadowConflictingTool(host, owner, dshTool) {
		t.Fatalf("应记录并存关系（repo 侧占用方为 goja 插件）")
	}
	if host.State("agent-teams") != PluginRunning {
		t.Fatalf("取消自动覆盖后 repo 版不应被停用（当前 %s）", host.State("agent-teams"))
	}
	if got := host.PluginToolOwners()["agent_teams_create"]; got != "agent-teams" {
		t.Fatalf("默认应 repo 优先生效，实际归属 %q", got)
	}

	// ② Inspect 两源合并 + 双侧冲突标注
	recs := host.Inspect()
	var repoRec, bridgeRec *PluginRecord
	for i := range recs {
		switch recs[i].Name {
		case "agent-teams":
			repoRec = &recs[i]
		case pkg:
			bridgeRec = &recs[i]
		}
	}
	if repoRec == nil || bridgeRec == nil {
		names := make([]string, 0, len(recs))
		for _, r := range recs {
			names = append(names, r.Name)
		}
		t.Fatalf("Inspect 应同时含 repo 版与桥插件，实际: %v", names)
	}
	if bridgeRec.Source != PluginSourceBridge || bridgeRec.State != "running" || bridgeRec.Version != "0.1.14" {
		t.Fatalf("桥记录字段不符: source=%s state=%s version=%s", bridgeRec.Source, bridgeRec.State, bridgeRec.Version)
	}
	if len(bridgeRec.Tools) != 1 || bridgeRec.Tools[0] != "agent_teams_create" {
		t.Fatalf("桥记录应含其工具，实际 %v", bridgeRec.Tools)
	}
	if len(repoRec.Conflicts) != 1 || repoRec.Conflicts[0].Active != ToolImplRepo || repoRec.Conflicts[0].Bridge != pkg {
		t.Fatalf("repo 记录冲突标注不符: %+v", repoRec.Conflicts)
	}
	if len(bridgeRec.Conflicts) != 1 || bridgeRec.Conflicts[0].Repo != "agent-teams" {
		t.Fatalf("桥记录冲突标注不符: %+v", bridgeRec.Conflicts)
	}

	// ③ 切到 DSH 版：Registry 生效实现换成桥工具，两侧插件状态不变
	if err := SetBridgeToolPreference(host, "agent_teams_create", ToolImplBridge); err != nil {
		t.Fatalf("切到 bridge 失败: %v", err)
	}
	if got := host.PluginToolOwners()["agent_teams_create"]; got != owner {
		t.Fatalf("切换后归属应为 %s，实际 %q", owner, got)
	}
	out, err := reg.Execute(context.Background(), "agent_teams_create", "{}")
	if err != nil || out != "dsh" {
		t.Fatalf("切换后应执行 DSH 实现，得到 %q err=%v", out, err)
	}
	if host.State("agent-teams") != PluginRunning {
		t.Fatalf("切换生效方不应改动 repo 插件运行状态")
	}

	// ④ 切回 repo 版：恢复 goja 实现
	if err := SetBridgeToolPreference(host, "agent_teams_create", ToolImplRepo); err != nil {
		t.Fatalf("切回 repo 失败: %v", err)
	}
	if got := host.PluginToolOwners()["agent_teams_create"]; got != "agent-teams" {
		t.Fatalf("切回后归属应为 agent-teams，实际 %q", got)
	}
	out, err = reg.Execute(context.Background(), "agent_teams_create", "{}")
	if err != nil {
		t.Fatalf("切回后执行失败: %v", err)
	}
	if out == "dsh" {
		t.Fatalf("切回后不应仍是 DSH 实现")
	}

	// ⑤ repo 版停用 → 挂起的桥工具自动接管
	if err := host.Unload("agent-teams"); err != nil {
		t.Fatalf("Unload repo 版失败: %v", err)
	}
	if got := host.PluginToolOwners()["agent_teams_create"]; got != owner {
		t.Fatalf("repo 停用后桥工具应接管，实际归属 %q", got)
	}
	out, err = reg.Execute(context.Background(), "agent_teams_create", "{}")
	if err != nil || out != "dsh" {
		t.Fatalf("repo 停用后应执行 DSH 实现，得到 %q err=%v", out, err)
	}
}

// TestBridgeCoexistNonRepoConflict 负例：占用方非 repo 侧 goja 插件（宿主内置
// 工具 / 另一桥插件）→ 不记录并存关系（保持既有严格语义，由调用方报错）。
func TestBridgeCoexistNonRepoConflict(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	b := &nodeBridge{tools: map[string]*Tool{}, toolOwner: map[string]string{}}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	// 无任何占用方（工具未注册）→ 非并存关系
	tool := &Tool{Name: "orphan_tool", Description: "x", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", nil }}
	if b.shadowConflictingTool(host, "node-bridge:pkg-a", tool) {
		t.Fatalf("无 repo 占用方时不应记录并存")
	}
	// 占用方是另一个桥插件 → 非并存关系
	other := &Tool{Name: "bridge_tool", Description: "x", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", nil }}
	if err := host.Context().forPlugin("node-bridge:pkg-b").RegisterTool(other); err != nil {
		t.Fatalf("首个桥插件注册应成功: %v", err)
	}
	if b.shadowConflictingTool(host, "node-bridge:pkg-a", other) {
		t.Fatalf("占用方为桥插件时不应记录并存")
	}
	if len(bridgeToolConflicts()) != 0 {
		t.Fatalf("不应产生并存记录，实际 %+v", bridgeToolConflicts())
	}
	// 切换不存在的并存记录 → 明确报错
	if err := SetBridgeToolPreference(host, "bridge_tool", ToolImplBridge); err == nil ||
		!strings.Contains(err.Error(), "无同名并存记录") {
		t.Fatalf("无并存记录时切换应报错，实际 %v", err)
	}
}

// TestSplitNpmSpec npm spec 拆分（scope 包名含 @，取最后一个 @ 之后为版本）。
func TestSplitNpmSpec(t *testing.T) {
	cases := []struct{ spec, pkg, ver string }{
		{"@nanmicoder/dsh-agent-teams@0.1.14", "@nanmicoder/dsh-agent-teams", "0.1.14"},
		{"cordis-plugin-android@0.0.7", "cordis-plugin-android", "0.0.7"},
		{"plain-pkg", "plain-pkg", ""},
		{"@scope/pkg", "@scope/pkg", ""},
	}
	for _, c := range cases {
		pkg, ver := splitNpmSpec(c.spec)
		if pkg != c.pkg || ver != c.ver {
			t.Errorf("%s → (%q,%q)，期望 (%q,%q)", c.spec, pkg, ver, c.pkg, c.ver)
		}
	}
}
