package agent

// project_overview 工具：递归扫描项目目录，返回 JSON 格式的项目概览。
//
// project_overview 统计文件总数、总大小、语言分布（按扩展名归类）、
// 最大文件列表和代码行数，以 JSON 格式输出。
//
// 自动排除 .git、node_modules、vendor 等常见非源码目录。
// 行数统计通过读取文件计算换行符数量实现（不加载全文到内存）。

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── 跳过目录 ──

// skipDirsForProjectOverview 在递归扫描时跳过的目录（与 projectindex.go / entryconfig.go 保持一致）。
var skipDirsForProjectOverview = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"out":          true,
	".idea":        true,
	".vscode":      true,
	"third_party":  true,
	"thirdparty":   true,
	"coverage":     true,
	".cache":       true,
	"logs":         true,
	".pair":        true,
}

// ── 语言扩展名映射 ──

// langExtMap 扩展名 → 语言名称（小写扩展名匹配）。
// 通用扩展名（如 .json）排在具体语言之后，按优先级顺序匹配。
var langExtMap = []struct {
	ext  string
	lang string
}{
	{".go", "Go"},
	{".tsx", "TypeScript"},
	{".ts", "TypeScript"},
	{".jsx", "JavaScript"},
	{".mjs", "JavaScript"},
	{".cjs", "JavaScript"},
	{".js", "JavaScript"},
	{".py", "Python"},
	{".rs", "Rust"},
	{".java", "Java"},
	{".kt", "Kotlin"},
	{".kts", "Kotlin"},
	{".swift", "Swift"},
	{".dart", "Dart"},
	{".rb", "Ruby"},
	{".php", "PHP"},
	{".lua", "Lua"},
	{".ex", "Elixir"},
	{".exs", "Elixir"},
	{".hs", "Haskell"},
	{".ml", "OCaml"},
	{".clj", "Clojure"},
	{".cljs", "ClojureScript"},
	{".scala", "Scala"},
	{".c", "C"},
	{".h", "C"},
	{".cpp", "C++"},
	{".cc", "C++"},
	{".cxx", "C++"},
	{".hpp", "C++"},
	{".hxx", "C++"},
	{".cs", "C#"},
	{".fs", "F#"},
	{".fsx", "F#"},
	{".r", "R"},
	{".m", "MATLAB"},
	{".mm", "Objective-C"},
	{".zig", "Zig"},
	{".nim", "Nim"},
	{".cr", "Crystal"},
	{".erl", "Erlang"},
	{".hrl", "Erlang"},
	{".md", "Markdown"},
	{".mdx", "Markdown"},
	{".json", "JSON"},
	{".jsonc", "JSON"},
	{".yaml", "YAML"},
	{".yml", "YAML"},
	{".toml", "TOML"},
	{".xml", "XML"},
	{".html", "HTML"},
	{".htm", "HTML"},
	{".css", "CSS"},
	{".scss", "SCSS"},
	{".less", "Less"},
	{".sass", "Sass"},
	{".sql", "SQL"},
	{".sh", "Shell"},
	{".bash", "Shell"},
	{".zsh", "Shell"},
	{".fish", "Shell"},
	{".bat", "Batch"},
	{".cmd", "Batch"},
	{".ps1", "PowerShell"},
	{".psm1", "PowerShell"},
	{".dockerfile", "Docker"},
	{".makefile", "Makefile"},
	{".cmake", "CMake"},
	{".proto", "Protobuf"},
	{".graphql", "GraphQL"},
	{".gql", "GraphQL"},
	{".svg", "SVG"},
	{".png", "Image"},
	{".jpg", "Image"},
	{".jpeg", "Image"},
	{".gif", "Image"},
	{".ico", "Image"},
	{".bmp", "Image"},
	{".webp", "Image"},
	{".ttf", "Font"},
	{".otf", "Font"},
	{".woff", "Font"},
	{".woff2", "Font"},
	{".eot", "Font"},
	{".dll", "DLL"},
	{".exe", "Executable"},
	{".so", "Dynamic Library"},
	{".dylib", "Dynamic Library"},
	{".a", "Static Library"},
	{".lib", "Static Library"},
	{".mod", "Go Module"},
	{".sum", "Go Module"},
	{".ndjson", "NDJSON"},
	{".def", "Module Definition"},
	{".pdf", "PDF"},
	{".zip", "Archive"},
	{".tar", "Archive"},
	{".gz", "Archive"},
	{".bz2", "Archive"},
	{".xz", "Archive"},
	{".7z", "Archive"},
	{".rar", "Archive"},
}

