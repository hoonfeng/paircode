package agent

// handoff.go — 会话交接（提交消息）：续跑 / 新提交对话时的历史判断与整理。
//
// 背景（2026-09-11，用户需求）：
//
//	对话历史会在两个时机被「直接全量注入」下一段 LLM 上下文——
//	① 续跑（段预算分段——★ 双闸门：步数 / 工具调用，tool_budget.go）：段边界 loop.Run(nil)
//	   携带上一段全部消息（含大量工具结果）；
//	② 新提交对话（buildWebLoopOpts）：从 store LoadAll 全量历史（仅达
//	   45% 压力阈值时才走 CondenseHistoryByPressure 规则压缩）。
//	长对话实测 historyTokens 130K+（llm-trace）→ 上下文膨胀、窗口吃紧。
//
// 机制（本文件）：在这两个时机做一次「判断 + 整理」：
//  1. 判断（规则门槛，轻量）：历史 token ≥ 阈值（窗口 30%，地板 24K）或
//     条数 ≥ 100 —— 未达阈值不动历史（小对话保持逐字节原样，KV 前缀稳定）；
//  2. 整理（一次 LLM 调用，失败回退规则式）：把历史整理成一条
//     「会话交接·提交消息」（相关性判断 + 目标/已完成/状态/决策/下一步）；
//     相关性（高/部分/无关）由同一次调用判断并写入交接文本，与当前任务
//     无关的历史细节折叠为一句概述；
//  3. 复用/刷新：交接记录持久化（{conv}.handoff.json）。自上次交接后的增量
//     token < refresh 阈值（12K）→ 复用上次交接文本（不再调 LLM）；
//     ≥ 阈值 → 基于「上次交接 + 取样历史」重新生成。
//
// 注入视图（缓存友好的「追加语义」）：
//
//	[交接消息] + [基点起的最近消息（含基点，原文保留）]（+ 调用方追加的当前任务消息）
//
//	同一交接周期内，历史前缀（交接消息）逐字节稳定、增量只追加尾部；
//	刷新交接时断一次前缀（有意的整理代价——把长历史一次折叠成交接消息）。
//
//	KV 缓存：复用周期内交接文本来自存盘（逐字节不变）+ 锚点后增量只追加 →
//	请求前缀逐字节稳定（缓存命中的前提）。前缀断裂只发生在两处（均有意、
//	频率受控）：① 首次整理（全量历史 → 交接视图，一次性）；② 刷新
//	（增量 ≥ 12K tokens 才重新生成）。未达触发阈值时历史**逐字节原样**，
//	缓存行为与不启用交接完全一致。
//
// 边界与安全性：
//   - 原文不删除：交接只替换「喂 LLM 的历史视图」；落盘/展示仍为完整时间线
//     （视图文本以 handoffTitle 开头，落盘锚点/真实任务轮次统计自动跳过，
//     见 isInjectedUserMessage / IsHandoffText）；
//   - 关闭：环境变量 PAIR_HANDOFF=0；阈值覆盖：PAIR_HANDOFF_TRIGGER_TOKENS /
//     PAIR_HANDOFF_REFRESH_TOKENS（便于测试与按需调节）；
//   - 增量基点：经「消息内容指纹序列」锚定（HandoffRecord.Anchor，连续
//     handoffAnchorSeq 条指纹）而非条数——续跑链路（loop.History）与 store
//     （LoadAll）的条数口径可能错位，内容锚定跨口径稳定；重复消息场景由
//     序列匹配防漂移；视图首条保证非孤立 tool（配对 tool_call 被折叠时
//     基点前移，宁可多保留）；锚点丢失时保守保留最近 handoffKeepRecentMsgs 条。
//   - 最近消息保留：交接折叠早期历史，视图至少保留最近 handoffKeepRecentMsgs
//     条原文（不丢最近执行细节）。

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// handoffTitle 交接注入文本的标题（兼作识别标记：文本以它开头，见 injected_msg.go）。
	handoffTitle = "【会话交接·提交消息】"
	// handoffTriggerRatio 触发整理的 token 占窗口比例阈值（与地板取大）。
	handoffTriggerRatio = 0.30
	// handoffTriggerMinTokens 触发阈值地板（窗口很小/未配置时的下限）。
	handoffTriggerMinTokens = 24000
	// handoffTriggerMinMsgs 触发整理的消息条数下限（与 token 条件取「或」）。
	handoffTriggerMinMsgs = 100
	// handoffRefreshTokens 自上次交接后的增量 token 达到该值才重新生成交接
	// （否则复用上次交接文本，避免每轮一次 LLM 调用）。
	handoffRefreshTokens = 12000
	// handoffKeepRecentMsgs 视图至少保留的最近消息条数（原文，不折叠）——
	// 交接折叠早期历史，最近的执行细节原样保留（不丢最近状态）。
	handoffKeepRecentMsgs = 16
	// handoffInputMaxMsgs LLM 整理输入的取样条数上限（取历史尾部；
	// 更早历史以规则式摘要并入输入，见 handoffSample）。
	handoffInputMaxMsgs = 120
	// handoffInputMsgRunes 取样中每条消息的截断长度（rune）。
	handoffInputMsgRunes = 300
	// handoffPrevTextRunes 上次交接文本纳入输入的截断长度（rune）。
	handoffPrevTextRunes = 3000
	// handoffTaskRunes 当前任务文本纳入输入的截断长度（rune）。
	handoffTaskRunes = 1500
	// handoffTimeoutDefault LLM 整理调用的默认超时。
	handoffTimeoutDefault = 90 * time.Second
	// handoffRecheckTokens 判官复检周期：自上次复检以来增量 tokens 达到该值
	// 再跑一次轻量相关性判官（B：复用期定期语义复检）。
	handoffRecheckTokens = 4000
	// 视图保留段深度（按相关性；视图 = [交接消息] + [基点起最近 keep 条]）：
	// 高=同一任务延续（保留全部近期细节）；部分=同项目不同任务；无关=全新任务。
	handoffKeepRelHigh    = handoffKeepRecentMsgs // 16
	handoffKeepRelPartial = 8
	handoffKeepRelNone    = 4
	// 相关性取值（交接文本小节标注 / 判官输出）。
	handoffRelHigh    = "高"
	handoffRelPartial = "部分"
	handoffRelNone    = "无关"
	// handoffJudgeTaskRunes / handoffJudgePrevRunes 判官输入的截断长度（rune）——
	// 判官只需任务与交接要点的粗粒度判断（极小输入）。
	handoffJudgeTaskRunes = 600
	handoffJudgePrevRunes = 1200
	// handoffJudgeMaxTokens 判官输出上限（默认建议值；实际由调用方实例的
	// MaxTokens 控制——web 层构建独立实例时设为其设定值）。
	handoffJudgeMaxTokens = 32
	// handoffJudgeTimeout 判官调用超时（失败零副作用，保持现状）。
	handoffJudgeTimeout = 30 * time.Second
	// handoffAnchorSep 锚点序列的分隔符（Anchor 字段内多段指纹拼接）。
	// \x1e 为 ASCII 记录分隔符，正常消息内容不含该控制字符——解析无歧义；
	// 旧版单指纹记录（无分隔符）解析为 1 段，按单条匹配兼容。
	handoffAnchorSep = "\x1e"
	// handoffAnchorSeq 序列锚的指纹条数（自基点起，含基点本身）。
	// 单指纹在历史含重复消息（同工具同结果反复出现）时会锚到错误位置
	// （视图漏段/前缀漂移）——连续多条指纹同时相同的概率可忽略，
	// 定位跨轮稳定（KV 前缀稳定的前提）。
	handoffAnchorSeq = 3
)

