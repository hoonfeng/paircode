// parallel_test.go 并行执行机制测试：共享上下文 + 冲突检测 + 并行编排 + 工具注册。

package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ─── SharedContext 基础操作 ───────────────────────────────────

func TestSharedContext_FileCache(t *testing.T) {
	pool := NewSharedContext()

	// 写缓存
	pool.StoreFile("agent1", "main.go", "package main")
	pool.StoreFile("agent1", "util.go", "package util")

	// 读缓存
	content, ok := pool.LoadFile("main.go")
	if !ok || content != "package main" {
		t.Errorf("读缓存失败：%q, %v", content, ok)
	}

	if !pool.HasFile("main.go") {
		t.Error("HasFile 应返回 true")
	}
	if pool.HasFile("nonexistent.go") {
		t.Error("HasFile 对不存在的文件应返回 false")
	}

	// CachedFiles
	files := pool.CachedFiles()
	if len(files) != 2 || files[0] != "main.go" || files[1] != "util.go" {
		t.Errorf("CachedFiles 结果错：%v", files)
	}
}

func TestSharedContext_Knowledge(t *testing.T) {
	pool := NewSharedContext()

	// 添加知识
	pool.AddKnowledge("planner", "arch", "MVC架构", "项目使用MVC架构")
	pool.AddKnowledge("coder", "db", "MySQL 8.0", "数据库使用MySQL")

	// 读知识
	entry, ok := pool.GetKnowledge("arch")
	if !ok || entry.Value != "MVC架构" || entry.Summary != "项目使用MVC架构" {
		t.Errorf("GetKnowledge arch 错：%+v", entry)
	}

	// 覆盖（隔开时间确保排序稳定）
	time.Sleep(time.Millisecond)
	pool.AddKnowledge("reviewer", "arch", "MVC+DDD", "架构改为MVC+DDD")
	entry, ok = pool.GetKnowledge("arch")
	if !ok || entry.Value != "MVC+DDD" {
		t.Errorf("覆盖后 arch 应为 MVC+DDD，得 %q", entry.Value)
	}

	// 不存在
	_, ok = pool.GetKnowledge("nonexistent")
	if ok {
		t.Error("不存在的 key 应返回 false")
	}

	// GetAllKnowledge 时序
	all := pool.GetAllKnowledge()
	if len(all) != 2 {
		t.Errorf("应有 2 条知识，得 %d", len(all))
	}
	// 最新的在前
	if all[0].Key != "arch" {
		t.Errorf("arch 应最新（后写入），得 %q", all[0].Key)
	}
}

func TestSharedContext_Variables(t *testing.T) {
	pool := NewSharedContext()

	// 设置变量
	pool.SetVar("planner", "db_host", "localhost")
	pool.SetVar("coder", "db_host", "127.0.0.1")

	// 读当前值（最新版本）
	val, ok := pool.GetVar("db_host")
	if !ok || val != "127.0.0.1" {
		t.Errorf("GetVar 应返回最新值，得 %q, %v", val, ok)
	}

	// 未设置
	_, ok = pool.GetVar("nonexistent")
	if ok {
		t.Error("不存在的变量应返回 false")
	}

	// 版本历史
	history := pool.GetVarHistory("db_host")
	if len(history) != 2 {
		t.Fatalf("应有 2 个版本，得 %d", len(history))
	}
	if history[0].AgentName != "planner" || history[1].AgentName != "coder" {
		t.Errorf("版本顺序错：%+v", history)
	}
}

func TestSharedContext_NoConflictSameValue(t *testing.T) {
	pool := NewSharedContext()
	pool.SetVar("agent1", "port", "8080")
	pool.SetVar("agent2", "port", "8080") // 同值不冲突
	conflicts := pool.DetectConflicts()
	if len(conflicts) != 0 {
		t.Errorf("同值不应冲突，得 %+v", conflicts)
	}
}

func TestSharedContext_Conflict(t *testing.T) {
	pool := NewSharedContext()
	pool.SetVar("agent1", "port", "8080")
	pool.SetVar("agent2", "port", "9090") // 不同值 → 冲突

	conflicts := pool.DetectConflicts()
	if len(conflicts) != 1 {
		t.Fatalf("应检测到 1 个冲突，得 %d", len(conflicts))
	}
	if conflicts[0].Name != "port" {
		t.Errorf("冲突变量名应为 port，得 %q", conflicts[0].Name)
	}
	if len(conflicts[0].Versions) != 2 {
		t.Errorf("应有 2 个版本，得 %d", len(conflicts[0].Versions))
	}
}

func TestSharedContext_SameAgentNoConflict(t *testing.T) {
	pool := NewSharedContext()
	pool.SetVar("agent1", "x", "1")
	pool.SetVar("agent1", "x", "2") // 同一 Agent 更新，不算冲突
	conflicts := pool.DetectConflicts()
	if len(conflicts) != 0 {
		t.Errorf("同一 Agent 更新不应算冲突，得 %+v", conflicts)
	}
}

