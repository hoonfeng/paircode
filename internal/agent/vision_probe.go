package agent

// vision_probe.go — image_probe：像素级可靠性扫描工具（UI 调试用）。
//
// 与 image_analyze 的启发式分析不同，本工具直接读取像素做**精确验证**：
// 查询指定坐标颜色、检查区域颜色/纯色性、搜索指定颜色元素位置、检测边框、
// 计算颜色对比度。用于定位 UI bug（元素渲染、布局错位、边框缺失、可读性等）。
//
// 规则驱动：rules 为 JSON 数组字符串，每条规则定义扫描目的，逐条返回检测报告。

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// probeRule 一条扫描规则。
type probeRule struct {
	Type      string `json:"type"`                // pixel / region / color_search / border / contrast
	X         int    `json:"x,omitempty"`         // pixel: 查询坐标
	Y         int    `json:"y,omitempty"`         // pixel: 查询坐标
	X1        int    `json:"x1,omitempty"`        // region/border/color_search: 区域左上
	Y1        int    `json:"y1,omitempty"`        // region/border/color_search: 区域左上
	X2        int    `json:"x2,omitempty"`        // region/border/color_search: 区域右下
	Y2        int    `json:"y2,omitempty"`        // region/border/color_search: 区域右下
	Color     string `json:"color,omitempty"`     // #RRGGBB 期望/目标颜色
	Color2    string `json:"color2,omitempty"`    // contrast: 第二个颜色
	Tolerance int    `json:"tolerance,omitempty"` // 颜色容差 0-255（每通道），默认 30
	Side      string `json:"side,omitempty"`      // border: top/bottom/left/right
	Max       int    `json:"max,omitempty"`       // color_search: 最大返回块数，默认 5
}

// probeResult 一条规则检测结果。
type probeResult struct {
	RuleNo  int    // 规则序号
	Type    string // 规则类型
	Pass    bool   // 是否通过（有期望值时）
	Summary string // 检测结论
	Detail  string // 详细数据
}

// registerProbeTool 注册 image_probe 工具。
func registerProbeTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "image_probe",
		UsageGuide: "像素级精确扫描图片（UI 调试专用）。与 image_analyze 的启发式分析不同，" +
			"本工具直接读取像素做精确验证：查坐标颜色、查区域主色/是否含某色、搜索某颜色元素位置、检测边框、算对比度。" +
			"rules 为 JSON 数组字符串，每条规则 {type, ...}。支持类型：\n" +
			"  - pixel: 查询坐标颜色 {type:\"pixel\", x, y}\n" +
			"  - region: 区域颜色统计 {type:\"region\", x1,y1,x2,y2, color?}(\"color\" 指定则校验该色是否出现在区域并返回占比)\n" +
			"  - color_search: 在全图/区域找指定颜色元素 {type:\"color_search\", color, x1?,y1?,x2?,y2?, max?}\n" +
			"  - border: 检测矩形边框 {type:\"border\", x1,y1,x2,y2, side:\"top|bottom|left|right\", expect_color?, min_thickness?}\n" +
			"  - contrast: 计算两色 WCAG 对比度 {type:\"contrast\", color, color2}\n" +
			"每条规则返回：检测值 + 是否通过（有期望时）。定位 UI bug 时用它精确验证可疑像素，不要仅靠 image_analyze 的猜测。",
		Description: "像素级精确扫描图片（UI 调试）。直接读取像素验证：坐标颜色、区域颜色/纯色性、指定颜色元素位置、边框、对比度。" +
			"基于实际像素数据，可靠性高于 image_analyze 的启发式检测。rules 为 JSON 数组字符串，逐条返回检测报告（含通过/失败判定）。",
		Parameters: objSchema(props{
			"path":  strProp("图片路径（工作区内）"),
			"rules": strProp("扫描规则 JSON 数组字符串，如 [{\"type\":\"pixel\",\"x\":10,\"y\":20},{\"type\":\"region\",\"x1\":0,\"y1\":0,\"x2\":100,\"y2\":50,\"color\":\"#FFFFFF\"}]"),
		}, "path", "rules"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			rulesJSON := argStr(args, "rules")
			if rulesJSON == "" {
				return "", fmt.Errorf("缺少 rules 参数（JSON 数组字符串，定义扫描规则）")
			}
			return probeImage(p, rulesJSON)
		},
	})
}

