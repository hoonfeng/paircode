// debug_card — 独立测试 demo，排查工具渲染组件无限撑高问题。
//
// 使用 goui 无窗口渲染模式（render shot），生成 PNG 快照 + 详细布局诊断。
// 运行方式:
//   go run ./cmd/debug_card/
//
//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/render"
	"github.com/hoonfeng/goui/pkg/types"
	"github.com/hoonfeng/goui/pkg/widget"
)

func main() {
	runtime.LockOSThread()
	// 启用布局日志
	os.Setenv("LAYOUT_DEBUG", "1")

	fmt.Println("=== 工具渲染组件无限撑高问题 - 深度诊断测试 ===")
	fmt.Println()
	fmt.Println("测试目标：复现 agent_card.go 中 activityRow 在 ScrollView 内的异常行为")
	fmt.Println()

	testScrollViewExpandInteraction()
	testNestedAgentCardStructure()
	testEmptyActivityData()
	testLayoutHeightChain()

	fmt.Println()
	fmt.Println("✅ 全部测试完成")
}

// ── 颜色常量 ──
func cRef(r, g, b uint8) *types.Color { c := types.ColorFromRGB(r, g, b); return &c }
func cRGBA(r, g, b, a uint8) types.Color { return types.ColorFromRGBA(r, g, b, a) }

var (
	bg       = cRef(30, 30, 30)
	bgMuted  = cRef(37, 37, 38)
	borderC  = cRef(45, 45, 45)
	fg       = cRef(204, 204, 204)
	fgMuted  = cRef(140, 140, 140)
	fgSubtle = cRef(100, 100, 100)
	accent   = cRef(0, 122, 204)
	success  = cRef(34, 197, 94)
	warning  = cRef(251, 191, 36)
	danger   = cRef(239, 68, 68)
)

func vgap(h float64) widget.Widget { return widget.Div(widget.Style{Height: h}) }
func hgap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }

func expand(w widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1}
}

func expandMin(w widget.Widget, minMain float64) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1, MinMain: minMain}
}

func text(s string, c *types.Color, sz float64) widget.Widget {
	t := widget.NewText(s, *c)
	t.Font.Size = sz
	t.Selectable = true
	return t
}

func shot(name string, root widget.Widget) {
	const w, h = 1200, 800
	sk := canvas.NewSkiaCanvas(w, h)
	defer sk.Release()
	pipe := render.NewPipeline(w, h, sk)
	pipe.SetRootElement(widget.CreateElementFor(root))
	if err := pipe.Render(); err != nil {
		fmt.Printf("  ❌ render 失败: %v\n", err)
		return
	}
	if err := sk.SaveToPNG(name); err != nil {
		fmt.Printf("  ❌ 保存 PNG 失败: %v\n", err)
		return
	}
	fmt.Printf("  ✅ %s 已保存\n", name)
}

