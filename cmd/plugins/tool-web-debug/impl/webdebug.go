// web_debug 工具：一站式网页验证——打开 URL、捕获控制台错误、截图、执行 JS、交互。
//
// 解决问题：agent 写完前端代码后只靠编译验证，无法发现运行时错误（JS 异常、
// 白屏、接口 404、样式错乱等）。web_debug 在一个无头浏览器会话中完成：
//   1. 打开 URL，监听所有 console 消息（error/warning/log/info/debug）和页面异常
//   2. 监听网络请求失败（404/500/CORS/超时）
//   3. 提取页面 DOM 结构概览（元素数量、关键标签层次）
//   4. 可选：在指定输入框输入文字（type_selector + type_text）
//   5. 可选：点击元素（click_selector）
//   6. 等待指定时间让异步操作完成
//   7. 可选：提取页面可见文字（text_extract）
//   8. 可选：执行任意 JS 并返回结果（eval）
//   9. 截图保存到 screenshots/ 目录，返回文件路径供 image_analyze 进一步分析
//
// 依赖：go-rod/rod（与 headless.go 共用）。

package impl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	. "github.com/hoonfeng/paircode/pkg/toolbin"
)

// consoleMessage 表示一条浏览器控制台消息。
type consoleMessage struct {
	Type string `json:"type"` // "error", "warning", "info", "log", "debug"
	Text string `json:"text"` // 消息文本
}

// networkFail 表示一次失败的网络请求。
type networkFail struct {
	URL        string `json:"url"`
	Type       string `json:"type"`       // XHR / Fetch / Script / Image / Stylesheet / Other
	ErrorText  string `json:"errorText"`  // 错误描述（如 net::ERR_CONNECTION_REFUSED）
	StatusCode int    `json:"statusCode"` // 0 表示连接级错误（非 HTTP）
}

// registerWebDebugTool 注册 web_debug 工具。
func Register(r *Registry, root string) {
	r.Register(&Tool{
		Name: "web_debug",
		UsageGuide: "一站式网页验证工具：在无头浏览器中打开 URL，检查控制台错误+网络请求失败+截图。" +
			"支持交互操作（click_selector/type_selector+type_text）、JS 求值（eval）、文字提取(text_extract)、" +
			"元素查询(element_query)。前端改动验证首选工具，比手动打开浏览器检查更全自动化。",
		Description: "一站式网页验证工具：在无头浏览器中打开 URL，捕获控制台错误/警告、" +
			"网络请求失败（404/500/CORS）、DOM 结构概览、元素查询（标签/样式/尺寸/可见性/属性）、" +
			"可选输入文字、点击元素、执行 JS、提取页面可见文字，最后截图保存。" +
			"用于验证前端改动是否正常工作（白屏、JS 异常、接口报错、样式错乱等）。" +
			"截图保存到 screenshots/ 目录，返回文件路径可用 image_analyze 进一步分析。" +
			"注意：首次使用会自动下载 Chromium（约 150MB），后续复用缓存。",
		ReadOnly: true,
		Parameters: ObjSchema(Props{
			"url":             StrProp("要验证的网页 URL（如 http://localhost:9090）"),
			"wait":            IntProp("可选：页面加载后等待毫秒数（默认 2000，给 JS 渲染和异步请求时间）"),
			"click_selector":  StrProp("可选：页面加载后点击的 CSS 选择器（如 '#submit-btn'）"),
			"type_selector":   StrProp("可选：要输入文字的 input/textarea 的 CSS 选择器"),
			"type_text":       StrProp("可选：要输入的文字内容（需配合 type_selector）"),
			"text_extract":    BoolProp("可选：提取页面可见纯文本内容（默认 false，内容过多时自动截断）"),
			"element_query":   StrProp("可选：CSS 选择器，查询匹配元素的详细信息（标签/类/样式/尺寸/可见性/属性/文本）"),
			"eval":            StrProp("可选：在页面上执行的 JavaScript 表达式（如 'document.title' 或 'JSON.stringify(window.appState)'）"),
			"screenshot":      BoolProp("可选：是否截图（默认 true）。截图保存到 screenshots/ 目录"),
			"viewport_width":  IntProp("可选：视口宽度（默认 1280）"),
			"viewport_height": IntProp("可选：视口高度（默认 900）"),
		}, "url"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			targetURL := strings.TrimSpace(ArgStr(args, "url"))
			if targetURL == "" {
				return "", fmt.Errorf("参数 url 不能为空")
			}

			waitMs := ArgInt(args, "wait", 2000)
			clickSel := ArgStr(args, "click_selector")
			typeSel := ArgStr(args, "type_selector")
			typeText := ArgStr(args, "type_text")
			evalJS := ArgStr(args, "eval")
			extractText := argBoolDef(args, "text_extract", false)
			elementQuery := ArgStr(args, "element_query")
			takeScreenshot := argBoolDef(args, "screenshot", true)
			vpWidth := ArgInt(args, "viewport_width", 1280)
			vpHeight := ArgInt(args, "viewport_height", 900)

			return webDebugRun(ctx, root, targetURL, webDebugOpts{
				waitMs:       waitMs,
				clickSel:     clickSel,
				typeSel:      typeSel,
				typeText:     typeText,
				evalJS:       evalJS,
				extractText:  extractText,
				elementQuery: elementQuery,
				screenshot:   takeScreenshot,
				vpWidth:      vpWidth,
				vpHeight:     vpHeight,
			})
		},
	})
}

