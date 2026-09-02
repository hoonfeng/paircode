// harness_tools —— 核心工具集对齐的工具注册。
//
// 背景（2026-08-15）：用户要求按设计范式重写 agent，并「暂时对齐工具集」
// 为下一步自举迭代（用 agent 开发 agent）做准备。harness 核心工具集：
//
//	tool-fs:             read / write / edit
//	tool-fs-search:      glob / grep
//	tool-bash:           bash（内部后台 + 120s 超时）
//	tool-web:            web_search / web_fetch
//	code-mode:           run_code
//
// ★ Round3（2026-09）：旧名别名层 registerHarnessAliases 已删除——基座工具直接以
// harness 命名注册（registerCoreTools 注册 read/write/edit/bash/glob/grep，
// 见 tools.go/search.go），本文件仅保留 run_code 独立实现（★ Round4：
// str_replace_editor 命令式壳已删除——read/write/edit 覆盖链完全）。
// Go 侧仅测试/归档基座，生产语义以 tool-harness JS 插件为准。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// RegisterHarnessTools 注册标准命名的核心工具集。
// 基座工具（read/write/edit/bash/glob/grep）已由 registerCoreTools 以新名注册
// （Round3 别名层删除）；本入口仅补 run_code 独立实现（★ Round4：
// str_replace_editor 命令式壳已删除——read/write/edit 覆盖链完全）。
func RegisterHarnessTools(r *Registry, root string) {
	registerRunCode(r, root)
}



// writeFileWithSnapshot 带快照写文件（复用全局快照机制）。
func writeFileWithSnapshot(root, p, content string) error {
	SnapshotBeforeWriteWithTracking(root, p)
	return os.WriteFile(p, []byte(content), 0o644)
}

// displayPath 显示工作区相对路径（稳定、可读）。
func displayPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// ─── run_code（对齐 code-mode）─────────────────────────────────

// registerRunCode 注册 run_code 工具：执行一段代码（对齐 Code Mode）。
// Go 侧简化版：把 code 写入临时文件，按语言执行并返回 stdout/stderr。
// ★node 语言含 tools. 调用时走 goja 宿主内嵌套工具调度（见 runCodeNested）。
// RegisterRunCode 注册 run_code 工具（统一二进制承载时导出入口）。
func RegisterRunCode(r *Registry, root string) { registerRunCode(r, root) }

func registerRunCode(r *Registry, root string) {
	r.Register(&Tool{
		Name: "run_code",
		Description: "执行一段代码并返回输出（对齐 Code Mode）。" +
			"参数：code（必填，要执行的程序体）、language（可选，auto/go/python/node，默认 auto 按内容探测）、" +
			"description（可选，简短说明）。仅返回程序的 stdout/stderr 与退出状态。" +
			"★嵌套工具调度：language=node 且代码内用 tools.xxx(args) 调用已注册工具（如 tools.read({path:'a.go'})），" +
			"在宿主内执行并逐条记录工具调用结果（对齐 Code Mode 嵌套调度）；不写 tools. 则走外部 node 进程。",
		UsageGuide: "harness 标准代码执行工具：快速验证算法/处理数据/调用本地库，不用写临时文件。" +
			"与 bash 的区别：run_code 直接执行代码片段（自动建临时文件），bash 执行 shell 命令。" +
			"node 语言可在代码里 tools.read/tools.grep 等嵌套调用注册表工具，批量处理文件再汇总输出。",
		Category: "执行",
		Parameters: objSchema(props{
			"code":        strProp("要执行的代码（必填）"),
			"language":    strProp("可选：auto（默认，按内容探测）/ go / python / node"),
			"description": strProp("可选：简短说明这段代码做什么"),
		}, "code"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			code := argStr(args, "code")
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("code 不能为空")
			}
			lang := argStr(args, "language")
			if lang == "" || lang == "auto" {
				lang = detectCodeLang(code)
			}
			// ★嵌套工具调度（对齐 Code Mode）：JS 代码内可 `tools.xxx(args)`
			// 调用注册表工具，goja 宿主内执行；每个子调度记录日志，仅精选结果返回。
			if lang == "node" && strings.Contains(code, "tools.") {
				return runCodeNested(ctx, r, code)
			}
			return runCodeSnippet(ctx, root, lang, code)
		},
	})
}

// detectCodeLang 按代码内容探测语言（auto 模式）。
func detectCodeLang(code string) string {
	trimmed := strings.TrimSpace(code)
	switch {
	case strings.HasPrefix(trimmed, "package main") || strings.HasPrefix(trimmed, "package "):
		return "go"
	case strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "def ") ||
		strings.HasPrefix(trimmed, "print(") || strings.Contains(trimmed, "\nprint("):
		return "python"
	case strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "let ") ||
		strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "function ") ||
		strings.HasPrefix(trimmed, "console.log"):
		return "node"
	default:
		return "go" // 默认 Go（项目主语言，自举迭代最常用）
	}
}

