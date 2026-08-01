// Command crop_btns crops the send-btn / ibb-btns region from cmp_edge.png
// and cmp_wbui.png for pixel comparison.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
)

func crop(name, out string) {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	// send-btn region: x 940..1030, y 720..780
	sub := image.NewRGBA(image.Rect(0, 0, 90, 60))
	ink := 0
	for y := 0; y < 60; y++ {
		for x := 0; x < 90; x++ {
			c := img.At(940+x, 720+y)
			r, g, b, _ := c.RGBA()
			if r < 30000 && g < 30000 && b < 30000 {
				ink++
			}
			sub.Set(x, y, c)
		}
	}
	fo, _ := os.Create(out)
	png.Encode(fo, sub)
	fo.Close()
	fmt.Printf("%s: dark ink=%d\n", out, ink)
	// obtn strip region: x 445..660, y 732..764
	sub2 := image.NewRGBA(image.Rect(0, 0, 215, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 215; x++ {
			sub2.Set(x, y, img.At(445+x, 732+y))
		}
	}
	fo2, _ := os.Create(strings.Replace(out, ".png", "_obtn.png", 1))
	png.Encode(fo2, sub2)
	fo2.Close()
	fmt.Printf("%s: obtn strip saved\n", strings.Replace(out, ".png", "_obtn.png", 1))
}

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	crop(wd+"\\dev\\desktop_probe\\cmp_edge.png", wd+"\\dev\\desktop_probe\\btn_edge_crop.png")
	crop(wd+"\\dev\\desktop_probe\\cmp_wbui.png", wd+"\\dev\\desktop_probe\\btn_wbui_crop.png")
}
