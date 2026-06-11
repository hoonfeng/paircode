//go:build windows

package bridge

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/user/gou-ide/cmd/companion/agent"
	"github.com/user/gou-ide/cmd/companion/ui/chat"
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/gou-ide/cmd/companion/ui/editor"
	"github.com/user/gou-ide/cmd/companion/ui/filetree"
	"github.com/user/gou-ide/cmd/companion/ui/state"
	"github.com/user/goui/pkg/animation"
	"github.com/user/goui/pkg/canvas"
	"github.com/user/goui/pkg/render"
	"github.com/user/goui/pkg/widget"
)

// TestAutonomousParams 自主模式：追加计划提示 + 放宽迭代上限；非自主：原样 + 默认上限。
func TestAutonomousParams(t *testing.T) {
	tn, n := AutonomousParams("做点事", false)
	if tn != "做点事" || n != 30 {
		t.Errorf("非自主 = (%q,%d)，期望原样 + 30", tn, n)
	}
	ta, a := AutonomousParams("做点事", true)
	if !strings.Contains(ta, "做点事") || !strings.Contains(ta, "update_plan") || a != 60 {
		t.Errorf("自主 = (%q,%d)，期望含原任务 + 计划提示 + 60", ta, a)
	}
}

// TestAskUserResolve ask_user：parseAsk 解析问题/选项；askUser 阻塞 → resolveAsk 送回答案。
func TestAskUserResolve(t *testing.T) {
	pa := chatpanel.ParseAsk(`{"question":"继续吗","options":["是","否"]}`)
	if pa.Question != "继续吗" || len(pa.Options) != 2 {
		t.Fatalf("ParseAsk = %+v", pa)
	}

	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore()}
	b := &AgentBridge{Cs: chatpanel.TheState}
	chatpanel.TheState.Bridge = b

	ansCh := make(chan string, 1)
	go func() {
		a, _ := b.askUser(context.Background(), map[string]any{"question": "继续吗"})
		ansCh <- a
	}()
	for i := 0; i < 200; i++ { // 等 askCh 就绪
		b.mu.Lock()
		ready := b.askCh != nil
		b.mu.Unlock()
		if ready {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	b.ResolveAsk("是")
	select {
	case a := <-ansCh:
		if a != "是" {
			t.Errorf("answer = %q，期望『是』", a)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("askUser 未返回")
	}
}

// 端到端（无窗口、无网络）：注入 MockProvider 的 loop，send 一条任务 → goroutine 跑 TAOR →
// 事件经动画帧泵 drain → 流式写进当前助手消息。手动 animation.Tick + EnsureLayout 模拟主循环。
func TestAgentBridgeStreamsIntoChat(t *testing.T) {
	// 复位聊天单例（ChatPanel.CreateState 返回它）。
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}

	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "note.txt"), []byte("AGENT_SEES_ME"), 0o644)
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "c1", Type: "function", Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"note.txt"}`}}}},
		{Content: "笔记内容是 AGENT_SEES_ME [FINAL]"},
	}}
	// 预置 loop（带 mock），跳过 buildProvider（无 key）。
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}

	// 挂 pipeline：SetState 需挂载的 Element + 全局钩子。
	animation.ResetScheduler()
	pipe := render.NewPipeline(360, 600, canvas.NewSoftCanvas(360, 600))
	rootEl := widget.CreateElementFor(chatpanel.Area())
	rootEl.Mount(nil, 0)
	pipe.SetRootElement(rootEl)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { pipe.MarkNeedsLayout() }
	defer func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil }()
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()

	chatpanel.TheState.Store.Draft = "读 note.txt 告诉我内容"
	chatpanel.TheState.Send()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		animation.Tick(time.Now()) // 推进帧泵 → drain（应用事件 + SetState）
		pipe.EnsureLayout()        // 消费 relayout

		th := chatpanel.TheState.Store.Active()
		last := th.Messages[len(th.Messages)-1]
		gotContent := strings.Contains(last.Text, "AGENT_SEES_ME")
		gotActivity := false
		for _, a := range last.Activities {
			if a.Tool == "read_file" && a.Done && strings.Contains(a.Result, "AGENT_SEES_ME") {
				gotActivity = true
			}
		}
		idle := !chatpanel.TheState.Bridge.IsRunning()
		if gotContent && gotActivity && idle && !last.Streaming {
			if err := pipe.Render(); err != nil { // 含工具活动+正文的消息卡渲染不报错
				t.Fatalf("含 Agent 回复的聊天渲染失败: %v", err)
			}
			if chatpanel.TheState.Bridge.(*AgentBridge).pump != nil {
				t.Error("结束后帧泵应已停")
			}
			return
		}
		time.Sleep(16 * time.Millisecond)
	}
	last := chatpanel.TheState.Store.Active().Messages
	t.Errorf("超时：Agent 回复未完整流入聊天。末条=%+v", last[len(last)-1])
}

// 手动审核：写类工具执行前出现「待批准」活动；用户「允许」后才写盘、活动完成。
func TestAgentBridgeApprovalApprove(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: false} // 手动审核
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"APPROVED"}`}}}},
		{Content: "已写入 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "写 out.txt"
	chatpanel.TheState.Send()

	// 阶段一：等到 write_file 活动进入「待批准」，且尚未写盘。
	var callID string
	if !pumpUntil(t, pipe, func() bool {
		for _, a := range lastActivities() {
			if a.Tool == "write_file" && a.AwaitingApproval {
				callID = a.CallID
				return true
			}
		}
		return false
	}, 5*time.Second) {
		t.Fatal("超时：未出现待批准的 write_file 活动")
	}
	if _, e := os.Stat(filepath.Join(root, "out.txt")); e == nil {
		t.Error("批准前不应写盘")
	}

	// 阶段二：批准 → 写盘、循环结算。
	chatpanel.ResolveApprovalUI(callID, true)
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 5*time.Second) {
		t.Fatal("超时：批准后未结算")
	}
	if b, _ := os.ReadFile(filepath.Join(root, "out.txt")); string(b) != "APPROVED" {
		t.Errorf("写入内容 = %q", b)
	}
	if err := pipe.Render(); err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
}

