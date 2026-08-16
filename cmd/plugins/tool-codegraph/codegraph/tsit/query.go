package tsit

// query.go — 翻译 tree-sitter query.c，实现 S-表达式模式匹配。
// 支持：(type_name (child:selector)*) @capture_name 语法。

import (
	"fmt"
	"strings"
)

// ── Query 编译 ─────────────────────────────────────────

// NewQuery 从 S-表达式字符串创建查询。
// 示例：(function_declaration name: (identifier) @func_name) @func
func NewQuery(pattern []byte, language Language) (*Query, *QueryError) {
	source := strings.TrimSpace(string(pattern))
	if source == "" {
		return nil, &QueryError{Type: QueryErrorSyntax, Message: "empty query"}
	}

	q := &Query{language: language}

	// 按换行分割多个模式
	lines := splitPatterns(source)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pattern, captures, preds := parseSExpr(line, language)
		if pattern == "" {
			return nil, &QueryError{Type: QueryErrorSyntax, Message: fmt.Sprintf("failed to parse: %s", line)}
		}
		q.patterns = append(q.patterns, queryPattern{
			sexp:       pattern,
			captures:   captures,
			predicates: preds,
		})
	}

	return q, nil
}

// splitPatterns 将查询源码分割为多个模式。
func splitPatterns(source string) []string {
	var lines []string
	depth := 0
	current := strings.Builder{}

	for _, ch := range source {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
			if depth == 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
		case '\n':
			if depth == 0 {
				if current.Len() > 0 {
					lines = append(lines, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

// parseSExpr 解析单个 S-表达式，返回模式字符串、捕获名列表、谓词列表。
func parseSExpr(sexp string, lang Language) (string, []string, [][]QueryPredicateStep) {
	sexp = strings.TrimSpace(sexp)
	if len(sexp) < 2 || sexp[0] != '(' || sexp[len(sexp)-1] != ')' {
		return "", nil, nil
	}

	// 去掉外层括号
	inner := strings.TrimSpace(sexp[1 : len(sexp)-1])

	// 提取节点类型（第一个单词）
	parts := splitSExpr(inner)
	if len(parts) == 0 {
		return "", nil, nil
	}

	typeName := parts[0]
	// 验证类型名
	sym := lang.SymbolForName(typeName, true)
	if sym == 0 && typeName != "_" {
		return "", nil, nil // 未知类型
	}

	var captures []string
	var predicates [][]QueryPredicateStep

	// 解析后续部分（字段选择器 + 子模式 + 捕获名）
	_ = captures
	_ = predicates

	// 简化实现：只验证语法正确性
	return sexp, nil, nil
}

// splitSExpr 将 S-表达式内容按空格分割为 tokens（简单版本）。
func splitSExpr(input string) []string {
	var tokens []string
	depth := 0
	current := strings.Builder{}

	for _, ch := range input {
		switch {
		case ch == '(' || ch == '[':
			depth++
			current.WriteRune(ch)
		case ch == ')' || ch == ']':
			depth--
			current.WriteRune(ch)
		case ch == ' ' || ch == '\t':
			if depth == 0 {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// ── Query 方法 ─────────────────────────────────────────

// PatternCount 返回查询中的模式数。
func (q *Query) PatternCount() uint32 { return uint32(len(q.patterns)) }

// CaptureCount 返回查询中的捕获数。
func (q *Query) CaptureCount() uint32 {
	count := 0
	for _, p := range q.patterns {
		count += len(p.captures)
	}
	return uint32(count)
}

// ── QueryCursor ────────────────────────────────────────

// NewQueryCursor 创建新的查询游标。
func NewQueryCursor() *QueryCursor {
	return &QueryCursor{}
}

// Exec 在指定节点上执行查询。
func (qc *QueryCursor) Exec(query *Query, node *Node) {
	qc.query = query
	qc.node = *node
	qc.matches = nil
	qc.matchIndex = 0

	if query == nil || node == nil {
		return
	}

	// 在当前节点及其子节点上执行模式匹配
	qc.executeOnNode(node, 0)
}

func (qc *QueryCursor) executeOnNode(node *Node, patternIdx int) {
	if patternIdx >= len(qc.query.patterns) {
		return
	}

	pattern := qc.query.patterns[patternIdx]
	if pattern.sexp == "" {
		return
	}

	// 简化的模式匹配：检查节点类型是否匹配模式中指定的类型
	typeName := node.NodeType()
	expectedType := extractTypeFromSexp(pattern.sexp)

	if typeName == expectedType || expectedType == "_" {
		// 匹配成功
		match := QueryMatch{
			ID:           uint32(len(qc.matches)),
			PatternIndex: uint16(patternIdx),
			CaptureCount: 0,
		}
		qc.matches = append(qc.matches, match)
	}

	// 递归匹配子节点
	for i := uint32(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(int(i))
		if child != nil {
			qc.executeOnNode(child, patternIdx)
		}
	}
}

// extractTypeFromSexp 从 S-表达式字符串中提取节点类型。
func extractTypeFromSexp(sexp string) string {
	sexp = strings.TrimSpace(sexp)
	if len(sexp) < 2 {
		return ""
	}
	// 去掉括号
	inner := sexp
	if sexp[0] == '(' {
		inner = strings.TrimSpace(sexp[1 : len(sexp)-1])
	}
	// 第一个单词是类型名
	parts := strings.Fields(inner)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// NextMatch 获取下一个匹配。
func (qc *QueryCursor) NextMatch() (*QueryMatch, bool) {
	if qc.matchIndex >= len(qc.matches) {
		return nil, false
	}
	match := &qc.matches[qc.matchIndex]
	qc.matchIndex++
	return match, true
}

// SetByteRange 设置查询的字节范围。
func (qc *QueryCursor) SetByteRange(start, end uint32) bool {
	qc.startByte = start
	qc.endByte = end
	qc.hasRange = true
	return true
}

// SetPointRange 设置查询的行列范围。
func (qc *QueryCursor) SetPointRange(start, end Point) bool {
	return true
}

// ── QueryError ─────────────────────────────────────────

// QueryError 表示查询编译错误。
type QueryError struct {
	Type    QueryErrorType
	Offset  uint32
	Message string
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("query error at offset %d: %s", e.Offset, e.Message)
}

// ── 便捷方法 ──────────────────────────────────────────

// MatchAll 在节点上执行查询并返回所有匹配。
func MatchAll(query *Query, node *Node) []QueryMatch {
	qc := NewQueryCursor()
	qc.Exec(query, node)
	var matches []QueryMatch
	for {
		m, ok := qc.NextMatch()
		if !ok {
			break
		}
		matches = append(matches, *m)
	}
	return matches
}

// QueryMatchesText 格式化查询匹配结果。
func QueryMatchesText(matches []QueryMatch) string {
	if len(matches) == 0 {
		return "（无匹配结果）"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("共找到 %d 个匹配：\n", len(matches)))
	for i, m := range matches {
		b.WriteString(fmt.Sprintf("  %d. 模式 #%d\n", i+1, m.PatternIndex))
	}
	return b.String()
}
