// Command gpu_clip_probe2 验证 GPU 后端嵌套/变换下的 ClipRoundRect。
// 场景1: 纯 clip+fill（基线） 场景2: Scale(1.25) 后 clip 场景3: 嵌套相同 clip
// 场景4: Scale+嵌套+渐变fill 场景5: 非整数 translate + clip
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

func report(name string, pix []byte, w, h int, checks []struct {
	x, y int
	want bool // true=绿(圆内), false=背景(圆外)
}) {
	fmt.Printf("== %s ==\n", name)
	pass := 0
	for _, chk := range checks {
		r, g, b := pixelAt(pix, w, h, chk.x, chk.y)
		isGreen := g > 200 && r < 100
		ok := isGreen == chk.want
		if ok {
			pass++
		}
		fmt.Printf("  (%d,%d)=#%02X%02X%02X 期望:%s %s\n", chk.x, chk.y, r, g, b,
			map[bool]string{true: "圆内绿", false: "圆外背景"}[chk.want],
			map[bool]string{true: "✓", false: "✗"}[ok])
	}
	fmt.Printf("  => %d/%d\n", pass, len(checks))
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
	win, err := glfw.CreateWindow(64, 64, "probe", nil, nil)
	if err != nil {
		fmt.Println("create window:", err)
		os.Exit(1)
	}
	defer win.Destroy()
	win.MakeContextCurrent()
	glfw.SwapInterval(0)

	glIface, err := skia.NewGLInterface(func(name string) unsafe.Pointer {
		return unsafe.Pointer(glfw.GetProcAddress(name))
	})
	if err != nil {
		fmt.Println("gl interface:", err)
		os.Exit(1)
	}
	defer glIface.Release()
	gpuCtx, err := skia.NewGLContext(glIface)
	if err != nil {
		fmt.Println("gl context:", err)
		os.Exit(1)
	}
	defer gpuCtx.Release()

	// 每场景重建 surface 保证干净 stencil
	newSurf := func() (*skia.Surface, *graphics.Canvas) {
		surf, err := skia.NewGPUSurfaceFromFBO(
			gpuCtx, 0, 64, 64, 0, 8,
			skia.GLRGBA8, skia.ColorTypeRGBA8888, skia.SurfaceOriginBottomLeft,
		)
		if err != nil {
			fmt.Println("gpu surface:", err)
			os.Exit(1)
		}
		return surf, graphics.NewCanvasFromSurface(surf, 64, 64)
	}

	// 通用检查点：clip rect (10,10,40,40,r6)，弧心(16,16)
	// 圆外：角(10,10)；圆内：弧心(16,16)；(16,14)
	baseChecks := []struct {
		x, y int
		want bool
	}{
		{10, 10, false}, {11, 11, false},
		{16, 16, true}, {16, 14, true}, {30, 30, true},
	}

	// 场景1: 基线 clip+fill
	{
		surf, c := newSurf()
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.FillRect(0, 0, 64, 64, graphics.Color{G: 255, A: 255})
		report("1基线 clip+fill", c.Pixels(), 64, 64, baseChecks)
		surf.Release()
	}

	// 场景2: Scale(1.25) 后 clip
	{
		surf, c := newSurf()
		c.Scale(1.25, 1.25)
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.FillRect(0, 0, 64, 64, graphics.Color{G: 255, A: 255})
		report("2 Scale(1.25)+clip", c.Pixels(), 64, 64, baseChecks)
		surf.Release()
	}

	// 场景3: 嵌套相同 clip（layer+walk）
	{
		surf, c := newSurf()
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.Save()
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.FillRect(0, 0, 64, 64, graphics.Color{G: 255, A: 255})
		c.Restore()
		report("3 嵌套相同clip", c.Pixels(), 64, 64, baseChecks)
		surf.Release()
	}

	// 场景4: Scale(1.25)+嵌套+渐变 fill（ctx-bar 场景）
	{
		surf, c := newSurf()
		c.Scale(1.25, 1.25)
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.Save()
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.FillLinearGradient(0, 0, 64, 64, graphics.Color{G: 255, A: 255}, graphics.Color{G: 200, A: 255})
		c.Restore()
		report("4 Scale+嵌套+渐变", c.Pixels(), 64, 64, baseChecks)
		surf.Release()
	}

	// 场景5: 非整数 translate(0,-3.5) + clip（scroll 场景）
	{
		surf, c := newSurf()
		c.Scale(1.25, 1.25)
		c.Translate(0, -3.5)
		c.ClipRoundRect(10, 10, 40, 40, 6)
		c.FillRect(0, 0, 64, 64, graphics.Color{G: 255, A: 255})
		report("5 Scale+非整translate+clip", c.Pixels(), 64, 64, baseChecks)
		surf.Release()
	}
}
