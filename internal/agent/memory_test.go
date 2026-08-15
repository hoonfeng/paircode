package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemoryTools 写记忆 → 读回 → 列出 → 搜索命中/不命中。
func TestMemoryTools(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	if _, err := reg.Execute(ctx, "memory_write",
		`{"name":"build-cmd","type":"project","description":"如何构建","content":"用 go build ./cmd/x"}`); err != nil {
		t.Fatalf("memory_write: %v", err)
	}

	got, err := reg.Execute(ctx, "memory_read", `{"name":"build-cmd"}`)
	if err != nil || !strings.Contains(got, "go build ./cmd/x") {
		t.Errorf("memory_read 应含正文：%v\n%s", err, got)
	}
	if list, _ := reg.Execute(ctx, "memory_list", `{}`); !strings.Contains(list, "build-cmd") {
		t.Errorf("memory_list 应含条目：\n%s", list)
	}
	if hit, _ := reg.Execute(ctx, "memory_search", `{"query":"构建"}`); !strings.Contains(hit, "build-cmd") {
		t.Errorf("memory_search 应命中：\n%s", hit)
	}
	if miss, _ := reg.Execute(ctx, "memory_search", `{"query":"无关词xyz"}`); strings.Contains(miss, "build-cmd") {
		t.Errorf("不该命中：\n%s", miss)
	}
	if _, err := reg.Execute(ctx, "memory_read", `{"name":"不存在"}`); err == nil {
		t.Error("读不存在的记忆应报错")
	}
}

// TestMemoryProjectParam 多项目工作区：memory 工具 project 参数路由到对应项目根，
// 记忆按项目隔离（写入 projB 的读不到主项目里，反之亦然）。
func TestMemoryProjectParam(t *testing.T) {
	primary := t.TempDir()
	projB := t.TempDir()
	old := WorkspaceRoots
	WorkspaceRoots = []string{primary, projB}
	defer func() { WorkspaceRoots = old }()

	reg := NewRegistry()
	RegisterDefaultTools(reg, primary)
	ctx := context.Background()

	// 写入主项目记忆
	if _, err := reg.Execute(ctx, "memory_write",
		`{"name":"主项目记忆","type":"project","description":"主项目","content":"在主项目根"}`); err != nil {
		t.Fatalf("memory_write(主项目): %v", err)
	}
	// 写入 projB 记忆（project 参数指定）
	if _, err := reg.Execute(ctx, "memory_write",
		`{"name":"B项目记忆","type":"project","description":"B项目","content":"在B项目根","project":"`+
			filepath.Base(projB)+`"}`); err != nil {
		t.Fatalf("memory_write(projB): %v", err)
	}

	// 主项目视角：只见主项目记忆
	if list, _ := reg.Execute(ctx, "memory_list", `{}`); strings.Contains(list, "B项目记忆") {
		t.Errorf("主项目 memory_list 不应见 B 项目记忆：\n%s", list)
	}
	// B 项目视角：只见 B 项目记忆
	listB, err := reg.Execute(ctx, "memory_list", `{"project":"`+filepath.Base(projB)+`"}`)
	if err != nil {
		t.Fatalf("memory_list(projB): %v", err)
	}
	if !strings.Contains(listB, "B项目记忆") || strings.Contains(listB, "主项目记忆") {
		t.Errorf("B 项目 memory_list 应只见 B 项目记忆：\n%s", listB)
	}
	// 物理隔离验证：projB/.pair/memory 存在且不含主项目文件
	if _, err := os.Stat(filepath.Join(projB, ".pair", "memory", "B项目记忆.md")); err != nil {
		t.Errorf("projB 记忆文件应存在: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projB, ".pair", "memory", "主项目记忆.md")); err == nil {
		t.Error("主项目记忆不应出现在 projB 目录")
	}
}

// TestMemoryUpdateNotDuplicate 同名→更新（非新建）；新名+相关内容→提示已有相关记忆（防碎片化）。
func TestMemoryUpdateNotDuplicate(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, _ := reg.Execute(ctx, "memory_write", `{"name":"db-pool","type":"project","description":"数据库连接池配置","content":"连接池上限设为 20，超时 30s，注意泄漏"}`)
	if !strings.Contains(out, "已记忆") {
		t.Errorf("首次应为新建，得 %q", out)
	}
	out2, _ := reg.Execute(ctx, "memory_write", `{"name":"db-pool","type":"project","description":"数据库连接池配置","content":"连接池上限改为 50"}`)
	if !strings.Contains(out2, "已更新") {
		t.Errorf("同名应为更新，得 %q", out2)
	}
	out3, _ := reg.Execute(ctx, "memory_write", `{"name":"db-pool-v2","type":"project","description":"数据库连接池上限","content":"数据库连接池上限超时泄漏配置调优"}`)
	if !strings.Contains(out3, "已有相关记忆") || !strings.Contains(out3, "db-pool") {
		t.Errorf("新名+相关内容应提示已有 db-pool，得 %q", out3)
	}
}

// TestMemoryIndex MEMORY.md 总览索引（渐进式披露）：写入生成/更新、删除移除；中文文件命名；不把索引当条目。
func TestMemoryIndex(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	reg.Execute(ctx, "memory_write", `{"name":"数据库连接池配置","type":"project","description":"连接池上限与超时","content":"上限 20，超时 30s"}`)
	reg.Execute(ctx, "memory_write", `{"name":"构建命令","type":"project","description":"如何构建","content":"go build ./cmd/x"}`)

	idx, err := os.ReadFile(filepath.Join(memoryDir(dir), "MEMORY.md"))
	if err != nil {
		t.Fatalf("应生成 MEMORY.md 索引：%v", err)
	}
	s := string(idx)
	if !strings.Contains(s, "记忆索引") || !strings.Contains(s, "数据库连接池配置") || !strings.Contains(s, "构建命令") {
		t.Errorf("索引应含表头 + 两条中文条目，得：\n%s", s)
	}
	if !strings.Contains(s, "(数据库连接池配置.md)") {
		t.Errorf("索引条目应链向中文文件名，得：\n%s", s)
	}

	list, _ := reg.Execute(ctx, "memory_list", "{}")
	if !strings.Contains(list, "数据库连接池配置") || strings.Contains(list, "MEMORY.md)") {
		t.Errorf("memory_list 应是总览且不含索引自身，得：\n%s", list)
	}

	reg.Execute(ctx, "memory_delete", `{"name":"构建命令"}`)
	idx2, _ := os.ReadFile(filepath.Join(memoryDir(dir), "MEMORY.md"))
	if strings.Contains(string(idx2), "构建命令") {
		t.Errorf("删除后索引应移除该条，得：\n%s", string(idx2))
	}
	if !strings.Contains(string(idx2), "数据库连接池配置") {
		t.Error("删除后另一条应仍在索引")
	}
}

// TestMemoryDelete 删除记忆；删不存在的报错。
func TestMemoryDelete(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()
	reg.Execute(ctx, "memory_write", `{"name":"tmp","description":"临时","content":"待删"}`)
	if _, err := reg.Execute(ctx, "memory_delete", `{"name":"tmp"}`); err != nil {
		t.Fatalf("memory_delete: %v", err)
	}
	if _, err := reg.Execute(ctx, "memory_read", `{"name":"tmp"}`); err == nil {
		t.Error("删除后读应报错")
	}
	if _, err := reg.Execute(ctx, "memory_delete", `{"name":"不存在"}`); err == nil {
		t.Error("删不存在的应报错")
	}
}
