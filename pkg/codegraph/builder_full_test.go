package codegraph

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func nameGo(i int) string {
	return "f" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".go"
}

// TestBuildFullRebuildsIndex 验证：BuildFull 后索引被全量重建
// （此前 BuildFull 不 SaveIndex → 增量检测把全盘文件误判为变更）。
func TestBuildFullRebuildsIndex(t *testing.T) {
	root := t.TempDir()
	// 造 2 个 go 文件
	writeFile(t, filepath.Join(root, "a.go"), "package a\nfunc A() {}\n")
	writeFile(t, filepath.Join(root, "b.go"), "package a\nfunc B() {}\n")

	store := NewStore(root)
	cfg := DefaultBuildConfig(root)
	cfg.AutoSave = true
	cfg.ModuleName = "testmod"
	b := NewBuilder(cfg)
	b.SetStore(store)

	if _, err := b.BuildFull(); err != nil {
		t.Fatalf("BuildFull err: %v", err)
	}

	idx, err := store.LoadIndex()
	if err != nil {
		t.Fatalf("LoadIndex err: %v", err)
	}
	if len(idx) != 2 {
		t.Fatalf("BuildFull 后索引应重建为 2 个文件, got %d (%v)", len(idx), idx)
	}
	if _, ok := idx["a.go"]; !ok {
		t.Fatalf("索引缺少 a.go")
	}
}

// TestIncrementalFallsBackToFull 验证：索引严重过期（变更文件占比 >50%）时
// 增量构建回退全量构建，保证图与磁盘一致。
func TestIncrementalFallsBackToFull(t *testing.T) {
	root := t.TempDir()
	// 30 个 go 文件，索引只含其中 5 个 → 25/30 = 83% 变更 → 应回退全量
	for i := 0; i < 30; i++ {
		writeFile(t, filepath.Join(root, nameGo(i)), "package p\nfunc F() {}\n")
	}

	store := NewStore(root)
	cfg := DefaultBuildConfig(root)
	cfg.AutoSave = true
	cfg.ModuleName = "testmod"
	b := NewBuilder(cfg)
	b.SetStore(store)

	// 全量构建一次（正常索引）
	if _, err := b.BuildFull(); err != nil {
		t.Fatalf("BuildFull err: %v", err)
	}

	// 模拟索引严重过期：只保留 5 条
	idx := make(map[string]time.Time)
	for i := 0; i < 5; i++ {
		fi, _ := os.Stat(filepath.Join(root, nameGo(i)))
		idx[nameGo(i)] = fi.ModTime()
	}
	if err := store.SaveIndex(idx); err != nil {
		t.Fatalf("SaveIndex err: %v", err)
	}

	// 再次增量构建：应回退 BuildFull（索引重建 → 30 文件）
	res, err := b.IncrementalBuild()
	if err != nil {
		t.Fatalf("IncrementalBuild err: %v", err)
	}
	if res.FilesParsed != 30 {
		t.Fatalf("索引严重过期应回退全量构建（解析 30 文件）, got %d", res.FilesParsed)
	}
	idx2, _ := store.LoadIndex()
	if len(idx2) != 30 {
		t.Fatalf("回退全量后索引应重建为 30, got %d", len(idx2))
	}
}

// TestNormalIncrementalNoFull 验证：小比例变更（<50%）走正常增量，不触发全量。
func TestNormalIncrementalNoFull(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		writeFile(t, filepath.Join(root, nameGo(i)), "package p\nfunc F() {}\n")
	}
	store := NewStore(root)
	cfg := DefaultBuildConfig(root)
	cfg.AutoSave = true
	cfg.ModuleName = "testmod"
	b := NewBuilder(cfg)
	b.SetStore(store)
	if _, err := b.BuildFull(); err != nil {
		t.Fatalf("BuildFull err: %v", err)
	}
	// 改 1 个文件 → 1/20 = 5% 变更 → 正常增量（只解析 1 个）
	writeFile(t, filepath.Join(root, "f00.go"), "package p\nfunc G() {}\n")
	res, err := b.IncrementalBuild()
	if err != nil {
		t.Fatalf("IncrementalBuild err: %v", err)
	}
	if res.FilesParsed != 1 {
		t.Fatalf("正常增量应只解析 1 个文件, got %d", res.FilesParsed)
	}
}
