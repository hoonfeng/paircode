// ═══════════════════════════════════════════════════════════════
// image_wire.go — 图片请求装配（对齐 dsh 请求侧语义）
//
// ★ 对齐参考（2026-09）：
//
//	· packages/llm/llm/src/content.ts — 句柄文本 requestImageHandleText /
//	  text-only 占位 textOnlyImageText / offload 占位 offloadedImageText /
//	  offload 前缀算法 offloadedImagePrefixCount
//	· packages/llm/llm-deepseek/src/serialize.ts — 工具结果图片合并成一条 user 消息
//	  （TOOL_RESULT_IMAGE_TEXT = "Attached image(s) from tool result:"）
//	· packages/llm/llm-deepseek/src/request-pricing.ts / adapter.ts — 默认预算
//	  （600 张 / 20MiB 内联 / 10MiB 字节量化）
//
// ★ 与 dsh 的映射：dsh 的 ImageBlock 在装配时换成 request preview（按路由预算），
//
//	模型看到「句柄 text 块 + image 块」交错序列。Go 侧 Message 结构是
//	「一个 Content + Images 列表」，故把句柄文本放在 ImagePart.Handle，
//	由各 provider 在 image 块前插入对应 text 块——语义等价。
//
// ═══════════════════════════════════════════════════════════════
package agent

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

// ── dsh 对齐常量 ──

// toolResultImageText 工具结果图片消息固定前缀（dsh TOOL_RESULT_IMAGE_TEXT）。
const toolResultImageText = "Attached image(s) from tool result:"

const (
	// defaultMaxImagesPerRequest 单请求图片张数上限（dsh DEFAULT_MAX_IMAGES_PER_REQUEST）。
	defaultMaxImagesPerRequest = 600
	// defaultMaxInlineImageBytes 单请求内联图片字节上限（base64 表示，dsh DEFAULT_MAX_INLINE_REQUEST_IMAGE_BYTES）。
	defaultMaxInlineImageBytes = 20 << 20
	// defaultInlineImageByteQuantum 超限后按量化步长移除最老图片（dsh DEFAULT_INLINE_IMAGE_OFFLOAD_BYTE_QUANTUM）。
	defaultInlineImageByteQuantum = 10 << 20
)

// imageOffloadPolicy 请求内图片裁剪策略（对齐 dsh RequestImageOffloadPolicy）。
type imageOffloadPolicy struct {
	MaxImages    int
	MaxBytes     int64
	CountQuantum int
	ByteQuantum  int64
}

// defaultImageOffloadPolicy 默认裁剪策略（对齐 llm-deepseek 默认值）。
func defaultImageOffloadPolicy() imageOffloadPolicy {
	return imageOffloadPolicy{
		MaxImages:    defaultMaxImagesPerRequest,
		MaxBytes:     defaultMaxInlineImageBytes,
		CountQuantum: 1,
		ByteQuantum:  defaultInlineImageByteQuantum,
	}
}

// ── 模型可见文本（逐字对齐 dsh content.ts）──

// imageAttachmentID 附件 id（sha256:hex）。
func imageAttachmentID(ref string) string {
	if ref == "" {
		return "sha256:unknown"
	}
	return "sha256:" + ref
}

// imageIdentity 模型可见身份（dsh imageIdentity）：有名称 → `"name" (sha256:...)`。
func imageIdentity(ref, name string) string {
	id := imageAttachmentID(ref)
	if strings.TrimSpace(name) == "" {
		return id
	}
	return fmt.Sprintf("%q (%s)", name, id)
}

