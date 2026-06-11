//go:build windows

package pty

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConPTYStreams 验证 ConPTY 起 cmd 后，echo 输出在进程「存活期间」就流出来
// （裸管道做不到——全缓冲到退出才刷新；伪控制台给真 tty 才行）。
// 关键修复：STARTF_USESTDHANDLES 置位 + std 句柄留空 → 子 shell 不继承父控制台、由伪控制台接管 I/O。
func TestConPTYStreams(t *testing.T) {
	p, err := Start(Shell{Name: "CMD", Path: "cmd"}, ".", 80, 25)
	if err != nil {
		t.Fatalf("ConPTY Start 失败: %v", err)
	}
	defer p.Close()

	var mu sync.Mutex
	captured := ""
	hit := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				mu.Lock()
				captured += string(buf[:n])
				ok := strings.Contains(captured, "CONPTY_STREAM_OK")
				mu.Unlock()
				if ok {
					select {
					case hit <- struct{}{}:
					default:
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(400 * time.Millisecond) // 等 shell 起来 + 打印初始提示符
	if _, err := p.Write([]byte("echo CONPTY_STREAM_OK\r\n")); err != nil {
		t.Fatalf("写命令失败: %v", err)
	}

	select {
	case <-hit:
		// ✓ 流式输出生效（持久 shell 存活期间即收到 echo）
	case <-time.After(5 * time.Second):
		mu.Lock()
		c := captured
		mu.Unlock()
		t.Errorf("超时：进程存活期间未流出 echo 输出。管道捕获 %d 字节：%q", len(c), trunc(c, 400))
	}
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
