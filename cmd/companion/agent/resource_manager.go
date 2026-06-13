// 统一资源管理 —— 将 Skills / MCP / 记忆 / 项目知识库 / 进化 / Lua 工具
// 全部纳入统一查询层（ResourceProvider 接口），便于 Agent 跨类型查看和管理。
// 写入保持各自独立工具，查询层统一。
// 为后期提取通用 Agent 基座铺垫。

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// ResourceType 资源类型枚举。
type ResourceType string

const (
	ResourceCapsules    ResourceType = "capsules"     // 经验胶囊
	ResourceGenes       ResourceType = "genes"        // 技能基因
	ResourceMemory      ResourceType = "memory"       // 项目记忆
	ResourceProjectInfo ResourceType = "project-info" // 项目知识库
	ResourceSkills      ResourceType = "skills"       // 技能
	ResourceMCPServers  ResourceType = "mcp-servers"  // MCP 服务器
	ResourceLuaTools    ResourceType = "lua-tools"    // Lua 自定义工具
)

// ResourceMeta 统一资源元信息（跨类型列表和搜索用）。
type ResourceMeta struct {
	ID          string       `json:"id"`          // 唯一标识
	Type        ResourceType `json:"type"`        // 资源类型
	Scope       string       `json:"scope"`       // 作用域: "global" | "project" | ""（无作用域）
	Name        string       `json:"name"`        // 显示名称
	Description string       `json:"description"` // 简要描述
	Tags        []string     `json:"tags"`        // 标签
	Size        int64        `json:"size"`        // 字节数
	ModifiedAt  string       `json:"modified_at"` // 修改时间（ISO 8601）
}

// ResourceProvider 某类资源的查询接口。
// 每种资源类型实现此接口，供 UnifiedResourceManager 统一查询。
type ResourceProvider interface {
	// Type 返回资源类型。
	Type() ResourceType

	// List 列出所有资源（scope=空 列出全部作用域，否则按 global/project 过滤）。
	List(scope string) ([]ResourceMeta, error)

	// Search 按关键词搜索资源（匹配 ID/名称/描述/标签）。
	Search(query, scope string) ([]ResourceMeta, error)

	// Count 返回资源数量（按 scope 过滤）。
	Count(scope string) (int, error)
}

// ─── 统一资源管理器 ──────────────────────────────────────────

// UnifiedResourceManager 统一资源管理器。
// 持有所有 ResourceProvider，提供跨类型查询。
type UnifiedResourceManager struct {
	mu        sync.RWMutex
	providers map[ResourceType]ResourceProvider
	root      string
}

// NewUnifiedResourceManager 创建统一资源管理器。
// root: 当前工作区根目录（用于查询项目级资源）。
func NewUnifiedResourceManager(root string) *UnifiedResourceManager {
	mgr := &UnifiedResourceManager{
		providers: make(map[ResourceType]ResourceProvider),
		root:      root,
	}

	// 注册内置 Provider
	mgr.Register(&capsuleProvider{root: root})
	mgr.Register(&geneProvider{root: root})
	mgr.Register(&memoryProvider{root: root})
	mgr.Register(&projectInfoProvider{root: root})
	mgr.Register(&skillsProvider{root: root})
	mgr.Register(&mcpServerProvider{})
	mgr.Register(&luaToolProvider{root: root})

	return mgr
}

// Register 注册一个资源 Provider。
func (m *UnifiedResourceManager) Register(p ResourceProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Type()] = p
}

// GetProvider 按类型获取 Provider。
func (m *UnifiedResourceManager) GetProvider(t ResourceType) ResourceProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.providers[t]
}

// ListAll 跨类型列出所有资源。
// resourceType 为空则列出全部类型，scope 为空则全部作用域。
func (m *UnifiedResourceManager) ListAll(resourceType string, scope string) []ResourceMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []ResourceMeta
	for rt, p := range m.providers {
		if resourceType != "" && string(rt) != resourceType {
			continue
		}
		items, err := p.List(scope)
		if err != nil {
			continue
		}
		result = append(result, items...)
	}

	// 按类型排序（保持可读性）
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// SearchAll 跨类型搜索资源。
func (m *UnifiedResourceManager) SearchAll(query, resourceType, scope string) []ResourceMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return m.ListAll(resourceType, scope)
	}

	var result []ResourceMeta
	for rt, p := range m.providers {
		if resourceType != "" && string(rt) != resourceType {
			continue
		}
		items, err := p.Search(q, scope)
		if err != nil {
			continue
		}
		result = append(result, items...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// CountAll 跨类型统计资源数量。
func (m *UnifiedResourceManager) CountAll(scope string) map[ResourceType]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[ResourceType]int)
	for rt, p := range m.providers {
		count, err := p.Count(scope)
		if err != nil {
			continue
		}
		result[rt] = count
	}
	return result
}

// ─── 全局单例 ─────────────────────────────────────────────

var (
	globalResourceMgr   *UnifiedResourceManager
	globalResourceMgrMu sync.Mutex
)

