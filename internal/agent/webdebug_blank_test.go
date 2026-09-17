package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// pageTextLenRe 匹配报告首部的「页面文字长度: N」。
var pageTextLenRe = regexp.MustCompile(`页面文字长度:\s*(\d+)`)

// reportTextLen 从 web_debug 报告中提取内置探测得到的正文文字长度。
func reportTextLen(t *testing.T, rep string) int {
	t.Helper()
	m := pageTextLenRe.FindStringSubmatch(rep)
	if len(m) < 2 {
		t.Fatalf("报告缺少「页面文字长度」行，格式可能已变：\n%.400s", rep)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("解析文字长度失败（%q）: %v", m[1], err)
	}
	return n
}

// TestWebDebugNoBlankFalsePositive 锁定「有文字 + 有元素 ⇒ 非白屏」的核心契约。
//
// 背景（真实缺陷，非臆测）：宿主工具面长期出现「页面文字长度: 0 ⚠️ 白屏！页面无文字内容」
// 的误报（历史会话 2026-08-15 起多次记录），而同一页面用 eval 复算得到 innerText
// 长度数千、元素上千。本测试用当前源码断言：
//  ① 内置 DOM 探测必须拿到真实文字长度（不得为 0）；
//  ② 报告不得输出任何白屏结论；
//  ③ 同一次调用的 element_query 与 eval 必须命中同一页面（内容互相印证，
//     证明「element_query / eval 不在同一页面上下文」的说法不成立或已修复）。
func TestWebDebugNoBlankFalsePositive(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过：需启动无头浏览器（冷启动约 12s）")
	}
	// ★ 正文文案不得含「白屏」二字：报告会回显 element_query 取到的文本，
	//   否则「报告不含白屏字样」的断言会被自己的测试数据污染（本轮实测踩到）。
	const body = "探测回归页面正文"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><title>探测页面</title></head>
<body style="overflow:hidden"><div id="app"><h1 id="t">%s</h1><p>第二段文字</p><span>第三段</span></div></body></html>`, body)
	}))
	defer srv.Close()

	rep, err := webDebugRun(context.Background(), t.TempDir(), srv.URL, webDebugOpts{
		waitMs:       500,
		timeoutMs:    60000,
		evalJS:       `"hasText=" + ((document.body.innerText || "").trim().length > 10)`,
		elementQuery: "#t",
	})
	if err != nil {
		t.Fatalf("webDebugRun 失败: %v", err)
	}

	n := reportTextLen(t, rep)
	if n <= 0 {
		t.Errorf("内置探测文字长度为 0（页面确实有文字）→ 白屏探测退化：\n%s", rep)
	}
	// 精确匹配白屏结论标记（不能只搜「白屏」二字：正文/工具名都可能自然出现）
	assertNoBlankVerdict(t, rep, n)
	if !strings.Contains(rep, body) {
		t.Errorf("element_query 未取到 #t 的文本（同页上下文校验失败）：\n%s", rep)
	}
	if !strings.Contains(rep, "hasText=true") {
		t.Errorf("同一次调用的 eval 未看到页面文字（上下文/包装问题）：\n%s", rep)
	}
	t.Logf("探测文字长度 = %d（非 0 ✓）、无白屏结论 ✓、element_query 与 eval 同页 ✓", n)
}

// TestWebDebugLiveIDEPage 对真实 IDE 页面（独立端口 9097 实例）跑一次内置探测。
// 该页面是 Vue 富应用：body 可见文字数千、元素上千 —— 正是「白屏误报」的真实复现场景
// （httptest 静态页只能覆盖小 DOM 路径）。实例未运行则跳过，不引入外部硬依赖。
func TestWebDebugLiveIDEPage(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过：需启动无头浏览器")
	}
	const addr = "127.0.0.1:9097"
	conn, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
	if err != nil {
		t.Skipf("本地 IDE 测试实例（%s）未运行，跳过：%v", addr, err)
	}
	_ = conn.Close()

	rep, err := webDebugRun(context.Background(), t.TempDir(), "http://"+addr+"/", webDebugOpts{
		waitMs:    3000,
		timeoutMs: 60000,
	})
	if err != nil {
		t.Fatalf("webDebugRun 失败: %v", err)
	}
	n := reportTextLen(t, rep)
	if n <= 100 {
		t.Errorf("真实 IDE 页面文字长度 = %d（应远大于 100）：\n%s", n, rep)
	}
	assertNoBlankVerdict(t, rep, n)
	t.Logf("真实 IDE 页面探测文字长度 = %d ✓、无白屏结论 ✓", n)
}

// assertNoBlankVerdict 断言报告未输出任何「白屏」结论标记。
func assertNoBlankVerdict(t *testing.T, rep string, textLen int) {
	t.Helper()
	for _, marker := range []string{"❌ 页面白屏", "⚠️ 白屏", "白屏！", "白屏探测失败"} {
		if strings.Contains(rep, marker) {
			t.Errorf("误报白屏（命中标记 %q，实际文字长度 %d）：\n%s", marker, textLen, rep)
		}
	}
}
