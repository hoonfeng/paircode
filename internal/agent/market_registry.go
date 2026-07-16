// 市场注册表 —— 自闭环：内置精选可安装的 MCP 服务器和技能条目。
// 从 cmd/companion/ui/marketplace/market_registry.go 迁移而来，仅保留内置注册表常量数据与本地搜索。
// 远程 API 搜索（npm/GitHub）、缓存、校验等 UI 层功能留在原处。
// 无 //go:build 标签，全平台可用。

package agent

import (
	"strings"
)

// ─── 注册表条目类型 ───

// MarketEntry 市场注册表条目。
type MarketEntry struct {
	ID          string   // 唯一标识（用于安装）
	Kind        string   // "mcp" 或 "skill"
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

// ─── 内置注册表（主数据源） ───
// 所有条目均来自真实 npm/GitHub 数据。
// 来源标注 Source 字段指向真实仓库，便于用户查证。

// BuiltinRegistry 完整的内置条目列表。
var BuiltinRegistry = []MarketEntry{
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
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
		Content: makeMarketSkillContent(
			"performance-optimization",
			"性能优化：算法复杂度/内存分配/并发模型/基准测试",
			"auto",
			"# 性能优化\n\n## 原则\n1. 先测量再优化：使用 pprof / benchstat 分析热点\n2. 避免过早优化：先写出正确的代码，再优化瓶颈\n3. 关注 N+1 问题：批量操作远优于逐条处理\n\n## Go 特有优化\n- 使用 sync.Pool 减少高频对象分配\n- 大 map 用 int key 代替 string key\n- 预分配 slice 容量（make([]T, 0, n)）\n- 使用 strings.Builder 代替 += 拼接\n- 避免 fmt.Sprintf 在高频路径中",
		),
	},
}

// makeMarketSkillContent 构造 SKILL.md 格式的技能内容（带 frontmatter）。
func makeMarketSkillContent(name, description, activation, body string) string {
	s := "---\n"
	s += "name: " + name + "\n"
	s += "description: " + description + "\n"
	s += "activation: " + activation + "\n"
	s += "---\n\n"
	s += body
	return s
}

// ─── 公开查询 API ───

// MarketAllEntries 获取所有内置条目。
func MarketAllEntries() []MarketEntry {
	out := make([]MarketEntry, len(BuiltinRegistry))
	copy(out, BuiltinRegistry)
	return out
}

// MarketSearch 在本地内置注册表中搜索匹配条目（不包含远程 API 搜索）。
// kind 可空/""=全部，"mcp"=仅 MCP，"skill"=仅技能。
// query 可空=返回全部。
func MarketSearch(query, kind string) []MarketEntry {
	var out []MarketEntry
	for _, e := range BuiltinRegistry {
		if kind != "" && kind != "all" && e.Kind != kind {
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

// MarketFind 按 ID 在本地内置注册表中查找条目。
// 不存在时返回 nil。
func MarketFind(id string) *MarketEntry {
	for _, e := range BuiltinRegistry {
		if e.ID == id {
			return &e
		}
	}
	return nil
}
