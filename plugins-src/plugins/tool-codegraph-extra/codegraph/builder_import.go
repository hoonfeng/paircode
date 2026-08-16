package codegraph

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ── 导入分析 ──────────────────────────────────────────

// ImportAnalyzer 从 Go 源文件中提取导入依赖关系，构建模块级依赖图。
type ImportAnalyzer struct {
	ModuleName string
	root       string
}

// NewImportAnalyzer 创建导入分析器。
func NewImportAnalyzer(root, moduleName string) *ImportAnalyzer {
	return &ImportAnalyzer{
		ModuleName: moduleName,
		root:       root,
	}
}

// ImportDep 一条导入依赖。
type ImportDep struct {
	SourceFile string // 源文件（相对路径）
	SourcePkg  string // 源包路径
	ImportPath string // 导入的目标路径
	Alias      string // 导入别名（如果有）
	Line       int    // 行号
}

// isInternalImport 判断导入路径是否属于当前模块（项目内部依赖）。
func (a *ImportAnalyzer) isInternalImport(impPath string) bool {
	return a.ModuleName != "" && strings.HasPrefix(impPath, a.ModuleName)
}

// ParseFileImports 分析单个文件的导入声明。
func (a *ImportAnalyzer) ParseFileImports(filePath string) ([]ImportDep, error) {
	absPath := filepath.Join(a.root, filePath)
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, absPath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("解析文件 %s 失败: %w", filePath, err)
	}

	pkgPath := a.packagePath(filePath)
	var deps []ImportDep

	for _, imp := range f.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		dep := ImportDep{
			SourceFile: filePath,
			SourcePkg:  pkgPath,
			ImportPath: impPath,
			Alias:      alias,
			Line:       fset.Position(imp.Pos()).Line,
		}
		deps = append(deps, dep)

		// 如果是内部依赖，还需要记录包级别的依赖关系
		if a.isInternalImport(impPath) {
			internalDeps = append(internalDeps, dep)
		}
	}

	return deps, nil
}

// 内部依赖缓存（包级依赖）
var internalDeps []ImportDep

// AnalyzePackageDeps 分析整个项目的包级别依赖图。
// 返回：包路径 → 依赖的包路径列表
func (a *ImportAnalyzer) AnalyzePackageDeps() (map[string][]string, error) {
	pkgDeps := make(map[string][]string)
	internalDeps = nil

	// 递归扫描所有 Go 文件
	err := filepath.Walk(a.root, func(path string, info os.FileInfo, err error) error {
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

		relPath, _ := filepath.Rel(a.root, path)
		relPath = filepath.ToSlash(relPath)

		deps, err := a.ParseFileImports(relPath)
		if err != nil {
			return nil // 跳过有问题的文件
		}

		for _, dep := range deps {
			if !a.isInternalImport(dep.ImportPath) {
				continue
			}
			// 记录包级依赖：源包 → 目标包
			if dep.SourcePkg != dep.ImportPath {
				seen := false
				for _, existing := range pkgDeps[dep.SourcePkg] {
					if existing == dep.ImportPath {
						seen = true
						break
					}
				}
				if !seen {
					pkgDeps[dep.SourcePkg] = append(pkgDeps[dep.SourcePkg], dep.ImportPath)
				}
			}
		}
		return nil
	})

	return pkgDeps, err
}

// packagePath 从文件路径推断包路径。
func (a *ImportAnalyzer) packagePath(filePath string) string {
	dir := filepath.Dir(filePath)
	dir = strings.ReplaceAll(dir, string(filepath.Separator), "/")
	if dir == "." {
		return a.ModuleName
	}
	if a.ModuleName != "" {
		return a.ModuleName + "/" + dir
	}
	return dir
}

// BuildImportGraph 构建导入关系图并加入已有图实例。
// 返回新增的实体和关系数。
func (a *ImportAnalyzer) BuildImportGraph(g *Graph) (int, error) {
	pkgDeps, err := a.AnalyzePackageDeps()
	if err != nil {
		return 0, err
	}

	count := 0
	for srcPkg, targets := range pkgDeps {
		srcID := EntityID(EntityPackage, "", srcPkg)
		for _, tgtPkg := range targets {
			tgtID := EntityID(EntityPackage, "", tgtPkg)

			// 添加包依赖关系
			g.AddRelation(&Relation{
				SourceID: srcID,
				TargetID: tgtID,
				Kind:     RelDependsOn,
			})
			count++
		}
	}
	return count, nil
}
