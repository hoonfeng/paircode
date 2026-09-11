// ═══════════════════════════════════════════════════════════════
// image_store.go — 图片附件内容寻址存储 + 请求变体缓存
//
// ★ 对齐参考（2026-09）：deepseek-harness/packages/attachment/attachment-local
//
//	· 归一化图按内容寻址落盘（sha256 即引用 id）
//	· 请求变体按「附件 id + 路由预算」哈希缓存（request-image.ts requestImageVariantId）
//
// ★ 目录布局（工作区根下）：
//
//	.pair/attachments/<sha256>.<ext>                      归一化附件（provider 无关）
//	.pair/attachments/request-images/<hh>/<sha256>        路由请求变体缓存（无扩展名，同 dsh）
//
// ★ 设计：归一化图只落盘一次，会话消息只存引用（Ref）+ 事实（宽高/字节），
//
//	发送时按当前路由预算投影成 data URL——历史图片不再重复占用消息存储体积。
//
// ═══════════════════════════════════════════════════════════════
package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// attachmentDirName 附件目录（相对工作区根）。
const attachmentDirName = ".pair/attachments"

// attachmentTransformVersion 请求变体缓存标识版本（与 dsh REQUEST_IMAGE_TRANSFORM_VERSION 同义）。
const attachmentTransformVersion = "request-image-go-v1"

// attachmentDir 返回工作区附件目录路径（不创建）。
func attachmentDir(workspaceRoot string) string {
	root := strings.TrimRight(workspaceRoot, "/\\")
	if root == "" {
		root = "."
	}
	return filepath.Join(root, filepath.FromSlash(attachmentDirName))
}

// attachmentRef 内容寻址引用 id（sha256 十六进制）。
func attachmentRef(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// attachmentExt 按媒体类型取文件扩展名。
func attachmentExt(mediaType string) string {
	switch mediaType {
	case "image/jpeg":
		return "jpg"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

// storeAttachment 把归一化字节写入内容寻址文件，返回引用 id 与路径。
// 同内容已存在（且字节数一致）时直接复用，保证幂等与去重。
func storeAttachment(dir string, data []byte, mediaType string) (ref, path string, err error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("附件内容为空")
	}
	ref = attachmentRef(data)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("创建附件目录失败：%w", err)
	}
	path = filepath.Join(dir, ref+"."+attachmentExt(mediaType))
	if st, statErr := os.Stat(path); statErr == nil && st.Size() == int64(len(data)) {
		return ref, path, nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", "", fmt.Errorf("写入附件失败：%w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", "", fmt.Errorf("落盘附件失败：%w", err)
	}
	return ref, path, nil
}

// loadAttachment 按引用 id 读取附件字节；扩展名不确定时按常见后缀依次尝试。
func loadAttachment(dir, ref, mediaType string) ([]byte, error) {
	if ref == "" {
		return nil, fmt.Errorf("附件引用为空")
	}
	candidates := []string{attachmentExt(mediaType), "png", "jpg", "webp", "gif"}
	seen := map[string]bool{}
	for _, ext := range candidates {
		if seen[ext] {
			continue
		}
		seen[ext] = true
		p := filepath.Join(dir, ref+"."+ext)
		if data, err := os.ReadFile(p); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("附件 %s 不存在", ref)
}

// requestVariantPath 请求变体缓存路径：对「附件 id + 路由预算」取 sha256。
func requestVariantPath(dir, ref string, lim ImageLimits) string {
	descriptor := fmt.Sprintf("%s|%s|%d|%d|%d",
		attachmentTransformVersion, ref, lim.MaxPixels, lim.MaxDimension, lim.MaxBytes)
	sum := sha256.Sum256([]byte(descriptor))
	h := hex.EncodeToString(sum[:])
	return filepath.Join(dir, "request-images", h[:2], h)
}

// cachedRequestVariant 读取并校验请求变体缓存：尺寸超预算或字节超目标视为失效。
func cachedRequestVariant(path string, lim ImageLimits) (encodedImage, bool) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return encodedImage{}, false
	}
	facts, _, err := decodeImageBytes(data)
	if err != nil {
		return encodedImage{}, false
	}
	if lim.MaxPixels > 0 && int64(facts.Width)*int64(facts.Height) > lim.MaxPixels {
		return encodedImage{}, false
	}
	if lim.MaxBytes > 0 && facts.Bytes > lim.MaxBytes {
		return encodedImage{}, false
	}
	return encodedImage{
		Data:      data,
		MediaType: facts.MediaType,
		Width:     facts.Width,
		Height:    facts.Height,
		HasAlpha:  facts.HasAlpha,
	}, true
}

// storeRequestVariant 原子写请求变体缓存（并发写同内容无害）。
func storeRequestVariant(path string, img encodedImage) error {
	if len(img.Data) == 0 {
		return fmt.Errorf("请求变体内容为空")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, img.Data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// projectAttachmentForRoute 把归一化附件投影成路由预算内的请求变体（带磁盘缓存）。
// 缓存未命中时执行降采样/重编码并写回缓存。
func projectAttachmentForRoute(dir, ref string, norm encodedImage, lim ImageLimits) (encodedImage, error) {
	if ref != "" {
		cachePath := requestVariantPath(dir, ref, lim)
		if img, ok := cachedRequestVariant(cachePath, lim); ok {
			return img, nil
		}
	}
	img, err := projectImageForRequest(norm, lim)
	if err != nil {
		return encodedImage{}, err
	}
	if ref != "" && len(img.Data) > 0 {
		_ = storeRequestVariant(requestVariantPath(dir, ref, lim), img)
	}
	return img, nil
}
