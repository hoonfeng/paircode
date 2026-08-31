package agent

import (
	"context"
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
