// 自主模式：外层设计者 Loop（update_plan + delegate_task）→ 内层执行 Loop（全部工具）。
// 外层 LLM 是真正的设计者，通过工具调用控制计划、分派任务、调整策略。
// 所有逻辑在 agent 包内完成，bridge 只需一句调用。

package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// RunAutonomous 运行自主模式。
//
// planProv: 规划 LLM Provider（外层设计者 Loop 使用）
// innerLoop: 预配置的执行 Loop（已含全部工具 + OnEvent）
// task: 用户目标
//
// 架构：
//
//	外层 Loop（设计者 Agent）
//	  工具: update_plan, delegate_task, generate_commit_message
//	  职责: 分析 → 规划 → 逐项委托 → 评估 → 调整 → 直至完成
//	  ↓ delegate_task
//	内层 Loop（执行 Agent，复用调用方传入的 loop）
//	  工具: 全部执行工具（read_file, write_file, run_command, update_tasks...）
//	  职责: 执行具体子任务，返回结果给外层
func RunAutonomous(ctx context.Context, planProv Provider, innerLoop *Loop, task string) (string, error) {
	// 1. 构建外层注册表
	outerReg := NewRegistry()
	RegisterPlanOnlyTools(outerReg) // update_plan

	// 注册 delegate_task：handler 运行内层 Loop
	outerReg.Register(&Tool{
		Name: "delegate_task",
		Description: "把任务委托给执行 agent。执行 agent 拥有完整工具集（读写文件、运行命令、搜索代码等），" +
			"可以独立完成具体任务。委托后等待结果返回，根据结果决定下一步。\n\n" +
			"每次只委托一个任务，等结果回来再决定下一步。",
		Parameters: objSchema(props{
			"task": strProp("要执行的具体任务描述，需清晰说明做什么、涉及哪些文件、预期产出"),
		}, "task"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			subTask := argStr(args, "task")
			if subTask == "" {
				return "", fmt.Errorf("task 不能为空")
			}

			// 内层 Loop 运行子任务
			msgs, runErr := innerLoop.Run(ctx, subTask, nil)
			if runErr != nil && !errors.Is(runErr, ErrMaxIterations) {
				return "", runErr
			}

			// 提取最终输出
			output := ""
			for i := len(msgs) - 1; i >= 0; i-- {
				if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
					output = msgs[i].Content
					break
				}
			}
			if output == "" {
				output = "(子任务未产出内容)"
			}
			return output, nil
		},
	})

	// 注册 generate_commit_message（外层也需要完成标记）
	outerReg.Register(&Tool{
		Name:        "generate_commit_message",
		Description: "全部任务完成后调用此工具记录提交信息，然后输出最终完成总结。",
		Parameters:  objSchema(props{"message": strProp("描述本次变更的句子")}, "message"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			msg := argStr(args, "message")
			return fmt.Sprintf("提交信息已记录: %s", msg), nil
		},
	})

	// 2. 创建外层 Loop
	outer := &Loop{
		Provider:      planProv,
		Registry:      outerReg,
		System:        outerDesignerPrompt,
		MaxIterations: innerLoop.MaxIterations,
		OnEvent:       innerLoop.OnEvent, // 共用事件推送通道
	}

	// 3. 运行外层 Loop（设计者 agent 开始工作）
	msgs, err := outer.Run(ctx, task, nil)
	if err != nil && !errors.Is(err, ErrMaxIterations) {
		return "", err
	}

	// 4. 提取最终输出
	output := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			output = msgs[i].Content
			break
		}
	}
	return output, nil
}

// outerDesignerPrompt 外层设计者 Agent 的系统提示语。
const outerDesignerPrompt = `你是项目设计者和总指挥。

# 你的角色
你是整个开发任务的**设计者和总指挥**，不是执行者。你的核心价值在于：
1. **理解目标** — 分析用户到底要什么
2. **制定计划** — 设计出完整的执行方案
3. **分派任务** — 把具体工作委托给执行 agent
4. **评估调整** — 根据执行结果动态调整计划

# 核心工具
- **update_plan** — 制定和更新执行计划（步骤清单），展示给用户看整体进度
- **delegate_task** — 把具体任务委托给执行 agent，等它完成后看结果
- **generate_commit_message** — 所有任务完成后调用，记录提交信息，然后输出总结

# 工作流程
1. 先理解用户目标，用 update_plan 列出完整的执行计划
2. 逐项调用 delegate_task 执行，每完成一项就更新 plan 状态
3. 根据每项的实际执行结果决定下一步：继续 / 调整计划 / 标记完成
4. 全部完成后调用 generate_commit_message，输出最终总结

# 重要原则
- **执行 agent 有完整工具集**（读写文件、运行命令、搜索代码等），可以独立完成你委托的任务
- **你不需要做具体执行**，你的工作是设计、决策、协调
- **每次只委托一个任务**，等结果回来后再决定下一步
- **结果不理想时可以调整**后续计划或重新委托
- **保持计划可见**：每次更新 plan 状态，让用户看到整体进度`
