// 项目级 Token / 上下文 / 消耗统计 —— 持久化存在安装目录，独立于对话记录。
// 这样即使用户清空对话历史，统计数据也不会丢失，可长期追踪整个项目的 LLM 使用情况。
//
//go:build windows

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProjectStats 项目级别的累计统计，持久化在 InstallDir()/.pair/stats.json。
// 所有 LLM 调用的用量都会累加到这里，不受对话记录影响。
type ProjectStats struct {
	mu sync.Mutex

	// 累计 Token 用量（所有会话、所有 LLM 调用的总和）
	TotalPromptTokens     int `json:"totalPromptTokens"`
	TotalCompletionTokens int `json:"totalCompletionTokens"`
	TotalTokens           int `json:"totalTokens"`

	// 缓存命中 / 未命中（API 返回的真实值）
	TotalCacheHitTokens  int `json:"totalCacheHitTokens"`
	TotalCacheMissTokens int `json:"totalCacheMissTokens"`

	// 上下文统计
	TotalContextUsed  int `json:"totalContextUsed"`  // 累计 prompt token（所有 LLM 调用 prompt 之和）
	ContextWindowSize int `json:"contextWindowSize"` // 上下文窗口上限（来自设置，供参考）

	// 推理 token（思考链）
	TotalReasoningTokens int `json:"totalReasoningTokens"`

	// 对话/轮次统计
	TotalTurns     int `json:"totalTurns"`     // 总轮次（用户消息数）
	TotalToolCalls int `json:"totalToolCalls"` // 总工具调用次数

	// LLM 调用次数累计
	TotalLLMCalls int `json:"totalLlmCalls"` // 总 LLM API 调用次数

	// 按模型的细分统计（模型名 → 该模型的用量）
	PerModel map[string]*ModelStats `json:"perModel,omitempty"`

	// 按日期的细分统计（YYYY-MM-DD → 该日用量）
	PerDay map[string]*DayStats `json:"perDay,omitempty"`

	// 累积估算费用（分币种）
	TotalCost     float64 `json:"totalCost"`
	CostCurrency  string  `json:"costCurrency"` // 币种符号，如 ¥、$

	// 首次记录时间 / 最近更新
	FirstRecord time.Time `json:"firstRecord"`
	LastUpdate  time.Time `json:"lastUpdate"`

	// 持久化文件路径
	filePath string
}

// ModelStats 按模型的细分统计。
type ModelStats struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	LLMCalls         int `json:"llmCalls"`
	CacheHitTokens   int `json:"cacheHitTokens"`
	CacheMissTokens  int `json:"cacheMissTokens"`
	Cost             float64 `json:"cost"`
}

// DayStats 按日期的细分统计。
type DayStats struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	LLMCalls         int `json:"llmCalls"`
	Turns            int `json:"turns"`
	ToolCalls        int `json:"toolCalls"`
	Cost             float64 `json:"cost"`
}

// statsFilePath 返回 stats.json 的完整路径。
func statsFilePath() string {
	return filepath.Join(InstallDir(), ".pair", "stats.json")
}

// globalStats 包级单例。
var globalStats *ProjectStats

// GetProjectStats 返回项目级统计实例（懒加载）。
func GetProjectStats() *ProjectStats {
	if globalStats == nil {
		globalStats = loadProjectStats()
	}
	return globalStats
}

// loadProjectStats 从磁盘加载，不存在则返回零值。
func loadProjectStats() *ProjectStats {
	ps := &ProjectStats{
		PerModel:       make(map[string]*ModelStats),
		PerDay:         make(map[string]*DayStats),
		CostCurrency:   "¥",
		ContextWindowSize: 1000000,
	}
	ps.filePath = statsFilePath()

	data, err := os.ReadFile(ps.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ps.FirstRecord = time.Now()
			ps.LastUpdate = time.Now()
		}
		return ps
	}

	if err := json.Unmarshal(data, ps); err != nil {
		// 损坏或旧格式 → 重新开始
		ps.FirstRecord = time.Now()
		ps.LastUpdate = time.Now()
		if ps.PerModel == nil {
			ps.PerModel = make(map[string]*ModelStats)
		}
		if ps.PerDay == nil {
			ps.PerDay = make(map[string]*DayStats)
		}
	}
	if ps.PerModel == nil {
		ps.PerModel = make(map[string]*ModelStats)
	}
	if ps.PerDay == nil {
		ps.PerDay = make(map[string]*DayStats)
	}
	return ps
}

// Save 将统计数据持久化到磁盘。
func (ps *ProjectStats) Save() {
	ps.mu.Lock()
	ps.LastUpdate = time.Now()
	filePath := ps.filePath
	ps.mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(filePath), 0755)
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filePath, data, 0644)
}

