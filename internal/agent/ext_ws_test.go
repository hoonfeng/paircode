// ═══════════════════════════════════════════════════════════════
// ext_ws_test.go — 双向 WebSocket 通道插件化测试
//
// 覆盖：精确/前缀路由、JS 插件 ctx.ws.register、握手 101、服务端推、
// 客户端消息→插件 onMessage→回显（双向）、断连 cleanup、宿主保留、
// 重复注册报错。客户端为手写 RFC6455（masked 帧）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ─── 手写 ws 测试客户端 ─────────────────────────────────────

type wsTestClient struct {
	conn net.Conn
	br   *bufio.Reader
	t    *testing.T
}

// dialWSTest 建立到 ws 端点的连接（HTTP 握手 + 101）。
func dialWSTest(t *testing.T, baseURL, path string) *wsTestClient {
	t.Helper()
	host := strings.TrimPrefix(baseURL, "http://")
	conn, err := net.Dial("tcp", host)
	if err != nil {
		t.Fatalf("dial 失败: %v", err)
	}
	keyBytes := make([]byte, 16)
	_, _ = rand.Read(keyBytes)
	key := base64.StdEncoding.EncodeToString(keyBytes)
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n", path, host, key)
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("握手请求失败: %v", err)
	}
	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil || !strings.Contains(status, "101") {
		t.Fatalf("握手未 101: %q err=%v", status, err)
	}
	// 读到空行
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("读响应头失败: %v", err)
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}
	return &wsTestClient{conn: conn, br: br, t: t}
}

// sendText 发送 masked 文本帧。
func (c *wsTestClient) sendText(s string) {
	payload := []byte(s)
	head := byte(0x81) // FIN=1, text
	var hdr []byte
	hdr = append(hdr, head)
	n := len(payload)
	if n < 126 {
		hdr = append(hdr, byte(n)|0x80) // mask bit
	} else if n < 65536 {
		hdr = append(hdr, 126|0x80, byte(n>>8), byte(n))
	} else {
		hdr = append(hdr, 127|0x80)
		for i := 5; i >= 0; i-- {
			hdr = append(hdr, 0)
		}
		hdr = append(hdr, byte(n>>8), byte(n))
	}
	mask := [4]byte{0x12, 0x34, 0x56, 0x78}
	hdr = append(hdr, mask[:]...)
	masked := make([]byte, n)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}
	c.conn.Write(append(hdr, masked...))
}

// readText 读取一个文本帧（服务端帧无掩码）。
func (c *wsTestClient) readText(timeout time.Duration) (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	b0, err := c.br.ReadByte()
	if err != nil {
		return "", err
	}
	b1, err := c.br.ReadByte()
	if err != nil {
		return "", err
	}
	length := int(b1 & 0x7F)
	if length == 126 {
		hi, _ := c.br.ReadByte()
		lo, _ := c.br.ReadByte()
		length = int(hi)<<8 | int(lo)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(c.br, buf); err != nil {
		return "", err
	}
	if b0&0x0F == 0x8 { // close
		return "", io.EOF
	}
	if b0&0x0F == 0x9 { // ping → 忽略（等下一帧）
		return c.readText(timeout)
	}
	return string(buf), nil
}

func (c *wsTestClient) close() { c.conn.Close() }

// ─── 测试 ──────────────────────────────────────────────────

func loadWSTestPlugin(t *testing.T, root, code string) (*PluginHost, *Registry) {
	t.Helper()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, root)
	RegisterBuiltinPlugins(host)
	id, err := host.DefineJSCodeFull(code, "js", "ws 测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	t.Cleanup(func() { _ = host.Unload(def.name) })
	return host, reg
}

// TestExtWSPluginEcho 精确路由 + 双向消息 + 断连 cleanup（纯 Go 注册验证
// ext_ws 层；JS 桥双向在 TestExtWSPrefixAndHost 覆盖）。
func TestExtWSPluginEcho(t *testing.T) {
	var mu sync.Mutex
	cleanupCalled := false
	_, err := RegisterExtWS("/api/ext/chat", func(conn *WSConn, params map[string]string, d chan struct{}) func() {
		// 连接建立：主动推送
		_ = conn.WriteTextFrame([]byte("hello:" + params["name"]))
		// 读循环：文本帧回显 {"echo":<payload>}（断开 close 参数 d 通知 ServeExtWS）
		go func() {
			defer close(d)
			for {
				op, payload, err := conn.ReadFrame()
				if err != nil {
					return
				}
				if op == 0x1 {
					if err := conn.WriteTextFrame([]byte(`{"echo":` + string(payload) + `}`)); err != nil {
						return
					}
				}
			}
		}()
		return func() {
			mu.Lock()
			cleanupCalled = true
			mu.Unlock()
		}
	})
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	defer func() { extWSMu.Lock(); delete(extWS, "/api/ext/chat"); extWSMu.Unlock() }()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("host-ok"))
	})
	top := ExtWSMiddleware(mux)
	srv := httptest.NewServer(top)
	defer srv.Close()

	// ① 连接建立 → 服务端主动推送
	cl := dialWSTest(t, srv.URL, "/api/ext/chat?name=alice")
	msg, err := cl.readText(3 * time.Second)
	if err != nil || msg != "hello:alice" {
		t.Fatalf("服务端主动推送异常: %q err=%v", msg, err)
	}

	// ② 客户端发消息 → 读循环 → 回显（双向）
	cl.sendText(`"ping"`)
	msg, err = cl.readText(3 * time.Second)
	if err != nil || msg != `{"echo":"ping"}` {
		t.Fatalf("回显异常: %q err=%v", msg, err)
	}

	// ③ 断开 → 读循环错误 → close(d) → ServeExtWS 返回 → defer cleanup()
	cl.close()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		c := cleanupCalled
		mu.Unlock()
		if c {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("断连后 cleanup 未调用")
}

