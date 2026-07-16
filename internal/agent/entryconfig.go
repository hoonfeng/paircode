package agent

// 入口点和配置文件检测工具：find_entry_points / find_config_files。
//
// find_entry_points 扫描项目目录，找出常见的程序入口文件（main.go、main.ts、app.py 等）。
// find_config_files 扫描项目目录，找出常见的配置文件（go.mod、package.json、tsconfig.json 等）。
//
// 两者都是纯文件系统扫描，不依赖 LSP 或语言服务器。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── 已知入口文件名模式 ──

// entryFileNames 常见的入口文件名（不含路径，大小写不敏感比较使用小写）。
var entryFileNames = map[string]bool{
	// Go
	"main.go": true,
	// TypeScript / JavaScript
	"main.ts":   true,
	"main.tsx":  true,
	"main.js":   true,
	"index.ts":  true,
	"index.tsx": true,
	"index.js":  true,
	"app.ts":    true,
	"app.tsx":   true,
	"app.js":    true,
	"app.mjs":   true,
	"server.ts": true,
	"server.js": true,
	"cli.ts":    true,
	"cli.js":    true,
	// Python
	"main.py":     true,
	"app.py":      true,
	"__main__.py": true,
	"manage.py":   true,
	"wsgi.py":     true,
	"asgi.py":     true,
	// Rust
	"main.rs": true,
	"lib.rs":  true,
	// C / C++
	"main.c":   true,
	"main.cpp": true,
	"main.cc":  true,
	"main.cxx": true,
	// Java
	"Main.java": true,
	// Ruby
	"main.rb": true,
	"app.rb":  true,
	// PHP
	"index.php": true,
	// Lua
	"main.lua": true,
	// Shell
	"main.sh": true,
	// Dart
	"main.dart": true,
	// Swift
	"main.swift": true,
	// Kotlin
	"main.kt": true,
}

// entryFileSuffixes 一些没有固定入口名、但特定扩展名文件可能是入口的语言。
// 这些后缀的文件如果出现在项目顶层或 cmd/ 下，被当作入口候选。
var entryFileSuffixes = []string{
	".ex",  // Elixir
	".exs", // Elixir 脚本
	".cr",  // Crystal
	".nim", // Nim
	".rkt", // Racket
	".hs",  // Haskell
	".ml",  // OCaml
	".fs",  // F#
	".fsx", // F# 脚本
}

// ── 已知配置文件名模式 ──

