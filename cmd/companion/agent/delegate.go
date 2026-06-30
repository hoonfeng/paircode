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
		Description: "多轮委托：把任务交给子 agent 运行至完成（子 agent 调 finish_task 或输出 [FINAL]）。" +
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

	maxIter := sa.MaxIter
	if singleTurn {
		maxIter = 1
	}
	child := &Loop{
		Provider:      parent.Provider,
		Registry:      childReg,
		System:        "", // 不插入 system，保前缀稳定（父 history 已含父 system）
		MaxIterations: maxIter,
		OnEvent:       parent.OnEvent,
		State:         parent.State, // 共享状态引用
		AgentTree:     tree,
	}

	// 子 task：子 system 作追加 instruction（不替换父 system，保前缀）
	childTask := task
	if sa.System != "" {
		childTask = "# 子 agent 指令（" + sa.Name + "）\n" + sa.System + "\n\n---\n\n# 任务\n" + task
	}

	// 子 history = 父当前历史，剥离末尾未配对 assistant tool_call（=delegate_task 本身），
	// 保前缀 = 父上一次 LLM 调用前缀 → 缓存命中。
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

// lastAssistantContent 取消息列表中最后一条非空 assistant 正文（剥离 [FINAL] 标记）。
func lastAssistantContent(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleAssistant && msgs[i].Content != "" {
			return stripFinal(msgs[i].Content)
		}
	}
	return ""
}
