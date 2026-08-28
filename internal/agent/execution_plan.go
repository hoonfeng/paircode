package agent

// 统一执行计划模型：融合 PlanRec / Task / ExecutionState 三者为单一模型。
// Plan 包含步骤、子任务、执行状态，由统一的 PlanManager 管理。

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

// ── 常量 ──

// PlanStatus 计划整体状态。
type PlanStatus string

const (
	PlanActive    PlanStatus = "active"
	PlanCompleted PlanStatus = "completed"
	PlanFailed    PlanStatus = "failed"
	PlanCancelled PlanStatus = "cancelled"
)

// StepStatus 步骤状态。
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepInProgress StepStatus = "in_progress"
	StepCompleted  StepStatus = "completed"
	StepFailed     StepStatus = "failed"
	StepCancelled  StepStatus = "cancelled"
)

// ── 核心类型 ──

// Step 执行计划中的一个步骤。
type Step struct {
	ID           string     `json:"id"`
	Description  string     `json:"description"`
	Status       StepStatus `json:"status"`
	Dependencies []string   `json:"dependencies,omitempty"`
	StartedAt    string     `json:"startedAt,omitempty"`
	CompletedAt  string     `json:"completedAt,omitempty"`
	Summary      string     `json:"summary,omitempty"`
}

