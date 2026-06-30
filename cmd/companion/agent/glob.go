package agent

// 支持 ** 的 glob 匹配。
// 标准库 filepath.Match / path.Match 只支持单层 *，不支持 ** 递归目录，
// 但工具 description 承诺了 **/*.go、src/**/auth* 等模式，导致合理命中失败。
// 本文件实现 ** 语义：** 匹配任意层目录（含零层）；* 匹配单层内非 / 字符；? 单字符。

import (
	"path"
	"path/filepath"
	"strings"
)

// matchGlob 支持 ** 的 glob 匹配。pattern 和 name 均为 slash 风格路径。
//   - **/*.go     匹配 a.go、foo/a.go、foo/bar/a.go
//   - src/**      匹配 src/a.go、src/foo/a.go
//   - src/**/auth* 匹配 src/auth.go、src/foo/auth.go
//   - *.go        仅匹配当前层 a.go（不含 / 时仅单层）
func matchGlob(pattern, name string) bool {
	pat := filepath.ToSlash(pattern)
	nm := filepath.ToSlash(name)
	return globMatchSegments(strings.Split(pat, "/"), strings.Split(nm, "/"))
}

// globMatchSegments 递归匹配已按 / 分割的路径段。
//   - patSeg[i]=="**" 时吃掉 0..len(nameSeg) 个段，剩余 pattern 递归匹配剩余 name。
//   - 其它段用 path.Match 做单层匹配（支持 * 和 ?，不含 /）。
func globMatchSegments(patSeg, nameSeg []string) bool {
	for len(patSeg) > 0 {
		if patSeg[0] == "**" {
			// ** 是最后一段：吃掉所有剩余 name 段
			if len(patSeg) == 1 {
				return true
			}
			// 尝试 ** 匹配 0..len(nameSeg) 个段
			for j := 0; j <= len(nameSeg); j++ {
				if globMatchSegments(patSeg[1:], nameSeg[j:]) {
					return true
				}
			}
			return false
		}
		if len(nameSeg) == 0 {
			return false
		}
		ok, err := path.Match(patSeg[0], nameSeg[0])
		if err != nil || !ok {
			return false
		}
		patSeg = patSeg[1:]
		nameSeg = nameSeg[1:]
	}
	// pattern 耗尽，name 也须耗尽才完全匹配
	return len(nameSeg) == 0
}

// matchGlobFilter 工具层 glob 过滤：pattern 含 / 或 ** 时按相对路径匹配，否则按文件名匹配。
// 语义：纯文件名模式（如 *.go）匹配任意深度的同名文件；路径模式（如 src/**/*.go）匹配相对路径。
func matchGlobFilter(pattern, base, rel string) bool {
	pat := filepath.ToSlash(pattern)
	if strings.Contains(pat, "/") || strings.Contains(pat, "**") {
		return matchGlob(pat, rel)
	}
	return matchGlob(pat, base)
}