// probeImage 执行像素级扫描并返回报告。
func probeImage(path, rulesJSON string) (string, error) {
	var rules []probeRule
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return "", fmt.Errorf("rules 解析失败（需为 JSON 数组）: %w", err)
	}
	if len(rules) == 0 {
		return "", fmt.Errorf("rules 不能为空")
	}

	// ★ 纯 contrast 规则不依赖图片（只需两个颜色值），跳过图片加载
	needImage := false
	for _, r := range rules {
		if r.Type != "contrast" {
			needImage = true
			break
		}
	}

	var img image.Image
	var format string
	var w, h int
	var bounds image.Rectangle
	if needImage {
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("无法打开图片 %q: %w", path, err)
		}
		defer f.Close()
		decoded, decFormat, err := image.Decode(f)
		if err != nil {
			return "", fmt.Errorf("无法解码图片（格式可能不支持）: %w", err)
		}
		img = decoded
		format = decFormat
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
		if w == 0 || h == 0 {
			return "", fmt.Errorf("图片尺寸无效：%dx%d", w, h)
		}
	}

	var results []probeResult
	for i, rule := range rules {
		if rule.Type == "contrast" {
			results = append(results, probeContrast(rule, i+1))
		} else {
			results = append(results, execProbeRule(img, bounds, w, h, rule, i+1))
		}
	}

	// 输出报告
	var b strings.Builder
	fmt.Fprintf(&b, "## 像素级扫描报告\n\n")
	fmt.Fprintf(&b, "| 属性 | 值 |\n|------|-----|\n")
	fmt.Fprintf(&b, "| 文件 | %s |\n", filepath.Base(path))
	fmt.Fprintf(&b, "| 格式 | %s |\n", format)
	fmt.Fprintf(&b, "| 尺寸 | %d × %d px |\n", w, h)
	fmt.Fprintf(&b, "| 规则数 | %d |\n", len(results))
	b.WriteString("\n> 本报告基于实际像素数据（非启发式猜测），用于 UI 调试精确验证。\n\n")

	b.WriteString("━━━ 逐条检测结果 ━━━\n\n")
	passCount := 0
	for _, res := range results {
		status := "—"
		if res.Type == "pixel" || res.Type == "contrast" {
			status = "信息"
		} else if res.Pass {
			status = "✓ 通过"
			passCount++
		} else {
			status = "✗ 失败"
		}
		fmt.Fprintf(&b, "**【规则 %d】%s**（%s）\n\n", res.RuleNo, ruleTypeName(res.Type), status)
		fmt.Fprintf(&b, "- 结论：%s\n", res.Summary)
		if res.Detail != "" {
			fmt.Fprintf(&b, "- 详情：%s\n", res.Detail)
		}
		b.WriteString("\n")
	}
	b.WriteString("━━━ 汇总 ━━━\n\n")
	fmt.Fprintf(&b, "- 通过 %d / %d 项（pixel/contrast 为信息性规则不计入）\n", passCount, passCount+countFail(results))

	return b.String(), nil
}

func countFail(results []probeResult) int {
	n := 0
	for _, r := range results {
		if r.Type != "pixel" && r.Type != "contrast" && !r.Pass {
			n++
		}
	}
	return n
}

func ruleTypeName(t string) string {
	switch t {
	case "pixel":
		return "坐标取色"
	case "region":
		return "区域颜色"
	case "color_search":
		return "颜色定位"
	case "border":
		return "边框检测"
	case "contrast":
		return "对比度"
	default:
		return t
	}
}

// execProbeRule 执行单条规则。
func execProbeRule(img image.Image, bounds image.Rectangle, w, h int, rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: rule.Type}
	if rule.Tolerance <= 0 {
		rule.Tolerance = 30
	}
	switch rule.Type {
	case "pixel":
		res = probePixel(img, bounds, w, h, rule, no)
	case "region":
		res = probeRegion(img, bounds, w, h, rule, no)
	case "color_search":
		res = probeColorSearch(img, bounds, w, h, rule, no)
	case "border":
		res = probeBorder(img, bounds, w, h, rule, no)
	case "contrast":
		res = probeContrast(rule, no)
	default:
		res.Summary = fmt.Sprintf("未知规则类型 %q（支持 pixel/region/color_search/border/contrast）", rule.Type)
	}
	return res
}

// ── pixel：坐标取色 ──

