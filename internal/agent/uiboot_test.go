package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeUIPkgFixture 在 dir 下生成一个 UI 插件包目录（package.json + index.js + client.js + assets/<name>.js）。
// withDshUI=true 时写入 dsh.ui 段；bundleContent 为空则不写 bundle。
func writeUIPkgFixture(t *testing.T, dir, name string, withDshUI bool, slot string, bundleContent string) {
	t.Helper()
	pkgDir := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(pkgDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := map[string]any{
		"name":    name,
		"purpose": "fixture " + name,
		"version": "0.1.0",
		"scope":   "global",
		"type":    "plugin",
		"main":    "index.js",
		"client":  "client.js",
	}
	if withDshUI {
		pkg["dsh"] = map[string]any{
			"ui": map[string]any{
				"platform":    "web",
				"slot":        slot,
				"kind":        "single",
				"scope":       "root",
				"inject":      []string{"@paircode/core"},
				"immediately": true,
			},
		}
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	for f, c := range map[string]string{
		"index.js":  "return {};",
		"client.js": "(ui)=>{}",
	} {
		if err := os.WriteFile(filepath.Join(pkgDir, f), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if bundleContent != "" {
		if err := os.WriteFile(filepath.Join(pkgDir, "assets", name+".js"), []byte(bundleContent), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBuildUIBootGraphFrom_OnlyDshUIPackagesIncluded(t *testing.T) {
	dir := t.TempDir()
	writeUIPkgFixture(t, dir, "ui-editor", true, "editor", "var x=1;")
	writeUIPkgFixture(t, dir, "ui-sidebar", true, "sidebar", "var y=2;")
	// 无 dsh.ui 的旧包（client.js 直载）→ 不进 boot 图
	writeUIPkgFixture(t, dir, "tool-shell", false, "", "var z=3;")

	graph := BuildUIBootGraphFrom(dir)
	if len(graph.Entries) != 2 {
		t.Fatalf("entries 数 = %d，应 2（仅 dsh.ui 包）: %+v", len(graph.Entries), graph.Entries)
	}
	if graph.Rev == "" {
		t.Fatal("图级 rev 不应为空")
	}
	if graph.Entries[0].ID != "ui-editor" || graph.Entries[1].ID != "ui-sidebar" {
		t.Fatalf("entries 排序错误: %+v", graph.Entries)
	}
	e := graph.Entries[0]
	if e.URL != "/plugins-assets/ui-editor/assets/ui-editor.js?rev="+e.Rev {
		t.Fatalf("url 不符合约定: %s", e.URL)
	}
	if e.Rev == "" {
		t.Fatal("entry rev 不应为空")
	}
	if len(e.Inject) != 1 || e.Inject[0] != "@paircode/core" {
		t.Fatalf("inject 未取自 dsh.ui: %+v", e.Inject)
	}
	if !e.Immediately {
		t.Fatal("immediately 缺省应为 true")
	}
	if len(e.External) != 1 || e.External[0] != "@paircode/core" {
		t.Fatalf("external 应为共享核心: %+v", e.External)
	}
}

func TestBuildUIBootGraphFrom_RevChangesWithBundleContent(t *testing.T) {
	dir := t.TempDir()
	writeUIPkgFixture(t, dir, "ui-editor", true, "editor", "AAA")
	g1 := BuildUIBootGraphFrom(dir)
	writeUIPkgFixture(t, dir, "ui-editor", true, "editor", "BBB")
	g2 := BuildUIBootGraphFrom(dir)
	if g1.Entries[0].Rev == g2.Entries[0].Rev {
		t.Fatal("bundle 内容变化后 entry rev 应变")
	}
	if g1.Rev == g2.Rev {
		t.Fatal("bundle 内容变化后图级 rev 应变")
	}
}

func TestBuildUIBootGraphFrom_NoDir_Empty(t *testing.T) {
	graph := BuildUIBootGraphFrom(filepath.Join(t.TempDir(), "nope"))
	if len(graph.Entries) != 0 {
		t.Fatalf("不存在目录应返回空 entries，got %d", len(graph.Entries))
	}
}

func TestGlobalPluginPackage_ParsesDshUI(t *testing.T) {
	var pkg GlobalPluginPackage
	if err := json.Unmarshal([]byte(`{"name":"ui-editor","main":"index.js","dsh":{"ui":{"slot":"editor","kind":"single","inject":["@paircode/core"]}}}`), &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg.Dsh == nil || pkg.Dsh.UI == nil || pkg.Dsh.UI.Slot != "editor" {
		t.Fatalf("dsh.ui 解析失败: %+v", pkg.Dsh)
	}
	if pkg.Dsh.UI.Kind != "single" || len(pkg.Dsh.UI.Inject) != 1 {
		t.Fatalf("dsh.ui 字段解析不完整: %+v", pkg.Dsh.UI)
	}
}