// GetResourceManager 获取全局统一资源管理器。
func GetResourceManager() *UnifiedResourceManager {
	globalResourceMgrMu.Lock()
	defer globalResourceMgrMu.Unlock()
	return globalResourceMgr
}

// SetResourceManager 设置全局统一资源管理器（bridge 初始化时调用）。
func SetResourceManager(mgr *UnifiedResourceManager) {
	globalResourceMgrMu.Lock()
	defer globalResourceMgrMu.Unlock()
	globalResourceMgr = mgr
}

// ─── Provider 实现 ─────────────────────────────────────────

// 1. capsuleProvider — 经验胶囊（来自 EvolutionEngine）

type capsuleProvider struct {
	root string
}

func (p *capsuleProvider) Type() ResourceType { return ResourceCapsules }

func (p *capsuleProvider) List(scope string) ([]ResourceMeta, error) {
	engine := NewEvolutionEngine(p.root)
	capsules := engine.SearchCapsules("", "", nil)
	var result []ResourceMeta
	for _, c := range capsules {
		s := c.Scope
		if s == "" {
			s = "global"
		}
		if scope != "" && s != scope {
			continue
		}
		result = append(result, ResourceMeta{
			ID:          c.ID,
			Type:        ResourceCapsules,
			Scope:       s,
			Name:        c.ID,
			Description: c.Solution.Summary,
			Tags:        c.Signal.ContextTags,
		})
	}
	return result, nil
}

func (p *capsuleProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) ||
			containsAnyStr(strings.ToLower(strings.Join(m.Tags, " ")), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *capsuleProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// 2. geneProvider — 技能基因（来自 EvolutionEngine）

type geneProvider struct {
	root string
}

func (p *geneProvider) Type() ResourceType { return ResourceGenes }

func (p *geneProvider) List(scope string) ([]ResourceMeta, error) {
	engine := NewEvolutionEngine(p.root)
	genes := engine.SearchGenes("", "", nil)
	var result []ResourceMeta
	for _, g := range genes {
		s := g.Scope
		if s == "" {
			s = "global"
		}
		if scope != "" && s != scope {
			continue
		}
		result = append(result, ResourceMeta{
			ID:          g.ID,
			Type:        ResourceGenes,
			Scope:       s,
			Name:        g.Name,
			Description: g.Description,
			Tags:        g.Tags,
		})
	}
	return result, nil
}

func (p *geneProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.Name), query) ||
			strings.Contains(strings.ToLower(m.Description), query) ||
			containsAnyStr(strings.ToLower(strings.Join(m.Tags, " ")), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *geneProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// 3. memoryProvider — 项目记忆（.pair/memory/）

type memoryProvider struct {
	root string
}

func (p *memoryProvider) Type() ResourceType { return ResourceMemory }

func (p *memoryProvider) dir() string { return filepath.Join(p.root, ".pair", "memory") }

func (p *memoryProvider) List(scope string) ([]ResourceMeta, error) {
	dir := p.dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil // 目录不存在不是错误
	}
	var result []ResourceMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fi, _ := e.Info()
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		desc := frontmatterField(string(data), "description")
		mtype := frontmatterField(string(data), "type")
		name := strings.TrimSuffix(e.Name(), ".md")
		result = append(result, ResourceMeta{
			ID:          name,
			Type:        ResourceMemory,
			Scope:       "project", // 记忆总是项目级
			Name:        name,
			Description: truncateStr(desc, 100),
			Size:        fi.Size(),
			ModifiedAt:  fi.ModTime().Format(time.RFC3339),
			Tags:        splitTags(mtype),
		})
	}
	return result, nil
}

