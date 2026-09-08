// ═══════════════════════════════════════════════════════════════
// image_pipeline.go — 图片准入/归一化/请求投影管线
//
// ★ 对齐参考（2026-09）：deepseek-harness/packages/attachment/attachment-local
//   image.ts（探测）· normalization.ts（归一化）· request-image.ts（请求变体）
//   · encoding.ts（质量阶梯 IMAGE_ENCODING_QUALITIES=[85,75,60]）
//
// ★ 三级限制（dsh 语义）：
//   1) 准入 admission        单图 ≤20MiB、总像素 ≤64M、长边 ≤8192
//   2) 归一化 normalization  总像素 ≤2048×2048、长边 ≤8192、字节 ≤4MiB（持久化、provider 无关）
//   3) 请求 request          按路由预算再投影（DeepSeek 640k 像素 / 1MiB；其他 2048² / 1MiB）
//
// ★ 透传规则（对齐 canPassThroughNormalization）：非 GIF、非动画、无元数据、
//   8bit、sRGB、字节/像素/长边均在限内 → 原样透传，不做任何重编码。
//
// ★ 偏离记录：dsh 用 sharp 的 WebP 编码器保留透明通道；Go 标准库无 WebP 编码器
//   （x/image/webp 仅解码）→ 有 alpha 时用 PNG 无损编码（字节超目标时逐级降采样）。
//   不透明图仍走 JPEG 质量阶梯 [85,75,60]，与 dsh 一致。
// ═══════════════════════════════════════════════════════════════
package agent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp" // 注册 WebP 解码器（image.Decode）+ 尺寸探测
)

// ImageLimits 一组图片尺寸/字节限制（对齐 dsh 的三级限制语义）。
type ImageLimits struct {
	MaxPixels    int64 // 总像素预算（宽×高）；超出按比例降采样
	MaxDimension int   // 长边上限（像素）；在像素预算之后应用，约束极端宽高比
	MaxBytes     int64 // 编码字节目标；质量阶梯逐个试，全超则取最小
}

var (
	// admissionImageLimits 准入限制：单图 ≤20MiB / 64M 像素 / 长边 8192。
	admissionImageLimits = ImageLimits{MaxPixels: 64_000_000, MaxDimension: 8192, MaxBytes: 20 << 20}
	// normalizationImageLimits 归一化限制（持久化、provider 无关）：2048×2048 / 8192 / 4MiB。
	normalizationImageLimits = ImageLimits{MaxPixels: 2048 * 2048, MaxDimension: 8192, MaxBytes: 4 << 20}
	// deepseekRequestImageLimits DeepSeek 路由请求预算：640k 像素 / 1MiB
	// （对齐 llm-deepseek DEFAULT_REQUEST_IMAGE_PIXEL_BUDGET / _MAX_BYTES）。
	deepseekRequestImageLimits = ImageLimits{MaxPixels: 640_000, MaxDimension: 8192, MaxBytes: 1 << 20}
	// defaultRequestImageLimits 其他路由请求预算：2048×2048 / 1MiB
	// （对齐 llm-pi-ai DEFAULT_REQUEST_IMAGE_PIXEL_BUDGET / _MAX_BYTES）。
	defaultRequestImageLimits = ImageLimits{MaxPixels: 2048 * 2048, MaxDimension: 8192, MaxBytes: 1 << 20}
)

// imageJPEGQualities JPEG 质量阶梯（对齐 dsh IMAGE_ENCODING_QUALITIES）。
var imageJPEGQualities = []int{85, 75, 60}

// imageFacts 一张图片的探测事实（对齐 dsh DetectedImage 的可用子集）。
type imageFacts struct {
	MediaType string
	Width     int
	Height    int
	HasAlpha  bool
	Animated  bool
	Depth     string // uchar | ushort
	Space     string // srgb | other
	Metadata  bool   // 是否携带元数据块（iCCP/tEXt/EXIF/ICC…）
	Bytes     int64
}

// encodedImage 一次编码/投影结果。
type encodedImage struct {
	Data      []byte
	MediaType string
	Width     int
	Height    int
	HasAlpha  bool
}

