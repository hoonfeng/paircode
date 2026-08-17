// ═══════════════════════════════════════════════════════════════
// ui-quick-exec — 快速执行插件（独立二进制）
//
// ★ 二进制插件协议（宿主 ctx.binary.exec 调用）：
//
//	stdin  一行 JSON：{"tool":"run","args":{...},"root":"<工作区根>"}
//	stdout 一行 JSON：{"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//	exit 0（协议错误 exit 2）
//
// ★ 为什么需要二进制（2026-08-17）：runCommand 原走 ctx.bash.exec——
// 宿主 runShellWithTimeout 硬编码 120s，打包类命令超 120s 被强制 kill。
// 本二进制经 ctx.binary.exec 执行，超时由 args.timeoutMs 精确控制
// （默认 600s），完全不经 120s 桥接。
//
// ═══════════════════════════════════════════════════════════════
package main

import (
	"github.com/hoonfeng/paircode/plugins-src/plugins/ui-quick-exec/impl"
	"github.com/hoonfeng/paircode/plugins-src/plugins/ui-quick-exec/toolbin"
)

func main() {
	req, root := toolbin.Boot()
	reg := toolbin.NewRegistry()
	impl.Register(reg, root)
	toolbin.Serve(reg, req)
}
