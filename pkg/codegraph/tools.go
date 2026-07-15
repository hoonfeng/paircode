package codegraph

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ── Agent 工具辅助 ──────────────────────────────────────

// 本文件提供 Agent 工具集成所需的辅助函数，将 codegraph 的查询结果
// 格式化为工具友好的返回文本。

// ── 构建工具 ──────────────────────────────────────────

// BuildResultText 将构建结果格式化为工具返回值。
func BuildResultText(result *BuildResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("代码知识图谱构建完成（耗时 %v）\n", result.Duration))
	b.WriteString(fmt.Sprintf("  - 解析文件: %d 个\n", result.FilesParsed))
	b.WriteString(fmt.Sprintf("  - 提取实体: %d 个\n", result.EntitiesAdded))
	b.WriteString(fmt.Sprintf("  - 建立关系: %d 条\n", result.RelationsAdded))

	if len(result.Errors) > 0 {
		b.WriteString(fmt.Sprintf("  - 警告/错误: %d 个\n", len(result.Errors)))
		for _, e := range result.Errors {
			b.WriteString(fmt.Sprintf("    · %s\n", e.Message))
		}
	}
	return b.String()
}

// ── 查询工具 ──────────────────────────────────────────

// FileStructureText 将文件结构树格式化为文本。
func FileStructureText(nodes []FileNode) string {
	if len(nodes) == 0 {
		return "（文件结构中无实体）"
	}
	var b strings.Builder
	b.WriteString("文件结构：\n")
	var printTree func(nodes []FileNode, indent string)
	printTree = func(nodes []FileNode, indent string) {
		for _, node := range nodes {
			kind := string(node.Entity.Kind)
			sig := ""
			if node.Entity.Signature != "" {
				sig = " " + node.Entity.Signature
			}
			b.WriteString(fmt.Sprintf("%s%s: %s (%s:%d)%s\n",
				indent, node.Entity.Name, kind, node.Entity.FilePath, node.Entity.Line, sig))
			if len(node.Children) > 0 {
				printTree(node.Children, indent+"  ")
			}
		}
	}
	printTree(nodes, "  ")
	return b.String()
}

// FunctionDefinitionText 格式化函数定义查询结果。
func FunctionDefinitionText(locs []FuncLocation) string {
	if len(locs) == 0 {
		return "未找到匹配的函数/方法定义。"
	}
	var b strings.Builder
	for i, loc := range locs {
		b.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, loc.Name, loc.Kind))
		b.WriteString(fmt.Sprintf("   文件: %s:%d\n", loc.FilePath, loc.Line))
		if loc.Package != "" {
			b.WriteString(fmt.Sprintf("   包: %s\n", loc.Package))
		}
		if loc.Receiver != "" {
			b.WriteString(fmt.Sprintf("   接收者: %s\n", loc.Receiver))
		}
		if loc.Signature != "" {
			b.WriteString(fmt.Sprintf("   签名: %s\n", loc.Signature))
		}
	}
	return b.String()
}

