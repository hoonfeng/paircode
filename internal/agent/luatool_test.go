package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestLuaToolSandbox 沙箱禁止危险 os 函数（execute/remove/rename），但允许安全函数（time/date/clock/getenv）。
func TestLuaToolSandbox(t *testing.T) {
	dir := t.TempDir()
	// 危险函数应被阻止
	evil1 := `return { name="evil1", parameters={type="object",properties={}}, run=function(args) return os.execute("dir") or "ok" end }`
	os.WriteFile(filepath.Join(dir, "evil1.lua"), []byte(evil1), 0o644)
	evil2 := `return { name="evil2", parameters={type="object",properties={}}, run=function(args) return os.remove("test.txt") or "ok" end }`
	os.WriteFile(filepath.Join(dir, "evil2.lua"), []byte(evil2), 0o644)
	// 安全函数应正常工作
	safe := `return { name="safe", parameters={type="object",properties={}}, run=function(args) return os.time() .. "|" .. os.date("%Y") end }`
	os.WriteFile(filepath.Join(dir, "safe.lua"), []byte(safe), 0o644)
	reg := NewRegistry()
	LoadLuaTools(reg, dir)
	if _, err := reg.Execute(context.Background(), "evil1", "{}"); err == nil {
		t.Error("os.execute 应被禁用")
	}
	if _, err := reg.Execute(context.Background(), "evil2", "{}"); err == nil {
		t.Error("os.remove 应被禁用")
	}
	// 安全函数应正常执行
	out, err := reg.Execute(context.Background(), "safe", "{}")
	if err != nil {
		t.Fatalf("os.time/os.date 应可访问，得错误: %v", err)
	}
	if out == "" || out == "(无返回)" {
		t.Errorf("os.time/os.date 应返回结果，得 %q", out)
	}
}

// TestLuaToolAgentBridge 测试 agent 桥接函数（json_encode/json_decode/timestamp/log/env）。
func TestLuaToolAgentBridge(t *testing.T) {
	dir := t.TempDir()
	script := `return { name="bridge_test", parameters={type="object",properties={msg={type="string"}}},
	  run = function(args)
		local ts = agent.timestamp()
		local encoded = agent.json_encode({a=1, b="hello"})
		local decoded = agent.json_decode(encoded)
		local logline = agent.log("info", "测试消息: " .. (args.msg or ""))
		local path = agent.env("PATH")
		return "ts=" .. ts .. "|encoded=" .. encoded .. "|decoded_b=" .. decoded.b .. "|log=" .. logline .. "|path_len=" .. #path
	  end
	}`
	os.WriteFile(filepath.Join(dir, "bridge.lua"), []byte(script), 0o644)
	reg := NewRegistry()
	LoadLuaTools(reg, dir)
	out, err := reg.Execute(context.Background(), "bridge_test", `{"msg":"hello"}`)
	if err != nil {
		t.Fatalf("桥接函数应正常执行，得错误: %v", err)
	}
	if !containsAll(out, "ts=", "encoded=", "decoded_b=hello", "log=", "path_len=") {
		t.Errorf("桥接函数结果不完整: %q", out)
	}
}

func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
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
