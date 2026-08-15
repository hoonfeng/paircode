// Command arrow_scan 系统扫描 stroke/V 高参数，输出与 Edge 参照最匹配的组合。
//go:build ignore

package main

import (
	"fmt"

	"wb-ui/platform/graphics"
)

// Edge s1 参照（220x20 select, cx=238.5, cy=86）：
// 黑核心（RGB<80）分布（相对 cy）：
//
//	cy-3: 灰 2px 两处（cap 顶）
//	cy-2: 黑 2px 两处（臂）→ 每臂中心距 6px
//	cy-1: 黑 3px 两处
//	cy+0: 黑 6px 连续（汇合）
//	cy+1: 黑 4px
//	cy+2: 黑 2px
//	cy+3: 灰 2px（尖端）
func main() {
	// 目标：每行黑像素数（相对 cy）：[-2]=4(两处2px), [-1]=6(两处3px), [0]=6, [1]=4, [2]=2
	target := map[int]int{-2: 4, -1: 6, 0: 6, 1: 4, 2: 2}
	targetRows := []int{-2, -1, 0, 1, 2}

	type combo struct {
		stroke float64
		a, b   float64 // 臂端 cy-a, 尖端 cy+b
	}
	combos := []combo{
		{2.0, 1.5, 1.5}, {2.0, 1.75, 1.75}, {2.0, 2.0, 2.0}, {2.0, 1.0, 2.0},
		{2.5, 1.5, 1.5}, {2.5, 2.0, 2.0}, {2.5, 1.0, 2.0}, {2.5, 2.0, 3.0},
		{3.0, 1.5, 1.5}, {3.0, 2.0, 2.0}, {3.0, 1.0, 2.0}, {3.0, 2.0, 3.0},
		{1.5, 2.0, 3.0}, {2.0, 2.0, 3.0}, {2.5, 2.0, 2.5}, {2.0, 1.0, 2.5},
	}

	fmt.Println("=== 参数扫描（对比 Edge 黑核心 5 行）===")
	for _, cb := range combos {
		canvas := graphics.NewCanvas(220, 20)
		canvas.FillRect(0, 0, 220, 20, graphics.Color{R: 255, G: 255, B: 255, A: 255})
		cx, cy := 110.0, 10.0
		pts := []graphics.Point{
			{X: cx - 3, Y: cy - cb.a},
			{X: cx, Y: cy + cb.b},
			{X: cx + 3, Y: cy - cb.a},
		}
		canvas.StrokePath(pts, cb.stroke, graphics.Color{R: 0, G: 0, B: 0, A: 255}, "round", "round")
		// 统计每行黑像素数（相对 cy）
		score := 0
		rows := ""
		for _, dy := range targetRows {
			y := int(cy) + dy
			cnt := 0
			for x := 0; x < 220; x++ {
				p := canvas.PixelAt(x, y)
				if p.R < 80 && p.G < 80 && p.B < 80 {
					cnt++
				}
			}
			want := target[dy]
			diff := cnt - want
			if diff < 0 {
				diff = -diff
			}
			score += diff
			rows += fmt.Sprintf(" dy%+d:%d ", dy, cnt)
		}
		// 额外惩罚：黑核心总行数不等于 5（统计 alpha>200 的行数）
		coreRows := 0
		for y := 3; y < 17; y++ {
			has := false
			for x := 0; x < 220; x++ {
				p := canvas.PixelAt(x, y)
				if p.R < 80 && p.G < 80 && p.B < 80 {
					has = true
					break
				}
			}
			if has {
				coreRows++
			}
		}
		coreScore := coreRows - 5
		if coreScore < 0 {
			coreScore = -coreScore
		}
		total := score + coreScore*3
		fmt.Printf("stroke=%.1f a=%.1f b=%.1f → score=%d coreRows=%d |%s|\n",
			cb.stroke, cb.a, cb.b, total, coreRows, rows)
	}
}