// 手动审核：用户「拒绝」后不写盘，拒绝结果回填到活动。
func TestAgentBridgeApprovalReject(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: false}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"X"}`}}}},
		{Content: "好的，不写了 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "写 out.txt"
	chatpanel.TheState.Send()

	var callID string
	if !pumpUntil(t, pipe, func() bool {
		for _, a := range lastActivities() {
			if a.AwaitingApproval {
				callID = a.CallID
				return true
			}
		}
		return false
	}, 5*time.Second) {
		t.Fatal("超时：未出现待批准活动")
	}

	chatpanel.ResolveApprovalUI(callID, false) // 拒绝
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 5*time.Second) {
		t.Fatal("超时：拒绝后未结算")
	}
	if _, e := os.Stat(filepath.Join(root, "out.txt")); e == nil {
		t.Error("拒绝后不应写盘")
	}
	var rejected bool
	for _, a := range lastActivities() {
		if a.Done && strings.Contains(a.Result, "拒绝") {
			rejected = true
		}
	}
	if !rejected {
		t.Error("拒绝结果应回填到活动（含『拒绝』字样）")
	}
}

// Agent 成功写文件后，已打开且无未存改动的编辑器标签应被重载为新内容（IDE 闭环）。
func TestAgentBridgeReloadsEditorAfterEdit(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true} // 自动审核：写直接执行
	editorpanel.Reset()                                                      // 复位编辑器单例
	filetreepanel.Reset()                                                    // 复位文件树单例
	root := t.TempDir()
	docAbs := filepath.Join(root, "doc.txt")
	os.WriteFile(docAbs, []byte("OLD"), 0o644)
	editorpanel.Editor.Open(docAbs) // 编辑器打开（读入 "OLD"）

	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"doc.txt","content":"NEW"}`}}}},
		{Content: "已更新 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "把 doc.txt 改成 NEW"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 5*time.Second) {
		t.Fatal("超时：未结算")
	}
	if b, _ := os.ReadFile(docAbs); string(b) != "NEW" {
		t.Fatalf("磁盘应为 NEW，得 %q", b)
	}
	found := false
	for _, tt := range editorpanel.Editor.Tabs() {
		if tt.Path() == docAbs {
			found = true
			if tt.Content() != "NEW" {
				t.Errorf("编辑器标签应被重载为 NEW，得 %q", tt.Content())
			}
		}
	}
	if !found {
		t.Fatal("编辑器应仍打开 doc.txt 标签")
	}
}

