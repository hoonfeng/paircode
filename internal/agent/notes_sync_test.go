package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNotesWriteSync notes/ 前缀路径写入：自动映射树分支 + 镜像 .agents/notes/ + 扫描不重复。
func TestNotesWriteSync(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	// 映射表：notes/implemented/architecture → 架构/、implemented/feature → 实现/、
	// implemented/process → 关键点/、decision → 设计思想/、其余 → 实现/
	cases := []struct{ notesPath, branchPath string }{
		{"notes/implemented/architecture/2026-08-15-x", "架构/2026-08-15-x"},
		{"notes/implemented/feature/2026-08-15-y", "实现/2026-08-15-y"},
		{"notes/implemented/process/2026-08-15-z", "关键点/2026-08-15-z"},
		{"notes/decision/2026-08-15-d", "设计思想/2026-08-15-d"},
		{"notes/inbox/2026-08-15-i", "实现/2026-08-15-i"},
	}
	for _, c := range cases {
		got, err := reg.Execute(ctx, "project_info_write", `{"path":"`+c.notesPath+`","content":"# 测试笔记\n内容"}`)
		if err != nil {
			t.Fatalf("write %s: %v", c.notesPath, err)
		}
		if !strings.Contains(got, c.branchPath) {
			t.Errorf("返回应含映射分支路径 %s：%q", c.branchPath, got)
		}
		// 树副本存在
		treeFP := filepath.Join(root, ".pair", "project-info", filepath.FromSlash(c.branchPath)+".md")
		if _, err := os.Stat(treeFP); err != nil {
			t.Errorf("树副本应存在 %s：%v", treeFP, err)
		}
		// 镜像存在
		mirrorFP := filepath.Join(root, ".agents", "notes", filepath.FromSlash(strings.TrimPrefix(c.notesPath, "notes/"))+".md")
		if _, err := os.Stat(mirrorFP); err != nil {
			t.Errorf("镜像应存在 %s：%v", mirrorFP, err)
		}
	}

	// 扫描：每条只出现一次（树分支路径），无 notes/ 前缀重复条目
	entries := scanInfoEntries(filepath.Join(root, ".pair", "project-info"))
	seen := map[string]bool{}
	for _, e := range entries {
		if seen[e.Path] {
			t.Errorf("重复条目：%s", e.Path)
		}
		seen[e.Path] = true
		if strings.HasPrefix(e.Path, "notes/") {
			t.Errorf("已镜像条目不应以 notes/ 前缀出现：%s", e.Path)
		}
	}
	for _, c := range cases {
		branch := strings.Split(c.branchPath, "/")[0]
		if !seen[c.branchPath] {
			t.Errorf("扫描应含树条目 %s", c.branchPath)
		}
		if branch == "实现" {
			continue
		}
		// 分支目录含该条目（list 树形可见）
		list, _ := reg.Execute(ctx, "project_info_list", "{}")
		if !strings.Contains(list, strings.Split(c.branchPath, "/")[1]) {
			t.Errorf("list 树应含 %s 下条目：\n%s", branch, list)
		}
	}
}

// TestNotesToBranchRel 映射函数边界：空/纯相对/未知前缀。
func TestNotesToBranchRel(t *testing.T) {
	cases := map[string]string{
		"implemented/architecture/a":             "架构/a",
		"implemented/feature/b":                  "实现/b",
		"implemented/process/c":                  "关键点/c",
		"implemented/decision/d":                 "设计思想/d",
		"implemented/e":                          "关键点/e",
		"decision/f":                             "设计思想/f",
		"inbox/g":                                "实现/g",
		"notes/implemented/architecture/a":       "架构/a",
		"other/h":                                "实现/h",
		"implemented/architecture/sub/deep/x":    "架构/x",
		"implemented/feature/2026-08-15-x.md":    "实现/2026-08-15-x.md",
		"notes/implemented/process/2026-08-15-p": "关键点/2026-08-15-p",
	}
	for in, want := range cases {
		got, ok := notesToBranchRel(in)
		if !ok || got != want {
			t.Errorf("notesToBranchRel(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
	if _, ok := notesToBranchRel(""); ok {
		t.Error("空路径应返回 false")
	}
	if _, ok := notesToBranchRel("notes/"); ok {
		t.Error("notes/ 应返回 false")
	}
}
