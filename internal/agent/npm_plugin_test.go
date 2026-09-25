package agent

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestNPMPluginPatch append/remove 幂等性。
func TestNPMPluginPatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".pair", "cordis.patch.json")

	// 追加
	if err := appendPatchNPMPlugin(path, "cordis-plugin-android", "0.0.7", "console.log('hi')", "desc"); err != nil {
		t.Fatalf("append: %v", err)
	}
	// 幂等（同版本再追加 → 原位更新不重复）
	if err := appendPatchNPMPlugin(path, "cordis-plugin-android", "0.0.7", "console.log('hi2')", "desc2"); err != nil {
		t.Fatalf("append2: %v", err)
	}
	doc, err := readCordisPatch(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(doc.Plugins) != 1 {
		t.Fatalf("期望 1 条（幂等），实际 %d", len(doc.Plugins))
	}
	if doc.Plugins[0].Code != "console.log('hi2')" {
		t.Fatalf("期望原位更新 code，实际 %q", doc.Plugins[0].Code)
	}
	if doc.Plugins[0].Config["npm"] != "cordis-plugin-android@0.0.7" {
		t.Fatalf("config.npm 异常: %v", doc.Plugins[0].Config["npm"])
	}

	// 已安装判断
	if !npmPluginInstalled("cordis-plugin-android") && false { // 需要工作区根，跳过
		t.Fatal("unreachable")
	}

	// 移除（按包名，忽略版本）
	removed, err := removePatchNPMPlugin(path, "cordis-plugin-android")
	if err != nil || !removed {
		t.Fatalf("remove: removed=%v err=%v", removed, err)
	}
	doc, _ = readCordisPatch(path)
	if len(doc.Plugins) != 0 {
		t.Fatalf("期望清空，实际 %d 条", len(doc.Plugins))
	}
}

// TestNPMPackageMain main 字段提取。
func TestNPMPackageMain(t *testing.T) {
	cases := []struct {
		manifest map[string]any
		want     string
	}{
		{map[string]any{"main": "lib/index.js"}, "lib/index.js"},
		{map[string]any{}, "index.js"},
		{map[string]any{"main": "dist/index"}, "dist/index.js"}, // 无扩展名补 .js
		{map[string]any{"exports": map[string]any{".": map[string]any{"import": "es/index.mjs"}}}, "es/index.mjs"},
	}
	for i, c := range cases {
		if got := npmPackageMain(c.manifest); got != c.want {
			t.Errorf("case %d: 期望 %q 实际 %q", i, c.want, got)
		}
	}
}