func probePixel(img image.Image, bounds image.Rectangle, w, h int, rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: "pixel"}
	x, y := rule.X, rule.Y
	if x < 0 || y < 0 || x >= w || y >= h {
		res.Summary = fmt.Sprintf("坐标 (%d,%d) 越界（图片 %dx%d）", x, y, w, h)
		return res
	}
	r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	r8, g8, b8 := r>>8, g>>8, b>>8
	alpha := a >> 8
	hex := fmt.Sprintf("#%02X%02X%02X", r8, g8, b8)
	res.Summary = fmt.Sprintf("坐标 (%d,%d) 颜色 = %s（RGB %d,%d,%d），颜色名「%s」", x, y, hex, r8, g8, b8, colorName(r8, g8, b8))
	// 周围 3x3 采样（判断是否在边缘/渐变/抗锯齿区）
	var det []string
	if x > 0 && y > 0 && x < w-1 && y < h-1 {
		var sumR, sumG, sumB, cnt int
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				rr, gg, bb, _ := img.At(bounds.Min.X+x+dx, bounds.Min.Y+y+dy).RGBA()
				sumR += int(rr >> 8)
				sumG += int(gg >> 8)
				sumB += int(bb >> 8)
				cnt++
			}
		}
		avgR, avgG, avgB := sumR/cnt, sumG/cnt, sumB/cnt
		diff := absDiff(uint32(r8), uint32(avgR)) + absDiff(uint32(g8), uint32(avgG)) + absDiff(uint32(b8), uint32(avgB))
		if diff > 60 {
			det = append(det, fmt.Sprintf("与周边均值差异大（Δ=%d），位于边缘/渐变区域", diff))
		}
		if alpha < 255 {
			det = append(det, fmt.Sprintf("半透明（alpha=%d）", alpha))
		}
	}
	if len(det) > 0 {
		res.Detail = strings.Join(det, "；")
	}
	return res
}

// ── region：区域颜色统计 ──

func probeRegion(img image.Image, bounds image.Rectangle, w, h int, rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: "region"}
	x1, y1, x2, y2 := clampRegion(rule, w, h)
	if x1 >= x2 || y1 >= y2 {
		res.Summary = "区域无效（x2>x1 且 y2>y1）"
		return res
	}
	rw, rh := x2-x1, y2-y1
	// 降采样步长
	step := 1
	if rw*rh > 200000 {
		step = 2
	}
	if rw*rh > 1000000 {
		step = 3
	}

	// 统计颜色（8 位量化）与期望色匹配
	quant := map[uint32]int{}
	total := 0
	expectR, expectG, expectB, hasExpect := parseHexColor(rule.Color)
	matchCount := 0
	maxR, maxG, maxB := 0, 0, 0
	maxKeyCount := 0

	for y := y1; y < y2; y += step {
		for x := x1; x < x2; x += step {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a < 128 {
				continue
			}
			r8, g8, b8 := r>>8, g>>8, b>>8
			key := ((r8 >> 4) << 8) | ((g8 >> 4) << 4) | (b8 >> 4)
			quant[key]++
			total++
			if hasExpect && colorMatch8(r8, g8, b8, uint32(expectR), uint32(expectG), uint32(expectB), uint32(rule.Tolerance)) {
				matchCount++
			}
			if quant[key] > maxKeyCount {
				maxKeyCount = quant[key]
				maxR, maxG, maxB = int(r8), int(g8), int(b8)
			}
		}
	}
	if total == 0 {
		res.Summary = fmt.Sprintf("区域 (%d,%d)-(%d,%d) 内无有效像素（全透明）", x1, y1, x2, y2)
		return res
	}
	mainHex := fmt.Sprintf("#%02X%02X%02X", maxR, maxG, maxB)
	mainPct := float64(maxKeyCount) / float64(total) * 100
	colorsN := len(quant)

	detail := fmt.Sprintf("区域 (%d,%d)-(%d,%d) %dx%d，主色 %s（占 %.0f%%），量化颜色种类 %d",
		x1, y1, x2, y2, rw, rh, mainHex, mainPct, colorsN)

	if hasExpect {
		matchPct := float64(matchCount) / float64(total) * 100
		res.Pass = matchPct >= 95 // 纯色区域期望色占比 ≥95% 视为通过
		if matchPct >= 95 {
			res.Summary = fmt.Sprintf("区域主色为期望色 %s（匹配 %.0f%%）", rule.Color, matchPct)
		} else {
			res.Summary = fmt.Sprintf("区域不是期望色 %s：匹配仅 %.0f%%，主色 %s", rule.Color, matchPct, mainHex)
		}
		res.Detail = detail
	} else {
		res.Summary = fmt.Sprintf("区域主色 %s（占 %.0f%%）", mainHex, mainPct)
		res.Detail = detail
	}
	return res
}

