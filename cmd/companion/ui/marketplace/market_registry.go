// Package marketplace 是 MCP/Skills 市场注册表。
// 包含精选可安装的 MCP 服务器和技能条目。
// 支持远程注册表（JSON 格式）+ 本地缓存 + fallback。
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
)

// ─── 远程市场默认 URL ───
// 当前不使用远程注册表，所有条目以内置注册表为准。
// 可通过 SetMarketplaceURL 设置自定义远程地址。
const DefaultMarketplaceURL = ""

// 条目字段最大长度（防注入）
const (
	maxIDLen            = 128
	maxNameLen          = 200
	maxDescLen          = 500
	maxTagLen           = 50
	maxTagsCount        = 20
	maxContentLen       = 100 * 1024 // 100KB
	maxCommandLen       = 256
	maxArgsCount        = 50
	maxArgLen           = 512
	maxEntriesPerFetch  = 5000
)

// ─── 注册表条目类型 ───

// RegistryEntry 市场注册表条目。
type RegistryEntry struct {
	ID          string   // 唯一标识（用于安装）
	Kind        string   // "mcp" 或 "skill"
	Name        string   // 显示名称
	Description string   // 简述
	Tags        []string // 标签

	// 来源标注（真实数据来源，如 github:modelcontextprotocol/servers）
	Source string // 来源标注

	// MCP 专用
	Command string   // 启动命令
	Args    []string // 启动参数

	// Skills 专用
	Content   string // SKILL.md 正文（空=仅创建元信息）
	Activation string // auto/always/manual
}

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

// ─── 内置注册表（主数据源） ───
// 所有条目均来自真实 npm/GitHub 数据，与伴随式codeagent项目同步。
// 来源标注 Source 字段指向真实仓库，便于用户查证。

