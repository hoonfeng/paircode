package tsit

// language.go — Language 接口定义 + Go 语言具体实现。
//
// 每个语言（Go、JS、Python 等）实现 Language 接口，提供：
//   - 节点类型名 ↔ Symbol 映射
//   - 字段名 ↔ FieldId 映射
//   - 词法分析配置
//   - AST 树构建（ParseFile 解析源码 → Tree）

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// ── Language 接口 ──────────────────────────────────────

// Language 接口定义了 Tree-sitter 语言的全部能力。
type Language interface {
	Name() string
	SymbolCount() uint32
	StateCount() uint32
	FieldCount() uint32
	ABIVersion() uint32

	SymbolForName(name string, isNamed bool) Symbol
	SymbolName(symbol Symbol) string
	SymbolType(symbol Symbol) SymbolType

	FieldNameForID(id FieldId) string
	FieldIDForName(name string) FieldId

	ParseFile(path string, source []byte) (*Tree, error)
	ParseTable() *ParseTable

	FieldNameForChild(node *Node, index int) string
}

// ── Go 语言实现 ───────────────────────────────────────

type GoLanguage struct {
	symbols       map[string]Symbol
	symbolNames   map[Symbol]string
	symbolTypes   map[Symbol]SymbolType
	fieldNames    map[FieldId]string
	fieldIDs      map[string]FieldId
	fieldForChild map[string]map[int]string
}

func NewGoLanguage() *GoLanguage {
	lang := &GoLanguage{
		symbols:       make(map[string]Symbol),
		symbolNames:   make(map[Symbol]string),
		symbolTypes:   make(map[Symbol]SymbolType),
		fieldNames:    make(map[FieldId]string),
		fieldIDs:      make(map[string]FieldId),
		fieldForChild: make(map[string]map[int]string),
	}
	lang.initSymbols()
	lang.initFields()
	return lang
}

func (l *GoLanguage) initSymbols() {
	goTypes := []string{
		"source_file", "package_clause", "import_declaration",
		"function_declaration", "method_declaration",
		"parameter_list", "parameter",
		"type_declaration", "type_spec", "struct_type", "interface_type",
		"field_declaration_list", "field_declaration",
		"var_declaration", "const_declaration",
		"expression_statement", "if_statement", "for_statement",
		"return_statement", "block", "call_expression",
		"selector_expression", "identifier", "literal",
		"binary_expression", "unary_expression",
		"comment", "ERROR",
	}
	for i, name := range goTypes {
		sym := Symbol(i + 1)
		l.symbols[name] = sym
		l.symbolNames[sym] = name
		if name == "comment" || name == "ERROR" {
			l.symbolTypes[sym] = SymbolTypeAnonymous
		} else {
			l.symbolTypes[sym] = SymbolTypeRegular
		}
	}
}

func (l *GoLanguage) initFields() {
	goFields := []string{
		"name", "type", "body", "results", "parameters",
		"receiver", "doc", "func", "decl", "specs",
		"names", "values", "key", "value", "x", "y",
		"sel", "args", "cond", "init", "post", "stmt",
		"lhs", "rhs", "tag", "comment",
	}
	for i, name := range goFields {
		fid := FieldId(i + 1)
		l.fieldNames[fid] = name
		l.fieldIDs[name] = fid
	}

	l.fieldForChild["function_declaration"] = map[int]string{0: "name", 1: "parameters", 2: "type", 3: "body"}
	l.fieldForChild["method_declaration"] = map[int]string{0: "receiver", 1: "name", 2: "parameters", 3: "type", 4: "body"}
	l.fieldForChild["field_declaration"] = map[int]string{0: "name", 1: "type", 2: "tag"}
	l.fieldForChild["call_expression"] = map[int]string{0: "function", 1: "arguments"}
	l.fieldForChild["binary_expression"] = map[int]string{0: "left", 1: "right"}
	l.fieldForChild["if_statement"] = map[int]string{0: "cond", 1: "consequence", 2: "alternative"}
	l.fieldForChild["for_statement"] = map[int]string{0: "init", 1: "cond", 2: "post", 3: "body"}
}

