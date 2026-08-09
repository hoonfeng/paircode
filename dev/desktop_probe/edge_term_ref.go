// Command edge_term_ref launches Edge headless on the terminal reference
// page (cmd/companion/web-ui/ide_ref_term.html, served over http), which
// renders FIXED cmd-like content through xterm.js (DOM renderer, same as
// wb-ui desktop) and serializes the terminal DOM tree into document.title.
// Parses the "TREE:" payload into dev/desktop_probe/term_tree_edge.json.
//
// Requires: python http.server on 9097 serving cmd/companion/web-ui
//   (cd cmd/companion/web-ui && python -m http.server 9097)
//
// Run: go run ./dev/desktop_probe/edge_term_ref.go
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
	port := os.Getenv("TERM_REF_PORT")
	if port == "" {
		port = "9097"
	}
	url := "http://localhost:" + port + "/ide_ref_term.html"
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--window-size=1280,800", "--virtual-time-budget=20000",
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
		s := stdout.String()
		if len(s) > 600 {
			s = s[:600]
		}
		fmt.Fprintln(os.Stderr, "stdout head:", s)
		fmt.Fprintln(os.Stderr, "no TREE payload; stderr:", stderr.String())
		os.Exit(1)
	}
	var v interface{}
	if err := json.Unmarshal([]byte(payload), &v); err != nil {
		fmt.Fprintf(os.Stderr, "bad JSON in payload (%d bytes): %v\n", len(payload), err)
		os.Exit(1)
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "term_tree_edge.json")
	var buf bytes.Buffer
	json.Indent(&buf, []byte(payload), "", " ")
	if err := os.WriteFile(out, buf.Bytes(), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("=== Edge 终端树 dump → %s (%d bytes) ===\n", out, buf.Len())
	printTermContainers(payload)
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

// printTermContainers 输出终端关键容器几何与字符行信息。
func printTermContainers(payload string) {
	var rootM map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &rootM); err != nil {
		fmt.Println("parse error:", err)
		return
	}
	fmt.Println("=== Edge 终端关键元素 ===")
	keys := map[string]bool{
		"xterm": true, "xterm-screen": true, "xterm-viewport": true,
		"xterm-rows": true, "xterm-row": true, "xterm-char": true,
		"xterm-cursor": true,
	}
	var find func(v map[string]interface{}, depth int)
	find = func(v map[string]interface{}, depth int) {
		cls, _ := v["cls"].(string)
		if cls != "" {
			for k := range keys {
				if strings.Contains(cls, k) {
					x, _ := v["x"].(float64)
					y, _ := v["y"].(float64)
					w, _ := v["w"].(float64)
					h, _ := v["h"].(float64)
					txt, _ := v["text"].(string)
					fs, _ := v["fs"].(string)
					lh, _ := v["lh"].(string)
					fmt.Printf("  %-16s x=%.1f y=%.1f w=%.1f h=%.1f fs=%s lh=%s txt=%q\n",
						k, x, y, w, h, fs, lh, txt)
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
	find(rootM, 0)
}
