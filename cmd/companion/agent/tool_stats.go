// 工具调用统计 —— 记录每次工具调用的结果，按工具名/来源聚合成功率。
// Agent 通过 tool_stats 工具查看统计，为自我迭代提供数据基础。
// 统计在 Loop 运行时自动记录，全局单例供桥接层访问。

package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ToolSource 工具来源类型。
type ToolSource string

const (
	ToolSourceBuiltin ToolSource = "builtin" // 内置 Go 工具
	ToolSourceLua     ToolSource = "lua"     // Lua 自定义工具
	ToolSourceMCP     ToolSource = "mcp"     // MCP 外部工具
)

// ToolCallRecord 一次工具调用的完整记录。
type ToolCallRecord struct {
	Name      string        `json:"name"`      // 工具名
	Source    ToolSource    `json:"source"`    // 来源
	Success   bool          `json:"success"`   // 是否成功
	Duration  time.Duration `json:"duration"`  // 耗时
	Timestamp time.Time     `json:"timestamp"` // 调用时间
}

// ToolStatsSummary 按工具聚合的统计摘要。
type ToolStatsSummary struct {
	Name     string `json:"name"`     // 工具名
	Source   string `json:"source"`   // 来源
	Calls    int    `json:"calls"`    // 总调用次数
	Success  int    `json:"success"`  // 成功次数
	Failures int    `json:"failures"` // 失败次数
	Rate     string `json:"rate"`     // 成功率（百分比字符串）
}

// ToolStatsRecorder 工具调用统计记录器。
// 嵌入 Loop，每次工具调用后自动记录。
// 线程安全，轮询方式查询。
type ToolStatsRecorder struct {
	mu      sync.RWMutex
	records []ToolCallRecord
	maxLen  int // 最大记录数（循环Buffer），默认2000
	pos     int // 循环写入位置
	full    bool
}

// NewToolStatsRecorder 创建统计记录器。
// maxRecords: 最大保留记录数（超过后循环覆盖旧记录），0=默认2000。
func NewToolStatsRecorder(maxRecords int) *ToolStatsRecorder {
	if maxRecords <= 0 {
		maxRecords = 2000
	}
	return &ToolStatsRecorder{
		records: make([]ToolCallRecord, maxRecords),
		maxLen:  maxRecords,
	}
}

// Record 记录一次工具调用结果。
func (r *ToolStatsRecorder) Record(name string, source ToolSource, success bool, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.records[r.pos] = ToolCallRecord{
		Name:      name,
		Source:    source,
		Success:   success,
		Duration:  duration,
		Timestamp: time.Now(),
	}
	r.pos++
	if r.pos >= r.maxLen {
		r.pos = 0
		r.full = true
	}
}

// Summary 返回按工具名聚合的统计摘要。
// minCalls: 最少调用次数过滤（0=全部显示）。
func (r *ToolStatsRecorder) Summary(minCalls int) []ToolStatsSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agg := map[string]*struct {
		source   string
		calls    int
		success  int
		failures int
	}{}

	count := r.maxLen
	if !r.full {
		count = r.pos
	}
	for i := 0; i < count; i++ {
		rec := r.records[i]
		if rec.Name == "" {
			continue // 空槽位（循环buffer未填满）
		}
		key := rec.Name + "|" + string(rec.Source)
		s, ok := agg[key]
		if !ok {
			agg[key] = &struct {
				source   string
				calls    int
				success  int
				failures int
			}{source: string(rec.Source)}
			s = agg[key]
		}
		s.calls++
		if rec.Success {
			s.success++
		} else {
			s.failures++
		}
	}

	result := make([]ToolStatsSummary, 0, len(agg))
	for key, s := range agg {
		name := strings.SplitN(key, "|", 2)[0]
		rate := 0.0
		if s.calls > 0 {
			rate = float64(s.success) / float64(s.calls) * 100
		}
		if minCalls > 0 && s.calls < minCalls {
			continue
		}
		result = append(result, ToolStatsSummary{
			Name:     name,
			Source:   s.source,
			Calls:    s.calls,
			Success:  s.success,
			Failures: s.failures,
			Rate:     fmt.Sprintf("%.1f%%", rate),
		})
	}

	// 按调用次数降序排列
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Calls > result[i].Calls {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// TotalCalls 返回总调用次数。
func (r *ToolStatsRecorder) TotalCalls() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.full {
		return r.maxLen
	}
	return r.pos
}

// RecentCalls 返回最近 N 条记录。
func (r *ToolStatsRecorder) RecentCalls(n int) []ToolCallRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n <= 0 || n > r.maxLen {
		n = r.maxLen
	}

	count := r.maxLen
	if !r.full {
		count = r.pos
	}
	if n > count {
		n = count
	}

	// 从最新（pos前一个）往前取n条
	result := make([]ToolCallRecord, 0, n)
	start := r.pos - 1
	if start < 0 {
		start = r.maxLen - 1
	}
	for i := 0; i < n; i++ {
		idx := start - i
		for idx < 0 {
			idx += r.maxLen
		}
		rec := r.records[idx]
		if rec.Name == "" {
			continue
		}
		result = append(result, rec)
	}
	return result
}

// Reset 清空所有记录。
func (r *ToolStatsRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = make([]ToolCallRecord, r.maxLen)
	r.pos = 0
	r.full = false
}

// ─── 全局单例 ─────────────────────────────────────────────

var (
	globalToolStats   *ToolStatsRecorder
	globalToolStatsMu sync.Mutex
)

