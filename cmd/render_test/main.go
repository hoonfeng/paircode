package main

import (
	"fmt"
	"os"
	"strings"

	"wb-ui/webkit"
)

type TestResult struct {
	Name   string
	Passed bool
	Detail string
}

type TestCase struct {
	Name     string
	HTML     string
	Width    int
	Height   int
	Checks   []PixelCheck
}

type PixelCheck struct {
	X, Y       int
	R, G, B, A uint8
	Label      string
	// Region 模式下在 (X,Y)-(X+W,Y+H) 区域内查找完全匹配像素（任一命中即通过）。
	// 用于文本等字形/抗锯齿像素位置不稳定的断言。
	Region bool
	W, H   int
}

var allResults []TestResult

func main() {
	os.Setenv("CGO_ENABLED", "1")
	fmt.Println("========================================")
	fmt.Println("  wb-ui 渲染引擎全面排测工具")
	fmt.Println("  覆盖: CSS颜色/盒模型/Flex/Grid/文本/定位/渲染属性")
	fmt.Println("========================================")
	fmt.Println()

	// 按层级运行测试
	runLayer("层级1: 基础 CSS 颜色与盒模型", runLayer1ColorAndBox)
	runLayer("层级2: 尺寸与定位", runLayer2SizingPosition)
	runLayer("层级3: Flexbox 全面测试", runLayer3Flexbox)
	runLayer("层级4: Grid 测试", runLayer4Grid)
	runLayer("层级5: 文本渲染测试", runLayer5Text)
	runLayer("层级6: 渲染属性测试", runLayer6RenderProps)
	runLayer("层级7: 高阶 CSS 测试", runLayer7AdvancedCSS)
	runLayer("层级8: 综合布局/IDE 模拟", runLayer8ComplexLayout)

	// 汇总报告
	printSummary()
}

func runLayer(name string, fn func()) {
	fmt.Printf("\n━━━ %s ━━━\n", name)
	before := len(allResults)
	fn()
	after := len(allResults)
	passed := 0
	for i := before; i < after; i++ {
		if allResults[i].Passed {
			passed++
		}
	}
	fmt.Printf("  → %d/%d 通过\n", passed, after-before)
}

func runTest(tc TestCase) {
	wv := webkit.NewWebView()
	wv.Resize(tc.Width, tc.Height)

	html := tc.HTML
	if !strings.Contains(html, "<html") {
		// 统一 body 黑背景：让「背景」断言明确且可预测（原期望 0,0,0,255），
		// 半透明颜色在黑色背景上的混合 = premultiplied 值（如 rgba(255,0,0,.5)
		// → (127,0,0,255)）。浏览器 body 默认白，此处用黑使期望可计算。
		html = `<html><head><meta charset="utf-8"></head><body style="background:#000">` + html + `</body></html>`
	}

	err := wv.LoadHTML(html)
	if err != nil {
		allResults = append(allResults, TestResult{
			Name: tc.Name, Passed: false,
			Detail: fmt.Sprintf("LoadHTML 失败: %v", err),
		})
		return
	}

	pixels, err := wv.Render()
	if err != nil {
		allResults = append(allResults, TestResult{
			Name: tc.Name, Passed: false,
			Detail: fmt.Sprintf("Render 失败: %v", err),
		})
		return
	}

	if len(pixels) != tc.Width*tc.Height*4 {
		allResults = append(allResults, TestResult{
			Name: tc.Name, Passed: false,
			Detail: fmt.Sprintf("像素尺寸错误: %d, 期望 %d", len(pixels), tc.Width*tc.Height*4),
		})
		return
	}

	for _, c := range tc.Checks {
		if c.Region {
			// 区域内任意像素完全匹配（r/g/b/a 相等）即通过——文本字形
			// 笔画内存在纯色像素，边缘抗锯齿像素不参与比较。
			found := false
			for ry := c.Y; ry < c.Y+c.H && ry < tc.Height && !found; ry++ {
				for rx := c.X; rx < c.X+c.W && rx < tc.Width; rx++ {
					ri := (ry*tc.Width + rx) * 4
					if ri+3 >= len(pixels) {
						continue
					}
					if pixels[ri] == c.R && pixels[ri+1] == c.G &&
						pixels[ri+2] == c.B && pixels[ri+3] == c.A {
						found = true
						break
					}
				}
			}
			if !found {
				allResults = append(allResults, TestResult{
					Name: tc.Name, Passed: false,
					Detail: fmt.Sprintf("  [%s] 区域(%d,%d)+%dx%d 内未找到 rgba(%d,%d,%d,%d)",
						c.Label, c.X, c.Y, c.W, c.H, c.R, c.G, c.B, c.A),
				})
				return
			}
			continue
		}
		idx := (c.Y*tc.Width + c.X) * 4
		if idx+3 >= len(pixels) {
			allResults = append(allResults, TestResult{
				Name: tc.Name, Passed: false,
				Detail: fmt.Sprintf("坐标越界 (%d,%d)", c.X, c.Y),
			})
			return
		}
		r, g, b, a := pixels[idx], pixels[idx+1], pixels[idx+2], pixels[idx+3]
		if r != c.R || g != c.G || b != c.B || a != c.A {
			allResults = append(allResults, TestResult{
				Name: tc.Name, Passed: false,
				Detail: fmt.Sprintf("  [%s] (%d,%d): 期望 rgba(%d,%d,%d,%d), 实际 rgba(%d,%d,%d,%d)",
					c.Label, c.X, c.Y, c.R, c.G, c.B, c.A, r, g, b, a),
			})
			return
		}
	}

	allResults = append(allResults, TestResult{
		Name: tc.Name, Passed: true, Detail: fmt.Sprintf("所有 %d 个检查点通过", len(tc.Checks)),
	})
}

func check(label string, x, y, r, g, b, a int) PixelCheck {
	return PixelCheck{X: x, Y: y, R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a), Label: label}
}

// checkRegion 在区域 (x,y)-(x+w,y+h) 内查找完全匹配像素（任一命中即通过）。
func checkRegion(label string, x, y, w, h, r, g, b, a int) PixelCheck {
	return PixelCheck{X: x, Y: y, R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a), Label: label,
		Region: true, W: w, H: h}
}

func printSummary() {
	fmt.Println("\n========================================")
	fmt.Println("  排测汇总报告")
	fmt.Println("========================================")
	total := len(allResults)
	passed := 0
	for _, r := range allResults {
		if r.Passed {
			passed++
		} else {
			fmt.Printf("  ❌ %s\n", r.Name)
			fmt.Printf("     %s\n", r.Detail)
		}
	}
	fmt.Printf("\n  总测试: %d\n", total)
	fmt.Printf("  通过:   %d\n", passed)
	fmt.Printf("  失败:   %d\n", total-passed)
	if passed == total {
		fmt.Println("\n  🎉 全部通过！")
	} else {
		fmt.Println("\n  ⚠️  有待修复的测试")
	}
}
