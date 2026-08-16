// ═══════════════════════════════════════════════════════════════
// Package toolbin — 插件独立二进制的公共运行骨架
//
// ★ 2026-08-16 第三轮：工具实现从「统一宿主二进制」拆分为「每插件独立
//   二进制」——cmd/plugins/tool-<组>/ 每个 main 只注册自己的工具组，
//   产物 .pair/plugins/tool-<组>/bin/tool-<组>.exe（插件目录自包含，
//   改某组实现 → 重编译对应二进制 → 替换 → 重启生效，主程序 exe 无需
//   重编译，其他插件二进制不受影响）。
//
// 协议（与宿主 ctx.binary.exec 对齐）：
//   stdin  JSON {"tool":"git_status","args":{...},"root":"<工作区根>"}
//   stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//   exit 0（协议错误 exit 2）
//
// 实现复用 agent 内置组注册（internal/agent/builtinPluginSpecs +
// RegisterToolGroups）：组规格即实现库，编译进本二进制。
// ═══════════════════════════════════════════════════════════════
package toolbin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

type request struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
	Root string         `json:"root"`
}

// Run 启动插件二进制：注册指定工具组（groups 为内置组名，如 "git"）。
func Run(groups ...string) {
	req, root := boot()
	reg := agent.NewRegistry()
	agent.RegisterToolGroups(reg, root, groups...)
	serve(reg, req)
}

// RunRunCode 启动 tool-harness 插件二进制：只注册 run_code
// （Code Mode：node 嵌套 goja 调度 + 外部进程执行，二进制内自持 goja）。
func RunRunCode() {
	req, root := boot()
	reg := agent.NewRegistry()
	agent.RegisterRunCode(reg, root)
	serve(reg, req)
}

// boot 解析 stdin 请求，返回（请求, 工作区根）。
func boot() (request, string) {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fail(fmt.Errorf("stdin JSON 解析失败: %v", err))
	}
	if req.Tool == "" {
		fail(fmt.Errorf("缺少 tool 字段"))
	}
	root := req.Root
	if root == "" {
		if wd, err := os.Getwd(); err == nil {
			root = wd
		}
	}
	return req, root
}

// serve 执行请求并输出结果。
func serve(reg *agent.Registry, req request) {
	argsJSON, _ := json.Marshal(req.Args)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	result, err := reg.Execute(ctx, req.Tool, string(argsJSON))
	if err != nil {
		fail(err)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "text": result})
}

// fail 输出错误并退出（协议错误 exit 2）。
func fail(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "error": err.Error()})
	os.Exit(2)
}
