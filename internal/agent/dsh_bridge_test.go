// ═══════════════════════════════════════════════════════════
// dsh_bridge_test.go — Round4 DSH 运行时桥测试
//
// 覆盖：
//   - TestDSHBridgeServices：dshService 服务面（agents/subagents/llm/
//     systemPrompt/commands）参数映射与错误路径
//   - TestBridgeEventSubscription：host 事件订阅白名单门控 + 载荷序列化
//   - TestNodeBridgeAgentStatusEvent：子 Agent 状态变更 → agent/status
//     事件经桥转发（载荷对齐 DSH scheduler 消费面）
//   - TestNodeBridgeDSHPluginE2E：真实 npm 安装 dsh-agent-teams →
//     cordis4 装载 → 13 个 agent_teams_* 工具注册 → 冒烟调用
//     （create/status/delete 两阶段）→ 磁盘状态落盘校验
//     环境不满足（无 node/npm/网络）时自动 Skip（环境依赖除外）。
// ═══════════════════════════════════════════════════════════
package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDSHBridgeServices DSH 服务面（dshService）映射与错误路径。
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
	emitBridgeEvent("agent/pre-step", map[string]any{"x": 1})
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
func TestNodeBridgeDSHPluginE2E(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("无 node 运行时")
	}
	npmCmd := "npm"
	if _, err := exec.LookPath(npmCmd); err != nil {
		npmCmd = "npm.cmd"
	}
	if _, err := exec.LookPath(npmCmd); err != nil {
		t.Skip("无 npm 运行时")
	}

	const pkgSpec = "@nanmicoder/dsh-agent-teams@0.1.14"
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "npm-cache")

	// 1. npm install 插件 + DSH peer（桥 package.json 与 npmInstallPlugin 同构）
	pkgJSON := `{"name":"cordis-bridge","version":"0.1.0","private":true,"dependencies":{"@cordisjs/core":"latest"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	// DSH peer 版本集：与 npmInstallDshPeers 的 devDependencies 优先策略一致
	// （插件构建/测试锁定的版本：cordis 稳定 4.0.1 + dsh-* rc.8——peer 范围
	// ^0.1.0-rc.6 与传递 peer rc.8 存在档位错配会 ERESOLVE，见 round4 实施记录）
	peers := []string{
		"@deepseek-ai/cordis@4.0.1",
		"@deepseek-ai/dsh-agent@0.1.0-rc.8",
		"@deepseek-ai/dsh-llm@0.1.0-rc.8",
		"@deepseek-ai/dsh-session@0.1.0-rc.8",
		"@deepseek-ai/dsh-subagent@0.1.0-rc.8",
		"@deepseek-ai/dsh-tools@0.1.0-rc.8",
		"@deepseek-ai/dsh-system-prompt@0.1.0-rc.8",
		"@deepseek-ai/dsh-commands@0.1.0-rc.8",
		"@deepseek-ai/schemastery@^3.18.1",
	}
	args := append([]string{"install", "--no-audit", "--no-fund", "--prefix", dir,
		"--fetch-retries", "3", "--fetch-retry-mintimeout", "2000", "--fetch-retry-maxtimeout", "10000",
		pkgSpec}, peers...)
	// 瞬时网络抖动防护：最多重试 2 次完整安装（npm registry 偶发 TLS/超时）
	var installErr error
	var installOut []byte
	for attempt := 0; attempt < 2; attempt++ {
		ictx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		cmd := exec.CommandContext(ictx, npmCmd, args...)
		cmd.Env = append(os.Environ(), "NPM_CONFIG_CACHE="+cacheDir)
		installOut, installErr = cmd.CombinedOutput()
		cancel()
		if installErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if installErr != nil {
		detail := strings.TrimSpace(string(installOut))
		// 附带 eresolve 报告（诊断关键信息）
		if report := readNewestFile(filepath.Join(cacheDir, "_logs"), "*-eresolve-report.txt"); report != "" {
			detail = report
		}
		if len(detail) > 1500 {
			detail = detail[len(detail)-1500:]
		}
		t.Skipf("npm install 失败（网络/环境依赖，跳过 E2E）: %v\n%s", installErr, detail)
	}

	// 2. bridge.js + plugins.json（runtime=dsh）
	if err := os.WriteFile(filepath.Join(dir, "bridge.js"), []byte(bridgeNodeJS), 0o644); err != nil {
		t.Fatal(err)
	}
	pluginsJSON := fmt.Sprintf(`{"plugins":[{"spec":%q,"runtime":"dsh"}]}`, pkgSpec)
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), []byte(pluginsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. spawn 桥（真实 node；装载 cordis4 + DSH 门面）
	bcmd := exec.Command(nodePath, filepath.Join(dir, "bridge.js"))
	bcmd.Dir = dir
	bcmd.Env = append(os.Environ(), "CORDIS_BRIDGE_DIR="+dir, "CORDIS_WORKSPACE_ROOT="+dir)
	stdin, _ := bcmd.StdinPipe()
	stdout, _ := bcmd.StdoutPipe()
	bcmd.Stderr = os.Stderr
	if err := bcmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bcmd.Process.Kill(); _, _ = bcmd.Process.Wait() }()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	toolsSeen := map[string]bool{}
	gotReady := false
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) && !gotReady {
		if !scanner.Scan() {
			t.Fatalf("bridge 提前退出: %v", scanner.Err())
		}
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		switch msg["t"] {
		case "tool":
			if def, ok := msg["def"].(map[string]any); ok {
				if name, ok := def["name"].(string); ok {
					toolsSeen[name] = true
				}
			}
		case "ready":
			gotReady = true
		}
	}
	if !gotReady {
		t.Fatalf("桥未 ready（已见工具 %d 个）", len(toolsSeen))
	}
	// 13 个 agent_teams_* 工具注册（t1 验收标准）
	const wantTools = 13
	gotTools := 0
	for name := range toolsSeen {
		if strings.HasPrefix(name, "agent_teams_") {
			gotTools++
		}
	}
	if gotTools != wantTools {
		t.Fatalf("agent_teams_* 工具应 %d 个，实际 %d（全部工具: %v）", wantTools, gotTools, toolsSeen)
	}

	// 4. 冒烟调用：create（staged 两阶段）→ status → delete
	invoke := func(tool string, args map[string]any, convID string) (string, error) {
		if _, err := stdin.Write([]byte(fmt.Sprintf(`{"t":"invoke","id":7,"tool":%q,"args":%s,"convId":%q,"wsRoot":%q}`+"\n",
			tool, mustJSON(args), convID, dir))); err != nil {
			return "", err
		}
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if !scanner.Scan() {
				return "", fmt.Errorf("无响应: %v", scanner.Err())
			}
			var resp struct {
				T    string `json:"t"`
				ID   int64  `json:"id"`
				OK   bool   `json:"ok"`
				Data string `json:"data"`
				Err  string `json:"error"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				continue
			}
			if resp.T == "log" {
				continue
			}
			if resp.T != "result" || resp.ID != 7 {
				continue
			}
			if !resp.OK {
				return "", fmt.Errorf("invoke 失败: %s", resp.Err)
			}
			return resp.Data, nil
		}
		return "", fmt.Errorf("invoke %s 超时", tool)
	}
	const captainID = "conv-captain-e2e"
	// create（approval=required → staged）
	data, err := invoke("agent_teams_create", map[string]any{
		"name": "demo-team", "approval": "required", "description": "E2E smoke",
	}, captainID)
	if err != nil {
		t.Fatalf("agent_teams_create 失败: %v", err)
	}
	var created struct {
		TeamID string `json:"team_id"`
		Phase  string `json:"phase"`
	}
	if err := json.Unmarshal([]byte(data), &created); err != nil || created.TeamID == "" {
		t.Fatalf("create 输出解析失败: %v data=%s", err, data)
	}
	if created.Phase != "staged" {
		t.Fatalf("approval=required 应 staged，实际 %q", created.Phase)
	}
	// 磁盘状态落盘（.agent-teams/<id>/team.json）
	stateFile := filepath.Join(dir, ".agent-teams", created.TeamID, "team.json")
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("团队状态未落盘 %s: %v", stateFile, err)
	}
	// status（队长视角快照）
	data, err = invoke("agent_teams_status", map[string]any{}, captainID)
	if err != nil {
		t.Fatalf("agent_teams_status 失败: %v", err)
	}
	var status struct {
		TeamID string `json:"team_id"`
		Phase  string `json:"phase"`
	}
	if err := json.Unmarshal([]byte(data), &status); err != nil || status.TeamID != created.TeamID || status.Phase != "staged" {
		t.Fatalf("status 输出异常: %v data=%s", err, data)
	}
	// delete（归档清理）
	data, err = invoke("agent_teams_delete", map[string]any{}, captainID)
	if err != nil {
		t.Fatalf("agent_teams_delete 失败: %v", err)
	}
	if !strings.Contains(data, `"deleted":true`) {
		t.Fatalf("delete 输出异常: %s", data)
	}
	t.Logf("DSH 插件 E2E 通过：13 工具注册 + create/status/delete 冒烟 + 状态落盘（%s）", created.TeamID)
}

// readNewestFile 读目录中最新的匹配文件（npm 诊断报告用；无则空串）。
func readNewestFile(dir, pattern string) string {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) == 0 {
		return ""
	}
	// Glob 返回排序结果；取最后一个（时间戳文件名递增）
	b, err := os.ReadFile(matches[len(matches)-1])
	if err != nil {
		return ""
	}
	return string(b)
}
