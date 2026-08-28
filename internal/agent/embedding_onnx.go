//go:build cgo

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yalue/onnxruntime_go"
)

// ONNXBackend ONNX Runtime 嵌入后端（bge-small-zh-v1.5, 512维）。
type ONNXBackend struct {
	session   *onnxruntime_go.AdvancedSession
	tokenizer *BERTTokenizer
	inputIDs  *onnxruntime_go.Tensor[int64]
	attention *onnxruntime_go.Tensor[int64]
	output    *onnxruntime_go.Tensor[float32]
	dim       int
}

// findModelDir 按优先级查找模型目录：便携模式 > 项目 config/models/ > 工作区 .pair/。
func findModelDir(root string) string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "models", "bge-small-zh-v1.5")
		if _, err := os.Stat(filepath.Join(p, "model.onnx")); err == nil {
			return p
		}
	}
	// 项目 config/models/ 目录（开发环境）
	p := filepath.Join(root, "config", "models", "bge-small-zh-v1.5")
	if _, err := os.Stat(filepath.Join(p, "model.onnx")); err == nil {
		return p
	}
	// 工作区 .pair/embeddings/
	p = filepath.Join(root, ".pair", "embeddings", "bge-small-zh-v1.5")
	if _, err := os.Stat(filepath.Join(p, "model.onnx")); err == nil {
		return p
	}
	return ""
}

// findOrtDll 查找 onnxruntime 运行时库（按平台选库名，2026-08-22 多平台化）：
//
//	windows → onnxruntime.dll；linux → libonnxruntime.so；darwin → libonnxruntime.dylib
//
// 优先级：发布版 bin/config/onnx/ > 项目 bin/config/onnx/ > exe 同目录 > 项目根。
func findOrtDll(root string) string {
	libName := "onnxruntime.dll"
	switch runtime.GOOS {
	case "linux":
		libName = "libonnxruntime.so"
	case "darwin":
		libName = "libonnxruntime.dylib"
	}
	candidates := []string{}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "bin", "config", "onnx", libName),
			filepath.Join(exeDir, libName))
	}
	candidates = append(candidates,
		filepath.Join(root, "bin", "config", "onnx", libName),
		filepath.Join(root, libName))
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// NewONNXBackend 创建 ONNX 嵌入后端（便携 models/ 优先）。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	embedDir := findModelDir(root)
	if embedDir == "" {
		return nil, fmt.Errorf("模型文件未找到（已检查便携 models/ 和 .pair/embeddings/）")
	}
	tokenizer, err := LoadBERTTokenizer(embedDir)
	if err != nil {
		return nil, fmt.Errorf("分词器失败: %w", err)
	}
	// 定位 onnxruntime.dll（发布版 bin/config/onnx/、开发版 bin/config/onnx/、exe 同目录兜底）
	if p := findOrtDll(root); p != "" {
		onnxruntime_go.SetSharedLibraryPath(p)
	}
	if err := onnxruntime_go.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("ONNX Runtime 初始化失败: %w", err)
	}
	seqLen, dim := int64(512), int64(512)
	shape := onnxruntime_go.NewShape(int64(1), seqLen)
	outputShape := onnxruntime_go.NewShape(int64(1), seqLen, dim)
	inIDs := make([]int64, 512)
	inMask := make([]int64, 512)
	inputTensor, _ := onnxruntime_go.NewTensor(shape, inIDs)
	maskTensor, _ := onnxruntime_go.NewTensor(shape, inMask)
	outputTensor, _ := onnxruntime_go.NewTensor(outputShape, make([]float32, 512*512))

	session, err := onnxruntime_go.NewAdvancedSession(
		filepath.Join(embedDir, "model.onnx"),
		[]string{"input_ids", "attention_mask"},
		[]string{"last_hidden_state"},
		[]onnxruntime_go.Value{inputTensor, maskTensor},
		[]onnxruntime_go.Value{outputTensor},
		nil,
	)
	if err != nil {
		inputTensor.Destroy()
		maskTensor.Destroy()
		outputTensor.Destroy()
		return nil, fmt.Errorf("创建 ONNX 会话失败: %w", err)
	}
	return &ONNXBackend{
		session: session, tokenizer: tokenizer,
		inputIDs: inputTensor, attention: maskTensor, output: outputTensor, dim: 512,
	}, nil
}

// Embed 计算文本嵌入（均值池化 + 归一化）。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	ids, mask, err := b.tokenizer.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("分词失败: %w", err)
	}
	copy(b.inputIDs.GetData(), ids)
	copy(b.attention.GetData(), mask)
	if err := b.session.Run(); err != nil {
		return nil, fmt.Errorf("推理失败: %w", err)
	}
	data := b.output.GetData()
	vec := MeanPool(data, mask, b.dim)
	Normalize(vec)
	return vec, nil
}

func (b *ONNXBackend) Available() bool { return b.session != nil }
func (b *ONNXBackend) Dim() int        { return b.dim }
func (b *ONNXBackend) Close() error {
	b.inputIDs.Destroy()
	b.attention.Destroy()
	b.output.Destroy()
	return b.session.Destroy()
}
