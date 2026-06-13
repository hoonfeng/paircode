package agent

// debug_* 工具集：DAP 调试器。
// 依赖 cmd/companion/debugger 包，通过 dlv dap 协议管理 Go 程序调试会话。
// 支持：启动/停止调试、断点管理、单步执行、栈帧/变量查看、表达式求值。

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/debugger"
)

// ─── 全局调试会话管理器 ───────────────────────────────────────────

var (
	debugMu      sync.Mutex
	debugSession *debugger.DebugSession
)

// registerDebugTools 注册全部调试工具到 registry。
func registerDebugTools(r *Registry, root string) {
	// debug_start — 启动调试会话
	r.Register(&Tool{
		Name: "debug_start",
		Description: "启动 Go 程序调试会话。需要安装 Delve（dlv）。" +
			"program 为 Go 程序路径（如 './cmd/app'）。" +
			"返回调试端口号。启动后可用其他 debug_* 工具控制调试。",
		Parameters: objSchema(props{
			"program": strProp("Go 程序路径，如 './cmd/app' 或 'main.go'"),
			"timeout": intProp("可选：启动超时秒数（默认 30s），dlv 编译大项目可能较慢"),
		}, "program"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			program := argStr(args, "program")
			if program == "" {
				return "", fmt.Errorf("program 不能为空")
			}
			timeout := argInt(args, "timeout", 30)
			if timeout <= 0 {
				timeout = 30
			}

			// 创建新的调试会话
			dlvCmd := "dlv" // 可从配置读取
			s := debugger.NewDebugSession(dlvCmd, program)

			dCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			defer cancel()

			if err := s.Start(dCtx, program); err != nil {
				return "", fmt.Errorf("启动调试会话失败: %w", err)
			}

			// 保存全局会话
			debugMu.Lock()
			if debugSession != nil {
				debugSession.Stop()
			}
			debugSession = s
			debugMu.Unlock()

			return fmt.Sprintf("调试会话已启动（端口: %d，程序: %s）\n"+
				"使用 debug_breakpoint 设置断点，debug_continue 继续执行", s.Port(), program), nil
		},
	})

	// debug_stop — 停止调试会话
	r.Register(&Tool{
		Name:        "debug_stop",
		Description: "停止当前调试会话，关闭 dlv 和 DAP 连接。",
		Parameters:  objSchema(props{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			debugMu.Lock()
			s := debugSession
			debugSession = nil
			debugMu.Unlock()

			if s == nil || !s.IsActive() {
				return "当前没有活跃的调试会话", nil
			}

			if err := s.Stop(); err != nil {
				return "", fmt.Errorf("停止调试会话失败: %w", err)
			}
			return "调试会话已停止", nil
		},
	})

	// getSession 获取当前调试会话（带错误提示）。
	getSession := func() (*debugger.DebugSession, error) {
		debugMu.Lock()
		s := debugSession
		debugMu.Unlock()
		if s == nil || !s.IsActive() {
			return nil, fmt.Errorf("没有活跃的调试会话，请先用 debug_start 启动")
		}
		return s, nil
	}

	// debug_breakpoint — 设置/清除断点
	r.Register(&Tool{
		Name: "debug_breakpoint",
		Description: "在指定文件的指定行设置断点。path 为源文件路径，lines 为行号数组。" +
			"传空 lines 数组将清除该文件的所有断点。" +
			"返回每个断点的验证状态（verified=true 表示断点设置成功）。",
		Parameters: objSchema(props{
			"path":  strProp("源文件路径，如 'main.go' 或 'src/server.go'"),
			"lines": intArrProp("行号数组，如 [10, 25, 42]。空数组=清除该文件断点"),
		}, "path", "lines"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			path := argStr(args, "path")
			if path == "" {
				return "", fmt.Errorf("path 不能为空")
			}
			lines := argIntSlice(args, "lines")

			bps, err := s.SetBreakpoints(path, lines)
			if err != nil {
				return "", fmt.Errorf("设置断点失败: %w", err)
			}

			if len(bps) == 0 {
				return "已清除 " + path + " 的所有断点", nil
			}

			var result string
			verified := 0
			for _, bp := range bps {
				status := "✓"
				if !bp.Verified {
					status = "✗"
				} else {
					verified++
				}
				result += fmt.Sprintf("  %s 行 %d", status, bp.Line)
				if bp.Message != "" {
					result += " — " + bp.Message
				}
				result += "\n"
			}
			return fmt.Sprintf("断点设置完成（%d/%d 已验证）:\n%s", verified, len(bps), result), nil
		},
	})

	// debug_continue — 继续执行
	r.Register(&Tool{
		Name: "debug_continue",
		Description: "从暂停状态继续执行程序（直到下一个断点、异常或程序退出）。" +
			"thread_id 可选，默认 1（主线程）。执行后进入运行状态。",
		Parameters: objSchema(props{
			"thread_id": intProp("可选：线程 ID（默认 1）"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			tid := argInt(args, "thread_id", 1)
			if err := s.Continue(tid); err != nil {
				return "", fmt.Errorf("continue 失败: %w", err)
			}
			return "程序已恢复执行，等待断点命中或退出…", nil
		},
	})

	// debug_next — 单步跳过（Step Over）
	r.Register(&Tool{
		Name: "debug_next",
		Description: "单步跳过（Step Over）：执行当前行，如果当前行是函数调用则不进入函数内部。" +
			"thread_id 可选，默认 1。",
		Parameters: objSchema(props{
			"thread_id": intProp("可选：线程 ID（默认 1）"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			tid := argInt(args, "thread_id", 1)
			if err := s.Next(tid); err != nil {
				return "", fmt.Errorf("next 失败: %w", err)
			}
			return "单步跳过，等待完成…", nil
		},
	})

	// debug_step_in — 单步进入（Step Into）
	r.Register(&Tool{
		Name: "debug_step_in",
		Description: "单步进入（Step Into）：如果当前行是函数调用，进入函数内部。" +
			"thread_id 可选，默认 1。",
		Parameters: objSchema(props{
			"thread_id": intProp("可选：线程 ID（默认 1）"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			tid := argInt(args, "thread_id", 1)
			if err := s.StepIn(tid); err != nil {
				return "", fmt.Errorf("stepIn 失败: %w", err)
			}
			return "单步进入，等待完成…", nil
		},
	})

	// debug_step_out — 单步跳出（Step Out）
	r.Register(&Tool{
		Name: "debug_step_out",
		Description: "单步跳出（Step Out）：执行到当前函数返回。" +
			"thread_id 可选，默认 1。",
		Parameters: objSchema(props{
			"thread_id": intProp("可选：线程 ID（默认 1）"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			tid := argInt(args, "thread_id", 1)
			if err := s.StepOut(tid); err != nil {
				return "", fmt.Errorf("stepOut 失败: %w", err)
			}
			return "单步跳出，等待完成…", nil
		},
	})

	// debug_stack — 查看调用栈
	r.Register(&Tool{
		Name: "debug_stack",
		Description: "查看指定线程的调用栈。" +
			"thread_id 可选（默认 1），levels 可选（默认 20，最大 50）。" +
			"返回栈帧列表（ID、函数名、文件、行号）。",
		Parameters: objSchema(props{
			"thread_id": intProp("可选：线程 ID（默认 1）"),
			"levels":    intProp("可选：最大栈帧数（默认 20，最大 50）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			tid := argInt(args, "thread_id", 1)
			levels := argInt(args, "levels", 20)
			if levels > 50 {
				levels = 50
			}

			frames, err := s.StackTrace(tid, levels)
			if err != nil {
				return "", fmt.Errorf("获取调用栈失败: %w", err)
			}

			if len(frames) == 0 {
				return "（调用栈为空，程序可能未暂停）", nil
			}

			result := fmt.Sprintf("调用栈（线程 %d, %d 帧）:\n", tid, len(frames))
			for i, f := range frames {
				src := f.Source.Path
				if src == "" {
					src = f.Source.Name
				}
				marker := ""
				if i == 0 {
					marker = " ← 当前"
				}
				if src != "" {
					result += fmt.Sprintf("  #%d %s (%s:%d)%s\n", i, f.Name, src, f.Line, marker)
				} else {
					result += fmt.Sprintf("  #%d %s%s\n", i, f.Name, marker)
				}
			}
			return result, nil
		},
	})

	// debug_variables — 查看变量
	r.Register(&Tool{
		Name: "debug_variables",
		Description: "查看当前暂停点的变量值。可指定栈帧 ID 查看不同帧的变量。" +
			"frame_id 可选（默认 0 = 当前帧，即栈顶）。" +
			"返回变量名、类型、值的列表。复杂对象可通过 variables_reference 展开子变量。",
		Parameters: objSchema(props{
			"frame_id":            intProp("可选：栈帧 ID（默认 0 = 当前帧）"),
			"variables_reference": intProp("可选：展开子变量（从之前输出中的 variables_reference 值传入）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}

			ref := argInt(args, "variables_reference", 0)
			frameID := argInt(args, "frame_id", 0)

			// 如果没有指定 variables_reference，先从栈帧获取变量引用
			if ref == 0 {
				frames, err := s.StackTrace(1, 1)
				if err != nil {
					return "", fmt.Errorf("获取当前帧失败: %w", err)
				}
				if len(frames) == 0 {
					return "（程序未暂停，无法获取变量）", nil
				}
				if frameID >= len(frames) {
					return "", fmt.Errorf("栈帧 %d 超出范围（共 %d 帧）", frameID, len(frames))
				}
				// 使用栈帧 ID 作为 scope 的 variablesReference
				scopes, err := s.GetScopes(frames[frameID].ID)
				if err != nil {
					return "", fmt.Errorf("获取作用域失败: %w", err)
				}
				if len(scopes) == 0 {
					return "（当前帧无变量作用域）", nil
				}
				ref = scopes[0].VariablesReference
			}

			vars, err := s.Variables(ref)
			if err != nil {
				return "", fmt.Errorf("获取变量失败: %w", err)
			}

			if len(vars) == 0 {
				return "（无变量）", nil
			}

			result := fmt.Sprintf("变量（%d 个）:\n", len(vars))
			for _, v := range vars {
				vr := ""
				if v.VariablesReference > 0 {
					vr = fmt.Sprintf(" [ref=%d]", v.VariablesReference)
				}
				result += fmt.Sprintf("  %s = %v (%s)%s\n", v.Name, v.Value, v.Type, vr)
			}
			return result, nil
		},
	})

	// debug_evaluate — 表达式求值
	r.Register(&Tool{
		Name: "debug_evaluate",
		Description: "在调试暂停状态下求值表达式。可查看变量值、调用函数、计算表达式。" +
			"expression 为要求值的表达式（如 'len(x)'、'fmt.Sprintf(\"%v\", s)'）。" +
			"frame_id 可选（默认 0 = 当前帧）。返回表达式的结果值和类型。",
		Parameters: objSchema(props{
			"expression": strProp("要求值的表达式，如 'len(list)'、'x == nil'、'fmt.Sprintf(\"%v\", result)'"),
			"frame_id":   intProp("可选：栈帧 ID（默认 0 = 当前帧）"),
		}, "expression"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			s, err := getSession()
			if err != nil {
				return "", err
			}
			expr := argStr(args, "expression")
			if expr == "" {
				return "", fmt.Errorf("expression 不能为空")
			}
			frameID := argInt(args, "frame_id", 0)

			resp, err := s.Evaluate(expr, frameID)
			if err != nil {
				return "", fmt.Errorf("求值失败: %w", err)
			}

			result := fmt.Sprintf("表达式: %s\n结果: %s\n类型: %s", expr, resp.Result, resp.Type)
			if resp.VariablesReference > 0 {
				result += fmt.Sprintf("\n（可展开，ref=%d）", resp.VariablesReference)
			}
			return result, nil
		},
	})

	// debug_status — 调试会话状态
	r.Register(&Tool{
		Name:        "debug_status",
		Description: "查看当前调试会话状态（空闲/运行中/已暂停/已退出）以及断点数量。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			debugMu.Lock()
			s := debugSession
			debugMu.Unlock()

			if s == nil {
				return "没有活跃的调试会话", nil
			}

			state := s.State()
			bps := s.Breakpoints()
			verified := 0
			for _, bp := range bps {
				if bp.Verified {
					verified++
				}
			}

			stateDesc := ""
			switch state {
			case debugger.StateIdle:
				stateDesc = "未启动"
			case debugger.StateInitialized:
				stateDesc = "已初始化"
			case debugger.StateRunning:
				stateDesc = "运行中"
			case debugger.StatePaused:
				stateDesc = fmt.Sprintf("已暂停（原因: %s）", debugger.FormatStopReason(s))
			case debugger.StateExited:
				stateDesc = "已退出"
			default:
				stateDesc = string(state)
			}

			return fmt.Sprintf("调试会话状态: %s\n端口: %d\n断点: %d（%d 已验证）\n程序: %s",
				stateDesc, s.Port(), len(bps), verified, s.Program()), nil
		},
	})
}

// ─── 辅助 ─────────────────────────────────────────────────────────

// intArrProp 生成整数数组类型的 JSON Schema 属性。
func intArrProp(desc string) map[string]any {
	return map[string]any{
		"type":        "array",
		"description": desc,
		"items":       map[string]any{"type": "integer"},
	}
}

// argIntSlice 从参数中提取整数切片。
func argIntSlice(args map[string]any, key string) []int {
	raw, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(raw))
	for _, v := range raw {
		switch n := v.(type) {
		case float64:
			out = append(out, int(n))
		case int:
			out = append(out, n)
		}
	}
	return out
}
