// autopilot.go — 「自主模式」决策器内核注册点（一切皆插件：能力 Go / 策略 JS）
//
// 语义（2026-09-21 新自主模式，取代旧的「任务队列驱动下一阶段」）：
//
//	工作 agent（主 Loop）每次自然结束（无 tool_call + 有正文）→ 宿主（SessionManager
//	的续轮处）调用**插件注册的决策器**：插件由此扮演「人」的角色——审核工作成果、
//	评判质量、必要时自行核查（经子 agent 能力派生一个带全部工具的监督者回合，见
//	subagent.go），然后决定下一步：
//	  · action="continue" + task → 宿主把指令作为新任务唤醒工作 agent 继续执行；
//	  · action="done"           → 整轮结束（工作 agent 的最终汇报即交付）。
//
// 归属划分：
//   - 内核（本文件 + subagent.go）：调用时机、唤醒工作 agent、轮次上限、子 agent 能力、
//     事件标注与下发；
//   - 插件（.pair/plugins/autopilot）：角色提示词、任务书构造、裁决语义、记录落盘、看板 UI。
//
// 未注册实现（未装插件 / 插件停用）→ 宿主按「无监督」处理（自然结束，零行为变化）。
// 环境变量 PAIR_AUTOPILOT=0/off/false/no 可强制禁用（排障/对照）。
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── 请求 / 决策 ───────────────────────────────────────────────

// AutopilotRequest 决策请求（宿主 → 插件决策器）。
type AutopilotRequest struct {
	ConvID        string `json:"convId"`
	WorkspaceRoot string `json:"workspaceRoot"`
	// Round 本次是第几次监督（1 基）。
	Round int `json:"round"`
	// MaxRounds 监督轮数上限（0 = 不限）。
	MaxRounds int `json:"maxRounds"`
	// Objective 用户原始任务（本会话首条真实用户消息；供插件构造任务书）。
	Objective string `json:"objective"`
	// WorkerReport 工作 agent 本轮汇报（自然结束时的最终正文）。
	WorkerReport string `json:"workerReport"`
	// WorkerTurns 工作 agent 已完成的任务轮数（含本轮的 Run 次数）。
	WorkerTurns int `json:"workerTurns"`
	// RecentHistory 工作会话最近消息（时间正序，供插件构造任务书/核查上下文）。
	RecentHistory []Message `json:"recentHistory,omitempty"`
}

// AutopilotDecision 决策结果（插件决策器 → 宿主）。
type AutopilotDecision struct {
	// Action "continue"（继续，Task 必填）| "done"（完成，整轮结束）。
	Action string `json:"action"`
	// Task 给工作 agent 的下一步指令（Action="continue" 时生效）。
	Task string `json:"task,omitempty"`
	// Assessment 评判文本（展示/日志；不影响宿主行为）。
	Assessment string `json:"assessment,omitempty"`
	// Handled 插件是否实际处理（false = 未生效 → 宿主按无监督处理）。
	Handled bool `json:"handled,omitempty"`
	// Error 插件侧错误（诊断）。
	Error string `json:"error,omitempty"`
}

// Continue 是否要求继续唤醒工作 agent。
func (d AutopilotDecision) Continue() bool {
	return strings.EqualFold(strings.TrimSpace(d.Action), "continue") && strings.TrimSpace(d.Task) != ""
}

// ── 注册表（单槽位；后注册覆盖，卸载自动还原）──

var (
	jsAutopilotMu  sync.RWMutex
	jsAutopilotVal *jsAutopilotImpl
)

// RegisterJSAutopilot 注册 JS 自主模式决策器，返回还原函数。
func RegisterJSAutopilot(impl *jsAutopilotImpl) (restore func()) {
	jsAutopilotMu.Lock()
	prev := jsAutopilotVal
	jsAutopilotVal = impl
	jsAutopilotMu.Unlock()
	return func() {
		jsAutopilotMu.Lock()
		if jsAutopilotVal == impl {
			jsAutopilotVal = prev
		}
		jsAutopilotMu.Unlock()
	}
}

