// rawlog.go — raw 消息抓样（协议调试设施）。
//
// 把收（getupdates）到的每条原始消息按行追加到 <data-dir>/raw/<account>.jsonl：
//
//	{"t":"2026-09-16T20:30:15.123+08:00","evt":"recv","account":"bot_x","raw":{…原帧…}}
//
// 用途：协议字段校准（媒体 media 字段/CDN 域，P0-1/P2-4）、故障复盘的原始证据。
// 设计：默认开启（数据留在本机）；单文件阈值轮转一级（<account>.1.jsonl），
// 单账号磁盘占用上限约 2×rawFileMaxBytes；开关见 config.RawDump / WX_BRIDGE_RAW_DUMP。
//
// 并发说明：仅由 pollLoop 单 goroutine 调用（enqueue 内），不做加锁。
package bridge

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// rawFileMaxBytes 单文件轮转阈值（32MB）。
const rawFileMaxBytes = 32 << 20

// rawLogger 单账号抓样器。
type rawLogger struct {
	log     logf
	path    string // 当前文件绝对路径
	account string
	f       *os.File
	size    int64
	warned  bool // 写失败只告警一次
}

// newRawLogger 创建抓样器：<dataDir>/raw/<account>.jsonl（目录自动创建）。
func newRawLogger(dataDir, account string, log logf) (*rawLogger, error) {
	dir := filepath.Join(dataDir, "raw")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	r := &rawLogger{
		log:     log,
		path:    filepath.Join(dir, sanitizeFileName(account)+".jsonl"),
		account: account,
	}
	return r, r.open()
}

func (r *rawLogger) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	r.f = f
	if fi, err := f.Stat(); err == nil {
		r.size = fi.Size()
	}
	return nil
}

// write 追加一条原始消息（单行 JSON；失败静默降级、首次告警）。
func (r *rawLogger) write(raw []byte) {
	if r == nil || r.f == nil || len(raw) == 0 {
		return
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return // 非法 JSON 不落（正常不应发生）
	}
	line, err := json.Marshal(struct {
		T    string          `json:"t"`
		Evt  string          `json:"evt"`
		Acct string          `json:"account"`
		Raw  json.RawMessage `json:"raw"`
	}{
		T:    time.Now().Format(time.RFC3339Nano),
		Evt:  "recv",
		Acct: r.account,
		Raw:  compact.Bytes(),
	})
	if err != nil {
		return
	}
	line = append(line, '\n')
	n, err := r.f.Write(line)
	if err != nil {
		if !r.warned {
			r.warned = true
			if r.log != nil {
				r.log("[raw] 抓样写入失败（后续静默）: %v", err)
			}
		}
		return
	}
	r.size += int64(n)
	if r.size >= rawFileMaxBytes {
		r.rotate()
	}
}

// rotate 轮转：当前文件 → <account>.1.jsonl（覆盖旧档），重开新档。
func (r *rawLogger) rotate() {
	_ = r.f.Close()
	r.f = nil
	dst := strings.TrimSuffix(r.path, ".jsonl") + ".1.jsonl"
	if err := os.Rename(r.path, dst); err != nil && r.log != nil {
		r.log("[raw] 轮转失败（继续追加当前文件）: %v", err)
	}
	r.size = 0
	if err := r.open(); err != nil && r.log != nil {
		r.log("[raw] 抓样重开失败（暂停写入）: %v", err)
	}
}

// close 关闭文件（幂等）。
func (r *rawLogger) close() {
	if r == nil || r.f == nil {
		return
	}
	_ = r.f.Close()
	r.f = nil
}

// sanitizeFileName 过滤 Windows 文件名非法字符。
func sanitizeFileName(s string) string {
	if s == "" {
		return "account"
	}
	return strings.Map(func(ch rune) rune {
		switch ch {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return ch
	}, s)
}
