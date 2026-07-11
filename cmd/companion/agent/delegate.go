// delegate.go 多 agent 委托工具：delegate_task / delegate_single_turn / finish_task / transfer_to_agent。
//
// 核心设计（与 ADK 关键差异）：**不做上下文隔离**。
//   - 子 Loop 用父 []Message 作 history 前缀（剥离末尾未配对的 assistant tool_call = 委托调用本身），
//     使子 agent 首次 LLM 调用的 messages 前缀与父上一次调用逐字节一致 → prompt cache 命中。
//   - 子 agent 专属 System 不作 system 消息插入（会破坏前缀），而是作为追加 instruction 拼到 task 前。
//   - 子 Loop 共享父 State 引用（跨 agent 传递中间结果，不塞进 messages）。
//
// 停止信号：子 agent 调 finish_task(result) → Loop.Run 检测到后置 finishResult 并退出；
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
// 子 agent 的 finish_task 在 runSubAgent 创建子 Loop 时单独注册（避免污染父表）。
// tree 为编排树；parent.AgentTree 应与 tree 一致。
func RegisterDelegateTools(parent *Loop, tree *AgentTree) {
	reg := parent.Registry

	reg.Register(&Tool{
		Name: "delegate_task",
		Description: "多轮委托：把任务交给子 agent 运行至完成（子 agent 调 finish_task 退出）。" +
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

	// 子 Registry：白名单裁剪或父副本。用副本/裁剪表注册 finish_task，避免污染父 Registry。
	var childReg *Registry
	if len(sa.Tools) > 0 {
		childReg = parent.Registry.Subset(sa.Tools)
	} else {
		childReg = parent.Registry.Copy()
	}
	// 子 agent 注册 finish_task（退出信号）
	childReg.Register(&Tool{
		Name:        "finish_task",
		Description: "任务完成信号：调用后子 agent 退出循环，result 作为委托结果返回父 agent。",
		Parameters:  objSchema(props{"result": strProp("任务结果摘要")}, "result"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return argStr(args, "result"), nil
		},
	})

	// ── 单轮委托：不走 Loop.Run，直接单次 LLM 调用 ──
	if singleTurn {
		return runSingleLLMCall(ctx, parent, sa, childReg, name, task)
	}

	// ── 多轮委托：走完整 Loop.Run（子 Agent 可调工具、finish_task → 退出） ──
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

	childMsgs, err := child.Run(ctx, childTask, history)
	if err != nil && !errors.Is(err, ErrMaxIterations) {
		return "", err
	}
	// 子最终结果：finish_task 的 result 优先，否则取最后 assistant 正文
	if child.finishResult != nil {
		return *child.finishResult, nil
	}
	if r := lastAssistantContent(childMsgs); r != "" {
		return r, nil
	}
	return "(子 agent 未产出结果)", nil
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

	// 结果通过函数返回值回传，EventFinal/EventDone 被 SubAgentSink 丢弃 → 不泄漏
	result := assistant.Content
	onEvent(Event{Type: EventFinal, Content: result})
	onEvent(Event{Type: EventDone, Content: result, DoneReason: "task_complete"})
	return result, nil
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
