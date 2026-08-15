package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunCodeNested run_code 嵌套工具调度：JS 代码内 tools.xxx 调用注册表工具。
func TestRunCodeNested(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello nested\nline2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	RegisterHarnessTools(reg, root)
	ctx := context.Background()

	// 代码里调 tools.read（嵌套调度）+ console.log
	code := `
const r = tools.read({path: "a.txt"});
console.log("READ_RESULT_HAS_HELLO=" + r.includes("hello"));
`
	got, err := reg.Execute(ctx, "run_code", `{"code":`+jsonQuote(code)+`,"language":"node"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"嵌套工具调用", "tools.read", "READ_RESULT_HAS_HELLO=true", "程序输出"} {
		if !strings.Contains(got, want) {
			t.Errorf("嵌套调度应含 %q：\n%s", want, got)
		}
	}

	// 嵌套错误：工具名不存在（goja TypeError——tools 对象只含已注册工具）
	code2 := `tools.nonexistent_tool({});`
	got2, err := reg.Execute(ctx, "run_code", `{"code":`+jsonQuote(code2)+`,"language":"node"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got2, "TypeError") && !strings.Contains(got2, "no member") {
		t.Errorf("未知工具应 TypeError：\n%s", got2)
	}

	// 语法错误
	code3 := `this is not js !!!`
	got3, err := reg.Execute(ctx, "run_code", `{"code":`+jsonQuote(code3)+`,"language":"node"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got3, "执行错误") && !strings.Contains(got3, "SyntaxError") {
		t.Errorf("语法错误应报告：\n%s", got3)
	}
}

// TestRunCodeNestedDisabled 不含 tools. 的 JS 仍走外部 node（原逻辑不受影响）。
func TestRunCodeNestedDisabled(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	RegisterHarnessTools(reg, root)
	ctx := context.Background()

	got, err := reg.Execute(ctx, "run_code", `{"code":"console.log(1+1)","language":"node"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("外部 node 应输出 2：\n%s", got)
	}
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
