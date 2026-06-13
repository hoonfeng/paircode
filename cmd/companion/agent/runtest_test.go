package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunTestPassing 创建包含通过测试的 Go 项目，运行 run_test 应返回成功。
func TestRunTestPassing(t *testing.T) {
	dir := t.TempDir()

	// 创建 Go 模块
	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "math.go", `package testmod

func Add(a, b int) int { return a + b }
`)
	writeFile(t, dir, "math_test.go", `package testmod

import "testing"

func TestAdd(t *testing.T) {
	if got := Add(1, 2); got != 3 {
		t.Fatalf("Add(1,2) = %d, want 3", got)
	}
}

func TestAddNegative(t *testing.T) {
	if got := Add(-1, -2); got != -3 {
		t.Fatalf("Add(-1,-2) = %d, want -3", got)
	}
}
`)

	// 初始化 go mod 并下载依赖（标准库无需下载，但 go.sum 需要）
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "run_test", `{"package_path":"."}`)
	if err != nil {
		t.Fatalf("run_test 失败: %v\n输出: %s", err, out)
	}

	// 应包含测试通过标志
	if !strings.Contains(out, "✅") {
		t.Errorf("应包含通过标志「✅」，输出: %s", out)
	}
	if !strings.Contains(out, "退出码 0") {
		t.Errorf("应包含退出码 0，输出: %s", out)
	}
	if !strings.Contains(out, "PASS") && !strings.Contains(out, "ok") {
		t.Errorf("应包含 PASS/ok 标记，输出: %s", out)
	}
	if !strings.Contains(out, "TestAdd") {
		t.Errorf("应包含测试名称 TestAdd，输出: %s", out)
	}
	if !strings.Contains(out, "TestAddNegative") {
		t.Errorf("应包含测试名称 TestAddNegative，输出: %s", out)
	}
}

// TestRunTestFailing 创建包含失败测试的 Go 项目，运行 run_test 应返回失败状态。
func TestRunTestFailing(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "fail_test.go", `package testmod

import "testing"

func TestAlwaysFails(t *testing.T) {
	t.Fatalf("预期失败")
}
`)

	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "run_test", `{"package_path":"."}`)
	// 测试失败时，错误输出应包含退出码信息，不视为 err（ExitError）
	if err != nil {
		t.Logf("run_test 返回 err=%v（可接受，因为测试失败）", err)
	}
	if out == "" {
		t.Fatal("输出不应为空")
	}
	// 应包含失败标志
	if !strings.Contains(out, "❌") {
		t.Errorf("应包含失败标志「❌」，输出: %s", out)
	}
	if !strings.Contains(out, "退出码") {
		t.Errorf("应包含退出码信息，输出: %s", out)
	}
	if !strings.Contains(out, "TestAlwaysFails") {
		t.Errorf("应包含测试名称 TestAlwaysFails，输出: %s", out)
	}
}

// TestRunTestPackageNotFound 包路径不存在时应报错。
func TestRunTestPackageNotFound(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "run_test", `{"package_path":"./nonexistent"}`)
	if err == nil {
		t.Fatal("不存在的包路径应报错")
	}
	if !strings.Contains(err.Error(), "不可访问") && !strings.Contains(err.Error(), "找不到") && !strings.Contains(err.Error(), "no such") {
		t.Logf("错误信息: %v", err)
	}
}

// TestRunTestNoGoFiles 包目录无 Go 文件时应报错。
func TestRunTestNoGoFiles(t *testing.T) {
	dir := t.TempDir()
	// 创建空目录
	os.MkdirAll(filepath.Join(dir, "emptypkg"), 0o755)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "run_test", `{"package_path":"./emptypkg"}`)
	if err == nil {
		t.Fatal("无 Go 文件的目录应报错")
	}
	if !strings.Contains(err.Error(), "没有 Go 源文件") {
		t.Errorf("错误信息应提示无 Go 文件，得到: %v", err)
	}
}

// TestRunTestWithPattern 使用 test_pattern 过滤应只运行匹配的测试。
func TestRunTestWithPattern(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "filter_test.go", `package testmod

import "testing"

func TestFoo(t *testing.T) {}
func TestBar(t *testing.T) {}
func TestBaz(t *testing.T) {}
`)

	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 只运行 TestFoo
	out, err := reg.Execute(ctx, "run_test", `{"package_path":".","test_pattern":"TestFoo"}`)
	if err != nil {
		t.Fatalf("run_test 失败: %v\n输出: %s", err, out)
	}
	if !strings.Contains(out, "TestFoo") {
		t.Errorf("输出应包含 TestFoo，输出: %s", out)
	}
	if strings.Contains(out, "TestBar") {
		t.Errorf("输出不应包含 TestBar（被过滤），输出: %s", out)
	}
	if strings.Contains(out, "TestBaz") {
		t.Errorf("输出不应包含 TestBaz（被过滤），输出: %s", out)
	}

	// 验证过滤参数出现在输出中
	if !strings.Contains(out, `-run "TestFoo"`) {
		t.Logf("输出应包含过滤模式标记（可选），输出: %s", out)
	}
}

// TestRunTestEmptyPackagePath 空 package_path 应报错。
func TestRunTestEmptyPackagePath(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "run_test", `{}`)
	if err == nil {
		t.Fatal("缺少 package_path 应报错")
	}
}

// ─── 辅助 ────────────────────────────────────────────────

func writeGoMod(t *testing.T, dir, modName string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+modName+"\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGoModTidy(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		// go mod tidy 可能在没网络时失败，但标准库测试不需要 external deps
		t.Logf("go mod tidy 输出: %s, err: %v", string(out), err)
	}
}
