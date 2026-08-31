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

	got, err := PluginPromptSections(host, "")
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
	if gotNil, _ := PluginPromptSections(nil, ""); gotNil != "" {
		t.Errorf("nil host 应返回空串：%q", gotNil)
	}
}

// TestPersonaSection 插件 persona 槽位（对齐 harness deployment:persona）：
// ① PersonaSection() 返回插件贡献的 persona 段；
// ② PluginPromptSections 排除 persona 段（不重复注入动态侧）；
// ③ DefaultSystemPromptWithPersona 用插件 persona 替换默认身份段、规则段保留。
func TestPersonaSection(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)

	// 插件贡献 persona 段 + 普通段
	host.ctx.AddSystemPromptSection(&PromptSection{Name: PERSONA_SECTION, Order: 0, Text: "你是 Mini-Coder，专注 Go 重构。\n不闲聊。"})
	host.ctx.AddSystemPromptSection(&PromptSection{Name: "plugin-rules", Order: 200, Text: "规则：改后必须 gofmt。"})

	// ① PersonaSection 命中
	ps := host.PersonaSection()
	if ps == nil {
		t.Fatal("PersonaSection() 应返回插件 persona 段")
	}
	if !strings.Contains(ps.Text, "Mini-Coder") {
		t.Errorf("persona 段内容不符：%q", ps.Text)
	}

	// ② 动态侧排除 persona 段
	dyn, err := PluginPromptSections(host, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dyn, "Mini-Coder") {
		t.Errorf("PluginPromptSections 不应包含 persona 段（已静态替换）：%q", dyn)
	}
	if !strings.Contains(dyn, "gofmt") {
		t.Errorf("普通插件段应正常输出：%q", dyn)
	}

	// ③ 默认 persona 被替换、规则段保留
	full := DefaultSystemPromptWithPersona([]string{root}, ps.Text)
	if !strings.Contains(full, "Mini-Coder") {
		t.Errorf("替换后应含插件 persona：%q", full)
	}
	if strings.Contains(full, "Pair CodeAgent") {
		t.Errorf("默认 persona 应被替换掉：%q", full)
	}
	if !strings.Contains(full, "# 工作区") || !strings.Contains(full, "# 核心规则") {
		t.Errorf("工作区/规则段应保留：\n%s", full)
	}

	// 空 persona → 等价默认
	if got := DefaultSystemPromptWithPersona([]string{root}, ""); got != DefaultSystemPrompt([]string{root}) {
		t.Errorf("空 persona 应等价默认提示")
	}
	// 无 persona 贡献 → PersonaSection nil
	host2 := NewPluginHost(NewRegistry(), nil, root)
	if host2.PersonaSection() != nil {
		t.Errorf("无 persona 贡献时应返回 nil")
	}
}

// TestRulesSection 插件 rules 槽位（对齐 harness deployment:rules）：
// ① RulesSection() 返回插件贡献的 rules 段；
// ② PluginPromptSections 排除 rules 段（不重复注入动态侧）；
// ③ DefaultSystemPromptWithOverrides 用插件 rules 替换默认规则段、身份/工作区保留；
// ④ persona+rules 组合替换互不干扰；
// ⑤ 全空时与默认提示逐字节一致。
func TestRulesSection(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	host := NewPluginHost(reg, nil, root)

	// 插件贡献 rules 段 + persona 段 + 普通段
	host.ctx.AddSystemPromptSection(&PromptSection{Name: RULES_SECTION, Order: 100, Text: "## 定制行为准则\n- 只改必要文件。\n- 改后必须 gofmt。"})
	host.ctx.AddSystemPromptSection(&PromptSection{Name: PERSONA_SECTION, Order: 0, Text: "你是 Mini-Coder。"})
	host.ctx.AddSystemPromptSection(&PromptSection{Name: "plugin-extra", Order: 300, Text: "额外：用中文回复。"})

	// ① RulesSection 命中
	rs := host.RulesSection()
	if rs == nil {
		t.Fatal("RulesSection() 应返回插件 rules 段")
	}
	if !strings.Contains(rs.Text, "定制行为准则") {
		t.Errorf("rules 段内容不符：%q", rs.Text)
	}

	// ② 动态侧排除 persona+rules 段，普通段保留
	dyn, err := PluginPromptSections(host, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dyn, "定制行为准则") || strings.Contains(dyn, "Mini-Coder") {
		t.Errorf("PluginPromptSections 不应包含 persona/rules 槽位段（已静态替换）：%q", dyn)
	}
	if !strings.Contains(dyn, "用中文回复") {
		t.Errorf("普通插件段应正常输出：%q", dyn)
	}

	// ③ rules 单独替换：身份/工作区保留，默认规则段被替换
	only := DefaultSystemPromptWithOverrides([]string{root}, "", rs.Text)
	if !strings.Contains(only, "Pair CodeAgent") {
		t.Errorf("仅换 rules 时 persona 应保留：%q", only)
	}
	if !strings.Contains(only, "# 工作区") {
		t.Errorf("工作区段应保留：%q", only)
	}
	if !strings.Contains(only, "定制行为准则") {
		t.Errorf("替换后应含插件 rules：%q", only)
	}
	if strings.Contains(only, "第一铁律") || strings.Contains(only, "# 核心规则") {
		t.Errorf("默认规则段应被替换掉（不应含第一铁律/核心规则）：\n%s", only)
	}

	// ④ persona+rules 组合：身份与准则都被替换
	both := DefaultSystemPromptWithOverrides([]string{root}, "你是 Full-Coder。", rs.Text)
	if !strings.Contains(both, "Full-Coder") {
		t.Errorf("组合时 persona 应替换：%q", both)
	}
	if !strings.Contains(both, "定制行为准则") {
		t.Errorf("组合时 rules 应替换：%q", both)
	}
	if strings.Contains(both, "Pair CodeAgent") || strings.Contains(both, "第一铁律") {
		t.Errorf("组合时默认 persona/规则段都应被替换：\n%s", both)
	}
	if !strings.Contains(both, "# 工作区") {
		t.Errorf("组合时工作区段应保留：%q", both)
	}

	// ⑤ 全空 → 逐字节一致
	if got := DefaultSystemPromptWithOverrides([]string{root}, "", ""); got != DefaultSystemPrompt([]string{root}) {
		t.Errorf("全空 overrides 应与默认提示逐字节一致")
	}
	// 无 rules 贡献 → RulesSection nil
	host2 := NewPluginHost(NewRegistry(), nil, root)
	if host2.RulesSection() != nil {
		t.Errorf("无 rules 贡献时应返回 nil")
	}
}
