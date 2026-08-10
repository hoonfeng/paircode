// Command edge_xtermcell_ref launches Edge headless on ide_ref_xtermcell.html
// to capture the real browser's rendering of an xterm-style DOM renderer
// structure (absolute inline-block spans inside 15px row divs) — the text
// baseline/vertical position inside each cell is the browser ground truth.
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
	url := "http://localhost:" + port + "/ide_ref_xtermcell.html"
	shot := "F:\\syproject\\gou-ide\\dev\\desktop_probe\\xtermcell_edge_shot.png"
	os.Remove(shot)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=400,80", "--force-device-scale-factor=1",
		"--virtual-time-budget=8000",
		"--screenshot="+shot, url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge screenshot: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	info, err := os.Stat(shot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no screenshot:", err)
		os.Exit(1)
	}
	fmt.Printf("saved: %s (%d bytes)\n", shot, info.Size())
}
