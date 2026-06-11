// 联网工具：web_fetch —— 抓取一个 http(s) 网页并返回纯文本（去 HTML 标签），供 agent 查阅文档/网页。
// 纯 Go、无 GUI/CGO 依赖；可用 httptest 离线测。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ─── SearXNG 配置（设置面板「网络服务」→SearXNG 地址）──────────
// 全局：宿主在设置加载/保存时经 SetSearxngURL 注入；为空则 web_search 用 DuckDuckGo。
var (
	searxngMu  sync.RWMutex
	searxngURL string
)

// SetSearxngURL 设置自托管 SearXNG 实例地址（自动去尾斜杠；空=禁用，回退 DuckDuckGo）。
// 复刻参考源 tools/api-config.ts 的 setSearxngUrl。
func SetSearxngURL(u string) {
	searxngMu.Lock()
	searxngURL = strings.TrimRight(strings.TrimSpace(u), "/")
	searxngMu.Unlock()
}

func getSearxngURL() string {
	searxngMu.RLock()
	defer searxngMu.RUnlock()
	return searxngURL
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	reBlockTag    = regexp.MustCompile(`(?i)</(p|div|li|tr|h[1-6]|section|article|header|footer|ul|ol|table|blockquote)>|<br\s*/?>`)
	reTag         = regexp.MustCompile(`(?s)<[^>]+>`)
	reBlankLines  = regexp.MustCompile(`\n{3,}`)
	reInlineWS    = regexp.MustCompile(`[ \t]{2,}`)
)

// htmlToText 把 HTML 粗略转可读纯文本：去 script/style、块标签→换行、去其余标签、解实体、压空白。
func htmlToText(s string) string {
	s = reScriptStyle.ReplaceAllString(s, " ")
	s = reBlockTag.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimSpace(reInlineWS.ReplaceAllString(ln, " "))
	}
	s = strings.Join(lines, "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// registerWebTools 注册联网工具：web_fetch（抓网页）+ web_search（搜索）。
func registerWebTools(r *Registry) {
	r.Register(&Tool{
		Name:        "web_fetch",
		Description: "抓取一个 http(s) 网页并返回其纯文本内容（去除 HTML 标签，超长截断）。用于查阅在线文档、API 参考、网页。",
		Parameters:  objSchema(props{"url": strProp("要抓取的网页 URL（必须 http:// 或 https://）")}, "url"),
		ReadOnly:    true,
		Handler:     webFetch,
	})
	r.Register(&Tool{
		Name:        "web_search",
		Description: "搜索网络，返回前若干条 标题/链接/摘要（已配置 SearXNG 则优先用之，否则 DuckDuckGo）。查文档、报错、库用法、最新信息时用；拿到链接可再用 web_fetch 读全文。",
		Parameters:  objSchema(props{"query": strProp("搜索关键词")}, "query"),
		ReadOnly:    true,
		Handler:     webSearch,
	})
}

const webFetchMaxBody = 2 << 20 // 2MB 读取上限
const webFetchMaxOut = 20000    // 返回文本上限

// httpGetBytes GET 一个 http(s) URL，返回限长 body + 状态码（web_fetch/web_search 共用）。
func httpGetBytes(ctx context.Context, rawurl string, limit int64) ([]byte, int, error) {
	if !strings.HasPrefix(rawurl, "http://") && !strings.HasPrefix(rawurl, "https://") {
		return nil, 0, fmt.Errorf("仅支持 http(s) URL：%q", rawurl)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "companion-agent/1.0")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	return body, resp.StatusCode, err
}

// webFetch 抓取 URL 内容并转纯文本。抽出便于离线测试。
func webFetch(ctx context.Context, args map[string]any) (string, error) {
	u := strings.TrimSpace(argStr(args, "url"))
	body, status, err := httpGetBytes(ctx, u, webFetchMaxBody)
	if err != nil {
		return "", err
	}
	text := htmlToText(string(body))
	if len(text) > webFetchMaxOut {
		text = text[:webFetchMaxOut] + "\n…（内容已截断）"
	}
	return fmt.Sprintf("URL: %s\nHTTP %d %s\n\n%s", u, status, http.StatusText(status), text), nil
}

// ddgSearchURL DuckDuckGo HTML 搜索端点（无需 key）；测试中可替换为 httptest。
var ddgSearchURL = "https://html.duckduckgo.com/html/"

// searxngResp SearXNG JSON API（/search?format=json）的结果结构（只取需要的字段）。
type searxngResp struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

// searxngSearch 查自托管 SearXNG 实例（JSON API），格式化前 8 条 标题/链接/摘要。
// 端点：{base}/search?q=...&format=json&language=auto。失败（含未开放 JSON 输出）返回 error→调用方回退 DDG。
func searxngSearch(ctx context.Context, base, q string) (string, error) {
	endpoint := base + "/search?format=json&language=auto&q=" + url.QueryEscape(q)
	body, status, err := httpGetBytes(ctx, endpoint, webFetchMaxBody)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("SearXNG HTTP %d", status)
	}
	var sr searxngResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", err // 非 JSON（多半实例未开放 json 格式）→ 回退
	}
	if len(sr.Results) == 0 {
		return "", fmt.Errorf("SearXNG 无结果")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "「%s」搜索结果（SearXNG）：\n", q)
	for i, r := range sr.Results {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&b, "\n%d. %s\n   %s\n", i+1, stripTags(r.Title), r.URL)
		if s := stripTags(r.Content); s != "" {
			fmt.Fprintf(&b, "   %s\n", s)
		}
	}
	return b.String(), nil
}

