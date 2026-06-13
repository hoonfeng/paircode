package agent

// 循环依赖检测工具：find_circular_deps —— 基于 Go import 静态分析，
// 检测项目内部包之间的循环依赖（包A→包B→...→包A），
// 使用 DFS + 路径追踪算法，复用 get_file_dependencies 的底层解析函数。
//
// 无 LSP 依赖，纯文件扫描 + 静态分析，可在任何 Go 项目上运行。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── 工具注册 ──

// registerCircularDepsTool 注册 find_circular_deps 工具。
func registerCircularDepsTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "find_circular_deps",
		Description: "检测 Go 项目中的循环依赖（包导入环）。" +
			"扫描项目中所有 Go 文件的 import 语句，构建包级别依赖图，" +
			"使用 DFS 算法检测循环依赖链（如 pkgA → pkgB → pkgC → pkgA）。" +
			"仅检测项目内部包的循环（以 go.mod 中定义的模块名为前缀的包）。" +
			"输出每个循环的完整链路，帮助开发者定位和修复导入循环问题。" +
			"注意：当前仅支持 Go 语言项目。",
		Parameters: objSchema(props{}),
		ReadOnly:   true,
		Handler:    findCircularDepsHandler(root),
	})
}

// ── 循环检测核心算法 ──

// depGraph 包导入路径 → 它导入的项目内部包路径列表。
type depGraph map[string][]string

// findCircularDepsHandler 创建 find_circular_deps 工具的处理函数。
func findCircularDepsHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		// 1. 读取模块名
		moduleName, err := readGoModuleName(root)
		if err != nil {
			return "", fmt.Errorf("读取 go.mod 失败：%w（find_circular_deps 需要 go.mod 确定模块名）", err)
		}

		// 2. 扫描所有 Go 文件，构建项目内部包依赖图
		graph, pkgOfFile, err := buildInternalDepGraph(root, moduleName)
		if err != nil {
			return "", fmt.Errorf("构建依赖图失败: %w", err)
		}

		if len(graph) == 0 {
			return "项目中未发现内部包依赖关系（所有 Go 文件仅导入标准库或外部依赖）", nil
		}

		// 3. 检测循环
		cycles := detectCycles(graph)

		// 4. 格式化输出
		return formatCycleReport(cycles, graph, pkgOfFile, moduleName, root), nil
	}
}

// buildInternalDepGraph 扫描项目中所有 Go 文件，构建包导入路径的依赖图（仅项目内部依赖）。
// 返回：
//   - graph: 包导入路径 → 它导入的项目内部包路径列表
//   - pkgOfFile: 文件相对路径 → 该文件的包导入路径（用于输出时提供示例文件）
//   - err: 扫描过程中的错误
func buildInternalDepGraph(root, moduleName string) (depGraph, map[string]string, error) {
	graph := make(depGraph)
	pkgOfFile := make(map[string]string) // 相对路径 → 导入路径

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		// 跳过测试文件？（保留测试文件也能检测循环，但测试文件通常不会参与循环）
		// 这里不过滤测试文件，确保检测全面

		// 计算该文件所在目录的包导入路径
		relDir, _ := filepath.Rel(root, filepath.Dir(path))
		relDir = filepath.ToSlash(relDir)
		if relDir == "." {
			// 根目录，直接使用模块名
			// 注意：根目录的 package 导入路径就是模块名自身
		}
		pkgImportPath := moduleName
		if relDir != "." {
			pkgImportPath = moduleName + "/" + relDir
		}

		// 记录文件所属的包（用相对路径）
		if relFilePath, err := filepath.Rel(root, path); err == nil {
			pkgOfFile[filepath.ToSlash(relFilePath)] = pkgImportPath
		}

		// 解析 import
		deps, err := parseGoFileImports(path)
		if err != nil {
			return nil // 跳过无法解析的文件
		}

		// 过滤出项目内部依赖
		var internalDeps []string
		for _, dep := range deps {
			if strings.HasPrefix(dep, moduleName) {
				// 去掉自身（同一个包内的文件互导不算循环依赖）
				if dep != pkgImportPath {
					internalDeps = append(internalDeps, dep)
				}
			}
		}

		if len(internalDeps) > 0 {
			// 去重
			seen := map[string]bool{}
			for _, d := range internalDeps {
				if !seen[d] {
					seen[d] = true
					graph[pkgImportPath] = append(graph[pkgImportPath], d)
				}
			}
		} else {
			// 即使没有内部依赖，也确保节点在图中（可能是独立包）
			if _, exists := graph[pkgImportPath]; !exists {
				graph[pkgImportPath] = nil
			}
		}

		return nil
	})

	return graph, pkgOfFile, err
}

