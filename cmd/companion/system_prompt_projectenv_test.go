package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
)

// TestBuildWebSystemPrompt_ProjectEnvStaysInDynamicSide 验证 S1 重构后组装链完整：
// 项目环境段仍在完整系统提示里（agent 依然看得到环境说明），且位于**唯一 CacheBoundary 之后**
// （动态侧——它是会话特定内容，不进静态前缀）。
func TestBuildWebSystemPrompt_ProjectEnvStaysInDynamicSide(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".pair"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".pair", "project.md"),
		[]byte("# 项目环境档案\n\n- 标记 marker-xyz-42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldFolders := core.Folders
	core.Folders = []string{dir}
	defer func() { core.Folders = oldFolders }()

	sys := buildWebSystemPrompt("conv-sys-assembly-test")
	if !strings.Contains(sys, "# 项目环境") {
		t.Fatalf("完整系统提示应包含「# 项目环境」段（S1 重构不得丢段）")
	}
	if !strings.Contains(sys, "marker-xyz-42") {
		t.Fatalf("项目环境段应包含 .pair/project.md 内容")
	}
	if n := strings.Count(sys, agent.CacheBoundary); n != 1 {
		t.Fatalf("必须只有一个 CacheBoundary，实际 %d 个", n)
	}
	idx := strings.Index(sys, agent.CacheBoundary)
	if strings.Contains(sys[:idx], "# 项目环境") {
		t.Errorf("项目环境段必须位于 CacheBoundary 之后（动态侧），不得进入静态前缀")
	}
}

// TestProjectEnvSectionCached_FreezesWithinSession 验证 S1（2026-09-13 缓存前缀治理）：
// 项目环境段在**会话内冻结** —— 首次读取后，即便 .pair/project.md 变化（agent 会按系统提示
// 主动改写它），同一 convID 的返回值逐字节不变。这样才能消除「system 动态后缀一变 →
// boundary 之后（含整个历史）全部 miss」这一最现实的前缀断裂源。
func TestProjectEnvSectionCached_FreezesWithinSession(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".pair"), 0o755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, ".pair", "project.md")
	if err := os.WriteFile(envPath, []byte("# 项目环境档案\n\n- 构建：go build\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldFolders := core.Folders
	core.Folders = []string{dir}
	defer func() { core.Folders = oldFolders }()

	convID := "conv-freeze-test"
	first := projectEnvSectionCached(convID)
	if !strings.Contains(first, "构建：go build") {
		t.Fatalf("首读应含 project.md 内容，实际=%q", first)
	}

	// 模拟 agent 自改 .pair/project.md（系统提示约定要求解决环境问题后更新它）
	if err := os.WriteFile(envPath, []byte("# 项目环境档案\n\n- 构建：go build\n- 新增：CGO_ENABLED=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if second := projectEnvSectionCached(convID); second != first {
		t.Errorf("会话内必须冻结：同一 convID 返回值应逐字节不变\n first=%q\nsecond=%q", first, second)
	}

	// 新会话应读到最新内容（冻结只作用于单个会话）
	third := projectEnvSectionCached("conv-other")
	if !strings.Contains(third, "CGO_ENABLED=1") {
		t.Errorf("新会话应读取最新 project.md，实际=%q", third)
	}
	if third == first {
		t.Errorf("新会话不应复用其它会话的冻结值")
	}
}

// TestProjectEnvSectionCached_NoSessionKey 验证 convID 为空（启动预热等无会话上下文）时
// 使用固定键冻结，保证同源请求逐字节一致。
func TestProjectEnvSectionCached_NoSessionKey(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".pair"), 0o755); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, ".pair", "project.md")
	if err := os.WriteFile(envPath, []byte("# 项目环境档案\n\n- v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldFolders := core.Folders
	core.Folders = []string{dir}
	defer func() { core.Folders = oldFolders }()

	a := projectEnvSectionCached("")
	if err := os.WriteFile(envPath, []byte("# 项目环境档案\n\n- v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b := projectEnvSectionCached(""); a != b {
		t.Errorf("convID 为空时应按固定键冻结：%q vs %q", a, b)
	}
}

// TestBuildProjectEnvSection_NoFolders 边界：无工作区 → 空段（不注入「# 项目环境」标题）。
func TestBuildProjectEnvSection_NoFolders(t *testing.T) {
	oldFolders := core.Folders
	core.Folders = nil
	defer func() { core.Folders = oldFolders }()
	if s := buildProjectEnvSection(); s != "" {
		t.Errorf("无工作区应返回空段，实际=%q", s)
	}
}
