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

// detectCircling 检测绕圈：同一操作反复失败 / 反复执行 → 返回「换思路」提示（空=未绕圈）。
// 失败优先（更早干预）；纯重复（即便成功）也提示——可能在原地打转无进展。
func (l *Loop) detectCircling() string {
	count := map[string]int{}
	fails := map[string]int{}
	for _, r := range l.recentCalls {
		count[r.sig]++
		if r.failed {
			fails[r.sig]++
		}
	}
	for sig, n := range fails {
		if n >= circlingFailStop {
			return "[系统提示·打破死循环] 操作 `" + shortSig(sig) + "` 已连续失败 " + strconv.Itoa(n) +
				" 次——别再用同样方式重试！请：① 先 read_file / run_command 检查真实状态、定位失败根因；" +
				"② 换一种工具或思路；③ 仍卡住就用 ask_user 说明卡点求助。绝不要原样再试。"
		}
	}
	for sig, n := range count {
		if n >= circlingRepeatStop {
			return "[系统提示·打破死循环] 你已重复同一操作 `" + shortSig(sig) + "` " + strconv.Itoa(n) +
				" 次，像在原地绕圈、无实质推进。停下来换思路：重新审视目标、换方式推进，或用 ask_user 求助。不要再重复同一步。"
		}
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
