// Command pxcheck reads pixels from desktop.png (fresh render).
package main

import (
	"fmt"
	"image/png"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	wd, _ := os.Getwd()
	f, err := os.Open(wd + "\\dev\\desktop_probe\\desktop.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	pts := [][2]int{{10, 60}, {10, 200}, {16, 45}, {30, 50}, {5, 40}, {40, 70}}
	for _, p := range pts {
		r, g, b, _ := img.At(p[0], p[1]).RGBA()
		fmt.Printf("(%3d,%3d) #%02x%02x%02x\n", p[0], p[1], r>>8, g>>8, b>>8)
	}
}
