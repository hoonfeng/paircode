package agent

// go_run 工具：执行 go run 并返回 stdout/stderr。
// 接受可选 path（字符串，默认 "."）和可选 args（字符串数组，传递给程序的参数）。
// 实现方式：在 root 目录下执行 "go run [path] [args...]" 并捕获输出。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerGoRunTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "go_run",
		Description: "在工作区内执行 go run 运行指定 Go 程序并返回输出。path 是相对于工作区的 Go 包/文件路径" +
			"（如 '.' './cmd/foo' 'main.go'），默认 '.'。args 是可选参数数组，传递给被运行的 Go 程序（如 ['-port=8080']）。" +
			"返回运行成功/失败状态、退出码和完整输出。注意：运行程序可能产生副作用。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path": strProp("Go 包或文件的路径（相对于工作区根），如 '.' 或 './cmd/foo'，默认为 '.'"),
				"args": map[string]any{
					"type":        "array",
					"description": "可选：传递给 Go 程序的命令行参数，如 ['-port=8080', '--verbose']",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		},
		ReadOnly:         false, // 运行程序可能产生副作用（创建文件、网络请求等）
		RequiresApproval: false, // 由前端决定是否需用户确认
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			pkgPath := argStr(args, "path")
			if pkgPath == "" {
				pkgPath = "."
			}

			// 验证路径在工作区内
			absPath, err := resolvePath(root, pkgPath)
			if err != nil {
				return "", fmt.Errorf("无效的路径: %w", err)
			}

			// 检查路径是否存在
			fi, err := os.Stat(absPath)
			if err != nil {
				return "", fmt.Errorf("路径不可访问: %w", err)
			}

			// 如果是目录，检查是否包含 Go 源文件（防止无意义运行）
			if fi.IsDir() {
				hasGo := false
				entries, _ := os.ReadDir(absPath)
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
						hasGo = true
						break
					}
				}
				if !hasGo {
					return "", fmt.Errorf("目录「%s」中没有 Go 源文件", pkgPath)
				}
			}

			// 计算相对于 root 的路径（go run 需要相对路径）
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

			// 构造 go run 参数
			argsList := []string{"run"}

			// 添加包/文件路径
			argsList = append(argsList, rel)

			// 添加传递给程序的参数
			progArgs := argStrSlice(args, "args")
			argsList = append(argsList, progArgs...)

			// 执行 go run
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
					return "", fmt.Errorf("执行 go run 失败: %w\n%s", err, capOutput(output, 16000))
				}
			}

			// 构建返回结果
			var b strings.Builder
			if exitCode == 0 {
				b.WriteString("✅ 运行成功（退出码 0）\n")
			} else {
				b.WriteString(fmt.Sprintf("❌ 运行失败（退出码 %d）\n", exitCode))
			}

			b.WriteString(fmt.Sprintf("路径: %s\n", pkgPath))
			b.WriteString(fmt.Sprintf("命令: go run %s", rel))
			for _, a := range progArgs {
				b.WriteString(" " + a)
			}
			b.WriteString("\n")

			b.WriteString("\n--- 程序输出 ---\n")
			b.WriteString(capOutput(output, 32000))

			if exitCode != 0 {
				b.WriteString(fmt.Sprintf("\n--- 进程以非零退出码 %d 结束 ---\n", exitCode))
			}

			return b.String(), nil
		},
	})
}
