// WebSocket 端点：单一全局连接推送所有 agent 会话事件。
// 替代原 SSE（handleChatEvents）方案，支持跨工作区并行对话。
//
//go:build windows

package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)


// websocketGUID 是 WebSocket 握手协议规定的固定 GUID。
const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// wsConn 封装一个已升级的 WebSocket 连接，提供最小化的帧读写。
type wsConn struct {
	netConn net.Conn
	br      *bufio.Reader
	bw      *bufio.Writer
}

// upgradeWebSocket 执行 HTTP→WebSocket 升级握手，返回封装的 wsConn。
// 若请求不合法（缺少 Upgrade 头或 Sec-WebSocket-Key），返回错误。
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "Missing Upgrade: websocket", http.StatusBadRequest)
		return nil, errInvalidUpgrade
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, errInvalidUpgrade
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		return nil, errInvalidUpgrade
	}

	// 计算 Sec-WebSocket-Accept
	hh := sha1.New()
	hh.Write([]byte(key + websocketGUID))
	accept := base64.StdEncoding.EncodeToString(hh.Sum(nil))

	conn, brw, err := h.Hijack()
	if err != nil {
		return nil, err
	}

	// 写 101 Switching Protocols 响应
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := brw.WriteString(resp); err != nil {
		conn.Close()
		return nil, err
	}
	if err := brw.Flush(); err != nil {
		conn.Close()
		return nil, err
	}

	return &wsConn{
		netConn: conn,
		br:      brw.Reader,
		bw:      brw.Writer,
	}, nil
}

// writeTextFrame 写入一个 WebSocket 文本帧（服务器→客户端，不掩码）。
func (c *wsConn) writeTextFrame(data []byte) error {
	c.netConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer c.netConn.SetWriteDeadline(time.Time{})
	// 帧头：FIN=1, opcode=0x1 (text)
	if err := c.bw.WriteByte(0x81); err != nil {
		return err
	}
	n := len(data)
	switch {
	case n < 126:
		if err := c.bw.WriteByte(byte(n)); err != nil {
			return err
		}
	case n < 65536:
		if err := c.bw.WriteByte(126); err != nil {
			return err
		}
		if err := c.bw.WriteByte(byte(n >> 8)); err != nil {
			return err
		}
		if err := c.bw.WriteByte(byte(n)); err != nil {
			return err
		}
	default:
		if err := c.bw.WriteByte(127); err != nil {
			return err
		}
		// 8 字节长度（高 4 字节为 0，因为我们不会超过 4GB）
		for i := 5; i >= 0; i-- {
			if err := c.bw.WriteByte(0); err != nil {
				return err
			}
		}
		if err := c.bw.WriteByte(byte(n >> 8)); err != nil {
			return err
		}
		if err := c.bw.WriteByte(byte(n)); err != nil {
			return err
		}
	}
	if _, err := c.bw.Write(data); err != nil {
		return err
	}
	return c.bw.Flush()
}

// writePingFrame 写入一个 Ping 帧（用于心跳保活）。
func (c *wsConn) writePingFrame() error {
	c.netConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer c.netConn.SetWriteDeadline(time.Time{})
	if err := c.bw.WriteByte(0x89); err != nil { // FIN=1, opcode=0x9 (ping)
		return err
	}
	if err := c.bw.WriteByte(0x00); err != nil { // 长度=0, 无掩码
		return err
	}
	return c.bw.Flush()
}

// writeCloseFrame 写入一个 Close 帧。
func (c *wsConn) writeCloseFrame() error {
	if err := c.bw.WriteByte(0x88); err != nil { // FIN=1, opcode=0x8 (close)
		return err
	}
	if err := c.bw.WriteByte(0x00); err != nil {
		return err
	}
	return c.bw.Flush()
}

// readFrame 读取一个 WebSocket 帧，返回 opcode 和 payload。
// 处理掩码解码、分片拼接（continuation frame）。
// 收到 ping 自动回复 pong，收到 close 返回 io.EOF。
func (c *wsConn) readFrame() (opcode byte, payload []byte, err error) {
	if c == nil {
		return 0, nil, io.ErrUnexpectedEOF
	}
	for {
		op, data, e := c.readSingleFrame()
		if e != nil {
			return 0, nil, e
		}
		switch op {
		case 0x9: // Ping → 回复 Pong
			c.writePongFrame(data)
			continue
		case 0xA: // Pong → 忽略
			continue
		case 0x8: // Close
			c.writeCloseFrame()
			return 0x8, data, io.EOF
		default: // 0x1 (text), 0x2 (binary), 0x0 (continuation)
			return op, data, nil
		}
	}
}

// readSingleFrame 读取单个帧（不含 continuation 处理）。
func (c *wsConn) readSingleFrame() (opcode byte, payload []byte, err error) {
	if c == nil || c.br == nil {
		return 0, nil, io.ErrUnexpectedEOF
	}
	// 字节 0: FIN(1) + RSV(3) + Opcode(4)
	b0, err := c.br.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	opcode = b0 & 0x0F

	// 字节 1: Mask(1) + PayloadLen(7)
	b1, err := c.br.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	masked := b1&0x80 != 0
	length := int(b1 & 0x7F)

	// 扩展长度
	if length == 126 {
		hi, err := c.br.ReadByte()
		if err != nil {
			return 0, nil, err
		}
		lo, err := c.br.ReadByte()
		if err != nil {
			return 0, nil, err
		}
		length = int(hi)<<8 | int(lo)
	} else if length == 127 {
		// 8 字节长度（只取低 4 字节，足够用）
		var buf [8]byte
		if _, err := io.ReadFull(c.br, buf[:]); err != nil {
			return 0, nil, err
		}
		length = int(buf[6])<<8 | int(buf[7])
	}

	// 掩码密钥
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.br, mask[:]); err != nil {
			return 0, nil, err
		}
	}

	// payload
	payload = make([]byte, length)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return 0, nil, err
	}

	// 解掩码
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, payload, nil
}

