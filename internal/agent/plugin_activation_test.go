package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOnDemandActivation 按需激活机制：声明 → 会话未激活时提示段/工具隐藏 → 命令激活后可见。
func TestOnDemandActivation(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	// 插件声明按需（真实 apply 时经 ctx.activation.declare；此处直接调宿主 API）
	DeclareOnDemandPlugin("agent-teams", "agent-teams")
	defer func() {
		ResetConvActivations("")
		onDemandMu.Lock()
		delete(onDemandPlugins, "agent-teams")
		delete(onDemandByCmd, "agent-teams")
		onDemandMu.Unlock()
	}()

	// 插件注册工具 + 提示段（测试 host 无插件上下文，显式填归属名）
	pc := host.Context()
	pc.Tools.Register(&Tool{Name: "agent_teams_create", Description: "测试", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil }})
	if _, err := host.claimTool("agent-teams", "agent_teams_create"); err != nil {
		t.Fatalf("claimTool 失败: %v", err)
	}
	pc.AddSystemPromptSection(&PromptSection{Name: "agent-teams-usage", Order: 117, Plugin: "agent-teams", Text: "When the user asks to run something with AgentTeams ..."})

	// ① 未激活：命令映射有，但提示段/工具对会话隐藏
	if !IsOnDemandPlugin("agent-teams") {
		t.Fatal("agent-teams 应声明为按需插件")
	}
	if m := OnDemandCommandMapping(); m["agent-teams"] != "agent-teams" {
		t.Fatalf("命令映射应有 agent-teams → agent-teams: %+v", m)
	}
	if IsPluginActiveInConv("conv_x", "agent-teams") {
		t.Error("未激活会话不应可见")
	}
	if secs, _ := PluginPromptSections(host, "conv_x"); strings.Contains(secs, "AgentTeams") {
		t.Errorf("未激活会话不应注入提示段: %q", secs)
	}
	// 未开会话（convID 空）同样隐藏
	if secs, _ := PluginPromptSections(host, ""); strings.Contains(secs, "AgentTeams") {
		t.Errorf("convID 空不应注入按需段: %q", secs)
	}

	// ② 命令激活：模拟 /agent-teams 执行
	if got := ActivateByCommand("conv_x", "agent-teams"); got != "agent-teams" {
		t.Fatalf("命令应激活插件，得到 %q", got)
	}
	if !IsPluginActiveInConv("conv_x", "agent-teams") {
		t.Fatal("激活后会话应可见")
	}
	// 其他会话不受影响
	if IsPluginActiveInConv("conv_y", "agent-teams") {
		t.Error("其他会话不应被激活")
	}
	// 提示段注入
	if secs, _ := PluginPromptSections(host, "conv_x"); !strings.Contains(secs, "AgentTeams") {
		t.Errorf("激活会话应注入提示段: %q", secs)
	}
	// 协议说明（激活通知）可取
	if notice := PluginActivationNotice("agent-teams"); !strings.Contains(notice, "AgentTeams") {
		t.Errorf("激活通知应含协议文本: %q", notice)
	}

	// ③ 工具面：未激活会话不合并，激活会话合并
	reg2 := NewRegistry()
	MergePluginToolsForConv(reg2, host, "conv_y")
	if _, ok := reg2.Get("agent_teams_create"); ok {
		t.Error("未激活会话不应合并插件工具")
	}
	reg3 := NewRegistry()
	MergePluginToolsForConv(reg3, host, "conv_x")
	if _, ok := reg3.Get("agent_teams_create"); !ok {
		t.Error("激活会话应合并插件工具")
	}

	// ④ 重置：会话删除不再可见
	ResetConvActivations("conv_x")
	if IsPluginActiveInConv("conv_x", "agent-teams") {
		t.Error("重置后会话不应再可见")
	}
}

// TestOnDemandAlwaysVisibleSection 方案 B：按需插件的 AlwaysVisible 段常驻注入
// （协议/引导段每轮可见），其余段与工具仍按会话激活过滤。
func TestOnDemandAlwaysVisibleSection(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	DeclareOnDemandPlugin("agent-teams", "agent-teams")
	defer func() {
		ResetConvActivations("")
		onDemandMu.Lock()
		delete(onDemandPlugins, "agent-teams")
		delete(onDemandByCmd, "agent-teams")
		onDemandMu.Unlock()
	}()

	pc := host.Context()
	pc.Tools.Register(&Tool{Name: "agent_teams_create", Description: "测试", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil }})
	if _, err := host.claimTool("agent-teams", "agent_teams_create"); err != nil {
		t.Fatalf("claimTool 失败: %v", err)
	}
	// 常驻引导段（方案 B）+ 普通协议段（激活后可见）
	pc.AddSystemPromptSection(&PromptSection{Name: "agent-teams:guide", Order: 116, Plugin: "agent-teams", Text: "GUIDE AgentTeams available; run /agent-teams to activate.", AlwaysVisible: true})
	pc.AddSystemPromptSection(&PromptSection{Name: "agent-teams", Order: 117, Plugin: "agent-teams", Text: "FULLPROTOCOL When the user asks to run something with AgentTeams ..."})

	// 未激活会话：引导段常驻、协议段隐藏、工具隐藏
	secs, _ := PluginPromptSections(host, "conv_x")
	if !strings.Contains(secs, "GUIDE") {
		t.Errorf("未激活会话应常驻注入引导段: %q", secs)
	}
	if strings.Contains(secs, "FULLPROTOCOL") {
		t.Errorf("未激活会话不应注入协议段: %q", secs)
	}
	reg2 := NewRegistry()
	MergePluginToolsForConv(reg2, host, "conv_x")
	if _, ok := reg2.Get("agent_teams_create"); ok {
		t.Error("未激活会话不应合并插件工具")
	}

	// 激活后：协议段 + 工具可见
	ActivatePluginInConv("conv_x", "agent-teams")
	if secs, _ := PluginPromptSections(host, "conv_x"); !strings.Contains(secs, "FULLPROTOCOL") {
		t.Errorf("激活会话应注入协议段: %q", secs)
	}
	reg3 := NewRegistry()
	MergePluginToolsForConv(reg3, host, "conv_x")
	if _, ok := reg3.Get("agent_teams_create"); !ok {
		t.Error("激活会话应合并插件工具")
	}
}