// configFileNames 常见的配置文件名（小写比较）。
var configFileNames = map[string]bool{
	// 项目级
	"go.mod":              true,
	"go.sum":              true,
	"package.json":        true,
	"package-lock.json":   true,
	"yarn.lock":           true,
	"pnpm-lock.yaml":      true,
	"cargo.toml":          true,
	"cargo.lock":          true,
	"pom.xml":             true,
	"build.gradle":        true,
	"build.gradle.kts":    true,
	"settings.gradle":     true,
	"settings.gradle.kts": true,
	"gemfile":             true,
	"gemfile.lock":        true,
	"requirements.txt":    true,
	"setup.py":            true,
	"setup.cfg":           true,
	"pyproject.toml":      true,
	"poetry.lock":         true,
	"composer.json":       true,
	"composer.lock":       true,
	"mix.exs":             true,
	"rebar.config":        true,
	"project.clj":         true,
	"deps.edn":            true,
	"makefile":            true,
	"gnumakefile":         true,
	"cmakelists.txt":      true,
	"dockerfile":          true,
	"vagrantfile":         true,
	// 语言服务器 / 编译器配置
	"tsconfig.json":       true,
	"jsconfig.json":       true,
	".babelrc":            true,
	"babel.config.js":     true,
	"babel.config.cjs":    true,
	".eslintrc":           true,
	".eslintrc.js":        true,
	".eslintrc.json":      true,
	".eslintrc.yaml":      true,
	".prettierrc":         true,
	".prettierrc.json":    true,
	".prettierrc.js":      true,
	"prettier.config.js":  true,
	"stylelint.config.js": true,
	".stylelintrc.json":   true,
	"webpack.config.js":   true,
	"webpack.config.ts":   true,
	"vite.config.ts":      true,
	"vite.config.js":      true,
	"rollup.config.js":    true,
	"rollup.config.ts":    true,
	"next.config.js":      true,
	"nuxt.config.ts":      true,
	"nuxt.config.js":      true,
	// Rust
	"rust-toolchain.toml": true,
	"rust-toolchain":      true,
	".rustfmt.toml":       true,
	"clippy.toml":         true,
	// Go
	".golangci.yml": true,
	"golangci.yml":  true,
	// Python
	"tox.ini":    true,
	"pytest.ini": true,
	".pylintrc":  true,
	"mypy.ini":   true,
	".flake8":    true,
	// 编辑器 / IDE
	".editorconfig":     true,
	".gitignore":        true,
	".gitattributes":    true,
	".gitmodules":       true,
	".gitlab-ci.yml":    true,
	".github/workflows": true,
	".dockerignore":     true,
	// 环境
	".env":            true,
	".env.example":    true,
	".env.local":      true,
	".env.production": true,
	// 容器 / 编排
	"docker-compose.yml":  true,
	"docker-compose.yaml": true,
	"docker-compose.json": true,
	"kubernetes":          true, // 目录
	// 文档
	"readme.md":          true,
	"readme.txt":         true,
	"license":            true,
	"contributing.md":    true,
	"changelog.md":       true,
	"code_of_conduct.md": true,
	// 其他
	"justfile":                true,
	"taskfile.yml":            true,
	"dependabot.yml":          true,
	".npmrc":                  true,
	".yarnrc":                 true,
	".nvmrc":                  true,
	"browserslistrc":          true,
	"lerna.json":              true,
	"nx.json":                 true,
	"turbo.json":              true,
	".parcelrc":               true,
	"biome.json":              true,
	".pre-commit-config.yaml": true,
}

// configFileSuffixes 通过扩展名识别的配置文件。
var configFileSuffixes = []string{
	".yml",
	".yaml",
	".json",
	".jsonc",
	".toml",
	".ini",
	".cfg",
	".conf",
	".env",
	".properties",
}

// ── skipDirsForEntryConfig 在该工具内部扫描时跳过的目录（缩小范围，列出的目录名大小写敏感比较）。
// 注意：这些可被子目录扫描覆盖，只用于自动递归下降。
var skipDirsForEntryConfig = map[string]bool{
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
	"assets":       true,
	"fonts":        true,
	"images":       true,
	"coverage":     true,
	".cache":       true,
	"logs":         true,
	".pair":        true,
}

// ── 工具注册 ──

// registerEntryConfigTools 注册入口点和配置文件检测工具。
func registerEntryConfigTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "find_entry_points",
		Description: "列出项目中所有检测到的入口文件（main.go、index.ts、app.py 等）。" +
			"通过扫描项目目录树匹配常见入口文件名实现，支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java 等语言。" +
			"可选参数：path 限定子目录，maxResults 限制返回数量。",
		Parameters: objSchema(props{
			"path":       strProp("可选：限定扫描的子目录路径，留空则扫描整个项目"),
			"maxResults": strProp("可选：最大返回结果数（默认 50，最大 200）"),
		}),
		ReadOnly: true,
		Handler:  findEntryPointsHandler(root),
	})

	r.Register(&Tool{
		Name: "find_config_files",
		Description: "列出项目中所有检测到的配置文件（go.mod、package.json、tsconfig.json、.gitignore、Makefile、Dockerfile 等）。" +
			"通过扫描项目目录树匹配常见配置文件名和扩展名实现。" +
			"可选参数：path 限定子目录，maxResults 限制返回数量。",
		Parameters: objSchema(props{
			"path":       strProp("可选：限定扫描的子目录路径，留空则扫描整个项目"),
			"maxResults": strProp("可选：最大返回结果数（默认 50，最大 200）"),
		}),
		ReadOnly: true,
		Handler:  findConfigFilesHandler(root),
	})
}

// ── find_entry_points handler ──

func findEntryPointsHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		searchRoot := root
		if sub := argStr(args, "path"); sub != "" {
			var err error
			searchRoot, err = resolvePath(root, sub)
			if err != nil {
				return "", err
			}
		}

		limit := argInt(args, "maxResults", 50)
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}

		var results []string
		walkErr := filepath.WalkDir(searchRoot, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir // 跳过无法访问的目录
			}
			if d.IsDir() {
				base := d.Name()
				if skipDirsForEntryConfig[base] && p != searchRoot {
					return filepath.SkipDir
				}
				return nil
			}
			if len(results) >= limit {
				return filepath.SkipAll
			}

			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			name := strings.ToLower(d.Name())

			if entryFileNames[name] {
				results = append(results, rel)
				return nil
			}

			// 检查后缀匹配（仅限顶层或 cmd/、src/、bin/ 下）
			for _, suffix := range entryFileSuffixes {
				if strings.HasSuffix(name, suffix) {
					dirs := filepath.Dir(rel)
					dirBase := strings.ToLower(filepath.Base(dirs))
					if dirs == "." || dirBase == "cmd" || dirBase == "src" || dirBase == "bin" || dirBase == "app" || dirBase == "exe" {
						results = append(results, rel)
						return nil
					}
				}
			}

			// Python __main__.py 已经在 entryFileNames 中，这里补充其他情况
			return nil
		})
		if walkErr != nil && walkErr != filepath.SkipAll {
			return "", fmt.Errorf("扫描入口文件失败: %w", walkErr)
		}

		if len(results) == 0 {
			return "未检测到入口文件", nil
		}

		sort.Strings(results)
		var b strings.Builder
		fmt.Fprintf(&b, "入口文件 (%d):\n", len(results))
		for _, r := range results {
			fmt.Fprintf(&b, "  %s\n", r)
		}
		if len(results) >= limit {
			fmt.Fprintf(&b, "\n[已达到最大结果数 %d，用 maxResults 或 path 缩小范围]", limit)
		}
		return b.String(), nil
	}
}

// ── find_config_files handler ──

func findConfigFilesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		searchRoot := root
		if sub := argStr(args, "path"); sub != "" {
			var err error
			searchRoot, err = resolvePath(root, sub)
			if err != nil {
				return "", err
			}
		}

		limit := argInt(args, "maxResults", 50)
		if limit <= 0 {
			limit = 50
		}
		if limit > 200 {
			limit = 200
		}

		var results []string
		walkErr := filepath.WalkDir(searchRoot, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if d.IsDir() {
				base := d.Name()
				if skipDirsForEntryConfig[base] && p != searchRoot {
					return filepath.SkipDir
				}
				// 特殊处理 .github/workflows 目录（作为整体匹配）
				if strings.HasSuffix(p, ".github/workflows") || strings.HasSuffix(p, ".github\\workflows") {
					rel, _ := filepath.Rel(root, p)
					if rel != "" && !contains(results, rel) {
						results = append(results, rel)
					}
				}
				return nil
			}
			if len(results) >= limit {
				return filepath.SkipAll
			}

			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			name := strings.ToLower(d.Name())

			// 精确匹配已知配置文件
			if configFileNames[name] {
				results = append(results, rel)
				return nil
			}

			// 通过扩展名匹配
			ext := filepath.Ext(name)
			for _, suffix := range configFileSuffixes {
				if ext == suffix {
					results = append(results, rel)
					return nil
				}
			}

			return nil
		})
		if walkErr != nil && walkErr != filepath.SkipAll {
			return "", fmt.Errorf("扫描配置文件失败: %w", walkErr)
		}

		if len(results) == 0 {
			return "未检测到配置文件", nil
		}

		sort.Strings(results)
		var b strings.Builder
		fmt.Fprintf(&b, "配置文件 (%d):\n", len(results))
		for _, r := range results {
			fmt.Fprintf(&b, "  %s\n", r)
		}
		if len(results) >= limit {
			fmt.Fprintf(&b, "\n[已达到最大结果数 %d，用 maxResults 或 path 缩小范围]", limit)
		}
		return b.String(), nil
	}
}

// contains 检查字符串切片是否包含指定字符串。
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