// imageExtension 媒体类型 → 带点扩展名（dsh extension）。
func imageExtension(mediaType string) string {
	switch mediaType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

// normalizedAccessText 归一化只读副本说明（dsh normalizedAccessText）。
func normalizedAccessText(accessPath, mediaType string, width, height int) string {
	return fmt.Sprintf(" Normalized copy (read-only; may be resized or re-encoded): %q (%dx%dpx, %s). Source dimensions, format, and byte size may differ. Copy to a writable path ending in %s before editing.",
		accessPath, width, height, mediaType, imageExtension(mediaType))
}

// textOnlyImageText 文本模型下的图片占位（dsh textOnlyImageText）。
func textOnlyImageText(ref string) string {
	digest := ref
	if len(digest) > 8 {
		digest = digest[:8]
	}
	if digest == "" {
		digest = "unknown"
	}
	return fmt.Sprintf("[image omitted because this model accepts text only; attachment sha256:%s]", digest)
}

// offloadedImageText 超预算被省略的图片占位（dsh offloadedImageText）。
func offloadedImageText(ref, name, accessPath, mediaType string, width, height int) string {
	identity := fmt.Sprintf("image omitted to fit request image limits; %s.", imageIdentity(ref, name))
	if accessPath == "" {
		return "[" + identity + " No local normalized image path is available; ask the user to attach it again if needed.]"
	}
	return "[" + identity + normalizedAccessText(accessPath, mediaType, width, height) + "]"
}

// imageHandleText 模型可见句柄（dsh requestImageHandleText）：
// 请求预览尺寸 + 归一化只读副本路径（有则给出）。
func imageHandleText(ref, name string, previewW, previewH int, accessPath, mediaType string, normW, normH int) string {
	preview := fmt.Sprintf("Image %s; request preview %dx%dpx.", imageIdentity(ref, name), previewW, previewH)
	if accessPath == "" {
		return preview + " It may be resized or re-encoded; source dimensions, format, and byte size may differ."
	}
	return preview + normalizedAccessText(accessPath, mediaType, normW, normH)
}

// formatImageReadOutput 图片读取的模型可见信封（对齐 dsh formatImageReadOutput）：
// 降采样时给出源尺寸与坐标换算倍数（同一倍数时只给一个）。
func formatImageReadOutput(displayPath string, img ImagePart) string {
	scaled := ""
	if img.OrigWidth > 0 && img.OrigHeight > 0 && img.Width > 0 && img.Height > 0 &&
		(img.OrigWidth != img.Width || img.OrigHeight != img.Height) {
		x := float64(img.OrigWidth) / float64(img.Width)
		y := float64(img.OrigHeight) / float64(img.Height)
		xs, ys := fmt.Sprintf("%.2f", x), fmt.Sprintf("%.2f", y)
		advice := "multiply coordinates by " + xs
		if xs != ys {
			advice = "multiply x coordinates by " + xs + " and y coordinates by " + ys
		}
		scaled = fmt.Sprintf(" (downscaled from %dx%d px; %s to locate features in the original file)",
			img.OrigWidth, img.OrigHeight, advice)
	}
	return fmt.Sprintf("<path>%s</path>\n<type>image</type>\n<content>\n%s image, %dx%d px, %d bytes%s\n</content>",
		displayPath, img.MimeType, img.Width, img.Height, img.Bytes, scaled)
}

// imageHandleBlock 句柄文本块内容（对齐 dsh imageHandle：前面已有内容时前置换行）。
func imageHandleBlock(handle string, precededByContent bool) string {
	if precededByContent {
		return "\n" + handle
	}
	return handle
}

// ── offload ──

// imageRepresentedBytes 图片在请求中的表示字节数：已内联用 data URL 长度，否则按归一化字节估 base64 长度。
func imageRepresentedBytes(img ImagePart) int {
	if img.Data != "" {
		return len(img.Data)
	}
	if img.Bytes <= 0 {
		return 0
	}
	return int((img.Bytes + 2) / 3 * 4)
}

// messagesHaveImages 消息序列中是否含图片。
func messagesHaveImages(msgs []Message) bool {
	for _, m := range msgs {
		if len(m.Images) > 0 {
			return true
		}
	}
	return false
}

// offloadImages 按 dsh offloadedImagePrefixCount 语义裁剪最老图片：
// 统计请求表示字节 → 超出张数/字节预算 → 量化后从最老开始移除并替换为占位文本。
func offloadImages(msgs []Message, policy imageOffloadPolicy) []Message {
	type entry struct {
		msg   int
		bytes int64
	}
	var all []entry
	var total int64
	for i, m := range msgs {
		for _, img := range m.Images {
			n := int64(imageRepresentedBytes(img))
			all = append(all, entry{msg: i, bytes: n})
			total += n
		}
	}
	if len(all) == 0 {
		return msgs
	}
	excessCount := 0
	if policy.MaxImages > 0 && len(all) > policy.MaxImages {
		excessCount = len(all) - policy.MaxImages
	}
	var excessBytes int64
	if policy.MaxBytes > 0 && total > policy.MaxBytes {
		excessBytes = total - policy.MaxBytes
	}
	if excessCount == 0 && excessBytes == 0 {
		return msgs
	}
	countQuantum := policy.CountQuantum
	if countQuantum < 1 {
		countQuantum = 1
	}
	byteQuantum := policy.ByteQuantum
	if byteQuantum < 1 {
		byteQuantum = 1
	}
	removeCount := 0
	if excessCount > 0 {
		removeCount = ((excessCount + countQuantum - 1) / countQuantum) * countQuantum
	}
	var removeBytes int64
	if excessBytes > 0 {
		removeBytes = ((excessBytes + byteQuantum - 1) / byteQuantum) * byteQuantum
	}
	count := 0
	var removedBytes int64
	for _, e := range all {
		byteTargetMet := removeBytes == 0 ||
			(byteQuantum == 1 && removedBytes >= removeBytes) ||
			(byteQuantum != 1 && removedBytes > removeBytes)
		if count >= removeCount && byteTargetMet {
			break
		}
		removedBytes += e.bytes
		count++
	}
	if count == 0 {
		return msgs
	}

	out := make([]Message, len(msgs))
	copy(out, msgs)
	remaining := count
	for i := range out {
		if remaining <= 0 {
			break
		}
		if len(out[i].Images) == 0 {
			continue
		}
		keep := make([]ImagePart, 0, len(out[i].Images))
		var placeholders []string
		for _, img := range out[i].Images {
			if remaining > 0 {
				placeholders = append(placeholders, offloadedImageText(
					img.Ref, img.Path, "", img.MimeType, img.Width, img.Height))
				remaining--
				continue
			}
			keep = append(keep, img)
		}
		m := out[i]
		m.Images = keep
		if len(placeholders) > 0 {
			m.Content = joinContentLines(m.Content, strings.Join(placeholders, "\n"))
		}
		out[i] = m
	}
	return out
}

// joinContentLines 在既有内容后追加一行文本（空内容时直接取追加文本）。
func joinContentLines(base, extra string) string {
	if strings.TrimSpace(extra) == "" {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return extra
	}
	return base + "\n" + extra
}

// takeCallImages 取走某次工具调用挂载的图片（幂等：取后清空）。
func (l *Loop) takeCallImages(callID string) []ImagePart {
	if callID == "" {
		return nil
	}
	l.imageMu.Lock()
	defer l.imageMu.Unlock()
	imgs := l.imageByCall[callID]
	if len(imgs) == 0 {
		return nil
	}
	delete(l.imageByCall, callID)
	return imgs
}

// attachToolResultImages 把尚未认领的工具结果图片挂到对应 tool 消息上：
// Go 循环在追加消息时已直接挂载；JS 循环等由外部组装 tool 消息的路径走这里。
func (l *Loop) attachToolResultImages(msgs []Message) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	l.imageMu.Lock()
	pending := len(l.imageByCall)
	l.imageMu.Unlock()
	if pending == 0 {
		return msgs
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		if out[i].Role != RoleTool || out[i].ToolCallID == "" || len(out[i].Images) > 0 {
			continue
		}
		if imgs := l.takeCallImages(out[i].ToolCallID); len(imgs) > 0 {
			out[i].Images = imgs
		}
	}
	return out
}

