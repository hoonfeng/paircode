// Command gpu_narrow_probe 复现用户报告（QQ20260804-193558）：
// GPU 后端下，窄内容（3px 渐变 fill）在圆角容器左端时，
// 圆弧外像素（dist≈7.8 对角点）是否出现内容色渗入（盖住圆角）。
// raster 同几何测试 PASS（无渗入）→ 验证 GPU 是否不同。
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
	if os.Getenv("WB_MSAA") == "4" {
		glfw.WindowHint(glfw.Samples, 4)
	}
	win, err := glfw.CreateWindow(120, 40, "probe", nil, nil)
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
		return
	}
	defer glIface.Release()

	gpuCtx, err := skia.NewGLContext(glIface)
	if err != nil {
		fmt.Println("gl context:", err)
		return
	}
	defer gpuCtx.Release()

	// 模拟用户场景：容器 (10,10)-(110,22) 12px 高 r=6 overflow:hidden；
	// 内容 3px 宽红色 (10,10)-(13,22) 贴左端（全部在圆弧区）。
	// 弧心 (16,16)。
	// samples 由环境变量 WB_GPU_SAMPLES 控制（默认 0，4 验证 MSAA 是否缓解渗入）。
	samples := 0
	if v := os.Getenv("WB_GPU_SAMPLES"); v == "4" {
		samples = 4
	}
	surf, err := skia.NewGPUSurfaceFromFBO(
		gpuCtx, 0, 120, 40, samples, 8,
		skia.GLRGBA8, skia.ColorTypeRGBA8888, skia.SurfaceOriginBottomLeft,
	)
	if err != nil {
		fmt.Println("gpu surface:", err)
		return
	}
	defer surf.Release()

	c := graphics.NewCanvasFromSurface(surf, 120, 40)

	// 模拟用户场景：容器 (10,10)-(110,22) 12px 高 r=6 overflow:hidden；
	// 内容 3px 宽红色 (10,10)-(13,22) 贴左端（全部在圆弧区）。
	// 弧心 (16,16)。
	c.ClipRoundRect(10, 10, 100, 12, 6)
	c.FillRect(10, 10, 3, 12, graphics.Color{R: 255, G: 0, B: 0, A: 255})

	pix := c.Pixels()
	if pix == nil {
		fmt.Println("read pixels failed")
		return
	}
	// 圆外对角 (10,10) dist=8.49：期望背景（r<100）
	// 圆外 (12,10) dist=sqrt(4+36)=6.32：期望背景（r<100）
	// 弧顶边界 (16,10) dist=6：像素中心(16.5,10.5) dist 5.52 圆内 → 红可接受
	// 圆内 (12,16) dist=4：红
	// 底部圆外 (11,22) dist=sqrt(25+36)=7.8：期望背景
	checks := []struct {
		x, y int
		name string
	}{
		{10, 10, "左上角对角圆外 dist=8.49"},
		{12, 10, "顶部圆弧外 1px dist=6.32"},
		{16, 10, "弧顶边界 dist=6"},
		{10, 16, "左弧顶 dist=6"},
		{12, 16, "圆弧内 dist=4"},
		{16, 16, "弧心"},
		{11, 22, "底部圆外 dist=7.8"},
		{10, 22, "左下对角圆外 dist=8.49"},
	}
	pass := 0
	for _, chk := range checks {
		r, g, b := pixelAt(pix, 120, 40, chk.x, chk.y)
		inside := chk.name == "圆弧内 dist=4" || chk.name == "弧心"
		ok := false
		if inside {
			ok = r > 150 && g < 100
		} else if chk.name == "弧顶边界 dist=6" || chk.name == "左弧顶 dist=6" {
			ok = true // 边界像素中心在圆内，红/混合均可接受
		} else {
			ok = r < 100 // 圆外不得有红
		}
		if ok {
			pass++
		}
		fmt.Printf("  (%d,%d) %s = #%02X%02X%02X %s\n", chk.x, chk.y, chk.name, r, g, b, map[bool]string{true: "✓", false: "✗ 圆外渗入!"}[ok])
	}
	fmt.Printf("  => clip+fill %d/%d 通过\n", pass, len(checks))

	// 对比：FillRoundRect（原生圆角绘制，无 clip）在 GPU 的 AA 过渡带。
	// 内容 (30,10)-(33,22) 3px 宽 r=5（模拟 ctx-bar-fill 自身圆角）。
	// 弧心 (35,16)。顶部行 y=10（dy=-6）：
	//   x=30 → dx=-5 dist=7.81 圆外 2.81
	//   x=31 → dx=-4 dist=7.21 圆外 2.21
	//   x=32 → dx=-3 dist=6.71 圆外 1.71
	//   x=33 → dx=-2 dist=6.32 圆外 1.32
	c.Clear(graphics.Color{})
	c.FillRoundRect(30, 10, 3, 12, 5, graphics.Color{R: 255, G: 0, B: 0, A: 255})
	pix2 := c.Pixels()
	fmt.Printf("  --- FillRoundRect (r=5) 顶部行渗入剖面 ---\n")
	for x := 30; x <= 36; x++ {
		r, g, b := pixelAt(pix2, 120, 40, x, 10)
		fmt.Printf("  (%d,10) = #%02X%02X%02X (dist %.2f)\n", x, r, g, b, float64(x-35)*float64(x-35)+36)
	}
}