// reloadIfOpen：已打开且无改动→重载；dirty→不覆盖；未打开→不动。
func TestEditorReloadIfOpen(t *testing.T) {
	editorpanel.Reset()
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("V1"), 0o644)
	editorpanel.Editor.Open(p)

	os.WriteFile(p, []byte("V2"), 0o644)
	if !editorpanel.Editor.ReloadIfOpen(p) {
		t.Fatal("无改动的已打开标签应被重载")
	}
	if editorpanel.Editor.Tabs()[0].Content() != "V2" {
		t.Errorf("重载后内容应为 V2，得 %q", editorpanel.Editor.Tabs()[0].Content())
	}

	editorpanel.Editor.Tabs()[0].SetDirty(true)
	os.WriteFile(p, []byte("V3"), 0o644)
	if editorpanel.Editor.ReloadIfOpen(p) {
		t.Error("dirty 标签不应被覆盖")
	}
	if editorpanel.Editor.Tabs()[0].Content() != "V2" {
		t.Errorf("dirty 后内容不应变，得 %q", editorpanel.Editor.Tabs()[0].Content())
	}

	if editorpanel.Editor.ReloadIfOpen(filepath.Join(dir, "nope.txt")) {
		t.Error("未打开的文件不应重载")
	}
}

// blockingProvider 的 Chat 阻塞到 ctx 取消（模拟长任务，用于测停止）。
type blockingProvider struct{}

func (blockingProvider) Name() string { return "block" }
func (blockingProvider) Chat(ctx context.Context, m []agent.Message, td []agent.ToolDefinition, oc func(agent.Chunk)) (agent.Message, error) {
	<-ctx.Done()
	return agent.Message{}, ctx.Err()
}

// 停止按钮：运行中 stop() 取消 ctx → loop 退出、消息收尾标 [已停止]（不显示底层取消错误）。
func TestAgentBridgeStop(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, t.TempDir())
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: t.TempDir(),
		loop: &agent.Loop{Provider: blockingProvider{}, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "跑个长任务"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return chatpanel.TheState.Bridge.IsRunning() }, 3*time.Second) {
		t.Fatal("应进入运行态")
	}
	chatpanel.TheState.Bridge.Stop()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 3*time.Second) {
		t.Fatal("停止后应结算")
	}
	msgs := chatpanel.TheState.Store.Active().Messages
	if last := msgs[len(msgs)-1]; !strings.Contains(last.Text, "已停止") {
		t.Errorf("停止后末条应含 [已停止]，得 %q", last.Text)
	}
	if strings.Contains(msgs[len(msgs)-1].Text, "[错误]") {
		t.Error("用户主动停止不应显示 [错误]")
	}
}

// Agent 卡 Markdown 正文（标题/列表/行内/代码块）+ 折叠两态都能无错渲染。
func TestAgentCardMarkdownAndCollapseRender(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	th := chatpanel.TheState.Store.Active()
	th.Messages = append(th.Messages, state.Message{
		Role:       state.Assistant,
		Thinking:   "先想一下方案…",
		Activities: []state.Activity{{CallID: "1", Tool: "read_file", Args: `{"path":"x.go"}`, Result: "ok", Done: true}},
		Text:       "# 标题\n\n说明文字 **加粗** 与 `行内代码`。\n\n- 项目一\n- 项目二\n\n```go\nfunc main() {}\n```\n",
	})
	pipe := newChatPipe(t)
	if err := pipe.Render(); err != nil {
		t.Fatalf("展开态渲染失败: %v", err)
	}
	// 折叠后再渲染（走折叠摘要路径）
	chatpanel.TheState.Store.Active().Messages[1].Collapsed = true
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	if err := pipe.Render(); err != nil {
		t.Fatalf("折叠态渲染失败: %v", err)
	}
}

// autoCollapse(收缩) 开：一轮完成后该助手消息应自动折叠。
func TestAgentBridgeAutoCollapse(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, AutoCollapse: true}
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, t.TempDir())
	mock := &agent.MockProvider{Responses: []agent.Message{{Content: "完成了 [FINAL]"}}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: t.TempDir(),
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "干活"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 5*time.Second) {
		t.Fatal("超时：未结算")
	}
	msgs := chatpanel.TheState.Store.Active().Messages
	if last := msgs[len(msgs)-1]; !last.Collapsed {
		t.Error("autoCollapse 开，完成后该消息应自动折叠")
	}
}

// 消息操作：删除移除该条。
func TestChatMessageDelete(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, HoveredMsg: -1}
	th := chatpanel.TheState.Store.Active()
	th.Messages = append(th.Messages, state.Message{Role: state.User, Text: "hi"}, state.Message{Role: state.Assistant, Text: "yo"})
	n := len(th.Messages)
	chatpanel.TheState.DeleteMessage(th, n-1)
	if len(th.Messages) != n-1 {
		t.Fatalf("删后应 %d 条，得 %d", n-1, len(th.Messages))
	}
	if th.Messages[len(th.Messages)-1].Text == "yo" {
		t.Error("末条 yo 应已删除")
	}
}

