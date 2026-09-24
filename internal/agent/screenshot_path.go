// screenshot_path.go — 截图/落盘类工具的产物目录解析（跨平台；无 build tag）。
//
// ★ 2026-09-17 多工作区修复（用户实测反馈「视觉验证截图不落盘」）：
//   原实现直接 filepath.Join(root, "screenshots")，而 root 有两类退化情形：
//     ① 多工作区/新会话切换后仍沿用旧 root（内嵌内核「首次 root 永久缓存」，
//        已在 embedded_tools.go 修复为按 root 键控重建）；
//     ② root 为空字符串（会话根未解析 / 未打开工作区）→ filepath.Join("", "screenshots")
//        得到**相对路径**，产物落到进程 cwd（仓库里曾出现 internal/agent/screenshots、
//        cmd/screenshots 这类"孤儿目录"，正是此路径退化的痕迹）。
//
//   本文件统一兜底顺序：调用方 root → core.Root()（工作区主根，实时）→
//   core.InstallDir()（未打开工作区时的安装根），并返回**绝对路径**，
//   使工具报告里的路径可直接定位、不再出现"找不到文件"。
package agent

import (
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/internal/core"
)

// artifactDir 解析产物目录 <base>/<sub>：base 逐级兜底（root → 工作区主根 → 安装根），
// 并绝对化（相对 cwd 的路径会让产物位置不可预期）。
func artifactDir(root, sub string) string {
	base := strings.TrimSpace(root)
	if base == "" {
		base = core.Root() // 当前工作区主根（实时：多工作区切换后随之变化）
	}
	if base == "" {
		base = core.InstallDir() // 未打开任何工作区 → 安装根（始终可写、可预期）
	}
	dir := filepath.Join(base, sub)
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// screenshotOutputDir 截图产物目录（screenshots/）。screenshot_* 与 web_debug 共用，
// 保证「同一次会话调用的截图落在同一处」。
func screenshotOutputDir(root string) string {
	return artifactDir(root, "screenshots")
}