// RecordLLMCall 记录一次 LLM 调用（供桥接层在收到 usage 事件时调用）。
//   - promptTokens: API 返回的输入 token 数
//   - completionTokens: API 返回的输出 token 数
//   - cacheHitTokens: 缓存命中 token（0 表示无缓存）
//   - cacheMissTokens: 缓存未命中 token
//   - reasoningTokens: 推理 token 数（DeepSeek reasoning）
//   - modelName: 模型名（如 "deepseek-v4-flash"），用于按模型细分
//   - cost: 本次调用的估算费用
//   - currency: 币种符号
func RecordLLMCall(promptTokens, completionTokens, cacheHitTokens, cacheMissTokens, reasoningTokens int, modelName string, cost float64, currency string) {
	ps := GetProjectStats()
	ps.mu.Lock()

	ps.TotalLLMCalls++
	ps.TotalPromptTokens += promptTokens
	ps.TotalCompletionTokens += completionTokens
	ps.TotalTokens = ps.TotalPromptTokens + ps.TotalCompletionTokens
	ps.TotalContextUsed += promptTokens
	ps.TotalCacheHitTokens += cacheHitTokens
	ps.TotalCacheMissTokens += cacheMissTokens
	ps.TotalReasoningTokens += reasoningTokens
	ps.TotalCost += cost
	if currency != "" {
		ps.CostCurrency = currency
	}

	// 按模型记录
	ms, ok := ps.PerModel[modelName]
	if !ok {
		ms = &ModelStats{}
		ps.PerModel[modelName] = ms
	}
	ms.LLMCalls++
	ms.PromptTokens += promptTokens
	ms.CompletionTokens += completionTokens
	ms.TotalTokens += promptTokens + completionTokens
	ms.CacheHitTokens += cacheHitTokens
	ms.CacheMissTokens += cacheMissTokens
	ms.Cost += cost

	// 按日期记录
	today := time.Now().Format("2006-01-02")
	ds, ok := ps.PerDay[today]
	if !ok {
		ds = &DayStats{}
		ps.PerDay[today] = ds
	}
	ds.LLMCalls++
	ds.PromptTokens += promptTokens
	ds.CompletionTokens += completionTokens
	ds.TotalTokens += promptTokens + completionTokens
	ds.Cost += cost

	ps.mu.Unlock()
	ps.Save()
}

// RecordTurn 记录一次用户轮次（含工具调用数）。
func RecordTurn(toolCalls int) {
	ps := GetProjectStats()
	ps.mu.Lock()
	ps.TotalTurns++
	ps.TotalToolCalls += toolCalls

	today := time.Now().Format("2006-01-02")
	ds, ok := ps.PerDay[today]
	if !ok {
		ds = &DayStats{}
		ps.PerDay[today] = ds
	}
	ds.Turns++
	ds.ToolCalls += toolCalls
	ps.mu.Unlock()
	ps.Save()
}

// GetProjectStatsSnapshot 返回一份线程安全的数据快照（供 UI 展示用）。
func GetProjectStatsSnapshot() *ProjectStats {
	ps := GetProjectStats()
	ps.mu.Lock()
	snapshot := &ProjectStats{
		TotalPromptTokens:     ps.TotalPromptTokens,
		TotalCompletionTokens: ps.TotalCompletionTokens,
		TotalTokens:           ps.TotalTokens,
		TotalCacheHitTokens:   ps.TotalCacheHitTokens,
		TotalCacheMissTokens:  ps.TotalCacheMissTokens,
		TotalContextUsed:      ps.TotalContextUsed,
		ContextWindowSize:     ps.ContextWindowSize,
		TotalReasoningTokens:  ps.TotalReasoningTokens,
		TotalTurns:            ps.TotalTurns,
		TotalToolCalls:        ps.TotalToolCalls,
		TotalLLMCalls:         ps.TotalLLMCalls,
		TotalCost:             ps.TotalCost,
		CostCurrency:          ps.CostCurrency,
		FirstRecord:           ps.FirstRecord,
		LastUpdate:            ps.LastUpdate,
	}
	// 深拷贝 map
	if ps.PerModel != nil {
		snapshot.PerModel = make(map[string]*ModelStats, len(ps.PerModel))
		for k, v := range ps.PerModel {
			cp := *v
			snapshot.PerModel[k] = &cp
		}
	}
	if ps.PerDay != nil {
		snapshot.PerDay = make(map[string]*DayStats, len(ps.PerDay))
		for k, v := range ps.PerDay {
			cp := *v
			snapshot.PerDay[k] = &cp
		}
	}
	ps.mu.Unlock()
	return snapshot
}

// ResetProjectStats 重置所有统计（慎用）。
func ResetProjectStats() {
	ps := GetProjectStats()
	ps.mu.Lock()
	ps.TotalPromptTokens = 0
	ps.TotalCompletionTokens = 0
	ps.TotalTokens = 0
	ps.TotalCacheHitTokens = 0
	ps.TotalCacheMissTokens = 0
	ps.TotalContextUsed = 0
	ps.TotalReasoningTokens = 0
	ps.TotalTurns = 0
	ps.TotalToolCalls = 0
	ps.TotalLLMCalls = 0
	ps.TotalCost = 0
	ps.PerModel = make(map[string]*ModelStats)
	ps.PerDay = make(map[string]*DayStats)
	ps.FirstRecord = time.Now()
	ps.LastUpdate = time.Now()
	ps.mu.Unlock()
	ps.Save()
}
