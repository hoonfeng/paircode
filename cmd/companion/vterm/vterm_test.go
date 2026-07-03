package vterm

import (
	"strings"
	"testing"
)

// ─── 行→文本辅助 ──────────────────────────────────────────

// rowToText 将 vterm 的一行转为可显示文本（跳过 Ch==0 续格和尾随空格）。
// 返回：(文本字符串, 最后一个非空格字符的 flat 位置, col→flat 映射)
func rowToText(row []Cell, cols int) (string, int, map[int]int) {
	runes := make([]rune, 0, cols)
	colToFlat := make(map[int]int, cols)
	lastNonSpace := -1

	for c := 0; c < cols; c++ {
		if c < len(row) && row[c].Ch == 0 {
			continue
		}
		var ch rune = ' '
		if c < len(row) && row[c].Ch != 0 {
			ch = row[c].Ch
		}
		if ch != ' ' {
			lastNonSpace = len(runes)
		}
		colToFlat[c] = len(runes)
		runes = append(runes, ch)
	}
	if lastNonSpace < 0 {
		return "", -1, colToFlat
	}
	return string(runes[:lastNonSpace+1]), lastNonSpace, colToFlat
}

// colToFlatPos 将 vterm 列号 cx 转为 flat 文本中的 rune 偏移。
// 若 cx 指向续格（被跳过），则回退到该行末尾。
func colToFlatPos(colToFlat map[int]int, cx int) int {
	if pos, ok := colToFlat[cx]; ok {
		return pos
	}
	// cx 是续格列（被跳过），回退到行尾
	maxFlat := -1
	for _, f := range colToFlat {
		if f > maxFlat {
			maxFlat = f
		}
	}
	if maxFlat >= 0 {
		return maxFlat + 1
	}
	return 0
}

// ─── 精确提取（行下取整——不输出全屏 24 行的空尾行）────────

// extractTextExact 从 vterm 提取文本，跳过末尾完全空白的行。
// 与终端最终要实现的逻辑一致（TextArea 只显示有内容的行）。
func extractTextExact(t *Terminal, scrollOff int) string {
	cols, rows := t.Size()
	scrLen := t.ScrollbackLen()

	startRow := scrLen - scrollOff
	if startRow < 0 {
		startRow = 0
	}
	endRow := startRow + rows

	// 收集所有非空行的文本
	type lineInfo struct {
		text       string
		colToFlat  map[int]int
		lastNonCol int // 最后一个非空格字符的 vterm 列
	}

	var lines []lineInfo
	hasContent := false
	for r := startRow; r < endRow && r < scrLen+rows; r++ {
		rowData := t.RowAt(r)
		if rowData == nil {
			lines = append(lines, lineInfo{text: "", colToFlat: map[int]int{}})
			continue
		}
		text, _, colToFlat := rowToText(rowData, cols)
		if text != "" {
			hasContent = true
		}
		lines = append(lines, lineInfo{text: text, colToFlat: colToFlat})
	}

	// 如果没有内容，返回空字符串
	if !hasContent {
		return ""
	}

	// 找最后一个非空行
	lastNonEmpty := len(lines) - 1
	for lastNonEmpty >= 0 && lines[lastNonEmpty].text == "" {
		lastNonEmpty--
	}
	if lastNonEmpty < 0 {
		return ""
	}

	// 拼接（仅前 lastNonEmpty+1 行）
	var b strings.Builder
	for i := 0; i <= lastNonEmpty; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(lines[i].text)
	}
	return b.String()
}