// extractToolImages 把 tool 消息携带的图片提取出来，合并成一条 user 消息
// （对齐 dsh serializeMessagesWithImages：连续工具结果的图片共享一条 user 消息，
// 前缀 "Attached image(s) from tool result:"，插在工具结果之后、下一条消息之前）。
func extractToolImages(msgs []Message) []Message {
	has := false
	for _, m := range msgs {
		if m.Role == RoleTool && len(m.Images) > 0 {
			has = true
			break
		}
	}
	if !has {
		return msgs
	}
	out := make([]Message, 0, len(msgs)+1)
	var pending []ImagePart
	flush := func() {
		if len(pending) == 0 {
			return
		}
		out = append(out, Message{Role: RoleUser, Content: toolResultImageText, Images: pending})
		pending = nil
	}
	for _, m := range msgs {
		if m.Role == RoleTool {
			if len(m.Images) > 0 {
				pending = append(pending, m.Images...)
				m.Images = nil // Message 为值类型，置空不影响原消息
			}
			out = append(out, m)
			continue
		}
		flush()
		out = append(out, m)
	}
	flush()
	return out
}

// ── 请求装配 ──

// requestImageLimits 当前路由的请求图片预算：DeepSeek 路由 640k 像素 / 1MiB，其他 2048² / 1MiB。
func (l *Loop) requestImageLimits() ImageLimits {
	if strings.Contains(strings.ToLower(providerModelName(l.getProvider())), "deepseek") {
		return deepseekRequestImageLimits
	}
	return defaultRequestImageLimits
}

// providerModelName 读取 Provider 的模型名（各 Provider 均有 Model 字段）。
func providerModelName(p Provider) string {
	switch v := p.(type) {
	case *OpenAIProvider:
		return v.Model
	case *AnthropicProvider:
		return v.Model
	case *ResponsesProvider:
		return v.Model
	}
	if m, ok := p.(interface{ ModelName() string }); ok {
		return m.ModelName()
	}
	return ""
}

