package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hoonfeng/paircode/internal/pty"
)

func main() {
	sh := pty.ShellByName("cmd")
	fmt.Printf("shell: %q\n", sh)
	p, err := pty.Start(sh, "F:\\syproject\\gou-ide", 80, 24)
	if err != nil {
		fmt.Println("start err:", err)
		os.Exit(1)
	}
	defer p.Close()

	buf := make([]byte, 4096)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		// 用 goroutine 读，主循环检查 deadline 避免阻塞
		done := make(chan int, 1)
		var n int
		var err error
		go func() {
			n, err = p.Read(buf)
			done <- n
		}()
		select {
		case n = <-done:
			if n > 0 {
				fmt.Printf("READ %d bytes:\n%q\n--- hex ---\n", n, string(buf[:n]))
				for i := 0; i < n; i++ {
					fmt.Printf("%02x ", buf[i])
					if (i+1)%16 == 0 {
						fmt.Println()
					}
				}
				fmt.Println()
			}
			if err != nil {
				fmt.Println("read err:", err)
				os.Exit(0)
			}
		case <-time.After(100 * time.Millisecond):
			// 无数据，继续等（主循环 check deadline）
		}
	}
	fmt.Println("DONE (2s window)")
}
