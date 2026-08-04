package agent

import (
	"context"
	"strings"
	"testing"
)

// 发散/收敛的模拟响应（MockProvider 按序返回）。
var (
	mockDivergeResp = Message{
		Role: RoleAssistant,
		Content: `{"candidates":[
			{"name":"直接修改","idea":"直接编辑目标文件","firstTool":"read_file","firstAction":"读取文件","risk":"可能不了解全貌"},
			{"name":"先调查","idea":"先搜索定位再修改","firstTool":"search_content","firstAction":"搜索关键词","risk":"耗时较长"},
			{"name":"分步验证","idea":"小步修改逐步验证","firstTool":"run_command","firstAction":"运行测试","risk":"需要环境"}]}`,
	}
	mockConvergeResp = Message{
		Role: RoleAssistant,
		Content: `{"plan":"先搜索定位目标，再读取文件确认内容，最后修改并验证。","firstTool":"search_content","order":["read_file","edit_file"],"acceptance":"修改后测试通过"}`,
	}
	mockMainFinish = Message{Role: RoleAssistant, Content: "任务完成，已按计划执行。"}
)

// TestDiffusionThinkInjectsPlan 开启扩散思考：发散+收敛成功 → 首轮注入计划。
func TestDiffusionThinkInjectsPlan(t *testing.T) {
	mp := &MockProvider{Responses: []Message{mockDivergeResp, mockConvergeResp, mockMainFinish}}
	reg := NewRegistry()
	loop := &Loop{
		Provider:      mp,
		Registry:      reg,
		System:        "你是测试 Agent。",
		MaxIterations: 5,
		DiffusionThink: DiffusionThinkOpts{
			Enabled:    true,
			Candidates: 3,
		},
	}
	msgs, err := loop.Run(context.Background(), "查找并修改某个函数的实现", nil)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("消息过少: %d", len(msgs))
	}
	if loop.DiffuseStats == nil || !loop.DiffuseStats.Triggered {
		t.Fatalf("扩散思考未触发: %+v", loop.DiffuseStats)
	}
	if loop.DiffuseStats.Candidates != 3 {
		t.Errorf("候选数 = %d, 期望 3", loop.DiffuseStats.Candidates)
	}
	if loop.DiffuseStats.SuggestedTool != "search_content" {
		t.Errorf("建议首步工具 = %q, 期望 search_content", loop.DiffuseStats.SuggestedTool)
	}
	if !strings.Contains(loop.DiffuseStats.Plan, "先搜索定位目标") {
		t.Errorf("收敛计划未解析: %q", loop.DiffuseStats.Plan)
	}
	if loop.DiffuseStats.TotalTokens <= 0 {
		t.Errorf("扩散 token 统计缺失: %+v", loop.DiffuseStats)
	}
	// 主模型首轮调用应收到注入消息（含【策略预演】标记）
	// MockProvider 第 3 次调用（主模型）的消息列表末尾应有注入
	if mp.calls != 3 {
		t.Errorf("Chat 调用次数 = %d, 期望 3（发散+收敛+主模型）", mp.calls)
	}
	_ = msgs
}

// TestDiffusionThinkDisabled 默认关闭：不触发、不注入。
func TestDiffusionThinkDisabled(t *testing.T) {
	mp := &MockProvider{Responses: []Message{mockMainFinish}}
	reg := NewRegistry()
	loop := &Loop{
		Provider:      mp,
		Registry:      reg,
		System:        "你是测试 Agent。",
		MaxIterations: 5,
	}
	msgs, err := loop.Run(context.Background(), "测试任务", nil)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if loop.DiffuseStats != nil {
		t.Errorf("默认关闭时 DiffuseStats 应为 nil, got %+v", loop.DiffuseStats)
	}
	if mp.calls != 1 {
		t.Errorf("Chat 调用次数 = %d, 期望 1（不触发扩散）", mp.calls)
	}
	_ = msgs
}

// TestDiffusionThinkParseFail 发散解析失败：不注入、正常继续。
func TestDiffusionThinkParseFail(t *testing.T) {
	mp := &MockProvider{Responses: []Message{
		{Role: RoleAssistant, Content: "抱歉，我无法生成 JSON。"}, // 发散失败
		mockMainFinish,
	}}
	reg := NewRegistry()
	loop := &Loop{
		Provider:      mp,
		Registry:      reg,
		System:        "你是测试 Agent。",
		MaxIterations: 5,
		DiffusionThink: DiffusionThinkOpts{
			Enabled: true,
		},
	}
	msgs, err := loop.Run(context.Background(), "测试任务", nil)
	if err != nil {
		t.Fatalf("Run 失败: %v", err)
	}
	if loop.DiffuseStats == nil {
		t.Fatalf("应有统计对象")
	}
	if loop.DiffuseStats.Triggered {
		t.Errorf("发散失败不应触发注入")
	}
	if !loop.DiffuseStats.ParseFail {
		t.Errorf("应标记 ParseFail")
	}
	if mp.calls != 2 {
		t.Errorf("Chat 调用次数 = %d, 期望 2（发散失败 + 主模型）", mp.calls)
	}
	_ = msgs
}

// TestDiffuseThinkOptsDefault 默认参数填充。
func TestDiffuseThinkOptsDefault(t *testing.T) {
	d := (DiffusionThinkOpts{Enabled: true}).default_()
	if d.Candidates != 3 {
		t.Errorf("默认候选数 = %d, 期望 3", d.Candidates)
	}
	if d.MaxTokens != 800 {
		t.Errorf("默认 MaxTokens = %d, 期望 800", d.MaxTokens)
	}
	d2 := (DiffusionThinkOpts{Enabled: true, Candidates: 9, MaxTokens: 100}).default_()
	if d2.Candidates != 5 {
		t.Errorf("候选数超限 = %d, 期望 5", d2.Candidates)
	}
}

// TestParseDivergeConverge 解析函数容错（带前后杂讯）。
func TestParseDivergeConverge(t *testing.T) {
	raw := "好的，以下是候选：\n```json\n{\"candidates\":[{\"name\":\"A\",\"idea\":\"i\",\"firstTool\":\"t\",\"firstAction\":\"a\",\"risk\":\"r\"}]}\n```"
	cands, ok := parseDiverge(raw)
	if !ok || len(cands) != 1 || cands[0].FirstTool != "t" {
		t.Errorf("parseDiverge 失败: %v ok=%v", cands, ok)
	}
	raw2 := "计划如下：{\"plan\":\"p\",\"firstTool\":\"f\",\"order\":[\"o1\"],\"acceptance\":\"a\"}"
	cr, ok := parseConverge(raw2)
	if !ok || cr.Plan != "p" || cr.FirstTool != "f" {
		t.Errorf("parseConverge 失败: %+v ok=%v", cr, ok)
	}
}
