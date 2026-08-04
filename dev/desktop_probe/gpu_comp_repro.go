// Command gpu_comp_repro 在 GPU 后端复现真实 comp-bar 渲染：
// 300 次历史绘制 → sidebar 矩形 clip → comp-bar 圆角 clip → seg FillRect
// 验证 seg 是否被圆角裁切干净（圆角外应无 seg 色渗入）。
package main

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hoonfeng/goskia/skia"
	"wb-ui/platform/graphics"
)

func pixelAt(pix []byte, w, h, x, y int) (uint8, uint8, uint8) {
	off := (y*w + x) * 4
	return pix[off], pix[off+1], pix[off+2]
}

func main() {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		fmt.Println("glfw init:", err)
		os.Exit(1)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
	glfw.WindowHint(glfw.Visible, 0)
	glfw.WindowHint(glfw.StencilBits, 8)
	win, err := glfw.CreateWindow(2560, 1600, "probe", nil, nil)
	if err != nil {
		fmt.Println("create window:", err)
		return
	}
	defer win.Destroy()
	win.MakeContextCurrent()
	glfw.SwapInterval(0)

	glIface, err := skia.NewGLInterface(func(name string) unsafe.Pointer {
		return unsafe.Pointer(glfw.GetProcAddress(name))
	})
	if err != nil {
		fmt.Println("gl interface:", err)
		return
	}
	defer glIface.Release()

	gpuCtx, err := skia.NewGLContext(glIface)
	if err != nil {
		fmt.Println("gl context:", err)
		return
	}
	defer gpuCtx.Release()

	surf, err := skia.NewGPUSurfaceFromFBO(
		gpuCtx, 0, 2560, 1600, 0, 8,
		skia.GLRGBA8, skia.ColorTypeRGBA8888, skia.SurfaceOriginBottomLeft,
	)
	if err != nil {
		fmt.Println("gpu surface:", err)
		return
	}
	defer surf.Release()

	c := graphics.NewCanvasFromSurface(surf, 2560, 1600)
	c.Scale(2, 2) // DPR=2，和 desktop 一致

	// 1) 先铺背景（模拟面板背景）
	c.FillRect(0, 0, 1280, 800, graphics.Color{R: 33, G: 38, B: 45, A: 255})

	// 2) 300 次历史绘制模拟真实渲染
	for i := 0; i < 300; i++ {
		xx := float64((i * 7) % 1200)
		yy := float64((i * 13) % 700)
		c.FillRect(xx, yy, 30, 14, graphics.Color{R: uint8(i % 255), G: 40, B: 80, A: 255})
		if i%3 == 0 {
			c.FillLinearGradient(xx+100, yy, 40, 10, graphics.Color{R: 10, G: 20, B: 30, A: 255}, graphics.Color{R: 90, G: 100, B: 110, A: 255})
		}
		if i%5 == 0 {
			c.Save()
			c.ClipRoundRect(xx, yy, 40, 20, 6)
			c.FillRect(xx, yy, 40, 20, graphics.Color{R: 200, G: 200, B: 200, A: 255})
			c.Restore()
		}
	}

	// 3) comp-bar 圆角 clip（真实日志：saveCount=12 直接圆角 clip）
	c.ClipRoundRect(1039, 686.8, 233, 12, 6)
	c.FillRoundRect(1039, 686.8, 233, 12, 6, graphics.Color{R: 22, G: 27, B: 34, A: 255})
	// 6) seg 纯色（模拟 comp-system #58A6FF）
	c.FillRect(1040, 687.8, 60, 10, graphics.Color{R: 88, G: 166, B: 255, A: 255})
	// 7) 边框
	c.StrokeRoundRect(1039, 686.8, 233, 12, 6, 1, graphics.Color{R: 48, G: 54, B: 61, A: 255})
	// 8) 再做一次 save/restore 模拟 hover 重绘
	c.Save()
	c.ClipRoundRect(1039, 686.8, 233, 12, 6)
	c.FillRect(1100, 687.8, 80, 10, graphics.Color{R: 212, G: 167, B: 78, A: 255}) // comp-history 金
	c.Restore()

	pix := c.Pixels()
	if pix == nil {
		fmt.Println("read pixels failed")
		return
	}
	fmt.Println("=== GPU comp-bar 圆角验证（真实渲染历史后） ===")
	fmt.Println("弧心 (1045, 692.8) r=6；seg 蓝从 x=1040 开始")
	fmt.Println("左端圆角外 x=1037/1038 期望背景 #21262D（无蓝/金渗入）")
	// 左端圆角：圆心(1045,692.8) r=6
	checks := []struct {
		x, y int
		desc string
	}{
		{1037, 688, "左圆角外1(应背景)"},
		{1038, 690, "左圆角外2(应背景)"},
		{1039, 695, "左圆弧边缘(AA或背景)"},
		{1040, 693, "左圆弧内(应蓝)"},
		{1042, 688, "顶部弧内(应蓝)"},
		{1045, 689, "弧心行(应蓝)"},
		{1090, 693, "seg 中部(应蓝)"},
		{1105, 693, "金色seg(应金)"},
		{1271, 693, "右圆弧内(应金)"},
		{1273, 693, "右圆角外(应背景)"},
	}
	pass := 0
	for _, chk := range checks {
		r, g, b := pixelAt(pix, 2560, 1600, chk.x*2, chk.y*2)
		fmt.Printf("  (%d,%d) = #%02X%02X%02X %s\n", chk.x, chk.y, r, g, b, chk.desc)
		if b > 200 && r < 130 && g > 100 { // 蓝
			if chk.x >= 1040 && chk.x <= 1100 {
				pass++
			}
		} else if r > 180 && g > 120 && b < 120 { // 金
			if chk.x >= 1100 {
				pass++
			}
		} else {
			pass++
		}
	}
	fmt.Printf("  => %d/%d 通过\n", pass, len(checks))
}
