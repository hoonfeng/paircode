package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestNPMPluginDiskName 验证 npm 包名 → 磁盘插件包目录名的转换：
//   - 裸名官方形态：paircode-plugin-x → x（前缀剥离，与 @paircode/x 语义等价）
//   - 普通裸名：plain-name → 原样
//   - scope 包：@scope/pkg → pkg（去掉 @ 与 /，目录名友好）
func TestNPMPluginDiskName(t *testing.T) {
	cases := map[string]string{
		"paircode-plugin-x":             "x",
		"paircode-plugin-wechat-bridge": "wechat-bridge",
		"@paircode/git":                 "git",
		"@someorg/tool":                 "tool",
		"plain-name":                    "plain-name",
		"other-plugin-name":             "other-plugin-name",
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

// TestUninstallNPMPluginDeletesDiskPackageWithoutWorkspace 回归锁定「未打开工作区」卸载删盘：
// 该场景下 core.Root() 为空（npmPluginProjectRoot() 也为空），而磁盘插件包目录取
// globalPluginsDir()（= <InstallDir>/.pair/plugins，与工作区无关，安装侧同源）。
// 修复前「删除插件包目录」被包在 projectRoot != "" 分支内 → 卸载静默跳过删盘却返回成功，
// 目录残留、重启后被重新装配（前端显示「已卸载」但插件仍在）。
func TestUninstallNPMPluginDeletesDiskPackageWithoutWorkspace(t *testing.T) {
	savedFolders := core.Folders
	core.Folders = nil
	t.Cleanup(func() { core.Folders = savedFolders })
	if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil {
		savedRoot := ph.Context().WorkspaceRoot
		ph.Context().WorkspaceRoot = ""
		t.Cleanup(func() { ph.Context().WorkspaceRoot = savedRoot })
	}
	if root := npmPluginProjectRoot(); root != "" {
		t.Skipf("测试环境仍存在工作区根 %q，本用例专测无工作区路径", root)
	}

	name := "test-npm-nofs-uninstall"
	dir := filepath.Join(globalPluginsDir(), npmPluginDiskName(name))
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		if entries, err := os.ReadDir(globalPluginsDir()); err == nil && len(entries) == 0 {
			_ = os.RemoveAll(globalPluginsDir())
		}
	})
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建插件包目录: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`","version":"0.0.1"}`), 0o644); err != nil {
		t.Fatalf("写 package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("return { name: '"+name+"', apply() {} }\n"), 0o644); err != nil {
		t.Fatalf("写 index.js: %v", err)
	}

	if err := uninstallNPMPlugin(name); err != nil {
		t.Fatalf("uninstallNPMPlugin(%s) 返回错误: %v", name, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("卸载后磁盘插件包目录仍存在（stat err=%v）—— 无工作区卸载未真正删盘", err)
	}
}

// TestUninstallNPMPluginMissingReturnsError 锁定「未安装 → 明确报错」：
// 无工作区时旧实现会静默成功（假成功），修复后应返回「未找到插件」错误，
// 以免市场面板把未装的条目误报为卸载成功。
func TestUninstallNPMPluginMissingReturnsError(t *testing.T) {
	savedFolders := core.Folders
	core.Folders = nil
	t.Cleanup(func() { core.Folders = savedFolders })

	name := "test-npm-not-installed-" + "zzz"
	dir := filepath.Join(globalPluginsDir(), npmPluginDiskName(name))
	_ = os.RemoveAll(dir)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	err := uninstallNPMPlugin(name)
	if err == nil {
		t.Fatalf("卸载未安装插件应返回错误，实际返回 nil（假成功）")
	}
	t.Logf("未安装卸载返回错误（预期）: %v", err)
}
