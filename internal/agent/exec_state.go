package agent

// 执行状态持久化 —— 将编排循环的完整状态存储到 .pair/tasks/execution-states/ 下。
// 支持断点续跑：服务重启后自动恢复未完成的执行状态。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// ── 类型定义 ───────────────────────────────────────────────

// ExecStatus 执行状态。
type ExecStatus string

const (
	ExecRunning   ExecStatus = "running"
	ExecPaused    ExecStatus = "paused"
	ExecCompleted ExecStatus = "completed"
	ExecFailed    ExecStatus = "failed"
	ExecCancelled ExecStatus = "cancelled"
)

// StepRecord 一轮 loop 完成的记录。
type StepRecord struct {
	StepNum     int    `json:"stepNum"`
	Description string `json:"description"`
	Status      string `json:"status"` // completed / failed / skipped
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	Summary     string `json:"summary,omitempty"`
}

// ExecutionState 编排循环的完整状态。
type ExecutionState struct {
	ID            string            `json:"id"`            // 唯一运行标识
	Task          string            `json:"task"`          // 原始用户任务
	MissionTask   string            `json:"missionTask"`   // 当前执行的任务文本
	LoopCount     int               `json:"loopCount"`     // 当前轮次
	MaxLoops      int               `json:"maxLoops"`      // 最大轮次
	Phase         string            `json:"phase"`         // 当前阶段描述
	Status        ExecStatus        `json:"status"`        // 运行状态
	CreatedAt     string            `json:"createdAt"`
	UpdatedAt     string            `json:"updatedAt"`
	CompletedSteps []StepRecord     `json:"completedSteps,omitempty"`
	Errors        []string          `json:"errors,omitempty"`
	ModifiedFiles []string          `json:"modifiedFiles,omitempty"`
	ConvID        string            `json:"convId,omitempty"` // 关联对话 ID
}

// ExecStateManager 执行状态管理器（并发安全）。
type ExecStateManager struct {
	mu          sync.Mutex
	root        string
	statesDir   string
}

// NewExecStateManager 创建状态管理器。
func NewExecStateManager(root string) *ExecStateManager {
	dir := filepath.Join(root, ".pair", "tasks", "execution-states")
	os.MkdirAll(dir, 0755)
	return &ExecStateManager{root: root, statesDir: dir}
}

func (m *ExecStateManager) stateFilePath(id string) string {
	return filepath.Join(m.statesDir, id+".json")
}

// ── 公共操作 ───────────────────────────────────────────────

// Create 创建一个新的执行状态。
func (m *ExecStateManager) Create(task string, maxLoops int, convID string) *ExecutionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().Format("2006-01-02 15:04:05")
	id := fmt.Sprintf("exec_%s_%d", time.Now().Format("20060102_150405"), time.Now().UnixMilli()%1000)
	state := &ExecutionState{
		ID:        id,
		Task:      task,
		MissionTask: task,
		LoopCount: 0,
		MaxLoops:  maxLoops,
		Phase:     "初始化",
		Status:    ExecRunning,
		CreatedAt: now,
		UpdatedAt: now,
		ConvID:    convID,
	}
	m.writeStateLocked(state)
	return state
}

// Save 保存状态更新。
func (m *ExecStateManager) Save(state *ExecutionState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	m.writeStateLocked(state)
}

// Load 加载指定 ID 的状态。
func (m *ExecStateManager) Load(id string) *ExecutionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readStateLocked(id)
}

// ListAll 列出所有执行状态（按更新时间倒序）。
func (m *ExecStateManager) ListAll() []*ExecutionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.statesDir)
	if err != nil {
		return nil
	}
	states := make([]*ExecutionState, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		s := m.readStateLocked(id)
		if s != nil {
			states = append(states, s)
		}
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].UpdatedAt > states[j].UpdatedAt
	})
	return states
}

// FindInterrupted 查找中断的（paused/running）执行状态，按更新时间倒序返回第一个。
func (m *ExecStateManager) FindInterrupted() *ExecutionState {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, err := os.ReadDir(m.statesDir)
	if err != nil {
		return nil
	}
	var candidates []*ExecutionState
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		s := m.readStateLocked(id)
		if s != nil && (s.Status == ExecRunning || s.Status == ExecPaused) {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].UpdatedAt > candidates[j].UpdatedAt
	})
	return candidates[0]
}

// MarkCompleted 标记为已完成。
func (m *ExecStateManager) MarkCompleted(state *ExecutionState, summary string) {
	state.Status = ExecCompleted
	state.Phase = "全部完成"
	if summary != "" {
		state.CompletedSteps = append(state.CompletedSteps, StepRecord{
			StepNum:     state.LoopCount,
			Description: "任务完成摘要",
			Status:      "completed",
			CompletedAt: time.Now().Format("2006-01-02 15:04:05"),
			Summary:     summary,
		})
	}
	m.Save(state)
}

