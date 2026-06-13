package agent

// 真机端到端冒烟：用真实 LLM key 跑完整 agent.Loop（建文件→读回），验证 LLM + 工具调用 + 循环闭环。
// 仅当设了环境变量 DEEPSEEK_KEY（或 LIVE_LLM_KEY）才跑；无 key 自动跳过，不影响离线 CI。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func liveKey() string {
	if k := os.Getenv("DEEPSEEK_KEY"); k != "" {
		return k
	}
	return os.Getenv("LIVE_LLM_KEY")
}

func TestLiveDeepSeek(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	prov := &OpenAIProvider{
		BaseURL:     "https://api.deepseek.com/v1",
		APIKey:      key,
		Model:       "deepseek-chat",
		Temperature: 0,
	}
	var ev []string
	loop := &Loop{
		Provider:      prov,
		Registry:      reg,
		System:        DefaultSystemPrompt([]string{root}),
		MaxIterations: 12,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, string(e.Type)+":"+e.Tool)
			} else {
				ev = append(ev, string(e.Type))
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	task := "在工作区创建文件 hello.txt，内容恰好是 goui works，然后用 read_file 读回确认。完成后回复并输出 [FINAL]。"
	msgs, err := loop.Run(ctx, task, nil)
	if err != nil {
		t.Fatalf("loop.Run 出错: %v（事件: %v）", err, ev)
	}
	data, rerr := os.ReadFile(filepath.Join(root, "hello.txt"))
	if rerr != nil {
		t.Fatalf("hello.txt 未被创建: %v（事件: %v）", rerr, ev)
	}
	if !strings.Contains(string(data), "goui works") {
		t.Errorf("文件内容不符: %q", string(data))
	}
	t.Logf("✓ 真机通过：%d 条消息；hello.txt=%q", len(msgs), string(data))
	t.Logf("事件流: %v", ev)
}

// TestLiveMemoryKB 真机验证较新子系统：记忆（含 MEMORY.md 索引）+ 项目知识库工具被真实 LLM 正确驱动。
func TestLiveMemoryKB(t *testing.T) {
	key := liveKey()
	if key == "" {
		t.Skip("未设 DEEPSEEK_KEY，跳过真机测试")
	}
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	prov := &OpenAIProvider{BaseURL: "https://api.deepseek.com/v1", APIKey: key, Model: "deepseek-chat", Temperature: 0}
	var ev []string
	loop := &Loop{
		Provider: prov, Registry: reg, System: DefaultSystemPrompt([]string{root}), MaxIterations: 16,
		OnEvent: func(e Event) {
			if e.Tool != "" {
				ev = append(ev, e.Tool)
			}
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	task := "依次完成三步：①用 memory_write 记一条记忆（name 用中文如「测试记忆」，type=project，描述和正文随便写中文）；" +
		"②用 project_info_write 写知识库条目（path=概览，内容首行 # 项目概览）；③用 memory_list 看记忆总览。完成后输出 [FINAL]。"
	if _, err := loop.Run(ctx, task, nil); err != nil {
		t.Fatalf("loop.Run 出错: %v（工具: %v）", err, ev)
	}
	joined := strings.Join(ev, " ")
	if !strings.Contains(joined, "memory_write") {
		t.Errorf("LLM 未调用 memory_write，工具序列: %v", ev)
	}
	if !strings.Contains(joined, "project_info_write") {
		t.Errorf("LLM 未调用 project_info_write，工具序列: %v", ev)
	}
	memEntries, _ := os.ReadDir(filepath.Join(root, ".pair", "memory"))
	if len(memEntries) < 2 { // ≥1 记忆文件 + MEMORY.md 索引
		t.Errorf("记忆目录应含记忆文件 + MEMORY.md 索引，得 %d 项", len(memEntries))
	}
	kbEntries, _ := os.ReadDir(filepath.Join(root, ".pair", "project-info"))
	if len(kbEntries) < 1 {
		t.Errorf("知识库目录应有条目，得 %d 项", len(kbEntries))
	}
	t.Logf("✓ 记忆+知识库真机通过；记忆目录 %d 项、知识库 %d 项；工具: %v", len(memEntries), len(kbEntries), ev)
}
