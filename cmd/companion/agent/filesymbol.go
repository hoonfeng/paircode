package agent

// 文件符号工具：get_file_symbols —— 基于 LSP 的 textDocument/documentSymbol，
// 让 Agent 能获取指定文件中所有符号（函数、类型、变量等）的列表和位置。
//
// 本工具直接与语言服务器通过 JSON-RPC over stdio 通信，不依赖 external LSP 包，
// 以确保 agent 包保持纯 Go 标准库依赖（无 CGO/goui 耦合）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)


// ── 工具注册 ──

// registerFileSymbolTools 注册符号和依赖分析相关的工具。
func registerFileSymbolTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "get_file_symbols",
		Description: "列出指定文件中所有检测到的符号（函数、类、接口、类型、常量、变量等），" +
			"返回每个符号的名称、种类、所在行号及子符号。基于 LSP（语言服务器协议）的 documentSymbol 能力，" +
			"支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java / JSON / HTML / CSS / YAML 等语言。",
		Parameters: objSchema(props{
			"filePath": strProp("文件路径（工作区内）"),
		}, "filePath"),
		ReadOnly: true,
		Handler:  getFileSymbolsHandler(root),
	})

	r.Register(&Tool{
		Name: "find_symbol_usages",
		Description: "搜索指定符号在项目中的所有引用位置（使用 LSP textDocument/references 能力）。" +
			"先通过 documentSymbol 在给定文件中定位符号定义，再查询其全部引用。返回每个引用的文件路径、行号和摘要上下文。" +
			"支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java 等语言。",
		Parameters: objSchema(props{
			"name":     strProp("符号名称（如函数名、类型名、变量名）"),
			"filePath": strProp("符号所在的文件路径（用于定位符号定义位置，必填）"),
		}, "name", "filePath"),
		ReadOnly: true,
		Handler:  findSymbolUsagesHandler(root),
	})

	r.Register(&Tool{
		Name: "list_exported_symbols",
		Description: "列出项目中所有导出的符号（首字母大写的函数、类型、结构体、接口、常量、变量），" +
			"通过静态分析扫描 Go 源文件，使用正则表达式提取导出符号定义。" +
			"支持按名称过滤（query）和按符号类型过滤（kind）。" +
			"注意：当前仅支持 Go 语言项目；无需安装语言服务器。",
		Parameters: objSchema(props{
			"query": strProp("可选：按符号名称过滤（子串匹配，大小写不敏感）"),
			"kind":  strProp("可选：按符号类型过滤（function, method, type, struct, interface, const, var, enum, class 等）"),
			"limit": strProp("可选：最大返回结果数（默认 200，最大 500）"),
		}),
		ReadOnly: true,
		Handler:  listExportedSymbolsHandler(root),
	})

	r.Register(&Tool{
		Name: "get_file_dependencies",
		Description: "获取指定 Go 文件的依赖关系：解析其 import 语句得到直接依赖，并扫描整个项目中导入其所在包的文件（反向依赖）。" +
			"将依赖分类为标准库、项目内部依赖和外部依赖。注意：当前仅支持 Go 语言。",
		Parameters: objSchema(props{
			"filePath": strProp("Go 文件路径（工作区内）"),
		}, "filePath"),
		ReadOnly: true,
		Handler:  getFileDependenciesHandler(root),
	})

	registerCircularDepsTool(r, root) // find_circular_deps（见 circulardeps.go）

	r.Register(&Tool{
		Name: "check_impact",
		Description: "分析修改指定文件后的影响范围：递归遍历整个项目的依赖关系图（基于 BFS），" +
			"找出所有直接和间接受该文件变更影响的文件（反向依赖链）。" +
			"通过扫描 Go import 语句实现，无需语言服务器。输出按影响层级组织的树形结构。" +
			"注意：当前仅支持 Go 语言；BFS 深度默认限制 10 层，最多返回 200 个文件。",
		Parameters: objSchema(props{
			"filePath": strProp("要检查的文件路径（工作区内）"),
		}, "filePath"),
		ReadOnly: true,
		Handler:  checkImpactHandler(root),
	})
}

// ── 工具处理函数 ──

func getFileSymbolsHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		filePath := argStr(args, "filePath")
		if filePath == "" {
			return "", fmt.Errorf("filePath 不能为空")
		}
		mgr := getLSPManager(root)
		return mgr.DocumentSymbol(ctx, filePath)
	}
}

