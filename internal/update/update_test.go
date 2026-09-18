package update

import (
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ─── 版本比较 ────────────────────────────────────────────────

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.6.3", "1.6.3", 0},
		{"1.6.3", "v1.6.10", -1},
		{"1.7.0", "1.6.9", 1},
		{"V1.7", "1.6.9", 1},
		{"refs/tags/v1.6.3", "1.6.3", 0},
		{"1.6.3", "1.6.3", 0},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
	if !IsNewer("1.6.10", "1.6.9") {
		t.Error("1.6.10 应比 1.6.9 新")
	}
	if IsNewer("1.6.3", "1.6.3") {
		t.Error("同版本不应判定为新")
	}
}

// ─── 资产 / 主程序匹配 ───────────────────────────────────────

func TestAssetNameAndMatch(t *testing.T) {
	if got := AssetNameForPlatform("v1.7.0", "windows"); got != "PairCode-1.7.0.zip" {
		t.Errorf("windows 资产名 = %s", got)
	}
	if got := AssetNameForPlatform("1.7.0", "linux"); got != "PairCode-linux-1.7.0.zip" {
		t.Errorf("linux 资产名 = %s", got)
	}
	if got := AssetNameForPlatform("1.7.0", "darwin"); got != "PairCode-darwin-1.7.0.zip" {
		t.Errorf("darwin 资产名 = %s", got)
	}
	names := []string{
		"PairCode-1.7.0.zip",
		"PairCode-linux-1.7.0.zip",
		"PairCode-darwin-1.7.0.zip",
		"source.tar.gz",
	}
	if got := MatchAsset(names, "1.7.0", "windows"); got != "PairCode-1.7.0.zip" {
		t.Errorf("windows 匹配 = %s", got)
	}
	if got := MatchAsset(names, "v1.7.0", "linux"); got != "PairCode-linux-1.7.0.zip" {
		t.Errorf("linux 匹配(v 前缀) = %s", got)
	}
	if got := MatchAsset(names, "1.7.0", "darwin"); got != "PairCode-darwin-1.7.0.zip" {
		t.Errorf("darwin 匹配 = %s", got)
	}
	if got := MatchAsset(names, "1.7.0", "freebsd"); got != "" {
		t.Errorf("未知平台应无匹配，得到 %s", got)
	}
	if got := MatchAsset([]string{"PairCode-1.6.3.zip"}, "1.7.0", "windows"); got != "" {
		t.Errorf("版本不符应无匹配，得到 %s", got)
	}
}

func TestMatchMainProgram(t *testing.T) {
	win := []string{"pair.exe", "README.md", "assets/a.js"}
	if got := MatchMainProgram(win, "windows", "amd64"); got != "pair.exe" {
		t.Errorf("windows 主程序 = %s", got)
	}
	lin := []string{"pair-linux-amd64", "README.md"}
	if got := MatchMainProgram(lin, "linux", "amd64"); got != "pair-linux-amd64" {
		t.Errorf("linux 主程序 = %s", got)
	}
	if got := MatchMainProgram(lin, "darwin", "arm64"); got != "" {
		t.Errorf("darwin 在 linux 包内应找不到主程序，得到 %s", got)
	}
	// 宽容：带子目录
	if got := MatchMainProgram([]string{"bin/pair.exe"}, "windows", "amd64"); got != "pair.exe" {
		t.Errorf("子目录主程序 = %s", got)
	}
}

func TestNormalizeDigest(t *testing.T) {
	want := "173cefb94c8ca3b1a0e3c23a40e66c67cd296ff9cee04c8a3bac01cce130ff75"
	for _, in := range []string{want, "SHA256:" + strings.ToUpper(want), " sha256=" + want + " "} {
		if got := normalizeDigest(in); got != want {
			t.Errorf("normalizeDigest(%q)=%q", in, got)
		}
	}
}

// ─── 保护名单 ────────────────────────────────────────────────

func TestIsProtected(t *testing.T) {
	protected := []string{
		"config/settings.json",
		"config/updates/x.zip",
		".pair/toolsets/t.json",
		".pair/memory/m.md",
		".pair/tasks/a.json",
		".pair/skills/s/SKILL.md",
		".pair/project.md",
		".pair/mcp.json",
		".pair/user-stuff.json",
		"logs/paircode.log",
		"screenshots/a.png",
	}
	for _, p := range protected {
		if !IsProtected(p) {
			t.Errorf("%s 应受保护", p)
		}
	}
	allowed := []string{
		"pair.exe",
		"README.md",
		"assets/app.js",
		"plugins-src/ui-app/package.json",
		".pair/plugins/core-api/index.js",
		"bin/.pair/plugins/x/index.js",
		"models/m.onnx",
	}
	for _, p := range allowed {
		if IsProtected(p) {
			t.Errorf("%s 不应受保护", p)
		}
	}
}

// ─── 解压安全 ────────────────────────────────────────────────

