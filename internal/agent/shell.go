// 后台命令工具：run_background / read_output / kill_process —— 后台跑长命令(dev server/watch)不阻塞 agent 循环。
// Windows: cmd /C(同 run_command,UTF-8)；输出经 io.Writer 累积到带锁缓冲(有尾部上限防撑爆内存)。
// 注意：进程在 app 退出时不自动清理，agent 用完应自行 kill_process（健壮的 job-object 清理留后续）。

package agent

import (
	"bytes"
	"context"
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
)

// ── 内置 bash 执行（Git Bash 资源，随 release 分发）──
// 优先使用项目自带的内置 bash（bin/bash/usr/bin/bash.exe）：
//   - POSIX 语法 + UTF-8 输出，LLM 一次成功率远高于 cmd/PowerShell（引号/&/编码坑少）；
//   - 回退链：内置 bash → 系统 Git Bash → PATH 中的 bash → cmd（原逻辑兜底）。
var (
	detectedBashOnce sync.Once
	detectedBashPath string
	detectedMsysBin  string
)

// bashCandidate 计算 exe 同目录内置 bash 候选路径。
func bashCandidate(exePath string) string {
	return filepath.Join(filepath.Dir(exePath), "bin", "bash", "usr", "bin", "bash.exe")
}

// detectBash 探测可用 bash，返回 bash 可执行文件路径与其 msys bin 目录（PATH 前缀用）。
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

// newShellCommand 构造 shell 命令：
//   - bash 可用 → bash -c（POSIX 语法 + UTF-8，msys bin 前置 PATH 使 ls/grep 等可用）
//   - 否则 → cmd /C chcp 65001 + 命令（原逻辑兜底）
func newShellCommand(command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.Command(bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return c
	}
	return exec.Command("cmd", "/C", "chcp 65001 >nul & "+command)
}

// newShellCommandContext 带 ctx 的版本（超时/取消）。
func newShellCommandContext(ctx context.Context, command string) *exec.Cmd {
	if bashPath, msysBin := detectBash(); bashPath != "" {
		c := exec.CommandContext(ctx, bashPath, "-c", command)
		applyBashEnv(c, msysBin)
		return c
	}
	return exec.CommandContext(ctx, "cmd", "/C", "chcp 65001 >nul & "+command)
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
type bgProc struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	buf     bytes.Buffer
	done    bool
	exitErr string
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
	return decodeCmdOutput(p.buf.Bytes()), p.done, p.exitErr
}

// bgRegistry 后台进程注册表（并发安全；全局单例 globalBG，跨 agent 轮次存活）。
// ★ globalBG 必须为包级单例：Registry 在每次发消息/每轮对话都会重建（web_server.go
//   buildWebLoopOpts 调 RegisterDefaultTools 新建 Registry），若 bgRegistry 随之重建，
//   上一轮 run_background 启动的进程（含已结束进程的输出缓冲）将在下一轮丢失——
//   read_output 读不到、kill_process 找不到 id。提升为全局后进程与输出跨轮保留。
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

func registerShellTools(r *Registry, bg *bgRegistry, root string) {
	// run_background / read_output / kill_process — 3 个后台命令工具共享同一份 bgRegistry。
	r.Register(&Tool{
		Name: "run_background",
		UsageGuide: "后台启动一条长命令，不阻塞 agent 循环。用于 dev server、npm run dev/watch 模式、调试服务、TCP 监听——这些场景只能用此工具，不可用 run_command。返回进程 id，之后用 read_output/kill_process 控制。比 run_command 更合适的长命令：run_background（不阻塞）+ read_output（分阶段读）+ kill_process（手动停止）。",
		Description: "在后台启动一条长命令，不阻塞 agent 循环（推荐用于 dev server、watch 模式、调试服务等）。" +
			"返回进程 id，随后用 read_output 读输出、kill_process 停止。" +
			"如果命令会长期运行或保持监听状态，优先用此工具。短查询请用 run_command。",
		Parameters: objSchema(props{"command": strProp("要后台执行的命令"), "cwd": strProp("可选工作目录（工作区内）")}, "command"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := strings.TrimSpace(argStr(args, "command"))
			if command == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			// ★ 信号监听已移除：不再阻止杀死自身进程或运行 companion
			dir := root
			if cwd := argStr(args, "cwd"); cwd != "" {
				var err error
				if dir, err = resolvePath(root, cwd); err != nil {
					return "", err
				}
			}
			id, err := bg.start(command, dir)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("已后台启动 id=%d。用 read_output(id=%d) 看输出、kill_process(id=%d) 停止。", id, id, id), nil
		},
	})

	r.Register(&Tool{
		Name:        "read_output",
		UsageGuide:  "读取后台进程的累积输出与运行状态。需先用 run_background 启动进程获得 id。比直接看终端更方便（自动截断保护+状态标记运行中/已结束）。",
		Description: "读取某后台进程（id）累积的输出与运行状态（运行中/已结束）。",
		Parameters:  objSchema(props{"id": intProp("进程 id")}, "id"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p := bg.get(argInt(args, "id", -1))
			if p == nil {
				return "", fmt.Errorf("无此后台进程 id")
			}
			out, done, exitErr := p.snapshot()
			status := "运行中"
			if done {
				status = "已结束"
				if exitErr != "" {
					status += "（" + exitErr + "）"
				}
			}
			return fmt.Sprintf("[%s]\n%s", status, capOutput(out, 16000)), nil
		},
	})

	r.Register(&Tool{
		Name:        "kill_process",
		UsageGuide:  "停止某后台进程（仅限通过 run_background 启动的）。进程跑偏/卡死/已不需要时用此工具停止。比 taskkill / pid 更方便（通过 id 直接操作）。",
		Description: "停止某后台进程（id）。只能杀死通过 run_background 启动的进程，无法操作外部进程。",
		Parameters:  objSchema(props{"id": intProp("进程 id")}, "id"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argInt(args, "id", -1)
			p := bg.get(id)
			if p == nil {
				return "", fmt.Errorf("无此后台进程 id")
			}
			// 已结束的进程也允许调用（kill 幂等，用于收尾确认）
			if p.cmd != nil && p.cmd.Process != nil {
				killProcessTree(p.cmd.Process.Pid)
			}
			return fmt.Sprintf("已停止 id=%d", id), nil
		},
	})
}

// killProcessTree 杀进程树（Windows: taskkill /T；Unix: 进程组 SIGKILL）。
func killProcessTree(pid int) {
	if err := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run(); err == nil {
		return
	}
	// Unix 兜底
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}
