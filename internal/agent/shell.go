// 会话式命令执行：exec_command / write_stdin / kill_process —— 对齐 codex unified exec
// 模型（同步等待 → yield 让出 → 会话续写）。bgRegistry 跨轮次存活（globalBG 单例）。
// Windows: bash 优先（Git Bash，UTF-8）、cmd /C 兜底；输出经 io.Writer 累积到带锁
// 缓冲（尾部上限防撑爆内存）+ 增量游标（write_stdin 读新输出段）。
// 注意：库层不自动清理（agent 用完应自行 kill_process）；宿主退出由
// cmd/companion/main.go 退出钩子调用 KillAllBackgroundProcesses() 统一收口
// （2026-09-16：补「子进程泄漏」缺口，退出路径 10s 内无残留）。

package agent

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"io"
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
)

// ── 内置 bash 执行（Git Bash 资源）──
// ★ 2026-09 Round3 ③.4：自带的 bin/bash（Git Bash 精简版，约 28MB）已移除
//   （打包体积瘦身；详见 packager.json dist.include 与仓库 bin/bash 删除）。
//   bash 服务仍保留（fs-api/git-api 的 ctx.bash 依赖）：探测链为
//   系统 Git Bash → PATH 中的 bash → cmd（原逻辑兜底）。
var (
	detectedBashOnce sync.Once
	detectedBashPath string
	detectedMsysBin  string
)

// detectBash 探测可用 bash，返回 bash 可执行文件路径与其 msys bin 目录（PATH 前缀用）。
func detectBash() (bashPath, msysBin string) {
	detectedBashOnce.Do(func() {
		// 1. 系统 Git Bash（首选）
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
		// 2. PATH 中的 bash（msys2/WSL 等，最后兜底）
		if p, err := exec.LookPath("bash"); err == nil {
			detectedBashPath = p
		}
	})
	return detectedBashPath, detectedMsysBin
}

// hideShellWindow 隐藏子进程控制台窗口（Windows；非 Windows 原样返回）。
// 所有 shell 子进程统一调用——父进程无控制台（后台/服务方式启动）时，
// cmd.exe/bash.exe 等 console 程序会自己弹出控制台窗口，必须显式隐藏。
func hideShellWindow(c *exec.Cmd) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if c.SysProcAttr == nil {
			c.SysProcAttr = &syscall.SysProcAttr{}
		}
		executil.HideWindow(c)
	}
	return c
}

// newShellCommand 构造 shell 命令：
//   - bash 可用 → bash -c（POSIX 语法 + UTF-8，msys bin 前置 PATH 使 ls/grep 等可用）
//   - 否则 → cmd /C chcp 65001 + 命令（原逻辑兜底）
func newShellCommand(command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.Command(bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return hideShellWindow(c)
	}
	return hideShellWindow(exec.Command("cmd", "/C", "chcp 65001 >nul & "+command))
}

// newShellCommandContext 带 ctx 的版本（超时/取消）。
func newShellCommandContext(ctx context.Context, command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.CommandContext(ctx, bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return hideShellWindow(c)
	}
	return hideShellWindow(exec.CommandContext(ctx, "cmd", "/C", "chcp 65001 >nul & "+command))
}

// applyBashEnv msys bin 前置 PATH：非登录 shell 不读 /etc/profile，须显式补 PATH
// 才能用 ls/cat/grep 等 msys 工具（Windows 程序 git/go/python 已在原 PATH 中）。
func applyBashEnv(c *exec.Cmd, msysBin string) {
	if msysBin == "" {
		return
	}
	c.Env = append(os.Environ(), "PATH="+msysBin+";"+os.Getenv("PATH"))
}

// bgProc 一个后台进程：cmd + 带锁输出缓冲 + 结束状态。实现 io.Writer 供 exec 直接写。
// ★ 2026-09 会话式执行（exec_command/write_stdin）：stdin 管道可写、输出带增量游标、
// doneCh 提供精确结束通知（wait 不再轮询）。
type bgProc struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser // ★ stdin 管道（会话式交互写入；nil = 不可写）
	mu      sync.Mutex
	buf     bytes.Buffer
	readOff int // ★ 增量读取游标：write_stdin 每次从上次读完处继续
	done    bool
	exitErr string
	doneCh  chan struct{} // ★ 结束通知（Wait 完成时 close）
}

