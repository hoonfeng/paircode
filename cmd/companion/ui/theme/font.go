//go:build windows

package theme

import (
	"github.com/user/gou-ide/cmd/companion/core"
	"github.com/user/goui/pkg/canvas"
)

// applyUIFont 把「界面字体」设置（字体族 + 粗/斜/下划线）应用到界面文字；空族=保持各处默认字号字体。
// 经 ui.SetFontHook(applyUIFont) 注入 ui 包（见 Apply），作用于所有 ui 文字组件。
func applyUIFont(f *canvas.Font) {
	if core.Settings.UIFontFamily != "" {
		f.Family = core.Settings.UIFontFamily
	}
	if core.Settings.UIFontBold {
		f.Weight = canvas.FontWeightBold
	}
	if core.Settings.UIFontItalic {
		f.Style = canvas.FontStyleItalic
	}
	if core.Settings.UIFontUnderline {
		f.Underline = true
	}
}
