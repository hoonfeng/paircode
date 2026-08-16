package codegraph

// builder.go — 图谱构建编排器。
//
// 多语言策略：
//   - Go 文件 → GoBuilder（go/parser 直接解析，实体提取完整精确）
//   - JS/TS/... → TsitBuilder（基于 tsit Tree-sitter API 的树遍历）
// 两个构建器写入同一个 Graph 实例，由 Builder 统一管理。

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── 构建配置 ──────────────────────────────────────────

// BuildConfig 构建配置。
type BuildConfig struct {
	Root          string // 工作区根目录
	ModuleName    string // 模块名（从 go.mod 读取）
	SkipVendor    bool   // 是否跳过 vendor 目录
	WithCallGraph bool   // 是否分析调用图
	MaxParallel   int    // 最大并行解析文件数（0=串行）
	AutoSave      bool   // 构建后自动保存
}

// DefaultBuildConfig 返回默认构建配置。
func DefaultBuildConfig(root string) BuildConfig {
	return BuildConfig{
		Root:          root,
		ModuleName:    "",
		SkipVendor:    true,
		WithCallGraph: true,
		MaxParallel:   0,
		AutoSave:      true,
	}
}

// BuildResult 一次构建的结果。
type BuildResult struct {
	FilesParsed    int          // 已解析文件数
	EntitiesAdded  int          // 新增实体数
	RelationsAdded int          // 新增关系数
	Errors         []BuildError // 非致命错误
	Duration       time.Duration
}

// BuildError 构建过程中的错误。
type BuildError struct {
	File    string
	Message string
}

// ── 构建器 ────────────────────────────────────────────

// Builder 图谱构建编排器。
// 根据文件扩展名路由到合适的语言构建器。
//   - .go  → GoBuilder (go/parser, 完整)
//   - .js/.ts → JSBuilder (goja 解析器)
//   - .py  → PyBuilder (规则解析)
//   - 其他 → TsitBuilder (tsit 树遍历, 预留)
type Builder struct {
	config         BuildConfig
	store          GraphStore
	graph          *Graph
	goBuilder      *GoBuilder
	jsBuilder      *JSBuilder
	pyBuilder      *PyBuilder
	langBuilder    *LangBuilder
	importAnalyzer *ImportAnalyzer
}

// NewBuilder 创建新的构建器。
func NewBuilder(config BuildConfig) *Builder {
	graph := NewGraph()
	return &Builder{
		config: config,
		store:  NewStore(config.Root),
		graph:  graph,
		goBuilder: &GoBuilder{
			ModuleName: config.ModuleName,
			root:       config.Root,
			graph:      graph,
		},
		jsBuilder: &JSBuilder{
			ModuleName: config.ModuleName,
			root:       config.Root,
			graph:      graph,
		},
		pyBuilder: &PyBuilder{
			ModuleName: config.ModuleName,
			root:       config.Root,
			graph:      graph,
		},
		langBuilder: &LangBuilder{
			ModuleName: config.ModuleName,
			root:       config.Root,
			graph:      graph,
		},
		importAnalyzer: NewImportAnalyzer(config.Root, config.ModuleName),
	}
}

// SetStore 设置自定义图谱存储（默认 JSONStore，可切换为 SQLiteStore）。
func (b *Builder) SetStore(s GraphStore) { b.store = s }

// Graph 返回当前构建的图实例。
func (b *Builder) Graph() *Graph { return b.graph }

// ReadModuleName 从 go.mod 读取模块名。
func ReadModuleName(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("读取 go.mod 失败: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("go.mod 中未找到 module 声明")
}

// DetectModuleName 尝试自动检测模块名。
func DetectModuleName(root string) string {
	if name, err := ReadModuleName(root); err == nil && name != "" {
		return name
	}
	// 尝试 go.work
	if data, err := os.ReadFile(filepath.Join(root, "go.work")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "use ") {
				// 可以进一步解析
			}
		}
	}
	return "unknown"
}

// ── 语言路由 ──────────────────────────────────────────

