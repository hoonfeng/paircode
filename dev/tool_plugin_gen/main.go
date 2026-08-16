//go:build toolsgen

// ═══════════════════════════════════════════════════════════════
// tool_plugin_gen — 磁盘工具插件生成器入口（go run ./dev/tool_plugin_gen）
//
// 用法（工作区根，需 CGO 无关——仅编译 agent 包）：
//   go run -tags toolsgen ./dev/tool_plugin_gen
//
// 作用：遍历尚未外置的复杂内置工具组，把每组注册的工具定义完整导出为
// .pair/plugins/tool-<组>/index.js 磁盘插件（api 外置 + execute 调
// ctx.hostTool 复用宿主 Go 执行器）。幂等，Go 侧描述变更后重跑即可。
// ═══════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/internal/agent"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "取工作目录失败:", err)
		os.Exit(1)
	}
	outDir := filepath.Join(root, ".pair", "plugins")
	written, err := agent.GenerateToolPlugins(root, outDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "生成失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已生成 %d 个工具组插件:\n", len(written))
	for _, f := range written {
		fmt.Println("  " + f)
	}
}
