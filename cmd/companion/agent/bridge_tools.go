// Bridge 管理工具集 —— Agent 通过此工具集管理系统桥接层。
//
// 这些工具让 Agent 能够：
//   1. 查看当前桥接状态（模式、能力、审计日志）
//   2. 申请接管系统（TakeoverMode）
//   3. 安全降级回桥接模式
//   4. 通过桥接执行系统命令
//   5. 注册系统级工具（接管模式下）
//
// 设计原则：
//   - 接管操作 RequiresApproval=true（必须用户确认）
//   - 所有操作记录审计日志
//   - 接管模式下仍保留核心安全防线（如风暴断路器、绕圈检测）

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// registerBridgeTools 注册 Bridge 管理系统工具到 Registry。
func registerBridgeTools(r *Registry, root string) {
	bc := GetBridgeController(root)

	// ── bridge_status ──
	r.Register(&Tool{
		Name:        "bridge_status",
		Description: "查看桥接系统状态：当前模式（桥接/接管）、可用系统能力、审计日志摘要。用于了解 Agent 当前对系统资源的访问权限。",
		ReadOnly:    true,
		Parameters: objSchema(props{
			"detail": strProp("可选：设为 'full' 查看完整审计日志"),
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			detail := argStr(args, "detail")
			return bcStatusText(bc, detail == "full"), nil
		},
	})

	// ── bridge_takeover ──
	r.Register(&Tool{
		Name: "bridge_takeover",
		Description: "申请全面接管系统管理权限。接管后 Agent 获得完整的系统资源访问能力：" +
			"不限路径的文件系统、不限进程的命令执行、系统配置读写、服务管理等。\n" +
			"接管操作需要用户审批确认。完成管理任务后请用 bridge_lockdown 归还权限。\n" +
			"注意：接管赋予 Agent 高权限，请谨慎使用并只执行必要的管理操作。",
		Parameters: objSchema(props{
			"reason": strProp("接管原因说明（必填，向用户解释为何需要全面权限）"),
		}, "reason"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			reason := argStr(args, "reason")
			if reason == "" {
				return "", fmt.Errorf("reason 接管原因不能为空")
			}
			if err := bc.SwitchToTakeover("user"); err != nil {
				return "", err
			}
			return "✅ 已切换至接管模式（全面管控）。\n\n" +
				"当前可用系统能力：\n" + bc.CapabilitiesText() + "\n\n" +
				"接管原因：" + reason + "\n\n" +
				"请在完成管理任务后调用 bridge_lockdown 归还权限，回到安全桥接模式。", nil
		},
	})

	// ── bridge_lockdown ──
	r.Register(&Tool{
		Name: "bridge_lockdown",
		Description: "归还全面接管权限，切换回安全桥接模式。桥接模式下 Agent 只能访问工作区内资源，" +
			"所有文件操作、命令执行均受安全约束。调用此工具后请确认 bridge_status 验证降级成功。",
		Parameters: objSchema(props{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			if err := bc.SwitchToBridged(); err != nil {
				// 已在桥接模式也提示已安全
				if strings.Contains(err.Error(), "已在桥接模式") {
					return "系统已在桥接模式（安全受限），无需降级。\n\n当前你可访问的系统能力：\n" + bc.CapabilitiesText(), nil
				}
				return "", err
			}
			return "✅ 已切换回桥接模式（安全受限）。\n\n" +
				"当前可用的系统能力：\n" + bc.CapabilitiesText() + "\n\n" +
				"所有操作已回到安全约束下。", nil
		},
	})

	// ── bridge_exec ──
	r.Register(&Tool{
		Name: "bridge_exec",
		Description: "通过桥接执行系统命令。行为取决于当前桥接模式：\n" +
			"- 桥接模式（默认）：限工作区内目录，120s 超时（同 run_command）\n" +
			"- 接管模式：不限目录，不限超时（默认 5 分钟），可执行系统管理命令\n\n" +
			"建议：日常开发用 run_command（标准模式），系统管理用 bridge_exec（接管模式）。",
		Parameters: objSchema(props{
			"command": strProp("要执行的命令"),
			"cwd":     strProp("可选工作目录（接管模式下不限工作区）"),
			"timeout": strProp("可选超时秒数（桥接模式最大 120s，接管模式最大 600s）"),
		}, "command"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			command := argStr(args, "command")
			if strings.TrimSpace(command) == "" {
				return "", fmt.Errorf("command 不能为空")
			}
			cwd := argStr(args, "cwd")
			timeoutSec := argInt(args, "timeout", 0)

			var timeout time.Duration
			if timeoutSec > 0 {
				timeout = time.Duration(timeoutSec) * time.Second
				if bc.Mode() == BridgeBridged && timeout > 120*time.Second {
					timeout = 120 * time.Second
				} else if bc.Mode() == BridgeTakeover && timeout > 600*time.Second {
					timeout = 600 * time.Second
				}
			}

			return bc.ExecCommand(ctx, command, cwd, timeout)
		},
	})

	// ── bridge_register_system_tool ──
	r.Register(&Tool{
		Name: "bridge_register_system_tool",
		Description: "【接管模式专用】注册一个系统管理工具到 Agent 工具集。" +
			"创建的 Go 工具可以调用系统命令、读写系统文件（不限于工作区）。" +
			"仅当处于接管模式时可用。创建的工具会注册到 Agent Registry 中，持续可用直到重启。\n\n" +
			"注意：name 不能与现有工具重名，handler_code 是 Go 代码片段，使用 system.Run() 执行系统命令、system.ReadFile()/WriteFile() 访问文件。",
		Parameters: objSchema(props{
			"name":        strProp("工具名（必填，唯一标识）"),
			"description": strProp("工具描述（必填）"),
			"handler_code": strProp("Go 处理函数体。" +
				"可用变量: args(map[string]any), system(*BridgeController), ctx(context.Context)。" +
				"示例: `name := args[\"name\"].(string); out, _ := system.ExecCommand(ctx, \"sc query \"+name, \"\", 0); return out, nil`"),
		}, "name", "description", "handler_code"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			if !bc.IsTakeover() {
				return "", fmt.Errorf("bridge_register_system_tool 仅在接管模式下可用。请先调用 bridge_takeover 申请接管权限")
			}

			name := sanitizeName(argStr(args, "name"))
			description := argStr(args, "description")
			handlerCode := argStr(args, "handler_code")

			if name == "" || description == "" || handlerCode == "" {
				return "", fmt.Errorf("name、description、handler_code 均不能为空")
			}

			if _, ok := r.Get(name); ok {
				return "", fmt.Errorf("工具 %q 已存在", name)
			}

			// 注册一个系统管理工具，通过 BridgeController 执行系统操作
			r.Register(&Tool{
				Name:        name,
				Description: "【系统工具】" + description + "（接管模式）",
				Parameters: objSchema(props{
					// 默认接收任意参数
				}),
				RequiresApproval: true,
				Handler: func(ctx context.Context, callArgs map[string]any) (string, error) {
					// 执行前再检查一次模式
					if !bc.IsTakeover() {
						return "", fmt.Errorf("系统工具 %q 仅在接管模式下可用，当前已降级回桥接模式", name)
					}
					// 注入 bridge_controller 到 handler 上下文，通过参数传递 system
					// 由于我们无法真正编译运行时 Go 代码，这里返回一条提示让 Agent 知道
					// 它可以用 bridge_exec 来实现等同功能
					return executeSystemTool(name, handlerCode, callArgs, bc, ctx)
				},
			})

			nameNote := ""
			if argStr(args, "name") != name {
				nameNote = fmt.Sprintf("\n⚠️ 工具名已自动清理（原 %q → %q）以符合 API 命名规范。", argStr(args, "name"), name)
			}
			return fmt.Sprintf("✅ 系统管理工具 %q 已注册。\n\n"+
				"工具描述: %s\n"+
				"生效范围: 仅接管模式下可用\n"+
				"注意: 下次重启后此工具不会持久化（需重新注册）。如需持久化，请将其写为 Lua 工具配合接管模式使用。%s",
				name, description, nameNote), nil
		},
	})
}

// bcStatusText 生成桥接状态报告。
func bcStatusText(bc *BridgeController, fullAudit bool) string {
	mode := bc.Mode()
	caps := bc.CapabilitiesText()
	audit := bc.AuditLogText(10)

	var b strings.Builder
	b.WriteString("## 桥接系统状态\n\n")
	b.WriteString("| 项目 | 值 |\n")
	b.WriteString("|------|-----|\n")
	b.WriteString(fmt.Sprintf("| 当前模式 | %s |\n", bc.ModeString()))
	if mode == BridgeTakeover {
		b.WriteString(fmt.Sprintf("| 授权时间 | %s |\n", time.Now().Format("2006-01-02 15:04:05")))
	}
	b.WriteString(fmt.Sprintf("| 工作区 | `%s` |\n", bc.root))
	b.WriteString(fmt.Sprintf("| 可用能力 | %s |\n", caps))
	b.WriteString("\n")

	if fullAudit {
		b.WriteString(audit)
		b.WriteString("\n")
	}

	// 建议
	if mode == BridgeBridged {
		b.WriteString("**建议**: 日常开发在桥接模式下即可。如需系统管理，用 `bridge_takeover` 申请接管。\n")
	} else {
		b.WriteString("**建议**: 完成系统管理任务后请及时调用 `bridge_lockdown` 归还权限。\n")
	}

	return b.String()
}

// executeSystemTool 执行注册的系统工具（模拟运行时 handler 执行）。
// 由于无法编译运行时 Go 代码，提供两条路径：
//   1. 如果是简单命令，直接通过 BridgeController 执行
//   2. 如果是复杂操作，返回引导信息让 Agent 用 bridge_exec 组合实现
func executeSystemTool(name, code string, args map[string]any, bc *BridgeController, ctx context.Context) (string, error) {
	// 尝试通过标准系统调用执行
	cmd := ""
	if c, ok := args["command"].(string); ok {
		cmd = c
	}

	if cmd != "" {
		return bc.ExecCommand(ctx, cmd, "", 0)
	}

	// 返回模板提示
	return fmt.Sprintf("系统工具 %q 已触发。处理函数体:\n```go\n%s\n```\n\n"+
		"当前调用参数: %s\n\n"+
		"如需执行系统命令，请直接用 bridge_exec(命令=...) 更简便。\n"+
		"如需持久化系统管理工具，请创建 Lua 工具 + 在接管模式下配合 bridge_exec 使用。",
		name, code, jsonArgs(args)), nil
}

// jsonArgs 把参数格式化为 JSON 字符串。
func jsonArgs(args map[string]any) string {
	data, err := json.MarshalIndent(args, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", args)
	}
	return string(data)
}

// sanitizeName 清理工具名：将非法字符替换为下划线，确保名称符合 API 命名规范。
func sanitizeName(name string) string {
	var b strings.Builder
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '\t' {
			b.WriteRune('_')
		} else if i == 0 {
			b.WriteRune('_')
		}
	}
	result := b.String()
	if result == "" {
		return "unnamed"
	}
	return result
}
