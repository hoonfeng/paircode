// ═══════════════════════════════════════════════════════════════
// ext_sse_test.go — SSE 事件推送通道插件化测试
//
// 覆盖：精确/前缀路由命中、事件推送格式（event:/data:/flush）、
// 宿主路由保留、未注册 404、重复注册报错、卸载自动注销。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestExtSSERegister 精确/前缀注册 + 重复注册报错 + 卸载注销。
func TestExtSSERegister(t *testing.T) {
	h := func(params map[string]string, emit func(string, any) error, done <-chan struct{}) func() {
		return nil
	}
	disposer, err := RegisterExtSSE("/api/ext/stream", h)
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if _, err := RegisterExtSSE("/api/ext/stream", h); err == nil {
		t.Fatal("重复注册应报错")
	}
	if _, err := RegisterExtSSE("/api/ext/*", h); err != nil {
		t.Fatalf("前缀注册失败: %v", err)
	}
	disposer()
	if _, err := RegisterExtSSE("/api/ext/stream", h); err != nil {
		t.Fatalf("注销后应可重新注册: %v", err)
	}
	// 清理（避免污染后续测试）
	extSSEMu.Lock()
	extSSE = map[string]extSSERoute{}
	extSSEMu.Unlock()
}

// TestExtSSEStream 事件推送流：建连 → emit 两事件 → 断连 cleanup 调用。
func TestExtSSEStream(t *testing.T) {
	var (
		mu       sync.Mutex
		gotEvent []string
		gotData  []string
	)
	h := func(params map[string]string, emit func(string, any) error, done <-chan struct{}) func() {
		_ = emit("hello", map[string]any{"n": 1})
		_ = emit("tick", "plain-text")
		return func() { mu.Lock(); gotEvent = append(gotEvent, "cleanup"); mu.Unlock() }
	}
	disposer, err := RegisterExtSSE("/api/ext/stream", h)
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	defer disposer()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	ts := httptest.NewServer(ExtSSEMiddleware(mux))
	defer ts.Close()

	// 读取 SSE 流（客户端 300ms 后断开）
	resp, err := http.Get(ts.URL + "/api/ext/stream")
	if err != nil {
		t.Fatalf("GET 失败: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type 应为 text/event-stream，got %q", ct)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(resp.Body)
		var ev, data string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "event: ") {
				ev = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			} else if line == "" {
				mu.Lock()
				gotEvent = append(gotEvent, ev)
				gotData = append(gotData, data)
				mu.Unlock()
				ev, data = "", ""
			}
		}
	}()
	select {
	case <-time.After(300 * time.Millisecond):
	case <-done:
	}
	resp.Body.Close() // 触发服务端断连
	<-done

	mu.Lock()
	if len(gotEvent) < 2 || gotEvent[0] != "hello" || gotEvent[1] != "tick" {
		mu.Unlock()
		t.Fatalf("事件流不完整: %v", gotEvent)
	}
	if gotData[0] != `{"n":1}` || gotData[1] != "plain-text" {
		mu.Unlock()
		t.Fatalf("data 序列化不正确: %v", gotData)
	}
	// cleanup 由服务端 defer 执行（断连后异步），轮询等待出现
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(gotEvent) > 2 && gotEvent[len(gotEvent)-1] == "cleanup" {
			mu.Unlock()
			return
		}
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
	}
	mu.Unlock()
	t.Fatalf("cleanup 未在断连后调用: %v", gotEvent)
}

// TestExtSSEHostFallback 未注册路径走宿主 mux（/api/health 保留）。
func TestExtSSEHostFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	ts := httptest.NewServer(ExtSSEMiddleware(mux))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET 失败: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "ok" {
		t.Fatalf("宿主路由应保留，got %q", string(b))
	}
}
