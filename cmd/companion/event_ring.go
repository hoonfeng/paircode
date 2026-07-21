package main

import (
	"sync"

	"github.com/hoonfeng/paircode/internal/agent"
)

// eventRing 环形缓冲：存储最近 N 个全局事件，用于断连后回放。
type eventRing struct {
	mu   sync.RWMutex
	buf  []agent.GlobalEvent
	head int   // 写入位置
	size int   // 当前使用量（≤ cap）
	seq  int64 // 全局序列号（单调递增）
}

// newEventRing 创建容量为 cap 的环形缓冲。
func newEventRing(cap int) *eventRing {
	return &eventRing{buf: make([]agent.GlobalEvent, cap)}
}

// push 写入一个事件，返回其序列号。
func (r *eventRing) push(ge agent.GlobalEvent) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	r.buf[r.head] = ge
	r.head = (r.head + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
	return r.seq
}

// replay 回放序列号在 lastSeq 之后的所有事件。
// 返回事件列表和新序列号。
func (r *eventRing) replay(lastSeq int64) ([]agent.GlobalEvent, int64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if lastSeq >= r.seq {
		return nil, r.seq
	}
	// 计算需要回放的事件数
	need := int(r.seq - lastSeq)
	if need > r.size {
		need = r.size
	}
	if need <= 0 {
		return nil, r.seq
	}
	// 从最旧的事件开始回放
	out := make([]agent.GlobalEvent, 0, need)
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + len(r.buf)) % len(r.buf)
		out = append(out, r.buf[idx])
	}
	// 只返回序列号在 lastSeq 之后的事件
	if int(r.seq)-r.size >= int(lastSeq) {
		return out, r.seq
	}
	cut := len(out) - need
	return out[cut:], r.seq
}

// len 返回当前缓冲中的事件数。
func (r *eventRing) len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}
