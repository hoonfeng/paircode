// Command edge_termfit_shot captures an Edge headless screenshot of the
// terminal fit reference page, so we can pixel-compare where the first
// line of text sits inside its 15px row box (baseline/top gap) between
// Edge (browser) and wb-ui (engine).
//
// Usage:
//   1. python -m http.server 9097   (in cmd/companion/web-ui)
//   2. go run ./dev/desktop_probe/edge_termfit_shot.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	out := filepath.Join(wd, "dev", "desktop_probe", "termfit_edge_shot.png")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=1280,800",
		"--virtual-time-budget=20000",
		"--screenshot="+out, url)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	fmt.Printf("=== Edge termfit screenshot → %s ===\n", out)
}
