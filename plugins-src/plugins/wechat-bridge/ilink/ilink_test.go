// Package ilink 测试：发送节流与 ret=-2 冷静期退避（2026-09-16 新增）。
//
// 背景：流式连发打爆服务端频率限制（ret=-2 prepare failed，约 15 分钟恢复）。
// 对策在 SendText/SendItem 统一入口 sendmsg 中实现：
//  1. acquireSendSlot：最小间隔 + 滑动窗口配额（可被 Close 取消）；
//  2. noteSendLimited：-2 进入冷静期（首次 15m、翻倍至上限 30m）后重试。
package ilink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// newSendTestServer 构造 sendmessage 测试服务；respOf 为第 n 次（1 起）请求
// 的响应体（nil 默认 {"ret":0}）。返回服务与「请求数 + 各请求时刻」读取函数。
func newSendTestServer(t *testing.T, respOf func(n int) map[string]any) (*httptest.Server, func() (int, []time.Time)) {
	t.Helper()
	var mu sync.Mutex
	var times []time.Time
	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		n := count
		times = append(times, time.Now())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{"ret": 0}
		if respOf != nil {
			body = respOf(n)
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv, func() (int, []time.Time) {
		mu.Lock()
		defer mu.Unlock()
		return count, append([]time.Time(nil), times...)
	}
}

// TestSendThrottleMinInterval 最小间隔生效：连续 3 次发送间隔 ≥ 最小间隔。
func TestSendThrottleMinInterval(t *testing.T) {
	srv, stat := newSendTestServer(t, nil)
	c := NewClient(srv.URL, "t")
	c.SetSendThrottle(SendThrottleOptions{
		MinInterval: 80 * time.Millisecond, WindowMax: 100, WindowDur: time.Second,
	})
	for i := 0; i < 3; i++ {
		if _, err := c.SendText("u", "hi", "ctx"); err != nil {
			t.Fatalf("SendText(%d): %v", i, err)
		}
	}
	n, times := stat()
	if n != 3 {
		t.Fatalf("请求数 = %d，期望 3", n)
	}
	for i := 1; i < len(times); i++ {
		if d := times[i].Sub(times[i-1]); d < 70*time.Millisecond {
			t.Fatalf("第 %d→%d 次请求间隔 %v < 70ms（节流未生效）", i, i+1, d)
		}
	}
}

// TestSendWindowQuota 滑动窗口配额生效：窗口满后第 3 条等待第 1 条滑出。
func TestSendWindowQuota(t *testing.T) {
	srv, stat := newSendTestServer(t, nil)
	c := NewClient(srv.URL, "t")
	c.SetSendThrottle(SendThrottleOptions{
		MinInterval: time.Millisecond, WindowMax: 2, WindowDur: 150 * time.Millisecond,
	})
	for i := 0; i < 3; i++ {
		if _, err := c.SendText("u", "hi", "ctx"); err != nil {
			t.Fatalf("SendText(%d): %v", i, err)
		}
	}
	n, times := stat()
	if n != 3 {
		t.Fatalf("请求数 = %d，期望 3", n)
	}
	if d := times[2].Sub(times[0]); d < 100*time.Millisecond {
		t.Fatalf("窗口配额未生效：第 1→3 条间隔 %v < 100ms", d)
	}
}

// TestSendLongWindowQuota 长窗口配额生效：短窗口放宽时仍受 15 分钟级长窗口约束。
func TestSendLongWindowQuota(t *testing.T) {
	srv, stat := newSendTestServer(t, nil)
	c := NewClient(srv.URL, "t")
	c.SetSendThrottle(SendThrottleOptions{
		MinInterval: time.Millisecond, WindowMax: 1000, WindowDur: time.Second,
		LongWindowMax: 2, LongWindowDur: 150 * time.Millisecond,
	})
	for i := 0; i < 3; i++ {
		if _, err := c.SendText("u", "hi", "ctx"); err != nil {
			t.Fatalf("SendText(%d): %v", i, err)
		}
	}
	n, times := stat()
	if n != 3 {
		t.Fatalf("请求数 = %d，期望 3", n)
	}
	if d := times[2].Sub(times[0]); d < 100*time.Millisecond {
		t.Fatalf("长窗口配额未生效：第 1→3 条间隔 %v < 100ms", d)
	}
}

// TestSendLimitBackoffRetry ret=-2 冷静期后重试成功。
func TestSendLimitBackoffRetry(t *testing.T) {
	srv, stat := newSendTestServer(t, func(n int) map[string]any {
		if n == 1 {
			return map[string]any{"ret": -2, "errmsg": "prepare failed"}
		}
		return map[string]any{"ret": 0}
	})
	c := NewClient(srv.URL, "t")
	c.SetSendThrottle(SendThrottleOptions{
		MinInterval: time.Millisecond, WindowMax: 1000, WindowDur: time.Second,
		LimitBase: 120 * time.Millisecond, LimitMax: time.Second, LimitMaxRetries: 2,
	})
	resp, err := c.SendText("u", "hi", "ctx")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if resp.Ret != 0 {
		t.Fatalf("resp.Ret = %d，期望重试后成功 0", resp.Ret)
	}
	n, times := stat()
	if n != 2 {
		t.Fatalf("请求数 = %d，期望 2（-2 后重试 1 次）", n)
	}
	if d := times[1].Sub(times[0]); d < 100*time.Millisecond {
		t.Fatalf("冷静期不足：重试间隔 %v < 100ms（期望 ≥ 120ms）", d)
	}
}

// TestSendLimitRetryExhausted 重试耗尽：保留 -2 响应返回上层。
func TestSendLimitRetryExhausted(t *testing.T) {
	srv, stat := newSendTestServer(t, func(n int) map[string]any {
		return map[string]any{"ret": -2, "errmsg": "prepare failed"}
	})
	c := NewClient(srv.URL, "t")
	c.SetSendThrottle(SendThrottleOptions{
		MinInterval: time.Millisecond, WindowMax: 1000, WindowDur: time.Second,
		LimitBase: 40 * time.Millisecond, LimitMax: time.Second, LimitMaxRetries: 1,
	})
	resp, err := c.SendText("u", "hi", "ctx")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if resp.Ret != -2 {
		t.Fatalf("resp.Ret = %d，期望保留 -2（重试耗尽）", resp.Ret)
	}
	if n, _ := stat(); n != 2 {
		t.Fatalf("请求数 = %d，期望 2（1 次原始 + 1 次重试）", n)
	}
}

// TestAcquireSendSlotCancelByClose Close 后等待被中断（不阻塞桥退出）。
func TestAcquireSendSlotCancelByClose(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "t")
	// 直接构造长冷静期（同包测试可访问内部状态）
	c.sendMu.Lock()
	c.limitUntil = time.Now().Add(time.Hour)
	c.sendMu.Unlock()
	go func() {
		time.Sleep(50 * time.Millisecond)
		c.Close()
	}()
	start := time.Now()
	err := c.acquireSendSlot()
	if err == nil {
		t.Fatal("Close 后 acquireSendSlot 应返回取消错误")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("取消耗时 %v，等待未被中断", elapsed)
	}
}
