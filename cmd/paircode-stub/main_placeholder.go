//go:build !stub

// 默认构建（不带 -tags stub）时占位：stub 需要先由构建器把主程序与 Skia
// 原生库复制到 payload/ 目录，普通 `go build ./...` 不应编译真实 stub。
package main

import "fmt"

func main() {
	fmt.Println("paircode-stub：请用构建器的 build-stub 步骤（-tags stub）生成单文件启动器")
}
