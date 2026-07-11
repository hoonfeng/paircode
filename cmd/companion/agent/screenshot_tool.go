// screenshot_tool.go 截图工具集：桌面截图 + 窗口截图 + 区域截图 + 网页截图。
//
// 提供四个工具：
//   - screenshot_desktop：截取整个桌面（所有显示器）
//   - screenshot_window：按窗口标题截取特定窗口
//   - screenshot_area：按坐标截取指定区域
//   - screenshot_webpage：用浏览器打开 URL 并截图
//
// 桌面/窗口/区域截图使用 Windows GDI API（golang.org/x/sys/windows），零额外依赖。
// 网页截图使用 go-rod（github.com/go-rod/rod），自动查找本地的 Edge/Chrome。
// 所有截图保存为 PNG 文件到工作区的 screenshots/ 目录。

package agent

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"golang.org/x/sys/windows"
)

// ── Windows GDI 常量 ───────────────────────────────────────

const (
	_SRCCOPY        = 0x00CC0020
	_DIB_RGB_COLORS = 0
	_BI_RGB         = 0
	_SM_CXSCREEN      = 0
	_SM_CYSCREEN      = 1
	_SM_XVIRTUALSCREEN  = 76
	_SM_YVIRTUALSCREEN  = 77
	_SM_CXVIRTUALSCREEN = 78
	_SM_CYVIRTUALSCREEN = 79
)

// ── Windows API ────────────────────────────────────────────

var (
	modgdi32  = windows.NewLazySystemDLL("gdi32.dll")
	moduser32 = windows.NewLazySystemDLL("user32.dll")

	// user32
	procGetDesktopWindow     = moduser32.NewProc("GetDesktopWindow")
	procGetDC                = moduser32.NewProc("GetDC")
	procReleaseDC            = moduser32.NewProc("ReleaseDC")
	procGetSystemMetrics     = moduser32.NewProc("GetSystemMetrics")
	procEnumWindows          = moduser32.NewProc("EnumWindows")
	procGetWindowTextW       = moduser32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = moduser32.NewProc("GetWindowTextLengthW")
	procGetWindowRect        = moduser32.NewProc("GetWindowRect")
	procIsWindowVisible      = moduser32.NewProc("IsWindowVisible")
	procGetForegroundWindow  = moduser32.NewProc("GetForegroundWindow")

	// gdi32
	procCreateDCW              = modgdi32.NewProc("CreateDCW")
	procCreateCompatibleDC     = modgdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modgdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = modgdi32.NewProc("SelectObject")
	procBitBlt                 = modgdi32.NewProc("BitBlt")
	procGetDIBits              = modgdi32.NewProc("GetDIBits")
	procDeleteDC               = modgdi32.NewProc("DeleteDC")
	procDeleteObject           = modgdi32.NewProc("DeleteObject")
)

// ── Windows 类型 ───────────────────────────────────────────

type _BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type _BITMAPINFO struct {
	BmiHeader _BITMAPINFOHEADER
	BmiColors [4]byte
}

// ── 窗口信息 ───────────────────────────────────────────────

type _windowInfo struct {
	HWND  windows.Handle
	Title string
	Rect  struct{ Left, Top, Right, Bottom int32 }
}

// ── 注册 ───────────────────────────────────────────────────

