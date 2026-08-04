package agent

// 模型扩散生成思想（Diffusion of Thought, DoT）—— 模拟扩散模型的两阶段生成过程，
// 为 agent 在任务起点提供「先想清楚再行动」的策略预演：
//
//   ═══ 发散阶段（加噪）═══
//     高熵生成：让 LLM 以较高温度、宽松约束，针对任务生成 N 个【多样化】候选行动方案。
//     候选覆盖不同策略（直接改/先调查/分步验证/并行/委托…），是"想法空间"的高熵采样。
//
//   ═══ 收敛阶段（去噪）═══
//     交叉评估：把 N 个候选一并交给 LLM，要求逐项评估（正确性/效率/风险/目标匹配度），
//     筛选或融合出【单一最优行动计划】（低熵决策）——相当于多步去噪后的"干净样本"。
//
//   ═══ 注入（决策）═══
//     收敛出的行动计划作为一条 user 消息注入【任务首轮】LLM 调用，主模型据此行动，
//     减少首轮盲目试错、无效工具调用。
//
// 约束对齐（为什么这样设计）：
//   • 只在 iter==0 触发：首轮 LLM 调用是冷缓存（无 KV 前缀可命中），注入思考草稿
//     不损失任何缓存收益；后续迭代前缀保持稳定 → 不破坏缓存前缀（同 compress.go 的设计约束）。
//   • 独立请求：发散/收敛是独立 Chat 调用，其消息序列不进主对话、不污染历史、
//     不破坏 tool_call↔tool_result 配对。
//   • 可开关：LoopOpts.DiffusionThink 三级传递（AppSettings → LoopOpts → Loop），默认关闭。
//   • 成本可控：发散 N×~60 + 收敛 ~400 ≈ 700~900 token/任务，且有 maxTokens 上限。
//   • 指标可采集：DiffuseStats 记录发散/收敛 token、候选数、耗时、收敛首步工具建议，
//     供 A/B 对比实验量化收益（token 消耗 / 结果质量 / 工具准确度 / 工具成功率）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DiffusionThinkOpts 扩散思考配置。
type DiffusionThinkOpts struct {
	Enabled    bool // 开关（默认关闭；实验时开启）
	Candidates int  // 发散候选数（默认 3，范围 2-5）
	MaxTokens  int  // 收敛输出上限（默认 800）
}

// default 返回填充默认值的副本。
func (o DiffusionThinkOpts) default_() DiffusionThinkOpts {
	if o.Candidates < 2 {
		o.Candidates = 3
	}
	if o.Candidates > 5 {
		o.Candidates = 5
	}
	if o.MaxTokens <= 0 {
		o.MaxTokens = 800
	}
	return o
}

// DiffuseStats 一次扩散思考的指标采集（实验对比用）。
type DiffuseStats struct {
	Triggered        bool   `json:"triggered"`          // 是否触发（发散+收敛均成功）
	ParseFail        bool   `json:"parseFail"`          // 响应解析失败（发散/收敛任一）
	Candidates       int    `json:"candidates"`         // 实际发散出的候选数
	DivergenceTokens int    `json:"divergenceTokens"`   // 发散请求 token 消耗
	ConvergenceTokens int   `json:"convergenceTokens"`  // 收敛请求 token 消耗
	TotalTokens      int    `json:"totalTokens"`        // 扩散思考总 token 消耗
	DurationMs       int64  `json:"durationMs"`         // 总耗时
	Plan             string `json:"plan"`               // 收敛出的行动计划
	SuggestedTool    string `json:"suggestedTool"`      // 建议的第一步工具
}

// ─── 发散阶段：高熵生成 N 个候选 ──────────────────────────

// divergeCandidate 一个候选行动方案。
type divergeCandidate struct {
	Name        string `json:"name"`        // 方案名
	Idea        string `json:"idea"`        // 一句话思路
	FirstTool   string `json:"firstTool"`   // 建议的第一个工具名
	FirstAction string `json:"firstAction"` // 第一步做什么
	Risk        string `json:"risk"`        // 主要风险
}

// divergeResponse 发散输出的 JSON 结构。
type divergeResponse struct {
	Candidates []divergeCandidate `json:"candidates"`
}

