// ═══════════════════════════════════════════════════════════
// dsh_bridge_test.go — 外部运行时桥测试（dsh 服务面与事件转发，2026-09 策略闸门）
//
// 覆盖：
//   - TestDSHBridgeServices：dshService 服务面（agents/subagents/llm/
//     systemPrompt/commands）参数映射与错误路径
//   - TestBridgeEventSubscription：host 事件订阅白名单门控 + 载荷序列化
//   - TestNodeBridgeAgentStatusEvent：子 Agent 状态变更 → agent/status
//     事件经桥转发（载荷对齐 DSH scheduler 消费面）
//   - TestBridgeSkipsExternalDSHEntries：2026-09 策略端到端——runtime="dsh"
//     外部 dsh 生态条目被桥过滤（零装载/零工具），防 dsh 环境安装的插件影响
//     IDE。原 TestNodeBridgeDSHPluginE2E（npm 装 dsh-agent-teams → 装载 13
//     工具）已随策略废止替换；node 轨（cordis3）装载由 node_bridge_e2e_test.go
//     TestNodeBridgeE2EHelloBridge 继续覆盖。
// ═══════════════════════════════════════════════════════════
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDSHBridgeServices 外部服务面（dshService）映射与错误路径。
func TestDSHBridgeServices(t *testing.T) {
	b := &nodeBridge{} // 无进程：只测映射/直连逻辑
	// 未知服务 → handled=false（走 mapBridgeService 既有路径）
	if handled, _, _ := b.dshService("fs", "read", map[string]any{"path": "x"}, "", nil); handled {
		t.Fatal("fs 服务不应由 dshService 处理")
	}
	// agents.get 未登记 → null
	handled, data, err := b.dshService("agents", "get", map[string]any{"convId": "conv-none"}, "", nil)
	if !handled || err != nil || data != "null" {
		t.Fatalf("agents.get 未登记应返回 null: handled=%v data=%q err=%v", handled, data, err)
	}
	// agents.followup 缺参数
	handled, _, err = b.dshService("agents", "followup", map[string]any{"convId": "c1"}, "", nil)
	if !handled || err == nil {
		t.Fatal("agents.followup 缺 text 应报错")
	}
	// agents.start 缺 task
	handled, _, err = b.dshService("agents", "start", map[string]any{"label": "x"}, "", nil)
	if !handled || err == nil || !strings.Contains(err.Error(), "task") {
		t.Fatalf("agents.start 缺 task 应报错: %v", err)
	}
	// subagents.getProvider('spawn') → 提供者描述
	handled, data, err = b.dshService("subagents", "getProvider", map[string]any{"name": "spawn"}, "", nil)
	if !handled || err != nil || !strings.Contains(data, `"capabilities"`) {
		t.Fatalf("subagents.getProvider(spawn) 异常: data=%q err=%v", data, err)
	}
	// subagents.startContinuable 无 spawner → 明确错误（非挂起）
	handled, _, err = b.dshService("subagents", "startContinuable", map[string]any{
		"label": "agent-teams:t1:m1", "prompt": "welcome", "persona": "you are m1",
		"parentConvId": "conv-cap", "provider2": "deepseek", "model": "deepseek-chat",
	}, "", nil)
	if !handled || err == nil || !strings.Contains(err.Error(), "未就绪") {
		t.Fatalf("startContinuable 无 spawner 应报错: %v", err)
	}
	// llm.current 无 spawner → {}
	handled, data, err = b.dshService("llm", "current", map[string]any{}, "", nil)
	if !handled || err != nil || data != "{}" {
		t.Fatalf("llm.current 空目录应返回 {}: data=%q err=%v", data, err)
	}
	// 未知方法
	if handled, _, _ = b.dshService("agents", "bogus", map[string]any{}, "", nil); !handled {
		t.Fatal("agents.bogus 应 handled")
	}
	if _, _, err = b.dshService("agents", "bogus", map[string]any{}, "", nil); err == nil {
		t.Fatal("agents.bogus 应报错")
	}
}

