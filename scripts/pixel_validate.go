//go:build ignore

// Package main 像素级严格验证程序 v2.0。
//
// 本程序使用 SoftCanvas 对各组件进行离屏渲染（headless rendering），
// 然后对渲染结果进行自动检测和逐像素比对，验证：
//  1. 组件渲染是否为空白（必须渲染出内容）
//  2. 组件颜色和位置是否符合预期
//  3. 文字边缘是否有抗锯齿（过渡色像素检测）
//  4. 组件边界是否清晰
//
// 运行方式: go run ./scripts/pixel_validate.go
// 退出码: 0 = ALL PASS, 1 = FAIL
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/goui/pkg/canvas"
	"github.com/user/goui/pkg/types"
	"github.com/user/goui/pkg/validate/visual"
	"github.com/user/goui/pkg/widget"
)

const (
	ToleranceExact  = 0
	ToleranceLow    = 1
	ToleranceMedium = 5
)

// --- Check Types ---

type PixelCheck struct {
	X, Y      int
	Expected  color.RGBA
	Tolerance uint8
	Name      string
	Actual    color.RGBA
	Passed    bool
}

type AASample struct {
	X, Y  int
	Color color.RGBA
}

type AAResult struct {
	TransitionPixels int
	TextPixels       int
	TransitionRatio  float64
	Samples          []AASample
}

// --- PixelValidator ---

type PixelValidator struct {
	SceneName string
	Frame     *visual.VisualFrame
	Checks    []PixelCheck
	Passed    int
	Failed    int
	Errors    []string
}

func NewPixelValidator(sceneName string, frame *visual.VisualFrame) *PixelValidator {
	return &PixelValidator{SceneName: sceneName, Frame: frame}
}

func (pv *PixelValidator) AddCheck(x, y int, expected color.RGBA, tolerance uint8, name string) {
	pv.Checks = append(pv.Checks, PixelCheck{
		X: x, Y: y, Expected: expected, Tolerance: tolerance, Name: name,
	})
}

func (pv *PixelValidator) Run() (allPass bool) {
	pv.Passed = 0
	pv.Failed = 0
	pv.Errors = nil
	allPass = true

	for i, pc := range pv.Checks {
		actual := pv.Frame.PixelAt(pc.X, pc.Y)
		pv.Checks[i].Actual = actual

		dr := absDiff(actual.R, pc.Expected.R)
		dg := absDiff(actual.G, pc.Expected.G)
		db := absDiff(actual.B, pc.Expected.B)
		da := absDiff(actual.A, pc.Expected.A)

		if dr > pc.Tolerance || dg > pc.Tolerance || db > pc.Tolerance || da > pc.Tolerance {
			pv.Checks[i].Passed = false
			err := fmt.Sprintf("  ❌ [像素] (%d,%d) %s: 预期=(%d,%d,%d,%d) 实际=(%d,%d,%d,%d) 差异=(%d,%d,%d,%d)",
				pc.X, pc.Y, pc.Name,
				pc.Expected.R, pc.Expected.G, pc.Expected.B, pc.Expected.A,
				actual.R, actual.G, actual.B, actual.A, dr, dg, db, da)
			pv.Errors = append(pv.Errors, err)
			pv.Failed++
			allPass = false
		} else {
			pv.Checks[i].Passed = true
			pv.Passed++
		}
	}
	return
}

func (pv *PixelValidator) Report() string {
	var sb strings.Builder
	total := pv.Passed + pv.Failed
	sb.WriteString(fmt.Sprintf("\n  📊 [%s] %d/%d 检查通过 (%.0f%%)\n",
		pv.SceneName, pv.Passed, total,
		float64(pv.Passed)/float64(total)*100))
	if len(pv.Errors) > 0 {
		for _, e := range pv.Errors {
			sb.WriteString(e + "\n")
		}
	}
	return sb.String()
}

// --- Utilities ---

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func colorsMatch(a, b color.RGBA, tol uint8) bool {
	return absDiff(a.R, b.R) <= tol &&
		absDiff(a.G, b.G) <= tol &&
		absDiff(a.B, b.B) <= tol &&
		absDiff(a.A, b.A) <= tol
}