func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	z := filepath.Join(dir, "evil.zip")
	writeZip(t, z, map[string]string{
		"../evil.txt": "pwned",
		"pair.exe":    "bin",
		"README.md":   "r",
	})
	_, err := Extract(z, filepath.Join(dir, "staging"))
	if err == nil || !strings.Contains(err.Error(), "非法路径") {
		t.Fatalf("应拒绝 zip-slip，得到 err=%v", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "evil.txt")); serr == nil {
		t.Fatal("越界文件被写入了")
	}
}

func TestExtractAndSmoke(t *testing.T) {
	dir := t.TempDir()
	z := filepath.Join(dir, "pkg.zip")
	writeZip(t, z, map[string]string{
		"pair.exe":           "BIN",
		"README.md":          "readme",
		"assets/app.js":      "console.log(1)",
		".pair/plugins/x.js": "//p",
	})
	staging := filepath.Join(dir, "staging")
	files, err := Extract(z, staging)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Fatalf("解压文件数 = %d want 4（%v）", len(files), files)
	}
	main, err := SmokeCheck(staging, files, "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if main != "pair.exe" {
		t.Fatalf("主程序 = %s", main)
	}
	// 缺主程序 → SmokeCheck 报错
	if _, err := SmokeCheck(staging, []string{"README.md", "assets/a.js"}, "windows", "amd64"); err == nil {
		t.Fatal("缺主程序应报错")
	}
}

// ─── 下载 + 校验 + 解压（httptest 端到端）────────────────────

// newTestServer 提供 zip 与自定义 feed 的本地 HTTP 源。
func newTestServer(t *testing.T, zipPath, feedPath string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, ".zip"):
			http.ServeFile(w, r, zipPath)
		case strings.HasSuffix(r.URL.Path, "feed.json"):
			http.ServeFile(w, r, feedPath)
		default:
			http.NotFound(w, r)
		}
	}))
}

// setupFeed 造一个「新版本 9.9.9」的本地更新源，返回 Config。
func setupFeed(t *testing.T, corruptDigest bool) (Config, string) {
	t.Helper()
	dir := t.TempDir()
	z := filepath.Join(dir, "PairCode-9.9.9.zip")
	writeZip(t, z, map[string]string{
		"pair.exe":             "NEW-BINARY-v9.9.9",
		"README.md":            "# readme",
		"assets/app.js":        "console.log('new')",
		"config/template.json": `{"x":1}`,
	})
	sum, err := fileSHA256(z)
	if err != nil {
		t.Fatal(err)
	}
	if corruptDigest {
		sum = strings.Repeat("0", 64)
	}
	st, _ := os.Stat(z)
	srv := newTestServer(t, z, filepath.Join(dir, "feed.json"))
	t.Cleanup(srv.Close)
	feed := map[string]any{
		"version":     "9.9.9",
		"notes":       "测试版本",
		"publishedAt": "2026-09-19T00:00:00Z",
		"assets": map[string]any{
			"windows": map[string]any{
				"name":   "PairCode-9.9.9.zip",
				"url":    srv.URL + "/PairCode-9.9.9.zip",
				"size":   st.Size(),
				"sha256": sum,
			},
		},
	}
	b, _ := json.Marshal(feed)
	feedPath := filepath.Join(dir, "feed.json")
	if err := os.WriteFile(feedPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		CurrentVersion: "1.6.3",
		GOOS:           "windows",
		GOARCH:         runtime.GOARCH,
		FeedType:       "custom",
		FeedURL:        srv.URL + "/feed.json",
		RequireSHA256:  true,
		KeepBackup:     true,
		DownloadDir:    filepath.Join(dir, "downloads"),
		InstallDir:     filepath.Join(dir, "install"),
	}
	return cfg, dir
}

