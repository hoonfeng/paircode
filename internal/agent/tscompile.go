// ═══════════════════════════════════════════════════════════
// tscompile.go — 内置 TS 编译器（esbuild 纯 Go，无 CGO/npm 依赖）
//
// 用于 JS 动态插件（cordis_define/cordis_run）装载 harness 生态的
// TypeScript 插件源码：TS → ES2020 JS，再交给 goja 执行。
// 单文件：无 import，直接 Transform 剥离类型注解。
// 多文件：含 import 时走 esbuild Build 的 stdin+bundle 模式，
// 相对导入内联打包，非相对包导入（@deepseek-ai/cordis 等）mock 成空模块，
// 输出 IIFE + globalName，goja 可直接执行（无 import/export 残留）。
// ═══════════════════════════════════════════════════════════
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

// compileTS 把 TS/TSX 源码转译为 ES2020 JS（类型注解剥离）。
// sourceName 用于错误信息定位（如 "dyn-1.ts"）。
func compileTS(src, sourceName string) (string, error) {
	res := api.Transform(src, api.TransformOptions{
		Loader:     api.LoaderTS,
		Target:     api.ES2020,
		Sourcefile: sourceName,
	})
	if len(res.Errors) > 0 {
		var sb strings.Builder
		for i, e := range res.Errors {
			if i > 0 {
				sb.WriteString("; ")
			}
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("第 %d 行: ", e.Location.Line+1)
			}
			sb.WriteString(loc + e.Text)
		}
		return "", fmt.Errorf("%s", sb.String())
	}
	return string(res.Code), nil
}

// TS 特征正则（快速探测，避免对纯 JS 插件做无谓转译）
var (
	tsInterfaceRe = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:interface|type|enum)\s+[A-Za-z_$]`)
	tsAnnotRe     = regexp.MustCompile(`[:?]\s*(?:string|number|boolean|any|void|unknown|never|object|bigint|symbol)\b|\[\]|Record\s*<|Map\s*<|Array\s*<|Promise\s*<`)
	tsGenericRe   = regexp.MustCompile(`[<(]\s*[A-Z][A-Za-z0-9_]*\s*[,>]`)
)

// looksLikeTS 启发式判断源码是否为 TypeScript（含类型注解语法）。
func looksLikeTS(src string) bool {
	if tsInterfaceRe.MatchString(src) {
		return true
	}
	if tsAnnotRe.MatchString(src) {
		return true
	}
	// 泛型函数/接口调用形态（如 function id<T>(x: T): T）
	return tsGenericRe.MatchString(src)
}

// detectPluginLanguage 探测动态插件源码语言：显式指定优先，否则启发式。
// language: "" | "auto" | "js" | "ts"。
func detectPluginLanguage(src, language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "ts", "typescript":
		return "ts"
	case "js", "javascript":
		return "js"
	}
	if looksLikeTS(src) {
		return "ts"
	}
	return "js"
}

// compilePluginSource 按语言编译插件源码，返回可直接交给 goja 的 JS。
// dir 非空且源码含 ESM 语法（import/export）时走多文件 bundle
// （相对导入内联，非相对包 mock；export default 转 IIFE）。
func compilePluginSource(src, language, sourceName, dir string) (string, error) {
	lang := detectPluginLanguage(src, language)
	if dir != "" && needsBundle(src) {
		return compileTSBundle(src, sourceName, dir, lang)
	}
	switch lang {
	case "ts":
		return compileTS(src, sourceName)
	default:
		return src, nil
	}
}

// importRe 匹配 ES module import 语句（含动态 import() 与 type-only import）。
var importRe = regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?(?:[^'"]*?from\s*)?['"][^'"]+['"]|^\s*import\s*\(`)

// hasImports 判断源码是否含 import 语句（需走 bundle 路径）。
func hasImports(src string) bool {
	return importRe.MatchString(src)
}

