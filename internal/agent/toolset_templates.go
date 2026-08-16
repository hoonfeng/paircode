// toolset_templates.go — 工具集构建模板：分析项目特征 → 模板匹配 → 生成插件。
//
// ★ 模板本身插件化（对齐「一切皆插件」）：任何插件（内置/市场/用户）都可以通过
//   ctx.toolset.registerTemplate({id, title, match, generate}) 注册一个工具集
//   构建模板。toolset_build 时宿主收集全部模板，按项目特征（ProjectProfile）匹配、
//   组合生成插件集合。
//
// 内置模板为宿主框架能力（内联于 NewPluginHost，不经过插件体系）：
//   - toolset.tpl.project-helper  构建/测试/运行命令助手（按语言生成）
//   - toolset.tpl.git-flow        Git 工作流辅助（提交检查/分支摘要）
//   - toolset.tpl.code-quality    lint/格式化
//   - toolset.tpl.web-api         HTTP 接口调试（web 项目命中）
//   - toolset.tpl.data-inspect    数据文件概览（csv/json/sqlite，数据项目命中）
// 市场/用户可用 JS 插件注册专属模板（ctx.toolset.registerTemplate），实现
// 「工具集构建处理」的完全可插拔。

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"wb-ui/goja"
)

// ─── 模板类型 ─────────────────────────────────────────────

// ToolsetTemplate 工具集构建模板：判定项目是否适用 + 生成插件定义。
type ToolsetTemplate struct {
	ID    string // 唯一 id（如 toolset.tpl.project-helper）
	Title string // 展示名

	// Tags 模板适用场景标签（语言无关，如 build/test/git/lint/api/data）。
	// 与 LLM 项目意图推荐（ProjectIntent.RecommendedTags）做意图匹配：
	// 静态特征未命中但意图标签命中时，模板仍强制加入组合。
	Tags []string

	// Go 模板：直接提供 Match/Generate 函数。
	Match    func(profile *ProjectProfile) bool
	Generate func(profile *ProjectProfile, requirement string) ([]ToolsetPlugin, error)

	// JS 模板（外部插件经 ctx.toolset.registerTemplate 注册）：
	// jsMatch/jsGenerate 为 goja.Callable，jsVM 为所属沙箱（回调在锁内执行）。
	jsMatch    goja.Callable
	jsGenerate goja.Callable
	jsVM       *goja.Runtime
	jsLock     func(func()) // 沙箱执行锁（与插件主执行流互斥）
}

// matchesIntent 意图标签是否命中模板场景（静态特征未命中时的补充判定）。
func (t *ToolsetTemplate) matchesIntent(intent *ProjectIntent) bool {
	if intent == nil || len(t.Tags) == 0 {
		return false
	}
	for _, it := range intent.RecommendedTags {
		for _, tt := range t.Tags {
			if strings.EqualFold(it, tt) {
				return true
			}
		}
	}
	return false
}

// profileToMap 把 ProjectProfile 转成 json tag 字段的 map（JS 模板收到
// profile.langs 等 json 命名；goja ToValue 默认用 Go 字段名，需显式转换）。
func profileToMap(p *ProjectProfile) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return map[string]any{"root": p.Root, "name": p.Name}
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return map[string]any{}
	}
	return m
}

// matches 判定模板是否适用于该项目（JS 模板走沙箱调用）。
func (t *ToolsetTemplate) matches(p *ProjectProfile) bool {
	if t.Match != nil {
		return t.Match(p)
	}
	if t.jsMatch != nil && t.jsVM != nil {
		var out bool
		var callErr error
		run := func() {
			r, err := t.jsMatch(goja.Undefined(), t.jsVM.ToValue(profileToMap(p)))
			if err != nil {
				callErr = err
				return
			}
			out = r.ToBoolean()
		}
		if t.jsLock != nil {
			t.jsLock(run)
		} else {
			run()
		}
		if callErr != nil {
			return false
		}
		return out
	}
	return true
}