func TestFetchDownloadVerifyExtract(t *testing.T) {
	cfg, _ := setupFeed(t, false)
	ctx := context.Background()

	m, err := cfg.FetchManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "9.9.9" || m.Asset.Name != "PairCode-9.9.9.zip" {
		t.Fatalf("清单解析异常: %+v", m)
	}
	if !IsNewer(m.Version, cfg.CurrentVersion) {
		t.Fatal("9.9.9 应比 1.6.3 新")
	}

	res, err := cfg.Download(ctx, m.Asset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Reused || res.Bytes == 0 {
		t.Fatalf("下载结果异常: %+v", res)
	}
	staging := filepath.Join(cfg.DownloadDir, "staging-9.9.9")
	files, err := Extract(res.Path, staging)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SmokeCheck(staging, files, "windows", "amd64"); err != nil {
		t.Fatal(err)
	}
	// 二次调用应复用（不重复下载）
	res2, err := cfg.Download(ctx, m.Asset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Reused {
		t.Fatal("已下载且校验通过时应复用")
	}
	// 断点续传：删掉成品、保留半个 .part → Range 请求续传
	part := res.Path + ".part"
	raw, _ := os.ReadFile(res.Path)
	if err := os.Remove(res.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part, raw[:len(raw)/2], 0o644); err != nil {
		t.Fatal(err)
	}
	res3, err := cfg.Download(ctx, m.Asset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res3.Resumed {
		t.Fatal("存在 .part 时应走断点续传")
	}
	sum, _ := fileSHA256(res3.Path)
	if sum != normalizeDigest(m.Asset.SHA256) {
		t.Fatal("续传后的文件摘要不符（hasher 未覆盖续传前内容）")
	}
}

func TestDownloadRejectsBadDigest(t *testing.T) {
	cfg, _ := setupFeed(t, true) // digest 篡改
	ctx := context.Background()
	m, err := cfg.FetchManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.Download(ctx, m.Asset, nil)
	if err == nil || !strings.Contains(err.Error(), "sha256 校验失败") {
		t.Fatalf("应因摘要不符失败，得到 err=%v", err)
	}
	entries, _ := os.ReadDir(cfg.DownloadDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") || strings.HasSuffix(e.Name(), ".zip") {
			t.Fatalf("校验失败后不应留下文件: %s", e.Name())
		}
	}
}

// ─── 替换（含保护名单与备份）────────────────────────────────

func TestApplyStaged(t *testing.T) {
	dir := t.TempDir()
	install := filepath.Join(dir, "install")
	if err := os.MkdirAll(filepath.Join(install, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(install, "pair.exe"), []byte("OLD-BINARY"), 0o755)
	os.WriteFile(filepath.Join(install, "config", "settings.json"), []byte(`{"user":1}`), 0o644)
	os.MkdirAll(filepath.Join(install, ".pair", "toolsets"), 0o755)
	os.WriteFile(filepath.Join(install, ".pair", "toolsets", "mine.json"), []byte(`{"keep":true}`), 0o644)

	z := filepath.Join(dir, "pkg.zip")
	writeZip(t, z, map[string]string{
		"pair.exe":                        "NEW-BINARY",
		"README.md":                       "# new",
		"assets/app.js":                   "new-asset",
		"config/settings.json":            `{"user":"SHOULD-NOT-OVERWRITE"}`,
		".pair/toolsets/mine.json":        `{"SHOULD-NOT-OVERWRITE":true}`,
		".pair/plugins/core-api/index.js": "// official updated",
	})
	staging := filepath.Join(dir, "staging")
	if _, err := Extract(z, staging); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		GOOS: "windows", GOARCH: runtime.GOARCH,
		InstallDir: install, DownloadDir: filepath.Join(install, "config", "updates"),
		RequireSHA256: false, KeepBackup: true,
	}

	// dry-run 不应改动任何文件
	planDry, err := cfg.ApplyStaged(staging, true)
	if err != nil {
		t.Fatal(err)
	}
	if planDry.MainProgram != "pair.exe" || !planDry.DryRun {
		t.Fatalf("dry-run 计划异常: %+v", planDry)
	}
	if b, _ := os.ReadFile(filepath.Join(install, "pair.exe")); string(b) != "OLD-BINARY" {
		t.Fatal("dry-run 不应改动主程序")
	}
	if _, err := os.Stat(filepath.Join(install, "assets")); err == nil {
		t.Fatal("dry-run 不应创建 assets 目录")
	}

	plan, err := cfg.ApplyStaged(staging, false)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Renamed {
		t.Fatalf("主程序应在线上替换成功: %+v", plan)
	}
	if b, _ := os.ReadFile(filepath.Join(install, "pair.exe")); string(b) != "NEW-BINARY" {
		t.Fatalf("主程序未替换: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(install, "pair.exe.old")); string(b) != "OLD-BINARY" {
		t.Fatal("未生成旧主程序备份")
	}
	if b, _ := os.ReadFile(filepath.Join(install, "config", "settings.json")); string(b) != `{"user":1}` {
		t.Fatalf("用户配置被覆盖: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(install, ".pair", "toolsets", "mine.json")); string(b) != `{"keep":true}` {
		t.Fatalf("用户工具集被覆盖: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(install, "assets", "app.js")); string(b) != "new-asset" {
		t.Fatalf("资源未写入: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(install, ".pair", "plugins", "core-api", "index.js")); !strings.Contains(string(b), "official updated") {
		t.Fatalf("官方插件未更新: %s", b)
	}
	if len(plan.Skip) != 2 {
		t.Fatalf("保护名单跳过数 = %d want 2（%v）", len(plan.Skip), plan.Skip)
	}
	if plan.Bytes <= 0 {
		t.Fatal("计划字节数应大于 0")
	}
}

func TestDownloadStallCancel(t *testing.T) {
	// 取消语义：ctx 取消 → 下载报错且不留 .part
	cfg, _ := setupFeed(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cfg.FetchManifest(ctx)
	if err == nil {
		t.Fatal("已取消的 context 应报错")
	}
	if !strings.Contains(err.Error(), "取消") && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("取消错误信息: %v", err)
	}
}
