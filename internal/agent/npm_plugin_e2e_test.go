package agent

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// TestMarketInstallNPMPluginE2E 模拟 npm registry（httptest），验证
// 兼容型 cordis 插件（纯 JS 无依赖）全链路安装成功：latest 查询 →
// tarball 下载解压 → main 提取 → patch 固化。
func TestMarketInstallNPMPluginE2E(t *testing.T) {
	// 构造纯 JS cordis 插件 tarball（export default 形态）
	pkgCode := `
export default function (ctx) {
  ctx.provide('demo', { hello: () => 'world' })
  return { name: 'demo-npm' }
}
`
	tgz := buildTestTarball(map[string]string{
		"package/package.json": `{"name":"demo-npm-plugin","version":"1.2.3","main":"lib/index.js"}`,
		"package/lib/index.js": pkgCode,
	})

	var gotTarballPath string
	mux := http.NewServeMux()
	mux.HandleFunc("/demo-npm-plugin/latest", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"name":        "demo-npm-plugin",
			"version":     "1.2.3",
			"description": "demo",
			"dist":        map[string]any{"tarball": gotTarballPath},
		})
	})
	mux.HandleFunc("/demo-npm-plugin/-/demo-npm-plugin-1.2.3.tgz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tgz)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	gotTarballPath = srv.URL + "/demo-npm-plugin/-/demo-npm-plugin-1.2.3.tgz"

	oldBase := npmRegistryBase
	npmRegistryBase = srv.URL
	defer func() { npmRegistryBase = oldBase }()

	// 全链路：info → tarball → main
	info, err := fetchNPMInfo("demo-npm-plugin")
	if err != nil {
		t.Fatalf("fetchNPMInfo: %v", err)
	}
	dir, manifest, err := fetchNPMPackage(info)
	if err != nil {
		t.Fatalf("fetchNPMPackage: %v", err)
	}
	defer removeAllDir(dir)
	if got := npmPackageMain(manifest); got != "lib/index.js" {
		t.Fatalf("main 期望 lib/index.js 实际 %q", got)
	}
	codeBytes, err := os.ReadFile(filepath.Join(dir, "lib", "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(codeBytes), "demo-npm") {
		t.Fatalf("源码内容异常: %.100s", string(codeBytes))
	}

	// 编译（goja 可执行形态）
	js, err := compilePluginSource(string(codeBytes), "js", "cordis-dyn.ts", dir)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}
	if _, err := goja.Compile("cordis-dyn.js", "(async () => {\n"+js+"\n})()", false); err != nil {
		t.Fatalf("goja 语法错误: %v", err)
	}
	t.Logf("纯 JS cordis 插件全链路 OK（info→tarball→main→编译）")
}

// TestMarketInstallNPMPluginPatchE2E patch 固化 + 卸载（不依赖 registry）。
func TestMarketInstallNPMPluginPatchE2E(t *testing.T) {
	dir := t.TempDir()
	patch := filepath.Join(dir, ".pair", "cordis.patch.json")
	if err := appendPatchNPMPlugin(patch, "demo-npm-plugin", "1.2.3", "code", "demo"); err != nil {
		t.Fatal(err)
	}
	// 幂等更新
	if err := appendPatchNPMPlugin(patch, "demo-npm-plugin", "1.2.3", "code2", "demo"); err != nil {
		t.Fatal(err)
	}
	doc, err := readCordisPatch(patch)
	if err != nil || len(doc.Plugins) != 1 {
		t.Fatalf("patch 幂等失败: %v len=%d", err, len(doc.Plugins))
	}
	if doc.Plugins[0].Code != "code2" {
		t.Fatalf("原位更新失败")
	}
	removed, err := removePatchNPMPlugin(patch, "demo-npm-plugin")
	if err != nil || !removed {
		t.Fatalf("remove 失败: %v %v", removed, err)
	}
	doc, _ = readCordisPatch(patch)
	if len(doc.Plugins) != 0 {
		t.Fatalf("卸载后应清空，实际 %d", len(doc.Plugins))
	}
}

// ── 测试辅助 ──

func buildTestTarball(files map[string]string) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// 目录项（确保目录存在）
	for _, d := range []string{"package/", "package/lib/"} {
		tw.WriteHeader(&tar.Header{Name: d, Typeflag: tar.TypeDir, Mode: 0o755})
	}
	for name, content := range files {
		tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))})
		tw.Write([]byte(content))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func removeAllDir(dir string) { _ = os.RemoveAll(dir) }
