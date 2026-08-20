// model_groups.go — 模型组（Model Group）存储
//
// ★ 2026-08-20 模型组概念：AI 配置从「单例 Key」升级为「多例模型组」。
//   模型组 = 用户命名的连接配置集合；每个组内包含多个「实例」。
//   实例 = 一个 API Key + 服务商/BaseURL + 模型列表（即 models.json 的每个服务商条目）。
//   对话面板选「模型组 + 模型」→ 自动匹配该模型所属实例的 Key/BaseURL，不再需要选服务商。
//
// 存储：config/model-groups.json（安装目录）
//   { "<组名>": ["<实例名>", ...] }   // 组名用户自定义；实例名 = models.json 的 provider 名

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// ModelGroups 模型组定义：组名 → 组内实例（provider）列表。
type ModelGroups map[string][]string

var (
	// GroupList 当前生效的模型组定义（加载自安装目录或空）。
	GroupList ModelGroups
)

// ModelGroupsPath 返回 model-groups.json 文件路径（安装目录 config/ 下）。
func ModelGroupsPath() string {
	return filepath.Join(InstallDir(), "config", "model-groups.json")
}

// LoadModelGroups 加载 model-groups.json。
//   - 文件存在且有效 → 使用文件内容
//   - 文件不存在/格式错误 → 空映射（不报错，模型组为可选功能）
func LoadModelGroups() {
	p := ModelGroupsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		GroupList = ModelGroups{}
		return
	}
	var g ModelGroups
	if err := json.Unmarshal(data, &g); err != nil {
		GroupList = ModelGroups{}
		return
	}
	if g == nil {
		g = ModelGroups{}
	}
	GroupList = g
}

// GetModelGroups 返回模型组定义（懒加载）。
func GetModelGroups() ModelGroups {
	if GroupList == nil {
		LoadModelGroups()
	}
	return GroupList
}

// GetGroupNames 返回全部模型组名（排序）。
func GetGroupNames() []string {
	g := GetModelGroups()
	names := make([]string, 0, len(g))
	for n := range g {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GetGroupInstances 返回指定模型组的实例（provider）列表。
func GetGroupInstances(group string) []string {
	g := GetModelGroups()
	return g[group]
}

// SetModelGroups 全量替换模型组定义。
func SetModelGroups(g ModelGroups) {
	GroupList = g
}

// SaveModelGroups 持久化模型组定义到 model-groups.json。
func SaveModelGroups() error {
	p := ModelGroupsPath()
	data, err := json.MarshalIndent(GroupList, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// EnsureModelGroups 确保模型组已加载（不存在则建空映射，不落盘——用户未配置模型组时保持默认行为）。
func EnsureModelGroups() {
	LoadModelGroups()
}