// MarkFailed 标记为失败。
func (m *ExecStateManager) MarkFailed(state *ExecutionState, err error) {
	state.Status = ExecFailed
	state.Errors = append(state.Errors, err.Error())
	m.Save(state)
}

// RecordFileChange 记录文件变更。
func (m *ExecStateManager) RecordFileChange(state *ExecutionState, filePath string) {
	for _, f := range state.ModifiedFiles {
		if f == filePath {
			return // 已存在
		}
	}
	state.ModifiedFiles = append(state.ModifiedFiles, filePath)
	m.Save(state)
}

// GetSummary 获取状态摘要（用于跨对话注入）。
func (state *ExecutionState) GetSummary() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("## 项目当前状态\n\n"))
	b.WriteString(fmt.Sprintf("- **运行 ID**: %s\n", state.ID))
	b.WriteString(fmt.Sprintf("- **任务**: %s\n", state.Task))
	b.WriteString(fmt.Sprintf("- **状态**: %s\n", state.Status))
	b.WriteString(fmt.Sprintf("- **当前轮次**: %d/%d\n", state.LoopCount, state.MaxLoops))
	b.WriteString(fmt.Sprintf("- **当前阶段**: %s\n", state.Phase))
	if len(state.ModifiedFiles) > 0 {
		b.WriteString(fmt.Sprintf("- **已修改文件**: %d 个\n", len(state.ModifiedFiles)))
		for _, f := range state.ModifiedFiles {
			b.WriteString(fmt.Sprintf("  - `%s`\n", f))
		}
	}
	if len(state.Errors) > 0 {
		b.WriteString(fmt.Sprintf("- **错误**: %d 个\n", len(state.Errors)))
		for _, e := range state.Errors {
			b.WriteString(fmt.Sprintf("  - `%s`\n", e))
		}
	}
	if len(state.CompletedSteps) > 0 {
		b.WriteString(fmt.Sprintf("- **已完成步骤**: %d 个\n", len(state.CompletedSteps)))
	}
	return b.String()
}

// ── 内部 ───────────────────────────────────────────────────

func (m *ExecStateManager) writeStateLocked(state *ExecutionState) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(m.stateFilePath(state.ID), data, 0644)
}

func (m *ExecStateManager) readStateLocked(id string) *ExecutionState {
	data, err := os.ReadFile(m.stateFilePath(id))
	if err != nil {
		return nil
	}
	var state ExecutionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return &state
}

// ── 全局实例 ──────────────────────────────────────────────

var (
	globalExecStateManager   *ExecStateManager
	globalExecStateManagerMu sync.Mutex
)

// InitExecStateManager 初始化全局执行状态管理器。
func InitExecStateManager(root string) *ExecStateManager {
	globalExecStateManagerMu.Lock()
	defer globalExecStateManagerMu.Unlock()
	globalExecStateManager = NewExecStateManager(root)
	return globalExecStateManager
}

// GetExecStateManager 获取全局执行状态管理器。
func GetExecStateManager() *ExecStateManager {
	globalExecStateManagerMu.Lock()
	defer globalExecStateManagerMu.Unlock()
	return globalExecStateManager
}

// ── Panic 恢复 ────────────────────────────────────────────

// RunWithRecovery 用 panic recover 包装执行函数 fn。
// 如果 fn panic，自动：
//   - 通过 DebugLogger 记录 panic 日志（含堆栈）
//   - 将 state 标记为失败
//   - 返回 panic 错误
//
// 用法：defer execMgr.RunWithRecovery(&state, "runOrchestrationLoop", ctx, convID)()
func (m *ExecStateManager) RunWithRecovery(state **ExecutionState, source string, ctx map[string]string) func() {
	return func() {
		if r := recover(); r != nil {
			// 构建堆栈信息
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			stackStr := string(stack[:n])

			// 记录到 debug 日志
			WritePanic(source, r, stackStr, ctx)

			// 标记状态失败
			if *state != nil {
				(*state).Status = ExecFailed
				(*state).Errors = append((*state).Errors, fmt.Sprintf("PANIC: %v", r))
				m.Save(*state)
			}

			// 重新 panic（外部 recover 可决定是否吞掉）
			panic(r)
		}
	}
}

// RecoverPanic 静默恢复 panic（不重新 panic，仅记录日志并标记状态失败）。
// 适用于编排循环等不希望因 panic 导致整个服务崩溃的场景。
func (m *ExecStateManager) RecoverPanic(state **ExecutionState, source string, ctx map[string]string) {
	if r := recover(); r != nil {
		stack := make([]byte, 4096)
		n := runtime.Stack(stack, false)

		WritePanic(source, r, string(stack[:n]), ctx)

		if *state != nil {
			(*state).Status = ExecFailed
			(*state).Errors = append((*state).Errors, fmt.Sprintf("PANIC: %v (已静默恢复)", r))
			m.Save(*state)
		}
	}
}

