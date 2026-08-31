// ═══════════════════════════════════════════════════════════
// node_plugins.go — npm 插件 Node 桥安装路径
//
// 插件 package.json 声明了运行时 npm 依赖（dependencies 非空）时，
// goja 沙箱无法运行（mock 空模块）——改走 Node 桥：
//  1. tarball 源码落盘 .pair/cordis/node/plugins/<pkg>@<ver>/（可查看）
//  2. 桥目录 npm install <pkg>@<ver>（真实 node_modules + @cordisjs/core）
//  3. plugins.json 记录要装载的插件（bridge.js 启动时加载）
//  4. 重启 Node 桥（或首次启动）→ 插件 apply 执行 → ctx.tools.register
//     的工具进 Go Registry（agent 可直接调用）
//
// patch（.pair/cordis.patch.json）仍记录条目（config.runtime="node" +
// config.npm），保证卸载/已安装判断/重启恢复统一走 patch 链路。
// ═══════════════════════════════════════════════════════════
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// nodePluginsFile 桥目录下的插件装载清单。
// ★ Round4：条目支持对象形态 {spec, runtime}（runtime: node|dsh）——
//   dsh = cordis4 + 外部服务面（@deepseek-ai/cordis）；node = cordis3 既有。
//   旧字符串形态（"pkg@ver"）读取时兼容（默认 runtime=node）。
type nodePluginsFile struct {
	Plugins []nodePluginEntry `json:"plugins"`
}

// nodePluginEntry 一条插件装载记录。
type nodePluginEntry struct {
	Spec    string `json:"spec"`
	Runtime string `json:"runtime,omitempty"` // node（cordis3 默认）| dsh（cordis4）
}

