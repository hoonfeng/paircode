package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectStructureOverview 生成项目结构概览：顶层目录、关键文件。
// 用于系统提示注入，让 Agent 无需每次探测项目环境。
func ProjectStructureOverview(roots []string) string {
	if len(roots) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## 项目结构概览\n\n")

	for ri, root := range roots {
		if ri > 0 {
			b.WriteString("\n---\n")
		}
		baseName := filepath.Base(root)
		b.WriteString(fmt.Sprintf("### %s\n", baseName))
		b.WriteString(fmt.Sprintf("根路径: %s\n\n", root))

		entries, err := os.ReadDir(root)
		if err != nil {
			b.WriteString("（无法读取目录）\n")
			continue
		}

		var dirs, files []string
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				dirs = append(dirs, name+"/")
			} else {
				files = append(files, name)
			}
		}

		if len(dirs) > 0 {
			b.WriteString(fmt.Sprintf("目录: %s\n", strings.Join(dirs, ", ")))
		}
		if len(files) > 0 {
			b.WriteString(fmt.Sprintf("关键文件: %s\n", strings.Join(files, ", ")))
		}
	}

	result := b.String()
	runes := []rune(result)
	if len(runes) > 2000 {
		result = string(runes[:2000]) + "…"
	}
	return result
}
