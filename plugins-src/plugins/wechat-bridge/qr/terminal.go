// terminal.go — 二维码终端渲染（调试/无 UI 环境用）。
//
// 用半块字符（▀▄█）把二维码画到终端；每行两个垂直像素合并为一个字符，
// 补偿终端字符高宽比 ≈ 2:1。方向按「深色终端背景」约定：白色模块 = 亮字符、
// 黑色模块 = 空（底色），得到正色二维码（微信可识别）。
package qr

import "strings"

// Terminal 渲染终端文本（含 4 模块静区）。
func (c *Code) Terminal() string {
	const pad = 4
	n := c.Size
	black := func(x, y int) bool {
		if x < 0 || y < 0 || x >= n || y >= n {
			return false
		}
		return c.Black(x, y)
	}
	var b strings.Builder
	b.Grow((n + pad*2 + 1) * (n + pad*2) / 2)
	for y := -pad; y < n+pad; y += 2 {
		for x := -pad; x < n+pad; x++ {
			top, bot := black(x, y), black(x, y+1)
			switch {
			case !top && !bot:
				b.WriteString("█") // 全白
			case !top && bot:
				b.WriteString("▀") // 上白下黑
			case top && !bot:
				b.WriteString("▄") // 上黑下白
			default:
				b.WriteString(" ") // 全黑
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Terminal 渲染文本为终端二维码（编码失败返回错误）。
func TerminalText(text string) (string, error) {
	code, err := Encode(text, L)
	if err != nil {
		return "", err
	}
	return code.Terminal(), nil
}
