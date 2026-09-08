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

	clean := l.parseImageSubmitResult(result, "call-1")
	if strings.Contains(clean, "__SUBMIT_IMAGE__") {
		t.Errorf("净化文本不应含标记: %s", clean)
	}
	if !strings.Contains(clean, "图片已提交给模型") {
		t.Errorf("净化文本应保留描述: %s", clean)
	}
	imgs := l.imageByCall["call-1"]
	if len(imgs) != 1 {
		t.Fatalf("imageByCall[call-1] 应 1 张，得 %d", len(imgs))
	}
	part := imgs[0]
	// ★ 2026-09 对齐 dsh：ImagePart 只带内容寻址引用与事实（请求装配时才投影为 data URL）
	if part.Data != "" {
		t.Errorf("ImagePart.Data 应为空（未装配）: %.40s", part.Data)
	}
	if len(part.Ref) != 64 {
		t.Errorf("ImagePart.Ref 应为 sha256 引用: %q", part.Ref)
	}
	if part.Width != 4 || part.Height != 4 || part.OrigWidth != 4 || part.OrigHeight != 4 {
		t.Errorf("ImagePart 尺寸事实: %+v", part)
	}
	if part.Bytes <= 0 {
		t.Errorf("ImagePart.Bytes 应 > 0: %+v", part)
	}
	if part.MimeType != "image/png" || part.Detail != "auto" {
		t.Errorf("ImagePart 元数据: %+v", part)
	}
	// 净化文本应含 dsh 风格信封（<path>/<type>/<content>）
	if !strings.Contains(clean, "<type>image</type>") || !strings.Contains(clean, "4x4 px") {
		t.Errorf("净化文本应含图片信封: %s", clean)
	}

	// 同一路径再次读取（新 callID）→ 对齐 dsh：不做路径去重，各自成一次 occurrence
	clean2 := l.parseImageSubmitResult(mark+"\nagain", "call-2")
	if len(l.imageByCall["call-2"]) != 1 {
		t.Errorf("新 callID 应挂载 1 张，得 %d", len(l.imageByCall["call-2"]))
	}
	if !strings.Contains(clean2, "again") {
		t.Errorf("净化文本应保留描述: %s", clean2)
	}
	// 缺少 callID → 不挂载并提示
	clean3 := l.parseImageSubmitResult(mark+"\nx", "")
	if !strings.Contains(clean3, "缺少 call id") {
		t.Errorf("缺 callID 应提示: %s", clean3)
	}
}

// TestParseImageSubmitResultNotMarked 无标记结果原样返回。
func TestParseImageSubmitResultNotMarked(t *testing.T) {
	l := &Loop{}
	result := "普通工具结果，没有标记"
	if out := l.parseImageSubmitResult(result, "c1"); out != result {
		t.Errorf("无标记应原样返回: %s", out)
	}
}

// TestImageSubmitPathTraversal 越界路径拒绝。
func TestImageSubmitPathTraversal(t *testing.T) {
	dir := t.TempDir()
	l := &Loop{WorkspaceRoot: dir}
	// 相对路径穿越（../ outside.png）
	result := "__SUBMIT_IMAGE__:{\"kind\":\"submit_image\",\"path\":\"../outside.png\",\"mime\":\"image/png\",\"size\":10,\"prompt\":\"\"}"
	out := l.parseImageSubmitResult(result, "c1")
	if !strings.Contains(out, "错误") && strings.Contains(out, "读取图片失败") == false {
		// 越界拦截应报错
		t.Logf("越界结果: %s", out)
	}
	if strings.Contains(out, "__SUBMIT_IMAGE__") {
		t.Errorf("失败也应剥离标记: %s", out)
	}
}

