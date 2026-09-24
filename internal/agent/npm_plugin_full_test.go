package agent

// npm_plugin_full_test.go — npm 安装「完整落盘」与路径安全单测（2026-09-18）：
//   - 完整包（client/assets/bin/README）文件级落盘与内容一致；
//   - 纯 JS 包（无附加文件）静默跳过、不报错（兼容旧插件）；
//   - safeJoinUnder 路径收敛（合法子路径通过 / 穿越与绝对路径拒绝）；
//   - copyPluginExtras 目标目录与源链接形态的防逃逸校验。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMarketInstallFullPackageFiles 验证「含 client/assets/bin/README 的完整包」
// 在安装链路中的文件级落盘：fetchNPMPackage 解压保持目录结构 →
// copyPluginExtras 拷贝 assets/（含子目录）/bin/README → 内容一致；
// 纯 JS 包（无附加文件）静默跳过不报错（兼容旧插件）。
func TestMarketInstallFullPackageFiles(t *testing.T) {
	tgz := buildTestTarball(map[string]string{
		"package/package.json":      `{"name":"demo-full","version":"2.0.0","main":"index.js","client":"client.js","scope":"global"}`,
		"package/index.js":          `return { name: 'demo-full', apply(){} }`,
		"package/client.js":         `(ui) => { ui.registerPanel({ id: 'x', title: 'X', render(){} }) }`,
		"package/README.md":         "# demo-full",
		"package/assets/a.css":      "body{}",
		"package/assets/sub/b.js":   "// b",
		"package/bin/demo-full.exe": "MZfake",
	})
	var tarballURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/demo-full/latest", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"name": "demo-full", "version": "2.0.0", "description": "demo",
			"dist": map[string]any{"tarball": tarballURL},
		})
	})
	mux.HandleFunc("/demo-full/-/demo-full-2.0.0.tgz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tgz)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	tarballURL = srv.URL + "/demo-full/-/demo-full-2.0.0.tgz"

	oldBase := npmRegistryBase
	npmRegistryBase = srv.URL
	defer func() { npmRegistryBase = oldBase }()

	info, err := fetchNPMInfo("demo-full")
	if err != nil {
		t.Fatalf("fetchNPMInfo: %v", err)
	}
	dir, manifest, err := fetchNPMPackage(info)
	if err != nil {
		t.Fatalf("fetchNPMPackage: %v", err)
	}
	defer removeAllDir(dir)

	// manifest 输入契约（完整落盘依赖 client/scope 字段）
	if got, _ := manifest["client"].(string); got != "client.js" {
		t.Fatalf("manifest client=%q，期望 client.js", got)
	}
	if got, _ := manifest["scope"].(string); got != "global" {
		t.Fatalf("manifest scope=%q，期望 global", got)
	}

	// 完整落盘（assets 含子目录 / bin / README 内容一致）
	dst := t.TempDir()
	if err := copyPluginExtras(dir, dst); err != nil {
		t.Fatalf("copyPluginExtras: %v", err)
	}
	for rel, want := range map[string]string{
		"README.md":         "# demo-full",
		"assets/a.css":      "body{}",
		"assets/sub/b.js":   "// b",
		"bin/demo-full.exe": "MZfake",
	} {
		b, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("落盘缺 %s: %v", rel, err)
		}
		if string(b) != want {
			t.Fatalf("%s 内容不符: %q", rel, string(b))
		}
	}

	// 纯 JS 包（无 assets/bin/README）→ 静默跳过、不报错、目标无新增
	plain := t.TempDir()
	if err := os.WriteFile(filepath.Join(plain, "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst2 := t.TempDir()
	if err := copyPluginExtras(plain, dst2); err != nil {
		t.Fatalf("纯 JS 包应静默跳过: %v", err)
	}
	ents, _ := os.ReadDir(dst2)
	if len(ents) != 0 {
		t.Fatalf("纯 JS 包不应产生落盘，实际 %d 项", len(ents))
	}
}

// TestSafeJoinUnder 覆盖路径收敛：合法子路径通过，穿越/绝对路径/空串拒绝。
func TestSafeJoinUnder(t *testing.T) {
	base := t.TempDir()
	ok := []string{"index.js", "lib/index.js", "a/b/c.js"}
	for _, rel := range ok {
		if _, err := safeJoinUnder(base, rel); err != nil {
			t.Fatalf("safeJoinUnder(%q) 应通过: %v", rel, err)
		}
	}
	bad := []string{"", "..", "../x", "a/../b", "/abs/x", `\abs\x`, `C:\x`}
	for _, rel := range bad {
		if _, err := safeJoinUnder(base, rel); err == nil {
			t.Fatalf("safeJoinUnder(%q) 应拒绝", rel)
		}
	}
}

// TestCopyPluginExtrasRejectsBadTarget 目标目录必须为绝对路径（含 assets 的源）。
func TestCopyPluginExtrasRejectsBadTarget(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "assets", "x.js"), []byte("// x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyPluginExtras(src, "relative/dir"); err == nil {
		t.Fatalf("相对目标目录应被拒绝")
	}
}

