// 无头浏览器工具：headless_browser —— 利用 Chrome DevTools Protocol 获取 JS 渲染后的页面内容。
//
// 原理：通过 rod 启动/连接 Chromium 实例，加载目标 URL，等待 JavaScript 渲染完成，
// 然后提取 DOM 文本内容。适用于 SPA / 需要 JS 渲染的页面。
//
// 依赖：go-rod/rod（自动管理 Chrome 实例，首次使用自动下载 Chromium）。
// 使用示例：
//
//	headless_browser url=https://example.com
//	headless_browser url=https://example.com wait_selector=#content raw=true

package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	defaultBrowserTimeout = 25 * time.Second       // 总超时
	defaultStableWait     = 800 * time.Millisecond // DOM 稳定等待时间
	defaultMaxStableWait  = 8 * time.Second        // 最大 DOM 稳定等待
	defaultMaxTextLen     = 15000                  // 最大提取文本长度
)

// blockedCIDR 内网地址块（SSRF 防护）。
var blockedCIDR = func() []*net.IPNet {
	var list []*net.IPNet
	for _, cidr := range []string{
		"127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"0.0.0.0/8", "169.254.0.0/16",
		"::1/128", "fc00::/7", "fe80::/10",
	} {
		_, n, _ := net.ParseCIDR(cidr)
		if n != nil {
			list = append(list, n)
		}
	}
	return list
}()

// isPrivateIP 检查 IP 是否为内网地址（SSRF 防护）。
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	for _, cidr := range blockedCIDR {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// headlessOptions 无头浏览器提取选项。
type headlessOptions struct {
	timeout       time.Duration // 总超时
	waitSelector  string        // 等待 CSS 选择器出现
	waitAfterLoad time.Duration // 加载完成后额外等待
	waitStable    bool          // 等待 DOM 稳定（SPA 友好）
	rawMode       bool          // 原始模式：返回所有可见文本
}

// headlessFetch 使用 rod 加载页面并提取文本内容。
func headlessFetch(ctx context.Context, targetURL string, opts headlessOptions) (string, error) {
	// ── SSRF 防护 ──
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("无效 URL: %w", err)
	}
	host := parsed.Hostname()
	if ips, err := net.LookupIP(host); err == nil {
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return "", fmt.Errorf("SSRF 防护：拒绝内网地址 %s (%s)", host, ip)
			}
		}
	}

	// ── 使用 launcher 启动浏览器 ──
	l := launcher.New().
		Headless(true).
		NoSandbox(true).
		Set("mute-audio", "1").
		Set("disable-gpu", "1").
		Set("no-first-run", "1").
		Set("no-default-browser-check", "1").
		Set("window-size", "1280,900")

	launchURL, err := l.Launch()
	if err != nil {
		return "", fmt.Errorf("启动 Chromium 失败: %w", err)
	}

	// ── 创建浏览器实例 ──
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

	// ── 导航到 URL（带超时） ──
	timeout := defaultBrowserTimeout
	if opts.timeout > 0 {
		timeout = opts.timeout
	}

	navCtx, navCancel := context.WithTimeout(ctx, timeout)
	defer navCancel()
	page = page.Context(navCtx)

	if err := page.Navigate(targetURL); err != nil {
		return "", fmt.Errorf("导航到 URL 失败: %w", err)
	}

	// ── 等待页面加载 ──
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("等待页面加载失败: %w", err)
	}

	// ── 等待选择器 ──
	if opts.waitSelector != "" {
		selCtx, selCancel := context.WithTimeout(ctx, 8*time.Second)
		defer selCancel()
		_ = page.Context(selCtx).WaitStable(2 * time.Second)
		_, _ = page.ElementR(opts.waitSelector, "")
	}

	// ── 等待 DOM 稳定（SPA 友好） ──
	if opts.waitStable {
		stableCtx, stableCancel := context.WithTimeout(ctx, defaultMaxStableWait)
		defer stableCancel()
		_ = page.Context(stableCtx).WaitStable(defaultStableWait)
	}

	// ── 额外等待 ──
	if opts.waitAfterLoad > 0 {
		time.Sleep(opts.waitAfterLoad)
	}

	// ── 提取页面标题 ──
	titleObj, err := page.Eval(`() => document.title`)
	pageTitle := ""
	if err == nil {
		pageTitle = titleObj.Value.String()
	}

	// ── 提取页面文本内容 ──
	var pageText string
	if opts.rawMode {
		// 原始模式：返回所有可见文本
		result, err := page.Eval(fmt.Sprintf(`
			() => {
				try {
					const body = document.body;
					let text = body ? (body.innerText || body.textContent || '') : '';
					text = text.replace(/\s+/g, ' ').trim();
					const maxLen = %d;
					if (text.length > maxLen) {
						text = text.slice(0, maxLen) + '\\n\\n[... 结果已截断，共 ' + text.length + ' 字符 ...]';
					}
					return JSON.stringify({ text, charCount: text.length });
				} catch(e) {
					return JSON.stringify({ text: '提取失败: ' + e.message, charCount: 0 });
				}
			}
		`, defaultMaxTextLen))
		if err == nil {
			pageText = result.Value.String()
		}
	} else {
		// 默认模式：启发式过滤噪音
		result, err := page.Eval(fmt.Sprintf(`
			() => {
				try {
					const CLONE = document.body.cloneNode(true);
					const REMOVE_SELECTORS = [
						'script', 'style', 'noscript', 'iframe', 'svg',
						'nav', 'header', 'footer',
						'[role="navigation"]', '[role="banner"]', '[role="contentinfo"]',
						'.nav', '.navbar', '.menu', '.sidebar', '.footer', '.header',
						'#nav', '#navbar', '#menu', '#sidebar', '#footer', '#header',
					];
					REMOVE_SELECTORS.forEach(sel => {
						try { CLONE.querySelectorAll(sel).forEach(el => el.remove()); } catch(e) {}
					});
					try {
						CLONE.querySelectorAll('*').forEach(el => {
							if (el.offsetWidth === 0 || el.offsetHeight === 0) el.remove();
						});
					} catch(e) {}
					const text = (CLONE.innerText || CLONE.textContent || '')
						.replace(/\s+/g, ' ')
						.trim();
					const maxLen = %d;
					if (text.length > maxLen) {
						return JSON.stringify({
							text: text.slice(0, maxLen) + '\\n\\n[... 结果已截断，共 ' + text.length + ' 字符 ...]',
							charCount: text.length,
						});
					}
					return JSON.stringify({ text, charCount: text.length });
				} catch(e) {
					return JSON.stringify({ text: '提取失败: ' + e.message, charCount: 0 });
				}
			}
		`, defaultMaxTextLen))
		if err == nil {
			pageText = result.Value.String()
		}
	}

	// 解析 JSON 结果
	textContent := pageText
	if strings.HasPrefix(textContent, `"`) && strings.HasSuffix(textContent, `"`) {
		textContent = textContent[1 : len(textContent)-1]
		// 转义处理
		textContent = strings.ReplaceAll(textContent, `\"`, `"`)
		textContent = strings.ReplaceAll(textContent, `\\n`, "\n")
		textContent = strings.ReplaceAll(textContent, `\\t`, "\t")
	}

	// 构建返回
	var b strings.Builder
	if pageTitle != "" {
		fmt.Fprintf(&b, "# %s\n", pageTitle)
	}
	fmt.Fprintf(&b, "来源: %s\n", targetURL)
	fmt.Fprintf(&b, "提取方式: headless-browser (rod)\n")
	fmt.Fprintf(&b, "\n---\n\n%s\n", textContent)

	return b.String(), nil
}

