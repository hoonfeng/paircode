// Command svg_ring inspects the conv-sidebar cache-ring SVG and compares its
// ink coverage between Edge and wb-ui.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"wb-ui/dom"
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

	// 1. Inspect cache-ring SVG content in wb-ui.
	distDir := filepath.Join(wd, "cmd", "desktop", "web-ui", "dist")
	htmlData, _ := os.ReadFile(distDir + "/index.html")
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	wv.Resize(1280, 800)
	_ = wv.LoadHTML(string(htmlData))
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()

	fmt.Println("=== cache-ring SVG content (wb-ui) ===")
	var walk func(o rendering.RenderObject)
	walk = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok && el.LocalName() == "svg" {
			cls := el.GetAttribute("class")
			if strings.Contains(cls, "cache-ring") {
				fmt.Printf("  svg.%s viewBox=%q\n", cls, el.GetAttribute("viewBox"))
				var dump func(n dom.Node, ind string)
				dump = func(n dom.Node, ind string) {
					if e, ok := n.(*dom.Element); ok {
						fmt.Printf("    %s<%s> d=%q r=%q stroke=%q\n", ind, e.LocalName(),
							e.GetAttribute("d"), e.GetAttribute("r"), e.GetAttribute("stroke"))
					}
					for c := n.FirstChild(); c != nil; c = c.NextSibling() {
						dump(c, ind+"  ")
					}
				}
				dump(el, "")
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(rv)

	// 2. Compare ink in the ring area (x=1039..1135, y=378..474).
	edge := load(wd + "\\dev\\desktop_probe\\cmp_edge.png")
	wb := load(wd + "\\dev\\desktop_probe\\cmp_wbui.png")
	eInk, wInk := 0, 0
	for y := 378; y < 474; y++ {
		for x := 1039; x < 1135; x++ {
			er, eg, eb := px(edge, x, y)
			wr, wg, wb2 := px(wb, x, y)
			if bgDiff(er, eg, eb) > 30 {
				eInk++
			}
			if bgDiff(wr, wg, wb2) > 30 {
				wInk++
			}
		}
	}
	fmt.Printf("\n=== cache-ring ink: edge=%d wb-ui=%d ===\n", eInk, wInk)
}

func bgDiff(r, g, b int) int { return abs(r-0x21) + abs(g-0x26) + abs(b-0x2d) }

func load(path string) image.Image {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

func px(img image.Image, x, y int) (int, int, int) {
	r, g, b, _ := img.At(x, y).RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
