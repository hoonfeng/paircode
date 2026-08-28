// ═══════════════════════════════════════════════════════════════
// wsconn_test.go — WSConn 帧编码/解码回归测试
//
// 回归背景（2026-08-22）：writeFrame 的 127 分支（8 字节长度）原实现
// 只写「6 字节 0 + 低 16 位」，n>=65536 时长度被截断成低 16 位，例如
// n=195945(0x2FD69) 写成 0xFD69=64873 → 客户端按错误长度切帧 → 帧边界
// 错位 → "Invalid frame header" → WebSocket 断线重连循环。
// （2026-08-21 snapshot 断线补偿推送 >64KB 大 JSON 后暴露此 bug）
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// readWSFrameClient 客户端侧解析一帧（服务器→客户端：不掩码），返回 opcode + payload。
func readWSFrameClient(t *testing.T, r io.Reader) (byte, []byte) {
	t.Helper()
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		t.Fatalf("读帧头失败: %v", err)
	}
	op := hdr[0] & 0x0F
	length := int(hdr[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			t.Fatalf("读扩展长度失败: %v", err)
		}
		length = int(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			t.Fatalf("读 8 字节长度失败: %v", err)
		}
		length = int(binary.BigEndian.Uint64(ext[:]))
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		t.Fatalf("读 payload 失败 len=%d: %v", length, err)
	}
	return op, payload
}

// TestWriteTextFrameLargePayload 大帧长度编码回归：声明的长度必须等于实际 payload。
func TestWriteTextFrameLargePayload(t *testing.T) {
	sizes := []int{65535, 65536, 100000, 200000, 1 << 20}
	for _, size := range sizes {
		size := size
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			server, client := net.Pipe()
			defer client.Close()

			conn := &WSConn{netConn: server, bw: bufio.NewWriter(server)}
			payload := bytes.Repeat([]byte{0x41}, size) // 'A' x size
			go func() {
				time.Sleep(20 * time.Millisecond) // 等待客户端就绪
				if err := conn.WriteTextFrame(payload); err != nil {
					t.Errorf("WriteTextFrame err=%v", err)
				}
				server.Close()
			}()

			op, got := readWSFrameClient(t, client)
			if op != 0x1 {
				t.Fatalf("opcode=%#x 期望 0x1", op)
			}
			if len(got) != size {
				t.Fatalf("payload 长度=%d 期望 %d（127 分支长度截断回归）", len(got), size)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("payload 内容不一致")
			}
		})
	}
}

// TestWriteTextFrameSmallPayload 小帧（126/直接长度分支）回归。
func TestWriteTextFrameSmallPayload(t *testing.T) {
	for _, size := range []int{0, 1, 125, 126, 1000, 65534} {
		server, client := net.Pipe()
		conn := &WSConn{netConn: server, bw: bufio.NewWriter(server)}
		payload := bytes.Repeat([]byte{0x42}, size)
		go func() {
			time.Sleep(20 * time.Millisecond)
			_ = conn.WriteTextFrame(payload)
			server.Close()
		}()
		op, got := readWSFrameClient(t, client)
		if op != 0x1 {
			t.Fatalf("opcode=%#x 期望 0x1", op)
		}
		if len(got) != size {
			t.Fatalf("size=%d 解析 len=%d", size, len(got))
		}
		client.Close()
	}
}

// TestReadFramePingPong 读循环处理协议层 Ping（回复 Pong）/ Pong（忽略）。
func TestReadFramePingPong(t *testing.T) {
	server, client := net.Pipe()
	conn := &WSConn{netConn: server, br: bufio.NewReader(server), bw: bufio.NewWriter(server)}
	defer client.Close()

	go func() {
		// 客户端：发 Ping 帧（OP=0x9, masked 客户端帧），等待服务端回 Pong，再发文本帧
		// （net.Pipe 同步语义：必须先读 Pong，否则服务端回 Pong 时阻塞、客户端写 txt 也阻塞 → 死锁）
		frame := []byte{0x89, 0x80, 0x00, 0x00, 0x00, 0x00} // FIN+ping, mask=1, 4字节掩码
		client.Write(frame)
		// 读服务端回应的 Pong 帧（op=0xA，不掩码）
		var hdr [2]byte
		io.ReadFull(client, hdr[:])
		if hdr[0]&0x0F != 0xA {
			t.Errorf("期望 Pong 帧，收到 opcode=%#x", hdr[0]&0x0F)
		}
		txt := []byte{0x81, 0x82, 0x00, 0x00, 0x00, 0x00, 'h', 'i'}
		client.Write(txt)
		client.Close()
	}()

	// 服务端读：第一帧 Ping 内部自动回 Pong 并 continue；第二帧 Text 返回
	op, payload, err := conn.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame err=%v", err)
	}
	if op != 0x1 || string(payload) != "hi" {
		t.Fatalf("op=%#x payload=%q 期望 0x1 'hi'", op, payload)
	}
	client.Close()
}
