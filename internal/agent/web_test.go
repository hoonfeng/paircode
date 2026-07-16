package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWebFetch 离线（httptest）验证：抓取 HTML → 去 script/style/标签、解实体、保留正文文本。
func TestWebFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<html><head><style>.x{color:red}</style></head>`+
			`<body><h1>标题</h1><p>Hello &amp; world</p><script>alert('bad')</script></body></html>`)
	}))
	defer srv.Close()

	out, err := webFetch(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "标题") || !strings.Contains(out, "Hello & world") {
		t.Errorf("正文缺失：%q", out)
	}
	if strings.Contains(out, "alert('bad')") || strings.Contains(out, "color:red") || strings.Contains(out, "<h1>") {
		t.Errorf("未去除 script/style/标签：%q", out)
	}
}

// TestWebSearch 离线（httptest 替换 ddgSearchURL）：解析 DDG HTML 结果 → 标题/真实链接/摘要。
func TestWebSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<div class="result">`+
			`<a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F&rut=x">Go 官网</a>`+
			`<a class="result__snippet" href="#">Build simple &amp; reliable software</a></div>`)
	}))
	defer srv.Close()
	old := ddgSearchURL
	ddgSearchURL = srv.URL
	defer func() { ddgSearchURL = old }()

	out, err := webSearch(context.Background(), map[string]any{"query": "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Go 官网") || !strings.Contains(out, "https://go.dev/") {
		t.Errorf("标题/解码链接缺失：%q", out)
	}
	if !strings.Contains(out, "Build simple & reliable software") {
		t.Errorf("摘要缺失：%q", out)
	}
}

func TestWebSearchEmptyQuery(t *testing.T) {
	if _, err := webSearch(context.Background(), map[string]any{"query": "  "}); err == nil {
		t.Error("空 query 应报错")
	}
}

// TestSearxngPreferred 配置 SearXNG 后 web_search 优先用之（JSON API），不走 DDG。
func TestSearxngPreferred(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "json" {
			t.Errorf("应带 format=json，得 %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"results":[{"title":"Go 官网","url":"https://go.dev/","content":"可靠的软件"}]}`)
	}))
	defer srv.Close()
	SetSearxngURL(srv.URL + "/") // 带尾斜杠，验证被清理
	defer SetSearxngURL("")

	out, err := webSearch(context.Background(), map[string]any{"query": "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SearXNG") || !strings.Contains(out, "Go 官网") || !strings.Contains(out, "https://go.dev/") {
		t.Errorf("未走 SearXNG 或结果缺失：%q", out)
	}
}

// TestSearxngFallbackToDDG SearXNG 不可用（500/非 JSON）时静默回退 DuckDuckGo。
func TestSearxngFallbackToDDG(t *testing.T) {
	searx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // 未开放 JSON / 故障
	}))
	defer searx.Close()
	ddg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2F&rut=x">回退结果</a>`)
	}))
	defer ddg.Close()
	SetSearxngURL(searx.URL)
	defer SetSearxngURL("")
	old := ddgSearchURL
	ddgSearchURL = ddg.URL
	defer func() { ddgSearchURL = old }()

	out, err := webSearch(context.Background(), map[string]any{"query": "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "回退结果") || strings.Contains(out, "SearXNG") {
		t.Errorf("未回退到 DDG：%q", out)
	}
}

// TestWebFetchRejectsNonHTTP 非 http(s) URL 应拒绝（挡 file:// 等）。
func TestWebFetchRejectsNonHTTP(t *testing.T) {
	if _, err := webFetch(context.Background(), map[string]any{"url": "file:///etc/passwd"}); err == nil {
		t.Error("应拒绝非 http(s) URL")
	}
}

// TestWebFetchRegistered web_fetch 应在默认工具集中（只读）。
func TestWebFetchRegistered(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	tool, ok := r.Get("web_fetch")
	if !ok {
		t.Fatal("web_fetch 未注册")
	}
	if !tool.ReadOnly {
		t.Error("web_fetch 应为只读（免审）")
	}
}