// ── find_symbol_usages 处理函数 ──

func findSymbolUsagesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		name := argStr(args, "name")
		filePath := argStr(args, "filePath")
		if name == "" || filePath == "" {
			return "", fmt.Errorf("name 和 filePath 不能为空")
		}
		mgr := getLSPManager(root)
		return mgr.References(ctx, filePath, 1, name)
	}
}

// 返回去重后的位置列表（可能存在重名符号如重载）。

type exportedSymbol struct {
	Name     string // 符号名
	Kind     string // 种类：function, struct, interface, constant, variable, type
	FilePath string // 文件路径（项目内相对路径）
	Line     int    // 行号（1 基）
}

// goExportedSymbolRegexps 按优先级顺序匹配 Go 导出符号的正则列表。
// 每个元组：(匹配正则, 种类名, 从匹配中提取符号名的子捕获组索引)
var goExportedSymbolRegexps = []struct {
	pattern   *regexp.Regexp
	kind      string
	nameGroup int
}{
	{regexp.MustCompile(`^type\s+([A-Z]\w*)\s+struct`), "struct", 1},
	{regexp.MustCompile(`^type\s+([A-Z]\w*)\s+interface`), "interface", 1},
	{regexp.MustCompile(`^type\s+([A-Z]\w*)`), "type", 1},
	{regexp.MustCompile(`^func\s+([A-Z]\w*)\s*\(`), "function", 1},
	{regexp.MustCompile(`^const\s+([A-Z]\w*)`), "constant", 1},
	{regexp.MustCompile(`^var\s+([A-Z]\w*)`), "variable", 1},
}

// scanGoExportedSymbols 扫描一个 Go 文件，返回其中所有导出符号。
func scanGoExportedSymbols(absPath, relPath string) ([]exportedSymbol, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var syms []exportedSymbol

	for lineIdx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		for _, rule := range goExportedSymbolRegexps {
			matches := rule.pattern.FindStringSubmatch(trimmed)
			if matches != nil && len(matches) > rule.nameGroup {
				name := matches[rule.nameGroup]
				// 确保首字母大写
				r, _ := utf8.DecodeRuneInString(name)
				if !unicode.IsUpper(r) {
					continue
				}
				syms = append(syms, exportedSymbol{
					Name:     name,
					Kind:     rule.kind,
					FilePath: relPath,
					Line:     lineIdx + 1, // 1 基
				})
				break // 一个符号只匹配一次
			}
		}
	}
	return syms, nil
}

// kindNameToGroup 将用户传入的 kind 参数匹配到内部种类分组名。
var kindNameAliases = map[string]string{
	"function":  "function",
	"func":      "function",
	"method":    "function",
	"struct":    "struct",
	"interface": "interface",
	"type":      "type",
	"constant":  "constant",
	"const":     "constant",
	"variable":  "variable",
	"var":       "variable",
	"class":     "type",
	"enum":      "type",
}

// listExportedSymbolsHandler 创建 list_exported_symbols 工具的处理函数。
// 使用静态文件扫描方式（正则表达式匹配 Go 源文件），不依赖 LSP 语言服务器。
// 支持 Go 语言项目的导出符号扫描（function、struct、interface、constant、variable、type）。
func listExportedSymbolsHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		query := argStr(args, "query")
		kindStr := argStr(args, "kind")
		limit := argInt(args, "limit", 200)
		if limit <= 0 {
			limit = 200
		}
		if limit > 500 {
			limit = 500
		}

		// 扫描项目中的所有 Go 源文件
		var allSyms []exportedSymbol
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			syms, err := scanGoExportedSymbols(path, filepath.ToSlash(rel))
			if err != nil {
				return nil // 跳过不可读文件
			}
			allSyms = append(allSyms, syms...)
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("扫描项目文件失败: %w", err)
		}

		if len(allSyms) == 0 {
			return "（项目中未找到导出符号）", nil
		}

		// 过滤
		var filtered []exportedSymbol
		for _, s := range allSyms {
			// 按名称过滤
			if query != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(query)) {
				continue
			}
			// 按 kind 过滤（支持别名）
			if kindStr != "" {
				expected, ok := kindNameAliases[strings.ToLower(kindStr)]
				if !ok || s.Kind != expected {
					continue
				}
			}
			filtered = append(filtered, s)
		}

		if len(filtered) == 0 {
			if kindStr != "" {
				return fmt.Sprintf("（未找到匹配 kind=%q 的导出符号）", kindStr), nil
			}
			return "（项目中未找到导出符号）", nil
		}

		// 截断
		if len(filtered) > limit {
			filtered = filtered[:limit]
		}

		// 格式化输出
		return formatExportedSymbolsStatic(filtered, limit), nil
	}
}

