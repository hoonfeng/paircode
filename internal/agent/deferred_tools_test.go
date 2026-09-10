package agent

// deferred_tools_test.go — 按需工具（deferred）与 tool_search 机制单测。
// 覆盖：默认隐藏 / 搜索命中提升 / 未发现拦截 / 无命中引导 / Copy-Subset 继承 /
// 保留名单一致性。对应 codex ToolExposure::Deferred 语义（2026-09-12）。

import (
	"context"
	"strings"
	"testing"
)

// TestDeferredToolsHiddenByDefault 按需工具默认不进 LLM 工具面。
func TestDeferredToolsHiddenByDefault(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "read", Handler: noopHandler})
	reg.Register(&Tool{Name: "csv_read", Handler: noopHandler})    // deferred
	reg.Register(&Tool{Name: "binary_hash", Handler: noopHandler}) // deferred

	names := map[string]bool{}
	for _, d := range reg.Definitions() {
		names[d.Function.Name] = true
	}
	if !names["read"] {
		t.Fatal("read 应可见")
	}
	if names["csv_read"] || names["binary_hash"] {
		t.Fatalf("按需工具不应进 Definitions：%v", names)
	}
	for _, n := range reg.EnabledNames() {
		if n == "csv_read" || n == "binary_hash" {
			t.Fatalf("按需工具不应进 EnabledNames：%s", n)
		}
	}
	for _, m := range reg.AllToolMeta() {
		if m.Name == "csv_read" && !m.Deferred {
			t.Fatal("csv_read 元信息应标 deferred")
		}
	}
}

// TestDeferredToolSearchAndDiscover tool_search 命中 → 本会话可见可调用。
func TestDeferredToolSearchAndDiscover(t *testing.T) {
	reg := NewRegistry()
	RegisterToolSearchTool(reg, "")
	reg.Register(&Tool{Name: "csv_read", Description: "读取 CSV 文件", Handler: noopHandler})
	reg.Register(&Tool{Name: "read_pdf", Description: "读取 PDF 文档", Handler: noopHandler})
	reg.Register(&Tool{Name: "read", Description: "读取文件", Handler: noopHandler})

	out, err := reg.Execute(context.Background(), "tool_search", `{"query":"csv"}`)
	if err != nil {
		t.Fatalf("tool_search 执行失败: %v", err)
	}
	if !strings.Contains(out, "csv_read") {
		t.Fatalf("tool_search 应命中 csv_read：%q", out)
	}
	// 发现后进 Definitions
	found := false
	for _, d := range reg.Definitions() {
		if d.Function.Name == "csv_read" {
			found = true
		}
	}
	if !found {
		t.Fatal("发现后 csv_read 应进 Definitions")
	}
	// 未命中的 read_pdf 仍隐藏
	for _, d := range reg.Definitions() {
		if d.Function.Name == "read_pdf" {
			t.Fatal("read_pdf 未发现不应可见")
		}
	}
	// 之后可直接执行
	if _, err := reg.Execute(context.Background(), "csv_read", `{}`); err != nil {
		t.Fatalf("发现后 csv_read 应可执行: %v", err)
	}
}

// TestDeferredToolExecuteStillAllowed 按需工具的「可见性」与「可执行性」分离：
// 未发现时不在 LLM 工具面，但直接执行（历史回放/内部路径）不受限——
// 对齐 codex ToolExposure::Deferred（仅 omit from model-visible list）。
func TestDeferredToolExecuteStillAllowed(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "read_pdf", Handler: noopHandler})
	// 未发现：不进工具面
	for _, d := range reg.Definitions() {
		if d.Function.Name == "read_pdf" {
			t.Fatal("未发现的按需工具不应进工具面")
		}
	}
	// 但可直接执行（不拦截）
	if _, err := reg.Execute(context.Background(), "read_pdf", `{}`); err != nil {
		t.Fatalf("按需工具直接执行不应被拦截（仅工具面隐藏）: %v", err)
	}
}

// TestDeferredNoHit 无命中时返回域概览引导（模型可换词再搜）。
func TestDeferredNoHit(t *testing.T) {
	reg := NewRegistry()
	RegisterToolSearchTool(reg, "")
	reg.Register(&Tool{Name: "csv_read", Description: "读取 CSV 文件", Handler: noopHandler})
	out, err := reg.Execute(context.Background(), "tool_search", `{"query":"量子计算"}`)
	if err != nil {
		t.Fatalf("tool_search 执行失败: %v", err)
	}
	if !strings.Contains(out, "未找到") {
		t.Fatalf("无命中应提示未找到：%q", out)
	}
}

// TestDeferredCopySubsetInherits 发现状态在 Copy/Subset 快照继承（会话隔离不泄漏）。
func TestDeferredCopySubsetInherits(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "csv_read", Handler: noopHandler})
	reg.MarkToolDiscovered("csv_read")
	if !reg.IsToolDiscovered("csv_read") {
		t.Fatal("mark 后应已发现")
	}
	cp := reg.Copy()
	if !cp.IsToolDiscovered("csv_read") {
		t.Fatal("Copy 应继承发现状态")
	}
	sub := reg.Subset([]string{"csv_read"})
	if !sub.IsToolDiscovered("csv_read") {
		t.Fatal("Subset 应继承发现状态")
	}
	// 子表发现不影响父表（独立 map）
	sub2 := reg.Copy()
	sub2.MarkToolDiscovered("other")
	if reg.IsToolDiscovered("other") {
		t.Fatal("子表发现不应泄漏到父表")
	}
}

// TestDeferredToolSearchIsDirectAndAligned tool_search 自身恒可见且属保留名单。
func TestDeferredToolSearchIsDirectAndAligned(t *testing.T) {
	reg := NewRegistry()
	RegisterToolSearchTool(reg, "")
	if !reg.IsEnabled("tool_search") {
		t.Fatal("tool_search 应默认启用")
	}
	vis := false
	for _, d := range reg.Definitions() {
		if d.Function.Name == "tool_search" {
			vis = true
		}
	}
	if !vis {
		t.Fatal("tool_search 应恒在 Definitions（发现入口不得自身被隐藏）")
	}
	if !HarnessAlignedToolNames["tool_search"] {
		t.Fatal("HarnessAlignedToolNames 应含 tool_search")
	}
	if isDeferredToolName("tool_search") {
		t.Fatal("tool_search 自身不应是按需工具")
	}
}

// TestDeferredNamesRegistered 名单项均为有效格式（防拼写错漏的静态检查）。
func TestDeferredNamesRegistered(t *testing.T) {
	if len(DeferredToolNames) == 0 {
		t.Fatal("DeferredToolNames 不应为空")
	}
	for n := range DeferredToolNames {
		if strings.TrimSpace(n) == "" || strings.ContainsAny(n, " \t") {
			t.Fatalf("非法工具名：%q", n)
		}
		// ★ 2026-09 工具面合并：单名工具（无下划线）白名单
		if n == "cordis" {
			continue
		}
		if !strings.Contains(n, "_") {
			t.Fatalf("工具名应符合命名规范（含 _）：%q", n)
		}
	}
}
