// Command edge_mainarea renders the companion frontend in Edge and crops the
// main-area (x327-425, y30-600) to compare with wb-ui.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func edgePath() string {
	for _, p := range []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(distDir))}
	go srv.Serve(ln)
	defer srv.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	edge := edgePath()
	edgeOut := filepath.Join(wd, "dev", "desktop_probe", "cmp_edge.png")
	url := fmt.Sprintf("http://127.0.0.1:%d/index.html", port)
	cmd := exec.Command(edge,
		"--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--force-device-scale-factor=1", "--window-size=1280,800",
		"--virtual-time-budget=8000", "--screenshot="+edgeOut,
		url)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("edge: %v %s", err, out)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(edgeOut); err == nil {
			break
		}
		if time.Now().After(deadline) {
			log.Fatal("timeout")
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(1500 * time.Millisecond)

	f, _ := os.Open(edgeOut)
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	// main-area: x327-430, y30-600
	sub := image.NewRGBA(image.Rect(0, 0, 103, 570))
	ink := 0
	for y := 0; y < 570; y++ {
		for x := 0; x < 103; x++ {
			c := img.At(327+x, 30+y)
			r, g, b, _ := c.RGBA()
			sub.Set(x, y, c)
			if r < 30000 && g < 30000 && b < 30000 {
				ink++
			}
		}
	}
	fo, _ := os.Create(wd + "\\dev\\desktop_probe\\mainarea_edge.png")
	png.Encode(fo, sub)
	fo.Close()
	fmt.Printf("main-area edge ink=%d (of %d)\n", ink, 103*570)
	_ = strings.ReplaceAll
}
