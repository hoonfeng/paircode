// donefast_test.go — done 即发快路径（2026-09-17）单元测试：锁定 onStreamEvent
// 对 done 事件的分发条件（快路径通道投递 vs 流式/忙对齐路径），防止回归：
//   - 非流式 + 已登记 + done → 投递（Content 透传）；
//   - 活跃流式会话 → 不投递（内容已由 OnFlush 发出，防重复发送）；
//   - drain 相位 → 不投递（前序任务的 done 只用于结束 drain）；
//   - 未登记通道 → 安全忽略（不 panic、不阻塞）。
package bridge

import (
	"testing"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/stream"
)

func TestDoneFastPathDelivery(t *testing.T) {
	b := &Bridge{log: func(string, ...any) {}}
	b.es = testES()

	// ① 已登记 + 非流式 → done 投递（content 透传）；其余类型事件不投
	ch := make(chan stream.Event, 1)
	b.mu.Lock()
	b.replyDoneCh = ch
	b.mu.Unlock()
	b.onStreamEvent(stream.Event{Type: "content", Content: "过程文本"})
	select {
	case <-ch:
		t.Fatal("content 事件不应进入快路径通道")
	default:
	}
	b.onStreamEvent(stream.Event{Type: "done", Content: "最终文本"})
	select {
	case ev := <-ch:
		if ev.Content != "最终文本" {
			t.Fatalf("done 事件应透传 Content，得 %q", ev.Content)
		}
	default:
		t.Fatal("done 事件应投递到快路径通道")
	}

	// ② 活跃流式会话 → 不投递（走 s.Feed 收尾，防与 OnFlush 已发内容重复）
	sess := stream.New(stream.Options{
		OnFlush: func(string, stream.FlushMeta) {},
		Logf:    func(string, ...any) {},
	})
	b.mu.Lock()
	b.activeSession = sess
	b.mu.Unlock()
	b.onStreamEvent(stream.Event{Type: "done", Content: "流式"})
	select {
	case <-ch:
		t.Fatal("流式模式 done 不应进入快路径通道")
	default:
	}
	sess.Abort()
	b.mu.Lock()
	b.activeSession = nil
	b.mu.Unlock()

	// ③ drain 相位 → 不投递（前序任务的 done 只结束 drain）
	dd := make(chan struct{})
	b.mu.Lock()
	b.draining = true
	b.drainDone = dd
	b.mu.Unlock()
	b.onStreamEvent(stream.Event{Type: "done", Content: "前序任务"})
	select {
	case <-ch:
		t.Fatal("drain 相位 done 不应进入快路径通道")
	default:
	}
	select {
	case <-dd:
		// 正确：done 应结束 drain
	default:
		t.Fatal("drain 相位的 done 应关闭 drainDone")
	}
	b.mu.Lock()
	still := b.draining
	b.mu.Unlock()
	if still {
		t.Fatal("done 后 draining 应为 false")
	}

	// ④ 未登记通道 + 非流式 → 安全忽略（不 panic、不阻塞）
	b.mu.Lock()
	b.replyDoneCh = nil
	b.mu.Unlock()
	b.onStreamEvent(stream.Event{Type: "done", Content: "无人接收"})
}