// ── color_search：定位指定颜色元素 ──

func probeColorSearch(img image.Image, bounds image.Rectangle, w, h int, rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: "color_search"}
	expectR, expectG, expectB, hasExpect := parseHexColor(rule.Color)
	if !hasExpect {
		res.Summary = "缺少 color 参数（#RRGGBB）"
		return res
	}
	x1, y1, x2, y2 := clampRegion(rule, w, h)
	if x1 >= x2 || y1 >= y2 {
		res.Summary = "区域无效"
		return res
	}
	step := 1
	if (x2-x1)*(y2-y1) > 500000 {
		step = 2
	}

	// 标记匹配像素，做连通域
	visited := make([][]bool, (y2-y1+step-1)/step)
	for i := range visited {
		visited[i] = make([]bool, (x2-x1+step-1)/step)
	}
	type block struct {
		x1, y1, x2, y2, cnt int
	}
	var blocks []block

	for y := y1; y < y2; y += step {
		for x := x1; x < x2; x += step {
			ix, iy := (x-x1)/step, (y-y1)/step
			if visited[iy][ix] {
				continue
			}
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			if a < 128 || !colorMatch8(r>>8, g>>8, b>>8, uint32(expectR), uint32(expectG), uint32(expectB), uint32(rule.Tolerance)) {
				visited[iy][ix] = true
				continue
			}
			// flood fill
			minX, minY, maxX, maxY, cnt := x, y, x+step, y+step, 0
			stack := [][2]int{{ix, iy}}
			visited[iy][ix] = true
			for len(stack) > 0 {
				cx, cy := stack[len(stack)-1][0], stack[len(stack)-1][1]
				stack = stack[:len(stack)-1]
				px, py := x1+cx*step, y1+cy*step
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
				cnt++
				for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx, ny := cx+d[0], cy+d[1]
					if nx < 0 || ny < 0 || nx >= len(visited[0]) || ny >= len(visited) || visited[ny][nx] {
						continue
					}
					nr, ng, nb, na := img.At(bounds.Min.X+x1+nx*step, bounds.Min.Y+y1+ny*step).RGBA()
					if na < 128 || !colorMatch8(nr>>8, ng>>8, nb>>8, uint32(expectR), uint32(expectG), uint32(expectB), uint32(rule.Tolerance)) {
						visited[ny][nx] = true
						continue
					}
					visited[ny][nx] = true
					stack = append(stack, [2]int{nx, ny})
				}
			}
			bw, bh := maxX-minX, maxY-minY
			if bw < 2 && bh < 2 {
				continue
			}
			blocks = append(blocks, block{minX, minY, maxX, maxY, cnt})
		}
	}
	if len(blocks) == 0 {
		res.Summary = fmt.Sprintf("在区域内未找到颜色 %s（容差 %d）", rule.Color, rule.Tolerance)
		return res
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i].cnt > blocks[j].cnt })
	maxN := rule.Max
	if maxN <= 0 {
		maxN = 5
	}
	if len(blocks) > maxN {
		blocks = blocks[:maxN]
	}
	var parts []string
	for i, bl := range blocks {
		parts = append(parts, fmt.Sprintf("块%d: (%d,%d)-(%d,%d) %dx%d", i+1, bl.x1, bl.y1, bl.x2, bl.y2, bl.x2-bl.x1, bl.y2-bl.y1))
	}
	res.Summary = fmt.Sprintf("找到 %d 个颜色 %s 的区域（显示前 %d 个）：", len(blocks), rule.Color, len(blocks))
	res.Detail = strings.Join(parts, "；")
	res.Pass = true
	return res
}

// ── border：边框检测 ──

