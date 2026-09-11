package agent

// 历史精简——按 companion 单循环裁剪：用启发式估算 + 单段摘要。
// ★ 本文件只负责「精简/摘要」的执行与数据。
//
// ★ 2026-09 保前缀精简（缓存修复，见 compact）：精简视图 = 开头 system 前缀
//   + N 个**不可变**占位块 + 首次精简写定的摘要块 + 最近 compactKeepRecent 条。
//   每次精简只在「占位块之后、最近段之前」追加一个新占位块——已发送过的字节
//   永不改写，相邻两次请求的公共前缀保持稳定，被保留的尾部无需重新 prefill
//   （旧实现删中段 = 前缀必断，实测单次白付 13K~54K tokens）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	compactRatio      = 0.90 // 重度精简触发阈值：90% 时全量精简（避免到 95% 急刹）
	compactRatioEarly = 0.45 // 预精简触发阈值：45% 时做轻度精简，让 Agent 更早感知早期上下文
	// compactKeepRecent 恒留最近条数。
	// ★ 2026-09 扩容 16 → 40：一次精简的代价 = 保留内容重新 prefill（缓存必断在精简点），
	//   keep 越大 → 触发频率越低、每段可用的无损上下文越多。实测
	//   （.pair/logs/llm-trace-20260911.jsonl）：keep=16 时 27 分钟触发 5 次精简，
	//   累计 miss 164K tokens（该会话 miss 的 22%）。
	compactKeepRecent      = 40 // 恒留最近条数
	compactKeepRecentEarly = 48 // 预精简保留条数：比全量精简更多，保持更多上下文
	compactMinDrop         = 2  // 中段可丢条数下限：太少不值得归纳
	compactLLMSlice        = 40 // LLM 摘要喂入的末尾非 system 条数上限（复刻参考 slice(-40)）
	// maxCompactSlots 精简占位块上限：每次精简追加一个不可变占位块（见 compact），
	// 达上限即停止再精简（宁可让窗口自然增长，也不无限追加占位块）。
	maxCompactSlots = 24
	// compactSummaryRunes 精简摘要文本上限（rune）：只写一次、之后字节不变，
	// 只需装下「原始目标锚点 + 已完成结论」。
	compactSummaryRunes = 500
	// compactMinKeep 保留段条数下限（极小窗口下也不能少于此，否则配对/连续性无保障）。
	compactMinKeep = 8
	// compactKeepTokenRatio 保留段的 token 预算占窗口比例：极小窗口下固定 keep
	// 会把精简阈值永久卡死（保留段本身就超阈值），按此比例收敛保留长度。
	compactKeepTokenRatio = 0.5
	// compactMinSavingRatio 精简收益下限：被删段需占当前上下文的比例（见 compactNotWorthwhile）。
	// 低于此比例的精简「删了也省不了多少」，跳过以避免压缩抖动与冷却浪费。
	compactMinSavingRatio = 0.25
)

// CompactHardFloor 绝对硬地板（token）：窗口配置超大（>此值）时相对阈值形同虚设，
// token 绝对量达硬地板即强制全量精简，保证单次注入有上限。
//
// ★ 2026-09 可配，**默认 0 = 关闭**（只用窗口比例阈值）：
//
//	历史问题：1M 窗口 + 120K 硬地板 → 上下文一过 120K（窗口的 12%）就被腰斩，
//	而一次腰斩 = 被保留历史全额重新 prefill（前缀缓存断在精简点）。实测最近会话
//	27 分钟内被腰斩 5 次、白付 164K input tokens（每次 miss 13K~54K），
//	而该模型窗口本身容得下 305K+（trace 实测未报错）。
//	现在默认交给窗口比例（45% 预精简 / 90% 全量精简）：想压低单请求成本 →
//	调小 contextMaxTokens；想恢复硬地板保护 → 设 PAIR_COMPACT_HARD_FLOOR=<token>。
func CompactHardFloor() int {
	if v := strings.TrimSpace(os.Getenv("PAIR_COMPACT_HARD_FLOOR")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
		log.Printf("[compact] PAIR_COMPACT_HARD_FLOOR=%q 非法（需 ≥0 整数），忽略", v)
	}
	return 0
}

// HardFloorExceeded 报告「硬地板已触发」（硬地板 ≤0 时恒 false）。
func HardFloorExceeded(tokens, maxContextTokens int) bool {
	floor := CompactHardFloor()
	return floor > 0 && maxContextTokens > floor && tokens >= floor
}

// CompactPolicy 精简策略摘要（启动/诊断日志用）。
func CompactPolicy(maxContextTokens int) string {
	floorStr := "关闭"
	if floor := CompactHardFloor(); floor > 0 {
		floorStr = strconv.Itoa(floor)
	}
	return fmt.Sprintf("预精简 %.0f%% / 全量精简 %.0f%% / 保留最近 %d 条 / 硬地板 %s（窗口 %d）",
		compactRatioEarly*100, compactRatio*100, compactKeepRecent, floorStr, maxContextTokens)
}

