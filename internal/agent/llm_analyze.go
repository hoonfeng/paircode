// llm_analyze.go — 工具集 LLM 项目意图分析：让 agent 理解「项目实际要实现
// 的目的」（而非仅静态依赖特征），据此推荐工具类别组合工具集。
//
// 通用性：不假设任何语言/框架——输入是语言无关的项目轻量上下文
// （README 摘要 + 文件树 + 静态特征），输出是语言无关的工具类别推荐
// （build/test/git/lint/api/data/...），模板按意图标签强制命中。
//
// 降级：LLM 不可用（未配置/超时/解析失败）时自动回退纯静态分析，不影响主流程。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ToolsetIntentTags 可推荐的工具类别（语言无关；LLM 只能从其中选择）。
// 模板 Tags 使用同一词汇表（toolset_templates.go 内置模板），意图匹配即标签相等。
var ToolsetIntentTags = []string{
	"build", "test", "run", "git", "lint", "api", "http",
	"data", "database", "docs", "docker", "debug", "deploy", "ci",
}

// ProjectIntent LLM 项目意图分析结论（结构化输出，强制 JSON）。
type ProjectIntent struct {
	Purpose         string         `json:"purpose"`         // 项目目的（一句话，作为工具集描述）
	BuildCmd        string         `json:"buildCmd"`        // 项目真实构建命令（如 npm run build / make / cargo build；未知留空）
	TestCmd         string         `json:"testCmd"`         // 测试命令
	RunCmd          string         `json:"runCmd"`          // 运行/启动命令
	LintCmd         string         `json:"lintCmd"`         // lint/检查命令（如 npx eslint . / ruff check .；无则空）
	FormatCmd       string         `json:"formatCmd"`       // 格式化命令（无则空）
	RecommendedTags []string       `json:"recommendedTags"` // 推荐工具类别（ToolsetIntentTags 子集）
	Notes           string         `json:"notes"`           // 补充说明（透传模板 generate 裁剪插件）
	CustomPlugins   []CustomPlugin `json:"customPlugins"`   // LLM 现场生成的项目专属插件（模板覆盖不到的能力缺口）
}

// CustomPlugin LLM 现场生成的项目专属插件：工具集模板组合覆盖不到的能力缺口
// （如 OpenAPI 校验、Protobuf 编译、数据库迁移等特殊栈），由 LLM 分析项目后
// 直接写出插件代码并入工具集（对齐 deepseek-harness「模型所写插件」模式：
// 注册时即校验——BuildToolset 会对 code 做 define 预检，失败剔除并给指导性错误）。
type CustomPlugin struct {
	Name    string `json:"name"`    // 插件名（小写字母/数字/-/_）
	Purpose string `json:"purpose"` // 用途一句话（工具集清单展示）
	Code    string `json:"code"`    // 插件代码：return { name, inject:[...], apply(ctx){...} }（纯 JS）
}

// applyToProfile 把 LLM 分析出的命令合入项目特征（生成器据此生成精确工具；
// LLM 未给出的命令保持原值，生成器对空命令跳过对应工具）。
func (it *ProjectIntent) applyToProfile(p *ProjectProfile) {
	if it == nil || p == nil {
		return
	}
	if strings.TrimSpace(it.BuildCmd) != "" {
		p.BuildCmd = strings.TrimSpace(it.BuildCmd)
	}
	if strings.TrimSpace(it.TestCmd) != "" {
		p.TestCmd = strings.TrimSpace(it.TestCmd)
	}
	if strings.TrimSpace(it.RunCmd) != "" {
		p.RunCmd = strings.TrimSpace(it.RunCmd)
	}
	if strings.TrimSpace(it.LintCmd) != "" {
		p.LintCmd = strings.TrimSpace(it.LintCmd)
	}
	if strings.TrimSpace(it.FormatCmd) != "" {
		p.FormatCmd = strings.TrimSpace(it.FormatCmd)
	}
}

// toolsetLLMProvider 返回 LLM provider（工具集分析用）；nil 表示不可用（跳过分析）。
// 包级变量便于测试替换（MockProvider 注入）。
var toolsetLLMProvider = func() Provider {
	// ★ 配置消费插件化：统一经装配点解析（存储基线 → 插件装配器覆盖）。
	cur := ResolveProviderParams()
	if cur.APIKey == "" || cur.BaseURL == "" {
		return nil
	}
	// ★ t1 S1：实现级插件槽位——插件注册的 Provider 实现（ctx.provider.register）
	//   对工具集 LLM 分析同样生效；未注册回退 OpenAI 兼容。
	cur.Temperature = 0.2
	cur.MaxTokens = 1024
	return CreateProvider(cur)
}

