package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteDB 是基于 SQLite 的 DB 实现。
type SQLiteDB struct {
	db   *sql.DB
	path string
}

// NewSQLiteDB 创建并初始化 SQLite 数据库。
func NewSQLiteDB(dbPath string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite 不支持并发写
	s := &SQLiteDB{db: db, path: dbPath}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// migrate 执行数据库迁移（建表）。
func (s *SQLiteDB) migrate() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			title TEXT DEFAULT '',
			workspace_root TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			summary TEXT DEFAULT '',
			msg_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conv_id TEXT NOT NULL REFERENCES conversations(id),
			idx INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT DEFAULT '',
			tool_calls TEXT DEFAULT '',
			tool_call_id TEXT DEFAULT '',
			name TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
			segments TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(conv_id, idx)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conv_id, idx)`,
		`CREATE TABLE IF NOT EXISTS plans (
			id TEXT PRIMARY KEY,
			conv_id TEXT REFERENCES conversations(id),
			status TEXT DEFAULT 'active',
			task TEXT DEFAULT '',
			reasoning TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS plan_steps (
			id TEXT PRIMARY KEY,
			plan_id TEXT NOT NULL REFERENCES plans(id),
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			dependencies TEXT DEFAULT '[]',
			sub_task_ids TEXT DEFAULT '[]',
			sort_order INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_plan_steps_plan ON plan_steps(plan_id)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			subject TEXT DEFAULT '',
			description TEXT DEFAULT '',
			status TEXT DEFAULT 'pending',
			dependencies TEXT DEFAULT '[]',
			conv_id TEXT REFERENCES conversations(id),
			plan_id TEXT REFERENCES plans(id),
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE TABLE IF NOT EXISTS memories (
			name TEXT PRIMARY KEY,
			type TEXT DEFAULT '',
			description TEXT DEFAULT '',
			body TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS project_info (
			path TEXT PRIMARY KEY,
			title TEXT DEFAULT '',
			level TEXT DEFAULT 'detail',
			body TEXT DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS code_entities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			file_path TEXT DEFAULT '',
			line INTEGER DEFAULT 0,
			signature TEXT DEFAULT '',
			package_name TEXT DEFAULT '',
			module TEXT DEFAULT '',
			UNIQUE(kind, name, file_path, line)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_name ON code_entities(name)`,
		`CREATE INDEX IF NOT EXISTS idx_entities_file ON code_entities(file_path)`,
		`CREATE TABLE IF NOT EXISTS code_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id INTEGER NOT NULL REFERENCES code_entities(id),
			target_id INTEGER NOT NULL REFERENCES code_entities(id),
			kind TEXT NOT NULL,
			UNIQUE(source_id, target_id, kind)
		)`,
		`CREATE TABLE IF NOT EXISTS evals (
			id TEXT PRIMARY KEY,
			task TEXT DEFAULT '',
			agent_model TEXT DEFAULT '',
			judge_model TEXT DEFAULT '',
			scores TEXT DEFAULT '{}',
			total INTEGER DEFAULT 0,
			strengths TEXT DEFAULT '',
			weaknesses TEXT DEFAULT '',
			feedback TEXT DEFAULT '',
			tool_calls INTEGER DEFAULT 0,
			tool_errors INTEGER DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			file_path TEXT NOT NULL,
			snapshot_id TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY(file_path, snapshot_id)
		)`,
	}
	for _, ddl := range tables {
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("exec %q: %w", ddl[:40], err)
		}
	}
	return nil
}

// Close 关闭数据库连接。
func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

// ── 对话相关 ──

func (s *SQLiteDB) CreateConversation(conv *Conversation) error {
	_, err := s.db.Exec(
		`INSERT INTO conversations (id, title, workspace_root, created_at, updated_at, summary, msg_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		conv.ID, conv.Title, conv.WorkspaceRoot, conv.CreatedAt.UTC().Format(time.RFC3339), conv.UpdatedAt.UTC().Format(time.RFC3339), conv.Summary, conv.MsgCount,
	)
	return err
}

func (s *SQLiteDB) GetConversation(id string) (*Conversation, error) {
	row := s.db.QueryRow(`SELECT id, title, workspace_root, created_at, updated_at, summary, msg_count FROM conversations WHERE id = ?`, id)
	var conv Conversation
	var createdAt, updatedAt string
	if err := row.Scan(&conv.ID, &conv.Title, &conv.WorkspaceRoot, &createdAt, &updatedAt, &conv.Summary, &conv.MsgCount); err != nil {
		return nil, err
	}
	conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &conv, nil
}

func (s *SQLiteDB) UpdateConversation(conv *Conversation) error {
	_, err := s.db.Exec(
		`UPDATE conversations SET title=?, workspace_root=?, updated_at=?, summary=?, msg_count=? WHERE id=?`,
		conv.Title, conv.WorkspaceRoot, conv.UpdatedAt.UTC().Format(time.RFC3339), conv.Summary, conv.MsgCount, conv.ID,
	)
	return err
}

func (s *SQLiteDB) ListConversations() ([]*Conversation, error) {
	rows, err := s.db.Query(`SELECT id, title, workspace_root, created_at, updated_at, summary, msg_count FROM conversations ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []*Conversation
	for rows.Next() {
		var conv Conversation
		var createdAt, updatedAt string
		if err := rows.Scan(&conv.ID, &conv.Title, &conv.WorkspaceRoot, &createdAt, &updatedAt, &conv.Summary, &conv.MsgCount); err != nil {
			return nil, err
		}
		conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		convs = append(convs, &conv)
	}
	return convs, nil
}

func (s *SQLiteDB) DeleteConversation(id string) error {
	_, err := s.db.Exec(`DELETE FROM conversations WHERE id = ?`, id)
	return err
}

// ── 消息相关 ──

func (s *SQLiteDB) AppendMessage(msg *Message) error {
	_, err := s.db.Exec(
		`INSERT INTO messages (conv_id, idx, role, content, tool_calls, tool_call_id, name, reasoning, segments, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		msg.ConvID, msg.Idx, msg.Role, msg.Content, msg.ToolCalls, msg.ToolCallID, msg.Name, msg.Reasoning, msg.Segments, msg.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) GetMessages(convID string, offset, limit int) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, conv_id, idx, role, content, tool_calls, tool_call_id, name, reasoning, segments, created_at FROM messages WHERE conv_id = ? ORDER BY idx ASC LIMIT ? OFFSET ?`,
		convID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []*Message
	for rows.Next() {
		var msg Message
		var createdAt string
		if err := rows.Scan(&msg.ID, &msg.ConvID, &msg.Idx, &msg.Role, &msg.Content, &msg.ToolCalls, &msg.ToolCallID, &msg.Name, &msg.Reasoning, &msg.Segments, &createdAt); err != nil {
			return nil, err
		}
		msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		msgs = append(msgs, &msg)
	}
	return msgs, nil
}

func (s *SQLiteDB) GetMessageCount(convID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conv_id = ?`, convID).Scan(&count)
	return count, err
}

func (s *SQLiteDB) DeleteMessages(convID string) error {
	_, err := s.db.Exec(`DELETE FROM messages WHERE conv_id = ?`, convID)
	return err
}

// ── 计划相关 ──

func (s *SQLiteDB) CreatePlan(plan *Plan) error {
	_, err := s.db.Exec(
		`INSERT INTO plans (id, conv_id, status, task, reasoning, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		plan.ID, plan.ConvID, plan.Status, plan.Task, plan.Reasoning, plan.CreatedAt.UTC().Format(time.RFC3339), plan.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) GetPlan(id string) (*Plan, error) {
	row := s.db.QueryRow(`SELECT id, conv_id, status, task, reasoning, created_at, updated_at FROM plans WHERE id = ?`, id)
	var plan Plan
	var createdAt, updatedAt string
	if err := row.Scan(&plan.ID, &plan.ConvID, &plan.Status, &plan.Task, &plan.Reasoning, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	plan.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	plan.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &plan, nil
}

func (s *SQLiteDB) UpdatePlan(plan *Plan) error {
	_, err := s.db.Exec(
		`UPDATE plans SET conv_id=?, status=?, task=?, reasoning=?, updated_at=? WHERE id=?`,
		plan.ConvID, plan.Status, plan.Task, plan.Reasoning, plan.UpdatedAt.UTC().Format(time.RFC3339), plan.ID,
	)
	return err
}

func (s *SQLiteDB) ListPlans(convID string) ([]*Plan, error) {
	rows, err := s.db.Query(`SELECT id, conv_id, status, task, reasoning, created_at, updated_at FROM plans WHERE conv_id = ? ORDER BY created_at DESC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []*Plan
	for rows.Next() {
		var plan Plan
		var createdAt, updatedAt string
		if err := rows.Scan(&plan.ID, &plan.ConvID, &plan.Status, &plan.Task, &plan.Reasoning, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		plan.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		plan.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		plans = append(plans, &plan)
	}
	return plans, nil
}

func (s *SQLiteDB) DeletePlan(id string) error {
	_, err := s.db.Exec(`DELETE FROM plans WHERE id = ?`, id)
	return err
}

// ── 计划步骤 ──

func (s *SQLiteDB) CreatePlanStep(step *PlanStep) error {
	_, err := s.db.Exec(
		`INSERT INTO plan_steps (id, plan_id, description, status, dependencies, sub_task_ids, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		step.ID, step.PlanID, step.Description, step.Status, step.Dependencies, step.SubTaskIDs, step.SortOrder,
	)
	return err
}

func (s *SQLiteDB) UpdatePlanStep(step *PlanStep) error {
	_, err := s.db.Exec(
		`UPDATE plan_steps SET description=?, status=?, dependencies=?, sub_task_ids=?, sort_order=? WHERE id=? AND plan_id=?`,
		step.Description, step.Status, step.Dependencies, step.SubTaskIDs, step.SortOrder, step.ID, step.PlanID,
	)
	return err
}

func (s *SQLiteDB) GetPlanSteps(planID string) ([]*PlanStep, error) {
	rows, err := s.db.Query(`SELECT id, plan_id, description, status, dependencies, sub_task_ids, sort_order FROM plan_steps WHERE plan_id = ? ORDER BY sort_order ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var steps []*PlanStep
	for rows.Next() {
		var step PlanStep
		if err := rows.Scan(&step.ID, &step.PlanID, &step.Description, &step.Status, &step.Dependencies, &step.SubTaskIDs, &step.SortOrder); err != nil {
			return nil, err
		}
		steps = append(steps, &step)
	}
	return steps, nil
}

// ── 任务相关 ──

func (s *SQLiteDB) CreateTask(task *Task) error {
	_, err := s.db.Exec(
		`INSERT INTO tasks (id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Subject, task.Description, task.Status, task.Dependencies, task.ConvID, task.PlanID, task.CreatedAt.UTC().Format(time.RFC3339), task.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) GetTask(id string) (*Task, error) {
	row := s.db.QueryRow(`SELECT id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at FROM tasks WHERE id = ?`, id)
	var task Task
	var createdAt, updatedAt string
	if err := row.Scan(&task.ID, &task.Subject, &task.Description, &task.Status, &task.Dependencies, &task.ConvID, &task.PlanID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &task, nil
}

func (s *SQLiteDB) UpdateTask(task *Task) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET subject=?, description=?, status=?, dependencies=?, conv_id=?, plan_id=?, updated_at=? WHERE id=?`,
		task.Subject, task.Description, task.Status, task.Dependencies, task.ConvID, task.PlanID, task.UpdatedAt.UTC().Format(time.RFC3339), task.ID,
	)
	return err
}

func (s *SQLiteDB) ListTasks(convID string, status string) ([]*Task, error) {
	var rows *sql.Rows
	var err error
	if convID != "" && status != "" {
		rows, err = s.db.Query(`SELECT id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at FROM tasks WHERE conv_id=? AND status=? ORDER BY created_at DESC`, convID, status)
	} else if convID != "" {
		rows, err = s.db.Query(`SELECT id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at FROM tasks WHERE conv_id=? ORDER BY created_at DESC`, convID)
	} else if status != "" {
		rows, err = s.db.Query(`SELECT id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at FROM tasks WHERE status=? ORDER BY created_at DESC`, status)
	} else {
		rows, err = s.db.Query(`SELECT id, subject, description, status, dependencies, conv_id, plan_id, created_at, updated_at FROM tasks ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		var task Task
		var createdAt, updatedAt string
		if err := rows.Scan(&task.ID, &task.Subject, &task.Description, &task.Status, &task.Dependencies, &task.ConvID, &task.PlanID, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *SQLiteDB) DeleteTask(id string) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// ── 记忆相关 ──

func (s *SQLiteDB) WriteMemory(mem *Memory) error {
	_, err := s.db.Exec(
		`INSERT INTO memories (name, type, description, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(name) DO UPDATE SET type=excluded.type, description=excluded.description, body=excluded.body, updated_at=excluded.updated_at`,
		mem.Name, mem.Type, mem.Description, mem.Body, mem.CreatedAt.UTC().Format(time.RFC3339), mem.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) ReadMemory(name string) (*Memory, error) {
	row := s.db.QueryRow(`SELECT name, type, description, body, created_at, updated_at FROM memories WHERE name = ?`, name)
	var mem Memory
	var createdAt, updatedAt string
	if err := row.Scan(&mem.Name, &mem.Type, &mem.Description, &mem.Body, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	mem.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	mem.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &mem, nil
}

func (s *SQLiteDB) SearchMemories(query string) ([]*Memory, error) {
	rows, err := s.db.Query(`SELECT name, type, description, body, created_at, updated_at FROM memories WHERE name LIKE ? OR body LIKE ? ORDER BY updated_at DESC LIMIT 20`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mems []*Memory
	for rows.Next() {
		var mem Memory
		var createdAt, updatedAt string
		if err := rows.Scan(&mem.Name, &mem.Type, &mem.Description, &mem.Body, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		mem.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		mem.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		mems = append(mems, &mem)
	}
	return mems, nil
}

func (s *SQLiteDB) ListMemories() ([]*Memory, error) {
	rows, err := s.db.Query(`SELECT name, type, description, body, created_at, updated_at FROM memories ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mems []*Memory
	for rows.Next() {
		var mem Memory
		var createdAt, updatedAt string
		if err := rows.Scan(&mem.Name, &mem.Type, &mem.Description, &mem.Body, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		mem.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		mem.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		mems = append(mems, &mem)
	}
	return mems, nil
}

func (s *SQLiteDB) DeleteMemory(name string) error {
	_, err := s.db.Exec(`DELETE FROM memories WHERE name = ?`, name)
	return err
}

// ── 知识库相关 ──

func (s *SQLiteDB) WriteProjectInfo(pi *ProjectInfo) error {
	_, err := s.db.Exec(
		`INSERT INTO project_info (path, title, level, body, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET title=excluded.title, level=excluded.level, body=excluded.body, updated_at=excluded.updated_at`,
		pi.Path, pi.Title, pi.Level, pi.Body, pi.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) ReadProjectInfo(path string) (*ProjectInfo, error) {
	row := s.db.QueryRow(`SELECT path, title, level, body, updated_at FROM project_info WHERE path = ?`, path)
	var pi ProjectInfo
	var updatedAt string
	if err := row.Scan(&pi.Path, &pi.Title, &pi.Level, &pi.Body, &updatedAt); err != nil {
		return nil, err
	}
	pi.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &pi, nil
}

func (s *SQLiteDB) SearchProjectInfo(query string) ([]*ProjectInfo, error) {
	rows, err := s.db.Query(`SELECT path, title, level, body, updated_at FROM project_info WHERE title LIKE ? OR body LIKE ? ORDER BY updated_at DESC LIMIT 20`, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pis []*ProjectInfo
	for rows.Next() {
		var pi ProjectInfo
		var updatedAt string
		if err := rows.Scan(&pi.Path, &pi.Title, &pi.Level, &pi.Body, &updatedAt); err != nil {
			return nil, err
		}
		pi.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		pis = append(pis, &pi)
	}
	return pis, nil
}

func (s *SQLiteDB) ListProjectInfo() ([]*ProjectInfo, error) {
	rows, err := s.db.Query(`SELECT path, title, level, body, updated_at FROM project_info ORDER BY path ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pis []*ProjectInfo
	for rows.Next() {
		var pi ProjectInfo
		var updatedAt string
		if err := rows.Scan(&pi.Path, &pi.Title, &pi.Level, &pi.Body, &updatedAt); err != nil {
			return nil, err
		}
		pi.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		pis = append(pis, &pi)
	}
	return pis, nil
}

func (s *SQLiteDB) DeleteProjectInfo(path string) error {
	_, err := s.db.Exec(`DELETE FROM project_info WHERE path = ?`, path)
	return err
}

// ── 代码实体 ──

func (s *SQLiteDB) UpsertCodeEntity(entity *CodeEntity) (int64, error) {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO code_entities (kind, name, file_path, line, signature, package_name, module) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entity.Kind, entity.Name, entity.FilePath, entity.Line, entity.Signature, entity.PackageName, entity.Module,
	)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRow(`SELECT id FROM code_entities WHERE kind=? AND name=? AND file_path=? AND line=?`, entity.Kind, entity.Name, entity.FilePath, entity.Line).Scan(&id)
	return id, err
}

func (s *SQLiteDB) SearchCodeEntities(query string, kind string) ([]*CodeEntity, error) {
	var rows *sql.Rows
	var err error
	if kind != "" {
		rows, err = s.db.Query(`SELECT id, kind, name, file_path, line, signature, package_name, module FROM code_entities WHERE name LIKE ? AND kind=? ORDER BY name ASC LIMIT 100`, "%"+query+"%", kind)
	} else {
		rows, err = s.db.Query(`SELECT id, kind, name, file_path, line, signature, package_name, module FROM code_entities WHERE name LIKE ? ORDER BY name ASC LIMIT 100`, "%"+query+"%")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ents []*CodeEntity
	for rows.Next() {
		var ent CodeEntity
		if err := rows.Scan(&ent.ID, &ent.Kind, &ent.Name, &ent.FilePath, &ent.Line, &ent.Signature, &ent.PackageName, &ent.Module); err != nil {
			return nil, err
		}
		ents = append(ents, &ent)
	}
	return ents, nil
}

func (s *SQLiteDB) GetCodeEntitiesByFile(filePath string) ([]*CodeEntity, error) {
	rows, err := s.db.Query(`SELECT id, kind, name, file_path, line, signature, package_name, module FROM code_entities WHERE file_path=? ORDER BY line ASC`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ents []*CodeEntity
	for rows.Next() {
		var ent CodeEntity
		if err := rows.Scan(&ent.ID, &ent.Kind, &ent.Name, &ent.FilePath, &ent.Line, &ent.Signature, &ent.PackageName, &ent.Module); err != nil {
			return nil, err
		}
		ents = append(ents, &ent)
	}
	return ents, nil
}

// ── 代码关系 ──

func (s *SQLiteDB) UpsertCodeRelation(sourceID, targetID int64, kind string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO code_relations (source_id, target_id, kind) VALUES (?, ?, ?)`,
		sourceID, targetID, kind,
	)
	return err
}

func (s *SQLiteDB) GetRelations(entityID int64) ([]*CodeRelation, error) {
	rows, err := s.db.Query(`SELECT id, source_id, target_id, kind FROM code_relations WHERE source_id=? OR target_id=? ORDER BY kind`, entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rels []*CodeRelation
	for rows.Next() {
		var rel CodeRelation
		if err := rows.Scan(&rel.ID, &rel.SourceID, &rel.TargetID, &rel.Kind); err != nil {
			return nil, err
		}
		rels = append(rels, &rel)
	}
	return rels, nil
}

// ── 评分 ──

func (s *SQLiteDB) CreateEval(eval *Eval) error {
	_, err := s.db.Exec(
		`INSERT INTO evals (id, task, agent_model, judge_model, scores, total, strengths, weaknesses, feedback, tool_calls, tool_errors, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		eval.ID, eval.Task, eval.AgentModel, eval.JudgeModel, eval.Scores, eval.Total, eval.Strengths, eval.Weaknesses, eval.Feedback, eval.ToolCalls, eval.ToolErrors, eval.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) ListEvals(limit int) ([]*Eval, error) {
	rows, err := s.db.Query(`SELECT id, task, agent_model, judge_model, scores, total, strengths, weaknesses, feedback, tool_calls, tool_errors, created_at FROM evals ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var evals []*Eval
	for rows.Next() {
		var eval Eval
		var createdAt string
		if err := rows.Scan(&eval.ID, &eval.Task, &eval.AgentModel, &eval.JudgeModel, &eval.Scores, &eval.Total, &eval.Strengths, &eval.Weaknesses, &eval.Feedback, &eval.ToolCalls, &eval.ToolErrors, &createdAt); err != nil {
			return nil, err
		}
		eval.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		evals = append(evals, &eval)
	}
	return evals, nil
}

// ── 快照 ──

func (s *SQLiteDB) CreateSnapshot(snap *Snapshot) error {
	_, err := s.db.Exec(
		`INSERT INTO snapshots (file_path, snapshot_id, created_at) VALUES (?, ?, ?)`,
		snap.FilePath, snap.SnapshotID, snap.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteDB) ListSnapshots(filePath string) ([]*Snapshot, error) {
	rows, err := s.db.Query(`SELECT file_path, snapshot_id, created_at FROM snapshots WHERE file_path=? ORDER BY created_at DESC`, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snaps []*Snapshot
	for rows.Next() {
		var snap Snapshot
		var createdAt string
		if err := rows.Scan(&snap.FilePath, &snap.SnapshotID, &createdAt); err != nil {
			return nil, err
		}
		snap.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		snaps = append(snaps, &snap)
	}
	return snaps, nil
}

func (s *SQLiteDB) DeleteSnapshots(before time.Time) (int, error) {
	res, err := s.db.Exec(`DELETE FROM snapshots WHERE created_at < ?`, before.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
