package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hoonfeng/paircode/internal/lsp"
)

// defaultLSPManager 是包级别的 LSP Manager 实例，懒初始化、跨工具共享。
// 因为所有工具共用同一个 Manager，ServerSpec 按语言启动一个服务器进程，
// 同一语言的多次查询复用同一连接，避免重复启动开销。
var (
	defaultLSPManager     *lsp.Manager
	defaultLSPManagerOnce sync.Once
	defaultLSPManagerRoot string
)

// getLSPManager 返回（或创建）工作区 root 对应的 LSP Manager。
func getLSPManager(root string) *lsp.Manager {
	defaultLSPManagerOnce.Do(func() {
		defaultLSPManagerRoot = root
		defaultLSPManager = lsp.NewManager(root, nil)
	})
	if defaultLSPManagerRoot != root {
		// 工作区路径变化时重建（不应发生，但保持健壮）
		defaultLSPManager.Close()
		defaultLSPManagerRoot = root
		defaultLSPManager = lsp.NewManager(root, nil)
	}
	return defaultLSPManager
}

// registerLSPTools 注册 4 个 LSP 工具到 Registry。
func registerLSPTools(r *Registry, root string) {
	mgr := getLSPManager(root)
	adapters := lsp.NewLSPTools(mgr)

	for _, a := range adapters {
		adapter := a // capture
		r.Register(&Tool{
			Name:        adapter.Name,
			Description: adapter.Desc,
			Parameters:  adapter.Param,
			ReadOnly:    true,
			Handler:     adapter.Exec,
		})
	}
}

// registerFindSymbolUsagesTool 注册 find_symbol_usages 工具（基于 LSP references）。
// 注：此工具已在 filesymbol.go 中注册（registerFileSymbolTools），
// 但此处是为兼容性保留的入口。实际注册由 filesymbol.go 完成。
func registerFindSymbolUsagesTool(r *Registry, root string) {
	// find_symbol_usages 已在 registerFileSymbolTools 中注册，
	// 此处不重复注册，仅做 keep-alive 引用确保 lsp 包的编译依赖
	_ = json.RawMessage{}
}

// init 确保 lsp 包被编译引入。
// 编译时若未使用 lsp 包会导致 import 错误，但实际使用了。
func initLSPCompileCheck() {
	_ = os.Stat
	_ = filepath.Join
	_ = fmt.Sprintf
}

// ── LSP 工具处理函数 ──
// 这些函数是 registerLSPTools 中注册的 Handler 闭包的别名形式，
// 便于单元测试直接调用。

// handleDefinition 是 lsp_definition 工具的处理函数。
func handleDefinition(ctx context.Context, args map[string]any) (string, error) {
	file, line, symbol, err := extractLSPPosArgs(args)
	if err != nil {
		return "", err
	}
	mgr := getLSPManager("") // 会从 once 获取已有的 Manager
	return mgr.Definition(ctx, file, line, symbol)
}

// extractLSPPosArgs 从参数 map 中提取 file/line/symbol。
func extractLSPPosArgs(args map[string]any) (file string, line int, symbol string, err error) {
	file, _ = args["file"].(string)
	symbol, _ = args["symbol"].(string)
	lineFloat, ok := args["line"].(float64)
	if !ok {
		return "", 0, "", fmt.Errorf("line (integer) is required")
	}
	line = int(lineFloat)
	if file == "" || symbol == "" || line < 1 {
		return "", 0, "", fmt.Errorf("file, line (>=1) and symbol are required")
	}
	return file, line, symbol, nil
}
