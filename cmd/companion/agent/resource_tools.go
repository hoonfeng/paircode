// 统一资源管理工具 —— Agent 通过以下工具统一查看/搜索/统计所有类型资源：
//   - resource_list    列出所有资源（按类型/作用域过滤）
//   - resource_search  跨类型搜索资源
//   - resource_stats   查看各类型资源数量统计
//
// 资源类型：capsules / genes / memory / project-info / skills / mcp-servers / lua-tools

package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// registerResourceTools 注册 resource_list / resource_search / resource_stats 工具。
func registerResourceTools(r *Registry, root string) {
	mgr := NewUnifiedResourceManager(root)

	// ── resource_list ──
	r.Register(&Tool{
		Name: "resource_list",
		Description: "列出所有智能资源（经验胶囊、技能基因、记忆、项目知识库、技能、MCP服务器、Lua工具）。" +
			"可按类型（type）和作用域（scope）过滤。不传 type 则列出全部类型。",
		Parameters: objSchema(props{
			"type":   strProp("可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"|\"project-info\"|\"skills\"|\"mcp-servers\"|\"lua-tools\"，不传=全部"),
			"scope":  strProp("可选：作用域过滤，\"global\"|\"project\"，不传=全部"),
			"detail": boolProp("可选：是否显示详细信息（含描述、标签等），默认 false"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			resourceType := argStr(args, "type")
			scope := argStr(args, "scope")
			detail := argBool(args, "detail")

			assets := mgr.ListAll(resourceType, scope)
			if len(assets) == 0 {
				msg := "暂未发现任何资源"
				if resourceType != "" {
					msg += fmt.Sprintf("（类型: %s）", resourceType)
				}
				if scope != "" {
					msg += fmt.Sprintf("（作用域: %s）", scope)
				}
				return msg + "。", nil
			}

			var b strings.Builder
			counts := countByResourceType(assets)
			fmt.Fprintf(&b, "## 资源清单（共 %d 项）\n\n", len(assets))

			// 类型统计
			b.WriteString("| 资源类型 | 数量 |\n|----------|------|\n")
			type order struct {
				label string
				count int
			}
			var ordered []order
			for _, rt := range []ResourceType{
				ResourceCapsules, ResourceGenes, ResourceMemory, ResourceProjectInfo,
				ResourceSkills, ResourceMCPServers, ResourceLuaTools,
			} {
				if c, ok := counts[rt]; ok {
					ordered = append(ordered, order{resourceTypeLabel(rt), c})
				}
			}
			for _, o := range ordered {
				fmt.Fprintf(&b, "| %s | %d |\n", o.label, o.count)
			}
			b.WriteString("\n")

			if detail {
				// 详细信息模式：按类型分组显示
				groups := map[ResourceType][]ResourceMeta{}
				for _, a := range assets {
					groups[a.Type] = append(groups[a.Type], a)
				}
				var types []ResourceType
				for rt := range groups {
					types = append(types, rt)
				}
				sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

				for _, rt := range types {
					items := groups[rt]
					fmt.Fprintf(&b, "### %s（%d）\n\n", resourceTypeLabel(rt), len(items))
					b.WriteString("| 名称 | 作用域 | 描述 |\n")
					b.WriteString("|------|--------|------|\n")
					for _, a := range items {
						scopeLabel := scopeToLabel(a.Scope)
						desc := truncateStr(a.Description, 60)
						fmt.Fprintf(&b, "| `%s` | %s | %s |\n", a.Name, scopeLabel, desc)
					}
					b.WriteString("\n")
				}
			} else {
				// 概览模式
				b.WriteString("| 名称 | 类型 | 作用域 |\n")
				b.WriteString("|------|------|--------|\n")
				for _, a := range assets {
					scopeLabel := scopeToLabel(a.Scope)
					typeLabel := resourceTypeLabel(a.Type)
					fmt.Fprintf(&b, "| `%s` | %s | %s |\n", a.Name, typeLabel, scopeLabel)
				}
			}

			b.WriteString("\n---\n")
			b.WriteString("💡 **提示**：\n")
			b.WriteString("- 用 `resource_search(query)` 搜索资源\n")
			b.WriteString("- 用 `resource_stats` 查看统计\n")
			b.WriteString("- 用 `tool_stats` 查看工具调用统计\n")

			return b.String(), nil
		},
	})

	// ── resource_search ──
	r.Register(&Tool{
		Name: "resource_search",
		Description: "跨类型搜索智能资源（经验胶囊、技能基因、记忆、知识库等）。" +
			"按名称、描述、标签模糊匹配。可用 type 和 scope 缩小范围。",
		Parameters: objSchema(props{
			"query": strProp("搜索关键词（必填，匹配名称/描述/标签）"),
			"type":  strProp("可选：资源类型过滤，如 \"capsules\"|\"genes\"|\"memory\"，不传=全部"),
			"scope": strProp("可选：作用域过滤，\"global\"|\"project\"，不传=全部"),
		}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := argStr(args, "query")
			resourceType := argStr(args, "type")
			scope := argStr(args, "scope")

			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}

			assets := mgr.SearchAll(query, resourceType, scope)
			if len(assets) == 0 {
				return fmt.Sprintf("未找到匹配 \"%s\" 的资源。", query), nil
			}

			var b strings.Builder
			fmt.Fprintf(&b, "## 搜索结果: \"%s\"（共 %d 项）\n\n", query, len(assets))
			b.WriteString("| 名称 | 类型 | 作用域 | 描述 |\n")
			b.WriteString("|------|------|--------|------|\n")
			for _, a := range assets {
				typeLabel := resourceTypeLabel(a.Type)
				scopeLabel := scopeToLabel(a.Scope)
				desc := truncateStr(a.Description, 50)
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", a.Name, typeLabel, scopeLabel, desc)
			}
			return b.String(), nil
		},
	})

	// ── resource_stats ──
	r.Register(&Tool{
		Name: "resource_stats",
		Description: "查看各类智能资源的数量统计。直观展示经验胶囊、基因、记忆、" +
			"知识库、技能、MCP服务器、Lua工具的数量分布。",
		Parameters: objSchema(props{
			"scope": strProp("可选：作用域过滤，\"global\"|\"project\"，不传=全部"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			scope := argStr(args, "scope")
			counts := mgr.CountAll(scope)

			var b strings.Builder
			scopeLabel := "全部"
			if scope != "" {
				scopeLabel = scopeToLabel(scope)
			}
			fmt.Fprintf(&b, "## 资源统计（%s）\n\n", scopeLabel)
			b.WriteString("| 资源类型 | 数量 |\n|----------|------|\n")

			total := 0
			for _, rt := range []ResourceType{
				ResourceCapsules, ResourceGenes, ResourceMemory, ResourceProjectInfo,
				ResourceSkills, ResourceMCPServers, ResourceLuaTools,
			} {
				if c, ok := counts[rt]; ok {
					label := resourceTypeLabel(rt)
					fmt.Fprintf(&b, "| %s | %d |\n", label, c)
					total += c
				}
			}
			fmt.Fprintf(&b, "\n**合计**: %d 项资源\n", total)

			b.WriteString("\n---\n")
			b.WriteString("💡 用 `resource_list` 查看明细，`resource_search` 搜索资源。\n")

			return b.String(), nil
		},
	})
}

// ─── 辅助 ──────────────────────────────────────────────────

func resourceTypeLabel(rt ResourceType) string {
	labels := map[ResourceType]string{
		ResourceCapsules:    "经验胶囊",
		ResourceGenes:       "技能基因",
		ResourceMemory:      "项目记忆",
		ResourceProjectInfo: "项目知识库",
		ResourceSkills:      "技能",
		ResourceMCPServers:  "MCP服务器",
		ResourceLuaTools:    "Lua工具",
	}
	if l, ok := labels[rt]; ok {
		return l
	}
	return string(rt)
}

func countByResourceType(assets []ResourceMeta) map[ResourceType]int {
	counts := map[ResourceType]int{}
	for _, a := range assets {
		counts[a.Type]++
	}
	return counts
}