// UnregisterJSAutopilot 注销指定实现（插件卸载时调用）。
func UnregisterJSAutopilot(impl *jsAutopilotImpl) {
	jsAutopilotMu.Lock()
	if jsAutopilotVal == impl {
		jsAutopilotVal = nil
	}
	jsAutopilotMu.Unlock()
}

// CurrentJSAutopilot 当前生效的决策器（nil = 未注册/被禁用 → 宿主按无监督处理）。
func CurrentJSAutopilot() *jsAutopilotImpl {
	if !autopilotEnabled() {
		return nil
	}
	jsAutopilotMu.RLock()
	defer jsAutopilotMu.RUnlock()
	return jsAutopilotVal
}

// AutopilotAvailable 是否已装配自主模式（宿主据此决定是否需要监督续轮）。
func AutopilotAvailable() bool {
	if !autopilotEnabled() {
		return false
	}
	jsAutopilotMu.RLock()
	defer jsAutopilotMu.RUnlock()
	return jsAutopilotVal != nil
}

// autopilotEnabled 是否允许自主模式（默认允许；环境变量可强制关闭）。
func autopilotEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("PAIR_AUTOPILOT"))) {
	case "0", "off", "false", "no", "disable", "disabled":
		return false
	}
	return true
}

// ── 宿主调用 ──────────────────────────────────────────────────

// DefaultSuperviseDecideTimeout 决策器单次调用超时（防插件侧卡死拖住整轮会话）。
const DefaultSuperviseDecideTimeout = 20 * time.Minute

