package codegraph

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestSQLiteStoreSaveIncremental 验证增量保存只操作变更文件，不涉及的文件保持不动。
//
// 测试步骤：
//  1. 创建内存 SQLite，建表
//  2. 构建含 fileA.go 和 fileB.go 的图
//  3. 全量 Save() → 验证两个文件都在
//  4. 修改 fileA.go 的实体（添加新函数）
//  5. SaveIncremental(graph, ["fileA.go"]) → 只更新 fileA
//  6. 重新 Load() → 验证 fileA 有新内容，fileB 原内容不变
func TestSQLiteStoreSaveIncremental(t *testing.T) {
	db := openMemDB(t)
	defer db.Close()

	store := NewSQLiteStore("", db)

	// 构建初始图（fileA.go + fileB.go）
	g1 := NewGraph()
	g1.AddEntity(&Entity{
		ID:       "fileA.go:function:HelloA",
		Kind:     "function",
		Name:     "HelloA",
		FilePath: "fileA.go",
		Line:     5,
	})
	g1.AddEntity(&Entity{
		ID:       "fileA.go:function:helperA",
		Kind:     "function",
		Name:     "helperA",
		FilePath: "fileA.go",
		Line:     10,
	})
	g1.AddEntity(&Entity{
		ID:       "fileB.go:function:HelloB",
		Kind:     "function",
		Name:     "HelloB",
		FilePath: "fileB.go",
		Line:     5,
	})
	g1.AddRelation(&Relation{
		SourceID: "fileA.go:function:HelloA",
		TargetID: "fileA.go:function:helperA",
		Kind:     "calls",
	})
	g1.AddRelation(&Relation{
		SourceID: "fileA.go:function:HelloA",
		TargetID: "fileB.go:function:HelloB",
		Kind:     "calls",
	})

	// 全量保存
	if err := store.Save(g1); err != nil {
		t.Fatalf("全量保存失败: %v", err)
	}

	// 验证初始内容
	loaded1 := loadGraph(t, store)
	if got := loaded1.Stats().EntityCount; got != 3 {
		t.Fatalf("初始实体数应为 3，实际 %d", got)
	}
	if got := loaded1.Stats().RelationCount; got != 2 {
		t.Fatalf("初始关系数应为 2，实际 %d", got)
	}
	fileBEntities := loaded1.GetEntitiesByFile("fileB.go")
	if len(fileBEntities) != 1 {
		t.Fatalf("fileB.go 应有 1 个实体，实际 %d", len(fileBEntities))
	}

	// 模拟 fileA.go 变更：移除 helperA，添加 NewFuncA
	g2 := NewGraph()
	g2.AddEntity(&Entity{
		ID:       "fileA.go:function:HelloA",
		Kind:     "function",
		Name:     "HelloA",
		FilePath: "fileA.go",
		Line:     5,
	})
	g2.AddEntity(&Entity{
		ID:       "fileA.go:function:NewFuncA",
		Kind:     "function",
		Name:     "NewFuncA",
		FilePath: "fileA.go",
		Line:     15,
	})
	g2.AddEntity(&Entity{
		ID:       "fileB.go:function:HelloB",
		Kind:     "function",
		Name:     "HelloB",
		FilePath: "fileB.go",
		Line:     5,
	})
	g2.AddRelation(&Relation{
		SourceID: "fileA.go:function:HelloA",
		TargetID: "fileA.go:function:NewFuncA",
		Kind:     "calls",
	})
	g2.AddRelation(&Relation{
		SourceID: "fileA.go:function:HelloA",
		TargetID: "fileB.go:function:HelloB",
		Kind:     "calls",
	})

	// 增量保存：只变更 fileA.go
	if err := store.SaveIncremental(g2, []string{"fileA.go"}); err != nil {
		t.Fatalf("增量保存失败: %v", err)
	}

	// 验证结果
	loaded2 := loadGraph(t, store)

	// 实体总数：fileA 从 2 变 2（HelloA + NewFuncA），fileB 维持 1（HelloB）
	if got := loaded2.Stats().EntityCount; got != 3 {
		t.Fatalf("增量后实体数应为 3，实际 %d", got)
	}

	// fileA.go 应包含 NewFuncA 但不包含 helperA
	fileAEntities := loaded2.GetEntitiesByFile("fileA.go")
	if len(fileAEntities) != 2 {
		t.Fatalf("fileA.go 应有 2 个实体，实际 %d", len(fileAEntities))
	}
	hasNewFunc, hasHelperA := false, false
	for _, e := range fileAEntities {
		if e.Name == "NewFuncA" {
			hasNewFunc = true
		}
		if e.Name == "helperA" {
			hasHelperA = true
		}
	}
	if !hasNewFunc {
		t.Fatal("fileA.go 应包含 NewFuncA，但未找到")
	}
	if hasHelperA {
		t.Fatal("fileA.go 应已移除 helperA，但仍然存在")
	}

	// fileB.go 应完全不变
	fileBEntities2 := loaded2.GetEntitiesByFile("fileB.go")
	if len(fileBEntities2) != 1 {
		t.Fatalf("fileB.go 实体数应为 1（不应被增量修改），实际 %d", len(fileBEntities2))
	}
	if fileBEntities2[0].Name != "HelloB" || fileBEntities2[0].Line != 5 {
		t.Fatalf("fileB.go 的 HelloB 应保持不变，实际 name=%s line=%d", fileBEntities2[0].Name, fileBEntities2[0].Line)
	}

	// 关系数应为 2
	if got := loaded2.Stats().RelationCount; got != 2 {
		t.Fatalf("增量后关系数应为 2，实际 %d", got)
	}
}

