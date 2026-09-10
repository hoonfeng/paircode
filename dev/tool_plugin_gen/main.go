//go:build toolsgen

// ═══════════════════════════════════════════════════════════════
// tool_plugin_gen — 磁盘工具插件生成器入口（build tag: toolsgen）。
//
// 用法（工作区根执行）：
//
//	go run -tags toolsgen ./dev/tool_plugin_gen
//
// 从内核注册函数（genToolGroups）重新生成 .pair/plugins/tool-* 声明文件
// （幂等：内容相同则不产生实际差异）。Go 侧工具描述/注册变更后重跑即可同步。
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
	for _, f := range written {
		fmt.Println("已生成:", f)
	}
	fmt.Printf("完成：%d 个文件（root=%s）\n", len(written), root)
}
