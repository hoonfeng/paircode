// Command edge_select_ref launches Edge headless on ide_ref_select.html to
// capture the REAL browser's native <select> rendering (arrow shape, colors,
// geometry) for comparison against wb-ui.
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
	url := "http://localhost:" + port + "/ide_ref_select.html"
	shot := "F:\\syproject\\gou-ide\\dev\\desktop_probe\\edge_select_shot.png"
	os.Remove(shot)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=640,420", "--force-device-scale-factor=1",
		"--virtual-time-budget=8000",
		"--screenshot="+shot, url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge screenshot: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	fmt.Println("=== Edge select 截图 ===")
	info, err := os.Stat(shot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no screenshot:", err)
		os.Exit(1)
	}
	fmt.Printf("saved: %s (%d bytes)\n", shot, info.Size())

	// Second pass: dump DOM for geometry.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel2()
	cmd2 := exec.CommandContext(ctx2, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=640,420", "--virtual-time-budget=8000",
		"--dump-dom", url)
	var out2, err2 bytes.Buffer
	cmd2.Stdout = &out2
	cmd2.Stderr = &err2
	if err := cmd2.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge dump: %v\n", err)
		os.Exit(1)
	}
	payload := extract(strings.Join([]string{out2.String(), err2.String()}, "\n"))
	fmt.Println("=== Edge select DOM/几何 ===")
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
	if strings.HasPrefix(t, "SELECT-REF:") {
		return t
	}
	return ""
}
