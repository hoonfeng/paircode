package agent

// ═══════════════════════════════════════════════════════════════
// commands_test.go — Round3 ④.2：ctx.commands 宿主命令面测试
//
// 覆盖：注册/清单/执行、插件卸载自动注销、未知命令报错、JS 插件桥
// （ctx.commands.register 端到端 + withLock 跨 goroutine 执行）。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"strings"
	"testing"
)

func resetCommandsForTest() {
	hostCmdMu.Lock()
	hostCommands = map[string]*HostCommand{}
	hostCmdOwner = map[string]string{}
	hostCmdOrder = nil
	hostCmdMu.Unlock()
}

// TestCommandsRegistry 注册/清单/执行/注销/未知命令。
func TestCommandsRegistry(t *testing.T) {
	resetCommandsForTest()
	t.Cleanup(resetCommandsForTest)

	// 注册（owner=plug-a）
	if err := RegisterHostCommand("agent-teams", "团队状态", func(ctx context.Context, args map[string]any) (string, error) {
		return "teams: " + strings.Join(argStrSlice(args, "ids"), ","), nil
	}, "plug-a"); err != nil {
		t.Fatalf("register: %v", err)
	}
	// 前导 "/" 归一化
	if err := RegisterHostCommand("/hello", "问候", func(ctx context.Context, args map[string]any) (string, error) {
		return "hi", nil
	}, "plug-b"); err != nil {
		t.Fatalf("register /hello: %v", err)
	}
	// 空名拒绝
	if err := RegisterHostCommand("", "x", func(ctx context.Context, args map[string]any) (string, error) { return "", nil }, "p"); err == nil {
		t.Error("空命令名应拒绝")
	}

	// 清单
	cmds := ListHostCommands()
	if len(cmds) != 2 {
		t.Fatalf("清单应 2 条，得 %d", len(cmds))
	}
	found := map[string]bool{}
	for _, c := range cmds {
		found[c["name"].(string)] = true
	}
	if !found["agent-teams"] || !found["hello"] {
		t.Errorf("清单异常: %v", cmds)
	}

	// 执行
	out, err := RunHostCommand("/agent-teams", map[string]any{"ids": []any{"t1", "t2"}})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "teams: t1,t2" {
		t.Errorf("执行结果异常: %q", out)
	}
	if _, err := RunHostCommand("no_such_cmd", nil); err == nil {
		t.Error("未知命令应报错")
	}

	// 插件卸载自动注销（plug-a 的命令消失，plug-b 保留）
	UnregisterHostCommands("plug-a")
	if FindHostCommand("agent-teams") != nil {
		t.Error("plug-a 卸载后 agent-teams 应注销")
	}
	if FindHostCommand("hello") == nil {
		t.Error("plug-b 的 hello 应保留")
	}
}

// TestJSPluginCommandsBridge ctx.commands JS 桥：register + run（withLock 跨 goroutine）。
func TestJSPluginCommandsBridge(t *testing.T) {
	resetCommandsForTest()
	t.Cleanup(resetCommandsForTest)

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`
	return {
		name: 'cmd-bridge-test',
		inject: ['commands'],
		apply(ctx) {
			ctx.commands.register({
				name: 'my-cmd',
				description: '测试命令',
				handler: (args) => {
					const who = (args && args.who) || 'world'
					return 'hello ' + who + ' from ' + ctx.app.workspaceRoot
				},
			})
		},
	}`, "cmd-bridge-test")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}

	// 清单含 my-cmd（owner=cmd-bridge-test）
	cmds := ListHostCommands()
	if len(cmds) != 1 || cmds[0]["name"] != "my-cmd" || cmds[0]["owner"] != "cmd-bridge-test" {
		t.Fatalf("JS 注册清单异常: %v", cmds)
	}

	// 执行（经 withLock 进入 VM；handler 内可访问 ctx 服务）
	out, err := RunHostCommand("my-cmd", map[string]any{"who": "成员"})
	if err != nil {
		t.Fatalf("JS 命令执行: %v", err)
	}
	if !strings.Contains(out, "hello 成员") || !strings.Contains(out, "C:\\ws") {
		t.Errorf("JS 命令输出异常: %q", out)
	}

	// 插件卸载 → 命令消失（无悬挂）
	if err := host.Unload("cmd-bridge-test"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	if FindHostCommand("my-cmd") != nil {
		t.Error("卸载后 my-cmd 应注销")
	}
}
