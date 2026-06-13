package agent

// run_test 工具：运行指定 Go 包的测试，返回成功/失败状态及详细输出。
// 接受 package_path（必须，相对于工作区根）和 test_pattern（可选，go test -run 过滤）。
// 实现方式：在 root 目录下执行 "go test -v -count=1 [package_path]" 并捕获输出。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerRunTestTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "run_test",
		Description: "运行工作区内指定 Go 包的测试并返回结果。package_path 是相对于工作区的包路径" +
			"（如 '.' './internal/lsp'），test_pattern 可选（对应 go test -run 过滤，如 'TestFoo'）。" +
			"返回 PASS/FAIL 状态、退出码、统计信息和详细测试输出。",
		Parameters: objSchema(props{
			"package_path": strProp("Go 包的目录路径（相对于工作区根），如 '.' 或 './internal/lsp'"),
			"test_pattern": strProp("可选：测试名过滤模式，对应 go test -run，如 'TestFoo'"),
		}, "package_path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			pkgPath := argStr(args, "package_path")
			if pkgPath == "" {
				return "", fmt.Errorf("package_path 不能为空")
			}

			// 验证路径在工作区内
			absPath, err := resolvePath(root, pkgPath)
			if err != nil {
				return "", fmt.Errorf("无效的包路径: %w", err)
			}

			// 检查目录是否存在
			fi, err := os.Stat(absPath)
			if err != nil {
				return "", fmt.Errorf("包目录不可访问: %w", err)
			}
			if !fi.IsDir() {
				return "", fmt.Errorf("「%s」不是目录", pkgPath)
			}

			// 检查是否有 Go 源文件
			entries, _ := os.ReadDir(absPath)
			hasGo := false
			hasTest := false
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
					hasGo = true
					if strings.HasSuffix(e.Name(), "_test.go") {
						hasTest = true
					}
				}
			}
			if !hasGo {
				return "", fmt.Errorf("目录「%s」中没有 Go 源文件", pkgPath)
			}

			// 计算相对于 root 的包路径（go test 需要相对路径）
			rel := pkgPath
			if !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".") {
				rel = "." + string(filepath.Separator) + rel
			} else if filepath.IsAbs(rel) {
				// 如果是绝对路径，转成相对于 root 的路径
				r, err := filepath.Rel(root, rel)
				if err == nil {
					rel = "." + string(filepath.Separator) + r
				}
			}

			// 构造 go test 参数
			argsList := []string{"test", "-v", "-count=1"}
			if tp := argStr(args, "test_pattern"); tp != "" {
				argsList = append(argsList, "-run", tp)
			}
			argsList = append(argsList, rel)

			// 执行 go test
			cmd := exec.CommandContext(ctx, "go", argsList...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			output := string(out)

			// 解析退出码
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					// 超时、命令未找到等其他错误
					return "", fmt.Errorf("执行 go test 失败: %w\n%s", err, capOutput(output, 16000))
				}
			}

			// 构建返回结果
			var b strings.Builder
			if exitCode == 0 {
				b.WriteString("✅ 测试全部通过（退出码 0）\n")
			} else {
				b.WriteString(fmt.Sprintf("❌ 存在失败测试（退出码 %d）\n", exitCode))
			}

			b.WriteString(fmt.Sprintf("包: %s\n", pkgPath))
			if tp := argStr(args, "test_pattern"); tp != "" {
				b.WriteString(fmt.Sprintf("过滤: -run %q\n", tp))
			}
			b.WriteString(fmt.Sprintf("有测试文件: %v\n", hasTest))
			b.WriteString("\n--- 测试输出 ---\n")
			b.WriteString(capOutput(output, 32000))

			return b.String(), nil
		},
	})
}
