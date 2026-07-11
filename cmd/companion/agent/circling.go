package agent

// 绕圈/死循环检测 —— companion 自加的硬化（参考的 TAOR 主循环里并无重复操作检测，仅「连续 3 轮全错」止损）。
// 追踪最近的工具调用（签名 + 成败），发现「同一操作反复失败」或「同一操作反复执行」即注入一条
// 「换思路」用户消息，让 Agent 下一轮看到、打破死循环（针对"遇 bug 一直绕圈"）。

import (
	"strconv"
	"strings"
)

const (
	circlingWindow     = 12 // 追踪最近多少次工具调用
	circlingRepeatStop = 3  // 同一操作（同工具+同参数）重复达此次数 → 提示
	circlingFailStop   = 2  // 同一操作失败达此次数 → 提示（失败更早干预）
)

// toolSig 一次工具调用的签名 + 是否失败。
type toolSig struct {
	sig    string
	failed bool
}

// trackCall 记录一次工具调用（绕圈检测用）。name+规范化参数 作签名，同操作→同签名。
func (l *Loop) trackCall(name, args string, failed bool) {
	l.recentCalls = append(l.recentCalls, toolSig{sig: name + "|" + normArgs(args), failed: failed})
	if len(l.recentCalls) > circlingWindow {
		l.recentCalls = l.recentCalls[len(l.recentCalls)-circlingWindow:]
	}
}

// detectCircling 检测绕圈：从尾部倒扫，仅当「连续多次相同操作（间无其他操作）」才提示。
// 不检测间隔重复——「build→读→改→build→读→改→build」是正常的编译修复循环，中间有不同操作，不算绕圈。
// 「build→build→build」或「edit_file→edit_file(没先读)」才是真绕圈。
// 返回非空=需注入的系统提示（空=未绕圈）。
func (l *Loop) detectCircling() string {
	n := len(l.recentCalls)
	if n < 2 {
		return ""
	}

	// ---- 1. 从尾部倒扫连续相同签名（纯重复） ----
	last := l.recentCalls[n-1].sig
	sameCount := 1
	for i := n - 2; i >= 0 && l.recentCalls[i].sig == last; i-- {
		sameCount++
	}
	if sameCount >= circlingRepeatStop {
		return "[系统提示·打破死循环] 你已连续 " + strconv.Itoa(sameCount) +
			" 次执行同一操作 `" + shortSig(last) + "`，中间没有任何其他操作——像在原地绕圈。请停下来换思路：先 read_file 看看当前状态，或换工具、换方式推进。别继续重复同一步。"
	}

	// ---- 2. 从尾部倒扫连续相同签名+失败 ----
	failCount := 0
	for i := n - 1; i >= 0 && l.recentCalls[i].sig == last; i-- {
		if l.recentCalls[i].failed {
			failCount++
		} else {
			break // 中间有成功调用→打断连续失败
		}
	}
	if failCount >= circlingFailStop {
		return "[系统提示·打破死循环] 操作 `" + shortSig(last) + "` 已连续失败 " + strconv.Itoa(failCount) +
			" 次且中间没有其他操作——别原样重试！请：① 先 read_file / run_command 检查真实状态、定位失败根因；" +
			"② 换一种工具或思路；③ 仍卡住就用 ask_user 说明卡点求助。"
	}

	return ""
}

// normArgs 规范化参数（去全部空白），使同参数→同签名。
func normArgs(args string) string {
	return strings.Join(strings.Fields(args), "")
}

// shortSig 签名转可读短串（工具名 参数预览）。
func shortSig(sig string) string {
	return truncRunesAgent(strings.ReplaceAll(sig, "|", " "), 60)
}
