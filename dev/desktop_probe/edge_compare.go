// Command edge_compare renders the companion frontend through wb-ui and Edge,
// then prints band diffs for fixed chrome regions (titlebar, activity bar,
// rp-header, status bar) which do not depend on API data.
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

	"wb-ui/rendering"
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

	// wb-ui render
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	raw, _ := wv.Render()
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
	savePNG(wd+"\\dev\\desktop_probe\\cmp_wbui.png", wb)

	// Edge render
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	srv := &http.Server{Handler: http.FileServer(http.Dir(distDir))}
	go srv.Serve(ln)
	defer srv.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	edgeOut := wd + "\\dev\\desktop_probe\\cmp_edge.png"
	url := fmt.Sprintf("http://127.0.0.1:%d/index.html", port)
	cmd := exec.Command(edge, "--headless", "--disable-gpu", "--no-sandbox", "--hide-scrollbars",
		"--force-device-scale-factor=1", "--window-size=1280,800",
		"--virtual-time-budget=8000", "--screenshot="+edgeOut, url)
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
	edgeImg, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	regions := []struct {
		name string
		x0   int
		y0   int
		x1   int
		y1   int
	}{
		{"titlebar y0-30", 0, 0, 1280, 30},
		{"activity-bar x0-48 y30-800", 0, 30, 48, 800},
		{"rp-header y30-67", 429, 30, 1280, 67},
		{"status-bar y778-800", 0, 778, 1280, 800},
		{"chat-messages y67-542", 429, 67, 1030, 542},
		{"chat-input y582-778", 429, 582, 1030, 778},
		{"conv-sidebar y67-778", 1030, 67, 1280, 778},
		{"sidebar y62-778", 48, 62, 327, 778},
	}
	for _, r := range regions {
		total := 0
		cols := make([]int, r.x1-r.x0)
		for x := r.x0; x < r.x1; x++ {
			for y := r.y0; y < r.y1; y++ {
				cr, cg, cb, _ := edgeImg.At(x, y).RGBA()
				wr2, wg, wb2, _ := wb.At(x, y).RGBA()
				if cr != wr2 || cg != wg || cb != wb2 {
					total++
					cols[x-r.x0]++
				}
			}
		}
		area := (r.x1 - r.x0) * (r.y1 - r.y0)
		// top 3 column peaks
		top3 := [3]int{-1, -1, -1}
		for _, v := range cols {
			if v > top3[0] {
				top3[2] = top3[1]
				top3[1] = top3[0]
				top3[0] = v
			} else if v > top3[1] {
				top3[2] = top3[1]
				top3[1] = v
			} else if v > top3[2] {
				top3[2] = v
			}
		}
		fmt.Printf("%-28s diff=%6d (%.1f%%) peakCols=%v\n", r.name, total, float64(total)*100/float64(area), top3)
	}
}

type rgba struct{ R, G, B, A uint8 }

func colorRGBA(r, g, b, a uint8) rgba { return rgba{r, g, b, a} }

func savePNG(path string, img *image.RGBA) {
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}

var _ = rendering.BoxGeometry
