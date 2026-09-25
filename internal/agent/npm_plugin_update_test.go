package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// resetNpmPkgMissingCache 清空推断包名负缓存（全局 map，跨测试残留会互相影响）。
func resetNpmPkgMissingCache() {
	npmPkgMissingMu.Lock()
	npmPkgMissingCache = map[string]time.Time{}
	npmPkgMissingMu.Unlock()
}

// TestInferOfficialPkg 推断候选 npm 包名：
//   - manifest name 是 scoped 包名（含 "/"）→ 直接作包名
//   - 否则按官方约定 @paircode/<磁盘插件名>
func TestInferOfficialPkg(t *testing.T) {
	cases := []struct{ dir, manifest, want string }{
		{"tool-art", "tool-art", "@paircode/tool-art"},
		{"tool-art", "", "@paircode/tool-art"},
		{"tool-art", "  tool-art  ", "@paircode/tool-art"},
		{"my-plugin", "@paircode/my-plugin", "@paircode/my-plugin"},
		{"my-plugin", "@otherorg/x", "@otherorg/x"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := inferOfficialPkg(c.dir, c.manifest); got != c.want {
			t.Errorf("inferOfficialPkg(%q,%q) = %q, want %q", c.dir, c.manifest, got, c.want)
		}
	}
}

// TestNPMPluginUpdateProbe 更新探测（单插件版本对比）：
//   - 无 config.npm 元数据 → 官方约定推断 + registry 校验通过才算 npm 来源
//   - 校验失败（404）→ 只回本地版本，不误报更新；且走负缓存不再重复请求
//   - 有 config.npm 元数据 → 声明优先（current 用声明版本，失败填 error）
func TestNPMPluginUpdateProbe(t *testing.T) {
	var hits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if strings.Contains(r.RequestURI, "%2Ftool-art") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "@paircode/tool-art", "version": "0.3.2",
				"dist": map[string]any{"tarball": "http://example.invalid/t.tgz"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oldBase := npmRegistryBaseOverride
	npmRegistryBaseOverride = srv.URL
	defer func() { npmRegistryBaseOverride = oldBase }()
	resetNpmPkgMissingCache()
	defer resetNpmPkgMissingCache()

	// ① 无元数据 + 本地 0.3.1 → 推断成功，可更新
	got := npmPluginUpdateProbe("tool-art", npmPluginMeta{ManifestName: "tool-art", LocalVersion: "0.3.1"})
	if got["pkg"] != "@paircode/tool-art" {
		t.Fatalf("① pkg = %v, want @paircode/tool-art（推断）", got["pkg"])
	}
	if got["current"] != "0.3.1" || got["latest"] != "0.3.2" {
		t.Fatalf("① current/latest = %v/%v, want 0.3.1/0.3.2", got["current"], got["latest"])
	}
	if got["updateable"] != true {
		t.Fatalf("① updateable = %v, want true", got["updateable"])
	}
	if got["error"] != "" {
		t.Fatalf("① error = %v, want 空（推断来源不报错）", got["error"])
	}

	// ② 无元数据 + 本地已是 latest → 不可更新
	got = npmPluginUpdateProbe("tool-art", npmPluginMeta{ManifestName: "tool-art", LocalVersion: "0.3.2"})
	if got["updateable"] != false {
		t.Fatalf("② updateable = %v, want false（本地即 latest）", got["updateable"])
	}

	// ③ 非官方包（registry 404）→ 只回本地版本，不提示更新
	before := atomic.LoadInt32(&hits)
	got = npmPluginUpdateProbe("test-local-only-pkg", npmPluginMeta{ManifestName: "test-local-only-pkg", LocalVersion: "1.4.0"})
	if got["pkg"] != "" || got["latest"] != "" || got["updateable"] != false {
		t.Fatalf("③ 非 npm 来源应无 pkg/latest/updateable，实际 %v", got)
	}
	if got["current"] != "1.4.0" {
		t.Fatalf("③ current = %v, want 1.4.0（本地版本仍要回给前端）", got["current"])
	}
	// 负缓存：第二次探测不再发请求
	_ = npmPluginUpdateProbe("test-local-only-pkg", npmPluginMeta{ManifestName: "test-local-only-pkg", LocalVersion: "1.4.0"})
	if after := atomic.LoadInt32(&hits); after != before+1 {
		t.Fatalf("③ 负缓存未生效：请求数 %d → %d（期望仅 +1）", before, after)
	}

	// ④ 已声明 npm 来源 → 声明优先（current 取声明版本）
	got = npmPluginUpdateProbe("some-dir", npmPluginMeta{
		Pkg: "@paircode/tool-art", Version: "0.3.0", ManifestName: "some-dir", LocalVersion: "9.9.9",
	})
	if got["current"] != "0.3.0" || got["latest"] != "0.3.2" || got["updateable"] != true {
		t.Fatalf("④ 声明来源对比异常: %v", got)
	}

	// ⑤ 声明来源但本地无版本记录 → 明确报错（不误报更新）
	got = npmPluginUpdateProbe("some-dir", npmPluginMeta{Pkg: "@paircode/tool-art", Version: ""})
	if got["error"] == "" {
		t.Fatalf("⑤ 无本地版本记录应填 error，实际 %v", got)
	}
	if got["updateable"] != false {
		t.Fatalf("⑤ 不应报可更新，实际 %v", got["updateable"])
	}
}

// TestNPMPluginCheckUpdatesScan 扫描集成：磁盘插件包（无 config.npm）+ 桩 registry，
// 验证清单包含该包（推断出的 pkg/current/latest）且按 name 排序。
func TestNPMPluginCheckUpdatesScan(t *testing.T) {
	var hits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if strings.Contains(r.RequestURI, "%2Ftool-update-probe-ut") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "@paircode/tool-update-probe-ut", "version": "2.5.0",
				"dist": map[string]any{"tarball": "http://example.invalid/t.tgz"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	oldBase := npmRegistryBaseOverride
	npmRegistryBaseOverride = srv.URL
	defer func() { npmRegistryBaseOverride = oldBase }()
	resetNpmPkgMissingCache()
	defer resetNpmPkgMissingCache()

	// 构造磁盘插件包（模拟手动放置的官方包：无 config.npm，只有 name/version）
	name := "tool-update-probe-ut"
	dir := filepath.Join(globalPluginsDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Skipf("全局插件目录不可写（%v）——跳过扫描集成测试", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	manifest := map[string]any{"name": name, "version": "2.4.0", "type": "plugin", "main": "index.js"}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0o644); err != nil {
		t.Fatalf("写 package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("return { name: '"+name+"', apply() {} }"), 0o644); err != nil {
		t.Fatalf("写 index.js: %v", err)
	}

	list := npmPluginCheckUpdates()
	if len(list) == 0 {
		t.Skip("全局插件目录为空——跳过扫描集成测试")
	}
	// 排序校验（name 升序）
	for i := 1; i < len(list); i++ {
		prev, _ := list[i-1]["name"].(string)
		cur, _ := list[i]["name"].(string)
		if prev > cur {
			t.Fatalf("清单未按 name 排序：%q 在 %q 之前", prev, cur)
		}
	}
	var found map[string]any
	for _, e := range list {
		if e["name"] == name {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatalf("清单中未找到 %s（共 %d 条）", name, len(list))
	}
	if found["pkg"] != "@paircode/"+name {
		t.Fatalf("pkg = %v, want @paircode/%s", found["pkg"], name)
	}
	if found["current"] != "2.4.0" || found["latest"] != "2.5.0" || found["updateable"] != true {
		t.Fatalf("版本对比异常: %v", found)
	}
	t.Logf("扫描 %d 个磁盘插件包，%s 推断为 npm 来源并可更新（%v → %v）",
		len(list), name, found["current"], found["latest"])
}
