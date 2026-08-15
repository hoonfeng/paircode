package agent

import (
	"strings"
	"testing"
)

// TestPromptRegistry 注册中心：段排序 / 变量插值 / 动态上下文 / 空段删除。
func TestPromptRegistry(t *testing.T) {
	reg := NewPromptRegistry()
	// 乱序注册（验证按 order 升序）
	reg.Section("tool-guide", 150, "用 read 而非 cat（模型 {{model}}，工作区 {{cwd}}）")
	reg.Section("identity", -100, "You are an AI agent powered by Pair IDE.")
	reg.Section("persona", 0, "本地 AI 结对编程助手")
	reg.Variable("model", func() string { return "deepseek-r1" })
	reg.Variable("cwd", func() string { return "/ws/proj" })
	reg.Context("git-status", func() string { return "工作区干净" })
	reg.Section("empty", 200, "   ")

	got, err := reg.Assemble()
	if err != nil {
		t.Fatal(err)
	}
	// 顺序：identity(-100) → persona(0) → tool-guide(150)；empty 被删除
	iID, iPer, iTool := strings.Index(got, "AI agent"), strings.Index(got, "结对编程"), strings.Index(got, "read 而非 cat")
	if !(iID >= 0 && iPer > iID && iTool > iPer) {
		t.Errorf("段应按 order 排序：\n%s", got)
	}
	for _, want := range []string{"deepseek-r1", "/ws/proj", "动态上下文", "git-status", "工作区干净"} {
		if !strings.Contains(got, want) {
			t.Errorf("组装应含 %q：\n%s", want, got)
		}
	}

	// 未知变量引用 → 明确报错
	reg2 := NewPromptRegistry()
	reg2.Section("s", 100, "引用 {{nonexistent_var}} 应该报错")
	if _, err := reg2.Assemble(); err == nil || !strings.Contains(err.Error(), "nonexistent_var") {
		t.Errorf("未知变量应报错：%v", err)
	}
	// 无值变量 → 替换为空
	reg3 := NewPromptRegistry()
	reg3.Section("s", 100, "值[{{empty_var}}]")
	reg3.Variable("empty_var", func() string { return "" })
	if got3, err := reg3.Assemble(); err != nil || got3 != "值[]" {
		t.Errorf("无值变量应替换为空：%q %v", got3, err)
	}
	// 无任何段 → 空串（零影响）
	if got4, _ := NewPromptRegistry().Assemble(); got4 != "" {
		t.Errorf("空注册表应返回空串：%q", got4)
	}
}

// TestPluginPromptSections 插件贡献段/变量组装进系统提示。
func TestPluginPromptSections(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)

	// 直接贡献段 + 变量（对齐 ctx.systemPrompt.section / variable）
	host.ctx.AddSystemPromptSection(&PromptSection{Name: "plugin-a", Order: 300, Text: "插件 A：处理 markdown"})
	host.ctx.AddSystemPromptSection(&PromptSection{Name: "plugin-b", Order: 200, Text: "插件 B：检查 {{plugin_b_var}}"})
	host.ctx.AddSystemPromptVariable(&PromptVariable{Name: "plugin_b_var", Provider: func() string { return "文档一致性" }})

	got, err := PluginPromptSections(host)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "插件 B：检查 文档一致性") || !strings.Contains(got, "插件 A：处理 markdown") {
		t.Errorf("插件段组装应含两段且变量已插值：\n%s", got)
	}
	iB, iA := strings.Index(got, "插件 B"), strings.Index(got, "插件 A")
	if !(iB >= 0 && iA > iB) {
		t.Errorf("插件段应按 order 排序（B=200 在 A=300 前）：\n%s", got)
	}

	// nil host → 空串
	if gotNil, _ := PluginPromptSections(nil); gotNil != "" {
		t.Errorf("nil host 应返回空串：%q", gotNil)
	}
}