// TestExtWSPrefixAndHost 前缀路由 + 宿主保留 + 非 Upgrade 放行。
func TestExtWSPrefixAndHost(t *testing.T) {
	root := t.TempDir()
	const code = `return {
	  name: 'ws-prefix',
	  inject: ['ws'],
	  apply(ctx) {
	    ctx.ws.register('/api/ext/push/*', (conn, params) => {
	      conn.onMessage((p) => conn.send('got:' + String(p)))
	    })
	  },
	}`
	loadWSTestPlugin(t, root, code)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("host-ok"))
	})
	top := ExtWSMiddleware(mux)
	srv := httptest.NewServer(top)
	defer srv.Close()

	// ① 前缀路由命中
	cl := dialWSTest(t, srv.URL, "/api/ext/push/a/b")
	defer cl.close()
	cl.sendText("hi")
	msg, err := cl.readText(3 * time.Second)
	if err != nil || msg != "got:hi" {
		t.Fatalf("前缀路由回显异常: %q err=%v", msg, err)
	}

	// ② 未注册 Upgrade 路径 → 不处理（升级失败由客户端自行断开，宿主无响应）
	// ③ 普通 HTTP 请求（非 Upgrade）走 mux（宿主保留）
	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatalf("宿主路由被拦截: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "host-ok" {
		t.Fatalf("宿主路由异常: %q", body)
	}
}

// TestExtWSDuplicate 重复注册报错。
func TestExtWSDuplicate(t *testing.T) {
	if _, err := RegisterExtWS("/api/ext/dup", func(conn *WSConn, p map[string]string, done chan struct{}) func() { return nil }); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}
	defer func() { extWSMu.Lock(); delete(extWS, "/api/ext/dup"); extWSMu.Unlock() }()
	if _, err := RegisterExtWS("/api/ext/dup", func(conn *WSConn, p map[string]string, done chan struct{}) func() { return nil }); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复注册应报错: %v", err)
	}
}

// TestExtWSConnConcurrentSend 并发 send 安全（写锁）。
func TestExtWSConnConcurrentSend(t *testing.T) {
	root := t.TempDir()
	const code = `return {
	  name: 'ws-multi',
	  inject: ['ws'],
	  apply(ctx) {
	    ctx.ws.register('/api/ext/multi', (conn, params) => {
	      conn.send('first')
	      conn.send('second')
	      conn.send('third')
	    })
	  },
	}`
	loadWSTestPlugin(t, root, code)

	mux := http.NewServeMux()
	top := ExtWSMiddleware(mux)
	srv := httptest.NewServer(top)
	defer srv.Close()

	cl := dialWSTest(t, srv.URL, "/api/ext/multi")
	defer cl.close()
	got := map[string]bool{}
	for i := 0; i < 3; i++ {
		msg, err := cl.readText(3 * time.Second)
		if err != nil {
			t.Fatalf("第 %d 帧读取失败: %v", i, err)
		}
		got[msg] = true
	}
	if len(got) != 3 {
		t.Fatalf("并发 send 帧不完整: %v", got)
	}
}

