// Package tsit 是 Tree-sitter 的纯 Go 翻译实现。
// 将 Tree-sitter C API (api.h) 完全移植到 Go，提供多语言语法解析能力。
// 核心架构：Parser → Language → Tree → Node → TreeCursor → Query
// 所有组件纯 Go 实现，零 CGO 依赖。
package tsit

import (
	"fmt"
	"io"
)

// ── ABI 版本 ──────────────────────────────────────────

// LanguageVersion 当前库支持的最新 ABI 版本。
const LanguageVersion = 15

// MinCompatibleLanguageVersion 支持的最低 ABI 版本。
const MinCompatibleLanguageVersion = 13

// ── 基础类型 ──────────────────────────────────────────

// Symbol 节点类型数值 ID。
type Symbol uint16

// StateId 解析器状态 ID。
type StateId uint16

// FieldId 字段 ID。
type FieldId uint16

// SymbolType 节点类型分类。
type SymbolType int

const (
	SymbolTypeRegular   SymbolType = iota // 常规节点
	SymbolTypeAnonymous                   // 匿名节点（字符串字面量）
	SymbolTypeSupertype                   // 超类型节点
	SymbolTypeAuxiliary                   // 辅助节点
)

// InputEncoding 输入编码类型。
type InputEncoding int

const (
	InputEncodingUTF8    InputEncoding = iota // UTF-8
	InputEncodingUTF16LE                      // UTF-16 小端
	InputEncodingUTF16BE                      // UTF-16 大端
	InputEncodingCustom                       // 自定义编码
)

// LogType 日志类型。
type LogType int

const (
	LogTypeParse LogType = iota // 解析日志
	LogTypeLex                  // 词法日志
)

// ── 位置与范围 ────────────────────────────────────────

// Point 表示源码中的行/列位置（0基）。
type Point struct {
	Row    uint32
	Column uint32
}

// Range 表示源码中的一个范围（字节偏移 + 行列位置）。
type Range struct {
	StartPoint Point
	EndPoint   Point
	StartByte  uint32
	EndByte    uint32
}

// InputEdit 描述一次文本编辑操作。
type InputEdit struct {
	StartByte   uint32
	OldEndByte  uint32
	NewEndByte  uint32
	StartPoint  Point
	OldEndPoint Point
	NewEndPoint Point
}

// ── 输入 ──────────────────────────────────────────────

// ReadFunc 读取源码的回调。byte_index 为当前字节偏移，position 为行列位置。
// 返回文本块指针和字节数；bytes_read=0 表示文档结束。
type ReadFunc func(payload interface{}, byteIndex uint32, position Point) ([]byte, uint32)

// Input 描述要解析的源码输入。
type Input struct {
	Payload  interface{}
	Read     ReadFunc
	Encoding InputEncoding
}

// StringInput 从字符串创建 Input。
func StringInput(source string) Input {
	data := []byte(source)
	return Input{
		Encoding: InputEncodingUTF8,
		Read: func(payload interface{}, byteIndex uint32, position Point) ([]byte, uint32) {
			if byteIndex >= uint32(len(data)) {
				return nil, 0
			}
			// 一次返回全部
			return data[byteIndex:], uint32(len(data) - int(byteIndex))
		},
	}
}

// ── 节点 ──────────────────────────────────────────────

// Node 语法树中的一个节点。不可变，通过树访问。
// context[4] 存储内部状态，类似 C 版本。
type Node struct {
	context [4]uint32
	id      unsafePointer
	tree    *Tree
}

// unsafePointer 模拟 C 的 void*（在纯 Go 中用 uintptr 表示）。
type unsafePointer = uintptr

// NullNode 表示一个空节点。
var NullNode = Node{}

// NewNode 创建一个新节点（由 Tree 内部使用）。
func newNode(id unsafePointer, tree *Tree, startByte, endByte uint32, startRow, startCol, endRow, endCol uint32) Node {
	return Node{
		context: [4]uint32{startByte, endByte, 0, 0},
		id:      id,
		tree:    tree,
	}
}

// ── 输入编码 ──────────────────────────────────────────

// DecodeFunc 解码一个码点。返回消耗的字节数，写入 codePoint。
// 写入 -1 表示输入无效。
type DecodeFunc func(data []byte) (bytesConsumed uint32, codePoint int32)

// ── 解析选项 ──────────────────────────────────────────

// ParseState 解析过程的中间状态。
type ParseState struct {
	Payload           interface{}
	CurrentByteOffset uint32
	HasError          bool
}

// ProgressCallback 解析进度回调。返回 true 表示取消解析。
type ProgressCallback func(state *ParseState) bool

// ParseOptions 解析选项。
type ParseOptions struct {
	Payload          interface{}
	ProgressCallback ProgressCallback
}

// ── 日志 ──────────────────────────────────────────────

// LogFunc 日志回调。
type LogFunc func(payload interface{}, logType LogType, message string)

