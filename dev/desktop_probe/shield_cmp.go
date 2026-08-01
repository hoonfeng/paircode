// Command shield_cmp crops the review-button shield icon from cmp_edge and
// cmp_wbui and compares ink (shield must draw a green outline, pts=37 path).
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
)

func shield(name, out string, x0, y0 int) {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	// shield icon area: first obtn (审核) icon, 16x16 at its left
	sub := image.NewRGBA(image.Rect(0, 0, 20, 20))
	green := 0
	light := 0
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			c := img.At(x0+x, y0+y)
			r, g, b, _ := c.RGBA()
			if g > 40000 && g > r+8000 && g > b+8000 {
				green++
			}
			if r > 40000 && g > 40000 && b > 40000 {
				light++
			}
			sub.Set(x, y, c)
		}
	}
	fo, _ := os.Create(out)
	png.Encode(fo, sub)
	fo.Close()
	fmt.Printf("%s: green-ink=%d light-ink=%d\n", out, green, light)
}

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	// Edge: first obtn at ~(451,736), icon at left padding 8px → (459,740)
	shield(wd+"\\dev\\desktop_probe\\cmp_edge.png", wd+"\\dev\\desktop_probe\\shield_edge.png", 456, 738)
	// wb-ui: first obtn at (450,736), icon at left padding 8px → (458,740)
	shield(wd+"\\dev\\desktop_probe\\cmp_wbui.png", wd+"\\dev\\desktop_probe\\shield_wbui.png", 455, 738)
}
