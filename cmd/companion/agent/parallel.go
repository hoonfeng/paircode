// parallel.go 子 Agent 并行执行机制：共享上下文池 + 并行编排 + 结果汇总。
//
// 核心设计原则：
// 1. **共享上下文不隔离** — 所有子 Agent 读写同一 SharedContext，避免重复收集数据浪费 token。
// 2. **上下文累积** — 子 Agent 发现的信息自动注入共享池，后续子 Agent 可复用。
// 3. **文件缓存** — 一个子 Agent 读取过的文件，其他子 Agent 直接取缓存，不重复读。
// 4. **变量版本化** — 冲突检测（同一变量被多个子 Agent 修改时标记冲突供汇总阶段处理）。
// 5. **知识去重** — 相同 Key 的知识条目不重复追加（后写入覆盖先写入，标记版本）。

package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── 共享上下文池 ─────────────────────────────────────────────

// KnowledgeEntry 一条知识条目（子 Agent 发现的关键信息）。
type KnowledgeEntry struct {
	AgentName string    `json:"agent_name"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Summary   string    `json:"summary"`
	Timestamp time.Time `json:"timestamp"`
}

// VarVersion 带版本的共享变量（冲突检测用）。
type VarVersion struct {
	AgentName string    `json:"agent_name"`
	Value     string    `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// SharedContext 共享上下文池——所有子 Agent 并发读写，避免重复收集数据。
//
// 使用方法：
//
//	pool := NewSharedContext()
//	pool.StoreFile("main.go", "package main...")  // 一个子 Agent 读后缓存
//	pool.AddKnowledge("planner", "架构", "MVC", "使用 MVC 架构")
//	pool.SetVar("db_host", "分析器", "localhost:3306")
//
//	// 另一个子 Agent 无需重复读取：
//	content, ok := pool.LoadFile("main.go")
//	entry, ok := pool.GetKnowledge("架构")
type SharedContext struct {
	mu sync.RWMutex

	// 文件缓存：path → content（子 Agent 读过的文件，其他 Agent 直接取缓存）
	files map[string]string

	// 知识累积：key → []KnowledgeEntry（各子 Agent 发现的信息，最新优先）
	knowledge map[string]*KnowledgeEntry

	// 共享变量（带版本历史，冲突检测用）
	variables map[string][]VarVersion

	// 诊断日志：记录每个子 Agent 的读写操作（供汇总阶段分析）
	accessLog []string
}

// NewSharedContext 创建共享上下文池。
func NewSharedContext() *SharedContext {
	return &SharedContext{
		files:     make(map[string]string),
		knowledge: make(map[string]*KnowledgeEntry),
		variables: make(map[string][]VarVersion),
	}
}

// ── 文件缓存 ────────────────────────────────────────────────

// StoreFile 缓存文件内容。agentName 为缓存者名（日志用）。
func (sc *SharedContext) StoreFile(agentName, path, content string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.files[path] = content
	sc.accessLog = append(sc.accessLog, fmt.Sprintf("[%s] cache file: %s (%d bytes)", agentName, path, len(content)))
}

// LoadFile 取缓存的文件内容。ok=false 表示未缓存，调用方需自行读取并缓存。
func (sc *SharedContext) LoadFile(path string) (content string, ok bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	content, ok = sc.files[path]
	return
}

// HasFile 检查文件是否已缓存。
func (sc *SharedContext) HasFile(path string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	_, ok := sc.files[path]
	return ok
}

// CachedFiles 返回所有已缓存文件路径（已排序）。
func (sc *SharedContext) CachedFiles() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	paths := make([]string, 0, len(sc.files))
	for p := range sc.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// ── 知识累积 ────────────────────────────────────────────────

// AddKnowledge 添加一条知识。同 Key 后写入覆盖先写入（保留最新值）。
// agentName 为发现者名，summary 为摘要（结果汇总用）。
func (sc *SharedContext) AddKnowledge(agentName, key, value, summary string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.knowledge[key] = &KnowledgeEntry{
		AgentName: agentName,
		Key:       key,
		Value:     value,
		Summary:   summary,
		Timestamp: time.Now(),
	}
	sc.accessLog = append(sc.accessLog, fmt.Sprintf("[%s] add knowledge: %s = %s", agentName, key, summary))
}

// GetKnowledge 按 Key 取知识条目。ok=false 表示不存在。
func (sc *SharedContext) GetKnowledge(key string) (entry *KnowledgeEntry, ok bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	entry, ok = sc.knowledge[key]
	return
}

// GetAllKnowledge 返回全部知识条目（按时间倒序）。
func (sc *SharedContext) GetAllKnowledge() []*KnowledgeEntry {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	entries := make([]*KnowledgeEntry, 0, len(sc.knowledge))
	for _, v := range sc.knowledge {
		entries = append(entries, v)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp.After(entries[j].Timestamp) })
	return entries
}

