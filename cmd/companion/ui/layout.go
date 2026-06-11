//go:build windows

package ui

import (
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

// 布局原语 —— 类 CSS flexbox。语义化替代到处手写的 Div(Style{FlexDirection:...})
// 与 Div(Style{Height:n}) 间距块，让面板代码只读「结构」。

// Row 水平容器（交叉轴居中）—— 类 `display:flex; align-items:center`。
func Row(children ...widget.Widget) widget.Widget {
	return widget.Div(Class(ClassRow), children)
}

// Col 垂直容器（子项拉伸）—— 类 `display:flex; flex-direction:column`。
func Col(children ...widget.Widget) widget.Widget {
	return widget.Div(Class(ClassCol), children)
}

// RowG / ColG 带间距(gap)的行 / 列。
func RowG(gap float64, children ...widget.Widget) widget.Widget {
	return widget.Div(widget.Merge(Class(ClassRow), widget.Style{Gap: gap}), children)
}
func ColG(gap float64, children ...widget.Widget) widget.Widget {
	return widget.Div(widget.Merge(Class(ClassCol), widget.Style{Gap: gap}), children)
}

// Box 带自定义 Style 的容器（需要额外内距 / 背景 / 固定尺寸时用）。
func Box(s widget.Style, children ...widget.Widget) widget.Widget {
	return widget.Div(s, children)
}

// Card 卡片容器（ui-card 样式类：次级面 + 边框 + 圆角 + 内距）。
func Card(children ...widget.Widget) widget.Widget {
	return widget.Div(Class(ClassCard), children)
}

// VGap / HGap 固定竖 / 横间距 —— 语义化替代到处写的 Div(Style{Height:n})。
func VGap(h float64) widget.Widget { return widget.Div(widget.Style{Height: h}) }
func HGap(w float64) widget.Widget { return widget.Div(widget.Style{Width: w}) }

// Expand 让子节点弹性占满主轴剩余空间（flex:1）。
func Expand(w widget.Widget) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1}
}

// ExpandMin 弹性占满主轴剩余空间，但主轴有下限（flex 空间不足时至少给 minMain，不被挤窄）。
func ExpandMin(w widget.Widget, minMain float64) widget.Widget {
	return &widget.Expanded{SingleChildWidget: widget.SingleChildWidget{Child: w}, Flex: 1, MinMain: minMain}
}

// Spacer 弹性空白，把两侧推到容器两端。
func Spacer() widget.Widget { return widget.SpacerDiv() }

// Divider 水平分隔线（随主题 Border 色）。
func Divider() widget.Widget {
	return widget.Div(widget.Style{Height: 1, BackgroundColor: Border})
}

// VLine 竖直分隔线（1px，随主题 Border 色）。
func VLine() widget.Widget {
	return widget.Div(widget.Style{Width: 1, BackgroundColor: Border})
}

// FlexRow / FlexCol 交叉轴拉伸(align-items:stretch)的行 / 列，接受动态子节点列表。
// 与 Row(交叉轴居中) 不同：子项在交叉轴拉伸填满；与 Col 等价但显式命名成对。
func FlexRow(children ...widget.Widget) widget.Widget { return flexBox("row", children) }
func FlexCol(children ...widget.Widget) widget.Widget { return flexBox("column", children) }
func flexBox(dir string, children []widget.Widget) widget.Widget {
	args := make([]interface{}, 0, len(children)+1)
	args = append(args, widget.Style{FlexDirection: dir, AlignItems: "stretch"})
	for _, c := range children {
		args = append(args, c)
	}
	return widget.Div(args...)
}

// Icon 主题化 Lucide 图标。
func Icon(name string, size float64, c types.Color) widget.Widget {
	return widget.Lucide(name, widget.IconSize(size), widget.IconColor(c))
}

// SectionHeader 带图标的小节标题 + 灰色副标题（设置 / 哲学等小节复用）。
func SectionHeader(icon, title, sub string) widget.Widget {
	return Row(Icon(icon, 13, *FgMuted), HGap(6), Label(title), HGap(6), Muted(sub))
}
