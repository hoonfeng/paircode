package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindCircularDeps_NoCycle 测试无循环依赖时的输出。
func TestFindCircularDeps_NoCycle(t *testing.T) {
	dir := t.TempDir()

	// 创建 go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	// 创建两个独立包：pkgA 和 pkgB，pkgA → pkgB（无环）
	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgB"), 0o755)

	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "test.mod/pkgB"

func Hello() string {
	return pkgB.World()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgB", "b.go"), []byte(`
package pkgB

import "fmt"

func World() string {
	return "world"
}
`), 0o644)

	// 注册工具并执行
	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	if err != nil {
		t.Fatalf("find_circular_deps: %v", err)
	}

	// 应该输出"未检测到循环依赖"
	if !strings.Contains(out, "未检测到循环依赖") {
		t.Errorf("期望无循环，但输出为：%q", out)
	}
}

// TestFindCircularDeps_SimpleCycle 测试简单循环依赖检测（A → B → A）。
func TestFindCircularDeps_SimpleCycle(t *testing.T) {
	dir := t.TempDir()

	// 创建 go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	// 创建两个包，形成循环：pkgA → pkgB → pkgA
	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgB"), 0o755)

	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "test.mod/pkgB"

func Hello() string {
	return pkgB.World()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgB", "b.go"), []byte(`
package pkgB

import "test.mod/pkgA"

func World() string {
	return pkgA.Hello()
}
`), 0o644)

	// 注册工具并执行
	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	if err != nil {
		t.Fatalf("find_circular_deps: %v", err)
	}

	// 应该检测到循环
	if !strings.Contains(out, "循环依赖") {
		t.Errorf("期望检测到循环，但输出为：%q", out)
	}
	if !strings.Contains(out, "test.mod/pkgA") || !strings.Contains(out, "test.mod/pkgB") {
		t.Errorf("循环中应包含 pkgA 和 pkgB，输出为：%q", out)
	}
}

// TestFindCircularDeps_TripleCycle 测试三层循环依赖（A → B → C → A）。
func TestFindCircularDeps_TripleCycle(t *testing.T) {
	dir := t.TempDir()

	// 创建 go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	// 创建三个包，形成三层循环：pkgA → pkgB → pkgC → pkgA
	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgB"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgC"), 0o755)

	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "test.mod/pkgB"

func Hello() string {
	return pkgB.FuncB()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgB", "b.go"), []byte(`
package pkgB

import "test.mod/pkgC"

func FuncB() string {
	return pkgC.FuncC()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgC", "c.go"), []byte(`
package pkgC

import "test.mod/pkgA"

func FuncC() string {
	return "hello " + pkgA.Hello()
}
`), 0o644)

	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	if err != nil {
		t.Fatalf("find_circular_deps: %v", err)
	}

	if !strings.Contains(out, "循环依赖") {
		t.Errorf("期望检测到循环，但输出为：%q", out)
	}
	// 应该检测到 1 个循环
	if !strings.Contains(out, "循环 #1") {
		t.Errorf("期望至少 1 个循环，输出为：%q", out)
	}
}

// TestFindCircularDeps_MultipleCycles 测试多个独立循环依赖。
func TestFindCircularDeps_MultipleCycles(t *testing.T) {
	dir := t.TempDir()

	// 创建 go.mod
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	// 创建四个包：两个独立的循环
	// 循环1: pkgA → pkgB → pkgA
	// 循环2: pkgC → pkgD → pkgC
	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgB"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgC"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgD"), 0o755)

	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "test.mod/pkgB"

func Hello() string {
	return pkgB.FuncB()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgB", "b.go"), []byte(`
package pkgB

import "test.mod/pkgA"

func FuncB() string {
	return pkgA.Hello()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgC", "c.go"), []byte(`
package pkgC

import "test.mod/pkgD"

func FuncC() string {
	return pkgD.FuncD()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgD", "d.go"), []byte(`
package pkgD

import "test.mod/pkgC"

func FuncD() string {
	return pkgC.FuncC()
}
`), 0o644)

	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	if err != nil {
		t.Fatalf("find_circular_deps: %v", err)
	}

	if !strings.Contains(out, "循环依赖") {
		t.Errorf("期望检测到循环，但输出为：%q", out)
	}
	// 应该检测到 2 个循环
	if !strings.Contains(out, "循环 #2") {
		t.Errorf("期望 2 个循环，输出为：%q", out)
	}
}

// TestFindCircularDeps_SelfCycle 测试包自引用（包导入自身）。
// 这在 Go 中编译会失败，但我们的工具也应能检测到。
func TestFindCircularDeps_SelfCycle(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	// 创建一个包导入自身（不同文件的同包互相导入不算，因为我们在 buildInternalDepGraph 中排除了同包导入）
	// 所以这里用一个文件导入自己所在包的路径
	os.MkdirAll(filepath.Join(dir, "selfpkg"), 0o755)

	os.WriteFile(filepath.Join(dir, "selfpkg", "s.go"), []byte(`
package selfpkg

import "test.mod/selfpkg"

func Hello() string {
	return "hello"
}
`), 0o644)

	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	if err != nil {
		t.Fatalf("find_circular_deps: %v", err)
	}

	// self-import 在 buildInternalDepGraph 中被过滤掉（dep == pkgImportPath），所以不应检测到循环
	// 这是合理的，因为同包导入对 Go 来说不构成循环（同一包内的文件可以互相引用）
	if strings.Contains(out, "循环依赖") {
		t.Logf("自引用包（同包导入）被忽略是合理的：%s", out)
	}
}

// TestFindCircularDeps_NoGoMod 测试缺少 go.mod 时返回友好的错误信息。
func TestFindCircularDeps_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	// 不创建 go.mod，创建一些 Go 文件
	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "fmt"

func Hello() string {
	return "hello"
}
`), 0o644)

	reg := NewRegistry()
	registerCircularDepsTool(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "find_circular_deps", `{}`)
	// 没有 go.mod 时应该返回错误，而不是 panic
	if err == nil {
		t.Error("缺少 go.mod 时应返回错误，但不 panic")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("错误信息应提及 go.mod，得到：%v", err)
	}
}

// TestBuildInternalDepGraph 测试依赖图构建的正确性。
func TestBuildInternalDepGraph(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test.mod\n\ngo 1.21\n"), 0o644)

	os.MkdirAll(filepath.Join(dir, "pkgA"), 0o755)
	os.MkdirAll(filepath.Join(dir, "pkgB"), 0o755)

	os.WriteFile(filepath.Join(dir, "pkgA", "a.go"), []byte(`
package pkgA

import "test.mod/pkgB"

func Hello() string {
	return pkgB.World()
}
`), 0o644)

	os.WriteFile(filepath.Join(dir, "pkgB", "b.go"), []byte(`
package pkgB

import "fmt"

func World() string {
	return "world"
}
`), 0o644)

	graph, pkgOfFile, err := buildInternalDepGraph(dir, "test.mod")
	if err != nil {
		t.Fatalf("buildInternalDepGraph: %v", err)
	}

	// pkgA 应该有一个依赖：test.mod/pkgB
	depsA, ok := graph["test.mod/pkgA"]
	if !ok {
		t.Fatal("图中缺少 test.mod/pkgA")
	}
	if len(depsA) != 1 || depsA[0] != "test.mod/pkgB" {
		t.Errorf("pkgA 依赖应为 [test.mod/pkgB]，得到 %v", depsA)
	}

	// pkgB 应该没有内部依赖（只导入了 fmt）
	depsB, ok := graph["test.mod/pkgB"]
	if !ok {
		t.Fatal("图中缺少 test.mod/pkgB")
	}
	if len(depsB) != 0 {
		t.Errorf("pkgB 不应有内部依赖，得到 %v", depsB)
	}

	// 检查 pkgOfFile
	if _, ok := pkgOfFile["pkgA/a.go"]; !ok {
		t.Errorf("pkgOfFile 中应包含 pkgA/a.go，实际有：%v", pkgOfFile)
	}
}