func (p *memoryProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *memoryProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// 4. projectInfoProvider — 项目知识库（.pair/project-info/）

type projectInfoProvider struct {
	root string
}

func (p *projectInfoProvider) Type() ResourceType { return ResourceProjectInfo }

func (p *projectInfoProvider) dir() string { return filepath.Join(p.root, ".pair", "project-info") }

func (p *projectInfoProvider) List(scope string) ([]ResourceMeta, error) {
	dir := p.dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var result []ResourceMeta
	p.scanDir(dir, "", &result, entries)
	return result, nil
}

func (p *projectInfoProvider) scanDir(dir, prefix string, result *[]ResourceMeta, entries []os.DirEntry) {
	for _, e := range entries {
		if e.IsDir() {
			sub, _ := os.ReadDir(filepath.Join(dir, e.Name()))
			p.scanDir(filepath.Join(dir, e.Name()), prefix+e.Name()+"/", result, sub)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fi, _ := e.Info()
		name := strings.TrimSuffix(e.Name(), ".md")
		fullPath := prefix + name
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		title := firstHeading(string(data), name)
		*result = append(*result, ResourceMeta{
			ID:          fullPath,
			Type:        ResourceProjectInfo,
			Scope:       "project",
			Name:        title,
			Description: truncateStr(string(data), 120),
			Size:        fi.Size(),
			ModifiedAt:  fi.ModTime().Format(time.RFC3339),
		})
	}
}

func (p *projectInfoProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Name), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *projectInfoProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// 5. skillsProvider — 技能（.pair/skills/<name>/SKILL.md）

type skillsProvider struct {
	root string
}

func (p *skillsProvider) Type() ResourceType { return ResourceSkills }

func (p *skillsProvider) skillsDir() string { return filepath.Join(p.root, ".pair", "skills") }

func (p *skillsProvider) List(scope string) ([]ResourceMeta, error) {
	baseDir := p.skillsDir()
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, nil
	}
	var result []ResourceMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := filepath.Join(baseDir, e.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		fi, _ := os.Stat(skillFile)
		content := string(data)
		desc := frontmatterField(content, "description")
		if desc == "" {
			desc = firstLine(content)
		}
		result = append(result, ResourceMeta{
			ID:          e.Name(),
			Type:        ResourceSkills,
			Scope:       "project", // Skills 始终项目级
			Name:        e.Name(),
			Description: truncateStr(desc, 100),
			Size:        fi.Size(),
			ModifiedAt:  fi.ModTime().Format(time.RFC3339),
		})
	}
	return result, nil
}

func (p *skillsProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *skillsProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// 6. mcpServerProvider — MCP 服务器（从 mcp.json 读取）

type mcpServerProvider struct{}

func (p *mcpServerProvider) Type() ResourceType { return ResourceMCPServers }

func (p *mcpServerProvider) List(scope string) ([]ResourceMeta, error) {
	// MCP 始终全局（用户级配置），返回简略信息
	cfgs := loadMCPConfigs()
	var result []ResourceMeta
	for _, cfg := range cfgs {
		result = append(result, ResourceMeta{
			ID:          cfg.Name,
			Type:        ResourceMCPServers,
			Scope:       "global",
			Name:        cfg.Name,
			Description: fmt.Sprintf("命令: %s %s", cfg.Command, strings.Join(cfg.Args, " ")),
		})
	}
	return result, nil
}

func (p *mcpServerProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *mcpServerProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// loadMCPConfigs 读取 mcp.json 配置（与 mcppanel 包解耦的内联实现）
func loadMCPConfigs() []struct {
	Name, Command string
	Args          []string
} {
	// 尝试从用户级和应用级配置加载
	type mcpEntry struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	type mcpFile struct {
		MCPservers []mcpEntry `json:"mcpServers"`
	}

	// 用户级: ~/.pair/mcp.json
	home, _ := os.UserHomeDir()
	paths := []string{}
	if home != "" {
		paths = append(paths, filepath.Join(home, ".pair", "mcp.json"))
	}
	// 应用级: InstallDir()/.pair/mcp.json
	paths = append(paths, filepath.Join(core.InstallDir(), ".pair", "mcp.json"))

	seen := map[string]bool{}
	var result []struct {
		Name, Command string
		Args          []string
	}
	for _, fp := range paths {
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		var f mcpFile
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		for _, s := range f.MCPservers {
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			result = append(result, struct {
				Name, Command string
				Args          []string
			}{s.Name, s.Command, s.Args})
		}
	}
	return result
}

// 7. luaToolProvider — Lua 自定义工具（.pair/tools/*.lua）

type luaToolProvider struct {
	root string
}

func (p *luaToolProvider) Type() ResourceType { return ResourceLuaTools }

func (p *luaToolProvider) toolsDir() string { return filepath.Join(p.root, ".pair", "tools") }

func (p *luaToolProvider) List(scope string) ([]ResourceMeta, error) {
	dir := p.toolsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var result []ResourceMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			continue
		}
		fi, _ := e.Info()
		name := strings.TrimSuffix(e.Name(), ".lua")
		data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		desc := extractLuaDesc(string(data))
		result = append(result, ResourceMeta{
			ID:          name,
			Type:        ResourceLuaTools,
			Scope:       "project",
			Name:        name,
			Description: desc,
			Size:        fi.Size(),
			ModifiedAt:  fi.ModTime().Format(time.RFC3339),
		})
	}
	return result, nil
}

func (p *luaToolProvider) Search(query, scope string) ([]ResourceMeta, error) {
	all, _ := p.List(scope)
	var matched []ResourceMeta
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.ID), query) ||
			strings.Contains(strings.ToLower(m.Description), query) {
			matched = append(matched, m)
		}
	}
	return matched, nil
}

func (p *luaToolProvider) Count(scope string) (int, error) {
	items, _ := p.List(scope)
	return len(items), nil
}

// extractLuaDesc 从 Lua 脚本源码中提取 description 字段。
func extractLuaDesc(src string) string {
	for _, line := range strings.Split(src, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description") || strings.HasPrefix(line, "-- description") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, "\"' `,")
				return truncateStr(val, 100)
			}
		}
	}
	return ""
}

// firstLine 取首行非空文本。
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return truncateStr(t, 100)
		}
	}
	return ""
}

// containsAnyStr 检查字符串是否包含任意子串（按空格分割）。
func containsAnyStr(s, substr string) bool {
	parts := strings.Fields(substr)
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// splitTags 按空格/逗号分割标签字符串。
func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
