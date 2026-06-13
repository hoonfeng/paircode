package agent

// 统一资产（Asset）管理工具
// 管理进化系统资产（经验胶囊 + 技能基因），Agent 通过以下工具统一管理：
//   - asset_list    查看所有资产（按作用域/类型过滤）
//   - asset_search  搜索资产
//   - asset_delete  删除资产
//
// 资产类型：
//   "capsules" — 经验胶囊（原 evolution_save_capsule）
//   "genes"    — 技能基因（原 evolution_save_gene）

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// registerAssetTools 注册 asset_list / asset_search / asset_delete 工具。
func registerAssetTools(r *Registry, root string) {
	mgr := NewAgentAssetManager(root)

	// ── asset_list ──
	r.Register(&Tool{
		Name: "asset_list",
		Description: "列出所有智能资产（经验胶囊 + 技能基因）。" +
			"可按作用域（global/project）和资产类型（capsules/genes）过滤。",
		Parameters: objSchema(props{
			"scope":  strProp("可选：作用域过滤，\"global\"（全局）或 \"project\"（项目级），不传则列出全部"),
			"type":   strProp("可选：资产类型过滤，\"capsules\"（胶囊）| \"genes\"（基因），不传则列出全部"),
			"detail": boolProp("可选：是否显示详细信息（包括描述、标签等），默认 false 只显示概览"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			scope := argStr(args, "scope")
			assetType := argStr(args, "type")
			detail := argBool(args, "detail")

			// 验证参数
			if scope != "" && scope != "global" && scope != "project" {
				return "", fmt.Errorf("scope 必须为 \"global\"、\"project\" 或空（全部），收到 %q", scope)
			}
			if assetType != "" && assetType != "capsules" && assetType != "genes" {
				return "", fmt.Errorf("type 必须为 \"capsules\"、\"genes\" 或空（全部），收到 %q", assetType)
			}

			assets := mgr.AllAssets(scope, assetType)
			if len(assets) == 0 {
				scopeMsg := "全部作用域"
				if scope != "" {
					scopeMsg = scope
				}
				typeMsg := "全部类型"
				if assetType != "" {
					typeMsg = assetType
				}
				return fmt.Sprintf("暂无智能资产（作用域: %s, 类型: %s）。\n\n"+
					"可使用以下工具创建:\n"+
					"  - `evolution_save_capsule` 保存经验胶囊\n"+
					"  - `evolution_save_gene` 保存技能基因", scopeMsg, typeMsg), nil
			}

			var b strings.Builder
			counts := countByType(assets)
			fmt.Fprintf(&b, "## 智能资产清单（共 %d 个）\n\n", len(assets))
			fmt.Fprintf(&b, "| 类型 | 数量 |\n|------|------|\n")
			fmt.Fprintf(&b, "| 胶囊 (capsules) | %d |\n", counts["capsules"])
			fmt.Fprintf(&b, "| 基因 (genes) | %d |\n", counts["genes"])
			b.WriteString("\n")

			if detail {
				// 详细信息模式
				b.WriteString("### 胶囊 (capsules)\n\n")
				b.WriteString("| ID | 摘要 | 作用域 | 创建时间 |\n")
				b.WriteString("|----|------|--------|----------|\n")
				for _, a := range assets {
					if a.Type != "capsule" {
						continue
					}
					scopeLabel := scopeToLabel(a.Scope)
					created := truncateStr(a.CreatedAt, 10)
					summary := truncateStr(a.Description, 60)
					fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", a.ID, summary, scopeLabel, created)
				}

				b.WriteString("\n### 基因 (genes)\n\n")
				b.WriteString("| 名称 | 描述 | 作用域 | 标签 |\n")
				b.WriteString("|------|------|--------|------|\n")
				for _, a := range assets {
					if a.Type != "gene" {
						continue
					}
					scopeLabel := scopeToLabel(a.Scope)
					tags := strings.Join(a.Tags, ", ")
					if tags == "" {
						tags = "-"
					}
					desc := truncateStr(a.Description, 50)
					fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", a.Name, desc, scopeLabel, tags)
				}
			} else {
				// 概览模式
				b.WriteString("| 名称 | 类型 | 作用域 | 描述 |\n")
				b.WriteString("|------|------|--------|------|\n")
				for _, a := range assets {
					typeLabels := map[string]string{"capsule": "胶囊", "gene": "基因"}
					typeLabel := typeLabels[a.Type]
					if typeLabel == "" {
						typeLabel = a.Type
					}
					scopeLabel := scopeToLabel(a.Scope)
					desc := truncateStr(a.Description, 50)
					fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", a.Name, typeLabel, scopeLabel, desc)
				}
			}

			count := mgr.GetAssetCount()
			b.WriteString("\n---\n")
			b.WriteString(fmt.Sprintf("**统计**: 胶囊 %d | 基因 %d | **合计 %d**\n\n",
				count["capsules_total"], count["genes_total"], count["total"]))
			b.WriteString("**提示**: 使用 `asset_search(query)` 搜索资产，`asset_delete(id, type)` 删除资产\n")
			return b.String(), nil
		},
	})

	// ── asset_search ──
	r.Register(&Tool{
		Name: "asset_search",
		Description: "搜索智能资产（经验胶囊 + 技能基因）。" +
			"按名称、描述、标签进行模糊匹配。",
		Parameters: objSchema(props{
			"query": strProp("搜索关键词（匹配名称、描述、标签）"),
			"scope": strProp("可选：作用域过滤，\"global\" 或 \"project\"，不传则全部"),
			"type":  strProp("可选：资产类型过滤，\"capsules\" | \"genes\"，不传则全部"),
		}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := argStr(args, "query")
			scope := argStr(args, "scope")
			assetType := argStr(args, "type")

			if query == "" {
				return "", fmt.Errorf("query 搜索关键词不能为空")
			}

			assets := mgr.SearchAssets(query, scope, assetType)
			if len(assets) == 0 {
				return fmt.Sprintf("未找到匹配 \"%s\" 的资产。", query), nil
			}

			var b strings.Builder
			fmt.Fprintf(&b, "## 搜索结果: \"%s\"（共 %d 个匹配）\n\n", query, len(assets))
			b.WriteString("| 名称 | 类型 | 作用域 | 描述 |\n")
			b.WriteString("|------|------|--------|------|\n")
			for _, a := range assets {
				typeLabels := map[string]string{"capsule": "胶囊", "gene": "基因"}
				typeLabel := typeLabels[a.Type]
				scopeLabel := scopeToLabel(a.Scope)
				desc := truncateStr(a.Description, 50)
				fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", a.Name, typeLabel, scopeLabel, desc)
			}
			return b.String(), nil
		},
	})

	// ── asset_delete ──
	r.Register(&Tool{
		Name: "asset_delete",
		Description: "删除指定智能资产（经验胶囊 / 技能基因）。" +
			"此操作不可逆。使用 asset_list 查看资产列表获取 ID。",
		Parameters: objSchema(props{
			"id":    strProp("资产 ID（胶囊或基因的 ID 字段）"),
			"type":  strProp("资产类型：\"capsules\" | \"genes\""),
			"scope": strProp("作用域：\"global\" 或 \"project\"，默认 \"global\""),
		}, "id", "type"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argStr(args, "id")
			assetType := argStr(args, "type")
			scope := argStr(args, "scope")
			if scope == "" {
				scope = "global"
			}
			if scope != "global" && scope != "project" {
				return "", fmt.Errorf("scope 必须为 \"global\" 或 \"project\"，收到 %q", scope)
			}
			if assetType != "capsules" && assetType != "genes" {
				return "", fmt.Errorf("type 必须为 \"capsules\" 或 \"genes\"，收到 %q", assetType)
			}

			switch assetType {
			case "capsules":
				store := mgr.Store()
				_ = store.DeleteFile(scope, "capsules", id+".json")
				// 旧路径兼容
				oldPath := filepath.Join(core.InstallDir(), ".pair", "evolution", scope, "capsules", id+".json")
				os.Remove(oldPath)
				return fmt.Sprintf("已删除胶囊 %q（作用域: %s）", id, scope), nil

			case "genes":
				store := mgr.Store()
				_ = store.DeleteFile(scope, "genes", id+".json")
				oldPath := filepath.Join(core.InstallDir(), ".pair", "evolution", scope, "genes", id+".json")
				os.Remove(oldPath)
				return fmt.Sprintf("已删除基因 %q（作用域: %s）", id, scope), nil
			}

			return "", fmt.Errorf("不支持的资产类型: %s", assetType)
		},
	})
}

// ── 辅助 ──────────────────────────────────────────────────

func countByType(assets []AssetInfo) map[string]int {
	counts := map[string]int{"capsules": 0, "genes": 0}
	for _, a := range assets {
		switch a.Type {
		case "capsule":
			counts["capsules"]++
		case "gene":
			counts["genes"]++
		}
	}
	return counts
}

func scopeToLabel(scope string) string {
	switch scope {
	case "global":
		return "全局"
	case "project":
		return "项目"
	default:
		return scope
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
