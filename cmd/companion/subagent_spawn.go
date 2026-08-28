// ═══════════════════════════════════════════════════════════
// subagent_spawn.go — 子 Agent（成员会话）启动器注入
//
// 背景（2026-08-28）：多智能体团队插件（.pair/plugins/agent-teams）需要
// 「队长派生成员会话、成员各自续聊」的能力。会话启动链路（Provider 构造 /
// 工具注册表 / 工具集白名单 / 审核配置 / 历史加载）全在 web 层，故此处把
// 启动能力注入 agent 包（agent.SetSubAgentSpawner），JS 插件经 ctx.agents
// 消费——与 SetSessionBridge 同构的注入模式（agent 包不反向依赖 web 层）。
//
// 成员会话与主会话完全同构：出现在会话列表、历史持久化、事件走 /ws（前端
// 可切进成员会话看它干活），差别只有三处：
//   1. System 追加成员 persona（团队协议 + 角色约束）
//   2. 模型可覆盖（跨服务商时按 models.json 解析 baseURL/apiKey）
//   3. 工具黑名单（队长专属工具对成员不可见）
// ═══════════════════════════════════════════════════════════

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

// installSubAgentSpawner 注入成员会话启动能力（startWebUI 调用一次）。
func (s *webServer) installSubAgentSpawner() {
	agent.SetGlobalSessionManager(agentMgr)
	agent.SetSubAgentSpawner(&agent.SubAgentSpawner{
		Start:         s.startSubAgentTurn,
		Stop:          func(convID string) { agentMgr.Stop(convID) },
		Running:       func(convID string) bool { return agentMgr.IsRunning(convID) },
		LastAssistant: subAgentLastAssistant,
		Models:        subAgentModelCatalog,
		Current:       subAgentCurrentRoute,
	})
	log.Printf("[subagent] 成员会话启动器已注入（JS 插件 ctx.agents 可用）")
}

// startSubAgentTurn 启动成员会话的一轮（异步；同步段只做快速校验与消息落盘）。
func (s *webServer) startSubAgentTurn(spec agent.SubAgentSpec) error {
	convID := strings.TrimSpace(spec.ConvID)
	if convID == "" {
		return fmt.Errorf("成员会话启动失败：convId 为空")
	}
	task := strings.TrimSpace(spec.Task)
	if task == "" {
		return fmt.Errorf("成员会话启动失败：本轮输入为空")
	}
	if !agent.ConfiguredProvider() {
		return fmt.Errorf("未配置 API key：请先在「设置 → AI」中配置服务商与模型")
	}
	if agentMgr.IsRunning(convID) {
		return fmt.Errorf("成员会话 %s 正在运行中（同一会话不可并行两轮）", convID)
	}
	wsRoot := strings.TrimSpace(spec.WsRoot)
	if wsRoot == "" {
		wsRoot = core.Root()
	}
	// 用户消息落盘（与 /api/chat/send 一致：先落盘再启动，刷新页面可见）
	if err := agentMgr.AppendPersistedUserMessageTo(wsRoot, convID, task); err != nil {
		return fmt.Errorf("成员会话消息落盘失败: %v", err)
	}

	go func() {
		opts := s.buildWebLoopOpts(convID, task, false, wsRoot)
		if opts.Provider == nil {
			agentMgr.PushStartError(convID, "未配置 AI 服务商（APIKey/BaseURL 为空）")
			return
		}
		opts.WorkspaceRoot = wsRoot

		// ① persona：追加到系统提示尾部（不替换宿主提示——成员仍需完整工具纪律）
		if persona := strings.TrimSpace(spec.System); persona != "" {
			opts.System = strings.TrimRight(opts.System, "\n") + "\n\n" + persona
		}

		// ② 模型覆盖（成员可用不同模型/服务商）
		if prov := buildSubAgentProvider(spec); prov != nil {
			opts.Provider = prov
		}

		// ③ 工具面：工作区工具集白名单 → 再摘除队长专属工具
		if opts.Registry != nil {
			agent.ApplyWorkspaceToolsetWhitelist(handler.GetPluginHost(), opts.Registry, wsRoot)
			for _, name := range spec.DenyTools {
				if n := strings.TrimSpace(name); n != "" {
					opts.Registry.Unregister(n)
				}
			}
		}

		// ④ 审核配置（与主会话同源：全局设置 + 工作区覆盖）
		opts.ReviewMode = core.Settings.ReviewMode
		opts.ReviewBlacklist = core.Settings.ReviewBlacklist
		opts.ReviewWhitelist = core.Settings.ReviewWhitelist
		if wsRoot != "" {
			if wrMode, wrBlack, wrWhite := agent.LoadWorkspaceReviewConfig(wsRoot); true {
				if wrMode != "" && wrMode != "auto" {
					opts.ReviewMode = wrMode
				}
				if wrBlack != nil {
					opts.ReviewBlacklist = wrBlack
				}
				if wrWhite != nil {
					opts.ReviewWhitelist = wrWhite
				}
			}
		}
		cur := agent.ResolveProviderParams()
		if opts.ReviewMode == "auto" && cur.ReviewModel != "" {
			if pm := strings.TrimSpace(cur.PlanModel); pm != "" && cur.BaseURL != "" && cur.APIKey != "" {
				// ★ t1 S1：实现级插件槽位（插件注册的 Provider 实现对新协议生效）
				rp := cur
				rp.Model = pm
				rp.Temperature = -1
				rp.ThinkingMode = "non-thinking"
				rp.MaxTokens = 0
				rp.Multimodal = false
				opts.ReviewProvider = agent.CreateProvider(rp)
			}
		}
		if spec.MaxIter > 0 {
			opts.MaxIterations = spec.MaxIter
		}

		setupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := agentMgr.Start(setupCtx, convID, task, opts); err != nil {
			log.Printf("[subagent] 启动失败 conv=%s label=%s err=%v", convID, spec.Label, err)
			agentMgr.PushStartError(convID, err.Error())
			return
		}
		log.Printf("[subagent] 轮次已启动 conv=%s label=%s model=%s", convID, spec.Label, opts.Provider.Name())
	}()
	return nil
}

