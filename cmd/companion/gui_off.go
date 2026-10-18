//go:build !gui

// 非 GUI 构建（默认）：纯 Web 服务，不引入 wb-ui 的 cgo 依赖。
package main

// runGUI 在非 gui 构建下不做事；返回 false 表示未进入 GUI 模式（main 继续
// 永久阻塞，行为与此前完全一致）。
func runGUI(int) bool { return false }