// esmStmtRe 匹配 ESM 顶层语句（import / export）——任一存在都需 bundle
// 转 IIFE（否则 export 残留 goja 语法错误）。
var esmStmtRe = regexp.MustCompile(`(?m)^\s*(import\s|export\s)`)

// needsBundle 判断源码是否含 ESM 语法（import 或 export）。
func needsBundle(src string) bool {
	return esmStmtRe.MatchString(src)
}

// dynPluginGlobalName bundle 产物挂载的全局名（IIFE globalName）。
const dynPluginGlobalName = "__dynPlugin"

// mockPackageOnResolve / mockPackageOnLoad：把非相对非绝对导入
// （@deepseek-ai/cordis、schemastery 等 harness 生态包）mock 成空模块。
// 插件代码对这些包的使用几乎都是类型标注（esbuild 自动擦除）或
// 经注入 ctx 访问，运行期不需要真实实现。
//
// ★ 命名导入兼容：空模块只有 default export，`import { x } from 'pkg'`
// 会报 No matching export。此处按 importer 源码提取命名导入清单，
// 为每个被导入名生成 `export const x = undefined`，保证编译通过
// （运行期访问这些 API 会得到 undefined，由插件自行兜底）。
func mockPackageOnResolve(src, dir, sourceName string) func(api.OnResolveArgs) (api.OnResolveResult, error) {
	// 预提取主源码（stdin 无真实文件，importer 是虚拟路径读不到）
	preset := extractAllImportNamesFromSource(src)
	return func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		var names []string
		if m, ok := preset[args.Path]; ok {
			names = m
		}
		// 相对导入文件里的包导入：importer 是 dir 下的真实文件，可读
		if real := filepath.Join(dir, args.Importer); real != sourceName {
			if st, err := os.Stat(real); err == nil && !st.IsDir() {
				names = mergeStringSlice(names, extractImportNames(real, args.Path))
			}
		}
		if len(names) > 0 {
			mockMu.Lock()
			mockNames[args.Path] = mergeStringSlice(mockNames[args.Path], names)
			mockMu.Unlock()
		}
		return api.OnResolveResult{Path: "mock-" + args.Path, Namespace: "mock-pkg"}, nil
	}
}

