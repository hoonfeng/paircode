package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"wb-ui/goja"

	. "github.com/hoonfeng/paircode/plugins-src/plugins/tool-harness/toolbin"
)

// RegisterRunCode 注册 run_code 工具（统一二进制承载时导出入口）。
// Register 注册 run_code（独立二进制入口）。
func Register(r *Registry, root string) { registerRunCode(r, root) }

func registerRunCode(r *Registry, root string) {
	r.Register(&Tool{
		Name: "run_code",
		Description: "执行一段代码并返回输出（对齐 deepseek-harness run_code / Code Mode）。" +
			"参数：code（必填，要执行的程序体）、language（可选，auto/go/python/node，默认 auto 按内容探测）、" +
			"description（可选，简短说明）。仅返回程序的 stdout/stderr 与退出状态。" +
			"★嵌套工具调度：language=node 且代码内用 tools.xxx(args) 调用已注册工具（如 tools.read({path:'a.go'})），" +
			"在宿主内执行并逐条记录工具调用结果（对齐 Code Mode 嵌套调度）；不写 tools. 则走外部 node 进程。",
		UsageGuide: "harness 标准代码执行工具：快速验证算法/处理数据/调用本地库，不用写临时文件。" +
			"与 bash 的区别：run_code 直接执行代码片段（自动建临时文件），bash 执行 shell 命令。" +
			"node 语言可在代码里 tools.read/tools.grep 等嵌套调用注册表工具，批量处理文件再汇总输出。",
		Category: "执行",
		Parameters: ObjSchema(Props{
			"code":        StrProp("要执行的代码（必填）"),
			"language":    StrProp("可选：auto（默认，按内容探测）/ go / python / node"),
			"description": StrProp("可选：简短说明这段代码做什么"),
		}, "code"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			code := ArgStr(args, "code")
			if strings.TrimSpace(code) == "" {
				return "", fmt.Errorf("code 不能为空")
			}
			lang := ArgStr(args, "language")
			if lang == "" || lang == "auto" {
				lang = detectCodeLang(code)
			}
			// ★嵌套工具调度（对齐 harness Code Mode）：JS 代码内可 `tools.xxx(args)`
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
// （每个已注册工具 → Registry.Execute），对齐 harness Code Mode 的嵌套调度：
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
			callBuf.WriteString(fmt.Sprintf("tools.%s(%s) => %s\n", name, argsJSON, CapOutput(res, 500)))
			return vm.ToValue(res)
		})
	}
	vm.Set("tools", tools)

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
	return CapOutput(b.String(), 16000), nil
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
	res := CapOutput(out, 16000)
	if exitErr != "" {
		res += "\n[退出: " + exitErr + "]"
	}
	return res, nil
}

// runShellWithTimeout 执行 shell 命令并返回输出（120s 超时 + ctx 取消）。
// 复用全局后台进程机制（不阻塞 loop 线程）。
func runShellWithTimeout(ctx context.Context, command, dir string) (string, string) {
	bg := globalBG
	id, err := bg.start(command, dir)
	if err != nil {
		return "", err.Error()
	}
	p := bg.get(id)
	if p == nil {
		return "", "内部错误：后台进程创建后丢失"
	}
	deadline := time.After(120 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return CapOutput(out, 16000), ctx.Err().Error()
		case <-deadline:
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			out, _, _ := p.snapshot()
			return CapOutput(out, 16000), "超时 120s 已终止"
		case <-ticker.C:
			out, done, exitErr := p.snapshot()
			if done {
				bg.mu.Lock()
				delete(bg.procs, id)
				bg.mu.Unlock()
				return CapOutput(out, 16000), exitErr
			}
		}
	}
}

// ─── 支撑：后台进程/shell（迁移自 shell.go）───
var (
	detectedBashOnce sync.Once
	detectedBashPath string
	detectedMsysBin  string
)

func bashCandidate(exePath string) string {
	return filepath.Join(filepath.Dir(exePath), "bin", "bash", "usr", "bin", "bash.exe")
}

func detectBash() (bashPath, msysBin string) {
	detectedBashOnce.Do(func() {
		// 1. 内置资源：exe 同目录 bin/bash/usr/bin/bash.exe
		if exe, err := os.Executable(); err == nil {
			cand := bashCandidate(exe)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				detectedBashPath = cand
				detectedMsysBin = filepath.Dir(cand)
				// 确保 msys 根 /tmp 存在（bash 找不到会打 stderr 警告）
				os.MkdirAll(filepath.Join(filepath.Dir(filepath.Dir(cand)), "tmp"), 0o755)
				return
			}
		}
		// 2. 系统 Git Bash（回退）
		for _, cand := range []string{
			`C:\Program Files\Git\usr\bin\bash.exe`,
			`C:\Program Files (x86)\Git\usr\bin\bash.exe`,
		} {
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				detectedBashPath = cand
				detectedMsysBin = filepath.Dir(cand)
				return
			}
		}
		// 3. PATH 中的 bash（msys2/WSL 等，最后兜底）
		if p, err := exec.LookPath("bash"); err == nil {
			detectedBashPath = p
		}
	})
	return detectedBashPath, detectedMsysBin
}