// ── activityRow 模拟（简化版，复刻 agent_card.go 的结构）──
func mockActivityRow(toolName string, args string, hasResult bool, result string, expanded bool) widget.Widget {
	headRow := []widget.Widget{}
	if hasResult {
		headRow = append(headRow,
			widget.Lucide("chevron-down", widget.IconSize(11), widget.IconColor(*fgMuted)),
			hgap(4),
		)
	} else {
		headRow = append(headRow, widget.Div(widget.Style{Width: 15}))
	}

	pillBg := accent
	pill := widget.Div(
		widget.Style{BackgroundColor: pillBg, BorderRadius: 4, Padding: types.EdgeInsetsLTRB(6, 2, 6, 2)},
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("terminal", widget.IconSize(11), widget.IconColor(*fg)),
			hgap(4),
			text(toolName, fg, 10),
		),
	)
	headRow = append(headRow, pill, hgap(6))
	// ★ 这里是关键：Expand 用于参数文本水平填充
	if args != "" {
		headRow = append(headRow, expand(text(args, fgMuted, 11)))
	} else {
		headRow = append(headRow, expand(widget.Div(widget.Style{})))
	}
	headDiv := widget.Div(widget.Style{FlexDirection: "row", AlignItems: "center"}, headRow)
	var head widget.Widget = headDiv
	if hasResult {
		head = &widget.Clickable{
			SingleChildWidget: widget.SingleChildWidget{Child: headDiv},
			OnClick:           func() {},
		}
	}

	kids := []widget.Widget{head}
	if hasResult && expanded {
		kids = append(kids, vgap(4),
			widget.Div(
				widget.Style{BackgroundColor: cRef(12, 14, 18), BorderRadius: 4,
					Padding: types.EdgeInsetsLTRB(8, 4, 8, 4)},
				text(result, fgSubtle, 10),
			),
		)
	} else if hasResult {
		kids = append(kids, vgap(3), text(result[:min(len(result), 88)], fgSubtle, 10))
	}

	return widget.Div(
		widget.Style{
			BackgroundColor: cRef(255, 255, 255),
			BorderColor:     borderC,
			BorderWidth:     1,
			BorderRadius:    6,
			Padding:         types.EdgeInsetsLTRB(8, 6, 8, 6),
			FlexDirection:   "column",
			AlignItems:      "start",
		},
		kids,
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ================================================================
// 测试 A：ScrollView + Expand 交互
// 模拟 chat.go 中 ScrollView 被 Expand 包裹的场景
// ================================================================
func testScrollViewExpandInteraction() {
	fmt.Println("=== 测试 A：ScrollView + Expand 交互 ===")
	fmt.Println("  场景：ScrollView 被 Expand 包裹在 Column 中（模拟 chat 面板）")

	// 构建消息内容（模拟 messageList 中的内容）
	activities := make([]widget.Widget, 0, 5)
	for i := 0; i < 5; i++ {
		toolNames := []string{"read_file", "shell_exec", "write_file", "edit_file", "search_content"}
		args := []string{
			"F:\\project\\main.go",
			"cd F:\\project; go build ./cmd/companion/ 2>&1",
			"F:\\project\\newfile.go",
			"old:func test()\nnew:func test2()",
			"func agentMessageCard",
		}
		results := []string{
			"// 文件内容...\npackage main\nimport (...)\nfunc main() {...}",
			"--- OK | 3.52s",
			"已写入 1234 字节",
			"替换成功",
			"找到 5 处匹配",
		}
		activities = append(activities,
			mockActivityRow(toolNames[i], args[i], true, results[i], i%2 == 0),
			vgap(4),
		)
	}

	// 模拟 messageList 返回的内容（agentMessageCard 的近似结构）
	shadowBar := widget.Div(
		widget.Style{Width: 5,
			Gradient: &paint.Gradient{
				Type:  paint.GradientLinear,
				Start: types.Point{X: 0, Y: 0},
				End:   types.Point{X: 1, Y: 0},
				Stops: []paint.ColorStop{
					{Offset: 0.0, Color: cRGBA(34, 197, 94, 55)},
					{Offset: 1.0, Color: types.ColorTransparent},
				},
			},
		},
	)

	cardBody := widget.Div(
		widget.Style{
			BackgroundColor: bgMuted, BorderColor: borderC, BorderWidth: 1, BorderRadius: 6,
			Padding: types.EdgeInsetsLTRB(14, 10, 14, 10),
			FlexDirection: "column", AlignItems: "start",
		},
		// 模拟思考块
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("chevron-down", widget.IconSize(11), widget.IconColor(*fgMuted)),
			hgap(4),
			text("思考", fgMuted, 10),
		),
		vgap(3),
		text("让我分析这段代码...", fgSubtle, 11),
		vgap(4),
		// ★ activity rows
		activities,
	)

	agentCard := widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "start"},
		shadowBar,
		expand(cardBody), // ★ 模拟 agentMessageCard 中的 expand
	)

	// 消息列表（模拟 messageList 使用 ui.FlexCol = AlignItems:stretch）
	msgList := widget.Div(
		widget.Style{Padding: types.EdgeInsets(10)},
		widget.Div(
			widget.Style{FlexDirection: "column", AlignItems: "stretch"},
			agentCard,
			vgap(6),
			// 再添加一个简单的 agent 卡片
			mockSimpleAgentCard("read_file", "F:\\data.txt"),
		),
	)

	// ScrollView → Expand → 放入有固定高度的容器中
	scroll := widget.NewScrollView(msgList)

	// 场景 1：有界容器（Height=400）
	fmt.Println("  场景 A1：有界容器（Height=400）→ ScrollView 应正常")
	root1 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start", Height: 460},
		widget.H2("有界容器 (400px) + ScrollView"),
		vgap(8),
		expand(scroll), // ★ ScrollView 被 Expand 包裹
	)
	shot("debug_card_a1_bounded.png", root1)

	// 场景 2：无界容器（模拟没有固定高度的父容器）
	fmt.Println("  场景 A2：无界容器 → 检查 ScrollView 高度是否 INF")
	root2 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start"},
		widget.H2("无界容器 + ScrollView（应不爆）"),
		vgap(8),
		scroll,
	)
	shot("debug_card_a2_unbounded.png", root2)
}

