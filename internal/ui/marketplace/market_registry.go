// Package marketplace 是 MCP/Skills 市场注册表的 UI 层薄壳代理。
//
// 所有内置注册表数据和安装业务逻辑已迁入 agent 包。
// 本文件仅保留 UI 层特有的功能：
//   - 类型别名（RegistryEntry = agent.MarketEntry）保持旧 API 兼容
//   - 远程注册表获取 / 缓存（FetchRemoteRegistry、Init）
//   - 实时 API 搜索（searchNPM / searchGitHub）
//   - Search 函数（本地部分委托 agent.MarketSearch，远程部分用 API 搜索）
//   - Find 函数（委托 agent.MarketFind，找不到再查远程缓存）
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
	"strings"
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

	searchCache   map[string]RegistryEntry
	searchCacheMu sync.RWMutex
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
// 本地部分委托 agent.MarketSearch，远程部分实时搜索 npm/GitHub。
func Search(query, kind string) []RegistryEntry {
	if kind == "" || kind == "all" {
		kind = ""
	}

	// 1. 本地搜索（委托 agent 内置注册表）
	local := agent.MarketSearch(query, kind)

	// 2. 没有查询关键词时只返回本地（MCP/技能）；
	//    插件市场无关键词时也实时搜 npm（返回热门 cordis 插件）
	if query == "" && kind != "plugin" {
		return local
	}

	// 3. 实时 API 搜索
	type apiResult struct {
		entries []RegistryEntry
		kind    string
	}
	ch := make(chan apiResult, 3)

	go func() {
		if kind == "" || kind == "mcp" {
			ch <- apiResult{searchNPM(query), "mcp"}
		} else {
			ch <- apiResult{nil, "mcp"}
		}
	}()
	go func() {
		if kind == "" || kind == "skill" {
			ch <- apiResult{searchGitHub(query), "skill"}
		} else {
			ch <- apiResult{nil, "skill"}
		}
	}()
	go func() {
		if kind == "" || kind == "plugin" {
			ch <- apiResult{searchNPMPlugins(query), "plugin"}
		} else {
			ch <- apiResult{nil, "plugin"}
		}
	}()

	var mcpResults, skillResults, pluginResults []RegistryEntry
	for i := 0; i < 3; i++ {
		r := <-ch
		switch r.kind {
		case "mcp":
			mcpResults = r.entries
		case "skill":
			skillResults = r.entries
		case "plugin":
			pluginResults = r.entries
		}
	}

	// 4. 合并去重（本地优先，远程补充）
	seen := make(map[string]bool, len(local))
	for _, e := range local {
		seen[e.ID] = true
	}
	for _, e := range mcpResults {
		if !seen[e.ID] {
			seen[e.ID] = true
			local = append(local, e)
		}
	}
	for _, e := range skillResults {
		if !seen[e.ID] {
			seen[e.ID] = true
			local = append(local, e)
		}
	}
	for _, e := range pluginResults {
		if !seen[e.ID] {
			seen[e.ID] = true
			local = append(local, e)
		}
	}

	// 5. 缓存搜索结果
	searchCacheMu.Lock()
	searchCache = make(map[string]RegistryEntry, len(local))
	for _, e := range local {
		searchCache[e.ID] = e
	}
	searchCacheMu.Unlock()

	return local
}

// ─── 实时 API 搜索（npm registry + GitHub）──

const (
	apiSearchTimeout = 8 * time.Second
	maxAPISearch     = 20
)