// generate 按项目特征 + 要求生成插件定义（JS 模板走沙箱调用）。
func (t *ToolsetTemplate) generate(p *ProjectProfile, requirement string) ([]ToolsetPlugin, error) {
	if t.Generate != nil {
		return t.Generate(p, requirement)
	}
	if t.jsGenerate != nil && t.jsVM != nil {
		var r goja.Value
		var genErr error
		run := func() {
			res, err := t.jsGenerate(goja.Undefined(), t.jsVM.ToValue(profileToMap(p)), t.jsVM.ToValue(requirement))
			if err != nil {
				genErr = err
				return
			}
			r = res
		}
		if t.jsLock != nil {
			t.jsLock(run)
		} else {
			run()
		}
		if genErr != nil {
			return nil, genErr
		}
		// 返回值：插件定义数组 [{name, purpose, code, client?}]
		if r == nil || goja.IsUndefined(r) || goja.IsNull(r) {
			return nil, nil
		}
		raw := r.Export()
		arr, ok := raw.([]any)
		if !ok {
			if m, ok2 := raw.(map[string]any); ok2 {
				arr = []any{m}
			} else {
				return nil, fmt.Errorf("模板 %s generate 应返回插件定义数组，得到 %T", t.ID, raw)
			}
		}
		var out []ToolsetPlugin
		for _, it := range arr {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			tp := ToolsetPlugin{
				Name:    mStrVal(m["name"]),
				Purpose: mStrVal(m["purpose"]),
				Code:    mStrVal(m["code"]),
				Client:  mStrVal(m["client"]),
			}
			if tp.Name == "" || strings.TrimSpace(tp.Code) == "" {
				continue
			}
			out = append(out, tp)
		}
		return out, nil
	}
	return nil, nil
}

// mStrVal 任意值转字符串（nil 安全）。
func mStrVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// ─── 项目特征分析（基础设施，内置）────────────────────────

// ProjectProfile 项目特征（toolset_build 分析产物）。
type ProjectProfile struct {
	Root       string   `json:"root"`
	Name       string   `json:"name"`
	Langs      []string `json:"langs"`      // go/typescript/javascript/python/rust/java/cpp...
	Frameworks []string `json:"frameworks"` // vue/react/express/fastapi/gin/next...
	BuildCmd   string   `json:"buildCmd"`
	TestCmd    string   `json:"testCmd"`
	RunCmd     string   `json:"runCmd"`
	LintCmd    string   `json:"lintCmd"`    // 通常由 LLM 分析给出（不按语言固化）
	FormatCmd  string   `json:"formatCmd"`  // 通常由 LLM 分析给出
	HasDocker  bool     `json:"hasDocker"`
	HasDBData  bool     `json:"hasDBData"` // csv/json/db/sqlite 数据文件
	HasAPI     bool     `json:"hasAPI"`    // api 目录/路由文件
	HasDocs    bool     `json:"hasDocs"`
}

// HasLang 是否含指定语言。
func (p *ProjectProfile) HasLang(lang string) bool {
	for _, l := range p.Langs {
		if l == lang {
			return true
		}
	}
	return false
}

// HasFramework 是否含指定框架。
func (p *ProjectProfile) HasFramework(fw string) bool {
	for _, f := range p.Frameworks {
		if f == fw {
			return true
		}
	}
	return false
}