// divergePrompt 发散提示词（复刻扩散"加噪"：宽松约束、强调多样性、不要求正确）。
func divergePrompt(n int) string {
	return fmt.Sprintf(`你是策略发散器。给定一个任务，请以【不同的策略思路】生成 %d 个候选行动方案。

要求：
1. 每个候选采用不同策略（例如：直接修改 / 先调查后修改 / 分步验证 / 并行处理 / 委托子任务等），方案之间差异要明显，覆盖不同权衡。
2. 每个候选必须具体：一句话思路 + 建议的第一步工具调用（工具名+该步做什么）+ 该方案的主要风险。
3. 允许候选有偏差甚至大胆——发散阶段不追求正确，追求覆盖面。`, n)
}

// ─── 收敛阶段：交叉评估 → 单一计划 ────────────────────────

// convergeResponse 收敛输出的 JSON 结构。
type convergeResponse struct {
	Plan      string   `json:"plan"`      // 总体行动计划（3-5 句）
	FirstTool string   `json:"firstTool"` // 第一步应调用的工具名
	Order     []string `json:"order"`     // 后续步骤顺序
	Acceptance string  `json:"acceptance"` // 完成标准
}

// convergePrompt 收敛提示词（复刻扩散"去噪"：交叉评估、筛选融合、低熵决策）。
func convergePrompt(cands []divergeCandidate) string {
	var b strings.Builder
	for i, c := range cands {
		fmt.Fprintf(&b, "方案%d【%s】\n  思路: %s\n  首步工具: %s（%s）\n  风险: %s\n",
			i+1, c.Name, c.Idea, c.FirstTool, c.FirstAction, c.Risk)
	}
	return fmt.Sprintf(`你是决策收敛器。以下是针对同一任务的 %d 个候选行动方案：

%s

请【交叉评估】这些方案（考虑正确性、效率、风险、与任务目标的匹配度），筛选或融合出最优行动方案，输出单一行动计划。
注意：收敛出的计划要可直接执行，第一步工具调用必须具体明确。`, len(cands), b.String())
}

// ─── 主流程 ───────────────────────────────────────────────

// diffuseThink 执行一次扩散思考（发散 → 收敛），返回注入消息（nil 表示未注入）。
// 注入内容作为 user 角色消息追加到 callMsgs（首轮 LLM 调用），并采集 DiffuseStats。
func (l *Loop) diffuseThink(ctx context.Context, task string, callMsgs []Message, tools []ToolDefinition) ([]Message, *DiffuseStats) {
	opts := l.DiffusionThink.default_()
	stats := &DiffuseStats{Candidates: opts.Candidates}
	start := time.Now()
	defer func() { stats.DurationMs = time.Since(start).Milliseconds() }()

	if l.Provider == nil {
		return callMsgs, stats
	}

	// ── 发散：生成 N 个候选 ──
	divergeMsgs := []Message{
		{Role: RoleSystem, Content: divergePrompt(opts.Candidates)},
		{Role: RoleUser, Content: "任务: " + task + "\n\n请输出 " + fmt.Sprint(opts.Candidates) + " 个候选行动方案，只输出 JSON：" +
			`{"candidates":[{"name":"方案名","idea":"一句话思路","firstTool":"工具名","firstAction":"第一步做什么","risk":"主要风险"}]}`},
	}
	divergeResp, divergeTok, err := l.thinkCall(ctx, divergeMsgs, nil, opts.MaxTokens, true /*高温度*/)
	if err != nil {
		return callMsgs, stats // 失败不注入，退回原逻辑
	}
	stats.DivergenceTokens = divergeTok
	stats.TotalTokens += divergeTok

	cands, ok := parseDiverge(divergeResp)
	if !ok || len(cands) == 0 {
		stats.ParseFail = true
		return callMsgs, stats
	}
	stats.Candidates = len(cands)

	// ── 收敛：交叉评估 → 单一计划 ──
	// 传精简工具名列表（不传完整 schema，省 token；模型据此选真实存在的工具名）
	toolNames := make([]string, 0, len(tools))
	for _, td := range tools {
		if td.Function.Name != "" {
			toolNames = append(toolNames, td.Function.Name)
		}
	}
	convergeMsgs := []Message{
		{Role: RoleSystem, Content: convergePrompt(cands)},
		{Role: RoleUser, Content: "任务: " + task +
			"\n\n可用工具（只能从中选择 firstTool）：" + strings.Join(toolNames, ", ") +
			"\n\n请交叉评估并输出最优行动计划，只输出 JSON：" +
			`{"plan":"总体行动计划(2-4句，直接决策，不要长篇回退预案)","firstTool":"第一步工具名(必须来自可用工具列表)","order":["步骤2","步骤3"],"acceptance":"完成标准"}`},
	}
	convergeResp, convergeTok, err := l.thinkCall(ctx, convergeMsgs, nil, opts.MaxTokens, false /*低温度*/)
	if err != nil {
		return callMsgs, stats
	}
	stats.ConvergenceTokens = convergeTok
	stats.TotalTokens += convergeTok

	plan, ok := parseConverge(convergeResp)
	if !ok || strings.TrimSpace(plan.Plan) == "" {
		stats.ParseFail = true
		return callMsgs, stats
	}
	stats.Plan = plan.Plan
	stats.SuggestedTool = plan.FirstTool
	stats.Triggered = true

	// ── 注入：收敛计划作为首轮 user 消息追加 ──
	injected := fmt.Sprintf("【策略预演 · 扩散思考】任务开始前已完成 %d 个方案的发散与收敛评估。\n\n行动计划：\n%s",
		stats.Candidates, plan.Plan)
	if plan.FirstTool != "" {
		injected += "\n\n建议第一步调用工具：`" + plan.FirstTool + "`。"
	}
	if len(plan.Order) > 0 {
		injected += "\n后续步骤：" + strings.Join(plan.Order, " → ")
	}
	if plan.Acceptance != "" {
		injected += "\n完成标准：" + plan.Acceptance
	}

	// 追加到 callMsgs 末尾（首轮冷缓存，不破坏前缀；不进 msgs/历史）
	callMsgs = append(callMsgs, Message{Role: RoleUser, Content: injected})
	return callMsgs, stats
}