// HandoffRecord 会话交接记录（持久化于 {conv}.handoff.json）。
type HandoffRecord struct {
	Text      string `json:"text"`      // 交接文本（含固定标题与背景前缀）
	CreatedAt string `json:"createdAt"` // 生成时间（RFC3339）
	// Anchor 增量基点：交接时保留段起点消息的「内容指纹」（handoffFingerprint）。
	// 用内容锚定而非条数：续跑链路（loop.History，含未落盘的续跑消息/交接视图）
	// 与 store（LoadAll，不含未落盘消息）的条数口径可能错位，内容锚定跨口径稳定。
	Anchor string `json:"anchor,omitempty"`
	// KeptTokens 生成时「锚后内容」的 token 估算（保留段基线）。
	// 刷新判断只看「相对基线的增量」——否则刻意保留的最近消息会被计入增量，
	// 阈值低于保留段时每轮都触发刷新（抖动）。
	KeptTokens int `json:"keptTokens,omitempty"`
	// MsgCount 生成时的非 system 历史消息条数（诊断/统计用，不参与切片）。
	MsgCount int `json:"msgCount"`
	// Relevance 与当前任务的相关性结论："高"/"部分"/"无关"。
	// 生成时从交接文本的「相关性」小节解析初始化；复用期由轻量判官
	// 周期性复检更新（判官发现关系变化 → 触发重生成）。
	// 控制视图保留段深度（高=16 条 / 部分=8 / 无关=4）。
	Relevance string `json:"relevance,omitempty"`
	// RelCheckedInc 上次判官复检时的增量 tokens（相对 KeptTokens 基线）——
	// 自复检以来的新增量再达 handoffRecheckTokens 才复检下一次（防每轮调用）。
	RelCheckedInc int `json:"relCheckedInc,omitempty"`
}

// handoffStore 交接记录持久化能力（MessageStore 实现；接口之外按需断言，
// 未实现该能力的存储（如未来其他后端）自动退化为「不启用交接」）。
type handoffStore interface {
	SaveHandoff(convID string, rec HandoffRecord) error
	LoadHandoff(convID string) (*HandoffRecord, error)
}

// HandoffEnabled 是否启用会话交接（默认启用；PAIR_HANDOFF=0/off/false 关闭）。
func HandoffEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PAIR_HANDOFF"))) {
	case "0", "off", "false", "no", "disable", "disabled":
		return false
	}
	return true
}