// ExecutionPlan 统一执行计划，融合 PlanRec / Task / ExecutionState。
type ExecutionPlan struct {
	ID        string     `json:"id"`
	ConvID    string     `json:"convId,omitempty"`
	Status    PlanStatus `json:"status"`
	Task      string     `json:"task"` // 原始用户任务
	Reasoning string     `json:"reasoning,omitempty"`

	Steps     []Step `json:"steps"`     // 执行步骤清单
	LoopCount int    `json:"loopCount"` // 当前循环轮次
	MaxLoops  int    `json:"maxLoops"`  // 最大轮次
	Phase     string `json:"phase"`     // 当前阶段描述

	Errors        []string `json:"errors,omitempty"`
	ModifiedFiles []string `json:"modifiedFiles,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// TaskSummary 任务统计摘要。
type TaskSummary struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
	Cancelled  int `json:"cancelled"`
	Failed     int `json:"failed"`
}

// ── 执行计划管理器 ──

// ExecutionManager 统一管理 ExecutionPlan 的创建、持久化、状态跃迁。
// 替代 PlanManager / TaskManager / ExecStateManager 三者的各自为政。
// 存储路径：.pair/plans/{id}.json（与旧 PlanManager 兼容）
// 线程绑定：.pair/plans/_bindings.json（与旧 PlanManager 兼容）
type ExecutionManager struct {
	mu       sync.RWMutex
	root     string
	plansDir string
	bindings map[string]string // threadID → planID
}

// NewExecutionManager 创建统一执行计划管理器。
func NewExecutionManager(root string) *ExecutionManager {
	dir := filepath.Join(root, ".pair", "plans")
	os.MkdirAll(dir, 0o755)
	em := &ExecutionManager{
		root:     root,
		plansDir: dir,
		bindings: make(map[string]string),
	}
	em.loadBindings()
	return em
}

func (em *ExecutionManager) planFilePath(id string) string {
	return filepath.Join(em.plansDir, id+".json")
}

func (em *ExecutionManager) bindingFilePath() string {
	return filepath.Join(em.plansDir, "_bindings.json")
}

// ── 公共操作 ──

// Create 创建一个新的执行计划。
func (em *ExecutionManager) Create(convID, task, reasoning string) *ExecutionPlan {
	em.mu.Lock()
	defer em.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("plan-%d", time.Now().UnixNano())
	plan := &ExecutionPlan{
		ID:        id,
		ConvID:    convID,
		Status:    PlanActive,
		Task:      task,
		Reasoning: reasoning,
		Steps:     []Step{},
		LoopCount: 0,
		MaxLoops:  25,
		Phase:     "初始化",
		CreatedAt: now,
		UpdatedAt: now,
	}
	em.writePlan(plan)
	return plan
}

// Load 加载指定线程绑定的计划；无绑定返回 nil。
func (em *ExecutionManager) Load(threadID string) *ExecutionPlan {
	em.mu.RLock()
	planID, ok := em.bindings[threadID]
	em.mu.RUnlock()
	if !ok {
		return nil
	}
	return em.Get(planID)
}

// Get 按 ID 获取计划。
func (em *ExecutionManager) Get(id string) *ExecutionPlan {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.readPlan(id)
}

// Save 保存计划到磁盘。
func (em *ExecutionManager) Save(plan *ExecutionPlan) {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
}

// UpdateStatus 更新计划整体状态（自动跃迁）。
func (em *ExecutionManager) UpdateStatus(planID string, status PlanStatus) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.Status = status
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// UpdateStep 更新某个步骤的状态。
func (em *ExecutionManager) UpdateStep(planID, stepID string, status StepStatus, summary string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range plan.Steps {
		if plan.Steps[i].ID == stepID {
			plan.Steps[i].Status = status
			if status == StepInProgress && plan.Steps[i].StartedAt == "" {
				plan.Steps[i].StartedAt = now
			}
			if status == StepCompleted || status == StepFailed {
				plan.Steps[i].CompletedAt = now
			}
			if summary != "" {
				plan.Steps[i].Summary = summary
			}
			plan.UpdatedAt = now
			em.writePlan(plan)
			return true
		}
	}
	return false
}

// AddSteps 向计划追加步骤。
func (em *ExecutionManager) AddSteps(planID string, steps []Step) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.Steps = append(plan.Steps, steps...)
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// ReplaceSteps 全量替换步骤列表（与 update_plan 工具的传整份模式对齐）。
func (em *ExecutionManager) ReplaceSteps(planID string, steps []Step) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.Steps = steps
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// IncrementLoop 增加循环轮次计数。
func (em *ExecutionManager) IncrementLoop(planID string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.LoopCount++
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// SetPhase 更新当前阶段描述。
func (em *ExecutionManager) SetPhase(planID, phase string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.Phase = phase
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// AddError 记录错误。
func (em *ExecutionManager) AddError(planID, errMsg string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	plan.Errors = append(plan.Errors, errMsg)
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// AddModifiedFile 记录被修改的文件。
func (em *ExecutionManager) AddModifiedFile(planID, filePath string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return false
	}
	for _, f := range plan.ModifiedFiles {
		if f == filePath {
			return true // 已存在
		}
	}
	plan.ModifiedFiles = append(plan.ModifiedFiles, filePath)
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	em.writePlan(plan)
	return true
}

// GetProgress 获取计划进度文本。用于 update_plan 工具的展示。
func (em *ExecutionManager) GetProgress(planID string) string {
	em.mu.RLock()
	defer em.mu.RUnlock()
	plan := em.readPlan(planID)
	if plan == nil {
		return "（计划不存在）"
	}

	total := len(plan.Steps)
	done := 0
	inProgress := 0
	for _, s := range plan.Steps {
		switch s.Status {
		case StepCompleted:
			done++
		case StepInProgress:
			inProgress++
		}
	}
	bar := buildProgressBar(done, total, 20)
	return fmt.Sprintf("进度: %s %d/%d (%.0f%%) | 进行中: %d | 轮次: %d/%d",
		bar, done, total, pct(done, total), inProgress, plan.LoopCount, plan.MaxLoops)
}

// GetSummary 获取任务统计摘要。
func (em *ExecutionManager) GetSummary(convID string) TaskSummary {
	em.mu.RLock()
	defer em.mu.RUnlock()
	plans := em.listPlans(convID)
	var s TaskSummary
	for _, p := range plans {
		s.Total++
		switch p.Status {
		case PlanCompleted:
			s.Completed++
		case PlanActive:
			s.InProgress++
		case PlanFailed:
			s.Failed++
		case PlanCancelled:
			s.Cancelled++
		}
	}
	return s
}

// ── 线程绑定 ──

func (em *ExecutionManager) Bind(threadID, planID string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.bindings[threadID] = planID
	em.saveBindings()
}

func (em *ExecutionManager) Unbind(threadID string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	if _, ok := em.bindings[threadID]; !ok {
		return false
	}
	delete(em.bindings, threadID)
	em.saveBindings()
	return true
}

func (em *ExecutionManager) IsBound(threadID string) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	_, ok := em.bindings[threadID]
	return ok
}

// ── 内部 ──

func (em *ExecutionManager) writePlan(plan *ExecutionPlan) {
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(em.planFilePath(plan.ID), data, 0o644)
}

func (em *ExecutionManager) readPlan(id string) *ExecutionPlan {
	data, err := os.ReadFile(em.planFilePath(id))
	if err != nil {
		return nil
	}
	var p ExecutionPlan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	return &p
}

func (em *ExecutionManager) listPlans(convID string) []*ExecutionPlan {
	entries, err := os.ReadDir(em.plansDir)
	if err != nil {
		return nil
	}
	var plans []*ExecutionPlan
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "_bindings.json" {
			continue
		}
		p := em.readPlan(strings.TrimSuffix(e.Name(), ".json"))
		if p != nil && (convID == "" || p.ConvID == convID) {
			plans = append(plans, p)
		}
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].CreatedAt > plans[j].CreatedAt })
	return plans
}

func (em *ExecutionManager) saveBindings() {
	data, _ := json.MarshalIndent(em.bindings, "", "  ")
	os.WriteFile(em.bindingFilePath(), data, 0o644)
}

func (em *ExecutionManager) loadBindings() {
	data, err := os.ReadFile(em.bindingFilePath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &em.bindings)
}

// ── 全局实例 ──

var (
	globalExecManager   *ExecutionManager
	globalExecManagerMu sync.Mutex
)

func InitExecutionManager(root string) *ExecutionManager {
	globalExecManagerMu.Lock()
	defer globalExecManagerMu.Unlock()
	globalExecManager = NewExecutionManager(root)
	return globalExecManager
}

func GetExecutionManager() *ExecutionManager {
	globalExecManagerMu.Lock()
	defer globalExecManagerMu.Unlock()
	return globalExecManager
}

// ── 工具函数 ──
// buildProgressBar 和 pct 定义在 task_tools.go 中
// 旧文件（plan_manager.go / task_manager.go / exec_state.go）中的 PlanStepRec、TaskStatus、ExecStatus、StepRecord
// 等旧类型仍保留在原文件中。新代码应使用 ExecutionPlan / Step / StepStatus / PlanStatus。