// webDebugOpts 聚合 web_debug 的可选参数。
type webDebugOpts struct {
	waitMs       int
	clickSel     string
	typeSel      string
	typeText     string
	evalJS       string
	extractText  bool
	elementQuery string
	screenshot   bool
	vpWidth      int
	vpHeight     int
}

// webDebugResult 聚合 web_debug 的运行结果。
type webDebugResult struct {
	title        string
	bodyTextLen  int
	consoleMsgs  []consoleMessage
	networkFails []networkFail
	evalResult   string
	pageText     string
	domOverview  string
	elementInfo  string
	screenshot   string
}

// webDebugRun 执行实际的网页调试流程。
func webDebugRun(ctx context.Context, root, targetURL string, opts webDebugOpts) (string, error) {
	// ── 启动浏览器 ──
	l := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("mute-audio", "1").
		Set("disable-gpu", "1").
		Set("no-first-run", "1").
		Set("no-default-browser-check", "1").
		Set("window-size", fmt.Sprintf("%d,%d", opts.vpWidth, opts.vpHeight))

	launchURL, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("启动 Chromium 失败: %w", err)
	}

	browser := rod.New().ControlURL(launchURL)
	if err := browser.Connect(); err != nil {
		return "", fmt.Errorf("连接浏览器失败: %w", err)
	}
	defer browser.Close()

	// ── 创建页面 ──
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return "", fmt.Errorf("创建页面失败: %w", err)
	}

	// 设置超时
	totalTimeout := 30 * time.Second
	if opts.waitMs > 20000 {
		totalTimeout = time.Duration(opts.waitMs)*time.Millisecond + 10*time.Second
	}
	pageCtx, pageCancel := context.WithTimeout(ctx, totalTimeout)
	defer pageCancel()
	page = page.Context(pageCtx)

	res := webDebugResult{}
	reqURLs := map[string]string{} // requestId → URL（用于关联失败请求）

	// ── 启用网络域（捕获请求失败） ──
	proto.NetworkEnable{}.Call(page)

	// ── 捕获所有控制台消息 + 网络失败 ──
	go page.EachEvent(
		// 网络请求开始（记录 URL 供失败时关联）
		func(e *proto.NetworkRequestWillBeSent) {
			reqURLs[string(e.RequestID)] = e.Request.URL
		},
		// 所有 console 消息
		func(e *proto.RuntimeConsoleAPICalled) {
			text := consoleArgsText(e.Args)
			if text == "" {
				return
			}
			res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
				Type: typeLabel(e.Type),
				Text: text,
			})
		},
		// 未捕获异常
		func(e *proto.RuntimeExceptionThrown) {
			text := ""
			if e.ExceptionDetails != nil {
				if e.ExceptionDetails.Text != "" {
					text = e.ExceptionDetails.Text
				} else if e.ExceptionDetails.Exception != nil {
					text = e.ExceptionDetails.Exception.Description
				}
			}
			if text == "" {
				return
			}
			res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
				Type: "error",
				Text: "未捕获异常: " + text,
			})
		},
		// 网络请求失败
		func(e *proto.NetworkLoadingFailed) {
			rType := "Other"
			if e.Type != "" {
				rType = string(e.Type)
			}
			url := reqURLs[string(e.RequestID)]
			if url == "" {
				url = string(e.RequestID) // 回退
			}
			res.networkFails = append(res.networkFails, networkFail{
				URL:       url,
				Type:      rType,
				ErrorText: e.ErrorText,
			})
		},
	)()
	// 注意：NetworkLoadingFailed 的 URL 通过 requestId 关联，在上面的 EachEvent 中
	// 无法直接拿到 URL，我们在 WaitLoad 后补查最近失败的请求。

	// ── 导航到 URL ──
	if err := page.Navigate(targetURL); err != nil {
		return "", fmt.Errorf("导航失败: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("等待页面加载失败: %w", err)
	}

	// ── 输入文字 ──
	if opts.typeSel != "" && opts.typeText != "" {
		el, err := page.Element(opts.typeSel)
		if err != nil {
			res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("找不到输入元素 '%s': %v", opts.typeSel, err),
			})
		} else {
			if err := el.Input(opts.typeText); err != nil {
				res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
					Type: "error",
					Text: fmt.Sprintf("输入文字失败 '%s': %v", opts.typeSel, err),
				})
			}
		}
	}

	// ── 点击元素 ──
	if opts.clickSel != "" {
		el, err := page.Element(opts.clickSel)
		if err != nil {
			res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("找不到点击元素 '%s': %v", opts.clickSel, err),
			})
		} else {
			if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
				res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
					Type: "error",
					Text: fmt.Sprintf("点击 '%s' 失败: %v", opts.clickSel, err),
				})
			}
		}
	}

	// ── 等待异步操作 ──
	if opts.waitMs > 0 {
		time.Sleep(time.Duration(opts.waitMs) * time.Millisecond)
	}

	// ── 获取页面标题 ──
	titleObj, err := page.Eval(`() => document.title`)
	if err == nil {
		res.title = titleObj.Value.String()
	}

	// ── 检查页面是否白屏 + DOM 概览 ──
	domObj, err := page.Eval(`() => {
		const d = document;
		if (!d || !d.body) return JSON.stringify({ textLen: 0, elCount: 0, overview: "无 body" });
		const text = (d.body.innerText || '').trim();
		const all = d.querySelectorAll('*');
		const tags = {};
		all.forEach(el => {
			const t = el.tagName.toLowerCase();
			tags[t] = (tags[t] || 0) + 1;
		});
		// 提取主要布局标签
		const topTags = ['div','span','p','h1','h2','h3','h4','h5','h6','a','button','input','img','ul','ol','li','table','tr','td','th','form','section','article','nav','header','footer','aside','main','canvas','svg','video','audio','select','textarea','label','iframe'];
		const overview = topTags.filter(t => tags[t]).map(t => t + ':' + tags[t]).join(', ');
		return JSON.stringify({ textLen: text.length, elCount: all.length, overview: overview || '(无匹配标签)' });
	}()`)
	if err == nil {
		// 手动解析 JSON 字符串
		domStr := domObj.Value.String()
		// 简单的文本长度提取
		res.bodyTextLen = extractIntField(domStr, "textLen")
		if res.bodyTextLen == 0 {
			// 回退到旧方式
			btObj, e2 := page.Eval(`() => { const t = document.body ? (document.body.innerText || '').trim() : ''; return t.length; }`)
			if e2 == nil {
				res.bodyTextLen = int(btObj.Value.Int())
			}
		}
		res.domOverview = extractStrField(domStr, "overview")
	}

	// ── 查询元素详细信息（可选） ──
	if opts.elementQuery != "" {
		jsQuery := fmt.Sprintf(`() => {
			const sel = %q;
			const els = document.querySelectorAll(sel);
			if (!els || els.length === 0) return JSON.stringify({ error: "没有找到匹配「" + sel + "」的元素" });
			const important = ['display','visibility','opacity','position','width','height',
				'margin-top','margin-right','margin-bottom','margin-left',
				'padding-top','padding-right','padding-bottom','padding-left',
				'color','background-color','font-size','font-weight','text-align',
				'z-index','overflow','overflow-x','overflow-y',
				'top','left','right','bottom','transform','border-radius',
				'box-shadow','cursor','pointer-events','user-select',
				'grid-template','flex-direction','align-items','justify-content',
				'white-space','text-overflow','line-height',
				'max-height','min-height','max-width','min-width'];
			const results = [];
			const max = Math.min(els.length, 20);
			for (let i = 0; i < max; i++) {
				const el = els[i];
				const cs = window.getComputedStyle(el);
				const rect = el.getBoundingClientRect();
				const info = {
					index: i,
					tag: el.tagName.toLowerCase(),
					id: el.id || '',
					classes: (el.className && typeof el.className === 'string') ? el.className.trim().split(/\s+/) : [],
					visible: cs.display !== 'none' && cs.visibility !== 'hidden' && parseFloat(cs.opacity) > 0,
					rect: { x: Math.round(rect.x), y: Math.round(rect.y), w: Math.round(rect.width), h: Math.round(rect.height) },
					text: (el.innerText || '').trim().substring(0, 200),
					children: el.children.length,
					attrs: {},
					styles: {}
				};
				// 提取关键属性
				for (const a of ['href','src','alt','placeholder','value','type','name','disabled','readonly',
					'aria-label','aria-hidden','data-testid','title','role','target','rel']) {
					if (el.hasAttribute(a)) info.attrs[a] = el.getAttribute(a);
				}
				// 提取关键样式
				for (const s of important) {
					const v = cs.getPropertyValue(s);
					if (v && v !== 'none' && v !== 'normal' && v !== '0px' && v !== 'auto' && v !== 'static') {
						info.styles[s] = v;
					}
				}
				results.push(info);
			}
			const extra = els.length > max ? ('\n（还有 ' + (els.length - max) + ' 个匹配元素未列出）') : '';
			return JSON.stringify({ count: els.length, shown: results.length, elements: results }) + extra;
		}`, opts.elementQuery)
		elObj, err := page.Eval(jsQuery)
		if err == nil {
			res.elementInfo = elObj.Value.String()
		}
	}

	// ── 提取页面可见文字（可选） ──
	if opts.extractText {
		textObj, err := page.Eval(`() => {
			const t = document.body ? (document.body.innerText || '').trim() : '';
			return t.length > 10000 ? t.substring(0, 10000) + '\n\n…（已截断，共' + t.length + '字符）' : t;
		}`)
		if err == nil {
			res.pageText = textObj.Value.String()
		}
	}
	// ── 执行自定义 JS ──
	if opts.evalJS != "" {
		// 【关键】rod 的 Eval 会把 JS 包成 function() { return (JS).apply(this, arguments) }
		// 所以必须传函数表达式（而非 IIFE），否则 .apply() 会报 "not a function"
		// 这里我们用 JSON.stringify 序列化结果，避免复杂对象被 String() 转成 "[object Object]"
		wrapped := fmt.Sprintf("() => { try { const r = %s; return JSON.stringify(r, null, 2); } catch(e) { return 'JS执行错误: ' + (e.message || e); } }", opts.evalJS)
		result, err := page.Eval(wrapped)
		if err != nil {
			res.evalResult = fmt.Sprintf("执行失败: %v", err)
		} else {
			res.evalResult = result.Value.String()
		}
	}

	// ── 截图 ──
	if opts.screenshot {
		ssDir := filepath.Join(root, "screenshots")
		os.MkdirAll(ssDir, 0o755)
		ssName := fmt.Sprintf("webdebug_%d.png", time.Now().UnixMilli())
		ssPath := filepath.Join(ssDir, ssName)
		img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
		if err != nil {
			res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("截图失败: %v", err),
			})
		} else {
			if err := os.WriteFile(ssPath, img, 0o644); err != nil {
				res.consoleMsgs = append(res.consoleMsgs, consoleMessage{
					Type: "error",
					Text: fmt.Sprintf("保存截图失败: %v", err),
				})
			} else {
				res.screenshot = ssPath
			}
		}
	}

	// ── 构建返回报告 ──
	return buildWebDebugReport(&res, root, targetURL), nil
}

