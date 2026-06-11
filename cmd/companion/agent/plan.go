// 规划工具：update_plan —— agent 维护一份可视任务清单（计划分解 + 进度跟踪），复刻 Claude Code TodoWrite。
// 纯数据工具（只读、免审）；清单经事件 Args 传给 UI（见 agent_bridge.go 拦截 update_plan 渲染清单卡）。

package agent

import (
	"context"
	"fmt"
)

func registerPlanTool(r *Registry) {
	r.Register(&Tool{
		Name: "update_plan",
		Description: "维护任务计划清单：传入完整步骤列表（每步 step 描述 + status：pending/in_progress/done）。" +
			"复杂任务应先用它列出计划，执行中随时更新状态（每次传全量整份清单）。清单会展示给用户。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"plan": map[string]any{
					"type":        "array",
					"description": "完整计划步骤（全量；状态变化时重传整份）",
					"items": map[string]any{
						"type": "object",
						"properties": props{
							"step":   strProp("步骤描述"),
							"status": map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "done"}, "description": "状态"},
						},
						"required": []string{"step", "status"},
					},
				},
			},
			"required": []string{"plan"},
		},
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			plan, _ := args["plan"].([]any)
			if len(plan) == 0 {
				return "", fmt.Errorf("plan 为空")
			}
			done := 0
			for _, it := range plan {
				if m, ok := it.(map[string]any); ok {
					if s, _ := m["status"].(string); s == "done" {
						done++
					}
				}
			}
			return fmt.Sprintf("计划已更新：共 %d 步（%d 完成）", len(plan), done), nil
		},
	})
}