// ─── KnowledgeSummary ──────────────────────────────────────

func TestSharedContext_KnowledgeSummary(t *testing.T) {
	pool := NewSharedContext()
	// 空池
	if sum := pool.KnowledgeSummary(); sum != "" {
		t.Errorf("空池应返回空，得 %q", sum)
	}
	pool.AddKnowledge("planner", "arch", "MVC", "MVC架构")
	pool.StoreFile("coder", "main.go", "package main")
	sum := pool.KnowledgeSummary()
	if !strings.Contains(sum, "MVC架构") || !strings.Contains(sum, "main.go") {
		t.Errorf("摘要应含知识和文件，得 %q", sum)
	}
}

// ─── BuildContextInject ─────────────────────────────────────

func TestSharedContext_BuildContextInject(t *testing.T) {
	pool := NewSharedContext()
	pool.AddKnowledge("planner", "arch", "MVC", "MVC架构")
	pool.StoreFile("coder", "main.go", "package main")

	inject := pool.BuildContextInject("executor")
	if !strings.Contains(inject, "MVC架构") {
		t.Error("注入应含知识摘要")
	}
	if !strings.Contains(inject, "main.go") {
		t.Error("注入应含文件列表")
	}
	if !strings.Contains(inject, "ctx_read_file") {
		t.Error("注入应含工具提示")
	}
}

// ─── 并行组构建 ─────────────────────────────────────────────

func TestBuildParallelGroups(t *testing.T) {
	orch := &ParallelOrchestrator{}

	subTasks := []SubTask{
		{ID: "step-1", Dependencies: []string{}},                          // 无依赖 → 组0
		{ID: "step-2", Dependencies: []string{"step-1"}},                 // 依赖 step-1 → 组1
		{ID: "step-3", Dependencies: []string{"step-1"}},                 // 依赖 step-1 → 组1（与 step-2 并行）
		{ID: "step-4", Dependencies: []string{"step-2", "step-3"}},       // 依赖 step-2,3 → 组2
		{ID: "step-5", Dependencies: []string{}},                          // 无依赖 → 组0
	}

	groups := orch.buildParallelGroups(subTasks)

	// 应有 3 组（0, 1, 2）
	if len(groups) != 3 {
		t.Fatalf("应有 3 组，得 %d", len(groups))
	}

	// 组0：step-1, step-5（可并行）
	if len(groups[0]) != 2 {
		t.Errorf("组0 应有 2 个任务，得 %v", groups[0])
	}

	// 组1：step-2, step-3（可并行）
	if len(groups[1]) != 2 {
		t.Errorf("组1 应有 2 个任务，得 %v", groups[1])
	}

	// 组2：step-4（串行）
	if len(groups[2]) != 1 || groups[2][0] != 3 {
		t.Errorf("组2 应有 step-4，得 %v", groups[2])
	}
}

func TestBuildParallelGroups_NoDeps(t *testing.T) {
	orch := &ParallelOrchestrator{}
	subTasks := []SubTask{
		{ID: "a", Dependencies: []string{}},
		{ID: "b", Dependencies: []string{}},
		{ID: "c", Dependencies: []string{}},
	}
	groups := orch.buildParallelGroups(subTasks)
	if len(groups) != 1 || len(groups[0]) != 3 {
		t.Errorf("全部无依赖应全在组0，得 %+v", groups)
	}
}

func TestBuildParallelGroups_Chain(t *testing.T) {
	orch := &ParallelOrchestrator{}
	subTasks := []SubTask{
		{ID: "step-1", Dependencies: []string{}},
		{ID: "step-2", Dependencies: []string{"step-1"}},
		{ID: "step-3", Dependencies: []string{"step-2"}},
	}
	groups := orch.buildParallelGroups(subTasks)
	if len(groups) != 3 {
		t.Fatalf("3 步链应有 3 组，得 %d", len(groups))
	}
	for i, g := range groups {
		if len(g) != 1 {
			t.Errorf("组%d 应有 1 个任务，得 %v", i, g)
		}
	}
}

// ─── 子 Agent 选择 ──────────────────────────────────────────

func TestSelectAgentForTask(t *testing.T) {
	root := &SubAgent{Name: "coordinator", Description: "协调器"}
	tree := NewAgentTree(root,
		&SubAgent{Name: "coder", Description: "编码/代码生成/重构"},
		&SubAgent{Name: "reviewer", Description: "代码审查/质量检查"},
		&SubAgent{Name: "planner", Description: "规划/任务分解"},
	)

	tests := []struct {
		desc string
		want string
	}{
		{"重构用户模块，需要编写新代码", "coder"},
		{"审查代码变更，检查质量问题", "reviewer"},
		{"代码实现一个新功能", "coder"},
		{"分解这个复杂任务", "planner"},
		{"分析架构", ""},
	}

	for _, tc := range tests {
		got := selectAgentForTask(tree, tc.desc)
		if tc.want != "" && got != tc.want {
			t.Errorf("selectAgentForTask(%q) = %q, 期望 %q", tc.desc, got, tc.want)
		}
	}
}