// analyzeProject 分析项目根目录 → 特征画像（供模板匹配）。
func analyzeProject(projectDir string) *ProjectProfile {
	p := &ProjectProfile{Root: projectDir}
	if projectDir == "" {
		return p
	}
	p.Name = filepath.Base(projectDir)

	hasFile := func(names ...string) bool {
		for _, n := range names {
			if st, err := os.Stat(filepath.Join(projectDir, n)); err == nil && !st.IsDir() {
				return true
			}
		}
		return false
	}
	hasDir := func(names ...string) bool {
		for _, n := range names {
			if st, err := os.Stat(filepath.Join(projectDir, n)); err == nil && st.IsDir() {
				return true
			}
		}
		return false
	}

	// 语言检测（★只收集事实，不从语言推导命令——项目语言不可预知，
	// 构建/测试/运行命令只来自：① 真实文件探测（Makefile/package.json scripts）
	// ② LLM 项目意图分析（llmAnalyzeProject）。无命令时生成器跳过对应工具。）
	if hasFile("go.mod") {
		p.Langs = append(p.Langs, "go")
	}
	if hasFile("package.json") {
		p.Langs = append(p.Langs, "node")
		// 命令探测：真实 scripts（语言无关，看项目自己怎么声明）
		if data, err := os.ReadFile(filepath.Join(projectDir, "package.json")); err == nil {
			var pkg struct {
				Scripts         map[string]string `json:"scripts"`
				Dependencies    map[string]string `json:"dependencies"`
				DevDependencies map[string]string `json:"devDependencies"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				if s, ok := pkg.Scripts["build"]; ok {
					p.BuildCmd = "npm run build" // 以 npm 为标准入口（scripts.build 存在）
					_ = s
				}
				if s, ok := pkg.Scripts["test"]; ok {
					p.TestCmd = "npm test"
					_ = s
				}
				for _, k := range []string{"dev", "start"} {
					if _, ok := pkg.Scripts[k]; ok {
						p.RunCmd = "npm run " + k
						break
					}
				}
				merge := func(m map[string]string) {
					for k := range m {
						lk := strings.ToLower(k)
						for _, fw := range []string{"vue", "react", "next", "nuxt", "express", "fastify", "nest", "svelte", "angular", "electron", "vite"} {
							if strings.Contains(lk, fw) {
								p.Frameworks = append(p.Frameworks, fw)
								break
							}
						}
					}
				}
				merge(pkg.Dependencies)
				merge(pkg.DevDependencies)
			}
		}
	}
	if hasFile("pyproject.toml", "requirements.txt", "setup.py", "Pipfile") {
		p.Langs = append(p.Langs, "python")
		if hasFile("pyproject.toml") {
			if data, err := os.ReadFile(filepath.Join(projectDir, "pyproject.toml")); err == nil {
				low := strings.ToLower(string(data))
				for _, fw := range []string{"fastapi", "flask", "django", "tornado", "aiohttp", "streamlit"} {
					if strings.Contains(low, fw) {
						p.Frameworks = append(p.Frameworks, fw)
					}
				}
			}
		}
	}
	if hasFile("Cargo.toml") {
		p.Langs = append(p.Langs, "rust")
	}
	if hasFile("pom.xml", "build.gradle", "build.gradle.kts") {
		p.Langs = append(p.Langs, "java")
	}
	if hasFile("CMakeLists.txt") {
		p.Langs = append(p.Langs, "cpp")
	}
	// 命令探测：真实文件（语言无关）
	if hasFile("Makefile", "makefile") && p.BuildCmd == "" {
		p.BuildCmd = "make"
		p.TestCmd = "make test"
	}

	// 环境/入口
	p.HasDocker = hasFile("Dockerfile", "docker-compose.yml", "compose.yaml")
	p.HasDocs = hasDir("docs") || hasFile("README.md")

	// 数据文件（浅扫两层）
	dataExts := map[string]bool{".csv": true, ".db": true, ".sqlite": true, ".sqlite3": true, ".jsonl": true}
	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(projectDir, path)
		if strings.HasPrefix(rel, ".pair") || strings.HasPrefix(rel, ".git") || strings.HasPrefix(rel, "node_modules") {
			return nil
		}
		if dataExts[strings.ToLower(filepath.Ext(path))] {
			p.HasDBData = true
		}
		if strings.HasSuffix(path, ".json") && strings.Contains(strings.ToLower(rel), "data") {
			p.HasDBData = true
		}
		return nil
	})

	// API 目录/路由文件
	p.HasAPI = hasDir("api", "apis", "routes", "router", "handlers", "controllers", "endpoints")
	if !p.HasAPI && len(p.Frameworks) > 0 {
		for _, fw := range p.Frameworks {
			if fw == "express" || fw == "fastapi" || fw == "flask" || fw == "django" || fw == "gin" || fw == "spring" || fw == "nest" {
				p.HasAPI = true
				break
			}
		}
	}
	// 去重
	dedupe := func(s []string) []string {
		seen := map[string]bool{}
		out := []string{}
		for _, v := range s {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
		return out
	}
	p.Langs = dedupe(p.Langs)
	p.Frameworks = dedupe(p.Frameworks)
	sort.Strings(p.Frameworks)
	return p
}

// ─── 内置模板（宿主框架能力，内联于 NewPluginHost）─────────

// registerBuiltinTemplates 注册宿主内置工具集构建模板（框架能力：toolset_build
// 的动态组合数据源——Generate 逻辑内嵌宿主，随宿主合理，不经过插件体系）。
// ★ 2026-08-16：原以插件 toolset-tpl-core 形态注册（不可启停，无意义），
//   现内联为宿主固有（NewPluginHost 调用），插件列表不再显示；
//   市场/用户插件仍可经 RegisterTemplate / ctx.toolset.registerTemplate 追加。
func registerBuiltinTemplates(ph *PluginHost) {
	// 1. 项目助手：构建/测试/运行（按语言生成命令）
	_ = ph.RegisterTemplate(&ToolsetTemplate{
		ID: "toolset.tpl.project-helper", Title: "项目构建/测试/运行助手",
		Tags: []string{"build", "test", "run"},
		Match: func(p *ProjectProfile) bool { return len(p.Langs) > 0 },
		Generate: func(p *ProjectProfile, req string) ([]ToolsetPlugin, error) {
			return genProjectHelper(p), nil
		},
	})
	// 2. Git 工作流辅助
	_ = ph.RegisterTemplate(&ToolsetTemplate{
		ID: "toolset.tpl.git-flow", Title: "Git 工作流辅助（提交检查/分支摘要）",
		Tags: []string{"git"},
		Match: func(p *ProjectProfile) bool { return true },
		Generate: func(p *ProjectProfile, req string) ([]ToolsetPlugin, error) {
			return genGitFlow(), nil
		},
	})
	// 3. 代码质量：lint/格式化（按语言生成命令）
	_ = ph.RegisterTemplate(&ToolsetTemplate{
		ID: "toolset.tpl.code-quality", Title: "代码质量（lint/格式化）",
		Tags: []string{"lint", "code-quality"},
		Match: func(p *ProjectProfile) bool { return len(p.Langs) > 0 },
		Generate: func(p *ProjectProfile, req string) ([]ToolsetPlugin, error) {
			return genCodeQuality(p), nil
		},
	})
	// 4. HTTP 接口调试（web 项目命中）
	_ = ph.RegisterTemplate(&ToolsetTemplate{
		ID: "toolset.tpl.web-api", Title: "HTTP 接口调试",
		Tags: []string{"api", "http"},
		Match: func(p *ProjectProfile) bool { return p.HasAPI || len(p.Frameworks) > 0 },
		Generate: func(p *ProjectProfile, req string) ([]ToolsetPlugin, error) {
			return genWebAPI(p), nil
		},
	})
	// 5. 数据文件概览（数据项目命中）
	_ = ph.RegisterTemplate(&ToolsetTemplate{
		ID: "toolset.tpl.data-inspect", Title: "数据文件概览（csv/json/sqlite）",
		Tags: []string{"data"},
		Match: func(p *ProjectProfile) bool { return p.HasDBData },
		Generate: func(p *ProjectProfile, req string) ([]ToolsetPlugin, error) {
			return genDataInspect(p), nil
		},
	})
}

// ─── 内置模板生成器 ───────────────────────────────────────

// jsTool 工具参数 schema 辅助（生成 JSON Schema 片段）。
const (
	schemaStr = `{"type":"string"}`
)

// jq 引号包裹（JS 字符串字面量）。
func jq(s string) string { return "'" + strings.ReplaceAll(s, "'", "\\'") + "'" }

// toolPrefix 项目短名前缀（工具名前缀，跨工具集不冲突；'gou-ide'→'gouide'）。
func toolPrefix(p *ProjectProfile) string {
	name := strings.ReplaceAll(p.Name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	if name == "" {
		return "project"
	}
	return strings.ToLower(name)
}

// genProjectHelper 生成项目构建/测试/运行助手插件。
// 工具名带项目短名前缀（如 gouide_build），避免多个工具集工具名冲突。
// ★ 命令来自真实探测或 LLM 分析（profile 字段）；命令缺失时不生成对应工具
//   （不按语言固化命令——项目语言不可预知）。_profile 工具始终生成。
func genProjectHelper(p *ProjectProfile) []ToolsetPlugin {
	buildCmd := p.BuildCmd
	testCmd := p.TestCmd
	runCmd := p.RunCmd
	langs := strings.Join(p.Langs, "/")
	if langs == "" {
		langs = "未知"
	}
	pre := toolPrefix(p)
	var code strings.Builder
	code.WriteString(`return {
  name: '` + pluginSafeName(p.Name+"-project-helper") + `',
  inject: ['bash'],
  apply(ctx) {
    const root = ctx.app.workspaceRoot;
`)
	if buildCmd != "" {
		fmt.Fprintf(&code, `    ctx.tools.register({
      name: '%s_build',
      description: '构建当前项目（%s）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const r = await ctx.bash.exec('%s', root);
        return r.error ? ('构建失败:\\n' + r.error) : ('构建成功\\n' + r.output);
      }
    });
`, pre, buildCmd, buildCmd)
	}
	if testCmd != "" {
		fmt.Fprintf(&code, `    ctx.tools.register({
      name: '%s_test',
      description: '运行项目测试（%s）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const r = await ctx.bash.exec('%s', root);
        return r.error ? ('测试失败:\\n' + r.error) : ('测试通过\\n' + r.output);
      }
    });
`, pre, testCmd, testCmd)
	}
	if runCmd != "" {
		fmt.Fprintf(&code, `    ctx.tools.register({
      name: '%s_run',
      description: '启动/运行当前项目（%s）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const r = await ctx.bash.exec('%s', root);
        return r.error ? ('运行失败:\\n' + r.error) : ('运行输出:\\n' + r.output);
      }
    });
`, pre, runCmd, runCmd)
	}
	fmt.Fprintf(&code, `    ctx.tools.register({
      name: '%s_profile',
      description: '输出当前项目特征（语言/框架/构建测试运行命令），帮助 agent 选择正确命令。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        return '语言: %s\\n构建: %s\\n测试: %s\\n运行: %s';
      }
    });
  }
};`, pre, langs, buildCmdOrNone(buildCmd), testCmdOrNone(testCmd), runCmdOrNone(runCmd))
	return []ToolsetPlugin{{
		Name:    pluginSafeName(p.Name + "-project-helper"),
		Purpose: "项目构建/测试/运行命令助手（" + langs + "）",
		Code:    code.String(),
	}}
}

// cmdOrNone 命令占位（JS 字符串安全）。
func cmdOrNone(cmd string) string {
	if cmd == "" {
		return "（未探测到）"
	}
	return cmd
}

func buildCmdOrNone(c string) string { return cmdOrNone(c) }
func testCmdOrNone(c string) string  { return cmdOrNone(c) }
func runCmdOrNone(c string) string   { return cmdOrNone(c) }

// genGitFlow 生成 Git 工作流辅助插件。
func genGitFlow() []ToolsetPlugin {
	code := `return {
  name: 'git-flow',
  inject: ['bash'],
  apply(ctx) {
    const root = ctx.app.workspaceRoot;
    ctx.tools.register({
      name: 'git_commit_check',
      description: '检查提交就绪状态：未提交变更、暂存区、最近提交（git status + diff stat）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const s = await ctx.bash.exec('git status --short', root);
        const d = await ctx.bash.exec('git diff --stat', root);
        const st = await ctx.bash.exec('git diff --cached --stat', root);
        const lg = await ctx.bash.exec('git log --oneline -5', root);
        return [
          '## 工作区状态', s.output || '(干净)', '',
          '## 未暂存变更', d.output || '(无)', '',
          '## 已暂存变更', st.output || '(无)', '',
          '## 最近提交', lg.output || '(无)'
        ].join('\n');
      }
    });
    ctx.tools.register({
      name: 'git_branch_summary',
      description: '当前分支与分支拓扑摘要（分支列表 + 领先/落后计数）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const cur = await ctx.bash.exec('git branch --show-current', root);
        const brs = await ctx.bash.exec('git branch -vv', root);
        return '当前分支: ' + (cur.output || '?').trim() + '\n\n' + (brs.output || '');
      }
    });
  }
};`
	return []ToolsetPlugin{{
		Name:    "git-flow",
		Purpose: "Git 工作流辅助（提交检查/分支摘要）",
		Code:    code,
	}}
}

// genCodeQuality 生成代码质量插件（按语言选 lint/格式化命令）。
// genCodeQuality 生成代码质量插件（lint/格式化）。
// ★ 命令来自 LLM 分析或真实探测（profile.LintCmd/FormatCmd），不按语言固化——
//   无命令时返回 nil（不生成该插件）。
func genCodeQuality(p *ProjectProfile) []ToolsetPlugin {
	lintCmd, fmtCmd := strings.TrimSpace(p.LintCmd), strings.TrimSpace(p.FormatCmd)
	if lintCmd == "" && fmtCmd == "" {
		return nil
	}
	var code strings.Builder
	code.WriteString(`return {
  name: 'code-quality',
  inject: ['bash'],
  apply(ctx) {
    const root = ctx.app.workspaceRoot;
`)
	if lintCmd != "" {
		fmt.Fprintf(&code, `    ctx.tools.register({
      name: 'lint_project',
      description: '运行代码检查（%s）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const r = await ctx.bash.exec('%s', root);
        return r.error ? ('lint 失败:\\n' + r.error) : ('lint 结果:\\n' + (r.output || '(无问题)'));
      }
    });
`, lintCmd, lintCmd)
	}
	if fmtCmd != "" {
		fmt.Fprintf(&code, `    ctx.tools.register({
      name: 'format_project',
      description: '格式化代码（%s）。需谨慎使用（会修改文件）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const r = await ctx.bash.exec('%s', root);
        return r.error ? ('格式化失败:\\n' + r.error) : ('格式化完成\\n' + r.output);
      }
    });
