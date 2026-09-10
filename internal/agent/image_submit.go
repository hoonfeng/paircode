// ═══════════════════════════════════════════════════════════════
// image_submit.go — 图片提交给 LLM 视觉识别（submit_image 工具支持）
//
// ★ 背景（2026-08-22）：agent 工作（测试 UI/截图/测试产出图片）时，图片
//
//	只落在磁盘（screenshots/ 等），LLM 永远"看不到"——agent 只能用本地
//	工具（DOM 分析/文本）猜测画面内容。本机制让工具能显式把图片随下一轮
//	LLM 请求一起发送（OpenAI 兼容 image_url 块），LLM 直接看图。
//
// ★ 协议：工具结果以标记行开头 → __SUBMIT_IMAGE__:{"kind":"submit_image",
//
//	"path":"...","mime":"...","size":123,"prompt":"..."}（磁盘插件 tool-web
//	生成）。本文件解析标记 → 读图 bytes（≤2MiB）→ ImagePart → 挂 pendingImages
//	→ buildCallContext 注入 user 消息（Images 字段）→ Provider.Chat 转块数组。
//	标记从结果文本剥离（净化后给 LLM 的文本不含标记）。
//
// ★ 防护：仅 Provider 多模态时注入（非视觉模型忽略，避免 400）；路径限
//
//	工作区内（resolvePath 越界拦截）；单图 ≤2MiB（超出报错）；会话内路径
//	去重 + 上限（imageInjectedN 防 40+ 张图撑爆上下文）。
//
// ═══════════════════════════════════════════════════════════════
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// imageSubmitMarker 工具结果标记行前缀（磁盘插件 tool-web 生成——原 tool-vision，2026-09-12 并入）。
const imageSubmitMarker = "__SUBMIT_IMAGE__:"

// imageSubmitMeta 标记 JSON 载荷。
type imageSubmitMeta struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Mime   string `json:"mime"`
	Size   int64  `json:"size"`
	Prompt string `json:"prompt"`
}

// imageSubmitMaxBytes 单图准入上限（对齐 dsh DEFAULT_MAX_REQUEST_IMAGE_BYTES = 20MiB）。
const imageSubmitMaxBytes = 20 << 20

// parseImageSubmitResult 解析工具结果中的 __SUBMIT_IMAGE__ 标记（read_image 插件生成）：
//   - 命中：准入检查 → 归一化 → 内容寻址落盘 → ImagePart 挂到 l.imageByCall[callID]；
//     返回剥离标记后的净化文本（含 dsh 风格信封）
//   - 未命中：原样返回（快速路径，无额外开销）
//
// ★ 线程安全：并行工具执行（runParallel/executeParallel）并发调用本函数，
//
//	挂载均在 imageMu 锁内完成。
func (l *Loop) parseImageSubmitResult(result, callID string) string {
	if !strings.HasPrefix(result, imageSubmitMarker) {
		return result
	}
	// 首行为标记行，其余为净化文本
	markLine, rest := splitFirstLine(result)
	metaStr := strings.TrimPrefix(markLine, imageSubmitMarker)
	var meta imageSubmitMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return "错误：图片标记解析失败：" + err.Error() + "\n" + rest
	}
	// ★ 2026-09 对齐 dsh：工具名 read_image（旧名 submit_image 仍接受）
	if (meta.Kind != "submit_image" && meta.Kind != "read_image") || meta.Path == "" {
		return "错误：图片标记无效（kind/path 缺失）\n" + rest
	}
	// 大小上限（标记声明值）
	if meta.Size > imageSubmitMaxBytes {
		return "错误：图片 " + humanBytes(meta.Size) + " 超过 " + humanBytes(imageSubmitMaxBytes) + " 准入限制——请压缩后提交\n" + rest
	}
	// 读图（相对路径/绝对路径均限工作区内；越界由 resolvePath 拦截）
	part, _, err := l.loadImagePart(meta.Path, meta.Mime, meta.Prompt)
	if err != nil {
		return "错误：读取图片失败：" + err.Error() + "\n" + rest
	}
	// 信封文本（对齐 dsh formatImageReadOutput）：模型看到的图片事实与降采样提示
	envelope := formatImageReadOutput(meta.Path, part)
	ns := strings.TrimSpace(rest)
	if ns != "" {
		ns = envelope + "\n" + ns
	} else {
		ns = envelope
	}
	// 挂到该次工具调用（装配时由 attachImagesToToolMsgs 认领到对应 tool 消息，
	// 对齐 dsh：图片作为工具结果的一部分进入持久化历史，请求侧由 offload 裁剪）
	l.imageMu.Lock()
	if l.imageByCall == nil {
		l.imageByCall = map[string][]ImagePart{}
	}
	if callID != "" {
		l.imageByCall[callID] = append(l.imageByCall[callID], part)
	} else {
		ns = "图片未注入（工具调用缺少 call id）：" + meta.Path + "\n" + ns
	}
	l.imageMu.Unlock()
	return ns
}

