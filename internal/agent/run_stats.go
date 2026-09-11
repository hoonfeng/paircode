package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════
// run_stats.go — 运行统计（每会话一次运行的耗时/步数/工具调用/token 速度）
//
// 设计（★ 2026-09-12 后端统计改造）：
//   - 权威真源在**后端**：前端不再自行累加（前端累加口径受丢事件/刷新影响，
//     且 token 速度曾用「整段执行墙钟」当分母 → 严重低估）。
//   - 粒度：一次「运行」= 一条用户消息触发的完整执行（含段预算自动续跑的所有段，
//     由 SessionManager 的会话生命周期界定，见 session_manager.go）。
//   - token 速度分母 = Σ 每次 LLM 调用的**生成阶段耗时**（首 chunk → 末 chunk，
//     不含 prefill/排队/工具执行/审批等待）；无流式时间戳时退化为 Σ 调用总耗时。
//   - 持久化：<workspaceRoot>/.pair/run-stats.json（每会话保留最近一次运行），
//     运行中（EndAt==0）只在内存，不落盘——避免半程数据被当结果。
// ═══════════════════════════════════════════════════════════════

// RunStats 一次运行的统计（JSON 字段名面向前端，camelCase）。
type RunStats struct {
	ConvID string `json:"convId,omitempty"`
	// RunID 本次运行标识（进程内唯一，便于前端区分「同一会话的两次运行」）。
	RunID string `json:"runId,omitempty"`

	StartAt int64 `json:"startAt"` // 开始时间（Unix 毫秒）
	EndAt   int64 `json:"endAt"`   // 结束时间（0 = 运行中）
	// DurationMs 墙钟总耗时（含工具/审批/思考等待；仅供显示，不参与速度计算）。
	DurationMs int64 `json:"durationMs"`
	// Running 是否仍在运行（EndAt == 0）。
	Running bool `json:"running"`

	Steps     int   `json:"steps"`     // 步数（一次 LLM 调用 + 工具执行 = 一步）
	ToolCalls int   `json:"toolCalls"` // 工具调用次数
	ToolMs    int64 `json:"toolMs"`    // 工具执行累计耗时（毫秒）
	LLMCalls  int   `json:"llmCalls"`  // LLM 调用次数
	LLMMs     int64 `json:"llmMs"`     // LLM 调用累计耗时（请求往返，毫秒）
	// GenMs LLM 生成阶段累计耗时（首 chunk → 末 chunk，毫秒）——token 速度的分母。
	GenMs int64 `json:"genMs"`

	PromptTokens     int `json:"promptTokens"`     // 累计输入 token
	CompletionTokens int `json:"completionTokens"` // 累计输出 token
	// TokensPerSecond 输出速度（completionTokens / genMs）；GenMs 缺失时用 LLMMs 退化。
	TokensPerSecond float64 `json:"tokensPerSecond"`
}

var (
	runStatsByRoot   = make(map[string]map[string]*RunStats) // root → convID → stats（内存，含运行中）
	runStatsMu       sync.Mutex
	runStatsSeq      int64 // RunID 递增序号（进程内）
	runStatsMaxConvs = 200 // 单工作区最多保留的会话条数（超限按 endAt 淘汰最旧）
)

// runStatsPath 返回指定工作区根的 run-stats.json 路径。
func runStatsPath(root string) string {
	if root == "" {
		return ""
	}
	pairDir := filepath.Join(root, ".pair")
	os.MkdirAll(pairDir, 0755)
	return filepath.Join(pairDir, "run-stats.json")
}

// normalizeStatsRoot 规范化工作区根路径。
// ★ 2026-09-12 修复「统计写了却查不到」：root 直接当内存缓存 key 时，同一工作区的
//
//	不同写法（多余分隔符 `F:\\ws`、尾分隔符 `F:\ws\`、含 `\.`）会分裂成多个 root ——
//	写侧与读侧落到不同缓存条目、且缓存一旦以某写法建立（哪怕当时磁盘还没有数据）
//	就不再回读磁盘 → 查询恒返回零值。统一 Clean 后同一工作区只有一个 root。
func normalizeStatsRoot(root string) string {
	if root == "" {
		return ""
	}
	return filepath.Clean(root)
}

// loadRunStatsLocked 从磁盘恢复某工作区的运行统计到内存（调用方需持锁）。
// 磁盘内容为 {convID: RunStats}；只恢复已结束（EndAt>0）的条目——运行中的条目
// 属于上一进程，进程重启后已失效。
func loadRunStatsLocked(root string) map[string]*RunStats {
	if m, ok := runStatsByRoot[root]; ok {
		return m
	}
	m := make(map[string]*RunStats)
	if path := runStatsPath(root); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var disk map[string]*RunStats
			if json.Unmarshal(data, &disk) == nil {
				for id, s := range disk {
					if s == nil || s.EndAt == 0 {
						continue
					}
					s.ConvID = id
					s.Running = false
					m[id] = s
				}
			}
		}
	}
	runStatsByRoot[root] = m
	return m
}