// registerScreenshotTools 注册四个截图工具。
func registerScreenshotTools(r *Registry, root string) {
	// ── screenshot_desktop ──
	r.Register(&Tool{
		Name: "screenshot_desktop",
		Description: "截取整个桌面（所有显示器），保存为 PNG 图片到 screenshots/ 目录。" +
			"返回文件路径、尺寸和截图时间。" +
			"之后可用 image_analyze 分析截图中的颜色/色块/图形，或用 image_ocr 识别文字。",
		Parameters: objSchema(props{
			"name": strProp("可选：自定义文件名（不含扩展名），默认自动生成时间戳名称"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			return captureDesktop(root, name)
		},
	})

	// ── screenshot_window ──
	r.Register(&Tool{
		Name: "screenshot_window",
		Description: "按窗口标题或标题子串截取特定窗口，保存为 PNG 图片到 screenshots/ 目录。" +
			"返回文件路径、窗口尺寸和截图时间。" +
			"如果多个窗口匹配同一标题子串，会列出所有匹配窗口供选择。",
		Parameters: objSchema(props{
			"title": strProp("窗口标题或标题子串（不区分大小写）。例如 \"记事本\"、\"Chrome\"、\"Calculator\""),
			"name":  strProp("可选：自定义文件名（不含扩展名），默认自动生成"),
		}, "title"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			title := argStr(args, "title")
			name := argStr(args, "name")
			if title == "" {
				return "", fmt.Errorf("参数 title 不能为空")
			}
			return captureWindow(root, title, name)
		},
	})

	// ── screenshot_area ──
	r.Register(&Tool{
		Name: "screenshot_area",
		Description: "按坐标截取指定区域，保存为 PNG 图片到 screenshots/ 目录。" +
			"区域坐标可以是绝对坐标（相对于桌面左上角），也可以是百分比（如 \"10% 20% 50% 30%\"）。" +
			"返回文件路径、区域尺寸和截图时间。",
		Parameters: objSchema(props{
			"left":   strProp("左边界：像素值或百分比（如 \"10%\"）"),
			"top":    strProp("上边界：像素值或百分比"),
			"right":  strProp("右边界：像素值或百分比"),
			"bottom": strProp("下边界：像素值或百分比"),
			"name":   strProp("可选：自定义文件名"),
		}, "left", "top", "right", "bottom"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			left := argStr(args, "left")
			top := argStr(args, "top")
			right := argStr(args, "right")
			bottom := argStr(args, "bottom")
			name := argStr(args, "name")
			return captureArea(root, left, top, right, bottom, name)
		},
	})

	// ── screenshot_webpage ──
	r.Register(&Tool{
		Name: "screenshot_webpage",
		Description: "打开指定 URL 的网页并截图，保存为 PNG 图片到 screenshots/ 目录。" +
			"使用本地的 Edge/Chrome 浏览器（无头模式）。" +
			"可设置 viewport 尺寸（默认 1920x1080）和等待时间。",
		Parameters: objSchema(props{
			"url":    strProp("要截图的网页 URL（如 \"https://example.com\"）"),
			"width":  intProp("可选：视口宽度（像素），默认 1920"),
			"height": intProp("可选：视口高度（像素），默认 1080"),
			"wait":   intProp("可选：页面加载后额外等待的毫秒数，默认 1000（给 JS 渲染时间）"),
			"name":   strProp("可选：自定义文件名"),
		}, "url"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			url := argStr(args, "url")
			width := argInt(args, "width", 1920)
			height := argInt(args, "height", 1080)
			waitMs := argInt(args, "wait", 1000)
			name := argStr(args, "name")
			if url == "" {
				return "", fmt.Errorf("参数 url 不能为空")
			}
			return captureWebpage(root, url, width, height, waitMs, name)
		},
	})
}

// ── 截图实现 ───────────────────────────────────────────────

// captureDesktop 截取整个桌面。
func captureDesktop(root, name string) (string, error) {
	vx := int(getSystemMetrics(_SM_XVIRTUALSCREEN))
	vy := int(getSystemMetrics(_SM_YVIRTUALSCREEN))
	vw := int(getSystemMetrics(_SM_CXVIRTUALSCREEN))
	vh := int(getSystemMetrics(_SM_CYVIRTUALSCREEN))

	if vw <= 0 || vh <= 0 {
		vx, vy = 0, 0
		vw = int(getSystemMetrics(_SM_CXSCREEN))
		vh = int(getSystemMetrics(_SM_CYSCREEN))
	}

	img, err := captureRect(vx, vy, vw, vh)
	if err != nil {
		return "", fmt.Errorf("桌面截图失败: %w", err)
	}

	return saveScreenshot(root, img, name, fmt.Sprintf("desktop_%dx%d", vw, vh))
}

// captureWindow 按窗口标题截取特定窗口。
func captureWindow(root, title string, name string) (string, error) {
	windows, err := enumWindows(title)
	if err != nil {
		return "", fmt.Errorf("枚举窗口失败: %w", err)
	}

	if len(windows) == 0 {
		allWindows, _ := enumWindows("")
		var titles []string
		for _, w := range allWindows {
			if w.Title != "" {
				titles = append(titles, w.Title)
			}
		}
		n := 20
		if len(titles) < n {
			n = len(titles)
		}
		return "", fmt.Errorf("未找到标题包含 %q 的窗口。当前可见窗口（前 20 个）：\n%s",
			title, strings.Join(titles[:n], "\n"))
	}

	if len(windows) > 1 {
		var details []string
		for i, w := range windows {
			details = append(details, fmt.Sprintf("  [%d] HWND=0x%X 标题=%q 位置=(%d,%d)-(%d,%d) 尺寸=%dx%d",
				i+1, w.HWND, w.Title, w.Rect.Left, w.Rect.Top, w.Rect.Right, w.Rect.Bottom,
				w.Rect.Right-w.Rect.Left, w.Rect.Bottom-w.Rect.Top))
		}
		return "", fmt.Errorf("有 %d 个窗口匹配 %q，请指定更精确的标题：\n%s",
			len(windows), title, strings.Join(details, "\n"))
	}

	w := windows[0]
	winW := int(w.Rect.Right - w.Rect.Left)
	winH := int(w.Rect.Bottom - w.Rect.Top)
	if winW <= 0 || winH <= 0 {
		return "", fmt.Errorf("窗口 %q 尺寸无效 (%dx%d)", w.Title, winW, winH)
	}

	img, err := captureRect(int(w.Rect.Left), int(w.Rect.Top), winW, winH)
	if err != nil {
		return "", fmt.Errorf("窗口 %q 截图失败: %w", w.Title, err)
	}

	label := fmt.Sprintf("window_%s_%dx%d", sanitizeName(w.Title), winW, winH)
	return saveScreenshot(root, img, name, label)
}

// captureArea 按坐标截取指定区域。
func captureArea(root, leftStr, topStr, rightStr, bottomStr, name string) (string, error) {
	vw := int(getSystemMetrics(_SM_CXVIRTUALSCREEN))
	vh := int(getSystemMetrics(_SM_CYVIRTUALSCREEN))
	if vw <= 0 || vh <= 0 {
		vw = int(getSystemMetrics(_SM_CXSCREEN))
		vh = int(getSystemMetrics(_SM_CYSCREEN))
	}

	left, err := parseCoord(leftStr, vw)
	if err != nil {
		return "", fmt.Errorf("left 参数无效: %w", err)
	}
	top, err := parseCoord(topStr, vh)
	if err != nil {
		return "", fmt.Errorf("top 参数无效: %w", err)
	}
	right, err := parseCoord(rightStr, vw)
	if err != nil {
		return "", fmt.Errorf("right 参数无效: %w", err)
	}
	bottom, err := parseCoord(bottomStr, vh)
	if err != nil {
		return "", fmt.Errorf("bottom 参数无效: %w", err)
	}

	if left >= right || top >= bottom {
		return "", fmt.Errorf("无效区域：left(%d) >= right(%d) 或 top(%d) >= bottom(%d)", left, right, top, bottom)
	}

	w := right - left
	h := bottom - top

	img, err := captureRect(left, top, w, h)
	if err != nil {
		return "", fmt.Errorf("区域截图失败: %w", err)
	}

	label := fmt.Sprintf("area_%dx%d", w, h)
	return saveScreenshot(root, img, name, label)
}

// captureWebpage 用浏览器打开 URL 并截图。
func captureWebpage(root, urlStr string, width, height, waitMs int, name string) (string, error) {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	path, ok := launcher.LookPath()
	if !ok || path == "" {
		return "", fmt.Errorf("未找到本地浏览器（Edge/Chrome/Chromium）。请先安装 Edge 或 Chrome。")
	}

	u := launcher.New().Headless(true).Bin(path).MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(urlStr)
	defer page.MustClose()

	page.MustSetViewport(width, height, 0, false)
	page.MustWaitLoad()

	if waitMs > 0 {
		time.Sleep(time.Duration(waitMs) * time.Millisecond)
	}

	buf, err := page.Screenshot(true, nil)
	if err != nil {
		return "", fmt.Errorf("网页截图失败: %w", err)
	}

	return saveScreenshotFromBytes(root, buf, name, fmt.Sprintf("webpage_%s_%dx%d",
		sanitizeName(extractHost(urlStr)), width, height))
}

// captureRect 用 Windows GDI 截取指定矩形区域的屏幕内容。
func captureRect(x, y, width, height int) (*image.RGBA, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("无效尺寸: %dx%d", width, height)
	}

	hwnd, _, _ := procGetDesktopWindow.Call()
	hdc, _, _ := procGetDC.Call(hwnd)
	if hdc == 0 {
		return nil, fmt.Errorf("GetDC 失败")
	}
	defer procReleaseDC.Call(hwnd, hdc)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdc)
	if hdcMem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC 失败")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateCompatibleBitmap.Call(hdc, uintptr(width), uintptr(height))
	if hBitmap == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap 失败")
	}
	defer procDeleteObject.Call(hBitmap)

	_, _, _ = procSelectObject.Call(hdcMem, hBitmap)

	ret, _, _ := procBitBlt.Call(hdcMem, 0, 0, uintptr(width), uintptr(height),
		hdc, uintptr(x), uintptr(y), _SRCCOPY)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt 失败")
	}

	bmi := _BITMAPINFO{}
	bmi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bmi.BmiHeader))
	bmi.BmiHeader.BiWidth = int32(width)
	bmi.BmiHeader.BiHeight = -int32(height)
	bmi.BmiHeader.BiPlanes = 1
	bmi.BmiHeader.BiBitCount = 32
	bmi.BmiHeader.BiCompression = _BI_RGB

	pixelData := make([]byte, width*height*4)

	_, _, _ = procGetDIBits.Call(hdc, hBitmap, 0, uintptr(height),
		uintptr(unsafe.Pointer(&pixelData[0])),
		uintptr(unsafe.Pointer(&bmi)),
		_DIB_RGB_COLORS)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < width*height; i++ {
		offset := i * 4
		b := pixelData[offset]
		g := pixelData[offset+1]
		r := pixelData[offset+2]
		a := pixelData[offset+3]
		img.Pix[offset] = r
		img.Pix[offset+1] = g
		img.Pix[offset+2] = b
		img.Pix[offset+3] = a
	}

	return img, nil
}

