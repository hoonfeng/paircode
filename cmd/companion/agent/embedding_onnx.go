//go:build onnx

package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yalue/onnxruntime_go"
)

// ── ONNX Runtime 嵌入后端 ─────────────────────────
//
// 使用 bge-small-zh-v1.5 ONNX 模型做向量嵌入。
// 需要：
//   1. ONNX Runtime 共享库（onnxruntime.dll / libonnxruntime.so）
//   2. 模型文件 .pair/embeddings/bge-small-zh-v1.5/model.onnx
//   3. 词表文件 .pair/embeddings/bge-small-zh-v1.5/vocab.txt
//
// 构建时需传递 CGO 标记：go build -tags onnx ./cmd/companion

// init 自动注册 ONNX 为全局嵌入后端。
func init() {
	// 延迟到首次 GetEmbeddingBackend 调用时初始化
}

// ONNXBackend ONNX Runtime 嵌入后端。
type ONNXBackend struct {
	session   *onnxruntime_go.AdvancedSession
	tokenizer *BERTTokenizer
	modelPath string
	dim       int
}

// NewONNXBackend 创建 ONNX 嵌入后端。
// root 为工作区根路径，模型文件在 .pair/embeddings/bge-small-zh-v1.5/ 下。
// 如果模型文件或 ONNX Runtime 不可用，返回错误（调用者应回退关键词搜索）。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	embedDir := filepath.Join(root, ".pair", "embeddings", "bge-small-zh-v1.5")
	modelPath := filepath.Join(embedDir, "model.onnx")

	// 检查模型文件是否存在
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("模型文件不存在 %s: %w\n"+
			"请下载并转换 bge-small-zh-v1.5: pip install optimum && optimum-cli export onnx --model BAAI/bge-small-zh-v1.5 %s",
			modelPath, err, embedDir)
	}

	// 加载分词器
	tokenizer, err := LoadBERTTokenizer(embedDir)
	if err != nil {
		return nil, fmt.Errorf("加载分词器失败: %w", err)
	}

	// 初始化 ONNX Runtime 环境
	// 自动检测 onnxruntime.dll / libonnxruntime.so
	if err := onnxruntime_go.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("初始化 ONNX Runtime 失败: %w\n"+
			"请确保 onnxruntime 共享库已在 PATH/LD_LIBRARY_PATH 中", err)
	}

	// 创建推理会话
	// bge-small-zh 的输入输出名
	inputNames := []string{"input_ids", "attention_mask"}
	outputNames := []string{"last_hidden_state"}

	session, err := onnxruntime_go.NewAdvancedSession(
		modelPath,
		inputNames,
		outputNames,
		nil, // 使用默认选项
		nil, // 不优化到文件
	)
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX 会话失败: %w", err)
	}

	backend := &ONNXBackend{
		session:   session,
		tokenizer: tokenizer,
		modelPath: modelPath,
		dim:       384, // bge-small-zh 输出维度
	}

	return backend, nil
}

// Embed 计算文本的嵌入向量。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	if b.session == nil {
		return nil, fmt.Errorf("ONNX 会话未初始化")
	}

	// 1. 分词
	inputIDs, attentionMask, err := b.tokenizer.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("分词失败: %w", err)
	}

	// 2. 创建输入张量
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

	// 3. 创建输出张量
	outputShape := onnxruntime_go.NewShape(int64(1), int64(len(inputIDs)), int64(b.dim))
	outputData := make([]float32, len(inputIDs)*b.dim)
	outputTensor, err := onnxruntime_go.NewTensor(outputShape, outputData)
	if err != nil {
		return nil, fmt.Errorf("创建输出张量失败: %w", err)
	}
	defer outputTensor.Destroy()

	// 4. 运行推理
	err = b.session.Run([]*onnxruntime_go.Tensor[float32]{
		&inputTensor.Tensor,
		&maskTensor.Tensor,
	}, []*onnxruntime_go.Tensor[float32]{
		&outputTensor.Tensor,
	})
	if err != nil {
		return nil, fmt.Errorf("ONNX 推理失败: %w", err)
	}

	// 5. 均值池化 + 归一化
	embedding := MeanPool(outputData, attentionMask, b.dim)
	Normalize(embedding)

	return embedding, nil
}

// Available 检查后端是否可用。
func (b *ONNXBackend) Available() bool {
	return b.session != nil
}

// Dim 返回嵌入维度。
func (b *ONNXBackend) Dim() int {
	return b.dim
}

// Close 释放 ONNX 会话资源。
func (b *ONNXBackend) Close() error {
	if b.session != nil {
		return b.session.Destroy()
	}
	return nil
}