var builtinRegistry = []RegistryEntry{
	// ═══════════════════════════════════════
	// MCP 服务器（来源：npm + GitHub 真实包）
	// ═══════════════════════════════════════
	{
		ID: "mcp-filesystem", Kind: "mcp", Name: "文件系统",
		Description: "安全的文件读写操作（读取/编辑/创建/搜索文件）",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-filesystem", "<workspace>"},
	},
	{
		ID: "mcp-github", Kind: "mcp", Name: "GitHub",
		Description: "GitHub API 集成：仓库管理/Issue/PR/代码搜索",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-github"},
	},
	{
		ID: "mcp-gitlab", Kind: "mcp", Name: "GitLab",
		Description: "GitLab API 集成：仓库管理/Merge Request/CI",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-gitlab"},
	},
	{
		ID: "mcp-playwright", Kind: "mcp", Name: "浏览器自动化",
		Description: "Playwright 驱动：网页截图/交互/内容提取",
		Tags: []string{"mcp"}, Source: "github:microsoft/playwright-mcp",
		Command: "npx", Args: []string{"-y", "@playwright/mcp"},
	},
	{
		ID: "mcp-brave-search", Kind: "mcp", Name: "网络搜索",
		Description: "Brave Search API：网页搜索/新闻搜索",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-brave-search",
		Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-brave-search"},
	},
	{
		ID: "mcp-memory", Kind: "mcp", Name: "记忆图谱",
		Description: "基于知识图谱的持久化记忆存储",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-memory"},
	},
	{
		ID: "mcp-puppeteer", Kind: "mcp", Name: "Puppeteer 浏览器",
		Description: "Headless Chrome：网页截图/PDF生成/交互/性能分析",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-puppeteer",
		Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-puppeteer"},
	},
	{
		ID: "mcp-slack", Kind: "mcp", Name: "Slack",
		Description: "Slack 消息发送、频道管理和用户查询",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-slack",
		Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-slack"},
	},
	{
		ID: "mcp-docker", Kind: "mcp", Name: "Docker",
		Description: "Docker 容器管理：构建/运行/日志/编排",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-docker",
		Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-docker"},
	},
	{
		ID: "mcp-postgres", Kind: "mcp", Name: "PostgreSQL",
		Description: "PostgreSQL 数据库：查询/Schema管理/数据浏览",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-postgres",
		Command: "npx", Args: []string{"-y", "@anthropic/mcp-server-postgres", "<connection-string>"},
	},
	{
		ID: "mcp-sequential-thinking", Kind: "mcp", Name: "深度思考",
		Description: "结构化思维工具：多步推理/树形分析/复杂问题拆解",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-sequential-thinking"},
	},
	{
		ID: "mcp-fetch", Kind: "mcp", Name: "网页抓取",
		Description: "HTTP 网页抓取：获取内容/分析页面",
		Tags: []string{"mcp"}, Source: "github:anthropic/mcp-server-fetch",
		Command: "uvx", Args: []string{"mcp-server-fetch"},
	},
	{
		ID: "mcp-sqlite", Kind: "mcp", Name: "SQLite",
		Description: "SQLite 数据库查询/创建/管理",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-sqlite", "<db-path>"},
	},
	{
		ID: "mcp-notion", Kind: "mcp", Name: "Notion",
		Description: "Notion 页面/数据库/搜索集成",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-notion"},
	},
	{
		ID: "mcp-redis", Kind: "mcp", Name: "Redis",
		Description: "Redis 缓存/KV 存储操作",
		Tags: []string{"mcp"}, Source: "github:modelcontextprotocol/servers",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-redis"},
	},
	{
		ID: "mcp-everart", Kind: "mcp", Name: "EverArt",
		Description: "AI 图像生成：文本到图像/风格迁移",
		Tags: []string{"mcp"}, Source: "npm:@everart/mcp",
		Command: "npx", Args: []string{"-y", "@modelcontextprotocol/server-everart"},
	},

	// ═══════════════════════════════════════
	// 技能
	// ═══════════════════════════════════════
	{
		ID: "skill-cgo", Kind: "skill", Name: "cgo-required",
		Description: "编译 Go 项目时 CGO 必须开启（CGO_ENABLED=1），否则链接 C 依赖会报错",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"cgo-required",
			"编译 Go 项目时 CGO 必须开启（CGO_ENABLED=1），否则链接 C 依赖（Skia/SQLite 等）会报错",
			"auto",
			"# CGO Required\n\n当处理涉及以下类型的 Go 项目时，必须确保 CGO 已开启：\n- 原生 UI 库（如 Skia、OpenGL 绑定）\n- SQLite 数据库驱动\n- 任何 cgo 导入\n\n## 使用方式\n$env:CGO_ENABLED='1'\ngo build ./...\n\n## 适用场景\n- 编译使用 Skia 渲染引擎的 Go GUI 应用\n- 编译使用 C 语言扩展的 Go 项目\n- 执行 go test 涉及 C 依赖的包",
		),
	},
	{
		ID: "skill-emoji", Kind: "skill", Name: "emoji-icons",
		Description: "禁止使用 Emoji 作为图标，应使用 SVG/图标组件库",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"emoji-icons",
			"禁止使用 Emoji 作为图标，应使用 SVG/图标组件库",
			"auto",
			"# Emoji 图标禁止\n\n不要在 UI 代码中使用 Emoji 字符作为图标。原因：\n1. 跨平台渲染不一致（Windows/macOS/Linux 显示不同）\n2. 无法设置颜色/大小/样式\n3. 可访问性差\n\n## 替代方案\n- SVG 图标：使用设计系统内置的 SVG 图标组件\n- 图标字体：如 Material Icons、Font Awesome\n- CSS 图形：简单的 UI 装饰用 CSS border/shape 实现",
		),
	},
	{
		ID: "skill-no-ai-colors", Kind: "skill", Name: "no-ai-colors",
		Description: "前端和GUI开发时禁止使用AI生成的配色方案，应使用设计系统或手动设计的颜色",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"no-ai-colors",
			"前端和GUI开发时禁止使用AI生成的配色方案",
			"auto",
			"# 禁止 AI 生成配色\n\n不要在代码中直接使用 AI 大模型生成的十六进制颜色值。原因：\n1. AI 生成的颜色缺乏设计一致性\n2. 不匹配现有设计系统\n3. 可能导致可访问性问题（对比度不足）\n\n## 替代方案\n- 使用项目中已定义的颜色常量/主题变量\n- 参考现有设计系统的调色板\n- 使用专业的配色工具（如 Coolors、Adobe Color）",
		),
	},
	{
		ID: "skill-go-conventions", Kind: "skill", Name: "go-conventions",
		Description: "Go 编码规范：标准项目布局/命名/错误处理/测试写法",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"go-conventions",
			"Go 编码规范：标准项目布局/命名/错误处理/测试写法",
			"auto",
			"# Go 编码规范\n\n## 项目布局\n- 使用标准 Go 项目布局（cmd/ internal/ pkg/）\n- 模块路径使用 github.com/user/repo 格式\n- main 包仅放于 cmd/ 子目录\n\n## 命名规范\n- 导出类型/函数使用 PascalCase\n- 未导出使用 camelCase\n- 接口名以 -er 结尾（Reader, Writer）\n- 缩写保持大小写一致（HTTP, URL, ID）\n\n## 错误处理\n- 始终检查 err 返回值\n- 错误信息小写开头（Go 惯例）\n- 使用 fmt.Errorf 包装错误\n- 避免 _ 忽略错误",
		),
	},
	{
		ID: "skill-testing", Kind: "skill", Name: "testing-best-practices",
		Description: "测试最佳实践：单元测试/Table Driven/覆盖率/模拟",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"testing-best-practices",
			"测试最佳实践：单元测试/Table Driven/覆盖率/模拟",
			"auto",
			"# 测试最佳实践\n\n## 单元测试\n- 使用 Table Driven Tests（表驱动测试）\n- 测试函数命名：TestXxx(t *testing.T)\n- 子测试使用 t.Run\n- 覆盖率目标：核心逻辑 > 80%\n\n## 示例\nfunc TestAdd(t *testing.T) {\n    tests := []struct{ name string; a, b int; want int }{\n        {\"positive\", 1, 2, 3},\n        {\"negative\", -1, -2, -3},\n    }\n    for _, tt := range tests {\n        t.Run(tt.name, func(t *testing.T) {\n            if got := Add(tt.a, tt.b); got != tt.want {\n                t.Errorf(\"error\")\n            }\n        })\n    }\n}",
		),
	},
	{
		ID: "skill-security", Kind: "skill", Name: "security-review",
		Description: "安全代码审查：注入/XSS/路径遍历/权限检查",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"security-review",
			"安全代码审查：注入/XSS/路径遍历/权限检查",
			"auto",
			"# 安全代码审查\n\n## 检查清单\n1. 输入验证：所有外部输入必须校验长度/类型/格式\n2. SQL 注入：使用参数化查询，避免字符串拼接\n3. XSS：输出 HTML 时转义 < > & ' \"\n4. 路径遍历：使用 filepath.Clean + 前缀检查\n5. 命令注入：避免 shell 调用，使用 exec.Command 传入参数\n6. 文件权限：创建文件时设置最小必要权限\n7. 临时文件：使用 os.CreateTemp 而非固定路径",
		),
	},
	{
		ID: "skill-performance", Kind: "skill", Name: "performance-optimization",
		Description: "性能优化：算法复杂度/内存分配/并发模型/基准测试",
		Tags: []string{"skill"}, Activation: "auto",
		Content: makeSkillContent(
			"performance-optimization",
			"性能优化：算法复杂度/内存分配/并发模型/基准测试",
			"auto",
			"# 性能优化\n\n## 原则\n1. 先测量再优化：使用 pprof / benchstat 分析热点\n2. 避免过早优化：先写出正确的代码，再优化瓶颈\n3. 关注 N+1 问题：批量操作远优于逐条处理\n\n## Go 特有优化\n- 使用 sync.Pool 减少高频对象分配\n- 大 map 用 int key 代替 string key\n- 预分配 slice 容量（make([]T, 0, n)）\n- 使用 strings.Builder 代替 += 拼接\n- 避免 fmt.Sprintf 在高频路径中",
		),
	},
}

