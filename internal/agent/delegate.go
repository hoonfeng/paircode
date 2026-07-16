// delegate.go 多 agent 委托工具：delegate_task / delegate_single_turn / transfer_to_agent。
//
// 核心设计（与 ADK 关键差异）：**不做上下文隔离**。
//   - 子 Loop 用父 []Message 作 history 前缀（剥离末尾未配对的 assistant tool_call = 委托调用本身），
//     使子 agent 首次 LLM 调用的 messages 前缀与父上一次调用逐字节一致 → prompt cache 命中。
//   - 子 agent 专属 System 不作 system 消息插入（会破坏前缀），而是作为追加 instruction 拼到 task 前。
//   - 子 Loop 共享父 State 引用（跨 agent 传递中间结果，不塞进 messages）。
//
// 停止信号：子 agent 输出内容后 Loop.Run 自然退出（content-only 终止）；
// delegate handler 从 child.finishResult 取子最终结果作为工具结果回传父。

package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// RegisterDelegateTools 向父 Loop 的 Registry 注册委托工具：
// delegate_task（多轮委托）/ delegate_single_turn（单轮委托）/ transfer_to_agent（控制权转移）。
// 子 agent 出口由 Loop.Run 自然终止提供。
// tree 为编排树；parent.AgentTree 应与 tree 一致。
func RegisterDelegateTools(parent *Loop, tree *AgentTree) {
	reg := parent.Registry

	reg.Register(&Tool{
		Name: "delegate_task",
		Description: "多轮委托：把任务交给子 agent 运行至完成。" +
			"子 agent 看到完整父历史（缓存命中），其产出作为本工具结果回传。协调器用它分派子任务给专家 agent（planner/coder/reviewer 等）。",
		Parameters: objSchema(props{
			"agent_name": strProp("目标子 agent 名（见系统提示的可用 agent 列表）"),
			"task":       strProp("委托给子 agent 的任务描述"),
		}, "agent_name", "task"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runSubAgent(ctx, parent, tree, argStr(args, "agent_name"), argStr(args, "task"), false)
		},
	})

	reg.Register(&Tool{
		Name:        "delegate_single_turn",
		Description: "单轮委托：让子 agent 只做 1 次 LLM 调用（不进多轮循环），结果直接返回。适合无需工具的简单子任务。",
		Parameters: objSchema(props{
			"agent_name": strProp("目标子 agent 名"),
			"input":      strProp("子 agent 的输入"),
		}, "agent_name", "input"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return runSubAgent(ctx, parent, tree, argStr(args, "agent_name"), argStr(args, "input"), true)
		},
	})

	reg.Register(&Tool{
		Name:        "transfer_to_agent",
		Description: "控制权转移：当前 agent 退出，目标 agent 接管同一对话历史。用于「该让 X agent 处理」的场景。",
		Parameters:  objSchema(props{"agent_name": strProp("目标 agent 名")}, "agent_name"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "agent_name")
			if tree.Find(name) == nil {
				return "", fmt.Errorf("未找到 agent %q（可用：%s）", name, strings.Join(tree.SubNames(), ", "))
			}
			parent.transferTarget = name // Loop.Run 检测到非空后退出当前循环
			return "控制权已转移给 " + name, nil
		},
	})
}

