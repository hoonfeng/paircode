//go:build windows

package ui

import (
	"github.com/user/goui/pkg/types"
	"github.com/user/goui/pkg/widget"
)

// CSS 样式类名（companion 用 widget.Div(ui.Class(...)) 引用；常量避免裸字符串拼错）。
const (
	ClassRow   = "ui-row"   // 行：水平、交叉轴居中
	ClassCol   = "ui-col"   // 列：垂直、子项拉伸
	ClassPanel = "ui-panel" // 面板：主背景填充的竖向容器
	ClassCard  = "ui-card"  // 卡片：次级面 + 边框 + 圆角 + 内距
)

func init() { defineClasses() } // 启动即注册（Apply 之前用 Class 也安全）

// Class 返回命名样式类（薄封装 widget.Class，便于 companion 只 import ui 即可引用）。
func Class(name string) widget.Style { return widget.Class(name) }

// defineClasses 注册全部可复用样式类。类持稳定令牌指针，Apply 就地改色即自动随之换肤。
func defineClasses() {
	widget.Define(ClassRow, widget.Style{FlexDirection: "row", AlignItems: "center"})
	widget.Define(ClassCol, widget.Style{FlexDirection: "column", AlignItems: "stretch"})
	widget.Define(ClassPanel, widget.Style{
		FlexDirection: "column", AlignItems: "stretch", BackgroundColor: Bg,
	})
	widget.Define(ClassCard, widget.Style{
		FlexDirection: "column", AlignItems: "stretch",
		BackgroundColor: BgSubtle, BorderColor: Border, BorderWidth: 1,
		BorderRadius: Radius, Padding: types.EdgeInsets(12),
	})
}