// thinkCall 执行一次扩散思考的子调用。返回响应文本 + token 消耗。
// highTemp=true 时若 Provider 是 OpenAIProvider 则用高温度副本（发散需要多样性）。
// 扩散子调用一律 non-thinking（快速、省 token——扩散只需要快速枚举与决策，不需要深度思考链）。
func (l *Loop) thinkCall(ctx context.Context, msgs []Message, tools []ToolDefinition, maxTokens int, highTemp bool) (string, int, error) {
	prov := l.Provider
	if op, ok := prov.(*OpenAIProvider); ok {
		clone := *op
		clone.ThinkingMode = "non-thinking" // 扩散子调用：关闭思考链，省时省 token
		if highTemp {
			clone.Temperature = 1.2 // 发散：较高温度，鼓励多样化
		}
		if maxTokens > 0 {
			clone.MaxTokens = maxTokens
		}
		prov = &clone
	}
	var usage *Usage
	resp, err := prov.Chat(ctx, msgs, tools, func(c Chunk) {
		if c.Usage != nil {
			usage = c.Usage
		}
	})
	if err != nil {
		return "", 0, err
	}
	tok := 0
	if usage != nil {
		tok = usage.TotalTokens
		if tok == 0 {
			tok = usage.PromptTokens + usage.CompletionTokens
		}
	} else {
		tok = estimateTokens(msgs) + estimateTokens([]Message{{Role: RoleAssistant, Content: resp.Content}})
	}
	return resp.Content, tok, nil
}

// ─── 解析 ────────────────────────────────────────────────

// parseDiverge 解析发散响应 JSON。宽容：定位 { 到末 }。
func parseDiverge(raw string) ([]divergeCandidate, bool) {
	i, j := strings.IndexByte(raw, '{'), strings.LastIndexByte(raw, '}')
	if i < 0 || j <= i {
		return nil, false
	}
	var dr divergeResponse
	if err := json.Unmarshal([]byte(raw[i:j+1]), &dr); err != nil {
		return nil, false
	}
	return dr.Candidates, len(dr.Candidates) > 0
}

// parseConverge 解析收敛响应 JSON。
func parseConverge(raw string) (convergeResponse, bool) {
	i, j := strings.IndexByte(raw, '{'), strings.LastIndexByte(raw, '}')
	if i < 0 || j <= i {
		return convergeResponse{}, false
	}
	var cr convergeResponse
	if err := json.Unmarshal([]byte(raw[i:j+1]), &cr); err != nil {
		return convergeResponse{}, false
	}
	return cr, cr.Plan != ""
}
