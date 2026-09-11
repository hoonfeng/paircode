// ═══════════════════════════════════════════════════════════
// session_wake.go — 会话唤醒投递（原「子 Agent 派发面」的最小替代）
//
// 背景（2026-09 架构调整）：子 Agent 实现（成员会话派生 / fork 派生 / 成员
// 事件桥）已整体删除。仍然需要的能力只有一条：**向已存在的会话投递一条输入、
// 把它唤醒继续跑一轮**——消费者是 agent-teams 插件（Web 面板「批准并运行」
// 后唤醒队长会话，插件经 ctx.agents.followup 调用，见
// .pair/plugins/agent-teams/index.js）。
//
// 本文件只做「投递 + 忙时排队」，不含任何会话派生语义：
//   · 真正启动一轮的能力由 web 层注入（SetSessionWakeHook，复用
//     launchConvRun 的 LoopOpts 构建链）——agent 包不反向依赖 web 层。
//   · 会话正在跑 → 消息入 FIFO 队列，轮次结束（EventDone/EventError）
//     自动续投下一条；宿主重启导致队列丢失时可重发（调用方语义）。
// ═══════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// SessionWakeHook 启动一轮会话的能力（web 层注入）。
// 约定：异步执行——内部自行落盘用户消息并调用 SessionManager.Start；
// 返回 nil 表示已受理（无论立即执行还是由 web 层自行处理忙态）。
type SessionWakeHook func(convID, wsRoot, text string) error

var (
	sessionWakeMu     sync.Mutex
	sessionWakeHook   SessionWakeHook
	sessionWakeQueue  = map[string][]string{} // convID → 待投递消息（FIFO）
	sessionWakeBridge bool
)

// SetSessionWakeHook 注入会话唤醒能力（web 层启动时调用一次；重复注入覆盖）。
func SetSessionWakeHook(h SessionWakeHook) {
	sessionWakeMu.Lock()
	sessionWakeHook = h
	sessionWakeMu.Unlock()
}

// SessionWakeReady 报告唤醒投递能力是否可用（插件可据此给出明确错误）。
func SessionWakeReady() bool {
	sessionWakeMu.Lock()
	defer sessionWakeMu.Unlock()
	return sessionWakeHook != nil
}

// WakeSession 向已存在的会话投递一条输入：空闲 → 立即起一轮；忙 → 排队。
// 返回 queued=true 表示已入队（当前轮结束后续投）。
func WakeSession(convID, text string) (queued bool, err error) {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return false, fmt.Errorf("唤醒失败：convId 不能为空")
	}
	if strings.TrimSpace(text) == "" {
		return false, fmt.Errorf("唤醒失败：消息内容不能为空")
	}

	sessionWakeMu.Lock()
	hook := sessionWakeHook
	if hook == nil {
		sessionWakeMu.Unlock()
		return false, fmt.Errorf("唤醒能力未就绪：会话唤醒器未注入（web 层未调用 SetSessionWakeHook）")
	}
	// 会话正在跑 → 排队（轮次结束由事件桥续投）
	if mgr := GlobalSessionManager(); mgr != nil && mgr.IsRunning(convID) {
		sessionWakeQueue[convID] = append(sessionWakeQueue[convID], text)
		sessionWakeMu.Unlock()
		ensureSessionWakeBridge()
		return true, nil
	}
	sessionWakeMu.Unlock()

	ensureSessionWakeBridge()
	if err := hook(convID, sessionWorkspaceRootFor(convID), text); err != nil {
		return false, err
	}
	return false, nil
}

// sessionWorkspaceRootFor 解析会话绑定的工作区根（无则空串 = web 层默认根）。
func sessionWorkspaceRootFor(convID string) string {
	mgr := GlobalSessionManager()
	if mgr == nil {
		return ""
	}
	if root := mgr.GetSessionWorkspaceRoot(convID); root != "" {
		return root
	}
	if _, root := mgr.FindConversation(convID); root != "" {
		return root
	}
	return ""
}

// ensureSessionWakeBridge 惰性启动事件桥（首次排队时启动，只启一次）：
// 监听会话轮次结束 → 续投该会话排队中的下一条消息。
func ensureSessionWakeBridge() {
	sessionWakeMu.Lock()
	if sessionWakeBridge {
		sessionWakeMu.Unlock()
		return
	}
	sessionWakeBridge = true
	sessionWakeMu.Unlock()

	mgr := GlobalSessionManager()
	if mgr == nil {
		sessionWakeMu.Lock()
		sessionWakeBridge = false
		sessionWakeMu.Unlock()
		log.Printf("[session-wake] 事件桥未启动：SessionManager 未注入")
		return
	}
	ch := mgr.SubscribeAll()
	go func() {
		for ge := range ch {
			if ge.Event.Type != EventDone && ge.Event.Type != EventError {
				continue
			}
			sessionWakeMu.Lock()
			q := sessionWakeQueue[ge.ConvID]
			if len(q) == 0 {
				sessionWakeMu.Unlock()
				continue
			}
			next := q[0]
			if len(q) == 1 {
				delete(sessionWakeQueue, ge.ConvID)
			} else {
				sessionWakeQueue[ge.ConvID] = q[1:]
			}
			hook := sessionWakeHook
			sessionWakeMu.Unlock()

			if hook == nil {
				continue
			}
			if err := hook(ge.ConvID, sessionWorkspaceRootFor(ge.ConvID), next); err != nil {
				log.Printf("[session-wake] 排队消息续投失败 conv=%s: %v", ge.ConvID, err)
			}
		}
	}()
	log.Printf("[session-wake] 事件桥已启动（监听会话轮次结束 → 续投排队消息）")
}
