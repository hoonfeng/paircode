// llm_analyze_test.go — LLM 项目意图分析：解析 / 命令注入 / 语言无关性验证。
package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseProjectIntent 解析 LLM 输出（容忍围栏/噪声，命令字段提取）。
func TestParseProjectIntent(t *testing.T) {
	raw := "好的，分析如下：\n" +
		"```json\n" +
		`{"purpose": "电商后端 API 服务", "buildCmd": "go build ./...", "testCmd": "go test ./...", "runCmd": "go run ./cmd/server", "lintCmd": "golangci-lint run", "formatCmd": "gofmt -w .", "recommendedTags": ["API", "build", "数据库"], "notes": "需要接口调试与数据库操作工具"}` +
		"\n```\n"
	it, err := parseProjectIntent(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if it.Purpose != "电商后端 API 服务" {
		t.Fatalf("purpose: %q", it.Purpose)
	}
	if it.BuildCmd != "go build ./..." || it.RunCmd != "go run ./cmd/server" {
		t.Fatalf("命令提取错误: %+v", it)
	}
	// 未知标签被清洗
	if len(it.RecommendedTags) != 2 || it.RecommendedTags[0] != "api" || it.RecommendedTags[1] != "build" {
		t.Fatalf("标签清洗错误: %v", it.RecommendedTags)
	}
}

// TestLLMIntentCommands LLM 分析命令注入生成器（语言无关：命令来自 LLM 而非固化模板）。
func TestLLMIntentCommands(t *testing.T) {
	project := mkToolsetGoProject(t) // go.mod + main.go（无 Makefile/package.json → 静态无命令）
	host := NewPluginHost(NewRegistry(), nil, project)

	// 注入 mock LLM：识别为 node 项目并给出真实命令
	old := toolsetLLMProvider
	toolsetLLMProvider = func() Provider {
		return &MockProvider{Responses: []Message{{
			Role: RoleAssistant,
			Content: `{"purpose":"Next.js 前端应用","buildCmd":"npm run build","testCmd":"npm test","runCmd":"npm run dev","lintCmd":"npx eslint .","formatCmd":"npx prettier --write .","recommendedTags":["build","test","lint"],"notes":"需要构建与代码质量工具"}`,
		}}}
	}
	defer func() { toolsetLLMProvider = old }()

	ts, err := BuildToolset(host, project, "llm-dev", "", "")
	if err != nil {
		t.Fatalf("BuildToolset: %v", err)
	}
	// 描述来自 LLM purpose
	if !strings.Contains(ts.Description, "Next.js 前端应用") {
		t.Fatalf("描述应为 LLM purpose: %q", ts.Description)
	}
	// 命令来自 LLM（不是语言固化）
	found := map[string]bool{}
	for _, p := range ts.Plugins {
		found[p.Name] = true
		if p.Name == strings.TrimSuffix(projectBase(project)+"-project-helper", "-project-helper-project-helper") {
			// 插件名 = basename-project-helper
		}
		if strings.Contains(p.Code, "npm run build") {
			found["build-npm"] = true
		}
		if strings.Contains(p.Code, "npx eslint .") {
			found["lint-eslint"] = true
		}
		if strings.Contains(p.Code, "go build") {
			found["lang-hardcoded"] = true // 不应出现（语言固化）
		}
	}
	if !found["build-npm"] || !found["lint-eslint"] {
		t.Fatalf("LLM 命令未注入: %v", found)
	}
	if found["lang-hardcoded"] {
		t.Fatal("不应出现语言固化命令（go build）——命令必须来自 LLM/探测")
	}
}

// TestNoLanguageHardcode 无 LLM 且无真实命令文件时：不生成 build/test/run/lint 工具
// （语言不可预知，不固化任何语言模板），仅保留 _profile 特征工具 + 通用 git-flow。
func TestNoLanguageHardcode(t *testing.T) {
	project := mkToolsetGoProject(t) // go.mod + main.go，无 Makefile/package.json
	host := NewPluginHost(NewRegistry(), nil, project)
	// 无 provider（core.Settings 未配置 → toolsetLLMProvider nil）
	old := toolsetLLMProvider
	toolsetLLMProvider = func() Provider { return nil }
	defer func() { toolsetLLMProvider = old }()

	ts, err := BuildToolset(host, project, "plain", "", "")
	if err != nil {
		t.Fatalf("BuildToolset: %v", err)
	}
	code := ""
	for _, p := range ts.Plugins {
		code += p.Code
	}
	// 不应出现语言固化命令
	for _, banned := range []string{"go build", "go test", "gofmt", "npm run", "cargo", "mvn "} {
		if strings.Contains(code, banned) {
			t.Fatalf("不应固化语言命令 %q（无 LLM/真实命令时应跳过）:\n%s", banned, truncateStr(code, 400))
		}
	}
	// 通用 git-flow 仍生成
	names := map[string]bool{}
	for _, p := range ts.Plugins {
		names[p.Name] = true
	}
	if !names["git-flow"] {
		t.Fatalf("git-flow 应生成: %v", names)
	}
}

// projectBase 取目录名（辅助断言）。
func projectBase(dir string) string {
	return filepath.Base(dir)
}

// TestParseProjectIntentCustomPlugins LLM 输出含 customPlugins：解析 + 清洗
// （code 非空、name 规范化、去重）。
func TestParseProjectIntentCustomPlugins(t *testing.T) {
	raw := "```json\n" +
		`{"purpose":"OpenAPI 服务生成器","recommendedTags":["api"],
		  "customPlugins":[
		    {"name":"openapi-gen","purpose":"OpenAPI 定义校验与生成","code":"return { name: 'openapi-gen', apply(ctx) {} };"},
		    {"name":"空 code","purpose":"应剔除","code":""},
		    {"name":"API Schema Tool","purpose":"规范化名称","code":"return { name: 'api-schema-tool', apply(ctx) {} };"},
		    {"name":"openapi-gen","purpose":"重复应剔除","code":"return { name: 'dup', apply(ctx) {} };"}
		  ]}` +
		"\n```\n"
	it, err := parseProjectIntent(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(it.CustomPlugins) != 2 {
		t.Fatalf("应清洗为 2 条（空 code 剔除、重复去重）: %+v", it.CustomPlugins)
	}
	if it.CustomPlugins[0].Name != "openapi-gen" {
		t.Fatalf("首条应为 openapi-gen: %v", it.CustomPlugins[0].Name)
	}
	if it.CustomPlugins[1].Name != "api-schema-tool" {
		t.Fatalf("name 应规范化为 api-schema-tool（大写/空格转小写中划线）: %v", it.CustomPlugins[1].Name)
	}
}

// TestBuildToolsetCustomPlugins LLM 现场生成的项目专属插件并入工具集：
// 模板产物共存、坏插件 define 预检剔除、与模板重名跳过、并入插件实际装载。
func TestBuildToolsetCustomPlugins(t *testing.T) {
	project := mkToolsetGoProject(t)
	host := NewPluginHost(NewRegistry(), nil, project)
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	old := toolsetLLMProvider
	toolsetLLMProvider = func() Provider {
		return &MockProvider{Responses: []Message{{
			Role:    RoleAssistant,
			Content: `{"purpose":"OpenAPI 服务生成器","recommendedTags":["api","build"],"customPlugins":[` +
				`{"name":"openapi-gen","purpose":"OpenAPI 定义校验与生成","code":"return { name: 'openapi-gen', inject: ['bash'], apply(ctx) { ctx.tools.register({ name: 'openapi_validate', description: '校验 OpenAPI 定义', parameters: { type: 'object', properties: { path: { type: 'string', description: 'OpenAPI 文件路径' } }, required: ['path'] }, execute: async (a) => { const r = await ctx.bash.exec('npx @redocly/cli lint ' + a.path); return r.error || r.output; } }); } };"},` +
				`{"name":"broken","purpose":"语法错误应被剔除","code":"return { name: 'broken', apply(ctx) { ctx.tools.register("},` +
				`{"name":"git-flow","purpose":"与模板重名应跳过","code":"return { name: 'git-flow', apply(ctx) {} };"}` +
				`]}`,
		}}}
	}
	defer func() { toolsetLLMProvider = old }()

	ts, err := BuildToolset(host, project, "llm-plugins", "", "")
	if err != nil {
		t.Fatalf("BuildToolset: %v", err)
	}
	counts := map[string]int{}
	for _, p := range ts.Plugins {
		counts[p.Name]++
	}
	// ① LLM 现场生成插件并入
	if counts["openapi-gen"] != 1 {
		t.Fatalf("openapi-gen 应并入: %v", counts)
	}
	// ② 坏插件预检失败剔除
	if counts["broken"] != 0 {
		t.Fatalf("broken 应被剔除: %v", counts)
	}
	// ③ 与模板重名跳过（git-flow 模板产物保留且仅一个）
	if counts["git-flow"] != 1 {
		t.Fatalf("git-flow 应仅模板产物一个: %v", counts)
	}
	// ④ 模板产物共存（git-flow 通用模板命中即为模板产物）
	// ⑤ 并入插件实际装载（apply 注册 openapi_validate）
	if _, ok := host.Get("openapi-gen"); !ok {
		t.Fatal("openapi-gen 应已装载（installToolset 定义+加载）")
	}
}

// mkToolsetGoProject 复用 toolset_test.go（避免重复定义冲突则此处独立实现）。
var _ = os.ReadFile
