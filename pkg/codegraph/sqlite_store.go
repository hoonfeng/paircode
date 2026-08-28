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

// Save 将图谱全量写入 SQLite（清表 → 逐文件 INSERT）。
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
		`INSERT INTO code_entities (kind, name, file_path, line, signature, package_name, module, entity_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
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
		result, err := entStmt.Exec(kind, e.Name, fp, e.Line, sig, pkgName, "", e.ID)
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

// SaveIncremental 增量保存：只删除并重新插入 changedFiles 中文件的实体和关系。
// 不涉及的文件保持不动，避免全量 DELETE+INSERT 的性能开销。
func (s *SQLiteStore) SaveIncremental(g *Graph, changedFiles []string) error {
	if len(changedFiles) == 0 {
		return s.Save(g)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := g.ToSnapshot()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 1. 删除变更文件中的旧实体及关联关系
	changedSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changedSet[f] = true
		// 删除涉及该文件实体的关系
		if _, err := tx.Exec(
			`DELETE FROM code_relations WHERE source_id IN (SELECT id FROM code_entities WHERE file_path = ?) OR target_id IN (SELECT id FROM code_entities WHERE file_path = ?)`,
			f, f,
		); err != nil {
			return fmt.Errorf("删除关系失败 (file=%s): %w", f, err)
		}
		// 删除该文件所有实体
		if _, err := tx.Exec(`DELETE FROM code_entities WHERE file_path = ?`, f); err != nil {
			return fmt.Errorf("删除实体失败 (file=%s): %w", f, err)
		}
	}

	// 2. 收集变更文件的实体 ID 集合
	changedEntityIDs := make(map[string]bool, len(changedFiles)*100)
	for _, e := range snapshot.Entities {
		if changedSet[e.FilePath] {
			changedEntityIDs[e.ID] = true
		}
	}

	// 3. 重新插入变更文件的实体
	entStmt, err := tx.Prepare(
		`INSERT INTO code_entities (kind, name, file_path, line, signature, package_name, module, entity_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("准备实体插入失败: %w", err)
	}
	for _, e := range snapshot.Entities {
		if !changedSet[e.FilePath] {
			continue
		}
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
		entStmt.Exec(kind, e.Name, fp, e.Line, sig, pkgName, "", e.ID)
	}
	entStmt.Close()

	// 4. 重建完整 entityID → rowid 映射（从数据库查询最新 rowid，优先原始 entity_id）
	entityMap := make(map[string]int64, len(snapshot.Entities))
	entRows, err := tx.Query(`SELECT id, entity_id, kind, name, file_path, line FROM code_entities`)
	if err != nil {
		return fmt.Errorf("查询实体映射失败: %w", err)
	}
	for entRows.Next() {
		var rowID int64
		var entityID, kind, name, fp string
		var line int
		if err := entRows.Scan(&rowID, &entityID, &kind, &name, &fp, &line); err != nil {
			continue
		}
		if entityID == "" {
			entityID = fp + ":" + kind + ":" + name
		}
		entityMap[entityID] = rowID
	}
	entRows.Close()

	// 5. 插入变更文件涉及的新的关系
	relStmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO code_relations (source_id, target_id, kind) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("准备关系插入失败: %w", err)
	}
	defer relStmt.Close()

	for _, r := range snapshot.Relations {
		// 只插入涉及变更文件实体的关系
		if !changedEntityIDs[r.SourceID] && !changedEntityIDs[r.TargetID] {
			continue
		}
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

// SaveIndex 将文件索引写入 SQLite 的 file_index 表。
func (s *SQLiteStore) SaveIndex(index map[string]time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 清空旧索引
	if _, err := tx.Exec(`DELETE FROM file_index`); err != nil {
		return err
	}

	// 逐条插入
	stmt, err := tx.Prepare(`INSERT INTO file_index (file_path, mtime) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for path, mt := range index {
		stmt.Exec(path, mt.Format(time.RFC3339Nano))
	}

	return tx.Commit()
}

// Load 从 SQLite 读取全部实体和关系，重建 Graph。
func (s *SQLiteStore) Load() (*Graph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g := NewGraph()

	entRows, err := s.db.Query(
		`SELECT id, kind, name, file_path, line, signature, package_name, entity_id FROM code_entities ORDER BY line ASC`,
	)
	if err != nil {
		return NewGraph(), nil
	}
	defer entRows.Close()

	rowIDToID := make(map[int64]string)
	for entRows.Next() {
		var rowID int64
		var kind, name, fp, sig, pkgName, entityID string
		var line int
		if err := entRows.Scan(&rowID, &kind, &name, &fp, &line, &sig, &pkgName, &entityID); err != nil {
			continue
		}
		entityKind := EntityKind(kind)
		// ★ 优先使用存储的原始实体 ID（entity_id）——ID 含调用点行号等唯一信息；
		//   旧数据（无 entity_id）回退拼接（含行号，减少 call_site 同名冲突）。
		id := entityID
		if id == "" {
			id = fmt.Sprintf("%s:%s:%s:%d", fp, kind, name, line)
		}
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

// LoadIndex 从 SQLite 的 file_index 表加载文件索引。
func (s *SQLiteStore) LoadIndex() (map[string]time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := make(map[string]time.Time)
	rows, err := s.db.Query(`SELECT file_path, mtime FROM file_index`)
	if err != nil {
		// 表不存在或查询失败，返回空索引
		return index, nil
	}
	defer rows.Close()

	for rows.Next() {
		var filePath, mtimeStr string
		if err := rows.Scan(&filePath, &mtimeStr); err != nil {
			continue
		}
		mt, err := time.Parse(time.RFC3339Nano, mtimeStr)
		if err != nil {
			continue
		}
		index[filePath] = mt
	}
	return index, nil
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