// 消息操作：重新生成删掉末条助手回复并重跑一轮。
func TestChatRegenerate(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, HoveredMsg: -1}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{{Content: "重新生成的回复 [FINAL]"}}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test", MaxIterations: 5}}
	pipe := newChatPipe(t)
	th := chatpanel.TheState.Store.Active()
	th.Messages = append(th.Messages, state.Message{Role: state.User, Text: "原任务"}, state.Message{Role: state.Assistant, Text: "旧回复"})

	chatpanel.TheState.Regenerate(th, len(th.Messages)-1)
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 5*time.Second) {
		t.Fatal("超时：重新生成未结算")
	}
	last := chatpanel.TheState.Store.Active().Messages
	if l := last[len(last)-1]; l.Role != state.Assistant || !strings.Contains(l.Text, "重新生成") {
		t.Errorf("末条应为新助手回复，得 %+v", l)
	}
	for _, m := range last {
		if m.Text == "旧回复" {
			t.Error("旧回复应被删除替换")
		}
	}
}

// Ctrl+F 搜索：匹配计数 + 过滤渲染无错；hover 操作叠加渲染无错。
func TestChatSearchAndHoverRender(t *testing.T) {
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, HoveredMsg: -1}
	th := chatpanel.TheState.Store.Active()
	th.Messages = []state.Message{
		{Role: state.User, Text: "实现快速排序"},
		{Role: state.Assistant, Text: "好的，用 Go 写 quicksort"},
		{Role: state.User, Text: "再加个二分查找"},
	}
	chatpanel.TheState.SearchQuery = "quicksort"
	if c := chatpanel.TheState.SearchMatchCount(); c != 1 {
		t.Errorf("quicksort 应匹配 1 条，得 %d", c)
	}
	chatpanel.TheState.SearchQuery = "查找"
	if c := chatpanel.TheState.SearchMatchCount(); c != 1 {
		t.Errorf("『查找』应匹配 1 条，得 %d", c)
	}
	chatpanel.TheState.SearchQuery = "不存在XYZ"
	if c := chatpanel.TheState.SearchMatchCount(); c != 0 {
		t.Errorf("无匹配应 0，得 %d", c)
	}
	// 开搜索 + 过滤渲染
	chatpanel.TheState.ShowSearch = true
	chatpanel.TheState.SearchQuery = "查找"
	widget.ClipboardWrite = func(string) {}
	defer func() { widget.ClipboardWrite = nil }()
	pipe := newChatPipe(t)
	if err := pipe.Render(); err != nil {
		t.Fatalf("搜索过滤渲染失败: %v", err)
	}
	// hover 末条 → 操作按钮叠加渲染
	chatpanel.TheState.ShowSearch = false
	chatpanel.TheState.SearchQuery = ""
	chatpanel.TheState.HoveredMsg = len(th.Messages) - 1
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	if err := pipe.Render(); err != nil {
		t.Fatalf("hover 操作叠加渲染失败: %v", err)
	}
}

// ─── 测试辅助 ─────────────────────────────────────────────

// newChatPipe 挂一个最小渲染管线（SetState 需挂载 Element + 全局钩子），返回管线。
func newChatPipe(t *testing.T) *render.Pipeline {
	t.Helper()
	animation.ResetScheduler()
	pipe := render.NewPipeline(360, 600, canvas.NewSoftCanvas(360, 600))
	rootEl := widget.CreateElementFor(chatpanel.Area())
	rootEl.Mount(nil, 0)
	pipe.SetRootElement(rootEl)
	widget.OnNeedsRepaint = func() { pipe.MarkNeedsRepaint() }
	widget.OnNeedsLayout = func() { pipe.MarkNeedsLayout() }
	t.Cleanup(func() { widget.OnNeedsRepaint = nil; widget.OnNeedsLayout = nil })
	pipe.MarkNeedsLayout()
	pipe.EnsureLayout()
	return pipe
}

// pumpUntil 手动逐帧推进（animation.Tick→帧泵 drain + EnsureLayout），直到 cond 成立或超时。
func pumpUntil(t *testing.T, pipe *render.Pipeline, cond func() bool, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		animation.Tick(time.Now())
		pipe.EnsureLayout()
		if cond() {
			return true
		}
		time.Sleep(8 * time.Millisecond)
	}
	return false
}

