package agent

// 任务追踪工具 —— 持久化任务管理 + 全量替换模式
// 与 update_plan 对齐：Agent 每次传入完整任务列表即可，不再需要 5 个 CRUD 工具。
//
// 定位：
//   - update_plan：外层编排 agent 用，内存级执行计划
//   - update_tasks：内层执行 agent 用，磁盘持久化任务进度

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

// registerTaskTools 注册 update_tasks 工具（全量替换）。
// 替代之前的 task_create/update/list/delete/summary 5 个工具。
func registerTaskTools(r *Registry, root string) {
	tm := UseTaskManager(root)

	r.Register(&Tool{
		Name: "update_tasks",
		Description: "维护任务列表：传入完整任务清单（全量替换），系统自动持久化到磁盘。" +
			"每项包含 subject（必填）、status（pending/in_progress/completed/cancelled）、description（可选）、dependencies（可选）。" +
			"复杂任务应先用它列出计划，执行中随时更新某任务的状态（每次传全量整份清单）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"tasks": map[string]any{
					"type":        "array",
					"description": "完整任务列表（全量；状态变化时重传整份）",
					"items": map[string]any{
						"type": "object",
						"properties": props{
							"id":           strProp("任务 ID（可选，不传则自动生成）"),
							"subject":      strProp("任务标题，用祈使句（如\"修复登录超时\"）"),
							"description":  strProp("详细描述（可选）：做什么、涉及哪些文件"),
							"status":       map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed", "cancelled"}, "description": "状态"},
							"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "依赖的任务 ID 列表（可选）"},
						},
						"required": []string{"subject", "status"},
					},
				},
			},
			"required": []string{"tasks"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			tasksRaw, _ := args["tasks"].([]any)
			if len(tasksRaw) == 0 {
				return "", fmt.Errorf("tasks 为空")
			}

			newTasks := make([]Task, 0, len(tasksRaw))
			for _, raw := range tasksRaw {
				m, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				t := Task{
					Subject:      argStr(m, "subject"),
					Description:  argStr(m, "description"),
					Status:       TaskPending,
					Dependencies: strSliceArg(m, "dependencies"),
				}
				if id := argStr(m, "id"); id != "" {
					t.ID = id
				}
				if s := argStr(m, "status"); s != "" {
					t.Status = TaskStatus(s)
				}
				newTasks = append(newTasks, t)
			}

			if err := tm.ReplaceAll(newTasks); err != nil {
				return "", fmt.Errorf("保存任务失败: %w", err)
			}

			summary := tm.GetSummary()
			bar := buildProgressBar(summary.Completed, summary.Total, 20)
			ready := tm.GetReady()
			blocked := tm.GetBlocked()

			var b strings.Builder
			fmt.Fprintf(&b, "任务列表已更新：共 %d 项（%d 完成，%d 进行中，%d 待执行）\n",
				summary.Total, summary.Completed, summary.InProgress, summary.Pending)
			fmt.Fprintf(&b, "进度: %s %d/%d (%.0f%%)\n", bar, summary.Completed, summary.Total, pct(summary.Completed, summary.Total))

			if len(ready) > 0 {
				fmt.Fprintf(&b, "\n🔄 可执行任务:\n")
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

// strSliceArg 从 map 中提取 []string 字段（兼容 JSON 反序列化后的 []any）。
func strSliceArg(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []string:
		return arr
	case []any:
		deps := make([]string, 0, len(arr))
		for _, a := range arr {
			if s, ok3 := a.(string); ok3 {
				deps = append(deps, s)
			}
		}
		return deps
	}
	return nil
}