// Logger 解析器日志配置。
type Logger struct {
	Payload interface{}
	Log     LogFunc
}

// ── 查询类型 ──────────────────────────────────────────

// Quantifier 捕获量词类型。
type Quantifier int

const (
	QuantifierZero       Quantifier = iota // 零次
	QuantifierZeroOrOne                    // 零或一次
	QuantifierZeroOrMore                   // 零或多次
	QuantifierOne                          // 恰好一次
	QuantifierOneOrMore                    // 一次或多次
)

// QueryErrorType 查询错误类型。
type QueryErrorType int

const (
	QueryErrorNone      QueryErrorType = iota // 无错误
	QueryErrorSyntax                          // 语法错误
	QueryErrorNodeType                        // 未知节点类型
	QueryErrorField                           // 未知字段
	QueryErrorCapture                         // 未知捕获名
	QueryErrorStructure                       // 结构错误
	QueryErrorLanguage                        // 语言不匹配
)

// QueryPredicateStepType 谓词步骤类型。
type QueryPredicateStepType int

const (
	QueryPredicateStepTypeDone    QueryPredicateStepType = iota // 结束标记
	QueryPredicateStepTypeCapture                               // 捕获引用
	QueryPredicateStepTypeString                                // 字符串字面量
)

// ── 接口定义 ──────────────────────────────────────────

// ParseTable 解析表，由语言定义提供。
// 这是 GLR 解析器的核心数据结构。
type ParseTable struct {
	// 动作表：状态 × 符号 → 动作列表
	Actions []TableAction

	// 跳转表：状态 × 非终结符 → 新状态
	Gotos []GotoEntry

	// 词法分析配置
	LexTable []LexEntry

	// 主别名表
	Aliases map[Symbol]string

	// 关键字类型集合
	KeywordSymbols map[Symbol]bool
}

// TableAction 解析动作。
type TableAction struct {
	Type   ActionType
	Symbol Symbol
	State  StateId
}

// ActionType 解析动作类型。
type ActionType int

const (
	ActionShift  ActionType = iota // 移进
	ActionReduce                   // 规约
	ActionAccept                   // 接受
	ActionError                    // 错误
)

// GotoEntry 跳转条目。
type GotoEntry struct {
	State     StateId
	Symbol    Symbol
	NextState StateId
}

// LexEntry 词法分析条目。
type LexEntry struct {
	Symbol    Symbol
	Pattern   string // 正则（简化实现）
	IsKeyword bool
	IsString  bool
}

// ── 语言元数据 ────────────────────────────────────────

// LanguageMetadata 语言元数据。
type LanguageMetadata struct {
	MajorVersion uint8
	MinorVersion uint8
	PatchVersion uint8
}

// ── 解析器接口 ────────────────────────────────────────

// Parser 是 Tree-sitter 解析器实例。
type Parser interface {
	// SetLanguage 设置解析器使用的语言。
	SetLanguage(language Language) error

	// Language 获取当前语言。
	Language() Language

	// Parse 解析源码并创建语法树。
	Parse(oldTree *Tree, input Input) (*Tree, error)

	// ParseCtx 解析源码（带上下文取消）。
	ParseCtx(oldTree *Tree, input Input, parseOptions *ParseOptions) (*Tree, error)

	// ParseString 解析字符串。
	ParseString(oldTree *Tree, source string) (*Tree, error)

	// Reset 重置解析器。
	Reset()

	// SetIncludedRanges 设置包含的解析范围。
	SetIncludedRanges(ranges []Range) bool

	// IncludedRanges 获取包含的解析范围。
	IncludedRanges() []Range

	// SetLogger 设置日志回调。
	SetLogger(logger Logger)

	// Close 释放资源。
	Close()
}

// internalNode 内部节点存储。
type internalNode struct {
	symbol          Symbol
	startByte       uint32
	endByte         uint32
	startPoint      Point
	endPoint        Point
	children        []int // 子节点索引
	parent          int   // 父节点索引（-1=根）
	namedChildCount int
	childCount      int
	isNamed         bool
	isExtra         bool
	isError         bool
	isMissing       bool
}

// ── 查询接口 ──────────────────────────────────────────

// Query 是编译后的 S-表达式查询。
type Query struct {
	patterns []queryPattern
	language Language
}

// queryPattern 一个查询模式。
type queryPattern struct {
	sexp       string
	captures   []string
	predicates [][]QueryPredicateStep
}

// QueryCapture 查询捕获结果。
type QueryCapture struct {
	Node  Node
	Index uint32
}

// QueryMatch 查询匹配结果。
type QueryMatch struct {
	ID           uint32
	PatternIndex uint16
	CaptureCount uint16
	Captures     []QueryCapture
}

// QueryPredicateStep 谓词步骤。
type QueryPredicateStep struct {
	Type    QueryPredicateStepType
	ValueID uint32
}