// sniffImageMediaType 按文件签名嗅探媒体类型（不支持的类型返回空串）。
func sniffImageMediaType(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "image/gif"
	case len(data) >= 12 && bytes.HasPrefix(data, []byte("RIFF")) && string(data[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}

// decodeImageBytes 解码图片并探测事实（返回解码像素，供缩放复用）。
func decodeImageBytes(data []byte) (imageFacts, image.Image, error) {
	if len(data) == 0 {
		return imageFacts{}, nil, fmt.Errorf("图片内容为空")
	}
	mt := sniffImageMediaType(data)
	if mt == "" {
		return imageFacts{}, nil, fmt.Errorf("不支持的图片格式（仅支持 PNG/JPEG/GIF/WebP）")
	}
	var facts imageFacts
	var err error
	switch mt {
	case "image/png":
		facts, err = probePNG(data)
	case "image/jpeg":
		facts, err = probeJPEG(data)
	case "image/gif":
		facts, err = probeGIF(data)
	case "image/webp":
		facts, err = probeWebP(data)
	}
	if err != nil {
		return imageFacts{}, nil, err
	}
	facts.Bytes = int64(len(data))
	img, _, decErr := image.Decode(bytes.NewReader(data))
	if decErr != nil {
		return imageFacts{}, nil, fmt.Errorf("图片解码失败：%w", decErr)
	}
	if facts.Width <= 0 || facts.Height <= 0 {
		b := img.Bounds()
		facts.Width, facts.Height = b.Dx(), b.Dy()
	}
	return facts, img, nil
}

// probePNG 解析 PNG 头与块：IHDR 提供宽高/位深/颜色类型，块扫描提供透明与元数据事实。
func probePNG(data []byte) (imageFacts, error) {
	if len(data) < 33 {
		return imageFacts{}, fmt.Errorf("PNG 文件不完整")
	}
	ihdr := data[8:33]
	if string(ihdr[4:8]) != "IHDR" {
		return imageFacts{}, fmt.Errorf("PNG IHDR 缺失")
	}
	facts := imageFacts{
		MediaType: "image/png",
		Width:     int(binary.BigEndian.Uint32(ihdr[8:12])),
		Height:    int(binary.BigEndian.Uint32(ihdr[12:16])),
		Depth:     "uchar",
		Space:     "srgb",
	}
	if ihdr[16] == 16 {
		facts.Depth = "ushort"
	}
	colorType := ihdr[17]
	facts.HasAlpha = colorType == 4 || colorType == 6
	for pos := 8; pos+12 <= len(data); {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		if length < 0 || pos+12+length > len(data) {
			break
		}
		switch string(data[pos+4 : pos+8]) {
		case "tRNS":
			facts.HasAlpha = true
		case "iCCP":
			facts.Metadata = true
			facts.Space = "other" // 自定义 ICC 配置 → 需重编码为 sRGB
		case "tEXt", "zTXt", "iTXt", "eXIf", "gAMA", "cHRM":
			facts.Metadata = true
		case "IEND":
			pos = len(data)
			continue
		}
		pos += 12 + length
	}
	return facts, nil
}

// probeJPEG 解析 JPEG 尺寸与 APP1(EXIF)/APP2(ICC) 元数据事实。
func probeJPEG(data []byte) (imageFacts, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return imageFacts{}, fmt.Errorf("JPEG 解析失败：%w", err)
	}
	facts := imageFacts{
		MediaType: "image/jpeg",
		Width:     cfg.Width,
		Height:    cfg.Height,
		Depth:     "uchar",
		Space:     "srgb",
	}
	for pos := 2; pos+4 <= len(data); {
		if data[pos] != 0xFF {
			pos++
			continue
		}
		marker := data[pos+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			pos += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // 进入扫描数据
			break
		}
		length := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if length < 2 || pos+2+length > len(data) {
			break
		}
		switch marker {
		case 0xE1, 0xE2: // APP1 EXIF / APP2 ICC
			facts.Metadata = true
		}
		pos += 2 + length
	}
	return facts, nil
}

// probeGIF 解析 GIF 帧数与透明事实（GIF 一律不透传，见 canPassThroughNormalization）。
func probeGIF(data []byte) (imageFacts, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return imageFacts{}, fmt.Errorf("GIF 解析失败：%w", err)
	}
	facts := imageFacts{
		MediaType: "image/gif",
		Width:     g.Config.Width,
		Height:    g.Config.Height,
		Depth:     "uchar",
		Space:     "srgb",
		Animated:  len(g.Image) > 1,
	}
	for _, pal := range g.Image {
		if pal == nil {
			continue
		}
		for _, c := range pal.Palette {
			if _, _, _, a := c.RGBA(); a < 0xFFFF {
				facts.HasAlpha = true
				break
			}
		}
		if facts.HasAlpha {
			break
		}
	}
	return facts, nil
}

