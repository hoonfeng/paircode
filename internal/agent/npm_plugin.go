// npm_plugin.go — npm/cordis 插件市场：从 npm registry 搜索 cordis 插件，
// 下载 tarball 取 main 源码 → 追加到 .pair/cordis.patch.json（跨重启存续）→
// 立即经 goja 宿主装载（esbuild 内联打包 + 非相对导入 mock）。
//
// 「插件市场」= npm 上的 cordis 插件生态
// （外部项目的插件市场指令转发 pnpm）；我方用 goja 沙箱 +
// 内置 esbuild 编译器直接执行 cordis 插件源码（ESM export default），
// 无需 node_modules——非相对导入 mock 空模块，相对导入内联打包。

package agent

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// npmOfficialRegistry npm 官方源（仅用于「tarball 域名对齐」判断，不作默认源）。
const npmOfficialRegistry = "https://registry.npmjs.org"

// npmMirrorRegistry 默认 registry = npmmirror 镜像。
// ★ 2026-09-25：此前默认官方源，国内直连极慢（实测 5.78MB 包 ≈3KB/s，
// 90s 只下 274KB，市场「安装」在前端 30s 超时内必然失败）；镜像同包 1.6s 下完。
const npmMirrorRegistry = "https://registry.npmmirror.com"

// npmRegistryBase npm registry 地址（测试可替换为 httptest server；
// 运行时可用环境变量 PAIRCODE_NPM_REGISTRY 覆盖——本地市场/私有 registry）。
var npmRegistryBase = func() string {
	if v := os.Getenv("PAIRCODE_NPM_REGISTRY"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return npmMirrorRegistry
}()

// npmFetchTimeout npm 拉取超时（下载 tarball 可能较慢：冷门包经镜像回源仍然慢，
// 大包如 @paircode/tool-model 体积数 MB）。★ 2026-09-25：120s → 300s，
// 避免慢网络下「下载一半被掐断」被误判为包不可用。
var npmFetchTimeout = 300 * time.Second

// npmHTTPClient 共享客户端：放宽 TLS 握手超时（慢网络下 Go 默认 10s 不够）。
// ★ 2026-09-25：TLS 30→60s、响应头 30→120s（原 30s 是「市场装包必失败」的
//   实际卡点：镜像回源冷包首字节可能 >30s，总超时 120s 根本来不及生效）。
var npmHTTPClient = &http.Client{
	Timeout: npmFetchTimeout,
	Transport: &http.Transport{
		TLSHandshakeTimeout:   60 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
		MaxIdleConns:          4,
	},
}

// npmMirrorTarball 把 registry 返回的 tarball 地址对齐到当前 registry base：
// 旧版本写入的安装元数据 / 缓存 / 镜像未改写的响应里可能残留官方源地址，
// 直接下载会静默退回慢源 —— 统一把 registry.npmjs.org 前缀改写成当前 base。
// （当前 base 就是官方源、或本就是当前 base 的地址 → 原样返回。）
func npmMirrorTarball(tarball string) string {
	if tarball == "" || npmRegistryBase == npmOfficialRegistry {
		return tarball
	}
	if strings.HasPrefix(tarball, npmOfficialRegistry+"/") {
		return npmRegistryBase + strings.TrimPrefix(tarball, npmOfficialRegistry)
	}
	return tarball
}

// npmHTTPGet 带一次重试的 GET（npm registry 偶发 TLS/网络抖动）。
func npmHTTPGet(url string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := npmHTTPClient.Get(url)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return nil, lastErr
}

// ─── npm 包拉取 ─────────────────────────────────────────────

// npmPackageInfo npm registry 包信息（latest 版本 + tarball 地址）。
type npmPackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
	Description string         `json:"description"`
	Manifest    map[string]any `json:"-"`
}

// fetchNPMInfo 查询 npm registry 获取包 latest 信息。
func fetchNPMInfo(pkg string) (*npmPackageInfo, error) {
	info, missing, err := fetchNPMInfoChecked(pkg)
	if err != nil {
		return nil, err
	}
	if missing {
		return nil, fmt.Errorf("npm %s 不存在或不可达（HTTP %d）", pkg, http.StatusNotFound)
	}
	return info, nil
}

// fetchNPMInfoChecked 查询 npm registry latest，并区分「包不存在」（missing=true，
// HTTP 404）与「瞬时网络错误」——推断官方包名（无 config.npm 元数据）时，
// 只有"确定不存在"才判为非 npm 来源，并可安全负缓存。
// ★ 2026-09-19 新增（市场更新提示：手动放置/复制的官方插件包没有安装元数据）。
func fetchNPMInfoChecked(pkg string) (*npmPackageInfo, bool, error) {
	url := npmRegistryBase + "/" + strings.ReplaceAll(pkg, "/", "%2F") + "/latest"
	resp, err := npmHTTPGet(url)
	if err != nil {
		return nil, false, fmt.Errorf("查询 npm %s 失败: %v", pkg, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("npm %s 不存在或不可达（HTTP %d）", pkg, resp.StatusCode)
	}
	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, false, fmt.Errorf("解析 npm 响应失败: %v", err)
	}
	if info.Dist.Tarball == "" {
		return nil, false, fmt.Errorf("npm %s 无 tarball 地址", pkg)
	}
	return &info, false, nil
}