func mergeStringSlice(a, b []string) []string {
	out := append([]string(nil), a...)
	seen := map[string]bool{}
	for _, n := range a {
		seen[n] = true
	}
	for _, n := range b {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func mockPackageOnLoad(args api.OnLoadArgs) (api.OnLoadResult, error) {
	pkg := strings.TrimPrefix(args.Path, "mock-")
	mockMu.RLock()
	names := append([]string(nil), mockNames[pkg]...)
	mockMu.RUnlock()
	var sb strings.Builder
	sb.WriteString("export default {};\n")
	for _, n := range names {
		sb.WriteString("export const " + n + " = undefined;\n")
	}
	contents := sb.String()
	return api.OnLoadResult{Contents: &contents, Loader: api.LoaderJS}, nil
}

// mockNames 记录各 mock 包被命名导入的导出名（跨 OnResolve 调用合并）。
var (
	mockNames = map[string][]string{}
	mockMu    sync.RWMutex
)

// importFromRe 匹配 `import { a, b as c } from 'pkg'` / `import x, { y } from 'pkg'` /
// `import * as ns from 'pkg'` / `import d from 'pkg'`（type-only 在循环内跳过）。
var importFromRe = regexp.MustCompile(`(?m)^\s*import\s+(?:type\s+)?(?:\{([^}]*)\}|\*\s+as\s+(\w+)|(\w+)(?:\s*,\s*\{([^}]*)\})?)\s*from\s*['"]([^'"]+)['"]`)

// extractAllImportNamesFromSource 从整段源码预提取所有非相对导入的命名清单。
func extractAllImportNamesFromSource(src string) map[string][]string {
	out := map[string][]string{}
	for _, m := range importFromRe.FindAllStringSubmatch(src, -1) {
		// type-only import（esbuild 自动擦除，无需 mock）
		head := strings.TrimSpace(m[0])
		if strings.HasPrefix(head, "import type") || strings.HasPrefix(head, "import\ttype") {
			continue
		}
		pkg := m[5]
		if pkg == "" || strings.HasPrefix(pkg, ".") {
			continue
		}
		add := func(name string) {
			if name == "" || name == "default" {
				return
			}
			if !sliceContains(out[pkg], name) {
				out[pkg] = append(out[pkg], name)
			}
		}
		add(m[2]) // namespace
		add(m[3]) // default 名
		for _, part := range strings.Split(m[1]+","+m[4], ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			name := strings.TrimSpace(strings.SplitN(part, " as ", 2)[0])
			add(name)
		}
	}
	return out
}

func sliceContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// extractImportNames 从 importer 源码提取对 pkg 的命名导入清单。
func extractImportNames(importer, pkg string) []string {
	if importer == "" {
		return nil
	}
	b, err := os.ReadFile(importer)
	if err != nil {
		return nil
	}
	return extractAllImportNamesFromSource(string(b))[pkg]
}

// compileTSBundle 编译含 import 的插件源码：esbuild Build(stdin) + bundle，
// 相对导入（./x、../x）按 dir 解析内联打包；非相对包导入 mock 空模块；
// 输出 IIFE（挂 globalName）追加 `return __dynPlugin.default;`，
// 使插件必须 `export default` 导出插件对象（对齐 harness 插件形态）。
func compileTSBundle(src, sourceName, dir, lang string) (string, error) {
	// 每次构建前清空 mock 命名表（包级全局，防上次构建残留）
	mockMu.Lock()
	mockNames = map[string][]string{}
	mockMu.Unlock()
	loader := api.LoaderJS
	if lang == "ts" {
		loader = api.LoaderTS
	}
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   src,
			ResolveDir: dir,
			Sourcefile: sourceName,
			Loader:     loader,
		},
		Bundle:        true,
		Write:         false,
		Format:        api.FormatIIFE,
		GlobalName:    dynPluginGlobalName,
		Target:        api.ES2020,
		Platform:      api.PlatformNeutral,
		AbsWorkingDir: dir,
		Plugins: []api.Plugin{{
			Name: "mock-packages",
			Setup: func(b api.PluginBuild) {
				b.OnResolve(api.OnResolveOptions{Filter: `^[^./]`}, mockPackageOnResolve(src, dir, sourceName))
				b.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "mock-pkg"}, mockPackageOnLoad)
			},
		}},
		LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		var sb strings.Builder
		for i, e := range result.Errors {
			if i > 0 {
				sb.WriteString("; ")
			}
			loc := ""
			if e.Location != nil {
				loc = fmt.Sprintf("%s 第 %d 行: ", e.Location.File, e.Location.Line+1)
			}
			sb.WriteString(loc + e.Text)
		}
		return "", fmt.Errorf("%s", sb.String())
	}
	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("bundle 无输出")
	}
	js := string(result.OutputFiles[0].Contents)
	// default 导出对齐插件形态，两种写法都支持：
	//   export default { name, apply(ctx) }           → 对象导出
	//   export default function(ctx) { ... }          → 函数导出（函数即 apply，
	//     harness 生态插件惯例；函数自带 name/apply 属性会被 LoadJSDynamic 误认，
	//     因此显式包装成 { name, apply }，插件名用函数名或兜底 cordis-dyn-bundle）
	return js + "\nvar __dynPluginResult = " + dynPluginGlobalName + ".default;" +
		"\nif (typeof __dynPluginResult === 'function') {" +
		"\n  return { name: __dynPluginResult.name || 'cordis-dyn-bundle', apply: __dynPluginResult };" +
		"\n}\nreturn __dynPluginResult;", nil
}