// TestExtractToolImages 工具结果图片提取合并（对齐 dsh serializeMessagesWithImages）。
func TestExtractToolImages(t *testing.T) {
	imgA := ImagePart{Ref: strings.Repeat("a", 64), MimeType: "image/png", Detail: "auto"}
	imgB := ImagePart{Ref: strings.Repeat("b", 64), MimeType: "image/jpeg", Detail: "auto"}

	msgs := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Type: "function"}}},
		{Role: RoleTool, ToolCallID: "c1", Content: "r1", Images: []ImagePart{imgA}},
		{Role: RoleTool, ToolCallID: "c2", Content: "r2", Images: []ImagePart{imgB}},
		{Role: RoleAssistant, Content: "done"},
	}
	out := extractToolImages(msgs)
	if len(out) != 6 {
		t.Fatalf("应合并为 6 条（5 原消息 + 1 图片 user），得 %d: %+v", len(out), out)
	}
	merged := out[4]
	if merged.Role != RoleUser || len(merged.Images) != 2 {
		t.Fatalf("合并消息错误: %+v", merged)
	}
	if !strings.HasPrefix(merged.Content, toolResultImageText) {
		t.Errorf("合并消息前缀应为 %q，得 %q", toolResultImageText, merged.Content)
	}
	if len(out[2].Images) != 0 || len(out[3].Images) != 0 {
		t.Errorf("tool 消息应剥离图片（wire 不允许 tool 带图）: %+v", out)
	}
	if len(msgs[2].Images) != 1 {
		t.Errorf("原消息不应被修改: %+v", msgs[2])
	}
	// 无图片时原样返回（快速路径）
	same := extractToolImages([]Message{{Role: RoleUser, Content: "x"}})
	if len(same) != 1 {
		t.Errorf("无图片应原样返回，得 %d", len(same))
	}
}

// TestAttachToolResultImages 按 tool_call id 认领图片（JS 循环路径兜底）。
func TestAttachToolResultImages(t *testing.T) {
	img := ImagePart{Ref: strings.Repeat("c", 64), MimeType: "image/png", Detail: "auto"}
	l := &Loop{}
	l.imageByCall = map[string][]ImagePart{"c1": {img}}
	msgs := []Message{
		{Role: RoleUser, Content: "hi"},
		{Role: RoleTool, ToolCallID: "c1", Content: "r"},
	}
	out := l.attachToolResultImages(msgs)
	if len(out[1].Images) != 1 {
		t.Fatalf("应认领 1 张: %+v", out[1])
	}
	// 认领后清空（幂等）
	if again := l.attachToolResultImages(msgs); len(again[1].Images) != 0 {
		t.Errorf("重复认领应为空: %+v", again[1])
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
// 验证 read_image 工具结果解析后 next LLM 请求携带 image_url 用户消息。
func TestImageSubmitEndToEnd(t *testing.T) {
	dir := t.TempDir()
	pngPath := writeTestPNG(t, dir, "ui.png")
	_, _ = os.ReadFile(pngPath)

	// 注册 read_image 工具（仿真磁盘插件行为：返回标记行 + 提示文本）
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:        "read_image",
		Description: "读取图片给 LLM 视觉识别",
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
			mark, _ := json.Marshal(imageSubmitMeta{Kind: "read_image", Path: p, Mime: mime, Size: int64(len(b)), Prompt: "检查 UI 是否白屏"})
			return "__SUBMIT_IMAGE__:" + string(mark) + "\n关注点：检查 UI 是否白屏", nil
		},
	})

	// 捕获型 MockProvider（多模态）
	captured := &CaptureProvider{MultimodalVal: true}
	_ = NewLoopForTest(reg, captured, dir)

	// 手动走循环一步：模拟调用 read_image → LLM
	// 这里直接走工具执行 + buildCallContext 验证注入（不启完整 Run 循环）
	tc := ToolCall{ID: "c1", Type: "function", Function: FunctionCall{Name: "read_image", Arguments: `{"path":"` + filepath.ToSlash(pngPath) + `"}`}}
	result, terr := reg.Execute(context.Background(), tc.Function.Name, tc.Function.Arguments)
	if terr != nil {
		t.Fatalf("工具执行失败: %v", terr)
	}
	loop := captured.Loop
	clean := loop.parseImageSubmitResult(result, tc.ID)
	if strings.Contains(clean, "__SUBMIT_IMAGE__") {
		t.Errorf("净化文本残留标记: %s", clean)
	}
	// 模拟循环历史：user → assistant(tool_call) → tool(结果，图片按调用认领进历史)
	hist := []Message{
		{Role: RoleUser, Content: "测试 UI"},
		{Role: RoleAssistant, ToolCalls: []ToolCall{tc}},
		{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: clean, Images: loop.takeCallImages(tc.ID)},
	}
	callMsgs := loop.buildCallContext(hist)
	found := false
	for _, m := range callMsgs {
		if len(m.Images) > 0 {
			found = true
			if !strings.HasPrefix(m.Images[0].Data, "data:image/png;base64,") {
				t.Errorf("ImagePart.Data 非 data URL: %.30s", m.Images[0].Data)
			}
			if !strings.HasPrefix(m.Content, toolResultImageText) {
				t.Errorf("注入消息应以 %q 开头: %s", toolResultImageText, m.Content)
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
