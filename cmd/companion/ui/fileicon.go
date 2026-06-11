//go:build windows

package ui

import (
	"path/filepath"
	"strings"

	"github.com/user/goui/pkg/types"
)

// 文件类型图标色（语义固定色，不随主题；文件树行悬停/选中色另用 FtHover/FtSelected）。
var (
	ftFolder = types.ColorFromRGB(134, 174, 224) // 文件夹图标：柔和蓝
	ftJson   = types.ColorFromRGB(250, 204, 21)  // json/yaml：黄
)

// extColor 按扩展名给代码文件上色（其余走调用方默认）。
func extColor(ext string) types.Color {
	switch ext {
	case ".go":
		return types.ColorFromRGB(0, 173, 216) // cyan
	case ".ts", ".tsx":
		return types.ColorFromRGB(49, 120, 198)
	case ".js", ".jsx", ".mjs":
		return types.ColorFromRGB(247, 223, 30)
	case ".py":
		return types.ColorFromRGB(82, 139, 195)
	case ".rs":
		return types.ColorFromRGB(222, 165, 132)
	case ".html":
		return types.ColorFromRGB(227, 76, 38)
	case ".css", ".scss":
		return types.ColorFromRGB(66, 184, 131)
	case ".sh", ".bat", ".ps1":
		return types.ColorFromRGB(63, 185, 80)
	default:
		return *ShellText
	}
}

// FileIcon 返回文件/目录的 Lucide 图标名 + 颜色。供文件树行 + 编辑器标签共用
// （收原始字段而非 *fileNode，避免反依赖文件树类型；编辑器标签传 isDir=false）。
func FileIcon(name string, isDir, expanded bool) (string, types.Color) {
	if isDir {
		if expanded {
			return "folder-open", ftFolder
		}
		return "folder", ftFolder
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".rs", ".c", ".cpp", ".h", ".java",
		".py", ".rb", ".php", ".cs", ".swift", ".kt", ".html", ".css", ".scss", ".sh", ".bat", ".ps1":
		return "file-code", extColor(strings.ToLower(filepath.Ext(name)))
	case ".json", ".yaml", ".yml", ".toml", ".mod", ".sum", ".lock":
		return "braces", ftJson
	case ".md", ".txt", ".rst", ".log":
		return "file-text", *ShellTextDim
	default:
		return "file", *ShellTextDim
	}
}
