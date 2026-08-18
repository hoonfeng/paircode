// plugin_undelete_test.go — 删除定义（UndefinePermanent）必须同步删除磁盘插件包，
// 防重启 LoadGlobalPlugins 扫描目录重新装配「复活」。
package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUndefinePermanentDiskPackage 验证「删除定义」= 删内存 + 删磁盘插件包：
//  1. define + 落盘插件包目录（globalPluginsDir/<name>/，cordis_define 的固化路径）
//  2. UndefinePermanent → 内存定义删除 + 磁盘包目录删除
//  3. 磁盘目录删除后 LoadGlobalPlugins 的装配依据（目录+package.json）不复存在 →
//     重启不再装配（复活根因消除）
func TestUndefinePermanentDiskPackage(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	name := "test-undelete-tmp"
	dir := filepath.Join(globalPluginsDir(), name)
	base := globalPluginsDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		if entries, err := os.ReadDir(base); err == nil && len(entries) == 0 {
			_ = os.RemoveAll(base) // 测试自建的空壳目录清理
		}
	})
	_ = os.RemoveAll(dir) // 清历史残留

	// 1. define + 落盘（模拟 cordis_define 的固化路径 syncDynamicPluginToToolset）
	jsCode := "return { name: '" + name + "', apply(ctx) { ctx.tools.register({ name: '" + name + "_tool', description: 't', parameters: {}, execute: async () => 'ok' }) } }"
	id, err := host.DefineJSCodeFull(jsCode, "", "测试", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatal("定义不存在")
	}
	if err := syncGlobalPlugin(ToolsetPlugin{Name: name, Purpose: "测试", Code: jsCode, Scope: "project"}); err != nil {
		t.Fatalf("落盘插件包: %v", err)
	}
	def.dir = dir // 模拟 applyGlobalPlugin 装载后记录的插件包目录
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Fatalf("磁盘插件包未落盘: %v", err)
	}

	// 2. UndefinePermanent：删内存 + 删磁盘
	if err := host.UndefinePermanent(name); err != nil {
		t.Fatalf("UndefinePermanent: %v", err)
	}
	if _, ok := host.GetJSDef(id); ok {
		t.Fatal("undefine 后内存定义应删除")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("磁盘插件包目录应已删除, stat err=%v", err)
	}
}

// TestRemoveJSDefDeletesDiskPackage cordis_undefine 工具路径同样删磁盘包。
func TestRemoveJSDefDeletesDiskPackage(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	name := "test-undelete-jsdef"
	dir := filepath.Join(globalPluginsDir(), name)
	base := globalPluginsDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		if entries, err := os.ReadDir(base); err == nil && len(entries) == 0 {
			_ = os.RemoveAll(base)
		}
	})
	_ = os.RemoveAll(dir)

	jsCode := "return { name: '" + name + "', apply(ctx) {} }"
	id, err := host.DefineJSCodeFull(jsCode, "", "测试", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if def == nil {
		t.Fatal("定义不存在")
	}
	if err := syncGlobalPlugin(ToolsetPlugin{Name: name, Purpose: "测试", Code: jsCode, Scope: "project"}); err != nil {
		t.Fatalf("落盘插件包: %v", err)
	}
	def.dir = dir
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Fatalf("磁盘插件包未落盘: %v", err)
	}

	if err := host.RemoveJSDef(id); err != nil {
		t.Fatalf("RemoveJSDef: %v", err)
	}
	if _, ok := host.GetJSDef(id); ok {
		t.Fatal("RemoveJSDef 后定义应删除")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("RemoveJSDef 后磁盘插件包目录应已删除, stat err=%v", err)
	}
}

// TestUndefinePermanentKeepsNonGlobalDir 目录不在全局插件目录内 → 不删磁盘（工具集
// 装卸/测试临时目录等内部 Undefine 场景不受影响）。
func TestUndefinePermanentKeepsNonGlobalDir(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	name := "test-undelete-safe"
	// 临时目录（不在 globalPluginsDir 下）
	tmp := t.TempDir()
	dir := filepath.Join(tmp, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	jsCode := "return { name: '" + name + "', apply(ctx) {} }"
	id, err := host.DefineJSCodeFull(jsCode, "", "测试", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	def, _ := host.GetJSDef(id)
	def.dir = dir

	if err := host.UndefinePermanent(name); err != nil {
		t.Fatalf("UndefinePermanent: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("非全局目录不应被删除, stat err=%v", err)
	}
	_ = id
}
