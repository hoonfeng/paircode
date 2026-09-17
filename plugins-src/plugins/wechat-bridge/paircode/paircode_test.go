package paircode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// —— 测试辅助 ——

func writeConvFile(t *testing.T, root, convID string, lines []map[string]any) {
	t.Helper()
	p := ConvFilePath(root, convID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := writeLines(f, lines); err != nil {
		t.Fatal(err)
	}
}

func appendConvFile(t *testing.T, root, convID string, lines []map[string]any) {
	t.Helper()
	f, err := os.OpenFile(ConvFilePath(root, convID), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := writeLines(f, lines); err != nil {
		t.Fatal(err)
	}
}

func writeLines(f *os.File, lines []map[string]any) error {
	for _, l := range lines {
		b, err := json.Marshal(l)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func msgLine(idx int, role, content string, withToolCalls bool) map[string]any {
	m := map[string]any{"role": role, "content": content}
	if withToolCalls {
		m["tool_calls"] = []map[string]any{{
			"id": "call_1", "type": "function",
			"function": map[string]any{"name": "exec_command", "arguments": "{}"},
		}}
	}
	return map[string]any{"idx": idx, "message": m, "eventType": role + "/message"}
}

// TestWaitForReplyReanchorAfterRewrite 复现 2026-09-17 实测事故：
// 桥投喂前读得基线 564，回合进行中「会话交接」把活跃 JSONL 整体重写
// （idx 从 0 重新编号、max=4），修复前「idx > lastSeen」恒不满足、等待
// 卡死到 30 分钟超时；修复后应检测到 max idx 回退并重锚全扫，识别重建
// 文件尾部的最终回复。
func TestWaitForReplyReanchorAfterRewrite(t *testing.T) {
	root := t.TempDir()
	const convID = "conv_test_rewrite"
	writeConvFile(t, root, convID, []map[string]any{
		msgLine(0, "user", "真正隔离了吗？", false),
		msgLine(1, "tool", "tool result", false),
		msgLine(2, "assistant", "", true), // 中间轮：带 tool_calls
		msgLine(3, "tool", "tool result", false),
		msgLine(4, "assistant", "最终回复正文", false),
	})

	w := &Waiter{WorkspaceRoot: root, ConvID: convID,
		PollMs: 50, QuietMs: 200, TimeoutMs: 4000}
	r := w.WaitForReply(564, nil) // 旧基线远大于重建文件 max idx(4)
	if r.Timeout {
		t.Fatal("等待超时：重建后未能重锚（修复前即此表现）")
	}
	if r.Text != "最终回复正文" {
		t.Fatalf("回复文本不符：%q", r.Text)
	}
	if r.Idx != 4 {
		t.Fatalf("回复 idx 不符：%d", r.Idx)
	}
}

// TestWaitForReplyNormalAppend 回归：文件正常追加（idx 递增）时不应触发
// 重锚，且只把「基线之后」的新行当候选（旧回复不得误发）。
func TestWaitForReplyNormalAppend(t *testing.T) {
	root := t.TempDir()
	const convID = "conv_test_append"
	writeConvFile(t, root, convID, []map[string]any{
		msgLine(0, "user", "旧问题", false),
		msgLine(1, "assistant", "旧回复（不得误发）", false),
	})

	w := &Waiter{WorkspaceRoot: root, ConvID: convID,
		PollMs: 50, QuietMs: 200, TimeoutMs: 4000}
	go func() {
		time.Sleep(150 * time.Millisecond)
		appendConvFile(t, root, convID, []map[string]any{
			msgLine(2, "user", "新问题", false),
			msgLine(3, "assistant", "", true),
			msgLine(4, "tool", "tool result", false),
			msgLine(5, "assistant", "新回复", false),
		})
	}()
	r := w.WaitForReply(1, nil) // 基线=旧文件 max idx
	if r.Timeout {
		t.Fatal("等待超时")
	}
	if r.Text != "新回复" {
		t.Fatalf("应返回新回复（不得误发旧回复）：%q", r.Text)
	}
	if r.Idx != 5 {
		t.Fatalf("回复 idx 不符：%d", r.Idx)
	}
}