// TestCopyPluginExtrasRejectsSymlinkSource 源 assets 为链接形态 → 拒绝（防链接逃逸）。
func TestCopyPluginExtrasRejectsSymlinkSource(t *testing.T) {
	src := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(src, "assets")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("当前环境不支持创建符号链接（%v），跳过", err) // Windows 无权限时跳过
	}
	if err := copyPluginExtras(src, t.TempDir()); err == nil {
		t.Fatalf("链接形态 assets 应被拒绝")
	}
}

// TestMarketInstallFullFlowLanding 安装全链路落盘（npmMarketInstall → goja 路径）：
// scoped 包名（URL %2F 编码）经 httptest registry 完成 latest 查询 → tarball 下载
// 解压 → main/client 读取 → 磁盘插件包固化（index/client/package.json）→ 附加文件
// （assets 递归/bin/README）落盘 → scope/config.npm 元数据。
// ★ 接线回归防护：验证安装流程确实调用 copyPluginExtras（修复前仅函数级测试，
// 安装链路未接线时本测试会因缺 assets/bin 而失败）。
func TestMarketInstallFullFlowLanding(t *testing.T) {
	const pkgName = "@test/pc-fullflow-demo"
	const diskName = "pc-fullflow-demo"
	tgz := buildTestTarball(map[string]string{
		"package/package.json":    `{"name":"@test/pc-fullflow-demo","version":"3.1.4","main":"index.js","client":"client.js","scope":"global","description":"full flow demo"}`,
		"package/index.js":        `return { name: 'pc-fullflow-demo', apply(){} }`,
		"package/client.js":       `(ui) => { ui.registerPanel({ id: 'y' }) }`,
		"package/README.md":       "# full flow demo",
		"package/assets/x.css":    "body{}",
		"package/assets/sub/y.js": "// y",
		"package/bin/demo.exe":    "MZbin",
	})
	var tarballURL string
	// 根 handler 手动路由：scoped 包请求路径含 %2F（r.URL.Path 为解码后路径）
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/latest"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": pkgName, "version": "3.1.4", "description": "full flow demo",
				"dist": map[string]any{"tarball": tarballURL},
			})
		case strings.HasSuffix(r.URL.Path, ".tgz"):
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	tarballURL = srv.URL + "/x.tgz"

	oldBase := npmRegistryBase
	npmRegistryBase = srv.URL
	defer func() { npmRegistryBase = oldBase }()

	// 目标目录：测试环境 InstallDir 回退到包 cwd（go-build 判定），即 internal/agent/.pair/plugins
	dst := filepath.Join(globalPluginsDir(), diskName)
	t.Cleanup(func() {
		_ = os.RemoveAll(dst)
		if es, err := os.ReadDir(globalPluginsDir()); err == nil && len(es) == 0 {
			_ = os.RemoveAll(globalPluginsDir())
		}
	})
	_ = os.RemoveAll(dst)

	msg, err := npmMarketInstall(pkgName)
	if err != nil {
		t.Fatalf("npmMarketInstall: %v", err)
	}
	t.Logf("安装返回: %s", msg)

	// 完整落盘（host 半/client 半/README/assets 递归/bin 内容一致）
	for rel, want := range map[string]string{
		"index.js":        `return { name: 'pc-fullflow-demo', apply(){} }`,
		"client.js":       `(ui) => { ui.registerPanel({ id: 'y' }) }`,
		"README.md":       "# full flow demo",
		"assets/x.css":    "body{}",
		"assets/sub/y.js": "// y",
		"bin/demo.exe":    "MZbin",
	} {
		b, rerr := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel)))
		if rerr != nil {
			t.Fatalf("落盘缺 %s: %v", rel, rerr)
		}
		if string(b) != want {
			t.Fatalf("%s 内容不符: %q", rel, string(b))
		}
	}

	// package.json：scope 透传 + client 声明 + config.npm 元数据（更新机制依赖）
	var pkg GlobalPluginPackage
	b, rerr := os.ReadFile(filepath.Join(dst, "package.json"))
	if rerr != nil {
		t.Fatalf("package.json 缺: %v", rerr)
	}
	if jerr := json.Unmarshal(b, &pkg); jerr != nil {
		t.Fatalf("package.json 解析失败: %v", jerr)
	}
	if pkg.Scope != "global" {
		t.Fatalf("scope=%q，期望 global（包声明透传）", pkg.Scope)
	}
	if pkg.Client != "client.js" {
		t.Fatalf("client=%q，期望 client.js", pkg.Client)
	}
	cfg, _ := pkg.Config["npm"].(map[string]any)
	if cfg == nil || cfg["pkg"] != pkgName || cfg["version"] != "3.1.4" {
		t.Fatalf("config.npm 元数据异常: %#v", pkg.Config)
	}
}
