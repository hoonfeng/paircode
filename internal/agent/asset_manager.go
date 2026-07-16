package agent

// AgentAssetManager — 统一智能资产管理器
// 管理进化系统资产（经验胶囊 + 技能基因），提供 Agent 可调用的统一查询接口。
// Agent 通过本管理器可以统一查看、搜索、管理所有类型的进化资产。
//
// 资产类型：
//   - "capsules" : 经验胶囊（Capsule），修复经验的持久化编码
//   - "genes"    : 技能基因（Gene），跨项目复用的最佳实践
//
// 所有资产统一存储在：
//   安装目录/.pair/assets/{scope}/{type}/  （全局）
//   工作区/.pair/assets/{scope}/{type}/   （项目级）

import (
	"strings"
	"sync"
)

// AssetInfo 统一资产信息（供 Agent 查看和搜索）。
type AssetInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`  // "capsule" | "gene"
	Scope       string   `json:"scope"` // "global" | "project"
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Version     string   `json:"version,omitempty"`
	CreatedAt   string   `json:"created_at"`
}

// AgentAssetManager 统一智能资产管理器。
// 持有 EvolutionEngine，提供统一操作接口。
type AgentAssetManager struct {
	mu        sync.RWMutex
	root      string
	evolution *EvolutionEngine
	store     *AssetStore
}

// NewAgentAssetManager 创建统一智能资产管理器。
// root: 当前工作区根目录。
func NewAgentAssetManager(root string) *AgentAssetManager {
	return &AgentAssetManager{
		root:      root,
		evolution: NewEvolutionEngine(root),
		store:     NewAssetStore(root),
	}
}

// ── 分管理器访问 ──────────────────────────────────────────

// Evolution 返回进化引擎。
func (m *AgentAssetManager) Evolution() *EvolutionEngine {
	return m.evolution
}

// Store 返回底层资产存储层。
func (m *AgentAssetManager) Store() *AssetStore {
	return m.store
}

// ── 统一列表 ──────────────────────────────────────────────

// AllAssets 列出所有类型的资产（胶囊 + 基因）。
// scope 为空时列出全部作用域，否则按 "global" | "project" 过滤。
// assetType 为空时列出全部类型，否则按 "capsules" | "genes" 过滤。
func (m *AgentAssetManager) AllAssets(scope, assetType string) []AssetInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assets := make([]AssetInfo, 0)

	// 列出胶囊
	if assetType == "" || assetType == "capsules" {
		capsules := m.evolution.SearchCapsules("", "", nil)
		for _, c := range capsules {
			s := c.Scope
			if s == "" {
				s = "global"
			}
			if scope != "" && s != scope {
				continue
			}
			assets = append(assets, AssetInfo{
				ID:          c.ID,
				Name:        c.ID,
				Type:        "capsule",
				Scope:       s,
				Description: c.Solution.Summary,
				Tags:        c.Signal.ContextTags,
				CreatedAt:   c.CreatedAt,
			})
		}
	}

	// 列出基因
	if assetType == "" || assetType == "genes" {
		genes := m.evolution.SearchGenes("", "", nil)
		for _, g := range genes {
			s := g.Scope
			if s == "" {
				s = "global"
			}
			if scope != "" && s != scope {
				continue
			}
			assets = append(assets, AssetInfo{
				ID:          g.ID,
				Name:        g.Name,
				Type:        "gene",
				Scope:       s,
				Description: g.Description,
				Tags:        g.Tags,
				CreatedAt:   "", // 基因没有创建时间字段
			})
		}
	}

	return assets
}

// SearchAssets 搜索所有类型的资产，返回匹配结果。
// query 为空时返回全部（相当于 AllAssets）。
// assetType 可为空（全部类型）或 "capsules" | "genes"。
func (m *AgentAssetManager) SearchAssets(query, scope, assetType string) []AssetInfo {
	all := m.AllAssets(scope, assetType)
	if query == "" {
		return all
	}

	q := strings.ToLower(query)
	matched := make([]AssetInfo, 0, len(all))
	for _, a := range all {
		if strings.Contains(strings.ToLower(a.Name), q) ||
			strings.Contains(strings.ToLower(a.Description), q) ||
			containsAny(strings.ToLower(strings.Join(a.Tags, " ")), q) {
			matched = append(matched, a)
		}
	}
	return matched
}

// GetAssetCount 获取各类资产的数量统计。
func (m *AgentAssetManager) GetAssetCount() map[string]int {
	result := map[string]int{
		"capsules_total": 0,
		"genes_total":    0,
		"total":          0,
	}

	status := m.evolution.GetStatus()
	result["capsules_total"] = status.CapsuleCount
	result["genes_total"] = status.GeneCount
	result["total"] += status.CapsuleCount + status.GeneCount

	return result
}

// ── 全局单例 ──────────────────────────────────────────────

var (
	globalAssetManager   *AgentAssetManager
	globalAssetManagerMu sync.Mutex
)

// GetAgentAssetManager 获取全局统一资产管理器。
func GetAgentAssetManager() *AgentAssetManager {
	globalAssetManagerMu.Lock()
	defer globalAssetManagerMu.Unlock()
	return globalAssetManager
}

// SetAgentAssetManager 设置全局统一资产管理器（在 bridge 初始化时调用）。
func SetAgentAssetManager(mgr *AgentAssetManager) {
	globalAssetManagerMu.Lock()
	defer globalAssetManagerMu.Unlock()
	globalAssetManager = mgr
}

// ── 辅助 ──────────────────────────────────────────────────

func containsAny(s, substr string) bool {
	parts := strings.Fields(substr)
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
