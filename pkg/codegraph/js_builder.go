package codegraph

// js_builder.go — JavaScript/TypeScript 解析器（基于 goja/parser）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/goja/ast"
	"github.com/hoonfeng/paircode/goja/parser"
)

// JSBuilder 基于 goja parser 的 JS/TS 解析器。
type JSBuilder struct {
	ModuleName string
	root       string
	graph      *Graph
}

func NewJSBuilder(root, moduleName string) *JSBuilder {
	return &JSBuilder{ModuleName: moduleName, root: root, graph: NewGraph()}
}
func (b *JSBuilder) Graph() *Graph     { return b.graph }
func (b *JSBuilder) Reset()            { b.graph = NewGraph() }
func (b *JSBuilder) SetGraph(g *Graph) { b.graph = g }

// ── 文件解析 ──────────────────────────────────────────

func (b *JSBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}
	program, err := parser.ParseFile(nil, filePath, string(source), 0)
	if err != nil {
		if program == nil {
			return "", fmt.Errorf("解析 JS 文件 %s 失败: %w", filePath, err)
		}
	}
	return b.extractEntities(program, filePath)
}

func (b *JSBuilder) extractEntities(program *ast.Program, filePath string) (string, error) {
	fileName := filepath.Base(filePath)
	fileID := EntityID(EntityFile, "", filePath)
	b.graph.AddEntity(&Entity{
		ID: fileID, Kind: EntityFile, Name: fileName,
		FilePath: filePath, Line: 1,
	})

	// DeclarationList 只有 var 声明
	for _, vd := range program.DeclarationList {
		b.processVarDecl(vd, filePath, fileID)
	}

	// Body 包含所有语句和函数/类声明
	for _, stmt := range program.Body {
		b.processStmt(stmt, filePath, fileID)
	}

	return fileID, nil
}

// processStmt 处理所有类型的语句和声明
func (b *JSBuilder) processStmt(stmt ast.Statement, filePath, fileID string) {
	switch s := stmt.(type) {
	case *ast.FunctionDeclaration:
		if s.Function != nil && s.Function.Name != nil {
			name := string(s.Function.Name.Name)
			entID := EntityID(EntityFunction, "", name)
			b.graph.AddEntity(&Entity{
				ID: entID, Kind: EntityFunction, Name: name,
				FQN: name, FilePath: filePath, Line: int(s.Function.Idx0()),
				Signature: fmt.Sprintf("function %s(...)", name),
			})
			b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: entID, Kind: RelContains, File: filePath})
		}

	case *ast.ClassDeclaration:
		if s.Class != nil && s.Class.Name != nil {
			className := string(s.Class.Name.Name)
			b.processClass(s.Class, className, filePath, fileID)
		}

	case *ast.VariableStatement:
		for _, binding := range s.List {
			b.processBinding(binding, filePath, fileID)
		}

	case *ast.ExpressionStatement:
		_ = s // 忽略表达式语句（调用等）
	}
}

// processVarDecl 处理 var 声明（来自 DeclarationList）
func (b *JSBuilder) processVarDecl(vd *ast.VariableDeclaration, filePath, fileID string) {
	for _, binding := range vd.List {
		b.processBinding(binding, filePath, fileID)
	}
}

// processBinding 处理单个绑定
func (b *JSBuilder) processBinding(binding *ast.Binding, filePath, fileID string) {
	if binding == nil {
		return
	}
	name := extractNameFromTarget(binding.Target)
	if name == "" || name == "_" {
		return
	}
	entID := EntityID(EntityVariable, "", name)
	b.graph.AddEntity(&Entity{
		ID: entID, Kind: EntityVariable, Name: name,
		FQN: name, FilePath: filePath, Line: int(binding.Idx0()),
		Signature: fmt.Sprintf("var %s", name),
	})
	b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: entID, Kind: RelContains, File: filePath})
}

// processClass 处理类声明
func (b *JSBuilder) processClass(cl *ast.ClassLiteral, className, filePath, fileID string) {
	classID := EntityID(EntityStruct, "", className)
	b.graph.AddEntity(&Entity{
		ID: classID, Kind: EntityStruct, Name: className,
		FQN: className, FilePath: filePath, Line: int(cl.Idx0()),
		Signature: fmt.Sprintf("class %s", className),
	})
	b.graph.AddRelation(&Relation{SourceID: fileID, TargetID: classID, Kind: RelContains, File: filePath})

	for _, elem := range cl.Body {
		switch e := elem.(type) {
		case *ast.MethodDefinition:
			methodName := extractExprName(e.Key)
			if methodName == "" {
				continue
			}
			methodID := EntityID(EntityMethod, "", className+"."+methodName)
			b.graph.AddEntity(&Entity{
				ID: methodID, Kind: EntityMethod, Name: methodName,
				FQN: className + "." + methodName, FilePath: filePath,
				Line:      int(e.Idx0()),
				Signature: fmt.Sprintf("%s.%s(...)", className, methodName),
				Metadata:  map[string]string{"receiver": className},
			})
			b.graph.AddRelation(&Relation{SourceID: classID, TargetID: methodID, Kind: RelDefines, File: filePath})
		}
	}
}

// ── 辅助 ──────────────────────────────────────────────

func extractNameFromTarget(t ast.BindingTarget) string {
	if id, ok := t.(*ast.Identifier); ok {
		return string(id.Name)
	}
	return ""
}

func extractExprName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return string(e.Name)
	case *ast.StringLiteral:
		return string(e.Value)
	}
	return ""
}

func isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".jsx", ".mjs", ".ts", ".tsx":
		return true
	}
	return false
}

// ── 批量解析 ──────────────────────────────────────────

func (b *JSBuilder) ParseDir(dirPath string) (int, []error) {
	var errors []error
	count := 0
	absDir := filepath.Join(b.root, dirPath)
	filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isJSFile(info.Name()) {
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