// searchNPMPlugins 搜索 npm 上的 cordis 插件（参考项目 deepseek-harness 的
// 插件市场形态：cordis 插件发布在 npm）。query 空 → 热门 cordis 插件。
// 过滤：名字含 plugin/cordis 且非框架本体（@cordisjs/core、cordis）。
func searchNPMPlugins(query string) []RegistryEntry {
	q := strings.TrimSpace(query)
	if q == "" {
		q = "cordis plugin"
	} else {
		q = q + " cordis"
	}
	searchQ := url.QueryEscape(q)
	apiURL := "https://registry.npmjs.org/-/v1/search?text=" + searchQ + "&size=" + fmt.Sprintf("%d", maxAPISearch)

	client := &http.Client{Timeout: apiSearchTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Objects []struct {
			Package struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Version     string   `json:"version"`
				Keywords    []string `json:"keywords"`
				Links       struct {
					Npm      string `json:"npm"`
					Homepage string `json:"homepage"`
				} `json:"links"`
			} `json:"package"`
			Score struct {
				Final float64 `json:"final"`
			} `json:"score"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []RegistryEntry
	for _, obj := range result.Objects {
		pkg := obj.Package
		low := strings.ToLower(pkg.Name)
		// 排除框架本体
		if low == "cordis" || low == "@cordisjs/core" || low == "@cordisjs/plugin-loader" {
			continue
		}
		// 插件判定：名字含 plugin 或描述标注 cordis 插件
		if !strings.Contains(low, "plugin") && !strings.Contains(strings.ToLower(pkg.Description), "cordis") {
			continue
		}
		name := pkg.Name
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			name = parts[1]
		}
		entries = append(entries, RegistryEntry{
			ID:          pkg.Name,
			Kind:        "plugin",
			Name:        name,
			Description: pkg.Description,
			Tags:        append([]string{"plugin", "cordis"}, pkg.Keywords...),
			Source:      "npm:" + pkg.Name + "@" + pkg.Version,
		})
	}
	return entries
}

func searchNPM(query string) []RegistryEntry {
	if query == "" {
		return nil
	}
	searchQ := url.QueryEscape(query + " mcp")
	apiURL := "https://registry.npmjs.org/-/v1/search?text=" + searchQ + "&size=" + fmt.Sprintf("%d", maxAPISearch)

	client := &http.Client{Timeout: apiSearchTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Objects []struct {
			Package struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Version     string   `json:"version"`
				Keywords    []string `json:"keywords"`
				Date        string   `json:"date"`
				Links       struct {
					Npm      string `json:"npm"`
					Homepage string `json:"homepage"`
				} `json:"links"`
				Publisher struct {
					Username string `json:"username"`
				} `json:"publisher"`
			} `json:"package"`
			Score struct {
				Final float64 `json:"final"`
			} `json:"score"`
		} `json:"objects"`
		Total int `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []RegistryEntry
	for _, obj := range result.Objects {
		pkg := obj.Package
		id := "npm-" + pkg.Name
		name := pkg.Name
		if strings.HasPrefix(name, "@") {
			if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
				name = parts[1]
			}
		} else {
			if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
				name = parts[1]
			}
		}

		entries = append(entries, RegistryEntry{
			ID: id, Kind: "mcp", Name: name,
			Description: pkg.Description,
			Tags:        append([]string{"mcp"}, pkg.Keywords...),
			Source:      pkg.Links.Npm,
			Command:     "npx", Args: []string{"-y", pkg.Name},
		})
	}
	return entries
}

func searchGitHub(query string) []RegistryEntry {
	if query == "" {
		return nil
	}
	searchQ := url.QueryEscape(query)
	apiURL := "https://api.github.com/search/repositories?q=" + searchQ + "&sort=stars&per_page=" + fmt.Sprintf("%d", maxAPISearch)

	client := &http.Client{Timeout: apiSearchTimeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Pair-CodeAgent/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var result struct {
		Items []struct {
			ID          int      `json:"id"`
			FullName    string   `json:"full_name"`
			Description string   `json:"description"`
			Stars       int      `json:"stargazers_count"`
			UpdatedAt   string   `json:"updated_at"`
			HTMLURL     string   `json:"html_url"`
			Topics      []string `json:"topics"`
			Language    string   `json:"language"`
		} `json:"items"`
		TotalCount int `json:"total_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []RegistryEntry
	for _, item := range result.Items {
		id := "gh-" + item.FullName
		name := item.FullName
		if parts := strings.SplitN(item.FullName, "/", 2); len(parts) == 2 {
			name = parts[1]
		}
		tags := append([]string{"skill"}, item.Topics...)
		if item.Language != "" {
			tags = append(tags, item.Language)
		}
		entries = append(entries, RegistryEntry{
			ID: id, Kind: "skill", Name: name,
			Description: item.Description, Tags: tags,
			Source: item.HTMLURL,
		})
	}
	return entries
}

// Find 按 ID 查找注册表条目（优先搜索缓存，再委托 agent）。
func Find(id string) *RegistryEntry {
	// 优先查瞬态搜索缓存（npm/GitHub 实时搜索结果）
	searchCacheMu.RLock()
	if cached, ok := searchCache[id]; ok {
		searchCacheMu.RUnlock()
		return &cached
	}
	searchCacheMu.RUnlock()

	// 再查远程缓存
	fetchMu.RLock()
	for _, e := range remoteEntries {
		if e.ID == id {
			fetchMu.RUnlock()
			return &e
		}
	}
	fetchMu.RUnlock()

	// 最后委托 agent 内置注册表
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
