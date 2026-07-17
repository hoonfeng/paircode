// PTY 终端 WebSocket 端点 —— 真正的终端模拟器（浏览器端 xterm.js + 服务端 ConPTY）。
//
// 协议：
//   客户端发文本帧（JSON 控制消息）：{type:"init", shell:"cmd", cwd:"..."} / {type:"resize", cols:N, rows:N}
//   服务端发文本帧：{type:"ready"} / {type:"error", msg:"..."} / {type:"closed"}
//   双向二进制帧 = 原始 PTY I/O 字节流（VT 转义序列，xterm.js 渲染）
//
// 安全措施：
//   - shell 白名单（仅 cmd/powershell/gitbash）
//   - cwd 路径校验（禁止穿越出工作区）
//   - PTY 关闭时强制终止子进程
//   - 并发 PTY 会话数限制（最多 16）
//
//go:build windows

package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/pty"
)

// ─── 并发限制 ──────────────────────────────────────────────

var (
	activePTYSessions int32 // 原子计数
	maxPTYSessions    int32 = 16
)

// ─── Shell 白名单 ──────────────────────────────────────────

var allowedShells = map[string]bool{
	"cmd":        true,
	"powershell": true,
	"gitbash":    true,
}

// ─── ptySession ────────────────────────────────────────────

type ptySession struct {
	mu     sync.Mutex
	sess   pty.PTY
	conn   *wsConn
	closed bool
	shell  string
	cols   int
	rows   int
	cwd    string
}

func (ps *ptySession) writeBinary(data []byte) error {
	ps.mu.Lock()
	conn := ps.conn
	ps.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.writeBinaryFrame(data)
}

func (ps *ptySession) writeText(data []byte) error {
	ps.mu.Lock()
	conn := ps.conn
	ps.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.writeTextFrame(data)
}

// close 关闭 PTY 和 WebSocket，原子计数减一。
func (ps *ptySession) close() {
	ps.mu.Lock()
	if ps.closed {
		ps.mu.Unlock()
		return
	}
	ps.closed = true
	sess := ps.sess
	conn := ps.conn
	ps.sess = nil
	ps.conn = nil
	ps.mu.Unlock()

	if sess != nil {
		sess.Close()
	}
	if conn != nil {
		conn.writeTextFrame([]byte(`{"type":"closed"}`))
		conn.Close()
	}
	atomic.AddInt32(&activePTYSessions, -1)
}

// ─── 处理器 ────────────────────────────────────────────────