// TestSQLiteStoreFullSave 验证全量保存与加载正确（回归测试）。
func TestSQLiteStoreFullSave(t *testing.T) {
	db := openMemDB(t)
	defer db.Close()

	store := NewSQLiteStore("", db)

	g := NewGraph()
	g.AddEntity(&Entity{ID: "main.go:function:main", Kind: "function", Name: "main", FilePath: "main.go", Line: 1})
	g.AddEntity(&Entity{ID: "main.go:variable:x", Kind: "variable", Name: "x", FilePath: "main.go", Line: 2})
	g.AddRelation(&Relation{SourceID: "main.go:function:main", TargetID: "main.go:variable:x", Kind: "references"})

	if err := store.Save(g); err != nil {
		t.Fatalf("全量保存失败: %v", err)
	}

	loaded := loadGraph(t, store)

	if got := loaded.Stats().EntityCount; got != 2 {
		t.Errorf("实体数 = %d，期望 2", got)
	}
	if got := loaded.Stats().RelationCount; got != 1 {
		t.Errorf("关系数 = %d，期望 1", got)
	}
}

// TestSQLiteStoreIncrementalPreservesRelations 验证增量保存后跨文件关系仍正确。
func TestSQLiteStoreIncrementalPreservesRelations(t *testing.T) {
	db := openMemDB(t)
	defer db.Close()

	store := NewSQLiteStore("", db)

	// 初始：fileA 有 funcA，fileB 有 funcB，funcA 调用 funcB
	g1 := NewGraph()
	g1.AddEntity(&Entity{ID: "A:function:funcA", Kind: "function", Name: "funcA", FilePath: "A", Line: 1})
	g1.AddEntity(&Entity{ID: "B:function:funcB", Kind: "function", Name: "funcB", FilePath: "B", Line: 1})
	g1.AddRelation(&Relation{SourceID: "A:function:funcA", TargetID: "B:function:funcB", Kind: "calls"})

	if err := store.Save(g1); err != nil {
		t.Fatalf("全量保存失败: %v", err)
	}

	// 修改 fileA 但保持 funcA→funcB 关系
	g2 := NewGraph()
	g2.AddEntity(&Entity{ID: "A:function:funcA", Kind: "function", Name: "funcA", FilePath: "A", Line: 3})
	g2.AddEntity(&Entity{ID: "B:function:funcB", Kind: "function", Name: "funcB", FilePath: "B", Line: 1})
	// 跨文件关系
	g2.AddRelation(&Relation{SourceID: "A:function:funcA", TargetID: "B:function:funcB", Kind: "calls"})
	// 新增文件内关系
	g2.AddRelation(&Relation{SourceID: "A:function:funcA", TargetID: "A:function:funcA", Kind: "references"})

	if err := store.SaveIncremental(g2, []string{"A"}); err != nil {
		t.Fatalf("增量保存失败: %v", err)
	}

	loaded := loadGraph(t, store)

	if got := loaded.Stats().EntityCount; got != 2 {
		t.Fatalf("实体数应为 2，实际 %d", got)
	}
	// 应有 2 个关系：funcA→funcB（跨文件） + funcA→funcA（文件内）
	if got := loaded.Stats().RelationCount; got != 2 {
		t.Fatalf("关系数应为 2，实际 %d", got)
	}
}

