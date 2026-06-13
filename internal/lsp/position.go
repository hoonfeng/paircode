// Package lsp 是一个精简的 LSP（Language Server Protocol）客户端实现。
// 它按语言按需启动语言服务器，通过磁盘同步文件内容，并将定义/引用/
// 悬停/诊断等只读 LSP 能力暴露为工具接口。
//
// 服务器不捆绑发行——它们通过 PATH 解析，缺少时返回清晰的安装提示。
package lsp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf16"
)

// Position 是零基 LSP 位置。Character 使用服务器在 initialize 阶段协商的编码
// （默认 utf-16，当双方都支持时改为 utf-8）。
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range 是两个 Position 之间的半开区间。
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location 是文件 URI 加 Range，definition/references 返回的形状。
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// pathToURI 将本地文件路径转为 file:// URI。
func pathToURI(p string) string {
	p = filepath.ToSlash(p)
	if runtime.GOOS == "windows" && len(p) > 1 && p[1] == ':' {
		p = "/" + p // C:/x → /C:/x 使得 URI 为 file:///C:/x
	}
	u := url.URL{Scheme: "file", Path: p}
	return u.String()
}

// uriToPath 将 file:// URI 转回本地文件路径。
func uriToPath(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	p := u.Path
	if runtime.GOOS == "windows" && len(p) > 2 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}

// locate 在 1-based 行上查找 symbol，返回 LSP Position（字节列按服务器编码转换）。
func locate(content string, line1 int, symbol, enc string) (Position, error) {
	lines := strings.Split(content, "\n")
	if line1 < 1 || line1 > len(lines) {
		return Position{}, fmt.Errorf("line %d out of range (file has %d lines)", line1, len(lines))
	}
	text := strings.TrimSuffix(lines[line1-1], "\r")
	col := strings.Index(text, symbol)
	if col < 0 {
		return Position{}, fmt.Errorf("symbol %q not found on line %d", symbol, line1)
	}
	return Position{Line: line1 - 1, Character: encodeChar(text[:col], enc)}, nil
}

func encodeChar(prefix, enc string) int {
	if enc == encodingUTF8 {
		return len(prefix)
	}
	return len(utf16.Encode([]rune(prefix)))
}

const (
	encodingUTF8  = "utf-8"
	encodingUTF16 = "utf-16"
)