func mockSimpleAgentCard(toolName, args string) widget.Widget {
	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "start"},
		widget.Div(widget.Style{Width: 5, BackgroundColor: success}),
		expand(widget.Div(
			widget.Style{BackgroundColor: bgMuted, BorderColor: borderC, BorderWidth: 1, BorderRadius: 6,
				Padding: types.EdgeInsetsLTRB(14, 10, 14, 10), FlexDirection: "column", AlignItems: "start"},
			mockActivityRow(toolName, args, true, "OK", false),
		)),
	)
}

// ================================================================
// 测试 B：嵌套 agentMessageCard 完整结构
// 完全复刻 agent_card.go 中的结构
// ================================================================
func testNestedAgentCardStructure() {
	fmt.Println("=== 测试 B：完整 agentMessageCard 嵌套结构 ===")
	fmt.Println("  场景：完全复刻 gou-ide 的聊天渲染结构")

	// 构建完整的 agentMessageCard 模拟
	agentCard := buildFullAgentCard()

	// ScrollView 包裹
	sv := widget.NewScrollView(
		widget.Div(
			widget.Style{Padding: types.EdgeInsets(10), FlexDirection: "column", AlignItems: "stretch"},
			agentCard,
			vgap(6),
			agentCard, // 重复以确保有足够内容
		),
	)

	fmt.Println("  场景 B1：有界容器（Height=500）")
	root1 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start", Height: 560},
		widget.H2("完整 agent 卡片结构 + ScrollView (500px)"),
		vgap(8),
		expand(sv),
	)
	shot("debug_card_b1_full_structure.png", root1)

	fmt.Println("  场景 B2：无界父容器")
	root2 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start"},
		widget.H2("完整结构 + 无界父容器"),
		vgap(8),
		sv,
	)
	shot("debug_card_b2_unbounded_struct.png", root2)
}

func buildFullAgentCard() widget.Widget {
	// 底部阴影条 + cardBody（完全模拟 agentMessageCard）
	shadowBar := widget.Div(
		widget.Style{Width: 5,
			Gradient: &paint.Gradient{
				Type:  paint.GradientLinear,
				Start: types.Point{X: 0, Y: 0},
				End:   types.Point{X: 1, Y: 0},
				Stops: []paint.ColorStop{
					{Offset: 0.0, Color: cRGBA(251, 191, 36, 55)},
					{Offset: 1.0, Color: types.ColorTransparent},
				},
			},
		},
	)

	kids := []widget.Widget{
		// agent header
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			text("Agent", fg, 12),
		),
		vgap(4),
		// thinking block
		widget.Div(
			widget.Style{FlexDirection: "row", AlignItems: "center"},
			widget.Lucide("chevron-down", widget.IconSize(11), widget.IconColor(*fgMuted)),
			hgap(4),
			text("思考", fgMuted, 10),
		),
		vgap(3),
		text("让我分析用户的需求并决定下一步行动...", fgSubtle, 11),
		vgap(6),
		// tool calls
		mockActivityRow("read_file", "F:\\project\\main.go", true,
			"package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}", true),
		vgap(4),
		mockActivityRow("shell_exec", "cd F:\\project; go build ./... 2>&1", true,
			"--- OK | 2.14s\n编译成功，无错误", true),
		vgap(4),
		mockActivityRow("search_content", `pattern:"func agentMessageCard"`, true,
			"找到 3 处匹配:\n  chat.go:42\n  agent_card.go:42\n  bridge.go:670", false),
		vgap(6),
		// text response
		text("分析完成。我已经查看了项目的主要入口文件 main.go，执行了构建验证，并搜索了相关函数。项目结构正常，构建通过。", fg, 13),
	}

	cardBody := widget.Div(
		widget.Style{
			BackgroundColor: bgMuted, BorderColor: borderC, BorderWidth: 1, BorderRadius: 6,
			Padding: types.EdgeInsetsLTRB(14, 10, 14, 10),
			FlexDirection: "column", AlignItems: "start",
		},
		kids,
	)

	return widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "start"},
		shadowBar,
		expand(cardBody),
	)
}

