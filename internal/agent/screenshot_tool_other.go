//go:build !windows

package agent

// screenshot_tool_other.go — Linux / macOS 截图工具实现。
//
// 历史：2026-08-22 为了让交叉编译通过，本文件曾只是「注册工具、运行时返回
// 不支持」的占位（Windows 版用 GDI，非 Windows 无实现）。2026-10 起补上真实
// 实现——截图是 Agent 视觉验证的关键能力，Linux/mac 缺失等于工具面残缺。
//
// 实现策略（尽量少依赖、行为与 Windows 版对齐）：
//   - macOS：系统自带 screencapture（-x 无声、-R 区域）。
//   - Linux：按顺序探测 grim(Wayland) / scrot / ImageMagick import /
//     gnome-screenshot / spectacle(KDE) / xfce4-screenshooter，用第一个可用的。
//   - 区域截图：先全屏取原始 PNG，再用 Go 标准库裁剪（不依赖各工具的参数差异）。
//   - 窗口截图：macOS 用 System Events 枚举窗口取 bounds；Linux 用 xdotool
//     （X11）。取不到时返回明确错误并提示改用 screenshot_area。

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// registerScreenshotTools 注册三个截图工具（跨平台实现）。
func registerScreenshotTools(r *Registry, root string) {
	// ── screenshot_desktop ──
	r.Register(&Tool{
		Name:       "screenshot_desktop",
		UsageGuide: "截取整个桌面（主显示器），保存为 PNG。用于查看当前桌面状态、验证 GUI 效果。自动保存到 screenshots/ 并返回路径。",
		Description: "截取整个桌面（主显示器），保存为 PNG 图片到 screenshots/ 目录。" +
			"返回文件路径、尺寸和截图时间。" +
			"之后可用多模态模型直接分析截图内容。",
		Parameters: objSchema(props{
			"name": strProp("可选：自定义文件名（不含扩展名），默认自动生成时间戳名称"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return captureDesktopNonWindows(root, argStr(args, "name"))
		},
	})

	// ── screenshot_window ──
	r.Register(&Tool{
		Name:       "screenshot_window",
		UsageGuide: "按窗口标题截取特定窗口，保存为 PNG。title 支持子串匹配（不区分大小写）。macOS 需授予辅助功能权限；Linux 需 X11 + xdotool。",
		Description: "按窗口标题或标题子串截取特定窗口，保存为 PNG 图片到 screenshots/ 目录。" +
			"返回文件路径、窗口尺寸和截图时间。取不到窗口时请改用 screenshot_area。",
		Parameters: objSchema(props{
			"title": strProp("窗口标题或标题子串（不区分大小写）。例如 \"终端\"、\"Chrome\"、\"Terminal\""),
			"name":  strProp("可选：自定义文件名（不含扩展名），默认自动生成"),
		}, "title"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			title := argStr(args, "title")
			if title == "" {
				return "", fmt.Errorf("参数 title 不能为空")
			}
			return captureWindowNonWindows(root, title, argStr(args, "name"))
		},
	})

	// ── screenshot_area ──
	r.Register(&Tool{
		Name:       "screenshot_area",
		UsageGuide: "按坐标截取指定屏幕区域。left/top/right/bottom 支持像素或百分比（如 10%）。用于截取界面局部细节。",
		Description: "按坐标截取指定区域，保存为 PNG 图片到 screenshots/ 目录。" +
			"区域坐标可以是绝对坐标（相对于主显示器左上角），也可以是百分比（如 \"10% 20% 50% 30%\"）。",
		Parameters: objSchema(props{
			"left":   strProp("左边界：像素值或百分比（如 \"10%\"）"),
			"top":    strProp("上边界：像素值或百分比"),
			"right":  strProp("右边界：像素值或百分比"),
			"bottom": strProp("下边界：像素值或百分比"),
			"name":   strProp("可选：自定义文件名"),
		}, "left", "top", "right", "bottom"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return captureAreaNonWindows(root,
				argStr(args, "left"), argStr(args, "top"),
				argStr(args, "right"), argStr(args, "bottom"),
				argStr(args, "name"))
		},
	})
}

// captureDesktopNonWindows 截取主显示器。
func captureDesktopNonWindows(root, name string) (string, error) {
	img, err := captureFullScreenImage()
	if err != nil {
		return "", fmt.Errorf("桌面截图失败: %w", err)
	}
	b := img.Bounds()
	return saveShotPNG(root, img, name, fmt.Sprintf("desktop_%dx%d", b.Dx(), b.Dy()))
}

// captureAreaNonWindows 区域截图：全屏 PNG + 标准库裁剪（语义与 Windows 版一致）。
func captureAreaNonWindows(root, leftStr, topStr, rightStr, bottomStr, name string) (string, error) {
	img, err := captureFullScreenImage()
	if err != nil {
		return "", fmt.Errorf("区域截图失败（全屏底图）: %w", err)
	}
	b := img.Bounds()
	vw, vh := b.Dx(), b.Dy()

	left, err := parseCoordNonWindows(leftStr, vw)
	if err != nil {
		return "", fmt.Errorf("left 参数无效: %w", err)
	}
	top, err := parseCoordNonWindows(topStr, vh)
	if err != nil {
		return "", fmt.Errorf("top 参数无效: %w", err)
	}
	right, err := parseCoordNonWindows(rightStr, vw)
	if err != nil {
		return "", fmt.Errorf("right 参数无效: %w", err)
	}
	bottom, err := parseCoordNonWindows(bottomStr, vh)
	if err != nil {
		return "", fmt.Errorf("bottom 参数无效: %w", err)
	}
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right > vw {
		right = vw
	}
	if bottom > vh {
		bottom = vh
	}
	if left >= right || top >= bottom {
		return "", fmt.Errorf("无效区域：left(%d) >= right(%d) 或 top(%d) >= bottom(%d)", left, right, top, bottom)
	}
	cropped := cropImage(img, left, top, right, bottom)
	return saveShotPNG(root, cropped, name, fmt.Sprintf("area_%dx%d", right-left, bottom-top))
}

// captureWindowNonWindows 按标题截取窗口。
func captureWindowNonWindows(root, title, name string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return captureWindowMac(root, title, name)
	default:
		return captureWindowLinux(root, title, name)
	}
}

// ── 全屏截图 ──────────────────────────────────────────────

// captureFullScreenImage 调系统截图命令得到 PNG，再解码为 image.Image。
func captureFullScreenImage() (image.Image, error) {
	data, err := captureFullScreenBytes()
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("截图结果不是有效 PNG: %w", err)
	}
	return img, nil
}