func TestSelectAgentForTask_EmptyTree(t *testing.T) {
	if got := selectAgentForTask(nil, "测试"); got != "" {
		t.Errorf("空树应返回空，得 %q", got)
	}
}

// ─── DecomposeTask ─────────────────────────────────────────

func TestDecomposeTask(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `好的，计划如下：
{"reasoning":"先分析再修改最后验证","steps":[
{"id":"step-1","description":"分析项目结构","dependencies":[]},
{"id":"step-2","description":"修改核心逻辑","dependencies":["step-1"]},
{"id":"step-3","description":"验证修改结果","dependencies":["step-2"]}]}`}}}

	parent := &Loop{
		Provider: mock,
		MaxIterations: 30,
		currentMsgs: nil,
	}

	tree := NewAgentTree(&SubAgent{Name: "coordinator", Description: "协调器"})
	orch := NewParallelOrchestrator(parent, tree, nil)

	result, err := orch.DecomposeTask(context.Background(), "重构配置模块", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.SubTasks) != 3 {
		t.Fatalf("应分解为 3 个子任务，得 %d", len(result.SubTasks))
	}

	if len(result.ParallelGroups) != 3 {
		t.Fatalf("3 步链应分 3 组，得 %d", len(result.ParallelGroups))
	}

	// 组0: step-1; 组1: step-2; 组2: step-3
	if len(result.ParallelGroups[0]) != 1 || result.ParallelGroups[0][0] != 0 {
		t.Errorf("组0 应为 step-1，得 %v", result.ParallelGroups[0])
	}
}

func TestDecomposeTask_Empty(t *testing.T) {
	mock := &MockProvider{Responses: []Message{{Content: `{"reasoning":"只是问候","steps":[]}`}}}
	parent := &Loop{Provider: mock}
	orch := NewParallelOrchestrator(parent, NewAgentTree(&SubAgent{Name: "root"}), nil)
	result, err := orch.DecomposeTask(context.Background(), "你好", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.SubTasks) != 0 {
		t.Errorf("问候应返回空步骤，得 %d", len(result.SubTasks))
	}
}

// ─── AggregateResults ──────────────────────────────────────

func TestAggregateResults(t *testing.T) {
	pool := NewSharedContext()
	orch := &ParallelOrchestrator{ContextPool: pool}

	results := []SubTaskResult{
		{TaskID: "step-1", AgentName: "planner", Output: "分析完成：项目使用MVC架构"},
		{TaskID: "step-2", AgentName: "coder", Output: "修改完成：添加了新API"},
	}

	// 先注入知识
	pool.AddKnowledge("planner", "arch", "MVC", "MVC架构")

	summary := orch.AggregateResults(results)

	if !strings.Contains(summary, "step-1") {
		t.Error("汇总应含 step-1")
	}
	if !strings.Contains(summary, "step-2") {
		t.Error("汇总应含 step-2")
	}
	if !strings.Contains(summary, "MVC架构") {
		t.Error("汇总应含知识")
	}
	if !strings.Contains(summary, "无冲突") {
		t.Error("汇总应含无冲突")
	}
}

func TestAggregateResults_WithConflict(t *testing.T) {
	pool := NewSharedContext()
	pool.SetVar("agent1", "port", "8080")
	pool.SetVar("agent2", "port", "9090")

	orch := &ParallelOrchestrator{ContextPool: pool}
	results := []SubTaskResult{
		{TaskID: "step-1", AgentName: "agent1", Output: "设置了 port=8080"},
		{TaskID: "step-2", AgentName: "agent2", Output: "设置了 port=9090"},
	}

	summary := orch.AggregateResults(results)
	if !strings.Contains(summary, "共享变量冲突") {
		t.Error("有冲突时汇总应含冲突提示")
	}
	if !strings.Contains(summary, "8080") || !strings.Contains(summary, "9090") {
		t.Error("冲突汇总应含具体值")
	}
}

func TestAggregateResults_Empty(t *testing.T) {
	orch := &ParallelOrchestrator{ContextPool: NewSharedContext()}
	summary := orch.AggregateResults(nil)
	if !strings.Contains(summary, "无子任务") {
		t.Errorf("空结果应返回提示，得 %q", summary)
	}
}

// ─── 并行上下文工具注册 ─────────────────────────────────────

