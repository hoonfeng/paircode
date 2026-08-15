// ═══════════════════════════════════════════════════════════════
// tscompile.go — 内置 TS 编译器（esbuild 纯 Go，无 CGO/npm 依赖）
//
// 用于 JS 动态插件（cordis_define/cordis_run）装载 harness 生态的
// TypeScript 插件源码：TS → ES2020 JS，再交给 goja 执行。
// 不解析 import（单文件模式）；需要 bundle 多文件插件时后续可用
// api.Build 的 stdin+external 模式扩展。
// ═══════════════════════════════════════════════════════════════

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
func compilePluginSource(src, language, sourceName string) (string, error) {
	switch detectPluginLanguage(src, language) {
	case "ts":
		return compileTS(src, sourceName)
	default:
		return src, nil
	}
}