func captureFullScreenBytes() ([]byte, error) {
	tmp, err := os.CreateTemp("", "paircode-shot-*.png")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	_ = os.Remove(tmpPath) // 工具自己创建；先删掉避免个别工具拒绝覆盖
	defer os.Remove(tmpPath)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("screencapture", "-x", "-t", "png", tmpPath)
	} else {
		cmd = linuxScreenshotCmd(tmpPath)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("截图命令 %s 执行失败: %v（%s）", strings.Join(cmd.Args, " "), err, strings.TrimSpace(stderr.String()))
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("读取截图文件失败: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("截图命令 %s 没有产生文件（stderr: %s）", strings.Join(cmd.Args, " "), strings.TrimSpace(stderr.String()))
	}
	return data, nil
}

// linuxScreenshotCmd 探测 Linux 上可用的截图工具。
// 顺序兼顾 Wayland（grim）与 X11（scrot/import）及桌面自带工具。
func linuxScreenshotCmd(out string) *exec.Cmd {
	candidates := [][]string{
		{"grim", out},
		{"scrot", "-o", out},
		{"import", "-window", "root", out},
		{"gnome-screenshot", "-f", out},
		{"spectacle", "-b", "-n", "-o", out},
		{"xfce4-screenshooter", "-f", "-s", out},
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c[0]); err == nil {
			return exec.Command(p, c[1:]...)
		}
	}
	return exec.Command("sh", "-c",
		"echo '未找到截图工具：请安装 grim(Wayland)、scrot/ImageMagick(X11)、gnome-screenshot 或 spectacle(KDE) 之一' >&2; exit 127")
}

// ── 窗口截图 ──────────────────────────────────────────────