// compactPlaceholderText 精简占位块文本（★ 恒定：同一槽位文本在任何时刻字节相同）。
//
// 前缀缓存要求「已发送过的字节不被改写」：精简视图里每删掉一批中段消息，就在该
// 位置留下一个**不可变**占位块。后续精简只允许「在更靠后的位置追加新槽位」，
// 绝不修改已有槽位文本——相邻两次请求的公共前缀除新槽位外逐字节稳定，
// 被保留的尾部（最近 N 条 + 本轮新工具结果）无需重新 prefill。
func compactPlaceholderText(slot int) string {
	return fmt.Sprintf("【已精简历史 #%d｜占位】此前有一段较早的对话（用户指令与工具结果）已归档到会话存储，不再占用上下文。", slot)
}

// compactTrailingText 精简摘要块文本（★ 首次精简时写定，之后字节不变）。
func compactTrailingText(summary string) string {
	return "【历史背景摘要｜非当前任务】以下为较早轮次的要点归纳（完整原文已归档到会话存储）：\n\n" + summary
}

// stableTrailingSummary 生成「写定后不再变化」的摘要块正文（只取确定性锚点）：
// 原始目标（本 Run 归档里的第一条真实用户消息）+ 最后一条有正文的助手结论。
//
// ★ 只认「真实历史」：精简视图块（占位块/摘要块）虽然也是 user 角色，但它们是
// 重建产物而非原始指令，必须跳过——否则第二次精简会把上一个摘要块当成「原始
// 目标」，摘要块文本随之变化，前缀又断（这正是首版实现的 bug）。
// ★ 刻意**不含**工具调用次数/文件清单等会随每次精简变化的统计。
func stableTrailingSummary(archive []Message) string {
	var goal, lastAssistant string
	for _, m := range archive {
		if isCompactViewBlock(m) {
			continue
		}
		switch m.Role {
		case RoleUser:
			if goal == "" && !strings.HasPrefix(m.Content, backgroundCtxMarker) {
				goal = truncRunesAgent(m.Content, 200)
			}
		case RoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				lastAssistant = truncRunesAgent(m.Content, 300)
			}
		}
	}
	var b strings.Builder
	if goal != "" {
		b.WriteString("原始目标：" + goal + "\n")
	}
	if lastAssistant != "" {
		b.WriteString("此前进展：" + lastAssistant)
	}
	if b.Len() == 0 {
		b.WriteString("此前若干轮次的目标与结论（原文已归档）。")
	}
	return truncRunesAgent(b.String(), compactSummaryRunes)
}

// maybeCompact 段内不计精简——**只追加，不处理**（2026-09 决策）。
//
// 为什么不中途精简（历史结论 + 实测）：
//   · 段内删历史 = 把「已经发过的字节」改写掉，provider 前缀缓存必断在删改点，
//     被保留的内容要全额重新 prefill（实测单次 miss 13K~54K tokens）；
//   · 体积控制已有更强的两道闸：① 每 toolCallBudget 次（默认 120，可在设置面板「Agent」组配置）工具调用结束本段并自动续跑下一段
//     （tool_budget.go，同会话续跑）；② 下一段（下一次 Run）开始时从 store 重新
//     加载历史并做跨段精简（CondenseHistoryByPressure）。段内因此不会无限增长，
//     段内每一步又都能 100% 命中前缀缓存。
//
// 本函数现在只负责两件事：
//   1. 前端「精简」按钮（l.CompactRequested）——用户显式要求时才做一次精简；
//   2. 阈值诊断——超阈值时发一条 notice（不再改消息），提示已达窗口比例。
func (l *Loop) maybeCompact(ctx context.Context, msgs []Message) []Message {
	// 主动精简请求（前端精简按钮）：忽略阈值，直接精简一次
	if l.CompactRequested {
		l.CompactRequested = false
		if out, summary, dropped := l.compact(ctx, msgs); dropped > 0 {
			l.CompressedSummaries = append(l.CompressedSummaries, summary)
			l.emit(Event{Type: EventCompacted, Content: fmt.Sprintf("已精简早期 %d 条历史对话，保留最近 %d 条", dropped, prefixLen(out))})
			return out
		}
		return msgs
	}
	return msgs // ★ 段内只追加：不做任何自动精简
}

