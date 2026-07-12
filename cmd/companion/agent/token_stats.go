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

// tokenStatsPath 返回指定工作区的 .pair/token-stats.json 路径
func tokenStatsPath(root string) string {
	if root == "" {
		return ""
	}
	pairDir := filepath.Join(root, ".pair")
	os.MkdirAll(pairDir, 0755)
	return filepath.Join(pairDir, "token-stats.json")
}

// SaveTokenUsageForRoot 按指定工作区根路径累积 token 用量到磁盘。
// root 为工作区根路径，不同工作区写入不同文件，实现多工作区隔离。
func SaveTokenUsageForRoot(root string, usage *Usage) {
	if usage == nil || root == "" {
		return
	}
	tokenStatsMu.Lock()
	defer tokenStatsMu.Unlock()

	// 先从磁盘读出已有累积值（防止跨进程丢失）
	if path := tokenStatsPath(root); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			var disk TokenStats
			if json.Unmarshal(data, &disk) == nil {
				if disk.TotalTokens > latestTokenStats.TotalTokens {
					latestTokenStats = disk
				}
			}
		}
	}

	// 叠加（累积）
	latestTokenStats.PromptTokens += usage.PromptTokens
	latestTokenStats.CompletionTokens += usage.CompletionTokens
	latestTokenStats.TotalTokens += usage.TotalTokens
	latestTokenStats.CacheHitTokens += usage.PromptCacheHitTokens
	latestTokenStats.CacheMissTokens += usage.PromptCacheMissTokens
	latestTokenStats.SystemTokens += usage.SystemTokens
	latestTokenStats.SkillsTokens += usage.SkillsTokens
	latestTokenStats.MCPTokens += usage.MCPTokens
	latestTokenStats.ToolTokens += usage.ToolTokens
	latestTokenStats.HistoryTokens += usage.HistoryTokens
	latestTokenStats.OtherTokens += usage.OtherTokens

	path := tokenStatsPath(root)
	if path == "" {
		return
	}
	data, _ := json.MarshalIndent(latestTokenStats, "", "  ")
	os.WriteFile(path, data, 0644)
}

// ResetTokenStats 重置累积 token 统计（全量清零并写盘）。
// 使用全局 WorkspaceRoots[0] 确定存储路径（UI 层调用）。
func ResetTokenStats() {
	root := ""
	if len(WorkspaceRoots) > 0 {
		root = WorkspaceRoots[0]
	}
	ResetTokenStatsForRoot(root)
}

// ResetTokenStatsForRoot 按指定工作区路径重置 token 统计。
func ResetTokenStatsForRoot(root string) {
	tokenStatsMu.Lock()
	defer tokenStatsMu.Unlock()
	latestTokenStats = TokenStats{}
	if path := tokenStatsPath(root); path != "" {
		data, _ := json.MarshalIndent(latestTokenStats, "", "  ")
		os.WriteFile(path, data, 0644)
	}
}

// ReadTokenStats 从磁盘读取已持久化的 token 统计。
// 使用全局 WorkspaceRoots[0] 确定存储路径（UI 层调用）。
// 外部宿主（web 服务）通过此函数获取 agent 自闭环保存的统计数据。
func ReadTokenStats() *TokenStats {
	root := ""
	if len(WorkspaceRoots) > 0 {
		root = WorkspaceRoots[0]
	}
	return ReadTokenStatsForRoot(root)
}

// ReadTokenStatsForRoot 按指定工作区路径读取 token 统计。
func ReadTokenStatsForRoot(root string) *TokenStats {
	tokenStatsMu.Lock()
	defer tokenStatsMu.Unlock()

	path := tokenStatsPath(root)
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

// SaveTokenUsage 累积 token 用量到磁盘（向后兼容，使用 WorkspaceRoots[0]）。
func SaveTokenUsage(usage *Usage) {
	root := ""
	if len(WorkspaceRoots) > 0 {
		root = WorkspaceRoots[0]
	}
	SaveTokenUsageForRoot(root, usage)
}
