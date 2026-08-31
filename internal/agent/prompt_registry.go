// prompt_registry.go —— 系统提示词组装注册表（对齐 deepseek-harness system-prompt）。
//
// harness 设计（packages/core/system-prompt）：
//   - sections（段）：{ name, order, text } 有序贡献；order 升序拼接（-100 身份 / 0 persona /
//     100-199 工具引导 / 之后插件段）。complete 段可整体替换（本实现不提供，避免过度设计）。
//   - contexts（有序动态上下文）：每次组装求值提供方，成为带来源的 runtime-context 快照
//     （本实现合并进段尾部，来源=段名）。
//   - variables（变量）：{{name}} 引用，组装时求值；未知引用/格式错误明确报错。
//   - tools（工具 schema）：由 Registry 承担（gou-ide 已按顺序输出工具定义），不重复。
//
// gou-ide 接入：插件已通过 ctx.systemPrompt.section 贡献段（PluginHost.Sections()），
// 但此前从未组装进系统提示词——本轮补上：PluginPromptSections 把插件段并入注册中心组装。
// 默认无插件段时组装为空串（零影响，向后兼容）。

package agent

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// promptVarRe 匹配 {{name}} 完整组（name 内不含 { }，防嵌套）。
var promptVarRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// ── 系统提示槽位约定（对齐 deepseek-harness system-prompt）──
// harness 用固定 section name/order 约定可替换槽位：
//
//	PERSONA_SECTION = "deployment:persona"（部署人格槽位，order=0）
//	RULES_SECTION   = "deployment:rules"（行为准则槽位，order=100）
//	HARNESS_IDENTITY = -100：harness 身份段（本实现并入默认 persona 文本，不单独暴露）
//
// 插件若以 name==PERSONA_SECTION 贡献段，组装时**替换**默认 persona 段
// （而非追加），对齐 harness「persona slot 可被 agent preset 替换」语义；
// 同理 name==RULES_SECTION 的段替换默认规则段（# 工作区之后的第一铁律/核心规则等
// 全部行为准则），规则槽位换准则、persona 槽位换人格，两者独立可组合。
const (
	PERSONA_SECTION = "deployment:persona"
	PERSONA_ORDER   = 0
	RULES_SECTION   = "deployment:rules"
	RULES_ORDER     = 100
)

// PromptRegistry 系统提示词组装注册表。
type PromptRegistry struct {
	mu        sync.Mutex
	sections  []promptRegSection
	contexts  []promptRegContext
	variables map[string]func() string
}

type promptRegSection struct {
	name  string
	order int
	text  string
}

type promptRegContext struct {
	name string
	prov func() string
}

// NewPromptRegistry 新建注册表。
func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{variables: map[string]func() string{}}
}

// Section 贡献一个有序段（order 升序拼接；默认 100）。
func (p *PromptRegistry) Section(name string, order int, text string) {
	if order == 0 {
		order = 100
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sections = append(p.sections, promptRegSection{name: name, order: order, text: text})
}

// Variable 注册提示词变量（{{name}} 插值；provider 返回当前值，空串=本次无值→替换为空）。
func (p *PromptRegistry) Variable(name string, prov func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.variables[name] = prov
}

// Context 贡献有序动态上下文（组装时求值，来源=段名，追加在段尾部）。
func (p *PromptRegistry) Context(name string, prov func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contexts = append(p.contexts, promptRegContext{name: name, prov: prov})
}

// AddPluginSections 并入插件贡献的系统提示片段（PluginHost.Sections()）。
func (p *PromptRegistry) AddPluginSections(secs []*PromptSection) {
	for _, s := range secs {
		if s == nil || strings.TrimSpace(s.Text) == "" {
			continue
		}
		p.Section(s.Name, s.Order, s.Text)
	}
}

// Assemble 组装系统提示词：段按 order 升序拼接 + 动态上下文段 + 变量插值，删除空段。
// 无任何段时返回 ""（零影响）。变量引用 {{name}}：未知引用报错；已注册但无值→空串。
func (p *PromptRegistry) Assemble() (string, error) {
	p.mu.Lock()
	sections := append([]promptRegSection(nil), p.sections...)
	contexts := append([]promptRegContext(nil), p.contexts...)
	vars := make(map[string]func() string, len(p.variables))
	for k, v := range p.variables {
		vars[k] = v
	}
	p.mu.Unlock()

	sort.SliceStable(sections, func(i, j int) bool { return sections[i].order < sections[j].order })

	var parts []string
	for _, s := range sections {
		if t := strings.TrimSpace(s.text); t != "" {
			parts = append(parts, t)
		}
	}
	for _, c := range contexts {
		v := strings.TrimSpace(c.prov())
		if v != "" {
			parts = append(parts, "# 动态上下文（"+c.name+"）\n"+v)
		}
	}
	if len(parts) == 0 {
		return "", nil
	}

	values := make(map[string]string, len(vars))
	for k, prov := range vars {
		values[k] = ""
		if prov != nil {
			values[k] = prov()
		}
	}

	var out strings.Builder
	for i, part := range parts {
		rendered, err := renderPrompt(part, values)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(rendered) == "" {
			continue
		}
		if i > 0 {
			out.WriteString("\n\n")
		}
		out.WriteString(rendered)
	}
	return out.String(), nil
}

// renderPrompt 对文本做 {{name}} 变量插值（严格：未知引用报错；无值→空串；孤立 {{ 按字面量）。
func renderPrompt(text string, vars map[string]string) (string, error) {
	var unknown []string
	// 先替换所有完整组：未知引用记录后清空，避免正则替换回调中报错中断流程
	replaced := promptVarRe.ReplaceAllStringFunc(text, func(m string) string {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(m, "{{"), "}}"))
		v, ok := vars[name]
		if !ok {
			unknown = append(unknown, name)
			return ""
		}
		return v
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("renderPrompt: 未知变量引用 {{%s}}", strings.Join(uniqueStrings(unknown), "}}, {{"))
	}
	return replaced, nil
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// PluginPromptSections 把插件贡献的系统提示段 + 变量组装为文本段（对齐 harness system-prompt 组装）。
// persona/rules 槽位段（name==PERSONA_SECTION / RULES_SECTION）不在此处输出——
// 它们已在静态侧由 DefaultSystemPromptWithOverrides 整体替换，此处跳过避免重复注入。
// 无插件段时返回 ""（零影响）。供 web_server/AgentBase 在构建 system prompt 时调用。
//
// ★ 2026-08-31 按需激活：声明为 on-demand 的插件（agent-teams 等）段仅在
//   convID 会话内已激活时注入；否则全部隐藏（工具同样隐藏，见 MergePluginTools）。
//   convID 为空 = 未开会话，按需插件的段一律不注入。
func PluginPromptSections(host *PluginHost, convID string) (string, error) {
	if host == nil {
		return "", nil
	}
	reg := NewPromptRegistry()
	for _, s := range host.Sections() {
		if s == nil || s.Name == PERSONA_SECTION || s.Name == RULES_SECTION || strings.TrimSpace(s.Text) == "" {
			continue
		}
		if !IsPluginActiveInConv(convID, s.Plugin) {
			continue // 按需插件未激活 → 隐藏该段
		}
		reg.Section(s.Name, s.Order, s.Text)
	}
	for _, v := range host.Variables() {
		prov := v.Provider
		reg.Variable(v.Name, func() string {
			if prov == nil {
				return ""
			}
			return prov()
		})
	}
	return reg.Assemble()
}
