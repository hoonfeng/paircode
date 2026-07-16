//go:build onnx

package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yalue/onnxruntime_go"
)

// ONNXBackend ONNX Runtime 嵌入后端。
type ONNXBackend struct {
	session   *onnxruntime_go.AdvancedSession
	tokenizer *BERTTokenizer
	modelPath string
	dim       int
}

// NewONNXBackend 创建 ONNX 嵌入后端。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	embedDir := filepath.Join(root, ".pair", "embeddings", "bge-small-zh-v1.5")
	modelPath := filepath.Join(embedDir, "model.onnx")

	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("模型文件不存在 %s: %w", modelPath, err)
	}

	tokenizer, err := LoadBERTTokenizer(embedDir)
	if err != nil {
		return nil, fmt.Errorf("加载分词器失败: %w", err)
	}

	if err := onnxruntime_go.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("初始化 ONNX Runtime 失败: %w", err)
	}

	inputNames := []string{"input_ids", "attention_mask"}
	outputNames := []string{"last_hidden_state"}

	session, err := onnxruntime_go.NewAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		nil, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX 会话失败: %w", err)
	}

	backend := &ONNXBackend{
		session:   session,
		tokenizer: tokenizer,
		modelPath: modelPath,
		dim:       512,
	}

	return backend, nil
}

// Embed 计算文本的嵌入向量。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	if b.session == nil {
		return nil, fmt.Errorf("ONNX 会话未初始化")
	}
	inputIDs, attentionMask, err := b.tokenizer.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("分词失败: %w", err)
	}
	shape := onnxruntime_go.NewShape(int64(1), int64(len(inputIDs)))
	inputTensor, err := onnxruntime_go.NewTensor(shape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("创建输入张量失败: %w", err)
	}
	defer inputTensor.Destroy()
	maskTensor, err := onnxruntime_go.NewTensor(shape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("创建掩码张量失败: %w", err)
	}
	defer maskTensor.Destroy()
	outputShape := onnxruntime_go.NewShape(int64(1), int64(len(inputIDs)), int64(b.dim))
	outputData := make([]float32, len(inputIDs)*b.dim)
	outputTensor, err := onnxruntime_go.NewTensor(outputShape, outputData)
	if err != nil {
		return nil, fmt.Errorf("创建输出张量失败: %w", err)
	}
	defer outputTensor.Destroy()
	err = b.session.Run(
		[]*onnxruntime_go.Tensor[float32]{&inputTensor.Tensor, &maskTensor.Tensor},
		[]*onnxruntime_go.Tensor[float32]{&outputTensor.Tensor},
	)
	if err != nil {
		return nil, fmt.Errorf("ONNX 推理失败: %w", err)
	}
	embedding := MeanPool(outputData, attentionMask, b.dim)
	Normalize(embedding)
	return embedding, nil
}

func (b *ONNXBackend) Available() bool { return b.session != nil }
func (b *ONNXBackend) Dim() int        { return b.dim }
func (b *ONNXBackend) Close() error {
	if b.session != nil {
		return b.session.Destroy()
	}
	return nil
}
