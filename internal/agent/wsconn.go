// ═══════════════════════════════════════════════════════════════
// wsconn.go — WebSocket 连接基础设施（RFC6455 最小实现）
//
// 从 cmd/companion/websocket_handler.go 抽离（2026-08-16）：main 不再
// 自实现 ws，改为复用本包；插件生态经 ext_ws.go（ctx.ws）注册双向通道。
//
// 提供：UpgradeWS 握手 + WSConn（帧读写/掩码解码/分片拼接/ping-pong/
// close/写锁并发安全）。纯标准库，跨平台。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// websocketGUID 是 WebSocket 握手协议规定的固定 GUID。
const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// ErrInvalidUpgrade 表示 WebSocket 升级握手失败（请求缺 Upgrade/Sec-WebSocket-Key 头）。
var ErrInvalidUpgrade = errors.New("websocket upgrade failed")

// WSConn 封装一个已升级的 WebSocket 连接，提供最小化的帧读写。
// 写操作并发安全（内部 writeMu）；读须单 goroutine（readFrame 循环）。
type WSConn struct {
	netConn net.Conn
	br      *bufio.Reader
	bw      *bufio.Writer

	writeMu sync.Mutex
	closed  bool

	// ★ 2026-08-22 读超时（>0 启用）：ReadFrame 每次循环（含 Ping/Pong 后
	//   continue）自动刷新读 deadline——否则 deadline 从连接建立起固定，
	//   客户端持续回 Pong 的连接仍会在固定时长后被误判死（实测复现：
	//   30s/60s Pong 均到达，70s 仍读超时断开）。由调用方经 SetReadTimeout 设置。
	readTO time.Duration
}

// UpgradeWS 执行 HTTP→WebSocket 升级握手，返回封装的 WSConn。
// 若请求不合法（缺少 Upgrade 头或 Sec-WebSocket-Key），返回 ErrInvalidUpgrade
// 并写入 HTTP 错误响应。
func UpgradeWS(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "Missing Upgrade: websocket", http.StatusBadRequest)
		return nil, ErrInvalidUpgrade
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, ErrInvalidUpgrade
	}

	h, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijack not supported", http.StatusInternalServerError)
		return nil, ErrInvalidUpgrade
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

	// br 使用 http.Hijack 返回的 brw.Reader（内含握手残留缓冲，标准做法）
	return &WSConn{
		netConn: conn,
		br:      brw.Reader,
		bw:      brw.Writer,
	}, nil
}

// WriteTextFrame 发送一个文本帧（opcode=0x1）。
func (c *WSConn) WriteTextFrame(data []byte) error {
	return c.writeFrame(0x81, data)
}

// WriteBinaryFrame 发送一个二进制帧（opcode=0x2）。
func (c *WSConn) WriteBinaryFrame(data []byte) error {
	return c.writeFrame(0x82, data)
}

// writeFrame 发送一帧（FIN=1 + opcode + 长度编码 + payload），带写锁。
func (c *WSConn) writeFrame(headByte byte, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	c.netConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer c.netConn.SetWriteDeadline(time.Time{})

	if err := c.bw.WriteByte(headByte); err != nil {
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
		// ★ 2026-08-22 修复：8 字节长度必须完整大端写出（b7..b0）。
		//   原实现只写 6 字节 0 + 低 16 位 → n>=65536 时长度被截断成低 16 位，
		//   客户端按错误长度切帧 → 帧边界错位 → "Invalid frame header" → 断线循环。
		//   （2026-08-21 snapshot 断线补偿推送 >64KB 大 JSON 后暴露）
		for i := 7; i >= 0; i-- {
			if err := c.bw.WriteByte(byte(n >> (8 * i))); err != nil {
				return err
			}
		}
	}
	if _, err := c.bw.Write(data); err != nil {
		return err
	}
	return c.bw.Flush()
}

// WritePingFrame 写入一个 Ping 帧（用于心跳保活）。
func (c *WSConn) WritePingFrame() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
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

// WriteCloseFrame 写入一个 Close 帧。
func (c *WSConn) WriteCloseFrame() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	if err := c.bw.WriteByte(0x88); err != nil { // FIN=1, opcode=0x8 (close)
		return err
	}
	if err := c.bw.WriteByte(0x00); err != nil {
		return err
	}
	return c.bw.Flush()
}

// WritePongFrame 回复 Pong（用 Ping 的 payload）。
func (c *WSConn) WritePongFrame(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
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

// ReadFrame 读取一个 WebSocket 帧，返回 opcode 和 payload。
// 处理掩码解码、分片拼接（continuation frame）。
// 收到 ping 自动回复 pong，收到 close 返回 io.EOF。
// ★ 2026-08-22 readTO>0 时每次循环（含 Ping/Pong 后 continue）自动刷新
//
//	读 deadline——否则 deadline 从连接建立起固定，客户端持续回 Pong 的
//	连接仍会在固定时长后被误判死（实测：30s/60s Pong 均到达、70s 仍超时断开）。
//
// 注意：必须单 goroutine 调用。
func (c *WSConn) ReadFrame() (opcode byte, payload []byte, err error) {
	if c == nil {
		return 0, nil, io.ErrUnexpectedEOF
	}
	for {
		if c.readTO > 0 {
			c.SetReadDeadline(time.Now().Add(c.readTO))
		}
		op, data, e := c.readSingleFrame()
		if e != nil {
			return 0, nil, e
		}
		switch op {
		case 0x9: // Ping → 回复 Pong
			c.WritePongFrame(data)
			continue
		case 0xA: // Pong → 忽略
			continue
		case 0x8: // Close
			c.WriteCloseFrame()
			return 0x8, data, io.EOF
		default: // 0x1 (text), 0x2 (binary), 0x0 (continuation)
			return op, data, nil
		}
	}
}

// readSingleFrame 读取单个帧（不含 continuation 处理）。
func (c *WSConn) readSingleFrame() (opcode byte, payload []byte, err error) {
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

// Close 关闭底层连接（标记 closed，后续写操作返回 io.ErrClosedPipe）。
func (c *WSConn) Close() error {
	c.writeMu.Lock()
	c.closed = true
	c.writeMu.Unlock()
	return c.netConn.Close()
}

// SetReadTimeout 启用 ReadFrame 内部读超时管理（每帧刷新 deadline）。
// 传 0 恢复默认（由调用方自行管理 SetReadDeadline）。
func (c *WSConn) SetReadTimeout(d time.Duration) {
	c.readTO = d
}

// SetDeadline 设置读写截止时间。
func (c *WSConn) SetDeadline(t time.Time) error {
	return c.netConn.SetDeadline(t)
}

// SetReadDeadline 设置读截止时间。
func (c *WSConn) SetReadDeadline(t time.Time) error {
	return c.netConn.SetReadDeadline(t)
}

// NetConn 返回底层 net.Conn（心跳/超时等高级场景）。
func (c *WSConn) NetConn() net.Conn { return c.netConn }
