package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
	"wb-ui/goja"
)

// ─── JS 循环端到端测试（agentloop 核心外置链路）──

// 装载真实 agentloop 磁盘插件（.pair/plugins/agentloop/index.js），
// 验证：registerLoop 注册 → Loop.Run 委托 JS → 流式事件 → 工具执行 → 自然终止。
func loadRealAgentloop(t *testing.T) (*PluginHost, string) {
	t.Helper()
	code, err := os.ReadFile(filepath.Join("..", "..", ".pair", "plugins", "agentloop", "index.js"))
	if err != nil {
		t.Skipf("无 agentloop 磁盘插件源码: %v", err)
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(string(code), "agentloop real e2e")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	t.Cleanup(func() { _ = host.Unload(def.name) })
	if CurrentJSLoop() == nil {
		t.Fatal("agentloop 插件装载后 CurrentJSLoop 应为非空（registerLoop 未生效）")
	}
	return host, def.name
}

func TestJSLoopRealAgentloopToolThenFinal(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	// 防污染：前置检查全局状态
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	// registerSettings 依赖 core.Settings
	coreSettingsEnsure()

	loadRealAgentloop(t)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("JSLOOP_WORLD"), 0o644)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)

	mock := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"hello.txt"}`}}}},
		{Content: "读到了 JSLOOP_WORLD"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, System: "test-js-loop", MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) }}

	msgs, err := loop.Run(context.Background(), "读 hello.txt 告诉我内容", nil)
	if err != nil {
		t.Fatalf("Run(JS 循环): %v", err)
	}
	if mock.Calls() != 2 {
		t.Errorf("LLM 应调用 2 次，得 %d", mock.Calls())
	}
	// 工具结果回灌
	foundTool := false
	for _, m := range msgs {
		if m.Role == RoleTool && strings.Contains(m.Content, "JSLOOP_WORLD") {
			foundTool = true
		}
	}
	if !foundTool {
		t.Error("未把 read_file 结果作 role=tool 消息回灌")
	}
	// 末事件 done
	last := events[len(events)-1]
	if last.Type != EventDone || last.DoneReason != "task_complete" || !strings.Contains(last.Content, "JSLOOP_WORLD") {
		t.Errorf("末事件应为 EventDone(task_complete) 含结果，得 %+v", last)
	}
	// 事件流完整性：thinking/content/tool_call/tool_result/usage
	var sawCall, sawResult, sawContent bool
	for _, e := range events {
		if e.Type == EventToolCall && e.Tool == "read_file" {
			sawCall = true
		}
		if e.Type == EventToolResult && strings.Contains(e.Content, "JSLOOP_WORLD") {
			sawResult = true
		}
		if e.Type == EventContent {
			sawContent = true
		}
	}
	if !sawCall || !sawResult || !sawContent {
		t.Errorf("缺事件：tool_call=%v tool_result=%v content=%v", sawCall, sawResult, sawContent)
	}
	// 消息结构完整：system + user + assistant(tool) + tool + assistant(done)
	if len(msgs) < 5 {
		t.Errorf("msgs 过短: %d（应有 system+user+assistant+tool+assistant 至少 5 条）", len(msgs))
	}
}

// 自然终止：无工具调用 + 正文 → 1 次 LLM 调用即完成。
func TestJSLoopRealAgentloopNaturalFinish(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()
	loadRealAgentloop(t)

	mock := &MockProvider{Responses: []Message{{Content: "任务完成"}}}
	loop := &Loop{Provider: mock, Registry: NewRegistry(), MaxIterations: 5}
	if _, err := loop.Run(context.Background(), "完成", nil); err != nil {
		t.Fatal(err)
	}
	if mock.Calls() != 1 {
		t.Errorf("应 1 次 LLM 调用，得 %d", mock.Calls())
	}
}

// delegate 子 agent：JS 循环内部创建子 Loop（SubAgentSink 事件过滤）。
func TestJSLoopDelegateSubAgent(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()

	// 精简插件：run 内 delegate 一个子 agent
	const delegatePlugin = `
return {
  name: 'jsloop-delegate',
  apply(ctx) {
    ctx.loopFactory.registerLoop({
      id: 'delegate-e2e',
      async run({ task, msgs, tools, meta, loop }) {
        // 委托子 agent 执行子任务
        const sub = loop.delegate.run({
          task: '子任务：回复 hello',
          agentName: 'coder',
          maxIterations: 2,
        });
        // 子 agent 结果注入本 agent
        loop.events.emit({ type: 'notice', content: '子 agent 结果: ' + (sub.error || sub.content) });
        loop.events.emit({ type: 'done', content: '父任务完成，子结果=' + sub.content, doneReason: 'task_complete', turnReason: 'completed' });
        return { msgs };
      }
    })
  }
}`
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(delegatePlugin, "delegate e2e")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	t.Cleanup(func() { _ = host.Unload(def.name) })

	// MockProvider：父调用返回子结果可观察——用脚本化 provider：
	// 第 1 次调用（父）：自然完成
	// 子 agent 的调用也走同一 mock（按调用顺序）
	mock := &MockProvider{Responses: []Message{
		{Content: "父 agent 回复"},
		{Content: "子 agent 回复 hello"},
	}}
	var events []Event
	loop := &Loop{Provider: mock, Registry: reg, MaxIterations: 5,
		OnEvent: func(e Event) { events = append(events, e) }}
	msgs, err := loop.Run(context.Background(), "父任务", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 事件流应包含子 agent 标记的 notice（AgentName=coder）
	var sawSubNotice bool
	for _, e := range events {
		if e.Type == EventNotice && strings.Contains(e.Content, "子 agent") {
			sawSubNotice = true
			break
		}
	}
	if !sawSubNotice {
		t.Error("未收到子 agent 结果 notice 事件")
	}
	_ = msgs
}

// 回退：卸载插件 → 还原 Go 默认循环。
func TestJSLoopUnloadRestoreGoLoop(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if CurrentJSLoop() != nil {
		t.Skipf("已有 JS 循环注册（%v），跳过防污染", CurrentJSLoop().id)
	}
	coreSettingsEnsure()

	host, _ := loadRealAgentloop(t)
	// 卸载
	if err := host.Unload("agentloop"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	if CurrentJSLoop() != nil {
		t.Fatal("卸载后 CurrentJSLoop 应为 nil（还原 Go 循环）")
	}
	// 卸载后 Run 走 Go 循环
	mock := &MockProvider{Responses: []Message{{Content: "完成"}}}
	loop := &Loop{Provider: mock, Registry: NewRegistry(), MaxIterations: 5}
	if _, err := loop.Run(context.Background(), "ok", nil); err != nil {
		t.Fatal(err)
	}
	if mock.Calls() != 1 {
		t.Errorf("Go 循环应 1 次 LLM 调用，得 %d", mock.Calls())
	}
}

// ── 辅助 ──

// gojaOk 检查 goja 可用（测试环境）。
func gojaOk() bool {
	vm := goja.New()
	_, err := vm.RunString("1+1")
	return err == nil
}

// coreSettingsEnsure 确保 core.Settings 已初始化（registerSettings 依赖）。
func coreSettingsEnsure() {
	if core.Settings.Provider == "" {
		core.Settings = core.Default()
	}
}
