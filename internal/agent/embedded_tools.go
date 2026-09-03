// embedded_tools.go — 宿主内嵌工具内核（JS 插件二进制回退）。
//
// ★ 2026-08-22「工具 JS/Go 原生迁移」：磁盘插件（index.js）负责 api 声明 + 编排，
//   执行走 ctx.binary.exec——插件目录 bin/*.exe 已移除（归档 bin/legacy-plugin-bins/），
//   宿主提供 Go 基础能力：binary.exec 找不到 exe 时回退本文件注册的内嵌内核
//   （internal/agent 内置工具实现，与已归档 exe 同源）。
//   · 插件 JS 无需改动（仍调 ctx.binary.exec——「api 声明在插件、执行走宿主」）。
//   · 惰性初始化（调用时构建一次，幂等）。

package agent

import (
	"context"
	"encoding/json"
)

// embeddedToolRegistry：内嵌工具内核注册表（进程内单例，只读）。
var embeddedToolRegistry *Registry

// embeddedToolRegistrars：内核注册函数（与 .pair/plugins/tool-*/bin/*.exe 同源）。
var embeddedToolRegistrars = []func(r *Registry, root string){
	registerBinaryTools,         // tool-binary（inspect/write/patch/hash/entropy）
	registerBinaryRETools,       // tool-binary 逆向组（binary_strings/find/patch_re/hash_re/entropy_re）
	registerCodeGraphTools,      // tool-codegraph（18 工具，sqlite 图谱）
	registerExtraCodeGraphTools, // tool-codegraph-extra（入口点/热点路径/导入分析等 13 工具）
	registerScreenshotTools,     // tool-screenshot（desktop/window/area/webpage）
	registerWebDebugTool,        // tool-web-debug（web_debug，go-rod）
	RegisterHarnessTools,        // tool-harness（run_code 含 goja 嵌套工具调度）
	// ★ 2026-09 Round4.5：tool-debug 已移除（纯命令行包装壳），registerDebugTools
	//   内核不再挂内嵌回退（内核实现保留于 debug_tools.go，供独立二进制复用）。
	registerOfficeTools,         // tool-office（word/xlsx/pdf 仍走宿主的内核）
}

// InitEmbeddedToolRegistry 构建内嵌工具注册表（幂等；懒调用）。
func InitEmbeddedToolRegistry(root string) *Registry {
	if embeddedToolRegistry != nil {
		return embeddedToolRegistry
	}
	r := NewRegistry()
	for _, f := range embeddedToolRegistrars {
		f(r, root)
	}
	embeddedToolRegistry = r
	return r
}

// callEmbeddedTool 内嵌内核执行：found=false 表示宿主无此工具（调用方走原二进制报错路径）。
func callEmbeddedTool(ctx context.Context, root, name string, args map[string]any) (text string, found bool, err error) {
	reg := InitEmbeddedToolRegistry(root)
	if _, ok := reg.Get(name); !ok {
		return "", false, nil
	}
	argsJSON, _ := json.Marshal(args)
	text, err = reg.Execute(ctx, name, string(argsJSON))
	return text, true, err
}