// GetToolStats 获取全局工具统计实例。
func GetToolStats() *ToolStatsRecorder {
	globalToolStatsMu.Lock()
	defer globalToolStatsMu.Unlock()
	return globalToolStats
}

// SetToolStats 设置全局工具统计实例（bridge 初始化时调用）。
func SetToolStats(ts *ToolStatsRecorder) {
	globalToolStatsMu.Lock()
	defer globalToolStatsMu.Unlock()
	globalToolStats = ts
}

// ─── tool_stats 工具 ─────────────────────────────────────

// inferToolSource 根据工具名推断来源。
func inferToolSource(name string) ToolSource {
	if strings.HasPrefix(name, "mcp__") {
		return ToolSourceMCP
	}
	// Lua 工具名不带前缀，但可以通过检查 Registry 中的 Tool 对象来确定
	// 这里由调用方明确指定来源
	return ToolSourceBuiltin
}

// registerToolStatsTool 注册 tool_stats 工具。
func registerToolStatsTool(r *Registry) {
	r.Register(&Tool{
		Name: "tool_stats",
		Description: "查看工具调用统计（成功率、调用次数）。" +
			"按工具名聚合，显示每个工具的调用次数/成功数/失败数/成功率。" +
			"可使用 min_calls 过滤低频工具，recent 查看最近调用记录。" +
			"Agent 可用此数据识别高频失败工具，主动优化或创建新工具替代。",
		Parameters: objSchema(props{
			"min_calls": intProp("可选：最少调用次数过滤（默认0=全部显示）"),
			"recent":    intProp("可选：显示最近 N 条调用记录（不传则不显示）"),
			"source":    strProp("可选：按来源过滤，\"builtin\" | \"lua\" | \"mcp\"（不传=全部）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			ts := GetToolStats()
			if ts == nil {
				return "工具统计未启用（全局实例未配置）。", nil
			}

			minCalls := argInt(args, "min_calls", 0)
			recent := argInt(args, "recent", 0)
			sourceFilter := argStr(args, "source")

			summary := ts.Summary(minCalls)
			if len(summary) == 0 {
				return "暂无工具调用统计记录（Agent 尚未调用任何工具）。", nil
			}

			// 按来源过滤
			if sourceFilter != "" {
				filtered := make([]ToolStatsSummary, 0, len(summary))
				for _, s := range summary {
					if s.Source == sourceFilter {
						filtered = append(filtered, s)
					}
				}
				summary = filtered
			}

			var b strings.Builder
			totalCalls := ts.TotalCalls()
			fmt.Fprintf(&b, "## 工具调用统计（共 %d 次调用）\n\n", totalCalls)

			// 按来源分组统计
			type groupStat struct {
				calls, success int
			}
			bySource := map[string]*groupStat{}
			for _, s := range summary {
				gs, ok := bySource[s.Source]
				if !ok {
					gs = &groupStat{}
					bySource[s.Source] = gs
				}
				gs.calls += s.Calls
				gs.success += s.Success
			}
			b.WriteString("| 来源 | 调用次数 | 成功率 |\n")
			b.WriteString("|------|---------|--------|\n")
			for _, src := range []string{"builtin", "lua", "mcp"} {
				if gs, ok := bySource[src]; ok {
					rate := 0.0
					if gs.calls > 0 {
						rate = float64(gs.success) / float64(gs.calls) * 100
					}
					srcLabel := map[string]string{"builtin": "内置工具", "lua": "Lua 自定义", "mcp": "MCP 外部"}[src]
					fmt.Fprintf(&b, "| %s | %d | %.1f%% |\n", srcLabel, gs.calls, rate)
				}
			}
			b.WriteString("\n")

			// 按工具名明细
			b.WriteString("| 工具名 | 来源 | 调用 | 成功 | 失败 | 成功率 |\n")
			b.WriteString("|--------|------|------|------|------|--------|\n")
			for _, s := range summary {
				srcLabel := map[string]string{"builtin": "内置", "lua": "Lua", "mcp": "MCP"}[s.Source]
				if srcLabel == "" {
					srcLabel = s.Source
				}
				fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %d | %s |\n",
					s.Name, srcLabel, s.Calls, s.Success, s.Failures, s.Rate)
			}

			// 最近调用记录
			if recent > 0 {
				recents := ts.RecentCalls(recent)
				if len(recents) > 0 {
					b.WriteString("\n### 最近调用记录\n\n")
					b.WriteString("| 时间 | 工具名 | 来源 | 结果 | 耗时 |\n")
					b.WriteString("|------|--------|------|------|------|\n")
					for _, rec := range recents {
						status := "✅"
						if !rec.Success {
							status = "❌"
						}
						srcLabel := map[string]string{"builtin": "内置", "lua": "Lua", "mcp": "MCP"}[string(rec.Source)]
						if srcLabel == "" {
							srcLabel = string(rec.Source)
						}
						fmt.Fprintf(&b, "| %s | `%s` | %s | %s | %v |\n",
							rec.Timestamp.Format("15:04:05"), rec.Name, srcLabel, status, rec.Duration.Round(time.Millisecond))
					}
				}
			}

			b.WriteString("\n---\n")
			b.WriteString("💡 **提示**：如果发现某工具失败率高，Agent 可以：\n")
			b.WriteString("1. 分析失败原因，调整调用方式\n")
			b.WriteString("2. 用 `lua_tool_create` 创建新工具替代\n")
			b.WriteString("3. 用 `evolution_save_capsule` 保存修复经验\n")

			return b.String(), nil
		},
	})
}