// KnowledgeSummary 生成知识累积的文本摘要（供注入子 Agent 上下文）。
func (sc *SharedContext) KnowledgeSummary() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	if len(sc.knowledge) == 0 && len(sc.files) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[共享上下文]\n")
	if len(sc.knowledge) > 0 {
		b.WriteString("已发现的关键信息：\n")
		for _, v := range sc.knowledge {
			fmt.Fprintf(&b, "  - %s: %s（由 %s 发现）\n", v.Key, v.Summary, v.AgentName)
		}
	}
	if len(sc.files) > 0 {
		b.WriteString("已缓存的文件：\n")
		for _, p := range sortKeys(sc.files) {
			b.WriteString("  - " + p + "\n")
		}
		b.WriteString("（以上文件无需重复读取，直接用 ctx_read_file 获取缓存内容）\n")
	}
	return b.String()
}

// ── 共享变量（版本化 + 冲突检测）───────────────────────────────

// SetVar 设置共享变量的当前值（追加版本历史）。
func (sc *SharedContext) SetVar(agentName, name, value string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.variables[name] = append(sc.variables[name], VarVersion{
		AgentName: agentName,
		Value:     value,
		Timestamp: time.Now(),
	})
	sc.accessLog = append(sc.accessLog, fmt.Sprintf("[%s] set var: %s = %s", agentName, name, value))
}

// GetVar 取共享变量的当前值（最新版本）。ok=false 表示未设置。
func (sc *SharedContext) GetVar(name string) (value string, ok bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	versions, exists := sc.variables[name]
	if !exists || len(versions) == 0 {
		return "", false
	}
	return versions[len(versions)-1].Value, true
}

// GetVarHistory 取共享变量的完整版本历史（冲突检测用）。
func (sc *SharedContext) GetVarHistory(name string) []VarVersion {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	versions, ok := sc.variables[name]
	if !ok {
		return nil
	}
	out := make([]VarVersion, len(versions))
	copy(out, versions)
	return out
}

// DetectConflicts 检测所有共享变量的冲突：同一变量被 2+ 不同子 Agent 修改且值不同。
// 返回冲突列表，每项含变量名和所有不同版本（按时间倒序）。
type VarConflict struct {
	Name     string       `json:"name"`
	Versions []VarVersion `json:"versions"`
}

func (sc *SharedContext) DetectConflicts() []VarConflict {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	var conflicts []VarConflict
	for name, versions := range sc.variables {
		if len(versions) < 2 {
			continue
		}
		// 检查是否有 2+ 不同 agent 设置了不同值
		agentValues := map[string]string{}
		for _, v := range versions {
			agentValues[v.AgentName] = v.Value
		}
		if len(agentValues) < 2 {
			continue // 同一 agent 更新多次但值不同不算冲突（时序覆盖）
		}
		// 检查值是否真的不同
		seen := map[string]bool{}
		distinctVals := 0
		for _, v := range agentValues {
			if !seen[v] {
				seen[v] = true
				distinctVals++
			}
		}
		if distinctVals < 2 {
			continue
		}
		conflicts = append(conflicts, VarConflict{
			Name:     name,
			Versions: append([]VarVersion(nil), versions...),
		})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Name < conflicts[j].Name })
	return conflicts
}

