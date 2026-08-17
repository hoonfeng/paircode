// Package marketplace 是 MCP/Skills 市场注册表的 UI 层薄壳代理。
//
// 所有注册表数据、安装与实时搜索逻辑已迁入 agent 包（无预设数据——
// 用户搜索才触发远程检索）。本文件仅保留 UI 层特有的功能：
//   - 类型别名（RegistryEntry = agent.MarketEntry）保持旧 API 兼容
//   - 远程注册表获取 / 缓存（FetchRemoteRegistry、Init）
//   - Search / Find（委托 agent.MarketSearch / agent.MarketFind）
//
//go:build windows

package marketplace

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

// ─── 远程市场默认 URL ───
const DefaultMarketplaceURL = ""

// 条目字段最大长度（防注入）
const (
	maxIDLen           = 128
	maxNameLen         = 200
	maxDescLen         = 500
	maxTagLen          = 50
	maxTagsCount       = 20
	maxContentLen      = 100 * 1024 // 100KB
	maxCommandLen      = 256
	maxArgsCount       = 50
	maxArgLen          = 512
	maxEntriesPerFetch = 5000
)

// ─── 类型别名（与 agent 兼容）──

// RegistryEntry 市场注册表条目。
type RegistryEntry = agent.MarketEntry

// RegistrySkill 快速创建技能条目的辅助函数。
func RegistrySkill(id, name, desc, activation, content string) RegistryEntry {
	return RegistryEntry{
		ID: id, Kind: "skill", Name: name,
		Description: desc, Tags: []string{"skill"},
		Activation: activation, Content: content,
	}
}

// RegistryMCP 快速创建 MCP 条目的辅助函数。
func RegistryMCP(id, name, desc, cmd string, args []string) RegistryEntry {
	return RegistryEntry{
		ID: id, Kind: "mcp", Name: name,
		Description: desc, Tags: []string{"mcp"},
		Command: cmd, Args: args,
	}
}

// ─── 远程注册表缓存状态 ───

var (
	remoteEntries []RegistryEntry
	remoteURL     = DefaultMarketplaceURL
	lastFetchTime time.Time
	fetchMu       sync.RWMutex
	lastFetchErr  string
	cacheInitOnce sync.Once
)

// SetMarketplaceURL 设置远程市场 URL（仅 HTTPS）。
func SetMarketplaceURL(rawURL string) error {
	fetchMu.Lock()
	defer fetchMu.Unlock()
	if rawURL == "" {
		remoteURL = DefaultMarketplaceURL
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 格式无效: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("仅支持 HTTPS 协议，不支持 %q", u.Scheme)
	}
	remoteURL = rawURL
	return nil
}

// GetMarketplaceURL 获取当前远程市场 URL。
func GetMarketplaceURL() string {
	fetchMu.RLock()
	defer fetchMu.RUnlock()
	return remoteURL
}

// GetLastFetchTime 获取上次成功获取的时间。
func GetLastFetchTime() time.Time {
	fetchMu.RLock()
	defer fetchMu.RUnlock()
	return lastFetchTime
}

// GetLastFetchErr 获取上次获取错误信息。
func GetLastFetchErr() string {
	fetchMu.RLock()
	defer fetchMu.RUnlock()
	return lastFetchErr
}

// FetchStatus 返回市场获取状态摘要。
func FetchStatus() string {
	fetchMu.RLock()
	defer fetchMu.RUnlock()
	if lastFetchErr != "" {
		return fmt.Sprintf("远程市场获取失败: %s（使用内置数据）", lastFetchErr)
	}
	if !lastFetchTime.IsZero() {
		return fmt.Sprintf("远程市场已获取（%s），共 %d 个条目", lastFetchTime.Format("15:04:05"), len(remoteEntries))
	}
	return "尚未获取远程市场（使用内置数据）"
}

// ─── 条目验证（远程获取时校验）──