// TestDSHBridgeSystemPromptCommand 有宿主时 systemPrompt.section 与 commands.register 生效。
func TestDSHBridgeSystemPromptCommand(t *testing.T) {
	reg := NewRegistry()
	ph := NewPluginHost(reg, nil, t.TempDir())
	SetGlobalPluginHost(ph)
	defer SetGlobalPluginHost(nil)
	b := &nodeBridge{}

	handled, data, err := b.dshService("systemPrompt", "section", map[string]any{
		"name": "agent-teams:usage", "order": float64(117), "text": "When the user asks to run AgentTeams...",
	}, "@nanmicoder/dsh-agent-teams", nil)
	if !handled || err != nil || data != "ok" {
		t.Fatalf("systemPrompt.section 异常: data=%q err=%v", data, err)
	}
	found := false
	for _, s := range ph.Sections() {
		if s.Name == "agent-teams:usage" && s.Order == 117 && strings.Contains(s.Text, "AgentTeams") {
			found = true
		}
	}
	if !found {
		t.Fatalf("systemPrompt 段未注册进宿主: %+v", ph.Sections())
	}

	handled, data, err = b.dshService("commands", "register", map[string]any{
		"name": "agent-teams", "description": "run a goal with a multi-agent team",
	}, "@nanmicoder/dsh-agent-teams", nil)
	if !handled || err != nil || data != "ok" {
		t.Fatalf("commands.register 异常: data=%q err=%v", data, err)
	}
	if cmd := FindHostCommand("agent-teams"); cmd == nil {
		t.Fatal("命令未注册进宿主命令表")
	}
	// 宿主执行 → 桥未就绪 → 明确错误（非悬挂）
	if _, err := RunHostCommand("agent-teams", map[string]any{"rawInput": "build X"}); err == nil || !strings.Contains(err.Error(), "桥未就绪") {
		t.Fatalf("无桥执行命令应报错: %v", err)
	}
	// 卸载（unregister）清理
	if handled, _, err = b.dshService("commands", "unregister", map[string]any{}, "@nanmicoder/dsh-agent-teams", nil); !handled || err != nil {
		t.Fatalf("commands.unregister 异常: %v", err)
	}
	if FindHostCommand("agent-teams") != nil {
		t.Fatal("卸载后命令应消失")
	}
	UnregisterHostCommands("node-bridge:@nanmicoder/dsh-agent-teams")
}

// TestBridgeEventSubscription host 事件订阅门控：白名单外零转发。
func TestBridgeEventSubscription(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	b := &nodeBridge{stdin: bufio.NewWriter(pw), subs: map[string]bool{"agent/status": true}}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	// 未订阅事件 → 不写入（管道无数据；用 goroutine 读 + 短超时判断）
	// ★ 2026-08-30：agent/pre-step 已升级为决策型中间件事件（见 dsh_prestep.go），
	//   emitBridgeEvent 单向通道不再使用该名——用观察型 agent/pre-tool 验证门控。
	emitBridgeEvent("agent/pre-tool", map[string]any{"x": 1})
	pw.Close() // 关闭写端 → 读端 EOF（证明无 pending 数据）
	_ = pr.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 4096)
	n, rerr := pr.Read(buf)
	if rerr == nil || n != 0 {
		t.Fatalf("未订阅事件不应转发（n=%d err=%v）", n, rerr)
	}
}