// ── get_file_dependencies 工具：静态分析 Go 文件的 import 依赖 ──

// goImportRegex 匹配单行 import 语句：import [alias] "path"
// 可含别名（_、.、标识符）或无别名。
var goImportRegex = regexp.MustCompile(`import\s+(?:(?:_|\.|\w+)\s+)?"([^"]+)"`)

// goImportBlockRegex 匹配 import 块中的每一行引号包路径。
var goImportBlockLineRegex = regexp.MustCompile(`^\s*(?:(?:_|\.|\w+)\s+)?"([^"]+)"`)

// goPackageRegex 匹配 package 声明。
var goPackageRegex = regexp.MustCompile(`^package\s+(\w+)`)

// goModuleRegex 匹配 go.mod 中的 module 声明。
var goModuleRegex = regexp.MustCompile(`^module\s+(\S+)`)

// parseGoFileImports 读取 Go 文件并解析其所有 import 的包路径（去重）。
func parseGoFileImports(absPath string) ([]string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	content := string(data)
	seen := map[string]bool{}
	var pkgs []string

	lines := strings.Split(content, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// 跳过注释
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			// 跳过块注释
			if strings.HasPrefix(line, "/*") {
				for i < len(lines) && !strings.Contains(lines[i], "*/") {
					i++
				}
			}
			i++
			continue
		}

		// 匹配 import "pkg"
		if matches := goImportRegex.FindStringSubmatch(line); matches != nil {
			pkg := matches[1]
			if !seen[pkg] {
				seen[pkg] = true
				pkgs = append(pkgs, pkg)
			}
			i++
			continue
		}

		// 匹配 import ( ... ) 块
		if strings.HasPrefix(line, "import") && strings.Contains(line, "(") {
			i++
			for i < len(lines) {
				cline := strings.TrimSpace(lines[i])
				if strings.HasPrefix(cline, "//") || strings.HasPrefix(cline, "/*") {
					if strings.HasPrefix(cline, "/*") {
						for i < len(lines) && !strings.Contains(lines[i], "*/") {
							i++
						}
					}
					i++
					continue
				}
				if cline == ")" {
					break
				}
				if m := goImportBlockLineRegex.FindStringSubmatch(cline); m != nil {
					pkg := m[1]
					if !seen[pkg] {
						seen[pkg] = true
						pkgs = append(pkgs, pkg)
					}
				}
				i++
			}
			i++
			continue
		}

		i++
	}

	return pkgs, nil
}

// parseGoPackageName 读取 Go 文件首行的 package 名。
func parseGoPackageName(absPath string) (string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || trimmed == "" {
			continue
		}
		if m := goPackageRegex.FindStringSubmatch(trimmed); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("未找到 package 声明")
}

// readGoModuleName 从 go.mod 中读取模块名。
func readGoModuleName(root string) (string, error) {
	gm := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(gm)
	if err != nil {
		return "", fmt.Errorf("读取 go.mod 失败: %w（get_file_dependencies 需要 go.mod 来确定导入路径）", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if m := goModuleRegex.FindStringSubmatch(trimmed); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("go.mod 中未找到 module 声明")
}

// scanReverseDependencies 扫描项目中所有 Go 文件，找出导入目标包路径的所有文件。
// skipFile 是要排除的文件自身。
func scanReverseDependencies(root, targetPkg, skipFile string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		abs, _ := filepath.Abs(path)
		if abs == skipFile {
			return nil
		}
		imports, err := parseGoFileImports(abs)
		if err != nil {
			return nil // 跳过不可读文件
		}
		for _, imp := range imports {
			if imp == targetPkg {
				rel, _ := filepath.Rel(root, path)
				files = append(files, filepath.ToSlash(rel))
				break
			}
		}
		return nil
	})
	return files, err
}

