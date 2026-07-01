package agent

// 阶段五端到端真机测试（5.3-5.7）：用真实 LLM 验证工具层/MCP/Skills/多 agent 编排。
// 仅当设了环境变量 DEEPSEEK_KEY（或 LIVE_LLM_KEY）才跑；无 key 自动跳过，不影响离线 CI。
//
// 覆盖场景：
//   5.3  edit_file CRLF 文件编辑（验证 edit_matcher 归一化匹配 + CRLF 保留）
//   5.4  find_files_by_pattern glob ** 递归匹配（验证 glob.go 的 ** 语义）
//   5.5  MCP go-sdk 端到端（进程内 server + 真实 LLM 调用 mcp.test.echo）
//   5.6  Skills L1 注入 + L2 load_skill（用 config/skills/emoji-icons 真实技能）
//   5.7  多 agent 委托（coordinator → coder，delegate_task + finish_task）

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ─── 5.3 edit_file CRLF 场景 ──

func TestLiveEditFileCRLF(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	crlfPath := filepath.Join(root, "crlf.txt")
	// 创建 CRLF 文件：LLM 用 LF 的 old_string 也能命中（edit_matcher 归一化匹配）
	if err := os.WriteFile(crlfPath, []byte("line1\r\nline2 old\r\nline3\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: DefaultSystemPrompt([]string{root}), MaxIterations: 10,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	task := "把文件 crlf.txt 中的 'line2 old' 改为 'line2 new'（用 edit_file），然后用 read_file 读回确认内容。完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	data, rerr := os.ReadFile(crlfPath)
	if rerr != nil {
		t.Fatalf("读取 crlf.txt 失败: %v", rerr)
	}
	if !strings.Contains(string(data), "line2 new") {
		t.Errorf("edit 后应含 'line2 new'，得 %q", string(data))
	}
	if !strings.Contains(string(data), "line2 new\r\n") {
		t.Errorf("CRLF 应被保留（edit_matcher restoreNewlines），得 %q", string(data))
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "edit_file") {
		t.Errorf("LLM 未调用 edit_file，工具序列: %v", ev)
	}
	t.Logf("✓ edit_file CRLF 真机通过；内容=%q；工具: %v", string(data), ev)
}

// ─── 5.4 glob ** 递归 ──

func TestLiveGlobRecursive(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	// 创建嵌套 .go 文件：src/a.go、src/sub/b.go、src/sub/deep/c.go
	for _, d := range []string{"src", "src/sub", "src/sub/deep"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	for _, f := range []string{"src/a.go", "src/sub/b.go", "src/sub/deep/c.go"} {
		os.WriteFile(filepath.Join(root, f), []byte("package main\n"), 0o644)
	}
	// 一个 .ts 干扰文件（不应被 **/*.go 匹配）
	os.WriteFile(filepath.Join(root, "src/x.ts"), []byte("// ts\n"), 0o644)

	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: DefaultSystemPrompt([]string{root}), MaxIterations: 10,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	task := "用 find_files_by_pattern 工具，pattern 参数填 '**/*.go'，查找工作区所有 Go 文件。列出找到的文件路径。完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "find_files_by_pattern") {
		t.Errorf("LLM 未调用 find_files_by_pattern，工具序列: %v", ev)
	}
	t.Logf("✓ glob ** 真机通过；工具: %v", ev)
}

// ─── 5.5 MCP go-sdk 端到端 ──

func TestLiveMCPInProcess(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	// 进程内 MCP server，提供 echo 工具（复用 mcp_test.go 的 startTestServer/newTestConn）
	echoSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string", "description": "要回显的文本"},
		},
		"required": []string{"text"},
	}
	clientT, sCancel := startTestServer(t, 100, testTool{
		def: &mcp.Tool{Name: "echo", Description: "原样回显输入文本", InputSchema: echoSchema},
		handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echoed: hello mcp"}}}, nil
		},
	})
	defer sCancel()
	conn := newTestConn(t, "test", clientT)
	defer conn.close()

	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	n, err := registerClientTools(reg, conn)
	if err != nil || n == 0 {
		t.Fatalf("注册 MCP 工具失败: n=%d err=%v", n, err)
	}

	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: DefaultSystemPrompt([]string{root}), MaxIterations: 10,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	task := "调用 mcp.test.echo 工具，参数 text 填 'hello mcp'，告诉我返回结果。完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "mcp.test.echo") {
		t.Errorf("LLM 未调用 mcp.test.echo，工具序列: %v", ev)
	}
	t.Logf("✓ MCP go-sdk 真机通过；工具: %v", ev)
}