// lastActivities 取当前会话最后一条消息的工具活动。
func lastActivities() []state.Activity {
	th := chatpanel.TheState.Store.Active()
	if th == nil || len(th.Messages) == 0 {
		return nil
	}
	return th.Messages[len(th.Messages)-1].Activities
}

// bridgeSettled 桥彻底结算：循环已结束(running=false)且帧泵已停(pump=nil)。stopPump 只在
// 「done 帧」里、把所有 pending 事件 drain 应用之后调用，故 pump==nil ⟹ 全部事件已落到消息上。
// （注意：单看 !isRunning() 会过早——goroutine 在 Tick 与 cond 间隙置 running=false 时，
// 末批事件可能还在 pending 未 drain；真实 app 主循环会继续出帧直到泵自停，测试须同样等到结算。）
func bridgeSettled(ab chatpanel.AgentBridge) bool {
	b := ab.(*AgentBridge)
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.running && b.pump == nil
}

// TestBuildCompressor 压缩 Provider 构建：启用+配齐→Provider；Key/地址留空复用主模型；禁用/配不齐→nil。
func TestBuildCompressor(t *testing.T) {
	saved := core.Settings
	defer func() { core.Settings = saved }()

	core.Settings = core.AppSettings{CompressEnabled: false}
	if BuildCompressor() != nil {
		t.Error("CompressEnabled=false 应返回 nil")
	}

	core.Settings = core.AppSettings{CompressEnabled: true, APIKey: "k", BaseURL: "b"} // 无压缩模型名
	if BuildCompressor() != nil {
		t.Error("配不齐（无模型）应返回 nil")
	}

	// Key/地址留空 → 复用主模型；默认 non-thinking / temp0.3 / 4K
	core.Settings = core.AppSettings{CompressEnabled: true, APIKey: "mainkey", BaseURL: "https://main/v1", CompressModel: "deepseek-v4-flash"}
	op, ok := BuildCompressor().(*agent.OpenAIProvider)
	if !ok {
		t.Fatal("应返回 *OpenAIProvider")
	}
	if op.APIKey != "mainkey" || op.BaseURL != "https://main/v1" {
		t.Errorf("留空应复用主模型 Key/地址，得 %q/%q", op.APIKey, op.BaseURL)
	}
	if op.Model != "deepseek-v4-flash" || op.ThinkingMode != "non-thinking" || op.Temperature != 0.3 || op.MaxTokens != 4096 {
		t.Errorf("压缩模型默认参数不对：%+v", op)
	}

	// 独立压缩 Key/地址/思考 → 用各自的
	core.Settings = core.AppSettings{CompressEnabled: true, APIKey: "mainkey", BaseURL: "https://main/v1",
		CompressAPIKey: "ckey", CompressBaseURL: "https://compress/v1", CompressModel: "glm-4.7-flash", CompressThinkingMode: "thinking"}
	op = BuildCompressor().(*agent.OpenAIProvider)
	if op.APIKey != "ckey" || op.BaseURL != "https://compress/v1" || op.Model != "glm-4.7-flash" || op.ThinkingMode != "thinking" {
		t.Errorf("独立压缩配置未生效：%+v", op)
	}
}

