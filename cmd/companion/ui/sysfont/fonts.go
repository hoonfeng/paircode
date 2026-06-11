// Package sysfont 枚举 Windows 已安装字体（外观设置的字体选择器用）：读注册表已安装字体名，
// 挑出编程/等宽字体作候选。companion 分层：纯 syscall 叶子工具，无 UI/状态依赖。
//
//go:build windows

package sysfont

import (
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

var (
	advapi32         = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx = advapi32.NewProc("RegOpenKeyExW")
	procRegEnumValue = advapi32.NewProc("RegEnumValueW")
	procRegCloseKey  = advapi32.NewProc("RegCloseKey")
)

// commonCodingFonts 常见编程/等宽字体：装了就置顶（顺序固定、好找）。
var commonCodingFonts = []string{
	"Cascadia Code", "Cascadia Mono", "Cascadia Code PL", "Cascadia Mono PL",
	"JetBrains Mono", "Fira Code", "Source Code Pro", "Hack", "Consolas",
	"Courier New", "Lucida Console", "Inconsolata", "Roboto Mono",
	"IBM Plex Mono", "Ubuntu Mono", "DejaVu Sans Mono", "Sarasa Mono SC",
	"Noto Sans Mono",
}

// System 枚举已安装字体名（注册表 Fonts 键的值名，去 "(TrueType)" 等后缀，去重排序）。失败返回 nil。
func System() []string {
	keyPath, err := syscall.UTF16PtrFromString(`SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`)
	if err != nil {
		return nil
	}
	const hkeyLocalMachine = 0x80000002
	const keyRead = 0x20019
	var hkey syscall.Handle
	if r, _, _ := procRegOpenKeyEx.Call(uintptr(hkeyLocalMachine), uintptr(unsafe.Pointer(keyPath)), 0, keyRead, uintptr(unsafe.Pointer(&hkey))); r != 0 {
		return nil
	}
	defer procRegCloseKey.Call(uintptr(hkey))

	seen := map[string]bool{}
	var out []string
	nameBuf := make([]uint16, 512)
	for i := uint32(0); ; i++ {
		nameLen := uint32(len(nameBuf))
		r, _, _ := procRegEnumValue.Call(uintptr(hkey), uintptr(i),
			uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(unsafe.Pointer(&nameLen)), 0, 0, 0, 0)
		if r != 0 { // ERROR_NO_MORE_ITEMS 或出错 → 结束
			break
		}
		name := cleanFontName(syscall.UTF16ToString(nameBuf[:nameLen]))
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// fontStyleWords 字重/字形词：从注册表字体名尾部剥掉，归并到「字族名」（Consolas Bold→Consolas）。
var fontStyleWords = map[string]bool{
	"regular": true, "bold": true, "italic": true, "oblique": true, "light": true,
	"medium": true, "semibold": true, "demibold": true, "thin": true, "black": true,
	"heavy": true, "extrabold": true, "extralight": true, "ultralight": true,
	"semilight": true, "book": true,
}

// cleanFontName 注册表字体名 → 字族名：去 "(TrueType)" 等后缀、" & " 多名、尾部字重/字形词；
// 跳过含逗号的老式位图字体（返回 ""）。
func cleanFontName(s string) string {
	if i := strings.Index(s, " ("); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, " & "); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if strings.Contains(s, ",") { // "Courier 10,12,15" 这类位图字体，跳过
		return ""
	}
	parts := strings.Fields(s)
	for len(parts) > 1 { // 剥尾部样式词，得字族名
		if fontStyleWords[strings.ToLower(parts[len(parts)-1])] {
			parts = parts[:len(parts)-1]
		} else {
			break
		}
	}
	return strings.Join(parts, " ")
}

// looksMonospace 名字像等宽/编程字体（启发式，给字体选择器筛候选用）。先排子串假阳性
// （uniCODE / MONOtype / barCODE 等），再匹关键字。
func looksMonospace(name string) bool {
	if strings.Contains(name, "等宽") {
		return true
	}
	lf := strings.ToLower(name)
	for _, bad := range []string{"unicode", "monotype", "barcode", "decode", "encode", "geocode"} {
		if strings.Contains(lf, bad) {
			return false
		}
	}
	for _, kw := range []string{"mono", "code", "consol", "courier", "typewriter", "fixedsys"} {
		if strings.Contains(lf, kw) {
			return true
		}
	}
	return false
}

// Available 字体选择器候选：已装的常见编程字体置顶 + 其余「名字像等宽」的已装字体；附带 always 一定包含的项
// （如当前已选字体）。读不到注册表时退回常见列表。
func Available(always ...string) []string {
	installed := System()
	have := map[string]bool{}
	for _, f := range installed {
		have[f] = true
	}
	seen := map[string]bool{}
	var out []string
	add := func(f string) {
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	for _, f := range always { // 当前选中等：保证可选中
		add(f)
	}
	for _, f := range commonCodingFonts {
		if len(installed) == 0 || have[f] { // 注册表读不到时也列常见字体
			add(f)
		}
	}
	for _, f := range installed { // 其余名字像等宽的已装字体
		if looksMonospace(f) {
			add(f)
		}
	}
	return out
}