// decodeWebPConfig 读取 WebP 尺寸（x/image/webp 的 DecodeConfig）。
func decodeWebPConfig(data []byte) (image.Config, error) {
	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return image.Config{}, fmt.Errorf("WebP 解析失败：%w", err)
	}
	return cfg, nil
}

// probeWebP 解析 WebP 尺寸与 alpha/动画事实（VP8X 扩展头）。
func probeWebP(data []byte) (imageFacts, error) {
	cfg, err := decodeWebPConfig(data)
	if err != nil {
		return imageFacts{}, err
	}
	facts := imageFacts{
		MediaType: "image/webp",
		Width:     cfg.Width,
		Height:    cfg.Height,
		Depth:     "uchar",
		Space:     "srgb",
	}
	if len(data) >= 21 && string(data[12:16]) == "VP8X" {
		flags := data[20]
		facts.HasAlpha = flags&0x10 != 0
		facts.Animated = flags&0x02 != 0
	}
	return facts, nil
}

// canPassThroughNormalization 是否满足归一化要求（可字节级透传）。
func canPassThroughNormalization(f imageFacts, lim ImageLimits) bool {
	return f.MediaType != "image/gif" &&
		!f.Animated &&
		!f.Metadata &&
		f.Depth == "uchar" &&
		f.Space == "srgb" &&
		(lim.MaxBytes <= 0 || f.Bytes <= lim.MaxBytes) &&
		(lim.MaxPixels <= 0 || int64(f.Width)*int64(f.Height) <= lim.MaxPixels) &&
		(lim.MaxDimension <= 0 || max(f.Width, f.Height) <= lim.MaxDimension)
}

// fitDimensions 计算降采样目标尺寸（对齐 dsh initialDimensions / requestImageDimensions）：
// 先按总像素预算等比缩小，再按长边上限约束，且不放大。
func fitDimensions(w, h int, lim ImageLimits) (int, int) {
	if w <= 0 || h <= 0 {
		return w, h
	}
	fw, fh := float64(w), float64(h)
	if lim.MaxPixels > 0 && int64(w)*int64(h) > lim.MaxPixels {
		s := math.Sqrt(float64(lim.MaxPixels) / (fw * fh))
		fw, fh = fw*s, fh*s
	}
	if lim.MaxDimension > 0 {
		if longest := math.Max(fw, fh); longest > float64(lim.MaxDimension) {
			s := float64(lim.MaxDimension) / longest
			fw, fh = fw*s, fh*s
		}
	}
	nw, nh := int(math.Floor(fw)), int(math.Floor(fh))
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	if nw > w {
		nw = w
	}
	if nh > h {
		nh = h
	}
	return nw, nh
}

// scaleImage 用 CatmullRom 重采样（质量优先，等价 sharp 默认内核）。
func scaleImage(src image.Image, w, h int) image.Image {
	if w <= 0 || h <= 0 {
		return src
	}
	if w == src.Bounds().Dx() && h == src.Bounds().Dy() {
		return src
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

// encodeJPEGBytes 编码 JPEG（质量 q）。
func encodeJPEGBytes(img image.Image, q int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
		return nil, fmt.Errorf("JPEG 编码失败：%w", err)
	}
	return buf.Bytes(), nil
}

// encodePNGBytes 编码 PNG（无损，保留透明通道）。
func encodePNGBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("PNG 编码失败：%w", err)
	}
	return buf.Bytes(), nil
}