func (p *bgProc) Write(b []byte) (int, error) {
	p.mu.Lock()
	p.buf.Write(b)
	const cap_ = 256 * 1024 // 防长跑进程输出无限增长：超限只留尾部
	if p.buf.Len() > cap_ {
		data := p.buf.Bytes()
		drop := len(data) - 192*1024
		tail := append([]byte(nil), data[drop:]...)
		p.buf.Reset()
		p.buf.Write(tail)
		// 缓冲前移 → 游标同步回退（保证增量语义不越界/不重复）
		if p.readOff -= drop; p.readOff < 0 {
			p.readOff = 0
		}
	}
	p.mu.Unlock()
	return len(b), nil
}

func (p *bgProc) snapshot() (out string, done bool, exitErr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 编码探测：子进程输出可能是 GBK（Windows 旧工具无视 chcp），UTF-8 优先、GBK 兜底
	return decodeCmdOutput(p.buf.Bytes()), p.done, p.exitErr
}

// writeStdin 向进程 stdin 写入字符（会话式执行 write_stdin；chars 空 = 不写只轮询）。
func (p *bgProc) writeStdin(chars string) error {
	p.mu.Lock()
	w := p.stdin
	done := p.done
	p.mu.Unlock()
	if chars == "" {
		return nil
	}
	if done {
		return fmt.Errorf("进程已结束，stdin 不可写")
	}
	if w == nil {
		return fmt.Errorf("该进程 stdin 不可写")
	}
	_, err := io.WriteString(w, chars)
	return err
}

// readNew 返回自上次读取以来的新输出（增量；游标随读前进）。snapshot 不受影响（全量）。
func (p *bgProc) readNew() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	data := p.buf.Bytes()
	if p.readOff > len(data) {
		p.readOff = len(data)
	}
	seg := data[p.readOff:]
	p.readOff = len(data)
	return decodeCmdOutput(seg)
}

// waitDone 等待进程结束最多 timeoutMs 毫秒（0=立即返回当前状态）。
func (p *bgProc) waitDone(timeoutMs int) bool {
	if p == nil {
		return true
	}
	if timeoutMs <= 0 {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.done
	}
	select {
	case <-p.doneCh:
		return true
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return false
	}
}

// exitCode 返回退出码（-1 = 未知）。
func (p *bgProc) exitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.ProcessState != nil {
		return p.cmd.ProcessState.ExitCode()
	}
	return -1
}

// bgRegistry 后台进程注册表（并发安全；全局单例 globalBG，跨 agent 轮次存活）。
// ★ globalBG 必须为包级单例：Registry 在每次发消息/每轮对话都会重建（web_server.go
//
//	buildWebLoopOpts 调 RegisterDefaultTools 新建 Registry），若 bgRegistry 随之重建，
//	上一轮 run_background 启动的进程（含已结束进程的输出缓冲）将在下一轮丢失——
//	read_output 读不到、kill_process 找不到 id。提升为全局后进程与输出跨轮保留。
var globalBG = &bgRegistry{procs: map[int]*bgProc{}}

type bgRegistry struct {
	mu    sync.Mutex
	procs map[int]*bgProc
	next  int
}

func (bg *bgRegistry) start(command, dir string) (int, error) {
	bg.mu.Lock()
	bg.next++
	id := bg.next
	p := &bgProc{doneCh: make(chan struct{})}
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
		executil.HideWindow(c)
	}
	stdin, err := c.StdinPipe() // ★ 会话式：stdin 开放给 write_stdin（不读 stdin 的命令不受影响）
	if err != nil {
		p.mu.Lock()
		p.done, p.exitErr = true, err.Error()
		p.mu.Unlock()
		close(p.doneCh)
		return 0, err
	}
	c.Stdout = p
	c.Stderr = p
	p.cmd = c
	p.stdin = stdin
	if err := c.Start(); err != nil {
		p.mu.Lock()
		p.done, p.exitErr = true, err.Error()
		p.mu.Unlock()
		close(p.doneCh)
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
		close(p.doneCh)
	}()
	return id, nil
}

func (bg *bgRegistry) get(id int) *bgProc {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	return bg.procs[id]
}

// list 列出全部后台进程（id + 状态 + 退出错误；会话清单/诊断用）。
func (bg *bgRegistry) list() []map[string]any {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	ids := make([]int, 0, len(bg.procs))
	for id := range bg.procs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		p := bg.procs[id]
		if p == nil {
			continue
		}
		p.mu.Lock()
		done := p.done
		exitErr := p.exitErr
		p.mu.Unlock()
		status := "running"
		if done {
			status = "done"
			if exitErr != "" {
				status = "error"
			}
		}
		entry := map[string]any{"id": id, "status": status}
		if done && exitErr != "" {
			entry["error"] = exitErr
		}
		out = append(out, entry)
	}
	return out
}

