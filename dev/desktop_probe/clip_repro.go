// Command clip_repro 最小复现：真实渲染参数下的 ClipRoundRect 行为
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"wb-ui/platform/graphics"
)

func dumpPixel(img *image.RGBA, w, h int, x, y int, label string) {
	off := (y*w + x) * 4
	fmt.Printf("[pix] %s (%d,%d) = #%02X%02X%02X\n", label, x, y,
		img.Pix[off], img.Pix[off+1], img.Pix[off+2])
}

func main() {
	// 场景 A：与单元测试相同（无 scale、小坐标）
	cA := graphics.NewCanvas(80, 80)
	cA.ClipRoundRect(10, 10, 40, 40, 6)
	cA.FillRect(0, 0, 80, 80, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgA := toRGBA(cA.Pixels(), 80, 80)
	dumpPixel(imgA, 80, 80, 10, 10, "A:clip左上角(10,10) 期望背景(透明/黑)")
	dumpPixel(imgA, 80, 80, 12, 10, "A:clip顶部圆角起点附近(12,10) 期望背景")
	dumpPixel(imgA, 80, 80, 16, 10, "A:clip顶部(16,10) 期望绿")
	dumpPixel(imgA, 80, 80, 11, 11, "A:clip圆角内侧(11,11) 期望过渡或绿")

	// 场景 B：真实参数（2560x1600 canvas + Scale(2,2) + 大坐标）
	cB := graphics.NewCanvas(2560, 1600)
	cB.Scale(2, 2)
	cB.ClipRoundRect(1039, 686.8, 233, 12, 6)
	cB.FillRect(0, 0, 1280, 800, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgB := toRGBA(cB.Pixels(), 2560, 1600)
	// 设备坐标 = 逻辑坐标 * 2。comp-bar 左上角逻辑 (1039,687) → 设备 (2078,1374)
	dumpPixel(imgB, 2560, 1600, 2078, 1374, "B:comp左上角(2078,1374) 期望背景")
	dumpPixel(imgB, 2560, 1600, 2082, 1374, "B:顶部圆角起点(2082,1374) 期望背景")
	dumpPixel(imgB, 2560, 1600, 2090, 1374, "B:顶部圆角内(2090,1374) 期望绿")
	dumpPixel(imgB, 2560, 1600, 2078, 1380, "B:左缘中部(2078,1380) 期望绿")
	dumpPixel(imgB, 2560, 1600, 2080, 1376, "B:圆角弧上(2080,1376) 期望过渡/绿")

	// 场景 C：无 Scale 大坐标（排除 Scale 因素）
	cC := graphics.NewCanvas(1280, 800)
	cC.ClipRoundRect(1039, 686.8, 233, 12, 6)
	cC.FillRect(0, 0, 1280, 800, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgC := toRGBA(cC.Pixels(), 1280, 800)
	dumpPixel(imgC, 1280, 800, 1039, 687, "C:comp左上角(1039,687) 期望背景")
	dumpPixel(imgC, 1280, 800, 1045, 687, "C:顶部圆角起点(1045,687) 期望背景")
	dumpPixel(imgC, 1280, 800, 1050, 687, "C:顶部圆角内(1050,687) 期望绿")
	dumpPixel(imgC, 1280, 800, 1039, 693, "C:左缘圆心高度(1039,693) 期望绿")
	// 场景 D：FillRoundRect 直接画圆角矩形（真实 comp-bar 参数）
	cD := graphics.NewCanvas(2560, 1600)
	cD.Scale(2, 2)
	cD.FillRoundRect(1039, 696, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	imgD := toRGBA(cD.Pixels(), 2560, 1600)
	fmt.Println("\n[场景D] FillRoundRect(1039,696,233,12,r=6) 期望圆角: 顶部行 y=696 仅 x=1045 附近红色")
	// 顶部行（设备 y=1392）
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgD, 2560, 1600, px, 1392, fmt.Sprintf("D:顶行y696 x%d", px/2))
	}
	for _, px := range []int{2078, 2082, 2086, 2090} {
		dumpPixel(imgD, 2560, 1600, px, 1394, fmt.Sprintf("D:y697 x%d", px/2))
	}
	dumpPixel(imgD, 2560, 1600, 2078, 1400, "D:y700 x1039(左缘中)")
	dumpPixel(imgD, 2560, 1600, 2082, 1400, "D:y700 x1041")
	dumpPixel(imgD, 2560, 1600, 2540, 1392, "D:y696 x1270(右上)")

	// 场景 E：StrokeRoundRect（border）
	cE := graphics.NewCanvas(2560, 1600)
	cE.Scale(2, 2)
	cE.StrokeRoundRect(1039, 696, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgE := toRGBA(cE.Pixels(), 2560, 1600)
	fmt.Println("\n[场景E] StrokeRoundRect(1039,696,233,12,r=6,w=1) 期望圆角边框")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgE, 2560, 1600, px, 1392, fmt.Sprintf("E:顶行y696 x%d", px/2))
	}
	dumpPixel(imgE, 2560, 1600, 2078, 1400, "E:y700 x1039(左缘中)")
	// 场景 F：1x（无 Scale）FillRoundRect + StrokeRoundRect
	cF := graphics.NewCanvas(1280, 800)
	cF.FillRoundRect(1039, 696, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	cF.StrokeRoundRect(1039, 696, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgF := toRGBA(cF.Pixels(), 1280, 800)
	fmt.Println("\n[场景F] 1x FillRoundRect+StrokeRoundRect r=6")
	for _, px := range []int{1039, 1041, 1043, 1045, 1047, 1049} {
		dumpPixel(imgF, 1280, 800, px, 696, fmt.Sprintf("F:顶行y696 x%d", px))
	}
	for _, px := range []int{1039, 1041, 1043, 1045} {
		dumpPixel(imgF, 1280, 800, px, 697, fmt.Sprintf("F:y697 x%d", px))
	}
	dumpPixel(imgF, 1280, 800, 1039, 700, "F:y700 x1039")
	dumpPixel(imgF, 1280, 800, 1271, 696, "F:y696 x1271(右上)")
	dumpPixel(imgF, 1280, 800, 1271, 700, "F:y700 x1271(右缘)")
	// 场景 G：Clip(矩形祖先) + ClipRoundRect 相交（真实渲染的核心差异）
	cG := graphics.NewCanvas(1280, 800)
	cG.Clip(graphics.Rect{X: 48, Y: 30, Width: 1232, Height: 770}) // sidebar 区域矩形
	cG.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cG.FillRect(1039, 695.8, 233, 12, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	cG.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgG := toRGBA(cG.Pixels(), 1280, 800)
	fmt.Println("\n[场景G] 矩形clip ∩ 圆角clip 相交后 FillRect+StrokeRoundRect")
	for _, px := range []int{1039, 1041, 1043, 1045, 1047} {
		dumpPixel(imgG, 1280, 800, px, 696, fmt.Sprintf("G:顶行y696 x%d", px))
	}
	for _, px := range []int{1039, 1041, 1043, 1045} {
		dumpPixel(imgG, 1280, 800, px, 697, fmt.Sprintf("G:y697 x%d", px))
	}
	dumpPixel(imgG, 1280, 800, 1045, 700, "G:y700 x1045")
	dumpPixel(imgG, 1280, 800, 1271, 700, "G:y700 x1271(右缘)")

	// 场景 H：2x CTM 下 FillRoundRect + StrokeRoundRect + ClipRoundRect
	cH := graphics.NewCanvas(2560, 1600)
	cH.Scale(2, 2)
	cH.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cH.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	cH.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgH := toRGBA(cH.Pixels(), 2560, 1600)
	fmt.Println("\n[场景H] 2x CTM: ClipRoundRect+FillRoundRect+StrokeRoundRect r=6")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgH, 2560, 1600, px, 1392, fmt.Sprintf("H:顶行y1392 x%d", px))
	}
	for _, px := range []int{2082, 2086, 2090} {
		dumpPixel(imgH, 2560, 1600, px, 1394, fmt.Sprintf("H:y1394 x%d", px))
	}
	dumpPixel(imgH, 2560, 1600, 2542, 1400, "H:y1400 x2542(右缘)")
	dumpPixel(imgH, 2560, 1600, 2090, 1400, "H:y1400 x2090")

	// 场景 I：先大量绘制（文本/渐变/圆角）模拟真实渲染历史，再画 comp-bar
	cI := graphics.NewCanvas(2560, 1600)
	cI.Scale(2, 2)
	// 模拟真实渲染的几百次操作
	for i := 0; i < 300; i++ {
		xx := float64((i * 7) % 1200)
		yy := float64((i * 13) % 700)
		cI.FillRect(xx, yy, 30, 14, graphics.Color{R: uint8(i % 255), G: 40, B: 80, A: 255})
		if i%3 == 0 {
			cI.FillLinearGradient(xx+100, yy, 40, 10, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})
		}
		if i%5 == 0 {
			cI.Save()
			cI.ClipRoundRect(xx, yy, 40, 20, 6)
			cI.FillRect(xx, yy, 40, 20, graphics.Color{R: 200, G: 200, B: 200, A: 255})
			cI.Restore()
		}
	}
	// 真实渲染上下文：sidebar 矩形 clip + comp-bar 圆角 clip
	cI.Clip(graphics.Rect{X: 48, Y: 30, Width: 278, Height: 748})
	cI.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cI.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	cI.StrokeRoundRect(1039, 695.8, 233, 12, 6, 1, graphics.Color{R: 0, G: 255, B: 0, A: 255})
	imgI := toRGBA(cI.Pixels(), 2560, 1600)
	fmt.Println("\n[场景I] 300次历史绘制后 comp-bar 圆角")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgI, 2560, 1600, px, 1392, fmt.Sprintf("I:顶行y1392 x%d", px))
	}
	dumpPixel(imgI, 2560, 1600, 2090, 1394, "I:y1394 x2090")
	dumpPixel(imgI, 2560, 1600, 2542, 1400, "I:y1400 x2542(右缘)")
	dumpPixel(imgI, 2560, 1600, 2090, 1400, "I:y1400 x2090")

	// 场景 J：渐变绘制后立即 FillRoundRect（验证 shader 释放是否污染）
	cJ := graphics.NewCanvas(2560, 1600)
	cJ.Scale(2, 2)
	cJ.FillLinearGradient(48, 30, 278, 748, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})
	cJ.FillLinearGradient(429, 67, 601, 711, graphics.Color{R: 100, G: 120, B: 140, A: 255}, graphics.Color{R: 200, G: 210, B: 220, A: 255})
	cJ.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cJ.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	imgJ := toRGBA(cJ.Pixels(), 2560, 1600)
	fmt.Println("\n[场景J] 两个渐变后 FillRoundRect 圆角")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgJ, 2560, 1600, px, 1392, fmt.Sprintf("J:顶行y1392 x%d", px))
	}
	dumpPixel(imgJ, 2560, 1600, 2542, 1400, "J:y1400 x2542(右缘)")
	dumpPixel(imgJ, 2560, 1600, 2090, 1400, "J:y1400 x2090")

	// 场景 K：FillRoundRect 后紧接 FillLinearGradient，再 FillRoundRect
	cK := graphics.NewCanvas(2560, 1600)
	cK.Scale(2, 2)
	cK.FillRect(0, 0, 100, 100, graphics.Color{R: 1, G: 2, B: 3, A: 255})
	cK.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cK.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	cK.Restore()
	cK.FillLinearGradient(200, 200, 100, 100, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})
	imgK := toRGBA(cK.Pixels(), 2560, 1600)
	fmt.Println("\n[场景K] FillRoundRect 后渐变再 FillRoundRect")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgK, 2560, 1600, px, 1392, fmt.Sprintf("K:顶行y1392 x%d", px))
	}
	dumpPixel(imgK, 2560, 1600, 2542, 1400, "K:y1400 x2542(右缘)")
	dumpPixel(imgK, 2560, 1600, 2090, 1400, "K:y1400 x2090")

	// 场景 L：重建真实祖先矩形 clip 链 + 圆角 clip + FillRoundRect
	cL := graphics.NewCanvas(2560, 1600)
	cL.Scale(2, 2)
	// 祖先矩形 clip 链（真实日志：conv-sidebar → cache-ring-wrap）
	cL.Clip(graphics.Rect{X: 1030, Y: 67, Width: 250, Height: 711}) // conv-sidebar
	cL.Clip(graphics.Rect{X: 1031, Y: 67, Width: 249, Height: 711}) // cache-ring-wrap
	cL.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cL.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	imgL := toRGBA(cL.Pixels(), 2560, 1600)
	fmt.Println("\n[场景L] 真实祖先 clip 链(conv-sidebar+cache-ring-wrap) + 圆角 clip + FillRoundRect")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgL, 2560, 1600, px, 1392, fmt.Sprintf("L:顶行y1392 x%d", px))
	}
	dumpPixel(imgL, 2560, 1600, 2542, 1400, "L:y1400 x2542(右缘)")
	dumpPixel(imgL, 2560, 1600, 2090, 1400, "L:y1400 x2090")

	// 场景 M：模拟 saveCount=11 深度 + 大量 translate 历史
	cM := graphics.NewCanvas(2560, 1600)
	cM.Scale(2, 2)
	// 大量 transform 历史（模拟滚动/平移）
	for i := 0; i < 40; i++ {
		cM.Save()
		cM.Translate(5, 3)
		cM.Clip(graphics.Rect{X: 429 + float64(i), Y: 67, Width: 601, Height: 711})
		cM.Restore()
	}
	// 9 层矩形 clip
	for i := 0; i < 9; i++ {
		cM.Save()
		cM.Clip(graphics.Rect{X: 1030, Y: 67, Width: 250, Height: 711})
	}
	cM.Save() // saveCount=11
	cM.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cM.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	imgM := toRGBA(cM.Pixels(), 2560, 1600)
	fmt.Println("\n[场景M] save深度11 + translate历史 + 圆角 clip")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgM, 2560, 1600, px, 1392, fmt.Sprintf("M:顶行y1392 x%d", px))
	}
	dumpPixel(imgM, 2560, 1600, 2542, 1400, "M:y1400 x2542(右缘)")
	dumpPixel(imgM, 2560, 1600, 2090, 1400, "M:y1400 x2090")

	// 场景 N：大量 DrawText + DrawRect 后 FillRoundRect
	cN := graphics.NewCanvas(2560, 1600)
	cN.Scale(2, 2)
	f := graphics.Font{Family: "Segoe UI", Size: 12, Weight: 400, Style: "normal"}
	for i := 0; i < 200; i++ {
		cN.DrawText(float64((i*37)%1200), float64((i*11)%700), "Test text "+fmt.Sprint(i), f, graphics.Color{R: 200, G: 200, B: 200, A: 255})
		cN.FillRect(float64((i*5)%1000), float64((i*7)%600), 50, 10, graphics.Color{R: 100, G: 100, B: 120, A: 255})
	}
	cN.ClipRoundRect(1039, 695.8, 233, 12, 6)
	cN.FillRoundRect(1039, 695.8, 233, 12, 6, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	imgN := toRGBA(cN.Pixels(), 2560, 1600)
	fmt.Println("\n[场景N] 200次DrawText+DrawRect后 FillRoundRect")
	for _, px := range []int{2078, 2082, 2086, 2090, 2094} {
		dumpPixel(imgN, 2560, 1600, px, 1392, fmt.Sprintf("N:顶行y1392 x%d", px))
	}
	dumpPixel(imgN, 2560, 1600, 2542, 1400, "N:y1400 x2542(右缘)")
	dumpPixel(imgN, 2560, 1600, 2090, 1400, "N:y1400 x2090")
}

func toRGBA(pix []byte, w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, pix)
	_ = png.Encode
	return img
}

var _ = os.Stdout