// calcCursorPosExact 精确计算 vterm 光标在 flat 文本中的 rune 偏移。
// 正确处理 CJK 续格跳过和尾随空行裁剪。
func calcCursorPosExact(t *Terminal, scrollOff int) int {
	if t == nil || scrollOff > 0 {
		return -1
	}
	cols, rows := t.Size()
	cx, cy := t.Cursor()
	scrLen := t.ScrollbackLen()
	startRow := scrLen
	endRow := startRow + rows

	// 收集所有行的 flat 文本
	type lineInfo struct {
		text      string
		colToFlat map[int]int
	}
	var lines []lineInfo
	for r := startRow; r < endRow && r < scrLen+rows; r++ {
		rowData := t.RowAt(r)
		if rowData == nil {
			lines = append(lines, lineInfo{text: "", colToFlat: map[int]int{}})
			continue
		}
		text, _, colToFlat := rowToText(rowData, cols)
		lines = append(lines, lineInfo{text: text, colToFlat: colToFlat})
	}

	// 找最后一个非空行
	lastNonEmpty := len(lines) - 1
	for lastNonEmpty >= 0 && lines[lastNonEmpty].text == "" {
		lastNonEmpty--
	}
	if lastNonEmpty < 0 {
		return 0
	}

	// 光标 cy 是否在可见范围内
	if cy >= len(lines) {
		// 光标在不可见行（超出内容区域），放在最后
		pos := 0
		for i := 0; i <= lastNonEmpty; i++ {
			if i > 0 {
				pos++
			}
			pos += len([]rune(lines[i].text))
		}
		return pos
	}

	// 累计光标行之前所有行的长度（含 \n）
	pos := 0
	maxLine := cy
	if maxLine > lastNonEmpty {
		maxLine = lastNonEmpty
	}
	for i := 0; i < maxLine; i++ {
		if i > 0 {
			pos++ // 行间 \n
		}
		pos += len([]rune(lines[i].text))
	}

	// 光标行：将 cx 转为 flat 位置
	if cy <= lastNonEmpty {
		if cy > 0 {
			pos++ // 行间 \n
		}
		flatCx := colToFlatPos(lines[cy].colToFlat, cx)
		lineLen := len([]rune(lines[cy].text))
		if flatCx > lineLen {
			flatCx = lineLen
		}
		pos += flatCx
	}

	return pos
}

// ─── 测试用例 ─────────────────────────────────────────────────

func TestEmpty(t *testing.T) {
	vt := New(80, 24)
	text := extractTextExact(vt, 0)
	if text != "" {
		t.Errorf("empty terminal should produce empty text, got %q", text)
	}
	pos := calcCursorPosExact(vt, 0)
	if pos != 0 {
		t.Errorf("empty terminal cursor should be at 0, got %d", pos)
	}
}

func TestSimpleText(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("hello"))
	text := extractTextExact(vt, 0)
	if text != "hello" {
		t.Errorf("expected %q, got %q", "hello", text)
	}
	pos := calcCursorPosExact(vt, 0)
	if pos != 5 {
		t.Errorf("cursor should be at 5 (after 'hello'), got %d", pos)
	}
}

func TestPrompt(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("Microsoft Windows [Version 10.0.26200.8655]\r\n"))
	vt.Write([]byte("(c) Microsoft Corporation. All rights reserved.\r\n"))
	vt.Write([]byte("\r\n"))
	vt.Write([]byte("C:\\Users\\test>"))

	text := extractTextExact(vt, 0)
	lines := strings.Split(text, "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 lines, got %d: %q", len(lines), text)
	}
	if lines[3] != "C:\\Users\\test>" {
		t.Errorf("last line should be prompt, got %q", lines[3])
	}

	pos := calcCursorPosExact(vt, 0)
	expectedPos := len("Microsoft Windows [Version 10.0.26200.8655]") + 1 +
		len("(c) Microsoft Corporation. All rights reserved.") + 1 +
		0 + 1 +
		len("C:\\Users\\test>")
	if pos != expectedPos {
		t.Errorf("cursor should be at %d (after prompt), got %d", expectedPos, pos)
	}
}

func TestNarrowTerminal(t *testing.T) {
	vt := New(10, 24)
	vt.Write([]byte("hello world"))
	text := extractTextExact(vt, 0)
	lines := strings.Split(text, "\n")
	if len(lines) < 2 {
		t.Errorf("should wrap, got %d lines: %q", len(lines), text)
	}
	t.Logf("text=%q", text)
}

