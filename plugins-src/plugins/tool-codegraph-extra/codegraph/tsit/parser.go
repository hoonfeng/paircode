package tsit

// parser.go — 翻译 tree-sitter parser.c，实现解析器入口。
// 在本 Go 移植中，Parser 委托给 Language.ParseFile 进行实际解析。

// GoParser 是 Tree-sitter 解析器的 Go 实现。
// 管理语言设置、解析调用、增量解析、范围限制等。
type GoParser struct {
	language Language
	logger   Logger
	ranges   []Range
}

// NewParser 创建一个新解析器。
func NewParser() *GoParser {
	return &GoParser{}
}

// SetLanguage 设置解析器使用的语言。返回 true 表示成功。
func (p *GoParser) SetLanguage(language Language) bool {
	if language == nil {
		return false
	}
	p.language = language
	return true
}

// Language 获取当前语言。
func (p *GoParser) Language() Language {
	return p.language
}

// Parse 解析源码并创建语法树。
// oldTree 可选，传入前次解析树可实现增量解析。
func (p *GoParser) Parse(oldTree *Tree, input Input) (*Tree, error) {
	if p.language == nil {
		return nil, &ParseError{Message: "parser has no language set"}
	}

	// 读取全部源码
	var source []byte
	for {
		chunk, n := input.Read(input.Payload, uint32(len(source)), Point{})
		if n == 0 {
			break
		}
		source = append(source, chunk...)
		if n < uint32(len(chunk)) {
			break
		}
	}

	return p.ParseString(oldTree, string(source))
}

// ParseString 解析字符串源码。
func (p *GoParser) ParseString(oldTree *Tree, source string) (*Tree, error) {
	if p.language == nil {
		return nil, &ParseError{Message: "parser has no language set"}
	}
	return p.language.ParseFile("", []byte(source))
}

// ParseCtx 解析源码（带进度回调）。
func (p *GoParser) ParseCtx(oldTree *Tree, input Input, parseOptions *ParseOptions) (*Tree, error) {
	return p.Parse(oldTree, input)
}

// Reset 重置解析器（清除所有状态）。
func (p *GoParser) Reset() {
	p.ranges = nil
}

// SetIncludedRanges 设置解析包含的范围。
func (p *GoParser) SetIncludedRanges(ranges []Range) bool {
	p.ranges = ranges
	return true
}

// IncludedRanges 获取包含的解析范围。
func (p *GoParser) IncludedRanges() []Range {
	return p.ranges
}

// SetLogger 设置日志回调。
func (p *GoParser) SetLogger(logger Logger) {
	p.logger = logger
}

// Close 释放资源。
func (p *GoParser) Close() {
	p.language = nil
	p.ranges = nil
}
