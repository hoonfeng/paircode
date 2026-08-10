// Command edge_cm6_ref: Edge headless dump the minimal CM6 page's gutter DOM
// structure (spacer/行号元素几何), as the browser baseline for wb-ui's CM6
// rendering comparison.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edge); err != nil {
		edge = `C:\Program Files\Microsoft\Edge\Application\msedge.exe`
	}
	port := os.Getenv("EDGE_REF_PORT")
	if port == "" {
		port = "9097"
	}
	url := "http://localhost:" + port + "/cm6_ref.html"
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=900,500",
		"--virtual-time-budget=15000",
		"--dump-dom", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	out := stdout.String()
	// 只输出 .cm- 相关 DOM 结构（gutter 前 6 个元素 + activeLine）
	start := bytes.Index([]byte(out), []byte("cm-gutterElement"))
	if start < 0 {
		fmt.Println("no gutterElement in dump; total", len(out))
		os.Exit(1)
	}
	// 找 .cm-editor 主体
	i := bytes.Index([]byte(out), []byte("cm-editor"))
	if i < 0 {
		i = 0
	}
	seg := out[i : i+6000]
	fmt.Println(seg)
}
