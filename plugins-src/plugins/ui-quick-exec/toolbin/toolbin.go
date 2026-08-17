// ═══════════════════════════════════════════════════════════════
// ★ 2026-08-16 自根 pkg/toolbin 内嵌（插件自包含；改协议请全局同步 17 个插件）。
// Package toolbin — 插件独立二进制的公共运行骨架
//
// ★ 2026-08-16 第三轮：工具实现从「统一宿主二进制」拆分为「每插件独立
//
//	二进制」——plugins-src/plugins/tool-<组>/ 每个 main 只注册自己的工具组。
//	★ 2026-08-16 第四轮：工具实现从 internal/agent 迁出（各插件目录 impl
//	包），独立二进制不再 import agent——本包提供轻量 Registry + 骨架。
//
// 协议（与宿主 ctx.binary.exec 对齐）：
//
//	stdin  JSON {"tool":"git_status","args":{...},"root":"<工作区根>"}
//	stdout JSON {"ok":true,"text":"..."} | {"ok":false,"error":"..."}
//	exit 0（协议错误 exit 2）
//
// 用法（各插件 main）：
//
//	reg := toolbin.NewRegistry()
//	impl.Register(reg, root)   // 注册本插件工具实现（impl 包在插件目录内）
//	toolbin.Serve(reg)         // 解析 stdin → 执行 → 输出
//
// ═══════════════════════════════════════════════════════════════
package toolbin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type request struct {
	Tool string         `json:"tool"`
	Args map[string]any `json:"args"`
	Root string         `json:"root"`
}

// Serve 执行 stdin 请求（req 为 nil 时从 stdin 解析）。
//
// ★ 超时策略（本插件专用）：args.timeoutMs 优先（默认 600s，夹到 [1s, 3600s]），
// 供 ui-quick-exec 跑长命令（打包等）——宿主 ctx.binary.exec 的 opts.timeout 是
// 外层保护，本超时是命令级精确控制（超时 kill 并返回 timedOut=true）。
func Serve(reg *Registry, req *request) {
	if req == nil {
		req, _ = Boot()
	}
	argsJSON, _ := json.Marshal(req.Args)
	timeout := 600 * time.Second
	if ms := req.Args["timeoutMs"]; ms != nil {
		var msInt int
		switch n := ms.(type) {
		case float64:
			msInt = int(n)
		case int:
			msInt = n
		case string:
			_, _ = fmt.Sscanf(n, "%d", &msInt)
		}
		if msInt >= 1000 && msInt <= 3600000 {
			timeout = time.Duration(msInt) * time.Millisecond
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := reg.Execute(ctx, req.Tool, string(argsJSON))
	if err != nil {
		fail(err)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "text": result})
}

// boot 解析 stdin 请求，返回（请求, 工作区根）。
// Boot 解析 stdin 请求，返回（请求, 工作区根）。
func Boot() (*request, string) {
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
	return &req, root
}

// fail 输出错误并退出（协议错误 exit 2）。
func fail(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": false, "error": err.Error()})
	os.Exit(2)
}