// maybeCompactLegacy 阈值驱动的段内精简（历史实现，保留供回归对照/将来需要时启用）。
// ★ 现行默认路径不调用它：见 maybeCompact 的说明（段内只追加）。
func (l *Loop) maybeCompactLegacy(ctx context.Context, msgs []Message) []Message {
	if l.MaxContextTokens <= 0 {
		return msgs // 未配置窗口上限 → 精简关闭
	}
	tokens := estimateTokens(msgs)
	if l.lastPromptTokens > tokens { // 取实测与估算较大者
		tokens = l.lastPromptTokens
	}
	// ★ 两档精简：45% 预精简（轻）+ 90% 全量精简（重）
	tokenRatio := float64(tokens) / float64(l.MaxContextTokens)
	// ★ 硬地板兜底（可配，默认关闭）：超大窗口配置下相对阈值形同虚设，
	//   显式配置硬地板时，token 绝对量达标即强制走全量精简分支。
	if HardFloorExceeded(tokens, l.MaxContextTokens) {
		tokenRatio = compactRatio
	}
	if tokenRatio < compactRatioEarly {
		return msgs // 未超任何阈值
	}
	// 预精简（45%-90%）：保留更多上下文，只留摘要数据
	if tokenRatio < compactRatio {
		return l.earlyCompact(ctx, msgs)
	}
	out, summary, dropped := l.compact(ctx, msgs)
	if dropped <= 0 {
		return msgs // 没精简成（中段太短，或全是要保的配对）
	}
	l.noteCompactSummary(summary)
	l.lastPromptTokens = 0 // 重置：精简后等下轮实测/重新估算
	l.emit(Event{Type: EventCompacted, Content: fmt.Sprintf("已精简早期 %d 条历史对话，保留最近 %d 条", dropped, prefixLen(out))})
	return out
}

// noteCompactSummary 记录一条精简摘要（数据留存）：最多保留 3 条，超出时把最旧的
// 两条合并，防止摘要列表无限膨胀。
func (l *Loop) noteCompactSummary(summary string) {
	const maxSummaries = 3
	if len(l.CompressedSummaries) >= maxSummaries {
		merged := "[历史摘要合并]\n" + l.CompressedSummaries[0] + "\n" + l.CompressedSummaries[1]
		if len(merged) > 600 {
			merged = merged[:600] + "…"
		}
		l.CompressedSummaries = append([]string{merged}, l.CompressedSummaries[2:]...)
	}
	l.CompressedSummaries = append(l.CompressedSummaries, summary)
	if len(l.CompressedSummaries) > maxSummaries {
		l.CompressedSummaries = l.CompressedSummaries[len(l.CompressedSummaries)-maxSummaries:]
	}
}

// compact 执行精简，返回「保前缀视图」：新 msgs、摘要文本、被精简条数。
//
// ★ 输入契约：传入的是**真实历史流**（会话消息），不是上一次的精简视图。
// 视图块（占位块/摘要块）带 compactSlotPrefix/compactTrailingPF 前缀，识别逻辑
// （isCompactViewBlock / stableTrailingSummary）会跳过它们，因此即使误传视图也
// 不会把块当成真实历史——但归档与「原始目标锚点」都以真实历史为权威来源。
//
// ★ 2026-09 保前缀重写（缓存修复，替代旧「删中段」实现）：
//
//	旧实现 out = System + 最近 16 条：每次精简都把中段整体删掉，于是「上一次请求的
//	前 N 条」与「本次请求的前 N 条」在第 2 条就分叉 → provider 前缀缓存全断，
//	保留下来没变的尾部也要全额重新 prefill（实测每次 miss 13K~54K tokens）。
//
//	新实现把被删段替换为**不可变占位块**，并在首次精简时写定一个
//	「之后字节与位置都不再变」的摘要块（保留原始目标锚点与早前结论）：
//
//	  视图 = [System…] + [占位#1..#P] + [摘要块（首次写定）] + [占位#P+1..#N] + 最近 N 条
//
//	占位块文本只取决于槽位号（compactPlaceholderText），摘要块只在第一次精简时写入、
//	位置随后钉死；因此后续每次精简只在摘要块**之后**追加一个新槽位，前缀逐字节单调
//	延展，被保留的尾部与摘要块全部命中缓存 → 精简代价从「整窗重新 prefill」
//	降到「新增占位块的几十个 token」（实测单次 13K~54K → 约 0.05K）。
//
// 配对保护：保留段不以孤立 tool 结果开头（其 assistant tool_call 若被精简，结果成孤儿，
// OpenAI 规范报错）——此时把前移跨过的 tool 条一起并入被精简段。
func (l *Loop) compact(ctx context.Context, msgs []Message) ([]Message, string, int) {
	prefix := prefixLen(msgs) // 开头连续 system 条数（恒留，且必须在最前）
	// ★ 边界 = max(按 keep 算出的起点, 上一次已提交的尾部起点)：
	//   取 max 保证边界**只向前推进**（绝不回退），且保留段长度被 keep 约束
	//   （视图不会在两次精简之间无限增长）。上一次视图里保留下来的尾部消息
	//   仍以原字节出现在新视图里——前缀缓存只丢边界推进掉的那几条。
	computeFrom := l.retainedStart(msgs, l.effectiveKeep(msgs, compactKeepRecent))
	archiveFrom := l.compactUnsentFrom
	if archiveFrom < prefix || archiveFrom > len(msgs) {
		archiveFrom = prefix
	}
	keepFrom := computeFrom
	if keepFrom < archiveFrom {
		keepFrom = archiveFrom // 边界不回退
	}
	if keepFrom < prefix {
		keepFrom = prefix
	}
	// 配对保护：保留段不能以孤立 tool 结果开头（其 assistant tool_call 被归档会破坏
	// OpenAI 规范）——把跨过的 tool 条一起并入归档。
	for keepFrom < len(msgs) && msgs[keepFrom].Role == RoleTool {
		keepFrom++
	}
	dropped := msgs[archiveFrom:keepFrom]
	if len(dropped) < compactMinDrop {
		return msgs, "", 0 // 中段太短，不值得精简
	}
	// 占位块上限：达上限后不再精简（宁可让窗口自然增长，也不无限追加占位块）。
	if l.compactSlots >= maxCompactSlots {
		return msgs, "", 0
	}
	// ★ 收益守卫：精简后视图 ≈ [system+占位块+摘要块] + 最近段。若保留段本身
	//   已占当前上下文的大头，精简不会真正缩小注入体积——此时**不能**精简，
	//   否则会白烧一次止损机会（冷却），随后每步重复触发，形成压缩抖动。
	if l.compactNotWorthwhile(msgs, keepFrom, prefix) {
		return msgs, "", 0
	}
	summary := l.summarize(ctx, dropped)
	// ★ 归档被删消息：精简视图仅供 LLM 提交；落盘/展示线由 fullHistory 还原
	//   （prefix + 归档 + 视图尾部 = 完整时间线），防止精简版覆盖 store。
	l.archiveCompactDropped(dropped)
	// 只在首次精简时写定摘要块文本（之后必须字节不变＝缓存命中前提），
	// 且只用确定性锚点（stableTrailingSummary），不含会随精简变化的统计。
	// 输入用「归档全量」而非本次 dropped：这样在任何一次精简触发时，
	// 同一个 Run 都能得到同一份写定文本（不受触发时机影响）。
	firstTrailing := l.compactTrailing == ""
	if firstTrailing {
		l.compactTrailing = compactTrailingText(stableTrailingSummary(l.compactArchive))
		l.compactTrailingPinned = true // 首次：摘要块钉在槽位 #1 之后
		l.compactSlotsAtPin = 0        // 与 buildCompactView 的顺序 (槽位→摘要→新槽位) 对齐
	}
	l.compactSlots++ // 追加一个新槽位（有序、只增不改）
	return l.buildCompactView(msgs, keepFrom), summary, len(dropped)
}

