package agent

// 任务追踪工具 —— 持久化任务管理 + 动态调整
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\tools\task-tools.ts

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ── 全局实例（由 bridge 或 RegisterDefaultTools 初始化）──

var (
	globalTM   *TaskManager
	tmInitOnce sync.Once
)

// UseTaskManager 返回全局 TaskManager 实例（一次初始化，后续复用）。
func UseTaskManager(root string) *TaskManager {
	tmInitOnce.Do(func() {
		globalTM = NewTaskManager(root)
	})
	return globalTM
}

// registerTaskTools 注册 task_create/update/list/delete/summary 工具。
func registerTaskTools(r *Registry, root string) {
	tm := UseTaskManager(root)

	// ── task_create ──
	r.Register(&Tool{
		Name: "task_create",
		Description: "创建新的子任务。创建后必须立即执行该任务：先调用 task_update 标记为 in_progress 开始执行，" +
			"执行完成后调用 task_update 标记为 completed 并说明结果。重复此流程直到所有子任务完成。",
		Parameters: objSchema(props{
			"subject":      strProp("任务标题，用祈使句（如\"修复登录超时\"）"),
			"description":  strProp("详细描述：做什么、涉及哪些文件。不要包含文件原始内容，只写摘要。"),
			"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "依赖的任务 ID 列表"},
		}, "subject", "description"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			subject := argStr(args, "subject")
			desc := argStr(args, "description")
			deps := argStrSlice(args, "dependencies")
			task := tm.Create(subject, desc, deps)
			return fmt.Sprintf("✅ 已创建任务 [%s] %s\n> %s\n\n状态: ⏳ 待执行\nID: `%s`", task.ID, task.Subject, task.Description, task.ID), nil
		},
	})

	// ── task_update ──
	r.Register(&Tool{
		Name: "task_update",
		Description: "更新子任务状态或内容。完成一项立即更新一项，不要批量更新。\n\n" +
			"工作流：\n1. 开始执行某任务 → status=in_progress\n" +
			"2. 执行完成后 → status=completed，在 description 中简要说明结果\n" +
			"3. 然后立刻开始下一个待执行任务",
		Parameters: objSchema(props{
			"id":           strProp("任务 ID"),
			"status":       map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed", "cancelled"}, "description": "新状态"},
			"subject":      strProp("新标题（可选）"),
			"description":  strProp("简要说明（可选）"),
			"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "新的依赖列表（可选）"},
		}, "id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			updates := map[string]any{}
			if v := argStr(args, "status"); v != "" {
				updates["status"] = v
			}
			if v := argStr(args, "subject"); v != "" {
				updates["subject"] = v
			}
			if v := argStr(args, "description"); v != "" {
				updates["description"] = v
			}
			if deps := argStrSlice(args, "dependencies"); deps != nil {
				updates["dependencies"] = deps
			}

			task := tm.Update(id, updates)
			if task == nil {
				return "", fmt.Errorf("任务 %s 不存在", id)
			}

			statusLabel := map[TaskStatus]string{
				TaskPending:    "⏳ 待执行",
				TaskInProgress: "🔄 进行中",
				TaskCompleted:  "✅ 已完成",
				TaskCancelled:  "❌ 已取消",
			}
			label := statusLabel[task.Status]
			if label == "" {
				label = string(task.Status)
			}

			ready := tm.GetReady()
			readyHint := ""
			if len(ready) > 0 {
				items := make([]string, 0, len(ready))
				for _, t := range ready {
					items = append(items, fmt.Sprintf("`[%s] %s`", t.ID, t.Subject))
				}
				readyHint = fmt.Sprintf("\n\n🔄 下一步可执行: %s", strings.Join(items, "、"))
			}

			summary := tm.GetSummary()
			bar := buildProgressBar(summary.Completed, summary.Total, 20)
			progress := fmt.Sprintf("\n> 进度: %s %d/%d", bar, summary.Completed, summary.Total)

			return fmt.Sprintf("**任务 [%s] %s** → %s%s%s", task.ID, task.Subject, label, readyHint, progress), nil
		},
	})

	// ── task_list ──
	r.Register(&Tool{
		Name:        "task_list",
		Description: "列出当前所有子任务及其状态。用于查看进度和规划下一步。",
		Parameters: objSchema(props{
			"status": map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed", "cancelled"}, "description": "按状态筛选（可选，不传则列出全部）"},
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			statusStr := argStr(args, "status")
			var status TaskStatus
			if statusStr != "" {
				status = TaskStatus(statusStr)
			}
			tasks := tm.List(status)
			if len(tasks) == 0 {
				if statusStr != "" {
					return fmt.Sprintf("（没有 %s 状态的任务）", statusStr), nil
				}
				return "（暂无任务。可用 task_create 创建。）", nil
			}

			statusIcon := map[TaskStatus]string{
				TaskPending:    "⏳",
				TaskInProgress: "🔄",
				TaskCompleted:  "✅",
				TaskCancelled:  "❌",
			}

			var b strings.Builder
			fmt.Fprintf(&b, "## 任务列表（共 %d 个）\n\n", len(tasks))
			for _, t := range tasks {
				icon := statusIcon[t.Status]
				if icon == "" {
					icon = "❓"
				}
				deps := ""
				if len(t.Dependencies) > 0 {
					deps = fmt.Sprintf(" [依赖: %s]", strings.Join(t.Dependencies, ", "))
				}
				fmt.Fprintf(&b, "%s `[%s]` %s%s\n", icon, t.ID, t.Subject, deps)
				if t.Description != "" && len(t.Description) < 80 {
					fmt.Fprintf(&b, "   %s\n", t.Description)
				}
			}
			return b.String(), nil
		},
	})

	// ── task_delete ──
	r.Register(&Tool{
		Name:             "task_delete",
		Description:      "删除一个不再需要的子任务。会自动清理其他任务对此任务的依赖。",
		Parameters:       objSchema(props{"id": strProp("要删除的任务 ID")}, "id"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			ok := tm.Delete(id)
			if !ok {
				return "", fmt.Errorf("任务 %s 不存在", id)
			}
			return fmt.Sprintf("✅ 已删除任务 %s", id), nil
		},
	})

	// ── task_summary ──
	r.Register(&Tool{
		Name:        "task_summary",
		Description: "获取任务进度摘要，包括完成率、阻塞任务列表。用于快速了解项目状态。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			summary := tm.GetSummary()
			blocked := tm.GetBlocked()
			ready := tm.GetReady()

			var b strings.Builder
			bar := buildProgressBar(summary.Completed, summary.Total, 20)
			fmt.Fprintf(&b, "总任务: %d | 已完成: %d | 进行中: %d | 待执行: %d | 已取消: %d\n",
				summary.Total, summary.Completed, summary.InProgress, summary.Pending, summary.Cancelled)
			fmt.Fprintf(&b, "进度: %s %d/%d (%.0f%%)\n", bar, summary.Completed, summary.Total, pct(summary.Completed, summary.Total))

			if len(ready) > 0 {
				fmt.Fprintf(&b, "\n🔄 可执行任务 (%d):\n", len(ready))
				for _, t := range ready {
					fmt.Fprintf(&b, "  [%s] %s\n", t.ID, t.Subject)
				}
			}

			if len(blocked) > 0 {
				fmt.Fprintf(&b, "\n⛔ 阻塞任务 (%d):\n", len(blocked))
				for _, bt := range blocked {
					blockers := make([]string, len(bt.BlockedBy))
					for i, b := range bt.BlockedBy {
						blockers[i] = fmt.Sprintf("[%s] %s", b.ID, b.Subject)
					}
					fmt.Fprintf(&b, "  [%s] %s ← 等待: %s\n", bt.Task.ID, bt.Task.Subject, strings.Join(blockers, ", "))
				}
			}

			return b.String(), nil
		},
	})
}

// ── 辅助 ───────────────────────────────────────────────────

func buildProgressBar(done, total, width int) string {
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	filled := done * width / total
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func pct(done, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(done) * 100 / float64(total)
}