// ── 可统计行数的扩展名（文本/代码文件） ──

// countableExts 可安全按文本统计行数的扩展名集合。
var countableExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".mjs": true, ".cjs": true, ".py": true, ".rs": true, ".java": true,
	".kt": true, ".kts": true, ".swift": true, ".dart": true, ".rb": true,
	".php": true, ".lua": true, ".ex": true, ".exs": true, ".hs": true,
	".ml": true, ".clj": true, ".cljs": true, ".scala": true, ".c": true,
	".h": true, ".cpp": true, ".cc": true, ".cxx": true, ".hpp": true,
	".hxx": true, ".cs": true, ".fs": true, ".fsx": true, ".r": true,
	".m": true, ".mm": true, ".zig": true, ".nim": true, ".cr": true,
	".erl": true, ".hrl": true, ".md": true, ".mdx": true, ".json": true,
	".jsonc": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true,
	".html": true, ".htm": true, ".css": true, ".scss": true, ".less": true,
	".sass": true, ".sql": true, ".sh": true, ".bash": true, ".zsh": true,
	".fish": true, ".bat": true, ".cmd": true, ".ps1": true, ".psm1": true,
	".mod": true, ".sum": true, ".ndjson": true, ".def": true, ".env": true,
	".conf": true, ".cfg": true, ".ini": true, ".properties": true,
	".gradle": true, ".sbt": true,
	".tex": true, ".rst": true, ".txt": true, ".log": true,
	".dockerfile": true, ".makefile": true, ".cmake": true, ".proto": true,
	".graphql": true, ".gql": true,
}

// ── 输出类型 ──

// langStat 一种语言的统计信息。
type langStat struct {
	Files      int    `json:"files"`
	Size       int64  `json:"size"`
	SizeHuman  string `json:"sizeHuman"`
	Lines      int    `json:"lines,omitempty"`
	LinesHuman string `json:"linesHuman,omitempty"`
}

// fileInfo 文件信息（用于最大文件列表）。
type fileInfo struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	SizeHuman  string `json:"sizeHuman"`
	Lines      int    `json:"lines,omitempty"`
	LinesHuman string `json:"linesHuman,omitempty"`
}

// projectOverviewOutput 整体输出结构。
type projectOverviewOutput struct {
	FileCount            int                 `json:"fileCount"`
	TotalSize            int64               `json:"totalSize"`
	TotalSizeHuman       string              `json:"totalSizeHuman"`
	LanguageDistribution map[string]langStat `json:"languageDistribution"`
	LargestFiles         []fileInfo          `json:"largestFiles"`
	TotalLines           int                 `json:"totalLines"`
	TotalLinesHuman      string              `json:"totalLinesHuman"`
	LargestFilesCount    int                 `json:"largestFilesCount"`
}

// ── 辅助函数 ──

// humanSize 将字节数转换为人类可读格式。
func humanSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(1<<30))
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// humanCount 将计数值转换为人类可读格式（千分位分隔）。
func humanCount(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%.1fK", float64(n)/1000)
}