// buildCompactView 组装精简视图（★ 追加语义，缓存命中的关键）。
//
// 顺序不变量：一旦某条块被发送过，它的下标就不能再变——
//
//	[System…] + [占位#1..P] + [摘要块] + [占位#P+1..N] + [最近段]
//
// 其中 P = 首次写定摘要块时已存在的槽位数（compactSlotsAtPin）：
//   · 首次精简：P=0 → [system] + [摘要块] + [新槽位#1] + [最近段]
//   · 后续精简：槽位 #1..P 与摘要块的位置全部保持，新槽位一律追加在摘要块**之后**
//     （若插在摘要块之前，摘要块整体后移一位＝已缓存的整段尾部作废）。
func (l *Loop) buildCompactView(msgs []Message, keepFrom int) []Message {
	prefix := prefixLen(msgs)
	// ★ 视图保留「从边界到当前末尾」的连续段：keepFrom 由调用方给出（压缩时的
	//   边界已做过配对保护处理，必须与归档范围严格对齐，不能在此重算）。
	if keepFrom < prefix || keepFrom > len(msgs) {
		keepFrom = l.retainedStart(msgs, l.effectiveKeep(msgs, compactKeepRecent))
	}
	if keepFrom < prefix {
		keepFrom = prefix
	}
	total := l.compactSlots
	pinned := l.compactSlotsAtPin
	if pinned > total {
		pinned = total
	}
	out := make([]Message, 0, prefix+total+1+len(msgs)-keepFrom)
	out = append(out, msgs[:prefix]...) // ① system 前缀（恒不变）
	for i := 1; i <= pinned; i++ {
		out = append(out, Message{Role: RoleUser, Content: compactPlaceholderText(i)}) // ② 摘要块之前的槽位（位置恒定）
	}
	if l.compactTrailing != "" {
		out = append(out, Message{Role: RoleUser, Content: l.compactTrailing}) // ③ 摘要块（写定后字节与位置都不变）
	}
	for i := pinned + 1; i <= total; i++ {
		out = append(out, Message{Role: RoleUser, Content: compactPlaceholderText(i)}) // ④ 摘要块之后追加的槽位
	}
	out = append(out, msgs[keepFrom:]...) // ⑤ 保留段（连续、从边界到末尾）
	// ★ 记账：下一次精简的边界就从这条未发送消息开始（缓存对齐，仅 Run 内有效）
	l.compactUnsentFrom = keepFrom
	l.compactUnsentValid = true
	return out
}

// archiveCompactDropped 把新被精简的消息记入归档（已归档过的部分不重复追加），
// 并在首次归档时记下「归档首条」的标题作为 fullHistory 的起点锚点。
func (l *Loop) archiveCompactDropped(dropped []Message) {
	if len(dropped) == 0 {
		return
	}
	l.compactArchive = append(l.compactArchive, dropped...)
}