// TestNodeBridgeAgentStatusEvent 子 Agent 状态变更 → agent/status 事件载荷。
func TestNodeBridgeAgentStatusEvent(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	b := &nodeBridge{stdin: bufio.NewWriter(pw), subs: map[string]bool{"agent/status": true}, ready: true}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	// 假 spawner（web 层注入同构）
	SetSubAgentSpawner(&SubAgentSpawner{Start: func(spec SubAgentSpec) error { return nil }})
	defer SetSubAgentSpawner(nil)

	rec, err := SpawnSubAgent(SubAgentSpec{Task: "调研 X", Label: "t1:r", Team: "t1", Member: "r", WsRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("SpawnSubAgent 失败: %v", err)
	}

	// 读桥输出：应收到 t=event name=agent/status 载荷 {agent:{id,status,session.header.cwd},status}
	_ = pw.Close()
	scanner := bufio.NewScanner(pr)
	got := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && scanner.Scan() {
		var msg struct {
			T       string          `json:"t"`
			Name    string          `json:"name"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.T != "event" || msg.Name != "agent/status" {
			continue
		}
		var payload struct {
			Agent  map[string]any `json:"agent"`
			Status string         `json:"status"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("事件载荷解析失败: %v", err)
		}
		if payload.Status != "running" {
			t.Fatalf("状态应 running，实际 %q", payload.Status)
		}
		if payload.Agent["id"] != rec.ConvID {
			t.Fatalf("事件 agent.id 应 %s 实际 %v", rec.ConvID, payload.Agent["id"])
		}
		session, _ := payload.Agent["session"].(map[string]any)
		header, _ := session["header"].(map[string]any)
		if cwd, _ := header["cwd"].(string); cwd == "" {
			t.Fatal("事件载荷应含 session.header.cwd（DSH scheduler 消费面）")
		}
		got = true
		break
	}
	if !got {
		t.Fatalf("未收到 agent/status 事件（subs=%v）", b.subs)
	}
}

// TestNodeBridgeDSHPluginE2E 端到端：npm 安装 dsh-agent-teams → cordis4 装载 →
// 13 个 agent_teams_* 工具注册 → create/status/delete 冒烟（临时端口、零 FATAL）。
// 环境不满足（无 node/npm/网络）时 Skip——与 t1「npm 安装需网络（环境依赖除外）」一致。
// TestBridgeSkipsExternalDSHEntries 2026-09 策略端到端验证：plugins.json 中
// runtime="dsh"（外部 dsh 生态插件）条目被桥过滤——桥正常 ready、零插件装载、
// 零工具注册（dsh 条目跳过即不 import 任何外部包，无需 npm 安装/网络）。
// 防止 dsh 环境安装的插件影响 IDE 工具面（round4 验证残留曾致桥反复崩溃）。
// ★ 原 TestNodeBridgeDSHPluginE2E（真实 npm 装 dsh-agent-teams → 装载 13 工具）
//   已随策略废止替换为闸门验证：外部 dsh 生态不再桥接，node 轨（cordis3）
//   装载路径由 node_bridge_e2e_test.go TestNodeBridgeE2EHelloBridge 继续覆盖。
func TestBridgeSkipsExternalDSHEntries(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("无 node 运行时")
	}
	dir := t.TempDir()
	// 桥脚本落盘（与真实桥同源：embed bridge_node.js）
	if err := os.WriteFile(filepath.Join(dir, "bridge.js"), []byte(bridgeNodeJS), 0o644); err != nil {
		t.Fatal(err)
	}
	// 仅 dsh 条目（round4 验证残留形态：@nanmicoder/dsh-agent-teams）
	pluginsJSON := `{"plugins":[{"spec":"@nanmicoder/dsh-agent-teams@0.1.14","runtime":"dsh"}]}`
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	bcmd := exec.Command(nodePath, filepath.Join(dir, "bridge.js"))
	bcmd.Dir = dir
	bcmd.Env = append(os.Environ(), "CORDIS_BRIDGE_DIR="+dir, "CORDIS_WORKSPACE_ROOT="+dir)
	stdout, _ := bcmd.StdoutPipe()
	bcmd.Stderr = os.Stderr
	if err := bcmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bcmd.Process.Kill(); _, _ = bcmd.Process.Wait() }()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	gotReady := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) && !gotReady {
		if !scanner.Scan() {
			t.Fatalf("bridge 提前退出: %v", scanner.Err())
		}
		var msg struct {
			T       string   `json:"t"`
			Plugins []string `json:"plugins"`
			Tools   []string `json:"tools"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue // 非 JSON 行（console 日志）跳过
		}
		if msg.T == "ready" {
			gotReady = true
			if len(msg.Plugins) != 0 {
				t.Fatalf("dsh 条目应被桥过滤（零装载），实际 %v", msg.Plugins)
			}
			if len(msg.Tools) != 0 {
				t.Fatalf("dsh 条目不应注册任何工具，实际 %d 个", len(msg.Tools))
			}
		}
	}
	if !gotReady {
		t.Fatal("桥未 ready（30s 超时）")
	}
	t.Log("闸门生效：外部 dsh 生态条目被过滤，桥正常 ready（零工具/零插件）")
}



// ═══════════════════════════════════════════════════════════
// dsh_prestep_test.go 段 —— DSH agent/pre-step 中间件瀑布桥
//
// 覆盖：
//   - TestDSHMsgConversion：Go Message ↔ DSH message 往返保真
//     （user/assistant(tool-call)/tool(tool-result)/system/reasoning）
//   - TestBridgePreStepDecision：applyPreStepDecision 决策解析
//     （enter 改写 / reject 拒绝 / 未知 kind 直通）
//   - TestBridgePreStepGate：无订阅者零开销直通（不写管道）
// ═══════════════════════════════════════════════════════════

// TestDSHMsgConversion 外部消息转换往返保真。
func TestDSHMsgConversion(t *testing.T) {
	in := []Message{
		{Role: RoleSystem, Content: "你是一名资深工程师。"},
		{Role: RoleUser, Content: "修复登录超时 bug"},
		{Role: RoleAssistant, Content: "我来分析。", Reasoning: "先看代码", ToolCalls: []ToolCall{
			{ID: "call-1", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"a.go"}`}},
		}},
		{Role: RoleTool, Content: "文件不存在", ToolCallID: "call-1"},
		{Role: RoleUser, Content: "继续", Images: []ImagePart{{Data: "data:image/png;base64,AAAA", MimeType: "image/png"}}},
	}
	dsh := msgsToDSH(in)
	// 外部结构抽查
	if dsh[0]["role"] != "system" || dsh[1]["role"] != "user" || dsh[2]["role"] != "assistant" {
		t.Fatalf("role 映射异常: system=%v user=%v assistant=%v", dsh[0]["role"], dsh[1]["role"], dsh[2]["role"])
	}
	// tool 消息 → role=user + source.kind=tool
	if dsh[3]["role"] != "user" {
		t.Fatalf("tool 消息应转 role=user，实际 %v", dsh[3]["role"])
	}
	src, _ := dsh[3]["source"].(map[string]any)
	if src["kind"] != "tool" || src["callId"] != "call-1" {
		t.Fatalf("tool source 异常: %+v", src)
	}
	// 往返保真
	back := dshToMsgs(dsh)
	if len(back) != len(in) {
		t.Fatalf("往返消息数不一致: %d vs %d", len(back), len(in))
	}
	for i := range in {
		if back[i].Role != in[i].Role {
			t.Fatalf("[%d] role 往返丢失: %s vs %s", i, back[i].Role, in[i].Role)
		}
		if back[i].Content != in[i].Content {
			t.Fatalf("[%d] content 往返丢失: %q vs %q", i, back[i].Content, in[i].Content)
		}
	}
	if back[2].Reasoning != in[2].Reasoning {
		t.Fatalf("reasoning 往返丢失: %q vs %q", back[2].Reasoning, in[2].Reasoning)
	}
	if len(back[2].ToolCalls) != 1 || back[2].ToolCalls[0].ID != "call-1" || back[2].ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("tool-call 往返丢失: %+v", back[2].ToolCalls)
	}
	if back[3].ToolCallID != "call-1" || back[3].Content != "文件不存在" || back[3].Role != RoleTool {
		t.Fatalf("tool-result 往返丢失: %+v", back[3])
	}
	if len(back[4].Images) != 1 || back[4].Images[0].Data != "data:image/png;base64,AAAA" {
		t.Fatalf("image 往返丢失: %+v", back[4].Images)
	}
}