// isGoFile 检查文件是否为 Go 源文件。
func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// isSupportedFile 检查文件是否支持。
func isSupportedFile(path string) bool {
	if isGoFile(path) || isJSFile(path) || isPyFile(path) {
		return true
	}
	name := filepath.Base(path)
	return isLangSupportedExtra(name)
}

// parseFile 根据语言路由解析单个文件。
//   - .go       → GoBuilder（go/parser, 完整实体提取）
//   - .js/.ts   → JSBuilder（goja 解析器）
//   - .py       → PyBuilder（规则解析）
//   - .rs/.java/.c/.cpp/.cs → LangBuilder（正则解析）
//   - .rb/.php/.swift/.kt/.dart/.lua/.sh/.sql/.vue  → LangBuilder（正则解析）
func (b *Builder) parseFile(filePath string) (string, error) {
	switch {
	case isGoFile(filePath):
		b.goBuilder.ModuleName = b.config.ModuleName
		return b.goBuilder.ParseFile(filePath)
	case isJSFile(filePath):
		b.jsBuilder.ModuleName = b.config.ModuleName
		return b.jsBuilder.ParseFile(filePath)
	case isPyFile(filePath):
		b.pyBuilder.ModuleName = b.config.ModuleName
		return b.pyBuilder.ParseFile(filePath)
	default:
		b.langBuilder.ModuleName = b.config.ModuleName
		return b.langBuilder.ParseFile(filePath)
	}
}

// parseFileInto 将文件解析结果写入指定图（用于增量构建）。
func (b *Builder) parseFileInto(filePath string, targetGraph *Graph) (string, error) {
	switch {
	case isGoFile(filePath):
		gb := &GoBuilder{ModuleName: b.config.ModuleName, root: b.config.Root, graph: targetGraph}
		return gb.ParseFile(filePath)
	case isJSFile(filePath):
		jb := &JSBuilder{ModuleName: b.config.ModuleName, root: b.config.Root, graph: targetGraph}
		return jb.ParseFile(filePath)
	case isPyFile(filePath):
		pb := &PyBuilder{ModuleName: b.config.ModuleName, root: b.config.Root, graph: targetGraph}
		return pb.ParseFile(filePath)
	default:
		lb := &LangBuilder{ModuleName: b.config.ModuleName, root: b.config.Root, graph: targetGraph}
		return lb.ParseFile(filePath)
	}
}

// ── 构建执行 ──────────────────────────────────────────

// BuildFull 执行完整构建。
//  1. 扫描工作区所有支持的文件
//  2. Go 文件 → GoBuilder 完整实体提取
//  3. 非 Go 文件 → TsitBuilder 树遍历
//  4. 导入分析 → 包级依赖图
func (b *Builder) BuildFull() (*BuildResult, error) {
	start := time.Now()

	// 自动检测模块名
	if b.config.ModuleName == "" {
		b.config.ModuleName = DetectModuleName(b.config.Root)
	}
	b.langBuilder.ModuleName = b.config.ModuleName
	b.goBuilder.ModuleName = b.config.ModuleName
	b.jsBuilder.ModuleName = b.config.ModuleName
	b.pyBuilder.ModuleName = b.config.ModuleName
	b.importAnalyzer.ModuleName = b.config.ModuleName

	result := &BuildResult{}
	filesParsed := 0
	var allErrors []BuildError

	// 扫描并解析所有支持的文件
	srcDir := b.config.Root
	filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "vendor" ||
				info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSupportedFile(info.Name()) {
			return nil
		}

		relPath, _ := filepath.Rel(b.config.Root, path)
		relPath = filepath.ToSlash(relPath)

		if _, err := b.parseFile(relPath); err != nil {
			allErrors = append(allErrors, BuildError{
				File:    relPath,
				Message: err.Error(),
			})
			return nil
		}
		filesParsed++
		return nil
	})

	result.FilesParsed = filesParsed
	result.EntitiesAdded = b.graph.Stats().EntityCount
	result.EntitiesAdded = max(result.EntitiesAdded, len(b.graph.entities))
	result.RelationsAdded = b.graph.Stats().RelationCount
	result.Errors = allErrors

	// 导入分析：构建包级依赖图
	if impCount, err := b.importAnalyzer.BuildImportGraph(b.graph); err != nil {
		result.Errors = append(result.Errors, BuildError{
			Message: fmt.Sprintf("导入分析失败: %v", err),
		})
	} else {
		result.RelationsAdded += impCount
	}

	result.Duration = time.Since(start)

	// 自动保存
	if b.config.AutoSave {
		if err := b.store.Save(b.graph); err != nil {
			result.Errors = append(result.Errors, BuildError{
				Message: fmt.Sprintf("保存图谱失败: %v", err),
			})
		}
	}

	return result, nil
}