// ── 精简视图识别（保前缀：视图里的占位块/摘要块在还原与落盘时必须剔除）──

const (
	compactSlotPrefix = "【已精简历史 #" // 占位块前缀（compactPlaceholderText）
	compactTrailingPF = "【历史背景摘要｜非当前任务】"
)

// isCompactViewBlock 判断消息是否为精简视图块（占位块 / 摘要块）。
// 识别用前缀匹配（不依赖 Go 侧字段，因为 JS 循环路径的消息经 goja 往返）。
func isCompactViewBlock(m Message) bool {
	return strings.HasPrefix(m.Content, compactSlotPrefix) ||
		strings.HasPrefix(m.Content, compactTrailingPF)
}

// fullHistory 还原完整时间线（落盘/展示线的唯一出口）：
//
//	完整历史 = [system…] + [compactArchive（所有被替代的真实消息，按时间序）] + [视图未发送段]
//
// 不变量：compact 每次把「从 compactUnsentFrom 到新保留起点」的真实消息全部追加进归档，
// 而视图里保留段 = [compactUnsentFrom .. 末尾]（旧消息逐字节保留、新消息追加末尾）。
// 因此「归档 + 保留段」恰好覆盖所有真实消息，且顺序与原始时间线一致。
// 精简视图块（占位块/摘要块）只是视图产物，绝不进入落盘与前端展示。
func (l *Loop) fullHistory(msgs []Message) []Message {
	if len(l.compactArchive) == 0 {
		return msgs
	}
	prefix := prefixLen(msgs)
	out := make([]Message, 0, prefix+len(l.compactArchive)+len(msgs))
	out = append(out, msgs[:prefix]...)    // ① system 前缀（真实历史）
	out = append(out, l.compactArchive...) // ② 被替代的真实消息（含最初任务）
	// ③ 视图里「从未发送过」的尾部消息（按原顺序；视图块跳过）
	start := 0
	for start < len(msgs) && msgs[start].Role == RoleSystem {
		start++
	}
	if l.compactUnsentFrom > start && l.compactUnsentFrom <= len(msgs) {
		start = l.compactUnsentFrom
	}
	for _, m := range msgs[start:] {
		if isCompactViewBlock(m) {
			continue // 视图产物，不是真实历史
		}
		out = append(out, m)
	}
	return out
}

// persist 统一持久化出口：还原完整时间线后写盘（防精简视图覆盖 store）。
func (l *Loop) persist(msgs []Message) {
	if l.OnBatchPersist == nil {
		return
	}
	l.OnBatchPersist(l.fullHistory(msgs))
}

// earlyCompact 预精简：在 45% 阈值时做轻度精简——用规则式摘要（不调 LLM，省 token）
// 留下「原始目标 + 已完成结论」，同样采用保前缀视图（占位块 + 摘要块 + 最近段）。
// 保留 compactKeepRecent 条最近消息；冷却更短（3 轮），允许在持续增长时再次触发。
func (l *Loop) earlyCompact(ctx context.Context, msgs []Message) []Message {
	prefix := prefixLen(msgs)
	archiveFrom := l.compactUnsentFrom
	if archiveFrom < prefix || archiveFrom > len(msgs) {
		archiveFrom = prefix
	}
	// 边界口径与全量精简一致；预精简保留更多（keepEarly）。
	keepFrom := l.retainedStart(msgs, l.effectiveKeep(msgs, compactKeepRecentEarly))
	if keepFrom < archiveFrom {
		keepFrom = archiveFrom
	}
	if keepFrom < prefix {
		keepFrom = prefix
	}
	for keepFrom < len(msgs) && msgs[keepFrom].Role == RoleTool {
		keepFrom++
	}
	dropped := msgs[archiveFrom:keepFrom]
	if len(dropped) < compactMinDrop {
		return msgs
	}
	if l.compactSlots >= maxCompactSlots {
		return msgs // 占位块达上限 → 不再精简
	}
	// 规则式简介（不用 LLM，节省 token）
	summary := ruleSummarize(dropped)
	l.CompressedSummaries = append(l.CompressedSummaries, summary)
	if len(l.CompressedSummaries) > 3 {
		l.CompressedSummaries = l.CompressedSummaries[len(l.CompressedSummaries)-3:]
	}
	// ★ 保前缀：归档被精简段 + 追加一个占位块（摘要块若尚未写定则本次写定），
	//   返回与全量精简同一套视图规则，避免两条路径产生不同前缀口径。
	l.archiveCompactDropped(dropped)
	firstTrailing := l.compactTrailing == ""
	if firstTrailing {
		l.compactTrailing = compactTrailingText(stableTrailingSummary(l.compactArchive))
		l.compactTrailingPinned = true
		l.compactSlotsAtPin = 0
	}
	l.compactSlots++
	l.emit(Event{Type: EventCompacted, Content: fmt.Sprintf("已整理 %d 条早期对话为背景摘要", len(dropped))})
	return l.buildCompactView(msgs, keepFrom)
}

