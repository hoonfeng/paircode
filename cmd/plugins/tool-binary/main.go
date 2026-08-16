// ═══════════════════════════════════════════════════════════════
// tool-binary — 统一宿主工具二进制（承载全部内置工具组实现）
//
// 为什么需要：插件=自包含包（源码+二进制+资源），依赖 Go 内核的工具组
// （codegraph/lsp/office/memory/verify/binary/git…）由本二进制独立承载，
// 产物 <插件目录>/bin/tool-binary.exe。改实现 → 重编译本二进制 → 替换
// 产物 → 重启生效（主程序 exe 无需重编译）。
//
// 协议（与 ctx.binary.exec 对齐）：
//   stdin  JSON {"tool":"codegraph_stats","args":{...},"root":"<工作区根>"}
//   stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//   exit 0（协议错误 exit 2）
//
// 实现复用 agent.RegisterDefaultTools + Registry.Execute：
//   - 注册全部内置 Go 工具组（builtin_plugin_specs，core 最先）
//   - 排除会话绑定工具（update_tasks 等，宿主内保持）
//   - 每次调用独立进程（无状态），root 由请求注入
// ═══════════════════════════════════════════════════════════════
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

// 会话绑定工具（宿主内保持，二进制不承载）：task 组的 update_tasks 等。
// 其余 SystemTool（ask_user/task_create/plan/tool_stats/history_*）不在
// RegisterDefaultTools 注册范围，无需排除。
var excludedTools = map[string]bool{
	"update_tasks": true,
}

type request struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
	Root string         `json:"root"`
}

func main() {
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
	if excludedTools[req.Tool] {
		fail(fmt.Errorf("工具 %s 为会话绑定工具，由宿主框架执行", req.Tool))
	}

	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	for name := range excludedTools {
		reg.Unregister(name)
	}

	argsJSON, _ := json.Marshal(req.Args)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	result, err := reg.Execute(ctx, req.Tool, string(argsJSON))
	if err != nil {
		fail(err)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "text": result})
}

func fail(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "error": err.Error()})
	os.Exit(2)
}
