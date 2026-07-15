-- conversations 对话表
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    title TEXT DEFAULT '',
    workspace_root TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    summary TEXT DEFAULT '',
    msg_count INTEGER DEFAULT 0
);

-- messages 消息表（替代 JSONL）
CREATE TABLE IF NOT EXISTS messages (
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
);
CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conv_id, idx);

-- plans 计划表
CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    conv_id TEXT REFERENCES conversations(id),
    status TEXT DEFAULT 'active',
    task TEXT DEFAULT '',
    reasoning TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- plan_steps 计划步骤表
CREATE TABLE IF NOT EXISTS plan_steps (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL REFERENCES plans(id),
    description TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    dependencies TEXT DEFAULT '[]',
    sub_task_ids TEXT DEFAULT '[]',
    sort_order INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_plan_steps_plan ON plan_steps(plan_id);

-- tasks 任务表
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    subject TEXT DEFAULT '',
    description TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    dependencies TEXT DEFAULT '[]',
    conv_id TEXT REFERENCES conversations(id),
    plan_id TEXT REFERENCES plans(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_conv ON tasks(conv_id);

-- memories 记忆表 + FTS5 全文索引
CREATE TABLE IF NOT EXISTS memories (
    name TEXT PRIMARY KEY,
    type TEXT DEFAULT '',
    description TEXT DEFAULT '',
    body TEXT DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    name, body, content=memories, content_rowid=rowid
);

-- project_info 知识库表
CREATE TABLE IF NOT EXISTS project_info (
    path TEXT PRIMARY KEY,
    title TEXT DEFAULT '',
    level TEXT DEFAULT 'detail',
    body TEXT DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- code_entities 代码实体表
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
CREATE INDEX IF NOT EXISTS idx_entities_pkg ON code_entities(package_name);

-- code_relations 代码关系表
CREATE TABLE IF NOT EXISTS code_relations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES code_entities(id),
    target_id INTEGER NOT NULL REFERENCES code_entities(id),
    kind TEXT NOT NULL,
    UNIQUE(source_id, target_id, kind)
);
CREATE INDEX IF NOT EXISTS idx_relations_source ON code_relations(source_id);
CREATE INDEX IF NOT EXISTS idx_relations_target ON code_relations(target_id);

-- evals 评分记录表
CREATE TABLE IF NOT EXISTS evals (
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
);

-- snapshots 快照元数据表
CREATE TABLE IF NOT EXISTS snapshots (
    file_path TEXT NOT NULL,
    snapshot_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY(file_path, snapshot_id)
);
