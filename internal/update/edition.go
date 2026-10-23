// edition.go — 发行版（web / gui）识别与资产匹配。
//
// 背景（2026-10-19）：发布流程改为默认只出 web 版包（PairCode-web-<ver>-<os>.zip），
// 桌面版（PairCode-gui-<ver>-<os>.zip）按需构建。旧的宽容匹配只看「版本号 + 平台词」：
//   - 同版本 web/gui 两包共存时，先命中谁取决于 GitHub assets 返回顺序；
//   - 只剩 web 包时，GUI 安装会把 web 包装进桌面版安装目录（主程序名不同），
//     主程序版本实际没变，等于「更新假成功」。
//
// 因此按安装目录主程序识别 edition，并只在该 edition 的资产里挑包；
// 找不到就明确报错，绝不跨发行版兜底。
package update

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 发行版标识（与 packager.json dist.editions[].name 对齐）。
const (
	EditionWeb = "web"
	EditionGUI = "gui"
)

// DetectEdition 按安装目录主程序识别当前发行版：
//   - PairCode[.exe] / pair.exe   → gui（一体化桌面版）
//   - pair-server[.exe]           → web
//   - 都不存在                    → ""（不限制，保持旧行为）
//
// gui 优先：旧布局的安装目录里可能同时留有 pair-server，用户实际双击的是
// PairCode/pair.exe，它才是当前运行的发行版。
func DetectEdition(installDir string) string {
	if strings.TrimSpace(installDir) == "" {
		return ""
	}
	exists := func(name string) bool {
		st, err := os.Stat(filepath.Join(installDir, name))
		return err == nil && !st.IsDir()
	}
	gui := "PairCode"
	server := "pair-server"
	if runtime.GOOS == "windows" {
		gui += ".exe"
		server += ".exe"
	}
	if exists(gui) || exists("pair.exe") {
		return EditionGUI
	}
	if exists(server) {
		return EditionWeb
	}
	return ""
}

// AssetNameForEdition 新版发布包命名：PairCode-<edition>-<ver>-<os>.zip
// （packager 的 dist.editions 对三平台都带平台词）。
func AssetNameForEdition(ver, goos, edition string) string {
	v := strings.TrimPrefix(strings.TrimSpace(ver), "v")
	s := strings.ToLower(strings.TrimSpace(goos))
	if s == "" {
		s = strings.ToLower(runtime.GOOS)
	}
	return "PairCode-" + strings.ToLower(strings.TrimSpace(edition)) + "-" + v + "-" + s + ".zip"
}

// hasEditionTag 资产名是否带发行版标记（web/gui 段）。拆分前的旧包名没有该段。
func hasEditionTag(name string) bool {
	ln := strings.ToLower(name)
	return strings.Contains(ln, "-web-") || strings.Contains(ln, "-gui-")
}

// matchEditionAsset 在限定发行版的资产里挑本平台该版本的 zip：
// 精确名（含 v 前缀写法）→ 名字含版本号 + 平台词 + edition 词。
// 返回空串表示该版本没有本发行版的本平台包（调用方不得跨发行版兜底）。
func matchEditionAsset(zips []string, ver, goos, edition string) string {
	ed := strings.ToLower(strings.TrimSpace(edition))
	if ed == "" {
		return ""
	}
	for _, want := range []string{
		AssetNameForEdition(ver, goos, ed),
		AssetNameForEdition("v"+strings.TrimPrefix(strings.TrimSpace(ver), "v"), goos, ed),
	} {
		for _, n := range zips {
			if strings.EqualFold(n, want) {
				return n
			}
		}
	}
	// 模糊：版本号 + 平台词 + 发行版词缺一不可
	v := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ver), "v"))
	s := strings.ToLower(strings.TrimSpace(goos))
	for _, n := range zips {
		ln := strings.ToLower(n)
		if !strings.Contains(ln, v) || !strings.Contains(ln, ed) {
			continue
		}
		if strings.Contains(ln, s) || (s == "darwin" && strings.Contains(ln, "macos")) {
			return n
		}
	}
	return ""
}
