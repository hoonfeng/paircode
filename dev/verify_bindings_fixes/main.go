// Command verify_bindings_fixes 验证 wb-ui 三个 DOM 绑定修复：
//  1. WebView.SetPrefersColorScheme → matchMedia(prefers-color-scheme) 动态化
//  2. CSS.supports 选择器形式真解析（未知伪类返回 false）
//  3. getComputedStyle 级联计算（style 标签 + specificity + inline + CSS 变量）
// 用法：set CGO_ENABLED=1 && go run ./dev/verify_bindings_fixes（cwd=gou-ide 根）
package main

import (
	"fmt"
	"os"

	"wb-ui/webkit"
)

func check(name string, got, want string) bool {
	if got == want {
		fmt.Printf("  ✓ %s = %q\n", name, got)
		return true
	}
	fmt.Printf("  ✗ %s = %q (want %q)\n", name, got, want)
	return false
}

func main() {
	passed, failed := 0, 0
	verdict := func(ok bool) {
		if ok {
			passed++
		} else {
			failed++
		}
	}

	wv := webkit.NewWebView()
	wv.Resize(1000, 700)

	// ── 修复 1：prefers-color-scheme 动态化 ────────────────
	fmt.Println("[1] prefers-color-scheme 动态化")
	wv.SetPrefersColorScheme("dark")
	wv.LoadHTML(`<!DOCTYPE html><html><body></body></html>`)
	ev, err := wv.EvalJS(`matchMedia('(prefers-color-scheme: dark)').matches`)
	if err != nil {
		fmt.Println("  ✗ EvalJS error:", err)
		failed++
	} else {
		verdict(check("matchMedia(dark).matches", ev.ToString(), "true"))
	}
	ev, _ = wv.EvalJS(`matchMedia('(prefers-color-scheme: light)').matches`)
	verdict(check("matchMedia(light).matches", ev.ToString(), "false"))

	wv.SetPrefersColorScheme("light")
	ev, _ = wv.EvalJS(`matchMedia('(prefers-color-scheme: light)').matches`)
	verdict(check("切回 light 后 light.matches", ev.ToString(), "true"))

	// ── 修复 2：CSS.supports 选择器形式 ────────────────────
	fmt.Println("[2] CSS.supports 选择器形式")
	for _, c := range []string{
		"typeof CSS.supports", "String(CSS.supports('foo'))",
		"(function(){try{return CSS.supports('foo')}catch(e){return 'ERR:'+e}})()",
		"CSS.supports('foo')", "CSS.supports('.foo')", "CSS.supports('a > b')",
		"CSS.supports('.foo:hover')", "CSS.supports('div[data-v-abc]::before')",
		"CSS.supports('a > b + c ~ d')",
	} {
		ev, err = wv.EvalJS(c)
		if err != nil {
			fmt.Printf("  ✗ %s → error %v\n", c, err)
		} else {
			fmt.Printf("  ? %s = %q\n", c, ev.ToString())
		}
	}
	ev, _ = wv.EvalJS(`CSS.supports('.foo:hover')`)
	verdict(check("CSS.supports('.foo:hover')", ev.ToString(), "true"))
	ev, _ = wv.EvalJS(`CSS.supports('a > b + c ~ d')`)
	verdict(check("CSS.supports('a > b + c ~ d')", ev.ToString(), "true"))
	ev, _ = wv.EvalJS(`CSS.supports('.foo:unknown-pseudo')`)
	verdict(check("CSS.supports('.foo:unknown-pseudo')", ev.ToString(), "false"))
	ev, _ = wv.EvalJS(`CSS.supports('div[data-v-abc]::before')`)
	verdict(check("CSS.supports('div[data-v-abc]::before')", ev.ToString(), "true"))
	ev, _ = wv.EvalJS(`CSS.supports('color: red')`)
	verdict(check("CSS.supports('color: red')（无括号声明，应为 false）", ev.ToString(), "false"))
	ev, _ = wv.EvalJS(`CSS.supports('(display: flex)')`)
	verdict(check("CSS.supports('(display: flex)')", ev.ToString(), "true"))

	// ── 修复 3：getComputedStyle 级联计算 ──────────────────
	fmt.Println("[3] getComputedStyle 级联计算")
	html := `<!DOCTYPE html><html><head><style>
		div { color: blue; font-size: 20px; }
		.high { color: red; --main-bg: #fff; --accent: #ff0; }
		.high { font-size: 30px; }
	</style></head><body>
		<div id="a" class="high"></div>
		<div id="b" class="high" style="color: green; --main-bg: #000"></div>
	</body></html>`
	wv.LoadHTML(html)
	ev, err = wv.EvalJS(`getComputedStyle(document.getElementById('a')).getPropertyValue('color')`)
	if err != nil {
		fmt.Println("  ✗ EvalJS error:", err)
		failed++
	} else {
		verdict(check("id=a color（.high 覆盖 div）", ev.ToString(), "red"))
	}
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('a')).getPropertyValue('font-size')`)
	verdict(check("id=a font-size（同特异性后者胜）", ev.ToString(), "30px"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('a')).getPropertyValue('--main-bg')`)
	verdict(check("id=a --main-bg（CSS 变量级联）", ev.ToString(), "#fff"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('a')).getPropertyValue('--accent')`)
	verdict(check("id=a --accent", ev.ToString(), "#ff0"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('a')).color`)
	verdict(check("id=a .color 直接属性", ev.ToString(), "red"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('b')).getPropertyValue('color')`)
	verdict(check("id=b color（inline 覆盖样式表）", ev.ToString(), "green"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('b')).getPropertyValue('--main-bg')`)
	verdict(check("id=b --main-bg（inline 覆盖变量）", ev.ToString(), "#000"))
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('b')).getPropertyValue('font-size')`)
	verdict(check("id=b font-size（样式表仍生效）", ev.ToString(), "30px"))

	// !important 优先级
	html2 := `<!DOCTYPE html><html><head><style>
		#x { color: blue !important; }
		#x { color: yellow; }
	</style></head><body><div id="x" style="color: red"></div></body></html>`
	wv.LoadHTML(html2)
	ev, _ = wv.EvalJS(`getComputedStyle(document.getElementById('x')).getPropertyValue('color')`)
	verdict(check("!important 高于 inline", ev.ToString(), "blue"))

	fmt.Printf("\n结果：%d 通过，%d 失败\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