// runSubAgent 创建并运行子 Loop。singleTurn=true 时只跑 1 轮（delegate_single_turn）。
//
// 缓存前缀稳定：history = parent.currentMsgs 剥离末尾未配对 assistant tool_call（=委托调用本身），
// 使子 Loop 首次 LLM 调用前缀 = 父上一次调用前缀。
//
// singleTurn=true 特化路径：不走完整 Loop.Run（该路径已移除最大迭代硬检查），
// 直接做单次 LLM 调用（Provider.Chat），取 assistant.Content 返回。
// 这样避免 nudge/绕圈检测等长循环逻辑影响单轮快速应答。
func runSubAgent(ctx context.Context, parent *Loop, tree *AgentTree, name, task string, singleTurn bool) (string, error) {
	sa := tree.Find(name)
	if sa == nil {
		return "", fmt.Errorf("未找到 agent %q（可用：%s）", name, strings.Join(tree.SubNames(), ", "))
	}

	// 子 Registry：白名单裁剪或父副本（自动排除外层专用工具：plan/task）。
	childReg := rebuildSubRegistry(parent.Registry, sa)

	// ── 单轮委托：不走 Loop.Run，直接单次 LLM 调用 ──
	if singleTurn {
		return runSingleLLMCall(ctx, parent, sa, childReg, name, task)
	}

	// ── 多轮委托：走完整 Loop.Run（子 Agent 可调工具、自然终止退出） ──
	maxIter := sa.MaxIter
	if maxIter <= 0 {
		maxIter = parent.MaxIterations
	}
	child := &Loop{
		Provider:      parent.Provider,
		Registry:      childReg,
		System:        "",
		MaxIterations: maxIter,
		OnEvent:       SubAgentSink(parent.OnEvent, name),
		State:         parent.State,
		AgentTree:     tree,
		// ★ 继承父的持久化回调，使子 agent 的每轮迭代独立落盘
		OnBatchPersist:   parent.OnBatchPersist,
		OnMessagePersist: parent.OnMessagePersist,
		// ★ 继承父的审核设置，使子 agent 的写操作也经过 AI 审核
		AutoReview:     parent.AutoReview,
		ReviewProvider: parent.ReviewProvider,
	}

	// 子 task：子 system 作追加 instruction（不替换父 system，保前缀）
	childTask := task
	if sa.System != "" {
		childTask = "# 子 agent 指令（" + sa.Name + "）\n" + sa.System + "\n\n---\n\n# 任务\n" + task
	}

	// 子 history = 父当前历史，剥离末尾未配对 assistant tool_call（=delegate_task 本身）
	history := parent.currentMsgs
	if len(history) > 0 && history[len(history)-1].Role == RoleAssistant && len(history[len(history)-1].ToolCalls) > 0 {
		history = history[:len(history)-1]
	}

	// ★ 记录执行日志：提取外层分析内容 + 委派任务
	if analysis := lastAssistantContent(history); analysis != "" {
		parent.LogAnalysis(analysis)
	}
	parent.LogDelegation(name, childTask)

	// ★ 委托前刷盘：将外层 agent 当前消息（含分析+delegate_task）落盘为独立 assistant 消息
	if parent.OnBatchPersist != nil {
		parent.OnBatchPersist(parent.currentMsgs)
	}
	// ★ 委派任务存为用户消息：使前端展示为独立气泡，清晰区分「外层分析」→「委派任务」→「内层执行」
	delegationMsg := Message{Role: RoleUser, Content: "【任务委派 → " + name + "】\n" + task}
	if parent.OnMessagePersist != nil {
		_ = parent.OnMessagePersist(delegationMsg)
	}
	// 将委派消息追加到子 history，使子 agent 首次调用时它作为「用户最新消息」
	history = append(history, delegationMsg)

	childMsgs, err := child.Run(ctx, childTask, history)
	if err != nil && !errors.Is(err, ErrMaxIterations) {
		return "", err
	}
	// 子最终结果：自然终止的内容优先，否则取最后 assistant 正文
	result := ""
	if child.finishResult != nil {
		result = *child.finishResult
	} else if r := lastAssistantContent(childMsgs); r != "" {
		result = r
	} else {
		result = "(子 agent 未产出结果)"
	}
	// ★ 记录执行日志：子 agent 的执行结果
	parent.LogResult(name, result)

	// ★ 包装结果，防止父 agent 将子 agent 的执行报告误判为「整体任务已完成」。
	// 子 agent 的输出原样回灌会让父 LLM 产生已完成全部工作的错觉，导致提前收尾。
	return fmt.Sprintf("【子 agent(%s)执行结果】\n%s\n\n---\n请根据以上结果决定下一步：\n1. **立即调用 update_plan 将当前项标记为 done**（无论是否还有剩余项，必须先更新计划状态）\n2. 如果还有剩余计划项，再调用 delegate_task 执行下一项\n3. 如果全部计划项都已是 done 状态，再调用 generate_commit_message 完成收尾", name, result), nil
}