func (l *GoLanguage) Name() string                          { return "go" }
func (l *GoLanguage) SymbolCount() uint32                   { return uint32(len(l.symbols)) }
func (l *GoLanguage) StateCount() uint32                    { return 100 }
func (l *GoLanguage) FieldCount() uint32                    { return uint32(len(l.fieldNames)) }
func (l *GoLanguage) ABIVersion() uint32                    { return LanguageVersion }
func (l *GoLanguage) NextState(state StateId, symbol Symbol) StateId { return 0 }
func (l *GoLanguage) ParseTable() *ParseTable               { return nil }

func (l *GoLanguage) SymbolForName(name string, isNamed bool) Symbol {
	if sym, ok := l.symbols[strings.ToLower(name)]; ok {
		return sym
	}
	return 0
}
func (l *GoLanguage) SymbolName(symbol Symbol) string {
	if name, ok := l.symbolNames[symbol]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_%d", symbol)
}
func (l *GoLanguage) SymbolType(symbol Symbol) SymbolType {
	if t, ok := l.symbolTypes[symbol]; ok {
		return t
	}
	return SymbolTypeRegular
}
func (l *GoLanguage) FieldNameForID(id FieldId) string     { return l.fieldNames[id] }
func (l *GoLanguage) FieldIDForName(name string) FieldId {
	if id, ok := l.fieldIDs[name]; ok {
		return id
	}
	return 0
}

// ── Go 文件解析 ───────────────────────────────────────

func (l *GoLanguage) ParseFile(path string, source []byte) (*Tree, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, source,
		parser.ParseComments|parser.AllErrors)
	if err != nil {
		if f == nil {
			return nil, fmt.Errorf("解析 Go 文件失败: %w", err)
		}
	}

	tree := &Tree{language: l}
	tree.root = l.buildTree(fset, f, source, tree)
	return tree, nil
}

// ── AST → Tree 转换 ──────────────────────────────────

func (l *GoLanguage) buildTree(fset *token.FileSet, f *ast.File, source []byte, tree *Tree) *Node {
	// 第一遍：DFS 遍历 ast.Node 树，为每个节点创建 internalNode
	// 用 map ast.Node → index 记录节点到 internalNode 索引的映射
	nodeIndex := make(map[ast.Node]int)

	var traverse func(node ast.Node) int
	traverse = func(node ast.Node) int {
		if node == nil {
			return -1
		}
		if idx, ok := nodeIndex[node]; ok {
			return idx
		}

		startPos := fset.Position(node.Pos())
		endPos := fset.Position(node.End())

		typeName := astNodeType(node)
		if typeName == "" {
			return -1
		}

		sym := l.SymbolForName(typeName, true)

		// 递归处理子节点
		var childIndices []int
		namedCount := 0

		// 通过 ast.Walk 收集直接子节点
		ast.Walk(collector{
			parent:   node,
			fset:     fset,
			traverse: traverse,
			children: &childIndices,
			named:    &namedCount,
		}, node)

		isNamed := typeName != "comment" && typeName != "ERROR"

		inner := internalNode{
			symbol:         sym,
			startByte:      uint32(node.Pos() - 1),
			endByte:        uint32(node.End() - 1),
			startPoint:     Point{Row: uint32(max(0, startPos.Line-1)), Column: uint32(max(0, startPos.Column-1))},
			endPoint:       Point{Row: uint32(max(0, endPos.Line-1)), Column: uint32(max(0, endPos.Column-1))},
			children:       childIndices,
			namedChildCount: namedCount,
			childCount:     len(childIndices),
			isNamed:        isNamed,
			isExtra:        typeName == "comment",
			isError:        typeName == "ERROR",
			parent:         -1,
		}

		tree.nodes = append(tree.nodes, inner)
		idx := len(tree.nodes) - 1
		nodeIndex[node] = idx

		// 设置父节点
		for _, cIdx := range childIndices {
			if cIdx >= 0 && cIdx < len(tree.nodes) {
				tree.nodes[cIdx].parent = idx
			}
		}

		return idx
	}

	rootIdx := traverse(f)
	if rootIdx < 0 || len(tree.nodes) == 0 {
		return &Node{id: 0, tree: tree}
	}

	root := &Node{
		id:   unsafePointer(rootIdx + 1),
		tree: tree,
		context: [4]uint32{
			tree.nodes[rootIdx].startByte,
			tree.nodes[rootIdx].endByte, 0, 0,
		},
	}
	_ = root
	return tree.nodesAsNode(rootIdx, tree)
}

