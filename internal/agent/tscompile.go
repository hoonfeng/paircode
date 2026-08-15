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
	"regexp"
	"strings"

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
// dir 非空且源码含 import 时走多文件 bundle（相对导入内联，非相对包 mock）。
func compilePluginSource(src, language, sourceName, dir string) (string, error) {
	lang := detectPluginLanguage(src, language)
	if dir != "" && hasImports(src) {
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

// dynPluginGlobalName bundle 产物挂载的全局名（IIFE globalName）。
const dynPluginGlobalName = "__dynPlugin"

// mockPackageOnResolve / mockPackageOnLoad：把非相对非绝对导入
// （@deepseek-ai/cordis、schemastery 等 harness 生态包）mock 成空模块。
// 插件代码对这些包的使用几乎都是类型标注（esbuild 自动擦除）或
// 经注入 ctx 访问，运行期不需要真实实现。
func mockPackageOnResolve(args api.OnResolveArgs) (api.OnResolveResult, error) {
	return api.OnResolveResult{Path: "mock-empty.ts", Namespace: "mock-pkg"}, nil
}

func mockPackageOnLoad(args api.OnLoadArgs) (api.OnLoadResult, error) {
	contents := "export default {};\n"
	return api.OnLoadResult{Contents: &contents, Loader: api.LoaderTS}, nil
}

// compileTSBundle 编译含 import 的插件源码：esbuild Build(stdin) + bundle，
// 相对导入（./x、../x）按 dir 解析内联打包；非相对包导入 mock 空模块；
// 输出 IIFE（挂 globalName）追加 `return __dynPlugin.default;`，
// 使插件必须 `export default` 导出插件对象（对齐 harness 插件形态）。
func compileTSBundle(src, sourceName, dir, lang string) (string, error) {
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
				b.OnResolve(api.OnResolveOptions{Filter: `^[^./]`}, mockPackageOnResolve)
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