func probeBorder(img image.Image, bounds image.Rectangle, w, h int, rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: "border"}
	x1, y1, x2, y2 := clampRegion(rule, w, h)
	if x1 >= x2 || y1 >= y2 {
		res.Summary = "区域无效"
		return res
	}
	expectR, expectG, expectB, hasExpect := parseHexColor(rule.Color) // color 视为期望边框色
	side := rule.Side

	// 沿边扫描：在边线内缩 2px 的条带上检测颜色变化
	var points []int // 扫描坐标（另一个轴）
	var innerY, outerY int
	length := 0
	switch side {
	case "top", "bottom":
		length = x2 - x1
		innerY = y1 + 2
		outerY = y1 + 6
		if side == "bottom" {
			innerY = y2 - 3
			outerY = y2 - 7
		}
		for p := 0; p < length; p++ {
			points = append(points, x1+p)
		}
	case "left", "right":
		length = y2 - y1
		innerY = x1 + 2
		outerY = x1 + 6
		if side == "right" {
			innerY = x2 - 3
			outerY = x2 - 7
		}
		for p := 0; p < length; p++ {
			points = append(points, y1+p)
		}
	default:
		res.Summary = fmt.Sprintf("side 参数无效 %q（应为 top/bottom/left/right）", side)
		return res
	}

	// 采样内外两层，统计"边框像素"（内层颜色与外层背景差异大的点）
	borderPixels := 0
	var borderColors [][3]uint32
	var gaps []int
	for i, p := range points {
		var ir, ig, ib, or_, og, ob uint32
		if side == "top" || side == "bottom" {
			ir, ig, ib, _ = img.At(bounds.Min.X+p, bounds.Min.Y+innerY).RGBA()
			or_, og, ob, _ = img.At(bounds.Min.X+p, bounds.Min.Y+outerY).RGBA()
		} else {
			ir, ig, ib, _ = img.At(bounds.Min.X+innerY, bounds.Min.Y+p).RGBA()
			or_, og, ob, _ = img.At(bounds.Min.X+outerY, bounds.Min.Y+p).RGBA()
		}
		// 内层与外层差异大 → 该处存在边框线
		diff := absDiff(ir>>8, or_>>8) + absDiff(ig>>8, og>>8) + absDiff(ib>>8, ob>>8)
		if diff > 90 {
			borderPixels++
			borderColors = append(borderColors, [3]uint32{ir >> 8, ig >> 8, ib >> 8})
		} else {
			gaps = append(gaps, i)
		}
	}

	if length == 0 {
		res.Summary = "边长为 0"
		return res
	}
	coverage := float64(borderPixels) / float64(length) * 100
	res.Detail = fmt.Sprintf("沿边扫描 %d px，边框像素 %d（覆盖率 %.0f%%），断裂 %d 处",
		length, borderPixels, coverage, len(gaps))

	// 边框主色
	var main [3]uint32
	bestCnt := 0
	cmap := map[[3]uint32]int{}
	for _, c := range borderColors {
		cmap[c]++
		if cmap[c] > bestCnt {
			bestCnt = cmap[c]
			main = c
		}
	}
	mainHex := fmt.Sprintf("#%02X%02X%02X", main[0], main[1], main[2])

	if coverage < 60 {
		res.Summary = fmt.Sprintf("该边边框缺失/断裂严重：覆盖率仅 %.0f%%（边框像素 %d/%d）", coverage, borderPixels, length)
		return res
	}
	// 边框厚度：从内层再向内扫，找连续同色
	thickness := probeBorderThickness(img, bounds, side, points, main, innerY, outerY)

	if hasExpect && !colorMatch8(main[0], main[1], main[2], uint32(expectR), uint32(expectG), uint32(expectB), 60) {
		res.Pass = false
		res.Summary = fmt.Sprintf("边框颜色不符：实际 %s，期望 %s（覆盖率 %.0f%%，厚度约 %dpx）", mainHex, rule.Color, coverage, thickness)
	} else {
		res.Pass = true
		res.Summary = fmt.Sprintf("边框完整：颜色 %s，厚度约 %dpx，覆盖率 %.0f%%", mainHex, thickness, coverage)
	}
	return res
}

