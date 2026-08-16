// ═══════════════════════════════════════════════════════════════
// tool-harness — 独立插件二进制（run_code：Code Mode 调度）
//
// tool-harness 磁盘插件 7 个工具（read/write/edit/glob/grep/bash/
// str_replace_editor）为 JS 原生化（调用实现在插件内），唯 run_code 走
// 二进制承载（node 嵌套 goja 调度 + 外部进程执行，二进制进程内自持 goja
// 运行时）——本二进制只注册 run_code。
//
// 协议（与宿主 ctx.binary.exec 对齐）：
//   stdin  JSON {"tool":"run_code","args":{...},"root":"<工作区根>"}
//   stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//   exit 0（协议错误 exit 2）
// 装配位置：.pair/plugins/tool-harness/bin/tool-harness.exe
// ═══════════════════════════════════════════════════════════════
package main

import "github.com/hoonfeng/paircode/pkg/toolbin"

func main() { toolbin.RunRunCode() }
