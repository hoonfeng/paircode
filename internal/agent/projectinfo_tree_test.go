package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInfoTreeStructure 知识库树形化：分支/子类/条目缩进树 + 叶子「标题（末段）」。
func TestInfoTreeStructure(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	reg.Execute(ctx, "project_info_write", `{"path":"概览","content":"# 项目概览\n树形知识库"}`)
	reg.Execute(ctx, "project_info_write", `{"path":"目标/目标-项目愿景","content":"# 项目愿景\n成为最佳"}`)
	reg.Execute(ctx, "project_info_write", `{"path":"架构/模块-agent","content":"# Agent 引擎模块\n职责说明"}`)
	reg.Execute(ctx, "project_info_write", `{"path":"架构/模块-渲染/细节-光标","content":"# 光标细节\n深层"}`)
	reg.Execute(ctx, "project_info_write", `{"path":"设计思想/决策-树形化","content":"# 决策：树形化\n权衡"}`)

	tree, err := reg.Execute(ctx, "project_info_tree", "{}")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"目标/", "架构/", "设计思想/",        // 分支节点
		"项目愿景", "Agent 引擎模块", "模块-agent", // 叶子标题 + 末段
		"├──", "└──", // 树形连接符
	} {
		if !strings.Contains(tree, want) {
			t.Errorf("tree 应含 %q：\n%s", want, tree)
		}
	}
	// 深层条目归入 架构/模块-渲染/ 分支下
	if i := strings.Index(tree, "模块-渲染/"); i < 0 || !strings.Contains(tree[i:], "光标细节") {
		t.Errorf("深层条目应挂在子分支下：\n%s", tree)
	}
	// 分级标注在 project_info_list（showLevel）；tree 为纯树
	list, err := reg.Execute(ctx, "project_info_list", "{}")
	if err != nil || !strings.Contains(list, "[overview]") || !strings.Contains(list, "[module]") || !strings.Contains(list, "[detail]") {
		t.Errorf("list 应带分级标注（overview/module/detail）：%v\n%s", err, list)
	}

	// 注入（ProjectKnowledge）也应输出树形目录
	kb := ProjectKnowledge(root, 3000)
	if !strings.Contains(kb, "知识库目录（树）") || !strings.Contains(kb, "目标/") {
		t.Errorf("自动注入应为树形目录：\n%s", kb)
	}
}

// TestAgentsNotesCompat .agents/notes/ 参考决策树作为知识库只读附加源（路径前缀 notes/）。
func TestAgentsNotesCompat(t *testing.T) {
	root := t.TempDir()
	notesDir := filepath.Join(root, ".agents", "notes", "implemented", "architecture")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	note := "# Agent Note: 2026-06-11-custom-schema-dsl\n\nStatus: implemented\n\n## Decision\n决定内容"
	if err := os.WriteFile(filepath.Join(notesDir, "2026-06-11-custom-schema-dsl.md"), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := scanInfoEntries(projectInfoDir(root))
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Path, "notes/implemented/architecture/") {
			found = true
			if e.Level != "detail" { // 2+ 层 → detail
				t.Errorf("notes 条目应 detail 级，实际 %s", e.Level)
			}
			if !strings.Contains(e.Content, "Decision") {
				t.Errorf("notes 条目正文应完整读取")
			}
		}
	}
	if !found {
		t.Fatal(".agents/notes 应并入知识库扫描（notes/ 前缀）")
	}

	// project_info_list 树形应含 notes/ 分支
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	if list, err := reg.Execute(context.Background(), "project_info_list", "{}"); err != nil || !strings.Contains(list, "notes/") {
		t.Errorf("list 应含 notes/ 分支：%v %q", err, list)
	}
}

// TestProjectRulesLayered AGENTS.md 分层加载：根 + docs/ + .agents/ 全注入（各标来源）。
func TestProjectRulesLayered(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("AGENTS.md", "# 根约定\n用 4 空格缩进")
	write("docs/AGENTS.md", "# 文档标准\n每个导出有 JSDoc")
	write(".agents/AGENTS.md", "# 流程规则\n门禁必须过")
	write(".pair/rules.md", "# 用户指令\n不要用 AI 配色")

	got := ProjectRules(root)
	for _, want := range []string{"AGENTS.md", "docs/AGENTS.md", ".agents/AGENTS.md",
		"根约定", "文档标准", "流程规则", "项目指令", "用 4 空格缩进", "JSDoc", "门禁必须过", "AI 配色"} {
		if !strings.Contains(got, want) {
			t.Errorf("分层注入应含 %q：\n%s", want, got)
		}
	}
	// 都不存在 → 空
	if ProjectRules(t.TempDir()) != "" {
		t.Error("无任何约定文件应返回空")
	}
}

// TestAgentsSkillsCompat .agents/skills/ 参考路径技能加载（与 .pair/skills 并列，均 project 级）。
func TestAgentsSkillsCompat(t *testing.T) {
	root := t.TempDir()
	agentsSkills := filepath.Join(root, ".agents", "skills", "dsh-doc-standards")
	if err := os.MkdirAll(agentsSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: dsh-doc-standards\ndescription: '文档标准审计工作流'\n---\n\n# 审计\n检查文档冗余"
	if err := os.WriteFile(filepath.Join(agentsSkills, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}

	all := LoadAllSkillsFromRoot(root, "", nil)
	found := false
	for _, s := range all {
		if s.Name == "dsh-doc-standards" {
			found = true
			if s.Level != LevelProject {
				t.Errorf(".agents/skills 技能应为 project 级，实际 %s", s.Level)
			}
			if !strings.Contains(s.Body, "审计") {
				t.Errorf("技能正文应加载：%q", s.Body)
			}
		}
	}
	if !found {
		t.Fatal(".agents/skills 技能未加载")
	}
}