// buildSubAgentProvider 构造成员模型 Provider（无覆盖返回 nil = 沿用会话默认）。
func buildSubAgentProvider(spec agent.SubAgentSpec) agent.Provider {
	model := strings.TrimSpace(spec.Model)
	provider := strings.TrimSpace(spec.Provider)
	if model == "" && provider == "" {
		return nil
	}
	cur := agent.ResolveProviderParams()
	baseURL, apiKey := cur.BaseURL, cur.APIKey
	if provider != "" && provider != cur.Provider {
		// 跨服务商：按 models.json 解析端点与该服务商独立 Key
		if u := core.GetProviderBaseURL(provider); u != "" {
			baseURL = u
		}
		if keys := core.GetProviderAPIKeys(); keys != nil {
			if k := strings.TrimSpace(keys[provider]); k != "" {
				apiKey = k
			}
		}
	}
	if model == "" {
		model = cur.Model
	}
	if baseURL == "" || apiKey == "" || model == "" {
		log.Printf("[subagent] 模型覆盖不完整（provider=%q model=%q baseURL=%v key=%v），沿用会话默认",
			provider, model, baseURL != "", apiKey != "")
		return nil
	}
	// ★ t1 S1：实现级插件槽位——按目标服务商名路由插件实现（未注册回退 OpenAI 兼容）
	cur.Provider = provider
	cur.BaseURL = baseURL
	cur.APIKey = apiKey
	cur.Model = model
	return agent.CreateProvider(cur)
}

// subAgentLastAssistant 取会话最近一条助手正文（队长汇总成员结果用）。
func subAgentLastAssistant(convID, wsRoot string) string {
	pick := func(msgs []agent.Message) string {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == agent.RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
				return msgs[i].Content
			}
		}
		return ""
	}
	if text := pick(agentMgr.GetHistory(convID)); text != "" {
		return text
	}
	// 内存会话已回收 → 读持久化历史（按会话根路由 store）
	root := strings.TrimSpace(wsRoot)
	if root == "" {
		root = core.Root()
	}
	if store := agentMgr.StoreFor(root); store != nil {
		if msgs, err := store.LoadAll(convID); err == nil {
			return pick(msgs)
		}
	}
	return ""
}

// subAgentModelCatalog 模型目录（插件 ctx.llm.models）：服务商 → 模型清单。
func subAgentModelCatalog() []map[string]any {
	if core.ModelList == nil {
		core.LoadModelList()
	}
	cur := agent.ResolveProviderParams()
	out := make([]map[string]any, 0, 16)
	for provider, entry := range core.ModelList {
		for _, model := range entry.Models {
			out = append(out, map[string]any{
				"provider":  provider,
				"model":     model,
				"label":     provider + "/" + model,
				"isDefault": provider == cur.Provider && model == cur.Model,
			})
		}
	}
	return out
}

// subAgentCurrentRoute 当前主模型路由（插件 ctx.llm.current）。
func subAgentCurrentRoute() map[string]any {
	cur := agent.ResolveProviderParams()
	return map[string]any{
		"provider":     cur.Provider,
		"model":        cur.Model,
		"thinkingMode": cur.ThinkingMode,
	}
}
