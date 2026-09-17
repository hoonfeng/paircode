// busy_align_test.go — 忙对齐（2026-09-17）单元测试：
//   - 会话活动状态机（content/tool_call 置忙、done 清除、error 不改、status 校正）；
//   - drain 相位（前序任务输出丢弃、done 结束、幂等）；
//   - waitDrain 出口（drainCh 关闭 / 状态校正 / 超时）。
//
// 背景：微信消息与 PC 端任务共用会话时，桥曾把「前序任务」的输出误当回复发回
// 微信（2026-09-17 现场：t8 收尾输出被当作「1+3等于几？」的回复）。本文件锁定
// 修复行为，防止回归。
package bridge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/stream"
)

func testES() *EventStream {
	return NewEventStream("ws://127.0.0.1:1/ws", time.Second, func(string, ...any) {})
}

// TestConvActivityStateMachine 状态机：content/tool_call 置忙、error 不改、done 清除。
func TestConvActivityStateMachine(t *testing.T) {
	es := testES()
	es.Watch("conv_a", func(stream.Event) {})

	if es.ConvActive("conv_a") {
		t.Fatal("初始应为不忙")
	}
	es.dispatchItem([]byte(`{"convId":"conv_a","type":"content","content":"hi"}`))
	if !es.ConvActive("conv_a") {
		t.Fatal("content 后应为忙")
	}
	es.dispatchItem([]byte(`{"convId":"conv_a","type":"tool_call","tool":"exec_command"}`))
	if !es.ConvActive("conv_a") {
		t.Fatal("tool_call 后应为忙")
	}
	es.dispatchItem([]byte(`{"convId":"conv_a","type":"error","content":"x"}`))
	if !es.ConvActive("conv_a") {
		t.Fatal("error 不应改变忙状态")
	}
	es.dispatchItem([]byte(`{"convId":"conv_a","type":"done","content":"full"}`))
	if es.ConvActive("conv_a") {
		t.Fatal("done 后应为空闲")
	}
	// 未跟踪会话不受影响（不记录、不返回忙）
	es.dispatchItem([]byte(`{"convId":"conv_b","type":"content","content":"x"}`))
	if es.ConvActive("conv_b") {
		t.Fatal("未跟踪会话应为不忙")
	}
	// ping/status 不应 panic、不应影响已跟踪会话
	es.dispatchItem([]byte(`{"type":"ping"}`))
	if es.ConvActive("conv_a") {
		t.Fatal("ping 后 conv_a 应仍空闲")
	}
}

// TestApplyStatusCorrection 断线漏 done 场景：status 快照按 runningConvs 校正。
func TestApplyStatusCorrection(t *testing.T) {
	es := testES()
	es.Watch("conv_a", func(stream.Event) {})
	es.Watch("conv_b", func(stream.Event) {})

	es.dispatchItem([]byte(`{"convId":"conv_a","type":"content","content":"x"}`))
	es.dispatchItem([]byte(`{"convId":"conv_b","type":"content","content":"y"}`))
	raw, _ := json.Marshal(map[string]any{
		"type":               "status",
		"runningConvs":       []string{"conv_b"},
		"runningByWorkspace": map[string]int{},
	})
	es.dispatchItem(raw)
	if es.ConvActive("conv_a") {
		t.Fatal("status 校正后 conv_a 应为空闲（不在 runningConvs）")
	}
	if !es.ConvActive("conv_b") {
		t.Fatal("status 校正后 conv_b 应仍忙（在 runningConvs）")
	}
}

// TestDrainOnStreamEvent drain 相位：前序任务输出被丢弃；done 结束 drain；幂等。
func TestDrainOnStreamEvent(t *testing.T) {
	b := &Bridge{log: func(string, ...any) {}}
	b.es = testES()

	ch := make(chan struct{})
	b.mu.Lock()
	b.draining = true
	b.drainDone = ch
	b.mu.Unlock()

	// drain 期间 content 被丢弃（不进 session、不结束 drain、不 panic）
	b.onStreamEvent(stream.Event{Type: "content", Content: "前序任务输出"})
	select {
	case <-ch:
		t.Fatal("content 不应结束 drain")
	default:
	}

	// done 结束 drain
	b.onStreamEvent(stream.Event{Type: "done"})
	select {
	case <-ch:
	default:
		t.Fatal("done 应关闭 drainDone")
	}
	b.mu.Lock()
	draining := b.draining
	b.mu.Unlock()
	if draining {
		t.Fatal("done 后 draining 应为 false")
	}

	// 幂等：再次 endDrain 不应 panic / 不应重复 close
	b.endDrain("测试重复调用")

	// drain 结束后事件正常路由（session 为 nil 时安全忽略）
	b.onStreamEvent(stream.Event{Type: "content", Content: "本消息回复"})
}

// TestWaitDrain waitDrain 三个出口：drainCh 关闭 / 状态校正 / 超时。
func TestWaitDrain(t *testing.T) {
	b := &Bridge{log: func(string, ...any) {}}
	b.es = testES()
	b.stopCh = make(chan struct{})

	// ① drainCh 已关闭 → 立即 true
	ch1 := make(chan struct{})
	b.mu.Lock()
	b.draining = true
	b.drainDone = ch1
	b.mu.Unlock()
	close(ch1)
	if !b.waitDrain(pendingMsg{}, ch1, time.Now().Add(5*time.Second)) {
		t.Fatal("drainCh 关闭应返回 true")
	}

	// ② 状态校正为空闲（未跟踪 conv → ConvActive=false）→ 首个 ticker 周期后 true
	ch2 := make(chan struct{})
	b.mu.Lock()
	b.draining = true
	b.drainDone = ch2
	b.mu.Unlock()
	start := time.Now()
	if !b.waitDrain(pendingMsg{}, ch2, time.Now().Add(10*time.Second)) {
		t.Fatal("状态校正路径应返回 true")
	}
	if time.Since(start) < time.Second {
		t.Fatal("状态校正应等待至少一个 ticker 周期（1.5s）")
	}

	// ③ 超时 → false（drainCh 不关闭、状态保持忙不可达场景用短 deadline 模拟）
	ch3 := make(chan struct{})
	b.mu.Lock()
	b.draining = true
	b.drainDone = ch3
	b.mu.Unlock()
	if b.waitDrain(pendingMsg{}, ch3, time.Now().Add(120*time.Millisecond)) {
		t.Fatal("超时应返回 false")
	}
}
