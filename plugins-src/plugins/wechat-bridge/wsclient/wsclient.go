// Package wsclient 最小 WebSocket 客户端（RFC6455 子集，纯标准库）。
//
// 用途：桥常驻订阅本机 PairCode 的 /ws 事件流（流式回复主路）。
// 客户端侧规则（与服务端相反）：
//   - 发送帧一律掩码（mask bit = 1）；
//   - 接收兼容掩码/非掩码（本机服务端框架不发掩码）；
//   - 收到 Ping 自动回 Pong；收到 Close 回 Close 并退出读循环。
//
// 蓝图：M1 Node 原型 temp/wx-bridge/src/ws.mjs（线上实测版本）。
package wsclient

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// wsGUID 是 WebSocket 握手协议规定的固定 GUID（RFC6455 §1.3）。
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// 帧操作码。
const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// Client 一个已握手完成的 WebSocket 客户端连接。
//
// 回调（OnMessage/OnClose）在读循环 goroutine 中调用，实现方需自行保证
// 并发安全；读循环在连接关闭或出错时退出并触发一次 OnClose。
type Client struct {
	conn net.Conn
	br   *bufio.Reader

	writeMu sync.Mutex
	closed  bool

	// OnMessage 收到完整消息（分片已拼接）。isText=false 表示二进制帧。
	OnMessage func(data []byte, isText bool)
	// OnClose 连接关闭（读循环退出时恰好调用一次）。
	OnClose func()
}

// Dial 建立 WebSocket 连接（含 HTTP 升级握手），成功后自动启动读循环。
// extraHeaders 可为 nil；握手期间使用 timeout 作为读写截止。
func Dial(rawURL string, timeout time.Duration, extraHeaders http.Header) (*Client, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("ws 地址解析失败: %w", err)
	}
	if u.Scheme != "ws" {
		return nil, fmt.Errorf("仅支持 ws:// 地址，收到: %s", rawURL)
	}
	host := u.Host
	if _, _, splitErr := net.SplitHostPort(host); splitErr != nil {
		host += ":80"
	}

	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}

	// 生成 Sec-WebSocket-Key（16 字节随机 base64）
	keyRaw := make([]byte, 16)
	if _, err := rand.Read(keyRaw); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyRaw)

	// 发送 HTTP 升级请求
	var sb strings.Builder
	sb.WriteString("GET " + u.RequestURI() + " HTTP/1.1\r\n")
	sb.WriteString("Host: " + u.Host + "\r\n")
	sb.WriteString("Upgrade: websocket\r\n")
	sb.WriteString("Connection: Upgrade\r\n")
	sb.WriteString("Sec-WebSocket-Version: 13\r\n")
	sb.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	for k, vals := range extraHeaders {
		for _, v := range vals {
			sb.WriteString(k + ": " + v + "\r\n")
		}
	}
	sb.WriteString("\r\n")

	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := io.WriteString(conn, sb.String()); err != nil {
		conn.Close()
		return nil, err
	}

	// 读取升级响应（101 无 body，剩余字节即首批 WS 帧）
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取升级响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, fmt.Errorf("WS 升级失败：HTTP %d", resp.StatusCode)
	}
	expect := acceptKey(key)
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != expect {
		conn.Close()
		return nil, errors.New("Sec-WebSocket-Accept 校验失败")
	}
	_ = conn.SetDeadline(time.Time{})

	c := &Client{conn: conn, br: br}
	go c.readLoop()
	return c, nil
}

// acceptKey 计算握手应答值：base64(sha1(key + GUID))。
func acceptKey(key string) string {
	h := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h[:])
}

// SendText 发送一个文本帧（已掩码）。
func (c *Client) SendText(text string) error {
	return c.sendFrame(opText, []byte(text))
}

// Connected 连接是否仍可用（未被本地关闭）。
func (c *Client) Connected() bool {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return !c.closed
}

// Close 发送 Close 帧（best-effort）并关闭底层连接。
// 读循环将随之退出并触发 OnClose。
func (c *Client) Close() error {
	_ = c.sendFrame(opClose, nil)
	c.writeMu.Lock()
	c.closed = true
	c.writeMu.Unlock()
	return c.conn.Close()
}

