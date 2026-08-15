// Command edge_cm6_ref: Edge headless dump the minimal CM6 page's gutter DOM
// structure (spacer/行号元素几何), as the browser baseline for wb-ui's CM6
// rendering comparison.
//go:build ignore

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
	// 优先 http；若 EDGE_REF_FILE 非空则用 file://（绕开代理/网络问题）
	fileURL := os.Getenv("EDGE_REF_FILE")
	var url string
	if fileURL != "" {
		url = "file:///" + fileURL
	} else {
		url = "http://127.0.0.1:" + port + "/cm6_ref.html"
	}
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
		fmt.Println("HEAD:", out[:min(len(out), 3000)])
		os.Exit(1)
	}
	var seg string
	// 找 .cm-editor 主体
	i := bytes.Index([]byte(out), []byte("cm-editor"))
	if i < 0 {
		i = 0
	}
	if i+6000 > len(out) {
		seg = out[i:]
	} else {
		seg = out[i : i+6000]
	}
	// 打印 title（CM6 是否执行）
	ti := bytes.Index([]byte(out), []byte("<title>"))
	if ti >= 0 {
		fmt.Println("TITLE:", out[ti:ti+200])
	}
	fmt.Println(seg)
}
