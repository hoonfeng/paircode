// Command gpu_bar_scan 裁剪 dump_gpu.png 的 ctx-bar 区域并水平扫描：
// 输出每像素 RGB，验证左端/右端圆弧（圆角 clip 是否生效）。
package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: gpu_bar_scan <png> <y_start> <y_end>")
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	ys, ye := 0, b.Max.Y
	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[2], "%d", &ys)
		fmt.Sscanf(os.Args[3], "%d", &ye)
	}
	// 取中间行（y = (ys+ye)/2）的水平扫描
	yMid := (ys + ye) / 2
	fmt.Printf("size=%dx%d scan row y=%d (y range %d-%d)\n", b.Max.X, b.Max.Y, yMid, ys, ye)
	// 分段输出：x 每 4px 一个采样 + 颜色变化点
	lastStart := 0
	prev := ""
	for x := 0; x < b.Max.X; x++ {
		r, g, bl, _ := img.At(x, yMid).RGBA()
		rr, gg, bb := byte(r>>8), byte(g>>8), byte(bl>>8)
		cur := fmt.Sprintf("#%02X%02X%02X", rr, gg, bb)
		if cur != prev {
			if prev != "" && (x > 0) {
				fmt.Printf("  x=%d..%d %s\n", lastStart, x-1, prev)
			}
			lastStart = x
			prev = cur
		}
	}
	if prev != "" {
		fmt.Printf("  x=%d..%d %s\n", lastStart, b.Max.X-1, prev)
	}
}
