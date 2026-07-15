package lsp

import (
	"context"
	"encoding/json"
	"fmt"
)

// LSPTool 是 LSP Manager 方法对 tool.Tool 接口的适配。
// 它实现了只读工具，LSP 批量查询时可以共享每语言一个服务器。
type LSPTool struct {
	mgr  *Manager
	name string
	desc string
	args func(context.Context) (string, error)
}

// CallTool 是描述 LSP 工具调用所需参数的简化接口。
type CallTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, file, symbol string, line int) (string, error)
}

// DefinitionTool 创建 "lsp_definition" 工具适配器。
// 外部使用方（agent）根据此结构注册到工具注册表。
//
// 返回的函数签名为：func(ctx, file, symbol string, line int) (string, error)
// 对应 Manager.Definition。
type ToolAdapter struct {
	Mgr   *Manager
	Name  string
	Desc  string
	Exec  func(ctx context.Context, args map[string]any) (string, error)
	Param map[string]any
}

// NewLSPTools 返回 4 个 LSP 工具的适配器列表，供 agent 注册。
// 每个适配器包含 Name、Description、Parameters Schema 和 Executor。
func NewLSPTools(mgr *Manager) []ToolAdapter {
	return []ToolAdapter{
		{
			Mgr:  mgr,
			Name: "lsp_definition",
			Desc: "Jump to where a symbol is defined. Give the file path (relative to workspace root), the 1-based line number the symbol appears on, and the symbol text itself.",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file":   map[string]any{"type": "string", "description": "Path to the source file, relative to the workspace root or absolute."},
					"line":   map[string]any{"type": "integer", "description": "1-based line number the symbol appears on."},
					"symbol": map[string]any{"type": "string", "description": "The exact symbol text on that line, e.g. \"executeBatch\"."},
				},
				"required": []string{"file", "line", "symbol"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				file, line, symbol, err := extractPosArgs(args)
				if err != nil {
					return "", err
				}
				return mgr.Definition(ctx, file, line, symbol)
			},
		},
		{
			Mgr:  mgr,
			Name: "lsp_references",
			Desc: "List every reference to a symbol across the workspace. Give the file path (relative to workspace root), the 1-based line number, and the symbol text.",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file":   map[string]any{"type": "string", "description": "Path to the source file, relative to the workspace root or absolute."},
					"line":   map[string]any{"type": "integer", "description": "1-based line number the symbol appears on."},
					"symbol": map[string]any{"type": "string", "description": "The exact symbol text on that line, e.g. \"executeBatch\"."},
				},
				"required": []string{"file", "line", "symbol"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				file, line, symbol, err := extractPosArgs(args)
				if err != nil {
					return "", err
				}
				return mgr.References(ctx, file, line, symbol)
			},
		},
		{
			Mgr:  mgr,
			Name: "lsp_hover",
			Desc: "Show the type signature and documentation for a symbol. Give the file path (relative to workspace root), the 1-based line number, and the symbol text.",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file":   map[string]any{"type": "string", "description": "Path to the source file, relative to the workspace root or absolute."},
					"line":   map[string]any{"type": "integer", "description": "1-based line number the symbol appears on."},
					"symbol": map[string]any{"type": "string", "description": "The exact symbol text on that line, e.g. \"executeBatch\"."},
				},
				"required": []string{"file", "line", "symbol"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				file, line, symbol, err := extractPosArgs(args)
				if err != nil {
					return "", err
				}
				return mgr.Hover(ctx, file, line, symbol)
			},
		},
		{
			Mgr:  mgr,
			Name: "lsp_diagnostics",
			Desc: "Report compiler/linter diagnostics (errors, warnings) for a file from its language server. Use after editing to check the change compiles.",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file": map[string]any{"type": "string", "description": "Path to the source file, relative to the workspace root or absolute."},
				},
				"required": []string{"file"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				file, ok := args["file"].(string)
				if !ok || file == "" {
					return "", fmt.Errorf("file is required")
				}
				return mgr.Diagnostics(ctx, file)
			},
		},
		{
			Mgr:  mgr,
			Name: "get_file_symbols",
			Desc: "列出指定文件中所有检测到的符号（函数、类、接口、类型、常量、变量等），返回每个符号的名称、种类、所在行号及子符号。基于 LSP（语言服务器协议）的 documentSymbol 能力，支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java 等语言。",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"filePath": map[string]any{"type": "string", "description": "文件路径（工作区内）"},
				},
				"required": []string{"filePath"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				file, ok := args["filePath"].(string)
				if !ok || file == "" {
					return "", fmt.Errorf("filePath is required")
				}
				return mgr.DocumentSymbol(ctx, file)
			},
		},
		{
			Mgr:  mgr,
			Name: "find_symbol_usages",
			Desc: "搜索指定符号在项目中的所有引用位置（使用 LSP textDocument/references 能力）。先通过 documentSymbol 在给定文件中定位符号定义，再查询其全部引用。返回每个引用的文件路径、行号和摘要上下文。支持 Go / TypeScript / JavaScript / Python / Rust / C/C++ / Java 等语言。",
			Param: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":     map[string]any{"type": "string", "description": "符号名称（如函数名、类型名、变量名）"},
					"filePath": map[string]any{"type": "string", "description": "符号所在的文件路径（用于定位符号定义位置，必填）"},
				},
				"required": []string{"name", "filePath"},
			},
			Exec: func(ctx context.Context, args map[string]any) (string, error) {
				name, _ := args["name"].(string)
				filePath, _ := args["filePath"].(string)
				if name == "" || filePath == "" {
					return "", fmt.Errorf("name and filePath are required")
				}
				return mgr.References(ctx, filePath, 1, name)
			},
		},
	}
}

func extractPosArgs(args map[string]any) (file string, line int, symbol string, err error) {
	file, _ = args["file"].(string)
	symbol, _ = args["symbol"].(string)
	lineFloat, ok := args["line"].(float64)
	if !ok {
		return "", 0, "", fmt.Errorf("line (int) is required")
	}
	line = int(lineFloat)
	if file == "" || symbol == "" || line < 1 {
		return "", 0, "", fmt.Errorf("file, line (>=1) and symbol are required")
	}
	return file, line, symbol, nil
}

// 确保 ToolAdapter 编译时兼容性检查。
var _ json.RawMessage = nil // 导入确保 json 被使用