// ── 子 Agent 上下文注入 ──────────────────────────────────────

// BuildContextInject 构建注入到子 Agent 的上下文文本。
// 包含：已缓存文件列表 + 已有知识 + 共享变量当前值。
// 子 Agent 看到后可以直接用 ctx_read_file / ctx_get_knowledge 获取内容，
// 避免重复读取/发现已存在的信息。
func (sc *SharedContext) BuildContextInject(agentName string) string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var b strings.Builder
	b.WriteString("【共享上下文 — 其他子 Agent 已完成的工作，请复用以下信息，避免重复收集】\n")

	// 知识累积
	if len(sc.knowledge) > 0 {
		b.WriteString("\n已发现的关键信息（直接使用，无需重复调查）：\n")
		for _, v := range sc.knowledge {
			b.WriteString(fmt.Sprintf("  • %s: %s\n", v.Key, v.Summary))
		}
	}

	// 文件缓存
	if len(sc.files) > 0 {
		b.WriteString("\n已读文件缓存（用 ctx_read_file 直接获取，无需再 read_file）：\n")
		for _, p := range sortKeys(sc.files) {
			b.WriteString("  • " + p + "\n")
		}
	}

	// 共享变量
	if len(sc.variables) > 0 {
		b.WriteString("\n已设置的共享变量（用 ctx_get_var 读取）：\n")
		for _, name := range sortKeys(sc.variables) {
			versions := sc.variables[name]
			if len(versions) > 0 {
				b.WriteString(fmt.Sprintf("  • %s = %s\n", name, versions[len(versions)-1].Value))
			}
		}
	}

	b.WriteString("\n工具提示：\n")
	b.WriteString("  - ctx_read_file(path)：获取已缓存的文件内容（不触发磁盘读）\n")
	b.WriteString("  - ctx_get_knowledge(key)：获取已有知识条目\n")
	b.WriteString("  - ctx_get_var(name)：获取共享变量值\n")
	b.WriteString("  - ctx_set_var(name, value)：设置共享变量（供后续子 Agent 用）\n")
	b.WriteString("  - ctx_add_knowledge(key, value, summary)：添加新知识（覆盖同名 Key）\n")

	return b.String()
}

// ─── 并行执行引擎 ─────────────────────────────────────────────

// taskGroup 一批可并行的子任务索引。
type taskGroup struct {
	indices []int          // subTasks 中的下标
	deps    map[string]bool // 本批任务依赖的 taskID（应在之前批次完成）
}

// SubTask 一个可并行执行的子任务。
type SubTask struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	AgentName    string   `json:"agent_name"`    // 执行该任务的子 Agent 名
	Input        string   `json:"input"`         // 任务输入
	Dependencies []string `json:"dependencies"`  // 依赖的子任务 ID（= 串行）
	IsContextual bool     `json:"is_contextual"` // 是否只用于收集上下文（无产出结果不影响汇总）
}

// SubTaskResult 子任务执行结果。
type SubTaskResult struct {
	TaskID      string           `json:"task_id"`
	AgentName   string           `json:"agent_name"`
	Output      string           `json:"output"`
	Error       string           `json:"error,omitempty"`
	FinishedMsg []Message        `json:"-"` // 子 Loop 返回的消息历史（供 AnalyseRun 用）
	Knowledge   []KnowledgeEntry `json:"knowledge,omitempty"`
}

// DecomposeResult 任务分解结果。
type DecomposeResult struct {
	Reasoning     string    `json:"reasoning"`
	SubTasks      []SubTask `json:"sub_tasks"`
	ParallelGroups [][]int  `json:"parallel_groups"` // 可并行执行的任务组（下标索引）
}

// ParallelOrchestrator 并行编排器。
// 负责：任务分解 → 依赖分析 → 并发执行 → 结果汇总。
type ParallelOrchestrator struct {
	Parent      *Loop           // 父 Loop（用于创建子 Loop）
	Tree        *AgentTree      // Agent 编排树
	ContextPool *SharedContext  // 共享上下文池
}