// TestConvToolsetWhitelistOnDemandRelease 按需激活 × 工具集白名单交互
// （2026-09-12 修复）：on-demand 插件（agent-teams）经 /命令 激活后，其工具
// 应豁免会话工具集白名单收敛（「激活 → 工具立即可见」声明语义）；未激活的
// 按需插件工具与未声明进工具集的非按需插件工具仍受白名单约束。
func TestConvToolsetWhitelistOnDemandRelease(t *testing.T) {
	// 隔离工具集目录：写一个「不声明任何插件」的瘦身集合（agent-teams 不在白名单）
	// ★ cleanup 恢复 testGlobalToolsetDir（包级隔离目录）——不能恢复 ""：
	//   空串会让 globalToolsetDir() 回落仓库根 .pair/toolsets（测试进程
	//   InstallDir()=getwd），真实预设文件存在 → hasWorkspaceToolsets()=true
	//   → 后续测试插件工具被可见性收敛误禁用（main_test.go 注释同述）。
	tsDir := t.TempDir()
	SetGlobalToolsetDirForTest(tsDir)
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	ts := Toolset{Name: presetNameDefault, Description: "测试瘦身集合"}
	data, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("序列化工具集失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tsDir, presetNameDefault+".json"), data, 0o644); err != nil {
		t.Fatalf("写工具集失败: %v", err)
	}

	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	DeclareOnDemandPlugin("agent-teams", "agent-teams")
	defer func() {
		ResetConvActivations("")
		onDemandMu.Lock()
		delete(onDemandPlugins, "agent-teams")
		delete(onDemandByCmd, "agent-teams")
		onDemandMu.Unlock()
	}()

	registerTestTool := func(plugin, name string) {
		pc := host.Context()
		pc.Tools.Register(&Tool{Name: name, Description: "测试", Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil }})
		if _, err := host.claimTool(plugin, name); err != nil {
			t.Fatalf("claimTool %s 失败: %v", name, err)
		}
	}
	registerTestTool("agent-teams", "agent_teams_create")
	registerTestTool("marketplace", "marketplace_search") // 非按需插件对照

	// ① 未激活会话（conv_y）：按需插件工具不入 reg（merge 过滤），天然隐藏
	reg2 := NewRegistry()
	RegisterDefaultTools(reg2, root)
	MergePluginToolsForConv(reg2, host, "conv_y")
	ApplyConvToolsetWhitelist(host, reg2, "conv_y", "")
	if _, ok := reg2.Get("agent_teams_create"); ok {
		t.Error("未激活会话不应合并按需插件工具")
	}

	// ② 激活会话（conv_x）：merge 放行 → 白名单收敛（瘦身集合未声明）→
	//    修复后按需激活放行 → 工具立即可见
	ActivatePluginInConv("conv_x", "agent-teams")
	reg3 := NewRegistry()
	RegisterDefaultTools(reg3, root)
	MergePluginToolsForConv(reg3, host, "conv_x")
	ApplyConvToolsetWhitelist(host, reg3, "conv_x", "")
	if !reg3.IsEnabled("agent_teams_create") {
		t.Error("激活会话的按需插件工具应豁免工具集白名单（立即可见）")
	}
	// 同一会话中未声明进工具集的非按需插件工具仍禁用（『装载 ≠ 可用』不变）
	if reg3.IsEnabled("marketplace_search") {
		t.Error("非按需插件未声明进工具集应保持禁用")
	}

	// ③ 重置激活：工具再次隐藏（工具仍并入 reg 但被白名单收敛禁用）
	ResetConvActivations("conv_x")
	reg4 := NewRegistry()
	RegisterDefaultTools(reg4, root)
	MergePluginToolsForConv(reg4, host, "conv_x") // 已激活的工具在 reg 里（Reset 后 IsPluginActiveInConv=false）
	ApplyConvToolsetWhitelist(host, reg4, "conv_x", "")
	if reg4.IsEnabled("agent_teams_create") {
		t.Error("激活重置后按需插件工具应恢复白名单约束（不可见）")
	}
}