// TestBridgePreStepDecision 决策 JSON 解析：enter/reject/未知。
func TestBridgePreStepDecision(t *testing.T) {
	// reject
	if _, reject, err := applyPreStepDecision(`{"kind":"reject"}`); err != nil || !reject {
		t.Fatalf("reject 决策异常: reject=%v err=%v", reject, err)
	}
	// enter 改写（追加一条 user 指令）
	rewritten, reject, err := applyPreStepDecision(`{"kind":"enter","messages":[
		{"role":"user","content":[{"type":"text","text":"修复登录超时 bug"}],"source":{"kind":"user"}},
		{"role":"user","content":[{"type":"text","text":"Activate AgentTeams protocol."}],"source":{"kind":"agent-teams-command"}}
	]}`)
	if err != nil || reject {
		t.Fatalf("enter 决策异常: reject=%v err=%v", reject, err)
	}
	if len(rewritten) != 2 || rewritten[1].Content != "Activate AgentTeams protocol." {
		t.Fatalf("enter 改写未生效: %+v", rewritten)
	}
	// 未知 kind → 直通
	if _, reject, err := applyPreStepDecision(`{"kind":"bogus"}`); err != nil || reject {
		t.Fatalf("未知 kind 应直通: reject=%v err=%v", reject, err)
	}
	// 非法 JSON → 报错
	if _, _, err := applyPreStepDecision(`not-json`); err == nil {
		t.Fatal("非法决策 JSON 应报错")
	}
}