// QueryCursor 查询游标，用于迭代查询结果。
type QueryCursor struct {
	query      *Query
	node       Node
	matches    []QueryMatch
	matchIndex int
	captureIdx int
	startByte  uint32
	endByte    uint32
	hasRange   bool
}

// ── 辅助函数 ──────────────────────────────────────────

// NodeType 返回节点的类型名。
func (n *Node) NodeType() string {
	if n == nil || n.tree == nil || n.id == 0 {
		return ""
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return ""
	}
	return n.tree.language.SymbolName(n.tree.nodes[idx].symbol)
}

// StartByte 返回节点起始字节偏移。
func (n *Node) StartByte() uint32 {
	if n == nil {
		return 0
	}
	return n.context[0]
}

// EndByte 返回节点结束字节偏移。
func (n *Node) EndByte() uint32 {
	if n == nil {
		return 0
	}
	return n.context[1]
}

// StartPoint 返回节点起始行列位置。
func (n *Node) StartPoint() Point {
	if n == nil || n.tree == nil {
		return Point{}
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return Point{}
	}
	return n.tree.nodes[idx].startPoint
}

// EndPoint 返回节点结束行列位置。
func (n *Node) EndPoint() Point {
	if n == nil || n.tree == nil {
		return Point{}
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return Point{}
	}
	return n.tree.nodes[idx].endPoint
}

// IsNull 检查节点是否为空。
func (n *Node) IsNull() bool {
	return n == nil || n.id == 0
}

// IsNamed 检查节点是否为命名节点。
func (n *Node) IsNamed() bool {
	if n == nil || n.tree == nil || n.id == 0 {
		return false
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return false
	}
	return n.tree.nodes[idx].isNamed
}

// HasError 检查节点或其子节点是否包含语法错误。
func (n *Node) HasError() bool {
	if n == nil || n.tree == nil {
		return false
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return false
	}
	return n.tree.nodes[idx].isError
}

// IsError 检查节点本身是否为语法错误节点。
func (n *Node) IsError() bool {
	if n == nil || n.tree == nil {
		return false
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return false
	}
	sym := n.tree.nodes[idx].symbol
	return sym == 0 // symbol 0 通常为 ERROR
}

// String 返回节点的 S-表达式表示。
func (n *Node) String() string {
	if n == nil || n.IsNull() {
		return "(NULL)"
	}
	return n.buildSExpr()
}

func (n *Node) buildSExpr() string {
	if n == nil || n.tree == nil {
		return ""
	}
	idx := int(n.id) - 1
	if idx < 0 {
		return ""
	}
	inner := n.tree.nodes[idx]
	typeName := n.tree.language.SymbolName(inner.symbol)

	if len(inner.children) == 0 {
		return fmt.Sprintf("(%s)", typeName)
	}

	s := fmt.Sprintf("(%s", typeName)
	for _, childIdx := range inner.children {
		if childIdx < 0 || childIdx >= len(n.tree.nodes) {
			continue
		}
		child := &Node{
			id:   unsafePointer(childIdx + 1),
			tree: n.tree,
			context: [4]uint32{
				n.tree.nodes[childIdx].startByte,
				n.tree.nodes[childIdx].endByte,
				0, 0,
			},
		}
		s += " " + child.buildSExpr()
	}
	s += ")"
	return s
}

// ── 错误 ──────────────────────────────────────────────

// ParseError 解析错误。
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string { return e.Message }

// ── 辅助 ──────────────────────────────────────────────

// DumpDot 输出 DOT 格式的语法树描述（用于可视化）。
func (n *Node) DumpDot(w io.Writer) {
	if n == nil || n.IsNull() {
		fmt.Fprintln(w, "digraph { null [label=\"NULL\"] }")
		return
	}
	fmt.Fprintln(w, "digraph tree_sitter {")
	fmt.Fprintln(w, "  node [shape=plaintext];")
	n.dumpDotNode(w, 0)
	fmt.Fprintln(w, "}")
}

func (n *Node) dumpDotNode(w io.Writer, id int) {
	if n == nil || n.tree == nil {
		return
	}
	idx := int(n.id) - 1
	if idx < 0 || idx >= len(n.tree.nodes) {
		return
	}
	inner := n.tree.nodes[idx]
	typeName := n.tree.language.SymbolName(inner.symbol)

	fmt.Fprintf(w, "  node_%d [label=\"%s\\n%d:%d-%d:%d\"];\n",
		id, typeName,
		inner.startPoint.Row, inner.startPoint.Column,
		inner.endPoint.Row, inner.endPoint.Column)

	for i, childIdx := range inner.children {
		if childIdx < 0 || childIdx >= len(n.tree.nodes) {
			continue
		}
		childID := id*100 + i + 1
		fmt.Fprintf(w, "  node_%d -> node_%d;\n", id, childID)

		child := &Node{
			id:   unsafePointer(childIdx + 1),
			tree: n.tree,
		}
		child.dumpDotNode(w, childID)
	}
}
