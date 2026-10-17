//go:build !cgo

// CGO_ENABLED=0 时的占位实现：onnxruntime_go 依赖 cgo，无 cgo 时该包
// 所有文件都被构建约束排除。保留本文件让 `go build ./...` 在三平台
// （含 CI 的 CGO_ENABLED=0 交叉构建）都能通过，运行时明确提示原因。
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("本次构建未启用 CGO：ONNX 语义检索后端不可用（走关键词降级）。")
	fmt.Println("需要验证时请用 CGO_ENABLED=1 重新构建：CGO_ENABLED=1 go run ./scripts/env/onnxtest")
	os.Exit(1)
}
