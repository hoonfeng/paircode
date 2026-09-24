package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestWebDebugTimeoutMs 验证 web_debug 的 timeout 参数：
//   - timeoutMs>0 显式生效（小于页面耗时→超时失败；大于→成功）
//   - 不传（0）走默认 30s 逻辑（页面 8s 应成功——默认行为未收紧）
//
// 注：totalTimeout 约束的是页面创建后的导航/交互阶段；Chromium 冷启动
// （本机 ~12s）在 rod 启动阶段之外，不计入 timeout——断言只要求
// 「短 timeout 必失败、长 timeout 必成功」的判定正确。
func TestWebDebugTimeoutMs(t *testing.T) {
	pageDelay := 8 * time.Second
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ★ 慢页面必须响应请求取消：客户端（导航超时放弃 / 浏览器关闭）断开后立即返回。
		//   否则 httptest.Server.Close 会一直等待这个 in-flight 请求（连接 state active），
		//   打印 "httptest.Server blocked in Close after 5 seconds"——实测 -count=2 每轮必现，
		//   每轮白等 5s。语义不变：客户端不放弃时仍等满 pageDelay（= 「页面慢」）。
		select {
		case <-r.Context().Done():
			return
		case <-time.After(pageDelay):
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><head><title>T-OK</title></head><body><h1 id='done'>PAGE-READY</h1></body></html>")
	}))
	defer func() {
		srv.CloseClientConnections()
		srv.Close()
	}()

	root := t.TempDir()

	// ── 路径 1：timeoutMs=3000 < 页面 8s → 必须超时失败（证明显式 timeout 生效）──
	t0 := time.Now()
	_, err := webDebugRun(context.Background(), root, srv.URL, webDebugOpts{
		waitMs:    500,
		timeoutMs: 3000,
		vpWidth:   800, vpHeight: 600,
	})
	el := time.Since(t0)
	if err == nil {
		t.Fatalf("路径1失败：timeoutMs=3000 应超时，却成功了")
	}
	if el > 30*time.Second {
		t.Fatalf("路径1失败：耗时 %v 异常（疑似挂死）", el)
	}
	t.Logf("路径1 通过：timeoutMs=3000 → %.1fs 超时失败（符合预期）: %v", el.Seconds(), err)

	// ── 路径 2：timeoutMs=20000 > 页面 8s → 必须成功 ──
	t0 = time.Now()
	out, err := webDebugRun(context.Background(), root, srv.URL, webDebugOpts{
		waitMs:    500,
		timeoutMs: 20000,
		vpWidth:   800, vpHeight: 600,
		extractText: true,
	})
	el = time.Since(t0)
	if err != nil {
		t.Fatalf("路径2失败：timeoutMs=20000 应成功（页面 8s），却报错: %v", err)
	}
	if el > 40*time.Second {
		t.Fatalf("路径2失败：耗时 %v 异常", el)
	}
	if !strings.Contains(out, "T-OK") && !strings.Contains(out, "PAGE-READY") {
		t.Logf("路径2 输出（无标题也接受，重点是不超时）: %.200s", out)
	}
	t.Logf("路径2 通过：timeoutMs=20000 → %.1fs 成功返回", el.Seconds())

	// ── 路径 3：timeoutMs=0（缺省）→ 默认 30s，页面 8s 应成功（旧行为兼容）──
	t0 = time.Now()
	_, err = webDebugRun(context.Background(), root, srv.URL, webDebugOpts{
		waitMs:    500,
		timeoutMs: 0,
		vpWidth:   800, vpHeight: 600,
	})
	el = time.Since(t0)
	if err != nil {
		t.Fatalf("路径3失败：缺省 timeout（默认 30s）下 8s 页面应成功，却报错: %v", err)
	}
	t.Logf("路径3 通过：缺省 timeout → %.1fs 成功返回（默认 30s 逻辑保留）", el.Seconds())
}