func validateEntry(e RegistryEntry) error {
	if len(e.ID) > maxIDLen {
		return fmt.Errorf("ID 过长: %d > %d", len(e.ID), maxIDLen)
	}
	if len(e.Name) > maxNameLen {
		return fmt.Errorf("Name 过长: %d > %d", len(e.Name), maxNameLen)
	}
	if len(e.Description) > maxDescLen {
		return fmt.Errorf("Description 过长: %d > %d", len(e.Description), maxDescLen)
	}
	if len(e.Tags) > maxTagsCount {
		return fmt.Errorf("Tags 数量过多: %d > %d", len(e.Tags), maxTagsCount)
	}
	for _, tag := range e.Tags {
		if len(tag) > maxTagLen {
			return fmt.Errorf("Tag 过长: %d > %d", len(tag), maxTagLen)
		}
	}
	if len(e.Content) > maxContentLen {
		return fmt.Errorf("Content 过长: %d > %d", len(e.Content), maxContentLen)
	}
	if len(e.Command) > maxCommandLen {
		return fmt.Errorf("Command 过长: %d > %d", len(e.Command), maxCommandLen)
	}
	if len(e.Args) > maxArgsCount {
		return fmt.Errorf("Args 数量过多: %d > %d", len(e.Args), maxArgsCount)
	}
	for _, arg := range e.Args {
		if len(arg) > maxArgLen {
			return fmt.Errorf("Arg 过长: %d > %d", len(arg), maxArgLen)
		}
	}
	if e.Kind != "mcp" && e.Kind != "skill" {
		return fmt.Errorf("无效 Kind: %q（仅支持 mcp/skill）", e.Kind)
	}
	return nil
}

// ─── 远程获取 ───

// FetchRemoteRegistry 从远程 URL 获取市场注册表 JSON 并缓存到本地。
func FetchRemoteRegistry(workspaceRoot string, force bool) error {
	fetchMu.Lock()
	defer fetchMu.Unlock()

	if !force && !lastFetchTime.IsZero() && time.Since(lastFetchTime) < 1*time.Hour {
		return nil
	}

	url := remoteURL
	if url == "" {
		lastFetchErr = ""
		return nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		lastFetchErr = ""
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		lastFetchErr = ""
		return nil
	}

	var registry struct {
		Version int             `json:"version"`
		Entries []RegistryEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registry); err != nil {
		lastFetchErr = fmt.Sprintf("JSON 解析失败: %v", err)
		return err
	}

	if len(registry.Entries) > maxEntriesPerFetch {
		lastFetchErr = fmt.Sprintf("远程条目过多（%d），超过上限 %d", len(registry.Entries), maxEntriesPerFetch)
		return fmt.Errorf("远程条目过多: %d > %d", len(registry.Entries), maxEntriesPerFetch)
	}

	for _, e := range registry.Entries {
		if err := validateEntry(e); err != nil {
			lastFetchErr = fmt.Sprintf("条目 %q 校验失败: %v", e.ID, err)
			return fmt.Errorf("条目 %q 校验失败: %w", e.ID, err)
		}
	}

	remoteEntries = registry.Entries
	lastFetchTime = time.Now()
	lastFetchErr = ""

	if workspaceRoot != "" {
		cacheDir := filepath.Join(workspaceRoot, ".pair", "marketplace")
		os.MkdirAll(cacheDir, 0755)
		cacheFile := filepath.Join(cacheDir, "registry_cache.json")
		if data, err := json.MarshalIndent(registry, "", "  "); err == nil {
			os.WriteFile(cacheFile, data, 0644)
		}
	}

	return nil
}

// loadCachedRegistry 从本地缓存加载远程注册表。
func loadCachedRegistry(workspaceRoot string) {
	if workspaceRoot == "" {
		return
	}
	cacheFile := filepath.Join(workspaceRoot, ".pair", "marketplace", "registry_cache.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return
	}
	var registry struct {
		Version int             `json:"version"`
		Entries []RegistryEntry `json:"entries"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return
	}
	if len(registry.Entries) == 0 {
		return
	}
	for _, e := range registry.Entries {
		if err := validateEntry(e); err != nil {
			return
		}
	}

	fetchMu.Lock()
	remoteEntries = registry.Entries
	lastFetchTime = time.Now()
	fetchMu.Unlock()
}

// ─── 查询（本地内置 + 远程 API）──

// Search 按关键词和类型搜索市场注册表。
// ★ 2026-08-17：无预设数据——query 空返回空（打开市场不展示任何条目）；
//   query 非空委托 agent.MarketSearch 实时远程搜索（npm/GitHub），并缓存结果。
func Search(query, kind string) []RegistryEntry {
	return agent.MarketSearch(query, kind)
}

// Find 按 ID 查找注册表条目：先查远程注册表缓存，再委托 agent（搜索缓存+内置）。
func Find(id string) *RegistryEntry {
	fetchMu.RLock()
	for _, e := range remoteEntries {
		if e.ID == id {
			fetchMu.RUnlock()
			return &e
		}
	}
	fetchMu.RUnlock()
	return agent.MarketFind(id)
}

// Init 初始化市场系统：尝试从本地缓存加载，异步获取远程。
func Init(workspaceRoot string) {
	cacheInitOnce.Do(func() {
		loadCachedRegistry(workspaceRoot)
	})
	go func() {
		FetchRemoteRegistry(workspaceRoot, false)
	}()
}
