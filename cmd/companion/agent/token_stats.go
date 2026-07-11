package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// TokenStats 保存到磁盘的 token 统计（工作区级别，覆盖式保存最新值）。
type TokenStats struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	CacheHitTokens   int `json:"cacheHitTokens"`
	CacheMissTokens  int `json:"cacheMissTokens"`
	SystemTokens     int `json:"systemTokens"`
	SkillsTokens     int `json:"skillsTokens"`
	MCPTokens        int `json:"mcpTokens"`
	ToolTokens       int `json:"toolTokens"`
	HistoryTokens    int `json:"historyTokens"`
	OtherTokens      int `json:"otherTokens"`
}

var (
	latestTokenStats TokenStats
	tokenStatsMu     sync.Mutex
)

// tokenStatsPath 返回 .pair/token-stats.json 路径
func tokenStatsPath() string {
	if len(WorkspaceRoots) == 0 || WorkspaceRoots[0] == "" {
		return ""
	}
	pairDir := filepath.Join(WorkspaceRoots[0], ".pair")
	os.MkdirAll(pairDir, 0755)
	return filepath.Join(pairDir, "token-stats.json")
}

// SaveTokenUsage 保存最新一次 LLM 调用的 token 用量（含 PromptBreakdown）到磁盘。
// 这是 agent 自闭环行为：agent 自己管理自己的上下文统计，不依赖外部宿主。
//
// 每次 LLM 调用后，Loop.Run 内部自动调用此函数。
// 前端 / 外部宿主可通过 ReadTokenStats 读取已持久化的统计。
func SaveTokenUsage(usage *Usage) {
	if usage == nil {
		return
	}
	tokenStatsMu.Lock()
	defer tokenStatsMu.Unlock()

	latestTokenStats = TokenStats{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		CacheHitTokens:   usage.PromptCacheHitTokens,
		CacheMissTokens:  usage.PromptCacheMissTokens,
		SystemTokens:     usage.SystemTokens,
		SkillsTokens:     usage.SkillsTokens,
		MCPTokens:        usage.MCPTokens,
		ToolTokens:       usage.ToolTokens,
		HistoryTokens:    usage.HistoryTokens,
		OtherTokens:      usage.OtherTokens,
	}

	path := tokenStatsPath()
	if path == "" {
		return
	}
	data, _ := json.MarshalIndent(latestTokenStats, "", "  ")
	os.WriteFile(path, data, 0644)
}

// ReadTokenStats 从磁盘读取已持久化的 token 统计。
// 外部宿主（web 服务）通过此函数获取 agent 自闭环保存的统计数据。
func ReadTokenStats() *TokenStats {
	tokenStatsMu.Lock()
	defer tokenStatsMu.Unlock()

	path := tokenStatsPath()
	if path == "" {
		return &latestTokenStats
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &latestTokenStats
	}
	var stats TokenStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return &latestTokenStats
	}
	// 内存最新值优先于磁盘（当前进程内新产生的统计尚未刷盘）
	if latestTokenStats.TotalTokens > stats.TotalTokens {
		return &latestTokenStats
	}
	return &stats
}
