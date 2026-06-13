package agent

// code_fix 工具：检查并修复 Go 代码中的已知模式（过时 API、简化语法等）。
// 接受 path（必须，文件或目录）参数，使用 go tool fix 自动检测并修复 Go 代码。
// apply=true 时直接写入修复结果（需审批），默认仅预览 diff。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func registerCodeFixTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "code_fix",
		Description: "检查并自动修复 Go 代码中的已知模式（如过时 API、简化语法等）。" +
			"接受 path（文件或目录），运行 go tool fix 检测可修复的代码模式。" +
			" apply=true 时直接写入修复结果（默认 false 仅预览 diff）。" +
			"支持单个文件或目录（递归处理所有 .go 文件）。",
		Parameters: objSchema(props{
			"path":  strProp("Go 文件或目录路径（相对于工作区的路径）"),
			"apply": boolProp("可选：是否直接写入修复后的结果（默认 false 仅预览 diff）"),
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

			// 检查 go 工具链是否可用
			if _, err := exec.LookPath("go"); err != nil {
				return "", fmt.Errorf("go 未安装或不在 PATH 中: %w", err)
			}

			// 获取当前 Go 版本（用于 -go 参数）
			goVer, err := detectGoVersion(ctx)
			if err != nil {
				goVer = "go1.21" // 降级：使用安全默认值
			}

			apply := argBool(args, "apply")

			if apply {
				// ── 写入模式：go tool fix（直接修复原文件） ──
				argsList := []string{"tool", "fix", "-go", goVer}
				if fi.IsDir() {
					argsList = append(argsList, absPath)
				} else {
					argsList = append(argsList, absPath)
				}
				cmd := exec.CommandContext(ctx, "go", argsList...)
				cmd.Dir = root
				out, err := cmd.CombinedOutput()
				output := strings.TrimSpace(string(out))

				if err != nil {
					return "", fmt.Errorf("go tool fix 执行失败: %w\n%s", err, output)
				}

				// 收集所有被修复的 .go 文件（相对路径）
				changedFiles := collectChangedGoFiles(root, absPath, fi.IsDir())

				if len(changedFiles) == 0 {
					return "✅ 未发现需要修复的代码模式。", nil
				}

				return fmt.Sprintf("✅ 已修复 %d 个文件:\n%s", len(changedFiles),
					strings.Join(changedFiles, "\n")), nil
			}

			// ── 预览模式：go tool fix -diff 显示差异 ──
			argsList := []string{"tool", "fix", "-diff", "-go", goVer}
			if fi.IsDir() {
				argsList = append(argsList, absPath)
			} else {
				argsList = append(argsList, absPath)
			}
			cmd := exec.CommandContext(ctx, "go", argsList...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			diff := string(out)

			if err != nil {
				return "", fmt.Errorf("go tool fix 检查失败: %w\n%s", err, diff)
			}

			if strings.TrimSpace(diff) == "" {
				return "✅ 未发现需要修复的代码模式，所有文件已使用最新的 Go 代码风格。", nil
			}

			// 有差异，显示修复 diff
			var b strings.Builder
			b.WriteString("📝 发现以下可自动修复的代码模式:\n\n")
			b.WriteString(diff)
			b.WriteString("\n---\n")
			b.WriteString("💡 使用 code_fix 并设置 apply=true 可应用以上修复。\n")

			return b.String(), nil
		},
	})
}

// detectGoVersion 获取当前 Go 版本字符串（如 "go1.24"），供 go tool fix -go 使用。
func detectGoVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOVERSION")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("无法获取 Go 版本: %w", err)
	}
	ver := strings.TrimSpace(string(out))
	// GOVERSION 返回 "go1.24.2" 格式，fix 接受 "go1.24" 格式
	// 截取前两个部分，如 "go1.24"
	if parts := strings.SplitN(ver, ".", 3); len(parts) >= 2 {
		return parts[0] + "." + parts[1], nil
	}
	return ver, nil
}

// collectChangedGoFiles 扫描指定路径下的 .go 文件，返回相对于 root 的路径列表。
// 用于 apply 模式后，向用户报告哪些文件被修复。
func collectChangedGoFiles(root, path string, isDir bool) []string {
	var files []string
	if isDir {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // 跳过无法访问的文件
			}
			if !info.IsDir() && strings.HasSuffix(p, ".go") {
				if rel, err := filepath.Rel(root, p); err == nil {
					files = append(files, filepath.ToSlash(rel))
				}
			}
			return nil
		})
	} else {
		if strings.HasSuffix(path, ".go") {
			if rel, err := filepath.Rel(root, path); err == nil {
				files = append(files, filepath.ToSlash(rel))
			}
		}
	}
	// 按文件名排序，保持输出稳定
	sort.Strings(files)
	return files
}
