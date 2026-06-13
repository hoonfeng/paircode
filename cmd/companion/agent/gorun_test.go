package agent

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestGoRunPassing 创建合法的 Go 项目，go_run 应成功。
func TestGoRunPassing(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "main.go", `package main

func main() {
	println("hello from go_run")
}
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "go_run", `{"path":"."}`)
	if err != nil {
		t.Fatalf("go_run 失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "✅") {
		t.Errorf("应包含成功标志「✅」，输出: %s", out)
	}
	if !strings.Contains(out, "退出码 0") {
		t.Errorf("应包含退出码 0，输出: %s", out)
	}
	if !strings.Contains(out, "hello from go_run") {
		t.Errorf("应包含程序输出「hello from go_run」，输出: %s", out)
	}
}

// TestGoRunWithArgs 传递 args 给 Go 程序应能收到。
func TestGoRunWithArgs(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "main.go", `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("args:", os.Args[1:])
}
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "go_run", `{"path":".","args":["hello","world","42"]}`)
	if err != nil {
		t.Fatalf("go_run 失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "✅") {
		t.Errorf("应包含成功标志「✅」，输出: %s", out)
	}
	if !strings.Contains(out, "[hello world 42]") {
		t.Errorf("应包含传递的参数「[hello world 42]」，输出: %s", out)
	}
}

// TestGoRunFailure 包含编译错误的 Go 文件，go_run 应失败。
func TestGoRunFailure(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "broken.go", `package main

func main() {
	println("hello"
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "go_run", `{"path":"."}`)
	// 编译失败时 handler 层错误被捕获，返回的错误可能是包装后的消息
	if err != nil {
		t.Logf("go_run 返回 err=%v（可接受，因为编译错误）", err)
	}
	if out == "" {
		t.Fatal("输出不应为空")
	}

	if !strings.Contains(out, "❌") && !strings.Contains(out, "失败") {
		t.Errorf("应包含失败标志，输出: %s", out)
	}
}

// TestGoRunInvalidPath 路径不存在时应报错。
func TestGoRunInvalidPath(t *testing.T) {
	dir := t.TempDir()

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "go_run", `{"path":"./nonexistent"}`)
	if err == nil {
		t.Fatal("不存在的路径应报错")
	}
	if !strings.Contains(err.Error(), "不可访问") && !strings.Contains(err.Error(), "找不到") && !strings.Contains(err.Error(), "no such") {
		t.Logf("错误信息: %v", err)
	}
}

// TestGoRunDefaultPath 省略 path 参数时使用默认值 "."。
func TestGoRunDefaultPath(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "main.go", `package main

func main() {
	println("default path test")
}
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 不传 path 应默认使用 .
	out, err := reg.Execute(ctx, "go_run", `{}`)
	if err != nil {
		t.Fatalf("go_run 失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "✅") {
		t.Errorf("应包含成功标志「✅」，输出: %s", out)
	}
	if !strings.Contains(out, "default path test") {
		t.Errorf("应包含程序输出，输出: %s", out)
	}
}

// TestGoRunPathIsFile path 指向一个 Go 文件而不是目录时应能运行。
func TestGoRunPathIsFile(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "hello.go", `package main

func main() {
	println("file path test")
}
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// path 指向一个 Go 文件
	out, err := reg.Execute(ctx, "go_run", `{"path":"hello.go"}`)
	if err != nil {
		t.Fatalf("go_run 失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "✅") {
		t.Errorf("应包含成功标志「✅」，输出: %s", out)
	}
	if !strings.Contains(out, "file path test") {
		t.Errorf("应包含程序输出，输出: %s", out)
	}
}

// TestGoRunEmptyDir path 指向没有 Go 源文件的目录时应报错。
func TestGoRunEmptyDir(t *testing.T) {
	dir := t.TempDir()

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 创建空目录
	emptyDir := dir + string(os.PathSeparator) + "emptydir"
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := reg.Execute(ctx, "go_run", `{"path":"emptydir"}`)
	if err == nil {
		t.Fatal("空目录应报错")
	}
	if !strings.Contains(err.Error(), "没有 Go 源文件") {
		t.Errorf("错误信息应提示没有 Go 源文件，得到: %v", err)
	}
}

// TestGoRunWithExitCode 程序以非零退出码退出时，go_run 应反映退出码。
func TestGoRunWithExitCode(t *testing.T) {
	dir := t.TempDir()

	writeGoMod(t, dir, "testmod")
	writeFile(t, dir, "main.go", `package main

import "os"

func main() {
	os.Exit(42)
}
`)
	runGoModTidy(t, dir)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "go_run", `{"path":"."}`)
	// os.Exit(42) 导致 go run 退出码非零，但 handler 内部捕获了 ExitError
	if err != nil {
		t.Logf("go_run 返回 err=%v（可接受，因为程序非零退出）", err)
	}
	if out == "" {
		t.Fatal("输出不应为空")
	}

	if !strings.Contains(out, "❌") && !strings.Contains(out, "42") {
		t.Errorf("应包含失败标志和退出码 42，输出: %s", out)
	}
}