// NewParallelOrchestrator 创建并行编排器。parent 为父 Loop（提供 Provider 和 Registry），
// tree 为 Agent 编排树，pool 为共享上下文池（nil 则自动创建）。
func NewParallelOrchestrator(parent *Loop, tree *AgentTree, pool *SharedContext) *ParallelOrchestrator {
	if pool == nil {
		pool = NewSharedContext()
	}
	return &ParallelOrchestrator{
		Parent:      parent,
		Tree:        tree,
		ContextPool: pool,
	}
}

// DecomposeTask 用 Planner 把任务分解为可并行执行的子任务。
// 分析步骤依赖关系，识别可以并行的任务组。
//
// 策略：
// - Planner 输出 PlanStep（含 dependencies）
// - 无依赖或依赖已完成的步骤 → 可并行
// - 同一依赖链上的步骤 → 串行（链间可并行）
func (po *ParallelOrchestrator) DecomposeTask(ctx context.Context, task string, history []Message) (*DecomposeResult, error) {
	// 用现有 Planner 做规划分解
	planner := &Planner{Provider: po.Parent.Provider}
	plan, err := planner.Plan(ctx, task, history)
	if err != nil {
		return nil, fmt.Errorf("任务分解失败: %w", err)
	}

	if len(plan.Steps) == 0 {
		return &DecomposeResult{Reasoning: plan.Reasoning}, nil
	}

	// 把 PlanStep 转为 SubTask
	subTasks := make([]SubTask, len(plan.Steps))
	stepToAgent := map[string]string{} // step description → agent name

	for i, step := range plan.Steps {
		// 决定使用哪个子 Agent：从描述中匹配关键字
		agentName := selectAgentForTask(po.Tree, step.Description)
		stepToAgent[step.Description] = agentName

		subTasks[i] = SubTask{
			ID:           step.ID,
			Description:  step.Description,
			AgentName:    agentName,
			Input:        step.Description,
			Dependencies: step.Dependencies,
			IsContextual: !step.IsDestructive, // 非破坏性操作视为上下文收集
		}
	}

	// 构建并行组
	parallelGroups := po.buildParallelGroups(subTasks)

	return &DecomposeResult{
		Reasoning:      plan.Reasoning,
		SubTasks:       subTasks,
		ParallelGroups: parallelGroups,
	}, nil
}

