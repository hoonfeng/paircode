package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProjectInfoTools 知识库 写→读→总览→搜→删→探索；中文路径；路径穿越被关在知识库目录内。
func TestProjectInfoTools(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	if _, err := reg.Execute(ctx, "project_info_write", `{"path":"概览","content":"# 项目概览\n这是一个 Go 项目"}`); err != nil {
		t.Fatal(err)
	}
	reg.Execute(ctx, "project_info_write", `{"path":"模块-agent","content":"# 模块 agent\nAgent 引擎"}`)

	full, err := reg.Execute(ctx, "project_info_read", `{"path":"概览"}`)
	if err != nil || !strings.Contains(full, "Go 项目") {
		t.Errorf("read 应返回全文：%v %q", err, full)
	}
	list, _ := reg.Execute(ctx, "project_info_list", "{}")
	if !strings.Contains(list, "概览") || !strings.Contains(list, "模块-agent") {
		t.Errorf("list 总览应含两条：%q", list)
	}
	hit, _ := reg.Execute(ctx, "project_info_search", `{"query":"引擎"}`)
	if !strings.Contains(hit, "模块-agent") {
		t.Errorf("search 应命中模块-agent：%q", hit)
	}
	reg.Execute(ctx, "project_info_delete", `{"path":"模块-agent"}`)
	if _, err := reg.Execute(ctx, "project_info_read", `{"path":"模块-agent"}`); err == nil {
		t.Error("删除后 read 应报错")
	}
	if exp, _ := reg.Execute(ctx, "project_info_explore", "{}"); !strings.Contains(exp, "项目结构概览") {
		t.Errorf("explore 应返回结构概览：%q", exp)
	}

	// 路径穿越：../../evil 被清理为知识库内的 evil.md，不写到 root 外
	reg.Execute(ctx, "project_info_write", `{"path":"../../evil","content":"# x"}`)
	if _, err := os.Stat(filepath.Join(root, "evil.md")); err == nil {
		t.Error("路径穿越应被阻止（不应写到知识库目录外）")
	}
}

// TestProjectKnowledgeInjection 自动注入：概览篇给正文 + 其余列目录（渐进式披露）；空→空。
func TestProjectKnowledgeInjection(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	if ProjectKnowledge(root, 2500) != "" {
		t.Error("空知识库应注入空")
	}
	reg.Execute(ctx, "project_info_write", `{"path":"概览","content":"# 项目概览\n核心是渲染引擎"}`)
	reg.Execute(ctx, "project_info_write", `{"path":"模块-渲染","content":"# 渲染模块\n细节内容"}`)
	kb := ProjectKnowledge(root, 2500)
	if !strings.Contains(kb, "项目知识库") || !strings.Contains(kb, "渲染引擎") {
		t.Errorf("应注入概览正文：%q", kb)
	}
	if !strings.Contains(kb, "知识库目录") || !strings.Contains(kb, "模块-渲染") {
		t.Errorf("应列其余条目目录：%q", kb)
	}
}