// TestBridgePreStepGate 无订阅者时零开销直通（不写管道）。
func TestBridgePreStepGate(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	b := &nodeBridge{stdin: bufio.NewWriter(pw), subs: map[string]bool{"agent/status": true}, ready: true}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	// 未订阅 agent/pre-step → 直通且管道无数据
	if !bridgePreStepSubscribed() {
		t.Log("未订阅确认（零开销门控）")
	} else {
		t.Fatal("未订阅时不应判定为已订阅")
	}
	rewritten, reject, err := bridgePreStep(context.Background(), []Message{{Role: RoleUser, Content: "x"}}, 1, 1)
	if err != nil || reject || rewritten != nil {
		t.Fatalf("未订阅应直通: rewritten=%v reject=%v err=%v", rewritten, reject, err)
	}
	pw.Close()
	_ = pr.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 4096)
	n, rerr := pr.Read(buf)
	if rerr == nil || n != 0 {
		t.Fatalf("未订阅时不应写桥（n=%d err=%v）", n, rerr)
	}
}

// TestBridgePreStepSubscribed 已订阅时发起 prestep 请求并解析回包。
func TestBridgePreStepSubscribed(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pr.Close()
	b := &nodeBridge{stdin: bufio.NewWriter(pw), subs: map[string]bool{"agent/pre-step": true}, ready: true, pending: map[int64]chan bridgeResult{}}
	globalNodeBridge = b
	defer func() { globalNodeBridge = nil }()

	if !bridgePreStepSubscribed() {
		t.Fatal("已订阅时应判定为已订阅")
	}
	// 后台读桥输出并模拟 Node 回包（enter 改写：追加指令）
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			var msg struct {
				T       string `json:"t"`
				ID      int64  `json:"id"`
				Payload struct {
					Messages []map[string]any `json:"messages"`
					Turn     int              `json:"turn"`
				} `json:"payload"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				continue
			}
			if msg.T != "prestep" {
				continue
			}
			if len(msg.Payload.Messages) != 1 || msg.Payload.Turn != 3 {
				t.Errorf("prestep 载荷异常: %+v", msg.Payload)
			}
			b.mu.Lock()
			ch := b.pending[msg.ID]
			delete(b.pending, msg.ID)
			b.mu.Unlock()
			if ch != nil {
				ch <- bridgeResult{ok: true, data: `{"kind":"enter","messages":[
					{"role":"user","content":[{"type":"text","text":"原消息"}],"source":{"kind":"user"}},
					{"role":"user","content":[{"type":"text","text":"Activated"}],"source":{"kind":"agent-teams-command"}}
				]}`}
			}
		}
	}()

	rewritten, reject, err := bridgePreStep(context.Background(), []Message{{Role: RoleUser, Content: "原消息"}}, 3, 1)
	if err != nil || reject {
		t.Fatalf("桥瀑布失败: reject=%v err=%v", reject, err)
	}
	if len(rewritten) != 2 || rewritten[1].Content != "Activated" {
		t.Fatalf("改写未生效: %+v", rewritten)
	}
}
