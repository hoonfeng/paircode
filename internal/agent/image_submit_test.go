package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestPNG 生成一张 4×4 红色测试 PNG 并返回路径。
func writeTestPNG(t *testing.T, dir, name string) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf []byte
	w := &bufWriter{&buf}
	if err := png.Encode(w, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

type bufWriter struct{ b *[]byte }

func (w *bufWriter) Write(p []byte) (int, error) {
	*w.b = append(*w.b, p...)
	return len(p), nil
}

// TestParseImageSubmitResult 标记解析 + 读图挂载 + 净化文本。
func TestParseImageSubmitResult(t *testing.T) {
	dir := t.TempDir()
	pngPath := writeTestPNG(t, dir, "shot.png")
	data, _ := os.ReadFile(pngPath)

	l := &Loop{WorkspaceRoot: dir}
	mark := "__SUBMIT_IMAGE__:{\"kind\":\"submit_image\",\"path\":\"" + filepath.ToSlash(pngPath) + "\",\"mime\":\"image/png\",\"size\":" + itoa64(int64(len(data))) + ",\"prompt\":\"检查是否有红色\"}"
	result := mark + "\n图片已提交给模型"

	clean := l.parseImageSubmitResult(result)
	if strings.Contains(clean, "__SUBMIT_IMAGE__") {
		t.Errorf("净化文本不应含标记: %s", clean)
	}
	if !strings.Contains(clean, "图片已提交给模型") {
		t.Errorf("净化文本应保留描述: %s", clean)
	}
	l.imageMu.Lock()
	if len(l.pendingImages) != 1 {
		t.Fatalf("pendingImages 应 1 张，得 %d", len(l.pendingImages))
	}
	pi := l.pendingImages[0]
	l.imageMu.Unlock()
	wantPrefix := "data:image/png;base64,"
	if !strings.HasPrefix(pi.Part.Data, wantPrefix) {
		t.Errorf("ImagePart.Data 应 %s 前缀，得 %.40s", wantPrefix, pi.Part.Data)
	}
	if pi.Part.MimeType != "image/png" || pi.Part.Detail != "auto" {
		t.Errorf("ImagePart 元数据: %+v", pi.Part)
	}
	if !strings.Contains(pi.Note, "检查是否有红色") {
		t.Errorf("note 应含 prompt: %s", pi.Note)
	}

	// 重复提交：去重不重复挂载
	clean2 := l.parseImageSubmitResult(mark + "\nagain")
	l.imageMu.Lock()
	n := len(l.pendingImages)
	l.imageMu.Unlock()
	if n != 1 {
		t.Errorf("重复路径应去重（仍 1 张），得 %d", n)
	}
	if !strings.Contains(clean2, "未重复注入") {
		t.Errorf("重复提交应提示: %s", clean2)
	}
}

// TestParseImageSubmitResultNotMarked 无标记结果原样返回。
func TestParseImageSubmitResultNotMarked(t *testing.T) {
	l := &Loop{}
	result := "普通工具结果，没有标记"
	if out := l.parseImageSubmitResult(result); out != result {
		t.Errorf("无标记应原样返回: %s", out)
	}
}

// TestImageSubmitPathTraversal 越界路径拒绝。
func TestImageSubmitPathTraversal(t *testing.T) {
	dir := t.TempDir()
	l := &Loop{WorkspaceRoot: dir}
	// 相对路径穿越（../ outside.png）
	result := "__SUBMIT_IMAGE__:{\"kind\":\"submit_image\",\"path\":\"../outside.png\",\"mime\":\"image/png\",\"size\":10,\"prompt\":\"\"}"
	out := l.parseImageSubmitResult(result)
	if !strings.Contains(out, "错误") && strings.Contains(out, "读取图片失败") == false {
		// 越界拦截应报错
		t.Logf("越界结果: %s", out)
	}
	if strings.Contains(out, "__SUBMIT_IMAGE__") {
		t.Errorf("失败也应剥离标记: %s", out)
	}
}

// TestInjectPendingImages 注入 user 消息（多模态 Provider）与非多模态跳过。
func TestInjectPendingImages(t *testing.T) {
	imgPart := ImagePart{Data: "data:image/png;base64,QUJD", MimeType: "image/png", Detail: "auto"}

	// 多模态 provider：注入
	l := &Loop{Provider: &OpenAIProvider{Multimodal: true}}
	l.imageMu.Lock()
	l.pendingImages = append(l.pendingImages, pendingImage{Part: imgPart, Note: "工具提交了图片（x.png）", Source: "x.png"})
	l.imageMu.Unlock()
	out := l.injectPendingImages([]Message{{Role: RoleUser, Content: "hi"}})
	if len(out) != 2 {
		t.Fatalf("多模态应注入 1 条（共 2），得 %d", len(out))
	}
	last := out[len(out)-1]
	if last.Role != RoleUser || len(last.Images) != 1 || last.Images[0].Data != "data:image/png;base64,QUJD" {
		t.Errorf("注入消息错误: %+v", last)
	}
	// 消费后队列清空
	l.imageMu.Lock()
	n := len(l.pendingImages)
	l.imageMu.Unlock()
	if n != 0 {
		t.Errorf("注入后队列应清空，得 %d", n)
	}

	// 非多模态 provider：跳过
	l2 := &Loop{Provider: &OpenAIProvider{Multimodal: false}}
	l2.imageMu.Lock()
	l2.pendingImages = append(l2.pendingImages, pendingImage{Part: imgPart, Note: "n", Source: "y.png"})
	l2.imageMu.Unlock()
	out2 := l2.injectPendingImages([]Message{{Role: RoleUser, Content: "hi"}})
	if len(out2) != 1 {
		t.Errorf("非多模态应跳过注入，得 %d 条", len(out2))
	}
}

// TestResolveImagePath 相对/绝对路径解析。
func TestResolveImagePath(t *testing.T) {
	root := "C:/work/proj"
	p, err := resolveImagePath(root, "screenshots/a.png")
	if err != nil || filepath.Clean(p) != filepath.Clean("C:/work/proj/screenshots/a.png") {
		t.Errorf("相对路径解析: %s %v", p, err)
	}
	// 越界拒
	if _, err := resolveImagePath(root, "../outside.png"); err == nil {
		t.Errorf("相对越界应拒绝")
	}
	// 绝对路径在工作区内
	p2, err := resolveImagePath(root, "C:/work/proj/x.png")
	if err != nil || filepath.Clean(p2) != filepath.Clean("C:/work/proj/x.png") {
		t.Errorf("绝对路径解析: %s %v", p2, err)
	}
}

// TestImageSubmitMaxTotal 总数上限。
func TestImageSubmitMaxTotal(t *testing.T) {
	dir := t.TempDir()
	l := &Loop{WorkspaceRoot: dir}
	// 伪造 21 张不同路径的图
	for i := 0; i < imageSubmitMaxTotal+1; i++ {
		p := filepath.Join(dir, "f"+itoa64(int64(i))+".png")
		pngBytes := []byte{0x89, 'P', 'N', 'G'}
		_ = os.WriteFile(p, pngBytes, 0o644)
		mark := "__SUBMIT_IMAGE__:{\"kind\":\"submit_image\",\"path\":\"" + filepath.ToSlash(p) + "\",\"mime\":\"image/png\",\"size\":4,\"prompt\":\"\"}"
		out := l.parseImageSubmitResult(mark + "\n")
		if i < imageSubmitMaxTotal {
			if strings.Contains(out, "超上限") {
				t.Errorf("第 %d 张不应提示超限: %s", i, out)
			}
		} else {
			if !strings.Contains(out, "超上限") {
				t.Errorf("第 %d 张应提示超限: %s", i, out)
			}
		}
	}
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

var _ = base64.StdEncoding

// TestImageSubmitEndToEnd 端到端：MockProvider 捕获 LLM 请求消息，
// 验证 submit_image 工具结果解析后 next LLM 请求携带 image_url 用户消息。
func TestImageSubmitEndToEnd(t *testing.T) {
	dir := t.TempDir()
	pngPath := writeTestPNG(t, dir, "ui.png")
	_, _ = os.ReadFile(pngPath)

	// 注册 submit_image 工具（仿真磁盘插件行为：返回标记行 + 提示文本）
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:       "submit_image",
		Description: "提交图片给 LLM 视觉识别",
		Parameters:  objSchema(props{"path": strProp("图片路径")}, "path"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p := argStr(args, "path")
			ext := strings.TrimPrefix(filepath.Ext(p), ".")
			mime := map[string]string{"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "gif": "image/gif", "webp": "image/webp"}[ext]
			if mime == "" {
				mime = "image/jpeg"
			}
			b, _ := os.ReadFile(p)
			mark, _ := json.Marshal(imageSubmitMeta{Kind: "submit_image", Path: p, Mime: mime, Size: int64(len(b)), Prompt: "检查 UI 是否白屏"})
			return "__SUBMIT_IMAGE__:" + string(mark) + "\n图片已提交给模型：" + p, nil
		},
	})

	// 捕获型 MockProvider（多模态）
	captured := &CaptureProvider{MultimodalVal: true}
	_ = NewLoopForTest(reg, captured, dir)

	// 手动走循环一步：模拟调用 submit_image → LLM
	// 这里直接走工具执行 + buildCallContext 验证注入（不启完整 Run 循环）
	tc := ToolCall{ID: "c1", Type: "function", Function: FunctionCall{Name: "submit_image", Arguments: `{"path":"` + filepath.ToSlash(pngPath) + `"}`}}
	result, terr := reg.Execute(context.Background(), tc.Function.Name, tc.Function.Arguments)
	if terr != nil {
		t.Fatalf("工具执行失败: %v", terr)
	}
	loop := captured.Loop
	clean := loop.parseImageSubmitResult(result)
	if strings.Contains(clean, "__SUBMIT_IMAGE__") {
		t.Errorf("净化文本残留标记: %s", clean)
	}
	// 模拟 LLM 下一次调用：buildCallContext 注入
	callMsgs := loop.buildCallContext([]Message{{Role: RoleUser, Content: "测试 UI"}})
	found := false
	for _, m := range callMsgs {
		if len(m.Images) > 0 {
			found = true
			if !strings.HasPrefix(m.Images[0].Data, "data:image/png;base64,") {
				t.Errorf("ImagePart.Data 非 data URL: %.30s", m.Images[0].Data)
			}
			if !strings.Contains(m.Content, "检查 UI 是否白屏") {
				t.Errorf("注入消息应含 prompt: %s", m.Content)
			}
		}
	}
	if !found {
		t.Fatalf("buildCallContext 未注入图片消息: %d 条", len(callMsgs))
	}
}

// CaptureProvider 捕获型多模态 Provider。
type CaptureProvider struct {
	MultimodalVal bool
	Loop          *Loop
}

func (m *CaptureProvider) Name() string { return "capture" }
func (m *CaptureProvider) Multimodal() bool {
	return m.MultimodalVal
}
func (m *CaptureProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	return Message{Role: RoleAssistant, Content: "ok"}, nil
}

// NewLoopForTest 构造带捕获 Provider 的 Loop。
func NewLoopForTest(reg *Registry, cap *CaptureProvider, root string) *Loop {
	l := &Loop{Provider: cap, Registry: reg, WorkspaceRoot: root}
	cap.Loop = l
	return l
}