// llmAnalyzeProject 调用 LLM 理解项目目的并推荐工具类别。
// 返回解析后的 ProjectIntent；LLM 不可用/失败时返回 error（调用方回退静态）。
func llmAnalyzeProject(ctx context.Context, prov Provider, projectDir string, p *ProjectProfile, requirement string) (*ProjectIntent, error) {
	if prov == nil {
		return nil, fmt.Errorf("无 LLM provider")
	}
	prompt := buildIntentPrompt(projectDir, p, requirement)
	resp, err := prov.Chat(ctx, []Message{{Role: RoleUser, Content: prompt}}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}
	return parseProjectIntent(resp.Content)
}

// parseProjectIntent 解析 LLM 输出为 ProjectIntent（容忍 ```json 围栏与前后噪声）。
func parseProjectIntent(content string) (*ProjectIntent, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	// 只截取第一个 { 到最后一个 }（容忍模型输出多余文本）
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("LLM 输出不含 JSON 对象: %s", truncateStr(content, 120))
	}
	var intent ProjectIntent
	if err := json.Unmarshal([]byte(content[start:end+1]), &intent); err != nil {
		return nil, fmt.Errorf("LLM 输出 JSON 解析失败: %v", err)
	}
	if strings.TrimSpace(intent.Purpose) == "" {
		return nil, fmt.Errorf("LLM 未给出 purpose")
	}
	// 清洗推荐类别：只保留词汇表内标签（去重、去未知项）
	seen := map[string]bool{}
	var tags []string
	for _, t := range intent.RecommendedTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		if containsTag(ToolsetIntentTags, t) {
			tags = append(tags, t)
		}
	}
	intent.RecommendedTags = tags
	// 清洗 customPlugins：name 规范化、code 非空、去重（保留第一个）
	seenP := map[string]bool{}
	var cps []CustomPlugin
	for _, cp := range intent.CustomPlugins {
		if strings.TrimSpace(cp.Code) == "" {
			continue
		}
		name := pluginSafeName(cp.Name)
		if name == "" || seenP[name] {
			continue
		}
		seenP[name] = true
		cps = append(cps, CustomPlugin{Name: name, Purpose: strings.TrimSpace(cp.Purpose), Code: cp.Code})
	}
	intent.CustomPlugins = cps
	return &intent, nil
}

func containsTag(list []string, tag string) bool {
	for _, t := range list {
		if t == tag {
			return true
		}
	}
	return false
}

// buildProjectContext 收集语言无关的项目轻量上下文（README 摘要 + 文件树 + 静态特征）。
func buildProjectContext(projectDir string, p *ProjectProfile) string {
	var b strings.Builder
	b.WriteString("## 项目静态特征\n")
	b.WriteString(fmt.Sprintf("- 名称: %s\n", p.Name))
	if len(p.Langs) > 0 {
		b.WriteString("- 语言: " + strings.Join(p.Langs, ", ") + "\n")
	}
	if len(p.Frameworks) > 0 {
		b.WriteString("- 框架: " + strings.Join(p.Frameworks, ", ") + "\n")
	}
	if p.BuildCmd != "" {
		b.WriteString(fmt.Sprintf("- 构建: %s\n", p.BuildCmd))
	}
	if p.HasAPI {
		b.WriteString("- 含 API 目录/接口代码\n")
	}
	if p.HasDBData {
		b.WriteString("- 含数据文件（csv/json/sqlite）\n")
	}
	if p.HasDocs {
		b.WriteString("- 含文档目录\n")
	}

	// README 摘要（目的理解的主要来源）
	if readme := readProjectReadme(projectDir); readme != "" {
		b.WriteString("\n## README（摘要，判断项目目的）\n")
		b.WriteString(readme)
		b.WriteString("\n")
	}

	// 文件树概览（顶层目录 + 关键源文件，截断）
	b.WriteString("\n## 文件树概览\n")
	b.WriteString(fileTreeOverview(projectDir))
	return b.String()
}

// readProjectReadme 读取 README（.md，含中文变体），截取前 3000 字。
func readProjectReadme(projectDir string) string {
	for _, name := range []string{"README.zh.md", "README.md", "readme.md", "README"} {
		data, err := os.ReadFile(filepath.Join(projectDir, name))
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(data))
		if len(s) > 3000 {
			s = s[:3000] + "\n…（截断）"
		}
		return s
	}
	return ""
}

