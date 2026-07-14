package codegraph

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── 构建编排器 ────────────────────────────────────────

// BuildConfig 构建配置。
type BuildConfig struct {
	// 工作区根目录
	Root string
	// 模块名（从 go.mod 读取）
	ModuleName string
	// 是否跳过 vendor 目录
	SkipVendor bool
	// 是否分析调用图（耗时较长）
	WithCallGraph bool
	// 最大并行解析文件数（0=串行）
	MaxParallel int
	// 构建后自动保存
	AutoSave bool
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
	FilesParsed  int            // 已解析文件数
	EntitiesAdded int           // 新增实体数
	RelationsAdded int          // 新增关系数
	Errors       []BuildError   // 非致命错误
	Duration     time.Duration  // 耗时
}

// BuildError 构建过程中的错误。
type BuildError struct {
	File    string
	Message string
}

// Builder 图谱构建编排器。
// 整合 Tree-sitter (tsit) AST 解析（多语言）、导入分析、调用图构建。
type Builder struct {
	config        BuildConfig
	store         *Store
	tsitBuilder   *TsitBuilder
	goBuilder     *GoBuilder
	importAnalyzer *ImportAnalyzer
}

// NewBuilder 创建新的构建器。
func NewBuilder(config BuildConfig) *Builder {
	return &Builder{
		config: config,
		store:  NewStore(config.Root),
		tsitBuilder: NewTsitBuilder(config.Root, config.ModuleName),
		goBuilder: NewGoBuilder(config.Root, config.ModuleName),
		importAnalyzer: NewImportAnalyzer(config.Root, config.ModuleName),
	}
}

// ReadModuleName 从 go.mod 文件中读取模块名。
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

// Graph 返回当前构建的图实例。
func (b *Builder) Graph() *Graph {
	if b.tsitBuilder != nil && b.tsitBuilder.Graph().Stats().EntityCount > 0 {
		return b.tsitBuilder.Graph()
	}
	if b.goBuilder != nil {
		return b.goBuilder.Graph()
	}
	return NewGraph()
}

// DetectModuleName 尝试自动检测模块名（从 go.mod 或 go.work）。
func DetectModuleName(root string) string {
	// 尝试 go.mod
	if name, err := ReadModuleName(root); err == nil && name != "" {
		return name
	}
	// 尝试 go.work
	if data, err := os.ReadFile(filepath.Join(root, "go.work")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				continue
			}
			if strings.Contains(line, "//") {
				continue
			}
		}
	}
	return "unknown"
}

// ── 构建执行 ──────────────────────────────────────────

// BuildFull 执行完整构建：Tree-sitter AST 解析 → 导入分析 → 调用图构建。
// 使用 tsit 的多语言解析器（Go/JS/TS 等）。
func (b *Builder) BuildFull() (*BuildResult, error) {
	start := time.Now()

	// 自动检测模块名
	if b.config.ModuleName == "" {
		b.config.ModuleName = DetectModuleName(b.config.Root)
		b.tsitBuilder.ModuleName = b.config.ModuleName
		b.importAnalyzer.ModuleName = b.config.ModuleName
	}

	result := &BuildResult{}

	// 1. Tree-sitter AST 解析（多语言）
	filesParsed, astErrors := b.tsitBuilder.ParseDir(".")
	result.FilesParsed = filesParsed
	for _, err := range astErrors {
		result.Errors = append(result.Errors, BuildError{
			File:    extractFilePath(err.Error()),
			Message: err.Error(),
		})
	}

	graph := b.tsitBuilder.Graph()
	result.EntitiesAdded = graph.Stats().EntityCount
	result.RelationsAdded = graph.Stats().RelationCount

	// 2. 导入分析：构建包级依赖图
	if impCount, err := b.importAnalyzer.BuildImportGraph(graph); err != nil {
		result.Errors = append(result.Errors, BuildError{
			Message: fmt.Sprintf("导入分析失败: %v", err),
		})
	} else {
		result.RelationsAdded += impCount
	}

	// 3. 调用图构建（可选）
	if b.config.WithCallGraph {
		// 调用图已在 ParseFuncDecl 中通过 parseCallExprs 部分构建
		// 后续可添加跨文件调用解析
	}

	result.Duration = time.Since(start)

	// 4. 自动保存
	if b.config.AutoSave {
		if err := b.store.Save(graph); err != nil {
			result.Errors = append(result.Errors, BuildError{
				Message: fmt.Sprintf("保存图谱失败: %v", err),
			})
		}
	}

	return result, nil
}

// ── 增量构建 ──────────────────────────────────────────

// IncrementalBuild 增量构建：只重新解析已变更的文件。
// 需要文件名→最后修改时间的索引。
func (b *Builder) IncrementalBuild() (*BuildResult, error) {
	start := time.Now()
	result := &BuildResult{}

	// 1. 加载已有图谱
	graph, err := b.store.Load()
	if err != nil {
		// 加载失败，回退到全量构建
		return b.BuildFull()
	}

	// 2. 加载文件索引
	index, err := b.store.LoadIndex()
	if err != nil {
		return b.BuildFull()
	}

	// 3. 扫描文件变化
	changedFiles, err := b.detectChangedFiles(index)
	if err != nil {
		return b.BuildFull()
	}

	if len(changedFiles) == 0 {
		result.FilesParsed = 0
		result.EntitiesAdded = 0
		result.RelationsAdded = 0
		result.Duration = time.Since(start)
		return result, nil
	}

	// 4. 对每个变更文件：移除旧实体 → 重新解析
	b.goBuilder = NewGoBuilder(b.config.Root, b.config.ModuleName)
	// 使用已有图（不新建）
	b.goBuilder.Reset()
	// 把已有图作为基础
	b.goBuilder.graph = graph

	for _, filePath := range changedFiles {
		// 移除旧数据
		graph.RemoveFileEntities(filePath)

		// 重新解析
		if _, err := b.goBuilder.ParseFile(filePath); err != nil {
			result.Errors = append(result.Errors, BuildError{
				File:    filePath,
				Message: err.Error(),
			})
			continue
		}
		result.FilesParsed++

		// 更新索引
		if fi, err := os.Stat(filepath.Join(b.config.Root, filePath)); err == nil {
			index[filePath] = fi.ModTime()
		}
	}

	result.EntitiesAdded = graph.Stats().EntityCount
	result.RelationsAdded = graph.Stats().RelationCount

	// 5. 保存图谱和索引
	if b.config.AutoSave {
		if err := b.store.Save(graph); err != nil {
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
		if !strings.HasSuffix(info.Name(), ".go") {
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
	// 格式如 "pkg/foo.go: some error" 或 "解析文件 pkg/foo.go 失败: ..."
	if idx := strings.Index(msg, ": "); idx > 0 {
		// 检查冒号前是否是文件路径
		candidate := msg[:idx]
		if strings.Contains(candidate, "/") || strings.Contains(candidate, ".go") {
			return candidate
		}
	}
	return ""
}

// LoadGraph 从存储加载图谱。如不存在则返回空图。
func LoadGraph(root string) (*Graph, error) {
	store := NewStore(root)
	return store.Load()
}

// SaveGraph 保存图谱到存储。
func SaveGraph(root string, g *Graph) error {
	store := NewStore(root)
	return store.Save(g)
}
