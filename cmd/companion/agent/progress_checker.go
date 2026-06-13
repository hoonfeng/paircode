package agent

// 进度检查工具（Progress Checker）
// 帮助 Agent 了解当前任务完成进度，识别未完成的任务并给出建议。
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\tools\progress-checker.ts

import (
	"context"
	"fmt"
	"strings"
)

// registerProgressChecker 注册 progress_checker 工具。
func registerProgressChecker(r *Registry, root string) {
	// 延迟获取 TaskManager（可能在注册时尚未初始化）
	r.Register(&Tool{
		Name: "progress_checker",
		Description: "检查当前任务完成进度，输出结构化进度报告，识别未完成的任务并给出执行建议。" +
			"使用场景：任务列表较长时、Agent 不确定下一步做什么时、或用户要求查看进度时。",
		Parameters: objSchema(props{
			"detail": map[string]any{
				"type": "string", "enum": []string{"summary", "full"},
				"description": "可选：详细模式，设为 \"full\" 显示每个任务的详细信息（含描述）",
			},
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			tm := GetTaskManager()
			if tm == nil {
				return "⚠ 任务管理器未初始化", nil
			}

			detailMode := argStr(args, "detail")
			if detailMode == "" {
				detailMode = "summary"
			}

			allTasks := tm.List("")
			if len(allTasks) == 0 {
				return "📋 **当前没有活跃的任务**\n\n没有需要跟踪的任务进度。可以使用 `task_create` 创建新任务。", nil
			}

			summary := tm.GetSummary()
			ready := tm.GetReady()
			pending := tm.List(TaskPending)
			inProgress := tm.List(TaskInProgress)
			completed := tm.List(TaskCompleted)
			blocked := tm.GetBlocked()

			// 进度条
			total := summary.Total
			done := summary.Completed
			bar := buildProgressBar(done, total, 20)

			var lines []string
			lines = append(lines, fmt.Sprintf("📋 **任务进度报告**"))
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("进度: %s %d/%d (%.0f%%)", bar, done, total, pct(done, total)))
			lines = append(lines, "")
			lines = append(lines, "📊 **状态统计**")
			lines = append(lines, fmt.Sprintf("  - ✅ 已完成: %d", len(completed)))
			lines = append(lines, fmt.Sprintf("  - 🔄 进行中: %d", len(inProgress)))
			lines = append(lines, fmt.Sprintf("  - ⏳ 待执行: %d", len(pending)))
			lines = append(lines, fmt.Sprintf("  - ⏩ 可立即执行: %d", len(ready)))
			if len(blocked) > 0 {
				lines = append(lines, fmt.Sprintf("  - ⛔ 被阻塞: %d", len(blocked)))
			}

			if len(ready) > 0 {
				lines = append(lines, "")
				lines = append(lines, "💡 **建议下一步**")
				for _, task := range ready {
					if len(lines) < 15 { // 最多显示 5 条建议
						lines = append(lines, fmt.Sprintf("  - `[%s]` %s", task.ID, task.Subject))
					}
				}
				if len(ready) > 5 {
					lines = append(lines, fmt.Sprintf("  - ...还有 %d 个待执行任务", len(ready)-5))
				}
			}

			if len(blocked) > 0 {
				lines = append(lines, "")
				lines = append(lines, "⛔ **被阻塞的任务**")
				for _, bt := range blocked {
					if len(lines) < 25 {
						blockers := make([]string, len(bt.BlockedBy))
						for i, b := range bt.BlockedBy {
							blockers[i] = fmt.Sprintf("`[%s]` %s", b.ID, b.Subject)
						}
						lines = append(lines, fmt.Sprintf("  - `[%s]` %s 等待 %s", bt.Task.ID, bt.Task.Subject, strings.Join(blockers, ", ")))
					}
				}
			}

			if detailMode == "full" {
				lines = append(lines, "")
				lines = append(lines, "📄 **所有任务详情**")
				statusIcon := map[TaskStatus]string{
					TaskPending:    "⏳",
					TaskInProgress: "🔄",
					TaskCompleted:  "✅",
					TaskCancelled:  "❌",
				}
				for _, task := range allTasks {
					icon := statusIcon[task.Status]
					if icon == "" {
						icon = "❓"
					}
					depsStr := ""
					if len(task.Dependencies) > 0 {
						depsStr = fmt.Sprintf(" (依赖: %s)", strings.Join(task.Dependencies, ", "))
					}
					desc := task.Description
					if len(desc) > 60 {
						desc = desc[:60] + "..."
					}
					lines = append(lines, fmt.Sprintf("  %s `[%s]` %s%s", icon, task.ID, task.Subject, depsStr))
					if desc != "" {
						lines = append(lines, fmt.Sprintf("     %s", desc))
					}
				}
			}

			lines = append(lines, "")
			lines = append(lines, "💡 **提示**: 使用 `task_create` 创建新任务，`task_update` 更新任务状态。")

			return strings.Join(lines, "\n"), nil
		},
	})
}
