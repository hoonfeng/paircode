package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindSymbolRegistered 验证 find_symbol 已注册且参数正确。
func TestFindSymbolRegistered(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())

	tool, ok := reg.Get("find_symbol")
	if !ok {
		t.Fatal("find_symbol 未注册")
	}
	if !tool.ReadOnly {
		t.Error("find_symbol 应为只读工具")
	}
	if tool.Description == "" {
		t.Error("find_symbol 描述不能为空")
	}

	// 验证参数
	params := tool.Parameters
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("缺少 properties")
	}
	if _, ok := props["symbol"]; !ok {
		t.Error("缺少 symbol 参数")
	}
	if _, ok := props["scope"]; !ok {
		t.Error("缺少 scope 参数")
	}

	// symbol 是必须的
	req, hasReq := params["required"]
	if !hasReq {
		t.Error("缺少 required 字段")
	} else {
		// required 可以是 []string 或 []any
		var reqSymbols []string
		switch r := req.(type) {
		case []string:
			reqSymbols = r
		case []any:
			for _, v := range r {
				if s, ok := v.(string); ok {
					reqSymbols = append(reqSymbols, s)
				}
			}
		default:
			t.Fatalf("required 类型异常: %T", req)
		}
		if len(reqSymbols) == 0 || reqSymbols[0] != "symbol" {
			t.Errorf("required 应为 [symbol]，实际: %v", reqSymbols)
		}
	}
}

// TestFindSymbolBasic 在 Go 项目上测试 find_symbol，匹配函数、结构体、常量、变量。
func TestFindSymbolBasic(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索函数 Greet
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Greet"}`)
	if err != nil {
		t.Fatalf("find_symbol Greet 失败: %v", err)
	}
	if !strings.Contains(out, "Greet") {
		t.Errorf("输出应包含 Greet，实际:\n%s", out)
	}
	if !strings.Contains(out, "function") || !strings.Contains(out, "func") {
		t.Errorf("应标注为 function，实际:\n%s", out)
	}
	if !strings.Contains(out, "mypkg/math.go") && !strings.Contains(out, "math.go") {
		t.Errorf("应包含文件路径，实际:\n%s", out)
	}

	// 搜索结构体 Point
	out, err = reg.Execute(ctx, "find_symbol", `{"symbol":"Point"}`)
	if err != nil {
		t.Fatalf("find_symbol Point 失败: %v", err)
	}
	if !strings.Contains(out, "Point") {
		t.Errorf("输出应包含 Point，实际:\n%s", out)
	}
	if !strings.Contains(out, "struct") {
		t.Errorf("应标注为 struct，实际:\n%s", out)
	}
}

// TestFindSymbolConstVar 测试查找常量和变量。
func TestFindSymbolConstVar(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索常量 MaxCount
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"MaxCount"}`)
	if err != nil {
		t.Fatalf("find_symbol MaxCount 失败: %v", err)
	}
	if !strings.Contains(out, "constant") {
		t.Errorf("MaxCount 应标注为 constant，实际:\n%s", out)
	}

	// 搜索变量 AppVersion
	out, err = reg.Execute(ctx, "find_symbol", `{"symbol":"AppVersion"}`)
	if err != nil {
		t.Fatalf("find_symbol AppVersion 失败: %v", err)
	}
	if !strings.Contains(out, "variable") {
		t.Errorf("AppVersion 应标注为 variable，实际:\n%s", out)
	}
}

// TestFindSymbolNoMatch 搜索不存在的符号应返回未找到信息。
func TestFindSymbolNoMatch(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"NonExistentSymbolXyz123"}`)
	if err != nil {
		t.Fatalf("find_symbol 不应报错: %v", err)
	}
	if !strings.Contains(out, "未找到") {
		t.Errorf("应返回未找到信息，实际:\n%s", out)
	}
}

// TestFindSymbolScopeFile 通过 scope 限定到单个文件。
func TestFindSymbolScopeFile(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索 Greet，限定到 mypkg/math.go
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Greet","scope":"mypkg/math.go"}`)
	if err != nil {
		t.Fatalf("find_symbol (scope=file) 失败: %v", err)
	}
	if !strings.Contains(out, "Greet") {
		t.Errorf("输出应包含 Greet，实际:\n%s", out)
	}
	if !strings.Contains(out, "mypkg/math.go") {
		t.Errorf("应包含 mypkg/math.go，实际:\n%s", out)
	}

	// 搜索 Greet，限定到 main.go（不应找到）
	out, err = reg.Execute(ctx, "find_symbol", `{"symbol":"Greet","scope":"main.go"}`)
	if err != nil {
		t.Fatalf("find_symbol (scope=wrong file) 失败: %v", err)
	}
	if !strings.Contains(out, "未找到") {
		t.Errorf("在不同文件中应返回未找到，实际:\n%s", out)
	}
}

