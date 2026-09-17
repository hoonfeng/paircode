// Package paircode PairCode 集成层：投喂消息 + 从会话 JSONL 获取最终回复。
//
// 会话文件：<workspaceRoot>/.pair/conversations/<convId>.jsonl
// 行结构（实测）：{"idx":int, "message":{...}, "eventType":"assistant/message"|"tool/result"|"user/message", ...}
//
// 蓝图：M1 Node 原型 temp/wx-bridge/src/paircode.mjs（线上实测版本）。
package paircode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// convFileLimitBytes 会话 JSONL 尾部读取上限（防大文件）。
const convFileLimitBytes int64 = 2_000_000

// ConvFilePath 返回会话 JSONL 文件路径。
func ConvFilePath(workspaceRoot, convID string) string {
	return filepath.Join(workspaceRoot, ".pair", "conversations", convID+".jsonl")
}

// Line 会话 JSONL 的一行（宽松结构，容忍字段缺失）。
type Line struct {
	Idx       int64       `json:"idx"`
	Message   LineMessage `json:"message"`
	Timestamp string      `json:"timestamp,omitempty"`
	EventType string      `json:"eventType,omitempty"`
	Step      int         `json:"step,omitempty"`
}

// LineMessage 消息体。
type LineMessage struct {
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content,omitempty"` // 多数为 string；保留原始 JSON 防类型异常
	ToolCalls []ToolCall      `json:"tool_calls,omitempty"`
}

// ToolCall 工具调用（OpenAI 兼容格式）。
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Text 返回 content 的文本形式（非字符串/空 → ""）。
func (m *LineMessage) Text() string {
	if len(m.Content) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(m.Content, &s) == nil {
		return s
	}
	return ""
}

// ReadConversation 读取会话 JSONL（读尾部至多 limitBytes，防大文件）。
// 返回解析成功的行（按文件顺序）；容忍半行/坏行。
func ReadConversation(workspaceRoot, convID string, limitBytes int64) []Line {
	if limitBytes <= 0 {
		limitBytes = convFileLimitBytes
	}
	f, err := os.Open(ConvFilePath(workspaceRoot, convID))
	if err != nil {
		return nil
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() == 0 {
		return nil
	}
	start := int64(0)
	if st.Size() > limitBytes {
		start = st.Size() - limitBytes
	}
	buf := make([]byte, st.Size()-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return nil
	}
	text := string(buf)
	if start > 0 {
		// 丢弃被截断的首行碎片
		i := strings.IndexByte(text, '\n')
		if i < 0 {
			return nil
		}
		text = text[i+1:]
	}
	out := make([]Line, 0, 64)
	for _, raw := range strings.Split(text, "\n") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		var o Line
		if json.Unmarshal([]byte(raw), &o) == nil {
			out = append(out, o)
		}
	}
	return out
}

