package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ─── 工具调用修复增强机制（参考 DeepSeek-Reasonix agent/agent.go）──

// ============================================================
// 1. Tool Output Truncation（工具输出截断）
// ============================================================

// maxToolOutputBytes 是单次工具结果的最大字节数。超过此值的输出会被头尾截断。
// ~32KB ≈ 8K tokens，足够容纳一次完整的文件读取或繁忙的 grep 结果，
// 同时防止一个意外的 "read this 5 MB log" 撑爆上下文窗口。
const maxToolOutputBytes = 32 * 1024

// TruncateToolOutput 当 s 超过 maxToolOutputBytes 时进行头尾截断，
// 返回截断后的 body 和一段用户可见的截断通知（空=未截断）。
func TruncateToolOutput(s string) (string, string) {
	if len(s) <= maxToolOutputBytes {
		return s, ""
	}
	keep := maxToolOutputBytes / 2
	head := snapToRuneBoundary(s, 0, keep)
	tail := snapToRuneBoundary(s, len(s)-keep, len(s))
	omitted := len(s) - len(head) - len(tail)
	notice := fmt.Sprintf("工具输出已截断: 省略了 %d 字节", omitted)
	body := head + fmt.Sprintf("\n\n…[截断 %d 字节 — 缩小参数范围重试以查看中间内容]…\n\n", omitted) + tail
	return body, notice
}

func snapToRuneBoundary(s string, lo, hi int) string {
	for lo > 0 && !utf8.RuneStart(s[lo]) {
		lo--
	}
	for hi < len(s) && !utf8.RuneStart(s[hi]) {
		hi++
	}
	return s[lo:hi]
}

// ============================================================
// 2. Storm Breaker（风暴断路器 — 防止同一错误反复重试）
// ============================================================

// stormBreakThreshold 是同一工具同一错误连续失败多少次后触发风暴断路器。
const stormBreakThreshold = 3

// StormSig 记录一轮工具调用的失败签名。
type StormSig struct {
	sig   string // (name, error) 序列化
	count int
}

// BatchStormSignature 生成一批工具调用的风暴签名。
// 仅当本轮所有调用都失败且未被拦截时返回签名。
// 签名键是 (toolName, errorMsg) 而非 (toolName, args)，
// 因为模型经常会修改参数但同样失败。
func BatchStormSignature(calls []ToolCall, errMsgs []string, blocked []bool) (string, bool) {
	if len(calls) == 0 {
		return "", false
	}
	var sb strings.Builder
	for i := range calls {
		if errMsgs[i] == "" || blocked[i] {
			return "", false // 有成功或被拦截的调用，不是纯失败
		}
		sb.WriteString(calls[i].Function.Name)
		sb.WriteByte(0)
		sb.WriteString(errMsgs[i])
		sb.WriteByte(0)
	}
	return sb.String(), true
}

// ApplyStormBreaker 检测风暴条件，超过阈值后重写模型看到的错误信息。
// 返回 (修改后的结果列表, 是否触发了风暴断路器)。
func ApplyStormBreaker(calls []ToolCall, results []string, errMsgs []string, blocked []bool, stormSig *StormSig) ([]string, bool) {
	sig, ok := BatchStormSignature(calls, errMsgs, blocked)
	if !ok {
		// 本轮有成功调用或混合错误，重置计数器
		stormSig.sig, stormSig.count = "", 0
		return results, false
	}
	if sig != stormSig.sig {
		stormSig.sig, stormSig.count = sig, 1
		return results, false
	}
	stormSig.count++
	if stormSig.count < stormBreakThreshold {
		return results, false
	}
	// 触发风暴断路器：重写第一个结果
	subject := fmt.Sprintf("%q", calls[0].Function.Name)
	if len(calls) > 1 {
		subject = fmt.Sprintf("本轮 %d 个工具调用", len(calls))
	}
	newResults := make([]string, len(results))
	copy(newResults, results)
	newResults[0] = results[0] + fmt.Sprintf(
		"\n\n[循环防护] %s 已连续失败 %d 次，错误完全相同。即使修改措辞重试也不会帮助：调用一直以相同方式失败。请换一种方式：检查参数是否正确、改用不同的工具，或在最终答复中说明阻塞原因。",
		subject, stormSig.count)
	return newResults, true
}

// ============================================================
// 3. Repeat Success Guard（重复成功防护 — 防止重复相同写操作）
// ============================================================

// repeatSuccessBreakThreshold 是同一写操作连续成功多少次后阻止再次执行。
const repeatSuccessBreakThreshold = 2

// RepeatSuccessGuard 检测并阻止重复的写操作。
type RepeatSuccessGuard struct {
	counts map[string]int
}

// NewRepeatSuccessGuard 创建新的重复成功防护。
func NewRepeatSuccessGuard() *RepeatSuccessGuard {
	return &RepeatSuccessGuard{counts: make(map[string]int)}
}

// Reset 清空所有计数。
func (g *RepeatSuccessGuard) Reset() {
	g.counts = make(map[string]int)
}

