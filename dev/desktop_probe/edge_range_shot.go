// Command edge_range_shot captures Edge headless screenshot of ide_ref_range.html
// for pixel-level comparison of the native range slider (track/thumb) vs wb-ui.
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
	url := "http://localhost:" + port + "/ide_ref_range.html"
	out := `F:\syproject\gou-ide\dev\desktop_probe\edge_range.png`
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=1280,800", "--virtual-time-budget=12000",
		"--screenshot="+out, url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	fmt.Println("edge_range.png saved")
}