// hideShellWindow 隐藏子进程控制台窗口（Windows；非 Windows 原样返回）。
// 父进程无控制台（后台/服务方式启动）时，cmd.exe/bash.exe 等 console 程序
// 会自己弹出控制台窗口，必须显式隐藏。
func hideShellWindow(c *exec.Cmd) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		c.SysProcAttr.HideWindow = true
	}
	return c
}

func newShellCommand(command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.Command(bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return hideShellWindow(c)
	}
	return hideShellWindow(exec.Command("cmd", "/C", "chcp 65001 >nul & "+command))
}

func newShellCommandContext(ctx context.Context, command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.CommandContext(ctx, bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return hideShellWindow(c)
	}
	return hideShellWindow(exec.CommandContext(ctx, "cmd", "/C", "chcp 65001 >nul & "+command))
}

func applyBashEnv(c *exec.Cmd, msysBin string) {
	if msysBin == "" {
		return
	}
	c.Env = append(os.Environ(), "PATH="+msysBin+";"+os.Getenv("PATH"))
}

type bgProc struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	buf     bytes.Buffer
	done    bool
	exitErr string
}

type bgRegistry struct {
	mu    sync.Mutex
	procs map[int]*bgProc
	next  int
}

func (p *bgProc) Write(b []byte) (int, error) {
	p.mu.Lock()
	p.buf.Write(b)
	const cap_ = 256 * 1024 // 防长跑进程输出无限增长：超限只留尾部
	if p.buf.Len() > cap_ {
		data := p.buf.Bytes()
		tail := append([]byte(nil), data[len(data)-192*1024:]...)
		p.buf.Reset()
		p.buf.Write(tail)
	}
	p.mu.Unlock()
	return len(b), nil
}

func (p *bgProc) snapshot() (out string, done bool, exitErr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 编码探测：子进程输出可能是 GBK（Windows 旧工具无视 chcp），UTF-8 优先、GBK 兜底
	return DecodeCmdOutput(p.buf.Bytes()), p.done, p.exitErr
}

func (bg *bgRegistry) start(command, dir string) (int, error) {
	bg.mu.Lock()
	bg.next++
	id := bg.next
	p := &bgProc{}
	bg.procs[id] = p
	bg.cleanupLocked() // 顺带清理超龄已完成记录（防长时间运行内存泄漏）
	bg.mu.Unlock()

	c := newShellCommand(command)
	c.Dir = dir
	// ★ 隐藏 cmd 窗口（不弹窗）但不隔离进程组，让子进程能被正常杀死。
	if runtime.GOOS == "windows" {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		c.SysProcAttr.HideWindow = true
	}
	c.Stdout = p
	c.Stderr = p
	p.cmd = c
	if err := c.Start(); err != nil {
		p.mu.Lock()
		p.done, p.exitErr = true, err.Error()
		p.mu.Unlock()
		return 0, err
	}
	go func() {
		err := c.Wait()
		p.mu.Lock()
		p.done = true
		if err != nil {
			p.exitErr = err.Error()
		}
		p.mu.Unlock()
	}()
	return id, nil
}

func (bg *bgRegistry) get(id int) *bgProc {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	return bg.procs[id]
}

func (bg *bgRegistry) cleanupLocked() {
	const keepDone = 24
	var doneIDs []int
	for id, p := range bg.procs {
		if p == nil {
			continue
		}
		p.mu.Lock()
		done := p.done
		p.mu.Unlock()
		if done {
			doneIDs = append(doneIDs, id)
		}
	}
	if len(doneIDs) <= keepDone {
		return
	}
	sort.Ints(doneIDs)
	for _, id := range doneIDs[:len(doneIDs)-keepDone] {
		delete(bg.procs, id)
	}
}

var globalBG = &bgRegistry{procs: map[int]*bgProc{}}

func killProcessTree(pid int) {
	if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run(); err == nil {
		return
	}
	// Unix 兜底
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

// jsTimeoutErr VM 执行超时标记（vm.Interrupt 携带值）。
var jsTimeoutErr = errors.New("JS 执行超时（疑似死循环，已强制中断）")

func runJSWithTimeout(vm *goja.Runtime, timeout time.Duration, fn func() error) error {
	if timeout <= 0 {
		return fn()
	}
	stopped := make(chan struct{})
	go func() {
		select {
		case <-time.After(timeout):
			vm.Interrupt(jsTimeoutErr)
		case <-stopped:
		}
	}()
	err := fn()
	close(stopped)
	vm.ClearInterrupt() // 竞态消除：JS 结束后 goroutine 若已置位 interrupt flag，清除之
	return err
}

// isJSTimeout 判断 err 是否为 runJSWithTimeout 的超时中断。
func isJSTimeout(err error) bool {
	var ie *goja.InterruptedError
	if errors.As(err, &ie) {
		return ie.Value() == jsTimeoutErr
	}
	return errors.Is(err, jsTimeoutErr)
}