// detectLanguage 根据文件扩展名返回语言名称（小写比较）。
func detectLanguage(name string) string {
	lower := strings.ToLower(name)
	for _, m := range langExtMap {
		if strings.HasSuffix(lower, m.ext) {
			return m.lang
		}
	}
	// 特殊文件名匹配
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "dockerfile":
		return "Docker"
	case "makefile", "gnumakefile":
		return "Makefile"
	case "gemfile":
		return "Ruby"
	}
	return "Unknown"
}

// countLines 统计文件行数（通过 bufio.Scanner 逐行读取，不加载全文到内存）。
// 对大文件（>50MB）跳过行数统计，返回 -1 表示跳过。
func countLines(path string, maxSize int64) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.Size() > maxSize {
		return -1, nil // 文件过大，跳过行数统计
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 64KB 缓冲区，最大 token 1MB
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("扫描 %s: %w", path, err)
	}
	return lines, nil
}

// ── 扫描逻辑 ──

// scanProjectOverview 递归扫描目录，收集统计信息。
// 参数：
//
//	root     — 扫描的根目录
//	current  — 当前要扫描的目录
//	countLns — 是否统计行数
//
// 返回：(文件数, 总大小, 语言统计, 最大文件列表, 总行数, 错误)
func scanProjectOverview(root, current string, countLns bool) (
	fileCount int,
	totalSize int64,
	langMap map[string]*langStat,
	largest []fileInfo,
	totalLines int,
	err error,
) {
	langMap = make(map[string]*langStat)

	walkErr := filepath.WalkDir(current, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if d.IsDir() {
			base := d.Name()
			if skipDirsForProjectOverview[base] && p != current {
				return filepath.SkipDir
			}
			// 跳过以 . 开头的隐藏目录（保留有意义目录）
			if strings.HasPrefix(base, ".") && base != "." && base != ".." {
				meaningful := map[string]bool{
					".github": true, ".vscode": true, ".idea": true,
					".gitlab": true, ".husky": true,
				}
				if !meaningful[base] {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// 跳过隐藏文件
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil // 跳过无法获取信息的文件
		}

		size := info.Size()
		fileCount++
		totalSize += size

		lang := detectLanguage(d.Name())
		ls, ok := langMap[lang]
		if !ok {
			ls = &langStat{}
			langMap[lang] = ls
		}
		ls.Files++
		ls.Size += size

		// 统计行数
		if countLns {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if countableExts[ext] || lang != "Unknown" {
				lines, lineErr := countLines(p, 50*1024*1024) // 最大 50MB
				if lineErr == nil && lines >= 0 {
					ls.Lines += lines
					totalLines += lines
				}
			}
		}

		// 记录最大文件（保留前 10 个）
		largest = append(largest, fileInfo{
			Path: p,
			Size: size,
		})

		return nil
	})

	if walkErr != nil && walkErr != filepath.SkipDir {
		return 0, 0, nil, nil, 0, fmt.Errorf("扫描目录失败: %w", walkErr)
	}

	return fileCount, totalSize, langMap, largest, totalLines, nil
}

// ── 工具处理函数 ──

// projectOverviewHandler 创建 project_overview 工具的处理函数。
func projectOverviewHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		searchRoot := root
		if sub := argStr(args, "path"); sub != "" {
			var err error
			searchRoot, err = resolvePath(root, sub)
			if err != nil {
				return "", err
			}
		}

		countLns := true
		if v := args["countLines"]; v != nil {
			if b, ok := v.(bool); ok {
				countLns = b
			}
		}

		maxFiles := argInt(args, "maxFiles", 10)
		if maxFiles <= 0 {
			maxFiles = 10
		}
		if maxFiles > 100 {
			maxFiles = 100
		}

		fileCount, totalSize, langMap, largestRaw, totalLines, err := scanProjectOverview(root, searchRoot, countLns)
		if err != nil {
			return "", err
		}

		if fileCount == 0 {
			return `{"fileCount":0,"totalSize":0,"totalSizeHuman":"0 B","languageDistribution":{},"largestFiles":[],"totalLines":0,"totalLinesHuman":"0","largestFilesCount":0}`, nil
		}

		// 排序最大文件（按大小降序）
		sort.SliceStable(largestRaw, func(i, j int) bool {
			return largestRaw[i].Size > largestRaw[j].Size
		})
		if len(largestRaw) > maxFiles {
			largestRaw = largestRaw[:maxFiles]
		}

		// 转换最大文件列表（补充 SizeHuman、Lines、LinesHuman）
		largestFiles := make([]fileInfo, len(largestRaw))
		for i, fi := range largestRaw {
			largestFiles[i] = fileInfo{
				Path:      filepath.ToSlash(fi.Path), // 统一使用 / 分隔符
				Size:      fi.Size,
				SizeHuman: humanSize(fi.Size),
			}
			// 如果是代码文件且统计了行数，补充行数信息
			if countLns {
				ext := strings.ToLower(filepath.Ext(fi.Path))
				if countableExts[ext] {
					lines, lineErr := countLines(fi.Path, 50*1024*1024)
					if lineErr == nil && lines >= 0 {
						largestFiles[i].Lines = lines
						largestFiles[i].LinesHuman = humanCount(lines)
					}
				}
			}
		}

		// 构建语言分布（排序后转为 map，确保 key 稳定）
		langDist := make(map[string]langStat, len(langMap))
		for _, lang := range sortedLangKeys(langMap) {
			ls := *langMap[lang]
			if !countLns {
				ls.Lines = 0
				ls.LinesHuman = ""
			} else {
				ls.LinesHuman = humanCount(ls.Lines)
			}
			ls.SizeHuman = humanSize(ls.Size)
			langDist[lang] = ls
		}

		// 生成相对路径
		for i := range largestFiles {
			rel, err := filepath.Rel(root, largestFiles[i].Path)
			if err == nil {
				largestFiles[i].Path = filepath.ToSlash(rel)
			}
		}

		output := projectOverviewOutput{
			FileCount:            fileCount,
			TotalSize:            totalSize,
			TotalSizeHuman:       humanSize(totalSize),
			LanguageDistribution: langDist,
			LargestFiles:         largestFiles,
			TotalLines:           totalLines,
			TotalLinesHuman:      humanCount(totalLines),
			LargestFilesCount:    maxFiles,
		}

		out, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return "", fmt.Errorf("JSON 序列化失败: %w", err)
		}

		return string(out), nil
	}
}