// ── 窗口枚举 ───────────────────────────────────────────────

var windowsCallback func(hwnd windows.Handle) bool

// enumWindows 枚举所有可见窗口，返回标题包含子串的窗口列表。
func enumWindows(substr string) ([]_windowInfo, error) {
	substr = strings.ToLower(substr)
	var results []_windowInfo
	var allWindows []_windowInfo

	cb := windows.NewCallback(func(hwnd uintptr) uintptr {
		h := windows.Handle(hwnd)

		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}

		lenProc, _, _ := procGetWindowTextLengthW.Call(hwnd)
		if lenProc == 0 {
			return 1
		}

		buf := make([]uint16, lenProc+1)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(lenProc+1))

		title := utf16.Decode(buf[:lenProc])
		titleStr := string(title)

		rect := struct{ Left, Top, Right, Bottom int32 }{}
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
		winW := rect.Right - rect.Left
		winH := rect.Bottom - rect.Top

		if winW < 10 || winH < 10 {
			return 1
		}

		info := _windowInfo{
			HWND:  h,
			Title: titleStr,
			Rect:  rect,
		}
		allWindows = append(allWindows, info)

		if substr == "" || strings.Contains(strings.ToLower(titleStr), substr) {
			results = append(results, info)
		}

		return 1
	})

	enumRet, _, _ := procEnumWindows.Call(cb, 0)
	if enumRet == 0 {
		return results, nil
	}

	if substr == "" {
		return allWindows, nil
	}
	return results, nil
}

