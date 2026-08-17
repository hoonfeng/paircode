// 市场注册表 —— 自闭环：搜索 + 安装。
// ★ 2026-08-17：无预设数据——内置注册表已清空。用户搜索（query 非空）才触发
//   实时远程搜索（npm MCP / GitHub skill / npm cordis 插件），搜索结果显示后
//   按 id 安装。MarketSearch/MarketFind 是 agent 工具与前端市场的统一入口。
// 无 //go:build 标签，全平台可用。

package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ─── 注册表条目类型 ───

// MarketEntry 市场注册表条目。
type MarketEntry struct {
	ID          string   // 唯一标识（用于安装）
	Kind        string   // "mcp" / "skill" / "plugin"
	Name        string   // 显示名称
	Description string   // 简述
	Tags        []string // 标签
	Source      string   // 来源标注（如 github:modelcontextprotocol/servers）

	// MCP 专用
	Command string   // 启动命令
	Args    []string // 启动参数

	// Skills 专用
	Content    string // SKILL.md 正文（空=仅创建元信息）
	Activation string // auto/always/manual
}

// MarketSkill 快速创建技能条目的辅助函数。
func MarketSkill(id, name, desc, activation, content string) MarketEntry {
	return MarketEntry{
		ID: id, Kind: "skill", Name: name,
		Description: desc, Tags: []string{"skill"},
		Activation: activation, Content: content,
	}
}

// MarketMCP 快速创建 MCP 条目的辅助函数。
func MarketMCP(id, name, desc, cmd string, args []string) MarketEntry {
	return MarketEntry{
		ID: id, Kind: "mcp", Name: name,
		Description: desc, Tags: []string{"mcp"},
		Command: cmd, Args: args,
	}
}

// ─── 内置注册表（★ 已清空：无预设数据） ───
// 2026-08-17：市场不再内置任何预设条目——打开市场不显示数据，
// 用户搜索时才通过远程 API（npm/GitHub）实时检索并展示。
// 保留空切片供 MarketAllEntries 返回；MarketSkill/MarketMCP 等辅助
// 构造函数保留（插件/脚本可自行构造条目走 MarketInstallEntry）。

// BuiltinRegistry 内置条目列表（恒为空——无预设数据）。
var BuiltinRegistry = []MarketEntry{}

// ─── 远程搜索（实时 npm/GitHub，仅搜索时触发）───

const (
	marketAPISearchTimeout = 8 * time.Second
	maxMarketAPISearch     = 20
)

var (
	marketCacheMu sync.RWMutex
	marketCache   = map[string]MarketEntry{} // 最近一次搜索结果的瞬态缓存（MarketFind 按 id 安装用）
)

// MarketSearch 搜索市场条目。
// ★ 2026-08-17：无预设数据——query 空返回空（打开市场不展示任何条目）；
//   query 非空才触发实时远程搜索（npm MCP / GitHub skill / npm cordis 插件），
//   并发请求、合并去重后返回并缓存（MarketFind 可查）。
// kind 可空/""=全部，"mcp"/"skill"/"plugin"=指定类型。
func MarketSearch(query, kind string) []MarketEntry {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	if kind == "" || kind == "all" {
		kind = ""
	}
	type apiResult struct {
		entries []MarketEntry
		kind    string
	}
	ch := make(chan apiResult, 3)
	go func() {
		if kind == "" || kind == "mcp" {
			ch <- apiResult{searchMarketNPM(query), "mcp"}
		} else {
			ch <- apiResult{nil, "mcp"}
		}
	}()
	go func() {
		if kind == "" || kind == "skill" {
			ch <- apiResult{searchMarketGitHub(query), "skill"}
		} else {
			ch <- apiResult{nil, "skill"}
		}
	}()
	go func() {
		if kind == "" || kind == "plugin" {
			ch <- apiResult{searchMarketNPMPlugins(query), "plugin"}
		} else {
			ch <- apiResult{nil, "plugin"}
		}
	}()

	var out []MarketEntry
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		r := <-ch
		for _, e := range r.entries {
			if !seen[e.ID] {
				seen[e.ID] = true
				out = append(out, e)
			}
		}
	}
	// 缓存搜索结果（MarketFind / MarketInstallScoped 按 id 使用）
	marketCacheMu.Lock()
	marketCache = make(map[string]MarketEntry, len(out))
	for _, e := range out {
		marketCache[e.ID] = e
	}
	marketCacheMu.Unlock()
	return out
}

