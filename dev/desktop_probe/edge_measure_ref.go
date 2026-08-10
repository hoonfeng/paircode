// Command edge_measure_ref captures Edge headless measurement results of
// text widths (space / ASCII / CJK) in Consolas 13px, to compare with
// wb-ui engine measureText (space=7.8 vs A=7.147 vs cjk=13).
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
	url := "http://localhost:" + port + "/ide_ref_measure.html"
	out := filepath.Join(wd, "tmp", "edge_measure.png")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, edge,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--window-size=400,200",
		"--virtual-time-budget=10000",
		"--dump-dom", url)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "edge run: %v\n%s\n", err, stderr.String())
		os.Exit(1)
	}
	fmt.Printf("=== Edge measure (dump-dom) ===\n%s\n", stdout.String())
	_ = out
}
