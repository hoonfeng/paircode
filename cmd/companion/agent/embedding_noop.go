//go:build !onnx

package agent

import "fmt"

// ── 非 ONNX 编译的回退实现 ──────────────────────────
//
// 构建时不带 -tags onnx 时使用此文件。
// 所有函数返回不可用状态，调用者自动回退关键词搜索。

// ONNXBackend 在此构建模式下不可用。
type ONNXBackend struct{}

// NewONNXBackend 返回错误提示用户启用 ONNX 构建。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	return nil, fmt.Errorf("ONNX Runtime 支持未启用。请使用 -tags onnx 构建：go build -tags onnx ./cmd/companion")
}

// Embed 不支持。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("ONNX 未启用")
}

// Available 返回 false。
func (b *ONNXBackend) Available() bool { return false }

// Dim 返回 0。
func (b *ONNXBackend) Dim() int { return 0 }

// Close 无操作。
func (b *ONNXBackend) Close() error { return nil }