// getFileDependenciesHandler 创建 get_file_dependencies 工具的处理函数。
func getFileDependenciesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		filePath := argStr(args, "filePath")
		if filePath == "" {
			return "", fmt.Errorf("filePath 不能为空")
		}
		absPath, err := resolvePath(root, filePath)
		if err != nil {
			return "", err
		}
		if filepath.Ext(absPath) != ".go" {
			return "", fmt.Errorf("get_file_dependencies 当前仅支持 Go 文件（.go），不支持 %s", filePath)
		}

		// 1. 解析 import 依赖
		deps, err := parseGoFileImports(absPath)
		if err != nil {
			return "", fmt.Errorf("解析 import 失败: %w", err)
		}

		// 2. 获取文件自身的 package 名
		pkgName, err := parseGoPackageName(absPath)
		if err != nil {
			pkgName = "(未知)"
		}

		// 3. 读取模块名，构造包导入路径
		moduleName, err := readGoModuleName(root)
		if err != nil {
			moduleName = ""
		}

		// 4. 计算当前文件所在目录的导入路径
		relDir := ""
		if moduleName != "" {
			rel, _ := filepath.Rel(root, filepath.Dir(absPath))
			relDir = filepath.ToSlash(rel)
		}

		// 5. 扫描反向依赖
		selfImportPath := ""
		if moduleName != "" && relDir != "" {
			selfImportPath = moduleName + "/" + relDir
		}
		var reverseDeps []string
		if selfImportPath != "" {
			reverseDeps, _ = scanReverseDependencies(root, selfImportPath, absPath)
		}
		if reverseDeps == nil {
			reverseDeps = []string{}
		}

		// 格式化输出
		var b strings.Builder
		fmt.Fprintf(&b, "文件：%s\n", filePath)
		fmt.Fprintf(&b, "包名：%s\n\n", pkgName)

		// 分类依赖：标准库 vs 项目依赖 vs 外部依赖
		var stdLibs, projDeps, extDeps []string
		for _, d := range deps {
			if moduleName != "" && strings.HasPrefix(d, moduleName) {
				projDeps = append(projDeps, d)
			} else if !strings.Contains(d, ".") {
				stdLibs = append(stdLibs, d)
			} else {
				extDeps = append(extDeps, d)
			}
		}
		sort.Strings(stdLibs)
		sort.Strings(projDeps)
		sort.Strings(extDeps)

		fmt.Fprintf(&b, "■ 依赖总计：%d 个\n\n", len(deps))

		if len(stdLibs) > 0 {
			fmt.Fprintf(&b, "▸ 标准库（%d 个）：\n", len(stdLibs))
			for _, d := range stdLibs {
				fmt.Fprintf(&b, "  • %s\n", d)
			}
			b.WriteString("\n")
		}
		if len(projDeps) > 0 {
			fmt.Fprintf(&b, "▸ 项目内部（%d 个）：\n", len(projDeps))
			for _, d := range projDeps {
				fmt.Fprintf(&b, "  • %s\n", d)
			}
			b.WriteString("\n")
		}
		if len(extDeps) > 0 {
			fmt.Fprintf(&b, "▸ 外部依赖（%d 个）：\n", len(extDeps))
			for _, d := range extDeps {
				fmt.Fprintf(&b, "  • %s\n", d)
			}
			b.WriteString("\n")
		}

		// 反向依赖
		fmt.Fprintf(&b, "■ 反向依赖（导入 %s 的文件）：%d 个\n", pkgName, len(reverseDeps))
		if len(reverseDeps) > 0 {
			sort.Strings(reverseDeps)
			for _, f := range reverseDeps {
				fmt.Fprintf(&b, "  • %s\n", f)
			}
		} else if selfImportPath != "" {
			fmt.Fprintf(&b, "  （项目中无其他文件导入当前包）\n")
		} else {
			fmt.Fprintf(&b, "  （无法确定包导入路径，跳过反向依赖扫描）\n")
		}

		return b.String(), nil
	}
}

// ── check_impact 工具：递归分析文件变更影响范围 ──

// impactResult 表示一层影响结果。
type impactResult struct {
	FilePath    string         // 受影响文件路径（项目内相对路径）
	Depth       int            // 影响层级（0=目标文件自身）
	PackagePath string         // 该文件的导入路径
	ImportedBy  []impactResult // 受该文件影响的下层文件（反向依赖它的文件）
}