// ── 读侧 ─────────────────────────────────────────────────────────────

// readLoop 读循环：解析帧、分发消息、维护 ping/pong 与关闭。
func (c *Client) readLoop() {
	defer func() {
		if c.OnClose != nil {
			c.OnClose()
		}
	}()
	for {
		op, payload, err := c.readMessage()
		if err != nil {
			return
		}
		switch op {
		case opText:
			if c.OnMessage != nil {
				c.OnMessage(payload, true)
			}
		case opBinary:
			if c.OnMessage != nil {
				c.OnMessage(payload, false)
			}
		case opPing:
			_ = c.sendFrame(opPong, payload)
		case opPong:
			// 心跳应答：无需处理
		case opClose:
			_ = c.sendFrame(opClose, nil)
			return
		}
	}
}

// readMessage 读取一条完整消息（控制帧直接返回；数据帧做分片拼接）。
func (c *Client) readMessage() (byte, []byte, error) {
	op, fin, payload, err := c.readSingleFrame()
	if err != nil {
		return 0, nil, err
	}
	if op >= 0x8 || fin {
		return op, payload, nil
	}
	// 分片拼接：续帧直至 FIN；分片间允许插入控制帧
	for {
		op2, fin2, more, err := c.readSingleFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op2 {
		case opContinuation:
			payload = append(payload, more...)
			if fin2 {
				return op, payload, nil
			}
		case opPing:
			_ = c.sendFrame(opPong, more)
		case opPong:
			// 忽略
		case opClose:
			_ = c.sendFrame(opClose, nil)
			return 0, nil, io.EOF
		default:
			return 0, nil, fmt.Errorf("分片序列中出现非法帧 opcode=0x%x", op2)
		}
	}
}

// readSingleFrame 读取单个帧头 + 载荷（掩码解码）。读到 EOF 返回错误。
func (c *Client) readSingleFrame() (op byte, fin bool, payload []byte, err error) {
	var b0 [1]byte
	if _, err = io.ReadFull(c.br, b0[:]); err != nil {
		return 0, false, nil, err
	}
	fin = b0[0]&0x80 != 0
	op = b0[0] & 0x0F

	var b1 [1]byte
	if _, err = io.ReadFull(c.br, b1[:]); err != nil {
		return 0, false, nil, err
	}
	masked := b1[0]&0x80 != 0
	length := uint64(b1[0] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.br, ext[:]); err != nil {
			return 0, false, nil, err
		}
		length = uint64(ext[0])<<8 | uint64(ext[1])
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.br, ext[:]); err != nil {
			return 0, false, nil, err
		}
		length = 0
		for i := 0; i < 8; i++ {
			length = length<<8 | uint64(ext[i])
		}
	}
	if length > 64<<20 {
		return 0, false, nil, fmt.Errorf("帧过大（%d 字节）", length)
	}
	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(c.br, mask[:]); err != nil {
			return 0, false, nil, err
		}
	}
	payload = make([]byte, int(length))
	if _, err = io.ReadFull(c.br, payload); err != nil {
		return 0, false, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return op, fin, payload, nil
}

// ── 写侧 ─────────────────────────────────────────────────────────────

// sendFrame 发送单帧（FIN=1，客户端掩码）。并发安全。
func (c *Client) sendFrame(op byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}

	frame := make([]byte, 0, len(payload)+14)
	frame = append(frame, 0x80|op)
	n := len(payload)
	switch {
	case n < 126:
		frame = append(frame, byte(n)|0x80)
	case n < 65536:
		frame = append(frame, 126|0x80, byte(n>>8), byte(n))
	default:
		frame = append(frame, 127|0x80, 0, 0, 0, 0,
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return err
	}
	frame = append(frame, mask[:]...)
	for i := 0; i < n; i++ {
		frame = append(frame, payload[i]^mask[i%4])
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer c.conn.SetWriteDeadline(time.Time{})
	_, err := c.conn.Write(frame)
	return err
}
