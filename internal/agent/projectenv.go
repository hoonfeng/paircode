package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadProjectEnv 读取 .pair/project.md 项目环境档案。
// 如果文件不存在或为空，返回空字符串。
// agent 可以自行通过 read / edit 维护这个文件，
// 记录项目编译方式、多端信息、环境配置等，避免反复探测环境浪费 token。
func ReadProjectEnv(root string) string {
	if root == "" {
		return ""
	}
	path := filepath.Join(root, ".pair", "project.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return ""
	}
	if len(content) > 3000 {
		content = content[:3000] + "\n…（已截断）"
	}
	return content
}