// buildWebDebugReport 构建可读的网页验证报告。
func buildWebDebugReport(res *webDebugResult, root, targetURL string) string {
	var b strings.Builder
	b.WriteString("# 网页验证报告\n\n")
	b.WriteString(fmt.Sprintf("URL: %s\n", targetURL))
	if res.title != "" {
		b.WriteString(fmt.Sprintf("页面标题: %s\n", res.title))
	}
	b.WriteString(fmt.Sprintf("页面文字长度: %d %s\n", res.bodyTextLen, func() string {
		if res.bodyTextLen == 0 {
			return "⚠️ 白屏！页面无文字内容"
		} else if res.bodyTextLen < 50 {
			return "⚠️ 内容过少，可能渲染异常"
		}
		return "✓ 正常"
	}()))

	// DOM 概览
	if res.domOverview != "" {
		b.WriteString(fmt.Sprintf("DOM 元素: %s\n", res.domOverview))
	}

	// 网络请求失败
	if len(res.networkFails) > 0 {
		b.WriteString("\n## 网络请求失败\n")
		for _, nf := range res.networkFails {
			b.WriteString(fmt.Sprintf("  [%s] %s（错误: %s）\n", nf.Type, nf.URL, nf.ErrorText))
		}
	}

	// 控制台消息
	b.WriteString("\n## 控制台消息\n")
	errors := 0
	warnings := 0
	logs := 0
	for _, msg := range res.consoleMsgs {
		switch msg.Type {
		case "error":
			errors++
			b.WriteString(fmt.Sprintf("  [ERROR] %s\n", msg.Text))
		case "warning":
			warnings++
			b.WriteString(fmt.Sprintf("  [WARN]  %s\n", msg.Text))
		case "log", "info", "debug":
			logs++
		}
	}
	if errors == 0 && warnings == 0 {
		b.WriteString("  ✓ 无错误/警告\n")
	} else {
		b.WriteString(fmt.Sprintf("\n  共 %d 条错误, %d 条警告（另有 %d 条 log/info/debug 已省略）\n", errors, warnings, logs))
	}

	// JS 执行结果
	if res.evalResult != "" {
		b.WriteString("\n## JS 执行结果\n")
		result := res.evalResult
		if len(result) > 2000 {
			result = result[:2000] + "\n…（已截断，共 " + fmt.Sprintf("%d", len(result)) + " 字符）"
		}
		b.WriteString(result + "\n")
	}

	// 元素查询结果
	if res.elementInfo != "" {
		b.WriteString("\n## 元素查询结果\n")
		info := res.elementInfo
		if len(info) > 3000 {
			info = info[:3000] + "\n…（已截断，共 " + fmt.Sprintf("%d", len(info)) + " 字符）"
		}
		b.WriteString(info + "\n")
	}

	// 页面可见文字
	if res.pageText != "" {
		b.WriteString("\n## 页面可见文字\n")
		text := res.pageText
		if len(text) > 3000 {
			text = text[:3000] + "\n…（已截断，共 " + fmt.Sprintf("%d", len(text)) + " 字符）"
		}
		b.WriteString(text + "\n")
	}

	// 截图信息
	if res.screenshot != "" {
		relPath, _ := filepath.Rel(root, res.screenshot)
		b.WriteString("\n## 截图\n")
		b.WriteString(fmt.Sprintf("文件: %s\n", relPath))
		b.WriteString("可用 image_analyze 分析截图内容（颜色/色块/图形），或用 image_ocr 识别文字。\n")
	}

	// 总结
	b.WriteString("\n## 验证结论\n")
	hasCritical := false
	for _, msg := range res.consoleMsgs {
		if msg.Type == "error" {
			hasCritical = true
			break
		}
	}
	if errors > 0 || hasCritical {
		b.WriteString(fmt.Sprintf("❌ 发现 %d 个错误，需要修复\n", errors))
	} else if res.bodyTextLen == 0 {
		b.WriteString("❌ 页面白屏，可能 JS 渲染失败\n")
	} else if len(res.networkFails) > 0 {
		b.WriteString(fmt.Sprintf("❌ 发现 %d 个网络请求失败，需要检查\n", len(res.networkFails)))
	} else if warnings > 0 {
		b.WriteString(fmt.Sprintf("⚠️ 无错误但有 %d 个警告，建议检查\n", warnings))
	} else {
		b.WriteString("✓ 未发现明显问题\n")
	}

	return b.String()
}

