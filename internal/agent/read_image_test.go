package agent

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadImage read_image：PNG 元信息 + base64；非图片/超限明确报错。
func TestReadImage(t *testing.T) {
	root := t.TempDir()
	reg := NewRegistry()
	RegisterDefaultTools(reg, root)
	ctx := context.Background()

	// 造一张 4x3 PNG
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{255, 0, 0, 255})
	fp := filepath.Join(root, "shot.png")
	f, _ := os.Create(fp)
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := reg.Execute(ctx, "read_image", `{"file_path":"shot.png"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"width: 4", "height: 3", "mediaType: image/png", "bytes(base64):"} {
		if !strings.Contains(got, want) {
			t.Errorf("read_image 应含 %q：\n%s", want, got)
		}
	}

	// 非图片
	if _, err := reg.Execute(ctx, "read_image", `{"file_path":"go.mod"}`); err == nil {
		t.Error("非图片应报错")
	}
	// 不存在
	if _, err := reg.Execute(ctx, "read_image", `{"file_path":"nope.png"}`); err == nil {
		t.Error("不存在应报错")
	}
}
