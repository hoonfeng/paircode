package qr

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestRenderPNG 冒烟测：生成 PNG、校验签名与尺寸、落盘供人工扫码。
func TestRenderPNG(t *testing.T) {
	const text = "https://ilinkai.weixin.qq.com/qr?token=smoke-test-0001"
	code, err := Encode(text, L)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	data, err := RenderPNG(text, 8)
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatal("PNG 签名错误")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("PNG 解码失败: %v", err)
	}
	want := (code.Size + 8) * 8 // 模块数 + 4 模块静区 ×2，每模块 8px
	if got := img.Bounds().Dx(); got != want {
		t.Fatalf("尺寸 %d != 期望 %d", got, want)
	}

	// 落盘到项目 temp/（符合临时产物规范），供人工/手机扫码验证
	dir := filepath.Join("..", "..", "..", "..", "temp", "wx-bridge-go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	out := filepath.Join(dir, "qr-smoke.png")
	if err := os.WriteFile(out, data, 0o644); err != nil {
		t.Fatalf("落盘: %v", err)
	}
	t.Logf("OK: %d 字节，%dx%d，二维码版本尺寸=%d 模块 → %s", len(data), want, want, code.Size, out)
}