// detectCycles 在依赖图中检测所有循环。
// 使用 DFS + 三色标记法（0=未访问, 1=正在访问, 2=已访问完成）。
// 返回所有检测到的唯一循环（每个循环用 "a→b→c" 格式的 key 去重）。
func detectCycles(graph depGraph) [][]string {
	// 状态：0=未访问, 1=正在访问（在当前DFS路径中）, 2=已访问完成
	state := make(map[string]int)
	// 当前DFS路径上的父节点
	parent := make(map[string]string)

	// 用于去重的 map：key 是循环的规范化表示
	cycleSet := make(map[string]bool)
	var cycles [][]string

	// 初始化所有节点的状态
	for node := range graph {
		state[node] = 0
	}

	// DFS 遍历
	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		state[node] = 1

		for _, neighbor := range graph[node] {
			// 只检查图中存在的节点
			if _, exists := graph[neighbor]; !exists {
				continue
			}

			if state[neighbor] == 0 {
				// 未访问，递归
				parent[neighbor] = node
				dfs(neighbor, append(path, neighbor))
			} else if state[neighbor] == 1 {
				// 发现后向边 → 循环！
				// 从 neighbor 到当前节点的路径构成一个循环
				cycle := extractCycle(path, neighbor)

				// 规范化去重：按字典序排序后拼接
				key := normalizeCycleKey(cycle)
				if !cycleSet[key] {
					cycleSet[key] = true
					cycles = append(cycles, cycle)
				}
			}
		}

		state[node] = 2
	}

	// 对每个未访问节点启动 DFS
	for node := range graph {
		if state[node] == 0 {
			dfs(node, []string{node})
		}
	}

	return cycles
}

// extractCycle 从 DFS 路径中提取从 cycleStart 开始的循环。
// path 是当前 DFS 路径上的节点列表（包含当前节点）。
// cycleStart 是循环的起点（即发现的后向边指向的节点）。
func extractCycle(path []string, cycleStart string) []string {
	// 找到 cycleStart 在路径中的位置
	startIdx := -1
	for i, p := range path {
		if p == cycleStart {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		return nil
	}

	// 提取从 startIdx 开始的路径 → 构成循环
	cycle := make([]string, 0, len(path)-startIdx+1)
	cycle = append(cycle, path[startIdx:]...)
	// 循环最后再加一次起点，形成闭环示例如 a→b→c→a
	cycle = append(cycle, cycleStart)
	return cycle
}

// normalizeCycleKey 将循环路径规范化为唯一 key，用于去重。
// 选择循环中字典序最小的节点作为起点，然后拼接。
func normalizeCycleKey(cycle []string) string {
	if len(cycle) <= 1 {
		return strings.Join(cycle, "→")
	}

	// 找到最小节点索引
	minIdx := 0
	for i := 1; i < len(cycle)-1; i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}

	// 从最小节点开始重新排列（但不包括最后的闭环重复节点）
	n := len(cycle) - 1 // 去掉最后重复的起点
	normalized := make([]string, 0, n)
	for i := 0; i < n; i++ {
		idx := (minIdx + i) % n
		normalized = append(normalized, cycle[idx])
	}

	return strings.Join(normalized, "→")
}

// ── 输出格式 ──

// formatCycleReport 格式化循环检测报告。
func formatCycleReport(cycles [][]string, graph depGraph, pkgOfFile map[string]string, moduleName, root string) string {
	var b strings.Builder

	if len(cycles) == 0 {
		b.WriteString("✅ 未检测到循环依赖！项目内部包之间的依赖关系健康。\n\n")

		// 补充统计信息
		pkgCount := len(graph)
		edgeCount := 0
		for _, deps := range graph {
			edgeCount += len(deps)
		}
		fmt.Fprintf(&b, "■ 统计：%d 个内部包，%d 条内部依赖边\n", pkgCount, edgeCount)
		return b.String()
	}

	// 有循环
	cycleCount := len(cycles)
	fmt.Fprintf(&b, "⚠ 检测到 %d 个循环依赖！\n\n", cycleCount)

	// 通过 sort 使输出稳定
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i][0] < cycles[j][0]
	})

	for i, cycle := range cycles {
		fmt.Fprintf(&b, "─── 循环 #%d ───\n", i+1)

		// 打印循环链路（美观格式）
		for j := 0; j < len(cycle)-1; j++ {
			from := cycle[j]
			fmt.Fprintf(&b, "  %s\n", from)
			fmt.Fprintf(&b, "   ↓ 导入\n")
		}
		fmt.Fprintf(&b, "  %s  → (闭环)\n\n", cycle[len(cycle)-1])

		// 提供每个环节的示例文件
		fmt.Fprintf(&b, "  ■ 循环链路：")
		for j, node := range cycle {
			if j > 0 {
				b.WriteString(" → ")
			}
			b.WriteString(node)
		}
		b.WriteString("\n\n")

		// 为每个环节找到示例文件
		fmt.Fprintf(&b, "  ■ 涉及的文件（每个包选一个示例）：\n")
		seenPkg := map[string]bool{}
		for _, node := range cycle {
			if seenPkg[node] {
				continue
			}
			seenPkg[node] = true

			// 找到一个属于该包的文件
			exampleFile := ""
			for file, pkg := range pkgOfFile {
				if pkg == node {
					exampleFile = file
					break
				}
			}
			if exampleFile != "" {
				fmt.Fprintf(&b, "    %s → %s\n", node, exampleFile)
			} else {
				fmt.Fprintf(&b, "    %s → (未找到源文件)\n", node)
			}
		}
		b.WriteString("\n")
	}

	// 统计
	pkgCount := len(graph)
	edgeCount := 0
	for _, deps := range graph {
		edgeCount += len(deps)
	}
	cyclePkgSet := map[string]bool{}
	for _, cycle := range cycles {
		for _, node := range cycle {
			cyclePkgSet[node] = true
		}
	}
	fmt.Fprintf(&b, "■ 统计：%d 个内部包，%d 条内部依赖边，涉及 %d 个包存在循环\n", pkgCount, edgeCount, len(cyclePkgSet))

	return b.String()
}
