// npm_plugin.go — npm/cordis 插件市场：从 npm registry 搜索 cordis 插件，
// 下载 tarball 取 main 源码 → 追加到 .pair/cordis.patch.json（跨重启存续）→
// 立即经 goja 宿主装载（esbuild 内联打包 + 非相对导入 mock）。
//
// 参考项目（deepseek-harness）的「插件市场」= npm 上的 cordis 插件生态
// （参考项目的插件市场指令转发 pnpm）；我方用 goja 沙箱 +
// 内置 esbuild 编译器直接执行 cordis 插件源码（ESM export default），
// 无需 node_modules——非相对导入 mock 空模块，相对导入内联打包。

package agent

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// npmRegistryBase npm registry 地址（测试可替换为 httptest server；
// 运行时可用环境变量 PAIRCODE_NPM_REGISTRY 覆盖——本地市场/私有 registry）。
var npmRegistryBase = func() string {
	if v := os.Getenv("PAIRCODE_NPM_REGISTRY"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://registry.npmjs.org"
}()

// npmFetchTimeout npm 拉取超时（下载 tarball 可能较慢）。
var npmFetchTimeout = 120 * time.Second

// npmHTTPClient 共享客户端：放宽 TLS 握手超时（慢网络下 Go 默认 10s 不够）。
var npmHTTPClient = &http.Client{
	Timeout: npmFetchTimeout,
	Transport: &http.Transport{
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          4,
	},
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
	url := npmRegistryBase + "/" + strings.ReplaceAll(pkg, "/", "%2F") + "/latest"
	resp, err := npmHTTPGet(url)
	if err != nil {
		return nil, fmt.Errorf("查询 npm %s 失败: %v", pkg, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm %s 不存在或不可达（HTTP %d）", pkg, resp.StatusCode)
	}
	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("解析 npm 响应失败: %v", err)
	}
	if info.Dist.Tarball == "" {
		return nil, fmt.Errorf("npm %s 无 tarball 地址", pkg)
	}
	return &info, nil
}

// fetchNPMPackage 下载 npm 包 tarball 并解压到临时目录，返回：
//   - dir：解压出的 package/ 目录（含源码，供 esbuild ResolveDir）
//   - manifest：package.json 内容
//   - err
//
// 调用方负责 os.RemoveAll(dir)。
func fetchNPMPackage(info *npmPackageInfo) (dir string, manifest map[string]any, err error) {
	resp, err := npmHTTPGet(info.Dist.Tarball)
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
	mainPath := filepath.Join(dir, mainFile)
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
	purpose := info.Description
	if purpose == "" {
		purpose = "npm cordis 插件 " + pkg
	}

	// ★ 2026-08-20 落盘统一：npm 插件固化为磁盘插件包
	//   <InstallDir>/.pair/plugins/<name>/（与 cordis_define 固化路径一致，
	//   重启 LoadGlobalPlugins 自动装配）；不再写 .pair/cordis.patch.json。
	pluginName := npmPluginDiskName(pkg)

	// 先装载（成功才固化；防重启后依旧装载失败的状态不一致）
	ph := GetGlobalPluginHost()
	var defID string
	if ph != nil {
		defID, err = ph.DefineJSCodeDir(code, "js", purpose, dir)
		if err != nil {
			return "", fmt.Errorf("插件编译失败（可能依赖 node 模块）: %v", err)
		}
		def, _ := ph.GetJSDef(defID)
		if def != nil {
			// 打卸载锚点（新磁盘插件形态无 config.npm；removeNPMPluginDefs 靠它匹配）
			ph.SetJSDefConfig(defID, "npm", pkg)
			if err := ph.LoadJSDynamic(def); err != nil {
				return "", fmt.Errorf("插件装载失败（可能依赖 node 模块）: %v", err)
			}
		}
	}

	// 固化为磁盘插件包（package.json + index.js，重启自动装配）
	// ★ config.npm 记录 npm 来源与版本（更新机制元数据：checkUpdates 扫描它）
	if err := syncGlobalPlugin(ToolsetPlugin{
		Name: pluginName, Purpose: purpose, Code: code, Scope: "project",
		Config: map[string]any{"npm": map[string]any{"pkg": pkg, "version": info.Version}},
	}); err != nil {
		if ph != nil && defID != "" {
			_ = ph.Unload(defID)
			_ = ph.RemoveJSDef(defID)
		}
		return "", fmt.Errorf("固化插件包 %s 失败: %v", pluginName, err)
	}

	msg := fmt.Sprintf("✅ 已安装 npm 插件「%s」v%s（插件目录 .pair/plugins/%s/，重启自动装配）", pkg, info.Version, pluginName)
	if ph == nil {
		msg += "。已保存。"
	} else {
		msg += "。已立即装载可用。"
	}
	return msg, nil
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

// primaryWorkspaceRoot 取主工作区根（WorkspaceRoots[0] 或全局根）。
func primaryWorkspaceRoot() string {
	if len(WorkspaceRoots) > 0 {
		return WorkspaceRoots[0]
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
	return name
}

// ─── npm 插件更新机制（2026-08-20：基于 npm registry 版本对比）────────
// 元数据载体：磁盘插件包 package.json 的 config.npm = { pkg, version }
// （npmMarketInstall 固化时写入；装配时透传 apply(ctx, config) 无副作用）。

// npmPluginMeta 单个磁盘插件的 npm 来源元数据。
type npmPluginMeta struct {
	Name    string `json:"name"`    // 磁盘插件名（.pair/plugins/<name>/）
	Pkg     string `json:"pkg"`     // npm 包名（@scope/pkg 或裸名）
	Version string `json:"version"` // 已安装的 npm 版本
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
	meta := npmPluginMeta{Name: pkg.Name}
	if cfg, ok := pkg.Config["npm"].(map[string]any); ok {
		meta.Pkg, _ = cfg["pkg"].(string)
		meta.Version, _ = cfg["version"].(string)
	}
	return meta, nil
}

// npmPluginMetaByPkg 按 npm 包名（或磁盘插件名）查已装元数据。
// 返回零值 meta 表示未安装/无 npm 来源。
func npmPluginMetaByPkg(pkg string) npmPluginMeta {
	dir := filepath.Join(globalPluginsDir(), npmPluginDiskName(pkg))
	if meta, err := npmPluginMetaPath(filepath.Join(dir)); err == nil && meta.Pkg != "" {
		return meta
	}
	// 入参可能是磁盘插件名而非 npm 全名：扫描全部目录匹配 config.npm.pkg
	entries, err := os.ReadDir(globalPluginsDir())
	if err != nil {
		return npmPluginMeta{}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := npmPluginMetaPath(filepath.Join(globalPluginsDir(), e.Name()))
		if err != nil || meta.Pkg == "" {
			continue
		}
		if meta.Pkg == pkg || meta.Name == pkg {
			return meta
		}
	}
	return npmPluginMeta{}
}

// npmPluginCheckUpdates 扫描全部磁盘插件的 config.npm 元数据，
// 逐个查询 npm registry latest 对比，返回可更新清单。
// 单个查询失败不阻塞整体（error 字段记录，方便前端提示网络问题）。
// ★ 返回 []map[string]any（小写 key）：goja 转 JS 对象时用字段名而非 json tag，
//
//	结构体字段名大写会导致前端/agent 取不到（2026-08-20 实测修正）。
func npmPluginCheckUpdates() []map[string]any {
	entries, err := os.ReadDir(globalPluginsDir())
	if err != nil {
		return nil
	}
	updates := make([]map[string]any, 0)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := npmPluginMetaPath(filepath.Join(globalPluginsDir(), e.Name()))
		if err != nil || meta.Pkg == "" {
			continue // 非 npm 来源（内置/工具集插件）跳过
		}
		info := map[string]any{
			"name": meta.Name, "pkg": meta.Pkg, "current": meta.Version,
			"latest": "", "updateable": false, "error": "",
		}
		if meta.Version == "" {
			info["error"] = "本地无版本记录（旧安装，重新安装一次即可支持更新）"
			updates = append(updates, info)
			continue
		}
		remote, err := fetchNPMInfo(meta.Pkg)
		if err != nil {
			info["error"] = "查询 registry 失败: " + err.Error()
			updates = append(updates, info)
			continue
		}
		info["latest"] = remote.Version
		info["updateable"] = remote.Version != meta.Version
		updates = append(updates, info)
	}
	return updates
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
