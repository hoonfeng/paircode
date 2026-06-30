package agent

// code_format 工具：检查并格式化 Go 代码。
// 接受 path（必须，文件或目录）参数，使用 gofmt 进行格式化检查并返回差异。
// apply=true 时直接写入格式化结果（需审批），默认仅预览差异。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func registerCodeFormatTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "code_format",
		Description: "检查并格式化 Go 代码。接受 path（文件或目录），运行 gofmt 检查 Go 代码格式" +
			"并返回需要格式化的文件和差异。 apply=true 时直接写入格式化结果（默认 false 仅预览差异）。" +
			"支持单个文件或目录（递归处理所有 .go 文件）。",
		Parameters: objSchema(props{
			"path":  strProp("Go 文件或目录路径（相对于工作区的路径）"),
			"apply": boolProp("可选：是否直接写入格式化后的结果（默认 false 仅预览差异）"),
		}, "path"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			targetPath := argStr(args, "path")
			if targetPath == "" {
				return "", fmt.Errorf("path 不能为空")
			}

			absPath, err := resolvePath(root, targetPath)
			if err != nil {
				return "", fmt.Errorf("无效路径: %w", err)
			}

			// 检查路径是否存在
			fi, err := os.Stat(absPath)
			if err != nil {
				return "", fmt.Errorf("路径不可访问: %w", err)
			}

			// 如果指定的是单个文件，确保是 .go 文件
			if !fi.IsDir() && filepath.Ext(absPath) != ".go" {
				return "", fmt.Errorf("「%s」不是 Go 源文件（扩展名必须为 .go）", targetPath)
			}

			// 检查 gofmt 是否可用
			if _, err := exec.LookPath("gofmt"); err != nil {
				return "", fmt.Errorf("gofmt 未安装或不在 PATH 中: %w", err)
			}

			apply := argBool(args, "apply")

			if apply {
				// ── 写入模式：先 gofmt -l（忽略 exit code）收集需改文件，再 gofmt -w 写入 ──
				// gofmt -l 列出非格式化文件时 exit 1（视作"有差异"），不能当错误。
				listArgs := []string{"-l"}
				if fi.IsDir() {
					listArgs = append(listArgs, absPath)
				} else {
					listArgs = append(listArgs, absPath)
				}
				listCmd := exec.CommandContext(ctx, "gofmt", listArgs...)
				listCmd.Dir = root
				listOut, _ := listCmd.CombinedOutput() // 忽略 exit code
				needFix := strings.TrimSpace(string(listOut))

				if needFix == "" {
					return "所有文件已符合 Go 格式规范，无需修改。", nil
				}

				// 实际写入
				writeArgs := []string{"-w"}
				if fi.IsDir() {
					writeArgs = append(writeArgs, absPath)
				} else {
					writeArgs = append(writeArgs, absPath)
				}
				writeCmd := exec.CommandContext(ctx, "gofmt", writeArgs...)
				writeCmd.Dir = root
				writeOut, wErr := writeCmd.CombinedOutput()
				if wErr != nil {
					return "", fmt.Errorf("gofmt 格式化失败: %w\n%s", wErr, strings.TrimSpace(string(writeOut)))
				}

				// 列出被修改的文件（相对路径）
				lines := strings.Split(needFix, "\n")
				var relFiles []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					rel, rErr := filepath.Rel(root, line)
					if rErr == nil {
						relFiles = append(relFiles, filepath.ToSlash(rel))
					} else {
						relFiles = append(relFiles, line)
					}
				}

				return fmt.Sprintf("✅ 已格式化 %d 个文件:\n%s", len(relFiles),
					strings.Join(relFiles, "\n")), nil
			}

			// ── 预览模式：gofmt -d 显示差异 ──
			argsList := []string{"-d"}
			if fi.IsDir() {
				argsList = append(argsList, absPath)
			} else {
				argsList = append(argsList, absPath)
			}
			cmd := exec.CommandContext(ctx, "gofmt", argsList...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			diff := string(out)

			// gofmt -d 有差异时部分版本 exit 1（视作"需格式化"），不能当错误；
			// 仅在无 diff 且 err 非 nil 时才报错。
			if strings.TrimSpace(diff) == "" {
				if err != nil {
					return "", fmt.Errorf("gofmt 检查失败: %w\n%s", err, diff)
				}
				return "✅ 所有文件已符合 Go 格式规范（gofmt 无差异输出）。", nil
			}

			// 有差异，显示格式化差异
			var b strings.Builder
			b.WriteString("📋 发现以下文件需要格式化:\n\n")
			b.WriteString(diff)
			b.WriteString("\n---\n")
			b.WriteString("💡 使用 code_format 并设置 apply=true 可写入以上格式化修改。\n")

			return b.String(), nil
		},
	})
}