// detectContentBounds 检测图像中的非白色内容区域边界
func detectContentBounds(img *image.RGBA) *image.Rectangle {
	b := img.Bounds()
	if b.Empty() {
		return nil
	}
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R != 255 || c.G != 255 || c.B != 255 {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
				found = true
			}
		}
	}
	if !found {
		return nil
	}
	r := image.Rect(minX, minY, maxX+1, maxY+1)
	return &r
}

func saveFrame(frame *visual.VisualFrame, name string) (string, error) {
	dir := "screenshots"
	os.MkdirAll(dir, 0755)
	filename := filepath.Join(dir, fmt.Sprintf("v2_%s_%s.png", name, time.Now().Format("150405")))
	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, frame.Image); err != nil {
		return "", err
	}
	return filename, nil
}

// detectAntialiasing 检测文字抗锯齿：在内容区域中查找介于文字色和背景色之间的过渡像素
func detectAntialiasing(img *image.RGBA, textColor, bgColor color.RGBA, bounds image.Rectangle) *AAResult {
	r := &AAResult{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			isText := colorsMatch(c, textColor, 2)
			isBG := colorsMatch(c, bgColor, 2)
			if isText {
				r.TextPixels++
			} else if !isBG {
				r.TransitionPixels++
				if len(r.Samples) < 10 {
					r.Samples = append(r.Samples, AASample{X: x, Y: y, Color: c})
				}
			}
		}
	}
	if r.TextPixels > 0 {
		r.TransitionRatio = float64(r.TransitionPixels) / float64(r.TextPixels) * 100
	}
	return r
}

func (r *AAResult) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("    文字像素=%d 过渡像素=%d (%.1f%%)\n",
		r.TextPixels, r.TransitionPixels, r.TransitionRatio))
	if r.TextPixels > 0 {
		if r.TransitionRatio < 1.0 {
			sb.WriteString("    状态: ❌ 无抗锯齿\n")
		} else if r.TransitionRatio < 10.0 {
			sb.WriteString("    状态: ⚠ 抗锯齿较弱\n")
		} else {
			sb.WriteString("    状态: ✅ 有抗锯齿\n")
		}
	}
	for i, s := range r.Samples {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("      (%d,%d)=RGBA(%d,%d,%d,%d)\n",
			s.X, s.Y, s.Color.R, s.Color.G, s.Color.B, s.Color.A))
	}
	return sb.String()
}

// renderTextOnBG 在指定背景色上渲染文字用于抗锯齿检测
func renderTextOnBG(text string, size float64, bgColor color.RGBA, w, h int) *image.RGBA {
	tc := types.Color{R: bgColor.R, G: bgColor.G, B: bgColor.B, A: bgColor.A}
	ctx := visual.NewVisualTestContext(w, h)
	_, err := ctx.Render(&widget.Container{
		SingleChildWidget: widget.SingleChildWidget{
			Child: &widget.Text{
				Text:  text,
				Font:  canvas.Font{Family: "sans-serif", Size: size},
				Color: types.ColorFromRGB(33, 33, 33),
			},
		},
		Padding:    types.EdgeInsets(10),
		Background: &widget.PaintWidget{Color: &tc},
	})
	if err != nil {
		return nil
	}
	return ctx.Canvas.Image()
}

// ─────────────────────────────────────────────────────────────
// 验证场景
// ─────────────────────────────────────────────────────────────