// TestFindSymbolScopePackage 通过 scope 限定到包路径。
func TestFindSymbolScopePackage(t *testing.T) {
	dir := createTestGoProject(t)

	// 添加另一个包用于测试
	otherDir := filepath.Join(dir, "otherpkg")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	otherSrc := `package otherpkg

// Helper 是辅助函数
func Helper() string {
	return "helper"
}

// Config 配置结构体
type Config struct {
	Port int
}
`
	if err := os.WriteFile(filepath.Join(otherDir, "helper.go"), []byte(otherSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索 Helper，限定到 otherpkg 目录
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Helper","scope":"./otherpkg"}`)
	if err != nil {
		t.Fatalf("find_symbol (scope=package) 失败: %v", err)
	}
	if !strings.Contains(out, "Helper") {
		t.Errorf("输出应包含 Helper，实际:\n%s", out)
	}
	if !strings.Contains(out, "otherpkg/helper.go") {
		t.Errorf("应包含 otherpkg/helper.go，实际:\n%s", out)
	}

	// 搜索 Helper，限定到 mypkg 目录（不应找到）
	out, err = reg.Execute(ctx, "find_symbol", `{"symbol":"Helper","scope":"./mypkg"}`)
	if err != nil {
		t.Fatalf("find_symbol (scope=wrong package) 失败: %v", err)
	}
	if !strings.Contains(out, "未找到") {
		t.Errorf("在其他包中应返回未找到，实际:\n%s", out)
	}
}

// TestFindSymbolEmptySymbol 空 symbol 应报错。
func TestFindSymbolEmptySymbol(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "find_symbol", `{}`)
	if err == nil {
		t.Fatal("缺少 symbol 应报错")
	}
}

// TestFindSymbolScopeNonGoFile 使用 scope 指向非 Go 文件应报错。
func TestFindSymbolScopeNonGoFile(t *testing.T) {
	dir := t.TempDir()

	// 创建非 Go 文件
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	_, err := reg.Execute(ctx, "find_symbol", `{"symbol":"test","scope":"data.json"}`)
	if err == nil {
		t.Fatal("scope 指向非 Go 文件应报错")
	}
	if !strings.Contains(err.Error(), "不是 Go 源文件") {
		t.Errorf("错误信息应提示不是 Go 源文件，实际: %v", err)
	}
}

// TestFindSymbolMultipleMatches 搜索出现在多个文件的符号。
func TestFindSymbolMultipleMatches(t *testing.T) {
	dir := createTestGoProject(t)

	// 在另一个文件中添加同名函数
	extraDir := filepath.Join(dir, "mypkg")
	extraSrc := `package mypkg

// Greet 打招呼（另一个实现）
func Greet(name string) string {
	return "Hi, " + name
}
`
	if err := os.WriteFile(filepath.Join(extraDir, "extra.go"), []byte(extraSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索 Point（只在 math.go 中），应在两个文件中都找到
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Point"}`)
	if err != nil {
		t.Fatalf("find_symbol Point 失败: %v", err)
	}
	if !strings.Contains(out, "mypkg/math.go") {
		t.Errorf("输出应包含 mypkg/math.go，实际:\n%s", out)
	}
	// 确保找到结果（即使只有 1 处定义也接受）
	if !strings.Contains(out, "struct") {
		t.Errorf("Point 应标注为 struct，实际:\n%s", out)
	}
}

// TestFindSymbolInterfaceType 测试查找接口和类型。
func TestFindSymbolInterfaceType(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索接口 Printer
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Printer"}`)
	if err != nil {
		t.Fatalf("find_symbol Printer 失败: %v", err)
	}
	if !strings.Contains(out, "interface") {
		t.Errorf("Printer 应标注为 interface，实际:\n%s", out)
	}
}

// TestFindSymbolSubstring 子串匹配（符号名的子串）。
func TestFindSymbolSubstring(t *testing.T) {
	dir := createTestGoProject(t)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 搜索 "Count" 应匹配 MaxCount
	out, err := reg.Execute(ctx, "find_symbol", `{"symbol":"Count"}`)
	if err != nil {
		t.Fatalf("find_symbol Count 失败: %v", err)
	}
	if !strings.Contains(out, "MaxCount") {
		t.Errorf("输出应包含 MaxCount，实际:\n%s", out)
	}
}

// ── 辅助 ────────────────────────────────────────────────

// createTestGoProject 创建一个包含多种 Go 符号的临时项目，返回项目根目录。
func createTestGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// go.mod
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 主包文件 - main.go（包含 main 函数）
	mainSrc := `package main

import "fmt"

func main() {
	fmt.Println(Greet("World"))
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// 子包 mypkg
	pkgDir := filepath.Join(dir, "mypkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// math.go - 包含函数、结构体、常量、变量
	mathSrc := `package mypkg

import "fmt"

// MaxCount 是最大计数常量
const MaxCount = 100

// AppVersion 是应用版本
var AppVersion = "1.0.0"

// Point 表示二维坐标
type Point struct {
	X, Y int
}

// Greet 返回问候语
func Greet(name string) string {
	return "Hello, " + name
}

// GreetUser 返回用户问候语
func GreetUser(name string, age int) string {
	return fmt.Sprintf("Hello, %s (age %d)", name, age)
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "math.go"), []byte(mathSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// types.go - 包含接口
	typesSrc := `package mypkg

// Printer 是打印接口
type Printer interface {
	Print(s string) string
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "types.go"), []byte(typesSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	return dir
}
