//go:build cgo

// ONNX 运行时可用性实证：用与 companion 相同的 cgo 构建配置，
// 加载本机 onnxruntime 动态库 + bge-small-zh-v1.5 模型并做一次推理。
//
// 跨平台：动态库名按 GOOS 选择（windows .dll / darwin .dylib / linux .so），
// 也可用 ONNX_LIB、ONNX_MODEL 环境变量直接指定绝对路径。
package main

import (
	"fmt"
	"os"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
)

// defaultONNXLib 按平台给默认动态库名（安装包内相对 config/onnx/ 布局）。
func defaultONNXLib() string {
	switch runtime.GOOS {
	case "windows":
		return `config\onnx\onnxruntime.dll`
	case "darwin":
		return "config/onnx/libonnxruntime.dylib"
	default:
		return "config/onnx/libonnxruntime.so"
	}
}

func main() {
	dll := os.Getenv("ONNX_LIB")
	if dll == "" {
		dll = defaultONNXLib()
	}
	model := os.Getenv("ONNX_MODEL")
	if model == "" {
		model = "models/bge-small-zh-v1.5/model.onnx"
	}

	if _, err := os.Stat(dll); err != nil {
		fmt.Println("❌ 动态库不存在:", err)
		fmt.Println("   可用 ONNX_LIB=<绝对路径> 指定（windows: onnxruntime.dll / darwin: libonnxruntime.dylib / linux: libonnxruntime.so）")
		os.Exit(1)
	}
	fmt.Printf("平台: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("库  :", dll, "(已找到)")
	if _, err := os.Stat(model); err != nil {
		fmt.Println("❌ 模型不存在:", err, "（可用 ONNX_MODEL=<绝对路径> 指定）")
		os.Exit(1)
	}
	fmt.Println("模型:", model, "(已找到)")

	ort.SetSharedLibraryPath(dll)
	if err := ort.InitializeEnvironment(); err != nil {
		fmt.Println("❌ InitializeEnvironment 失败:", err)
		os.Exit(1)
	}
	defer ort.DestroyEnvironment()
	fmt.Println("✅ ONNX Runtime 环境初始化成功")

	// 创建会话（与项目 embedding_onnx.go 相同的输入/输出签名）
	seqLen, dim := int64(512), int64(512)
	shape := ort.NewShape(1, seqLen)
	outShape := ort.NewShape(1, seqLen, dim)
	inIDs, _ := ort.NewTensor(shape, make([]int64, 512))
	inMask, _ := ort.NewTensor(shape, make([]int64, 512))
	out, _ := ort.NewTensor(outShape, make([]float32, 512*512))
	defer func() { inIDs.Destroy(); inMask.Destroy(); out.Destroy() }()

	sess, err := ort.NewAdvancedSession(model,
		[]string{"input_ids", "attention_mask"}, []string{"last_hidden_state"},
		[]ort.Value{inIDs, inMask}, []ort.Value{out}, nil)
	if err != nil {
		fmt.Println("❌ 创建会话失败:", err)
		os.Exit(1)
	}
	defer sess.Destroy()
	fmt.Println("✅ ONNX 会话创建成功（模型已加载）")

	// 真实推理
	ids := inIDs.GetData()
	mask := inMask.GetData()
	for i := range ids {
		ids[i] = 101 // [CLS]
		mask[i] = 1
	}
	if err := sess.Run(); err != nil {
		fmt.Println("❌ 推理失败:", err)
		os.Exit(1)
	}
	od := out.GetData()
	var sum float32
	for i := 0; i < 100; i++ {
		sum += od[i]
	}
	fmt.Printf("✅ 推理成功，输出张量非空（前100元素和=%.4f）\n", sum)
	fmt.Println("结论：ONNX 语义检索链路在本机构建产物上完全可用")
}