// loadImagePart 读图 → 准入检查 → 归一化 → 内容寻址落盘 → ImagePart（只带引用与事实）。
// 对齐 dsh：准入（20MiB / 64M 像素 / 长边 8192）→ 归一化（2048² / 4MiB）→ 附件存储；
// 模型实际收到的请求变体在装配时按路由预算投影（见 image_wire.go hydrateImages）。
func (l *Loop) loadImagePart(path, mime, prompt string) (ImagePart, string, error) {
	full, err := resolveImagePath(l.WorkspaceRoot, path)
	if err != nil {
		return ImagePart{}, "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return ImagePart{}, "", err
	}
	// 1) 准入检查（媒体类型以文件签名嗅探为准，声明 mime 仅作兜底）
	if int64(len(data)) > admissionImageLimits.MaxBytes {
		return ImagePart{}, "", fmt.Errorf("图片 %s 超过准入上限 %s",
			humanBytes(int64(len(data))), humanBytes(admissionImageLimits.MaxBytes))
	}
	facts, _, err := decodeImageBytes(data)
	if err != nil {
		return ImagePart{}, "", err
	}
	if admissionImageLimits.MaxPixels > 0 && int64(facts.Width)*int64(facts.Height) > admissionImageLimits.MaxPixels {
		return ImagePart{}, "", fmt.Errorf("图片 %dx%d 像素超过准入上限 %d",
			facts.Width, facts.Height, admissionImageLimits.MaxPixels)
	}
	if admissionImageLimits.MaxDimension > 0 && max(facts.Width, facts.Height) > admissionImageLimits.MaxDimension {
		return ImagePart{}, "", fmt.Errorf("图片长边 %d 超过准入上限 %d",
			max(facts.Width, facts.Height), admissionImageLimits.MaxDimension)
	}
	// 2) 归一化（已满足限制则字节级透传，否则降采样 + 质量阶梯编码）
	norm, err := normalizeImageBytes(data, normalizationImageLimits)
	if err != nil {
		return ImagePart{}, "", err
	}
	// 3) 内容寻址落盘（同内容复用同一附件）
	ref, _, err := storeAttachment(attachmentDir(l.WorkspaceRoot), norm.Data, norm.MediaType)
	if err != nil {
		return ImagePart{}, "", err
	}
	if mime == "" {
		mime = norm.MediaType
	}
	part := ImagePart{
		MimeType:   norm.MediaType,
		Detail:     "auto",
		Ref:        ref,
		Path:       path,
		Width:      norm.Width,
		Height:     norm.Height,
		Bytes:      int64(len(norm.Data)),
		OrigWidth:  facts.Width,
		OrigHeight: facts.Height,
	}
	note := "工具提交了图片（" + path + "，" + humanBytes(int64(len(norm.Data)))
	if facts.Width != norm.Width || facts.Height != norm.Height {
		note += fmt.Sprintf("，已从 %dx%d 归一化", facts.Width, facts.Height)
	}
	note += "）"
	if strings.TrimSpace(prompt) != "" {
		note += "；关注点：" + strings.TrimSpace(prompt)
	}
	return part, note, nil
}

// resolveImagePath 图片路径解析：相对路径拼工作区根；绝对路径走 resolvePath
// 多根越界拦截（工作区内校验）。
func resolveImagePath(primaryRoot, p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("图片路径为空")
	}
	if filepath.IsAbs(p) {
		return resolvePath(primaryRoot, p)
	}
	// 相对路径：拼主根（分隔符归一）
	rel := strings.ReplaceAll(p, "\\", "/")
	rel = strings.TrimLeft(rel, "/")
	root := strings.TrimRight(primaryRoot, "/\\")
	return resolvePath(primaryRoot, root+"/"+rel)
}

// providerSupportsMultimodal 判断 Provider 是否支持多模态（图片输入）：

// providerSupportsMultimodal 判断 Provider 是否支持多模态（图片输入）：
//   - OpenAIProvider 有 Multimodal 字段 → 类型断言读取
//   - 其他 Provider 实现（Mock/Reviewer 等）→ 默认 false（保守：不注入图片）
func providerSupportsMultimodal(p Provider) bool {
	if p == nil {
		return false
	}
	if p, ok := p.(interface{ Multimodal() bool }); ok {
		return p.Multimodal()
	}
	if p, ok := p.(*OpenAIProvider); ok {
		return p.Multimodal
	}
	return false
}

// supportsMultimodal 当前会话 Provider 是否支持多模态（详见 providerSupportsMultimodal）。
func (l *Loop) supportsMultimodal() bool {
	return providerSupportsMultimodal(l.getProvider())
}

// splitFirstLine 拆分首行与剩余。
func splitFirstLine(s string) (first, rest string) {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx], strings.TrimSpace(s[idx+1:])
	}
	return s, ""
}

// humanBytes 字节数友好显示（KB/MB）。
func humanBytes(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1fKB", float64(n)/1024)
	default:
		return fmt.Sprintf("%dB", n)
	}
}