// ExecuteSubTasks 并发执行一组子任务（同一组的互不依赖，可并行）。
// 每组的执行结果会注入共享上下文池，后续组可复用。
//
// 执行流：
//  1. 为每个子任务创建子 Loop（共享 po.ContextPool）
//  2. 注入共享上下文（已有知识/文件缓存）到子任务的 Input 前缀
//  3. 并行启动所有无依赖的子任务
//  4. 等待全部完成
//  5. 收集结果、检测冲突
//  6. 返回汇总
func (po *ParallelOrchestrator) ExecuteSubTasks(ctx context.Context, subTasks []SubTask) ([]SubTaskResult, error) {
	if len(subTasks) == 0 {
		return nil, nil
	}

	// 按依赖关系分批执行（每批内可并行）
	// 用拓扑排序分批
	remaining := map[int]bool{}
	taskByID := map[string]int{}
	for i := range subTasks {
		remaining[i] = true
		taskByID[subTasks[i].ID] = i
	}

	var batches []taskGroup
	for len(remaining) > 0 {
		// 找本批 ready 的任务（所有依赖已在之前批次完成）
		completed := map[string]bool{}
		for _, batch := range batches {
			for _, idx := range batch.indices {
				completed[subTasks[idx].ID] = true
			}
		}

		var ready []int
		for i := range remaining {
			task := subTasks[i]
			allDepsMet := true
			for _, dep := range task.Dependencies {
				if !completed[dep] {
					allDepsMet = false
					break
				}
			}
			if allDepsMet {
				ready = append(ready, i)
			}
		}

		if len(ready) == 0 {
			// 死锁或循环依赖——强制执行剩余任务中依赖最少的
			minDeps := -1
			for i := range remaining {
				d := len(subTasks[i].Dependencies)
				if minDeps < 0 || d < minDeps {
					minDeps = d
				}
			}
			for i := range remaining {
				if len(subTasks[i].Dependencies) == minDeps {
					ready = append(ready, i)
				}
			}
		}

		group := taskGroup{indices: ready, deps: map[string]bool{}}
		for _, i := range ready {
			delete(remaining, i)
			for _, dep := range subTasks[i].Dependencies {
				group.deps[dep] = true
			}
		}
		batches = append(batches, group)
	}

	// 按批执行（每批内并行）
	var allResults []SubTaskResult
	for batchIdx, batch := range batches {
		select {
		case <-ctx.Done():
			return allResults, ctx.Err()
		default:
		}

		// 为每批注入到目前累积的上下文
		if batchIdx > 0 {
			injectCtx := po.ContextPool.BuildContextInject("")
			if injectCtx != "" {
				// 把共享上下文注入给所有待执行的子 Agent
				for _, idx := range batch.indices {
					subTasks[idx].Input = injectCtx + "\n\n---\n\n# 当前子任务\n" + subTasks[idx].Input
				}
			}
		}

		// 并发执行本批任务
		results := po.executeBatch(ctx, subTasks, batch)
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// executeBatch 并发执行一批子任务（互不依赖）。
func (po *ParallelOrchestrator) executeBatch(ctx context.Context, subTasks []SubTask, batch taskGroup) []SubTaskResult {
	var wg sync.WaitGroup
	resultCh := make(chan SubTaskResult, len(batch.indices))

	for _, idx := range batch.indices {
		wg.Add(1)
		task := subTasks[idx]
		go func(t SubTask) {
			defer wg.Done()
			result := po.runSingleSubAgent(ctx, t)
			resultCh <- result
		}(task)
	}

	wg.Wait()
	close(resultCh)

	var results []SubTaskResult
	for r := range resultCh {
		results = append(results, r)
		// 执行结果注入共享上下文
		po.injectResultToContext(r)
	}

	return results
}

// runSingleSubAgent 运行单个子 Agent（在一个 goroutine 中）。
func (po *ParallelOrchestrator) runSingleSubAgent(ctx context.Context, task SubTask) SubTaskResult {
	sa := po.Tree.Find(task.AgentName)
	if sa == nil {
		return SubTaskResult{
			TaskID:    task.ID,
			AgentName: task.AgentName,
			Error:     fmt.Sprintf("未找到子Agent %q", task.AgentName),
		}
	}

	// 子 Registry：白名单裁剪或父副本
	var childReg *Registry
	if len(sa.Tools) > 0 {
		childReg = po.Parent.Registry.Subset(sa.Tools)
	} else {
		childReg = po.Parent.Registry.Copy()
	}

	// 注册并行上下文工具 + finish_task
	registerParallelContextTools(childReg, po.ContextPool, task.AgentName)
	childReg.Register(&Tool{
		Name:        "finish_task",
		Description: "任务完成信号：调用后子 Agent 退出循环，result 作为任务结果返回。",
		Parameters:  objSchema(props{"result": strProp("任务结果摘要")}, "result"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return argStr(args, "result"), nil
		},
	})

	maxIter := sa.MaxIter
	if maxIter <= 0 {
		maxIter = po.Parent.MaxIterations
	}

	child := &Loop{
		Provider:      po.Parent.Provider,
		Registry:      childReg,
		System:        "", // 不插入 system，保前缀一致
		MaxIterations: maxIter,
		OnEvent:       SubAgentSink(po.Parent.OnEvent, task.AgentName),
		State:         po.Parent.State,
		AgentTree:     po.Tree,
		Autonomous:    po.Parent.Autonomous,
	}

	// 子 task：子 system 作追加 instruction
	childTask := task.Input
	if sa.System != "" {
		childTask = "# 子 Agent 指令（" + sa.Name + "）\n" + sa.System + "\n\n---\n\n# 任务\n" + task.Input
	}

	// 父历史剥离末尾未配对 assistant tool_call（=并行调用本身）
	history := po.Parent.currentMsgs
	if len(history) > 0 && history[len(history)-1].Role == RoleAssistant && len(history[len(history)-1].ToolCalls) > 0 {
		history = history[:len(history)-1]
	}

	childMsgs, err := child.Run(ctx, childTask, history)

	result := SubTaskResult{
		TaskID:    task.ID,
		AgentName: task.AgentName,
	}

	if err != nil && !strings.Contains(err.Error(), "已达最大迭代数") {
		result.Error = err.Error()
	}
	if child.finishResult != nil {
		result.Output = *child.finishResult
	} else if r := lastAssistantContent(childMsgs); r != "" {
		result.Output = r
	}
	if result.Output == "" && result.Error == "" {
		result.Output = "(子 Agent 未产出结果)"
	}

	result.FinishedMsg = childMsgs

	return result
}

// injectResultToContext 把子 Agent 的执行结果注入共享上下文池。
func (po *ParallelOrchestrator) injectResultToContext(result SubTaskResult) {
	if result.Error != "" {
		return
	}
	if result.Output == "" {
		return
	}

	// 提取结果摘要作为知识
	summary := result.Output
	if len([]rune(summary)) > 200 {
		summary = string([]rune(summary)[:200]) + "..."
	}
	po.ContextPool.AddKnowledge(result.AgentName, "task_"+result.TaskID, result.Output, summary)
}

// buildParallelGroups 构建并行组索引（供 DecomposeTask 返回）。
// 每组的任务互不依赖，可并行执行。
func (po *ParallelOrchestrator) buildParallelGroups(subTasks []SubTask) [][]int {
	// 建立 taskID → index 映射
	idToIdx := map[string]int{}
	for i, t := range subTasks {
		idToIdx[t.ID] = i
	}

	// 每个任务分配一个组号（同组可并行）
	groupOf := make([]int, len(subTasks))
	maxGroup := 0
	for i, t := range subTasks {
		if len(t.Dependencies) == 0 {
			groupOf[i] = 0 // 无依赖→第一组
		} else {
			// 取所有依赖中最大的组号 + 1
			maxDepGroup := 0
			for _, dep := range t.Dependencies {
				if idx, ok := idToIdx[dep]; ok {
					if groupOf[idx]+1 > maxDepGroup {
						maxDepGroup = groupOf[idx] + 1
					}
				}
			}
			groupOf[i] = maxDepGroup
			if maxDepGroup > maxGroup {
				maxGroup = maxDepGroup
			}
		}
	}

	// 按组号收集
	groups := make([][]int, maxGroup+1)
	for i, g := range groupOf {
		groups[g] = append(groups[g], i)
	}
	return groups
}

// AggregateResults 汇总所有子任务执行结果。
// 策略：
//  1. 合并所有子任务的输出（按任务 ID 排序）
//  2. 检测共享变量冲突并标记
//  3. 去重知识条目
//  4. 生成汇总报告
func (po *ParallelOrchestrator) AggregateResults(results []SubTaskResult) string {
	if len(results) == 0 {
		return "（无子任务执行）"
	}

	// 按 TaskID 排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].TaskID != results[j].TaskID {
			return results[i].TaskID < results[j].TaskID
		}
		return results[i].AgentName < results[j].AgentName
	})

	var b strings.Builder
	b.WriteString("# 并行任务执行汇总\n\n")

	// 各子任务结果
	b.WriteString("## 子任务执行结果\n\n")
	for _, r := range results {
		b.WriteString(fmt.Sprintf("### %s（%s）\n", r.TaskID, r.AgentName))
		if r.Error != "" {
			b.WriteString(fmt.Sprintf("⚠️ 执行出错：%s\n\n", r.Error))
		} else {
			b.WriteString(r.Output + "\n\n")
		}
	}

	// 冲突检测
	conflicts := po.ContextPool.DetectConflicts()
	if len(conflicts) > 0 {
		b.WriteString("## ⚠️ 共享变量冲突\n\n")
		b.WriteString("以下变量被多个子 Agent 设置不同值，需人工裁决：\n\n")
		for _, c := range conflicts {
			b.WriteString(fmt.Sprintf("**%s**：\n", c.Name))
			for _, v := range c.Versions {
				b.WriteString(fmt.Sprintf("  - %s → %q（%s）\n", v.AgentName, v.Value, v.Timestamp.Format("15:04:05")))
			}
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## ✅ 无冲突\n\n")
		b.WriteString("所有共享变量一致，无冲突。\n\n")
	}

	// 知识摘要
	knowledge := po.ContextPool.GetAllKnowledge()
	if len(knowledge) > 0 {
		b.WriteString("## 累积知识\n\n")
		for _, k := range knowledge {
			b.WriteString(fmt.Sprintf("- **%s**（%s）: %s\n", k.Key, k.AgentName, k.Summary))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// ── 辅助 ───────────────────────────────────────────────────

// selectAgentForTask 根据任务描述匹配合适的子 Agent。
// 按关键词匹配 AgentTree 中所有子 Agent 的 Description，选择最相关的。
// 无匹配时返回空字符串（=由主 Agent 自己执行）。
func selectAgentForTask(tree *AgentTree, description string) string {
	if tree == nil || tree.Root == nil {
		return ""
	}

	// 确定性顺序：收集所有非 root agent 名并按名字排序
	names := make([]string, 0, len(tree.Children))
	for name := range tree.Children {
		if name != tree.Root.Name {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	bestName := ""
	bestScore := 0

	for _, name := range names {
		sa := tree.Children[name]
		score := agentTaskScore(sa.Description, description)
		if score > bestScore {
			bestScore = score
			bestName = name
		}
	}
	return bestName
}

// agentTaskScore 计算 agent 描述与任务描述的相关性得分。
// 策略：先试精确子串匹配（高权重），再试字符重叠（低权重），
// 最后按关键词长度归一化以防短词泛滥。
func agentTaskScore(agentDesc, taskDesc string) int {
	keywords := splitKeywords(agentDesc)
	score := 0
	for _, kw := range keywords {
		if len(kw) < 2 {
			continue
		}
		// 精确子串匹配（中文词组完全出现）→ 高权重
		if strings.Contains(taskDesc, kw) {
			score += len(kw) * 10
			continue
		}
		// 字符级重叠匹配
		overlap := countRuneOverlap(kw, taskDesc)
		if overlap >= 2 {
			// 用 overlap² 增加区分度，同时扣除不匹配字符作为惩罚
			miss := len([]rune(kw)) - overlap
			net := overlap*overlap - miss
			if net > 0 {
				score += net
			}
		}
	}
	return score
}

// countRuneOverlap 统计 s 中的字符在 t 中出现的次数（仅中文/字母/数字，剔除标点空格）。
func countRuneOverlap(s, t string) int {
	tRunes := make(map[rune]int) // char → 在 t 中的出现次数
	for _, r := range t {
		if isMeaningfulRune(r) {
			tRunes[r]++
		}
	}
	count := 0
	for _, r := range s {
		if isMeaningfulRune(r) && tRunes[r] > 0 {
			count++
			tRunes[r]-- // 不重复计
		}
	}
	return count
}

// isMeaningfulRune 是否有意义的字符（中文、字母、数字，排除标点空格）。
func isMeaningfulRune(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	if r >= 0x4E00 && r <= 0x9FFF { // 常用汉字
		return true
	}
	return false
}

// splitKeywords 将描述文本拆分为关键词列表。
// 按空白、标点符号、中英文常见分隔符拆分，并过滤空串。
func splitKeywords(s string) []string {
	// 将常见分隔符替换为空格（只含 ASCII 和标准 Unicode 字面量）
	s = strings.NewReplacer(
		"/", " ", "\\", " ", "|", " ", "-", " ", "_", " ",
		",", " ", ";", " ", "(", " ", ")", " ",
		".", " ", ":", " ",
	).Replace(s)
	fields := strings.Fields(s)
	return fields
}

// registerParallelContextTools 注册并行上下文工具到子 Registry。
// 子 Agent 通过这些工具访问共享上下文池，无需重复收集数据。
func registerParallelContextTools(reg *Registry, pool *SharedContext, agentName string) {
	// ctx_read_file：读共享文件缓存（不触发磁盘 I/O）
	reg.Register(&Tool{
		Name:        "ctx_read_file",
		Description: "从共享上下文读取已缓存的文件内容。仅返回已由其他子 Agent 读取过的文件，不会触发磁盘读。如果文件未缓存，返回未找到提示（请用 read_file 读取后自动缓存）。",
		Parameters:  objSchema(props{"path": strProp("文件路径")}, "path"),
		ReadOnly:    true,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			path := argStr(args, "path")
			content, ok := pool.LoadFile(path)
			if !ok {
				return fmt.Sprintf("文件 %s 未在缓存中，请用 read_file 读取（读取后自动加入共享缓存）", path), nil
			}
			return content, nil
		},
	})

	// ctx_get_knowledge：获取已有知识条目
	reg.Register(&Tool{
		Name:        "ctx_get_knowledge",
		Description: "从共享上下文获取指定 Key 的知识条目（其他子 Agent 已发现的信息）。返回知识条目的值、摘要和发现者。",
		Parameters:  objSchema(props{"key": strProp("知识条目 Key")}, "key"),
		ReadOnly:    true,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			key := argStr(args, "key")
			entry, ok := pool.GetKnowledge(key)
			if !ok {
				return fmt.Sprintf("未找到 key=%q 的知识条目", key), nil
			}
			return fmt.Sprintf("Key: %s\n发现者: %s\n摘要: %s\n完整值:\n%s", entry.Key, entry.AgentName, entry.Summary, entry.Value), nil
		},
	})

	// ctx_add_knowledge：添加新知识
	reg.Register(&Tool{
		Name:        "ctx_add_knowledge",
		Description: "向共享上下文添加一条知识条目（覆盖同名 Key）。用此工具向其他子 Agent 共享你发现的关键信息，避免他们重复调查。",
		Parameters:  objSchema(props{"key": strProp("知识条目标识 Key（简短英文，如 arch、db_config）"), "value": strProp("知识的完整内容"), "summary": strProp("一句话摘要（供其他子 Agent 快速了解）")}, "key", "value", "summary"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			key := argStr(args, "key")
			value := argStr(args, "value")
			summary := argStr(args, "summary")
			pool.AddKnowledge(agentName, key, value, summary)
			return fmt.Sprintf("已添加知识：%s = %s", key, summary), nil
		},
	})

	// ctx_get_var：读共享变量
	reg.Register(&Tool{
		Name:        "ctx_get_var",
		Description: "从共享上下文读取指定变量的当前值（其他子 Agent 已设置的）。如果未设置则返回未找到。",
		Parameters:  objSchema(props{"name": strProp("变量名")}, "name"),
		ReadOnly:    true,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			value, ok := pool.GetVar(name)
			if !ok {
				return fmt.Sprintf("变量 %q 未设置", name), nil
			}
			return fmt.Sprintf("%s = %s", name, value), nil
		},
	})

	// ctx_set_var：设置共享变量
	reg.Register(&Tool{
		Name:        "ctx_set_var",
		Description: "设置共享变量的当前值。其他子 Agent 可通过 ctx_get_var 读取。注意：多个子 Agent 设置同一变量的不同值会被检测为冲突。",
		Parameters:  objSchema(props{"name": strProp("变量名（简短英文，如 db_host、port）"), "value": strProp("变量值")}, "name", "value"),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			value := argStr(args, "value")
			pool.SetVar(agentName, name, value)
			return fmt.Sprintf("已设置共享变量 %s = %s", name, value), nil
		},
	})

	// ctx_knowledge_summary：获取共享上下文的文本摘要
	reg.Register(&Tool{
		Name:        "ctx_knowledge_summary",
		Description: "获取共享上下文的文本摘要：已发现的关键信息、已缓存的文件列表、已设置的共享变量。用于了解其他子 Agent 已完成的工作。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			summary := pool.KnowledgeSummary()
			if summary == "" {
				return "共享上下文为空（尚无其他子 Agent 完成工作）", nil
			}
			return summary, nil
		},
	})
}

// sortKeys 返回 map 的 key 排序列表。
func sortKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
