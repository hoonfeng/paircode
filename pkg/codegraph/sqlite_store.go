package codegraph

// sqlite_store.go — 图谱的 SQLite 增量存储实现。
//
// 替换原有的全量 JSON 序列化（30MB graph.json），改为按文件粒度写入 SQLite。
// 全量构建：清空 code_entities/code_relations 表，再逐文件插入。
// 增量构建：只删除并重新插入变更文件对应的实体和关系。
//
// 要求底层已有一个打开且 migrate 过的 *sql.DB（含 code_entities / code_relations 表）。

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SQLiteStore 基于 SQLite 的图谱存储（共享外部 DB 连接）。
type SQLiteStore struct {
	db          *sql.DB
	root        string
	mu          sync.Mutex
	cachedGraph *Graph
	cachedAt    time.Time
}

// NewSQLiteStore 创建 SQLite 图谱存储，使用外部已打开的 *sql.DB。
func NewSQLiteStore(root string, db *sql.DB) *SQLiteStore {
	return &SQLiteStore{
		db:   db,
		root: root,
	}
}

// Save 将图谱增量写入 SQLite（全量：清表 → 逐文件 INSERT 实体 → INSERT 关系）。
func (s *SQLiteStore) Save(g *Graph) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := g.ToSnapshot()
	if len(snapshot.Entities) == 0 && len(snapshot.Relations) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM code_relations`); err != nil {
		return fmt.Errorf("清空关系表失败: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM code_entities`); err != nil {
		return fmt.Errorf("清空实体表失败: %w", err)
	}

	// 插入实体，建立 entity id → rowid 映射
	entityMap := make(map[string]int64, len(snapshot.Entities))
	entStmt, err := tx.Prepare(
		`INSERT INTO code_entities (kind, name, file_path, line, signature, package_name, module) VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("准备实体插入失败: %w", err)
	}
	defer entStmt.Close()

	for _, e := range snapshot.Entities {
		fp := e.FilePath
		kind := string(e.Kind)
		sig := e.Signature
		if sig == "" {
			sig = e.FQN
		}
		pkgName := ""
		if idx := lastDot(e.FQN); idx > 0 {
			pkgName = e.FQN[:idx]
		}
		result, err := entStmt.Exec(kind, e.Name, fp, e.Line, sig, pkgName, "")
		if err != nil {
			continue
		}
		rowID, _ := result.LastInsertId()
		entityMap[e.ID] = rowID
	}
	entStmt.Close()

	// 插入关系
	relStmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO code_relations (source_id, target_id, kind) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("准备关系插入失败: %w", err)
	}
	defer relStmt.Close()

	for _, r := range snapshot.Relations {
		srcID, ok := entityMap[r.SourceID]
		if !ok {
			continue
		}
		tgtID, ok := entityMap[r.TargetID]
		if !ok {
			continue
		}
		relStmt.Exec(srcID, tgtID, string(r.Kind))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	s.cachedGraph = g
	s.cachedAt = time.Now()
	return nil
}

// SaveIndex 文件索引暂不存入 SQLite（全量构建时由 builder 层记录 mtime）。
func (s *SQLiteStore) SaveIndex(index map[string]time.Time) error {
	return nil
}

// Load 从 SQLite 读取全部实体和关系，重建 Graph。
func (s *SQLiteStore) Load() (*Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g := NewGraph()

	entRows, err := s.db.Query(
		`SELECT id, kind, name, file_path, line, signature, package_name FROM code_entities ORDER BY line ASC`,
	)
	if err != nil {
		return NewGraph(), nil
	}
	defer entRows.Close()

	rowIDToID := make(map[int64]string)
	for entRows.Next() {
		var rowID int64
		var kind, name, fp, sig, pkgName string
		var line int
		if err := entRows.Scan(&rowID, &kind, &name, &fp, &line, &sig, &pkgName); err != nil {
			continue
		}
		entityKind := EntityKind(kind)
		id := fp + ":" + kind + ":" + name
		rowIDToID[rowID] = id

		g.AddEntity(&Entity{
			ID:        id,
			Kind:      entityKind,
			Name:      name,
			FilePath:  fp,
			Line:      line,
			Signature: sig,
			FQN:       pkgName + "." + name,
		})
	}
	entRows.Close()

	relRows, err := s.db.Query(`SELECT source_id, target_id, kind FROM code_relations`)
	if err != nil {
		return g, nil
	}
	defer relRows.Close()

	for relRows.Next() {
		var srcID, tgtID int64
		var kind string
		if err := relRows.Scan(&srcID, &tgtID, &kind); err != nil {
			continue
		}
		srcEntityID, ok1 := rowIDToID[srcID]
		tgtEntityID, ok2 := rowIDToID[tgtID]
		if !ok1 || !ok2 {
			continue
		}
		g.AddRelation(&Relation{
			SourceID: srcEntityID,
			TargetID: tgtEntityID,
			Kind:     RelationKind(kind),
		})
	}

	s.cachedGraph = g
	s.cachedAt = time.Now()
	return g, nil
}

// CachedGraph 返回缓存的图（如果已加载且未过时）。
func (s *SQLiteStore) CachedGraph(maxAge int) *Graph {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedGraph == nil {
		return nil
	}
	if maxAge > 0 && time.Since(s.cachedAt) > time.Duration(maxAge)*time.Second {
		return nil
	}
	return s.cachedGraph
}

// Exists 检查 code_entities 表中是否有数据。
func (s *SQLiteStore) Exists() bool {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM code_entities`).Scan(&count)
	return count > 0
}

// Delete 清空图谱数据。
func (s *SQLiteStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`DELETE FROM code_relations`)
	_, _ = s.db.Exec(`DELETE FROM code_entities`)
	s.cachedGraph = nil
	return nil
}

// LoadIndex 返回空索引（全量构建模式下不需要文件索引）。
func (s *SQLiteStore) LoadIndex() (map[string]time.Time, error) {
	return make(map[string]time.Time), nil
}

// lastDot 返回字符串中最后一个 '.' 的位置（用于拆分 FQN）。
func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