// UnmarshalJSON 兼容旧字符串条目（"pkg@ver"）与新对象条目。
func (f *nodePluginsFile) UnmarshalJSON(b []byte) error {
	var raw struct {
		Plugins []json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	f.Plugins = nil
	for _, item := range raw.Plugins {
		var s string
		if err := json.Unmarshal(item, &s); err == nil {
			f.Plugins = append(f.Plugins, nodePluginEntry{Spec: s, Runtime: "node"})
			continue
		}
		var e nodePluginEntry
		if err := json.Unmarshal(item, &e); err != nil || e.Spec == "" {
			continue
		}
		if e.Runtime == "" {
			e.Runtime = "node"
		}
		f.Plugins = append(f.Plugins, e)
	}
	return nil
}

// Specs 返回全部装载 spec（字符串形态，兼容旧消费者）。
func (f *nodePluginsFile) Specs() []string {
	out := make([]string, 0, len(f.Plugins))
	for _, e := range f.Plugins {
		out = append(out, e.Spec)
	}
	return out
}

// readNodePluginsFile 读 plugins.json（不存在 → 空清单）。
func readNodePluginsFile(path string) (*nodePluginsFile, error) {
	doc := &nodePluginsFile{Plugins: []nodePluginEntry{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(b, doc); err != nil {
		return nil, fmt.Errorf("plugins.json 解析失败: %v", err)
	}
	if doc.Plugins == nil {
		doc.Plugins = []nodePluginEntry{}
	}
	return doc, nil
}

// writeNodePluginsFile 写 plugins.json（对象条目形态）。
func writeNodePluginsFile(path string, doc *nodePluginsFile) error {
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// nodePluginNeedsNode 判断插件是否需 Node 桥（goja 沙箱跑不了）：
//   - dependencies 非空（真实 npm 依赖，mock 空模块运行期 undefined）
//   - peerDependencies 里 cordis/@cordisjs/core 主版本为 4
//     （goja 内置 CordisApi 对齐 cordis3 API，无 cordis4 的 inject 等）
//   - ★ Round4：peerDependencies 里 @deepseek-ai/cordis 主版本为 4
//     （外部插件形态——npm bundle + cordis4 Context，goja 轨无法运行）
//
// 仅 peer cordis3 / 零依赖 → goja 沙箱（快、无需 node）。
func nodePluginNeedsNode(manifest map[string]any) bool {
	return nodePluginRuntime(manifest) != ""
}

// nodePluginRuntime 判定插件运行时轨：
//
//	"dsh"  → Node 桥 cordis4 分支（@deepseek-ai/cordis ^4 peer；外部服务面）
//	"node" → Node 桥 cordis3 分支（dependencies 非空 / cordis ^4 peer）
//	""     → goja 沙箱（无 npm 依赖、无 cordis4）
func nodePluginRuntime(manifest map[string]any) string {
	peers, _ := manifest["peerDependencies"].(map[string]any)
	if v, ok := peers["@deepseek-ai/cordis"].(string); ok && strings.HasPrefix(strings.TrimSpace(v), "^4") {
		return "dsh"
	}
	if deps, _ := manifest["dependencies"].(map[string]any); len(deps) > 0 {
		return "node"
	}
	for _, k := range []string{"cordis", "@cordisjs/core"} {
		if v, ok := peers[k].(string); ok && strings.HasPrefix(strings.TrimSpace(v), "^4") {
			return "node"
		}
	}
	return ""
}

// ── 原生依赖（C/C++ 编译）提示 ─────────────────────────────
// 部分 npm 包依赖原生模块（node-gyp 编译）。多数自带 prebuilt 二进制
// （npm 直接可装）；无 prebuilt 时需本机 C++ 编译环境。
// 检测命中时安装不阻塞，仅在结果里给出「需装什么 / 开源替代」提示。
var nativeDepHints = map[string]string{
	"better-sqlite3": "better-sqlite3 自带 prebuilt 二进制（npm 直接安装）；源码编译需 VS Build Tools(C++) + Python 3，或改用纯 JS 替代 sql.js / @sqlite.org/sqlite-wasm",
	"sharp":          "sharp 自带 prebuilt（libvips），npm 可直接安装；如需纯 JS 替代可用 @resvg/resvg-js / jimp",
	"canvas":         "node-canvas 需预装 GTK/Cairo（Windows: GTK runtime）或 VS Build Tools；纯 JS 替代：@napi-rs/canvas（prebuilt）",
	"sqlite3":        "sqlite3 自带 prebuilt；源码编译需 VS Build Tools + Python；替代 better-sqlite3（prebuilt）",
	"bcrypt":         "bcrypt 自带 prebuilt；替代 bcryptjs（纯 JS）",
	"node-sass":      "node-sass 已废弃，官方建议 dart-sass（纯 JS sass 包）；源码编译需 VS Build Tools + Python",
	"fsevents":       "fsevents 仅 macOS 需要，Windows/Linux 可安全忽略（可选依赖）",
	"robotjs":        "robotjs 需 VS Build Tools + Python 编译（无 prebuilt）；替代 nut-js（纯 JS，Windows）",
	"serialport":     "serialport 自带 prebuilt；源码编译需 VS Build Tools + Python",
	"usb":            "usb 自带 prebuilt；源码编译需 VS Build Tools + Python",
	"leveldown":      "leveldown 自带 prebuilt；替代 classic-level（prebuilt）",
	"puppeteer":      "puppeteer 依赖 Chromium（安装时下载）；需联网；替代 playwright（自带浏览器）",
	"playwright":     "playwright 需下载浏览器（npx playwright install）；纯 JS 替代：无（需真实浏览器）",
	"keyboard":       "keyboard 需 VS Build Tools 编译（Windows hook）；替代 robotjs",
	"nut-js":         "nut-js 纯 JS（Windows）无需编译，可直接安装",
	"opencv4nodejs":  "opencv4nodejs 需本机 OpenCV + VS Build Tools；替代 @u4/opencv4nodejs（prebuilt）或纯 JS jimp",
	"sharp-libvips":  "sharp 依赖（通常 prebuilt，无需手动安装）",
	"ref":            "ref 需编译；替代 ref-napi（prebuilt）",
	"ref-napi":       "ref-napi 自带 prebuilt；源码编译需 VS Build Tools + Python",
	"ioredis":        "ioredis 纯 JS 无需编译",
	"iconv-lite":     "iconv-lite 纯 JS 无需编译",
}

// nativeDepHint 检查插件 manifest 依赖中是否含原生模块，返回提示（空=无）。
func nativeDepHint(manifest map[string]any) string {
	hit := []string{}
	deps, _ := manifest["dependencies"].(map[string]any)
	for name := range deps {
		if h, ok := nativeDepHints[name]; ok {
			hit = append(hit, "· "+name+"："+h)
		}
	}
	if len(hit) == 0 {
		return ""
	}
	return "\n📦 原生模块提示：" + strings.Join(hit, "\n")
}

// marketInstallNPMPluginNode Node 桥安装路径：
// 源码落盘 + npm install + plugins.json 记录 + 重启桥装载。
// ★ Round4：runtime 判定（node=cordis3 既有 / dsh=cordis4+外部服务面）；
//   dsh 插件 peer 全部 optional（npm 不自动装）→ 显式安装 @deepseek-ai/* peer。
func marketInstallNPMPluginNode(info *npmPackageInfo, srcDir string, auto bool) (string, error) {
	pkg := info.Name
	ver := info.Version
	spec := pkg + "@" + ver
	projectRoot := npmPluginProjectRoot()
	if projectRoot == "" {
		return "", fmt.Errorf("无工作区根，无法安装插件")
	}
	runtime := nodePluginRuntime(info.Manifest)
	if runtime == "" {
		runtime = "node"
	}
	bridgeDir := nodeBridgeDir()
	pluginsDir := filepath.Join(bridgeDir, "plugins")

	// 1. 源码落盘（保留 tarball 解压内容供查看；含 package.json/main 源码）
	target := filepath.Join(pluginsDir, strings.ReplaceAll(spec, "/", "_"))
	if err := copyDir(srcDir, target); err != nil {
		return "", fmt.Errorf("插件源码落盘失败: %v", err)
	}

	// 2. npm install（真实 node_modules；@cordisjs/core 作为桥运行时）
	if err := npmInstallPlugin(bridgeDir, spec); err != nil {
		return "", fmt.Errorf("npm install 失败（%v）——插件源码已保存在 %s，可手动 npm install 后重试", err, target)
	}
	// ★ Round4：dsh 插件 peer 全 optional → npm 不会自动装；显式安装
	//   @deepseek-ai/*（cordis + dsh-* 非 client 包）供 cordis4 装载分支 import。
	if runtime == "dsh" {
		if err := npmInstallDshPeers(bridgeDir, info.Manifest); err != nil {
			return "", fmt.Errorf("DSH peer 安装失败: %v（插件已装；可手动 npm install 后重启桥）", err)
		}
	}

	// 3. plugins.json 记录（幂等；条目带 runtime）
	pluginsFile := filepath.Join(bridgeDir, "plugins.json")
	doc, err := readNodePluginsFile(pluginsFile)
	if err != nil {
		return "", err
	}
	found := false
	for i, e := range doc.Plugins {
		if e.Spec == spec {
			doc.Plugins[i].Runtime = runtime // 更新 runtime（幂等重装）
			found = true
			break
		}
	}
	if !found {
		doc.Plugins = append(doc.Plugins, nodePluginEntry{Spec: spec, Runtime: runtime})
		if err := writeNodePluginsFile(pluginsFile, doc); err != nil {
			return "", err
		}
	} else {
		if err := writeNodePluginsFile(pluginsFile, doc); err != nil {
			return "", err
		}
	}

	// 4. patch 记录（统一卸载/判断链路；runtime 透传）
	patchPath := filepath.Join(projectRoot, ".pair", "cordis.patch.json")
	entry := cordisPatchPlugin{
		Code:     "",
		Language: "js",
		Purpose:  info.Description,
		Config: map[string]any{
			"npm":     spec,
			"runtime": runtime,
			"dir":     target,
		},
	}
	if err := upsertPatchPlugin(patchPath, entry); err != nil {
		return "", err
	}

	// 5. 重启桥装载（首次安装则启动）
	// ★ 2026-08-17 修复：安装后必须重启桥才能装载新插件。
	//   旧逻辑 nodeBridgeHasPlugin(spec) 检查 plugins.json——但第 3 步刚把 spec
	//   写入 plugins.json → 恒 true → 永不重启桥 → 新插件「安装了但没生效」。
	//   桥在跑则先关再启，新桥从 plugins.json 装载全部插件（含新装）。
	ph := GetGlobalPluginHost()
	if ph != nil {
		if globalNodeBridge != nil {
			globalNodeBridge.Close()
			globalNodeBridge = nil
		}
		if _, err := ensureNodeBridge(ph, bridgeDir); err != nil {
			return "", fmt.Errorf("Node 桥启动失败: %v（插件已安装，重启应用后自动装载）", err)
		}
	}
	// 原生依赖提示（不阻塞安装）
	hint := nativeDepHint(info.Manifest)
	runtimeLabel := "Node 运行时桥（真实 node 环境执行 npm 依赖）"
	if runtime == "dsh" {
		runtimeLabel = "外部运行时（cordis4 + 外部服务面，@deepseek-ai/cordis）"
	}
	return fmt.Sprintf("✅ 已安装 npm 插件「%s」v%s（%s）%s", pkg, ver, runtimeLabel, hint), nil
}

// dshPeerSpecs 提取 外部插件的运行时 peer 依赖（@deepseek-ai/* 非 client 包）。
// ★ 版本选择：优先 devDependencies（插件构建/测试锁定的版本集——peer 范围如
//   ^0.1.0-rc.6 与传递 peer（dsh-user-approval rc.8 等）存在 rc 档位错配，
//   npm 会 ERESOLVE；devDeps 精确版本集可复现装载），缺失回退 peerDependencies 范围。
// client 半（@deepseek-ai/dsh-client-*）与 react 由宿主 UI 槽位承载，不装入桥。
func dshPeerSpecs(manifest map[string]any) []string {
	peers, _ := manifest["peerDependencies"].(map[string]any)
	devs, _ := manifest["devDependencies"].(map[string]any)
	seen := map[string]bool{}
	specs := make([]string, 0, len(peers))
	for name := range peers {
		if !strings.HasPrefix(name, "@deepseek-ai/") {
			continue
		}
		if strings.HasPrefix(name, "@deepseek-ai/dsh-client") {
			continue // 客户端半（Web 面板）不装载
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		ver := ""
		if v, ok := devs[name].(string); ok {
			ver = v // 构建验证版本优先（精确 pin，规避 rc 档位 ERESOLVE）
		} else if v, ok := peers[name].(string); ok {
			ver = v
		}
		if strings.TrimSpace(ver) == "" {
			ver = "latest"
		}
		specs = append(specs, name+"@"+strings.TrimSpace(ver))
	}
	return specs
}

// npmInstallDshPeers 在桥目录安装 DSH peer 依赖（@deepseek-ai/cordis + dsh-* 服务面包）。
func npmInstallDshPeers(bridgeDir string, manifest map[string]any) error {
	specs := dshPeerSpecs(manifest)
	if len(specs) == 0 {
		return nil
	}
	// 桥基础 package.json 已由 npmInstallPlugin 保证存在
	npmCmd := "npm"
	if _, err := exec.LookPath("npm"); err != nil {
		npmCmd = "npm.cmd"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	args := append([]string{"install", "--no-audit", "--no-fund", "--prefix", bridgeDir}, specs...)
	cmd := exec.CommandContext(ctx, npmCmd, args...)
	if runtime.GOOS == "windows" {
		executil.HideWindow(cmd)
	}
	cmd.Env = append(os.Environ(), "npm_config_yes=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		last := strings.TrimSpace(string(out))
		if len(last) > 400 {
			last = last[len(last)-400:]
		}
		return fmt.Errorf("npm install DSH peers %v 失败: %v（%s）", specs, err, last)
	}
	return nil
}

// npmInstallPlugin 在桥目录执行 npm install（Windows 用 npm.cmd）。
func npmInstallPlugin(bridgeDir, spec string) error {
	if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		return err
	}
	// 基础 package.json（npm install 需要；private 防误发布）
	pkgJSON := filepath.Join(bridgeDir, "package.json")
	if _, err := os.Stat(pkgJSON); os.IsNotExist(err) {
		base := map[string]any{
			"name":    "cordis-bridge",
			"version": "0.1.0",
			"private": true,
			"dependencies": map[string]any{
				"@cordisjs/core": "latest",
			},
		}
		b, _ := json.MarshalIndent(base, "", "  ")
		if err := os.WriteFile(pkgJSON, b, 0o644); err != nil {
			return err
		}
	}
	npmCmd := "npm"
	if _, err := exec.LookPath("npm"); err != nil {
		npmCmd = "npm.cmd" // Windows 通常需要 .cmd 包装器
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, npmCmd, "install", "--no-audit", "--no-fund", "--prefix", bridgeDir, spec)
	// 隐藏子进程控制台窗口（无控制台父进程时 console 程序会自己弹窗）
	if runtime.GOOS == "windows" {
		executil.HideWindow(cmd)
	}
	cmd.Env = append(os.Environ(), "npm_config_yes=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		last := strings.TrimSpace(string(out))
		if len(last) > 400 {
			last = last[len(last)-400:]
		}
		return fmt.Errorf("npm install %s 失败: %v（%s）", spec, err, last)
	}
	return nil
}

// uninstallNodePlugin 卸载 Node 桥插件：plugins.json 移除 + patch 移除 + 重启桥。
func uninstallNodePlugin(pkg string) error {
	bridgeDir := nodeBridgeDir()
	pluginsFile := filepath.Join(bridgeDir, "plugins.json")
	doc, err := readNodePluginsFile(pluginsFile)
	if err != nil {
		return err
	}
	out := doc.Plugins[:0]
	removed := false
	for _, e := range doc.Plugins {
		if specMatchesPkg(e.Spec, pkg) {
			removed = true
			continue
		}
		out = append(out, e)
	}
	if removed {
		doc.Plugins = out
		if err := writeNodePluginsFile(pluginsFile, doc); err != nil {
			return err
		}
	}
	// 重启桥（移除插件）
	if globalNodeBridge != nil {
		globalNodeBridge.Close()
		globalNodeBridge = nil
		if ph := GetGlobalPluginHost(); ph != nil {
			if _, err := ensureNodeBridge(ph, bridgeDir); err != nil {
				log.Printf("[node-bridge] 卸载后桥重启失败: %v", err)
			}
		}
	}
	return nil
}

func specMatchesPkg(spec, pkg string) bool {
	at := strings.LastIndex(spec, "@")
	name := spec
	if at > 0 {
		name = spec[:at]
	}
	return name == pkg
}

// upsertPatchPlugin 幂等追加/更新 patch 条目（key 为 config.npm）。
func upsertPatchPlugin(path string, entry cordisPatchPlugin) error {
	doc, err := readCordisPatch(path)
	if err != nil {
		return err
	}
	key, _ := entry.Config["npm"].(string)
	for i := range doc.Plugins {
		src, _ := doc.Plugins[i].Config["npm"].(string)
		if src == key {
			doc.Plugins[i] = entry
			return writeCordisPatch(path, doc)
		}
	}
	doc.Plugins = append(doc.Plugins, entry)
	return writeCordisPatch(path, doc)
}

// copyDir 递归复制目录（源码落盘用）。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// ★ 2026-08-18：跳过符号链接/junction（Windows npm 本地链接如
		//   local-test-plugin → _tmp_bridge_plugin；复制链接目标会失败）。
		//   链接目标在复制场景无意义（node_modules 复制按真实文件拷贝）。
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
