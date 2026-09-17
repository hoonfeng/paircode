// eventstream.go — 常驻订阅 PairCode /ws 全局事件流。
//
//   - 断线自动重连（指数退避 1s→30s）
//   - Watch(convID, onEvent)：按会话订阅（每账号一个 watch，事件按 convId 路由）
//   - Connected()：连接状态维护，供桥判断「流式通道是否可用」
//   - ★ 忙对齐（2026-09-17）：被 Watch 的会话维护「运行活动状态」
//     （content/tool_call 置忙、done 清除、status 快照按 runningConvs 权威校正），
//     供桥在投喂前判断「会话是否已有任务在跑」（排队对齐，防回复错配）。
//
// 蓝图：M1 Node 原型 temp/wx-bridge/src/eventstream.mjs（线上实测版本）。
package bridge

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/stream"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/wsclient"
)

// EventStream 常驻事件流订阅器（进程内单例，多账号共享一条连接）。
type EventStream struct {
	url         string
	connTimeout time.Duration
	logf        logf

	mu         sync.Mutex
	running    bool
	conn       *wsclient.Client
	connected  bool
	watches    map[string]*watchEntry
	acts       map[string]*convActivity // 被 Watch 会话的运行活动状态（忙对齐用）
	retryDelay time.Duration

	stopCh   chan struct{}
	loopDone chan struct{}
}

type watchEntry struct {
	convID  string
	onEvent func(ev stream.Event)
	closed  bool
}

// convActivity 会话运行活动状态（由事件流 + status 快照共同维护）。
// active=true 表示「该会话有任务在跑」。
type convActivity struct {
	active bool
	lastAt time.Time
}

// NewEventStream 创建事件流（url 形如 ws://127.0.0.1:9090/ws）。
func NewEventStream(url string, connTimeout time.Duration, log logf) *EventStream {
	return &EventStream{
		url:         url,
		connTimeout: connTimeout,
		logf:        log,
		watches:     map[string]*watchEntry{},
		acts:        map[string]*convActivity{},
		retryDelay:  time.Second,
		stopCh:      make(chan struct{}),
		loopDone:    make(chan struct{}),
	}
}

// Start 启动连接循环（非阻塞；断线自动重连）。
func (es *EventStream) Start() {
	es.mu.Lock()
	if es.running {
		es.mu.Unlock()
		return
	}
	es.running = true
	es.mu.Unlock()
	go es.loop()
}

// Stop 停止连接循环并关闭连接。
func (es *EventStream) Stop() {
	es.mu.Lock()
	if !es.running {
		es.mu.Unlock()
		return
	}
	es.running = false
	conn := es.conn
	es.mu.Unlock()
	close(es.stopCh)
	if conn != nil {
		_ = conn.Close()
	}
}

// Connected 当前是否已连接。
func (es *EventStream) Connected() bool {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.connected
}

// Watch 订阅某会话的事件；返回取消函数（幂等）。
// 同一 convID 重复 Watch 会覆盖旧订阅。
func (es *EventStream) Watch(convID string, onEvent func(ev stream.Event)) func() {
	entry := &watchEntry{convID: convID, onEvent: onEvent}
	es.mu.Lock()
	es.watches[convID] = entry
	if es.acts[convID] == nil {
		es.acts[convID] = &convActivity{}
	}
	es.mu.Unlock()
	return func() {
		es.mu.Lock()
		if es.watches[convID] == entry {
			delete(es.watches, convID)
		}
		entry.closed = true
		es.mu.Unlock()
	}
}

// loop 连接主循环（指数退避重连）。
func (es *EventStream) loop() {
	defer close(es.loopDone)
	for {
		select {
		case <-es.stopCh:
			return
		default:
		}
		es.connectOnce()
		select {
		case <-es.stopCh:
			return
		case <-time.After(es.currentRetryDelay()):
		}
		es.bumpRetryDelay()
	}
}

func (es *EventStream) currentRetryDelay() time.Duration {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.retryDelay
}

func (es *EventStream) bumpRetryDelay() {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.retryDelay *= 2
	if es.retryDelay > 30*time.Second {
		es.retryDelay = 30 * time.Second
	}
}