// fileTreeOverview 顶层目录 + 各目录源文件计数 + 关键文件（跳过 .git/node_modules 等）。
func fileTreeOverview(projectDir string) string {
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".idea": true, ".vscode": true, "dist": true, "build": true, "vendor": true, "target": true}
	var b strings.Builder
	var topDirs []string
	topFiles := 0
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "(读取失败)"
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			if !skipDirs[e.Name()] {
				topDirs = append(topDirs, e.Name())
			}
		} else {
			topFiles++
		}
	}
	sort.Strings(topDirs)
	for _, d := range topDirs {
		n := countFiles(filepath.Join(projectDir, d), 0, 60)
		fmt.Fprintf(&b, "  %s/ (%d 文件)\n", d, n)
	}
	fmt.Fprintf(&b, "  (根目录 %d 个文件)\n", topFiles)
	// 关键清单：根 + 一级目录的关键源文件（前 40 个）
	extRank := map[string]int{".go": 0, ".py": 1, ".ts": 2, ".js": 3, ".vue": 4, ".rs": 5, ".java": 6, ".c": 7, ".cpp": 8, ".h": 9, ".sql": 10, ".sh": 11, ".yml": 12, ".yaml": 13}
	var files []string
	for _, d := range append([]string{""}, topDirs...) {
		dir := filepath.Join(projectDir, d)
		es, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range es {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if _, ok := extRank[ext]; ok {
				files = append(files, filepath.Join(d, e.Name()))
			}
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return extRank[strings.ToLower(filepath.Ext(files[i]))] < extRank[strings.ToLower(filepath.Ext(files[j]))]
	})
	if len(files) > 40 {
		files = files[:40]
	}
	for _, f := range files {
		b.WriteString("  " + f + "\n")
	}
	return b.String()
}

func countFiles(dir string, depth, max int) int {
	if depth > 2 || max <= 0 {
		return 0
	}
	n := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			n += countFiles(filepath.Join(dir, e.Name()), depth+1, max-n)
		} else {
			n++
		}
		if n >= max {
			break
		}
	}
	return n
}

// buildIntentPrompt 构造 LLM 意图分析提示词（语言无关，强制 JSON 输出）。
func buildIntentPrompt(projectDir string, p *ProjectProfile, requirement string) string {
	var b strings.Builder
	b.WriteString("你是项目分析助手。阅读下面的项目上下文，判断这个项目**实际要实现的目的**（不是只看依赖），")
	b.WriteString("然后推荐开发该项目的 agent 需要哪些工具类别。\n\n")
	b.WriteString(buildProjectContext(projectDir, p))
	if strings.TrimSpace(requirement) != "" {
		b.WriteString("\n## 用户补充要求\n" + strings.TrimSpace(requirement) + "\n")
	}
	b.WriteString(`
## 输出要求
只输出一个 JSON 对象（不要任何解释文字）：
{
  "purpose": "用一句话概括项目实际目的（中文，30 字以内）",
  "buildCmd": "项目的构建命令（根据 README/配置文件真实判断，如 npm run build / make / cargo build / go build ./...；不确定则空串）",
  "testCmd": "测试命令（如 npm test / pytest / cargo test；不确定则空串）",
  "runCmd": "运行/启动命令（如 npm run dev / uvicorn main:app --reload；不确定则空串）",
  "lintCmd": "代码检查命令（如 npx eslint . / ruff check . / golangci-lint run；没有则空串）",
  "formatCmd": "代码格式化命令（如 npx prettier --write . / ruff format .；没有则空串）",
  "recommendedTags": ["从以下类别中选择 2~6 个：` + strings.Join(ToolsetIntentTags, " / ") + `"],
  "notes": "针对项目目的，说明 agent 开发时需要哪些具体工具能力（30~80 字，中文）",
  "customPlugins": [
    {
      "name": "项目专属插件名（小写字母/数字/中划线，如 openapi-gen）",
      "purpose": "用途一句话（20 字内，中文）",
      "code": "插件代码，见下方格式"
    }
  ]
}
命令必须来自项目实际（README/配置文件/文件结构），不要臆造；recommendedTags 只能使用上面给出的类别词，不能自造。

## customPlugins 生成规则（只在必要时写，宁缺毋滥）

模板已覆盖 build/test/git/lint/api/data/docs 等通用能力，**不要为这些写插件**。
只有检测到模板覆盖不到的**项目专属能力缺口**（如 OpenAPI 生成校验、Protobuf 编译、数据库迁移、
特殊格式转换、项目特有脚本封装等）时才写，最多 2 个。

插件代码格式（纯 JS，禁止 import/require/TS 语法；async 函数体）：

~~~js
return {
  name: '插件名',
  inject: ['bash', 'fs'],            // 按需声明：bash(命令) / fs(文件) / web(HTTP) / logger / timer
  apply(ctx) {
    ctx.tools.register({
      name: '工具名',
      description: '一句话说明做什么、参数怎么用',
      parameters: {
        type: 'object',
        properties: { 参数名: { type: 'string', description: '说明' } },
        required: ['参数名']
      },
      execute: async (args) => {
        // 可调用 ctx.bash.exec('命令', '工作区根目录') → {output, error}
        // ctx.fs.read(path) / ctx.fs.write(path, content) / ctx.fs.exists(path) / ctx.fs.list(dir) / ctx.fs.stat(path)
        // ctx.web.fetch(url) → {ok, status, text}
        return '结果文本';
      }
    });
  }
};
~~~

要点：参数 schema 完整（name/description/parameters.required 齐全）；execute 返回字符串；
只在 execute 内部用 async 服务调用，apply 同步注册即可；一个插件可注册 1~2 个紧密相关工具。
`)
	return b.String()
}
