package agent

import (
	"os"
	"path/filepath"
)

// ═══════════════════════════════════════════════════════════════
// runtime_assets.go — 运行时资源外置（主程序只保留框架）
//
// 运行时脚本/模板（cordis.bundle.js、bridge_node.js、ide_ref*.html 等）
// 外置到 <exe 目录>/.pair/assets/runtime/，可独立更新替换、不重新编译；
// 缺失时回退内嵌（//go:embed fallback，保证单文件分发仍可运行）。
// 加载顺序：外部文件优先 → embed 兜底。
// ═══════════════════════════════════════════════════════════════

// RuntimeAssetDir 返回运行时资源目录（<exe 目录>/.pair/assets/runtime/）。
func RuntimeAssetDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), ".pair", "assets", "runtime")
}

// LoadRuntimeAsset 优先读外部资源文件；存在返回 (data, true)，
// 否则返回内嵌 fallback 与 (fallback, false)。
func LoadRuntimeAsset(name string, fallback []byte) (data []byte, external bool) {
	if dir := RuntimeAssetDir(); dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return b, true
		}
	}
	return fallback, false
}

// LoadRuntimeAssetString 字符串版（JS 运行时脚本场景）。
func LoadRuntimeAssetString(name string, fallback string) (data string, external bool) {
	if dir := RuntimeAssetDir(); dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return string(b), true
		}
	}
	return fallback, false
}
