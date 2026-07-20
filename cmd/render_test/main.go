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
		html = `<html><head><meta charset="utf-8"></head><body>` + html + `</body></html>`
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
