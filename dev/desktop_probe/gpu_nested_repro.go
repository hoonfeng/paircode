// gpu_nested_repro：模拟真实渲染的 Save 嵌套结构验证 GPU 圆角 clip
// 真实序列（日志）：
//   [layer] 进入 → Save → ClipRoundRect(层入口, r=6)
//   → FillRoundRect 背景 → StrokeRoundRect 边框
//   → Save → ClipRoundRect(walk, r=6, 同矩形) → 6×seg FillRect → Restore
//   → Restore
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
	c.Scale(2, 2)

	// 背景
	c.FillRect(0, 0, 1280, 800, graphics.Color{R: 33, G: 38, B: 45, A: 255})

	// ===== 完全模拟真实 comp-bar 序列 =====
	// 层入口 Save + clip
	c.Save()
	fmt.Println("after Save1:", c.SaveCount())
	c.ClipRoundRect(1039, 686.8, 233, 12, 6)
	fmt.Println("after entry clip:", c.SaveCount())
	// 背景（圆角）
	c.FillRoundRect(1039, 686.8, 233, 12, 6, graphics.Color{R: 22, G: 27, B: 34, A: 255})
	// 边框
	c.StrokeRoundRect(1039, 686.8, 233, 12, 6, 1, graphics.Color{R: 48, G: 54, B: 61, A: 255})
	// walk 层 Save + clip（同矩形）
	c.Save()
	fmt.Println("after Save2:", c.SaveCount())
	c.ClipRoundRect(1039, 686.8, 233, 12, 6)
	fmt.Println("after walk clip:", c.SaveCount())
	// 6 个 seg FillRect
	colors := []graphics.Color{
		{R: 88, G: 166, B: 255, A: 255},
		{R: 63, G: 185, B: 80, A: 255},
		{R: 163, G: 113, B: 247, A: 255},
		{R: 210, G: 153, B: 34, A: 255},
		{R: 247, G: 120, B: 186, A: 255},
		{R: 139, G: 148, B: 158, A: 255},
	}
	widths := []float64{18, 12, 8, 25, 22, 15}
	x := 1040.0
	for i := 0; i < 6; i++ {
		w := widths[i] / 100 * 231
		c.FillRect(x, 687.8, w, 10, colors[i])
		x += w
	}
	c.Restore()
	c.Restore()

	pix := c.Pixels()
	if pix == nil {
		fmt.Println("read pixels failed")
		return
	}
	fmt.Println("=== GPU 嵌套 Save+clip 模拟 ===")
	checks := []struct {
		x, y int
		desc string
		want string
	}{
		{1037, 690, "左圆角外(应背景#21262D)", "bg"},
		{1038, 693, "左圆角外2(应背景)", "bg"},
		{1040, 689, "弧内(应蓝或AA)", "blue"},
		{1045, 691, "弧心(应蓝)", "blue"},
		{1090, 692, "seg 中部(应蓝)", "blue"},
		{1130, 692, "第二段(应绿)", "green"},
		{1160, 692, "第三段(应紫)", "purple"},
		{1271, 692, "右圆弧内(应灰)", "gray"},
		{1273, 690, "右圆角外(应背景)", "bg"},
	}
	pass := 0
	for _, chk := range checks {
		r, g, b := pixelAt(pix, 2560, 1600, chk.x*2, chk.y*2)
		ok := false
		switch chk.want {
		case "bg":
			ok = r == 33 && g == 38 && b == 45
		case "blue":
			ok = b > 200 && r < 130
		case "green":
			ok = g > 150 && r < 120 && b < 120
		case "purple":
			ok = b > 180 && r > 100 && r < 200
		case "gray":
			ok = r > 110 && r < 170 && g > 110 && g < 170 && b > 120 && b < 180
		}
		if ok {
			pass++
		}
		fmt.Printf("  (%d,%d) = #%02X%02X%02X %s %v\n", chk.x, chk.y, r, g, b, chk.desc, ok)
	}
	fmt.Printf("  => %d/%d 通过\n", pass, len(checks))
}
