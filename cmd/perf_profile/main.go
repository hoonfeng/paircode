// perf_profile — headless 渲染性能分析工具
// 构建: $env:CGO_ENABLED='1'; go build -o perf_profile.exe ./cmd/perf_profile/
// 运行: ./perf_profile.exe
// 输出: cpu.pprof / mem.pprof
//
//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

func main() {
	runtime.LockOSThread()
	fmt.Println("=== goui 性能分析工具 ===")

	// CPU Profile
	cpuF, err := os.Create("cpu.pprof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建cpu profile失败: %v\n", err)
		os.Exit(1)
	}
	pprof.StartCPUProfile(cpuF)
	fmt.Println("CPU profile 采集开始...")

	runScenarios()

	pprof.StopCPUProfile()
	cpuF.Close()
	fmt.Println("CPU profile 已保存: cpu.pprof")

	// Memory Profile
	runtime.GC()
	memF, err := os.Create("mem.pprof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建mem profile失败: %v\n", err)
		os.Exit(1)
	}
	pprof.WriteHeapProfile(memF)
	memF.Close()
	fmt.Println("Memory profile 已保存: mem.pprof")

	fmt.Println("\n=== 完成 ===")
	fmt.Println("运行 'go tool pprof -http=:8080 cpu.pprof' 查看火焰图")
	fmt.Println("运行 'go tool pprof -http=:8081 mem.pprof' 查看内存")
}

func renderOne(name string, w, h int, root widget.Widget) time.Duration {
	sk := canvas.NewSkiaCanvas(w, h)
	defer sk.Release()
	pipe := render.NewPipeline(w, h, sk)
	pipe.SetRootElement(widget.CreateElementFor(root))
	start := time.Now()
	if err := pipe.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "%s 失败: %v\n", name, err)
		return 0
	}
	d := time.Since(start)
	fmt.Printf("  %s: %v\n", name, d)
	return d
}

func runScenarios() {
	// 场景1: 10000行文本
	fmt.Println("\n── 场景1: 10000行文本 ──")
	lines := make([]widget.Widget, 10000)
	for i := 0; i < 10000; i++ {
		lines[i] = widget.NewText(
			fmt.Sprintf("第 %d 行 - 性能测试文本行，用于测试渲染引擎对大量文本的绘制性能。", i+1),
			types.ColorFromRGB(200, 200, 200))
	}
	r1 := widget.Div(
		widget.Style{Width: 800, Height: 600, BackgroundColor: types.ColorRef(30, 30, 30)},
		widget.NewScrollView(widget.VBox(lines...)),
	)
	for i := 0; i < 3; i++ {
		renderOne(fmt.Sprintf("文本(第%d轮)", i+1), 800, 600, r1)
	}

	// 场景2: 500个按钮
	fmt.Println("\n── 场景2: 500个按钮 ──")
	var args []interface{}
	args = append(args, widget.Style{Width: 800, Height: 600, BackgroundColor: types.ColorRef(30, 30, 30), FlexDirection: "column"})
	for i := 0; i < 500; i++ {
		args = append(args, &widget.Button{
			Text: fmt.Sprintf("Btn%d", i+1), FontSize: 11,
			Color: types.ColorFromRGB(60, 60, 60), TextColor: types.ColorFromRGB(200, 200, 200),
			MinWidth: 80, MinHeight: 22,
		})
	}
	r2 := widget.Div(args...)
	for i := 0; i < 3; i++ {
		renderOne(fmt.Sprintf("按钮(第%d轮)", i+1), 800, 600, r2)
	}

	// 场景3: 嵌套布局
	fmt.Println("\n── 场景3: 嵌套布局 ──")
	var bn func(int) widget.Widget
	bn = func(d int) widget.Widget {
		if d <= 0 {
			return widget.Div(widget.Style{Width: 15, Height: 15, BackgroundColor: types.ColorRef(80, 80, 80)})
		}
		var a []interface{}
		a = append(a, widget.Style{FlexDirection: "row", Gap: 1, Padding: types.EdgeInsets(1),
			BackgroundColor: types.ColorRef(50, 50, 50)})
		for i := 0; i < 3; i++ {
			a = append(a, bn(d-1))
		}
		return widget.Div(a...)
	}
	r3 := widget.Div(
		widget.Style{Width: 800, Height: 600, BackgroundColor: types.ColorRef(30, 30, 30)},
		widget.NewScrollView(bn(5)),
	)
	for i := 0; i < 3; i++ {
		renderOne(fmt.Sprintf("嵌套(第%d轮)", i+1), 800, 600, r3)
	}

	// 场景4: CodeEditor 大文件
	fmt.Println("\n── 场景4: CodeEditor 大文件 ──")
	var sb strings.Builder
	sb.WriteString("package perf_test\n\nvar (\n")
	for i := 0; i < 5000; i++ {
		sb.WriteString(fmt.Sprintf("\tperfVar%d = %d\n", i, i*100))
	}
	sb.WriteString(")\n")
	ce := widget.NewCodeEditor("perf_test.go", sb.String())
	ce.FontSize = 13
	r4 := widget.Div(
		widget.Style{Width: 800, Height: 600, BackgroundColor: types.ColorRef(30, 30, 30)},
		ce,
	)
	for i := 0; i < 3; i++ {
		renderOne(fmt.Sprintf("编辑器(第%d轮)", i+1), 800, 600, r4)
	}

	fmt.Println("\n── 全部场景完成 ──")
}
