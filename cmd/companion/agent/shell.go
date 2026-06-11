// 后台命令工具：run_background / read_output / kill_process —— 后台跑长命令(dev server/watch)不阻塞 agent 循环。
// Windows: cmd /C(同 run_command,UTF-8)；输出经 io.Writer 累积到带锁缓冲(有尾部上限防撑爆内存)。
// 注意：进程在 app 退出时不自动清理，agent 用完应自行 kill_process（健壮的 job-object 清理留后续）。

package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

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
	return p.buf.String(), p.done, p.exitErr
}

// bgRegistry 后台进程注册表（并发安全；每个 Registry 一份，跨 agent 轮次存活）。
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
	bg.mu.Unlock()

	c := exec.Command("cmd", "/C", "chcp 65001 >nul & "+command)
	c.Dir = dir
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

// registerShellTools 注册后台命令工具（3 个工具共享一份 bgRegistry）。
func registerShellTools(r *Registry, root string) {
	bg := &bgRegistry{procs: map[int]*bgProc{}}

	r.Register(&Tool{
		Name: "run_background",
		Description: "在后台启动一条长命令（如 dev server / watch），立即返回进程 id、不阻塞循环。" +
			"随后用 read_output 读输出、kill_process 停止。短命令请用 run_command。",
		Parameters:       objSchema(props{"command": strProp("要后台执行的命令"), "cwd": strProp("可选工作目录（工作区内）")}, "command"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := strings.TrimSpace(argStr(args, "command"))
			if command == "" {
				return "", fmt.Errorf("command 不能为空")
			}
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
		Name:             "kill_process",
		Description:      "停止某后台进程（id）。",
		Parameters:       objSchema(props{"id": intProp("进程 id")}, "id"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			id := argInt(args, "id", -1)
			p := bg.get(id)
			if p == nil {
				return "", fmt.Errorf("无此后台进程 id")
			}
			if p.cmd != nil && p.cmd.Process != nil {
				p.cmd.Process.Kill()
			}
			return fmt.Sprintf("已停止 id=%d", id), nil
		},
	})
}
