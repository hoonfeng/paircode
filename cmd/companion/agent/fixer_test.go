package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodeFixPreview 测试 code_fix 预览模式在 Go 文件上的运行。
func TestCodeFixPreview(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 创建一个合法的 Go 文件
	writeFile(t, dir, "main.go", `package testmod

import "fmt"

func Greet(name string) string {
	return fmt.Sprintf("Hello, %s", name)
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_fix", `{"path":"main.go"}`)
	if err != nil {
		t.Fatalf("code_fix 预览失败: %v\n输出: %s", err, out)
	}

	// 预览模式下应返回正常结果（文件无问题或显示 diff）
	if !strings.Contains(out, "未发现需要修复") && !strings.Contains(out, "发现以下可自动修复") {
		t.Errorf("预览模式输出不符合预期: %s", out)
	}
}

// TestCodeFixApply 测试 code_fix apply 模式。
func TestCodeFixApply(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	writeFile(t, dir, "fixme.go", `package testmod

import "fmt"

func Add(a, b int) int {
	return a + b
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_fix", `{"path":"fixme.go","apply":true}`)
	if err != nil {
		t.Fatalf("code_fix apply 失败: %v\n输出: %s", err, out)
	}

	// apply 模式应返回成功标记
	if !strings.Contains(out, "已修复") && !strings.Contains(out, "未发现需要修复") {
		t.Errorf("apply 模式输出应包含成功标记，输出: %s", out)
	}

	// 验证文件内容未损坏（仍然是合法的 Go 代码）
	absPath := filepath.Join(dir, "fixme.go")
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "package testmod") {
		t.Errorf("文件内容被损坏: %s", content)
	}
}

// TestCodeFixDir 测试 code_fix 对目录的递归处理能力。
func TestCodeFixDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 创建子目录
	subDir := filepath.Join(dir, "subpkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, dir, "main.go", `package testmod

func Foo() int {
	return 42
}
`)
	writeFile(t, subDir, "helper.go", `package subpkg

func Bar() string {
	return "hello"
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_fix", `{"path":".","apply":true}`)
	if err != nil {
		t.Fatalf("code_fix 目录 apply 失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "已修复") && !strings.Contains(out, "未发现需要修复") {
		t.Errorf("目录模式输出不符合预期: %s", out)
	}
}

// TestCodeFixNonGoFile 测试对非 .go 文件的拒绝。
func TestCodeFixNonGoFile(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	writeFile(t, dir, "data.json", `{"a":1}`)

	_, err := reg.Execute(ctx, "code_fix", `{"path":"data.json"}`)
	if err == nil {
		t.Fatal("非 .go 文件应报错")
	}
	if !strings.Contains(err.Error(), ".go") {
		t.Errorf("错误信息应提示需要 .go 文件, 得到: %v", err)
	}
}

// TestCodeFixEmptyDir 测试空目录（无 Go 文件）的行为。
func TestCodeFixEmptyDir(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 空目录应该正常返回，不报错
	out, err := reg.Execute(ctx, "code_fix", `{"path":".","apply":true}`)
	if err != nil {
		t.Fatalf("空目录应正常处理，但得到错误: %v", err)
	}
	if !strings.Contains(out, "未发现需要修复") {
		t.Errorf("空目录应提示未发现需要修复，输出: %s", out)
	}
}

// TestCodeFixRegistered 验证 code_fix 工具已正确注册。
func TestCodeFixRegistered(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())

	tool, ok := reg.Get("code_fix")
	if !ok {
		t.Fatal("code_fix 未注册")
	}
	if !tool.RequiresApproval {
		t.Error("code_fix 应设置 RequiresApproval=true")
	}
	if tool.Description == "" {
		t.Error("code_fix 描述不能为空")
	}

	// 验证参数
	params := tool.Parameters
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("缺少 properties")
	}
	if _, ok := props["path"]; !ok {
		t.Error("缺少 path 参数")
	}
	if _, ok := props["apply"]; !ok {
		t.Error("缺少 apply 参数")
	}

	// path 应为必填
	reqRaw := params["required"]
	if reqRaw == nil {
		t.Error("缺少 required 字段")
	} else {
		reqLen := 0
		if req, ok := reqRaw.([]any); ok {
			reqLen = len(req)
		} else if req, ok := reqRaw.([]string); ok {
			reqLen = len(req)
		}
		if reqLen == 0 {
			t.Error("path 应设为必填参数")
		}
	}
}

// TestCodeFixAlreadyClean 测试已经干净的代码（无需修复）。
func TestCodeFixAlreadyClean(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 写入干净的 Go 文件
	writeFile(t, dir, "clean.go", `package testmod

import "fmt"

func PrintGreeting(name string) string {
	return fmt.Sprintf("Hi, %s!", name)
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_fix", `{"path":"clean.go"}`)
	if err != nil {
		t.Fatalf("code_fix 预览失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "未发现需要修复") {
		t.Errorf("干净的代码应提示无需修复, 输出: %s", out)
	}
}

// TestCodeFixPreviewOnDir 测试对目录执行预览模式。
func TestCodeFixPreviewOnDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	writeFile(t, dir, "hello.go", `package testmod

func Hello() string {
	return "world"
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_fix", `{"path":"."}`)
	if err != nil {
		t.Fatalf("code_fix 目录预览失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "未发现需要修复") && !strings.Contains(out, "发现以下可自动修复") {
		t.Errorf("目录预览模式输出不符合预期: %s", out)
	}
}
