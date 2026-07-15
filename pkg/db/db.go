package db

import "time"

// DB 是数据库接口，定义所有存储操作。
// 后续实现可以使用 SQLite 或其他后端。
type DB interface {
	// 对话
	CreateConversation(conv *Conversation) error
	GetConversation(id string) (*Conversation, error)
	UpdateConversation(conv *Conversation) error
	ListConversations() ([]*Conversation, error)
	DeleteConversation(id string) error

	// 消息
	AppendMessage(msg *Message) error
	GetMessages(convID string, offset, limit int) ([]*Message, error)
	GetMessageCount(convID string) (int, error)
	DeleteMessages(convID string) error

	// 计划
	CreatePlan(plan *Plan) error
	GetPlan(id string) (*Plan, error)
	UpdatePlan(plan *Plan) error
	ListPlans(convID string) ([]*Plan, error)
	DeletePlan(id string) error

	// 计划步骤
	CreatePlanStep(step *PlanStep) error
	UpdatePlanStep(step *PlanStep) error
	GetPlanSteps(planID string) ([]*PlanStep, error)

	// 任务
	CreateTask(task *Task) error
	GetTask(id string) (*Task, error)
	UpdateTask(task *Task) error
	ListTasks(convID string, status string) ([]*Task, error)
	DeleteTask(id string) error

	// 记忆
	WriteMemory(mem *Memory) error
	ReadMemory(name string) (*Memory, error)
	SearchMemories(query string) ([]*Memory, error)
	ListMemories() ([]*Memory, error)
	DeleteMemory(name string) error

	// 知识库
	WriteProjectInfo(pi *ProjectInfo) error
	ReadProjectInfo(path string) (*ProjectInfo, error)
	SearchProjectInfo(query string) ([]*ProjectInfo, error)
	ListProjectInfo() ([]*ProjectInfo, error)
	DeleteProjectInfo(path string) error

	// 代码实体
	UpsertCodeEntity(entity *CodeEntity) (int64, error)
	SearchCodeEntities(query string, kind string) ([]*CodeEntity, error)
	GetCodeEntitiesByFile(filePath string) ([]*CodeEntity, error)

	// 代码关系
	UpsertCodeRelation(sourceID, targetID int64, kind string) error
	GetRelations(entityID int64) ([]*CodeRelation, error)

	// 评分
	CreateEval(eval *Eval) error
	ListEvals(limit int) ([]*Eval, error)

	// 快照
	CreateSnapshot(snap *Snapshot) error
	ListSnapshots(filePath string) ([]*Snapshot, error)
	DeleteSnapshots(before time.Time) (int, error)

	// 维护
	Close() error
}