func TestRegisterParallelContextTools(t *testing.T) {
	pool := NewSharedContext()
	pool.StoreFile("agent1", "test.go", "package test")
	pool.AddKnowledge("agent1", "key1", "val1", "测试知识")

	reg := NewRegistry()
	registerParallelContextTools(reg, pool, "testAgent")

	// ctx_read_file
	_, ok := reg.Get("ctx_read_file")
	if !ok {
		t.Fatal("ctx_read_file 应已注册")
	}

	// ctx_get_knowledge
	_, ok = reg.Get("ctx_get_knowledge")
	if !ok {
		t.Fatal("ctx_get_knowledge 应已注册")
	}

	// ctx_add_knowledge
	_, ok = reg.Get("ctx_add_knowledge")
	if !ok {
		t.Fatal("ctx_add_knowledge 应已注册")
	}

	// ctx_get_var
	_, ok = reg.Get("ctx_get_var")
	if !ok {
		t.Fatal("ctx_get_var 应已注册")
	}

	// ctx_set_var
	_, ok = reg.Get("ctx_set_var")
	if !ok {
		t.Fatal("ctx_set_var 应已注册")
	}

	// ctx_knowledge_summary
	_, ok = reg.Get("ctx_knowledge_summary")
	if !ok {
		t.Fatal("ctx_knowledge_summary 应已注册")
	}
}

// ─── 端点 -> 工具的路径测试 ───────────────────────────────

// TestRegisterParallelTools 验证并行工具注册。
func TestRegisterParallelTools(t *testing.T) {
	parent := &Loop{
		Registry:      NewRegistry(),
		State:         map[string]any{},
		currentMsgs:   nil,
	}

	tree := NewAgentTree(&SubAgent{Name: "root"})
	RegisterParallelTools(parent, tree, nil)

	// parallel_decompose
	_, ok := parent.Registry.Get("parallel_decompose")
	if !ok {
		t.Fatal("parallel_decompose 应已注册")
	}

	// parallel_execute
	_, ok = parent.Registry.Get("parallel_execute")
	if !ok {
		t.Fatal("parallel_execute 应已注册")
	}

	// ctx_set_var（主 Registry 版）
	_, ok = parent.Registry.Get("ctx_set_var")
	if !ok {
		t.Fatal("ctx_set_var 应已注册到主 Registry")
	}

	// ctx_get_var（主 Registry 版）
	_, ok = parent.Registry.Get("ctx_get_var")
	if !ok {
		t.Fatal("ctx_get_var 应已注册到主 Registry")
	}
}

// TestContextToolExecution 验证上下文工具的 handler 逻辑。
func TestContextToolExecution(t *testing.T) {
	pool := NewSharedContext()
	reg := NewRegistry()
	registerParallelContextTools(reg, pool, "testAgent")

	// 1. ctx_add_knowledge
	handler, ok := reg.Get("ctx_add_knowledge")
	if !ok {
		t.Fatal("ctx_add_knowledge 未注册")
	}
	result, err := handler.Handler(context.Background(), map[string]any{
		"key":     "test",
		"value":   "test_value",
		"summary": "测试知识",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "已添加知识") {
		t.Errorf("结果应含成功提示，得 %q", result)
	}

	// 2. ctx_get_knowledge
	getHandler, _ := reg.Get("ctx_get_knowledge")
	result, err = getHandler.Handler(context.Background(), map[string]any{
		"key": "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "test_value") {
		t.Errorf("应返回知识值，得 %q", result)
	}

	// 3. ctx_set_var
	setVarHandler, _ := reg.Get("ctx_set_var")
	result, err = setVarHandler.Handler(context.Background(), map[string]any{
		"name":  "myvar",
		"value": "myvalue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "已设置") {
		t.Errorf("应含成功提示，得 %q", result)
	}

	// 4. ctx_get_var
	getVarHandler, _ := reg.Get("ctx_get_var")
	result, err = getVarHandler.Handler(context.Background(), map[string]any{
		"name": "myvar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "myvalue") {
		t.Errorf("应返回变量值，得 %q", result)
	}

	// 5. ctx_read_file（未缓存）
	readFileHandler, _ := reg.Get("ctx_read_file")
	result, err = readFileHandler.Handler(context.Background(), map[string]any{
		"path": "nonexistent.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "未在缓存中") {
		t.Errorf("不存在的文件应提示未缓存，得 %q", result)
	}

	// 6. ctx_read_file（已缓存）
	pool.StoreFile("testAgent", "cached.go", "cached content")
	result, err = readFileHandler.Handler(context.Background(), map[string]any{
		"path": "cached.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "cached content") {
		t.Errorf("应返回缓存内容，得 %q", result)
	}

	// 7. ctx_knowledge_summary
	summaryHandler, _ := reg.Get("ctx_knowledge_summary")
	result, err = summaryHandler.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "测试知识") {
		t.Errorf("摘要应含知识，得 %q", result)
	}
}
