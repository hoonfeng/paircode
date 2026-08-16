package codegraph

// py_builder.go — Python 语言解析器。
// 基于正则表达式提取函数/类/变量实体。
// Python 无官方 Go 解析器，用规则匹配主流语法。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PyBuilder Python 文件解析器。
type PyBuilder struct {
	ModuleName string
	root       string
	graph      *Graph
}

func NewPyBuilder(root, moduleName string) *PyBuilder {
	return &PyBuilder{ModuleName: moduleName, root: root, graph: NewGraph()}
}
func (b *PyBuilder) Graph() *Graph     { return b.graph }
func (b *PyBuilder) Reset()            { b.graph = NewGraph() }
func (b *PyBuilder) SetGraph(g *Graph) { b.graph = g }

// ── 正则模式 ──────────────────────────────────────────

var (
	reFuncDef    = regexp.MustCompile(`^\s*def\s+([a-zA-Z_]\w*)\s*\(`)
	reClassDef   = regexp.MustCompile(`^\s*class\s+([a-zA-Z_]\w*)\s*(?:\(|:|$)`)
	reImport     = regexp.MustCompile(`^\s*import\s+(.+)`)
	reFromImport = regexp.MustCompile(`^\s*from\s+(\S+)\s+import\s+(.+)`)
	reAssign     = regexp.MustCompile(`^\s*([a-zA-Z_]\w*)\s*=\s*`)
)

// parseContext 解析上下文跟踪缩进层级。
type parseContext struct {
	indentStack []int // 缩进栈
	depth       int   // 当前深度
}

// ── 文件解析 ──────────────────────────────────────────

func (b *PyBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	return b.extractEntities(filePath, string(source))
}

func (b *PyBuilder) extractEntities(filePath, source string) (string, error) {
	fileName := filepath.Base(filePath)
	fileID := EntityID(EntityFile, "", filePath)
	b.graph.AddEntity(&Entity{
		ID: fileID, Kind: EntityFile, Name: fileName,
		FilePath: filePath, Line: 1,
	})

	lines := strings.Split(source, "\n")
	ctx := &parseContext{}

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过空行和注释
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 检测缩进层级
		indent := countIndent(line)
		ctx.updateDepth(indent)

		lineNum := lineNo + 1

		switch {
		case reFuncDef.MatchString(trimmed):
			matches := reFuncDef.FindStringSubmatch(trimmed)
			funcName := matches[1]
			funcID := EntityID(EntityFunction, "", funcName)
			b.graph.AddEntity(&Entity{
				ID: funcID, Kind: EntityFunction, Name: funcName,
				FQN: funcName, FilePath: filePath, Line: lineNum,
				Signature: fmt.Sprintf("def %s(...)", funcName),
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: funcID, Kind: RelContains, File: filePath})

		case reClassDef.MatchString(trimmed):
			matches := reClassDef.FindStringSubmatch(trimmed)
			className := matches[1]
			classID := EntityID(EntityStruct, "", className)
			b.graph.AddEntity(&Entity{
				ID: classID, Kind: EntityStruct, Name: className,
				FQN: className, FilePath: filePath, Line: lineNum,
				Signature: fmt.Sprintf("class %s", className),
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: classID, Kind: RelContains, File: filePath})

		case reFromImport.MatchString(trimmed):
			matches := reFromImport.FindStringSubmatch(trimmed)
			modName := matches[1]
			impID := EntityID(EntityImport, filePath, modName)
			b.graph.AddEntity(&Entity{
				ID: impID, Kind: EntityImport, Name: modName,
				FQN: modName, FilePath: filePath, Line: lineNum,
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: impID, Kind: RelImports, File: filePath})

		case reImport.MatchString(trimmed):
			matches := reImport.FindStringSubmatch(trimmed)
			modName := strings.Fields(matches[1])[0]
			impID := EntityID(EntityImport, filePath, modName)
			b.graph.AddEntity(&Entity{
				ID: impID, Kind: EntityImport, Name: modName,
				FQN: modName, FilePath: filePath, Line: lineNum,
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: impID, Kind: RelImports, File: filePath})

		case reAssign.MatchString(trimmed) && ctx.depth == 0:
			// 仅顶层变量赋值
			matches := reAssign.FindStringSubmatch(trimmed)
			varName := matches[1]
			if strings.HasPrefix(varName, "_") {
				continue
			}
			varID := EntityID(EntityVariable, "", varName)
			b.graph.AddEntity(&Entity{
				ID: varID, Kind: EntityVariable, Name: varName,
				FQN: varName, FilePath: filePath, Line: lineNum,
				Signature: fmt.Sprintf("%s = ...", varName),
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: varID, Kind: RelContains, File: filePath})
		}
	}

	return fileID, nil
}

// ── 辅助 ──────────────────────────────────────────────

func countIndent(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

func (ctx *parseContext) updateDepth(indent int) {
	if len(ctx.indentStack) == 0 || indent > ctx.indentStack[len(ctx.indentStack)-1] {
		ctx.indentStack = append(ctx.indentStack, indent)
		ctx.depth = len(ctx.indentStack) - 1
	} else {
		// 弹出直到找到匹配的缩进
		for len(ctx.indentStack) > 0 && indent < ctx.indentStack[len(ctx.indentStack)-1] {
			ctx.indentStack = ctx.indentStack[:len(ctx.indentStack)-1]
		}
		if len(ctx.indentStack) > 0 && indent == ctx.indentStack[len(ctx.indentStack)-1] {
			// 同一层级
		} else {
			ctx.indentStack = append(ctx.indentStack, indent)
		}
		ctx.depth = len(ctx.indentStack) - 1
	}
}

func isPyFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".py")
}

// ── 批量解析 ──────────────────────────────────────────

func (b *PyBuilder) ParseDir(dirPath string) (int, []error) {
	var errors []error
	count := 0
	absDir := filepath.Join(b.root, dirPath)
	filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "__pycache__" ||
				info.Name() == "venv" || info.Name() == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isPyFile(info.Name()) {
			return nil
		}
		relPath, _ := filepath.Rel(b.root, path)
		relPath = filepath.ToSlash(relPath)
		if _, err := b.ParseFile(relPath); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", relPath, err))
			return nil
		}
		count++
		return nil
	})
	return count, errors
}
