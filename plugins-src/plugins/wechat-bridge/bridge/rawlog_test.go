// rawlog_test.go — raw 抓样器单元测试（写入格式/轮转/nil 安全/文件名清洗）。
package bridge

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func noopLogf(string, ...any) {}

func TestRawLogWriteAndRotate(t *testing.T) {
	dir := t.TempDir()
	r, err := newRawLogger(dir, "bot_x@im.bot", noopLogf)
	if err != nil {
		t.Fatalf("newRawLogger: %v", err)
	}
	defer r.close()

	// 写两条（第二条为非紧凑原帧，应被压成单行）；第三条非法 JSON 应被忽略。
	m1 := `{"message_type":1,"message_id":"m1","item_list":[{"type":1,"text_item":{"text":"hi"}}]}`
	m2 := "{\n  \"message_type\": 1,\n  \"message_id\": \"m2\",\n  \"item_list\": []\n}"
	r.write([]byte(m1))
	r.write([]byte(m2))
	r.write([]byte("not-json"))

	b, err := os.ReadFile(r.path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("行数=%d 期望 2：%q", len(lines), b)
	}
	var rec struct {
		T    string          `json:"t"`
		Evt  string          `json:"evt"`
		Acct string          `json:"account"`
		Raw  json.RawMessage `json:"raw"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("行1不是合法 JSON: %v", err)
	}
	if rec.Evt != "recv" || rec.Acct != "bot_x@im.bot" || rec.T == "" {
		t.Fatalf("字段不符: %+v", rec)
	}
	var inner map[string]any
	if err := json.Unmarshal(rec.Raw, &inner); err != nil || inner["message_id"] != "m1" {
		t.Fatalf("raw 不符: %v %v", inner, err)
	}

	// 轮转：当前文件 → .1（覆盖已有旧档），新档为空且可继续写。
	bakPath := strings.TrimSuffix(r.path, ".jsonl") + ".1.jsonl"
	if err := os.WriteFile(bakPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.rotate()
	if _, err := os.Stat(r.path); err != nil {
		t.Fatalf("轮转后新档不存在: %v", err)
	}
	b1, _ := os.ReadFile(bakPath)
	if string(b1) != string(b) {
		t.Fatalf(".1 档内容应为轮转前内容: %q", b1)
	}
	if fi, err := os.Stat(r.path); err != nil || fi.Size() != 0 {
		t.Fatalf("新档应为空: %v", err)
	}
	r.write([]byte(m1))
	if fi, err := os.Stat(r.path); err != nil || fi.Size() == 0 {
		t.Fatalf("轮转后写入失败: %v", err)
	}
}

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(`a/b\c:d*e?f"g<h>i|j`)
	want := "a_b_c_d_e_f_g_h_i_j"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
	if sanitizeFileName("") != "account" {
		t.Fatalf("空名应回落 account")
	}
}

func TestRawLoggerNilSafe(t *testing.T) {
	var r *rawLogger
	r.write([]byte("{}")) // nil 接收者不得 panic
	r.close()
}