// SuperviseDecideTimeout 生效超时；环境变量 PAIR_AUTOPILOT_TIMEOUT（秒）可覆盖（排障用）。
func SuperviseDecideTimeout() time.Duration {
	if s := strings.TrimSpace(os.Getenv("PAIR_AUTOPILOT_TIMEOUT")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return DefaultSuperviseDecideTimeout
}

// RunAutopilotDecision 调用当前决策器（未注册 → Handled=false，宿主按无监督处理）。
// ctx 取消、决策器 panic、调用超时都被隔离为「未处理」，不拖垮会话。
//
// ★ 超时兜底（2026-09-21）：决策器在插件 VM 锁下执行，一旦插件内部卡死（如再取同一
// 把非重入锁、等待永不结算的 Promise），会话会被永久拖住（前端无响应、无法收尾）。
// 故此处按超时收尾：记日志告警并返回「未处理」——会话继续走收尾路径。
// 注意：超时后插件 VM 锁可能仍被那个卡死的 goroutine 占用（该插件后续调用会阻塞），
// 需重载插件/重启宿主恢复——但用户不会再被一个死会话无限挂住。
func RunAutopilotDecision(ctx context.Context, req AutopilotRequest) AutopilotDecision {
	impl := CurrentJSAutopilot()
	if impl == nil {
		return AutopilotDecision{Handled: false}
	}
	if err := ctx.Err(); err != nil {
		return AutopilotDecision{Handled: false, Error: "已取消：" + err.Error()}
	}
	// ★ 诊断：决策器调用前后各一行——「有开始无返回」即决策器内部阻塞（插件侧问题）。
	log.Printf("[autopilot] 决策器调用开始 conv=%s round=%d objectiveLen=%d reportLen=%d history=%d",
		req.ConvID, req.Round, len(req.Objective), len(req.WorkerReport), len(req.RecentHistory))

	type decideResult struct {
		d   AutopilotDecision
		err error
	}
	done := make(chan decideResult, 1) // 带缓冲：超时后 goroutine 仍可投递退出，不泄漏
	go func() {
		d, err := impl.decide(ctx, req)
		done <- decideResult{d: d, err: err}
	}()
	timer := time.NewTimer(SuperviseDecideTimeout())
	defer timer.Stop()

	var (
		d   AutopilotDecision
		err error
	)
	select {
	case r := <-done:
		d, err = r.d, r.err
	case <-ctx.Done():
		log.Printf("[autopilot] 决策器调用被取消 conv=%s round=%d: %v", req.ConvID, req.Round, ctx.Err())
		return AutopilotDecision{Handled: false, Error: "已取消：" + ctx.Err().Error()}
	case <-timer.C:
		log.Printf("[autopilot] ★ 决策器调用超时（%s）conv=%s round=%d：按未处理收尾（插件可能自锁/卡死，其 VM 锁或仍被占用，需重载插件恢复）",
			SuperviseDecideTimeout(), req.ConvID, req.Round)
		return AutopilotDecision{Handled: false, Error: "决策器调用超时"}
	}
	if err != nil {
		log.Printf("[autopilot] 决策器执行失败 conv=%s round=%d: %v", req.ConvID, req.Round, err)
		return AutopilotDecision{Handled: false, Error: err.Error()}
	}
	d.Handled = true
	log.Printf("[autopilot] 决策器返回 conv=%s round=%d action=%q taskLen=%d", req.ConvID, req.Round, d.Action, len(d.Task))
	return d
}

// AutopilotDecisionNotice 决策结果的展示文案（宿主发给前端的通知）。
func AutopilotDecisionNotice(d AutopilotDecision, round int) string {
	switch {
	case !d.Handled:
		return ""
	case d.Continue():
		return fmt.Sprintf("自主模式：监督者审核后要求继续（第 %d 次监督）", round)
	case strings.EqualFold(strings.TrimSpace(d.Action), "done"):
		return fmt.Sprintf("自主模式：监督者判定任务完成（第 %d 次监督）", round)
	default:
		return fmt.Sprintf("自主模式：监督者未给出明确裁决（第 %d 次监督），按完成收尾", round)
	}
}

// ── 决策请求的上下文取材（宿主组装；插件据此构造任务书）──

const (
	// DefaultMaxSuperviseRounds 单条消息默认最多监督轮数（防失控；0 或负 = 用本默认）。
	DefaultMaxSuperviseRounds = 20
	// AutopilotHistoryTail 决策请求携带的最近消息条数（尾部；控制请求体积）。
	AutopilotHistoryTail = 30
	// autopilotHistoryMsgChars 单条历史消息注入决策请求的最大字符数。
	autopilotHistoryMsgChars = 1200
)

// MaxSuperviseRoundsOrDefault 本 Loop 生效的监督轮数上限（0/负 = 默认）。
func (l *Loop) MaxSuperviseRoundsOrDefault() int {
	if l == nil || l.MaxSuperviseRounds <= 0 {
		return DefaultMaxSuperviseRounds
	}
	return l.MaxSuperviseRounds
}

// FirstUserTask 取会话首个真实用户任务（跳过系统注入消息，见 injected_msg.go）。
func FirstUserTask(history []Message) string {
	for _, m := range history {
		if m.Role != RoleUser {
			continue
		}
		if hasPrefixInjected(m.Content) {
			continue
		}
		if s := strings.TrimSpace(m.Content); s != "" {
			return s
		}
	}
	return ""
}

// LastAssistantContent 取消息序列中最后一条 assistant 正文（工作 agent 本轮汇报）。
func LastAssistantContent(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != RoleAssistant {
			continue
		}
		if s := strings.TrimSpace(msgs[i].Content); s != "" {
			return s
		}
	}
	return ""
}

// RecentHistoryTail 取历史尾部 n 条（保留角色与工具名，单条截断；供插件构造核查上下文）。
func RecentHistoryTail(history []Message, n int) []Message {
	if n <= 0 || len(history) == 0 {
		return nil
	}
	start := len(history) - n
	if start < 0 {
		start = 0
	}
	out := make([]Message, 0, len(history)-start)
	for _, m := range history[start:] {
		if m.Role == RoleSystem {
			continue
		}
		cp := m
		cp.Content = truncStr(m.Content, autopilotHistoryMsgChars)
		cp.Images = nil // 决策请求不带图片（体积）
		out = append(out, cp)
	}
	return out
}
