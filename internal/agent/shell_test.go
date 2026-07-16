//go:build windows

package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRunBackground 后台启动 echo → 轮询 read_output 至结束 → 输出含 echo 内容。
func TestRunBackground(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	ctx := context.Background()

	out, err := r.Execute(ctx, "run_background", `{"command":"echo bg_hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "id=1") {
		t.Fatalf("应返回 id=1，得 %q", out)
	}

	var ro string
	for i := 0; i < 200; i++ { // 轮询至结束（echo 很快）
		ro, _ = r.Execute(ctx, "read_output", `{"id":1}`)
		if strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(ro, "bg_hello") {
		t.Errorf("输出缺 echo 内容：%q", ro)
	}
}

// TestReadOutputUnknown 读未知 id 应报错。
func TestReadOutputUnknown(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	if _, err := r.Execute(context.Background(), "read_output", `{"id":999}`); err == nil {
		t.Error("未知 id 应报错")
	}
}
