package agent

import "testing"

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
	if !strContains(msg, "dynamic") {
		t.Errorf("返回信息应含 dynamic，实际 %s", msg)
	}
	ts, err := loadToolset(root, toolsetProject, dynamicToolsetName)
	if err != nil || len(ts.Plugins) != 1 || ts.Plugins[0].Name != "hello" {
		t.Fatalf("工具集 dynamic 应有 hello 条目: %+v err=%v", ts, err)
	}
	// 同名更新（版本追加场景）
	def2 := &jsPluginDef{id: "dyn-100", name: "hello", purpose: "更新版", code: `return { name: 'hello', apply(ctx) {} };`}
	if _, err := syncDynamicPluginToToolset(root, def2, "hello", ""); err != nil {
		t.Fatalf("更新同步失败: %v", err)
	}
	ts, _ = loadToolset(root, toolsetProject, dynamicToolsetName)
	if len(ts.Plugins) != 1 {
		t.Errorf("更新后应仍 1 个条目（覆盖），实际 %d", len(ts.Plugins))
	}
	if ts.Plugins[0].Purpose != "更新版" {
		t.Errorf("条目 purpose 应更新，实际 %q", ts.Plugins[0].Purpose)
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
