// ═══════════════════════════════════════════════════════════════
// terminal_ws.go — PTY 终端 WebSocket 端点（框架层）
//
// 从 cmd/companion/terminal_ws.go 下沉（2026-08-16）：终端能力由框架层
// 提供（agent.ServeTerminalWS），main 只保留一行路由注册。
//
// 协议：
//   客户端发文本帧（JSON 控制消息）：{type:"init", shell:"bash", cwd:"..."} / {type:"resize", cols:N, rows:N}
//   服务端发文本帧：{type:"ready"} / {type:"error", msg:"..."} / {type:"closed"}
//   双向二进制帧 = 原始 PTY I/O 字节流（VT 转义序列，xterm.js 渲染）
//
// 安全措施：
//   - shell 白名单（仅 pty.DetectShells 探测到的本机解释器；兼容历史别名）
//   - cwd 路径校验（禁止穿越出工作区）
//   - PTY 关闭时强制终止子进程（Windows ConPTY / Unix pts 均支持）
//   - 并发 PTY 会话数限制（最多 16）
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/pty"
)

// ─── 并发限制 ──────────────────────────────────────────────

var (
	activePTYSessions int32 // 原子计数
	maxPTYSessions    int32 = 16
)

// ─── ptySession ────────────────────────────────────────────

type ptySession struct {
	mu     sync.Mutex
	sess   pty.PTY
	conn   *WSConn
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
	return conn.WriteBinaryFrame(data)
}

func (ps *ptySession) writeText(data []byte) error {
	ps.mu.Lock()
	conn := ps.conn
	ps.mu.Unlock()
	if conn == nil {
		return nil
	}
	return conn.WriteTextFrame(data)
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
		conn.WriteTextFrame([]byte(`{"type":"closed"}`))
		conn.Close()
	}
	atomic.AddInt32(&activePTYSessions, -1)
}

// ─── 处理器 ────────────────────────────────────────────────