`, fmtCmd, fmtCmd)
	}
	code.WriteString(`  }
};`)
	return []ToolsetPlugin{{
		Name:    "code-quality",
		Purpose: "代码质量（lint/格式化）",
		Code:    code.String(),
	}}
}

// genWebAPI 生成 HTTP 接口调试插件（web 项目）。
func genWebAPI(p *ProjectProfile) []ToolsetPlugin {
	base := "http://localhost:8080"
	code := `return {
  name: 'web-api',
  inject: ['web', 'bash'],
  apply(ctx) {
    ctx.tools.register({
      name: 'http_request',
      description: '发送 HTTP 请求调试接口（GET/POST）。URL 参数如 http://localhost:8080/api/ping。',
      parameters: {
        type: 'object',
        properties: {
          url: { type: 'string', description: '完整 URL（如 http://localhost:8080/api/users）' },
          method: { type: 'string', description: 'GET/POST/PUT/DELETE' }
        },
        required: ['url']
      },
      execute: async (args) => {
        const url = args.url || '` + base + `';
        const m = (args.method || 'GET').toUpperCase();
        if (m === 'GET') {
          const r = await ctx.web.fetch(url);
          return JSON.stringify({ ok: r.ok, status: r.status, text: (r.text || '').slice(0, 4000) }, null, 2);
        }
        const r = await ctx.bash.exec('curl -s -X ' + m + ' ' + url, ctx.app.workspaceRoot);
        return r.error ? ('请求失败: ' + r.error) : r.output;
      }
    });
    ctx.tools.register({
      name: 'api_probe',
      description: '探测本地服务是否在运行（对常见端口发 GET，报告可达性）。',
      parameters: { type: 'object', properties: {} },
      execute: async () => {
        const ports = [8080, 3000, 5173, 8000, 9090];
        const out = [];
        for (const port of ports) {
          const r = await ctx.web.fetch('http://localhost:' + port + '/');
          out.push(':' + port + ' -> ' + (r.ok ? '可达 (' + r.status + ')' : '不可达 (' + r.status + ')'));
        }
        return out.join('\n');
      }
    });
  }
};`
	return []ToolsetPlugin{{
		Name:    "web-api",
		Purpose: "HTTP 接口调试（web 项目）",
		Code:    code,
	}}
}

// genDataInspect 生成数据文件概览插件（数据项目）。
func genDataInspect(p *ProjectProfile) []ToolsetPlugin {
	code := `return {
  name: 'data-inspect',
  inject: ['bash', 'fs'],
  apply(ctx) {
    const root = ctx.app.workspaceRoot;
    ctx.tools.register({
      name: 'csv_preview',
      description: '预览 CSV 文件（前 10 行 + 列名）。参数 path 为相对工作区根的文件路径。',
      parameters: {
        type: 'object',
        properties: { path: { type: 'string', description: 'CSV 文件相对路径' } },
        required: ['path']
      },
      execute: async (args) => {
        const r = await ctx.bash.exec('head -10 ' + args.path, root);
        return r.error ? ('读取失败: ' + r.error) : r.output;
      }
    });
    ctx.tools.register({
      name: 'json_preview',
      description: '预览 JSON 文件结构（键名 + 顶层数组长度）。',
      parameters: {
        type: 'object',
        properties: { path: { type: 'string', description: 'JSON 文件相对路径' } },
        required: ['path']
      },
      execute: async (args) => {
        const r = await ctx.bash.exec('python -c "import json,sys;d=json.load(open(sys.argv[1]));print(type(d).__name__, (len(d) if hasattr(d,\'__len__\') else \'-\')); print(list(d.keys())[:20] if isinstance(d,dict) else \'\')" ' + args.path, root);
        return r.error ? ('读取失败: ' + r.error) : r.output;
      }
    });
    ctx.tools.register({
      name: 'sqlite_query',
      description: '对 SQLite 数据库执行只读查询（SELECT）。参数 db 为相对路径，sql 为查询语句。',
      parameters: {
        type: 'object',
        properties: {
          db: { type: 'string', description: 'SQLite 文件相对路径' },
          sql: { type: 'string', description: 'SELECT 查询（只读）' }
        },
        required: ['db', 'sql']
      },
      execute: async (args) => {
        const r = await ctx.bash.exec('python -c "import sqlite3,sys;c=sqlite3.connect(sys.argv[1]);print(c.execute(sys.argv[2]).fetchmany(20))" ' + args.db + ' ' + JSON.stringify(args.sql), root);
        return r.error ? ('查询失败: ' + r.error) : r.output;
      }
    });
  }
};`
	return []ToolsetPlugin{{
		Name:    "data-inspect",
		Purpose: "数据文件概览（csv/json/sqlite）",
		Code:    code,
	}}
}

// pluginSafeName 插件名规范化（工具集名派生时去特殊字符）。
func pluginSafeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	name := strings.ToLower(b.String())
	name = strings.Trim(name, "-")
	if name == "" {
		return "toolset-helper"
	}
	return name
}