// cleanupLocked 清理已完成且超龄的进程记录（调用方须持有 bg.mu）。
// 仅保留最近 keepDone 个已结束进程（其输出缓冲仍可读），运行中的永不清理。
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

// ─── 会话式命令执行工具（exec 组，2026-09 工具重构）────────
// ★ 对齐 codex unified exec 模型：exec_command 一个工具覆盖「同步等待
//   + 长命令让出 + 会话复用」；write_stdin 轮询增量输出/写 stdin；
//   kill_process 终止。取代 run_background/read_output/job_list
//   （异步轮询式：无同步等待/无 stdin/无增量语义）。
//   生产面同名工具由 tool-exec 磁盘插件承载（.pair/plugins/tool-exec → ctx.process）。

// clampExecYield yield 毫秒钳制（默认 10000，范围 250-30000）。
func clampExecYield(v int) int {
	if v <= 0 {
		return 10000
	}
	if v < 250 {
		return 250
	}
	if v > 30000 {
		return 30000
	}
	return v
}

// capExecOutput token 预算截断（1 token ≈ 4 字符；超限保留尾部，前部给省略提示）。
func capExecOutput(s string, maxTokens int) string {
	tokens := maxTokens
	if tokens <= 0 {
		tokens = 10000
	}
	budget := tokens * 4
	if len(s) <= budget {
		return s
	}
	return fmt.Sprintf("…[前部输出已省略 %d 字符]\n%s", len(s)-budget, s[len(s)-budget:])
}

// execStatusLine 生成会话状态行（与 tool-exec 插件同格式）。
func execStatusLine(sessionID int, done bool, exitCode int, exitErr string, hint bool) string {
	if !done {
		if hint {
			return fmt.Sprintf("[运行中 session_id=%d — 用 write_stdin 轮询输出/写 stdin，kill_process 停止]", sessionID)
		}
		return fmt.Sprintf("[运行中 session_id=%d]", sessionID)
	}
	code := "退出码未知"
	if exitCode >= 0 {
		code = fmt.Sprintf("退出码=%d", exitCode)
	}
	if exitErr != "" {
		return fmt.Sprintf("[已结束 %s · %s]", code, exitErr)
	}
	return fmt.Sprintf("[已结束 %s]", code)
}