// TestAgentBridgeCompactionNote 低窗口 + 多轮工具循环 → 助手消息出现「上下文已压缩」提示且渲染无错。
func TestAgentBridgeCompactionNote(t *testing.T) {
	saved := core.Settings
	defer func() { core.Settings = saved }()
	core.Settings = core.AppSettings{ContextMaxTokens: 120} // 低窗口触发；CompressEnabled=false→规则式摘要

	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	var responses []agent.Message
	for i := 0; i < 16; i++ { // 多轮 read_file 不同文件把上下文堆高（不同路径→不触发绕圈检测）
		os.WriteFile(filepath.Join(root, "f"+strconv.Itoa(i)+".txt"), []byte(strings.Repeat("padding content ", 20)), 0o644)
		responses = append(responses, agent.Message{ToolCalls: []agent.ToolCall{{ID: "c" + strconv.Itoa(i), Type: "function",
			Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"f` + strconv.Itoa(i) + `.txt"}`}}}})
	}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: &agent.MockProvider{Responses: responses}, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "读一批文件"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 8*time.Second) {
		t.Fatal("超时：未结算")
	}
	last := chatpanel.TheState.Store.Active().Messages
	m := last[len(last)-1]
	var compacted bool
	for _, n := range m.Notes {
		if strings.Contains(n, "压缩") {
			compacted = true
		}
	}
	if !compacted {
		t.Errorf("低窗口多轮应出现上下文压缩提示，得 notes=%v", m.Notes)
	}
	if err := pipe.Render(); err != nil {
		t.Fatalf("含压缩提示的消息卡渲染失败: %v", err)
	}
}

// TestBuildRoleProvider 规划/审核 Provider：复用主模型接口+Key、换模型、non-thinking；缺模型/主未配→nil。
func TestBuildRoleProvider(t *testing.T) {
	saved := core.Settings
	defer func() { core.Settings = saved }()

	core.Settings = core.AppSettings{PlanModel: "plan-m"} // 主模型未配
	if BuildPlanProvider() != nil {
		t.Error("主模型未配应 nil")
	}
	core.Settings = core.AppSettings{BaseURL: "https://x/v1", APIKey: "k", PlanModel: "plan-m", ReviewModel: "rev-m"}
	pp, _ := BuildPlanProvider().(*agent.OpenAIProvider)
	if pp == nil || pp.Model != "plan-m" || pp.BaseURL != "https://x/v1" || pp.APIKey != "k" || pp.ThinkingMode != "non-thinking" {
		t.Errorf("规划 provider 错：%+v", pp)
	}
	rp, _ := BuildReviewProvider().(*agent.OpenAIProvider)
	if rp == nil || rp.Model != "rev-m" || rp.ThinkingMode != "non-thinking" {
		t.Errorf("审核 provider 错：%+v", rp)
	}
	core.Settings.PlanModel = ""
	if BuildPlanProvider() != nil {
		t.Error("规划模型名空应 nil")
	}
}

// TestAgentBridgePlannerPhase 自主模式 → 规划 Agent 先列计划，计划卡被填充，再执行。
func TestAgentBridgePlannerPhase(t *testing.T) {
	saved, savedMP := core.Settings, MakePlanner
	defer func() { core.Settings, MakePlanner = saved, savedMP }()
	core.Settings = core.AppSettings{}
	MakePlanner = func() *agent.Planner { // 注入 mock 规划 Agent
		return &agent.Planner{Provider: &agent.MockProvider{Responses: []agent.Message{
			{Content: `{"reasoning":"先读后改","steps":[{"id":"step-1","description":"读配置"},{"id":"step-2","description":"改配置"}]}`}}}}
	}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, Autonomous: true} // 自主触发规划
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, t.TempDir())
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: t.TempDir(),
		loop: &agent.Loop{Provider: &agent.MockProvider{Responses: []agent.Message{{Content: "执行完毕 [FINAL]"}}}, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "重构配置"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 6*time.Second) {
		t.Fatal("超时：未结算")
	}
	if len(chatpanel.TheState.Plan) != 2 || chatpanel.TheState.Plan[0].Step != "读配置" {
		t.Errorf("规划 Agent 应填充计划卡 2 步，得 %+v", chatpanel.TheState.Plan)
	}
}

// TestAgentBridgeAIReview AI 审核驳回写操作 → 不写盘，建议回灌到活动结果。
func TestAgentBridgeAIReview(t *testing.T) {
	saved, savedMR := core.Settings, MakeReviewer
	defer func() { core.Settings, MakeReviewer = saved, savedMR }()
	core.Settings = core.AppSettings{AIReview: true}
	MakeReviewer = func() *agent.Reviewer { // 注入 mock 审核 Agent（驳回写操作）
		return &agent.Reviewer{Provider: &agent.MockProvider{Responses: []agent.Message{
			{Content: `{"verdict":"驳回","suggestions":["这个写法不安全，改用参数化"],"summary":"有注入风险"}`}}}}
	}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "w1", Type: "function", Function: agent.FunctionCall{Name: "write_file", Arguments: `{"path":"out.txt","content":"X"}`}}}},
		{Content: "好的，我换个方式 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "写 out.txt"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 6*time.Second) {
		t.Fatal("超时：未结算")
	}
	if _, e := os.Stat(filepath.Join(root, "out.txt")); e == nil {
		t.Error("AI 审核驳回的 write_file 不应写盘")
	}
	var fedback bool
	for _, a := range lastActivities() {
		if a.Tool == "write_file" && strings.Contains(a.Result, "不安全") {
			fedback = true
		}
	}
	if !fedback {
		t.Error("AI 审核驳回应把建议回灌到活动结果")
	}
}

// TestAgentBridgeBenchmark 评测开 + 有工具活动 → 任务完成后产生评分卡且渲染无错；无工具活动→不评。
func TestAgentBridgeBenchmark(t *testing.T) {
	saved, savedME := core.Settings, MakeEvaluator
	defer func() { core.Settings, MakeEvaluator = saved, savedME }()
	core.Settings = core.AppSettings{Benchmark: true}
	MakeEvaluator = func() *agent.Evaluator { // 注入 mock 评测 Agent
		return &agent.Evaluator{Provider: &agent.MockProvider{Responses: []agent.Message{
			{Content: `{"scores":{"completion":36,"correctness":27,"depth":16,"efficiency":6},"total":85,"strengths":["清晰"],"weaknesses":[],"feedback":"完成良好"}`}}}}
	}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0o644)
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{ // 读文件(产生工具活动) → [FINAL]
		{ToolCalls: []agent.ToolCall{{ID: "r1", Type: "function", Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"f.txt"}`}}}},
		{Content: "读完了 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "读 f.txt"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 6*time.Second) {
		t.Fatal("超时：未结算")
	}
	m := chatpanel.TheState.Store.Active().Messages
	last := m[len(m)-1]
	if last.Eval == nil {
		t.Fatal("有工具活动 + 评测开 应产生评分")
	}
	if last.Eval.Total != 85 || last.Eval.Completion != 36 {
		t.Errorf("评分错：%+v", last.Eval)
	}
	if err := pipe.Render(); err != nil {
		t.Fatalf("评分卡渲染失败: %v", err)
	}
}

