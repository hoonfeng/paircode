package agent

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// makeTestPNG 生成 w×h 的测试 PNG（横向渐变，保证编码后非平凡大小）。
func makeTestPNG(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if alpha {
				a = uint8((x * 255) / max(1, w))
			}
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: uint8((x + y) % 256), A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("测试 PNG 编码失败: %v", err)
	}
	return buf.Bytes()
}

// TestFitDimensions 像素预算 / 长边约束 / 不放大。
func TestFitDimensions(t *testing.T) {
	lim := deepseekRequestImageLimits
	w, h := fitDimensions(2000, 1500, lim)
	if int64(w)*int64(h) > lim.MaxPixels {
		t.Errorf("降采样后像素 %dx%d 超过预算 %d", w, h, lim.MaxPixels)
	}
	// 保持宽高比（容差 1px）
	if ratio := float64(w) / float64(h); ratio < 1.32 || ratio > 1.35 {
		t.Errorf("宽高比失真: %dx%d", w, h)
	}
	// 不放大
	w2, h2 := fitDimensions(10, 10, lim)
	if w2 != 10 || h2 != 10 {
		t.Errorf("不应放大: %dx%d", w2, h2)
	}
	// 长边约束（极端宽高比）
	lim2 := ImageLimits{MaxPixels: 100_000_000, MaxDimension: 8192, MaxBytes: 1 << 20}
	w3, h3 := fitDimensions(20000, 10, lim2)
	if w3 > 8192 {
		t.Errorf("长边应受 8192 约束: %dx%d", w3, h3)
	}
}

