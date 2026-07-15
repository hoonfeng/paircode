// Package db 提供 SQLite 存储层，用于替代现有的 JSON/JSONL/MD 文件存储。
// 设计目标：
//   - 单一数据库文件 .pair/pair.db
//   - 支持事务
//   - FTS5 全文检索
//   - 渐进式迁移：与现有 JSON 存储并存，逐步接管
package db

import "time"

// ── 常量 ──

// DefaultDBPath 数据库文件默认路径（相对于工作区根）。
const DefaultDBPath = ".pair/pair.db"

// ── 对话相关 ──

// Conversation 对话元数据（对应 conversations 表）。
type Conversation struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	WorkspaceRoot string    `json:"workspaceRoot"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Summary       string    `json:"summary"`
	MsgCount      int       `json:"msgCount"`
}

// Message 消息（对应 messages 表）。
type Message struct {
	ID         int64     `json:"id"`
	ConvID     string    `json:"convId"`
	Idx        int       `json:"idx"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	ToolCalls  string    `json:"toolCalls,omitempty"`  // JSON
	ToolCallID string    `json:"toolCallId,omitempty"`
	Name       string    `json:"name,omitempty"`
	Reasoning  string    `json:"reasoning,omitempty"`
	Segments   string    `json:"segments,omitempty"`    // JSON
	CreatedAt  time.Time `json:"createdAt"`
}

// ── 计划相关 ──

// Plan 执行计划（对应 plans 表）。
type Plan struct {
	ID        string    `json:"id"`
	ConvID    string    `json:"convId,omitempty"`
	Status    string    `json:"status"` // active / completed / cancelled / replaced
	Task      string    `json:"task"`
	Reasoning string    `json:"reasoning"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// PlanStep 计划步骤（对应 plan_steps 表）。
type PlanStep struct {
	ID           string    `json:"id"`
	PlanID       string    `json:"planId"`
	Description  string    `json:"description"`
	Status       string    `json:"status"` // pending / in_progress / completed / cancelled
	Dependencies string    `json:"dependencies,omitempty"` // JSON array
	SubTaskIDs   string    `json:"subTaskIds,omitempty"`   // JSON array
	SortOrder    int       `json:"sortOrder"`
}

// ── 任务相关 ──

// Task 子任务（对应 tasks 表）。
type Task struct {
	ID           string    `json:"id"`
	Subject      string    `json:"subject"`
	Description  string    `json:"description"`
	Status       string    `json:"status"` // pending / in_progress / completed / cancelled
	Dependencies string    `json:"dependencies,omitempty"` // JSON array
	ConvID       string    `json:"convId,omitempty"`
	PlanID       string    `json:"planId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ── 记忆相关 ──

// Memory 长时记忆（对应 memories 表 + FTS5 索引）。
type Memory struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ── 知识库相关 ──

// ProjectInfo 项目知识库条目（对应 project_info 表）。
type ProjectInfo struct {
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	Level     string    `json:"level"` // overview / module / detail
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ── 代码实体相关 ──

// CodeEntity 代码实体（函数、类型、变量等，对应 code_entities 表）。
type CodeEntity struct {
	ID          int64  `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	FilePath    string `json:"filePath"`
	Line        int    `json:"line"`
	Signature   string `json:"signature"`
	PackageName string `json:"packageName"`
	Module      string `json:"module"`
}

// CodeRelation 代码实体间关系（调用、导入、包含等，对应 code_relations 表）。
type CodeRelation struct {
	ID       int64  `json:"id"`
	SourceID int64  `json:"sourceId"`
	TargetID int64  `json:"targetId"`
	Kind     string `json:"kind"` // calls / imports / contains / implements
}

// ── 评分相关 ──

// Eval 评分记录（对应 evals 表）。
type Eval struct {
	ID          string    `json:"id"`
	Task        string    `json:"task"`
	AgentModel  string    `json:"agentModel"`
	JudgeModel  string    `json:"judgeModel"`
	Scores      string    `json:"scores"`      // JSON
	Total       int       `json:"total"`
	Strengths   string    `json:"strengths"`
	Weaknesses  string    `json:"weaknesses"`
	Feedback    string    `json:"feedback"`
	ToolCalls   int       `json:"toolCalls"`
	ToolErrors  int       `json:"toolErrors"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ── 快照相关 ──

// Snapshot 文件快照元数据（对应 snapshots 表，文件内容仍在磁盘）。
type Snapshot struct {
	FilePath   string    `json:"filePath"`
	SnapshotID string    `json:"snapshotId"`
	CreatedAt  time.Time `json:"createdAt"`
}