// Check 检查此调用是否触发了重复成功防护。
// 返回 (是否拦截, 拦截消息)。
func (g *RepeatSuccessGuard) Check(toolName, argsJSON string, readOnly bool) (bool, string) {
	if readOnly {
		return false, "" // 只读工具不拦截
	}
	sig := repeatSuccessSignature(toolName, argsJSON)
	if sig == "" {
		return false, ""
	}
	count := g.counts[sig]
	if count < repeatSuccessBreakThreshold {
		return false, ""
	}
	return true, fmt.Sprintf(
		"[循环防护] %q 已成功执行 %d 次，参数完全相同。再次执行不会带来新进展。请改用其他方式：用 edit_file 做文件修改、用 read_file 或 run_command 验证结果，或在最终答复中说明情况。",
		toolName, count)
}

// Record 记录一次成功的工具调用。
func (g *RepeatSuccessGuard) Record(toolName, argsJSON string, readOnly bool) {
	if readOnly {
		return
	}
	sig := repeatSuccessSignature(toolName, argsJSON)
	if sig == "" {
		return
	}
	g.counts[sig]++
}

func repeatSuccessSignature(toolName, argsJSON string) string {
	switch toolName {
	case "write_file", "edit_file", "multi_edit":
		return toolName + "\x00" + canonicalToolArgs(argsJSON)
	default:
		return ""
	}
}

func canonicalToolArgs(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return strings.TrimSpace(raw)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return string(b)
	}
	return compact.String()
}

// ============================================================
// 4. Final Readiness Check（最终答案就绪检查）
// ============================================================

// FinalReadiness 检查模型是否可以给出最终答案。
// 参考 DeepSeek-Reasonix 的 finalReadinessCheck。
type FinalReadiness struct {
	HasTodos    bool     // 是否有待办事项
	Incomplete  []string // 未完成的事项描述
	HasReceipts bool     // 是否有成功的工具调用记录
}

// CheckFinalReadiness 检查最终答案就绪状态。
// todos 是当前待办列表（task 工具的状态）。
func CheckFinalReadiness(todos []string) FinalReadiness {
	if len(todos) == 0 {
		return FinalReadiness{}
	}
	incomplete := make([]string, 0, len(todos))
	for _, todo := range todos {
		t := strings.TrimSpace(todo)
		if t != "" && !strings.HasPrefix(strings.ToLower(t), "done") &&
			!strings.HasPrefix(strings.ToLower(t), "completed") &&
			!strings.HasPrefix(strings.ToLower(t), "✅") {
			incomplete = append(incomplete, t)
		}
	}
	return FinalReadiness{
		HasTodos:    len(todos) > 0,
		Incomplete:  incomplete,
		HasReceipts: len(incomplete) < len(todos),
	}
}

// ReadinessRetryMessage 生成最终答案就绪检查失败的重试提示。
func ReadinessRetryMessage(reason string) string {
	return "主机最终答案就绪检查未通过。在给出最终答案前，请处理以下未完成的待办事项：" + reason + "。执行所需的工具调用，然后当就绪满足时再回答。"
}

// ============================================================
// 5. Evidence Ledger（证据账本 — 追踪工具调用证据）
// ============================================================

// EvidenceLedger 追踪同一用户 turn 内的工具调用证据。
// 供 complete_step 工具验证引用的证据是否确实发生过。
type EvidenceLedger struct {
	receipts []Receipt
}

// Receipt 一次工具调用的证据记录。
type Receipt struct {
	ToolName string `json:"tool_name"`
	Args     string `json:"args"`
	Success  bool   `json:"success"`
	ReadOnly bool   `json:"read_only"`
}

// NewEvidenceLedger 创建新的证据账本。
func NewEvidenceLedger() *EvidenceLedger {
	return &EvidenceLedger{}
}

// Reset 清空所有证据。
func (l *EvidenceLedger) Reset() {
	l.receipts = nil
}

// Record 记录一次工具调用的证据。
func (l *EvidenceLedger) Record(toolName, argsJSON string, success, readOnly bool) {
	l.receipts = append(l.receipts, Receipt{
		ToolName: toolName,
		Args:     canonicalToolArgs(argsJSON),
		Success:  success,
		ReadOnly: readOnly,
	})
}

// HasSuccessfulReceipt 检查是否存在成功的工具调用记录。
func (l *EvidenceLedger) HasSuccessfulReceipt() bool {
	for _, r := range l.receipts {
		if r.Success {
			return true
		}
	}
	return false
}

// LatestWriterIndex 返回最近一次成功写工具调用的索引。
func (l *EvidenceLedger) LatestWriterIndex() (int, bool) {
	for i := len(l.receipts) - 1; i >= 0; i-- {
		if !l.receipts[i].ReadOnly && l.receipts[i].Success {
			return i, true
		}
	}
	return 0, false
}

// Snapshot 返回当前所有证据的快照。
func (l *EvidenceLedger) Snapshot() []Receipt {
	out := make([]Receipt, len(l.receipts))
	copy(out, l.receipts)
	return out
}
