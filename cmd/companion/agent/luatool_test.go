package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const sampleLuaTool = `
return {
  name = "word_count",
  description = "统计文本字数",
  parameters = { type="object", properties={ text={ type="string", description="文本" } }, required={"text"} },
  run = function(args)
    return "字数: " .. #(args.text or "")
  end,
}
`

// TestLuaToolLoadAndRun 加载 .lua 工具 → 注册（需审批）+ schema 透传 + 执行得结果。
func TestLuaToolLoadAndRun(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "wc.lua"), []byte(sampleLuaTool), 0o644)
	reg := NewRegistry()
	loaded := LoadLuaTools(reg, dir)
	if len(loaded) != 1 || loaded[0] != "word_count" {
		t.Fatalf("应注册 word_count，得 %v", loaded)
	}
	tool, ok := reg.Get("word_count")
	if !ok || !tool.RequiresApproval {
		t.Fatal("word_count 应注册且默认需审批")
	}
	if tool.Parameters["type"] != "object" {
		t.Errorf("参数 schema 应透传，得 %+v", tool.Parameters)
	}
	out, err := reg.Execute(context.Background(), "word_count", `{"text":"hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "字数: 5" {
		t.Errorf("执行结果错：%q", out)
	}
}

// TestLuaToolSandbox 沙箱禁止访问 os（os 为 nil）→ 执行出错，挡住越权。
func TestLuaToolSandbox(t *testing.T) {
	dir := t.TempDir()
	bad := `return { name="evil", parameters={type="object",properties={}}, run=function(args) return os.getenv("PATH") end }`
	os.WriteFile(filepath.Join(dir, "evil.lua"), []byte(bad), 0o644)
	reg := NewRegistry()
	LoadLuaTools(reg, dir)
	if _, err := reg.Execute(context.Background(), "evil", "{}"); err == nil {
		t.Error("沙箱应禁止访问 os（应执行出错）")
	}
}

// TestLuaToolBadScriptSkipped 语法错/缺 name 的脚本跳过，只注册合法的。
func TestLuaToolBadScriptSkipped(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "broken.lua"), []byte(`this is not lua {{{`), 0o644)
	os.WriteFile(filepath.Join(dir, "noname.lua"), []byte(`return { description="无名" }`), 0o644)
	os.WriteFile(filepath.Join(dir, "good.lua"), []byte(sampleLuaTool), 0o644)
	reg := NewRegistry()
	loaded := LoadLuaTools(reg, dir)
	if len(loaded) != 1 || loaded[0] != "word_count" {
		t.Errorf("坏脚本应跳过、只注册合法的，得 %v", loaded)
	}
}

// TestRegistryUnregister 动态卸载工具（Lua 热重载用）。
func TestRegistryUnregister(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{Name: "x", Handler: func(context.Context, map[string]any) (string, error) { return "", nil }})
	reg.Unregister("x")
	if _, ok := reg.Get("x"); ok {
		t.Error("Unregister 后不应存在")
	}
	for _, d := range reg.Definitions() {
		if d.Function.Name == "x" {
			t.Error("Definitions 仍含已移除工具")
		}
	}
}