// ── 辅助函数 ───────────────────────────────────────────────

func getSystemMetrics(index int) int {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(ret)
}

func parseCoord(s string, total int) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		pctStr := strings.TrimSuffix(s, "%")
		var pct float64
		if _, err := fmt.Sscanf(pctStr, "%f", &pct); err != nil {
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

func extractHost(urlStr string) string {
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")
	if idx := strings.Index(urlStr, "/"); idx > 0 {
		urlStr = urlStr[:idx]
	}
	if idx := strings.Index(urlStr, "?"); idx > 0 {
		urlStr = urlStr[:idx]
	}
	return urlStr
}

func saveScreenshot(root string, img image.Image, customName string, label string) (string, error) {
	screenshotsDir := filepath.Join(root, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		return "", fmt.Errorf("创建 screenshots 目录失败: %w", err)
	}

	filename := customName
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("screenshot_%s_%s.png", timestamp, sanitizeName(label))
	} else if !strings.HasSuffix(filename, ".png") {
		filename += ".png"
	}
	path := filepath.Join(screenshotsDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return "", fmt.Errorf("PNG 编码失败: %w", err)
	}

	bounds := img.Bounds()
	return fmt.Sprintf("✅ 截图已保存\n  路径: %s\n  尺寸: %dx%d\n  时间: %s",
		path, bounds.Dx(), bounds.Dy(),
		time.Now().Format("2006-01-02 15:04:05")), nil
}

func saveScreenshotFromBytes(root string, data []byte, customName string, label string) (string, error) {
	screenshotsDir := filepath.Join(root, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		return "", fmt.Errorf("创建 screenshots 目录失败: %w", err)
	}

	filename := customName
	if filename == "" {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("screenshot_%s_%s.png", timestamp, sanitizeName(label))
	} else if !strings.HasSuffix(filename, ".png") {
		filename += ".png"
	}
	path := filepath.Join(screenshotsDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	bounds := "?"
	if img, _, err := image.Decode(strings.NewReader(string(data))); err == nil {
		bb := img.Bounds()
		bounds = fmt.Sprintf("%dx%d", bb.Dx(), bb.Dy())
	}

	return fmt.Sprintf("✅ 网页截图保存\n  路径: %s\n  尺寸: %s\n  时间: %s",
		path, bounds,
		time.Now().Format("2006-01-02 15:04:05")), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