// TestAgentBridgeAutoIterate 评分不足(50<80) + 自动迭代开 → 据建议重跑一轮，评分提升到 85，有迭代提示。
func TestAgentBridgeAutoIterate(t *testing.T) {
	saved, savedME := core.Settings, MakeEvaluator
	defer func() { core.Settings, MakeEvaluator = saved, savedME }()
	core.Settings = core.AppSettings{Benchmark: true, AutoIterate: true, ReviewRetries: 3}
	MakeEvaluator = func() *agent.Evaluator { // 初评 50 → 改进后 85
		return &agent.Evaluator{Provider: &agent.MockProvider{Responses: []agent.Message{
			{Content: `{"scores":{"completion":20,"correctness":15,"depth":10,"efficiency":5},"total":50,"weaknesses":["缺错误处理"],"feedback":"基本完成但有不足"}`},
			{Content: `{"scores":{"completion":36,"correctness":27,"depth":16,"efficiency":6},"total":85,"strengths":["健壮"],"feedback":"改进到位"}`},
		}}}
	}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "f.txt"), []byte("x"), 0o644)
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{ // 两轮，每轮 [读文件 → FINAL]
		{ToolCalls: []agent.ToolCall{{ID: "r1", Type: "function", Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"f.txt"}`}}}},
		{Content: "初版完成 [FINAL]"},
		{ToolCalls: []agent.ToolCall{{ID: "r2", Type: "function", Function: agent.FunctionCall{Name: "read_file", Arguments: `{"path":"f.txt"}`}}}},
		{Content: "改进完成 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "做个任务"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 8*time.Second) {
		t.Fatal("超时：未结算")
	}
	last := chatpanel.TheState.Store.Active().Messages
	m := last[len(last)-1]
	if m.Eval == nil || m.Eval.Total != 85 {
		t.Fatalf("自动迭代后应显示改进分 85，得 %+v", m.Eval)
	}
	var iterated bool
	for _, n := range m.Notes {
		if strings.Contains(n, "自动迭代") {
			iterated = true
		}
	}
	if !iterated {
		t.Error("应有自动迭代提示")
	}
}

// TestAgentBridgeLuaTools .pair/tools/*.lua 自定义工具被热加载，Agent 调用即执行（沙箱）。
func TestAgentBridgeLuaTools(t *testing.T) {
	saved := core.Settings
	defer func() { core.Settings = saved }()
	core.Settings = core.AppSettings{LuaTools: true}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	root := t.TempDir()
	toolsDir := filepath.Join(root, ".pair", "tools")
	os.MkdirAll(toolsDir, 0o755)
	os.WriteFile(filepath.Join(toolsDir, "wc.lua"),
		[]byte(`return { name="word_count", description="字数", parameters={type="object",properties={text={type="string"}},required={"text"}}, run=function(args) return "字数: "..#(args.text or "") end }`), 0o644)
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	mock := &agent.MockProvider{Responses: []agent.Message{
		{ToolCalls: []agent.ToolCall{{ID: "l1", Type: "function", Function: agent.FunctionCall{Name: "word_count", Arguments: `{"text":"hello"}`}}}},
		{Content: "统计完成 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "数 hello 的字数"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 6*time.Second) {
		t.Fatal("超时：未结算")
	}
	var ran bool
	for _, a := range lastActivities() {
		if a.Tool == "word_count" && strings.Contains(a.Result, "字数: 5") {
			ran = true
		}
	}
	if !ran {
		t.Error("Lua 自定义工具 word_count 应被热加载并执行")
	}
}

// TestAgentBridgeExploreVerify 自主模式跑「探索→执行→验证」编排：探索/验证阶段提示出现，且只读（不写盘）。
func TestAgentBridgeExploreVerify(t *testing.T) {
	saved := core.Settings
	defer func() { core.Settings = saved }()
	core.Settings = core.AppSettings{} // Benchmark/AIReview 关、无 planModel
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true, Autonomous: true}
	root := t.TempDir()
	reg := agent.NewRegistry()
	agent.RegisterDefaultTools(reg, root)
	// 探索/执行/验证 三阶段共用此 mock（顺序消费）：各返回无工具调用的 [FINAL] 文本即结束本阶段。
	mock := &agent.MockProvider{Responses: []agent.Message{
		{Content: "探索发现：项目是 Go [FINAL]"},
		{Content: "执行完成 [FINAL]"},
		{Content: "验证通过 [FINAL]"},
	}}
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState, root: root,
		loop: &agent.Loop{Provider: mock, Registry: reg, System: "test"}}
	pipe := newChatPipe(t)

	chatpanel.TheState.Store.Draft = "做个任务"
	chatpanel.TheState.Send()
	if !pumpUntil(t, pipe, func() bool { return bridgeSettled(chatpanel.TheState.Bridge) }, 8*time.Second) {
		t.Fatal("超时：未结算")
	}
	m := chatpanel.TheState.Store.Active().Messages
	last := m[len(m)-1]
	var explored, verified bool
	for _, n := range last.Notes {
		if strings.Contains(n, "探索阶段") {
			explored = true
		}
		if strings.Contains(n, "验证阶段") {
			verified = true
		}
	}
	if !explored || !verified {
		t.Errorf("自主模式应跑探索 + 验证阶段，notes=%v", last.Notes)
	}
}

// TestActivitySummary 跨轮连续性：助手消息的工具活动压成一行摘要（成功 ✓ / 失败·拒绝 ✗），供下轮免重复。
func TestActivitySummary(t *testing.T) {
	acts := []state.Activity{
		{Tool: "read_file", Args: `{"path":"x.go"}`, Result: "内容", Done: true},
		{Tool: "edit_file", Args: `{"path":"x.go"}`, Result: "Error: 失败", Done: true},
	}
	s := activitySummary(acts)
	if !strings.Contains(s, "read_file") || !strings.Contains(s, "✓") {
		t.Errorf("成功活动应标 ✓，得 %q", s)
	}
	if !strings.Contains(s, "edit_file") || !strings.Contains(s, "✗") {
		t.Errorf("失败活动应标 ✗，得 %q", s)
	}
	if activitySummary(nil) != "" {
		t.Error("无活动应返回空")
	}
}

// 无 API key 时 send 给出明确提示、不挂起（不建 loop、不进循环）。
func TestAgentBridgeNoKeyHint(t *testing.T) {
	if BuildProvider() != nil {
		t.Skip("环境配了 API key，跳过无-key 提示测试（避免真网络调用）")
	}
	chatpanel.TheState = &chatpanel.ChatState{Store: state.NewChatStore(), AutoReview: true}
	// 确保环境无 key（测试环境本就无，双保险不依赖）。
	chatpanel.TheState.Bridge = &AgentBridge{Cs: chatpanel.TheState} // loop=nil → start 走 buildProvider
	chatpanel.TheState.Store.Draft = "你好"
	chatpanel.TheState.Send()

	th := chatpanel.TheState.Store.Active()
	last := th.Messages[len(th.Messages)-1]
	if last.Role != state.Assistant || !strings.Contains(last.Text, "API key") {
		t.Errorf("无 key 应给出配置提示，得 %+v", last)
	}
	if chatpanel.TheState.Bridge.IsRunning() {
		t.Error("无 key 不应进入运行态")
	}
}