// encodeWithinLimit 按质量阶梯编码：返回第一个满足字节目标的候选，全超则返回最小的。
// 有 alpha → PNG（保留透明）；否则 JPEG [85,75,60]。
func encodeWithinLimit(img image.Image, hasAlpha bool, maxBytes int64) (encodedImage, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if hasAlpha {
		data, err := encodePNGBytes(img)
		if err != nil {
			return encodedImage{}, err
		}
		return encodedImage{Data: data, MediaType: "image/png", Width: w, Height: h, HasAlpha: true}, nil
	}
	var smallest encodedImage
	for _, q := range imageJPEGQualities {
		data, err := encodeJPEGBytes(img, q)
		if err != nil {
			return encodedImage{}, err
		}
		cand := encodedImage{Data: data, MediaType: "image/jpeg", Width: w, Height: h}
		if maxBytes <= 0 || int64(len(data)) <= maxBytes {
			return cand, nil
		}
		if smallest.Data == nil || len(data) < len(smallest.Data) {
			smallest = cand
		}
	}
	return smallest, nil
}

// reencodeImage 按限制降采样 + 质量阶梯编码；字节仍超目标时逐级降采样（最多 6 级）。
func reencodeImage(img image.Image, hasAlpha bool, srcW, srcH int, lim ImageLimits) (encodedImage, error) {
	tw, th := fitDimensions(srcW, srcH, lim)
	out, err := encodeWithinLimit(scaleImage(img, tw, th), hasAlpha, lim.MaxBytes)
	if err != nil {
		return encodedImage{}, err
	}
	if lim.MaxBytes <= 0 {
		return out, nil
	}
	for i := 0; i < 6 && int64(len(out.Data)) > lim.MaxBytes && tw > 16 && th > 16; i++ {
		tw, th = max(1, tw*3/4), max(1, th*3/4)
		cand, err := encodeWithinLimit(scaleImage(img, tw, th), hasAlpha, lim.MaxBytes)
		if err != nil {
			return encodedImage{}, err
		}
		if len(cand.Data) < len(out.Data) {
			out = cand
		}
		if int64(len(cand.Data)) <= lim.MaxBytes {
			return cand, nil
		}
	}
	return out, nil
}

// normalizeImageBytes 归一化图片（对齐 dsh normalizeImage）：
//   - 已满足限制 → 字节级透传
//   - 否则 → 按像素预算/长边降采样 + 质量阶梯编码
func normalizeImageBytes(data []byte, lim ImageLimits) (encodedImage, error) {
	facts, img, err := decodeImageBytes(data)
	if err != nil {
		return encodedImage{}, err
	}
	if canPassThroughNormalization(facts, lim) {
		return encodedImage{
			Data:      data,
			MediaType: facts.MediaType,
			Width:     facts.Width,
			Height:    facts.Height,
			HasAlpha:  facts.HasAlpha,
		}, nil
	}
	return reencodeImage(img, facts.HasAlpha, facts.Width, facts.Height, lim)
}

// projectImageForRequest 按路由预算投影请求变体（对齐 dsh createRequestImage）：
// 尺寸已在预算内且字节 ≤ 目标 → 原样；否则解码后降采样 + 重编码。
func projectImageForRequest(norm encodedImage, lim ImageLimits) (encodedImage, error) {
	inPixels := lim.MaxPixels <= 0 || int64(norm.Width)*int64(norm.Height) <= lim.MaxPixels
	inBytes := lim.MaxBytes <= 0 || int64(len(norm.Data)) <= lim.MaxBytes
	if inPixels && inBytes {
		return norm, nil
	}
	facts, img, err := decodeImageBytes(norm.Data)
	if err != nil {
		return encodedImage{}, err
	}
	return reencodeImage(img, facts.HasAlpha || norm.HasAlpha, norm.Width, norm.Height, lim)
}
