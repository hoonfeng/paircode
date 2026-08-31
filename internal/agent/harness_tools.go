// harness_tools —— 核心工具集对齐的工具注册。
//
// 背景（2026-08-15）：用户要求按设计范式重写 agent，并「暂时对齐工具集」
// 为下一步自举迭代（用 agent 开发 agent）做准备。harness 核心工具集：
//
//	tool-fs:             read / write / edit
//	tool-fs-search:      glob / grep
//	tool-str-replace:    str_replace_editor（view/create/str_replace/insert）
//	tool-bash:           bash（含 run_in_background）
//	tool-web:            web_search / web_fetch
//	code-mode:           run_code
//
// ★ Round3（2026-09）：旧名别名层 registerHarnessAliases 已删除——基座工具直接以
// harness 命名注册（registerCoreTools 注册 read/write/edit/bash/glob/grep，
// 见 tools.go/search.go），本文件仅保留 str_replace_editor 与 run_code 两个
// 独立实现。Go 侧仅测试/归档基座，生产语义以 tool-harness JS 插件为准。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// RegisterHarnessTools 注册标准命名的核心工具集。
// 基座工具（read/write/edit/bash/glob/grep）已由 registerCoreTools 以新名注册
// （Round3 别名层删除）；本入口仅补 str_replace_editor / run_code 两个独立实现。
func RegisterHarnessTools(r *Registry, root string) {
	registerStrReplaceEditor(r, root)
	registerRunCode(r, root)
}

// ─── str_replace_editor（对齐 tool-str-replace-editor）──────────────

// strReplaceEditorDesc — 命令式编辑器描述。
const strReplaceEditorDesc = "Custom editing tool for viewing, creating and editing files（对齐 str_replace_editor 惯例）\n" +
	"* `command` 必填：view / create / str_replace / insert\n" +
	"* `view` 显示文件内容（带行号）；path 为目录时列出非隐藏文件/目录最多 2 层\n" +
	"* `create` 创建新文件（path 已存在则报错）；内容在 `file_text`\n" +
	"* `str_replace` 把 `old_str` 替换为 `new_str`——old_str 必须精确匹配且唯一（含空白！不唯一则拒绝）\n" +
	"* `insert` 在 `insert_line` 之后插入 `new_str`\n" +
	"* `view` 支持 `view_range` 数组限定行范围（如 [11,12]，[-1] 到文件尾）\n" +
	"* 长输出会被截断并标记 `<response clipped>`"

func registerStrReplaceEditor(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "str_replace_editor",
		Description: strReplaceEditorDesc,
		UsageGuide: "harness 标准命令式编辑器（Claude 系工具）：view 查看、create 创建、str_replace 精确替换（唯一匹配）、insert 行后插入。" +
			"与 edit 相比更适合『需要先查看行号、再精确替换』的流程；带行号输出方便后续定位。",
		Category: "文件",
		Parameters: objSchema(props{
			"command":     strProp("要执行的命令：view / create / str_replace / insert（必填）"),
			"path":        strProp("文件或目录路径（工作区内；view 支持目录，其他命令须为文件）"),
			"file_text":   strProp("create 命令的文件内容"),
			"insert_line": intProp("insert 命令：在此行之后插入 new_str（1 基）"),
			"new_str":     strProp("str_replace 的替换新内容 / insert 的插入内容"),
			"old_str":     strProp("str_replace 的原文（须精确且唯一匹配）"),
			"view_range":  map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "view 命令行范围，如 [11,12]；[-1] 表示到文件尾"},
		}, "command", "path"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			switch command {
			case "view":
				return sreView(root, p, argIntSlice(args, "view_range"), ctx)
			case "create":
				return sreCreate(p, argStr(args, "file_text"), args, root)
			case "str_replace":
				return sreReplace(p, argStr(args, "old_str"), argStr(args, "new_str"), args, root)
			case "insert":
				return sreInsert(p, argInt(args, "insert_line", 0), argStr(args, "new_str"), args, root)
			default:
				return "", fmt.Errorf("无效 command %q：应为 view/create/str_replace/insert", command)
			}
		},
		RequiresApproval: true,
	})
}