// registerHeadlessBrowserTool 注册 headless_browser 工具。
func registerHeadlessBrowserTool(r *Registry) {
	r.Register(&Tool{
		Name: "headless_browser",
		Description: "启动无头 Chrome 浏览器获取 JavaScript 渲染后的页面内容（含 SPA）。" +
			"适用于 web_fetch 无法获取的 JS 动态页面。" +
			"可选参数：wait_selector（等待 CSS 选择器出现）、wait_after_load（加载后额外等待毫秒）、" +
			"raw（原始模式，返回所有可见文本不做过滤）。" +
			"注意：首次使用会自动下载 Chromium（约 150MB），后续复用缓存。",
		Parameters: objSchema(props{
			"url":             strProp("目标 URL（必须 http/https）"),
			"timeout":         intProp("可选：总超时（毫秒），默认 25000"),
			"wait_selector":   strProp("可选：等待 CSS 选择器出现后提取，如 '#content'"),
			"wait_after_load": intProp("可选：加载完成后额外等待（毫秒）"),
			"wait_stable":     boolProp("可选：等待 DOM 稳定（SPA 友好），默认 true"),
			"raw":             boolProp("可选：原始模式，返回所有可见文本不做过滤，默认 false"),
		}, "url"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			targetURL := strings.TrimSpace(argStr(args, "url"))
			if targetURL == "" {
				return "", fmt.Errorf("url 不能为空")
			}
			if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
				return "", fmt.Errorf("仅支持 http(s) URL：%q", targetURL)
			}

			opts := headlessOptions{
				waitStable: true, // 默认 true
				rawMode:    argBool(args, "raw"),
			}
			if t := argInt(args, "timeout", 0); t > 0 {
				opts.timeout = time.Duration(t) * time.Millisecond
			}
			if sel := argStr(args, "wait_selector"); sel != "" {
				opts.waitSelector = sel
			}
			if w := argInt(args, "wait_after_load", 0); w > 0 {
				opts.waitAfterLoad = time.Duration(w) * time.Millisecond
			}
			// wait_stable 参数：显式 false 才关闭
			if v, ok := args["wait_stable"]; ok {
				if b, ok := v.(bool); ok && !b {
					opts.waitStable = false
				}
			}

			return headlessFetch(ctx, targetURL, opts)
		},
	})
}