// hydrateImages 装配请求消息中的图片（对齐 dsh request-image + serializeMessagesWithImages）：
//  1. 按预算 offload 最老图片（占位文本替换）
//  2. 多模态路由：Ref → 归一化附件 → 路由预算投影 → base64 data URL + 模型可见句柄
//  3. 非多模态路由：图片替换为 text-only 占位
//
// 返回深拷贝消息（Images 切片重新分配），绝不污染持久化历史。
func (l *Loop) hydrateImages(msgs []Message) []Message {
	if !messagesHaveImages(msgs) {
		return msgs
	}
	trimmed := offloadImages(msgs, defaultImageOffloadPolicy())
	if !messagesHaveImages(trimmed) {
		return trimmed
	}
	multimodal := l.supportsMultimodal()
	dir := attachmentDir(l.WorkspaceRoot)
	lim := l.requestImageLimits()

	out := make([]Message, len(trimmed))
	copy(out, trimmed)
	for i := range out {
		if len(out[i].Images) == 0 {
			continue
		}
		imgs := make([]ImagePart, len(out[i].Images))
		copy(imgs, out[i].Images)
		m := out[i]
		if !multimodal {
			placeholders := make([]string, 0, len(imgs))
			for _, img := range imgs {
				placeholders = append(placeholders, textOnlyImageText(img.Ref))
			}
			m.Images = nil
			joined := strings.Join(placeholders, "\n")
			if strings.TrimSpace(m.Content) == toolResultImageText {
				m.Content = joined
			} else {
				m.Content = joinContentLines(m.Content, joined)
			}
			out[i] = m
			continue
		}
		for j := range imgs {
			imgs[j] = l.projectImagePart(imgs[j], dir, lim)
		}
		m.Images = imgs
		out[i] = m
	}
	return out
}

// projectImagePart 单张图片投影：引用 → 路由预算内 data URL + 句柄。
// 附件缺失/投影失败 → 占位文本（模型可据此请求用户重新附加）。
func (l *Loop) projectImagePart(img ImagePart, dir string, lim ImageLimits) ImagePart {
	if img.Ref == "" {
		return img // 旧格式（内联 base64）：保持原样，兼容历史会话
	}
	accessPath := img.attachmentPath(dir)
	normW, normH := img.Width, img.Height
	if url, ok := l.cachedImageURL(img.Ref, lim); ok {
		img.Data = url
		img.Handle = imageHandleText(img.Ref, img.Path, img.Width, img.Height, accessPath, img.MimeType, normW, normH)
		return img
	}
	data, err := loadAttachment(dir, img.Ref, img.MimeType)
	if err != nil {
		img.Data = ""
		img.Handle = offloadedImageText(img.Ref, img.Path, accessPath, img.MimeType, img.Width, img.Height)
		return img
	}
	norm := encodedImage{Data: data, MediaType: img.MimeType, Width: img.Width, Height: img.Height}
	proj, projErr := projectAttachmentForRoute(dir, img.Ref, norm, lim)
	if projErr != nil {
		img.Data = ""
		img.Handle = offloadedImageText(img.Ref, img.Path, accessPath, img.MimeType, img.Width, img.Height)
		return img
	}
	url := "data:" + proj.MediaType + ";base64," + base64.StdEncoding.EncodeToString(proj.Data)
	img.Data = url
	img.Handle = imageHandleText(img.Ref, img.Path, proj.Width, proj.Height, accessPath, proj.MediaType, normW, normH)
	l.cacheImageURL(img.Ref, lim, url)
	return img
}

// attachmentPath 归一化附件的只读绝对路径（模型可用文件工具读取）。
func (p ImagePart) attachmentPath(dir string) string {
	if p.Ref == "" || dir == "" {
		return ""
	}
	return filepath.Join(dir, p.Ref+imageExtension(p.MimeType))
}

// imageURLCacheKey 投影缓存键（附件 id + 路由预算）。
func imageURLCacheKey(ref string, lim ImageLimits) string {
	return fmt.Sprintf("%s|%d|%d", ref, lim.MaxPixels, lim.MaxBytes)
}

// cachedImageURL 读投影缓存（同图在历史中重复出现时避免反复读盘/编码）。
func (l *Loop) cachedImageURL(ref string, lim ImageLimits) (string, bool) {
	key := imageURLCacheKey(ref, lim)
	l.imageCacheMu.Lock()
	defer l.imageCacheMu.Unlock()
	if l.imageURLCache == nil {
		return "", false
	}
	url, ok := l.imageURLCache[key]
	return url, ok
}

// cacheImageURL 写投影缓存（上限 64 条，超限清空重建）。
func (l *Loop) cacheImageURL(ref string, lim ImageLimits, url string) {
	key := imageURLCacheKey(ref, lim)
	l.imageCacheMu.Lock()
	defer l.imageCacheMu.Unlock()
	if l.imageURLCache == nil {
		l.imageURLCache = map[string]string{}
	}
	if len(l.imageURLCache) >= 64 {
		l.imageURLCache = map[string]string{}
	}
	l.imageURLCache[key] = url
}
