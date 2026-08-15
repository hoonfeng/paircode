// Command edge_termfit_ref collects the terminal-fit reference tree from
// Edge headless — the SAME page (ide_ref_termfit.html, served over http)
// renders a fixed 800px terminal container and runs FitAddon.fit() right
// after open() (real TerminalPanel flow). The page serializes the tree +
// diag (cols/rows/cell size/line widths) into document.title.
//
// Usage:
//   1. python -m http.server 9097   (in cmd/companion/web-ui)
//   2. go run ./dev/desktop_probe/edge_termfit_ref.go
//go:build ignore

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
	port := os.Getenv("EDGE_REF_PORT")
	if port == "" {
		port = "9097"
	}
	url := "http://localhost:" + port + "/ide_ref_termfit.html"
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
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
		fmt.Fprintf(os.Stderr, "bad JSON (%d bytes): %v\n", len(payload), err)
		os.Exit(1)
	}
	// 提取 __diag 单独打印
	var diag string
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &m); err == nil {
		if d, ok := m["__diag"]; ok {
			db, _ := json.Marshal(d)
			diag = string(db)
		}
	}
	out := filepath.Join(wd, "dev", "desktop_probe", "termfit_tree_edge.json")
	var buf bytes.Buffer
	json.Indent(&buf, []byte(payload), "", " ")
	if err := os.WriteFile(out, buf.Bytes(), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("=== Edge termfit tree → %s (%d bytes) ===\n", out, buf.Len())
	fmt.Println("EDGE_DIAG:", diag)
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
