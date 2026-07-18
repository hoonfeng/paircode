//go:build !cgo

package agent

import "fmt"

// ── 非 CGo 编译的回退实现 ──
//
// 构建环境无 C 编译器（CGO_ENABLED=0）时使用此文件。
// NewONNXBackend 返回错误，调用者自动回退关键词搜索。

// ONNXBackend 在此构建模式下不可用。
type ONNXBackend struct{}

// NewONNXBackend 返回错误提示缺少 C 编译器。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	return nil, fmt.Errorf("ONNX Runtime 需要 CGo。请安装 GCC 或 MinGW-w64 后重新构建")
}

// Embed 不支持。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("ONNX 未启用（CGo 不可用）")
}

// Available 返回 false。
func (b *ONNXBackend) Available() bool { return false }

// Dim 返回 0。
func (b *ONNXBackend) Dim() int { return 0 }

// Close 无操作。
func (b *ONNXBackend) Close() error { return nil }
