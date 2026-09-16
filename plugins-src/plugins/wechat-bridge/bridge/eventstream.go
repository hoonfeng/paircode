// eventstream.go — 常驻订阅 PairCode /ws 全局事件流。
//
//   - 断线自动重连（指数退避 1s→30s）
//   - Watch(convID, onEvent)：按会话订阅（每账号一个 watch，事件按 convId 路由）
//   - Connected()：连接状态维护，供桥判断「流式通道是否可用」
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
	retryDelay time.Duration

	stopCh   chan struct{}
	loopDone chan struct{}
}

type watchEntry struct {
	convID  string
	onEvent func(ev stream.Event)
	closed  bool
}

// NewEventStream 创建事件流（url 形如 ws://127.0.0.1:9090/ws）。
func NewEventStream(url string, connTimeout time.Duration, log logf) *EventStream {
	return &EventStream{
		url:         url,
		connTimeout: connTimeout,
		logf:        log,
		watches:     map[string]*watchEntry{},
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

// dispatchItem 解析单条事件并按 convId 路由到 watch。
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
	if ev.Type == "ping" || ev.Type == "status" || ev.ConvID == "" {
		return
	}
	es.mu.Lock()
	w := es.watches[ev.ConvID]
	es.mu.Unlock()
	if w == nil || w.closed {
		return
	}
	w.onEvent(ev)
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
