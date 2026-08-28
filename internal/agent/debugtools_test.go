package agent

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestDebugToolsRegistered 验证新的通用调试工具已注册到 Registry。
func TestDebugToolsRegistered(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)

	expected := []string{
		"debug_inject_log",
		"debug_run_capture",
		"debug_analyze_output",
		"debug_parse_stack",
		"debug_cleanup_logs",
		"debug_watch",
		"debug_evaluate_session",
	}

	for _, name := range expected {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("调试工具 %q 未注册", name)
		}
	}
}

// TestDebugInjectLogNoFile 验证缺少 file 报错。
func TestDebugInjectLogNoFile(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_inject_log", `{}`)
	if err == nil {
		t.Error("缺少 file 应报错")
	}
}

// TestDebugInjectLogEmptyFile 验证空 file 报错。
func TestDebugInjectLogEmptyFile(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_inject_log", `{"file": "", "lines": [1]}`)
	if err == nil {
		t.Error("空 file 应报错")
	}
}

// TestDebugInjectLogEmptyLines 验证空 lines 报错。
func TestDebugInjectLogEmptyLines(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_inject_log", `{"file": "test.go", "lines": []}`)
	if err == nil {
		t.Error("空 lines 应报错")
	}
}

// TestDebugInjectLogFileNotExist 验证不存在的文件报错。
func TestDebugInjectLogFileNotExist(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_inject_log", `{"file": "nonexistent.go", "lines": [1]}`)
	if err == nil {
		t.Error("不存在的文件应报错")
	}
}

// TestDebugRunCaptureNoCommand 验证缺少命令报错。
func TestDebugRunCaptureNoCommand(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_run_capture", `{}`)
	if err == nil {
		t.Error("缺少 command 应报错")
	}
}

// TestDebugAnalyzeOutputNoText 验证缺少输出报错。
func TestDebugAnalyzeOutputNoText(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_analyze_output", `{}`)
	if err == nil {
		t.Error("缺少 output 应报错")
	}
}

// TestDebugParseStackNoText 验证缺少文本报错。
func TestDebugParseStackNoText(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "debug_parse_stack", `{}`)
	if err == nil {
		t.Error("缺少 text 应报错")
	}
}

// TestDebugCleanupLogsWithoutInjection 验证无注入时的清理。
func TestDebugCleanupLogsWithoutInjection(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	// 创建一个临时文件，不含注入日志
	tmpFile := root + "/test.go"
	if err := os.WriteFile(tmpFile, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Execute(ctx, "debug_cleanup_logs", `{"file": "test.go"}`)
	if err != nil {
		t.Errorf("清理无注入文件不应报错: %v", err)
	}
	if !strings.Contains(out, "没有找到") {
		t.Errorf("预期 '没有找到'，得到: %s", out)
	}
}

// TestDebugWatchToolRegistered 验证 debug_watch 的基本注册和报错。
func TestDebugWatchToolRegistered(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	// 缺少 pattern 报错
	_, err := reg.Execute(ctx, "debug_watch", `{"command": "echo hi"}`)
	if err == nil {
		t.Error("缺少 pattern 应报错")
	}

	// 缺少 command 报错
	_, err = reg.Execute(ctx, "debug_watch", `{"pattern": "*.go"}`)
	if err == nil {
		t.Error("缺少 command 应报错")
	}
}

// TestDebugToolNamesInDefinitions 验证新调试工具出现在工具定义列表中。
func TestDebugToolNamesInDefinitions(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)

	defs := reg.Definitions()
	newTools := []string{
		"debug_inject_log",
		"debug_run_capture",
		"debug_analyze_output",
		"debug_parse_stack",
		"debug_cleanup_logs",
		"debug_watch",
		"debug_evaluate_session",
	}

	for _, name := range newTools {
		found := false
		for _, d := range defs {
			if d.Function.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q 未出现在工具定义列表中", name)
		}
	}
}

// TestDebugEvaluateSession 验证评分工具基本功能。
func TestDebugEvaluateSession(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	// 无日志时不应报错
	out, err := reg.Execute(ctx, "debug_evaluate_session", `{}`)
	if err != nil {
		t.Errorf("无日志时不应报错: %v", err)
	}
	if !strings.Contains(out, "未找到执行日志") {
		t.Errorf("预期提示无日志，得到: %s", out)
	}
}

// TestAnalyzeOutput 测试输出分析函数。
func TestAnalyzeOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		hasErr bool
	}{
		{"正常输出", "Hello World\n", false},
		{"含错误", "Error: something went wrong\n", true},
		{"含 panic", "panic: index out of range\n", true},
		{"含异常", "Exception in thread main\n", true},
		{"空输出", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzeOutputText(tt.output)
			if analysis.HasError != tt.hasErr {
				t.Errorf("HasError = %v, 期望 %v", analysis.HasError, tt.hasErr)
			}
		})
	}
}

// TestParseStackFrame 测试堆栈帧解析。
func TestParseStackFrame(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		lang  string
		file  string
		lineN int
	}{
		{"Go堆栈", "main.foo() /path/file.go:10 +0x100", "Go", "/path/file.go", 10},
		{"Python堆栈", `  File "main.py", line 42, in my_func`, "Python", "main.py", 42},
		{"JS堆栈", "  at myFunc (/app/src/index.ts:25:10)", "JS/TS", "/app/src/index.ts", 25},
		{"Java堆栈", "  at com.example.Foo.bar(Foo.java:88)", "Java", "Foo.java", 88},
		{"Rust堆栈", "  at /src/main.rs:15:5", "JS/TS", "/src/main.rs", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames := tryParseStackFrame(tt.line)
			if len(frames) == 0 {
				t.Fatalf("未能解析: %s", tt.line)
			}
			f := frames[0]
			if f.Lang != tt.lang {
				t.Errorf("Lang = %s, 期望 %s", f.Lang, tt.lang)
			}
			if f.File != tt.file {
				t.Errorf("File = %s, 期望 %s", f.File, tt.file)
			}
			if f.Line != tt.lineN {
				t.Errorf("Line = %d, 期望 %d", f.Line, tt.lineN)
			}
		})
	}
}

// TestSessionScore 测试评分逻辑。
func TestSessionScore(t *testing.T) {
	log := &ExecutionLog{
		Entries: []ExecutionEntry{
			{Round: 1, Agent: "outer", Phase: "analysis", Summary: "分析用户需求"},
			{Round: 2, Agent: "outer", Phase: "execution", Summary: "执行 read_file"},
			{Round: 3, Agent: "outer", Phase: "execution", Summary: "任务完成"},
		},
	}
	score := evaluateSession(log, nil)

	if !score.Completed {
		t.Error("应标记为已完成")
	}
	if score.CompletionScore <= 0 {
		t.Error("完成度评分应大于 0")
	}
	if score.OverallScore <= 0 {
		t.Error("总分应大于 0")
	}
}