// ================================================================
// 测试 C：空数据 activityRow（边缘情况）
// 模拟 bridge.go 可能产生的空数据
// ================================================================
func testEmptyActivityData() {
	fmt.Println("=== 测试 C：空数据 activityRow ===")
	fmt.Println("  场景：模拟 bridge.go 可能产生的空/异常数据")

	cases := []struct {
		name       string
		toolName   string
		args       string
		hasResult  bool
		result     string
		expanded   bool
	}{
		{"空工具名", "", "", false, "", false},
		{"空参数", "shell_exec", "", false, "", false},
		{"空结果", "read_file", `{"path":"test.go"}`, false, "", false},
		{"正常-无结果", "write_file", `{"path":"t.go"}`, false, "", false},
		{"正常-有结果", "search_content", `{"pattern":"func"}`, true, "找到 3 处", false},
	}

	rows := make([]widget.Widget, 0, len(cases))
	for _, c := range cases {
		label := text(fmt.Sprintf("【%s】", c.name), warning, 10)
		row := mockActivityRow(c.toolName, c.args, c.hasResult, c.result, c.expanded)
		rows = append(rows, label, row, vgap(4))
	}

	root := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start"},
		widget.H2("空数据边缘情况测试"),
		vgap(8),
		widget.Div(
			widget.Style{FlexDirection: "column", AlignItems: "start", Gap: 2},
			rows,
		),
	)

	shot("debug_card_c_empty_data.png", root)
}

// ================================================================
// 测试 D：布局高度链诊断
// 输出从根到最深层组件的高度诊断
// ================================================================
func testLayoutHeightChain() {
	fmt.Println("=== 测试 D：布局高度链诊断 ===")
	fmt.Println("  目标：检查 ScrollView → Column → agentCard 的高度传递链")

	// 构建最小但完整的层级
	activityRow := mockActivityRow("read_file", "F:\\test.go", true, "文件内容...\n多行\n内容", true)
	cardBody := widget.Div(
		widget.Style{BackgroundColor: bgMuted, BorderColor: borderC, BorderWidth: 1, BorderRadius: 6,
			Padding: types.EdgeInsetsLTRB(14, 10, 14, 10), FlexDirection: "column", AlignItems: "start"},
		activityRow,
	)
	agentCard := widget.Div(
		widget.Style{FlexDirection: "row", AlignItems: "start"},
		widget.Div(widget.Style{Width: 5, BackgroundColor: success}),
		expand(cardBody),
	)
	msgList := widget.Div(
		widget.Style{Padding: types.EdgeInsets(10), FlexDirection: "column", AlignItems: "stretch"},
		agentCard,
	)

	// D1：ScrollView + 无界父容器
	fmt.Println("  场景 D1：ScrollView 在无界 Column 中")
	sv := widget.NewScrollView(msgList)
	root1 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start"},
		widget.H2("D1: ScrollView + 无界"),
		vgap(8),
		sv,
	)
	shot("debug_card_d1_sv_unbounded.png", root1)

	// D2：固定高度容器 → Expand(ScrollView)
	fmt.Println("  场景 D2：固定Height=400 → Expand(ScrollView)")
	sv2 := widget.NewScrollView(msgList)
	root2 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start", Height: 460},
		widget.H2("D2: 固定 400px + Expand(SV)"),
		vgap(8),
		expand(sv2),
	)
	shot("debug_card_d2_sv_expand_fixed.png", root2)

	// D3：不固定高度 + Expand(ScrollView)
	fmt.Println("  场景 D3：无固定高度 + Expand(ScrollView)")
	sv3 := widget.NewScrollView(msgList)
	root3 := widget.Div(
		widget.Style{BackgroundColor: bg, Padding: types.EdgeInsets(20),
			FlexDirection: "column", AlignItems: "start"},
		widget.H2("D3: 无固定高度 + Expand(SV)"),
		vgap(8),
		expand(sv3),
	)
	shot("debug_card_d3_sv_expand_unbounded.png", root3)
}
