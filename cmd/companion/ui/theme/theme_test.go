//go:build windows

package theme

import (
	"testing"

	"github.com/hoonfeng/paircode/cmd/companion/ui"
	"github.com/hoonfeng/goui/pkg/types"
)

func colEq(c types.Color, r, g, b uint8) bool { return c.R == r && c.G == g && c.B == b }

// TestApplyTheme 切主题把核心颜色令牌灌进 ui 单套调色板（内容 ui.Bg / 外壳 ui.ShellEditor / 语义 ui.Success）；未知名回退 dark；可无损切回 dark。
func TestApplyTheme(t *testing.T) {
	defer Apply("dark") // 复位，免影响其它测试（全局令牌共享）

	Apply("light")
	if !colEq(*ui.Bg, 255, 255, 255) {
		t.Errorf("light 主背景应白，得 %+v", *ui.Bg)
	}
	if !colEq(*ui.Fg, 0x1f, 0x23, 0x28) {
		t.Errorf("light 主文字应深 #1f2328，得 %+v", *ui.Fg)
	}
	if !colEq(*ui.ShellEditor, 255, 255, 255) { // 外壳编辑区随 bg-primary
		t.Errorf("light 外壳编辑区应随主背景，得 %+v", *ui.ShellEditor)
	}
	if !colEq(*ui.Success, 0x1a, 0x7f, 0x37) { // git 语义色随 success
		t.Errorf("light *ui.Success 应随 success #1a7f37，得 %+v", *ui.Success)
	}

	Apply("dracula")
	if !colEq(*ui.Bg, 0x28, 0x2a, 0x36) {
		t.Errorf("dracula 主背景应 #282a36，得 %+v", *ui.Bg)
	}
	if !colEq(*ui.Accent, 0xbd, 0x93, 0xf9) {
		t.Errorf("dracula 强调应紫 #bd93f9，得 %+v", *ui.Accent)
	}

	Apply("不存在的主题") // 回退 dark
	if !colEq(*ui.Bg, 0x0d, 0x11, 0x17) {
		t.Errorf("未知主题应回退 dark #0d1117，得 %+v", *ui.Bg)
	}

	Apply("dark")
	if !colEq(*ui.Bg, 13, 17, 23) || !colEq(*ui.Fg, 230, 237, 243) || !colEq(*ui.ShellEditor, 30, 30, 30) {
		t.Errorf("切回 dark 应恢复原值，得 bg=%+v text=%+v editor=%+v", *ui.Bg, *ui.Fg, *ui.ShellEditor)
	}
}
