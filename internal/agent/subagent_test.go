// subagent_test.go 测试 AgentTree 构造/查找 + Registry.Copy/Subset 白名单裁剪。

package agent

import (
	"context"
	"testing"
)

func TestNewAgentTree(t *testing.T) {
	root := &SubAgent{Name: "coordinator", Description: "协调器"}
	planner := &SubAgent{Name: "planner", Description: "规划"}
	coder := &SubAgent{Name: "coder", Description: "编码"}
	tree := NewAgentTree(root, planner, coder)

	if tree.Root == nil || tree.Root.Name != "coordinator" {
		t.Errorf("Root 应为 coordinator")
	}
	if tree.Find("coordinator") == nil {
		t.Error("应找到 root")
	}
	if tree.Find("planner") == nil {
		t.Error("应找到 planner")
	}
	if tree.Find("missing") != nil {
		t.Error("不应找到 missing")
	}
	if tree.ParentOf("planner") != "coordinator" {
		t.Errorf("planner 父应为 coordinator，得 %q", tree.ParentOf("planner"))
	}
	if tree.ParentOf("coordinator") != "" {
		t.Errorf("root 父应为空，得 %q", tree.ParentOf("coordinator"))
	}
	subs := tree.SubNames()
	if len(subs) != 2 {
		t.Errorf("子 agent 应 2 个，得 %d", len(subs))
	}
}

func TestAgentTree_Add(t *testing.T) {
	root := &SubAgent{Name: "root"}
	tree := NewAgentTree(root)
	reviewer := &SubAgent{Name: "reviewer", Description: "审查"}
	tree.Add(reviewer, "root")
	if tree.Find("reviewer") == nil {
		t.Error("Add 后应找到 reviewer")
	}
	if tree.ParentOf("reviewer") != "root" {
		t.Error("reviewer 父应为 root")
	}
}

func TestRegistry_Copy(t *testing.T) {
	noop := func(_ context.Context, _ map[string]any) (string, error) { return "", nil }
	reg := NewRegistry()
	reg.Register(&Tool{Name: "a", Handler: noop})
	cp := reg.Copy()
	cp.Register(&Tool{Name: "b", Handler: noop})

	if _, ok := reg.Get("b"); ok {
		t.Error("Copy 后子表新增不应影响原表")
	}
	if _, ok := cp.Get("a"); !ok {
		t.Error("Copy 应含原工具 a")
	}
	if _, ok := cp.Get("b"); !ok {
		t.Error("Copy 应含新工具 b")
	}
	// 钩子应共享
	reg.BeforeTool = func(_ context.Context, _ string, _ map[string]any) (bool, string, error) { return true, "", nil }
	cp2 := reg.Copy()
	if cp2.BeforeTool == nil {
		t.Error("Copy 应继承钩子")
	}
}

func TestRegistry_Subset(t *testing.T) {
	noop := func(_ context.Context, _ map[string]any) (string, error) { return "", nil }
	reg := NewRegistry()
	reg.Register(&Tool{Name: "read_file", Handler: noop})
	reg.Register(&Tool{Name: "write_file", Handler: noop})
	reg.Register(&Tool{Name: "edit_file", Handler: noop})

	sub := reg.Subset([]string{"read_file"})
	if _, ok := sub.Get("read_file"); !ok {
		t.Error("Subset 应含 read_file")
	}
	if _, ok := sub.Get("write_file"); ok {
		t.Error("Subset 不应含 write_file")
	}
	if _, ok := sub.Get("edit_file"); ok {
		t.Error("Subset 不应含 edit_file")
	}
	// Definitions 顺序应只含白名单
	defs := sub.Definitions()
	if len(defs) != 1 || defs[0].Function.Name != "read_file" {
		t.Errorf("Subset Definitions 应只含 read_file，得 %v", defs)
	}
}
