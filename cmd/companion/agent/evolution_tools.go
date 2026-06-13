package agent

// BES 工具 — Agent 可调用的进化能力接口
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\tools\evolution-tools.ts

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// registerEvolutionTools 注册 evolution_save_capsule/search_capsules/save_gene/status。
func registerEvolutionTools(r *Registry, root string) {
	ee := NewEvolutionEngine(root)

	// ── evolution_save_capsule ──
	r.Register(&Tool{
		Name: "evolution_save_capsule",
		Description: "保存经验胶囊：将修复经验持久化到记忆系统供未来复用。" +
			"当 Agent 成功修复一个错误后调用此工具，将修复过程编码为经验胶囊。",
		Parameters: objSchema(props{
			"errorType": map[string]any{
				"type": "string", "enum": []string{"模块未找到", "类型错误", "运行时错误", "语法错误", "网络错误", "权限错误", "其他"},
				"description": "错误类型",
			},
			"errorPattern": strProp("错误消息模式（用于模糊匹配），如 \"Cannot find module '...'\""),
			"toolName":     strProp("失败的工具名称"),
			"contextTags":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "上下文标签，如 [\"typescript\",\"electron\",\"react\"]"},
			"summary":      strProp("修复摘要，一句话描述修复了什么"),
			"steps":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "修复步骤列表"},
			"keyChanges":   strProp("核心代码变更摘要"),
			"scope":        strProp("作用域：\"global\"（跨项目共享）或 \"project\"（仅当前项目），默认 \"global\""),
		}, "errorType", "errorPattern", "toolName", "summary", "steps", "keyChanges"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			scope := argStr(args, "scope")
			if scope == "" {
				scope = "global"
			}
			capsule := &Capsule{
				ID: fmt.Sprintf("capsule-%s-%d", argStr(args, "errorType"), time.Now().UnixNano()),
				Signal: CapsuleSignal{
					ErrorType:    argStr(args, "errorType"),
					ErrorPattern: argStr(args, "errorPattern"),
					ToolName:     argStr(args, "toolName"),
					ContextTags:  argStrSlice(args, "contextTags"),
				},
				Solution: CapsuleSolution{
					Summary:    argStr(args, "summary"),
					Steps:      argStrSlice(args, "steps"),
					KeyChanges: argStr(args, "keyChanges"),
				},
				Validation: CapsuleValidation{
					Status:       "pending",
					SuccessCount: 0,
				},
				Scope: scope,
			}

			if err := ee.SaveCapsule(capsule); err != nil {
				return "", fmt.Errorf("保存胶囊失败: %w", err)
			}
			return fmt.Sprintf("✅ 经验胶囊已保存: %s\n类型: %s\n作用域: %s\n摘要: %s", capsule.ID, capsule.Signal.ErrorType, scope, capsule.Solution.Summary), nil
		},
	})

	// ── evolution_search_capsules ──
	r.Register(&Tool{
		Name:        "evolution_search_capsules",
		Description: "搜索匹配的经验胶囊：根据错误信息检索历史修复方案。Agent 遇到错误时调用此工具查找已知的修复方案。",
		Parameters: objSchema(props{
			"errorMessage": strProp("错误消息全文，用于模糊匹配历史修复方案"),
			"toolName":     strProp("当前工具名称"),
			"contextTags":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "上下文标签，如 [\"typescript\",\"electron\"]"},
		}, "errorMessage", "toolName"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			errorMessage := argStr(args, "errorMessage")
			toolName := argStr(args, "toolName")
			contextTags := argStrSlice(args, "contextTags")

			capsules := ee.SearchCapsules(errorMessage, toolName, contextTags)
			if len(capsules) == 0 {
				return "未找到匹配的经验胶囊。这是新问题，修复后可用 evolution_save_capsule 保存经验。", nil
			}

			var b strings.Builder
			for i, c := range capsules {
				if i > 0 {
					b.WriteString("\n\n---\n\n")
				}
				scope := c.Scope
				if scope == "" {
					scope = "global"
				}
				fmt.Fprintf(&b, "### 匹配 %d: %s\n", i+1, c.ID)
				fmt.Fprintf(&b, "- 错误类型: %s\n", c.Signal.ErrorType)
				fmt.Fprintf(&b, "- 作用域: %s\n", scope)
				fmt.Fprintf(&b, "- 摘要: %s\n", c.Solution.Summary)
				fmt.Fprintf(&b, "- 验证状态: %s (成功复用 %d 次)\n", c.Validation.Status, c.Validation.SuccessCount)
				fmt.Fprintf(&b, "- 修复步骤:\n")
				for j, step := range c.Solution.Steps {
					fmt.Fprintf(&b, "  %d. %s\n", j+1, step)
				}
				if c.Solution.KeyChanges != "" {
					kc := c.Solution.KeyChanges
					if len(kc) > 300 {
						kc = kc[:300] + "..."
					}
					fmt.Fprintf(&b, "- 关键变更:\n  ```\n  %s\n  ```\n", kc)
				}
			}
			return b.String(), nil
		},
	})

	// ── evolution_save_gene ──
	r.Register(&Tool{
		Name:        "evolution_save_gene",
		Description: "保存技能基因：记录可跨项目复用的编程最佳实践。当发现通用的编程模式或最佳实践时调用此工具。",
		Parameters: objSchema(props{
			"name":        strProp("技能中文名称，如「Electron IPC 安全调用模式」"),
			"description": strProp("技能描述"),
			"category": map[string]any{
				"type": "string", "enum": []string{"架构设计", "设计模式", "配置管理", "调试技巧", "工作流程"},
				"description": "技能分类",
			},
			"languages":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "适用语言，如 [\"typescript\"]"},
			"frameworks": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "适用框架，如 [\"electron\", \"react\"]"},
			"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "自定义标签"},
			"body":       strProp("技能详细描述（Markdown）"),
			"examples":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "示例代码片段"},
			"scope":      strProp("作用域：\"global\"（跨项目共享）或 \"project\"（仅当前项目），默认 \"global\""),
		}, "name", "description", "category", "body"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			scope := argStr(args, "scope")
			if scope == "" {
				scope = "global"
			}
			gene := &Gene{
				ID:          fmt.Sprintf("gene-%s", name),
				Name:        name,
				Description: argStr(args, "description"),
				Category:    argStr(args, "category"),
				Languages:   argStrSlice(args, "languages"),
				Frameworks:  argStrSlice(args, "frameworks"),
				Tags:        argStrSlice(args, "tags"),
				Body:        argStr(args, "body"),
				Examples:    argStrSlice(args, "examples"),
				UsageCount:  0,
				Scope:       scope,
			}

			if err := ee.SaveGene(gene); err != nil {
				return "", fmt.Errorf("保存基因失败: %w", err)
			}
			return fmt.Sprintf("✅ 技能基因已保存: %s\n分类: %s\n作用域: %s\n描述: %s", gene.ID, gene.Category, scope, gene.Description), nil
		},
	})

	// ── evolution_status ──
	r.Register(&Tool{
		Name:        "evolution_status",
		Description: "查看 BES（Bugee Evolution System）状态。返回当前进化引擎的运行状态、项目指纹和已积累的进化资产数量。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			status := ee.GetStatus()
			return fmt.Sprintf("## BES 状态\n\n- 项目指纹: `%s`\n- 经验胶囊数: %d (全局+项目)\n- 技能基因数: %d (全局+项目)\n\n"+
				"可用命令:\n  - `evolution_save_capsule` 保存修复经验\n"+
				"  - `evolution_search_capsules` 搜索历史方案\n"+
				"  - `evolution_save_gene` 保存最佳实践",
				status.Fingerprint, status.CapsuleCount, status.GeneCount), nil
		},
	})
}
