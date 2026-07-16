// parallel_tools.go 并行执行工具：注册到父 Loop 的 Registry，供 Agent 调用。
//
// 工具列表：
//   - parallel_execute: 分解任务 → 并行执行子 Agent → 汇总结果
//   - parallel_decompose: 仅分解任务（预览计划，不执行）
//   - ctx_set_var / ctx_get_var: 直接读/写父 Loop 的 State（跨 Agent 共享状态）

package agent

import (
	"context"
	"fmt"
)

// RegisterParallelTools 向父 Loop 的 Registry 注册并行执行工具。
// tree 为 Agent 编排树（必须已包含子 Agent）；pool 为共享上下文池（nil 则自动创建）。
func RegisterParallelTools(parent *Loop, tree *AgentTree, pool *SharedContext) {
	reg := parent.Registry
	if pool == nil {
		pool = NewSharedContext()
	}

	// parallel_decompose：仅分解任务为并行子任务（预览用）
	reg.Register(&Tool{
		Name: "parallel_decompose",
		Description: "任务分解预览：分析任务依赖关系，输出可并行的子任务列表和各分组的并行度。" +
			"不会实际执行任何子任务，仅用于预览分解结果。结果是 JSON 格式的分解计划。",
		Parameters: objSchema(props{
			"task": strProp("要分解的复杂任务描述"),
		}, "task"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			task := argStr(args, "task")
			if task == "" {
				return "", fmt.Errorf("task 不能为空")
			}
			orch := NewParallelOrchestrator(parent, tree, pool)
			result, err := orch.DecomposeTask(ctx, task, parent.currentMsgs)
			if err != nil {
				return "", err
			}
			// 格式化输出
			out := fmt.Sprintf("## 任务分解结果\n\n**思路**：%s\n\n**共 %d 个子任务**：\n\n",
				result.Reasoning, len(result.SubTasks))
			for i, t := range result.SubTasks {
				agentName := t.AgentName
				if agentName == "" {
					agentName = "（主Agent）"
				}
				deps := "无依赖"
				if len(t.Dependencies) > 0 {
					deps = "依赖: " + fmt.Sprintf("%v", t.Dependencies)
				}
				out += fmt.Sprintf("%d. **%s** (%s) — %s\n   → 由 %s 执行\n",
					i+1, t.ID, t.Description, deps, agentName)
			}

			out += "\n**并行组**：\n"
			for gi, group := range result.ParallelGroups {
				var names []string
				for _, idx := range group {
					names = append(names, result.SubTasks[idx].ID)
				}
				out += fmt.Sprintf("  - 第 %d 批（可并行）: %v\n", gi+1, names)
			}
			return out, nil
		},
	})

	// parallel_execute：分解 + 并行执行 + 汇总
	reg.Register(&Tool{
		Name: "parallel_execute",
		Description: "并行执行：把复杂任务分解为多个子任务，让多个子 Agent 并行执行。" +
			"子 Agent 共享上下文池（不隔离），一个 Agent 发现的文件/知识可直接被其他 Agent 复用，避免重复收集数据浪费 token。" +
			"适用于：多文件重构、多模块分析、并行调研等场景。",
		Parameters: objSchema(props{
			"task": strProp("要并行执行的复杂任务描述"),
		}, "task"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			task := argStr(args, "task")
			if task == "" {
				return "", fmt.Errorf("task 不能为空")
			}

			orch := NewParallelOrchestrator(parent, tree, pool)

			// 1. 分解任务
			decompose, err := orch.DecomposeTask(ctx, task, parent.currentMsgs)
			if err != nil {
				return "", fmt.Errorf("任务分解失败: %w", err)
			}
			if len(decompose.SubTasks) == 0 {
				return "任务无需分解，可直接执行。", nil
			}

			// 2. 通知分解结果
			parent.emit(Event{Type: EventNotice, Content: fmt.Sprintf(
				"任务分解为 %d 个子任务，分 %d 批并行执行",
				len(decompose.SubTasks), len(decompose.ParallelGroups))})

			// 3. 并行执行
			results, err := orch.ExecuteSubTasks(ctx, decompose.SubTasks)
			if err != nil {
				return "", fmt.Errorf("并行执行出错: %w", err)
			}

			// 4. 汇总结果
			summary := orch.AggregateResults(results)

			// 5. 通知完成
			successCount := 0
			for _, r := range results {
				if r.Error == "" {
					successCount++
				}
			}
			parent.emit(Event{Type: EventNotice, Content: fmt.Sprintf(
				"并行执行完成：%d/%d 子任务成功", successCount, len(results))})

			return summary, nil
		},
	})

	// ctx_set_var：设置父 Loop 的 State（跨 Agent 共享）
	reg.Register(&Tool{
		Name:        "ctx_set_var",
		Description: "设置共享状态变量（跨 Agent 持久化）。所有子 Agent 可读写父 Loop.State 中的变量，用于传递中间结果。注意：被 parallel_execute 中的子 Agent 修改时会自动检测冲突。",
		Parameters:  objSchema(props{"name": strProp("变量名"), "value": strProp("变量值（字符串）")}, "name", "value"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			value := argStr(args, "value")
			if parent.State == nil {
				parent.State = map[string]any{}
			}
			parent.State[name] = value
			pool.SetVar("coordinator", name, value)
			return fmt.Sprintf("已设置共享变量 %s = %s", name, value), nil
		},
	})

	// ctx_get_var：读取父 Loop 的 State
	reg.Register(&Tool{
		Name:        "ctx_get_var",
		Description: "读取共享状态变量（跨 Agent）。返回父 Loop.State 中指定变量的当前值。",
		Parameters:  objSchema(props{"name": strProp("变量名")}, "name"),
		ReadOnly:    true,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			if parent.State == nil {
				return fmt.Sprintf("变量 %q 未设置（State 为空）", name), nil
			}
			v, ok := parent.State[name]
			if !ok {
				return fmt.Sprintf("变量 %q 未设置", name), nil
			}
			return fmt.Sprintf("%s = %v", name, v), nil
		},
	})
}