func (s *webServer) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	// 并发限制
	if atomic.LoadInt32(&activePTYSessions) >= maxPTYSessions {
		http.Error(w, "Too many active terminal sessions", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgradeWebSocket(w, r)
	if err != nil {
		return
	}

	atomic.AddInt32(&activePTYSessions, 1)

	ps := &ptySession{
		conn:  conn,
		cols:  80,
		rows:  24,
		shell: "cmd",
		cwd:   safeCWD(""),
	}

	// 等待客户端发送 init 消息（文本帧 JSON）
	opcode, payload, err := conn.readFrame()
	if err != nil {
		atomic.AddInt32(&activePTYSessions, -1)
		conn.Close()
		return
	}

	if opcode == 0x1 {
		ps.handleInitMessage(payload)
	}

	// 启动 PTY
	if err := ps.startPTY(); err != nil {
		ps.writeText([]byte(`{"type":"error","msg":"` + jsonEscape(err.Error()) + `"}`))
		ps.close()
		return
	}

	ps.writeText([]byte(`{"type":"ready"}`))

	// 后台读 PTY 输出 → WebSocket 二进制帧
	go ps.ptyReader()

	// 主循环：读 WebSocket → 写 PTY
	ps.wsReader()
}

// handleInitMessage 解析并校验 init 消息。
func (ps *ptySession) handleInitMessage(payload []byte) {
	var msg struct {
		Type  string `json:"type"`
		Shell string `json:"shell"`
		Cwd   string `json:"cwd"`
		Cols  int    `json:"cols"`
		Rows  int    `json:"rows"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Type != "init" {
		return
	}

	// shell 白名单校验
	if msg.Shell != "" && allowedShells[msg.Shell] {
		ps.shell = msg.Shell
	}

	// cwd 安全校验
	if msg.Cwd != "" {
		ps.cwd = safeCWD(msg.Cwd)
	}

	// 终端尺寸
	if msg.Cols > 0 && msg.Cols <= 500 {
		ps.cols = msg.Cols
	}
	if msg.Rows > 0 && msg.Rows <= 200 {
		ps.rows = msg.Rows
	}
}

// safeCWD 校验并返回安全工作目录。
// 优先用工作区根路径；传入路径必须在工作区范围内（禁止穿越）。
func safeCWD(requested string) string {
	root := core.Root()

	if requested == "" {
		if root != "" {
			return root
		}
		cwd, _ := os.Getwd()
		return cwd
	}

	// 如果设了工作区根，校验路径是否在工作区内
	if root != "" {
		absReq, err := filepath.Abs(requested)
		if err != nil {
			return root
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return root
		}
		// 检查是否在工作区根目录下（或等于工作区根）
		rel, err := filepath.Rel(absRoot, absReq)
		if err != nil || strings.HasPrefix(rel, "..") {
			return root
		}
		return absReq
	}

	return requested
}

// startPTY 创建 ConPTY 会话。
func (ps *ptySession) startPTY() error {
	sh := ptyShellFor(ps.shell)
	sess, err := pty.Start(sh, ps.cwd, ps.cols, ps.rows)
	if err != nil {
		return err
	}

	ps.mu.Lock()
	ps.sess = sess
	ps.mu.Unlock()

	return nil
}

// ptyReader 从 PTY 读取输出并发送到 WebSocket（二进制帧）。
func (ps *ptySession) ptyReader() {
	buf := make([]byte, 65536)
	for {
		ps.mu.Lock()
		sess := ps.sess
		closed := ps.closed
		ps.mu.Unlock()

		if closed || sess == nil {
			return
		}

		n, err := sess.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			if wErr := ps.writeBinary(data); wErr != nil {
				log.Printf("[terminal-ws] 写 WebSocket 失败: %v", wErr)
				ps.close()
				return
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[terminal-ws] PTY 读错误: %v", err)
			}
			ps.close()
			return
		}
	}
}

// wsReader 从 WebSocket 读取数据并写入 PTY。
func (ps *ptySession) wsReader() {
	defer ps.close()

	for {
		// 通过锁获取 conn，防止 ptyReader 协程并发 close() 置 nil 导致 panic
		ps.mu.Lock()
		conn := ps.conn
		ps.mu.Unlock()
		if conn == nil {
			return
		}

		opcode, payload, err := conn.readFrame()
		if err != nil {
			return
		}

		switch opcode {
		case 0x1: // 文本帧 = JSON 控制消息
			ps.handleTextMessage(payload)
		case 0x2: // 二进制帧 = 键盘输入
			ps.mu.Lock()
			sess := ps.sess
			ps.mu.Unlock()
			if sess != nil && len(payload) > 0 {
				// 键盘输入直接透传到 PTY，xterm.js 已处理好所有转义序列
				if _, wErr := sess.Write(payload); wErr != nil {
					log.Printf("[terminal-ws] PTY 写错误: %v", wErr)
					return
				}
			}
		}
	}
}

// handleTextMessage 处理 JSON 控制消息。
func (ps *ptySession) handleTextMessage(payload []byte) {
	var msg struct {
		Type string `json:"type"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "resize":
		if msg.Cols > 0 && msg.Cols <= 500 && msg.Rows > 0 && msg.Rows <= 200 {
			ps.mu.Lock()
			ps.cols = msg.Cols
			ps.rows = msg.Rows
			sess := ps.sess
			ps.mu.Unlock()
			if sess != nil {
				sess.Resize(msg.Cols, msg.Rows)
			}
		}
	case "close":
		ps.close()
	}
}

// ─── wsConn 扩展：writeBinaryFrame ─────────────────────────

func (c *wsConn) writeBinaryFrame(data []byte) error {
	// 帧头：FIN=1, opcode=0x2 (binary)
	c.bw.WriteByte(0x82)
	n := len(data)
	switch {
	case n < 126:
		c.bw.WriteByte(byte(n))
	case n < 65536:
		c.bw.WriteByte(126)
		c.bw.WriteByte(byte(n >> 8))
		c.bw.WriteByte(byte(n))
	default:
		c.bw.WriteByte(127)
		for i := 5; i >= 0; i-- {
			c.bw.WriteByte(0)
		}
		c.bw.WriteByte(byte(n >> 8))
		c.bw.WriteByte(byte(n))
	}
	c.bw.Write(data)
	return c.bw.Flush()
}

// ─── Shell 辅助 ──────────────────────────────────────────

func ptyShellFor(code string) pty.Shell {
	switch code {
	case "cmd":
		return pty.ShellByName("CMD")
	case "powershell":
		if s := pty.ShellByName("PowerShell"); s.Name == "PowerShell" {
			return s
		}
		return pty.ShellByName("PowerShell 7")
	case "gitbash":
		return pty.ShellByName("Git Bash")
	}
	return pty.DefaultShell()
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