// MarketAllEntries 获取所有市场条目（★ 无预设数据——恒返回空，搜索才显示）。
func MarketAllEntries() []MarketEntry {
	return nil
}

// MarketFind 按 ID 查找市场条目：先查最近搜索缓存（远程搜索结果），
// 再查内置注册表（已清空，恒不命中）。
func MarketFind(id string) *MarketEntry {
	marketCacheMu.RLock()
	if e, ok := marketCache[id]; ok {
		marketCacheMu.RUnlock()
		return &e
	}
	marketCacheMu.RUnlock()
	for _, e := range BuiltinRegistry {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

// ─── 远程 API 搜索实现（npm registry + GitHub）──

// searchMarketNPMPlugins 搜索 npm 上的 cordis 插件。过滤：名字含 plugin/cordis
// 且非框架本体（@cordisjs/core、cordis）。
func searchMarketNPMPlugins(query string) []MarketEntry {
	q := strings.TrimSpace(query)
	q = q + " cordis"
	searchQ := url.QueryEscape(q)
	apiURL := "https://registry.npmjs.org/-/v1/search?text=" + searchQ + "&size=" + fmt.Sprintf("%d", maxMarketAPISearch)

	client := &http.Client{Timeout: marketAPISearchTimeout}
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
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []MarketEntry
	for _, obj := range result.Objects {
		pkg := obj.Package
		low := strings.ToLower(pkg.Name)
		if low == "cordis" || low == "@cordisjs/core" || low == "@cordisjs/plugin-loader" {
			continue
		}
		if !strings.Contains(low, "plugin") && !strings.Contains(strings.ToLower(pkg.Description), "cordis") {
			continue
		}
		name := pkg.Name
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			name = parts[1]
		}
		entries = append(entries, MarketEntry{
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

// searchMarketNPM 搜索 npm 上的 MCP 服务器（query + " mcp"）。
func searchMarketNPM(query string) []MarketEntry {
	if query == "" {
		return nil
	}
	searchQ := url.QueryEscape(query + " mcp")
	apiURL := "https://registry.npmjs.org/-/v1/search?text=" + searchQ + "&size=" + fmt.Sprintf("%d", maxMarketAPISearch)

	client := &http.Client{Timeout: marketAPISearchTimeout}
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
					Npm string `json:"npm"`
				} `json:"links"`
			} `json:"package"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []MarketEntry
	for _, obj := range result.Objects {
		pkg := obj.Package
		id := "npm-" + pkg.Name
		name := pkg.Name
		if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
			name = parts[1]
		}
		entries = append(entries, MarketEntry{
			ID: id, Kind: "mcp", Name: name,
			Description: pkg.Description,
			Tags:        append([]string{"mcp"}, pkg.Keywords...),
			Source:      pkg.Links.Npm,
			Command:     "npx", Args: []string{"-y", pkg.Name},
		})
	}
	return entries
}

// searchMarketGitHub 搜索 GitHub 仓库作为技能条目（按 stars 排序）。
func searchMarketGitHub(query string) []MarketEntry {
	if query == "" {
		return nil
	}
	searchQ := url.QueryEscape(query)
	apiURL := "https://api.github.com/search/repositories?q=" + searchQ + "&sort=stars&per_page=" + fmt.Sprintf("%d", maxMarketAPISearch)

	client := &http.Client{Timeout: marketAPISearchTimeout}
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
			FullName    string   `json:"full_name"`
			Description string   `json:"description"`
			HTMLURL     string   `json:"html_url"`
			Topics      []string `json:"topics"`
			Language    string   `json:"language"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	var entries []MarketEntry
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
		entries = append(entries, MarketEntry{
			ID: id, Kind: "skill", Name: name,
			Description: item.Description, Tags: tags,
			Source: item.HTMLURL,
		})
	}
	return entries
}