// ─── 5.6 Skills L1 注入 + L2 load_skill ──

func TestLiveSkills(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	// 指向真实 config/skills 目录（测试工作目录为 cmd/companion/agent/）
	skillDir, _ := filepath.Abs(filepath.Join("..", "..", "config", "skills"))
	origDir := SkillSystemDir
	SkillSystemDir = skillDir
	defer func() { SkillSystemDir = origDir }()

	skills := LoadAllSkills()
	if FindSkill(skills, "emoji-icons") == nil {
		t.Skipf("emoji-icons 技能未找到（skillDir=%s），跳过", skillDir)
	}
	// L1 验证：PromptSkills 应含 emoji-icons（系统提示注入）
	l1 := PromptSkills(skills)
	if !strings.Contains(l1, "emoji-icons") {
		t.Errorf("L1 提示词应含 emoji-icons: %q", l1)
	}

	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	// 注册简易 load_skill 工具（复刻 agenttools.loadSkillFull 逻辑，避免 import agenttools 循环依赖）
	reg.Register(&Tool{
		Name: "load_skill", Description: "加载某技能的完整 SKILL.md 正文", ReadOnly: true,
		Parameters: objSchema(props{"name": strProp("技能名")}, "name"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name, _ := args["name"].(string)
			ss := LoadAllSkills()
			s := FindSkill(ss, name)
			if s == nil {
				return "", fmt.Errorf("未找到技能 %q", name)
			}
			return "# 技能：" + s.Name + "\n" + s.Description + "\n\n" + SkillBodyWithTools(*s), nil
		},
	})

	sysPrompt := DefaultSystemPrompt([]string{root}) + l1
	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: sysPrompt, MaxIterations: 10,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	task := "用 load_skill 工具加载 'emoji-icons' 技能的正文，然后告诉我这个技能的核心规则是什么。完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "load_skill") {
		t.Errorf("LLM 未调用 load_skill，工具序列: %v", ev)
	}
	t.Logf("✓ Skills L1+L2 真机通过；L1 含 emoji-icons；工具: %v", ev)
}

// ─── 5.7 多 agent 委托 ──

func TestLiveMultiAgent(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)

	// 编排树：coordinator（协调器）→ coder（编码者）
	tree := NewAgentTree(
		&SubAgent{Name: "coordinator", Description: "协调器，分配任务给子 agent"},
		&SubAgent{Name: "coder", Description: "编码者，创建文件",
			System:  "你是编码专家，用 write_file 完成文件创建任务，完成后用 finish_task 工具报告结果。",
			MaxIter: 6},
	)

	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: DefaultSystemPrompt([]string{root}), MaxIterations: 15,
		AgentTree: tree, State: map[string]any{},
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	RegisterDelegateTools(loop, tree)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	task := "用 delegate_task 委托 'coder' 子 agent 在工作区创建文件 done.txt，内容为 'delegated ok'。等子 agent 完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "delegate_task") {
		t.Errorf("LLM 未调用 delegate_task，工具序列: %v", ev)
	}
	data, rerr := os.ReadFile(filepath.Join(root, "done.txt"))
	if rerr != nil {
		t.Errorf("done.txt 未被创建: %v（工具: %v）", rerr, ev)
	} else if !strings.Contains(string(data), "delegated") {
		t.Errorf("done.txt 内容不符: %q", string(data))
	}
	t.Logf("✓ 多 agent 委托真机通过；done.txt=%q；工具: %v", string(data), ev)
}