// registerShellTools 注册会话式命令执行工具（exec_command/write_stdin/kill_process，
// 共享同一份 bgRegistry）。库侧实现（独立宿主/测试）；生产同源工具在 tool-exec 插件。
func registerShellTools(r *Registry, bg *bgRegistry, root string) {
	r.Register(&Tool{
		Name:        "exec_command",
		UsageGuide:  "统一 shell 执行：短命令（git/构建/测试/文件查询）同步返回结果；长进程（dev server/watch/TCP 监听）等待 yield_time_ms 后返回 session_id，用 write_stdin 轮询输出/写 stdin、kill_process 终止。原生 shell 语法（bash 优先、cmd 兜底），比 run_code 包装更直接。",
		Description: "执行一条 shell 命令。yield_time_ms 内完成返回全量输出与退出码；超时仍在运行返回 session_id（会话可续：write_stdin 轮询/交互，kill_process 停止）。",
		Parameters: objSchema(props{
			"command":           strProp("要执行的 shell 命令（bash 语法；无 bash 时 cmd 兜底）"),
			"workdir":           strProp("可选：工作目录（工作区内，省略=工作区根）"),
			"yield_time_ms":     intProp("可选：等待毫秒（默认 10000，范围 250-30000）"),
			"max_output_tokens": intProp("可选：输出 token 预算（默认 10000，超限保留尾部）"),
			"project":           projectSchemaProp(),
		}, "command"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := strings.TrimSpace(argStr(args, "command"))
			if command == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			dir := root
			if wd := argStr(args, "workdir"); wd != "" {
				var err error
				if dir, err = resolvePathFor(root, args, wd); err != nil {
					return "", err
				}
			}
			yieldMs := clampExecYield(argInt(args, "yield_time_ms", 10000))
			id, err := bg.start(command, dir)
			if err != nil {
				return "", err
			}
			proc := bg.get(id)
			if proc == nil {
				return "", fmt.Errorf("内部错误：进程创建后丢失")
			}
			// 等待期间监听 ctx 取消：轮次中止时杀掉本进程（未转会话的半启动状态），
			// 不让 yield 阻塞后续处理（已返回 session_id 的进程由 agent 显式 kill）。
			var done bool
			waitCh := make(chan bool, 1)
			go func() { waitCh <- proc.waitDone(yieldMs) }()
			select {
			case done = <-waitCh:
			case <-ctx.Done():
				if proc.cmd != nil && proc.cmd.Process != nil {
					killProcessTree(proc.cmd.Process.Pid)
				}
				body := capExecOutput(proc.readNew(), argInt(args, "max_output_tokens", 0))
				if strings.TrimSpace(body) == "" {
					body = "（无输出）"
				}
				return body + "\n[已中断: " + ctx.Err().Error() + "]", nil
			}
			out := proc.readNew()
			_, _, exitErr := proc.snapshot()
			body := capExecOutput(out, argInt(args, "max_output_tokens", 0))
			if strings.TrimSpace(body) == "" {
				body = "（无输出）"
			}
			return body + "\n" + execStatusLine(id, done, proc.exitCode(), exitErr, true), nil
		},
	})

	r.Register(&Tool{
		Name:        "write_stdin",
		UsageGuide:  "向 exec_command 的会话进程写 stdin（chars 空 = 仅轮询）并读取增量输出（自上次读取起）。交互式进程用 chars 写入输入；长进程轮询新输出用空 chars。等待 yield_time_ms 后返回。",
		Description: "向会话进程（session_id）写入 stdin 并返回自上次读取以来的增量输出；chars 省略 = 仅轮询。",
		Parameters: objSchema(props{
			"session_id":        intProp("会话 id（exec_command 返回）"),
			"chars":             strProp("可选：写入 stdin 的字符（省略/空 = 仅轮询输出）"),
			"yield_time_ms":     intProp("可选：等待毫秒（写入默认 250；轮询默认 5000）"),
			"max_output_tokens": intProp("可选：输出 token 预算（默认 10000）"),
		}, "session_id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argInt(args, "session_id", -1)
			proc := bg.get(id)
			if proc == nil {
				return "", fmt.Errorf("无此会话 id=%d", id)
			}
			chars := argStr(args, "chars")
			if err := proc.writeStdin(chars); err != nil {
				return "", err
			}
			yieldMs := argInt(args, "yield_time_ms", 0)
			if yieldMs <= 0 {
				if chars == "" {
					yieldMs = 5000
				} else {
					yieldMs = 250
				}
			}
			done := proc.waitDone(yieldMs)
			out := proc.readNew()
			_, _, exitErr := proc.snapshot()
			body := capExecOutput(out, argInt(args, "max_output_tokens", 0))
			if strings.TrimSpace(body) == "" {
				if chars != "" {
					body = "（已写入 stdin，无新输出）"
				} else {
					body = "（无新输出）"
				}
			}
			return body + "\n" + execStatusLine(id, done, proc.exitCode(), exitErr, false), nil
		},
	})

	r.Register(&Tool{
		Name:        "kill_process",
		UsageGuide:  "终止 exec_command 的会话进程（dev server/watch/卡死命令）。仅限通过 exec_command 启动的会话；已结束的会话调用无害（幂等收尾）。",
		Description: "终止会话进程（session_id）。已结束会话调用无害（幂等）。",
		Parameters:  objSchema(props{"session_id": intProp("会话 id（exec_command 返回）")}, "session_id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argInt(args, "session_id", -1)
			proc := bg.get(id)
			if proc == nil {
				return "", fmt.Errorf("无此会话 id=%d", id)
			}
			// 已结束的进程也允许调用（kill 幂等，用于收尾确认）
			if proc.cmd != nil && proc.cmd.Process != nil {
				killProcessTree(proc.cmd.Process.Pid)
			}
			return fmt.Sprintf("已停止 session_id=%d", id), nil
		},
	})
}

// killProcessTree 杀进程树（Windows: taskkill /T；Unix: 进程组 SIGKILL）。
func killProcessTree(pid int) {
	if err := hideShellWindow(exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid))).Run(); err == nil {
		return
	}
	// Unix 兜底
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

// KillAllBackgroundProcesses 终止全部后台进程（宿主退出清理；cmd/companion/main.go
// 退出钩子调用）。覆盖所有经 runBackground / exec_command 启动的进程——agent 的
// dev server、插件后台进程等，堵住「子进程孤儿化残留」缺口。
// 幂等：无进程时无副作用；已结束的进程跳过（不做 PID 复用误杀）。
func KillAllBackgroundProcesses() {
	globalBG.mu.Lock()
	procs := make([]*bgProc, 0, len(globalBG.procs))
	for _, p := range globalBG.procs {
		if p != nil {
			procs = append(procs, p)
		}
	}
	globalBG.mu.Unlock()
	for _, p := range procs {
		p.mu.Lock()
		done := p.done
		cmd := p.cmd
		p.mu.Unlock()
		if done || cmd == nil || cmd.Process == nil {
			continue
		}
		killProcessTree(cmd.Process.Pid)
	}
}
