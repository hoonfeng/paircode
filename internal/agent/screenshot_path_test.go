// screenshot_path_test.go — 截图产物目录解析（多工作区/未打开工作区兜底）验证。

package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// TestScreenshotOutputDirFallback 锁定 artifactDir/screenshotOutputDir 的退化兜底不变量：
//
//	① root 非空        → <root>/screenshots（原语义不变）
//	② root 空 + 有工作区 → <core.Root()>/screenshots（跟随当前工作区）
//	③ root 空 + 无工作区 → <core.InstallDir()>/screenshots（绝对路径）
//
// ★ 背景（2026-09-17 用户实测）：原先 root 为空时 filepath.Join 会产出**相对路径**
// screenshots/（相对进程 cwd），产物落到不可预期位置 → 用户视角「截图不落盘」。
// 仓库里的 internal/agent/screenshots、cmd/screenshots 孤儿目录即此路径退化的痕迹。
func TestScreenshotOutputDirFallback(t *testing.T) {
	// ① root 非空：直用（语义不回归）
	dir := t.TempDir()
	if got, want := screenshotOutputDir(dir), filepath.Join(dir, "screenshots"); got != want {
		t.Errorf("root 非空应直用：got=%q want=%q", got, want)
	}

	saved := core.Folders
	defer func() { core.Folders = saved }()

	// ② root 空 + 有工作区：回落工作区主根
	ws := t.TempDir()
	core.Folders = []string{ws}
	if got, want := screenshotOutputDir(""), filepath.Join(ws, "screenshots"); got != want {
		t.Errorf("root 空应回落工作区主根：got=%q want=%q", got, want)
	}

	// ③ root 空 + 无工作区：回落安装根，且必须是绝对路径
	core.Folders = nil
	got := screenshotOutputDir("")
	if !filepath.IsAbs(got) {
		t.Errorf("无工作区时必须为绝对路径（不得退化为 cwd 相对路径）：got=%q", got)
	}
	if !strings.HasSuffix(got, string(filepath.Separator)+"screenshots") {
		t.Errorf("应以 screenshots 结尾：got=%q", got)
	}
	if got == filepath.Join("screenshots") {
		t.Errorf("不得退化为相对路径 screenshots：got=%q", got)
	}
}
