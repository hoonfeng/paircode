// web_debug 工具：一站式网页验证——打开 URL、捕获控制台错误、截图、执行 JS、交互。
//
// 解决问题：agent 写完前端代码后只靠编译验证，无法发现运行时错误（JS 异常、
// 白屏、接口 404、样式错乱等）。web_debug 在一个无头浏览器会话中完成：
//   1. 打开 URL，监听 console.error/warning 和页面异常
//   2. 可选：在指定输入框输入文字（type_selector + type_text）
//   3. 可选：点击元素（click_selector）
//   4. 等待指定时间让异步操作完成
//   5. 可选：执行任意 JS 并返回结果
//   6. 截图保存到 screenshots/ 目录，返回文件路径供 image_analyze 进一步分析
//
// 依赖：go-rod/rod（与 headless.go 共用）。

package agent

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
)

// consoleMessage 表示一条浏览器控制台消息。
type consoleMessage struct {
	Type string `json:"type"` // "error", "warning", "info", "log"
	Text string `json:"text"` // 消息文本
}

// registerWebDebugTool 注册 web_debug 工具。
func registerWebDebugTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "web_debug",
		Description: "一站式网页验证工具：在无头浏览器中打开 URL，捕获控制台错误/警告，" +
			"可选输入文字、点击元素、执行 JS，最后截图保存。" +
			"用于验证前端改动是否正常工作（白屏、JS 异常、接口报错、样式错乱等）。" +
			"截图保存到 screenshots/ 目录，返回文件路径可用 image_analyze 进一步分析。" +
			"注意：首次使用会自动下载 Chromium（约 150MB），后续复用缓存。",
		Parameters: objSchema(props{
			"url":             strProp("要验证的网页 URL（如 http://localhost:9090）"),
			"wait":            intProp("可选：页面加载后等待毫秒数（默认 2000，给 JS 渲染和异步请求时间）"),
			"click_selector":  strProp("可选：页面加载后点击的 CSS 选择器（如 '#submit-btn'）"),
			"type_selector":   strProp("可选：要输入文字的 input/textarea 的 CSS 选择器"),
			"type_text":       strProp("可选：要输入的文字内容（需配合 type_selector）"),
			"eval":            strProp("可选：在页面上执行的 JavaScript 表达式（如 'document.title' 或 'JSON.stringify(window.appState)'）"),
			"screenshot":      boolProp("可选：是否截图（默认 true）。截图保存到 screenshots/ 目录"),
			"viewport_width":  intProp("可选：视口宽度（默认 1280）"),
			"viewport_height": intProp("可选：视口高度（默认 900）"),
		}, "url"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			targetURL := strings.TrimSpace(argStr(args, "url"))
			if targetURL == "" {
				return "", fmt.Errorf("参数 url 不能为空")
			}

			waitMs := argInt(args, "wait", 2000)
			clickSel := argStr(args, "click_selector")
			typeSel := argStr(args, "type_selector")
			typeText := argStr(args, "type_text")
			evalJS := argStr(args, "eval")
			takeScreenshot := argBoolDef(args, "screenshot", true)
			vpWidth := argInt(args, "viewport_width", 1280)
			vpHeight := argInt(args, "viewport_height", 900)

			return webDebugRun(ctx, root, targetURL, webDebugOpts{
				waitMs:      waitMs,
				clickSel:    clickSel,
				typeSel:     typeSel,
				typeText:    typeText,
				evalJS:      evalJS,
				screenshot:  takeScreenshot,
				vpWidth:     vpWidth,
				vpHeight:    vpHeight,
			})
		},
	})
}

