package agent

// go_build 工具：执行 go build 并返回 stdout/stderr。
// 接受可选 path（字符串，默认 "."）和 flags（可选字符串数组）参数。
// 实现方式：在 root 目录下执行 "go build [flags] [path]" 并捕获输出。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerGoBuildTools(r *Registry, root string) {
	r.Register(&Tool{
		Name: "go_build",
		Description: "在工作区内执行 go build 构建指定包并返回编译结果。path 是相对于工作区的包路径" +
			"（如 '.' './internal/lsp'），默认 '.'。flags 是可选参数数组（如 ['-v', '-race']）。" +
			"返回构建成功/失败状态、退出码和详细编译输出。",
		Parameters: map[string]any{
			"type": "object",
			"properties": props{
				"path": strProp("Go 包的目录路径（相对于工作区根），如 '.' 或 './internal/lsp'，默认为 '.'"),
				"flags": map[string]any{
					"type":        "array",
					"description": "可选：传递给 go build 的额外参数，如 ['-v', '-race', '-tags=debug']",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
		},
		ReadOnly:         false,
		RequiresApproval: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			pkgPath := argStr(args, "path")
			if pkgPath == "" {
				pkgPath = "."
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

			// 计算相对于 root 的包路径（go build 需要相对路径）
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

			// 构造 go build 参数
			argsList := []string{"build"}

			// 添加额外 flags
			flags := argStrSlice(args, "flags")
			argsList = append(argsList, flags...)

			// 添加包路径
			argsList = append(argsList, rel)

			// 执行 go build
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
					return "", fmt.Errorf("执行 go build 失败: %w\n%s", err, capOutput(output, 16000))
				}
			}

			// 构建返回结果
			var b strings.Builder
			if exitCode == 0 {
				b.WriteString("✅ 构建成功（退出码 0）\n")
			} else {
				b.WriteString(fmt.Sprintf("❌ 构建失败（退出码 %d）\n", exitCode))
			}

			b.WriteString(fmt.Sprintf("包: %s\n", pkgPath))
			b.WriteString(fmt.Sprintf("参数: go build %s %s\n", strings.Join(flags, " "), rel))
			b.WriteString("\n--- 构建输出 ---\n")
			b.WriteString(capOutput(output, 32000))

			return b.String(), nil
		},
	})
}
