package debugger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

// dapConn 管理与 dlv dap 服务器的 TCP 连接和 JSON-RPC 通信。
type dapConn struct {
	mu       sync.Mutex
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
	pending  map[int64]chan<- Message // seq → response channel
	events   chan Message             // 事件通道（外部消费）
	nextSeq  int64
	closed   bool
	onEvent  func(Message) // 可选事件回调
}

// newDAPConn 创建到 host:port 的 DAP 连接。
func newDAPConn(host string, port int) (*dapConn, error) {
	addr := fmt.Sprintf("%s:%d", host, port)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("连接 dlv dap 失败 (%s): %w", addr, err)
	}
	dc := &dapConn{
		conn:    c,
		reader:  bufio.NewReader(c),
		writer:  bufio.NewWriter(c),
		pending: make(map[int64]chan<- Message),
		events:  make(chan Message, 64),
		nextSeq: 1,
	}
	go dc.readLoop()
	return dc, nil
}

// send 发送请求并等待响应。
func (dc *dapConn) send(command string, args any) (Message, error) {
	dc.mu.Lock()
	if dc.closed {
		dc.mu.Unlock()
		return Message{}, fmt.Errorf("DAP 连接已关闭")
	}
	seq := dc.nextSeq
	dc.nextSeq++
	respCh := make(chan Message, 1)
	dc.pending[seq] = respCh
	dc.mu.Unlock()

	// 构造请求
	req := map[string]any{
		"seq":       seq,
		"type":      "request",
		"command":   command,
		"arguments": args,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("DAP 请求序列化失败: %w", err)
	}
	// 发送：Content-Length 头 + JSON 体（dlv dap 使用标准 DAP 帧）
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	dc.mu.Lock()
	if _, err := dc.writer.WriteString(header); err != nil {
		dc.mu.Unlock()
		return Message{}, fmt.Errorf("DAP 写入头失败: %w", err)
	}
	if _, err := dc.writer.Write(data); err != nil {
		dc.mu.Unlock()
		return Message{}, fmt.Errorf("DAP 写入数据失败: %w", err)
	}
	if err := dc.writer.Flush(); err != nil {
		dc.mu.Unlock()
		return Message{}, fmt.Errorf("DAP flush 失败: %w", err)
	}
	dc.mu.Unlock()

	// 等待响应
	resp := <-respCh
	return resp, nil
}

// readLoop 持续读取 DAP 消息并分发。
func (dc *dapConn) readLoop() {
	defer close(dc.events)
	for {
		msg, err := dc.readMessage()
		if err != nil {
			// 连接关闭或读错误：通知所有 pending
			dc.mu.Lock()
			for seq, ch := range dc.pending {
				close(ch)
				delete(dc.pending, seq)
			}
			dc.closed = true
			dc.mu.Unlock()
			return
		}
		dc.dispatch(msg)
	}
}

// readMessage 读取一条 DAP 消息。
func (dc *dapConn) readMessage() (Message, error) {
	// 读取 Content-Length 头
	var contentLen int
	for {
		line, err := dc.reader.ReadString('\n')
		if err != nil {
			return Message{}, fmt.Errorf("读取 DAP 头失败: %w", err)
		}
		line = line[:len(line)-1] // 去掉 \n
		if line == "" || line == "\r" {
			break // 空行 = 头结束
		}
		if _, err := fmt.Sscanf(line, "Content-Length: %d", &contentLen); err == nil {
			// 找到了 Content-Length
		}
	}
	if contentLen <= 0 {
		return Message{}, fmt.Errorf("DAP 消息 Content-Length 无效: %d", contentLen)
	}

	// 读取 JSON 体
	buf := make([]byte, contentLen)
	if _, err := dc.reader.Read(buf); err != nil {
		return Message{}, fmt.Errorf("读取 DAP 体失败: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(buf, &msg); err != nil {
		return Message{}, fmt.Errorf("DAP JSON 解析失败: %w", err)
	}
	return msg, nil
}

// dispatch 将消息路由到 pending 响应通道或事件通道。
func (dc *dapConn) dispatch(msg Message) {
	if msg.Type == MsgResponse {
		dc.mu.Lock()
		ch, ok := dc.pending[msg.RequestSeq]
		delete(dc.pending, msg.RequestSeq)
		dc.mu.Unlock()
		if ok {
			ch <- msg
			return
		}
	}
	if msg.Type == MsgEvent {
		dc.events <- msg
		if dc.onEvent != nil {
			dc.onEvent(msg)
		}
	}
}

// close 关闭连接。
func (dc *dapConn) close() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if dc.closed {
		return
	}
	dc.closed = true
	dc.conn.Close()
	for seq, ch := range dc.pending {
		close(ch)
		delete(dc.pending, seq)
	}
}
