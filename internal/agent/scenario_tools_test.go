package agent

// scenario_tools_test.go — 场景创造工具（scenario_scan/scenario_create）单测。
// 覆盖（2026-09-11）：
//   - 宿主注册（scenario_scan/scenario_create）；
//   - scan 盘点输出（插件/内置组分节）；
//   - create 组合创建（插件 + builtin 组 + 框架组条目）与落盘（临时全局目录）；
//   - 校验：非法名/预设名/未知插件/未知组/重复/白名单未知工具；
//   - 白名单 → DisabledTools；overwrite 覆盖重建。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkScenarioHost 构造场景创造测试宿主（全局工具集目录隔离 + 动态插件 test-tools）。
func mkScenarioHost(t *testing.T) (*PluginHost, *Registry, string) {
	t.Helper()
	root := t.TempDir()
	SetGlobalToolsetDirForTest(t.TempDir())
	t.Cleanup(func() { SetGlobalToolsetDirForTest(testGlobalToolsetDir) })
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	RegisterHarnessTools(reg, root)
	RegisterHostFrameworkTools(reg, root)
	ph := NewPluginHost(reg, nil, root)
	RegisterCordisTools(reg, ph, root)
	RegisterToolsetTools(reg, root, ph)
	RegisterScenarioTools(reg, root, ph)
	SetGlobalPluginHost(ph)
	t.Cleanup(func() { SetGlobalPluginHost(nil) })
	// 动态插件：提供组合测试用的插件工具（test_alpha/test_beta）
	code := `return { name: 'test-tools', apply(ctx) {
		ctx.tools.register({ name: 'test_alpha', description: '测试工具 A', parameters: { type: 'object', properties: {} }, execute: () => 'a' })
		ctx.tools.register({ name: 'test_beta', description: '测试工具 B', parameters: { type: 'object', properties: {} }, execute: () => 'b' })
	} }`
	id, err := ph.DefineJSCodeFull(code, "", "场景测试插件", "", "")
	if err != nil {
		t.Fatalf("DefineJSCodeFull: %v", err)
	}
	if err := ph.LoadJSDynamic(mustDef(t, ph, id)); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	return ph, reg, root
}

func TestRegisterScenarioTools(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	for _, name := range []string{"scenario_scan", "scenario_create"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("缺少工具 %s", name)
		}
	}
}

func TestScenarioScan(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	out, err := reg.Execute(context.Background(), "scenario_scan", `{}`)
	if err != nil {
		t.Fatalf("scenario_scan 执行失败: %v", err)
	}
	for _, want := range []string{"### 插件", "test-tools", "### 内置工具组", "builtin:core"} {
		if !strings.Contains(out, want) {
			t.Errorf("scan 输出缺少 %q\n%s", want, out)
		}
	}
	outFull, err := reg.Execute(context.Background(), "scenario_scan", `{"detail":"full"}`)
	if err != nil {
		t.Fatalf("scenario_scan full 执行失败: %v", err)
	}
	if !strings.Contains(outFull, "test_alpha（测试工具 A）") {
		t.Errorf("scan(full) 应附工具描述\n%s", outFull)
	}
}

func TestScenarioCreate(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	out, err := reg.Execute(context.Background(), "scenario_create",
		`{"name":"测试场景","description":"单测场景","plugins":["test-tools","builtin:core"]}`)
	if err != nil {
		t.Fatalf("scenario_create 执行失败: %v", err)
	}
	if !strings.Contains(out, "✅") || !strings.Contains(out, "测试场景") {
		t.Errorf("create 输出异常: %s", out)
	}
	// 落盘（安装目录语义：globalToolsetDir = 测试临时目录）
	path := filepath.Join(globalToolsetDir(), "测试场景.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("场景文件未落盘 %s: %v", path, err)
	}
	ts, err := loadToolset("", toolsetProject, "测试场景")
	if err != nil {
		t.Fatalf("loadToolset: %v", err)
	}
	names := map[string]ToolsetPlugin{}
	for _, p := range ts.Plugins {
		names[p.Name] = p
	}
	if _, ok := names["test-tools"]; !ok {
		t.Errorf("缺少插件条目 test-tools: %+v", ts.Plugins)
	}
	if p, ok := names["builtin:core"]; !ok || p.Builtin != "core" || len(p.Tools) == 0 {
		t.Errorf("builtin:core 条目异常: %+v", p)
	}
	// 框架组条目（system/plugin-mgmt/toolset-mgmt）自动附加
	for _, n := range []string{"builtin:system", "builtin:toolset-mgmt"} {
		if _, ok := names[n]; !ok {
			t.Errorf("缺少框架组条目 %s: %+v", n, ts.Plugins)
		}
	}
}