// saveRunStatsLocked 原子写盘（调用方需持锁）：只写已结束条目，按 endAt 淘汰最旧。
func saveRunStatsLocked(root string) {
	m := runStatsByRoot[root]
	path := runStatsPath(root)
	if path == "" || m == nil {
		return
	}
	out := make(map[string]*RunStats, len(m))
	for id, s := range m {
		if s == nil || s.EndAt == 0 {
			continue // 运行中不落盘（半程数据不作为结果）
		}
		out[id] = s
	}
	if len(out) > runStatsMaxConvs {
		ids := make([]string, 0, len(out))
		for id := range out {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return out[ids[i]].EndAt < out[ids[j]].EndAt })
		for _, id := range ids[:len(out)-runStatsMaxConvs] {
			delete(out, id)
			delete(m, id)
		}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// BeginRunStatsFor 开始一次运行统计（按工作区 + 会话）：
// 清零并返回新的统计对象（同一会话的新运行覆盖上一次；上一次已结束的值仍可读盘）。
// Loop 通过 BeginRunStats 挂载返回值，在运行期间累加各字段。
func BeginRunStatsFor(root, convID string) *RunStats {
	if convID == "" {
		return nil
	}
	root = normalizeStatsRoot(root)
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	m := loadRunStatsLocked(root)
	runStatsSeq++
	s := &RunStats{
		ConvID:  convID,
		StartAt: time.Now().UnixMilli(),
		Running: true,
	}
	s.RunID = fmt.Sprintf("%s#%d-%d", convID, s.StartAt, runStatsSeq)
	m[convID] = s
	return s
}

// EndRunStatsFor 结束一次运行统计：定格 endAt、派生速度并落盘。
// 未开始的会话（无内存条目）无操作。
func EndRunStatsFor(root, convID string) *RunStats {
	if convID == "" {
		return nil
	}
	root = normalizeStatsRoot(root)
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	m := loadRunStatsLocked(root)
	s := m[convID]
	if s == nil || s.EndAt != 0 {
		return s
	}
	s.EndAt = time.Now().UnixMilli()
	deriveRunStats(s)
	s.Running = false
	saveRunStatsLocked(root)
	return s
}

// RunStatsFor 读一次运行的统计（内存优先——含运行中的实时值；否则读磁盘）。
// 不存在时返回零值对象（StartAt==0，前端据此判定「无数据」）。
func RunStatsFor(root, convID string) *RunStats {
	if convID == "" {
		return &RunStats{}
	}
	root = normalizeStatsRoot(root)
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	m := loadRunStatsLocked(root)
	if s, ok := m[convID]; ok && s != nil {
		deriveRunStats(s)
		out := *s
		return &out
	}
	return &RunStats{}
}

// ResetRunStatsFor 清空某会话的运行统计（内存 + 磁盘；新建/清空对话时调用）。
func ResetRunStatsFor(root, convID string) {
	if convID == "" {
		return
	}
	root = normalizeStatsRoot(root)
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	m := loadRunStatsLocked(root)
	delete(m, convID)
	saveRunStatsLocked(root)
}

// ─── Loop 侧累加（运行期间）──

// addLLMCall 记录一次 LLM 调用：用量 + 调用总耗时 + 生成阶段耗时。
// usage 可为 nil（provider 未回传用量时只计时）。
func (s *RunStats) addLLMCall(usage *Usage, llmMs, genMs int64) {
	if s == nil {
		return
	}
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	s.LLMCalls++
	if llmMs > 0 {
		s.LLMMs += llmMs
	}
	if genMs > 0 {
		s.GenMs += genMs
	}
	if usage != nil {
		s.PromptTokens += usage.PromptTokens
		s.CompletionTokens += usage.CompletionTokens
	}
}

// addToolCall 记录一次工具调用及其耗时。
func (s *RunStats) addToolCall(dur time.Duration) {
	if s == nil {
		return
	}
	runStatsMu.Lock()
	defer runStatsMu.Unlock()
	s.ToolCalls++
	if dur > 0 {
		s.ToolMs += dur.Milliseconds()
	}
}

// addStep 记一步（beginStep 调用；两条循环路径共用）。
func (s *RunStats) addStep() {
	if s == nil {
		return
	}
	runStatsMu.Lock()
	s.Steps++
	runStatsMu.Unlock()
}

// deriveRunStats 派生字段（耗时 / token 速度）。调用方需持锁。
func deriveRunStats(s *RunStats) {
	if s == nil {
		return
	}
	end := s.EndAt
	if end == 0 {
		end = time.Now().UnixMilli()
	}
	if s.StartAt > 0 && end >= s.StartAt {
		s.DurationMs = end - s.StartAt
	}
	denom := s.GenMs
	if denom <= 0 {
		denom = s.LLMMs // 退化：无流式时间戳（非流式 provider）时用总耗时
	}
	if denom > 0 && s.CompletionTokens > 0 {
		s.TokensPerSecond = float64(s.CompletionTokens) / (float64(denom) / 1000.0)
	} else {
		s.TokensPerSecond = 0
	}
}

// ─── Loop 接线（Loop 定义见 loop.go）──

// SetRunStats 挂载本次运行的统计累加器（SessionManager 在会话启动时调用一次）。
// Loop 在分段续跑中复用同一实例 → 累加跨段自然延续（一次运行 = 一条用户消息的完整执行）。
func (l *Loop) SetRunStats(s *RunStats) {
	if l == nil {
		return
	}
	l.stats = s
}

// ─── 全局主根便捷入口（UI/宿主层无会话上下文时使用）──

// CurrentRootForRunStats 取全局主根（根实时快照[0]——★ 2026-09-09 合并 core.Folders）。
func CurrentRootForRunStats() string {
	if roots := workspaceRootsSnapshot(); len(roots) > 0 {
		return roots[0]
	}
	return ""
}

// ReadRunStats 取全局主根下某会话的运行统计。
func ReadRunStats(convID string) *RunStats {
	return RunStatsFor(CurrentRootForRunStats(), convID)
}

// ResetRunStats 清空全局主根下某会话的运行统计。
func ResetRunStats(convID string) {
	ResetRunStatsFor(CurrentRootForRunStats(), convID)
}
