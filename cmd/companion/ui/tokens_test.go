//go:build windows

package ui

import (
	"testing"

	"github.com/user/goui/pkg/types"
	"github.com/user/goui/pkg/widget"
)

// TestApplySwitchesTokens Apply 就地改令牌指针；样式类持稳定指针、绘制即见新色（换肤总入口）。
func TestApplySwitchesTokens(t *testing.T) {
	defer Apply(darkTokens()) // 测试后复原默认深色
	custom := darkTokens()
	custom.Bg = types.ColorFromRGB(1, 2, 3)
	Apply(custom)
	if *Bg != custom.Bg {
		t.Fatalf("Apply 未切换 *Bg, 得 %v", *Bg)
	}
	// ui-panel 类持稳定指针 Bg，就地改后该类背景指针应指向新色。
	got := widget.Class(ClassPanel)
	if got.BackgroundColor == nil || *got.BackgroundColor != custom.Bg {
		t.Errorf("ui-panel 背景未随新令牌更新")
	}
}

// TestApplyDefaults 缺省补全：Radius 零回落 6、OnAccent 全透明回落白。
func TestApplyDefaults(t *testing.T) {
	defer Apply(darkTokens())
	Apply(Tokens{}) // 全零
	if Radius != 6 {
		t.Errorf("Radius 零值应回落 6, 得 %v", Radius)
	}
	if OnAccent.A == 0 {
		t.Errorf("OnAccent 全透明应回落不透明白")
	}
}

// TestBuildersProduceWidgets helper 都能产出非 nil widget（基本冒烟）。
func TestBuildersProduceWidgets(t *testing.T) {
	noop := func() {}
	cases := map[string]widget.Widget{
		"PrimaryBtn": PrimaryBtn("保存", noop),
		"GhostBtn":   GhostBtn("取消", noop),
		"Toggle":     Toggle("开关", true, noop),
		"Field":      Field("标签", Input("", "", 0, nil)),
		"Card":       Card(Text("内容")),
		"Row":        Row(Text("a"), Spacer(), Text("b")),
		"Pill":       Pill("启用", "禁用", true, noop),
	}
	for name, w := range cases {
		if w == nil {
			t.Errorf("%s 产出 nil", name)
		}
	}
}