// effectiveKeep 计算实际保留条数：keep 是「按条数」的目标，但极小窗口下
// 固定 keep（如 40 条 ≈ 上万 token）会永远大于阈值 → 精简被永久卡死、上下文失控。
// 因此再按 token 预算收敛一次：保留段不超过窗口的一半（也不少于 compactMinKeep 条）。
func (l *Loop) effectiveKeep(msgs []Message, keep int) int {
	if keep < compactMinKeep {
		keep = compactMinKeep
	}
	if keep >= len(msgs) {
		keep = len(msgs) - 1
	}
	if l.MaxContextTokens > 0 {
		budget := float64(l.MaxContextTokens) * compactKeepTokenRatio
		kept, n := 0.0, 0
		for i := len(msgs) - 1; i >= 0 && n < keep; i-- {
			t := estimateTokensFloat([]Message{msgs[i]})
			if n > 0 && kept+t > budget {
				break // 再往前就超预算：保留段到此为止
			}
			kept += t
			n++
		}
		if n < keep {
			keep = n
		}
	}
	if keep < compactMinKeep {
		keep = compactMinKeep
	}
	return keep
}

// retainedStart 返回本次精简「保留段」的起点下标（视图里首条被提交的真实消息）：
//
//	保留段 = 最近 keep 条（keep 由 effectiveKeep 按窗口比例收敛）
//
// ★ 口径说明：保留段必须是**历史消息本身**（切片，不重写、不插块），
// 相邻两次视图里同一批消息的字节完全一致；精简只在「保留段起点前移」时
// 让旧视图前缀失效——一次只前移「最近 keep 条」那么多，而不是每步重排。
// （旧实现每步重取最近 N 条 ⇒ 每步整段重付 prefill；现在只在精简步付一次。）
func (l *Loop) retainedStart(msgs []Message, keep int) int {
	prefix := prefixLen(msgs)
	start := len(msgs) - keep
	if start < prefix {
		start = prefix
	}
	return start
}

// compactNotWorthwhile 报告「精简无收益」：被移除内容太少，压缩后体积仍然庞大。
// 判据：被删段的 token 量达不到「当前上下文 + 精简视图」的 compactMinSavingRatio 时，
// 精简换不来体积下降——跳过它，保住冷却窗口（避免每步重复触发压缩抖动）。
func (l *Loop) compactNotWorthwhile(msgs []Message, keepFrom, prefix int) bool {
	if keepFrom <= prefix || keepFrom > len(msgs) {
		return true
	}
	droppedTokens := estimateTokensFloat(msgs[prefix:keepFrom])
	totalTokens := estimateTokensFloat(msgs)
	if totalTokens <= 0 {
		return true
	}
	// 被删段占比需 ≥ compactMinSavingRatio；否则「删了也省不了多少」。
	return droppedTokens/totalTokens < compactMinSavingRatio
}

// summarize 把丢弃的中段消息归纳成一条摘要文本（带标记前缀）。
// Compressor 非空→LLM 摘要（原版 prompt），失败/空/无 Compressor→规则式摘要。
func (l *Loop) summarize(ctx context.Context, dropped []Message) string {
	if l.Compressor != nil {
		if s := l.llmSummarize(ctx, dropped); s != "" {
			return "[历史对话摘要 — LLM]\n\n" + s
		}
	}
	return "[历史对话摘要 — 规则]\n\n" + ruleSummarize(dropped)
}

