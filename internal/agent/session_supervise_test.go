// session_supervise_test.go — 自主模式（监督者）端到端回归（★ 2026-09-21）
//
// 钉死新的自主模式语义：工作 agent 每次自然结束（无 tool_call + 有正文）时，宿主在
// 会话续轮处调用**插件注册的决策器**（ctx.loopFactory.registerAutopilot → 本测试用
// goja 里的等价实现）：
//   · action=continue + task → 把指令作为新任务唤醒工作 agent 继续执行；
//   · action=done            → 整轮收尾。
// 旧实现（任务清单队列驱动「下一阶段」）已删除，本测试同时防止其回流。
package agent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// TestSessionManager_AutopilotSuperviseResume 监督续跑：一轮 continue + 一轮 done。
func TestSessionManager_AutopilotSuperviseResume(t *testing.T) {
	// 固定走默认 Go 内核工厂（不受插件装配器影响，保证测试确定性）
	restore := ReplaceLoopFactory(goLoopFactory{})
	defer restore()

	// ── 决策器：第 1 次要求继续，第 2 次判定完成；记录收到的请求供断言 ──
	vm := goja.New()
	if _, err := vm.RunString(`globalThis.__seen = [];`); err != nil {
		t.Fatalf("初始化 goja: %v", err)
	}
	fnVal, err := vm.RunString(`(function (req) {
	  globalThis.__seen.push(req);
	  if (req.round === 1) {
	    return { action: 'continue', task: '继续：跑 go test ./... 验证后汇报输出', assessment: '声称完成但没有验证证据' };
	  }
	  return { action: 'done', assessment: '已核对验证输出，任务达成' };
	})`)
	if err != nil {
		t.Fatalf("构造决策器失败: %v", err)
	}
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		t.Fatal("决策器不是函数")
	}
	impl := &jsAutopilotImpl{id: "test-autopilot", fn: fn, vm: vm} // plugin=nil：单测直接执行
	restoreAP := RegisterJSAutopilot(impl)
	defer restoreAP()

	dir := t.TempDir()
	reg := NewRegistry()
	reg.Register(&Tool{
		Name: "noop", Description: "dummy", Parameters: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil },
	})
	// 两次 Run 各自然结束一次（无 tool_call + 有正文）
	prov := &MockProvider{Responses: []Message{
		{Role: RoleAssistant, Content: "我已完成自主模式插件的实现。"},
		{Role: RoleAssistant, Content: "已跑通测试并附上输出。"},
	}}

	m := NewSessionManager()
	convID := "conv-supervise"
	opts := LoopOpts{
		Provider:      prov,
		Registry:      reg,
		WorkspaceRoot: dir,
		Autonomous:    true, // 自主模式开关（前端「自主」按钮 → 每次请求传参）
	}
	if err := m.Start(context.Background(), convID, "实现自主模式（插件化）", opts); err != nil {
		t.Fatalf("Start: %v", err)
	}

	ch := m.Subscribe(convID)
	var mu sync.Mutex
	var notices []string
	if ch != nil {
		go func() {
			for e := range ch {
				if e.Type == EventNotice {
					mu.Lock()
					notices = append(notices, e.Content)
					mu.Unlock()
				}
			}
		}()
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && m.IsRunning(convID) {
		time.Sleep(20 * time.Millisecond)
	}
	if m.IsRunning(convID) {
		t.Fatal("会话未在 20s 内结束（监督续跑可能失控）")
	}
	time.Sleep(300 * time.Millisecond) // 等 fan-out 投递完末段事件

	// ① 两次工作 agent Run（首次任务 + 监督者指令续跑）
	if prov.Calls() != 2 {
		t.Errorf("LLM 调用 = %d, want 2（首次 + 监督续跑一次）", prov.Calls())
	}
	// ② 监督者指令作为新任务进入历史（可溯源）。
	//   注：本测试的决策器直接返回指令（不含前缀）；生产路径由 autopilot 插件加
	//   「【监督者指令·第 N 次监督】」来源前缀（见 .pair/plugins/autopilot/index.js）。
	hist := m.GetHistory(convID)
	var injected []string
	for _, msg := range hist {
		if msg.Role == RoleUser && strings.Contains(msg.Content, "跑 go test ./... 验证") {
			injected = append(injected, msg.Content)
		}
	}
	if len(injected) != 1 {
		t.Fatalf("应有 1 条监督者指令进入历史，得 %d（历史共 %d 条）", len(injected), len(hist))
	}
	// ③ 决策器被调用两次（第二次判定 done 后收尾，不再有第三次）
	seenLen, serr := vm.RunString(`globalThis.__seen.length`)
	if serr != nil {
		t.Fatalf("读取决策器调用记录失败: %v", serr)
	}
	if got := int(seenLen.ToInteger()); got != 2 {
		t.Errorf("决策器调用次数 = %d, want 2（done 后必须收尾）", got)
	}
	// ④ 决策请求字段：目标/汇报/轮次正确传递（插件据此构造任务书）
	req1, jerr := vm.RunString(`JSON.stringify(globalThis.__seen[0])`)
	if jerr != nil {
		t.Fatalf("读取决策请求失败: %v", jerr)
	}
	s := req1.String()
	if !strings.Contains(s, "实现自主模式") {
		t.Errorf("决策请求缺少用户目标：%s", s)
	}
	if !strings.Contains(s, "我已完成自主模式插件的实现") {
		t.Errorf("决策请求缺少工作 agent 本轮汇报：%s", s)
	}
	// ⑤ 用户可见通知（继续/完成各一条）
	mu.Lock()
	snapshot := append([]string(nil), notices...)
	mu.Unlock()
	var sawContinue, sawDone bool
	for _, n := range snapshot {
		if strings.Contains(n, "监督者审核后要求继续") {
			sawContinue = true
		}
		if strings.Contains(n, "监督者判定任务完成") {
			sawDone = true
		}
	}
	if !sawContinue || !sawDone {
		t.Errorf("应推送「要求继续」与「判定完成」通知，实得 %v", snapshot)
	}
}

