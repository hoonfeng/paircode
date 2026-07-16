package agent

// find_files_by_pattern 工具：按 glob 模式匹配查找文件，附带基础文件信息（大小、行数估计等）。
// 自动跳过 .git/node_modules/vendor 等非源码目录，支持递归搜索。
// 轻量级实现，纯文件系统扫描 + 换行符统计，无需语言服务器。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ── 工具注册 ──

// registerFindFilesByPatternTool 注册 find_files_by_pattern 工具。
func registerFindFilesByPatternTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "find_files_by_pattern",
		Description: "按 glob 模式匹配查找文件（如 \"*.go\"、\"**/*.test.ts\"、\"src/**/auth*\"），" +
			"附带每个文件的语言、大小和行数预估。自动跳过非源码目录（.git/node_modules/vendor 等）。" +
			"本工具只读、免审批、限定工作区内。",
		Parameters: objSchema(props{
			"pattern":    strProp("Glob 模式，如 \"*.go\"、\"**/*.test.ts\"、\"src/**/auth*\""),
			"language":   strProp("可选：按语言过滤，如 \"go\"、\"typescript\"、\"python\""),
			"maxResults": intProp("可选：最大返回结果数（默认 50，最大 200）"),
		}, "pattern"),
		ReadOnly: true,
		Handler:  findFilesByPatternHandler(root),
	})
}

// ── 处理函数 ──

type fileMatch struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Size     int64  `json:"size"`
	Lines    int    `json:"lines"`
}

// extLangMap 常见文件扩展名→语言名映射。
var extLangMap = map[string]string{
	".go": "go", ".ts": "typescript", ".tsx": "typescript",
	".js": "javascript", ".jsx": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".py": "python", ".rs": "rust", ".java": "java", ".kt": "kotlin",
	".swift": "swift", ".c": "c", ".h": "c", ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".hpp": "cpp",
	".cs": "csharp", ".rb": "ruby", ".php": "php", ".lua": "lua",
	".sh": "shell", ".bash": "shell", ".zsh": "shell", ".ps1": "powershell",
	".yaml": "yaml", ".yml": "yaml", ".json": "json", ".toml": "toml",
	".xml": "xml", ".html": "html", ".css": "css", ".scss": "scss", ".less": "less",
	".sql": "sql", ".md": "markdown",
	".dart": "dart", ".ex": "elixir", ".exs": "elixir",
	".erl": "erlang", ".hs": "haskell", ".scala": "scala",
	".zig": "zig", ".svelte": "svelte", ".vue": "vue",
}

// skipDirsForFindFiles 递归扫描时跳过的目录。
var skipDirsForFindFiles = map[string]bool{
	".git": true, ".svn": true, ".hg": true, ".idea": true, ".vscode": true,
	"node_modules": true, "vendor": true, "__pycache__": true, ".venv": true, "venv": true,
	"dist": true, "build": true, "out": true, "target": true, ".next": true, ".nuxt": true,
	".cache": true, ".pair": true, "coverage": true, ".terraform": true,
}

func findFilesByPatternHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		pattern := strings.TrimSpace(argStr(args, "pattern"))
		if pattern == "" {
			return "", fmt.Errorf("pattern 不能为空")
		}
		langFilter := strings.TrimSpace(argStr(args, "language"))
		maxResults := argInt(args, "maxResults", 50)
		if maxResults <= 0 || maxResults > 200 {
			maxResults = 200
		}

		var results []fileMatch
		var mu sync.Mutex

		// 递归扫描目录
		filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if skipDirsForFindFiles[info.Name()] && p != root {
					return filepath.SkipDir
				}
				return nil
			}
			// 匹配 glob 模式（支持 ** 递归）— 先匹配文件名，再匹配相对路径
			rel, _ := filepath.Rel(root, p)
			relSlash := filepath.ToSlash(rel)
			matched := matchGlobFilter(pattern, info.Name(), relSlash)
			if !matched {
				return nil
			}

			// 语言过滤
			ext := strings.ToLower(filepath.Ext(p))
			lang := extLangMap[ext]
			if lang == "" {
				lang = "unknown"
			}
			if langFilter != "" && !strings.EqualFold(lang, langFilter) {
				return nil
			}

			relPath, _ := filepath.Rel(root, p)
			lines := estimateLines(p, 100000)

			mu.Lock()
			results = append(results, fileMatch{
				Path:     filepath.ToSlash(relPath),
				Language: lang,
				Size:     info.Size(),
				Lines:    lines,
			})
			mu.Unlock()
			return nil
		})

		if len(results) == 0 {
			return "（未找到匹配的文件）", nil
		}

		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Path < results[j].Path
		})

		total := len(results)
		if len(results) > maxResults {
			results = results[:maxResults]
		}

		var b strings.Builder
		fmt.Fprintf(&b, "找到 %d 个匹配文件（显示前 %d）：\n\n", total, maxResults)
		for _, f := range results {
			fmt.Fprintf(&b, "%-60s  %-12s  %8d 字节  %5d 行\n", f.Path, f.Language, f.Size, f.Lines)
		}
		if total > maxResults {
			fmt.Fprintf(&b, "\n...还有 %d 个文件未显示。用 maxResults 参数增加返回数量（最大 200）。\n", total-maxResults)
		}
		return b.String(), nil
	}
}

// estimateLines 估算文件行数（仅读前 maxBytes 字节中的换行符）。
func estimateLines(p string, maxBytes int) int {
	f, err := os.Open(p)
	if err != nil {
		return 0
	}
	defer f.Close()
	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	if n <= 0 {
		return 0
	}
	lines := 0
	for i := 0; i < n; i++ {
		if buf[i] == '\n' {
			lines++
		}
	}
	return lines
}