// runCodeNested 嵌套工具调度：JS 代码在 goja 宿主内执行，暴露 tools 命名空间
// （每个已注册工具 → Registry.Execute），对齐 Code Mode 的嵌套调度：
// 「程序调用注册表工具；每个子调度记录日志，仅外层精选结果进入模型历史」。
// console.log 捕获为程序输出；tools.xxx 调用逐条记录（结果截断，防刷屏）。
// run_code 自身不绑定（防无限递归）。
func runCodeNested(ctx context.Context, r *Registry, code string) (string, error) {
	vm := goja.New()
	var logBuf, callBuf strings.Builder

	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, fmt.Sprintf("%v", a.Export()))
		}
		logBuf.WriteString(strings.Join(parts, " ") + "\n")
		return goja.Undefined()
	})
	console.Set("error", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, fmt.Sprintf("%v", a.Export()))
		}
		logBuf.WriteString("error: " + strings.Join(parts, " ") + "\n")
		return goja.Undefined()
	})
	vm.Set("console", console)

	tools := vm.NewObject()
	for _, name := range r.EnabledNames() { // 只注入启用工具（禁用工具不得在沙箱暴露）
		if name == "run_code" { // 防递归
			continue
		}
		name := name
		tools.Set(name, func(call goja.FunctionCall) goja.Value {
			var argsJSON string
			switch len(call.Arguments) {
			case 0:
				argsJSON = "{}"
			case 1:
				data, err := json.Marshal(call.Arguments[0].Export())
				if err != nil {
					callBuf.WriteString(fmt.Sprintf("tools.%s => 参数序列化失败: %v\n", name, err))
					return vm.ToValue("")
				}
				argsJSON = string(data)
			default:
				var arr []any
				for _, a := range call.Arguments {
					arr = append(arr, a.Export())
				}
				data, _ := json.Marshal(arr)
				argsJSON = string(data)
			}
			res, err := r.Execute(ctx, name, argsJSON)
			if err != nil {
				callBuf.WriteString(fmt.Sprintf("tools.%s(%s) => 错误: %v\n", name, argsJSON, err))
				return vm.ToValue("")
			}
			callBuf.WriteString(fmt.Sprintf("tools.%s(%s) => %s\n", name, argsJSON, capOutput(res, 500)))
			return vm.ToValue(res)
		})
	}
	vm.Set("tools", tools)

	// ★ 2026-09-01 原生调用感知（js_native_guard.go）：tools.xxx 是宿主工具执行
	//   （可能几分钟——bash 命令/子 agent 等），必须让看门狗在其阻塞期间暂停计时，
	//   否则工具跑完后 JS 恢复即撞 interrupt flag → 误判「疑似死循环」。
	wrapNativeGuards(vm, tools, 0)
	defer jsForgetNative(vm)

	runErr := runJSWithTimeout(vm, 30*time.Second, func() error {
		_, err := vm.RunString(code)
		return err
	})

	var b strings.Builder
	if callBuf.Len() > 0 {
		b.WriteString("[嵌套工具调用]\n" + callBuf.String())
	}
	if logBuf.Len() > 0 {
		b.WriteString("[程序输出]\n" + logBuf.String())
	}
	if runErr != nil {
		if callBuf.Len() > 0 || logBuf.Len() > 0 {
			b.WriteString("[执行错误] ")
		}
		b.WriteString(runErr.Error() + "\n")
	}
	if b.Len() == 0 {
		return "（程序无输出）", nil
	}
	return capOutput(b.String(), 16000), nil
}

// runCodeSnippet 执行代码片段：写临时文件 → 运行 → 清理。
func runCodeSnippet(ctx context.Context, root, lang, code string) (string, error) {
	tmp, err := os.MkdirTemp("", "run_code_*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	var file, cmd string
	// shellPath：Windows 反斜杠转正斜杠（bash 下 \ 是转义符会损坏路径；正斜杠 Windows 程序均接受），
	// 再加引号防空格路径。
	shellPath := func(p string) string {
		return `"` + strings.ReplaceAll(p, "\\", "/") + `"`
	}
	switch lang {
	case "go":
		file = filepath.Join(tmp, "main.go")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "go run " + shellPath(file)
	case "python":
		file = filepath.Join(tmp, "main.py")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "python " + shellPath(file)
	case "node":
		file = filepath.Join(tmp, "main.js")
		if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
			return "", err
		}
		cmd = "node " + shellPath(file)
	default:
		return "", fmt.Errorf("不支持的语言 %q（可用 go/python/node）", lang)
	}

	out, exitErr := runShellWithTimeout(ctx, cmd, tmp)
	res := capOutput(out, 16000)
	if exitErr != "" {
		res += "\n[退出: " + exitErr + "]"
	}
	return res, nil
}

// runShellWithTimeout 执行 shell 命令并返回输出（120s 超时 + ctx 取消）。
// 复用全局后台进程机制（不阻塞 loop 线程）。
func runShellWithTimeout(ctx context.Context, command, dir string) (string, string) {
	return runShellWithTimeoutN(ctx, command, dir, 120*time.Second)
}

// runShellWithTimeoutN ★ 2026-08-22 带自定义超时的版本（ctx.bash.exec 第三参
// timeout 用；JS 插件需要短超时（如 debug_run_capture 60s 默认）时覆盖）。
// 语义与 runShellWithTimeout 完全一致，仅 deadline 可配。
func runShellWithTimeoutN(ctx context.Context, command, dir string, timeout time.Duration) (string, string) {
	bg := globalBG
	id, err := bg.start(command, dir)
	if err != nil {
		return "", err.Error()
	}
	p := bg.get(id)
	if p == nil {
		return "", "内部错误：后台进程创建后丢失"
	}
	deadline := time.After(timeout)
	if timeout <= 0 {
		deadline = nil // nil channel select 永不触发（不设超时）
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return capOutput(out, 16000), ctx.Err().Error()
		case <-deadline:
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return capOutput(out, 16000), "超时 " + timeout.String() + " 已终止"
		case <-ticker.C:
			out, done, exitErr := p.snapshot()
			if done {
				bg.mu.Lock()
				delete(bg.procs, id)
				bg.mu.Unlock()
				return capOutput(out, 16000), exitErr
			}
		}
	}
}
