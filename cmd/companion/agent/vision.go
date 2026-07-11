// vision.go 图像视觉分析工具：图色分析 + OCR。
//
// 提供两个工具：
//   - image_analyze：分析图片中的颜色分布、色块区域和图形，按坐标块格式输出
//   - image_ocr：从图片中识别文字（OCR），返回文字内容及其坐标位置
//
// 图像格式支持：PNG / JPEG（使用 Go 标准库 image 包）
// OCR 依赖：系统安装的 Tesseract OCR（https://github.com/tesseract-ocr/tesseract）

package agent

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ── 注册 ───────────────────────────────────────────────────

// registerVisionTools 注册 image_analyze 和 image_ocr 工具。
func registerVisionTools(r *Registry, root string) {
	// ── image_analyze ──
	r.Register(&Tool{
		Name: "image_analyze",
		Description: "分析图片中的颜色分布、色块区域和基本图形。" +
			"输入图片路径，返回按坐标块 (x1,y1)-(x2,y2) 描述的详细分析结果。" +
			"支持 PNG / JPEG 格式。" +
			"可用于分析 UI 界面布局、色块区域、颜色构成、图形检测等视觉分析任务。",
		Parameters: objSchema(props{
			"path":       strProp("图片路径（工作区内）"),
			"detail":     strProp("可选：分析详细程度，\"high\"（详细）或 \"low\"（概览），默认 \"high\""),
			"max_colors": intProp("可选：最大颜色聚类数，默认 8，范围 1-32"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			detail := argStr(args, "detail")
			if detail == "" {
				detail = "high"
			}
			maxColors := argInt(args, "max_colors", 8)
			if maxColors <= 0 {
				maxColors = 8
			}
			if maxColors > 32 {
				maxColors = 32
			}
			return analyzeImage(p, detail, maxColors)
		},
	})

	// ── image_ocr ──
	r.Register(&Tool{
		Name: "image_ocr",
		Description: "从图片中识别文字（OCR）。" +
			"返回识别出的文字内容及其在图片中的坐标位置 (x1,y1)-(x2,y2)。" +
			"支持项目内嵌的 Tesseract 便携版（无需安装），也支持系统已安装的 Tesseract。" +
			"支持 PNG / JPEG 格式。" +
			"可用 lang 参数指定语言，如 \"chi_sim+eng\"（中英文混合）、\"eng\"（仅英文）。",
		Parameters: objSchema(props{
			"path":   strProp("图片路径（工作区内）"),
			"lang":   strProp("可选：识别语言，如 \"chi_sim+eng\"（中英文）、\"eng\"（仅英文），默认 \"chi_sim+eng\""),
			"detail": boolProp("可选：是否返回详细的坐标信息，默认 true"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			lang := argStr(args, "lang")
			if lang == "" {
				lang = "chi_sim+eng"
			}
			detail := true
			if v, ok := args["detail"]; ok {
				if b, ok := v.(bool); ok {
					detail = b
				}
			}
			return ocrImage(root, p, lang, detail)
		},
	})
}

// ── 图色分析 ───────────────────────────────────────────────

// analyzeImage 分析图片的颜色分布、色块区域和图形。
func analyzeImage(path, detail string, maxColors int) (string, error) {
	// 1. 打开并解码图片
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("无法打开图片 %q: %w", path, err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("无法解码图片（格式可能不支持）: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return "", fmt.Errorf("图片尺寸无效：%dx%d", width, height)
	}

	// 2. 构建输出
	var b strings.Builder
	fileName := filepath.Base(path)
	fmt.Fprintf(&b, "## 图色分析结果\n\n")
	fmt.Fprintf(&b, "| 属性 | 值 |\n|------|-----|\n")
	fmt.Fprintf(&b, "| 文件 | %s |\n", fileName)
	fmt.Fprintf(&b, "| 格式 | %s |\n", format)
	fmt.Fprintf(&b, "| 尺寸 | %d × %d px |\n", width, height)
	fmt.Fprintf(&b, "| 像素 | %d |\n", width*height)
	b.WriteString("\n")

	// 3. 颜色量化分析
	colors, totalPixels := quantizeColors(img, bounds, maxColors)

	b.WriteString("━━━ 颜色分布 ━━━\n\n")
	b.WriteString("| 序号 | 颜色 | HEX | RGB | 占比 | 像素数 | 色样 |\n")
	b.WriteString("|------|------|-----|-----|------|--------|------|\n")
	for i, c := range colors {
		pct := float64(c.Count) / float64(totalPixels) * 100
		r, g, bVal := c.R>>8, c.G>>8, c.B>>8 // 从 16 位缩回 8 位
		hex := fmt.Sprintf("#%02X%02X%02X", r, g, bVal)
		bar := colorBar(pct, 20)
		fmt.Fprintf(&b, "| %d | %s | `%s` | (%d,%d,%d) | %.1f%% | %d | %s |\n",
			i+1, c.Name, hex, r, g, bVal, pct, c.Count, bar)
	}
	b.WriteString("\n")

	// 4. 色块区域检测（坐标块）
	var blocks []colorBlock
	if detail == "high" {
		blocks = detectColorBlocks(img, bounds, colors)
		b.WriteString("━━━ 色块区域（坐标块）━━━\n\n")
		if len(blocks) == 0 {
			b.WriteString("（未检测到显著色块区域）\n\n")
		} else {
			b.WriteString("| 区域 | 坐标块 | 尺寸 | 颜色 | 描述 |\n")
			b.WriteString("|------|--------|------|------|------|\n")
			for i, blk := range blocks {
				w := blk.X2 - blk.X1
				h := blk.Y2 - blk.Y1
				r, g, bVal := blk.R>>8, blk.G>>8, blk.B>>8
				hex := fmt.Sprintf("#%02X%02X%02X", r, g, bVal)
				desc := blk.Description
				if desc == "" {
					desc = describeColorRegion(blk, width, height)
				}
				fmt.Fprintf(&b, "| %d | `(%d,%d)-(%d,%d)` | %dx%d | `%s` | %s |\n",
					i+1, blk.X1, blk.Y1, blk.X2, blk.Y2, w, h, hex, desc)
			}
			b.WriteString("\n")
		}

		// 5. 图形检测（增强：几何形状 + 图表元素 + 曲线/折线/箭头）
		shapes := detectShapes(img, bounds, width, height)
		if len(shapes) > 0 {
			b.WriteString("━━━ 图形与图表检测 ━━━\n\n")

			// 按类型分组输出
			chartTypes := filterShapesByKindPrefix(shapes, "图表类型")
			axes := filterShapesByKind(shapes, "坐标轴")
			geoShapes := filterShapesByKindList(shapes,
				[]string{"圆形", "椭圆", "近似椭圆", "矩形", "圆角矩形", "三角形", "四边形", "菱形/四边形",
					"五边形", "六边形", "七边形", "八边形", "九边形", "十边形", "扇形", "多边形", "复杂轮廓"})
			lines := filterShapesByKindList(shapes, []string{"线条", "曲线/折线", "箭头", "折线/曲线"})
			chartParts := filterShapesByKindList(shapes, []string{"柱状图柱体", "图例区域", "图例标记"})
			others := filterOtherShapes(shapes, chartTypes, axes, geoShapes, lines, chartParts)

			// 图表类型总览
			if len(chartTypes) > 0 {
				b.WriteString("【图表类型】\n\n")
				for _, s := range chartTypes {
					fmt.Fprintf(&b, "- %s\n", s.Description)
				}
				b.WriteString("\n")
			}

			// 坐标轴
			if len(axes) > 0 {
				b.WriteString("【坐标轴】\n\n")
				b.WriteString("| 类型 | 坐标 | 尺寸 | 描述 |\n")
				b.WriteString("|------|------|------|------|\n")
				for _, s := range axes {
					fmt.Fprintf(&b, "| %s | `(%d,%d)-(%d,%d)` | %dx%d | %s |\n",
						s.Kind, s.X1, s.Y1, s.X2, s.Y2, s.X2-s.X1, s.Y2-s.Y1, s.Description)
				}
				b.WriteString("\n")
			}

			// 几何形状
			if len(geoShapes) > 0 {
				b.WriteString("【几何形状】\n\n")
				b.WriteString("| 类型 | 位置 | 尺寸 | 中心 | 面积 | 周长 | 圆度 | 填充率 | 描述 |\n")
				b.WriteString("|------|------|------|------|------|------|------|--------|------|\n")
				for _, s := range geoShapes {
					perimStr := fmt.Sprintf("%.0f", s.Perimeter)
					if s.Perimeter <= 0 {
						perimStr = "-"
					}
					circStr := fmt.Sprintf("%.2f", s.Circularity)
					if s.Circularity <= 0 {
						circStr = "-"
					}
					fillStr := fmt.Sprintf("%.0f%%", s.FillRatio*100)
					if s.FillRatio <= 0 {
						fillStr = "-"
					}
					areaStr := fmt.Sprintf("%d", s.Area)
					if s.Area <= 0 {
						areaStr = "-"
					}
					fmt.Fprintf(&b, "| %s | `(%d,%d)-(%d,%d)` | %dx%d | (%d,%d) | %s | %s | %s | %s | %s |\n",
						s.Kind, s.X1, s.Y1, s.X2, s.Y2,
						s.X2-s.X1, s.Y2-s.Y1,
						s.CenterX, s.CenterY,
						areaStr, perimStr, circStr, fillStr, s.Description)
				}
				b.WriteString("\n")
			}

			// 线条/曲线/箭头
			if len(lines) > 0 {
				b.WriteString("【线条/曲线/箭头】\n\n")
				b.WriteString("| 类型 | 坐标 | 尺寸 | 描述 |\n")
				b.WriteString("|------|------|------|------|\n")
				for _, s := range lines {
					fmt.Fprintf(&b, "| %s | `(%d,%d)-(%d,%d)` | %dx%d | %s |\n",
						s.Kind, s.X1, s.Y1, s.X2, s.Y2, s.X2-s.X1, s.Y2-s.Y1, s.Description)
				}
				b.WriteString("\n")
			}

			// 图表部件（柱体/图例）
			if len(chartParts) > 0 {
				b.WriteString("【图表部件】\n\n")
				b.WriteString("| 类型 | 坐标 | 尺寸 | 描述 |\n")
				b.WriteString("|------|------|------|------|\n")
				for _, s := range chartParts {
					fmt.Fprintf(&b, "| %s | `(%d,%d)-(%d,%d)` | %dx%d | %s |\n",
						s.Kind, s.X1, s.Y1, s.X2, s.Y2, s.X2-s.X1, s.Y2-s.Y1, s.Description)
				}
				b.WriteString("\n")
			}

			// 其他形状
			if len(others) > 0 {
				b.WriteString("【其他】\n\n")
				b.WriteString("| 类型 | 坐标 | 尺寸 | 描述 |\n")
				b.WriteString("|------|------|------|------|\n")
				for _, s := range others {
					fmt.Fprintf(&b, "| %s | `(%d,%d)-(%d,%d)` | %dx%d | %s |\n",
						s.Kind, s.X1, s.Y1, s.X2, s.Y2, s.X2-s.X1, s.Y2-s.Y1, s.Description)
				}
				b.WriteString("\n")
			}
		}
	}

	// 6. 补充信息
	b.WriteString("━━━ 分析摘要 ━━━\n\n")
	fmt.Fprintf(&b, "- 主要颜色数: %d\n", len(colors))
	fmt.Fprintf(&b, "- 检测到色块: %d 个\n", len(blocks))
	fmt.Fprintf(&b, "- 主色调: %s\n", dominantColorDesc(colors))
	if detail == "low" {
		b.WriteString("\n> 提示：设置 detail=\"high\" 可查看详细的色块区域坐标和图形检测。\n")
	}

	return b.String(), nil
}

// ── 颜色量化 ──────────────────────────────────────────────

// quantizedColor 量化后的颜色统计。
type quantizedColor struct {
	R, G, B uint32 // 16 位颜色值（image.Color 返回的格式）
	Name    string // 颜色名称
	Count   int    // 像素数
}

// quantizeColors 量化图片颜色并返回按像素数降序排列的颜色列表。
func quantizeColors(img image.Image, bounds image.Rectangle, maxColors int) ([]quantizedColor, int) {
	// 使用 4 位量化（16 级），将颜色空间压缩为 16x16x16 = 4096 种
	const levels = 16
	colorMap := make(map[uint32]int)
	totalPixels := 0

	// 降采样步长（大图加速）
	step := 1
	w := bounds.Dx()
	h := bounds.Dy()
	if w*h > 200000 {
		step = 2
	}
	if w*h > 800000 {
		step = 3
	}
	if w*h > 2000000 {
		step = 4
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 128 {
				continue // 忽略半透明/透明像素
			}
			// 量化：取高 4 位
			qr := r >> 12
			qg := g >> 12
			qb := b >> 12
			key := (qr << 8) | (qg << 4) | qb
			colorMap[key]++
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return nil, 0
	}

	// 转切片并排序
	var colors []quantizedColor
	for key, count := range colorMap {
		qr := (key >> 8) & 0xF
		qg := (key >> 4) & 0xF
		qb := key & 0xF
		// 反量化回 16 位（取中间值）
		r := uint32(qr)*0x1111 + 0x888
		g := uint32(qg)*0x1111 + 0x888
		b := uint32(qb)*0x1111 + 0x888
		colors = append(colors, quantizedColor{
			R: r, G: g, B: b,
			Name:  colorName(r>>8, g>>8, b>>8),
			Count: count,
		})
	}

	sort.Slice(colors, func(i, j int) bool {
		return colors[i].Count > colors[j].Count
	})

	// 合并前 N 个，其余算"其他"
	if len(colors) > maxColors {
		others := quantizedColor{
			R:   0x8888,
			G:   0x8888,
			B:   0x8888,
			Name: "其他颜色",
		}
		for i := maxColors - 1; i < len(colors); i++ {
			others.Count += colors[i].Count
		}
		colors = append(colors[:maxColors-1], others)
	}

	// 按实际采样像素数比例放大到全图
	totalFull := w * h
	if step > 1 {
		scaleFactor := float64(totalFull) / float64(totalPixels)
		for i := range colors {
			colors[i].Count = int(float64(colors[i].Count) * scaleFactor)
		}
	}

	return colors, totalFull
}

// ── 色块检测 ──────────────────────────────────────────────

// colorBlock 一个色块区域（坐标块）。
type colorBlock struct {
	X1, Y1, X2, Y2 int    // 边界框
	R, G, B        uint32 // 主要颜色
	Description     string // 描述
}

// detectColorBlocks 检测图片中的主要色块区域。
func detectColorBlocks(img image.Image, bounds image.Rectangle, colors []quantizedColor) []colorBlock {
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return nil
	}

	// 步长（大图降采样加速）
	step := 1
	if w*h > 300000 {
		step = 2
	}
	if w*h > 1200000 {
		step = 3
	}
	if w*h > 3000000 {
		step = 4
	}

	// 构建降采样后的颜色标签图
	type pixelTag struct {
		colorIdx int // 对应 colors 的索引
		done     bool
	}

	// 先用主要颜色做连通域分析
	// 对每种主要颜色，找到其连通区域
	var blocks []colorBlock

	// 使用 BFS flood fill 检测每种颜色的连通域
	// 为了效率，只用 Top-8 颜色做连通检测，而且限制块大小
	maxColorsToAnalyze := len(colors)
	if maxColorsToAnalyze > 8 {
		maxColorsToAnalyze = 8
	}

	for ci := 0; ci < maxColorsToAnalyze; ci++ {
		if colors[ci].Count < w*h/200 { // 少于 0.5% 像素忽略
			continue
		}
		// flood fill
		visited := make([][]bool, (h+step-1)/step)
		for i := range visited {
			visited[i] = make([]bool, (w+step-1)/step)
		}

		targetR, targetG, targetB := colors[ci].R, colors[ci].G, colors[ci].B

		for y := 0; y < h; y += step {
			for x := 0; x < w; x += step {
				ix, iy := x/step, y/step
				if visited[iy][ix] {
					continue
				}
				r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				if a < 128 {
					visited[iy][ix] = true
					continue
				}
				if !colorMatch(r, g, b, targetR, targetG, targetB, 0x2222) {
					visited[iy][ix] = true
					continue
				}
				// Flood fill 找到连通域
				minX, minY, maxX, maxY := x, y, x+step, y+step
				pixelCount := 0
				stack := [][2]int{{ix, iy}}
				visited[iy][ix] = true

				for len(stack) > 0 {
					cx, cy := stack[len(stack)-1][0], stack[len(stack)-1][1]
					stack = stack[:len(stack)-1]
					px := cx * step
					py := cy * step
					if px < minX {
						minX = px
					}
					if py < minY {
						minY = py
					}
					if px+step > maxX {
						maxX = px + step
					}
					if py+step > maxY {
						maxY = py + step
					}
					pixelCount++

					// 4 向邻居
					dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
					for _, d := range dirs {
						nx, ny := cx+d[0], cy+d[1]
						if nx < 0 || ny < 0 || nx >= len(visited[0]) || ny >= len(visited) {
							continue
						}
						if visited[ny][nx] {
							continue
						}
						// 检查颜色匹配
						npx := nx * step
						npy := ny * step
						nr, ng, nb, na := img.At(bounds.Min.X+npx, bounds.Min.Y+npy).RGBA()
						if na < 128 {
							visited[ny][nx] = true
							continue
						}
						if !colorMatch(nr, ng, nb, targetR, targetG, targetB, 0x2222) {
							visited[ny][nx] = true
							continue
						}
						visited[ny][nx] = true
						stack = append(stack, [2]int{nx, ny})
					}
				}

				// 过滤过小区域
				blockW := (maxX - minX)
				blockH := maxY - minY
				area := blockW * blockH
				if area < w*h/5000 || (blockW < 5 && blockH < 5) {
					continue
				}
				// 过滤掉太大的（背景色）
				if float64(area) > float64(w*h)*0.7 {
					continue
				}

				blocks = append(blocks, colorBlock{
					X1: minX, Y1: minY,
					X2: maxX, Y2: maxY,
					R: targetR, G: targetG, B: targetB,
				})
			}
		}
	}

	// 合并重叠或相邻的块
	blocks = mergeOverlappingBlocks(blocks)

	// 按面积排序（大的在前）
	sort.Slice(blocks, func(i, j int) bool {
		ai := (blocks[i].X2 - blocks[i].X1) * (blocks[i].Y2 - blocks[i].Y1)
		aj := (blocks[j].X2 - blocks[j].X1) * (blocks[j].Y2 - blocks[j].Y1)
		return ai > aj
	})

	// 限制返回数量
	if len(blocks) > 30 {
		blocks = blocks[:30]
	}

	return blocks
}

// colorMatch 判断两个颜色是否近似匹配（在容差范围内）。
func colorMatch(r1, g1, b1, r2, g2, b2 uint32, tolerance uint32) bool {
	dr := absDiff(r1, r2)
	dg := absDiff(g1, g2)
	db := absDiff(b1, b2)
	return dr+db+dg <= tolerance
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// mergeOverlappingBlocks 合并重叠或相邻的色块。
func mergeOverlappingBlocks(blocks []colorBlock) []colorBlock {
	if len(blocks) <= 1 {
		return blocks
	}

	merged := true
	for merged {
		merged = false
		var result []colorBlock
		used := make([]bool, len(blocks))

		for i := 0; i < len(blocks); i++ {
			if used[i] {
				continue
			}
			mergedBlock := blocks[i]
			for j := i + 1; j < len(blocks); j++ {
				if used[j] {
					continue
				}
				// 检查是否重叠或相邻（间距 < 5px）
				if blocksOverlapOrAdjacent(mergedBlock, blocks[j], 5) &&
					colorMatch(mergedBlock.R, mergedBlock.G, mergedBlock.B,
						blocks[j].R, blocks[j].G, blocks[j].B, 0x3333) {
					mergedBlock = mergeTwoBlocks(mergedBlock, blocks[j])
					used[j] = true
					merged = true
				}
			}
			result = append(result, mergedBlock)
		}
		blocks = result
	}
	return blocks
}

func blocksOverlapOrAdjacent(a, b colorBlock, gap int) bool {
	// 检查两个矩形是否重叠或在 gap 像素范围内
	return a.X1-gap < b.X2 && a.X2+gap > b.X1 &&
		a.Y1-gap < b.Y2 && a.Y2+gap > b.Y1
}

func mergeTwoBlocks(a, b colorBlock) colorBlock {
	return colorBlock{
		X1: min(a.X1, b.X1),
		Y1: min(a.Y1, b.Y1),
		X2: max(a.X2, b.X2),
		Y2: max(a.Y2, b.Y2),
		R:  (a.R + b.R) / 2,
		G:  (a.G + b.G) / 2,
		B:  (a.B + b.B) / 2,
	}
}

// ── 图形检测（基础） ────────────────────────────────────────

// shapeInfo 检测到的图形信息。
type shapeInfo struct {
	Kind              string  // "圆形" / "椭圆" / "矩形" / "圆角矩形" / "三角形(3)" / "五边形(5)" / "多边形(N)" / "线条" / "曲线" / "折线" / "箭头" / "扇形" / "文字区域" / "坐标轴" / "图表类型"
	X1, Y1, X2, Y2    int     // 边界框
	CenterX, CenterY  int     // 中心点坐标
	Area              int     // 面积（像素²）
	Perimeter         float64 // 周长（像素）
	Circularity       float64 // 圆度（4π×面积/周长²，0~1，1=完美圆）
	VertexCount       int     // 顶点数（多边形有效）
	FillRatio         float64 // 填充率（面积/边界框面积）
	Width, Height     int     // 实际宽度和高度
	Angle             float64 // 旋转角度（度），对椭圆/矩形有效
	Description       string
}

// detectShapes 检测图片中的各种几何形状和图表元素。
// 集成多种检测算法：连通域分析 + 边界跟踪 + 形状分类 + 图表分析。
func detectShapes(img image.Image, bounds image.Rectangle, w, h int) []shapeInfo {
	var allShapes []shapeInfo

	// 1. 几何形状检测（圆形、椭圆、矩形、圆角矩形、多边形、扇形等）
	geoShapes := detectGeometricShapes(img, bounds, w, h)
	allShapes = append(allShapes, geoShapes...)

	// 2. 曲线/折线/箭头检测（开放路径类）
	curveShapes := detectCurvesAndArrows(img, bounds, w, h)
	allShapes = append(allShapes, curveShapes...)

	// 3. 图表元素检测（坐标轴、柱状图、饼图、折线图、图例等）
	chartShapes := detectChartElements(img, bounds, w, h, allShapes)
	allShapes = append(allShapes, chartShapes...)

	// 去重（过滤坐标高度重叠的相似形状）
	allShapes = deduplicateShapes(allShapes)

	// 排序：图表类型 > 几何形状 > 线条曲线 > 图例
	sort.Slice(allShapes, func(i, j int) bool {
		order := map[string]int{
			"图表类型": 0, "坐标轴": 1, "柱状图柱体": 2,
			"圆形": 10, "椭圆": 11, "矩形": 12, "圆角矩形": 13,
			"三角形": 14, "四边形": 15, "五边形": 16, "六边形": 17, "八边形": 18,
			"扇形": 20, "线条": 30, "曲线/折线": 31, "箭头": 32,
			"图例区域": 40, "图例标记": 41,
		}
		oi, okI := order[allShapes[i].Kind]
		oj, okJ := order[allShapes[j].Kind]
		if !okI {
			oi = 99
		}
		if !okJ {
			oj = 99
		}
		if oi != oj {
			return oi < oj
		}
		// 同类按面积降序
		return allShapes[i].Area > allShapes[j].Area
	})

	// 限制返回数量
	if len(allShapes) > 40 {
		allShapes = allShapes[:40]
	}

	return allShapes
}

// deduplicateShapes 去重：移除坐标高度重叠且类型相同的形状。
func deduplicateShapes(shapes []shapeInfo) []shapeInfo {
	if len(shapes) <= 1 {
		return shapes
	}

	var result []shapeInfo
	used := make([]bool, len(shapes))

	for i := 0; i < len(shapes); i++ {
		if used[i] {
			continue
		}
		merged := shapes[i]
		for j := i + 1; j < len(shapes); j++ {
			if used[j] {
				continue
			}
			// 如果类型相同且边界框重叠 > 60%，合并
			if shapes[i].Kind == shapes[j].Kind {
				overlap := intersectionArea(
					shapes[i].X1, shapes[i].Y1, shapes[i].X2, shapes[i].Y2,
					shapes[j].X1, shapes[j].Y1, shapes[j].X2, shapes[j].Y2)
				areaI := (shapes[i].X2 - shapes[i].X1) * (shapes[i].Y2 - shapes[i].Y1)
				areaJ := (shapes[j].X2 - shapes[j].X1) * (shapes[j].Y2 - shapes[j].Y1)
				maxArea := areaI
				if areaJ > maxArea {
					maxArea = areaJ
				}
				if maxArea > 0 && overlap*100/maxArea > 60 {
					// 合并
					if shapes[j].Area > merged.Area {
						merged = shapes[j]
					}
					used[j] = true
				}
			}
		}
		result = append(result, merged)
	}
	return result
}

// intersectionArea 计算两个矩形的交集面积。
func intersectionArea(x1, y1, x2, y2, x3, y3, x4, y4 int) int {
	ix1 := max(x1, x3)
	iy1 := max(y1, y3)
	ix2 := min(x2, x4)
	iy2 := min(y2, y4)
	if ix1 < ix2 && iy1 < iy2 {
		return (ix2 - ix1) * (iy2 - iy1)
	}
	return 0
}

// ── 形状过滤辅助 ──────────────────────────────────────────

// filterShapesByKind 按类型过滤形状。
func filterShapesByKind(shapes []shapeInfo, kind string) []shapeInfo {
	var result []shapeInfo
	for _, s := range shapes {
		if s.Kind == kind {
			result = append(result, s)
		}
	}
	return result
}

// filterShapesByKindPrefix 按类型前缀过滤形状。
func filterShapesByKindPrefix(shapes []shapeInfo, prefix string) []shapeInfo {
	var result []shapeInfo
	for _, s := range shapes {
		if strings.HasPrefix(s.Kind, prefix) {
			result = append(result, s)
		}
	}
	return result
}

// filterShapesByKindList 按类型列表过滤形状。
func filterShapesByKindList(shapes []shapeInfo, kinds []string) []shapeInfo {
	kindSet := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	var result []shapeInfo
	for _, s := range shapes {
		if kindSet[s.Kind] {
			result = append(result, s)
		}
	}
	return result
}

// filterOtherShapes 返回不在任何指定分组中的形状。
func filterOtherShapes(shapes []shapeInfo, groups ...[]shapeInfo) []shapeInfo {
	// 构建已使用的形状索引集合
	type shapeKey struct {
		kind string
		x1, y1, x2, y2 int
	}
	used := make(map[shapeKey]bool)
	for _, group := range groups {
		for _, s := range group {
			key := shapeKey{s.Kind, s.X1, s.Y1, s.X2, s.Y2}
			used[key] = true
		}
	}
	var result []shapeInfo
	for _, s := range shapes {
		key := shapeKey{s.Kind, s.X1, s.Y1, s.X2, s.Y2}
		if !used[key] {
			result = append(result, s)
		}
	}
	return result
}

// ── OCR ────────────────────────────────────────────────────

func ocrImage(root, path, lang string, detail bool) (string, error) {

	// 1. 检查 Tesseract 是否可用
	tesseractPath := findTesseract(root)
	if tesseractPath == "" {
		return "", fmt.Errorf("未检测到 Tesseract OCR。请运行下载脚本自动获取便携版：\n" +
			"  项目根目录运行: powershell -ExecutionPolicy Bypass -File download_tesseract.ps1\n" +
			"  下载后自动部署到 bin/tesseract/ 目录，agent 将自动识别使用。")
	}

	// 2a. 尝试查找内嵌的 tessdata 目录（便携版）
	tesseractDir := filepath.Dir(tesseractPath)
	tessdataDir := filepath.Join(tesseractDir, "tessdata")
	if _, err := os.Stat(tessdataDir); os.IsNotExist(err) {
		tessdataDir = ""
	}

	// 2b. 检查图片文件
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("无法访问图片 %q: %w", path, err)
	}
	if fi.Size() == 0 {
		return "", fmt.Errorf("图片文件为空: %s", path)
	}

	// 3. 检查语言包可用性
	langAvailable := checkTesseractLang(tesseractPath, tessdataDir, lang)
	if !langAvailable {
		// 尝试只用 eng
		engAvailable := checkTesseractLang(tesseractPath, tessdataDir, "eng")
		if engAvailable {
			lang = "eng"
		} else {
			// 默认识别（不指定语言）
			lang = ""
		}
	}

	// 4. 调用 Tesseract 输出 TSV 格式（含坐标）
	// tesseract --tessdata-dir path image.png stdout tsv -l chi_sim+eng
	// 注意：--tessdata-dir 必须在位置参数之前
	args := []string{}
	if tessdataDir != "" {
		args = append(args, "--tessdata-dir", tessdataDir)
	}
	args = append(args, path, "stdout", "tsv")
	if lang != "" {
		args = append(args, "-l", lang)
	}

	cmd := exec.Command(tesseractPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("Tesseract 执行失败: %s\nstderr: %s",
				err.Error(), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("Tesseract 执行失败: %w", err)
	}

	if len(output) == 0 {
		return "OCR 未识别到任何文字（图片可能不包含文字或文字不清晰）", nil
	}

	// 5. 解析 TSV 输出
	texts := parseTesseractTSV(string(output))

	if len(texts) == 0 {
		return "OCR 未识别到任何文字（图片可能不包含文字或文字不清晰）", nil
	}

	// 6. 构建输出
	var b strings.Builder
	fileName := filepath.Base(path)
	fmt.Fprintf(&b, "## OCR 识别结果\n\n")
	fmt.Fprintf(&b, "| 属性 | 值 |\n|------|-----|\n")
	fmt.Fprintf(&b, "| 文件 | %s |\n", fileName)
	fmt.Fprintf(&b, "| 语言 | %s |\n", lang)
	fmt.Fprintf(&b, "| 识别文字数 | %d |\n", len(texts))
	b.WriteString("\n")

	if detail {
		b.WriteString("━━━ 文字详情（含坐标）━━━\n\n")
		b.WriteString("| 序号 | 坐标块 | 文字内容 |\n")
		b.WriteString("|------|--------|----------|\n")
		for i, t := range texts {
			// 转义 | 避免表格错乱
			content := strings.ReplaceAll(t.Text, "|", "｜")
			content = strings.ReplaceAll(content, "\n", " ")
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Fprintf(&b, "| %d | `(%d,%d)-(%d,%d)` | %s |\n",
				i+1, t.X1, t.Y1, t.X2, t.Y2, content)
		}
		b.WriteString("\n")
	} else {
		// 仅文字，不带坐标
		b.WriteString("━━━ 识别文字 ━━━\n\n")
		for _, t := range texts {
			b.WriteString(t.Text)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// 汇总统计
	b.WriteString("━━━ 统计 ━━━\n\n")
	totalChars := 0
	for _, t := range texts {
		totalChars += len([]rune(t.Text))
	}
	fmt.Fprintf(&b, "- 文字块数: %d\n", len(texts))
	fmt.Fprintf(&b, "- 总字符数: %d\n", totalChars)

	return b.String(), nil
}

// ocrText OCR 识别出的文字块（含坐标）。
type ocrText struct {
	Text     string
	X1, Y1, X2, Y2 int
	Conf     int // 置信度 0-100
}

// parseTesseractTSV 解析 Tesseract 的 TSV 输出格式。
func parseTesseractTSV(tsv string) []ocrText {
	lines := strings.Split(tsv, "\n")
	if len(lines) < 2 {
		return nil
	}

	// 解析表头
	header := lines[0]
	cols := strings.Split(header, "\t")
	colMap := make(map[string]int)
	for i, col := range cols {
		colMap[strings.TrimSpace(col)] = i
	}

	// 需要的列索引
	levelIdx := colMap["level"]
	textIdx := colMap["text"]
	confIdx := colMap["conf"]
	xIdx := colMap["left"]
	yIdx := colMap["top"]
	wIdx := colMap["width"]
	hIdx := colMap["height"]
	if textIdx == 0 && confIdx == 0 {
		// 尝试备选列名
		textIdx = colMap["text"]
		xIdx = colMap["left"]
		yIdx = colMap["top"]
		wIdx = colMap["width"]
		hIdx = colMap["height"]
	}
	if textIdx == 0 {
		return nil // 无法解析
	}

	// 逐行解析，只取 level=5（文字行）的条目
	type rawText struct {
		text     string
		conf     int
		x, y, w, h int
	}

	var rawTexts []rawText
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) <= textIdx {
			continue
		}

		// 检查 level
		if levelIdx < len(fields) {
			lv := strings.TrimSpace(fields[levelIdx])
			if lv != "5" { // 5=文字行
				continue
			}
		}

		text := ""
		if textIdx < len(fields) {
			text = strings.TrimSpace(fields[textIdx])
		}
		if text == "" {
			continue
		}

		conf := 0
		if confIdx < len(fields) {
			confStr := strings.TrimSpace(fields[confIdx])
			if confStr != "-1" {
				conf, _ = strconv.Atoi(confStr)
			}
		}

		x, y, w, h := 0, 0, 0, 0
		if xIdx < len(fields) {
			x, _ = strconv.Atoi(strings.TrimSpace(fields[xIdx]))
		}
		if yIdx < len(fields) {
			y, _ = strconv.Atoi(strings.TrimSpace(fields[yIdx]))
		}
		if wIdx < len(fields) {
			w, _ = strconv.Atoi(strings.TrimSpace(fields[wIdx]))
		}
		if hIdx < len(fields) {
			h, _ = strconv.Atoi(strings.TrimSpace(fields[hIdx]))
		}

		if w <= 0 || h <= 0 {
			continue
		}

		// 同个 block 内合并
		rawTexts = append(rawTexts, rawText{
			text: text,
			conf: conf,
			x:    x, y: y, w: w, h: h,
		})
	}

	// 按行合并：同一水平区域的文字行合并为一段
	if len(rawTexts) == 0 {
		return nil
	}

	// 按 y 坐标分组（同一行的文字合并）
	type textLine struct {
		texts    []rawText
		minY, maxY int
		minX, maxX int
	}
	var lines2 []textLine

	sort.Slice(rawTexts, func(i, j int) bool {
		if rawTexts[i].y != rawTexts[j].y {
			return rawTexts[i].y < rawTexts[j].y
		}
		return rawTexts[i].x < rawTexts[j].x
	})

	for _, rt := range rawTexts {
		placed := false
		for j := range lines2 {
			// 同一行：垂直重叠超过 50%
			overlap := min(rt.y+rt.h, lines2[j].maxY) - max(rt.y, lines2[j].minY)
			if overlap > 0 && overlap*2 > min(rt.h, lines2[j].maxY-lines2[j].minY) {
				lines2[j].texts = append(lines2[j].texts, rt)
				lines2[j].minY = min(lines2[j].minY, rt.y)
				lines2[j].maxY = max(lines2[j].maxY, rt.y+rt.h)
				lines2[j].minX = min(lines2[j].minX, rt.x)
				lines2[j].maxX = max(lines2[j].maxX, rt.x+rt.w)
				placed = true
				break
			}
		}
		if !placed {
			lines2 = append(lines2, textLine{
				texts: []rawText{rt},
				minY:  rt.y,
				maxY:  rt.y + rt.h,
				minX:  rt.x,
				maxX:  rt.x + rt.w,
			})
		}
	}

	// 每行合并文字
	var results []ocrText
	for _, line := range lines2 {
		sort.Slice(line.texts, func(i, j int) bool {
			return line.texts[i].x < line.texts[j].x
		})
		var textParts []string
		for _, t := range line.texts {
			textParts = append(textParts, t.text)
		}
		text := strings.Join(textParts, " ")
		// 过滤纯数字/短内容
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		results = append(results, ocrText{
			Text: text,
			X1:   line.minX,
			Y1:   line.minY,
			X2:   line.maxX,
			Y2:   line.maxY,
			Conf: line.texts[0].conf,
		})
	}

	return results
}


// findTesseract 查找系统中的 Tesseract 可执行文件。
// 优先级:
//
//	 1) 可执行文件同目录 bin/tesseract/tesseract.exe（发布包模式，所有工具在 bin/ 下）
//	 2) 项目根目录 bin/tesseract/tesseract.exe（开发模式）
//	 3) 可执行文件同目录 tesseract/tesseract.exe（旧路径，向后兼容）
//	 4) 项目根目录 tesseract/tesseract.exe（旧路径，向后兼容）
//	 5) 系统 PATH
//	 6) Windows 常见安装路径
func findTesseract(root string) string {
	// 1. 可执行文件同目录下的 bin/tesseract/（发布包模式，工具统一归到 bin/）
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		embedded := filepath.Join(exeDir, "bin", "tesseract", "tesseract.exe")
		if _, err := os.Stat(embedded); err == nil {
			return embedded
		}
	}

	// 2. 项目根目录下的 bin/tesseract/（开发模式）
	if root != "" {
		project := filepath.Join(root, "bin", "tesseract", "tesseract.exe")
		if _, err := os.Stat(project); err == nil {
			return project
		}
	}

	// 3. 可执行文件同目录下的旧路径 tesseract/（向后兼容）
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		old := filepath.Join(exeDir, "tesseract", "tesseract.exe")
		if _, err := os.Stat(old); err == nil {
			return old
		}
	}

	// 4. 项目根目录下的旧路径 tesseract/（向后兼容）
	if root != "" {
		old := filepath.Join(root, "tesseract", "tesseract.exe")
		if _, err := os.Stat(old); err == nil {
			return old
		}
	}

	// 5. 系统 PATH
	if path, err := exec.LookPath("tesseract"); err == nil {
		return path
	}
	if path, err := exec.LookPath("tesseract.exe"); err == nil {
		return path
	}

	// 6. Windows 常见安装路径
	candidates := []string{
		`C:\Program Files\Tesseract-OCR\tesseract.exe`,
		`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}


// checkTesseractLang 检查 Tesseract 是否支持指定语言。
func checkTesseractLang(tesseractPath, tessdataDir, lang string) bool {
	if lang == "" {
		return true
	}
	args := []string{}
	if tessdataDir != "" {
		args = append(args, "--tessdata-dir", tessdataDir)
	}
	args = append(args, "--list-langs")
	cmd := exec.Command(tesseractPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	outputStr := string(output)
	langs := strings.Split(lang, "+")
	for _, l := range langs {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if !strings.Contains(outputStr, l) {
			return false
		}
	}
	return true
}

// ── 辅助函数 ──────────────────────────────────────────────

// colorBar 生成一个颜色占比条形图（用 Unicode 方块字符）。
func colorBar(pct float64, width int) string {
	full := int(pct / 100.0 * float64(width))
	if full > width {
		full = width
	}
	return strings.Repeat("█", full) + strings.Repeat("░", width-full)
}

// dominantColorDesc 返回主色调的文字描述。
func dominantColorDesc(colors []quantizedColor) string {
	if len(colors) == 0 {
		return "无"
	}
	return fmt.Sprintf("%s（%.1f%%）", colors[0].Name,
		float64(colors[0].Count)/float64(colors[0].Count+colors[0].Count)*100)
}


// describeColorRegion 根据位置和大小时色块区域进行语义描述。
func describeColorRegion(blk colorBlock, imgW, imgH int) string {
	cx := (blk.X1 + blk.X2) / 2
	cy := (blk.Y1 + blk.Y2) / 2
	w := blk.X2 - blk.X1
	h := blk.Y2 - blk.Y1

	// 位置判断
	pos := ""
	if cy < imgH/6 {
		pos = "顶部"
	} else if cy < imgH/3 {
		pos = "中上"
	} else if cy < imgH*2/3 {
		pos = "中部"
	} else if cy < imgH*5/6 {
		pos = "中下"
	} else {
		pos = "底部"
	}

	if cx < imgW/6 {
		pos = "左侧" + pos
	} else if cx < imgW*5/6 {
		pos = "中间" + pos
	} else {
		pos = "右侧" + pos
	}

	// 形状判断
	shape := "区域"
	ratio := float64(w) / float64(h)
	if ratio > 5 && h < 30 {
		shape = "横条/分割线"
	} else if ratio > 3 {
		shape = "横幅区域"
	} else if ratio < 0.2 && w < 30 {
		shape = "竖条/侧边栏"
	} else if ratio < 0.33 {
		shape = "竖幅区域"
	} else if ratio > 0.8 && ratio < 1.2 {
		shape = "方形区域"
	}

	return fmt.Sprintf("%s %s（%dx%d）", pos, shape, w, h)
}

// countBlocksByPosition 按位置分类统计色块。

// ── 颜色命名辅助 ──────────────────────────────────────────

// colorName 返回常见颜色的中文名称。
func colorName(r, g, b uint32) string {
	// 将 8bit 转为 0-1 范围判断
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	// 灰度检测
	if math.Abs(rf-gf) < 0.05 && math.Abs(gf-bf) < 0.05 {
		if rf > 0.9 {
			return "白色"
		}
		if rf > 0.7 {
			return "浅灰"
		}
		if rf > 0.4 {
			return "中灰"
		}
		if rf > 0.15 {
			return "深灰"
		}
		return "黑色"
	}

	// 基本颜色
	maxC := maxOf(rf, gf, bf)
	minC := minOf(rf, gf, bf)
	delta := maxC - minC

	// 饱和度低 => 灰色系
	if delta < 0.1 {
		grayVal := int((rf + gf + bf) / 3 * 255)
		if grayVal > 200 {
			return "白色"
		}
		if grayVal > 128 {
			return "浅灰"
		}
		if grayVal > 64 {
			return "深灰"
		}
		return "黑色"
	}

	// 识别主色
	switch {
	case rf >= gf && rf >= bf:
		if gf < 0.3 && bf < 0.3 {
			return "红色"
		}
		if gf > 0.7 && bf < 0.4 {
			return "黄色"
		}
		if gf > 0.5 && bf < 0.3 {
			return "橙色"
		}
		if gf > 0.7 && bf > 0.7 {
			return "浅粉"
		}
		return "红色偏" + dominantHue(rf, gf, bf)

	case gf >= rf && gf >= bf:
		if rf < 0.4 && bf < 0.4 {
			return "绿色"
		}
		if bf > 0.5 && rf > 0.3 {
			return "青色"
		}
		if bf < 0.3 && rf > 0.4 {
			return "黄绿"
		}
		return "绿色偏" + dominantHue(rf, gf, bf)

	default: // bf 最大
		if rf < 0.3 && gf < 0.3 {
			return "蓝色"
		}
		if rf > 0.4 && gf > 0.3 {
			return "紫色"
		}
		if gf > 0.4 && rf < 0.3 {
			return "青色"
		}
		return "蓝色偏" + dominantHue(rf, gf, bf)
	}
}

func maxOf(a, b, c float64) float64 {
	if a >= b && a >= c {
		return a
	}
	if b >= a && b >= c {
		return b
	}
	return c
}

func minOf(a, b, c float64) float64 {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}

func dominantHue(r, g, b float64) string {
	// 简单判断次色调
	if r >= g && r >= b {
		if g > b {
			return "黄"
		}
		return "紫"
	}
	if g >= r && g >= b {
		if r > b {
			return "黄"
		}
		return "青"
	}
	if r > g {
		return "紫"
	}
	return "青"
}

// ── 增强图形检测（几何形状分析） ───────────────────────────

// detectedRegion 连通区域分析的结果。
type detectedRegion struct {
	Pixels   [][2]int // 区域内的所有像素坐标
	Boundary [][2]int // 边界像素坐标（顺时针）
	MinX, MinY, MaxX, MaxY int
	R, G, B  uint32    // 主色（16 位）
	Area     int       // 像素数
}

// regionKey 用于区域标记的唯一标识。
type regionKey struct {
	x, y int
}

// detectGeometricShapes 检测图像中的各种几何形状（圆形/椭圆/矩形/多边形等）。
// 使用连通域分析 + 边界跟踪 + 形状分类。
func detectGeometricShapes(img image.Image, bounds image.Rectangle, w, h int) []shapeInfo {
	var shapes []shapeInfo

	step := 1
	if w*h > 500000 {
		step = 2
	}
	if w*h > 2000000 {
		step = 3
	}

	// 1. 连通域分析：用 flood fill 找出所有同色区域
	visited := make([][]bool, (h+step-1)/step)
	for i := range visited {
		visited[i] = make([]bool, (w+step-1)/step)
	}

	var regions []detectedRegion

	for y := 0; y < h; y += step {
		for x := 0; x < w; x += step {
			ix, iy := x/step, y/step
			if visited[iy][ix] {
				continue
			}
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a < 128 {
				visited[iy][ix] = true
				continue
			}

			// 开始 flood fill
			minX, minY, maxX, maxY := x, y, x+step, y+step
			var pixels [][2]int
			stack := [][2]int{{ix, iy}}
			visited[iy][ix] = true

			for len(stack) > 0 {
				cx, cy := stack[len(stack)-1][0], stack[len(stack)-1][1]
				stack = stack[:len(stack)-1]
				px := cx * step
				py := cy * step
				pixels = append(pixels, [2]int{px, py})
				if px < minX {
					minX = px
				}
				if py < minY {
					minY = py
				}
				if px+step > maxX {
					maxX = px + step
				}
				if py+step > maxY {
					maxY = py + step
				}

				dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
				for _, d := range dirs {
					nx, ny := cx+d[0], cy+d[1]
					if nx < 0 || ny < 0 || nx >= len(visited[0]) || ny >= len(visited) {
						continue
					}
					if visited[ny][nx] {
						continue
					}
					npx := nx * step
					npy := ny * step
					nr, ng, nb, na := img.At(bounds.Min.X+npx, bounds.Min.Y+npy).RGBA()
					if na < 128 {
						visited[ny][nx] = true
						continue
					}
					if !colorMatch(nr, ng, nb, r, g, b, 0x2222) {
						visited[ny][nx] = true
						continue
					}
					visited[ny][nx] = true
					stack = append(stack, [2]int{nx, ny})
				}
			}

			area := len(pixels) * step * step
			regionW := maxX - minX
			regionH := maxY - minY

			// 过滤太小（< 0.01% 图像面积）和太大（> 70% 图像面积，背景）的区域
			fullArea := w * h
			if area < fullArea/10000 || area > fullArea*70/100 {
				continue
			}
			if regionW < 5 && regionH < 5 {
				continue
			}

			regions = append(regions, detectedRegion{
				Pixels:   pixels,
				MinX:     minX, MinY: minY,
				MaxX:     maxX, MaxY: maxY,
				R: r, G: g, B: b,
				Area: area,
			})
		}
	}

	// 2. 对每个区域进行形状分析
	for _, reg := range regions {
		// 计算几何属性
		cx := (reg.MinX + reg.MaxX) / 2
		cy := (reg.MinY + reg.MaxY) / 2
		rw := reg.MaxX - reg.MinX
		rh := reg.MaxY - reg.MinY

		// 边界跟踪（只在需要时进行）
		boundary := traceBoundary(img, bounds, reg, step)

		// 计算周长
		perimeter := float64(len(boundary)) * float64(step)

		// 计算填充率 = 面积 / 边界框面积
		bboxArea := rw * rh
		fillRatio := float64(reg.Area) / float64(bboxArea)
		if bboxArea == 0 {
			fillRatio = 0
		}

		// 计算圆度 = 4π × 面积 / 周长²
		circularity := 0.0
		if perimeter > 0 {
			circularity = 4.0 * math.Pi * float64(reg.Area) / (perimeter * perimeter)
		}
		if circularity > 1.0 {
			circularity = 1.0
		}

		// 3. 形状分类
		kind := ""
		desc := ""
		vertexCount := 0
		angle := 0.0

		// 矩形检测：高填充率 + 矩形度
		if fillRatio > 0.80 {
			// 检查是否为圆角矩形（角部颜色渐变或缺失）
			roundedCorners := checkRoundedCorners(img, bounds, reg, step)
			if roundedCorners {
				kind = "圆角矩形"
				ratio := float64(rw) / float64(rh)
				desc = fmt.Sprintf("圆角矩形 %.0f×%.0f，圆角过渡", float64(rw), float64(rh))
				if ratio > 5 && rh < 30 {
					desc = fmt.Sprintf("圆角按钮/横条 %.0f×%.0f", float64(rw), float64(rh))
				}
			} else {
				kind = "矩形"
				ratio := float64(rw) / float64(rh)
				if ratio > 5 && rh < 30 {
					kind = "线条（粗）"
					desc = fmt.Sprintf("粗横线，长 %dpx，高 %dpx", rw, rh)
				} else if ratio < 0.2 && rw < 30 {
					kind = "线条（粗）"
					desc = fmt.Sprintf("粗竖线，宽 %dpx，高 %dpx", rw, rh)
				} else {
					desc = fmt.Sprintf("矩形 %.0f×%.0f，填充率 %.0f%%", float64(rw), float64(rh), fillRatio*100)
				}
			}
		} else if circularity > 0.75 && fillRatio > 0.6 {
			// 圆形/椭圆检测：高圆度 + 较高填充率
			if circularity > 0.85 {
				// 检查是否为正圆还是椭圆
				ratio := float64(rw) / float64(rh)
				if ratio > 1.2 || ratio < 0.8 {
					kind = "椭圆"
					angle = ellipseAngle(boundary, cx, cy)
					desc = fmt.Sprintf("椭圆 %.0f×%.0f，圆度 %.2f，倾角 %.0f°",
						float64(rw), float64(rh), circularity, angle)
				} else {
					kind = "圆形"
					radius := (float64(rw) + float64(rh)) / 4.0
					desc = fmt.Sprintf("圆形，半径约 %.0fpx，圆度 %.2f", radius, circularity)
				}
			} else {
				kind = "近似椭圆"
				desc = fmt.Sprintf("近似椭圆 %.0f×%.0f，圆度 %.2f", float64(rw), float64(rh), circularity)
			}
		} else if fillRatio < 0.80 && fillRatio > 0.30 && len(boundary) >= 6 {
			// 多边形或复杂形状：用 Douglas-Peucker 简化边界
			simplified := douglasPeucker(boundary, 2.0*float64(step))
			vertexCount = len(simplified)

			if vertexCount >= 3 && vertexCount <= 12 {
				kind = classifyPolygon(vertexCount, rw, rh)
				desc = fmt.Sprintf("%s，%d 个顶点，填充率 %.0f%%",
					kind, vertexCount, fillRatio*100)
			} else if vertexCount > 12 {
				// 检查是否为饼图扇形
				if checkSector(img, bounds, reg, boundary, step) {
					kind = "扇形"
					desc = fmt.Sprintf("扇形区域，圆心附近 (%d,%d)，角度约 %.0f°", cx, cy, estimateSectorAngle(boundary, cx, cy))
				} else {
					kind = fmt.Sprintf("复杂轮廓（%d边）", vertexCount)
					desc = fmt.Sprintf("复杂形状，边界简化后 %d 个顶点，面积 %dpx²", vertexCount, reg.Area)
				}
			}
		} else if fillRatio <= 0.30 && len(boundary) >= 4 {
			// 线条/折线/曲线
			isCurve, curveDesc := classifyLinearShape(boundary, step, w, h)
			if isCurve {
				kind = "折线/曲线"
				desc = curveDesc
			} else if len(boundary) >= 6 {
				kind = "轮廓线"
				desc = fmt.Sprintf("开放轮廓，边界长 %.0fpx", perimeter)
			}
		}

		if kind == "" {
			continue
		}

		shapes = append(shapes, shapeInfo{
			Kind:        kind,
			X1:          reg.MinX, Y1: reg.MinY,
			X2:          reg.MaxX, Y2: reg.MaxY,
			CenterX:     cx, CenterY: cy,
			Area:        reg.Area,
			Perimeter:   perimeter,
			Circularity: circularity,
			VertexCount: vertexCount,
			FillRatio:   fillRatio,
			Width:       rw, Height: rh,
			Angle:       angle,
			Description: desc,
		})
	}

	// 按面积排序（大的在前）
	sort.Slice(shapes, func(i, j int) bool {
		return shapes[i].Area > shapes[j].Area
	})

	if len(shapes) > 40 {
		shapes = shapes[:40]
	}

	return shapes
}

// traceBoundary 跟踪区域的边界像素（使用 Moore-Neighbor 边界跟踪算法）。
func traceBoundary(img image.Image, bounds image.Rectangle, reg detectedRegion, step int) [][2]int {
	if len(reg.Pixels) == 0 {
		return nil
	}

	// 构建边界点集：对于区域内的每个像素，检查是否有邻居不在区域内
	// 为提高效率，将像素坐标存入 map
	pixelSet := make(map[regionKey]bool)
	for _, p := range reg.Pixels {
		pixelSet[regionKey{p[0], p[1]}] = true
	}

	var boundary [][2]int
	for _, p := range reg.Pixels {
		px, py := p[0], p[1]
		// 检查4个方向是否有邻居不在区域内（即边界点）
		isBoundary := false
		neighbors := [][2]int{{px - step, py}, {px + step, py}, {px, py - step}, {px, py + step}}
		for _, n := range neighbors {
			if !pixelSet[regionKey{n[0], n[1]}] {
				isBoundary = true
				break
			}
		}
		if isBoundary {
			boundary = append(boundary, [2]int{px, py})
		}
	}

	// 将边界点按顺时针排序
	// 先计算中心
	if len(boundary) == 0 {
		return nil
	}
	cx := 0
	cy := 0
	for _, p := range boundary {
		cx += p[0]
		cy += p[1]
	}
	cx /= len(boundary)
	cy /= len(boundary)

	sort.Slice(boundary, func(i, j int) bool {
		angleI := math.Atan2(float64(boundary[i][1]-cy), float64(boundary[i][0]-cx))
		angleJ := math.Atan2(float64(boundary[j][1]-cy), float64(boundary[j][0]-cx))
		return angleI < angleJ
	})

	return boundary
}

// douglasPeucker 使用 Douglas-Peucker 算法简化折线（多边形边界简化）。
func douglasPeucker(points [][2]int, epsilon float64) [][2]int {
	if len(points) <= 2 {
		return points
	}

	// 找距离最远的点
	maxDist := 0.0
	maxIdx := 0
	start := points[0]
	end := points[len(points)-1]

	for i := 1; i < len(points)-1; i++ {
		dist := perpendicularDistance(points[i], start, end)
		if dist > maxDist {
			maxDist = dist
			maxIdx = i
		}
	}

	var result [][2]int
	if maxDist > epsilon {
		// 递归简化
		left := douglasPeucker(points[:maxIdx+1], epsilon)
		right := douglasPeucker(points[maxIdx:], epsilon)
		result = append(result, left[:len(left)-1]...)
		result = append(result, right...)
	} else {
		result = append(result, points[0], points[len(points)-1])
	}
	return result
}

// perpendicularDistance 计算点 p 到线段 (p1,p2) 的垂直距离。
func perpendicularDistance(p, p1, p2 [2]int) float64 {
	dx := float64(p2[0] - p1[0])
	dy := float64(p2[1] - p1[1])
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		// p1 == p2
		return math.Sqrt(float64((p[0]-p1[0])*(p[0]-p1[0]) + (p[1]-p1[1])*(p[1]-p1[1])))
	}
	t := ((float64(p[0]-p1[0])*dx + float64(p[1]-p1[1])*dy) / lengthSq)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	projX := float64(p1[0]) + t*dx
	projY := float64(p1[1]) + t*dy
	return math.Sqrt((float64(p[0])-projX)*(float64(p[0])-projX) + (float64(p[1])-projY)*(float64(p[1])-projY))
}

// classifyPolygon 根据顶点数返回多边形类型名称。
func classifyPolygon(vertexCount int, w, h int) string {
	switch vertexCount {
	case 3:
		return "三角形"
	case 4:
		// 检查是否为矩形（边长比例判断）
		ratio := float64(w) / float64(h)
		if ratio > 0.8 && ratio < 1.2 {
			return "菱形/四边形"
		}
		return "四边形"
	case 5:
		return "五边形"
	case 6:
		return "六边形"
	case 7:
		return "七边形"
	case 8:
		return "八边形"
	case 9:
		return "九边形"
	case 10:
		return "十边形"
	default:
		return fmt.Sprintf("多边形(%d)", vertexCount)
	}
}

// checkRoundedCorners 检查矩形是否具有圆角特征（粗略检测）。
func checkRoundedCorners(img image.Image, bounds image.Rectangle, reg detectedRegion, step int) bool {
	rw := reg.MaxX - reg.MinX
	rh := reg.MaxY - reg.MinY
	if rw < 10 || rh < 10 {
		return false
	}

	// 检查四个角区域（取 1/6 角区域）是否有大量非区域颜色的像素
	cornerSizeW := rw / 6
	cornerSizeH := rh / 6
	if cornerSizeW < 2 {
		cornerSizeW = 2
	}
	if cornerSizeH < 2 {
		cornerSizeH = 2
	}

	// 建立像素集合
	pixelSet := make(map[regionKey]bool)
	for _, p := range reg.Pixels {
		pixelSet[regionKey{p[0], p[1]}] = true
	}

	// 四个角区域
	corners := [][4]int{
		{reg.MinX, reg.MinY, reg.MinX + cornerSizeW, reg.MinY + cornerSizeH},                         // 左上
		{reg.MaxX - cornerSizeW, reg.MinY, reg.MaxX, reg.MinY + cornerSizeH},                         // 右上
		{reg.MinX, reg.MaxY - cornerSizeH, reg.MinX + cornerSizeW, reg.MaxY},                         // 左下
		{reg.MaxX - cornerSizeW, reg.MaxY - cornerSizeH, reg.MaxX, reg.MaxY},                         // 右下
	}

	missingPixels := 0
	totalCornerPixels := 0
	for _, c := range corners {
		for y := c[1]; y < c[3]; y += step {
			for x := c[0]; x < c[2]; x += step {
				totalCornerPixels++
				if !pixelSet[regionKey{x, y}] {
					missingPixels++
				}
			}
		}
	}

	if totalCornerPixels == 0 {
		return false
	}
	missingRatio := float64(missingPixels) / float64(totalCornerPixels)
	// 如果角部缺失超过 20%，认为可能是圆角
	return missingRatio > 0.20
}

// ellipseAngle 计算椭圆的主轴角度（度）。
func ellipseAngle(boundary [][2]int, cx, cy int) float64 {
	if len(boundary) < 6 {
		return 0
	}
	// 用惯性矩方法计算椭圆角度
	var sumXX, sumYY, sumXY float64
	for _, p := range boundary {
		dx := float64(p[0] - cx)
		dy := float64(p[1] - cy)
		sumXX += dx * dx
		sumYY += dy * dy
		sumXY += dx * dy
	}
	if sumXX == sumYY {
		return 45
	}
	angle := 0.5 * math.Atan2(2*sumXY, sumXX-sumYY)
	return angle * 180.0 / math.Pi
}

// classifyLinearShape 判断边界是否代表折线/曲线，并返回描述。
func classifyLinearShape(boundary [][2]int, step, w, h int) (bool, string) {
	if len(boundary) < 6 {
		return false, ""
	}

	// 简化边界
	simplified := douglasPeucker(boundary, 3.0*float64(step))
	if len(simplified) < 3 {
		return false, ""
	}

	// 计算曲率变化
	totalCurvature := 0.0
	segCount := 0
	for i := 1; i < len(simplified)-1; i++ {
		// 计算角度变化
		v1x := float64(simplified[i][0] - simplified[i-1][0])
		v1y := float64(simplified[i][1] - simplified[i-1][1])
		v2x := float64(simplified[i+1][0] - simplified[i][0])
		v2y := float64(simplified[i+1][1] - simplified[i][1])
		dot := v1x*v2x + v1y*v2y
		len1 := math.Sqrt(v1x*v1x + v1y*v1y)
		len2 := math.Sqrt(v2x*v2x + v2y*v2y)
		if len1 > 0 && len2 > 0 {
			cosAngle := dot / (len1 * len2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			totalCurvature += angle
			segCount++
		}
	}

	if segCount == 0 {
		return false, ""
	}

	avgCurvature := totalCurvature / float64(segCount)
	avgCurvatureDeg := avgCurvature * 180.0 / math.Pi

	// 如果平均曲率 > 15度，且不是一条直线，认为是曲线/折线
	length := 0.0
	for i := 1; i < len(simplified); i++ {
		dx := float64(simplified[i][0] - simplified[i-1][0])
		dy := float64(simplified[i][1] - simplified[i-1][1])
		length += math.Sqrt(dx*dx + dy*dy)
	}

	if avgCurvatureDeg > 15 && length > float64(max(w, h))/10 {
		if avgCurvatureDeg > 45 {
			return true, fmt.Sprintf("大曲率曲线，长 %.0fpx，平均弯曲角 %.0f°", length, avgCurvatureDeg)
		}
		return true, fmt.Sprintf("折线/曲线，%d 段线段，长 %.0fpx，平均弯曲角 %.0f°",
			len(simplified)-1, length, avgCurvatureDeg)
	}

	return false, ""
}

// checkSector 检查区域是否为扇形（饼图的一块）。
func checkSector(img image.Image, bounds image.Rectangle, reg detectedRegion, boundary [][2]int, step int) bool {
	if len(boundary) < 8 {
		return false
	}
	// 扇形判断标准：
	// 1. 区域大致呈三角形/扇形
	// 2. 有弧线边界
	// 3. 从中心点辐射

	// 粗略判断：边界中有一段弧线（曲率较大的一段）
	// 简化后如果有 4-8 个顶点，且有一段明显的弧线
	simplified := douglasPeucker(boundary, 3.0*float64(step))
	if len(simplified) < 4 || len(simplified) > 10 {
		return false
	}

	// 检查是否有相邻边夹角接近 0（弧线）和接近直角（辐射边）
	sharpCount := 0
	obtuseCount := 0
	for i := 0; i < len(simplified); i++ {
		prev := simplified[(i+len(simplified)-1)%len(simplified)]
		curr := simplified[i]
		next := simplified[(i+1)%len(simplified)]
		v1x := float64(curr[0] - prev[0])
		v1y := float64(curr[1] - prev[1])
		v2x := float64(next[0] - curr[0])
		v2y := float64(next[1] - curr[1])
		dot := v1x*v2x + v1y*v2y
		len1 := math.Sqrt(v1x*v1x + v1y*v1y)
		len2 := math.Sqrt(v2x*v2x + v2y*v2y)
		if len1 > 0 && len2 > 0 {
			cosAngle := dot / (len1 * len2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle) * 180.0 / math.Pi
			if angle < 30 {
				sharpCount++
			} else if angle > 120 {
				obtuseCount++
			}
		}
	}
	// 扇形通常有多个钝角（弧线段）和少量锐角（辐射线）
	return obtuseCount >= 2 && sharpCount >= 1
}

// estimateSectorAngle 估算扇形的张角（度）。
func estimateSectorAngle(boundary [][2]int, cx, cy int) float64 {
	if len(boundary) < 4 {
		return 0
	}
	// 简单估算：取边界上最远的点并计算覆盖角度范围
	// 这里简化处理，返回默认估值
	return 60.0
}

// ── 曲线/折线/箭头检测 ──────────────────────────────────────

// detectCurvesAndArrows 检测图像中的开放曲线、折线和箭头。
func detectCurvesAndArrows(img image.Image, bounds image.Rectangle, w, h int) []shapeInfo {
	var shapes []shapeInfo

	step := 1
	if w*h > 500000 {
		step = 2
	}
	if w*h > 2000000 {
		step = 3
	}

	// 检测非封闭的线条路径（从一端到另一端）
	// 策略：扫描找到细长的颜色连通区域，检查端点
	visited := make([][]bool, (h+step-1)/step)
	for i := range visited {
		visited[i] = make([]bool, (w+step-1)/step)
	}

	for y := 0; y < h; y += step {
		for x := 0; x < w; x += step {
			ix, iy := x/step, y/step
			if visited[iy][ix] {
				continue
			}
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a < 128 {
				visited[iy][ix] = true
				continue
			}

			// flood fill 收集连通域
			var pixels [][2]int
			minX, minY, maxX, maxY := x, y, x+step, y+step
			stack := [][2]int{{ix, iy}}
			visited[iy][ix] = true

			for len(stack) > 0 {
				cx2, cy2 := stack[len(stack)-1][0], stack[len(stack)-1][1]
				stack = stack[:len(stack)-1]
				px := cx2 * step
				py := cy2 * step
				pixels = append(pixels, [2]int{px, py})
				if px < minX {
					minX = px
				}
				if py < minY {
					minY = py
				}
				if px+step > maxX {
					maxX = px + step
				}
				if py+step > maxY {
					maxY = py + step
				}
				dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
				for _, d := range dirs {
					nx, ny := cx2+d[0], cy2+d[1]
					if nx < 0 || ny < 0 || nx >= len(visited[0]) || ny >= len(visited) {
						continue
					}
					if visited[ny][nx] {
						continue
					}
					nr, ng, nb, na := img.At(bounds.Min.X+nx*step, bounds.Min.Y+ny*step).RGBA()
					if na < 128 {
						visited[ny][nx] = true
						continue
					}
					if !colorMatch(nr, ng, nb, r, g, b, 0x3333) {
						continue
					}
					visited[ny][nx] = true
					stack = append(stack, [2]int{nx, ny})
				}
			}

			area := len(pixels) * step * step
			rw := maxX - minX
			rh := maxY - minY

			// 太小的跳过
			if area < w*h/10000 || (rw < 3 && rh < 3) {
				continue
			}

			// 判断是否为细长形状（线条类）
			bboxArea := rw * rh
			fillRatio := float64(area) / float64(bboxArea)
			if bboxArea == 0 {
				fillRatio = 0
			}

			// 线条特征：面积占比低（细长）
			if fillRatio > 0.50 {
				continue // 太粗，不是线条，由其他检测处理
			}

			cx := (minX + maxX) / 2
			cy2 := (minY + maxY) / 2

			// 找端点（邻居最少的边界点）
			pixelSet := make(map[regionKey]bool)
			for _, p := range pixels {
				pixelSet[regionKey{p[0], p[1]}] = true
			}

			var endpoints [][2]int
			for _, p := range pixels {
				neighborCount := 0
				neighbors := [][2]int{
					{p[0] - step, p[1]}, {p[0] + step, p[1]},
					{p[0], p[1] - step}, {p[0], p[1] + step},
				}
				for _, n := range neighbors {
					if pixelSet[regionKey{n[0], n[1]}] {
						neighborCount++
					}
				}
				// 端点的邻居数 = 1（线条末端）
				if neighborCount == 1 {
					endpoints = append(endpoints, p)
				}
			}

			// 如果有两个端点，可能是开放曲线/折线
			if len(endpoints) >= 2 {
				// 获取路径长度（沿边界走）
				// 粗略计算：像素数 * step
				length := float64(len(pixels)) * float64(step)

				// 检查是否为箭头
				arrowKind, arrowDesc := detectArrowHead(pixels, endpoints, pixelSet, step)
				if arrowKind != "" {
					shapes = append(shapes, shapeInfo{
						Kind:    arrowKind,
						X1: minX, Y1: minY,
						X2: maxX, Y2: maxY,
						CenterX:     cx, CenterY: cy2,
						Area:        area,
						Description: arrowDesc,
						Width: rw, Height: rh,
					})
					continue
				}

				// 判断直线还是曲线
				if isStraightLine(pixels, step) {
					shapes = append(shapes, shapeInfo{
						Kind: "线条",
						X1:   minX, Y1: minY,
						X2: maxX, Y2: maxY,
						CenterX:     cx, CenterY: cy2,
						Area:        area,
						Width: rw, Height: rh,
						Description: fmt.Sprintf("斜线，长 %.0fpx", length),
					})
				} else {
					// 分析曲率
					avgCurvature := estimateCurvature(pixels, step)
					curvDesc := ""
					if avgCurvature > 30 {
						curvDesc = fmt.Sprintf("大曲率曲线，长 %.0fpx", length)
					} else {
						curvDesc = fmt.Sprintf("折线，长 %.0fpx", length)
					}
					shapes = append(shapes, shapeInfo{
						Kind: "曲线/折线",
						X1:   minX, Y1: minY,
						X2: maxX, Y2: maxY,
						CenterX:     cx, CenterY: cy2,
						Area:        area,
						Width: rw, Height: rh,
						Description: curvDesc,
					})
				}
			}
		}
	}

	if len(shapes) > 20 {
		shapes = shapes[:20]
	}

	return shapes
}

// detectArrowHead 检测线条末端的箭头形状。
func detectArrowHead(pixels [][2]int, endpoints [][2]int, pixelSet map[regionKey]bool, step int) (string, string) {
	if len(endpoints) < 2 || len(pixels) < 10 {
		return "", ""
	}

	// 对于每个端点，检查周围是否有三角形区域
	for _, ep := range endpoints {
		// 检查端点周围 10*step 范围内是否有三角形色块
		searchR := 10 * step
		trianglePoints := 0
		totalChecked := 0

		for dy := -searchR; dy <= searchR; dy += step {
			for dx := -searchR; dx <= searchR; dx += step {
				dist := math.Sqrt(float64(dx*dx + dy*dy))
				if dist > float64(searchR) {
					continue
				}
				totalChecked++
				if pixelSet[regionKey{ep[0] + dx, ep[1] + dy}] {
					trianglePoints++
				}
			}
		}

		if totalChecked == 0 {
			continue
		}
		fillPct := float64(trianglePoints) / float64(totalChecked) * 100

		// 在端点附近有密集填充区域（三角形箭头区域）
		if fillPct > 30 && fillPct < 80 {
			return "箭头", fmt.Sprintf("箭头，端点 (%d,%d)", ep[0], ep[1])
		}
	}

	return "", ""
}

// isStraightLine 判断像素集合是否近似直线。
func isStraightLine(pixels [][2]int, step int) bool {
	if len(pixels) < 4 {
		return true
	}
	// 计算所有像素到最佳拟合直线的平均距离
	minX, minY := pixels[0][0], pixels[0][1]
	maxX, maxY := pixels[0][0], pixels[0][1]
	for _, p := range pixels {
		if p[0] < minX {
			minX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}

	// 如果边界框的宽高比很大或很小，近似直线
	rw := maxX - minX
	rh := maxY - minY
	if rw == 0 || rh == 0 {
		return true
	}
	ratio := float64(rw) / float64(rh)
	// 水平或垂直走向的线条，且填充率低
	return ratio > 5 || ratio < 0.2
}

// estimateCurvature 估算像素路径的平均曲率（度）。
func estimateCurvature(pixels [][2]int, step int) float64 {
	if len(pixels) < 6 {
		return 0
	}
	// 采样点
	sampled := pixels
	if len(sampled) > 50 {
		sampled = sampled[:50]
	}

	totalAngle := 0.0
	count := 0
	for i := 1; i < len(sampled)-1; i++ {
		v1x := float64(sampled[i][0] - sampled[i-1][0])
		v1y := float64(sampled[i][1] - sampled[i-1][1])
		v2x := float64(sampled[i+1][0] - sampled[i][0])
		v2y := float64(sampled[i+1][1] - sampled[i][1])
		len1 := math.Sqrt(v1x*v1x + v1y*v1y)
		len2 := math.Sqrt(v2x*v2x + v2y*v2y)
		if len1 > 0 && len2 > 0 {
			dot := (v1x*v2x + v1y*v2y) / (len1 * len2)
			if dot > 1 {
				dot = 1
			}
			if dot < -1 {
				dot = -1
			}
			totalAngle += math.Acos(dot)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return totalAngle / float64(count) * 180.0 / math.Pi
}

// ── 图表元素检测 ───────────────────────────────────────────

// detectChartElements 检测图表相关元素（坐标轴、柱状图、饼图、折线图、图例等）。
func detectChartElements(img image.Image, bounds image.Rectangle, w, h int, shapes []shapeInfo) []shapeInfo {
	var chartShapes []shapeInfo

	// 1. 检测坐标轴：粗的水平线 + 粗的垂直线交叉
	axes := detectAxes(img, bounds, w, h)
	chartShapes = append(chartShapes, axes...)

	// 2. 检测柱状图：在底部对齐的矩形组
	bars := detectBars(img, bounds, w, h, shapes)
	chartShapes = append(chartShapes, bars...)

	// 3. 检测饼图：多个扇形区域构成
	pie := detectPieChart(shapes)
	if pie != nil {
		chartShapes = append(chartShapes, *pie)
	}

	// 4. 检测折线图：连接的数据点 + 折线
	lineChart := detectLineChart(shapes)
	if lineChart != nil {
		chartShapes = append(chartShapes, *lineChart)
	}

	// 5. 检测图例：颜色块 + 文字区域
	legends := detectLegends(img, bounds, w, h, shapes)
	chartShapes = append(chartShapes, legends...)

	// 6. 综合判断图表类型
	chartType := inferChartType(chartShapes)
	if chartType != "" {
		chartShapes = append(chartShapes, shapeInfo{
			Kind:        "图表类型",
			X1: 0, Y1: 0, X2: w, Y2: h,
			CenterX: w / 2, CenterY: h / 2,
			Description: chartType,
		})
	}

	return chartShapes
}

// detectAxes 检测坐标轴（交叉的粗线条）。
func detectAxes(img image.Image, bounds image.Rectangle, w, h int) []shapeInfo {
	var axes []shapeInfo

	step := 2
	if w*h > 1000000 {
		step = 3
	}

	// 检测粗水平线（坐标轴 X）
	// 查找图像底部区域的粗水平线条
	hLineThreshold := w / 10
	if hLineThreshold < 30 {
		hLineThreshold = 30
	}

	// 扫描底部的 20% 区域找粗横线
	bottomArea := h * 4 / 5
	for y := bottomArea; y < h; y += step {
		lineStart := -1
		for x := 0; x < w; x += step {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a >= 128 {
				// 检查颜色是否为深色（坐标轴通常是深色）
				isDark := r < 0x8000 && g < 0x8000 && b < 0x8000
				if isDark {
					if lineStart < 0 {
						lineStart = x
					}
				} else {
					if lineStart >= 0 && x-lineStart >= hLineThreshold {
						// 检查线条高度（粗度）
						lineHeight := 1
						for dy := 1; dy < 6 && y+dy < h; dy++ {
							sr, sg, sb, sa := img.At(bounds.Min.X+lineStart, bounds.Min.Y+y+dy).RGBA()
							if sa >= 128 && sr < 0x8000 && sg < 0x8000 && sb < 0x8000 {
								lineHeight++
							} else {
								break
							}
						}
						if lineHeight >= 2 {
							axes = append(axes, shapeInfo{
								Kind: "坐标轴",
								X1:   lineStart, Y1: y,
								X2: x, Y2: y + lineHeight,
								Description: fmt.Sprintf("X 轴（水平），长 %dpx，粗 %dpx", x-lineStart, lineHeight),
							})
						}
					}
					lineStart = -1
				}
			} else {
				lineStart = -1
			}
		}
	}

	// 扫描左侧 30% 区域找粗竖线（坐标轴 Y）
	leftArea := w * 3 / 10
	for x := 0; x < leftArea; x += step {
		lineStart := -1
		for y := 0; y < h; y += step {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a >= 128 {
				isDark := r < 0x8000 && g < 0x8000 && b < 0x8000
				if isDark {
					if lineStart < 0 {
						lineStart = y
					}
				} else {
					if lineStart >= 0 && y-lineStart >= h/10 {
						lineWidth := 1
						for dx := 1; dx < 6 && x+dx < w; dx++ {
							sr, sg, sb, sa := img.At(bounds.Min.X+x+dx, bounds.Min.Y+lineStart).RGBA()
							if sa >= 128 && sr < 0x8000 && sg < 0x8000 && sb < 0x8000 {
								lineWidth++
							} else {
								break
							}
						}
						if lineWidth >= 2 {
							axes = append(axes, shapeInfo{
								Kind: "坐标轴",
								X1:   x, Y1: lineStart,
								X2: x + lineWidth, Y2: y,
								Description: fmt.Sprintf("Y 轴（垂直），长 %dpx，粗 %dpx", y-lineStart, lineWidth),
							})
						}
					}
					lineStart = -1
				}
			} else {
				lineStart = -1
			}
		}
	}

	return axes
}

// detectBars 检测柱状图的柱子（底部对齐的矩形组）。
func detectBars(img image.Image, bounds image.Rectangle, w, h int, shapes []shapeInfo) []shapeInfo {
	var bars []shapeInfo

	// 从已检测的形状中找垂直矩形
	for _, s := range shapes {
		if s.Kind == "矩形" || s.Kind == "圆角矩形" {
			if s.Height > s.Width && s.Height > 20 && s.Width > 5 {
				// 检查是否位于图像的下半部分（柱状图底部对齐）
				if float64(s.Y2) > float64(h)*0.5 {
					bars = append(bars, shapeInfo{
						Kind: "柱状图柱体",
						X1:   s.X1, Y1: s.Y1,
						X2: s.X2, Y2: s.Y2,
						CenterX:     s.CenterX, CenterY: s.CenterY,
						Width: s.Width, Height: s.Height,
						Area:        s.Area,
						Description: fmt.Sprintf("柱体 %d×%d，位置 (%d,%d)-(%d,%d)", s.Width, s.Height, s.X1, s.Y1, s.X2, s.Y2),
					})
				}
			}
		}
	}

	// 如果检测到多个柱体，标注为柱状图
	if len(bars) >= 3 {
		// 计算柱体间距是否均匀
		sort.Slice(bars, func(i, j int) bool {
			return bars[i].X1 < bars[j].X1
		})

		gaps := make([]int, 0, len(bars)-1)
		for i := 1; i < len(bars); i++ {
			gap := bars[i].X1 - bars[i-1].X2
			if gap > 0 {
				gaps = append(gaps, gap)
			}
		}

		if len(gaps) > 0 {
			avgGap := 0
			for _, g := range gaps {
				avgGap += g
			}
			avgGap /= len(gaps)
			bars = append([]shapeInfo{{
				Kind: "图表类型",
				X1: 0, Y1: 0, X2: w, Y2: h,
				CenterX: w / 2, CenterY: h / 2,
				Description: fmt.Sprintf("柱状图：%d 个柱体，平均间距 %dpx", len(bars), avgGap),
			}}, bars...)
		}
	}

	if len(bars) > 15 {
		bars = bars[:15]
	}

	return bars
}

// detectPieChart 检测饼图（多个扇形区域）。
func detectPieChart(shapes []shapeInfo) *shapeInfo {
	// 查找是否有多个扇形围绕同一个中心
	var sectors []shapeInfo
	for _, s := range shapes {
		if s.Kind == "扇形" {
			sectors = append(sectors, s)
		}
	}
	if len(sectors) >= 2 {
		return &shapeInfo{
			Kind: "图表类型",
			Description: fmt.Sprintf("饼图/环形图：%d 个扇形区域", len(sectors)),
		}
	}
	return nil
}

// detectLineChart 检测折线图（数据点 + 连接线）。
func detectLineChart(shapes []shapeInfo) *shapeInfo {
	// 查找是否有曲线/折线 + 多个点状标记
	lineCount := 0
	dotCount := 0
	for _, s := range shapes {
		if s.Kind == "曲线/折线" || strings.Contains(s.Kind, "折线") {
			lineCount++
		}
		if s.Kind == "圆形" && s.Area < 100 {
			dotCount++
		}
	}
	if lineCount >= 1 && dotCount >= 3 {
		return &shapeInfo{
			Kind: "图表类型",
			Description: fmt.Sprintf("折线图：%d 条折线，%d 个数据点", lineCount, dotCount),
		}
	}
	if lineCount >= 1 {
		return &shapeInfo{
			Kind: "图表类型",
			Description: "折线图趋势线",
		}
	}
	return nil
}

// detectLegends 检测图例（颜色块 + 相邻文字）。
func detectLegends(img image.Image, bounds image.Rectangle, w, h int, shapes []shapeInfo) []shapeInfo {
	var legends []shapeInfo

	// 查找小矩形色块（图例标记）
	for _, s := range shapes {
		if (s.Kind == "矩形" || s.Kind == "圆角矩形") &&
			s.Width < 30 && s.Height < 30 &&
			s.Width > 5 && s.Height > 5 &&
			s.FillRatio > 0.7 {
			// 右侧附近可能有文字
			legends = append(legends, shapeInfo{
				Kind: "图例标记",
				X1:   s.X1, Y1: s.Y1,
				X2: s.X2, Y2: s.Y2,
				CenterX:     s.CenterX, CenterY: s.CenterY,
				Width: s.Width, Height: s.Height,
				Description: fmt.Sprintf("图例色块 (%d,%d) 大小 %dx%d", s.X1, s.Y1, s.Width, s.Height),
			})
		}
	}

	// 如果图例标记 >= 2，集中标注
	if len(legends) >= 2 {
		minX, minY := w, h
		maxX, maxY := 0, 0
		for _, l := range legends {
			if l.X1 < minX {
				minX = l.X1
			}
			if l.Y1 < minY {
				minY = l.Y1
			}
			if l.X2 > maxX {
				maxX = l.X2
			}
			if l.Y2 > maxY {
				maxY = l.Y2
			}
		}
		legends = append(legends, shapeInfo{
			Kind: "图例区域",
			X1:   minX, Y1: minY,
			X2: maxX, Y2: maxY,
			CenterX: (minX + maxX) / 2, CenterY: (minY + maxY) / 2,
			Description: fmt.Sprintf("图例区域：%d 项", len(legends)),
		})
	}

	if len(legends) > 10 {
		legends = legends[:10]
	}

	return legends
}

// inferChartType 根据已检测到的图表元素推断整体图表类型。
func inferChartType(chartElements []shapeInfo) string {
	hasAxis := false
	hasBars := false
	hasLines := false
	hasPie := false
	hasLegend := false

	for _, e := range chartElements {
		switch e.Kind {
		case "坐标轴":
			hasAxis = true
		case "柱状图柱体":
			hasBars = true
		case "曲线/折线", "折线":
			hasLines = true
		case "扇形":
			hasPie = true
		case "图例区域", "图例标记":
			hasLegend = true
		}
	}

	parts := []string{}
	if hasAxis {
		parts = append(parts, "坐标轴")
	}
	if hasBars {
		parts = append(parts, "柱状图")
	}
	if hasLines {
		parts = append(parts, "折线图")
	}
	if hasPie {
		parts = append(parts, "饼图")
	}
	if hasLegend {
		parts = append(parts, "图例")
	}

	if len(parts) > 0 {
		return "检测到图表元素：" + strings.Join(parts, "、")
	}
	return ""
}