// newNodeRef 从索引创建 Node 引用。
func (t *Tree) nodesAsNode(idx int, tree *Tree) *Node {
	if idx < 0 || idx >= len(tree.nodes) {
		return &Node{id: 0, tree: tree}
	}
	return &Node{
		id:   unsafePointer(idx + 1),
		tree: tree,
		context: [4]uint32{
			tree.nodes[idx].startByte, tree.nodes[idx].endByte, 0, 0,
		},
	}
}

// collector 实现 ast.Visitor，只收集 parent 的直接子节点。
type collector struct {
	parent   ast.Node
	fset     *token.FileSet
	traverse func(ast.Node) int
	children *[]int
	named    *int
}

func (c collector) Visit(node ast.Node) ast.Visitor {
	if node == nil || node == c.parent {
		return c
	}
	// 检查是否为直接子节点（通过位置判断）
	if isDirectChild(c.parent, node, c.fset) {
		idx := c.traverse(node)
		if idx >= 0 {
			*c.children = append(*c.children, idx)
			if node.Pos() != 0 {
				*c.named++
			}
		}
		return nil // 不深入（由 traverse 自己深入）
	}
	return c
}

func isDirectChild(parent, child ast.Node, fset *token.FileSet) bool {
	pStart, pEnd := parent.Pos(), parent.End()
	cStart, cEnd := child.Pos(), child.End()
	return cStart >= pStart && cEnd <= pEnd && (cStart > pStart || cEnd < pEnd)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── 节点类型推断 ──────────────────────────────────────

func astNodeType(node ast.Node) string {
	switch n := node.(type) {
	case *ast.File:
		return "source_file"
	case *ast.Ident:
		return "identifier"
	case *ast.BasicLit:
		return "literal"
	case *ast.Comment:
		return "comment"
	case *ast.CommentGroup:
		return "comment_group"
	case *ast.Field:
		return "field_declaration"
	case *ast.FieldList:
		return "field_declaration_list"
	case *ast.TypeSpec:
		return "type_spec"
	case *ast.ValueSpec:
		return "value_spec"
	case *ast.ImportSpec:
		return "import_spec"
	case *ast.GenDecl:
		switch n.Tok {
		case token.IMPORT:
			return "import_declaration"
		case token.TYPE:
			return "type_declaration"
		case token.VAR:
			return "var_declaration"
		case token.CONST:
			return "const_declaration"
		}
		return "gen_declaration"
	case *ast.FuncDecl:
		if n.Recv != nil {
			return "method_declaration"
		}
		return "function_declaration"
	case *ast.FuncLit:
		return "func_literal"
	case *ast.FuncType:
		return "function_type"
	case *ast.StructType:
		return "struct_type"
	case *ast.InterfaceType:
		return "interface_type"
	case *ast.ArrayType:
		return "array_type"
	case *ast.MapType:
		return "map_type"
	case *ast.ChanType:
		return "channel_type"
	case *ast.StarExpr:
		return "pointer_type"
	case *ast.SelectorExpr:
		return "selector_expression"
	case *ast.CallExpr:
		return "call_expression"
	case *ast.IndexExpr:
		return "index_expression"
	case *ast.SliceExpr:
		return "slice_expression"
	case *ast.KeyValueExpr:
		return "key_value_expression"
	case *ast.BinaryExpr:
		return "binary_expression"
	case *ast.UnaryExpr:
		return "unary_expression"
	case *ast.ParenExpr:
		return "parenthesized_expression"
	case *ast.CompositeLit:
		return "composite_literal"
	case *ast.BlockStmt:
		return "block"
	case *ast.ExprStmt:
		return "expression_statement"
	case *ast.ReturnStmt:
		return "return_statement"
	case *ast.IfStmt:
		return "if_statement"
	case *ast.ForStmt:
		return "for_statement"
	case *ast.RangeStmt:
		return "range_statement"
	case *ast.SwitchStmt:
		return "switch_statement"
	case *ast.TypeSwitchStmt:
		return "type_switch_statement"
	case *ast.CaseClause:
		return "case_clause"
	case *ast.CommClause:
		return "comm_clause"
	case *ast.SendStmt:
		return "send_statement"
	case *ast.DeclStmt:
		return "declaration_statement"
	case *ast.LabeledStmt:
		return "labeled_statement"
	case *ast.BranchStmt:
		return "branch_statement"
	case *ast.AssignStmt:
		return "assignment_statement"
	case *ast.IncDecStmt:
		return "inc_dec_statement"
	case *ast.DeferStmt:
		return "defer_statement"
	case *ast.GoStmt:
		return "go_statement"
	case *ast.SelectStmt:
		return "select_statement"
	case *ast.EmptyStmt:
		return "empty_statement"
	case *ast.BadExpr, *ast.BadStmt:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN_%T", node)
	}
}

// ── 字段名查询 ────────────────────────────────────────

func (l *GoLanguage) FieldNameForChild(node *Node, index int) string {
	if node == nil || node.tree == nil {
		return ""
	}
	typeName := node.NodeType()
	if fields, ok := l.fieldForChild[typeName]; ok {
		if name, ok := fields[index]; ok {
			return name
		}
	}
	return ""
}

// ── JavaScript 语言实现（骨架） ─────────────────────

type JavaScriptLanguage struct{}

func NewJavaScriptLanguage() *JavaScriptLanguage { return &JavaScriptLanguage{} }

func (l *JavaScriptLanguage) Name() string                    { return "javascript" }
func (l *JavaScriptLanguage) SymbolCount() uint32             { return 100 }
func (l *JavaScriptLanguage) StateCount() uint32              { return 100 }
func (l *JavaScriptLanguage) FieldCount() uint32              { return 0 }
func (l *JavaScriptLanguage) ABIVersion() uint32              { return LanguageVersion }
func (l *JavaScriptLanguage) SymbolForName(name string, isNamed bool) Symbol { return 1 }
func (l *JavaScriptLanguage) SymbolName(symbol Symbol) string               { return "node" }
func (l *JavaScriptLanguage) SymbolType(symbol Symbol) SymbolType           { return SymbolTypeRegular }
func (l *JavaScriptLanguage) FieldNameForID(id FieldId) string              { return "" }
func (l *JavaScriptLanguage) FieldIDForName(name string) FieldId            { return 0 }
func (l *JavaScriptLanguage) NextState(state StateId, symbol Symbol) StateId { return 0 }
func (l *JavaScriptLanguage) ParseTable() *ParseTable                       { return nil }
func (l *JavaScriptLanguage) FieldNameForChild(node *Node, index int) string { return "" }

func (l *JavaScriptLanguage) ParseFile(path string, source []byte) (*Tree, error) {
	tree := &Tree{language: l}
	tree.nodes = []internalNode{{
		symbol: 1, startByte: 0, endByte: uint32(len(source)),
		startPoint: Point{0, 0}, endPoint: Point{0, 0},
		isNamed: true, parent: -1,
	}}
	tree.root = &Node{id: unsafePointer(1), tree: tree}
	return tree, nil
}

// ── 语言检测 ──────────────────────────────────────────

func DetectLanguage(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return NewGoLanguage()
	case ".js", ".jsx", ".mjs":
		return NewJavaScriptLanguage()
	case ".ts", ".tsx":
		return NewJavaScriptLanguage()
	default:
		return nil
	}
}
