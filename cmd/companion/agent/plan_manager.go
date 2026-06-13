package agent

// 计划管理器（PlanManager）—— 持久化计划管理，存储在工作区 .pair/plans/ 下。
// 每条计划绑定到一个线程，存为 .json 文件。
// 复刻参考 F:\syproject\伴随式codeagent\src\agent\tools\plan-manager.ts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── 类型定义 ───────────────────────────────────────────────

// PlanStepRec 计划中的一个步骤（持久化版本，含状态）。
type PlanStepRec struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Status       string   `json:"status"` // pending / in_progress / completed / cancelled
	Dependencies []string `json:"dependencies,omitempty"`
	SubTaskIDs   []string `json:"sub_task_ids,omitempty"`
}

// PlanRec 一个完整的执行计划记录（持久化版本，区别于 planner.go 中的 Plan 输出）。
type PlanRec struct {
	ID        string        `json:"id"`
	ThreadID  string        `json:"thread_id"`
	Status    string        `json:"status"` // active / completed / cancelled / replaced
	Task      string        `json:"task"`
	Reasoning string        `json:"reasoning"`
	Steps     []PlanStepRec `json:"steps"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// PlanManager 计划管理器。
type PlanManager struct {
	mu       sync.RWMutex
	root     string
	plansDir string
	// bindings: threadID → planID
	bindings map[string]string
}

// NewPlanManager 创建计划管理器。
func NewPlanManager(root string) *PlanManager {
	dir := filepath.Join(root, ".pair", "plans")
	os.MkdirAll(dir, 0o755)
	return &PlanManager{
		root:     root,
		plansDir: dir,
		bindings: make(map[string]string),
	}
}

func (pm *PlanManager) planFilePath(id string) string {
	return filepath.Join(pm.plansDir, id+".json")
}

func (pm *PlanManager) bindingFilePath() string {
	return filepath.Join(pm.plansDir, "_bindings.json")
}

// ── 公共操作 ───────────────────────────────────────────────

// Create 从任务描述创建一个新计划。
func (pm *PlanManager) Create(threadID, task, reasoning string, steps []PlanStepRec) *PlanRec {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	id := fmt.Sprintf("plan-%d", time.Now().UnixNano())

	if steps == nil {
		steps = []PlanStepRec{}
	}

	plan := &PlanRec{
		ID: id, ThreadID: threadID, Status: "active",
		Task: task, Reasoning: reasoning, Steps: steps,
		CreatedAt: now, UpdatedAt: now,
	}

	pm.writePlanLocked(plan)
	return plan
}

// Load 加载指定线程绑定的计划；无绑定返回 nil。
func (pm *PlanManager) Load(threadID string) *PlanRec {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	planID, ok := pm.bindings[threadID]
	if !ok {
		return nil
	}

	plan := pm.readPlanLocked(planID)
	if plan == nil {
		return nil
	}
	return plan
}

// Save 保存计划到磁盘（覆盖）。
func (pm *PlanManager) Save(plan *PlanRec) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	plan.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	pm.writePlanLocked(plan)
}

// Bind 将当前线程绑定的计划与线程关联。
func (pm *PlanManager) Bind(threadID string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 查找该线程最新创建的计划
	entries, err := os.ReadDir(pm.plansDir)
	if err != nil {
		return false
	}

	var newestPlan *PlanRec
	for _, e := range entries {
		if e.IsDir() || !isJSONFile(e.Name()) || e.Name() == "_bindings.json" {
			continue
		}
		plan := pm.readPlanLocked(trimJSONExt(e.Name()))
		if plan != nil && plan.ThreadID == threadID {
			if newestPlan == nil || plan.CreatedAt > newestPlan.CreatedAt {
				newestPlan = plan
			}
		}
	}

	if newestPlan == nil {
		return false
	}

	pm.bindings[threadID] = newestPlan.ID
	pm.saveBindingsLocked()
	return true
}

// Unbind 解绑线程的计划。
func (pm *PlanManager) Unbind(threadID string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.bindings[threadID]; !ok {
		return false
	}

	delete(pm.bindings, threadID)
	pm.saveBindingsLocked()
	return true
}

// IsBound 检查线程是否绑定了计划。
func (pm *PlanManager) IsBound(threadID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, ok := pm.bindings[threadID]
	return ok
}

// GetProgressText 获取计划进度文本。
func (pm *PlanManager) GetProgressText(threadID string) string {
	plan := pm.Load(threadID)
	if plan == nil {
		return "（未绑定计划）"
	}

	total := len(plan.Steps)
	done := 0
	for _, s := range plan.Steps {
		if s.Status == "completed" {
			done++
		}
	}

	bar := buildProgressBar(done, total, 20)
	return fmt.Sprintf("进度: %s %d/%d (%.0f%%)", bar, done, total, pct(done, total))
}

// ── 内部 ───────────────────────────────────────────────────

func (pm *PlanManager) writePlanLocked(plan *PlanRec) {
	data, _ := json.MarshalIndent(plan, "", "  ")
	os.WriteFile(pm.planFilePath(plan.ID), data, 0o644)
}

func (pm *PlanManager) readPlanLocked(id string) *PlanRec {
	data, err := os.ReadFile(pm.planFilePath(id))
	if err != nil {
		return nil
	}
	var p PlanRec
	if err := json.Unmarshal(data, &p); err != nil {
		return nil
	}
	return &p
}

func (pm *PlanManager) saveBindingsLocked() {
	data, _ := json.MarshalIndent(pm.bindings, "", "  ")
	os.WriteFile(pm.bindingFilePath(), data, 0o644)
}

func (pm *PlanManager) loadBindingsLocked() {
	data, err := os.ReadFile(pm.bindingFilePath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &pm.bindings)
}

// ── 全局实例 ──────────────────────────────────────────────

var (
	globalPlanManager   *PlanManager
	globalPlanManagerMu sync.Mutex
)

// InitPlanManager 初始化全局计划管理器。
func InitPlanManager(root string) *PlanManager {
	globalPlanManagerMu.Lock()
	defer globalPlanManagerMu.Unlock()
	globalPlanManager = NewPlanManager(root)
	globalPlanManager.loadBindingsLocked()
	return globalPlanManager
}

// GetPlanManager 获取全局计划管理器。
func GetPlanManager() *PlanManager {
	globalPlanManagerMu.Lock()
	defer globalPlanManagerMu.Unlock()
	return globalPlanManager
}

// ── 小工具 ─────────────────────────────────────────────────

func isJSONFile(name string) bool {
	return len(name) > 5 && name[len(name)-5:] == ".json"
}

func trimJSONExt(name string) string {
	return name[:len(name)-5]
}