// runSingleLLMCall 单轮委托：直接做 1 次 LLM 调用（不进 Loop.Run），取 assistant.Content 返回。
// 用于 delegate_single_turn 工具。
// 事件经过 SubAgentSink 过滤：EventFinal/EventDone 丢弃（结果通过函数返回值回传），thinking/content/usage 标记 AgentName 后转发。
func runSingleLLMCall(ctx context.Context, parent *Loop, sa *SubAgent, childReg *Registry, name, task string) (string, error) {
	// 子 task（追加子 system）
	childTask := task
	if sa.System != "" {
		childTask = "# 子 agent 指令（" + sa.Name + "）\n" + sa.System + "\n\n---\n\n# 任务\n" + task
	}

	// 构造 messages：父 history 剥离末尾未配对 assistant tool_call → 追加 user 消息（childTask）
	history := parent.currentMsgs
	if len(history) > 0 && history[len(history)-1].Role == RoleAssistant && len(history[len(history)-1].ToolCalls) > 0 {
		history = history[:len(history)-1]
	}
	msgs := make([]Message, 0, len(history)+2)
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: RoleUser, Content: childTask})

	// ★ 记录执行日志：single turn 的分析和委派
	if analysis := lastAssistantContent(history); analysis != "" {
		parent.LogAnalysis(analysis)
	}
	parent.LogDelegation(name, childTask)

	// ★ 单轮委托也执行刷盘：外层分析+委派落盘为独立 assistant 消息
	if parent.OnBatchPersist != nil {
		parent.OnBatchPersist(parent.currentMsgs)
	}
	// ★ 委派任务存为用户消息
	delegationMsg := Message{Role: RoleUser, Content: "【任务委派 → " + name + "】\n" + task}
	if parent.OnMessagePersist != nil {
		_ = parent.OnMessagePersist(delegationMsg)
	}

	// 事件经过 SubAgentSink 过滤：EventFinal/EventDone 丢弃，其他事件标记 AgentName 后转发
	onEvent := SubAgentSink(parent.OnEvent, name)

	// 单次 LLM 调用
	assistant, err := parent.Provider.Chat(ctx, msgs, childReg.Definitions(), func(c Chunk) {
		if c.Reasoning != "" {
			onEvent(Event{Type: EventThinking, Content: c.Reasoning})
		}
		if c.Content != "" {
			onEvent(Event{Type: EventContent, Content: c.Content})
		}
		if c.Usage != nil && c.Usage.PromptTokens > 0 {
			usage := *c.Usage
			onEvent(Event{Type: EventUsage, Usage: &usage})
		}
	})
	if err != nil {
		return "", err
	}

	// ★ 记录执行日志：single turn 执行结果
	parent.LogResult(name, assistant.Content)

	// 结果通过函数返回值回传，EventFinal/EventDone 被 SubAgentSink 丢弃 → 不泄漏
	result := assistant.Content
	onEvent(Event{Type: EventFinal, Content: result})
	onEvent(Event{Type: EventDone, Content: result, DoneReason: "task_complete"})
	return fmt.Sprintf("【子 agent(%s)执行结果】\n%s\n\n---\n请根据以上结果决定下一步：\n1. **立即调用 update_plan 将当前项标记为 done**（无论是否还有剩余项，必须先更新计划状态）\n2. 如果还有剩余计划项，再调用 delegate_task 执行下一项\n3. 如果全部计划项都已是 done 状态，再调用 generate_commit_message 完成收尾", name, result), nil
}

// lastAssistantContent 取消息列表中最后一条非空 assistant 正文。
func lastAssistantContent(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleAssistant && msgs[i].Content != "" {
			return msgs[i].Content
		}
	}
	return ""
}

// ── 外层 agent 专用工具：不允许透传给子 agent ──
// update_plan 是自主模式编排 agent 用来维护执行计划清单的，内层子 agent
// 不需要看到它。task_* 工具（task_create/update/list/delete/summary）则是
// 给内层子 agent 追踪自身任务进度用的，需要保留在子 registry 中。
var outerOnlyTools = map[string]bool{
	"update_plan": true,
}

// rebuildSubRegistry 创建子 agent 的注册表，自动排除外层专用工具。
// 若子 agent 有显式白名单 (sa.Tools)，则先过滤掉外层专用工具，再 Subset。
// 若无白名单，则从父注册表 Copy 后移除外层专用工具。
func rebuildSubRegistry(parent *Registry, sa *SubAgent) *Registry {
	// 有白名单：先过滤掉外层专用工具
	if len(sa.Tools) > 0 {
		filtered := make([]string, 0, len(sa.Tools))
		for _, t := range sa.Tools {
			if !outerOnlyTools[t] {
				filtered = append(filtered, t)
			}
		}
		return parent.Subset(filtered)
	}
	// 无白名单：继承父的全部工具，但排除外层专用工具
	names := make([]string, 0, len(parent.order))
	for _, n := range parent.order {
		if !outerOnlyTools[n] {
			names = append(names, n)
		}
	}
	return parent.Subset(names)
}