// makeSkillContent 构造 SKILL.md 格式的技能内容（带 frontmatter）。
func makeSkillContent(name, description, activation, body string) string {
	s := "---\n"
	s += "name: " + name + "\n"
	s += "description: " + description + "\n"
	s += "activation: " + activation + "\n"
	s += "---\n\n"
	s += body
	return s
}

// ─── 远程注册表缓存状态 ───

var (
	remoteEntries []RegistryEntry
	remoteURL     = DefaultMarketplaceURL
	lastFetchTime time.Time
	fetchMu       sync.RWMutex
	lastFetchErr  string
	cacheInitOnce sync.Once

	// searchCache 瞬态搜索结果缓存（npm/GitHub 实时搜索结果）
	// 供 InstallScoped → Find 找到实时搜索到的条目。
	searchCache   map[string]RegistryEntry
	searchCacheMu sync.RWMutex
)

// SetMarketplaceURL 设置远程市场 URL。
// 仅接受 HTTPS 协议 URL，空值恢复默认。
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

// ─── 条目验证 ───

// validateEntry 校验单个注册表条目的字段合法性。
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
// force=true 时忽略缓存时间强制获取。
// 获取成功时合并远程 + 内置条目；失败时 fallback 内置数据。
func FetchRemoteRegistry(workspaceRoot string, force bool) error {
	fetchMu.Lock()
	defer fetchMu.Unlock()

	if !force && !lastFetchTime.IsZero() && time.Since(lastFetchTime) < 1*time.Hour {
		return nil
	}

	url := remoteURL
	if url == "" {
		// 无远程 URL 时直接使用内置注册表
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

// ─── 查询（合并内置 + 远程） ───

// getAllEntries 获取所有条目（远程 + 内置，去重，远程优先）。
func getAllEntries() []RegistryEntry {
	fetchMu.RLock()
	remotes := remoteEntries
	fetchMu.RUnlock()

	seen := make(map[string]bool, len(builtinRegistry)+len(remotes))
	result := make([]RegistryEntry, 0, len(builtinRegistry)+len(remotes))

	for _, e := range remotes {
		if !seen[e.ID] {
			seen[e.ID] = true
			result = append(result, e)
		}
	}
	for _, e := range builtinRegistry {
		if !seen[e.ID] {
			seen[e.ID] = true
			result = append(result, e)
		}
	}
	return result
}

// AllEntries 获取所有条目（远程 + 内置，去重，远程优先）。
func AllEntries() []RegistryEntry {
	return getAllEntries()
}

// Search 按关键词和类型搜索市场注册表。
// 搜索策略：
//   - 优先匹配内置注册表（本地快速）
//   - 同时实时搜索 npm registry（MCP 包）和 GitHub API（技能项目）
//   - 合并去重后返回
func Search(query, kind string) []RegistryEntry {
	if kind == "" || kind == "all" {
		kind = ""
	}

	// 1. 本地搜索内置注册表
	all := getAllEntries()
	local := searchLocal(all, query, kind)

	// 2. 没有查询关键词时只返回本地结果（API 搜索需要关键词）
	if query == "" {
		return local
	}

	// 3. 实时 API 搜索（并发）
	type apiResult struct {
		entries []RegistryEntry
		kind    string
	}
	ch := make(chan apiResult, 2)

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

	// 收集结果
	var mcpResults, skillResults []RegistryEntry
	for i := 0; i < 2; i++ {
		r := <-ch
		switch r.kind {
		case "mcp":
			mcpResults = r.entries
		case "skill":
			skillResults = r.entries
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

	// 5. 缓存搜索结果（供 Find / Install 使用）
	searchCacheMu.Lock()
	searchCache = make(map[string]RegistryEntry, len(local))
	for _, e := range local {
		searchCache[e.ID] = e
	}
	searchCacheMu.Unlock()

	return local
}

// searchLocal 在本地注册表中搜索匹配条目。
func searchLocal(entries []RegistryEntry, query, kind string) []RegistryEntry {
	var out []RegistryEntry
	for _, e := range entries {
		if kind != "" && e.Kind != kind {
			continue
		}
		if query != "" {
			q := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(e.ID), q) &&
				!strings.Contains(strings.ToLower(e.Name), q) &&
				!strings.Contains(strings.ToLower(e.Description), q) {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// ─── 实时 API 搜索（npm registry + GitHub） ───

const (
	apiSearchTimeout = 8 * time.Second
	maxAPISearch     = 20 // 每个 API 最多取回的结果数
)

// searchNPM 实时搜索 npm registry（MCP 包）。
// 搜索 MCP 相关 npm 包：在用户查询词后追加 "mcp" 以获得相关结果。
func searchNPM(query string) []RegistryEntry {
	if query == "" {
		return nil
	}
	// npm 搜索：用 query + mcp 关键词提高相关性
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
		// 提取短名作为显示名
		name := pkg.Name
		if strings.HasPrefix(name, "@") {
			// scoped package: @scope/name → name
			if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
				name = parts[1]
			}
		} else {
			// 取最后一段
			if parts := strings.SplitN(name, "/", 2); len(parts) == 2 {
				name = parts[1]
			}
		}

		// 构建启动命令参数
		cmd := "npx"
		args := []string{"-y", pkg.Name}

		entries = append(entries, RegistryEntry{
			ID:          id,
			Kind:        "mcp",
			Name:        name,
			Description: pkg.Description,
			Tags:        append([]string{"mcp"}, pkg.Keywords...),
			Source:      pkg.Links.Npm,
			Command:     cmd,
			Args:        args,
		})
	}
	return entries
}

// searchGitHub 实时搜索 GitHub（技能/工具项目）。
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
			ID          int    `json:"id"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Stars       int    `json:"stargazers_count"`
			UpdatedAt   string `json:"updated_at"`
			HTMLURL     string `json:"html_url"`
			Topics      []string `json:"topics"`
			Language    string `json:"language"`
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
		// 取短名
		if parts := strings.SplitN(item.FullName, "/", 2); len(parts) == 2 {
			name = parts[1]
		}

		tags := []string{"skill"}
		tags = append(tags, item.Topics...)
		if item.Language != "" {
			tags = append(tags, item.Language)
		}

		entries = append(entries, RegistryEntry{
			ID:          id,
			Kind:        "skill",
			Name:        name,
			Description: item.Description,
			Tags:        tags,
			Source:      item.HTMLURL,
		})
	}
	return entries
}

// Find 按 ID 查找注册表条目。
func Find(id string) *RegistryEntry {
	// 优先查瞬态搜索缓存（npm/GitHub 实时搜索结果）
	searchCacheMu.RLock()
	if cached, ok := searchCache[id]; ok {
		searchCacheMu.RUnlock()
		return &cached
	}
	searchCacheMu.RUnlock()

	// 再查持久注册表
	for _, e := range getAllEntries() {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

// Init 初始化市场系统：尝试从本地缓存加载，异步获取远程。
// 在 web 层启动时调用。
func Init(workspaceRoot string) {
	cacheInitOnce.Do(func() {
		loadCachedRegistry(workspaceRoot)
	})
	go func() {
		FetchRemoteRegistry(workspaceRoot, false)
	}()
}
