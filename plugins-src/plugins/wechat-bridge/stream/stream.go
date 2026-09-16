// Package stream 流式回复会话（StreamSession）。
//
// 职责：把 PairCode /ws 事件流中某会话一次的 assistant 输出，实时切分成「片」
// 并通过 OnFlush 回调发出（微信侧追加式消息），并处理：
//   - 首片尽早发出（首个句末 或 首块后 FirstMaxWaitMs）
//   - 中间轮防御：tool_call 事件到来时丢弃「未发送」缓冲（工具轮的碎碎念不发用户）
//   - done 终态：用 done.content 全文校验（前缀比对），漏发的补差、不重复发
//   - 发送串行化（内部消费 goroutine），保证片序
//
// 蓝图：M1 Node 原型 temp/wx-bridge/src/stream.mjs（线上实测版本）。
// 字符计数：所有阈值按 rune（字符）计——与 JS 对中文的行为一致。
//
// 并发契约（与审核结论）：
//   - 共享状态（buffer/sentText/state/flushes…）一律在 mu 保护下读写；
//   - 消费 goroutine **零锁**：只读发送队列，避免与持锁入队方互等（无死锁）；
//   - Abort 语义：调用方持锁时清空未消费队列（abort 后不再发送），随后
//     只入队完成标记；「已开始发送的片」不中断（对齐 stream.mjs 行为）；
//   - goroutine 生命周期：消费 goroutine 恰在完成标记后退出并发 Done 信号。
//     调用方契约：每个 Session 的任意退出路径必须到达 finalize/Abort 之一
//     （桥侧用 defer session.Abort() 兜底；Abort 幂等）。
package stream

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

// sentenceEnd 句末字符集（对齐 stream.mjs SENTENCE_END）。
const sentenceEnd = "。！？!?；;\n"

func isSentenceEnd(r rune) bool {
	return strings.ContainsRune(sentenceEnd, r)
}

// Event 事件流消息（/ws JSON 直接反序列化）。
type Event struct {
	Type    string          `json:"type"`
	Content string          `json:"content"`
	Tool    string          `json:"tool"`
	Args    string          `json:"args"`
	ConvID  string          `json:"convId"`
	Turn    json.RawMessage `json:"turn"`
	Step    json.RawMessage `json:"step"`
}

// FlushMeta 单片发送元信息。
type FlushMeta struct {
	N     int    // 第几片（1 起）
	First bool   // 是否首片
	Len   int    // 字符数
	Final bool   // 是否收尾片
	Turn  string // 事件流 turn（首块记录）
	Step  string // 事件流 step
}

// Summary 会话完成摘要。
type Summary struct {
	Text      string // 已发送累计文本
	Flushes   int
	ToolCalls int
	Reason    string // done | error | aborted
	Aborted   bool
}

// Options 会话参数（零值 = 默认）。
type Options struct {
	FirstMinChars  int // 首片最小字符（达到且遇句末即发）
	FirstMaxWaitMs int // 首块起多久必发首片
	NextMinChars   int // 后续片最小字符
	NextMaxWaitMs  int // 上片起多久必发下一片
	SoftLimit      int // 单片软上限（超限强切）

	// OnFlush 发送一片（在内部消费 goroutine 中串行调用；实现须并发安全对待
	// 调用顺序即可——本包保证同会话内串行）。
	OnFlush func(text string, meta FlushMeta)
	// OnDone 完成回调（全部片发送后调用一次；abort 时不调用）。
	OnDone func(full string, sum Summary)
	// OnToolCall 工具调用观察回调。
	OnToolCall func(tool, args string)
	// Logf 可选日志。
	Logf func(format string, args ...any)
}

// 会话状态。
const (
	stateWaiting   = "waiting"
	stateStreaming = "streaming"
	stateDone      = "done"
	stateAborted   = "aborted"
)

// Session 一次 assistant 输出的流式切片会话。零值不可用，用 New 创建。
type Session struct {
	mu     sync.Mutex
	opts   Options
	tuning Options // 生效参数（填充默认值后）

	buffer    string // 未发送的增量
	sentText  string // 已发送累计（done 前缀校验用）
	state     string
	startedAt time.Time // 首个 content 到达时刻
	lastFlush time.Time
	flushes   int
	toolCalls int
	metaTurn  string
	metaStep  string

	timer *time.Timer

	flushCh chan flushItem
	doneCh  chan Summary
}

type flushItem struct {
	text  string
	meta  FlushMeta
	final bool // 完成标记：执行后 OnDone + Done 信号
	sum   Summary
}

