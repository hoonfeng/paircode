// Command edge_setmodal_ref launches Edge headless on ide_ref_setmodal.html
// (standalone settings-modal replica) to collect the REAL browser's modal
// geometry for comparison against wb-ui.
//go:build ignore

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
	port := os.Getenv("EDGE_REF_PORT")
	if port == "" {
		port = "9097"
	}
	url := "http://localhost:" + port + "/ide_ref_setmodal.html"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=1280,800", "--virtual-time-budget=12000",
		"--dump-dom", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	payload := extract(strings.Join([]string{stdout.String(), stderr.String()}, "\n"))
	if payload == "" {
		s := stdout.String()
		lt := strings.Index(s, "<title>")
		if lt >= 0 {
			rest := s[lt+7:]
			gt := strings.Index(rest, "</title>")
			if gt >= 0 {
				fmt.Println("title =", rest[:gt][:200])
			}
		}
		os.Exit(1)
	}
	fmt.Println("=== Edge 设置面板 modal 几何（浏览器标准）===")
	fmt.Println(payload)
}

func extract(s string) string {
	lt := strings.Index(s, "<title>")
	if lt < 0 {
		return ""
	}
	rest := s[lt+7:]
	gt := strings.Index(rest, "</title>")
	if gt < 0 {
		return ""
	}
	t := rest[:gt]
	if strings.HasPrefix(t, "SETMODAL-REF:") {
		return t
	}
	return ""
}