// probeBorderThickness 估算边框厚度（从内层继续向内扫，统计主色连续像素数）。
func probeBorderThickness(img image.Image, bounds image.Rectangle, side string, points []int, main [3]uint32, innerY, outerY int) int {
	// 在边框代表点上，从内层向元素内部扫描，找主色连续长度
	sampleIdx := 0
	if len(points) > 0 {
		sampleIdx = len(points) / 2
	}
	p := points[sampleIdx]
	maxThick := 0
	step := 2
	for start := 0; start < 12; start += step {
		thick := 0
		for t := start; t < 40; t += step {
			var r, g, b uint32
			if side == "top" || side == "bottom" {
				yy := innerY
				if side == "bottom" {
					yy = innerY - t
				} else {
					yy = innerY + t
				}
				if yy < bounds.Min.Y || yy >= bounds.Max.Y {
					break
				}
				r, g, b, _ = img.At(bounds.Min.X+p, yy).RGBA()
			} else {
				xx := innerY
				if side == "right" {
					xx = innerY - t
				} else {
					xx = innerY + t
				}
				if xx < bounds.Min.X || xx >= bounds.Max.X {
					break
				}
				r, g, b, _ = img.At(xx, bounds.Min.Y+p).RGBA()
			}
			if colorMatch8(r>>8, g>>8, b>>8, main[0], main[1], main[2], 60) {
				thick = t + step
			} else {
				break
			}
		}
		if thick > maxThick {
			maxThick = thick
		}
	}
	if maxThick <= 0 {
		return 2 // 兜底
	}
	return maxThick
}

// ── contrast：WCAG 对比度 ──

func probeContrast(rule probeRule, no int) probeResult {
	res := probeResult{RuleNo: no, Type: "contrast"}
	r1, g1, b1, ok1 := parseHexColor(rule.Color)
	r2, g2, b2, ok2 := parseHexColor(rule.Color2)
	if !ok1 || !ok2 {
		res.Summary = "需要 color 和 color2 两个 #RRGGBB 颜色"
		return res
	}
	ratio := wcagContrast(float64(r1)/255, float64(g1)/255, float64(b1)/255,
		float64(r2)/255, float64(g2)/255, float64(b2)/255)
	verdict := "不达标"
	if ratio >= 4.5 {
		verdict = "达到 AA（正文）"
	}
	if ratio >= 7 {
		verdict = "达到 AAA"
	}
	if ratio >= 3 && ratio < 4.5 {
		verdict = "达到 AA（大文本/UI 组件）"
	}
	res.Summary = fmt.Sprintf("对比度 %s vs %s = %.2f:1（%s）", rule.Color, rule.Color2, ratio, verdict)
	res.Detail = fmt.Sprintf("WCAG 2.1：AA 正文需 ≥4.5，AA 大文本/UI 组件需 ≥3.0，AAA 需 ≥7.0")
	return res
}

// wcagContrast 计算 WCAG 对比度（0~21）。
func wcagContrast(r1, g1, b1, r2, g2, b2 float64) float64 {
	l1 := wcagLuminance(r1, g1, b1)
	l2 := wcagLuminance(r2, g2, b2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func wcagLuminance(r, g, b float64) float64 {
	lin := func(c float64) float64 {
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// ── 辅助 ──

// clampRegion 归一化区域坐标（支持缺省=全图）。
func clampRegion(rule probeRule, w, h int) (x1, y1, x2, y2 int) {
	x1, y1, x2, y2 = rule.X1, rule.Y1, rule.X2, rule.Y2
	if x1 == 0 && y1 == 0 && x2 == 0 && y2 == 0 {
		return 0, 0, w, h
	}
	if x1 < 0 {
		x1 = 0
	}
	if y1 < 0 {
		y1 = 0
	}
	if x2 <= 0 || x2 > w {
		x2 = w
	}
	if y2 <= 0 || y2 > h {
		y2 = h
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	return
}

// parseHexColor 解析 #RRGGBB（支持无 # 前缀）。
func parseHexColor(s string) (r, g, b int, ok bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 {
		var v int
		if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err == nil {
			_ = v
			return r, g, b, true
		}
	}
	// 3 位简写 #RGB
	if len(s) == 3 {
		if _, err := fmt.Sscanf(s, "%1x%1x%1x", &r, &g, &b); err == nil {
			r = r*16 + r
			g = g*16 + g
			b = b*16 + b
			return r, g, b, true
		}
	}
	return 0, 0, 0, false
}

// colorMatch8 8 位颜色匹配（容差为每通道和）。
func colorMatch8(r1, g1, b1, r2, g2, b2 uint32, tolerance uint32) bool {
	return absDiff(r1, r2)+absDiff(g1, g2)+absDiff(b1, b2) <= tolerance
}