// checkImpactHandler 创建 check_impact 工具的处理函数。
// 基于 get_file_dependencies 的底层函数，递归遍历依赖关系图，
// 找出修改指定文件后所有可能受影响的文件（直接/间接依赖该文件包的文件）。
func checkImpactHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		filePath := argStr(args, "filePath")
		if filePath == "" {
			return "", fmt.Errorf("filePath 不能为空")
		}
		absPath, err := resolvePath(root, filePath)
		if err != nil {
			return "", err
		}
		if filepath.Ext(absPath) != ".go" {
			return "", fmt.Errorf("check_impact 当前仅支持 Go 文件（.go），不支持 %s", filePath)
		}

		// 读取模块名
		moduleName, err := readGoModuleName(root)
		if err != nil {
			return "", fmt.Errorf("读取 go.mod 失败: %w", err)
		}

		// 计算目标文件所在目录的导入路径
		relDir, _ := filepath.Rel(root, filepath.Dir(absPath))
		relDir = filepath.ToSlash(relDir)
		targetImportPath := moduleName + "/" + relDir

		// 解析目标文件的 package 名
		pkgName, err := parseGoPackageName(absPath)
		if err != nil {
			pkgName = "(未知)"
		}

		// 获取目标文件的直接依赖
		deps, err := parseGoFileImports(absPath)
		if err != nil {
			deps = nil
		}

		// BFS 遍历反向依赖图
		type queueItem struct {
			importPath string // 要扫描的导入路径
			filePath   string // 来源文件路径（用于展示层级）
			depth      int    // 当前深度
			parentIdx  int    // 在 result 切片中的父节点索引（-1 表示根）
		}

		// 找导入某包的所有文件
		// cache 避免重复扫描同一个包的依赖
		reverseCache := map[string][]string{} // importPath → 文件列表

		// results 存放所有找到的受影响文件
		type impactNode struct {
			filePath string
			depth    int
			parent   int // parent index in results
			idx      int // self index
		}
		var results []impactNode

		// visited 用于去重文件路径
		visited := map[string]int{} // filePath → index in results

		// 初始队列：扫描目标文件自身的包路径
		queue := []queueItem{{importPath: targetImportPath, filePath: filePath, depth: 0, parentIdx: -1}}

		// 将目标文件自身加入结果
		selfIdx := 0
		results = append(results, impactNode{
			filePath: filePath,
			depth:    0,
			parent:   -1,
			idx:      0,
		})
		visited[filePath] = 0

		maxDepth := 10  // 最大递归深度
		maxFiles := 200 // 最大结果数

		// BFS
		for len(queue) > 0 && len(results) < maxFiles {
			item := queue[0]
			queue = queue[1:]

			if item.depth >= maxDepth {
				continue
			}

			// 从缓存或实际扫描获取反向依赖
			reverseFiles, ok := reverseCache[item.importPath]
			if !ok {
				// 扫描导入此包的所有文件（跳过目标文件自身）
				skipFile := ""
				if item.depth == 0 {
					skipFile = absPath
				}
				files, err := scanReverseDependencies(root, item.importPath, skipFile)
				if err != nil {
					continue
				}
				reverseCache[item.importPath] = files
				reverseFiles = files
			}

			for _, rf := range reverseFiles {
				if len(results) >= maxFiles {
					break
				}

				// 已访问过则跳过（避免循环）
				if _, seen := visited[rf]; seen {
					continue
				}

				// 获取反向依赖文件的绝对路径
				rfAbs := filepath.Join(root, rf)

				// 计算该文件的导入路径
				rfRelDir, _ := filepath.Rel(root, filepath.Dir(rfAbs))
				rfRelDir = filepath.ToSlash(rfRelDir)
				rfImportPath := moduleName + "/" + rfRelDir

				// 找到父节点在 results 中的索引
				parentIdx := -1
				if item.depth == 0 && item.filePath == filePath {
					parentIdx = selfIdx
				} else if item.parentIdx >= 0 && item.parentIdx < len(results) {
					// 对于嵌套层级，找最近的上层节点
					parentIdx = item.parentIdx
				}

				newIdx := len(results)
				results = append(results, impactNode{
					filePath: rf,
					depth:    item.depth + 1,
					parent:   parentIdx,
					idx:      newIdx,
				})
				visited[rf] = newIdx

				// 将新文件加入队列，继续扫描它的反向依赖
				queue = append(queue, queueItem{
					importPath: rfImportPath,
					filePath:   rf,
					depth:      item.depth + 1,
					parentIdx:  newIdx,
				})
			}
		}

		// 格式化输出
		var b strings.Builder
		fmt.Fprintf(&b, "文件：%s\n", filePath)
		fmt.Fprintf(&b, "包名：%s\n", pkgName)
		fmt.Fprintf(&b, "导入路径：%s\n", targetImportPath)
		if len(deps) > 0 {
			fmt.Fprintf(&b, "\n■ 直接依赖：%d 个包\n", len(deps))
		}

		// 构建树形输出
		if len(results) <= 1 {
			fmt.Fprintf(&b, "\n■ 影响分析：无其他文件受当前文件影响\n")
			return b.String(), nil
		}

		fmt.Fprintf(&b, "\n■ 影响分析：修改 %s 后，以下文件可能受影响：\n\n", filePath)

		// 按层级组织
		type levelInfo struct {
			files []impactNode
		}
		levels := map[int]*levelInfo{}
		maxLevel := 0
		for i, r := range results {
			if i == 0 {
				continue // 跳过自身
			}
			if levels[r.depth] == nil {
				levels[r.depth] = &levelInfo{}
			}
			levels[r.depth].files = append(levels[r.depth].files, r)
			if r.depth > maxLevel {
				maxLevel = r.depth
			}
		}

		for d := 1; d <= maxLevel; d++ {
			lvl := levels[d]
			if lvl == nil {
				continue
			}
			// 按文件名排序
			sort.Slice(lvl.files, func(i, j int) bool {
				return lvl.files[i].filePath < lvl.files[j].filePath
			})
			prefix := "│  "
			mark := "├─"
			if d == 1 {
				fmt.Fprintf(&b, "  第 1 层（直接依赖目标包）：\n")
				prefix = "   "
				mark = "  • "
			} else {
				fmt.Fprintf(&b, "  第 %d 层（间接依赖）：\n", d)
				prefix = "   "
				mark = "  • "
			}
			for _, f := range lvl.files {
				fmt.Fprintf(&b, "%s%s%s\n", prefix, mark, f.filePath)
			}
			b.WriteString("\n")
		}

		// 统计信息
		total := len(results) - 1 // 排除自身
		fmt.Fprintf(&b, "■ 总计：%d 个文件可能受影响", total)
		if maxDepth > 0 && len(queue) > 0 {
			fmt.Fprintf(&b, "（BFS 深度限制 %d 层）", maxDepth)
		}
		b.WriteString("\n")

		return b.String(), nil
	}
}

