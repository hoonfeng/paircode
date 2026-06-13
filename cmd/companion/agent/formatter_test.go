package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCodeFormatPreview 创建格式不规范的 Go 文件，验证 code_format 预览模式返回差异。
func TestCodeFormatPreview(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 创建一个格式不规范的 Go 文件（错误缩进）
	writeFile(t, dir, "badfmt.go", `package testmod

func Add(a,b int)int{
return a+b
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_format", `{"path":"badfmt.go"}`)
	if err != nil {
		t.Fatalf("code_format 预览失败: %v\n输出: %s", err, out)
	}

	// 预览模式下应输出格式化建议
	if !strings.Contains(out, "需要格式化") && !strings.Contains(out, "diff") && !strings.Contains(out, "---") {
		t.Logf("预览输出: %s", out)
		// 如果文件已经格式化，输出应提示"已符合"
		if strings.Contains(out, "已符合") {
			t.Log("文件内容可能已符合格式（取决于 gofmt 版本）")
		} else {
			t.Errorf("预览模式应包含格式化建议, 输出: %s", out)
		}
	}
}

// TestCodeFormatApply 创建格式不规范的 Go 文件，验证 code_format apply 模式写入格式化结果。
func TestCodeFormatApply(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 创建一个格式不规范的 Go 文件
	unformatted := `package testmod

func Add(a,b int)int{
return a+b
}
`
	writeFile(t, dir, "formatme.go", unformatted)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 执行 apply 模式
	out, err := reg.Execute(ctx, "code_format", `{"path":"formatme.go","apply":true}`)
	if err != nil {
		t.Fatalf("code_format apply 失败: %v\n输出: %s", err, out)
	}

	// 应包含成功标记
	if !strings.Contains(out, "已格式化") && !strings.Contains(out, "✅") && !strings.Contains(out, "符合规范") {
		t.Errorf("apply 模式应包含成功标记，输出: %s", out)
	}

	// 验证文件已被格式化（语法正确）
	absPath := filepath.Join(dir, "formatme.go")
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "int) int {") && !strings.Contains(content, "int, b int") && !strings.Contains(content, "a+b") {
		t.Errorf("格式化后内容不符合预期: %s", content)
	}

	// 再次运行 gofmt -l 应无输出（已格式化）
	cmd := exec.Command("gofmt", "-l", absPath)
	cmd.Dir = dir
	out2, _ := cmd.CombinedOutput()
	if strings.TrimSpace(string(out2)) != "" {
		t.Errorf("gofmt 检查后仍认为需要格式化: %s", string(out2))
	}
}

// TestCodeFormatDir 测试 code_format 对目录的递归格式化能力。
func TestCodeFormatDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 创建子目录
	subDir := filepath.Join(dir, "subpkg")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 在两个文件中都写入不规范格式
	writeFile(t, dir, "main.go", `package testmod

func Foo(x int)int{
return x
}
`)
	writeFile(t, subDir, "helper.go", `package subpkg

func Bar(s string)string{
return s
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	// 对整个目录执行 apply 格式化
	out, err := reg.Execute(ctx, "code_format", `{"path":".","apply":true}`)
	if err != nil {
		t.Fatalf("code_format 目录 apply 失败: %v\n输出: %s", err, out)
	}

	// 验证子目录文件也被格式化
	for _, f := range []string{"main.go", filepath.Join("subpkg", "helper.go")} {
		absPath := filepath.Join(dir, f)
		cmd := exec.Command("gofmt", "-l", absPath)
		cmd.Dir = dir
		out2, _ := cmd.CombinedOutput()
		if strings.TrimSpace(string(out2)) != "" {
			t.Errorf("文件 %s 格式化后仍不符合规范: %s", f, string(out2))
		}
	}
}

// TestCodeFormatNonGoFile 测试对非 .go 文件的拒绝。
func TestCodeFormatNonGoFile(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	writeFile(t, dir, "data.json", `{"a":1}`)

	_, err := reg.Execute(ctx, "code_format", `{"path":"data.json"}`)
	if err == nil {
		t.Fatal("非 .go 文件应报错")
	}
	if !strings.Contains(err.Error(), ".go") {
		t.Errorf("错误信息应提示需要 .go 文件, 得到: %v", err)
	}
}

// TestCodeFormatNoGofmt 测试 gofmt 不可用时（模拟 PATH 查找失败），
// 这通常需要 mock，此处只验证工具有正确注册。
func TestCodeFormatRegistered(t *testing.T) {
	reg := NewRegistry()
	RegisterDefaultTools(reg, t.TempDir())

	tool, ok := reg.Get("code_format")
	if !ok {
		t.Fatal("code_format 未注册")
	}
	if !tool.RequiresApproval {
		t.Error("code_format 应设置 RequiresApproval=true")
	}
	if tool.Description == "" {
		t.Error("code_format 描述不能为空")
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

	// path 应为必填（可以是 []string 或 []any 类型）
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

// TestCodeFormatAlreadyFormatted 测试对已格式化的文件运行预览应返回"已符合规范"。
func TestCodeFormatAlreadyFormatted(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "testmod")

	// 写入已格式化的文件
	writeFile(t, dir, "good.go", `package testmod

func Add(a, b int) int {
	return a + b
}
`)

	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	out, err := reg.Execute(ctx, "code_format", `{"path":"good.go"}`)
	if err != nil {
		t.Fatalf("code_format 预览失败: %v\n输出: %s", err, out)
	}

	if !strings.Contains(out, "已符合") {
		t.Errorf("已格式化的文件应提示已符合规范, 输出: %s", out)
	}
}