// TestSessionManager_AutopilotRoundsLimit 监督轮数上限：决策器永要求继续时，宿主按上限收尾。
func TestSessionManager_AutopilotRoundsLimit(t *testing.T) {
	restore := ReplaceLoopFactory(goLoopFactory{})
	defer restore()

	vm := goja.New()
	fnVal, err := vm.RunString(`(function (req) {
	  return { action: 'continue', task: '继续（第 ' + req.round + ' 次要求）' };
	})`)
	if err != nil {
		t.Fatalf("构造决策器失败: %v", err)
	}
	fn, _ := goja.AssertFunction(fnVal)
	impl := &jsAutopilotImpl{id: "always-continue", fn: fn, vm: vm}
	restoreAP := RegisterJSAutopilot(impl)
	defer restoreAP()

	dir := t.TempDir()
	reg := NewRegistry()
	reg.Register(&Tool{Name: "noop", Description: "dummy", Parameters: map[string]any{"type": "object"},
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil }})
	// 永远自然结束（正文非空），由监督轮数上限收敛
	prov := &MockProvider{Responses: []Message{
		{Role: RoleAssistant, Content: "完成一版"}, {Role: RoleAssistant, Content: "又完成一版"},
		{Role: RoleAssistant, Content: "再完成一版"}, {Role: RoleAssistant, Content: "最后完成一版"},
	}}

	m := NewSessionManager()
	convID := "conv-supervise-limit"
	opts := LoopOpts{
		Provider:           prov,
		Registry:           reg,
		WorkspaceRoot:      dir,
		Autonomous:         true,
		MaxSuperviseRounds: 2, // 上限 2：首轮 + 2 次监督续跑 = 3 次 Run，之后收尾
	}
	if err := m.Start(context.Background(), convID, "任务", opts); err != nil {
		t.Fatalf("Start: %v", err)
	}

	ch := m.Subscribe(convID)
	var mu sync.Mutex
	var notices []string
	if ch != nil {
		go func() {
			for e := range ch {
				if e.Type == EventNotice {
					mu.Lock()
					notices = append(notices, e.Content)
					mu.Unlock()
				}
			}
		}()
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && m.IsRunning(convID) {
		time.Sleep(20 * time.Millisecond)
	}
	if m.IsRunning(convID) {
		t.Fatal("会话未在 20s 内结束（监督轮数上限未生效）")
	}
	time.Sleep(300 * time.Millisecond)

	if prov.Calls() != 3 {
		t.Errorf("LLM 调用 = %d, want 3（首轮 + 上限 2 次续跑）", prov.Calls())
	}
	mu.Lock()
	snapshot := append([]string(nil), notices...)
	mu.Unlock()
	found := false
	for _, n := range snapshot {
		if strings.Contains(n, "监督轮数已达上限") {
			found = true
		}
	}
	if !found {
		t.Errorf("达监督轮数上限应推送通知，实得 %v", snapshot)
	}
}