// Send 投喂消息到 PairCode（POST /api/chat/send，与 UI 发送同构）。
func Send(baseURL, convID, workspaceRoot, message string, timeout time.Duration) error {
	body, err := json.Marshal(map[string]string{
		"message":       message,
		"convId":        convID,
		"workspaceRoot": workspaceRoot,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost,
		strings.TrimSuffix(baseURL, "/")+"/api/chat/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var data struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(raw, &data)
	if resp.StatusCode < 200 || resp.StatusCode > 299 || !data.OK {
		snippet := string(raw)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, orStr(data.Error, snippet))
	}
	return nil
}

func orStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Result 等待结果。
type Result struct {
	Text    string // 最终回复文本
	Idx     int64  // 对应会话行 idx
	Timeout bool   // 超时未获得最终回复
}

// Waiter 回复等待器：监听会话 JSONL 新增行，识别「最终回复」= 无 tool_calls 的
// assistant 消息，并以静默期（QuietMs）确认其稳定性；中间轮（带 tool_calls）
// 与 tool 结果会取消候选；遇到 ask_user 回调通知（不结束等待，等最终结果）。
type Waiter struct {
	WorkspaceRoot string
	ConvID        string
	PollMs        int // 轮询间隔
	QuietMs       int // 静默期：候选出现后无新行多久判定完成
	TimeoutMs     int // 单轮等待上限

	// Logf 可选日志（nil 静默）。
	Logf func(format string, args ...any)
}

func (w *Waiter) logf(format string, args ...any) {
	if w.Logf != nil {
		w.Logf(format, args...)
	}
}

// BaselineIdx 投喂前记录当前最大 idx（作为等待基线）；空文件返回 -1。
func (w *Waiter) BaselineIdx() int64 {
	lines := ReadConversation(w.WorkspaceRoot, w.ConvID, 0)
	if len(lines) == 0 {
		return -1
	}
	return lines[len(lines)-1].Idx
}

// WaitForReply 等待投喂后的最终回复。baseline 为投喂前基线 idx；
// onAskUser 可为 nil（ask_user 出现时回调一次问题文本，不结束等待）。
func (w *Waiter) WaitForReply(baseline int64, onAskUser func(question string)) Result {
	pollMs := w.PollMs
	if pollMs <= 0 {
		pollMs = 1500
	}
	quietMs := w.QuietMs
	if quietMs <= 0 {
		quietMs = 15000
	}
	timeoutMs := w.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 30 * 60 * 1000
	}

	file := ConvFilePath(w.WorkspaceRoot, w.ConvID)
	t0 := time.Now()
	lastSeen := baseline
	var candText string
	var candIdx int64
	var candAt time.Time
	haveCand := false
	fileSig := ""
	askNotified := false

	for time.Since(t0) < time.Duration(timeoutMs)*time.Millisecond {
		time.Sleep(time.Duration(pollMs) * time.Millisecond)

		sig := ""
		if st, err := os.Stat(file); err == nil {
			sig = fmt.Sprintf("%d:%d", st.Size(), st.ModTime().UnixNano())
		}
		if sig != fileSig {
			fileSig = sig
			lines := ReadConversation(w.WorkspaceRoot, w.ConvID, 0)
			// ★ 会话文件「重建/轮换」检测与重锚（2026-09-17 实锤）：会话「交接」
			//   会把活跃 JSONL 整体重写（idx 从 0 重新编号），旧基线可能大于新
			//   文件的 max idx——「idx > lastSeen」恒不满足，等待会一直卡到 30
			//   分钟超时（实测：投喂前读得基线 564，回合中被重建为 0..222，最终
			//   回复永不发出）。检测到 max idx 回退即重锚（lastSeen=-1 全扫）：
			//   下方处理分支天然忽略 user/tool 行与带 tool_calls 的 assistant
			//   行，重建文件尾部的最终回复仍会被识别为候选。
			maxIdx := int64(-1)
			for _, o := range lines {
				if o.Idx > maxIdx {
					maxIdx = o.Idx
				}
			}
			if maxIdx < lastSeen {
				w.logf("会话文件已重建（max idx=%d < 基线 %d），重锚全扫", maxIdx, lastSeen)
				lastSeen = -1
			}
			for _, o := range lines {
				if o.Idx <= lastSeen {
					continue
				}
				lastSeen = o.Idx
				switch o.Message.Role {
				case "assistant":
					tcs := o.Message.ToolCalls
					if q, ok := findAskUser(tcs); ok {
						if !askNotified && onAskUser != nil {
							askNotified = true
							onAskUser(q)
						}
						haveCand = false
						continue
					}
					content := strings.TrimSpace(o.Message.Text())
					if len(tcs) == 0 && content != "" {
						candText, candIdx, candAt = content, o.Idx, time.Now()
						haveCand = true
					} else {
						haveCand = false // 中间轮或空消息：取消候选
					}
				case "tool", "user":
					haveCand = false
				}
			}
		}

		if haveCand && time.Since(candAt) >= time.Duration(quietMs)*time.Millisecond {
			return Result{Text: candText, Idx: candIdx}
		}
	}
	return Result{Timeout: true}
}

// findAskUser 从 tool_calls 中提取 ask_user 问题文本（无则 ok=false）。
func findAskUser(tcs []ToolCall) (string, bool) {
	for _, tc := range tcs {
		if tc.Function.Name != "ask_user" {
			continue
		}
		var args struct {
			Question  string `json:"question"`
			Questions []struct {
				Question string `json:"question"`
			} `json:"questions"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if len(args.Questions) > 0 {
			parts := make([]string, 0, len(args.Questions))
			for _, q := range args.Questions {
				if q.Question != "" {
					parts = append(parts, q.Question)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n"), true
			}
		}
		return args.Question, true
	}
	return "", false
}
