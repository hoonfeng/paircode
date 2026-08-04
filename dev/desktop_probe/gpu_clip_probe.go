// Command gpu_clip_probe 验证 GPU 后端下 ClipRoundRect 是否生效：
// stencil=0（旧）vs stencil=8（修复）。在 GLFW 窗口的 GPU surface 上
// 画 ClipRoundRect(10,10,40,40,r6)+全屏绿，读像素验证圆角外应被裁剪。
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

func runCase(stencil int, name string) {
	fmt.Printf("===== case %s (stencil=%d) =====\n", name, stencil)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCompatProfile)
	glfw.WindowHint(glfw.Visible, 0)
	glfw.WindowHint(glfw.StencilBits, stencil)
	win, err := glfw.CreateWindow(64, 64, "probe", nil, nil)
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
		gpuCtx, 0, 64, 64, 0, stencil,
		skia.GLRGBA8, skia.ColorTypeRGBA8888, skia.SurfaceOriginBottomLeft,
	)
	if err != nil {
		fmt.Println("gpu surface:", err)
		return
	}
	defer surf.Release()

	c := graphics.NewCanvasFromSurface(surf, 64, 64)
	c.ClipRoundRect(10, 10, 40, 40, 6)
	c.FillRect(0, 0, 64, 64, graphics.Color{R: 0, G: 255, B: 0, A: 255})

	pix := c.Pixels()
	if pix == nil {
		fmt.Println("read pixels failed")
		return
	}
	// 弧心 (16,16) r=6。圆外点期望背景(透明→黑)，圆内点期望绿。
	checks := []struct {
		x, y int
		want string
	}{
		{10, 10, "背景(圆外)"}, // dx=-6 dy=-6 圆外
		{16, 10, "背景(弧顶上方)"}, // dy=-6 圆外
		{10, 16, "背景(左弧顶)"}, // dx=-6 圆外
		{12, 12, "绿(圆内)"},  // dx=-4 dy=-4 圆内
		{16, 16, "绿(弧心)"},
		{16, 14, "绿(弧心上方)"}, // dy=-2 圆内
		{14, 16, "绿(左缘内)"}, // dx=-2 圆内
		{30, 30, "绿(内部)"},
	}
	pass := 0
	for _, chk := range checks {
		r, g, b := pixelAt(pix, 64, 64, chk.x, chk.y)
		ok := false
		if chk.want == "绿(圆内)" || chk.want == "绿(弧心)" || chk.want == "绿(弧心上方)" || chk.want == "绿(左缘内)" || chk.want == "绿(内部)" {
			ok = g > 200 && r < 100
		} else {
			ok = g < 100 // 背景（透明清理后接近黑）
		}
		if ok {
			pass++
		}
		fmt.Printf("  (%d,%d) = #%02X%02X%02X 期望:%s %s\n", chk.x, chk.y, r, g, b, chk.want, map[bool]string{true: "✓", false: "✗"}[ok])
	}
	fmt.Printf("  => %d/%d 通过\n", pass, len(checks))
}

func main() {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		fmt.Println("glfw init:", err)
		os.Exit(1)
	}
	defer glfw.Terminate()

	runCase(0, "旧(stencil=0)")
	runCase(8, "修复(stencil=8)")
}