// sortedLangKeys 返回语言映射的排序后的 key 列表（按文件数降序）。
func sortedLangKeys(m map[string]*langStat) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		// 按文件数降序
		if m[keys[i]].Files != m[keys[j]].Files {
			return m[keys[i]].Files > m[keys[j]].Files
		}
		// 文件数相同按名称升序
		return keys[i] < keys[j]
	})
	return keys
}

// ── 工具注册 ──

// registerProjectOverviewTool 注册 project_overview 工具。
func registerProjectOverviewTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "project_overview",
		Description: "获取全面的项目概览（JSON 格式），" +
			"包含：fileCount（文件总数）、totalSize（总字节数）、totalSizeHuman（人类可读大小）、" +
			"languageDistribution（按语言分类的文件数/大小/行数）、" +
			"largestFiles（最大文件列表，含路径/大小/行数）、" +
			"totalLines（总行数）、totalLinesHuman（人类可读行数）。" +
			"自动排除 .git、node_modules、vendor 等常见非源码目录。" +
			"可选参数：path 限定子目录，countLines 是否统计行数（默认 true），maxFiles 最大文件列表长度（默认 10）。",
		Parameters: objSchema(props{
			"path":       strProp("可选：限定扫描的子目录路径，留空则扫描整个项目"),
			"countLines": boolProp("可选：是否统计代码行数（默认 true），关闭可加快扫描速度"),
			"maxFiles":   intProp("可选：最大文件列表的长度（默认 10，最大 100）"),
		}),
		ReadOnly: true,
		Handler:  projectOverviewHandler(root),
	})
}
