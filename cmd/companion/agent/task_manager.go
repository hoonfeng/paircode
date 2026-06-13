package agent

// 任务管理器（TaskManager）—— 持久化任务管理，存储在工作区 .pair/tasks/ 下。
// 每条任务存为一个 .json 文件。
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\tools\task-manager.ts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 类型定义 ───────────────────────────────────────────────

// TaskStatus 任务状态。
type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskCancelled  TaskStatus = "cancelled"
)

// Task 一条子任务。
type Task struct {
	ID           string     `json:"id"`
	Subject      string     `json:"subject"`
	Description  string     `json:"description"`
	Status       TaskStatus `json:"status"`
	Dependencies []string   `json:"dependencies"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// TaskSummary 任务统计摘要。
type TaskSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
	Cancelled  int `json:"cancelled"`
}

// TaskManager 任务管理器（并发安全）。
type TaskManager struct {
	mu       sync.RWMutex
	root     string
	tasksDir string
}

// NewTaskManager 创建任务管理器。
func NewTaskManager(root string) *TaskManager {
	dir := filepath.Join(root, ".pair", "tasks")
	os.MkdirAll(dir, 0o755)
	return &TaskManager{root: root, tasksDir: dir}
}

func (tm *TaskManager) taskFilePath(id string) string {
	return filepath.Join(tm.tasksDir, id+".json")
}

// ── 公共操作 ───────────────────────────────────────────────

func (tm *TaskManager) Create(subject, description string, dependencies []string) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	if dependencies == nil {
		dependencies = []string{}
	}
	task := &Task{
		ID: id, Subject: subject, Description: description,
		Status: TaskPending, Dependencies: dependencies,
		CreatedAt: now, UpdatedAt: now,
	}
	tm.writeTaskLocked(task)
	return task
}

func (tm *TaskManager) Update(id string, updates map[string]any) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	task := tm.readTaskLocked(id)
	if task == nil {
		return nil
	}
	if v, ok := updates["status"]; ok {
		if s, ok2 := v.(string); ok2 {
			task.Status = TaskStatus(s)
		}
	}
	if v, ok := updates["subject"]; ok {
		if s, ok2 := v.(string); ok2 {
			task.Subject = s
		}
	}
	if v, ok := updates["description"]; ok {
		if s, ok2 := v.(string); ok2 {
			task.Description = s
		}
	}
	if v, ok := updates["dependencies"]; ok {
		switch arr := v.(type) {
		case []string:
			task.Dependencies = arr
		case []any:
			deps := make([]string, 0, len(arr))
			for _, a := range arr {
				if s, ok3 := a.(string); ok3 {
					deps = append(deps, s)
				}
			}
			task.Dependencies = deps
		}
	}
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	tm.writeTaskLocked(task)
	return task
}

func (tm *TaskManager) Get(id string) *Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.readTaskLocked(id)
}

func (tm *TaskManager) List(status TaskStatus) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tasks := tm.readAllLocked()
	if status == "" {
		sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt > tasks[j].CreatedAt })
		return tasks
	}
	filtered := make([]*Task, 0)
	for _, t := range tasks {
		if t.Status == status {
			filtered = append(filtered, t)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].CreatedAt > filtered[j].CreatedAt })
	return filtered
}

func (tm *TaskManager) Delete(id string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	p := tm.taskFilePath(id)
	if _, err := os.Stat(p); err != nil {
		return false
	}
	os.Remove(p)
	return true
}

func (tm *TaskManager) GetSummary() TaskSummary {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	tasks := tm.readAllLocked()
	var s TaskSummary
	s.Total = len(tasks)
	for _, t := range tasks {
		switch t.Status {
		case TaskCompleted:
			s.Completed++
		case TaskInProgress:
			s.InProgress++
		case TaskPending:
			s.Pending++
		case TaskCancelled:
			s.Cancelled++
		}
	}
	return s
}

func (tm *TaskManager) GetReady() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	all := tm.readAllLocked()
	completed := map[string]bool{}
	for _, t := range all {
		if t.Status == TaskCompleted {
			completed[t.ID] = true
		}
	}
	ready := make([]*Task, 0)
	for _, t := range all {
		if t.Status != TaskPending {
			continue
		}
		allDepsMet := true
		for _, dep := range t.Dependencies {
			if !completed[dep] {
				allDepsMet = false
				break
			}
		}
		if allDepsMet {
			ready = append(ready, t)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i].CreatedAt < ready[j].CreatedAt })
	return ready
}

// BlockedTask 被阻塞的任务及阻塞原因。
type BlockedTask struct {
	Task      *Task   `json:"task"`
	BlockedBy []*Task `json:"blocked_by"`
}

func (tm *TaskManager) GetBlocked() []BlockedTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	all := tm.readAllLocked()
	taskMap := map[string]*Task{}
	for _, t := range all {
		taskMap[t.ID] = t
	}
	blocked := make([]BlockedTask, 0)
	for _, t := range all {
		if t.Status != TaskPending || len(t.Dependencies) == 0 {
			continue
		}
		blockers := make([]*Task, 0)
		for _, dep := range t.Dependencies {
			depTask, exists := taskMap[dep]
			if !exists || depTask.Status != TaskCompleted {
				if depTask == nil {
					depTask = &Task{ID: dep, Subject: "(已删除)", Status: TaskCancelled}
				}
				blockers = append(blockers, depTask)
			}
		}
		if len(blockers) > 0 {
			blocked = append(blocked, BlockedTask{Task: t, BlockedBy: blockers})
		}
	}
	return blocked
}

// ── 全局实例 ──────────────────────────────────────────────

var (
	globalTaskManager   *TaskManager
	globalTaskManagerMu sync.Mutex
)

// InitTaskManager 初始化全局任务管理器实例。
func InitTaskManager(root string) *TaskManager {
	globalTaskManagerMu.Lock()
	defer globalTaskManagerMu.Unlock()
	globalTaskManager = NewTaskManager(root)
	return globalTaskManager
}

// GetTaskManager 获取全局任务管理器（未初始化则返回 nil）。
func GetTaskManager() *TaskManager {
	globalTaskManagerMu.Lock()
	defer globalTaskManagerMu.Unlock()
	return globalTaskManager
}

// ── 内部 ───────────────────────────────────────────────────

func (tm *TaskManager) writeTaskLocked(task *Task) {
	data, _ := json.MarshalIndent(task, "", "  ")
	os.WriteFile(tm.taskFilePath(task.ID), data, 0o644)
}

func (tm *TaskManager) readTaskLocked(id string) *Task {
	data, err := os.ReadFile(tm.taskFilePath(id))
	if err != nil {
		return nil
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil
	}
	return &t
}

func (tm *TaskManager) readAllLocked() []*Task {
	entries, err := os.ReadDir(tm.tasksDir)
	if err != nil {
		return []*Task{}
	}
	tasks := make([]*Task, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tm.tasksDir, e.Name()))
		if err != nil {
			continue
		}
		var t Task
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		tasks = append(tasks, &t)
	}
	return tasks
}
