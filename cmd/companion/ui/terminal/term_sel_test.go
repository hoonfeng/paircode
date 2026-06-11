//go:build windows

package termpanel

import (
	"testing"

	"github.com/hoonfeng/paircode/cmd/companion/vterm"
)

// TestTermCopySelection 拖选文本提取：首行从 colA、末行到 colB、中间整行，去行尾空格、行间换行。
func TestTermCopySelection(t *testing.T) {
	vt := vterm.New(40, 6)
	vt.Write([]byte("Hello terminal world\r\nDrag to select me here\r\nLine three of text\r\n"))

	// 跨行选区：行0 第6列 → 行1 第4列
	ts := &terminalState{vt: vt, hasSel: true, selAR: 0, selAC: 6, selCR: 1, selCC: 4}
	if got := ts.copySelection(); got != "terminal world\nDrag" {
		t.Errorf("跨行 copySelection=%q，期望 %q", got, "terminal world\nDrag")
	}
	// 反向拖（游标在锚点之前）应归一化得到同样文本
	ts.selAR, ts.selAC, ts.selCR, ts.selCC = 1, 4, 0, 6
	if got := ts.copySelection(); got != "terminal world\nDrag" {
		t.Errorf("反向 copySelection=%q，期望 %q", got, "terminal world\nDrag")
	}
	// 单行选区
	ts2 := &terminalState{vt: vt, hasSel: true, selAR: 2, selAC: 0, selCR: 2, selCC: 4}
	if got := ts2.copySelection(); got != "Line" {
		t.Errorf("单行 copySelection=%q，期望 %q", got, "Line")
	}
}

// TestCellInSel 选区命中判定（首行从 colA 起、末行到 colB 止、中间整行）。
func TestCellInSel(t *testing.T) {
	s := &vtSel{rowA: 0, colA: 6, rowB: 1, colB: 4}
	cases := []struct {
		r, c int
		want bool
	}{
		{0, 5, false}, {0, 6, true}, {0, 39, true}, // 首行 colA 起到行尾
		{1, 0, true}, {1, 3, true}, {1, 4, false}, // 末行起到 colB 止
		{2, 0, false}, // 选区外
	}
	for _, c := range cases {
		if got := cellInSel(c.r, c.c, s); got != c.want {
			t.Errorf("cellInSel(%d,%d)=%v，期望 %v", c.r, c.c, got, c.want)
		}
	}
}
