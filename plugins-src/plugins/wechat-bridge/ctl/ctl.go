// Package ctl 实现 stdio 控制面：stdout 行 JSON 事件输出、stdin 行 JSON 命令读取。
//
// 契约（方案 §3.1）：
//
//	壳→桥（stdin）：{"cmd":"status|stop|relogin|verify_code|accounts.remove|qr.refresh", ...}
//	桥→壳（stdout）：{"evt":"hello|status|account|log|error", ...}
//
// 桥为纯行协议：stdout 只允许事件 JSON（日志走 stderr），避免污染协议流。
package ctl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"
)

// Bus 事件输出总线（并发安全；每事件一行 JSON 写 stdout）。
type Bus struct {
	mu sync.Mutex
	w  io.Writer
}

// NewBus 创建绑定 stdout 的事件总线。
func NewBus() *Bus {
	return &Bus{w: os.Stdout}
}

// NewBusTo 创建写往指定 writer 的总线（测试用）。
func NewBusTo(w io.Writer) *Bus {
	return &Bus{w: w}
}

// Emit 输出一条事件（map 序列化为一行 JSON）。
func (b *Bus) Emit(m map[string]any) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	_, _ = b.w.Write(append(data, '\n'))
}

// Hello 启动握手事件。
func (b *Bus) Hello(version string, pid, port int, reused bool, accounts []string) {
	if accounts == nil {
		accounts = []string{}
	}
	b.Emit(map[string]any{
		"evt": "hello", "version": version, "pid": pid,
		"port": port, "reused": reused, "accounts": accounts,
	})
}

// Log 日志事件（level: info|warn|error）。
func (b *Bus) Log(level, msg string) {
	b.Emit(map[string]any{"evt": "log", "level": level, "msg": msg})
}

// Error 错误事件（fatal=true 表示桥即将退出）。
func (b *Bus) Error(account, msg string, fatal bool) {
	m := map[string]any{"evt": "error", "msg": msg, "fatal": fatal}
	if account != "" {
		m["account"] = account
	}
	b.Emit(m)
}

// Command 壳→桥命令（字段按命令类型选用）。
type Command struct {
	Cmd     string `json:"cmd"`
	Account string `json:"account,omitempty"` // relogin/verify_code/qr.refresh/accounts.remove
	Code    string `json:"code,omitempty"`    // verify_code
	Alias   string `json:"alias,omitempty"`   // 预留：添加账号别名
}

// ReadCommands 从 stdin 循环读行 JSON 命令。
//   - 每条合法命令调用 onCmd
//   - stdin EOF（宿主退出/管道断开）或读取错误时调用 onEOF 后返回（自守护关键路径：
//     EOF 即宿主已死，桥必须自杀，避免孤儿残留）
//   - 坏行静默忽略（协议鲁棒性）
func ReadCommands(onCmd func(Command), onEOF func()) {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var c Command
		if err := json.Unmarshal(line, &c); err != nil || c.Cmd == "" {
			continue
		}
		onCmd(c)
	}
	if onEOF != nil {
		onEOF()
	}
}