// TestNormalizeImagePassThrough 已满足限制的图片字节级透传。
func TestNormalizeImagePassThrough(t *testing.T) {
	data := makeTestPNG(t, 8, 8, false)
	out, err := normalizeImageBytes(data, normalizationImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	if !bytes.Equal(out.Data, data) {
		t.Errorf("满足限制应原样透传（字节不一致）")
	}
	if out.Width != 8 || out.Height != 8 || out.MediaType != "image/png" {
		t.Errorf("透传事实错误: %+v", out)
	}
}

// TestNormalizeImageDownscalesToBudget 超预算图片降采样到像素/字节预算内。
func TestNormalizeImageDownscalesToBudget(t *testing.T) {
	data := makeTestPNG(t, 2000, 1500, false)
	out, err := normalizeImageBytes(data, deepseekRequestImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	if int64(out.Width)*int64(out.Height) > deepseekRequestImageLimits.MaxPixels {
		t.Errorf("像素 %dx%d 超预算 %d", out.Width, out.Height, deepseekRequestImageLimits.MaxPixels)
	}
	if int64(len(out.Data)) > deepseekRequestImageLimits.MaxBytes {
		t.Errorf("字节 %d 超预算 %d", len(out.Data), deepseekRequestImageLimits.MaxBytes)
	}
	if out.MediaType != "image/jpeg" {
		t.Errorf("不透明图应编码为 JPEG，得 %s", out.MediaType)
	}
	// 归一化（2048²）预算下不缩放，仅重编码到 4MiB 内
	norm, err := normalizeImageBytes(data, normalizationImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	if norm.Width != 2000 || norm.Height != 1500 {
		t.Errorf("2048² 预算内不应缩放: %dx%d", norm.Width, norm.Height)
	}
}

// TestNormalizeImageKeepsAlpha 透明图保留 alpha（PNG 编码，不转 JPEG）。
func TestNormalizeImageKeepsAlpha(t *testing.T) {
	data := makeTestPNG(t, 300, 300, true)
	out, err := normalizeImageBytes(data, deepseekRequestImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	if out.MediaType != "image/png" {
		t.Errorf("含 alpha 应保留 PNG，得 %s", out.MediaType)
	}
	facts, _, err := decodeImageBytes(out.Data)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	if !facts.HasAlpha {
		t.Errorf("输出应保留 alpha 通道")
	}
}

// TestProjectImageForRequest 请求投影：预算内原样、超预算再降采样。
func TestProjectImageForRequest(t *testing.T) {
	norm, err := normalizeImageBytes(makeTestPNG(t, 800, 600, false), normalizationImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	same, err := projectImageForRequest(norm, defaultRequestImageLimits)
	if err != nil {
		t.Fatalf("投影失败: %v", err)
	}
	if !bytes.Equal(same.Data, norm.Data) {
		t.Errorf("预算内应原样复用归一化字节")
	}
	small, err := projectImageForRequest(norm, deepseekRequestImageLimits)
	if err != nil {
		t.Fatalf("投影失败: %v", err)
	}
	if int64(small.Width)*int64(small.Height) > deepseekRequestImageLimits.MaxPixels {
		t.Errorf("投影后像素 %dx%d 超预算", small.Width, small.Height)
	}
	if int64(len(small.Data)) > deepseekRequestImageLimits.MaxBytes {
		t.Errorf("投影后字节 %d 超预算", len(small.Data))
	}
}

// TestOffloadImagesOldestFirst 超预算时最老图片优先替换为占位（对齐 dsh 量化算法）。
func TestOffloadImagesOldestFirst(t *testing.T) {
	const mib = 1 << 20
	// 每张归一化 3MiB → base64 表示恰为 4MiB；6 张共 24MiB > 20MiB 预算
	img := func(i int) ImagePart {
		return ImagePart{Ref: strings.Repeat("a", 63) + string(rune('0'+i)), Bytes: 3 * mib, Width: 100, Height: 100}
	}
	msgs := make([]Message, 6)
	for i := range msgs {
		msgs[i] = Message{Role: RoleUser, Content: "m" + itoa64(int64(i)), Images: []ImagePart{img(i)}}
	}
	// 字节预算：excess 4MiB → 量化 10MiB → 移除最老 3 张
	policy := imageOffloadPolicy{MaxImages: 600, MaxBytes: 20 * mib, CountQuantum: 1, ByteQuantum: 10 * mib}
	out := offloadImages(msgs, policy)
	for i := 0; i < 3; i++ {
		if len(out[i].Images) != 0 {
			t.Errorf("第 %d 条（最老）应被 offload，得 %d 张", i, len(out[i].Images))
		}
		if !strings.Contains(out[i].Content, "image omitted to fit request image limits") {
			t.Errorf("第 %d 条应含 offload 占位文本: %s", i, out[i].Content)
		}
	}
	for i := 3; i < 6; i++ {
		if len(out[i].Images) != 1 {
			t.Errorf("第 %d 条（较新）应保留，得 %d 张", i, len(out[i].Images))
		}
	}
	// 张数预算：MaxImages=2 → 移除最老 4 张
	byCount := offloadImages(msgs, imageOffloadPolicy{MaxImages: 2, CountQuantum: 1})
	if len(byCount[3].Images) != 0 || len(byCount[4].Images) != 1 || len(byCount[5].Images) != 1 {
		t.Errorf("张数预算裁剪错误: %d/%d/%d",
			len(byCount[3].Images), len(byCount[4].Images), len(byCount[5].Images))
	}
	// 预算内不改动
	if same := offloadImages(msgs, imageOffloadPolicy{MaxImages: 600, MaxBytes: 100 * mib, ByteQuantum: 10 * mib}); len(same[0].Images) != 1 {
		t.Errorf("预算内不应裁剪")
	}
}

// TestHydrateImagesProjectsRoute 多模态路由：引用投影为路由预算内 data URL + 句柄。
func TestHydrateImagesProjectsRoute(t *testing.T) {
	dir := t.TempDir()
	raw := makeTestPNG(t, 1200, 900, false)
	norm, err := normalizeImageBytes(raw, normalizationImageLimits)
	if err != nil {
		t.Fatalf("归一化失败: %v", err)
	}
	ref, _, err := storeAttachment(attachmentDir(dir), norm.Data, norm.MediaType)
	if err != nil {
		t.Fatalf("落盘失败: %v", err)
	}
	part := ImagePart{
		Ref: ref, MimeType: norm.MediaType, Path: "shot.png",
		Width: norm.Width, Height: norm.Height, Bytes: int64(len(norm.Data)),
		OrigWidth: 1200, OrigHeight: 900,
	}
	l := &Loop{WorkspaceRoot: dir, Provider: &OpenAIProvider{Multimodal: true, Model: "deepseek-v4"}}
	msgs := []Message{{Role: RoleUser, Content: toolResultImageText, Images: []ImagePart{part}}}
	out := l.hydrateImages(msgs)
	if len(out[0].Images) != 1 {
		t.Fatalf("应保留 1 张图，得 %d", len(out[0].Images))
	}
	got := out[0].Images[0]
	if !strings.HasPrefix(got.Data, "data:image/") {
		t.Errorf("Data 应为 data URL: %.40s", got.Data)
	}
	if !strings.Contains(got.Handle, "request preview ") {
		t.Errorf("句柄应含 request preview: %s", got.Handle)
	}
	if !strings.Contains(got.Handle, "Normalized copy (read-only") {
		t.Errorf("句柄应含归一化只读副本说明: %s", got.Handle)
	}
	// 原始消息不被污染（持久化只存引用）
	if msgs[0].Images[0].Data != "" || msgs[0].Images[0].Handle != "" {
		t.Errorf("hydrate 不应污染原消息: %+v", msgs[0].Images[0])
	}
}

// TestHydrateImagesTextOnlyPlaceholder 非多模态路由：图片替换为 text-only 占位。
func TestHydrateImagesTextOnlyPlaceholder(t *testing.T) {
	dir := t.TempDir()
	l := &Loop{WorkspaceRoot: dir, Provider: &OpenAIProvider{Multimodal: false}}
	part := ImagePart{Ref: strings.Repeat("b", 64), MimeType: "image/png", Width: 10, Height: 10, Bytes: 100}
	msgs := []Message{{Role: RoleUser, Content: toolResultImageText, Images: []ImagePart{part}}}
	out := l.hydrateImages(msgs)
	if len(out[0].Images) != 0 {
		t.Errorf("文本模型不应发送图片: %d", len(out[0].Images))
	}
	if !strings.Contains(out[0].Content, "[image omitted because this model accepts text only; attachment sha256:") {
		t.Errorf("应含 text-only 占位: %s", out[0].Content)
	}
	if strings.Contains(out[0].Content, toolResultImageText) {
		t.Errorf("文本模型占位不应保留工具结果前缀: %s", out[0].Content)
	}
}

// TestFormatImageReadOutput dsh 风格信封 + 降采样坐标提示。
func TestFormatImageReadOutput(t *testing.T) {
	part := ImagePart{MimeType: "image/jpeg", Width: 800, Height: 600, Bytes: 12345, OrigWidth: 1600, OrigHeight: 1200}
	out := formatImageReadOutput("shot.png", part)
	for _, want := range []string{"<path>shot.png</path>", "<type>image</type>", "image/jpeg image, 800x600 px, 12345 bytes", "multiply coordinates by 2.00"} {
		if !strings.Contains(out, want) {
			t.Errorf("信封缺 %q:\n%s", want, out)
		}
	}
	// 未降采样 → 无提示
	plain := formatImageReadOutput("a.png", ImagePart{MimeType: "image/png", Width: 4, Height: 4, Bytes: 10, OrigWidth: 4, OrigHeight: 4})
	if strings.Contains(plain, "downscaled") {
		t.Errorf("未降采样不应提示: %s", plain)
	}
}

// TestAttachmentRoundTrip 内容寻址存储：同内容复用同一引用、读取一致。
func TestAttachmentRoundTrip(t *testing.T) {
	dir := attachmentDir(t.TempDir())
	data := makeTestPNG(t, 16, 16, false)
	ref1, path1, err := storeAttachment(dir, data, "image/png")
	if err != nil {
		t.Fatalf("落盘失败: %v", err)
	}
	ref2, path2, err := storeAttachment(dir, data, "image/png")
	if err != nil {
		t.Fatalf("重复落盘失败: %v", err)
	}
	if ref1 != ref2 || path1 != path2 {
		t.Errorf("同内容应复用同一附件: %s/%s", ref1, ref2)
	}
	got, err := loadAttachment(dir, ref1, "image/png")
	if err != nil || !bytes.Equal(got, data) {
		t.Errorf("读取附件失败: %v", err)
	}
	if _, err := loadAttachment(dir, strings.Repeat("0", 64), "image/png"); err == nil {
		t.Errorf("不存在的引用应报错")
	}
}