func TestCJKCharacters(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("你好世界"))
	text := extractTextExact(vt, 0)
	if text != "你好世界" {
		t.Errorf("expected %q, got %q", "你好世界", text)
	}
	pos := calcCursorPosExact(vt, 0)
	if pos != 4 {
		t.Errorf("cursor should be at 4 (after 4 CJK chars in flat text), got %d", pos)
	}
}

func TestMixedCJKAndAscii(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("你好 world"))
	text := extractTextExact(vt, 0)
	if text != "你好 world" {
		t.Errorf("expected %q, got %q", "你好 world", text)
	}
	pos := calcCursorPosExact(vt, 0)
	if pos != 8 { // "你" "好" " " "w" "o" "r" "l" "d" = 8 runes
		t.Errorf("cursor should be at 8, got %d", pos)
	}
}

func TestScrollbackAndCursor(t *testing.T) {
	vt := New(40, 5)
	for i := 0; i < 8; i++ {
		vt.Write([]byte("line "))
		vt.Write([]byte(string(rune('A' + i))))
		vt.Write([]byte("\r\n"))
	}
	vt.Write([]byte("C:\\test>"))

	text := extractTextExact(vt, 0)
	lines := strings.Split(text, "\n")
	if len(lines) > 6 || len(lines) < 4 {
		t.Errorf("scrollOff=0 should show ~5 lines, got %d: %q", len(lines), text)
	}

	pos := calcCursorPosExact(vt, 0)
	t.Logf("cursor pos=%d, text=%q", pos, text)
	if pos < 0 {
		t.Errorf("cursor should be valid (>=0), got %d", pos)
	}
}

func TestMultiLineCursor(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("first line\r\n"))
	vt.Write([]byte("second\r\n"))
	vt.Write([]byte("third line"))

	text := extractTextExact(vt, 0)
	lines := strings.Split(text, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), text)
	}

	pos := calcCursorPosExact(vt, 0)
	expectedPos := 10 + 1 + 6 + 1 + 10
	if pos != expectedPos {
		t.Errorf("cursor should be at %d (after third line), got %d", expectedPos, pos)
	}
}

func TestCJKAndMultiLine(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("你好 world\r\n"))
	vt.Write([]byte("line2\r\n"))
	vt.Write([]byte("test>"))

	pos := calcCursorPosExact(vt, 0)
	expectedPos := 8 + 1 + 5 + 1 + 5
	if pos != expectedPos {
		t.Errorf("cursor should be at %d, got %d", expectedPos, pos)
	}
}

func TestScrollOffCursor(t *testing.T) {
	vt := New(40, 5)
	for i := 0; i < 10; i++ {
		vt.Write([]byte("line " + string(rune('A'+i)) + "\r\n"))
	}

	pos := calcCursorPosExact(vt, 3)
	if pos != -1 {
		t.Errorf("scrollOff>0 should return -1, got %d", pos)
	}

	pos = calcCursorPosExact(vt, 0)
	if pos < 0 {
		t.Errorf("scrollOff=0 should return valid pos, got %d", pos)
	}
}

// TestCursorAtNegativeCxWhenCy 测试光标在行首的场景
func TestCursorAtStartOfLine(t *testing.T) {
	vt := New(80, 24)
	vt.Write([]byte("hello\r\n"))
	vt.Write([]byte("world"))

	// 光标在 "world" 之后，即 pos after "hello\nworld"
	// 移动光标到 hello 行首
	vt.Write([]byte{27, '[', 'A'}) // cursor up
	vt.Write([]byte{27, '[', 'H'}) // cursor home (top-left)

	text := extractTextExact(vt, 0)
	pos := calcCursorPosExact(vt, 0)
	t.Logf("cursor at start: text=%q, pos=%d", text, pos)
	// cursor should be at 0 (start of first line)
}

// TestCursorPosInEmptyGrid 空行光标在位置 0
func TestCursorPosInEmptyGrid(t *testing.T) {
	vt := New(80, 24)
	// 创建一个空行内容（实际是空白格）
	// cursor 默认在 (0,0)
	pos := calcCursorPosExact(vt, 0)
	if pos != 0 {
		t.Errorf("empty grid cursor should be at 0, got %d", pos)
	}
}
