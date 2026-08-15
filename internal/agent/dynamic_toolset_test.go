package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractJSPluginName(t *testing.T) {
	cases := map[string]string{
		`return { name: 'hello', apply(ctx) {} };`:                     "hello",
		`return { name: "world", apply(ctx, config) {} };`:             "world",
		`return (ctx, config) => { console.log('x') };`:                "",
		`return function myPlugin(ctx) { ctx.tools.register({}); };`:   "myPlugin",
		`(async () => { return { name: 'ts-plugin', apply(ctx) {} } })`: "ts-plugin",
	}
	for code, want := range cases {
		if got := extractJSPluginName(code); got != want {
			t.Errorf("extractJSPluginName(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestSyncDynamicPluginToToolset(t *testing.T) {
	root := t.TempDir()
	def := &jsPluginDef{id: "dyn-99", name: "hello", purpose: "测试插件", code: `return { name: 'hello', apply(ctx) {} };`}
	msg, err := syncDynamicPluginToToolset(root, def, "hello", "")
	if err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	// ★ 2026-08-15 重构：动态插件同步到全局插件包（<InstallDir>/.pair/plugins/），不再写工具集。
	if !strContains(msg, "插件包") {
		t.Errorf("返回信息应含「插件包」，实际 %s", msg)
	}
	// 清理测试产物（同步目标为全局插件包目录，测试后删除避免污染安装目录）
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(GlobalPluginsPath(), "hello")) })
	// 同名更新（版本追加场景）
	def2 := &jsPluginDef{id: "dyn-100", name: "hello", purpose: "更新版", code: `return { name: 'hello', apply(ctx) {} };`}
	if _, err := syncDynamicPluginToToolset(root, def2, "hello", ""); err != nil {
		t.Fatalf("更新同步失败: %v", err)
	}
}

func strContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
