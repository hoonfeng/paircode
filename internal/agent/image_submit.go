// ═══════════════════════════════════════════════════════════════
// image_submit.go — 图片提交给 LLM 视觉识别（submit_image 工具支持）
//
// ★ 背景（2026-08-22）：agent 工作（测试 UI/截图/测试产出图片）时，图片
//   只落在磁盘（screenshots/ 等），LLM 永远"看不到"——agent 只能用本地
//   工具（DOM 分析/文本）猜测画面内容。本机制让工具能显式把图片随下一轮
//   LLM 请求一起发送（OpenAI 兼容 image_url 块），LLM 直接看图。
//
// ★ 协议：工具结果以标记行开头 → __SUBMIT_IMAGE__:{"kind":"submit_image",
//   "path":"...","mime":"...","size":123,"prompt":"..."}（磁盘插件 tool-vision
//   生成）。本文件解析标记 → 读图 bytes（≤2MiB）→ ImagePart → 挂 pendingImages
//   → buildCallContext 注入 user 消息（Images 字段）→ Provider.Chat 转块数组。
//   标记从结果文本剥离（净化后给 LLM 的文本不含标记）。
//
// ★ 防护：仅 Provider 多模态时注入（非视觉模型忽略，避免 400）；路径限
//   工作区内（resolvePath 越界拦截）；单图 ≤2MiB（超出报错）；会话内路径
//   去重 + 上限（imageInjectedN 防 40+ 张图撑爆上下文）。
// ═══════════════════════════════════════════════════════════════
package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// imageSubmitMarker 工具结果标记行前缀（磁盘插件 tool-vision 生成）。
const imageSubmitMarker = "__SUBMIT_IMAGE__:"

// imageSubmitMeta 标记 JSON 载荷。
type imageSubmitMeta struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Mime   string `json:"mime"`
	Size   int64  `json:"size"`
	Prompt string `json:"prompt"`
}

// imageSubmitMaxBytes 单图上限（DeepSeek 视觉接口建议 ≤2MiB）。
const imageSubmitMaxBytes = 2 * 1024 * 1024

// imageSubmitMaxTotal 会话内可提交图片总数（防上下文爆炸）。
const imageSubmitMaxTotal = 20

// parseImageSubmitResult 解析工具结果中的 __SUBMIT_IMAGE__ 标记：
//   - 命中：读图 → ImagePart → 挂 l.pendingImages；返回剥离标记后的净化文本
//   - 未命中：原样返回（快速路径，无额外开销）
//
// ★ 线程安全：并行工具执行（runParallel/executeParallel）并发调用本函数，
//   挂载与去重均在 imageMu 锁内完成。
func (l *Loop) parseImageSubmitResult(result string) string {
	if !strings.HasPrefix(result, imageSubmitMarker) {
		return result
	}
	// 首行为标记行，其余为净化文本
	markLine, rest := splitFirstLine(result)
	metaStr := strings.TrimPrefix(markLine, imageSubmitMarker)
	var meta imageSubmitMeta
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return "错误：submit_image 标记解析失败：" + err.Error() + "\n" + rest
	}
	if meta.Kind != "submit_image" || meta.Path == "" {
		return "错误：submit_image 标记无效（kind/path 缺失）\n" + rest
	}
	// 大小上限（标记声明值）
	if meta.Size > imageSubmitMaxBytes {
		return "错误：图片 " + humanBytes(meta.Size) + " 超过 2MiB 限制——请压缩后提交\n" + rest
	}
	// 读图（相对路径/绝对路径均限工作区内；越界由 resolvePath 拦截）
	part, note, err := l.loadImagePart(meta.Path, meta.Mime, meta.Prompt)
	if err != nil {
		return "错误：读取图片失败：" + err.Error() + "\n" + rest
	}
	ns := strings.TrimSpace(rest)
	if ns == "" {
		ns = "图片已提交：请查看并分析。"
	}
	// 挂载（锁内：路径去重 + 总数上限）
	l.imageMu.Lock()
	if l.imageInjected == nil {
		l.imageInjected = map[string]bool{}
	}
	key := meta.Path
	if !l.imageInjected[key] && l.imageInjectedN < imageSubmitMaxTotal {
		l.pendingImages = append(l.pendingImages, pendingImage{Part: part, Note: note, Source: key})
		l.imageInjected[key] = true
		l.imageInjectedN++
	} else {
		ns = "图片已提交（重复或已超上限，未重复注入）：" + meta.Path + "\n" + ns
	}
	l.imageMu.Unlock()
	return ns
}

// loadImagePart 读图文件 → ImagePart（base64 data URL）+ 说明文本。
func (l *Loop) loadImagePart(path, mime, prompt string) (ImagePart, string, error) {
	full, err := resolveImagePath(l.WorkspaceRoot, path)
	if err != nil {
		return ImagePart{}, "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return ImagePart{}, "", err
	}
	if len(data) > imageSubmitMaxBytes {
		return ImagePart{}, "", fmt.Errorf("图片 %s 超过 2MiB 限制", humanBytes(int64(len(data))))
	}
	if mime == "" {
		mime = "image/png"
	}
	part := ImagePart{
		Data:     "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		MimeType: mime,
		Detail:   "auto",
	}
	note := "工具提交了图片（" + path + "，" + humanBytes(int64(len(data))) + "）"
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

// injectPendingImages 把待注入图片追加到 callMsgs 末尾——
// 每轮迭代末尾一次性消费（清空队列；同一轮图片消息紧跟工具结果之后）。
// ★ 仅当 Provider 支持多模态时注入（非视觉模型跳过——buildOpenAIMessages
//   在 multimodal=false 时忽略 Images，图片不会发出；此处提前跳过省开销）。
func (l *Loop) injectPendingImages(callMsgs []Message) []Message {
	l.imageMu.Lock()
	if len(l.pendingImages) == 0 {
		l.imageMu.Unlock()
		return callMsgs
	}
	if !l.supportsMultimodal() {
		l.imageMu.Unlock()
		return callMsgs // 非视觉模型：跳过图片（LLM 看不到，注入无意义）
	}
	imgs := append([]pendingImage(nil), l.pendingImages...)
	l.pendingImages = nil
	l.imageMu.Unlock()

	for _, img := range imgs {
		callMsgs = append(callMsgs, Message{
			Role:    RoleUser,
			Content: "【图片】" + img.Note,
			Images:  []ImagePart{img.Part},
		})
	}
	return callMsgs
}

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