// ── 增量构建 ──────────────────────────────────────────

// IncrementalBuild 增量构建：只重新解析已变更的文件。
func (b *Builder) IncrementalBuild() (*BuildResult, error) {
	start := time.Now()
	result := &BuildResult{}

	// 加载已有图谱
	graph, err := b.store.Load()
	if err != nil {
		return b.BuildFull()
	}
	// 用已有图替换当前图
	b.graph = graph
	b.goBuilder.graph = graph
	b.jsBuilder.graph = graph
	b.pyBuilder.graph = graph
	b.langBuilder.graph = graph

	// 加载文件索引
	index, err := b.store.LoadIndex()
	if err != nil {
		return b.BuildFull()
	}

	// 扫描文件变化
	changedFiles, err := b.detectChangedFiles(index)
	if err != nil {
		return b.BuildFull()
	}

	if len(changedFiles) == 0 {
		result.Duration = time.Since(start)
		return result, nil
	}

	// 对每个变更文件：移除旧实体 → 重新解析
	for _, filePath := range changedFiles {
		graph.RemoveFileEntities(filePath)

		if _, err := b.parseFileInto(filePath, graph); err != nil {
			result.Errors = append(result.Errors, BuildError{
				File:    filePath,
				Message: err.Error(),
			})
			continue
		}
		result.FilesParsed++

		// 更新索引
		absPath := filepath.Join(b.config.Root, filePath)
		if fi, err := os.Stat(absPath); err == nil {
			index[filePath] = fi.ModTime()
		}
	}

	result.EntitiesAdded = graph.Stats().EntityCount
	result.RelationsAdded = graph.Stats().RelationCount

	// 保存图谱（增量保存：只写变更文件）和索引
	if b.config.AutoSave {
		if err := b.store.SaveIncremental(graph, changedFiles); err != nil {
			result.Errors = append(result.Errors, BuildError{
				Message: fmt.Sprintf("保存图谱失败: %v", err),
			})
		}
		if err := b.store.SaveIndex(index); err != nil {
			result.Errors = append(result.Errors, BuildError{
				Message: fmt.Sprintf("保存索引失败: %v", err),
			})
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// detectChangedFiles 检测自上次构建以来变更的文件。
func (b *Builder) detectChangedFiles(index map[string]time.Time) ([]string, error) {
	var changed []string

	err := filepath.Walk(b.config.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "vendor" ||
				info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSupportedFile(info.Name()) {
			return nil
		}

		relPath, _ := filepath.Rel(b.config.Root, path)
		relPath = filepath.ToSlash(relPath)

		lastMod, exists := index[relPath]
		if !exists || info.ModTime().After(lastMod) {
			changed = append(changed, relPath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(changed)
	return changed, nil
}

// ── 工具 ──────────────────────────────────────────────

// extractFilePath 从错误消息中提取文件路径。
func extractFilePath(msg string) string {
	if idx := strings.Index(msg, ": "); idx > 0 {
		candidate := msg[:idx]
		if strings.Contains(candidate, "/") || strings.Contains(candidate, ".go") {
			return candidate
		}
	}
	return ""
}

// LoadGraph 从存储加载图谱。
func LoadGraph(root string) (*Graph, error) {
	store := NewStore(root)
	return store.Load()
}

// SaveGraph 保存图谱到存储。
func SaveGraph(root string, g *Graph) error {
	store := NewStore(root)
	return store.Save(g)
}

// max 返回较大值。
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