func TestScenarioCreateValidation(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	cases := []struct {
		desc string
		args string
		want string
	}{
		{"非法名（空格）", `{"name":"bad name","plugins":["test-tools"]}`, "不合法"},
		{"预设名保留", `{"name":"办公","plugins":["test-tools"]}`, "保留"},
		{"未知插件", `{"name":"x-scene","plugins":["no-such-plugin"]}`, "不存在或未装载"},
		{"未知内置组", `{"name":"x-scene","plugins":["builtin:no-such-group"]}`, "不存在"},
		{"空 plugins", `{"name":"x-scene","plugins":[]}`, "不能为空"},
		{"白名单未知工具", `{"name":"x-scene","plugins":[{"plugin":"test-tools","tools":["nope"]}]}`, "没有工具"},
	}
	for _, c := range cases {
		_, err := reg.Execute(context.Background(), "scenario_create", c.args)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s：期望错误含 %q，实际 %v", c.desc, c.want, err)
		}
	}
	// 重复创建（无 overwrite）被拒；overwrite=true 覆盖成功
	if _, err := reg.Execute(context.Background(), "scenario_create", `{"name":"重复场景","plugins":["test-tools"]}`); err != nil {
		t.Fatalf("首次创建失败: %v", err)
	}
	if _, err := reg.Execute(context.Background(), "scenario_create", `{"name":"重复场景","plugins":["test-tools"]}`); err == nil || !strings.Contains(err.Error(), "已存在") {
		t.Errorf("重复创建应被拒绝（加 overwrite），实际 %v", err)
	}
	if _, err := reg.Execute(context.Background(), "scenario_create", `{"name":"重复场景","plugins":["builtin:core"],"overwrite":"true"}`); err != nil {
		t.Fatalf("overwrite 覆建失败: %v", err)
	}
	ts, err := loadToolset("", toolsetProject, "重复场景")
	if err != nil {
		t.Fatalf("loadToolset: %v", err)
	}
	found := false
	for _, p := range ts.Plugins {
		if p.Builtin == "core" {
			found = true
		}
		if p.Name == "test-tools" {
			t.Errorf("overwrite 后不应保留旧条目 test-tools: %+v", ts.Plugins)
		}
	}
	if !found {
		t.Errorf("overwrite 后应含 builtin:core: %+v", ts.Plugins)
	}
}

func TestScenarioCreateWhitelist(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	_, err := reg.Execute(context.Background(), "scenario_create",
		`{"name":"白名单场景","plugins":[{"plugin":"test-tools","tools":["test_alpha"]}]}`)
	if err != nil {
		t.Fatalf("白名单创建失败: %v", err)
	}
	ts, err := loadToolset("", toolsetProject, "白名单场景")
	if err != nil {
		t.Fatalf("loadToolset: %v", err)
	}
	for _, p := range ts.Plugins {
		if p.Name != "test-tools" {
			continue
		}
		if len(p.DisabledTools) != 1 || p.DisabledTools[0] != "test_beta" {
			t.Errorf("白名单外工具应进 DisabledTools，实际 %v", p.DisabledTools)
		}
		return
	}
	t.Fatal("缺少 test-tools 条目")
}

// TestScenarioCreateEntryJSON 验证落盘 JSON 字段（name/description/builtinsInited）。
func TestScenarioCreateEntryJSON(t *testing.T) {
	_, reg, _ := mkScenarioHost(t)
	if _, err := reg.Execute(context.Background(), "scenario_create",
		`{"name":"json场景","description":"字段校验","plugins":["builtin:core"]}`); err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(globalToolsetDir(), "json场景.json"))
	if err != nil {
		t.Fatalf("读文件: %v", err)
	}
	var ts Toolset
	if err := json.Unmarshal(data, &ts); err != nil {
		t.Fatalf("解析 JSON: %v", err)
	}
	if ts.Name != "json场景" || ts.Description != "字段校验" || !ts.BuiltinsInited {
		t.Errorf("落盘字段异常: %+v", ts)
	}
	if ts.Version == "" {
		t.Errorf("version 不应为空")
	}
}
