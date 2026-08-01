// Command edge_compare renders the desktop Vue app through Edge headless
// (full JS) and wb-ui (jsc), then pixel-compares the two screenshots.
// Edge is the reference: any large diff region is a wb-ui rendering defect.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"wb-ui/webkit"
)

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				re := regexp.MustCompile(`\[data-v-[a-f0-9]+\]`)
				return re.ReplaceAllString(string(data), ""), nil
			}
		}
	}
}

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")

	// ── 1. wb-ui render ──
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	raw, err := wv.Render()
	if err != nil {
		log.Fatal(err)
	}
	w, h := wv.Width(), wv.Height()
	wb := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			if off+3 < len(raw) {
				wb.SetRGBA(x, y, color.RGBA{R: raw[off], G: raw[off+1], B: raw[off+2], A: raw[off+3]})
			}
		}
	}
	savePNG(filepath.Join(wd, "dev", "desktop_probe", "cmp_wbui.png"), wb)

	// ── 2. Edge headless render (full JS) via local HTTP server ──
	// ES modules are blocked on file:// (CORS), so serve dist over HTTP.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{Handler: http.FileServer(http.Dir(distDir))}
	go srv.Serve(ln)
	defer srv.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	edge := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	edgeOut := filepath.Join(wd, "dev", "desktop_probe", "cmp_edge.png")
	url := fmt.Sprintf("http://127.0.0.1:%d/index.html", port)
	cmd := exec.Command(edge,
		"--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--force-device-scale-factor=1", "--window-size=1280,800",
		"--virtual-time-budget=8000", "--screenshot="+edgeOut,
		url)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("edge: %v %s", err, out)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(edgeOut); err == nil {
			break
		}
		if time.Now().After(deadline) {
			log.Fatal("edge screenshot timeout")
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2000 * time.Millisecond) // allow async JS to settle

	f, _ := os.Open(edgeOut)
	edgeImg, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatalf("decode edge png: %v", err)
	}
	eb := image.NewRGBA(edgeImg.Bounds())
	for y := 0; y < eb.Bounds().Dy(); y++ {
		for x := 0; x < eb.Bounds().Dx(); x++ {
			eb.SetRGBA(x, y, edgeImg.At(x, y).(color.RGBA))
		}
	}

	// ── 3. Pixel compare ──
	diff := 0
	var bbox image.Rectangle
	first := true
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if colorDiff(wb.RGBAAt(x, y), eb.RGBAAt(x, y)) > 32 {
				diff++
				if first {
					bbox = image.Rect(x, y, x+1, y+1)
					first = false
				} else {
					bbox = bbox.Union(image.Rect(x, y, x+1, y+1))
				}
			}
		}
	}
	rate := float64(diff) / float64(w*h) * 100
	fmt.Printf("Edge: %dx%d, wb-ui: %dx%d\n", eb.Bounds().Dx(), eb.Bounds().Dy(), w, h)
	fmt.Printf("DIFF pixels: %d (%.1f%%), bbox=(%d,%d)-(%d,%d)\n", diff, rate,
		bbox.Min.X, bbox.Min.Y, bbox.Max.X, bbox.Max.Y)

	// ── 4. Row/column diff histogram (locate problem regions) ──
	fmt.Println("=== diff density by 64px bands ===")
	for by := 0; by < h; by += 64 {
		cnt := 0
		for y := by; y < by+64 && y < h; y++ {
			for x := 0; x < w; x++ {
				if colorDiff(wb.RGBAAt(x, y), eb.RGBAAt(x, y)) > 32 {
					cnt++
				}
			}
		}
		if cnt > 0 {
			fmt.Printf("  y=%3d..%-3d: %d diff px\n", by, by+64, cnt)
		}
	}
	fmt.Println("=== diff density by 128px columns ===")
	for bx := 0; bx < w; bx += 128 {
		cnt := 0
		for x := bx; x < bx+128 && x < w; x++ {
			for y := 0; y < h; y++ {
				if colorDiff(wb.RGBAAt(x, y), eb.RGBAAt(x, y)) > 32 {
					cnt++
				}
			}
		}
		if cnt > 0 {
			fmt.Printf("  x=%3d..%-3d: %d diff px\n", bx, bx+128, cnt)
		}
	}
}

func colorDiff(a, b color.RGBA) int {
	d := 0
	d += absDiff(int(a.R), int(b.R))
	d += absDiff(int(a.G), int(b.G))
	d += absDiff(int(a.B), int(b.B))
	return d / 3
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func savePNG(path string, img *image.RGBA) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