// sreView 实现 str_replace_editor 的 view 命令：带行号显示文件，或列目录（最多 2 层）。
func sreView(root, p string, viewRange []int, ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("路径 %s 不存在：%w", displayPath(root, p), err)
	}
	if info.IsDir() {
		if len(viewRange) > 0 {
			return "", fmt.Errorf("path 指向目录时不允许 view_range")
		}
		rows, err := sreListDir(p, 0)
		if err != nil {
			return "", err
		}
		sort.Strings(rows)
		out := fmt.Sprintf("以下是 %s 中最多 2 层深度的文件/目录（排除隐藏项、node_modules）：\n%s\n",
			displayPath(root, p), strings.Join(rows, "\n"))
		return capOutput(out, 16000), nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	content := string(data)
	allLines := strings.Split(content, "\n")
	// 去掉末尾空行（split 产生的多余项）
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}
	initial, final := 1, len(allLines)
	prompt := fmt.Sprintf("以下是 %s 的内容（共 %d 行）：", displayPath(root, p), len(allLines))
	if len(viewRange) == 2 {
		initial, final = viewRange[0], viewRange[1]
		// ★ 容错：行号超界自动 clamp（不报错），避免 LLM 凭记忆传行号差几行就整次失败。
		//   历史数据：view_range 超界失败占 str_replace_editor 失败近半，多数只差 1-30 行。
		clamped := false
		if initial < 1 {
			initial = 1
			clamped = true
		}
		if initial > len(allLines) {
			initial = len(allLines) // 首元素超界 → 显示最后一行
			clamped = true
		}
		if final != -1 && final > len(allLines) {
			final = len(allLines) // 次元素超界 → 截断到文件尾
			clamped = true
		}
		if final != -1 && final < initial {
			// 逻辑错误（次 < 首）仍报错，但给出可恢复提示
			return "", fmt.Errorf("view_range 次元素 %d 应 >= 首元素 %d（文件共 %d 行；可用 view_range=[%d, %d] 读末尾）",
				viewRange[1], initial, len(allLines), max(1, len(allLines)-20), len(allLines))
		}
		if final == -1 {
			final = len(allLines)
		}
		prompt += fmt.Sprintf("（view_range=[%d, %d]）", initial, final)
		if clamped {
			prompt += fmt.Sprintf("（行号已自动修正到文件范围 [1, %d]）", len(allLines))
		}
	}
	lines := allLines[initial-1 : final]
	var b strings.Builder
	b.WriteString(prompt + ":\n")
	for i, line := range lines {
		fmt.Fprintf(&b, "%6d  %s\n", initial+i, line)
	}
	b.WriteString("\n")
	return capOutput(b.String(), 16000), nil
}

// sreListDir 递归列目录（深度 ≤ 2），排除隐藏项/node_modules/__pycache__。
func sreListDir(p string, depth int) ([]string, error) {
	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	var rows []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "__pycache__" {
			continue
		}
		rel := filepath.ToSlash(name)
		if e.IsDir() {
			rows = append(rows, "d\t"+rel)
			if depth < 2 {
				sub, err := sreListDir(filepath.Join(p, name), depth+1)
				if err != nil {
					continue
				}
				for _, s := range sub {
					// s 形如 "d\tsub" / "f\tfile"，拼接父目录前缀
					sep := strings.IndexByte(s, '\t')
					if sep < 0 {
						continue
					}
					rows = append(rows, s[:sep]+"\t"+filepath.ToSlash(filepath.Join(name, s[sep+1:])))
				}
			}
		} else {
			rows = append(rows, "f\t"+rel)
		}
	}
	return rows, nil
}

// sreCreate 实现 create 命令：创建新文件（path 已存在则拒绝）。
func sreCreate(p, fileText string, args map[string]any, root string) (string, error) {
	if _, err := os.Stat(p); err == nil {
		return "", fmt.Errorf("create 失败：%s 已存在。要修改请用 str_replace/insert 或 edit 工具", displayPath(root, p))
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(fileText), 0o644); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已创建（%d 字节）", displayPath(root, p), len(fileText)), nil
}

// sreReplace 实现 str_replace 命令：old_str 精确且唯一匹配替换。
func sreReplace(p, oldStr, newStr string, args map[string]any, root string) (string, error) {
	if oldStr == "" {
		return "", fmt.Errorf("str_replace 命令需要非空 old_str")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	content := string(data)
	n := strings.Count(content, oldStr)
	if n == 0 {
		// ★ 复用 edit 的相似行诊断：给出文件中包含 old_str 关键词的行 + 建议（比单纯报错更易恢复）
		return "", diagnoseNotFound(content, oldStr)
	}
	if n > 1 {
		// ★ 复用 edit 的多次命中诊断：列出所有命中起始行号，引导加长上下文
		return "", diagnoseMultiple(content, oldStr)
	}
	out := strings.Replace(content, oldStr, newStr, 1)
	if err := writeFileWithSnapshot(root, p, out); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已编辑（str_replace 成功，替换 1 处）", displayPath(root, p)), nil
}

// sreInsert 实现 insert 命令：在 insert_line 之后插入 new_str。
func sreInsert(p string, insertLine int, newStr string, args map[string]any, root string) (string, error) {
	if insertLine < 1 {
		return "", fmt.Errorf("insert 命令需要 insert_line >= 1（1 基）")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if insertLine > len(lines) {
		return "", fmt.Errorf("insert 失败：insert_line %d 超出文件总行数 %d", insertLine, len(lines))
	}
	var b strings.Builder
	for i, line := range lines {
		b.WriteString(line)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
		if i+1 == insertLine {
			b.WriteString(newStr)
			if !strings.HasSuffix(newStr, "\n") {
				b.WriteString("\n")
			}
		}
	}
	if err := writeFileWithSnapshot(root, p, b.String()); err != nil {
		return "", err
	}
	if FileChangeCallback != nil {
		FileChangeCallback(argStr(args, "path"))
	}
	return fmt.Sprintf("文件 %s 已编辑（在第 %d 行后插入 %d 字符）", displayPath(root, p), insertLine, len(newStr)), nil
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
