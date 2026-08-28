package agent

// 回归测试：两个已知问题的修复验证
//  1. 后台进程跨轮次存活：Registry 每轮对话重建（web_server.go buildWebLoopOpts），
//     bgRegistry 若随之重建则 run_background 的进程在下一轮丢失 → 已改为全局单例 globalBG。

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestBGCrossRegistry 模拟 web 端两轮对话（两个独立 Registry），
// 第一轮 run_background 启动的进程，第二轮 read_output/kill_process 应仍可访问。
func TestBGCrossRegistry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	r1 := NewRegistry()
	RegisterDefaultTools(r1, root)
	out, err := r1.Execute(ctx, "run_background", `{"command":"echo cross_round_ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`id=(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("run_background 应返回进程 id，得 %q", out)
	}
	idArg := `{"id":` + m[1] + `}`

	// 模拟下一轮对话：全新 Registry（每轮 buildWebLoopOpts 都会新建）
	r2 := NewRegistry()
	RegisterDefaultTools(r2, root)

	var ro string
	for i := 0; i < 200; i++ { // 轮询至结束
		ro, err = r2.Execute(ctx, "read_output", idArg)
		if err == nil && strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || !strings.Contains(ro, "cross_round_ok") {
		t.Fatalf("跨轮次读输出失败: err=%v ro=%q", err, ro)
	}
	// kill 也应能找到进程（幂等）
	if _, err := r2.Execute(ctx, "kill_process", idArg); err != nil {
		t.Fatalf("跨轮次 kill_process 失败: %v", err)
	}
}
