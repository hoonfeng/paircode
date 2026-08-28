//go:build !windows

package agent

// screenshot_tool_other.go — 非 Windows 平台的截图工具实现。
//
// ★ 2026-08-22：screenshot_tool.go 使用 Windows GDI API（golang.org/x/sys/windows），
//   原无 build tag 导致 Linux/mac 交叉编译失败
//   （"golang.org/x/sys/windows: build constraints exclude all Go files"）。
//   本文件提供同签名注册入口：工具面保持一致，Handler 返回"仅 Windows 支持"。

import (
	"context"
	"fmt"
	"runtime"
)

// registerScreenshotTools 非 Windows 平台：注册截图工具但运行时报不支持。
func registerScreenshotTools(r *Registry, root string) {
	unsupported := func(name string) (string, error) {
		return "", fmt.Errorf("截图工具（%s）仅支持 Windows（依赖 GDI API）；本平台为 %s",
			name, runtime.GOOS)
	}
	r.Register(&Tool{
		Name:        "screenshot_desktop",
		UsageGuide:  "截取整个桌面（所有显示器），保存为 PNG。仅 Windows 支持（GDI API）。",
		Description: "截取整个桌面（所有显示器），保存为 PNG 图片到 screenshots/ 目录。仅 Windows 支持。",
		Parameters: objSchema(props{
			"name": strProp("可选：自定义文件名（不含扩展名），默认自动生成时间戳名称"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return unsupported("screenshot_desktop")
		},
	})
	r.Register(&Tool{
		Name:        "screenshot_window",
		UsageGuide:  "按窗口标题截取特定窗口，保存为 PNG。仅 Windows 支持（GDI API）。",
		Description: "按窗口标题或标题子串截取特定窗口，保存为 PNG 图片到 screenshots/ 目录。仅 Windows 支持。",
		Parameters: objSchema(props{
			"title": strProp("窗口标题或标题子串（不区分大小写）"),
			"name":  strProp("可选：自定义文件名（不含扩展名），默认自动生成"),
		}, "title"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return unsupported("screenshot_window")
		},
	})
	r.Register(&Tool{
		Name:        "screenshot_area",
		UsageGuide:  "按坐标截取指定屏幕区域。仅 Windows 支持（GDI API）。",
		Description: "按坐标截取指定区域，保存为 PNG 图片到 screenshots/ 目录。仅 Windows 支持。",
		Parameters: objSchema(props{
			"left":   strProp("左边界：像素值或百分比"),
			"top":    strProp("上边界：像素值或百分比"),
			"right":  strProp("右边界：像素值或百分比"),
			"bottom": strProp("下边界：像素值或百分比"),
			"name":   strProp("可选：自定义文件名"),
		}, "left", "top", "right", "bottom"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return unsupported("screenshot_area")
		},
	})
}
