// toolset_test.go — 工具集（Toolset）动态构建/固化/导出/导入/市场安装验证。
package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// mkToolsetGoProject 造一个 Go 项目临时目录（go.mod + main.go）。
func mkToolsetGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module demo\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestToolsetBuildPersistExportImport 全链路：构建 → 固化 → 列表 → 导出 → 导入全局 → 启动装载。
func TestToolsetBuildPersistExportImport(t *testing.T) {
	project := mkToolsetGoProject(t)

	host := NewPluginHost(NewRegistry(), nil, project)
	RegisterBuiltinPlugins(host) // 含 toolset-tpl-core（模板插件）
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	// 1. 构建（模板驱动）
	ts, err := BuildToolset(host, project, "dev", "", "Go 项目开发辅助")
	if err != nil {
		t.Fatalf("BuildToolset: %v", err)
	}
	if len(ts.Plugins) == 0 {
		t.Fatal("构建应生成插件")
	}
	if ts.Project == "" {
		t.Fatal("Project 应为项目 basename")
	}
	// 模板命中：通用模板应命中（git-flow）
	names := map[string]bool{}
	for _, p := range ts.Plugins {
		names[p.Name] = true
	}
	if !names["git-flow"] {
		t.Fatalf("git-flow 模板应命中（通用）: %v", names)
	}

	// 2. 固化到工作区 .pair/toolsets/
	if err := saveToolset(project, toolsetProject, ts); err != nil {
		t.Fatalf("saveToolset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".pair", "toolsets", ts.Name+".json")); err != nil {
		t.Fatalf("固化文件应存在: %v", err)
	}
	metas := listToolsets(project, toolsetProject)
	if len(metas) != 1 || metas[0].Name != "dev" {
		t.Fatalf("project 列表应含 dev: %+v", metas)
	}

	// 3. 插件已装载（工具可见）
	if got := host.State("git-flow"); got != PluginRunning {
		t.Fatalf("git-flow 应 running, got %v", got)
	}

	// 4. 导出（发布 JSON）
	content, err := ExportToolsetJSON(ts, []string{"go"}, "test", "github:demo/repo")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	var pub ToolsetPublish
	if err := json.Unmarshal([]byte(content), &pub); err != nil {
		t.Fatalf("导出 JSON 解析失败: %v", err)
	}
	if pub.Kind != "toolset" || len(pub.Toolset.Plugins) == 0 {
		t.Fatal("导出结构异常")
	}

	// 5. 导入到全局（新宿主模拟另一工作区启动自动装载）
	host2 := NewPluginHost(NewRegistry(), nil, project)
	RegisterBuiltinPlugins(host2)
	if err := importToolsetJSON(host2, project, content, "user"); err != nil {
		t.Fatalf("importToolsetJSON: %v", err)
	}
	if _, err := os.Stat(filepath.Join(core.InstallDir(), ".pair", "toolsets", ts.Name+".json")); err != nil {
		t.Fatalf("全局固化应存在: %v", err)
	}
	// 新宿主立即装载成功
	if got := host2.State("git-flow"); got != PluginRunning {
		t.Fatalf("导入后 git-flow 应 running, got %v", got)
	}

	// 6. LoadAllToolsets（模拟启动自动装配）——已有工具集再次装载不报错
	LoadAllToolsets(host2, project)

	// 7. 删除
	if err := removeToolset(project, toolsetProject, ts.Name); err != nil {
		t.Fatalf("removeToolset: %v", err)
	}
	if err := removeToolset(project, toolsetGlobal, ts.Name); err != nil {
		t.Fatalf("removeToolset global: %v", err)
	}
}

// TestToolsetMarketInstall 市场 plugin 类型安装（固化 + 装载）。
func TestToolsetMarketInstall(t *testing.T) {
	project := mkToolsetGoProject(t)
	host := NewPluginHost(NewRegistry(), nil, project)
	RegisterBuiltinPlugins(host)
	SetGlobalPluginHost(host)
	defer SetGlobalPluginHost(nil)

	entry := MarketFind("plugin-go-dev")
	if entry == nil || entry.Kind != "plugin" {
		t.Fatalf("内置注册表应有 plugin-go-dev 插件条目")
	}
	msg, err := MarketInstallScoped("plugin-go-dev", false, "project")
	if err != nil {
		t.Fatalf("MarketInstallScoped: %v", err)
	}
	if !strings.Contains(msg, "go-dev") {
		t.Fatalf("安装消息异常: %s", msg)
	}
	// 已固化到工作区
	if _, err := os.Stat(filepath.Join(project, ".pair", "toolsets", "go-dev.json")); err != nil {
		t.Fatalf("插件工具集应固化到工作区: %v", err)
	}
	// 已装载
	if got := host.State("go-dev-project-helper"); got != PluginRunning {
		t.Fatalf("go-dev-project-helper 应 running, got %v", got)
	}
	if got := host.State("git-flow"); got != PluginRunning {
		t.Fatalf("git-flow 应 running, got %v", got)
	}
}

// TestToolsetJSTemplate JS 插件注册工具集模板（ctx.toolset）→ 构建收集。
func TestToolsetJSTemplate(t *testing.T) {
	project := mkToolsetGoProject(t)
	host := NewPluginHost(NewRegistry(), nil, project)
	RegisterBuiltinPlugins(host)

	// 用 JS 插件注册一个专属模板（市场/用户扩展工具集构建处理）
	_, err := host.DefineJSCodeFull(`
  return {
    name: 'my-tpl',
    apply(ctx) {
      ctx.toolset.registerTemplate({
        id: 'toolset.tpl.my-helper',
        title: '自定义助手',
        match: (profile) => profile.langs.includes('go'),
        generate: (profile, requirement) => {
          const code = 'return {\n' +
            "  name: 'my-helper',\n" +
            "  inject: ['bash'],\n" +
            '  apply(ctx) {\n' +
            "    ctx.tools.register({\n" +
            "      name: 'my_hello',\n" +
            "      description: '自定义助手工具',\n" +
            "      parameters: { type: 'object', properties: {} },\n" +
            "      execute: async () => 'hello from my-helper'\n" +
            '    });\n' +
            '  }\n' +
            '};';
          return [{
            name: 'my-helper',
            purpose: '自定义模板生成的助手（' + (requirement || '通用') + '）',
            code: code
          }];
        }
      });
    }
  };
`, "", "JS 模板插件", "", "")
	if err != nil {
		t.Fatalf("define 模板插件: %v", err)
	}
	defs := host.JSDefs()
	if err := host.LoadJSDynamic(defs[len(defs)-1]); err != nil {
		t.Fatalf("load 模板插件: %v", err)
	}
	// 模板已注册
	if host.Template("toolset.tpl.my-helper") == nil {
		t.Fatal("JS 模板应已注册")
	}

	// 构建收集到 JS 模板生成的插件
	ts, err := BuildToolset(host, project, "jsdev", "", "测试要求")
	if err != nil {
		t.Fatalf("BuildToolset: %v", err)
	}
	found := false
	for _, p := range ts.Plugins {
		if p.Name == "my-helper" && strings.Contains(p.Purpose, "测试要求") {
			found = true
		}
	}
	if !found {
		t.Fatalf("JS 模板生成的插件应包含（含 requirement 透传）: %+v", ts.Plugins)
	}
	// JS 模板生成的插件可装载
	if got := host.State("my-helper"); got != PluginRunning {
		t.Fatalf("my-helper 应 running, got %v", got)
	}
}

// importToolsetJSON 工具集导入（测试辅助，复用 toolset_import 逻辑）。
func importToolsetJSON(ph *PluginHost, projectRoot, content, scope string) error {
	var pub ToolsetPublish
	if err := json.Unmarshal([]byte(content), &pub); err != nil {
		return err
	}
	ts := &pub.Toolset
	if ts.Name == "" || len(ts.Plugins) == 0 {
		return &jsonErr{"导入内容不是有效工具集"}
	}
	s := toolsetProject
	if scope == "user" {
		s = toolsetGlobal
	}
	if err := saveToolset(projectRoot, s, ts); err != nil {
		return err
	}
	return installToolset(ph, ts)
}

type jsonErr struct{ msg string }

func (e *jsonErr) Error() string { return e.msg }

// TestToolsetEdit 工具集手动编辑（插件化思路）：add_plugin / rm_tool / enable_tool / rm_plugin。
func TestToolsetEdit(t *testing.T) {
	project := mkToolsetGoProject(t)
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, project)
	RegisterBuiltinPlugins(host)

	// 1. 造初始工具集（含插件 p1-base）并固化+装载
	p1Code := `return {
  name: 'p1-base',
  apply(ctx) {
    ctx.tools.register({ name: 'p1_tool', description: 'p1 工具', parameters: { type: 'object', properties: {} }, execute: async () => 'p1' })
  }
}`
	ts := &Toolset{
		Name: "edit-demo",
		Plugins: []ToolsetPlugin{
			{Name: "p1-base", Purpose: "基础插件", Code: p1Code},
		},
	}
	if err := saveToolset(project, toolsetProject, ts); err != nil {
		t.Fatalf("saveToolset: %v", err)
	}
	if err := installToolset(host, ts); err != nil {
		t.Fatalf("installToolset: %v", err)
	}
	if _, ok := reg.Get("p1_tool"); !ok {
		t.Fatal("p1_tool 应已注册")
	}

	// 2. 宿主定义一个 JS 动态插件 p2-ext（含两个工具）
	p2Code := `return {
  name: 'p2-ext',
  apply(ctx) {
    ctx.tools.register({ name: 'p2_tool_a', description: 'p2 a', parameters: { type: 'object', properties: {} }, execute: async () => 'a' })
    ctx.tools.register({ name: 'p2_tool_b', description: 'p2 b', parameters: { type: 'object', properties: {} }, execute: async () => 'b' })
  }
}`
	defID, err := host.DefineJSCodeFull(p2Code, "", "扩展插件", "", "")
	if err != nil {
		t.Fatalf("DefineJSCodeFull: %v", err)
	}
	def, _ := host.GetJSDef(defID)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	if _, ok := reg.Get("p2_tool_a"); !ok {
		t.Fatal("p2_tool_a 应已注册")
	}

	// 3. add_plugin：把宿主插件 p2-ext 收编进工具集（即时装载 + 回写 JSON）
	msg, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "add_plugin", "plugin_name": "p2-ext"})
	if err != nil {
		t.Fatalf("add_plugin: %v", err)
	}
	if !strings.Contains(msg, "p2-ext") {
		t.Fatalf("消息应含 p2-ext: %s", msg)
	}
	ts2, err := loadToolset(project, toolsetProject, "edit-demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts2.Plugins) != 2 {
		t.Fatalf("插件数应为 2: %d", len(ts2.Plugins))
	}
	data, _ := os.ReadFile(filepath.Join(project, ".pair", "toolsets", "edit-demo.json"))
	if !strings.Contains(string(data), "p2-ext") {
		t.Fatal("固化 JSON 应含 p2-ext")
	}

	// 4. 重名 add：默认拒绝；overwrite=true 覆盖
	if _, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "add_plugin", "plugin_name": "p2-ext"}); err == nil {
		t.Fatal("重名 add 应报错")
	}
	if _, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "add_plugin", "plugin_name": "p2-ext", "overwrite": "true"}); err != nil {
		t.Fatalf("overwrite add: %v", err)
	}

	// 5. rm_tool：摘除 p2_tool_b（插件保留、工具对 agent 不可见）
	if _, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "rm_tool", "plugin_name": "p2-ext", "tool": "p2_tool_b"}); err != nil {
		t.Fatalf("rm_tool: %v", err)
	}
	ts3, _ := loadToolset(project, toolsetProject, "edit-demo")
	foundP2 := false
	for i := range ts3.Plugins {
		if ts3.Plugins[i].Name == "p2-ext" {
			foundP2 = true
			if len(ts3.Plugins[i].DisabledTools) != 1 || ts3.Plugins[i].DisabledTools[0] != "p2_tool_b" {
				t.Fatalf("DisabledTools 应为 [p2_tool_b]: %v", ts3.Plugins[i].DisabledTools)
			}
		}
	}
	if !foundP2 {
		t.Fatal("p2-ext 应保留在工具集")
	}
	if reg.IsEnabled("p2_tool_b") {
		t.Fatal("p2_tool_b 应被禁用")
	}
	if !reg.IsEnabled("p2_tool_a") {
		t.Fatal("p2_tool_a 应仍启用")
	}

	// 6. enable_tool：恢复
	if _, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "enable_tool", "plugin_name": "p2-ext", "tool": "p2_tool_b"}); err != nil {
		t.Fatalf("enable_tool: %v", err)
	}
	if !reg.IsEnabled("p2_tool_b") {
		t.Fatal("p2_tool_b 应恢复启用")
	}
	ts4, _ := loadToolset(project, toolsetProject, "edit-demo")
	for i := range ts4.Plugins {
		if ts4.Plugins[i].Name == "p2-ext" && len(ts4.Plugins[i].DisabledTools) != 0 {
			t.Fatalf("DisabledTools 应清空: %v", ts4.Plugins[i].DisabledTools)
		}
	}

	// 7. rm_plugin：移除 p1-base（工具一并卸载）
	if _, err := toolsetEdit(host, project, map[string]any{"name": "edit-demo", "action": "rm_plugin", "plugin_name": "p1-base"}); err != nil {
		t.Fatalf("rm_plugin: %v", err)
	}
	ts5, _ := loadToolset(project, toolsetProject, "edit-demo")
	if len(ts5.Plugins) != 1 || ts5.Plugins[0].Name != "p2-ext" {
		t.Fatalf("移除后应只剩 p2-ext: %v", ts5.Plugins)
	}
	if _, ok := reg.Get("p1_tool"); ok {
		t.Fatal("p1_tool 应已卸载")
	}
}
