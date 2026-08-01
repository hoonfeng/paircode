// Command crop_token crops the token-stats panel region from desktop.png.
package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	in := wd + "\\dev\\desktop_probe\\desktop.png"
	f, err := os.Open(in)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	// token panel: x=1030..1280, y=370..560
	sub := image.NewRGBA(image.Rect(0, 0, 250, 190))
	for y := 0; y < 190; y++ {
		for x := 0; x < 250; x++ {
			sub.Set(x, y, img.At(1030+x, 370+y))
		}
	}
	out := wd + "\\dev\\desktop_probe\\token_crop.png"
	fo, err := os.Create(out)
	if err != nil {
		log.Fatal(err)
	}
	png.Encode(fo, sub)
	fo.Close()
	fmt.Println("saved", out)
}
