package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNPMPluginDiskName 验证 npm 包名 → 磁盘插件包目录名的转换：
//   - 裸名：paircode-plugin-x → 原样
//   - scope 包：@scope/pkg → pkg（去掉 @ 与 /，目录名友好）
func TestNPMPluginDiskName(t *testing.T) {
	cases := map[string]string{
		"paircode-plugin-x": "paircode-plugin-x",
		"@paircode/git":     "git",
		"@someorg/tool":     "tool",
		"plain-name":        "plain-name",
	}
	for in, want := range cases {
		if got := npmPluginDiskName(in); got != want {
			t.Errorf("npmPluginDiskName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNPMPluginDiskPackageRoundTrip 验证 npm 插件安装落盘 → 重启装配闭环：
//  1. syncGlobalPlugin 固化 .pair/plugins/<name>/（模拟市场安装的落盘步骤）
//  2. 新宿主 LoadGlobalPlugins 扫描装配（模拟重启自动装配）
//  3. 插件工具对 agent 可见
func TestNPMPluginDiskPackageRoundTrip(t *testing.T) {
	name := "test-npm-pkg-roundtrip"
	dir := filepath.Join(globalPluginsDir(), name)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		if entries, err := os.ReadDir(globalPluginsDir()); err == nil && len(entries) == 0 {
			_ = os.RemoveAll(globalPluginsDir())
		}
	})
	_ = os.RemoveAll(dir)

	// 1. 固化磁盘插件包（市场安装 npm 插件后的形态）
	jsCode := "return { name: '" + name + "', apply(ctx) { ctx.tools.register({ name: '" + name + "_t', description: 'npm 插件测试工具', parameters: {}, execute: async () => 'ok' }) } }"
	if err := syncGlobalPlugin(ToolsetPlugin{Name: name, Purpose: "npm 插件测试", Code: jsCode, Scope: "project"}); err != nil {
		t.Fatalf("固化插件包: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err != nil {
		t.Fatalf("package.json 未落盘: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.js")); err != nil {
		t.Fatalf("index.js 未落盘: %v", err)
	}

	// 2. 新宿主重启装配（模拟 LoadGlobalPlugins 启动扫描）
	host2 := NewPluginHost(NewRegistry(), nil, "")
	n := LoadGlobalPlugins(host2)
	if n < 1 {
		t.Fatalf("LoadGlobalPlugins 装配数为 %d，应 ≥1", n)
	}
	if _, ok := host2.Get(name); !ok {
		t.Fatalf("插件 %s 未装配到新宿主", name)
	}
	// 3. 工具可见
	if _, ok := host2.Context().Tools.Get(name + "_t"); !ok {
		t.Fatalf("插件工具 %s_t 未注册", name)
	}
	t.Logf("落盘→重启装配闭环 OK（%d 个插件装配）", n)
}

// TestNPMPluginInstalledDiskPackage 验证已装检查识别磁盘插件包形态。
func TestNPMPluginInstalledDiskPackage(t *testing.T) {
	host := NewPluginHost(NewRegistry(), nil, "")
	name := "test-npm-installed-pkg"
	dir := filepath.Join(globalPluginsDir(), name)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	_ = os.RemoveAll(dir)

	jsCode := "return { name: '" + name + "', apply(ctx) { ctx.tools.register({ name: '" + name + "_t', description: 't', parameters: {}, execute: async () => 'ok' }) } }"
	if err := syncGlobalPlugin(ToolsetPlugin{Name: name, Purpose: "测试", Code: jsCode, Scope: "project"}); err != nil {
		t.Fatalf("固化: %v", err)
	}
	if !npmPluginInstalled(name) {
		t.Fatal("npmPluginInstalled 应识别磁盘插件包（已安装）")
	}
	_ = host
}