var (
	reDDGAnchor  = regexp.MustCompile(`(?is)<a\b([^>]*class="result__a"[^>]*)>(.*?)</a>`)
	reDDGHref    = regexp.MustCompile(`href="([^"]+)"`)
	reDDGSnippet = regexp.MustCompile(`(?is)class="result__snippet"[^>]*>(.*?)</a>`)
)

type ddgResult struct{ title, url, snippet string }

func stripTags(s string) string {
	return strings.TrimSpace(html.UnescapeString(reTag.ReplaceAllString(s, "")))
}

// decodeDDGHref 解出 DDG 跳转链接里的真实 URL（//duckduckgo.com/l/?uddg=ENCODED）。
func decodeDDGHref(href string) string {
	if i := strings.Index(href, "uddg="); i >= 0 {
		v := href[i+len("uddg="):]
		if j := strings.IndexByte(v, '&'); j >= 0 {
			v = v[:j]
		}
		if dec, err := url.QueryUnescape(v); err == nil {
			return dec
		}
	}
	return href
}

// parseDDGResults 从 DuckDuckGo HTML 结果页粗略抽取 标题/链接/摘要（按出现顺序配对）。
func parseDDGResults(htmlBody string) []ddgResult {
	anchors := reDDGAnchor.FindAllStringSubmatch(htmlBody, -1)
	snips := reDDGSnippet.FindAllStringSubmatch(htmlBody, -1)
	out := make([]ddgResult, 0, len(anchors))
	for i, m := range anchors {
		href := ""
		if h := reDDGHref.FindStringSubmatch(m[1]); h != nil {
			href = h[1]
		}
		r := ddgResult{title: stripTags(m[2]), url: decodeDDGHref(href)}
		if i < len(snips) {
			r.snippet = stripTags(snips[i][1])
		}
		out = append(out, r)
	}
	return out
}

// webSearch 搜索网络，返回前 8 条。已配置 SearXNG（设置→网络服务）则优先用之，
// 失败/无结果再回退 DuckDuckGo（无需 key，尽力而为，依赖 DDG HTML，可能被限流/改版）。
func webSearch(ctx context.Context, args map[string]any) (string, error) {
	q := strings.TrimSpace(argStr(args, "query"))
	if q == "" {
		return "", fmt.Errorf("query 不能为空")
	}
	if base := getSearxngURL(); base != "" { // 优先自托管 SearXNG
		if out, err := searxngSearch(ctx, base, q); err == nil && out != "" {
			return out, nil
		}
		// 失败/空 → 静默回退 DuckDuckGo
	}
	body, status, err := httpGetBytes(ctx, ddgSearchURL+"?q="+url.QueryEscape(q), webFetchMaxBody)
	if err != nil {
		return "", err
	}
	results := parseDDGResults(string(body))
	if len(results) == 0 {
		return fmt.Sprintf("「%s」无搜索结果（HTTP %d；可能被限流或页面改版）。", q, status), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "「%s」搜索结果：\n", q)
	for i, r := range results {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&b, "\n%d. %s\n   %s\n", i+1, r.title, r.url)
		if r.snippet != "" {
			fmt.Fprintf(&b, "   %s\n", r.snippet)
		}
	}
	return b.String(), nil
}