// fetchNPMPackage 下载 npm 包 tarball 并解压到临时目录，返回：
//   - dir：解压出的 package/ 目录（含源码，供 esbuild ResolveDir）
//   - manifest：package.json 内容
//   - err
//
// 调用方负责 os.RemoveAll(dir)。
func fetchNPMPackage(info *npmPackageInfo) (dir string, manifest map[string]any, err error) {
	resp, err := npmHTTPGet(npmMirrorTarball(info.Dist.Tarball))
	if err != nil {
		return "", nil, fmt.Errorf("下载 tarball 失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("下载 tarball 失败（HTTP %d）", resp.StatusCode)
	}

	tmp, err := os.MkdirTemp("", "npm_plugin_*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }

	// gzip + tar 解压（npm tarball 顶层目录为 package/）
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("解压 tarball 失败: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("解压 tarball 失败: %v", err)
		}
		// 防路径穿越：只保留 package/ 前缀
		name := filepath.Clean(hdr.Name)
		rel := strings.TrimPrefix(name, "package"+string(filepath.Separator))
		if name == "package" || strings.HasPrefix(name, "package"+string(filepath.Separator)) {
			target := filepath.Join(tmp, rel)
			if !strings.HasPrefix(target, filepath.Clean(tmp)+string(filepath.Separator)) {
				cleanup()
				return "", nil, fmt.Errorf("tarball 含非法路径: %s", hdr.Name)
			}
			switch hdr.Typeflag {
			case tar.TypeDir:
				os.MkdirAll(target, 0o755)
			case tar.TypeReg:
				os.MkdirAll(filepath.Dir(target), 0o755)
				f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
				if err != nil {
					cleanup()
					return "", nil, err
				}
				if _, err := io.Copy(f, tr); err != nil {
					f.Close()
					cleanup()
					return "", nil, err
				}
				f.Close()
			}
		}
	}

	manifest = map[string]any{}
	if b, err := os.ReadFile(filepath.Join(tmp, "package.json")); err == nil {
		_ = json.Unmarshal(b, &manifest)
	}
	return tmp, manifest, nil
}

// npmPackageMain 从 package.json 提取主入口文件（默认 index.js）。
func npmPackageMain(manifest map[string]any) string {
	if m, ok := manifest["main"].(string); ok && strings.TrimSpace(m) != "" {
		m = strings.TrimSpace(m)
		if !strings.HasSuffix(m, ".js") && !strings.HasSuffix(m, ".ts") && !strings.HasSuffix(m, ".cjs") && !strings.HasSuffix(m, ".mjs") {
			// 无扩展名 → 补 .js（node 默认解析）
			m += ".js"
		}
		return m
	}
	// exports 字段的 import 入口
	if exp, ok := manifest["exports"].(map[string]any); ok {
		if dot, ok := exp["."].(map[string]any); ok {
			if imp, ok := dot["import"].(string); ok {
				return imp
			}
		}
	}
	return "index.js"
}

// ─── cordis.patch.json 读写 ──────────────────────────────────

// cordisPatchPlugin patch 插件条目（与 LoadCordisPatch 的解析结构对齐）。
type cordisPatchPlugin struct {
	Code     string         `json:"code"`
	Language string         `json:"language"`
	Purpose  string         `json:"purpose"`
	Config   map[string]any `json:"config,omitempty"`
}

type cordisPatchDoc struct {
	Plugins []cordisPatchPlugin `json:"plugins"`
}

// readCordisPatch 读取 patch 文件（不存在 → 空文档）。
func readCordisPatch(path string) (*cordisPatchDoc, error) {
	doc := &cordisPatchDoc{Plugins: []cordisPatchPlugin{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, doc); err != nil {
		return nil, fmt.Errorf("cordis.patch.json 解析失败: %v", err)
	}
	if doc.Plugins == nil {
		doc.Plugins = []cordisPatchPlugin{}
	}
	return doc, nil
}

func writeCordisPatch(path string, doc *cordisPatchDoc) error {
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// appendPatchNPMPlugin 追加/更新一个 npm 插件条目到 patch。
// 已存在相同 config.npm 的条目 → 原位更新（幂等）。
func appendPatchNPMPlugin(path, pkg, version, code, purpose string) error {
	doc, err := readCordisPatch(path)
	if err != nil {
		return err
	}
	key := pkg + "@" + version
	entry := cordisPatchPlugin{
		Code:     code,
		Language: "js",
		Purpose:  purpose,
		Config:   map[string]any{"npm": key},
	}
	for i := range doc.Plugins {
		if src, _ := doc.Plugins[i].Config["npm"].(string); src == key {
			doc.Plugins[i] = entry // 原位更新
			return writeCordisPatch(path, doc)
		}
	}
	doc.Plugins = append(doc.Plugins, entry)
	return writeCordisPatch(path, doc)
}

// removePatchNPMPlugin 从 patch 移除指定 npm 插件条目（返回是否移除）。
func removePatchNPMPlugin(path, pkg string) (bool, error) {
	doc, err := readCordisPatch(path)
	if err != nil {
		return false, err
	}
	out := doc.Plugins[:0]
	removed := false
	for _, p := range doc.Plugins {
		src, _ := p.Config["npm"].(string)
		if src == pkg || strings.HasPrefix(src, pkg+"@") {
			removed = true
			continue
		}
		out = append(out, p)
	}
	if !removed {
		return false, nil
	}
	doc.Plugins = out
	return true, writeCordisPatch(path, doc)
}

// ─── 安装 ────────────────────────────────────────────────────

// marketInstallNPMPlugin 安装 npm/cordis 插件：
// 下载 tarball → 取 main 源码 → 追加 .pair/cordis.patch.json（跨重启）→
// 立即经 goja 宿主装载。装载失败回滚 patch（与工具集安装失败回滚一致）。
// npmInstallPlugin 安装 npm 插件（通用能力，ctx.npm.install 暴露）：
// 下载 tarball → goja 沙箱或 Node 运行时桥 → 固化为磁盘插件包
// <InstallDir>/.pair/plugins/<name>/（重启 LoadGlobalPlugins 自动装配）。
func npmMarketInstall(pkg string) (string, error) {
	if pkg == "" {
		return "", fmt.Errorf("npm 插件缺包名")
	}

	info, err := fetchNPMInfo(pkg)
	if err != nil {
		return "", err
	}
	dir, manifest, err := fetchNPMPackage(info)
	if err != nil {
		return "", err
	}
	// 需真实 node 环境（npm 依赖或 cordis4）→ 走 Node 桥；
	// 否则 goja 沙箱（快、无需 node）。
	if nodePluginNeedsNode(manifest) {
		info.Manifest = manifest // 原生依赖提示用
		msg, err := marketInstallNPMPluginNode(info, dir, true)
		if err != nil {
			return "", err
		}
		return msg, nil
	}
	defer os.RemoveAll(dir)

	mainFile := npmPackageMain(manifest)
	mainPath, err := safeJoinUnder(dir, mainFile)
	if err != nil {
		return "", fmt.Errorf("包 %s 主入口路径非法（%s）: %v", pkg, mainFile, err)
	}
	codeBytes, err := os.ReadFile(mainPath)
	if err != nil {
		// 兜底：尝试 index.js
		mainPath = filepath.Join(dir, "index.js")
		codeBytes, err = os.ReadFile(mainPath)
		if err != nil {
			return "", fmt.Errorf("包 %s 主入口 %s 不存在（无 index.js）", pkg, mainFile)
		}
	}
	code := string(codeBytes)
	// ★ 2026-09-18 完整落盘①：读 client 半与 scope（对齐官方发布包白名单
	//   index.js/client.js/assets/bin；旧逻辑只固化 main→index.js，带 UI 半或
	//   平台二进制的插件装后缺件——实测 @paircode/tool-art 仅剩 index.js）。
	//   client 仅接受包根纯文件名（生态约定 client.js），拒绝任何路径穿越。
	clientCode := ""
	if cf, ok := manifest["client"].(string); ok {
		cf = strings.TrimSpace(cf)
		if cf != "" && filepath.Base(cf) == cf && cf != "." && cf != ".." && !strings.ContainsAny(cf, `/\`) {
			if cb, rerr := os.ReadFile(filepath.Join(dir, cf)); rerr == nil {
				clientCode = string(cb)
			} else {
				log.Printf("[npm-install] %s client 半读取失败（%s）: %v", pkg, cf, rerr)
			}
		} else if cf != "" {
			log.Printf("[npm-install] %s client 字段非法（%q），跳过 client 半", pkg, cf)
		}
	}
	scope := "project"
	if sc, ok := manifest["scope"].(string); ok && strings.TrimSpace(sc) == "global" {
		scope = "global"
	}
	if scope != "global" && scope != "project" { // 防御：下游按两态读取
		scope = "project"
	}
	purpose := info.Description
	if purpose == "" {
		purpose = "npm cordis 插件 " + pkg
	}

	// ★ 2026-08-20 落盘统一：npm 插件固化为磁盘插件包
	//   <InstallDir>/.pair/plugins/<name>/（与 cordis(op=define) 固化路径一致，
	//   重启 LoadGlobalPlugins 自动装配）；不再写 .pair/cordis.patch.json。
	pluginName := npmPluginDiskName(pkg)
	// 磁盘落盘名白名单校验：作为 <InstallDir>/.pair/plugins/ 下的目录名，
	// 必须是单段安全名（拒绝空、"."、".."、路径分隔符——防目录穿越）。
	if pluginName == "" || pluginName == "." || pluginName == ".." || strings.ContainsAny(pluginName, `/\`) {
		return "", fmt.Errorf("npm 包名 %q 无法安全落盘（非法目录名）", pkg)
	}

	// 先装载（成功才固化；防重启后依旧装载失败的状态不一致）
	ph := GetGlobalPluginHost()
	var defID string
	if ph != nil {
		// client 半随定义登记（立即装载会话前端即可取内联源码，不等重启）。
		// DefineJSCodeDir 即 clientCode 空的 DefineJSCodeFull（jsplugin.go 包装），
		// 此为既有接口的超集升级：仅新增 client 语法预检与登记，装载/回滚语义不变；
		// clientCode 与 host 半 code 同信任模型（均为包作者的源码文本，非拼接数据），
		// 无注入面。
		defID, err = ph.DefineJSCodeFull(code, "js", purpose, dir, clientCode)
		if err != nil {
			return "", fmt.Errorf("插件编译失败（可能依赖 node 模块）: %v", err)
		}
		def, _ := ph.GetJSDef(defID)
		if def != nil {
			// 打卸载锚点（新磁盘插件形态无 config.npm；removeNPMPluginDefs 靠它匹配）
			ph.SetJSDefConfig(defID, "npm", pkg)
			def.scope = scope // 磁盘包 scope 对齐（重启装配按磁盘包同源读取）
			if err := ph.LoadJSDynamic(def); err != nil {
				return "", fmt.Errorf("插件装载失败（可能依赖 node 模块）: %v", err)
			}
		}
	}

	// 固化为磁盘插件包（**整包**：保 assets/client.js/lib + 原始 manifest 的 dsh.ui
	// 等字段，重启自动装配）——★ 2026-09 L1 修 G2：此前只写 main 源码，
	// UI 插件（dsh.ui + assets bundle）市场安装后永远装不上。
	// ★ config.npm 记录 npm 来源与版本（更新机制元数据：checkUpdates 扫描它）
	// ★ 2026-09-24 合并说明：本地曾用 syncGlobalPlugin（单文件）+ copyPluginExtras
	//   （assets/bin/README 附加落盘）组合解决同类问题；现统一改用官方整包方案
	//   （覆盖更全：含 lib/，且 package.json 保留原始字段）。本地 scope 透传语义
	//   由 manifest 原样保留承接（无 scope 字段时同步写入默认 project，行为一致）。
	if err := syncGlobalPluginPackage(pluginName, purpose, dir, manifest, map[string]any{
		"npm": map[string]any{"pkg": pkg, "version": info.Version},
	}); err != nil {
		// 失败即清理（本地语义保留）：防残留「目录在但残缺」的插件包；
		// removeGlobalPluginPackage 自带范围收敛（仅限全局插件目录内）。
		_ = removeGlobalPluginPackage(filepath.Join(globalPluginsDir(), pluginName))
		if ph != nil && defID != "" {
			_ = ph.Unload(defID)
			_ = ph.RemoveJSDef(defID)
		}
		return "", fmt.Errorf("固化插件包 %s 失败: %v", pluginName, err)
	}

	// ★ 附加文件（assets/bin/README、lib/）已由上方 syncGlobalPluginPackage 整包
	//   落盘覆盖（2026-09-24 合并官方 v1.6.6 起统一为单一落盘路径）——不再单独
	//   调用 copyPluginExtras；该函数保留供组件级复用与回归基准（含目标收敛、
	//   symlink 防护、原子换入等校验）。

	msg := fmt.Sprintf("✅ 已安装 npm 插件「%s」v%s（插件目录 .pair/plugins/%s/，重启自动装配）", pkg, info.Version, pluginName)
	if ph == nil {
		msg += "。已保存。"
	} else {
		msg += "。已立即装载可用。"
	}
	return msg, nil
}

// safeJoinUnder 拼接 base 下的相对路径并拒绝路径穿越（tarball/manifest 内容安全）：
// 绝对路径、（任意段）".."、以及拼接后越界 base 的路径一律报错。
// 允许子目录（如 main="lib/index.js"），但不允许逃出解压根目录。
func safeJoinUnder(base, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("空路径")
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", fmt.Errorf("绝对路径不允许: %q", rel)
	}
	for _, seg := range strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return "", fmt.Errorf("路径含 .. 穿越: %q", rel)
		}
	}
	p := filepath.Join(base, filepath.FromSlash(rel))
	if !strings.HasPrefix(filepath.Clean(p), filepath.Clean(base)+string(filepath.Separator)) {
		return "", fmt.Errorf("路径越界: %q", rel)
	}
	return p, nil
}

// copyDirAtomic 将 src 目录完整拷贝到 dst（先拷贝到同级临时目录，再整目录换入）：
// 拷贝失败时目标保持原状（不出现半拷贝），临时目录被清理。
// Windows 兼容：目录 os.Rename 需目标不存在——换入前先移除旧目标。
func copyDirAtomic(src, dst string) (err error) {
	tmp := dst + ".tmp-copy"
	_ = os.RemoveAll(tmp) // 清理上次残留
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tmp) // 失败不留临时目录
		}
	}()
	if cerr := copyDir(src, tmp); cerr != nil {
		return cerr
	}
	if rerr := os.RemoveAll(dst); rerr != nil {
		return rerr
	}
	return os.Rename(tmp, dst)
}

// copyPluginExtras 把 npm 包解压目录中的插件附加文件完整落盘到磁盘插件包：
// assets/、bin/（目录递归、原子换入）与 README.md（源不存在则跳过——纯 JS 插件
// 属正常形态，静默兼容旧插件；client.js 由 syncGlobalPlugin 依 Client 参数写入）。
// 与 scripts/publish-official-plugins.mjs 的打包白名单保持对称（发布↔安装）。
// 失败语义：仅回滚本函数新增内容（临时目录清理）；已存在的目标目录不被半写破坏。
func copyPluginExtras(srcDir, dstDir string) error {
	// 目标路径收敛校验：必须为干净绝对目录（非根），防异常拼接造成越界写。
	base := filepath.Clean(dstDir)
	if !filepath.IsAbs(base) || base == filepath.Clean(string(filepath.Separator)) {
		return fmt.Errorf("插件目录非法: %q", dstDir)
	}
	for _, sub := range []string{"assets", "bin"} {
		src := filepath.Join(srcDir, sub)
		fi, err := os.Lstat(src) // Lstat：不跟随链接（防链接逃逸写出 base 之外）
		if err != nil {
			if os.IsNotExist(err) {
				continue // 无该目录（纯 JS 插件常见）——静默跳过
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return fmt.Errorf("%s 形态异常（链接/非目录）: %s", sub, src)
		}
		// 落点收敛（filepath.Rel 语义判断越界，防 ../ 逃逸）
		dst := filepath.Join(base, sub)
		if rel, rerr := filepath.Rel(base, dst); rerr != nil || rel == ".." || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, `..\`) {
			return fmt.Errorf("落点越界: %q", dst)
		}
		if cerr := copyDirAtomic(src, dst); cerr != nil {
			return fmt.Errorf("拷贝 %s/ 失败: %v", sub, cerr)
		}
		// bin/ 为可执行产物：Unix 上补可执行位（Windows 无该语义，调用无害——
		// 仅影响只读位；实际执行以扩展名判定）。
		if sub == "bin" {
			_ = filepath.Walk(dst, func(p string, fi os.FileInfo, werr error) error {
				if werr == nil && !fi.IsDir() {
					_ = os.Chmod(p, 0o755)
				}
				return nil
			})
		}
	}
	// README.md（npm 自动入包文件；临时文件 + rename 原子写）
	if b, rerr := os.ReadFile(filepath.Join(srcDir, "README.md")); rerr == nil {
		dstFile := filepath.Join(dstDir, "README.md")
		tmpFile := dstFile + ".tmp"
		if werr := os.WriteFile(tmpFile, b, 0o644); werr != nil {
			return fmt.Errorf("写入 README.md 失败: %v", werr)
		}
		if rerr := os.Rename(tmpFile, dstFile); rerr != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("换入 README.md 失败: %v", rerr)
		}
	}
	return nil
}

// uninstallNPMPlugin 卸载 npm 插件：
// ① 磁盘插件包目录 .pair/plugins/<name>/（2026-08-20 新安装形态）删除；
// ② cordis.patch.json 条目移除（旧安装向后兼容）+ 宿主卸载/移除定义。
func uninstallNPMPlugin(pkg string) error {
	projectRoot := npmPluginProjectRoot()
	var runtime string
	if projectRoot != "" {
		patchPath := filepath.Join(projectRoot, ".pair", "cordis.patch.json")
		doc, err := readCordisPatch(patchPath)
		if err != nil {
			return err
		}
		// 识别 runtime 类型（node 桥插件 vs goja 插件；仅旧 patch 安装有）
		for _, p := range doc.Plugins {
			src, _ := p.Config["npm"].(string)
			if src == pkg || strings.HasPrefix(src, pkg+"@") {
				runtime, _ = p.Config["runtime"].(string)
			}
		}
		removed, err := removePatchNPMPlugin(patchPath, pkg)
		if err != nil {
			return err
		}
		// 不在 patch（新安装走磁盘插件包）→ 删除插件包目录
		if !removed {
			dir := filepath.Join(globalPluginsDir(), npmPluginDiskName(pkg))
			if _, serr := os.Stat(filepath.Join(dir, "package.json")); serr == nil {
				if err := removeGlobalPluginPackage(dir); err != nil {
					return fmt.Errorf("删除插件包目录失败: %w", err)
				}
			} else if !os.IsNotExist(serr) {
				return fmt.Errorf("检查插件包目录失败: %w", serr)
			} else {
				// 磁盘插件包也不存在 → 未安装
				return fmt.Errorf("未找到插件 %s（无 .pair/plugins/%s/，也不在 cordis.patch.json）", pkg, npmPluginDiskName(pkg))
			}
		}
	}
	if runtime == "node" || runtime == "dsh" {
		// Node 桥插件（cordis3 或 Round4 DSH/cordis4）：plugins.json 移除 + 重启桥 + 清理源码目录
		if err := uninstallNodePlugin(pkg); err != nil {
			return err
		}
	}
	// 宿主卸载：遍历 defs，config.npm 匹配的卸载并移除定义
	if ph := GetGlobalPluginHost(); ph != nil {
		ph.removeNPMPluginDefs(pkg)
	}
	return nil
}

// removeNPMPluginDefs 卸载并移除 config.npm 匹配的插件定义（npm 插件卸载用）。
// 新安装形态（磁盘插件包）无 config.npm——按插件目录（d.dir 末尾 == 磁盘包名）匹配。
func (h *PluginHost) removeNPMPluginDefs(pkg string) {
	diskName := npmPluginDiskName(pkg)
	for _, d := range h.JSDefs() {
		if src, _ := d.config["npm"].(string); src == pkg || strings.HasPrefix(src, pkg+"@") {
			_ = h.RemoveJSDef(d.id)
			continue
		}
		dirBase := filepath.Base(filepath.Clean(d.dir))
		if dirBase == diskName || d.name == diskName {
			_ = h.RemoveJSDef(d.id)
		}
	}
}

// primaryWorkspaceRoot 取主工作区根（根快照[0]——WorkspaceRoots 优先，否则 core.Folders[0]）。
func primaryWorkspaceRoot() string {
	if roots := workspaceRootsSnapshot(); len(roots) > 0 {
		return roots[0]
	}
	return ""
}

// npmPluginProjectRoot 取 npm 插件安装/判断用的工作区根（与工具集安装同链路）。
func npmPluginProjectRoot() string {
	if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil && ph.Context().WorkspaceRoot != "" {
		return ph.Context().WorkspaceRoot
	}
	if r := primaryWorkspaceRoot(); r != "" {
		return r
	}
	return core.Root()
}

// npmPluginInstalled 检查 npm 插件是否已安装：
// ① 磁盘插件包目录 .pair/plugins/<name>/（2026-08-20 新安装形态）；
// ② cordis.patch.json config.npm 匹配（旧安装向后兼容）。
func npmPluginInstalled(id string) bool {
	if _, err := os.Stat(filepath.Join(globalPluginsDir(), npmPluginDiskName(id), "package.json")); err == nil {
		return true
	}
	projectRoot := npmPluginProjectRoot()
	if projectRoot == "" {
		return false
	}
	doc, err := readCordisPatch(filepath.Join(projectRoot, ".pair", "cordis.patch.json"))
	if err != nil {
		return false
	}
	for _, p := range doc.Plugins {
		src, _ := p.Config["npm"].(string)
		if src == id || strings.HasPrefix(src, id+"@") {
			return true
		}
	}
	return false
}

// npmPluginDiskName npm 包名 → 磁盘插件包目录名：
// @scope/pkg → pkg（目录名不友好字符清理）；裸名原样。
func npmPluginDiskName(pkg string) string {
	name := strings.TrimPrefix(pkg, "@")
	if i := strings.Index(name, "/"); i > 0 {
		return name[i+1:]
	}
	// 裸名官方形态（市场 searchNpmPlugins 第二约定）：paircode-plugin-<name> 与
	// @paircode/<name> 语义等价 → 剥前缀，磁盘名对齐短名（同插件替换/升级不重名）。
	return strings.TrimPrefix(name, "paircode-plugin-")
}

// ─── npm 插件更新机制（2026-08-20：基于 npm registry 版本对比）────────
// 元数据载体：磁盘插件包 package.json 的 config.npm = { pkg, version }
// （npmMarketInstall 固化时写入；装配时透传 apply(ctx, config) 无副作用）。

// npmPluginMeta 单个磁盘插件的 npm 来源元数据。
type npmPluginMeta struct {
	Name         string `json:"name"`    // 磁盘插件名（.pair/plugins/<name>/）
	Pkg          string `json:"pkg"`     // npm 包名（@scope/pkg 或裸名）
	Version      string `json:"version"` // 已安装的 npm 版本（config.npm.version）
	ManifestName string `json:"-"`       // ★ package.json 的 name 字段（推断包名用：scoped 名直接可作包名）
	LocalVersion string `json:"-"`       // ★ package.json 的 version 字段（无安装元数据时的本地版本）
}

// npmPluginMetaPath 读磁盘插件包的 config.npm 元数据；无则返回零值。
func npmPluginMetaPath(pkgDir string) (npmPluginMeta, error) {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return npmPluginMeta{}, err
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return npmPluginMeta{}, err
	}
	meta := npmPluginMeta{Name: pkg.Name, ManifestName: pkg.Name, LocalVersion: pkg.Version}
	if cfg, ok := pkg.Config["npm"].(map[string]any); ok {
		meta.Pkg, _ = cfg["pkg"].(string)
		meta.Version, _ = cfg["version"].(string)
	}
	return meta, nil
}

// npmPluginMetaByPkg 按 npm 包名（或磁盘插件名）查已装元数据。
// 返回零值 meta 表示未安装/无 npm 来源。
// ★ 2026-09-19：手动放置的官方插件包（无 config.npm）也按目录名命中，并补上
// 推断包名 + 本地版本——否则市场对这些包的「更新」动作会判为「非 npm 来源」。
func npmPluginMetaByPkg(pkg string) npmPluginMeta {
	base := globalPluginsDir()
	diskName := npmPluginDiskName(pkg)
	if meta, err := npmPluginMetaPath(filepath.Join(base, diskName)); err == nil && (meta.Pkg != "" || meta.LocalVersion != "") {
		if meta.Pkg == "" {
			cand := inferOfficialPkg(diskName, meta.ManifestName)
			if cand == "" {
				cand = pkg
			}
			meta.Pkg, meta.Version = cand, meta.LocalVersion
		}
		return meta
	}
	// 入参可能是磁盘插件名而非 npm 全名：扫描全部目录匹配 config.npm.pkg
	entries, err := os.ReadDir(base)
	if err != nil {
		return npmPluginMeta{}
	}
	for _, e := range entries {
		if !isPluginDirEntry(base, e) {
			continue
		}
		meta, err := npmPluginMetaPath(filepath.Join(base, e.Name()))
		if err != nil || meta.Pkg == "" {
			continue
		}
		if meta.Pkg == pkg || meta.Name == pkg {
			return meta
		}
	}
	return npmPluginMeta{}
}

// ─── 更新检查：安装元数据 + 官方约定推断 ────────────────────────────
// npmOfficialPkgPrefix 官方插件 npm scope 前缀（推断依据：磁盘插件名 → @paircode/<name>）。
const npmOfficialPkgPrefix = "@paircode/"

// npmPkgMissingTTL 推断包「确定不存在」（404）结论的负缓存时长。非 npm 来源的
// 磁盘插件包（内置包/UI 包）在 registry 上查不到，缓存可避免每次点「检查更新」
// 都对几十个包打一轮无用网络往返；只缓存 404，网络抖动不入缓存。
const npmPkgMissingTTL = 10 * time.Minute

var (
	npmPkgMissingMu    sync.Mutex
	npmPkgMissingCache = map[string]time.Time{} // pkg → 判定 404 的时间
)

// inferOfficialPkg 推断磁盘插件包的候选 npm 包名（未校验存在性）：
//   - manifest name 含 "/"（scoped 包名，如 @scope/pkg）→ 直接作包名
//   - 否则按官方命名约定 @paircode/<磁盘插件名>
func inferOfficialPkg(dirName, manifestName string) string {
	if name := strings.TrimSpace(manifestName); strings.Contains(name, "/") {
		return name
	}
	d := strings.TrimSpace(dirName)
	if d == "" {
		return ""
	}
	return npmOfficialPkgPrefix + d
}

// npmProbeLatestCached 查包 latest（带 404 负缓存）：
// 返回 ok=false 表示「非 npm 包或查询失败」——不误报更新。
// 成功与瞬时错误都不缓存（保证点「检查更新」拿到的是实时 latest）。
func npmProbeLatestCached(pkg string) (*npmPackageInfo, bool) {
	npmPkgMissingMu.Lock()
	at, cached := npmPkgMissingCache[pkg]
	npmPkgMissingMu.Unlock()
	if cached && time.Since(at) < npmPkgMissingTTL {
		return nil, false
	}
	info, missing, err := fetchNPMInfoChecked(pkg)
	if err != nil {
		return nil, false
	}
	if missing {
		npmPkgMissingMu.Lock()
		npmPkgMissingCache[pkg] = time.Now()
		npmPkgMissingMu.Unlock()
		return nil, false
	}
	return info, true
}

// npmPluginUpdateProbe 单插件版本对比：（name, pkg, current, latest, updateable, error）
//   - pkg 为空 = 非 npm 来源（内置/工具集插件）→ 前端只显示本地版本，不提示更新
//   - error 仅对「已声明 npm 来源」的失败填写；推断失败的静默（避免满屏网络噪音）
func npmPluginUpdateProbe(dirName string, meta npmPluginMeta) map[string]any {
	info := map[string]any{
		"name": dirName, "pkg": "", "current": meta.LocalVersion,
		"latest": "", "updateable": false, "error": "",
	}
	if meta.Pkg == "" {
		// ★ 无安装元数据（手动放置/复制到磁盘的官方插件包）：约定推断 + registry 校验
		cand := inferOfficialPkg(dirName, meta.ManifestName)
		if cand == "" {
			return info
		}
		remote, ok := npmProbeLatestCached(cand)
		if !ok {
			return info // 非 npm 来源 / 查询失败：仅回本地版本
		}
		info["pkg"] = cand
		info["latest"] = remote.Version
		if meta.LocalVersion == "" {
			return info
		}
		info["updateable"] = remote.Version != meta.LocalVersion
		return info
	}
	info["pkg"] = meta.Pkg
	info["current"] = meta.Version
	if meta.Version == "" {
		info["error"] = "本地无版本记录（旧安装，重新安装一次即可支持更新）"
		return info
	}
	remote, err := fetchNPMInfo(meta.Pkg)
	if err != nil {
		info["error"] = "查询 registry 失败: " + err.Error()
		return info
	}
	info["latest"] = remote.Version
	info["updateable"] = remote.Version != meta.Version
	return info
}

// npmPluginCheckUpdates 扫描全部磁盘插件包，逐个与 npm registry latest 对比，
// 返回版本对照清单（**含非 npm 来源条目**——前端据此显示本地版本）。
// ★ 2026-09-19：除 config.npm 声明的安装元数据外，按官方约定推断包名并 registry
// 校验——手动放置/复制的官方插件包此前完全不参与检查（已安装面板永远显示
// "无 npm 来源插件"、市场也永远不提示更新）。
// 单个查询失败不阻塞整体（error 字段记录，方便前端提示网络问题）；6 路并发。
// ★ 返回 []map[string]any（小写 key）：goja 转 JS 对象时用字段名而非 json tag，
//
//	结构体字段名大写会导致前端/agent 取不到（2026-08-20 实测修正）。
func npmPluginCheckUpdates() []map[string]any {
	base := globalPluginsDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	type probeTarget struct {
		name string
		meta npmPluginMeta
	}
	targets := make([]probeTarget, 0, len(entries))
	for _, e := range entries {
		if !isPluginDirEntry(base, e) {
			continue
		}
		meta, err := npmPluginMetaPath(filepath.Join(base, e.Name()))
		if err != nil {
			continue // 无 package.json / 解析失败 → 不是插件包目录
		}
		targets = append(targets, probeTarget{name: e.Name(), meta: meta})
	}
	out := make([]map[string]any, 0, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6) // 并发上限：几十个包串行查询太慢
	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t probeTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			info := npmPluginUpdateProbe(t.name, t.meta)
			mu.Lock()
			out = append(out, info)
			mu.Unlock()
		}(t)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool {
		ni, _ := out[i]["name"].(string)
		nj, _ := out[j]["name"].(string)
		return ni < nj
	})
	return out
}

// npmPluginUpdateInfo 单个插件的更新检查结果。
type npmPluginUpdateInfo struct {
	Name       string `json:"name"`       // 磁盘插件名
	Pkg        string `json:"pkg"`        // npm 包名
	Current    string `json:"current"`    // 已装版本（config.npm.version）
	Latest     string `json:"latest"`     // registry latest
	Updateable bool   `json:"updateable"` // 是否有新版
	Error      string `json:"error,omitempty"`
}

// npmPluginUpdate 更新指定 npm 插件到 registry latest：
//  1. 预查 latest + 版本对比（失败/已最新 → 旧插件不受影响）
//  2. 卸载旧插件（磁盘目录 + 宿主 def）
//  3. 重新安装（npmMarketInstall 下载新版本 → 装载 → 固化）
//
// 返回人类可读消息。
func npmPluginUpdate(pkg string) (string, error) {
	meta := npmPluginMetaByPkg(pkg)
	if meta.Pkg == "" {
		return "", fmt.Errorf("插件 %s 非 npm 来源或未安装（无法更新）", pkg)
	}
	info, err := fetchNPMInfo(meta.Pkg)
	if err != nil {
		return "", fmt.Errorf("查询 %s 最新版本失败: %v", meta.Pkg, err)
	}
	if meta.Version != "" && info.Version == meta.Version {
		return fmt.Sprintf("「%s」已是最新版本 v%s，无需更新", meta.Pkg, meta.Version), nil
	}
	oldVer := meta.Version
	if err := uninstallNPMPlugin(meta.Pkg); err != nil {
		return "", fmt.Errorf("卸载旧版本失败: %v", err)
	}
	msg, err := npmMarketInstall(meta.Pkg)
	if err != nil {
		return "", fmt.Errorf("更新 %s 失败（旧版 v%s 已卸载，可重试或重新安装）: %v", meta.Pkg, oldVer, err)
	}
	return fmt.Sprintf("%s（%s → %s）", msg, oldVer, info.Version), nil
}
