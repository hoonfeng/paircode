// ═══════════════════════════════════════════════════════════════
// tool-verify — 独立插件二进制（verify 工具组）
//
// ★ 二进制插件协议（宿主 ctx.binary.exec 调用）：
//   stdin  一行 JSON：{"tool":"...","args":{...},"root":"<工作区根>"}
//   stdout 一行 JSON：{"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//   exit 0（协议错误 exit 2）
//
// 实现复用 internal/agent 内置组注册（builtinPluginSpecs + RegisterToolGroups），
// 编译进本二进制；宿主进程不再承载。装配位置：
//   .pair/plugins/tool-verify/bin/tool-verify.exe（插件目录自包含）
// 改实现 → 重编译本二进制 → 替换产物 → 重启生效（其他插件不受影响）。
// ═══════════════════════════════════════════════════════════════
package main

import "github.com/hoonfeng/paircode/pkg/toolbin"

func main() { toolbin.Run("verify") }
