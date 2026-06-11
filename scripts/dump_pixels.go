//go:build ignore

// 像素分析工具 - 读取截图并输出关键区域的颜色值
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// 获取最新的截图文件
	matches, _ := filepath.Glob("screenshots/*.png")
	var targetFiles []string
	for _, m := range matches {
		if strings.Contains(m, "pixel_") {
			targetFiles = append(targetFiles, m)
		}
	}

	if len(targetFiles) == 0 {
		fmt.Println("No pixel screenshots found")
		os.Exit(1)
	}

	for _, path := range targetFiles {
		fmt.Printf("\n═════ %s ═════\n", filepath.Base(path))
		analyzeImage(path)
	}
}

func analyzeImage(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		bounds := img.Bounds()
		rgba = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}

	bounds := rgba.Bounds()
	fmt.Printf("  Size: %dx%d\n", bounds.Dx(), bounds.Dy())

	// 1. 扫描非白色像素的边界框
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	pixelCount := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rgba.RGBAAt(x, y)
			if c.R != 255 || c.G != 255 || c.B != 255 {
				pixelCount++
				if x < minX { minX = x }
				if y < minY { minY = y }
				if x > maxX { maxX = x }
				if y > maxY { maxY = y }
			}
		}
	}

	fmt.Printf("  Non-white pixels: %d\n", pixelCount)
	fmt.Printf("  Content bounding box: (%d,%d)-(%d,%d) [%dx%d]\n",
		minX, minY, maxX, maxY, maxX-minX+1, maxY-minY+1)

	// 2. 输出内容区域内的一些采样点
	fmt.Println("  Sample pixels in content area:")
	for y := minY; y <= maxY && y < minY+30; y += 5 {
		for x := minX; x <= maxX && x < minX+30; x += 5 {
			c := rgba.RGBAAt(x, y)
			if c.R != 255 || c.G != 255 || c.B != 255 {
				fmt.Printf("    (%d,%d) → RGBA(%d,%d,%d,%d)\n", x, y, c.R, c.G, c.B, c.A)
			}
		}
	}

	// 3. 检查抗锯齿 - 统计所有非完全透明非完全不透明的像素
	edgePixels := 0
	alphaGradPixels := 0
	alphaHist := make(map[uint8]int)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rgba.RGBAAt(x, y)
			if c.A > 0 && c.A < 255 {
				edgePixels++
			}
			if c.R != 255 || c.G != 255 || c.B != 255 || c.A != 255 {
				if c.A > 0 && c.A < 255 {
					alphaGradPixels++
					alphaHist[c.A]++
				}
				// 检查非白且非全透明的像素：任何带有alpha信息的像素
				if c.A > 0 && c.A < 255 && (c.R != 255 || c.G != 255 || c.B != 255) {
					if len(alphaHist) < 5 {
						alphaHist[c.A]++
					}
				}
			}
		}
	}

	fmt.Printf("\n  Antialiasing check:\n")
	fmt.Printf("    Pixels with 0<alpha<255: %d\n", edgePixels)
	fmt.Printf("    Non-white pixels with gradient alpha: %d\n", alphaGradPixels)

	if len(alphaHist) > 0 {
		fmt.Printf("    Alpha value distribution:\n")
		for a := uint8(1); a < 255; a++ {
			if alphaHist[a] > 0 {
				fmt.Printf("      alpha=%d: %d pixels\n", a, alphaHist[a])
			}
		}
	}

	// 4. 特别检查文字边缘 - 查找非黑色且alpha渐变的像素
	fmt.Printf("\n  Text edge analysis:\n")
	textEdgeCount := 0
	for y := minY; y <= maxY && y < minY+20; y++ {
		for x := minX; x <= maxX && x < minX+80; x++ {
			c := rgba.RGBAAt(x, y)
			// 文本边缘像素：颜色介于文字色和背景色之间（有alpha混合）
			isTextEdge := c.A > 0 && c.A < 255 && c.R != 255
			if isTextEdge && textEdgeCount < 15 {
				fmt.Printf("    Text edge at (%d,%d): RGBA(%d,%d,%d,%d)\n", x, y, c.R, c.G, c.B, c.A)
				textEdgeCount++
			}
		}
	}
	if textEdgeCount == 0 {
		// Try looking for any non-white pixel
		fmt.Println("    No text edge pixels found with gradient alpha. Checking any non-white pixel...")
		count := 0
		for y := minY; y <= maxY && count < 10; y++ {
			for x := minX; x <= maxX && count < 10; x++ {
				c := rgba.RGBAAt(x, y)
				if c.R != 255 || c.G != 255 || c.B != 255 {
					fmt.Printf("    Pixel at (%d,%d): RGBA(%d,%d,%d,%d)\n", x, y, c.R, c.G, c.B, c.A)
					count++
				}
			}
		}
	}
}