// captureWindowMac 用 System Events 枚举窗口 → screencapture -R。
// 需要「辅助功能」权限（系统设置 → 隐私与安全性 → 辅助功能）。
func captureWindowMac(root, title, name string) (string, error) {
	script := `tell application "System Events"
	set out to ""
	repeat with p in (every application process whose visible is true)
		try
			repeat with w in windows of p
				set t to name of w
				if t is missing value then set t to ""
				set {px, py} to position of w
				set {pw, ph} to size of w
				set out to out & t & "\t" & px & "\t" & py & "\t" & pw & "\t" & ph & "\n"
			end repeat
		end try
	end repeat
	return out
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("枚举 macOS 窗口失败（需辅助功能权限）: %w", err)
	}
	type win struct {
		title      string
		x, y, w, h int
	}
	var all []win
	needle := strings.ToLower(title)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) != 5 {
			continue
		}
		var x, y, w, h int
		if _, err := fmt.Sscanf(fields[1]+" "+fields[2]+" "+fields[3]+" "+fields[4], "%d %d %d %d", &x, &y, &w, &h); err != nil {
			continue
		}
		all = append(all, win{fields[0], x, y, w, h})
	}
	var hits []win
	for _, w := range all {
		if w.w > 0 && w.h > 0 && strings.Contains(strings.ToLower(w.title), needle) {
			hits = append(hits, w)
		}
	}
	if len(hits) == 0 {
		titles := make([]string, 0, 10)
		for _, w := range all {
			if w.title != "" && len(titles) < 10 {
				titles = append(titles, w.title)
			}
		}
		return "", fmt.Errorf("未找到标题匹配 %q 的窗口（可见窗口：%s）；可改用 screenshot_area",
			title, strings.Join(titles, " | "))
	}
	w := hits[0]
	tmp, err := os.CreateTemp("", "paircode-win-*.png")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	_ = os.Remove(tmpPath)
	defer os.Remove(tmpPath)
	region := fmt.Sprintf("%d,%d,%d,%d", w.x, w.y, w.w, w.h)
	var stderr bytes.Buffer
	cmd := exec.Command("screencapture", "-x", "-t", "png", "-R", region, tmpPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("窗口截图失败: %v（%s）", err, strings.TrimSpace(stderr.String()))
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("窗口截图不是有效 PNG: %w", err)
	}
	return saveShotPNG(root, img, name, fmt.Sprintf("window_%dx%d", w.w, w.h))
}

// captureWindowLinux 用 xdotool（X11）定位窗口再截该区域。
func captureWindowLinux(root, title, name string) (string, error) {
	xdo, err := exec.LookPath("xdotool")
	if err != nil {
		return "", fmt.Errorf("Linux 窗口截图需要 xdotool（X11）；当前环境无 xdotool/可能是 Wayland，请改用 screenshot_area")
	}
	ids, err := exec.Command(xdo, "search", "--onlyvisible", "--name", title).Output()
	if err != nil || strings.TrimSpace(string(ids)) == "" {
		return "", fmt.Errorf("未找到标题匹配 %q 的窗口；可改用 screenshot_area", title)
	}
	id := strings.Fields(string(ids))[0]
	geo, err := exec.Command(xdo, "getwindowgeometry", "--shell", id).Output()
	if err != nil {
		return "", fmt.Errorf("读取窗口几何失败: %w", err)
	}
	vals := map[string]int{}
	for _, line := range strings.Split(string(geo), "\n") {
		line = strings.TrimSpace(line)
		if kv := strings.SplitN(line, "=", 2); len(kv) == 2 {
			var n int
			if _, err := fmt.Sscanf(kv[1], "%d", &n); err == nil {
				vals[kv[0]] = n
			}
		}
	}
	x, y, w, h := vals["X"], vals["Y"], vals["WIDTH"], vals["HEIGHT"]
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("窗口 %s 几何无效（%dx%d @ %d,%d）；可改用 screenshot_area", id, w, h, x, y)
	}

	img, err := captureFullScreenImage()
	if err != nil {
		return "", fmt.Errorf("窗口截图底图失败: %w", err)
	}
	b := img.Bounds()
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > b.Dx() {
		w = b.Dx() - x
	}
	if y+h > b.Dy() {
		h = b.Dy() - y
	}
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("窗口区域超出屏幕范围；可改用 screenshot_area")
	}
	cropped := cropImage(img, x, y, x+w, y+h)
	return saveShotPNG(root, cropped, name, fmt.Sprintf("window_%dx%d", w, h))
}

// ── 图像处理与保存 ────────────────────────────────────────

// cropImage 从 (left,top)-(right,bottom) 裁剪（右下开区间）。
func cropImage(src image.Image, left, top, right, bottom int) image.Image {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	rect := image.Rect(left, top, right, bottom)
	if si, ok := src.(subImager); ok {
		return si.SubImage(rect)
	}
	dst := image.NewRGBA(image.Rect(0, 0, right-left, bottom-top))
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			dst.Set(x-left, y-top, src.At(x, y))
		}
	}
	return dst
}

// parseCoordNonWindows 与 Windows 版 parseCoord 语义一致：像素或百分比。
func parseCoordNonWindows(s string, total int) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		var pct float64
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "%"), "%f", &pct); err != nil {
			return 0, fmt.Errorf("无法解析百分比 %q: %w", s, err)
		}
		return int(math.Round(pct / 100.0 * float64(total))), nil
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return 0, fmt.Errorf("无法解析坐标 %q: %w", s, err)
	}
	return v, nil
}

// saveShotPNG 保存 PNG 并返回与 Windows 版同格式的结果文本。
func saveShotPNG(root string, img image.Image, customName, label string) (string, error) {
	dir := screenshotOutputDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建 screenshots 目录失败: %w", err)
	}
	filename := customName
	if filename == "" {
		filename = fmt.Sprintf("screenshot_%s_%s.png", time.Now().Format("20060102_150405"), sanitizeName(label))
	} else if !strings.HasSuffix(filename, ".png") {
		filename += ".png"
	}
	path := filepath.Join(dir, filename)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("PNG 编码失败: %w", err)
	}
	b := img.Bounds()
	return fmt.Sprintf("✅ 截图已保存\n  路径: %s\n  尺寸: %dx%d\n  时间: %s",
		path, b.Dx(), b.Dy(), time.Now().Format("2006-01-02 15:04:05")), nil
}