// New 创建会话并启动内部发送消费 goroutine。
func New(opts Options) *Session {
	t := opts
	if t.FirstMinChars <= 0 {
		t.FirstMinChars = 20
	}
	if t.FirstMaxWaitMs <= 0 {
		t.FirstMaxWaitMs = 1000
	}
	if t.NextMinChars <= 0 {
		t.NextMinChars = 160
	}
	if t.NextMaxWaitMs <= 0 {
		t.NextMaxWaitMs = 2500
	}
	if t.SoftLimit <= 0 {
		t.SoftLimit = 1500
	}
	s := &Session{
		opts:    opts,
		tuning:  t,
		state:   stateWaiting,
		flushCh: make(chan flushItem, 1024),
		doneCh:  make(chan Summary, 1),
	}
	go s.consumeLoop()
	return s
}

// Done 返回完成信号通道（finalize/Abort 后收到恰好一个摘要）。
func (s *Session) Done() <-chan Summary { return s.doneCh }

// SentText 已发送累计文本（并发安全；供 JSONL 兜底前缀比对）。
func (s *Session) SentText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sentText
}

func (s *Session) logf(format string, args ...any) {
	if s.opts.Logf != nil {
		s.opts.Logf(format, args...)
	}
}

// Feed 喂一条事件流消息（调用方已按 convId 过滤；未知类型忽略）。
func (s *Session) Feed(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateDone || s.state == stateAborted {
		return
	}
	switch ev.Type {
	case "content":
		if ev.Content == "" {
			return
		}
		if s.startedAt.IsZero() {
			s.startedAt = time.Now()
			s.metaTurn = rawToString(ev.Turn)
			s.metaStep = rawToString(ev.Step)
			s.logf("[stream] 首块到达（turn=%s step=%s）", orQ(s.metaTurn), orQ(s.metaStep))
		}
		s.buffer += ev.Content
		if s.state != stateStreaming {
			s.state = stateStreaming
		}
		s.maybeFlushLocked()
		s.armTimerLocked()
	case "tool_call":
		s.toolCalls++
		if s.buffer != "" {
			s.logf("[stream] tool_call（%s）：丢弃未发缓冲 %d 字符（中间轮）", orQ(ev.Tool), runeLen(s.buffer))
			s.buffer = ""
		}
		if s.opts.OnToolCall != nil {
			s.opts.OnToolCall(ev.Tool, ev.Args)
		}
	case "error":
		c := ev.Content
		if len(c) > 200 {
			c = c[:200]
		}
		s.logf("[stream] 收到 error 事件：%s", c)
		s.finalizeLocked("", "error")
	case "done":
		s.finalizeLocked(ev.Content, "done")
	default:
		// notice/usage/thinking/status/... 忽略
	}
}

// Abort 中止（另一路先完成时调用）：停发剩余缓冲，幂等。
// 注意：已开始发送的片不中断；未消费的发送队列被清空（不再发送）。
func (s *Session) Abort() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateDone || s.state == stateAborted {
		return
	}
	s.state = stateAborted
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	// 清空未消费发送项（abort 后不再发送）
	for {
		select {
		case <-s.flushCh:
			continue
		default:
		}
		break
	}
	sum := Summary{Text: s.sentText, Flushes: s.flushes, ToolCalls: s.toolCalls, Reason: "aborted", Aborted: true}
	s.flushCh <- flushItem{final: true, sum: sum}
}

// ── 内部（调用方须持锁）──────────────────────────────────────────────

func (s *Session) maybeFlushLocked() {
	first := s.flushes == 0
	minC := s.tuning.NextMinChars
	if first {
		minC = s.tuning.FirstMinChars
	}
	if runeLen(s.buffer) < minC {
		return
	}
	if cut := s.findCutLocked(first); cut > 0 {
		s.flushLocked(cut)
	}
}

// findCutLocked 返回可切长度（0=暂不切）：句末优先，否则超 softLimit 强切。
func (s *Session) findCutLocked(first bool) int {
	t := s.tuning
	minC := t.NextMinChars
	if first {
		minC = t.FirstMinChars
	}
	buf := []rune(s.buffer)
	hardCap := t.SoftLimit
	if first {
		hardCap = t.FirstMinChars * 4
		if hardCap < 320 {
			hardCap = 320
		}
	}
	if len(buf) >= hardCap {
		for i := hardCap - 1; i >= minC-1; i-- {
			if isSentenceEnd(buf[i]) {
				return i + 1
			}
		}
		return hardCap
	}
	for i := minC - 1; i < len(buf); i++ {
		if isSentenceEnd(buf[i]) {
			return i + 1
		}
	}
	return 0
}

func (s *Session) armTimerLocked() {
	if s.timer != nil {
		s.timer.Stop()
	}
	first := s.flushes == 0
	wait := time.Duration(s.tuning.NextMaxWaitMs) * time.Millisecond
	since := s.lastFlush
	if first {
		wait = time.Duration(s.tuning.FirstMaxWaitMs) * time.Millisecond
		since = s.startedAt
	}
	remain := time.Until(since.Add(wait))
	if remain < 150*time.Millisecond {
		remain = 150 * time.Millisecond
	}
	s.timer = time.AfterFunc(remain, s.flushByTimer)
}

