package agent

import (
	"strings"
	"testing"
)

// ─── 精确匹配 ───────────────────────────────────────────────

func TestApplyEdit_ExactMatch(t *testing.T) {
	content := "line1\nline2\nline3"
	out, err := ApplyEdit(content, EditOptions{OldString: "line2", NewString: "LINE2"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "line1\nLINE2\nline3"
	if out != want {
		t.Errorf("结果不匹配\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_ExactMatchMultiLine(t *testing.T) {
	content := "package main\n\nfunc foo() {\n\treturn\n}\n"
	old := "func foo() {\n\treturn\n}"
	out, err := ApplyEdit(content, EditOptions{OldString: old, NewString: "func foo() {\n\treturn 1\n}"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	if !strings.Contains(out, "return 1") {
		t.Errorf("多行替换失败: %q", out)
	}
	if strings.Contains(out, "return\n}") {
		t.Errorf("原文未被替换: %q", out)
	}
}

// ─── CRLF 归一化匹配 ────────────────────────────────────────

func TestApplyEdit_CRLFFileLFQuery(t *testing.T) {
	// 文件含 \r\n，LLM 给的 old_string 只含 \n
	content := "line1\r\nline2\r\nline3\r\n"
	out, err := ApplyEdit(content, EditOptions{OldString: "line2", NewString: "LINE2"})
	if err != nil {
		t.Fatalf("CRLF 文件 + LF 查询应匹配成功: %v", err)
	}
	// 替换后应保留 CRLF 风格
	want := "line1\r\nLINE2\r\nline3\r\n"
	if out != want {
		t.Errorf("CRLF 风格未保留\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_CRLFNewStringPreserved(t *testing.T) {
	// new_string 含 \n，替换进 CRLF 文件应转为 \r\n
	content := "a\r\nb\r\nc\r\n"
	out, err := ApplyEdit(content, EditOptions{OldString: "b", NewString: "x\ny"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "a\r\nx\r\ny\r\nc\r\n"
	if out != want {
		t.Errorf("new_string 换行未转为 CRLF\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_LFFileStaysLF(t *testing.T) {
	content := "a\nb\nc\n"
	out, err := ApplyEdit(content, EditOptions{OldString: "b", NewString: "x\ny"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "a\nx\ny\nc\n"
	if out != want {
		t.Errorf("LF 文件应保持 LF\n got: %q\nwant: %q", out, want)
	}
}

// ─── 空白折叠匹配 ───────────────────────────────────────────

func TestApplyEdit_WhitespaceFoldIndent(t *testing.T) {
	// 文件用 tab 缩进，LLM 给的 old_string 用空格
	content := "func foo() {\n\treturn 1\n}\n"
	// old_string 用 4 空格代替 tab
	out, err := ApplyEdit(content, EditOptions{OldString: "    return 1", NewString: "    return 2"})
	if err != nil {
		t.Fatalf("空白折叠应匹配 tab/space 差异: %v", err)
	}
	if !strings.Contains(out, "return 2") {
		t.Errorf("替换失败: %q", out)
	}
}

func TestApplyEdit_WhitespaceFoldTrailing(t *testing.T) {
	// 文件行尾有多余空格
	content := "line1   \nline2\nline3\n"
	out, err := ApplyEdit(content, EditOptions{OldString: "line1", NewString: "LINE1"})
	if err != nil {
		t.Fatalf("行尾空白差异应容忍: %v", err)
	}
	if !strings.Contains(out, "LINE1") {
		t.Errorf("替换失败: %q", out)
	}
}

func TestApplyEdit_WhitespaceFoldMultiSpace(t *testing.T) {
	// 文件用双空格，LLM 给单空格
	content := "foo  bar\nbaz\n"
	out, err := ApplyEdit(content, EditOptions{OldString: "foo bar", NewString: "qux"})
	if err != nil {
		t.Fatalf("连续空白折叠应匹配: %v", err)
	}
	if !strings.Contains(out, "qux") {
		t.Errorf("替换失败: %q", out)
	}
}

// ─── 行号定位模式 ───────────────────────────────────────────

func TestApplyEdit_LineModeSingleLine(t *testing.T) {
	content := "line1\nline2\nline3\n"
	out, err := ApplyEdit(content, EditOptions{LineStart: 2, NewString: "REPLACED"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "line1\nREPLACED\nline3\n"
	if out != want {
		t.Errorf("单行行号替换失败\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_LineModeRange(t *testing.T) {
	content := "l1\nl2\nl3\nl4\nl5\n"
	out, err := ApplyEdit(content, EditOptions{LineStart: 2, LineEnd: 4, NewString: "X\nY"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "l1\nX\nY\nl5\n"
	if out != want {
		t.Errorf("行段替换失败\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_LineModeWithOldStringValidation(t *testing.T) {
	content := "l1\nl2\nl3\n"
	// old_string 与行段内容匹配（空白容忍）
	out, err := ApplyEdit(content, EditOptions{LineStart: 2, OldString: "l2", NewString: "L2"})
	if err != nil {
		t.Fatalf("校验通过应成功: %v", err)
	}
	want := "l1\nL2\nl3\n"
	if out != want {
		t.Errorf("校验+替换失败\n got: %q\nwant: %q", out, want)
	}
}

func TestApplyEdit_LineModeOldStringMismatch(t *testing.T) {
	content := "l1\nl2\nl3\n"
	_, err := ApplyEdit(content, EditOptions{LineStart: 2, OldString: "WRONG", NewString: "L2"})
	if err == nil {
		t.Fatal("old_string 与行段不匹配应报错")
	}
}

func TestApplyEdit_LineModeCRLFPreserved(t *testing.T) {
	content := "l1\r\nl2\r\nl3\r\n"
	out, err := ApplyEdit(content, EditOptions{LineStart: 2, NewString: "L2"})
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	want := "l1\r\nL2\r\nl3\r\n"
	if out != want {
		t.Errorf("行号模式应保留 CRLF\n got: %q\nwant: %q", out, want)
	}
}

// ─── 多次出现 ───────────────────────────────────────────────

func TestApplyEdit_MultipleOccurrences(t *testing.T) {
	content := "foo\nbar\nfoo\n"
	_, err := ApplyEdit(content, EditOptions{OldString: "foo", NewString: "x"})
	if err == nil {
		t.Fatal("多次出现应报错")
	}
	if !strings.Contains(err.Error(), "多次") && !strings.Contains(err.Error(), "不唯一") {
		t.Errorf("错误信息应提示多次/不唯一: %v", err)
	}
}

// ─── 未找到 ─────────────────────────────────────────────────

func TestApplyEdit_NotFound(t *testing.T) {
	content := "line1\nline2\nline3\n"
	_, err := ApplyEdit(content, EditOptions{OldString: "NOTEXIST", NewString: "x"})
	if err == nil {
		t.Fatal("未找到应报错")
	}
	if !strings.Contains(err.Error(), "未找到") {
		t.Errorf("错误信息应提示未找到: %v", err)
	}
	// 诊断应含行号上下文
	if !strings.Contains(err.Error(), "L") {
		t.Errorf("诊断应含行号上下文: %v", err)
	}
}

// ─── 空字符串 ───────────────────────────────────────────────

func TestApplyEdit_EmptyOldString(t *testing.T) {
	content := "line1\nline2\n"
	_, err := ApplyEdit(content, EditOptions{OldString: "", NewString: "x"})
	if err == nil {
		t.Fatal("空 old_string 且无 line_start 应报错")
	}
}

// ─── 辅助函数测试 ───────────────────────────────────────────

func TestNormalizeNewlines(t *testing.T) {
	cases := map[string]string{
		"a\r\nb":     "a\nb",
		"a\rb":       "a\nb",
		"a\r\nb\r\nc": "a\nb\nc",
		"a\nb":       "a\nb",
	}
	for in, want := range cases {
		if got := normalizeNewlines(in); got != want {
			t.Errorf("normalizeNewlines(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRestoreNewlines(t *testing.T) {
	// ref 为 CRLF → 输出应转 CRLF
	out := restoreNewlines("a\nb\nc", "ref\r\n")
	if out != "a\r\nb\r\nc" {
		t.Errorf("CRLF 恢复失败: %q", out)
	}
	// ref 为 LF → 输出保持 LF
	out = restoreNewlines("a\nb\nc", "ref\n")
	if out != "a\nb\nc" {
		t.Errorf("LF 保持失败: %q", out)
	}
	// ref 含孤立 \r → 也应判断为 CRLF 风格
	out = restoreNewlines("a\nb\nc", "ref\r")
	if out != "a\r\nb\r\nc" {
		t.Errorf("孤立 \\r 应触发 CRLF 风格: %q", out)
	}
	// 空 ref → 保持 LF
	out = restoreNewlines("a\nb\nc", "")
	if out != "a\nb\nc" {
		t.Errorf("空 ref 应保持 LF: %q", out)
	}
}

func TestFoldWS(t *testing.T) {
	cases := map[string]string{
		"  hello   world  ": "hello world",
		"hello\t\tworld":    "hello world",
		"\t  spaced  \t":    "spaced",
		"single":            "single",
		"":                  "",
	}
	for in, want := range cases {
		if got := foldWS(in); got != want {
			t.Errorf("foldWS(%q) = %q, want %q", in, got, want)
		}
	}
}

// ─── 端到端：模拟 LLM 典型场景 ──────────────────────────────

func TestApplyEdit_Scenario_CRLFFile_TabIndent_LLMGivesSpaceLF(t *testing.T) {
	// 综合：CRLF 文件 + tab 缩进 + LLM 给空格缩进 + LF 换行
	content := "package main\r\n\r\nfunc foo() {\r\n\treturn 1\r\n}\r\n"
	old := "func foo() {\n    return 1\n}"
	new := "func foo() {\n    return 2\n}"
	out, err := ApplyEdit(content, EditOptions{OldString: old, NewString: new})
	if err != nil {
		t.Fatalf("综合场景应匹配成功: %v", err)
	}
	// 应保留 CRLF
	if !strings.Contains(out, "\r\n") {
		t.Errorf("应保留 CRLF: %q", out)
	}
	if !strings.Contains(out, "return 2") {
		t.Errorf("应含新内容: %q", out)
	}
	if strings.Contains(out, "return 1\r\n}") {
		t.Errorf("原文应被替换: %q", out)
	}
}