// ClassHierarchyText 格式化类型层次查询结果。
func ClassHierarchyText(h *ClassHierarchy) string {
	if h == nil {
		return "未找到匹配的类型。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("类型: %s (%s:%d)\n", h.Type.Name, h.Type.FilePath, h.Type.Line))
	b.WriteString(fmt.Sprintf("签名: %s\n", h.Type.Signature))

	if len(h.Fields) > 0 {
		b.WriteString("\n字段:\n")
		for _, f := range h.Fields {
			b.WriteString(fmt.Sprintf("  - %s %s\n", f.Name, f.Signature))
		}
	}
	if len(h.Methods) > 0 {
		b.WriteString("\n方法:\n")
		for _, m := range h.Methods {
			b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", m.Signature, m.FilePath, m.Line))
		}
	}
	if len(h.Embedded) > 0 {
		b.WriteString("\n嵌入类型:\n")
		for _, e := range h.Embedded {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	return b.String()
}

// CallInfoText 格式化调用关系查询结果。
func CallInfoText(calls []CallInfo, title string) string {
	if len(calls) == 0 {
		return fmt.Sprintf("%s: (无匹配)", title)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s（共 %d 条）：\n", title, len(calls)))
	for _, c := range calls {
		b.WriteString(fmt.Sprintf("  %s (%s:%d)\n", c.CallerName, c.CallerFile, c.CallerLine))
	}
	return b.String()
}

// ImpactResultText 格式化影响分析结果。
func ImpactResultText(r *ImpactResult) string {
	if r.StartEntity == nil {
		return r.Summary
	}
	var b strings.Builder
	b.WriteString(r.Summary + "\n\n")

	if len(r.AffectedFiles) > 0 {
		b.WriteString("影响的文件:\n")
		for _, f := range r.AffectedFiles {
			b.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}
	if len(r.AffectedFuncs) > 0 {
		b.WriteString("\n影响的函数/方法:\n")
		for _, f := range r.AffectedFuncs {
			b.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}
	return b.String()
}

// GitHistoryText 格式化 Git 历史结果。
func GitHistoryText(commits []CommitInfo) string {
	if len(commits) == 0 {
		return "未找到相关提交历史。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("提交历史（共 %d 条）：\n\n", len(commits)))
	for _, c := range commits {
		b.WriteString(fmt.Sprintf("  %s | %s | %s\n", c.Hash, c.Date, c.Author))
		b.WriteString(fmt.Sprintf("  %s\n", c.Message))
		if len(c.Files) > 0 {
			// 只显示前 5 个文件
			displayFiles := c.Files
			if len(displayFiles) > 5 {
				displayFiles = displayFiles[:5]
			}
			b.WriteString(fmt.Sprintf("  文件: %s\n", strings.Join(displayFiles, ", ")))
			if len(c.Files) > 5 {
				b.WriteString(fmt.Sprintf("  ... 还有 %d 个文件\n", len(c.Files)-5))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ── 搜索工具 ──────────────────────────────────────────

// GraphStatsText 格式化图谱统计信息。
func GraphStatsText(stats GraphStats) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("代码知识图谱统计:\n"))
	b.WriteString(fmt.Sprintf("  实体总数: %d\n", stats.EntityCount))
	b.WriteString(fmt.Sprintf("  关系总数: %d\n", stats.RelationCount))
	b.WriteString(fmt.Sprintf("  覆盖文件: %d 个\n", stats.FileCount))
	b.WriteString(fmt.Sprintf("  覆盖包:  %d 个\n", stats.PackageCount))

	if len(stats.KindCounts) > 0 {
		b.WriteString("\n各类实体分布:\n")
		// 按数量降序排列
		type kv struct{ k, v string }
		var items []kv
		for k, v := range stats.KindCounts {
			items = append(items, kv{k, fmt.Sprintf("%d", v)})
		}
		for _, item := range items {
			if v := atoi(item.v); v > 0 {
				b.WriteString(fmt.Sprintf("  %s: %s\n", item.k, item.v))
			}
		}
	}
	return b.String()
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// EnsureBuildIfNeeded 检查图谱是否已构建，如未构建则自动触发构建。
// 返回构建后的图谱和是否刚执行了构建。
func EnsureBuildIfNeeded(root string) (*Graph, bool, error) {
	store := NewStore(root)
	if store.Exists() {
		graph, err := store.Load()
		if err != nil {
			return nil, false, err
		}
		return graph, false, nil
	}

	// 自动构建
	moduleName := DetectModuleName(root)
	config := DefaultBuildConfig(root)
	config.ModuleName = moduleName
	config.AutoSave = true

	builder := NewBuilder(config)
	result, err := builder.BuildFull()
	if err != nil {
		return nil, false, err
	}
	_ = result
	return builder.Graph(), true, nil
}

// NormalizeFilePath 规范化文件路径（统一使用正斜杠，相对路径）。

func NormalizeFilePath(root, path string) string {
	p := filepath.ToSlash(path)
	if filepath.IsAbs(path) {
		if rel, err := filepath.Rel(root, path); err == nil {
			p = filepath.ToSlash(rel)
		}
	}
	return p
}
// ── 新增格式函数 ───────────────────────────────────

// PatternSearchText 生成可读的模式搜索结果。
func PatternSearchText(hits []PatternSearchHit) string {
	if len(hits) == 0 {
		return "未找到匹配结果。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("模式搜索结果（共 %d 条）：\n\n", len(hits)))
	for _, h := range hits {
		b.WriteString(fmt.Sprintf("  [%s] %s (%s:%d)\n", h.MatchedIn, h.EntityName, h.FilePath, h.Line))
		if h.Signature != "" {
			b.WriteString(fmt.Sprintf("    签名: %s\n", h.Signature))
		}
		if h.Snippet != "" {
			b.WriteString(fmt.Sprintf("    片段: %s\n", h.Snippet))
		}
	}
	return b.String()
}

// CallChainText 生成可读的调用链树。
func CallChainText(nodes []CallChainNode) string {
	if len(nodes) == 0 {
		return "（无调用链）"
	}
	var b strings.Builder
	b.WriteString("调用链追踪：\n\n")
	var printTree func(nodes []CallChainNode, indent string)
	printTree = func(ns []CallChainNode, indent string) {
		for _, n := range ns {
			direction := ""
			if n.Depth > 0 {
				arrow := "← 被 "
				if n.Depth < 10 {
					arrow = "← "
				}
				direction = arrow
			}
			b.WriteString(fmt.Sprintf("%s%s%s (%s:%d)\n", indent, direction, n.Name, n.FilePath, n.Line))
			if len(n.Children) > 0 {
				printTree(n.Children, indent+"  ")
			}
		}
	}
	printTree(nodes, "")
	return b.String()
}

// DeadCodeText 生成可读的死代码报告。
func DeadCodeText(r *DeadCodeResult) string {
	if r == nil || r.Total == 0 {
		return "未检测到疑似死代码。"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("死代码检测结果（共 %d 个）：\n\n", r.Total))

	if len(r.Functions) > 0 {
		b.WriteString(fmt.Sprintf("■ 函数/方法（%d 个）：\n", len(r.Functions)))
		for _, f := range r.Functions {
			b.WriteString(fmt.Sprintf("  %s (%s:%d) — %s\n", f.Name, f.FilePath, f.Line, f.Reason))
		}
		b.WriteString("\n")
	}
	if len(r.Types) > 0 {
		b.WriteString(fmt.Sprintf("■ 类型（%d 个）：\n", len(r.Types)))
		for _, t := range r.Types {
			b.WriteString(fmt.Sprintf("  %s (%s:%d) — %s\n", t.Name, t.FilePath, t.Line, t.Reason))
		}
		b.WriteString("\n")
	}
	if len(r.Variables) > 0 {
		b.WriteString(fmt.Sprintf("■ 变量/常量（%d 个）：\n", len(r.Variables)))
		for _, v := range r.Variables {
			b.WriteString(fmt.Sprintf("  %s (%s:%d) — %s\n", v.Name, v.FilePath, v.Line, v.Reason))
		}
	}
	return b.String()
}

// ModuleArchitectureText 生成可读的模块架构报告。
func ModuleArchitectureText(arch *ModuleArchitecture) string {
	if arch == nil {
		return "（模块为空）"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("■ 模块: %s\n", arch.Directory))
	b.WriteString(fmt.Sprintf("  文件: %d | 函数: %d\n\n", arch.FileCount, arch.FunctionCount))

	if len(arch.ExportedFuncs) > 0 {
		b.WriteString(fmt.Sprintf("导出函数（%d 个）：\n", len(arch.ExportedFuncs)))
		for _, f := range arch.ExportedFuncs {
			b.WriteString(fmt.Sprintf("  %s\n", f))
		}
		b.WriteString("\n")
	}
	if len(arch.Types) > 0 {
		b.WriteString(fmt.Sprintf("类型（%d 个）：\n", len(arch.Types)))
		for _, t := range arch.Types {
			b.WriteString(fmt.Sprintf("  %s\n", t))
		}
		b.WriteString("\n")
	}
	if len(arch.ComplexHotspots) > 0 {
		b.WriteString("复杂度热点（前 5）：\n")
		for _, h := range arch.ComplexHotspots {
			b.WriteString(fmt.Sprintf("  %s (评分 %s, 复杂度 %d, %d 行)\n", h.Name, h.Grade, h.Complexity, h.LOC))
		}
		b.WriteString("\n")
	}
	if len(arch.Imports) > 0 {
		b.WriteString(fmt.Sprintf("外部依赖（%d 个）：\n", len(arch.Imports)))
		for _, dep := range arch.Imports {
			b.WriteString(fmt.Sprintf("  %s\n", dep))
		}
	}
	return b.String()
}

