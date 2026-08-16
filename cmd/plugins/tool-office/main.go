// ═══════════════════════════════════════════════════════════════
// tool-office — 独立插件二进制（office 工具组）
//
// ★ 二进制插件协议（宿主 ctx.binary.exec 调用）：
//
//	stdin  一行 JSON：{{"tool":"...","args":{{...}},"root":"<工作区根>"}}
//	stdout 一行 JSON：{{"ok":true,"text":"..."}} | {{"ok":false,"error":"..."}}
//	exit 0（协议错误 exit 2）
//
// ★ 2026-08-16 第四轮：实现随插件外置——cmd/plugins/tool-office/impl/ 自持
//
//	本组全部实现（不 import internal/agent），本二进制只链接自身实现，
//	改实现 → 重编译本二进制 → 替换产物 → 重启生效。
//
// ═══════════════════════════════════════════════════════════════
package main

import (
	"github.com/hoonfeng/paircode/cmd/plugins/tool-office/impl"
	"github.com/hoonfeng/paircode/cmd/plugins/tool-office/toolbin"
)

func main() {
	req, root := toolbin.Boot()
	reg := toolbin.NewRegistry()
	impl.Register(reg, root)
	toolbin.Serve(reg, req)
}
