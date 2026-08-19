// ═══════════════════════════════════════════════════════════════
// exec.go — ui-quick-exec 工具实现（run：shell 命令执行，可配超时）
//
// ★ 2026-08-17：从 ctx.bash.exec（宿主 runShellWithTimeout 120s 硬编码
// 超时）迁移——打包类长命令（>120s）原会被强制 kill。现走本二进制，
// 超时由 toolbin.Serve 的 args.timeoutMs 精确控制（默认 600s，
// 超时 kill 并返回 timedOut=true），完全不经宿主 120s 桥接。
// ═══════════════════════════════════════════════════════════════
package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/ui-quick-exec/toolbin"
)

// Register 注册 ui-quick-exec 工具组。
func Register(reg *toolbin.Registry, root string) {
	reg.Register(&toolbin.Tool{
		Name:        "run",
		Description: "在工作区目录执行 shell 命令（超时可配：0 = 不超时，缺省 600s），返回完整输出/退出码/耗时",
		Category:    "exec",
		Parameters: toolbin.ObjSchema(toolbin.Props{
			"command":   toolbin.StrProp("要执行的 shell 命令（Windows cmd 语法）"),
			"cwd":       toolbin.StrProp("工作目录（缺省 = 工作区根）"),
			"timeoutMs": toolbin.IntProp("超时毫秒（缺省 600000 = 600 秒；0 = 不超时；超时强制结束并返回 timedOut=true）"),
		}, "command"),
		Handler: runCommand(root),
	})
	reg.Register(&toolbin.Tool{
		Name:        "ping",
		Description: "连通性探测（返回 ok）",
		Category:    "exec",
		Parameters:  toolbin.ObjSchema(toolbin.Props{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return "ok (ui-quick-exec v1)", nil
		},
	})
}

// runCommand 执行 shell 命令（cmd /c），合并输出；超时由传入 ctx 控制
// （toolbin.Serve 已按 args.timeoutMs 建好超时上下文）。
func runCommand(root string) toolbin.ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		command := toolbin.ArgStr(args, "command")
		if command == "" {
			return "", fmt.Errorf("command 不能为空")
		}
		cwd := toolbin.ArgStr(args, "cwd")
		if cwd == "" {
			cwd = root
		}

		// ★ 不用 CommandContext：其 ctx 取消时先杀 cmd 父进程，taskkill /T
		// 会因父进程已死而找不到进程树，子进程（如打包的子 exe）残留。
		// 改用手动管理：goroutine 监听 ctx.Done()，趁 cmd 还活着先 taskkill /T
		// 杀整棵进程树，再 Kill 兜底。
		cmd := exec.Command("cmd", "/c", command)
		// 隐藏子进程控制台窗口（无控制台父进程时 console 程序会自己弹窗）
		if runtime.GOOS == "windows" {
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		cmd.Dir = cwd
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("启动命令失败: %v", err)
		}
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if cmd.Process != nil {
					_ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
					_ = cmd.Process.Kill()
				}
			case <-done:
			}
		}()
		start := time.Now()
		err := cmd.Wait()
		close(done)
		duration := time.Since(start)

		result := map[string]any{
			"output":     toolbin.DecodeCmdOutput(buf.Bytes()),
			"exitCode":   0,
			"timedOut":   false,
			"durationMs": duration.Milliseconds(),
		}
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				result["timedOut"] = true
				result["exitCode"] = -1
			} else if ee, ok := err.(*exec.ExitError); ok {
				result["exitCode"] = ee.ExitCode()
			} else {
				return "", fmt.Errorf("执行失败: %v", err)
			}
		}
		// 防大输出撑爆 JSON/弹窗（保头 3/4 + 尾 1/4，200KB 上限）
		result["output"] = toolbin.CapOutput(result["output"].(string), 200*1024)
		b, _ := json.Marshal(result)
		return string(b), nil
	}
}
