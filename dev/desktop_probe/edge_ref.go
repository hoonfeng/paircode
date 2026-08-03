// Command edge_ref launches Edge headless on the IDE reference wrapper
// (cmd/companion/web-ui/ide_ref.html, served over http), which serializes the
// COMPLETE element tree of the real IDE into document.title, then parses the
// "TREE:" payload into dev/desktop_probe/ide_tree_edge.json.
//
// Requires: python http.server on 9093 serving cmd/companion/web-ui
//   (cd cmd/companion/web-ui && python -m http.server 9093)
//
// Run: go run ./dev/desktop_probe/edge_ref.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	wd, _ := os.Getwd()
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	if _, err := os.Stat(edge); err != nil {
		edge = `C:\Program Files\Microsoft\Edge\Application\msedge.exe`
	}
	// 端口可用环境变量覆盖（默认 9096；9090 是正在运行的 companion，不动它）。
	port := os.Getenv("EDGE_REF_PORT")
	if port == "" {
		port = "9096"
	}
	url := "http://localhost:" + port + "/ide_ref.html"
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--window-size=1280,800", "--virtual-time-budget=15000",
		"--dump-dom", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	payload := extractTree(stdout.String())
	if payload == "" {
		payload = extractTree(stderr.String())
	}
	if payload == "" {
		// 诊断：打印 stdout 头部
		s := stdout.String()
		if len(s) > 600 { s = s[:600] }
		fmt.Fprintln(os.Stderr, "stdout head:", s)
		fmt.Fprintln(os.Stderr, "no TREE payload; stderr:", stderr.String())
		os.Exit(1)
	}
	// JSON 校验 + 落盘
	var v interface{}
	if err := json.Unmarshal([]byte(payload), &v); err != nil {
		fmt.Fprintf(os.Stderr, "bad JSON in payload (%d bytes): %v\n", len(payload), err)
		os.Exit(1)
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "ide_tree_edge.json")
	var buf bytes.Buffer
	json.Indent(&buf, []byte(payload), "", " ")
	if err := os.WriteFile(out, buf.Bytes(), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("=== Edge 完整元素树 dump → %s (%d bytes) ===\n", out, buf.Len())
	// 打印根布局概览：找到 main-area / right-container 等关键容器的几何
	printKeyContainers(payload)
}

func extractTree(s string) string {
	idx := strings.Index(s, "TREE:")
	if idx < 0 {
		return ""
	}
	rest := s[idx+5:]
	end := strings.Index(rest, "</title>")
	if end >= 0 {
		return rest[:end]
	}
	end = strings.IndexByte(rest, '<')
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// printKeyContainers 输出关键容器的几何（与 wb-ui ide_tree_dump 对照）。
func printKeyContainers(payload string) {
	var root struct {
		Children []struct {
			Tag      string `json:"tag"`
			ID       string `json:"id"`
			Class    string `json:"class"`
			X, Y     float64
			W, H     float64
			Children []struct {
				Tag   string `json:"tag"`
				Class string `json:"class"`
				X, Y  float64
				W, H  float64
				Children []struct {
					Tag   string `json:"tag"`
					Class string `json:"class"`
					X, Y  float64
					W, H  float64
				} `json:"children"`
			} `json:"children"`
		} `json:"children"`
	}
	if err := json.Unmarshal([]byte(payload), &root); err != nil {
		fmt.Println("parse error:", err)
		return
	}
	keys := map[string]bool{"titlebar": true, "activity-bar": true, "sidebar": true, "main-area": true, "right-container": true, "status-bar": true, "editor-area": true, "bottom-panel": true}
	fmt.Println("=== Edge 关键容器几何 ===")
	var walk func(ns []json.RawMessage, depth int)
	_ = walk
	// 简化：递归找 class 匹配的节点
	var find func(v map[string]interface{}, depth int)
	find = func(v map[string]interface{}, depth int) {
		cls, _ := v["class"].(string)
		if cls != "" {
			for k := range keys {
				if strings.Contains(cls, k) {
					x, _ := v["x"].(float64)
					y, _ := v["y"].(float64)
					w, _ := v["w"].(float64)
					h, _ := v["h"].(float64)
					fmt.Printf("  %-16s x=%.1f y=%.1f w=%.1f h=%.1f cls=%q\n", k, x, y, w, h, cls)
				}
			}
		}
		if ch, ok := v["children"].([]interface{}); ok {
			for _, c := range ch {
				if m, ok := c.(map[string]interface{}); ok {
					find(m, depth+1)
				}
			}
		}
	}
	var rootM map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &rootM); err == nil {
		find(rootM, 0)
	}
}
