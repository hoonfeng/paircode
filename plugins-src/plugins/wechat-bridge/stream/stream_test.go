package stream

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// collector 收集回调结果。
type collector struct {
	mu    sync.Mutex
	fls   []string
	full  string
	sum   Summary
	doneN int
}

func (c *collector) onFlush(text string, meta FlushMeta) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fls = append(c.fls, text)
}

func (c *collector) onDone(full string, sum Summary) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.full = full
	c.sum = sum
	c.doneN++
}

func (c *collector) snapshot() ([]string, string, Summary) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fls := append([]string(nil), c.fls...)
	return fls, c.full, c.sum
}

func newTestSession(c *collector) *Session {
	return New(Options{
		FirstMinChars:  5,
		FirstMaxWaitMs: 120,
		NextMinChars:   10,
		NextMaxWaitMs:  200,
		SoftLimit:      60,
		OnFlush:        c.onFlush,
		OnDone:         c.onDone,
	})
}

// 等待完成信号（带超时防测试挂死）。
func waitDone(t *testing.T, s *Session) Summary {
	t.Helper()
	select {
	case sum := <-s.Done():
		return sum
	case <-time.After(3 * time.Second):
		t.Fatal("等待会话完成超时")
		return Summary{}
	}
}

// 1) 句末切分：首片遇句末即发；done 全文核对。
func TestSentenceFlushAndDone(t *testing.T) {
	var c collector
	s := newTestSession(&c)
	s.Feed(Event{Type: "content", Content: "你好世界。"})
	// 首片应在 content 后立即发出（5 字符以上遇句末）
	deadline := time.Now().Add(time.Second)
	for {
		fls, _, _ := c.snapshot()
		if len(fls) >= 1 {
			if fls[0] != "你好世界。" {
				t.Fatalf("首片内容不符: %q", fls[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("首片未按时发出")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Feed(Event{Type: "done", Content: "你好世界。第二段落很长很长。"})
	sum := waitDone(t, s)
	fls, full, _ := c.snapshot()
	joined := strings.Join(fls, "")
	if joined != "你好世界。第二段落很长很长。" {
		t.Fatalf("分片拼接不符: %q", joined)
	}
	if full != "你好世界。第二段落很长很长。" {
		t.Fatalf("OnDone full 不符: %q", full)
	}
	if sum.Reason != "done" {
		t.Fatalf("reason 不符: %s", sum.Reason)
	}
}

// 2) tool_call 丢弃未发缓冲（中间轮不发）。
func TestToolCallDropsBuffer(t *testing.T) {
	var c collector
	s := newTestSession(&c)
	s.Feed(Event{Type: "content", Content: "短句"}) // 未达首片阈值
	s.Feed(Event{Type: "tool_call", Tool: "exec_command", Args: "{}"})
	s.Feed(Event{Type: "content", Content: "最终答复内容。"})
	s.Feed(Event{Type: "done", Content: "最终答复内容。"})
	waitDone(t, s)
	fls, _, _ := c.snapshot()
	joined := strings.Join(fls, "")
	if strings.Contains(joined, "短句") {
		t.Fatalf("中间轮缓冲不应发出: %q", joined)
	}
	if joined != "最终答复内容。" {
		t.Fatalf("最终内容不符: %q", joined)
	}
}

// 3) 定时必发：未遇句末但超过 FirstMaxWaitMs。
func TestTimerFlush(t *testing.T) {
	var c collector
	s := New(Options{
		FirstMinChars:  5,
		FirstMaxWaitMs: 80,
		NextMinChars:   10,
		NextMaxWaitMs:  120,
		SoftLimit:      200,
		OnFlush:        c.onFlush,
		OnDone:         c.onDone,
	})
	s.Feed(Event{Type: "content", Content: "没有句末的长内容啊啊啊啊啊"}) // ≥5 字符但无句末
	deadline := time.Now().Add(2 * time.Second)
	for {
		fls, _, _ := c.snapshot()
		if len(fls) >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("定时首片未发出")
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Feed(Event{Type: "done", Content: ""})
	waitDone(t, s)
}

// 4) abort 后不再发送、OnDone 不调用。
func TestAbortStopsFlush(t *testing.T) {
	var c collector
	s := newTestSession(&c)
	s.Feed(Event{Type: "content", Content: "第一句内容。"})
	time.Sleep(30 * time.Millisecond) // 让首片发出
	s.Abort()
	sum := waitDone(t, s)
	if !sum.Aborted {
		t.Fatalf("摘要 Aborted 应为 true: %+v", sum)
	}
	c.mu.Lock()
	doneN := c.doneN
	c.mu.Unlock()
	if doneN != 0 {
		t.Fatalf("abort 时不应触发 OnDone（实际 %d 次）", doneN)
	}
	// 后续 feed 被忽略
	s.Feed(Event{Type: "content", Content: "后续不应出现。"})
	s.Feed(Event{Type: "done", Content: "后续不应出现。"})
	time.Sleep(50 * time.Millisecond)
	fls, _, _ := c.snapshot()
	for _, f := range fls {
		if strings.Contains(f, "后续") {
			t.Fatalf("abort 后不应再发送: %q", f)
		}
	}
}

// 5) done 全文不以已发为前缀：仅补发剩余缓冲、不重复发。
func TestDoneNotPrefix(t *testing.T) {
	var c collector
	s := newTestSession(&c)
	s.Feed(Event{Type: "content", Content: "已发出的开头。"})
	time.Sleep(30 * time.Millisecond)
	s.Feed(Event{Type: "content", Content: "剩余缓冲段。"})
	s.Feed(Event{Type: "done", Content: "完全不同的全文内容。"})
	waitDone(t, s)
	fls, _, _ := c.snapshot()
	joined := strings.Join(fls, "")
	if !strings.HasPrefix(joined, "已发出的开头。") {
		t.Fatalf("已发内容不应被重复删除: %q", joined)
	}
	if !strings.Contains(joined, "剩余缓冲段。") {
		t.Fatalf("剩余缓冲应补发: %q", joined)
	}
	if strings.Contains(joined, "完全不同的全文内容。") {
		t.Fatalf("非前缀 done 全文不应整段重发: %q", joined)
	}
}

// 6) 软上限强切：超长无句末文本被硬切分片。
func TestSoftLimitHardCut(t *testing.T) {
	var c collector
	s := New(Options{
		FirstMinChars:  5,
		FirstMaxWaitMs: 1000,
		NextMinChars:   10,
		NextMaxWaitMs:  1000,
		SoftLimit:      30,
		OnFlush:        c.onFlush,
		OnDone:         c.onDone,
	})
	long := strings.Repeat("字", 100) // 无任何句末
	s.Feed(Event{Type: "content", Content: long})
	deadline := time.Now().Add(2 * time.Second)
	for {
		fls, _, _ := c.snapshot()
		if len(fls) >= 3 { // 100 字符 / 30 软上限 → 至少 3 片
			break
		}
		if time.Now().After(deadline) {
			fls, _, _ := c.snapshot()
			t.Fatalf("硬切未发生（已发 %d 片）", len(fls))
		}
		time.Sleep(10 * time.Millisecond)
	}
	s.Feed(Event{Type: "done", Content: long})
	waitDone(t, s)
	fls, _, _ := c.snapshot()
	if got := strings.Join(fls, ""); got != long {
		t.Fatalf("拼接不等于原文（len %d vs %d）", len([]rune(got)), len([]rune(long)))
	}
}

// 7) 会话忙错误不终结：等待宿主排队续跑，后续 content/done 正常流转。
func TestBusyErrorDoesNotFinalize(t *testing.T) {
	var c collector
	s := newTestSession(&c)
	s.Feed(Event{Type: "error", Content: "该会话已有运行中的任务"})
	// 不应终结（Done 无信号）
	select {
	case sum := <-s.Done():
		t.Fatalf("忙错误不应终结会话（收到 %+v）", sum)
	case <-time.After(150 * time.Millisecond):
	}
	// 宿主排队续跑后的事件流照常
	s.Feed(Event{Type: "content", Content: "排队后的回复。"})
	s.Feed(Event{Type: "done", Content: "排队后的回复。"})
	sum := waitDone(t, s)
	if sum.Reason != "done" {
		t.Fatalf("reason 不符: %s", sum.Reason)
	}
	fls, full, _ := c.snapshot()
	if joined := strings.Join(fls, ""); joined != "排队后的回复。" {
		t.Fatalf("分片拼接不符: %q", joined)
	}
	if full != "排队后的回复。" {
		t.Fatalf("OnDone full 不符: %q", full)
	}
}