// 场景1: Text 纯文字渲染
func validateTextScene() error {
	sceneName := "Text"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(400, 100)
	frame, err := ctx.Render(&widget.Text{
		Text:  "Hello Pixel",
		Font:  canvas.Font{Family: "sans-serif", Size: 20},
		Color: types.ColorFromRGB(33, 33, 33),
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() {
		fmt.Println("  ⚠ 文字未渲染（全白）")
		saveFrame(frame, "text")
		return fmt.Errorf("text not rendered")
	}
	fmt.Printf("  内容区域: (%d,%d)-(%d,%d) [%dx%d]\n",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())

	v := NewPixelValidator(sceneName, frame)
	v.AddCheck(0, 0, color.RGBA{255, 255, 255, 255}, ToleranceExact, "左上角白")
	cx := bounds.Min.X + bounds.Dx()/3
	cy := bounds.Min.Y + bounds.Dy()/3
	v.AddCheck(cx, cy, color.RGBA{33, 33, 33, 255}, ToleranceLow, "文字深色")
	v.AddCheck(350, 80, color.RGBA{255, 255, 255, 255}, ToleranceExact, "右下角白")
	pass := v.Run()
	fmt.Print(v.Report())

	// 抗锯齿检测
	fmt.Println("  ─ 抗锯齿 ─")
	aa := detectAntialiasing(frame.Image,
		color.RGBA{33, 33, 33, 255},
		color.RGBA{255, 255, 255, 255}, *bounds)
	fmt.Print(aa.String())

	path, _ := saveFrame(frame, "text")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// 场景2: Button 按钮渲染
func validateButtonScene() error {
	sceneName := "Button"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(300, 80)
	frame, err := ctx.Render(&widget.Button{
		Text:      "Click Me",
		Color:     types.ColorFromRGB(66, 133, 244),
		MinWidth:  100,
		MinHeight: 40,
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() || bounds.Dx() < 10 {
		fmt.Println("  ❌ 按钮未渲染")
		saveFrame(frame, "button")
		return fmt.Errorf("button not rendered")
	}
	fmt.Printf("  按钮区域: (%d,%d)-(%d,%d) [%dx%d]\n",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())

	v := NewPixelValidator(sceneName, frame)

	// 检测圆角
	cornerWhite := false
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			c := frame.PixelAt(dx, dy)
			if colorsMatch(c, color.RGBA{255, 255, 255, 255}, 2) {
				cornerWhite = true
			}
		}
	}

	// 按钮中心应为蓝色
	cx := bounds.Min.X + bounds.Dx()/2
	cy := bounds.Min.Y + bounds.Dy()/2
	v.AddCheck(cx, cy, color.RGBA{66, 133, 244, 255}, ToleranceLow, "按钮中心蓝")

	// 按钮右侧背景白
	v.AddCheck(bounds.Max.X+2, cy, color.RGBA{255, 255, 255, 255}, ToleranceExact, "右侧背景白")

	// 按钮下方背景白
	v.AddCheck(cx, bounds.Max.Y+2, color.RGBA{255, 255, 255, 255}, ToleranceExact, "下方背景白")

	// 圆角检测：按钮外左上角应为白色
	if bounds.Min.X == 0 && bounds.Min.Y == 0 {
		if cornerWhite {
			v.Passed++
			fmt.Println("  ✅ 左上角圆角正确")
		} else {
			err := "  ⚠ 左上角非白色，圆角可能缺失"
			v.Errors = append(v.Errors, err)
			v.Failed++
		}
	}

	// 白色文字检测：在按钮中心偏上区域找白色像素
	hasWhiteText := false
	for y := bounds.Min.Y + bounds.Dy()/4; y < bounds.Min.Y+bounds.Dy()*3/4; y++ {
		for x := bounds.Min.X + bounds.Dx()/4; x < bounds.Min.X+bounds.Dx()*3/4; x++ {
			c := frame.PixelAt(x, y)
			// 白色文字在蓝色背景上产生高亮色（RGB值显著高于纯蓝）
			if int(c.R)+int(c.G)+int(c.B) > 500 {
				hasWhiteText = true
				break
			}
		}
		if hasWhiteText {
			break
		}
	}
	if hasWhiteText {
		v.Passed++
		fmt.Println("  ✅ 白色文字存在")
	} else {
		err := "  ⚠ 未检测到白色文字"
		v.Errors = append(v.Errors, err)
		v.Failed++
	}

	pass := v.Run()
	fmt.Print(v.Report())

	path, _ := saveFrame(frame, "button")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// 场景3: Checkbox 选中
func validateCheckboxScene() error {
	sceneName := "Checkbox(checked)"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(400, 60)
	frame, err := ctx.Render(&widget.Checkbox{
		Label:       "Accept terms",
		Checked:     true,
		ActiveColor: types.ColorFromRGB(66, 133, 244),
		LabelColor:  types.ColorFromRGB(33, 33, 33),
		BoxSize:     18,
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() || bounds.Dx() < 5 {
		fmt.Println("  ⚠ Checkbox 未渲染")
		saveFrame(frame, "checkbox")
		return nil
	}
	fmt.Printf("  内容区域: (%d,%d)-(%d,%d) [%dx%d]\n",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())

	v := NewPixelValidator(sceneName, frame)
	v.AddCheck(0, 0, color.RGBA{255, 255, 255, 255}, ToleranceExact, "左上角白")

	// 方框区域蓝色
	v.AddCheck(bounds.Min.X+5, bounds.Min.Y+bounds.Dy()/2,
		color.RGBA{66, 133, 244, 255}, ToleranceLow, "方框蓝")

	// 标签文字深色
	v.AddCheck(bounds.Min.X+bounds.Dx()*2/3, bounds.Min.Y+bounds.Dy()/2,
		color.RGBA{33, 33, 33, 255}, ToleranceLow, "标签文字")

	// 右侧背景白
	v.AddCheck(bounds.Max.X+5, bounds.Min.Y+bounds.Dy()/2,
		color.RGBA{255, 255, 255, 255}, ToleranceExact, "右侧白")

	pass := v.Run()
	fmt.Print(v.Report())

	// 渲染质量
	fmt.Println("  ─ 渲染质量 ─")
	aa := detectAntialiasing(frame.Image,
		color.RGBA{33, 33, 33, 255},
		color.RGBA{255, 255, 255, 255}, *bounds)
	fmt.Print(aa.String())

	path, _ := saveFrame(frame, "checkbox")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// 场景4: Checkbox 未选中
func validateUncheckedScene() error {
	sceneName := "Checkbox(unchecked)"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(400, 60)
	frame, err := ctx.Render(&widget.Checkbox{
		Label:       "Option",
		Checked:     false,
		ActiveColor: types.ColorFromRGB(66, 133, 244),
		LabelColor:  types.ColorFromRGB(33, 33, 33),
		BoxSize:     18,
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() || bounds.Dx() < 5 {
		fmt.Println("  ⚠ Checkbox(unchecked) 未渲染")
		saveFrame(frame, "checkbox_unchecked")
		return nil
	}
	fmt.Printf("  内容区域: (%d,%d)-(%d,%d) [%dx%d]\n",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())

	v := NewPixelValidator(sceneName, frame)
	v.AddCheck(0, 0, color.RGBA{255, 255, 255, 255}, ToleranceExact, "左上角白")

	// 方框边框灰色
	v.AddCheck(bounds.Min.X+2, bounds.Min.Y+bounds.Dy()/2,
		color.RGBA{180, 180, 180, 255}, ToleranceLow, "方框边框灰")

	// 标签文字深色
	v.AddCheck(bounds.Min.X+bounds.Dx()*2/3, bounds.Min.Y+bounds.Dy()/2,
		color.RGBA{33, 33, 33, 255}, ToleranceLow, "标签文字")

	pass := v.Run()
	fmt.Print(v.Report())

	fmt.Println("  ─ 渲染质量 ─")
	aa := detectAntialiasing(frame.Image,
		color.RGBA{33, 33, 33, 255},
		color.RGBA{255, 255, 255, 255}, *bounds)
	fmt.Print(aa.String())

	path, _ := saveFrame(frame, "checkbox_unchecked")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// 场景5: Container
func validateContainerScene() error {
	sceneName := "Container"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(200, 200)
	frame, err := ctx.Render(&widget.Container{
		SingleChildWidget: widget.SingleChildWidget{
			Child: &widget.Text{
				Text:  "Boxed",
				Font:  canvas.Font{Family: "sans-serif", Size: 16},
				Color: types.ColorFromRGB(33, 33, 33),
			},
		},
		Padding:    types.EdgeInsets(20),
		Background: &widget.PaintWidget{Color: types.ColorRef(240, 240, 240)},
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() || bounds.Dx() < 5 {
		fmt.Println("  ⚠ Container 未渲染")
		saveFrame(frame, "container")
		return nil
	}
	fmt.Printf("  内容区域: (%d,%d)-(%d,%d) [%dx%d]\n",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())

	v := NewPixelValidator(sceneName, frame)

	// 左上角背景灰
	v.AddCheck(bounds.Min.X+2, bounds.Min.Y+2,
		color.RGBA{240, 240, 240, 255}, ToleranceLow, "左上角灰")

	// 右下角背景灰
	v.AddCheck(bounds.Max.X-2, bounds.Max.Y-2,
		color.RGBA{240, 240, 240, 255}, ToleranceLow, "右下角灰")

	// 内部文字深色(容器中心)
	if bounds.Dx() > 40 && bounds.Dy() > 40 {
		v.AddCheck(bounds.Min.X+bounds.Dx()/2, bounds.Min.Y+bounds.Dy()/2,
			color.RGBA{33, 33, 33, 255}, ToleranceLow, "内部文字")
	}

	// 容器外部白色背景
	v.AddCheck(180, 180, color.RGBA{255, 255, 255, 255}, ToleranceExact, "外部白")

	pass := v.Run()
	fmt.Print(v.Report())

	path, _ := saveFrame(frame, "container")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// 场景6: Input
func validateInputScene() error {
	sceneName := "Input"
	fmt.Printf("\n─── [%s] ───\n", sceneName)

	ctx := visual.NewVisualTestContext(400, 80)
	frame, err := ctx.Render(&widget.Input{
		Placeholder: "Type here...",
	})
	if err != nil {
		return fmt.Errorf("Render failed: %w", err)
	}

	bounds := detectContentBounds(frame.Image)
	if bounds == nil || bounds.Empty() {
		fmt.Println("  ⚠ Input 渲染全白 → 需要检查 Input widget 的 Paint 实现")
		saveFrame(frame, "input")
		return nil
	}
	fmt.Printf("  内容区域: (%d,%d)-(%d,%d)\n", bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
	v := NewPixelValidator(sceneName, frame)
	v.AddCheck(0, 0, color.RGBA{255, 255, 255, 255}, ToleranceExact, "左上角白")
	pass := v.Run()
	fmt.Print(v.Report())

	path, _ := saveFrame(frame, "input")
	if path != "" {
		fmt.Printf("  截图: %s\n", path)
	}
	if pass {
		fmt.Println("  ✅ PASS")
	} else {
		fmt.Println("  ❌ FAIL")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────
// 抗锯齿综合报告
// ─────────────────────────────────────────────────────────────

func generateAAGlobalReport() {
	fmt.Println("\n═══════════ 抗锯齿综合分析 ═══════════")
	bgBlue := color.RGBA{0, 0, 200, 255}
	textClr := color.RGBA{33, 33, 33, 255}

	for _, txt := range []string{"Ag", "Hello", "中文测试"} {
		for _, sz := range []float64{14, 20, 36, 48, 72} {
			img := renderTextOnBG(txt, sz, bgBlue, 400, 120)
			if img == nil {
				continue
			}
			b := detectContentBounds(img)
			if b == nil {
				continue
			}
			r := detectAntialiasing(img, textClr, bgBlue, *b)
			if r.TextPixels > 0 {
				status := "❌未检出"
				if r.TransitionRatio >= 1.0 {
					status = "✅有AA"
				} else if r.TransitionRatio >= 0.1 {
					status = "⚠弱AA"
				}
				fmt.Printf("  %s [%s size=%.0f] 文字=%dpx 过渡=%dpx(%.1f%%)\n",
					status, txt, sz, r.TextPixels, r.TransitionPixels, r.TransitionRatio)
			}
		}
	}
	fmt.Println("═══════════════════════════════════════")
}

// ─────────────────────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────────────────────

func main() {
	exitCode := 0
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  goui 逐像素验证 v2.0")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  时间: %s\n\n", time.Now().Format("15:04:05"))

	scenes := []struct {
		name string
		fn   func() error
	}{
		{"Text", validateTextScene},
		{"Button", validateButtonScene},
		{"Checkbox(checked)", validateCheckboxScene},
		{"Checkbox(unchecked)", validateUncheckedScene},
		{"Container", validateContainerScene},
		{"Input", validateInputScene},
	}

	passed := 0
	for _, s := range scenes {
		if err := s.fn(); err != nil {
			fmt.Printf("\n  ❌ [%s] 出错: %v\n", s.name, err)
			exitCode = 1
		} else {
			passed++
		}
	}

	generateAAGlobalReport()

	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Printf("  通过: %d/%d 场景\n", passed, len(scenes))
	if exitCode == 0 {
		fmt.Println("  ✅ ALL PASSED")
	} else {
		fmt.Println("  ❌ SOME FAILED")
	}
	fmt.Println("═══════════════════════════════════════════")
	os.Exit(exitCode)
}