// connectOnce 建立一次连接并阻塞到断开/停止。
func (es *EventStream) connectOnce() {
	cli, err := wsclient.Dial(es.url, es.connTimeout, nil)
	if err != nil {
		es.logf("事件流握手失败: %v", err)
		return
	}
	closed := make(chan struct{})
	cli.OnMessage = func(data []byte, isText bool) {
		if !isText {
			return
		}
		es.onMessage(data)
	}
	cli.OnClose = func() { close(closed) }

	es.mu.Lock()
	es.conn = cli
	es.connected = true
	es.retryDelay = time.Second
	es.mu.Unlock()
	es.logf("事件流已连接：%s", es.url)

	select {
	case <-closed:
		es.logf("事件流连接断开，将自动重连")
	case <-es.stopCh:
		_ = cli.Close()
	}

	es.mu.Lock()
	es.conn = nil
	es.connected = false
	es.mu.Unlock()
}

// onMessage 处理一条 WS 文本消息（JSON 对象或数组）。
func (es *EventStream) onMessage(raw []byte) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return
	}
	if trimmed[0] == '[' {
		var arr []json.RawMessage
		if json.Unmarshal(trimmed, &arr) != nil {
			return
		}
		for _, item := range arr {
			es.dispatchItem(item)
		}
		return
	}
	es.dispatchItem(trimmed)
}

// dispatchItem 解析单条事件：更新会话活动状态并按 convId 路由到 watch。
func (es *EventStream) dispatchItem(raw json.RawMessage) {
	defer func() {
		if p := recover(); p != nil {
			es.logf("事件分发 panic（已隔离）: %v", p)
		}
	}()
	var ev stream.Event
	if json.Unmarshal(raw, &ev) != nil {
		return
	}
	if ev.Type == "ping" {
		return
	}
	if ev.Type == "status" {
		// ★ 忙对齐：status 携带 runningConvs（权威运行集快照；连接建立与
		//   done/error 后推送）——据此校正各会话活动状态，兜底 done 事件丢失。
		es.applyStatus(raw)
		return
	}
	if ev.ConvID == "" {
		return
	}
	// ★ 忙对齐：content/tool_call 置「忙」、done 清除、snapshot 视为忙；
	//   error 不改变状态（忙错误的误报与终结性错误均不改变运行集事实）。
	switch ev.Type {
	case "content", "tool_call", "snapshot":
		es.setConvActive(ev.ConvID, true)
	case "done":
		es.setConvActive(ev.ConvID, false)
	}
	es.mu.Lock()
	w := es.watches[ev.ConvID]
	es.mu.Unlock()
	if w == nil || w.closed {
		return
	}
	w.onEvent(ev)
}

// setConvActive 更新被跟踪会话的活动状态（未跟踪的会话忽略）。
func (es *EventStream) setConvActive(convID string, active bool) {
	es.mu.Lock()
	if a := es.acts[convID]; a != nil {
		a.active = active
		a.lastAt = time.Now()
	}
	es.mu.Unlock()
}

// applyStatus 用 status 快照的 runningConvs 校正全部被跟踪会话的活动状态。
func (es *EventStream) applyStatus(raw json.RawMessage) {
	var st struct {
		RunningConvs []string `json:"runningConvs"`
	}
	if json.Unmarshal(raw, &st) != nil {
		return
	}
	running := make(map[string]bool, len(st.RunningConvs))
	for _, id := range st.RunningConvs {
		running[id] = true
	}
	now := time.Now()
	es.mu.Lock()
	for id, a := range es.acts {
		a.active = running[id]
		a.lastAt = now
	}
	es.mu.Unlock()
}

// ConvActive 会话当前是否处于「有任务运行」状态（忙对齐用）。
// 未被 Watch 跟踪的会话返回 false（按「不忙」处理）。
func (es *EventStream) ConvActive(convID string) bool {
	es.mu.Lock()
	defer es.mu.Unlock()
	if a := es.acts[convID]; a != nil {
		return a.active
	}
	return false
}

// wsURL 从 PairCode 地址推导事件流地址（http→ws）。
func wsURL(base string) string {
	base = strings.TrimSuffix(base, "/")
	switch {
	case strings.HasPrefix(base, "http://"):
		return "ws://" + strings.TrimPrefix(base, "http://") + "/ws"
	case strings.HasPrefix(base, "https://"):
		return "wss://" + strings.TrimPrefix(base, "https://") + "/ws"
	case strings.HasPrefix(base, "ws://"), strings.HasPrefix(base, "wss://"):
		return base + "/ws"
	}
	return "ws://" + base + "/ws"
}