// llmSummarize 用摘要模型生成智能摘要——复刻参考 manager.ts:1126-1150：
// 取末 compactLLMSlice 条非 system、每条裁 200 字、role 中文化，套原版 prompt，单条 user 消息无工具。
func (l *Loop) llmSummarize(ctx context.Context, dropped []Message) string {
	start := 0
	if len(dropped) > compactLLMSlice {
		start = len(dropped) - compactLLMSlice
	}
	var conv strings.Builder
	for _, m := range dropped[start:] {
		if m.Role == RoleSystem {
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
		conv.WriteString(role + ":" + toolInfo + " " + truncRunesAgent(m.Content, 200) + "\n")
	}
	prompt := "你是对话记录整理助手。根据以下对话内容，生成一份对话摘要。请提炼：用户原始目标、已完成的关键操作与结论、待办事项、关键的推理要点。忽略错误输出。简洁，200 字以内。\n\n" +
		conv.String() + "\n---\n对话摘要："
	msg, err := l.Compressor.Chat(ctx, []Message{{Role: RoleUser, Content: prompt}}, nil, nil)
	if err != nil {
		return "" // 摘要失败 → 回退规则式（复刻参考 catch）
	}
	s := strings.TrimSpace(msg.Content)
	if len([]rune(s)) < 10 { // 太短视作失败（复刻参考 length>10 判定）
		return ""
	}
	return s
}

// ruleSummarize 规则式拼接摘要（无摘要模型/摘要失败时的兜底）——复刻参考 manager.ts:1157-1189
// 的小节式结构，按 companion 可得信息裁剪：目标 / 进展 / 关键上下文 / 下一步。
func ruleSummarize(dropped []Message) string {
	var goal, lastAssistant string
	toolCalls, toolErrors := 0, 0
	files := map[string]bool{}
	for _, m := range dropped {
		switch m.Role {
		case RoleUser:
			if goal == "" && !strings.HasPrefix(m.Content, "[历史对话摘要") {
				goal = truncRunesAgent(m.Content, 200) // 首条用户消息 ≈ 原始目标
			}
		case RoleAssistant:
			if strings.TrimSpace(m.Content) != "" {
				lastAssistant = truncRunesAgent(m.Content, 400) // 末条有正文的助手消息 ≈ 关键上下文
			}
			toolCalls += len(m.ToolCalls)
			for _, tc := range m.ToolCalls {
				if p := argPath(tc.Function.Arguments); p != "" {
					files[p] = true
				}
			}
		case RoleTool:
			if strings.HasPrefix(strings.TrimSpace(m.Content), "Error:") {
				toolErrors++
			}
		}
	}
	var b strings.Builder
	if goal != "" {
		b.WriteString("## 目标\n" + goal + "\n\n")
	}
	fmt.Fprintf(&b, "## 进展\n已完成 %d 次工具调用", toolCalls)
	if toolErrors > 0 {
		fmt.Fprintf(&b, "（其中 %d 次错误）", toolErrors)
	}
	b.WriteString("。\n")
	if len(files) > 0 {
		b.WriteString("涉及文件: " + strings.Join(sortedKeys(files), ", ") + "\n")
	}
	if lastAssistant != "" {
		b.WriteString("\n## 关键上下文\n" + lastAssistant + "\n")
	}
	b.WriteString("\n## 下一步\n继续执行未完成的任务。")
	return b.String()
}

// estimateTokens 估算消息列表 token 数——复刻参考 manager.ts 启发式（无 tiktoken）：
// CJK 字 ×1.5、其余字符 ×0.25、每条 +4 开销、工具参数 ×0.25+8、tool_call_id +6。向上取整。
func estimateTokens(msgs []Message) int {
	total := 0.0
	for _, m := range msgs {
		total += 4
		total += textTokens(m.Content)
		total += textTokens(m.Reasoning)
		for _, tc := range m.ToolCalls {
			total += textTokens(tc.Function.Name)
			total += float64(len([]rune(tc.Function.Arguments)))*0.25 + 8
		}
		if m.ToolCallID != "" {
			total += 6
		}
	}
	return int(math.Ceil(total))
}

// textTokens 文本 token 估算：CJK 字 ×1.5、其余 ×0.25（复刻参考 CJK/拉丁分计）。
func textTokens(s string) float64 {
	cjk, other := 0, 0
	for _, r := range s {
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
	}
	return float64(cjk)*1.5 + float64(other)*0.25
}

// isCJK 是否中日韩表意字（复刻参考正则 /[一-鿿㐀-䶿豈-﫿]/ 的 Unicode 区间）。
func isCJK(r rune) bool {
	switch {
	case r >= 0x4E00 && r <= 0x9FFF: // CJK 统一表意
		return true
	case r >= 0x3400 && r <= 0x4DBF: // 扩展 A
		return true
	case r >= 0xF900 && r <= 0xFAFF: // 兼容表意
		return true
	}
	return false
}

// prefixLen 开头连续 system 消息条数（精简时恒留前缀，利于 KV 缓存复用）。
func prefixLen(msgs []Message) int {
	n := 0
	for n < len(msgs) && msgs[n].Role == RoleSystem {
		n++
	}
	return n
}

// argPath 从工具参数 JSON 取 path 字段（规则摘要列已访问文件用）。
func argPath(argsJSON string) string {
	var m map[string]any
	if json.Unmarshal([]byte(argsJSON), &m) == nil {
		if p, ok := m["path"].(string); ok {
			return strings.TrimSpace(p)
		}
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// EstimateBreakdown 估算 prompt token 的构成类别。返回 PromptBreakdown。
// actualPromptTokens > 0 时用其归一化（从 API 拿到精确值后调和），否则用内部估算。
// 返回的各分量之和 ≈ prompt tokens，用于前端渲染构成占比条。
func EstimateBreakdown(msgs []Message, tools []ToolDefinition, actualPromptTokens int) (pb PromptBreakdown) {
	if len(msgs) == 0 {
		return
	}

	// ── 用 textTokens 估算各类原始 token 数 ──
	sysCoreRaw, skillsRaw, histRaw, userRaw, toolRaw, mcpRaw := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0

	// 1. System prompt（首条 role=system）— 拆为 核心提示词 和 Skills 两段
	if msgs[0].Role == RoleSystem {
		content := msgs[0].Content
		// 定位 "可用技能" 标记，其之前为核心提示词，之后为 skills
		skillIdx := strings.Index(content, "可用技能")
		if skillIdx >= 0 {
			sysCoreRaw = textTokens(content[:skillIdx])
			skillsPart := content[skillIdx:]
			// skills 到下一个章节 "##" 或 "# " 处结束
			if end := strings.Index(skillsPart[1:], "##"); end >= 0 {
				skillsPart = skillsPart[:end+1]
			} else if end := strings.Index(skillsPart[1:], "\n# "); end >= 0 {
				skillsPart = skillsPart[:end+1]
			}
			skillsRaw = textTokens(skillsPart)
		} else {
			sysCoreRaw = textTokens(content) // 无 skills 段 → 全部为核心提示词
		}
	}

	// 2. 历史消息（索引 1～倒数第 1）
	for i := 1; i < len(msgs)-1; i++ {
		histRaw += estimateTokensFloat([]Message{msgs[i]})
	}

	// 3. 最后一条 user 消息（当前输入）
	if len(msgs) > 1 {
		userRaw = estimateTokensFloat([]Message{msgs[len(msgs)-1]})
	} else if len(msgs) == 1 {
		userRaw = estimateTokensFloat(msgs)
	}

	// 4. 内置工具定义
	if len(tools) > 0 {
		// 分离 MCP 工具和普通工具
		builtinTools := make([]ToolDefinition, 0, len(tools))
		mcpTools := make([]ToolDefinition, 0, len(tools))
		for _, t := range tools {
			if strings.HasPrefix(t.Function.Name, "mcp_") {
				mcpTools = append(mcpTools, t)
			} else {
				builtinTools = append(builtinTools, t)
			}
		}
		if len(builtinTools) > 0 {
			if data, err := json.Marshal(builtinTools); err == nil {
				toolRaw = textTokens(string(data))
			}
		}
		if len(mcpTools) > 0 {
			if data, err := json.Marshal(mcpTools); err == nil {
				mcpRaw = textTokens(string(data))
			}
		}
	}

	// ── 确定归一化基准 promptTokens ──
	promptTokens := actualPromptTokens
	if promptTokens <= 0 {
		promptTokens = int(estimateTokensFloat(msgs) + toolRaw + mcpRaw)
		if promptTokens <= 0 {
			promptTokens = 1000 // 兜底
		}
	}

	// ── 原始总量（各分量不重叠）──
	totalRaw := sysCoreRaw + skillsRaw + histRaw + userRaw + toolRaw + mcpRaw
	if totalRaw <= 0 {
		pb.SystemTokens = promptTokens // 实在算不出就全归系统
		return
	}

	// ── 等比缩放到 promptTokens ──
	ratio := float64(promptTokens) / totalRaw
	pb.SystemTokens = int(sysCoreRaw * ratio)
	pb.SkillsTokens = int(skillsRaw * ratio)
	pb.HistoryTokens = int(histRaw * ratio)
	pb.OtherTokens = int(userRaw * ratio)
	pb.ToolTokens = int(toolRaw * ratio)
	pb.MCPTokens = int(mcpRaw * ratio)

	// 因取整可能差 ±1~3，调整到 SystemTokens（最大分量）
	total := pb.SystemTokens + pb.SkillsTokens + pb.HistoryTokens + pb.OtherTokens + pb.ToolTokens + pb.MCPTokens
	if diff := promptTokens - total; diff > 0 {
		pb.SystemTokens += diff
	} else if diff < 0 {
		if pb.SystemTokens+diff >= 0 {
			pb.SystemTokens += diff
		}
	}
	return
}

// estimateTokensFloat 同 estimateTokens 但返回 float64（内部计算用，避免反复取整）。
func estimateTokensFloat(msgs []Message) float64 {
	total := 0.0
	for _, m := range msgs {
		total += 4
		total += textTokens(m.Content)
		total += textTokens(m.Reasoning)
		for _, tc := range m.ToolCalls {
			total += textTokens(tc.Function.Name)
			total += float64(len([]rune(tc.Function.Arguments)))*0.25 + 8
		}
		if m.ToolCallID != "" {
			total += 6
		}
	}
	return total
}

// NormalizeBreakdown 把估算的各分量按实际 prompt_tokens 等比缩放（确保总和 ≈ promptTokens）。
func NormalizeBreakdown(pb PromptBreakdown, promptTokens int) PromptBreakdown {
	total := pb.SystemTokens + pb.SkillsTokens + pb.MCPTokens + pb.ToolTokens + pb.HistoryTokens + pb.OtherTokens
	if total <= 0 || promptTokens <= 0 {
		return pb
	}
	ratio := float64(promptTokens) / float64(total)
	// 先归一化各分量
	return PromptBreakdown{
		SystemTokens:  int(float64(pb.SystemTokens) * ratio),
		SkillsTokens:  int(float64(pb.SkillsTokens) * ratio),
		MCPTokens:     int(float64(pb.MCPTokens) * ratio),
		ToolTokens:    int(float64(pb.ToolTokens) * ratio),
		HistoryTokens: int(float64(pb.HistoryTokens) * ratio),
		OtherTokens:   int(float64(pb.OtherTokens) * ratio),
	}
}

// truncRunesAgent 按 rune 截断（去首尾空白），超长加省略号。
func truncRunesAgent(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
