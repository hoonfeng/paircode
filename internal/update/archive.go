// archive.go — 更新包安全解压（zip-slip / zip bomb 防护）与包结构冒烟检查。
package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxExtractBytes = 500 << 20 // 解压总量上限
	maxEntryBytes   = 300 << 20 // 单文件上限
	maxEntries      = 50000     // 条目数上限
)

// Extract 把更新包解压到 dst（先清空 dst），返回解压出的相对路径清单（正斜杠）。
//
// 安全措施：拒绝绝对路径 / `..` 越界 / 含盘符的条目；跳过符号链接等非普通文件；
// 限制条目数、单文件大小与总量（防 zip bomb）。即可执行位按条目名与 zip 权限推断。
func Extract(zipPath, dst string) ([]string, error) {
	if err := os.RemoveAll(dst); err != nil {
		return nil, fmt.Errorf("清理暂存目录失败: %w", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, fmt.Errorf("创建暂存目录失败: %w", err)
	}
	root, err := filepath.Abs(dst)
	if err != nil {
		return nil, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("打开更新包失败: %w", err)
	}
	defer zr.Close()

	files := make([]string, 0, len(zr.File))
	var total int64
	for _, f := range zr.File {
		if len(files) >= maxEntries {
			return nil, fmt.Errorf("更新包条目数超过上限 %d", maxEntries)
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		name = strings.TrimPrefix(name, "./")
		if name == "" || strings.HasSuffix(name, "/") || f.FileInfo().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() { // 跳过符号链接 / 设备 / FIFO
			continue
		}
		clean := path.Clean(name)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") ||
			path.IsAbs(clean) || strings.Contains(clean, ":") {
			return nil, fmt.Errorf("更新包含非法路径条目: %s", f.Name)
		}
		if int64(f.UncompressedSize64) > maxEntryBytes {
			return nil, fmt.Errorf("更新包内单文件超限（%s: %d 字节）", clean, f.UncompressedSize64)
		}
		total += int64(f.UncompressedSize64)
		if total > maxExtractBytes {
			return nil, fmt.Errorf("更新包解压总量超过上限 %d 字节", int64(maxExtractBytes))
		}

		target := filepath.Join(root, filepath.FromSlash(clean))
		if !strings.HasPrefix(target, root+string(os.PathSeparator)) {
			return nil, fmt.Errorf("更新包条目越界: %s", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("创建目录失败（%s）: %w", filepath.Dir(target), err)
		}
		if err := extractOne(f, target, clean); err != nil {
			return nil, err
		}
		files = append(files, clean)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("更新包为空")
	}
	return files, nil
}

// extractOne 写出单个条目（实际字节数再校验一次上限——UncompressedSize64 可被伪造）。
func extractOne(f *zip.File, target, clean string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("读取条目失败（%s）: %w", clean, err)
	}
	defer rc.Close()
	mode := os.FileMode(0o644)
	if isExecutableEntry(clean, f.Mode()) {
		mode = 0o755
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("写入文件失败（%s）: %w", clean, err)
	}
	n, cerr := io.Copy(out, io.LimitReader(rc, maxEntryBytes+1))
	if cerr != nil {
		out.Close()
		_ = os.Remove(target)
		return fmt.Errorf("解压失败（%s）: %w", clean, cerr)
	}
	if n > maxEntryBytes {
		out.Close()
		_ = os.Remove(target)
		return fmt.Errorf("条目实际大小超限（%s）", clean)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("关闭文件失败（%s）: %w", clean, err)
	}
	return nil
}

// isExecutableEntry 推断条目是否需要可执行位：zip 权限位 / .exe/.sh/.bat / 无扩展名的 pair*。
func isExecutableEntry(clean string, m os.FileMode) bool {
	if m&0o111 != 0 {
		return true
	}
	base := strings.ToLower(path.Base(clean))
	switch {
	case strings.HasSuffix(base, ".exe"),
		strings.HasSuffix(base, ".sh"),
		strings.HasSuffix(base, ".bat"),
		strings.HasSuffix(base, ".cmd"):
		return true
	case strings.HasPrefix(base, "pair") && !strings.Contains(base[4:], "."):
		return true
	}
	return false
}

// SmokeCheck 校验解压结果像一个合法更新包：返回包内主程序名（找不到即报错）。
// 结构性标志：主程序 + （README.md 或 .pair/ 或 plugins-src/ 之一）。
func SmokeCheck(staging string, files []string, goos, goarch string) (string, error) {
	main := MatchMainProgram(files, goos, goarch)
	if main == "" {
		return "", fmt.Errorf("更新包内未找到本平台主程序（期望 %s）", MainProgramName(goos, goarch))
	}
	hasMark := false
	for _, f := range files {
		if strings.EqualFold(f, "README.md") || strings.HasPrefix(f, ".pair/") ||
			strings.HasPrefix(f, "plugins-src/") || strings.EqualFold(f, "pair.exe") {
			hasMark = true
			break
		}
	}
	if !hasMark {
		return "", fmt.Errorf("更新包结构异常：缺少 README.md / .pair / plugins-src 等标识")
	}
	for _, f := range files {
		if strings.EqualFold(f, main) {
			return main, nil
		}
	}
	// 主程序可能位于子目录：返回实际相对路径
	for _, f := range files {
		if strings.EqualFold(path.Base(f), main) {
			return f, nil
		}
	}
	return main, nil
}