// flushByTimer 定时触发（独立 goroutine，自锁）：到点必发。
func (s *Session) flushByTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == stateDone || s.state == stateAborted {
		return
	}
	if s.buffer == "" {
		return // 没内容：等下次 content 再 arm
	}
	t := s.tuning
	first := s.flushes == 0
	minC := t.NextMinChars
	if first {
		minC = t.FirstMinChars
	}
	buf := []rune(s.buffer)
	if minC > len(buf) {
		minC = len(buf)
	}
	cut := len(buf)
	if cut > t.SoftLimit {
		cut = t.SoftLimit
	}
	for i := cut - 1; i >= minC-1; i-- {
		if isSentenceEnd(buf[i]) {
			cut = i + 1
			break
		}
	}
	s.flushLocked(cut)
}

// flushLocked 切出前 n 个字符入发送队列；剩余仍达阈值则继续切。
func (s *Session) flushLocked(n int) {
	buf := []rune(s.buffer)
	if n > len(buf) {
		n = len(buf)
	}
	if n <= 0 {
		return
	}
	piece := string(buf[:n])
	s.buffer = string(buf[n:])
	s.flushes++
	s.sentText += piece
	s.lastFlush = time.Now()
	meta := FlushMeta{N: s.flushes, First: s.flushes == 1, Len: runeLen(piece), Turn: s.metaTurn, Step: s.metaStep}
	if meta.First {
		s.logf("[stream] 第 %d 片发出（%d 字符）（首片）", meta.N, meta.Len)
	} else {
		s.logf("[stream] 第 %d 片发出（%d 字符）", meta.N, meta.Len)
	}
	s.flushCh <- flushItem{text: piece, meta: meta}
	// 剩余仍超阈值则继续切
	minNext := s.tuning.NextMinChars
	if len([]rune(s.buffer)) >= minNext {
		if cut := s.findCutLocked(false); cut > 0 {
			s.flushLocked(cut)
		}
	} else {
		s.armTimerLocked()
	}
}

// finalizeLocked 终态处理：全文前缀校验 + 收尾片入队 + 完成标记入队。
// 前缀边界说明（工具轮丢缓冲场景）：若 done 全文不以已发内容为前缀
// （中间轮误发/顺序异常），只补发剩余缓冲、不重复发全文——与 stream.mjs 一致；
// 若为前缀则按全文修正 buffer（full-sent），保证不丢字不重复。
func (s *Session) finalizeLocked(full, reason string) {
	if s.state == stateDone || s.state == stateAborted {
		return
	}
	s.state = stateDone
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if full != "" {
		if strings.HasPrefix(full, s.sentText) {
			missing := full[len(s.sentText):]
			if missing != s.buffer {
				s.logf("[stream] done 全文与缓冲不一致（full-sent=%d buffer=%d），按全文修正",
					runeLen(missing), runeLen(s.buffer))
			}
			s.buffer = missing
		} else {
			s.logf("[stream] done 全文不以已发内容为前缀（sent=%d full=%d），仅补发剩余缓冲",
				runeLen(s.sentText), runeLen(full))
		}
	}
	tail := s.buffer
	s.buffer = ""
	if tail != "" {
		s.flushes++
		s.sentText += tail
		s.flushCh <- flushItem{text: tail, meta: FlushMeta{
			N: s.flushes, First: s.flushes == 1, Len: runeLen(tail), Final: true, Turn: s.metaTurn, Step: s.metaStep,
		}}
		s.logf("[stream] 收尾片（%d 字符，reason=%s）", runeLen(tail), reason)
	}
	fullOut := full
	if fullOut == "" {
		fullOut = s.sentText
	}
	sum := Summary{Text: s.sentText, Flushes: s.flushes, ToolCalls: s.toolCalls, Reason: reason}
	s.flushCh <- flushItem{final: true, sum: sum, text: fullOut}
}

// consumeLoop 串行消费发送队列（零锁）：保证片序；完成标记触发 OnDone 与 Done。
func (s *Session) consumeLoop() {
	for it := range s.flushCh {
		if it.final {
			if s.opts.OnDone != nil && !it.sum.Aborted {
				s.opts.OnDone(it.text, it.sum)
			}
			s.doneCh <- it.sum
			return
		}
		if s.opts.OnFlush != nil {
			s.opts.OnFlush(it.text, it.meta)
		}
	}
}

// ── 辅助 ─────────────────────────────────────────────────────────────

func runeLen(s string) int { return len([]rune(s)) }

func rawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return string(raw)
}

func orQ(s string) string {
	if s == "" {
		return "?"
	}
	return s
}