// consoleArgsText 从 RuntimeConsoleAPICalled 的参数中提取文本内容。
func consoleArgsText(args []*proto.RuntimeRemoteObject) string {
	text := ""
	for _, arg := range args {
		if arg.Value.String() != "" {
			text += arg.Value.String() + " "
		} else if arg.Description != "" {
			text += arg.Description + " "
		}
	}
	return strings.TrimSpace(text)
}

// typeLabel 把 RuntimeConsoleAPICalledType 转为简短的标签。
func typeLabel(t proto.RuntimeConsoleAPICalledType) string {
	switch t {
	case proto.RuntimeConsoleAPICalledTypeError:
		return "error"
	case proto.RuntimeConsoleAPICalledTypeWarning:
		return "warning"
	case proto.RuntimeConsoleAPICalledTypeInfo:
		return "info"
	case proto.RuntimeConsoleAPICalledTypeDebug:
		return "debug"
	default:
		return "log"
	}
}

// extractIntField 从 JSON 字符串中提取整数字段值（简单解析，不依赖 encoding/json）。
func extractIntField(jsonStr, field string) int {
	marker := fmt.Sprintf(`"%s":`, field)
	idx := strings.Index(jsonStr, marker)
	if idx < 0 {
		return 0
	}
	valStart := idx + len(marker)
	// 跳过空白
	for valStart < len(jsonStr) && (jsonStr[valStart] == ' ' || jsonStr[valStart] == '\t') {
		valStart++
	}
	if valStart >= len(jsonStr) {
		return 0
	}
	// 读取数字
	val := 0
	for valStart < len(jsonStr) && jsonStr[valStart] >= '0' && jsonStr[valStart] <= '9' {
		val = val*10 + int(jsonStr[valStart]-'0')
		valStart++
	}
	return val
}

// extractStrField 从 JSON 字符串中提取字符串字段值（简单解析）。
func extractStrField(jsonStr, field string) string {
	marker := fmt.Sprintf(`"%s":`, field)
	idx := strings.Index(jsonStr, marker)
	if idx < 0 {
		return ""
	}
	valStart := idx + len(marker)
	// 跳过空白
	for valStart < len(jsonStr) && (jsonStr[valStart] == ' ' || jsonStr[valStart] == '\t') {
		valStart++
	}
	if valStart >= len(jsonStr) {
		return ""
	}
	if jsonStr[valStart] != '"' {
		return ""
	}
	valStart++
	end := strings.IndexByte(jsonStr[valStart:], '"')
	if end < 0 {
		return ""
	}
	return jsonStr[valStart : valStart+end]
}

// argBoolDef 从 args 中取 bool 值，不存在则返回 def。
func argBoolDef(args map[string]any, key string, def bool) bool {
	if v, ok := args[key]; ok {
		switch t := v.(type) {
		case bool:
			return t
		case string:
			return t == "true" || t == "1"
		case float64:
			return t != 0
		}
	}
	return def
}