// HandoffTriggerThreshold 触发整理的 token 阈值（环境变量 > 比例与地板取大）。
func HandoffTriggerThreshold(maxContextTokens int) int {
	if v := strings.TrimSpace(os.Getenv("PAIR_HANDOFF_TRIGGER_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		log.Printf("[handoff] PAIR_HANDOFF_TRIGGER_TOKENS=%q 非法（需正整数），忽略", v)
	}
	t := int(float64(EffectiveCondenseBudget(maxContextTokens)) * handoffTriggerRatio)
	if t < handoffTriggerMinTokens {
		t = handoffTriggerMinTokens
	}
	return t
}

// handoffRefreshThreshold 刷新（重新生成）的增量 token 阈值（环境变量可覆盖）。
func handoffRefreshThreshold() int {
	if v := strings.TrimSpace(os.Getenv("PAIR_HANDOFF_REFRESH_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		log.Printf("[handoff] PAIR_HANDOFF_REFRESH_TOKENS=%q 非法（需正整数），忽略", v)
	}
	return handoffRefreshTokens
}

// handoffRecheckThreshold 判官复检周期（环境变量 PAIR_HANDOFF_RECHECK_TOKENS 可覆盖）。
func handoffRecheckThreshold() int {
	if v := strings.TrimSpace(os.Getenv("PAIR_HANDOFF_RECHECK_TOKENS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		log.Printf("[handoff] PAIR_HANDOFF_RECHECK_TOKENS=%q 非法（需正整数），忽略", v)
	}
	return handoffRecheckTokens
}

// keepForRelevance 按相关性返回视图保留段深度（空/未知 → 高，保守不缩）。
func keepForRelevance(rel string) int {
	switch rel {
	case handoffRelNone:
		return handoffKeepRelNone
	case handoffRelPartial:
		return handoffKeepRelPartial
	}
	return handoffKeepRelHigh
}

// parseRelevance 解析相关性标注：取文本内**最早出现**的「高/部分/无关」
// （中文优先；无则小写英文 high/partial/none 兜底）。用于：
//  1. 判官输出解析（期望一个词，多一字容错）；
//  2. 交接文本生成后从「## 与当前任务的相关性」小节解析初始值。
//
// 扫描窗口限于前 200 runes（判官输出极短；交接文本的小节在前部）。
func parseRelevance(s string) string {
	if s == "" {
		return ""
	}
	head := truncRunesAgent(s, 200)
	best, bestIdx := "", -1
	for _, tag := range []string{handoffRelNone, handoffRelPartial, handoffRelHigh} {
		if i := strings.Index(head, tag); i >= 0 && (bestIdx == -1 || i < bestIdx) {
			best, bestIdx = tag, i
		}
	}
	if best != "" {
		return best
	}
	low := strings.ToLower(head)
	for _, p := range []struct{ en, zh string }{
		{"none", handoffRelNone}, {"partial", handoffRelPartial}, {"high", handoffRelHigh},
	} {
		if i := strings.Index(low, p.en); i >= 0 && (bestIdx == -1 || i < bestIdx) {
			best, bestIdx = p.zh, i
		}
	}
	return best
}

// HandoffJudgeProvider 构建判官 Provider（独立轻量实例）：
// 在给定参数副本上覆盖 non-thinking + Temperature=-1 + MaxTokens=32
// （判官只需输出一个词）——与主对话通道完全隔离，单次成本可忽略。
// 参数不全（Key/地址/模型缺失）返回 nil（调用方跳过判官，行为退化为原状）。
func HandoffJudgeProvider(base ProviderParams) Provider {
	if base.APIKey == "" || base.BaseURL == "" || base.Model == "" {
		return nil
	}
	base.ThinkingMode = "non-thinking"
	base.Temperature = -1
	base.MaxTokens = handoffJudgeMaxTokens
	base.Multimodal = false
	return CreateProvider(base)
}

// handoffJudgeRelevance 轻量相关性判官（B：复用期定期语义复检）：
// 用**独立实例**（non-thinking、极小 max_tokens）判「当前任务」与
// 「历史交接要点」的关系，只输出一个词：高 / 部分 / 无关。
// 设计约束：极小输入（任务 ≤600 + 交接 ≤1200 runes）+ 极小输出（一个词）；
// 失败（nil / 报错 / 输出不可解析）返回 ""——调用方保持现状，零副作用。
func handoffJudgeRelevance(ctx context.Context, prov Provider, rec *HandoffRecord, task string) string {
	if prov == nil || rec == nil || strings.TrimSpace(rec.Text) == "" || strings.TrimSpace(task) == "" {
		return ""
	}
	hctx := ctx
	if hctx == nil {
		hctx = context.Background()
	}
	hctx, cancel := context.WithTimeout(hctx, handoffJudgeTimeout)
	defer cancel()
	prompt := "判断「当前任务」与「历史交接要点」的关系，只输出一个词：高、部分 或 无关。\n" +
		"高=同一任务的延续；部分=同项目不同任务；无关=全新任务。\n\n" +
		"【当前任务】\n" + truncRunesAgent(task, handoffJudgeTaskRunes) + "\n\n" +
		"【历史交接要点】\n" + truncRunesAgent(rec.Text, handoffJudgePrevRunes) + "\n\n只输出一个词："
	msg, err := prov.Chat(hctx, []Message{{Role: RoleUser, Content: prompt}}, nil, nil)
	if err != nil {
		log.Printf("[handoff] 判官调用失败（保持现状，下次周期重试）: %v", err)
		return ""
	}
	rel := parseRelevance(msg.Content)
	if rel == "" {
		log.Printf("[handoff] 判官输出不可解析（保持现状）: %q", truncRunesAgent(msg.Content, 60))
	}
	return rel
}

// stripSystemMsgs 返回去除 system 消息后的列表副本（统一条数/切片口径：
// LoadAll 不含 system；loop.History 可能含 system——两者经此对齐）。
func stripSystemMsgs(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem {
			continue
		}
		out = append(out, m)
	}
	return out
}

// isHandoffText 判断消息是否为「会话交接·提交消息」注入块（视图产物识别）。
func isHandoffText(content string) bool {
	return strings.Contains(content, handoffTitle)
}

// handoffFingerprint 计算消息内容指纹（增量锚点）：
// role + 正文长度 + 正文前 80 rune（区分度足够，内容微调不敏感场景可接受）。
func handoffFingerprint(m Message) string {
	return string(m.Role) + "|" + strconv.Itoa(len(m.Content)) + "|" + truncRunesAgent(m.Content, 80)
}

// handoffAnchor 计算交接基点指纹（序列锚）——保留段深度取 handoffKeepRecentMsgs。
func handoffAnchor(history []Message) string {
	return handoffAnchorAt(history, handoffKeepRecentMsgs)
}

// handoffAnchorAt 计算交接基点指纹（序列锚），保留段深度可指定（keep ≥ 1）。
// 基点首选：历史（去 system）尾部倒数第 keep 条——
// 视图中转载「基点起」的最近消息（含基点本身），早期历史折叠进交接消息。
//
// ★ 边界对齐：若基点本身是 tool 消息（=视图首条），说明其配对的
// assistant tool_call 将被折叠——视图将以孤立 tool 开头（OpenAI 规范下
// 该条会被 sanitizeToolPairing 丢弃 → 内容丢失）。此时把基点向前移动，
// 直到基点不再是 tool（宁可多保留几条，不制造孤儿；与 compress.go
// 的 compact 配对保护同因）。
//
// ★ 序列锚：记录「基点起连续 handoffAnchorSeq 条」的指纹（不足取全部）。
// 单指纹在历史含重复消息时会锚到错误位置（视图漏段/前缀漂移），
// 连续多条指纹同时相同的概率可忽略——跨轮定位稳定。
func handoffAnchorAt(history []Message, keep int) string {
	hist := stripSystemMsgs(history)
	if len(hist) == 0 {
		return ""
	}
	if keep < 1 {
		keep = 1
	}
	idx := len(hist) - keep
	if idx < 0 {
		idx = 0
	}
	// 边界对齐：视图首条（=基点本身）不能是孤立 tool（向前移动基点）。
	for idx > 0 && hist[idx].Role == RoleTool {
		idx--
	}
	n := handoffAnchorSeq
	if idx+n > len(hist) {
		n = len(hist) - idx
	}
	parts := make([]string, 0, n)
	for k := 0; k < n; k++ {
		parts = append(parts, handoffFingerprint(hist[idx+k]))
	}
	return strings.Join(parts, handoffAnchorSep)
}

// ShouldHandoff 判断历史是否需要交接整理（规则门槛，轻量）。
// 返回（是否需要，可读原因）。未达阈值时保持历史逐字节原样（KV 前缀稳定）。
func ShouldHandoff(history []Message, maxContextTokens int) (bool, string) {
	if !HandoffEnabled() {
		return false, "已关闭（PAIR_HANDOFF=0）"
	}
	hist := stripSystemMsgs(history)
	if len(hist) == 0 {
		return false, ""
	}
	tokens := estimateTokens(hist)
	trigger := HandoffTriggerThreshold(maxContextTokens)
	if tokens >= trigger {
		return true, fmt.Sprintf("历史 %d 条 / ~%d tokens ≥ 触发阈值 %d", len(hist), tokens, trigger)
	}
	if len(hist) >= handoffTriggerMinMsgs {
		return true, fmt.Sprintf("历史 %d 条 ≥ 触发条数 %d（~%d tokens）", len(hist), handoffTriggerMinMsgs, tokens)
	}
	return false, ""
}

// handoffIncrement 返回自上次交接基点起的增量消息（去 system，**含基点本身**）。
// 兜底（锚丢失）时按相关性保留段深度取最近消息（keepForRelevance）。
// 含锚是续跑链路复用成立的前提：视图=[交接]+[锚起保留段]，Run 后 loop.History
// 里锚消息作为视图首条存在——下次段边界仍能指纹匹配复用（否则锚被视图丢弃 →
// 每段重建交接 → 每次前缀断裂，KV 缓存无法跨段命中）。
// 基点经内容指纹序列定位（从末尾向前找）；锚点找不到（历史被压缩/回滚/跨口径）
// 时保守返回最近 handoffKeepRecentMsgs 条（宁可多带，不丢最近状态）。
func handoffIncrement(history []Message, rec *HandoffRecord) []Message {
	hist := stripSystemMsgs(history)
	if rec == nil || rec.Anchor == "" {
		return hist
	}
	if i := handoffAnchorIndex(hist, rec); i >= 0 {
		return hist[i:]
	}
	keep := handoffKeepRecentMsgs
	if rec != nil && rec.Relevance != "" {
		keep = keepForRelevance(rec.Relevance)
	}
	if len(hist) > keep {
		return hist[len(hist)-keep:]
	}
	return hist
}

// handoffAnchorIndex 在历史（已去 system）中定位锚点消息下标；未找到返回 -1。
// 序列锚（handoffAnchorSep 拼接的多段指纹）要求「连续匹配」；旧版单指纹记录
// 解析为单段，退化为单条匹配（兼容存量 .handoff.json）。
//
// ★ 2026-09-12 修复（缓存命中率骤降根因）：**定位优先用记录内的基线位置**，
//
//	 其次从前往后取第一个匹配。
//	 原实现从末尾向前找第一个匹配——消息内容重复时（同一工具同参数反复调用、
//	 模型套话反复出现、"第N步"这类模板化正文）锚序列会在历史中多处命中，
//	 从末尾找会定位到**最近一次**出现处（实测距末尾 15 条，真基点在第 107 条，
//	 漂移 366 条）→ 两个后果：
//	   ① ComposeHandoffView 的保留段退化为「最近十几条」，每次跨轮重排 →
//	      provider 前缀缓存从交接消息之后整段失效（实测命中率 98%→56%，
//	      每轮白付 ~24K tokens）；
//	   ② ShouldRefreshHandoff 的 increment ≈ KeptTokens → delta ≈ 0 →
//	      永不刷新（交接文本老化，新内容进不去）。
//	 定位优先级：
//	1. 记录生成时的基点位置（MsgCount − 保留深度，再按「基点非 tool」对齐）——
//	   历史 append-only 时该位置跨轮**恒定且精确**，能穿透重复消息干扰
//	   （既保持折叠瘦身，又保证前缀单调）；
//	2. 从前往后第一个匹配（历史前部逐字节不变 → 跨轮稳定）；
//	3. 均未命中 → 返回 -1（调用方按「锚点丢失」保守处理）。
//	 注：定位偏早只会让保留段更长（语义无损，缓存仍稳定）；偏晚才是致命
//	 （每轮重排 → 缓存全断）。
func handoffAnchorIndex(hist []Message, rec *HandoffRecord) int {
	if rec == nil || rec.Anchor == "" {
		return -1
	}
	parts := strings.Split(rec.Anchor, handoffAnchorSep)
	n := len(parts)
	if n == 0 {
		return -1
	}
	matchAt := func(i int) bool {
		if i < 0 || i+n > len(hist) {
			return false
		}
		for k := 0; k < n; k++ {
			if handoffFingerprint(hist[i+k]) != parts[k] {
				return false
			}
		}
		return true
	}
	// ① 生成时基线位置（精确）：MsgCount 即生成时的历史条数，
	//    基点 = MsgCount − 保留深度，并按生成时的「跳过孤立 tool」规则前移。
	if rec.MsgCount > 0 {
		idx := rec.MsgCount - keepForRelevance(rec.Relevance)
		if idx > len(hist) {
			idx = len(hist)
		}
		for idx > 0 && idx < len(hist) && hist[idx].Role == RoleTool {
			idx--
		}
		if matchAt(idx) {
			return idx
		}
	}
	// ② 首个匹配（历史前部不变 → 跨轮稳定）
	for i := 0; i+n <= len(hist); i++ {
		if matchAt(i) {
			return i
		}
	}
	return -1
}

// ShouldRefreshHandoff 判断是否需要重新生成交接（相对上次交接的增量是否达阈值）。
// 「增量」= 当前锚后内容 tokens − 生成时保留段基线（rec.KeptTokens），
// 即上次交接之后**新增**的内容量（不含刻意保留的最近消息）。
// 锚点丢失（历史被压缩/回滚/跨口径）→ 保守刷新（重建交接最稳）。
// 返回（是否刷新，增量 token 估算）。
func ShouldRefreshHandoff(history []Message, rec *HandoffRecord) (bool, int) {
	if rec == nil || strings.TrimSpace(rec.Text) == "" {
		return true, estimateTokens(stripSystemMsgs(history))
	}
	hist := stripSystemMsgs(history)
	if handoffAnchorIndex(hist, rec) < 0 {
		return true, estimateTokens(hist) // 锚点丢失：重建
	}
	incTokens := estimateTokens(handoffIncrement(history, rec))
	delta := incTokens - rec.KeptTokens
	if delta < 0 {
		delta = 0
	}
	return delta >= handoffRefreshThreshold(), delta
}

// ComposeHandoffView 组装注入视图：[交接消息] + [自 rec 基点起的增量消息（含基点）]。
// rec 为「本次交接记录」（复用时为旧记录；新生成时为刚保存的新记录——
// 增量=基点起的保留段）。
// 视图不含 system（由 Run 统一构建）；增量中的旧交接块会被跳过（视图内至多一份交接）。
func ComposeHandoffView(history []Message, rec *HandoffRecord, text string) []Message {
	inc := handoffIncrement(history, rec)
	out := make([]Message, 0, 1+len(inc))
	out = append(out, Message{Role: RoleUser, Content: text})
	for _, m := range inc {
		if isHandoffText(m.Content) {
			continue // 跳过历史中的旧交接块（新交接已覆盖其信息）
		}
		out = append(out, m)
	}
	return out
}

// handoffSample 构建 LLM 整理的输入文本（历史尾部取样 + 上次交接 + 当前任务）。
func handoffSample(history []Message, prev *HandoffRecord, task string) string {
	hist := stripSystemMsgs(history)
	start := 0
	if len(hist) > handoffInputMaxMsgs {
		start = len(hist) - handoffInputMaxMsgs
	}
	var b strings.Builder
	// 更早历史（取样上限之外）：规则式摘要并入输入（信息粗但全——防长历史
	// 在取样截断处整体丢失；近期部分仍按原文节选，保证关键细节进 LLM）。
	if start > 0 {
		early := hist[:start]
		b.WriteString("（更早 " + strconv.Itoa(len(early)) + " 条历史｜规则摘要）\n")
		b.WriteString(ruleSummarize(early) + "\n\n")
	}
	for _, m := range hist[start:] {
		if isHandoffText(m.Content) {
			continue
		}
		role := "工具"
		switch m.Role {
		case RoleUser:
			role = "用户"
		case RoleAssistant:
			role = "助手"
		}
		toolInfo := ""
		if len(m.ToolCalls) > 0 {
			toolInfo = " [工具调用]"
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			content = "（无正文）"
		}
		b.WriteString(role + ":" + toolInfo + " " + truncRunesAgent(content, handoffInputMsgRunes) + "\n")
	}
	if b.Len() == 0 {
		b.WriteString("（历史无可取内容）\n")
	}

	var out strings.Builder
	out.WriteString("你是对话交接助手。请阅读下方【即将继续的任务】与【对话历史节选】，输出一份《会话交接·提交消息》，供继续工作的 AI 助手快速恢复状态。\n\n")
	out.WriteString("要求：\n")
	out.WriteString("1. 先判断历史与「即将继续的任务」的相关性：高（同一任务的延续）/ 部分（同项目不同任务）/ 无关（全新任务）。无关或部分相关时，历史中与任务无关的细节只保留一句概述，不展开。\n")
	out.WriteString("2. 输出 Markdown 正文，包含以下小节（无内容的小节省略）：\n")
	out.WriteString("## 与当前任务的相关性\n（一行：高/部分/无关 + 一句话说明）\n")
	out.WriteString("## 任务目标\n## 已完成\n## 当前状态（关键文件、构建/测试结果、产物落点）\n## 关键决策与坑\n## 待完成 / 下一步\n")
	out.WriteString("3. 具体优于笼统：保留文件名、命令、结论、错误原因等可执行信息；不要编造历史中没有的内容。\n")
	out.WriteString("4. 全文不超过 700 字，只输出正文（不要额外解释、不要代码围栏）。\n\n")
	out.WriteString("【即将继续的任务】\n")
	out.WriteString(truncRunesAgent(task, handoffTaskRunes))
	out.WriteString("\n")
	if prev != nil && strings.TrimSpace(prev.Text) != "" {
		out.WriteString("\n【上一份交接（可作基线，无需重复其全文）】\n")
		out.WriteString(truncRunesAgent(prev.Text, handoffPrevTextRunes))
		out.WriteString("\n")
	}
	out.WriteString("\n【对话历史节选（尾部 " + strconv.Itoa(min(len(hist), handoffInputMaxMsgs)) + " 条）】\n")
	out.WriteString(b.String())
	return out.String()
}

// BuildHandoffText 调用一次 LLM 把历史整理成「会话交接·提交消息」；
// 失败（无 Provider / 报错 / 输出异常）回退规则式交接。
// 返回文本以 handoffTitle 开头（注入即用 + 系统注入消息识别标记）。
func BuildHandoffText(ctx context.Context, prov Provider, prev *HandoffRecord, history []Message, task string) string {
	body := ""
	if prov != nil {
		prompt := handoffSample(history, prev, task)
		msg, err := prov.Chat(ctx, []Message{{Role: RoleUser, Content: prompt}}, nil, nil)
		if err != nil {
			log.Printf("[handoff] LLM 整理失败（回退规则式）: %v", err)
		} else {
			if s := sanitizeHandoffBody(msg.Content); len([]rune(s)) >= 40 {
				body = s
			} else {
				log.Printf("[handoff] LLM 输出过短（清洗后 %d runes），回退规则式", len([]rune(s)))
			}
		}
	} else {
		log.Printf("[handoff] 无可用 Provider，使用规则式交接")
	}
	if body == "" {
		body = ruleHandoffFallback(history)
	}
	return handoffTitle + "\n" + body
}

// sanitizeHandoffBody 清洗 LLM 输出：去首尾空白、剥代码围栏、去重复标题行。
func sanitizeHandoffBody(s string) string {
	s = strings.TrimSpace(s)
	// 剥 ``` 围栏（模型偶发整段包裹）
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	// 去模型复述的标题行（首行如 "【会话交接·提交消息】"）
	if i := strings.Index(s, "\n"); i >= 0 && strings.Contains(s[:i], handoffTitle) {
		s = strings.TrimSpace(s[i+1:])
	}
	return s
}

// ruleHandoffFallback 规则式交接（LLM 不可用时兜底；复用既有摘要基建）。
func ruleHandoffFallback(history []Message) string {
	hist := stripSystemMsgs(history)
	var b strings.Builder
	b.WriteString("## 与当前任务的相关性\n（规则式兜底：未做 LLM 相关性判断，历史按时间顺序概述）\n\n")
	if s := stableTrailingSummary(hist); s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}
	b.WriteString("\n（本条为规则式交接：共 " + strconv.Itoa(len(hist)) + " 条历史消息已折叠，完整原文见会话存储。）")
	return b.String()
}

// BuildHandoffView 一站式会话交接：判断 →（生成或复用）→ 组装视图（含 B/C 语义复检）。
//
// 复用期内：自上次复检的增量达 handoffRecheckThreshold 且有判官实例时，
// 轻量判官复检相关性（B）；判官结论与记录不同 → **强制重生成**（C：保留段
// 深度随新相关性重定：高=16 / 部分=8 / 无关=4）；相同 → 仅推进复检基线。
// 生成（首次 / 刷新 / 判官触发）时：从交接文本解析相关性 → 按深度设锚；
// 判官结论优先于文本解析（关系判定以判官为准）。
//
// judge 为判官 Provider（独立轻量实例，non-thinking、极小 max_tokens）；
// nil = 跳过判官（续跑段边界等同一任务延续场景无需复检）。
// 返回 (view, true) 表示本次应使用交接视图；返回 (nil, false) 表示不启用/
// 未达阈值/存储不支持（调用方保持原逻辑：原样或按压力精简）。
// 副作用：达到刷新条件时调用一次 LLM 并持久化交接记录；调用方可据返回值发提示。
func BuildHandoffView(ctx context.Context, prov Provider, judge Provider, store ConversationStore, convID string, history []Message, task string, maxContextTokens int) ([]Message, bool) {
	if !HandoffEnabled() || convID == "" || store == nil || len(history) == 0 {
		return nil, false
	}
	ok, reason := ShouldHandoff(history, maxContextTokens)
	if !ok {
		return nil, false
	}
	hs, sok := store.(handoffStore)
	if !sok {
		return nil, false // 存储不支持交接记录（无法跨轮复用）→ 不启用
	}

	prev, lerr := hs.LoadHandoff(convID)
	if lerr != nil {
		log.Printf("[handoff] 读取交接记录失败 conv=%s: %v", convID, lerr)
	}
	var rec *HandoffRecord
	forceRefresh := false
	rel := ""
	if prev != nil && strings.TrimSpace(prev.Text) != "" {
		curRel := prev.Relevance
		if curRel == "" {
			curRel = handoffRelHigh // 旧记录无相关性字段 → 保守按高（不缩保留段）
		}
		if need, inc := ShouldRefreshHandoff(history, prev); !need {
			// ★ B：复用期定期语义复检（独立判官实例；增量达周期才调用）
			if judge != nil && inc-prev.RelCheckedInc >= handoffRecheckThreshold() {
				if newRel := handoffJudgeRelevance(ctx, judge, prev, task); newRel != "" && newRel != curRel {
					// ★ C：任务关系变化 → 强制重生成（保留段深度随新相关性重定）
					forceRefresh = true
					rel = newRel
					log.Printf("[handoff] conv=%s 判官复检：相关性 %s → %s（自复检增量 ~%d tokens），刷新交接",
						convID, curRel, newRel, inc)
				} else {
					if newRel != "" {
						log.Printf("[handoff] conv=%s 判官复检：保持 %s（自复检增量 ~%d tokens）", convID, newRel, inc)
					}
					prev.RelCheckedInc = inc // 无论成败都推进（防失败时每轮重试风暴）
					if serr := hs.SaveHandoff(convID, *prev); serr != nil {
						log.Printf("[handoff] 保存复检基线失败 conv=%s: %v", convID, serr)
					}
				}
			}
			if !forceRefresh {
				rec = prev // 增量未达刷新阈值 → 复用（不调 LLM）
				log.Printf("[handoff] conv=%s 复用上次交接（增量 ~%d tokens < 刷新阈值 %d）（%s）",
					convID, inc, handoffRefreshThreshold(), reason)
			}
		}
	}
	if rec == nil {
		// 首次 / 增量达刷新阈值 / 判官发现关系变化 → 生成新交接
		hctx := ctx
		var cancel context.CancelFunc
		if hctx == nil {
			hctx = context.Background()
		}
		hctx, cancel = context.WithTimeout(hctx, handoffTimeoutDefault)
		defer cancel()
		text := BuildHandoffText(hctx, prov, prev, history, task)
		if rel == "" {
			rel = parseRelevance(text) // 生成时从文本「相关性」小节解析
		}
		if rel == "" {
			rel = handoffRelHigh // 解析失败 → 保守按高（不缩保留段）
		}
		rec = &HandoffRecord{
			Text:      text,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Anchor:    handoffAnchorAt(history, keepForRelevance(rel)),
			MsgCount:  len(stripSystemMsgs(history)),
			Relevance: rel,
			// RelCheckedInc=0：新交接即新复检周期的起点
		}
		// 保留段基线：锚后内容 tokens（刷新判断只看相对基线的新增量）。
		rec.KeptTokens = estimateTokens(handoffIncrement(history, rec))
		if serr := hs.SaveHandoff(convID, *rec); serr != nil {
			log.Printf("[handoff] 保存交接记录失败 conv=%s: %v", convID, serr)
		}
		log.Printf("[handoff] conv=%s 已生成新交接（%d 条历史，相关性 %s，%s）", convID, rec.MsgCount, rel, reason)
	}
	view := ComposeHandoffView(history, rec, rec.Text)
	return view, true
}

// buildLoopHandoffView 续跑段边界的会话交接调用（Provider 优先压缩模型，其次主模型）。
// history 取 loop.History（本段结束时的完整时间线，Run defer 已更新）。
// judge=nil：续跑是同一任务的延续，任务关系不会突变——段边界无需判官复检
// （语义复检由新提交入口的 BuildHandoffView 携带独立判官实例完成）。
func buildLoopHandoffView(runCtx context.Context, l *Loop, store ConversationStore, convID, task string) ([]Message, bool) {
	var prov Provider
	if l.Compressor != nil {
		prov = l.Compressor
	} else {
		prov = l.getProvider()
	}
	return BuildHandoffView(runCtx, prov, nil, store, convID, l.History, task, l.MaxContextTokens)
}

// ── 插件化边界入口（2026-09-12：交接策略外置到 agentloop 插件）──
//
// 设计：执行位置在宿主（物理约束——用户输入边界在会话装配前；段边界在 Run 之外），
// 策略实现在插件（ctx.loopFactory.registerHandoff）。宿主在这两处调用注册实现，
// 未注册或执行失败 → 回退本文件上方 Go 默认实现（可回退，零回归）。
//
// ★ 语义要点（缓存铁律）：两个入口都是**跨轮边界**——
//   · 用户输入边界：上一轮已结束，重排历史发生在「尚不可复用的前缀」上；
//   · 段边界：上一段已结束、下一段尚未开始（同一条用户消息内的 step 循环之外），
//     不会切断轮内 append-only 的前缀。
//   一次 Run 的 step 之间**不做任何整理**（见 compress.go maybeCompact：段内只追加）。

// HandoffUserTurnView 用户输入边界（新提交对话 / 点继续执行）的整理视图：
// JS 实现（插件 registerHandoff.onUserTurn）优先；未注册或执行失败 → 回退 Go 默认。
// 返回 (view, applied, notice)；applied=false 时调用方保持原逻辑
// （原样历史或按 token 压力精简）。notice 非空时由调用方透传给用户。
func HandoffUserTurnView(ctx context.Context, prov, judge Provider, store ConversationStore,
	convID, workspaceRoot string, history []Message, task string, maxContextTokens int) ([]Message, bool, string) {
	if impl := CurrentJSHandoff(); impl != nil {
		view, ok, notice, err := impl.UserTurnView(convID, workspaceRoot, history, task,
			maxContextTokens, prov, judge, store)
		if err != nil {
			log.Printf("[handoff] JS 交接实现（用户输入边界）失败，回退 Go 默认: %v", err)
		} else {
			// ok=false = JS 明确判定「不启用」（未达阈值/已关闭）→ 不回退重复判定
			return view, ok, notice
		}
	}
	view, ok := BuildHandoffView(ctx, prov, judge, store, convID, history, task, maxContextTokens)
	return view, ok, ""
}

// HandoffSegmentView 段续跑边界（跨轮）的整理视图：JS 实现优先，回退 Go 默认。
// 返回的 view 作为下一段初始历史注入（loop.Run(runCtx, contMsg, view)）。
// Provider 取压缩模型（轻量）优先，其次主模型——与 Go 默认实现同口径。
func HandoffSegmentView(runCtx context.Context, l *Loop, store ConversationStore, convID, task string) ([]Message, bool, string) {
	if impl := CurrentJSHandoff(); impl != nil {
		var prov Provider
		if l.Compressor != nil {
			prov = l.Compressor
		} else {
			prov = l.getProvider()
		}
		view, ok, notice, err := impl.SegmentView(convID, l.WorkspaceRoot, l.History, task,
			l.MaxContextTokens, prov, store)
		if err != nil {
			log.Printf("[handoff] JS 交接实现（段边界）失败，回退 Go 默认: %v", err)
		} else {
			return view, ok, notice
		}
	}
	view, ok := buildLoopHandoffView(runCtx, l, store, convID, task)
	return view, ok, ""
}
