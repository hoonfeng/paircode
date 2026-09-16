// render.go — 微信桥二维码渲染便捷层。
//
// 在 vendored rsc.io/qr 之上提供一步式渲染：文本 → PNG 字节（含 4 模块静区）。
// 供本地 HTTP /qr.png 与调试命令使用；二维码始终实时渲染、不落盘（方案决策②）。
package qr

// DefaultScale 是默认的每 QR 模块像素数（8 → v10 码 520×520 像素，屏幕扫描清晰）。
const DefaultScale = 8

// RenderPNG 把文本编码为二维码并渲染成 PNG 字节。
//   - scale：每 QR 模块的像素数（<2 时取 DefaultScale）
//   - 纠错级别 L（20% 冗余）——与 M1 实测可扫参数一致，同版本容量最大
func RenderPNG(text string, scale int) ([]byte, error) {
	if scale < 2 {
		scale = DefaultScale
	}
	code, err := Encode(text, L)
	if err != nil {
		return nil, err
	}
	code.Scale = scale
	return code.PNG(), nil
}
