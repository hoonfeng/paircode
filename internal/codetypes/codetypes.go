// Package codetypes 提供跨 UI 框架的共享类型定义。
// 这些类型从 goui 的 widget 包中解耦出来，使后端代码不依赖具体 UI 框架。
package codetypes

// CodeLoc 表示代码中的一个位置（文件:行:列）。
type CodeLoc struct {
	File string
	Line int
	Col  int
}

// CodeSym 表示代码符号（用于文档大纲/转到符号）。
type CodeSym struct {
	Name  string
	Kind  int
	Line  int
	Depth int
}

// CustomProviderConfig 自定义语言 Provider 配置（结构化编辑器）。
type CustomProviderConfig struct {
	Name        string   `json:"name"`        // 语言名（如 "vue"、"kotlin"）
	Extensions  []string `json:"extensions"`  // 文件扩展名列表（不含点，如 ["vue"]）
	UseProvider string   `json:"useProvider"` // 重用已有 provider 的名称（如 "js"、"go"）
}