// writePongFrame 回复 Pong（用 Ping 的 payload）。
func (c *wsConn) writePongFrame(data []byte) error {
	if err := c.bw.WriteByte(0x8A); err != nil { // FIN=1, opcode=0xA (pong)
		return err
	}
	n := len(data)
	if n < 126 {
		c.bw.WriteByte(byte(n))
	} else {
		c.bw.WriteByte(126)
		c.bw.WriteByte(byte(n >> 8))
		c.bw.WriteByte(byte(n))
	}
	c.bw.Write(data)
	return c.bw.Flush()
}

// Close 关闭底层连接。
func (c *wsConn) Close() error {
	return c.netConn.Close()
}

// SetDeadline 设置读写截止时间。
func (c *wsConn) SetDeadline(t time.Time) error {
	return c.netConn.SetDeadline(t)
}

// errInvalidUpgrade 表示 WebSocket 升级握手失败。
var errInvalidUpgrade = &wsUpgradeError{}

type wsUpgradeError struct{}

func (e *wsUpgradeError) Error() string { return "websocket upgrade failed" }

// handleWebSocket 是 WebSocket 端点处理器。
// 升级连接后，订阅 agentMgr 全局事件流，将所有会话事件以 JSON 推送到 WebSocket。
// 每条消息格式：{convId, type, content, tool, args, callId, usage, doneReason}
// 连接建立时先发送 {type:"status", runningConvs:[...]} 同步初始状态。
func (s *webServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	wsc, err := upgradeWebSocket(w, r)
	if err != nil {
		return // upgradeWebSocket 已写过 HTTP 错误响应
	}
	defer wsc.Close()

	// 订阅全局事件流
	ch := agentMgr.SubscribeAll()
	defer agentMgr.UnsubscribeAll(ch)

	// 发送初始状态：当前运行中的所有会话 + 按工作区分组的运行计数
	wsc.writeTextFrame(buildStatusPayload())

	// 心跳定时器（每 30 秒发一次 ping）
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// 后台读取客户端消息（处理 close 帧、忽略其他）
	// 设置 60s 读超时，配合 30s 心跳：2 次心跳未回复即为断开
	clientClosed := make(chan struct{}, 1)
	go func() {
		defer func() {
			select {
			case clientClosed <- struct{}{}:
			default:
			}
		}()
		for {
			wsc.netConn.SetReadDeadline(time.Now().Add(70 * time.Second))
			_, _, e := wsc.readFrame()
			if e != nil {
				return
			}
			// 忽略客户端发来的文本消息（当前不支持命令通道）
		}
	}()

	// 主循环：从全局 channel 读取事件，推送到 WebSocket
	// 当事件类型为 done（会话结束）时，额外推送一次 status 消息，
	// 让前端的"工作区/对话列表"运行状态计数保持同步。
	for {
		select {
		case <-clientClosed:
			return
		case <-heartbeat.C:
			if err := wsc.writePingFrame(); err != nil {
				return
			}
		case ge, ok := <-ch:
			if !ok {
				// channel 被关闭（不应发生，但兜底）
				return
			}
			payload := buildWSPayload(ge)
			if err := wsc.writeTextFrame(payload); err != nil {
				return
			}
			// done/error 事件表示会话运行集将发生变化，追加 status 更新。
			// 注意：EventDone 发射时 session_manager 的 Running 标志可能尚未置 false
			// （loop.Run 返回 → defer 设置 Running=false → close(Events)），
			// 因此短暂等待 50ms 让 Running 状态落地后再查询，避免刚结束的会话仍出现在 running 列表。
			if ge.Event.Type == agent.EventDone || ge.Event.Type == agent.EventError {
				time.Sleep(50 * time.Millisecond)
				if err := wsc.writeTextFrame(buildStatusPayload()); err != nil {
					return
				}
			}
		}
	}
}


// buildStatusPayload 构造一条 status 消息，包含：
//   - runningConvs: 当前所有运行中的 convID 列表
//   - runningByWorkspace: 按工作区根路径分组的运行计数 {wsRoot: count}
//
// 前端据此更新工作区列表的脉冲点+计数，以及对话列表的"运行中"标签。
// 在 WebSocket 连接建立时和每次 done 事件后推送。
func buildStatusPayload() []byte {
	running := agentMgr.ListRunning()
	counts := make(map[string]int, 8)
	for _, id := range running {
		ws := agentMgr.GetWorkspaceRoot(id)
		if ws == "" {
			continue
		}
		counts[ws]++
	}
	msg := map[string]any{
		"type":               "status",
		"runningConvs":       running,
		"runningByWorkspace": counts,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] status JSON encode error: %v", err)
		return []byte(`{"type":"status","runningConvs":[],"runningByWorkspace":{}}`)
	}
	return data
}

// buildWSPayload 将 GlobalEvent 编码为 WebSocket JSON 消息。
func buildWSPayload(ge agent.GlobalEvent) []byte {
	e := ge.Event
	msg := map[string]any{
		"convId":     ge.ConvID,
		"type":       string(e.Type),
		"content":    e.Content,
		"tool":       e.Tool,
		"args":       e.Args,
		"callId":     e.CallID,
		"doneReason": e.DoneReason,
	}
	if e.Usage != nil {
		msg["usage"] = e.Usage
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] JSON encode error: %v", err)
		return []byte(`{"type":"error","content":"JSON encode failed"}`)
	}
	return data
}