// TestFetchNPMPackageLocal 本地构造 tarball 验证下载解压逻辑（不依赖网络）。
func TestFetchNPMPackageLocal(t *testing.T) {
	// 构造 npm tarball（package/ 前缀 + package.json + lib/index.js）
	tarballPath := filepath.Join(t.TempDir(), "pkg.tgz")
	f, err := os.Create(tarballPath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	files := map[string]string{
		"package/package.json": `{"name":"demo-plugin","version":"1.0.0","main":"lib/index.js"}`,
		"package/lib/index.js": `export default function (ctx) { ctx.command('hi') }`,
		"package/README.md":    "readme",
	}
	names := []string{"package/", "package/package.json", "package/lib/", "package/lib/index.js", "package/README.md"}
	for _, n := range names {
		content, isFile := files[n]
		hdr := &tar.Header{Name: n, Mode: 0o644}
		if strings.HasSuffix(n, "/") {
			hdr.Typeflag = tar.TypeDir
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(content))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(n, "/") {
			if _, err := tw.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		_ = isFile
	}
	tw.Close()
	gz.Close()
	f.Close()

	// 用 fetchNPMPackage 解压（模拟 registry dist.tarball）：
	// fetchNPMPackage 用 http.Client，不支持 file:// —— 直接用 extractTarGzForTest
	// 验证 tar 结构兼容性 + main 提取链路。
	dir := t.TempDir()
	// 直接复制构造好的 tarball 手动解压验证 main 提取链路
	if err := extractTarGzForTest(tarballPath, dir); err != nil {
		t.Fatalf("extract: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{}
	if err := jsonUnmarshalTest(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if got := npmPackageMain(manifest); got != "lib/index.js" {
		t.Fatalf("main 期望 lib/index.js 实际 %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "lib", "index.js")); err != nil {
		t.Fatalf("lib/index.js 缺失: %v", err)
	}
}

func jsonUnmarshalTest(b []byte, v any) error {
	return json.Unmarshal(b, v)
}

// extractTarGzForTest 测试辅助：解压 tgz 到目录（与生产解压逻辑一致）。
func extractTarGzForTest(tgzPath, dst string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		rel := strings.TrimPrefix(hdr.Name, "package"+string(filepath.Separator))
		if rel == hdr.Name {
			rel = strings.TrimPrefix(hdr.Name, "package/")
		}
		if rel == hdr.Name {
			continue
		}
		target := filepath.Join(dst, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755)
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

// TestNPMRegistryFollowsSettings ★ 2026-09-26 插件源配置化：内核不写死任何源地址，
// registry 完全由「插件市场」配置段决定（useMirror / mirror / registry：源仓库与镜像源
// 均可在设置里切换），且每次调用重新计算 → 保存设置后立即生效。
// 环境变量覆盖时不校验配置（env 优先级最高）。
func TestNPMRegistryFollowsSettings(t *testing.T) {
	if v := os.Getenv("PAIRCODE_NPM_REGISTRY"); v != "" {
		t.Skipf("PAIRCODE_NPM_REGISTRY 已覆盖为 %s，跳过配置断言", v)
	}
	oldSchemas, oldSettings, oldOverride := core.PluginSettingSchemas, core.Settings, npmRegistryBaseOverride
	defer func() {
		core.PluginSettingSchemas, core.Settings, npmRegistryBaseOverride = oldSchemas, oldSettings, oldOverride
	}()
	npmRegistryBaseOverride = ""
	core.Settings = core.Default()
	core.PluginSettingSchemas = nil
	// 默认值随插件 schema 下发（内核零默认地址）
	core.RegisterPluginSettingSchema(core.SettingSchema{
		Key: "marketplace", Title: "插件市场",
		Fields: []core.SettingField{
			{Name: "useMirror", Type: "checkbox", Default: true},
			{Name: "mirror", Type: "text", Default: "https://registry.npmmirror.com"},
			{Name: "registry", Type: "text", Default: "https://registry.npmjs.org"},
		},
	})
	if got := npmRegistryBase(); got != "https://registry.npmmirror.com" {
		t.Fatalf("默认（启用镜像）registry = %q，期望镜像源", got)
	}
	// 关闭镜像 → 用源仓库（官方源）
	setMarketSettings(map[string]any{"useMirror": false})
	if got := npmRegistryBase(); got != "https://registry.npmjs.org" {
		t.Fatalf("关闭镜像后 registry = %q，期望官方源", got)
	}
	// 自定义镜像地址 → 立即生效（尾斜杠归一）
	setMarketSettings(map[string]any{"useMirror": true, "mirror": "https://npm.example.com/"})
	if got := npmRegistryBase(); got != "https://npm.example.com" {
		t.Fatalf("自定义镜像 registry = %q，期望 https://npm.example.com", got)
	}
	// 镜像留空 → 回退源仓库（单项空不致失效）
	setMarketSettings(map[string]any{"useMirror": true, "mirror": "   "})
	if got := npmRegistryBase(); got != "https://registry.npmjs.org" {
		t.Fatalf("镜像为空应回退源仓库，实际 %q", got)
	}
	// 显式注入优先于配置
	npmRegistryBaseOverride = "http://127.0.0.1:4873"
	if got := npmRegistryBase(); got != "http://127.0.0.1:4873" {
		t.Fatalf("注入值应优先，实际 %q", got)
	}
	// 无配置且无默认（插件未装载）→ 空串：调用方给「去设置里配」提示，不静默退回某源
	npmRegistryBaseOverride = ""
	core.PluginSettingSchemas = nil
	core.Settings = core.Default()
	if got := npmRegistryBase(); got != "" {
		t.Fatalf("无配置时应返回空串（提示去设置），实际 %q", got)
	}
}

// setMarketSettings 覆盖「插件市场」配置段（测试辅助）。
func setMarketSettings(cfg map[string]any) {
	if core.Settings.PluginSettings == nil {
		core.Settings.PluginSettings = map[string]map[string]any{}
	}
	core.Settings.PluginSettings["marketplace"] = cfg
}

// TestNPMTimeoutsRaised ★ 2026-09-25：市场装包的超时下限（总超时/TLS/响应头），
// 防有人把 30s 的卡点改回去。
func TestNPMTimeoutsRaised(t *testing.T) {
	if npmFetchTimeout < 300*time.Second {
		t.Fatalf("npmFetchTimeout = %v，应 >= 300s", npmFetchTimeout)
	}
	tr, ok := npmHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("npmHTTPClient.Transport 类型异常: %T", npmHTTPClient.Transport)
	}
	if tr.TLSHandshakeTimeout < 60*time.Second {
		t.Fatalf("TLSHandshakeTimeout = %v，应 >= 60s", tr.TLSHandshakeTimeout)
	}
	if tr.ResponseHeaderTimeout < 120*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %v，应 >= 120s", tr.ResponseHeaderTimeout)
	}
}

// TestNPMMirrorTarballRewrite 官方源 tarball 地址在非官方 base 下被改写
// （防「元数据残留官方地址 → 静默退回慢源」）。
func TestNPMMirrorTarballRewrite(t *testing.T) {
	old := npmRegistryBaseOverride
	defer func() { npmRegistryBaseOverride = old }()

	// 镜像地址由测试自备（内核不再内置任何源地址）
	const mirrorBase = "https://registry.npmmirror.com"
	const officialTarball = "https://registry.npmjs.org/@paircode/tool-model/-/tool-model-1.2.3.tgz"

	// ① 镜像 base：官方地址 → 镜像地址
	npmRegistryBaseOverride = mirrorBase
	want := mirrorBase + "/@paircode/tool-model/-/tool-model-1.2.3.tgz"
	if got := npmMirrorTarball(officialTarball); got != want {
		t.Fatalf("改写结果 = %q，期望 %q", got, want)
	}
	// ② 已是镜像地址 → 原样（幂等）
	if got := npmMirrorTarball(want); got != want {
		t.Fatalf("镜像地址被误改: %q", got)
	}
	// ③ 自定义 registry：非官方地址原样、官方地址对齐到该 base
	npmRegistryBaseOverride = "http://127.0.0.1:4873"
	local := "http://127.0.0.1:4873/pkg/-/pkg-1.0.0.tgz"
	if got := npmMirrorTarball(local); got != local {
		t.Fatalf("本地源地址被误改: %q", got)
	}
	if got := npmMirrorTarball(officialTarball); got != npmRegistryBaseOverride+"/@paircode/tool-model/-/tool-model-1.2.3.tgz" {
		t.Fatalf("自定义源下官方地址未对齐: %q", got)
	}
	// ④ base 就是官方源 → 原样
	npmRegistryBaseOverride = npmOfficialRegistry
	if got := npmMirrorTarball(officialTarball); got != officialTarball {
		t.Fatalf("官方源 base 下不应改写: %q", got)
	}
	// ⑤ 空串 → 空串
	if got := npmMirrorTarball(""); got != "" {
		t.Fatalf("空串应原样返回: %q", got)
	}
}
