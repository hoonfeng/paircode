// Package marketplace 是 MCP/Skills 市场注册表。
// 包含精选可安装的 MCP 服务器和技能条目。
//
//go:build windows

package marketplace

// ─── 注册表条目类型 ───

// RegistryEntry 市场注册表条目。
type RegistryEntry struct {
	ID          string   // 唯一标识（用于安装）
	Kind        string   // "mcp" 或 "skill"
	Name        string   // 显示名称
	Description string   // 简述
	Tags        []string // 标签

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

// ─── 市场注册表 ───

// Registry 所有可安装的市场条目。
var Registry = []RegistryEntry{
	// ═══════════════════════════════════════
	// MCP 服务器
	// ═══════════════════════════════════════
	RegistryMCP("mcp-filesystem", "文件系统", "安全的文件读写操作（读取/编辑/创建/搜索文件）", "npx", []string{"-y", "@modelcontextprotocol/server-filesystem", "<workspace>"}),
	RegistryMCP("mcp-github", "GitHub", "GitHub API 集成：仓库管理/Issue/PR/代码搜索", "npx", []string{"-y", "@modelcontextprotocol/server-github"}),
	RegistryMCP("mcp-gitlab", "GitLab", "GitLab API 集成：仓库管理/Merge Request/CI", "npx", []string{"-y", "@modelcontextprotocol/server-gitlab"}),
	RegistryMCP("mcp-playwright", "浏览器自动化", "Playwright 驱动：网页截图/交互/内容提取", "npx", []string{"-y", "@playwright/mcp"}),
	RegistryMCP("mcp-brave-search", "网络搜索", "Brave Search API：网页搜索/新闻搜索", "npx", []string{"-y", "@modelcontextprotocol/server-brave-search"}),
	RegistryMCP("mcp-sqlite", "SQLite", "SQLite 数据库查询/创建/管理", "npx", []string{"-y", "@modelcontextprotocol/server-sqlite", "<db-path>"}),
	RegistryMCP("mcp-puppeteer", "Puppeteer 浏览器", "Headless Chrome：网页截图/PDF/控制台", "npx", []string{"-y", "@modelcontextprotocol/server-puppeteer"}),
	RegistryMCP("mcp-memory", "记忆图谱", "知识图谱存储：实体/关系记忆持久化", "npx", []string{"-y", "@modelcontextprotocol/server-memory"}),
	RegistryMCP("mcp-slack", "Slack", "Slack 消息/频道/用户管理集成", "npx", []string{"-y", "@modelcontextprotocol/server-slack"}),
	RegistryMCP("mcp-notion", "Notion", "Notion 页面/数据库/搜索集成", "npx", []string{"-y", "@modelcontextprotocol/server-notion"}),
	RegistryMCP("mcp-jira", "Jira", "Atlassian Jira：Issue/项目/工作流管理", "npx", []string{"-y", "@modelcontextprotocol/server-jira"}),
	RegistryMCP("mcp-docker", "Docker", "Docker 容器/镜像/日志管理", "npx", []string{"-y", "@modelcontextprotocol/server-docker"}),
	RegistryMCP("mcp-postgres", "PostgreSQL", "PostgreSQL 数据库查询/Schema 管理", "npx", []string{"-y", "@modelcontextprotocol/server-postgres", "<connection-string>"}),
	RegistryMCP("mcp-redis", "Redis", "Redis 缓存/KV 存储操作", "npx", []string{"-y", "@modelcontextprotocol/server-redis"}),
	RegistryMCP("mcp-aws-s3", "AWS S3", "Amazon S3 存储桶/对象操作", "npx", []string{"-y", "@modelcontextprotocol/server-aws-s3"}),
	RegistryMCP("mcp-sequential-thinking", "深度思考", "结构化思维工具：多步推理/树形分析", "npx", []string{"-y", "@modelcontextprotocol/server-sequential-thinking"}),
	RegistryMCP("mcp-everart", "EverArt", "AI 图像生成：文本到图像/风格迁移", "npx", []string{"-y", "@modelcontextprotocol/server-everart"}),
	RegistryMCP("mcp-fetch", "网页抓取", "HTTP 网页抓取：获取内容/分析页面", "uvx", []string{"mcp-server-fetch"}),

	// ═══════════════════════════════════════
	// 技能
	// ═══════════════════════════════════════
	RegistrySkill("skill-cgo", "cgo-required", "编译 Go 项目时 CGO 必须开启（CGO_ENABLED=1），否则链接 C 依赖会报错", "auto", makeSkillContent(
		"cgo-required",
		"编译 Go 项目时 CGO 必须开启（CGO_ENABLED=1），否则链接 C 依赖（Skia/SQLite 等）会报错",
		"auto",
		"# CGO Required\n\n当处理涉及以下类型的 Go 项目时，必须确保 CGO 已开启：\n- 原生 UI 库（如 Skia、OpenGL 绑定）\n- SQLite 数据库驱动\n- 任何 cgo 导入\n\n## 使用方式\n$env:CGO_ENABLED='1'\ngo build ./...\n\n## 适用场景\n- 编译使用 Skia 渲染引擎的 Go GUI 应用\n- 编译使用 C 语言扩展的 Go 项目\n- 执行 go test 涉及 C 依赖的包",
	)),
	RegistrySkill("skill-emoji", "emoji-icons", "禁止使用 Emoji 作为图标，应使用 SVG/图标组件库", "auto", makeSkillContent(
		"emoji-icons",
		"禁止使用 Emoji 作为图标，应使用 SVG/图标组件库",
		"auto",
		"# Emoji 图标禁止\n\n不要在 UI 代码中使用 Emoji 字符作为图标。原因：\n1. 跨平台渲染不一致（Windows/macOS/Linux 显示不同）\n2. 无法设置颜色/大小/样式\n3. 可访问性差\n\n## 替代方案\n- SVG 图标：使用设计系统内置的 SVG 图标组件\n- 图标字体：如 Material Icons、Font Awesome\n- CSS 图形：简单的 UI 装饰用 CSS border/shape 实现",
	)),
	RegistrySkill("skill-no-ai-colors", "no-ai-colors", "前端和GUI开发时禁止使用AI生成的配色方案，应使用设计系统或手动设计的颜色", "auto", makeSkillContent(
		"no-ai-colors",
		"前端和GUI开发时禁止使用AI生成的配色方案",
		"auto",
		"# 禁止 AI 生成配色\n\n不要在代码中直接使用 AI 大模型生成的十六进制颜色值。原因：\n1. AI 生成的颜色缺乏设计一致性\n2. 不匹配现有设计系统\n3. 可能导致可访问性问题（对比度不足）\n\n## 替代方案\n- 使用项目中已定义的颜色常量/主题变量\n- 参考现有设计系统的调色板\n- 使用专业的配色工具（如 Coolors、Adobe Color）",
	)),
	RegistrySkill("skill-go-conventions", "go-conventions", "Go 编码规范：标准项目布局/命名/错误处理/测试写法", "auto", makeSkillContent(
		"go-conventions",
		"Go 编码规范：标准项目布局/命名/错误处理/测试写法",
		"auto",
		"# Go 编码规范\n\n## 项目布局\n- 使用标准 Go 项目布局（cmd/ internal/ pkg/）\n- 模块路径使用 github.com/user/repo 格式\n- main 包仅放于 cmd/ 子目录\n\n## 命名规范\n- 导出类型/函数使用 PascalCase\n- 未导出使用 camelCase\n- 接口名以 -er 结尾（Reader, Writer）\n- 缩写保持大小写一致（HTTP, URL, ID）\n\n## 错误处理\n- 始终检查 err 返回值\n- 错误信息小写开头（Go 惯例）\n- 使用 fmt.Errorf 包装错误\n- 避免 _ 忽略错误",
	)),
	RegistrySkill("skill-testing", "testing-best-practices", "测试最佳实践：单元测试/Table Driven/覆盖率/模拟", "auto", makeSkillContent(
		"testing-best-practices",
		"测试最佳实践：单元测试/Table Driven/覆盖率/模拟",
		"auto",
		"# 测试最佳实践\n\n## 单元测试\n- 使用 Table Driven Tests（表驱动测试）\n- 测试函数命名：TestXxx(t *testing.T)\n- 子测试使用 t.Run\n- 覆盖率目标：核心逻辑 > 80%\n\n## 示例\nfunc TestAdd(t *testing.T) {\n    tests := []struct{ name string; a, b int; want int }{\n        {\"positive\", 1, 2, 3},\n        {\"negative\", -1, -2, -3},\n    }\n    for _, tt := range tests {\n        t.Run(tt.name, func(t *testing.T) {\n            if got := Add(tt.a, tt.b); got != tt.want {\n                t.Errorf(\"error\")\n            }\n        })\n    }\n}",
	)),
	RegistrySkill("skill-security", "security-review", "安全代码审查：注入/XSS/路径遍历/权限检查", "auto", makeSkillContent(
		"security-review",
		"安全代码审查：注入/XSS/路径遍历/权限检查",
		"auto",
		"# 安全代码审查\n\n## 检查清单\n1. 输入验证：所有外部输入必须校验长度/类型/格式\n2. SQL 注入：使用参数化查询，避免字符串拼接\n3. XSS：输出 HTML 时转义 < > & ' \"\n4. 路径遍历：使用 filepath.Clean + 前缀检查\n5. 命令注入：避免 shell 调用，使用 exec.Command 传入参数\n6. 文件权限：创建文件时设置最小必要权限\n7. 临时文件：使用 os.CreateTemp 而非固定路径",
	)),
	RegistrySkill("skill-performance", "performance-optimization", "性能优化：算法复杂度/内存分配/并发模型/基准测试", "auto", makeSkillContent(
		"performance-optimization",
		"性能优化：算法复杂度/内存分配/并发模型/基准测试",
		"auto",
		"# 性能优化\n\n## 原则\n1. 先测量再优化：使用 pprof / benchstat 分析热点\n2. 避免过早优化：先写出正确的代码，再优化瓶颈\n3. 关注 N+1 问题：批量操作远优于逐条处理\n\n## Go 特有优化\n- 使用 sync.Pool 减少高频对象分配\n- 大 map 用 int key 代替 string key\n- 预分配 slice 容量（make([]T, 0, n)）\n- 使用 strings.Builder 代替 += 拼接\n- 避免 fmt.Sprintf 在高频路径中",
	)),
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

// ─── 查询函数 ───

// Search 按关键词和类型搜索市场注册表。
func Search(query, kind string) []RegistryEntry {
	if kind == "" || kind == "all" {
		kind = ""
	}
	var out []RegistryEntry
	for _, e := range Registry {
		if kind != "" && e.Kind != kind {
			continue
		}
		if query != "" {
			q := stringsToLower(query)
			if !contains(stringsToLower(e.ID), q) &&
				!contains(stringsToLower(e.Name), q) &&
				!contains(stringsToLower(e.Description), q) {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// Find 按 ID 查找注册表条目。
func Find(id string) *RegistryEntry {
	for _, e := range Registry {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

// contains returns true if substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || inString(s, substr))
}

func inString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringsToLower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}