// TestIncrementalBuildDetectChanges 验证增量构建能正确检测文件变更。
func TestIncrementalBuildDetectChanges(t *testing.T) {
	root := t.TempDir()

	mainGo := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainGo, []byte(`package main
func main() {}
`), 0644); err != nil {
		t.Fatalf("写入 main.go 失败: %v", err)
	}

	pairDir := filepath.Join(root, ".pair", "codegraph")
	os.MkdirAll(pairDir, 0755)

	config := DefaultBuildConfig(root)
	config.ModuleName = "testmod"
	config.AutoSave = false

	builder := NewBuilder(config)

	result, err := builder.BuildFull()
	if err != nil {
		t.Fatalf("首次完整构建失败: %v", err)
	}
	if result.FilesParsed == 0 {
		t.Fatal("首次构建应解析文件，但结果为 0")
	}

	index := make(map[string]time.Time)
	for _, e := range builder.Graph().entities {
		if absPath := filepath.Join(root, e.FilePath); fileExists(absPath) {
			if fi, err := os.Stat(absPath); err == nil {
				index[e.FilePath] = fi.ModTime()
			}
		}
	}

	// 无变更
	changed, err := builder.detectChangedFiles(index)
	if err != nil {
		t.Fatalf("检测变更失败: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("无变更时应检测到 0，实际 %d", len(changed))
	}

	// 修改文件
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(mainGo, []byte(`package main
func main() { println("hello") }
`), 0644); err != nil {
		t.Fatalf("修改 main.go 失败: %v", err)
	}

	changed2, err := builder.detectChangedFiles(index)
	if err != nil {
		t.Fatalf("检测变更失败: %v", err)
	}
	if len(changed2) != 1 || changed2[0] != "main.go" {
		t.Fatalf("应检测到 main.go 变更，实际 %v", changed2)
	}
}

// ── 辅助函数 ──

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := createTables(db); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	return db
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS code_entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			file_path TEXT DEFAULT '',
			line INTEGER DEFAULT 0,
			signature TEXT DEFAULT '',
			package_name TEXT DEFAULT '',
			module TEXT DEFAULT '',
			UNIQUE(kind, name, file_path, line)
		);
		CREATE INDEX IF NOT EXISTS idx_entities_name ON code_entities(name);
		CREATE INDEX IF NOT EXISTS idx_entities_file ON code_entities(file_path);
		CREATE TABLE IF NOT EXISTS code_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL REFERENCES code_entities(id),
			target_id INTEGER NOT NULL REFERENCES code_entities(id),
			kind TEXT NOT NULL,
			UNIQUE(source_id, target_id, kind)
		);
		CREATE TABLE IF NOT EXISTS file_index (
			file_path TEXT PRIMARY KEY,
			mtime TEXT NOT NULL
		);
	`)
	return err
}

func loadGraph(t *testing.T, store *SQLiteStore) *Graph {
	t.Helper()
	g, err := store.Load()
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	return g
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
