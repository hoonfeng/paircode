// ═══════════════════════════════════════════════════════════
// session_wake.go — 会话唤醒投递注入（原 subagent_spawn.go 的最小替代）
//
// 背景（2026-09 架构调整）：子 Agent（成员会话派生 / fork 派生 / 成员事件桥）
// 实现已整体删除。仍然保留的唯一能力是**向已存在的会话投递一条输入把它唤醒
// 继续跑一轮**——消费者是 agent-teams 插件（Web 面板「批准并运行」后唤醒
// 队长会话，插件经 ctx.agents.followup 调用）。
//
// 与主链路完全同构：复用 launchConvRun（LoopOpts 构建 / 工具集白名单 /
// 审核解析 / SessionManager.Start），因此唤醒的会话与用户手动发消息跑一轮
// 没有差别——差别只是由插件发起。
// ═══════════════════════════════════════════════════════════

package main

import (
	"log"

	"github.com/hoonfeng/paircode/internal/agent"
)

// installSessionWake 注入会话唤醒能力（startWebUI 调用一次）。
func (s *webServer) installSessionWake() {
	agent.SetGlobalSessionManager(agentMgr)
	agent.SetSessionWakeHook(func(convID, wsRoot, text string) error {
		if wsRoot == "" {
			// 会话历史所在工作区（无则回退全局主根）——保证历史加载与落盘同库。
			wsRoot = agentMgr.GetSessionWorkspaceRoot(convID)
			if wsRoot == "" {
				if _, root := agentMgr.FindConversation(convID); root != "" {
					wsRoot = root
				}
			}
		}
		// 与用户发消息同一条链路：先落盘用户消息（与 /api/chat/send 一致，
		// 刷新页面即可见），再后台启动一轮。
		if err := agentMgr.AppendPersistedUserMessageTo(wsRoot, convID, text); err != nil {
			log.Printf("[session-wake] 用户消息落盘失败 conv=%s: %v", convID, err)
		}
		s.launchConvRun(convID, wsRoot, text, false)
		return nil
	})
	log.Printf("[session-wake] 会话唤醒投递器已注入（JS 插件 ctx.agents.followup 可用）")
}
