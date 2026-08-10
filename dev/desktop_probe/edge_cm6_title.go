// Command edge_cm6_title: Edge headless 加载 cm6_ref.html，提取 <title>CM6-REF:</title>
// 里的几何基准数据（gutter/行号宽度、# 注释字符 x 偏移），作为 wb-ui 的浏览器基准。
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edge); err != nil {
		edge = `C:\Program Files\Microsoft\Edge\Application\msedge.exe`
	}
	fmt.Println("[dbg] edge=", edge)
	if _, err := os.Stat(edge); err != nil {
		fmt.Println("[dbg] edge NOT FOUND")
		os.Exit(2)
	}
	fmt.Println("[dbg] edge exists")
	port := os.Getenv("EDGE_REF_PORT")
	if port == "" {
		port = "9093"
	}
	url := "http://localhost:" + port + "/cm6_ref2.html"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--user-data-dir=F:\\syproject\\gou-ide\\dev\\desktop_probe\\tmp_edge_profile",
		"--window-size=900,600",
		"--virtual-time-budget=20000",
		"--dump-dom", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	s := stdout.String()
	// 打印 dump 尾部 1200 字符（看 JS 是否执行、有 cm- 结构）
	ln := len(s)
	if ln > 1200 {
		fmt.Println("DUMP-TAIL:", s[ln-1200:])
	}
	// 提取 title 标签内容
	lt := strings.Index(s, "<title>")
	if lt < 0 {
		fmt.Println("no title; total", len(s))
		os.Exit(1)
	}
	rest := s[lt+7:]
	gt := strings.Index(rest, "</title>")
	if gt < 0 {
		fmt.Println("no title close")
		os.Exit(1)
	}
	t := rest[:gt]
	fmt.Println("TITLE:", t)
	// 也输出 .cm- 相关 DOM 片段（验证结构）
	ci := strings.Index(s, "cm-editor")
	if ci >= 0 {
		seg := s[ci : ci+1500]
		// 找 gutterElement 的第一个
		gi := strings.Index(seg, "gutterElement")
		if gi >= 0 {
			fmt.Println("GUTTER-EL:", seg[gi-80:gi+200])
		}
	}
}