// webDebugOpts 聚合 web_debug 的可选参数。
type webDebugOpts struct {
	waitMs     int
	clickSel   string
	typeSel    string
	typeText   string
	evalJS     string
	screenshot bool
	vpWidth    int
	vpHeight   int
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

	// ── 捕获控制台消息 ──
	var consoleMsgs []consoleMessage
	go page.EachEvent(
		func(e *proto.RuntimeConsoleAPICalled) {
			if e.Type == proto.RuntimeConsoleAPICalledTypeError || e.Type == proto.RuntimeConsoleAPICalledTypeWarning {
				text := ""
				for _, arg := range e.Args {
					if arg.Value.String() != "" {
						text += arg.Value.String() + " "
					} else if arg.Description != "" {
						text += arg.Description + " "
					}
				}
				consoleMsgs = append(consoleMsgs, consoleMessage{
					Type: string(e.Type),
					Text: strings.TrimSpace(text),
				})
			}
		},
		func(e *proto.RuntimeExceptionThrown) {
			text := ""
			if e.ExceptionDetails != nil {
				if e.ExceptionDetails.Text != "" {
					text = e.ExceptionDetails.Text
				} else if e.ExceptionDetails.Exception != nil {
					text = e.ExceptionDetails.Exception.Description
				}
			}
			consoleMsgs = append(consoleMsgs, consoleMessage{
				Type: "error",
				Text: "未捕获异常: " + text,
			})
		},
	)()

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
			consoleMsgs = append(consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("找不到输入元素 '%s': %v", opts.typeSel, err),
			})
		} else {
			if err := el.Input(opts.typeText); err != nil {
				consoleMsgs = append(consoleMsgs, consoleMessage{
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
			consoleMsgs = append(consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("找不到点击元素 '%s': %v", opts.clickSel, err),
			})
		} else {
			if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
				consoleMsgs = append(consoleMsgs, consoleMessage{
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
	pageTitle := ""
	if err == nil {
		pageTitle = titleObj.Value.String()
	}

	// ── 检查页面是否白屏 ──
	bodyTextObj, err := page.Eval(`() => { const t = document.body ? (document.body.innerText || '').trim() : ''; return t.length; }`)
	bodyTextLen := 0
	if err == nil {
		bodyTextLen = int(bodyTextObj.Value.Int())
	}

	// ── 执行自定义 JS ──
	evalResult := ""
	if opts.evalJS != "" {
		// 包装为函数调用以支持表达式和语句
		wrapped := fmt.Sprintf("(() => { try { return String(%s); } catch(e) { return 'JS执行错误: ' + e.message; } })()", opts.evalJS)
		result, err := page.Eval(wrapped)
		if err != nil {
			evalResult = fmt.Sprintf("执行失败: %v", err)
		} else {
			evalResult = result.Value.String()
		}
	}

	// ── 截图 ──
	screenshotPath := ""
	if opts.screenshot {
		ssDir := filepath.Join(root, "screenshots")
		os.MkdirAll(ssDir, 0o755)
		ssName := fmt.Sprintf("webdebug_%d.png", time.Now().UnixMilli())
		ssPath := filepath.Join(ssDir, ssName)
		img, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
		if err != nil {
			consoleMsgs = append(consoleMsgs, consoleMessage{
				Type: "error",
				Text: fmt.Sprintf("截图失败: %v", err),
			})
		} else {
			if err := os.WriteFile(ssPath, img, 0o644); err != nil {
				consoleMsgs = append(consoleMsgs, consoleMessage{
					Type: "error",
					Text: fmt.Sprintf("保存截图失败: %v", err),
				})
			} else {
				screenshotPath = ssPath
			}
		}
	}

	// ── 构建返回结果 ──
	var b strings.Builder
	b.WriteString("# 网页验证报告\n\n")
	b.WriteString(fmt.Sprintf("URL: %s\n", targetURL))
	b.WriteString(fmt.Sprintf("页面标题: %s\n", pageTitle))
	b.WriteString(fmt.Sprintf("页面文字长度: %d %s\n", bodyTextLen, func() string {
		if bodyTextLen == 0 {
			return "⚠️ 白屏！页面无文字内容"
		} else if bodyTextLen < 50 {
			return "⚠️ 内容过少，可能渲染异常"
		}
		return "✓ 正常"
	}()))

	// 控制台消息
	b.WriteString("\n## 控制台消息\n")
	errors := 0
	warnings := 0
	for _, msg := range consoleMsgs {
		if msg.Type == "error" {
			errors++
			b.WriteString(fmt.Sprintf("  [ERROR] %s\n", msg.Text))
		} else if msg.Type == "warning" {
			warnings++
			b.WriteString(fmt.Sprintf("  [WARN]  %s\n", msg.Text))
		}
	}
	if len(consoleMsgs) == 0 {
		b.WriteString("  ✓ 无错误/警告\n")
	} else {
		b.WriteString(fmt.Sprintf("\n  共 %d 条错误, %d 条警告\n", errors, warnings))
	}

	// JS 执行结果
	if evalResult != "" {
		b.WriteString("\n## JS 执行结果\n")
		// 截断过长的结果
		if len(evalResult) > 2000 {
			evalResult = evalResult[:2000] + "\n…（已截断，共 " + fmt.Sprintf("%d", len(evalResult)) + " 字符）"
		}
		b.WriteString(evalResult + "\n")
	}

	// 截图信息
	if screenshotPath != "" {
		relPath, _ := filepath.Rel(root, screenshotPath)
		b.WriteString("\n## 截图\n")
		b.WriteString(fmt.Sprintf("文件: %s\n", relPath))
		b.WriteString("可用 image_analyze 分析截图内容（颜色/色块/图形），或用 image_ocr 识别文字。\n")
	}

	// 总结
	b.WriteString("\n## 验证结论\n")
	if errors > 0 {
		b.WriteString(fmt.Sprintf("❌ 发现 %d 个错误，需要修复\n", errors))
	} else if bodyTextLen == 0 {
		b.WriteString("❌ 页面白屏，可能 JS 渲染失败\n")
	} else if warnings > 0 {
		b.WriteString(fmt.Sprintf("⚠️ 无错误但有 %d 个警告，建议检查\n", warnings))
	} else {
		b.WriteString("✓ 未发现明显问题\n")
	}

	return b.String(), nil
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