// formatExportedSymbolsStatic 将静态扫描的导出符号格式化为可读文本。
func formatExportedSymbolsStatic(syms []exportedSymbol, limit int) string {
	// 按种类分组
	type group struct {
		kindName string
		symbols  []exportedSymbol
	}
	kindOrder := []string{"function", "struct", "interface", "type", "constant", "variable"}
	groupsMap := make(map[string]*group)
	for _, k := range kindOrder {
		groupsMap[k] = &group{kindName: k}
	}

	for _, s := range syms {
		if _, ok := groupsMap[s.Kind]; !ok {
			groupsMap[s.Kind] = &group{kindName: s.Kind}
		}
		groupsMap[s.Kind].symbols = append(groupsMap[s.Kind].symbols, s)
	}

	var b strings.Builder
	total := len(syms)
	limited := ""
	if total >= limit {
		limited = fmt.Sprintf("（仅显示前 %d 个）", limit)
	}
	fmt.Fprintf(&b, "项目中导出符号共 %d 个%s：\n\n", total, limited)

	for _, kind := range kindOrder {
		g := groupsMap[kind]
		if len(g.symbols) == 0 {
			continue
		}
		// 按名称排序
		sort.Slice(g.symbols, func(i, j int) bool {
			return g.symbols[i].Name < g.symbols[j].Name
		})
		fmt.Fprintf(&b, "■ %s（%d 个）:\n", g.kindName, len(g.symbols))
		for _, s := range g.symbols {
			fmt.Fprintf(&b, "  %s → %s:%d\n", s.Name, s.FilePath, s.Line)
		}
		b.WriteString("\n")
	}

	return b.String()
}