// ServeTerminalWS 是 /api/terminal/ws 端点处理器（框架层）：
// xterm.js + ConPTY 双向终端桥。
func ServeTerminalWS(w http.ResponseWriter, r *http.Request) {
	// 并发限制
	if atomic.LoadInt32(&activePTYSessions) >= maxPTYSessions {
		http.Error(w, "Too many active terminal sessions", http.StatusServiceUnavailable)
		return
	}

	conn, err := UpgradeWS(w, r)
	if err != nil {
		return
	}

	atomic.AddInt32(&activePTYSessions, 1)

	ps := &ptySession{
		conn: conn,
		cols: 80,
		rows: 24,
		cwd:  safeCWD(""),
	}

	// 等待客户端发送 init 消息（文本帧 JSON），15s 超时防泄漏：
	// 连接后不发 init 的僵尸连接（半开/崩溃）会被清理，避免并发计数永久占满 → 503
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	opcode, payload, err := conn.ReadFrame()
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		atomic.AddInt32(&activePTYSessions, -1)
		conn.Close()
		return
	}

	if opcode == 0x1 {
		if err := ps.handleInitMessage(payload); err != nil {
			ps.writeText([]byte(`{"type":"error","msg":"` + jsonEscape(err.Error()) + `"}`))
			ps.close()
			return
		}
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
// 返回非 nil 表示客户端指定了本机不存在的 shell（应回错误而不是静默降级）。
func (ps *ptySession) handleInitMessage(payload []byte) error {
	var msg struct {
		Type  string `json:"type"`
		Shell string `json:"shell"`
		Cwd   string `json:"cwd"`
		Cols  int    `json:"cols"`
		Rows  int    `json:"rows"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil || msg.Type != "init" {
		return nil // 非 init / 坏帧：忽略，走默认 shell
	}

	// shell 白名单校验：必须是本机探测到的解释器（兼容历史别名）
	if msg.Shell != "" {
		sh, err := resolveShell(msg.Shell)
		if err != nil {
			return err
		}
		ps.shell = sh.Name
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
	return nil
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

// startPTY 创建伪终端会话（Windows ConPTY / Unix pts）。
func (ps *ptySession) startPTY() error {
	sh, err := resolveShell(ps.shell)
	if err != nil {
		return err
	}
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
// ptyStats* 是 WB_PTY_STATS=1 的诊断计数（Read 次数/字节量，每 3s 一行）：
// 用于区分「空转的零字节读」与「shell 真实高频输出」。
var (
	ptyStatsOn   = os.Getenv("WB_PTY_STATS") != ""
	ptyStatReads atomic.Int64
	ptyStatBytes atomic.Int64
	ptyStatLast  atomic.Int64
)

func (ps *ptySession) ptyReader() {
	buf := make([]byte, 65536)
	// ★ 空读退避（性能关键）：ConPTY 管道在 shell 退出/半关闭等状态下可能
	//   出现「立即返回 (0, nil)」——此前的循环直接重试，单核 100% 空转
	//   （实测打开编辑器后终端面板挂起即整机 CPU 100%，pprof 显示
	//   ptyReader→windowsPty.Read 占 43% 采样）。这里对「无数据无错误」
	//   的读让出 2ms（连续空读超过 50 次后降频到 10ms），正常路径
	//   （n>0 或 err!=nil）不受影响。
	zeroReads := 0
	for {
		ps.mu.Lock()
		sess := ps.sess
		closed := ps.closed
		ps.mu.Unlock()

		if closed || sess == nil {
			return
		}

		n, err := sess.Read(buf)
		if ptyStatsOn {
			if n > 0 {
				ptyStatBytes.Add(int64(n))
			}
			reads := ptyStatReads.Add(1)
			last := ptyStatLast.Load()
			if last == 0 {
				ptyStatLast.Store(time.Now().UnixNano())
			} else if time.Since(time.Unix(0, last)) >= 3*time.Second &&
				ptyStatLast.CompareAndSwap(last, time.Now().UnixNano()) {
				log.Printf("[pty-stats] reads=%d bytes=%d closed=%v",
					reads, ptyStatBytes.Load(), closed)
				ptyStatReads.Store(0)
				ptyStatBytes.Store(0)
			}
		}
		if n > 0 {
			zeroReads = 0
			data := make([]byte, n)
			copy(data, buf[:n])
			if wErr := ps.writeBinary(data); wErr != nil {
				log.Printf("[terminal-ws] 写 WebSocket 失败: %v", wErr)
				ps.close()
				return
			}
		}
		if n == 0 && err == nil {
			zeroReads++
			if zeroReads == 5000 {
				log.Printf("[terminal-ws] PTY 连续 %d 次空读（无数据无错误），已降频等待；多半是 shell 已退出但管道未关", zeroReads)
			}
			d := 2 * time.Millisecond
			if zeroReads > 50 {
				d = 10 * time.Millisecond
			}
			ps.mu.Lock()
			closedNow := ps.closed
			ps.mu.Unlock()
			if closedNow {
				return
			}
			time.Sleep(d)
			continue
		}
		if err != nil {
			// ★ 正常关闭竞态不打印：wsReader 返回 → ps.close() → sess.Close()
			//   → 此处读已关闭的 sess → "file already closed"（无害）。
			//   只有未标记关闭时的读错误才是真异常。
			ps.mu.Lock()
			closed := ps.closed
			ps.mu.Unlock()
			if closed {
				return
			}
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

		opcode, payload, err := conn.ReadFrame()
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

// ─── Shell 辅助 ──────────────────────────────────────────

// resolveShell 把客户端 shell 标识解析为本机真实可用的 shell。
//
// 标识 = pty.DetectShells 返回的 Name（bash/zsh/fish/sh/CMD/PowerShell/Git Bash…），
// 同时兼容历史前端写死的三个值（cmd/powershell/gitbash）——旧 localStorage
// 里可能还存着它们。
func resolveShell(code string) (pty.Shell, error) {
	shells := pty.DetectShells()
	if code == "" {
		if len(shells) > 0 {
			return shells[0], nil
		}
		return pty.DefaultShell(), nil
	}
	for _, s := range shells {
		if strings.EqualFold(s.Name, code) {
			return s, nil
		}
	}
	aliases := map[string][]string{
		"cmd":        {"CMD"},
		"powershell": {"PowerShell", "PowerShell 7"},
		"gitbash":    {"Git Bash"},
		"bash":       {"bash"},
		"zsh":        {"zsh"},
		"fish":       {"fish"},
		"sh":         {"sh"},
	}
	for _, name := range aliases[strings.ToLower(code)] {
		for _, s := range shells {
			if strings.EqualFold(s.Name, name) {
				return s, nil
			}
		}
	}
	return pty.Shell{}, fmt.Errorf("不支持的终端类型 %q（本机可用：%s）", code, shellNames(shells))
}

// shellNames 逗号拼接可用 shell 名（错误信息用）。
func shellNames(shells []pty.Shell) string {
	names := make([]string, 0, len(shells))
	for _, s := range shells {
		names = append(names, s.Name)
	}
	if len(names) == 0 {
		return "无"
	}
	return strings.Join(names, ", ")
}

// ServeTerminalShells 返回本机可用的终端解释器列表。
//
// 前端下拉框据此渲染（Linux/macOS 是 bash/zsh/…，Windows 是 CMD/PowerShell/
// Git Bash），不再写死平台选项——这是 2026-10 之前 Linux/macOS 终端不可用的
// 直接原因之一（前端只会发 cmd/powershell/gitbash，Unix 端解析不到）。
func ServeTerminalShells(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	shells := pty.DetectShells()
	type shellItem struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Default bool   `json:"default"`
	}
	out := struct {
		Shells []shellItem `json:"shells"`
	}{Shells: make([]shellItem, 0, len(shells))}
	for i, s := range shells {
		out.Shells = append(out.Shells, shellItem{Name: s.Name, Path: s.Path, Default: i == 0})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return s
}
