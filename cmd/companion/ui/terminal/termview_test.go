//go:build windows

package termpanel

import (
	"testing"

	"github.com/user/goui/pkg/event"
)

func TestKeyToVT(t *testing.T) {
	char := func(r rune) *event.KeyEvent {
		return &event.KeyEvent{BaseEvent: event.BaseEvent{EventType: event.TypeKeyChar}, Char: r}
	}
	keyd := func(key string, mods event.ModifierKeys) *event.KeyEvent {
		return &event.KeyEvent{BaseEvent: event.BaseEvent{EventType: event.TypeKeyDown}, Key: key, Mods: mods}
	}
	cases := []struct {
		ev   *event.KeyEvent
		want string
	}{
		{char('a'), "a"},
		{char('A'), "A"},
		{char('中'), "中"},
		{char(0x01), ""}, // 控制字符经 KeyChar → 不发（由 KeyDown 处理）
		{keyd("Enter", 0), "\r"},
		{keyd("Backspace", 0), "\x7f"},
		{keyd("Tab", 0), "\t"},
		{keyd("Escape", 0), "\x1b"},
		{keyd("ArrowUp", 0), "\x1b[A"},
		{keyd("ArrowLeft", 0), "\x1b[D"},
		{keyd("Delete", 0), "\x1b[3~"},
		{keyd("C", event.ModCtrl), "\x03"}, // Ctrl+C
		{keyd("c", event.ModCtrl), "\x03"},
		{keyd("D", event.ModCtrl), "\x04"}, // Ctrl+D
		{keyd("F1", 0), ""},                // 未映射
	}
	for _, c := range cases {
		if got := string(keyToVT(c.ev)); got != c.want {
			t.Errorf("keyToVT(type=%d key=%q char=%q mods=%d)=%q 期望 %q",
				c.ev.Type(), c.ev.Key, string(c.ev.Char), c.ev.Mods, got, c.want)
		}
	}
}
