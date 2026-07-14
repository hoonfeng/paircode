//go:build windows

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ModelListMap 按服务商分组可用模型列表。
// 加载自 config/models.json（安装目录），运行时通过 GetModels(provider) 查询。
// 若文件不存在或加载失败，使用内置默认列表。
type ModelListMap map[string][]string

var (
	// ModelList 当前生效的模型列表（加载自安装目录或内置默认）。
	ModelList ModelListMap

	// defaultModels 仅在 models.json 不存在或解析失败时使用的兜底列表。
	defaultModels = ModelListMap{
		"deepseek":          {"deepseek-r1", "deepseek-v4-pro", "deepseek-v4-flash"},
		"openai":            {"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "o1", "o3-mini", "o4-mini"},
		"anthropic":         {"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-4-sonnet-20250514", "claude-4-haiku-latest"},
		"openai-compatible": {"custom"},
		"custom":            {"custom"},
	}
)

// ModelsPath 返回 models.json 文件路径（安装目录 config/ 下）。
func ModelsPath() string {
	return filepath.Join(InstallDir(), "config", "models.json")
}

// LoadModelList 加载 models.json。
//   - 文件存在且有效 → 完全使用文件内容（忽略内置默认）
//   - 文件不存在 → 使用内置默认列表
//   - 文件存在但格式错误 → 使用内置默认列表
func LoadModelList() {
	p := ModelsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		// 文件不存在或不可读 → 使用内置默认
		useDefaultModels()
		return
	}

	var fileModels ModelListMap
	if err := json.Unmarshal(data, &fileModels); err != nil {
		// 格式错误 → 使用内置默认
		useDefaultModels()
		return
	}

	// 文件有效 → 完全使用文件数据
	ModelList = fileModels
}

// useDefaultModels 将 ModelList 设为内置默认列表的副本。
func useDefaultModels() {
	ModelList = make(ModelListMap, len(defaultModels))
	for k, v := range defaultModels {
		list := make([]string, len(v))
		copy(list, v)
		ModelList[k] = list
	}
}

// GetModels 返回指定服务商的可用模型列表。若不存在则返回空切片。
func GetModels(provider string) []string {
	if ModelList == nil {
		LoadModelList()
	}
	list, ok := ModelList[provider]
	if !ok {
		return nil
	}
	return list
}

// GetProviders 返回 ModelList 中的所有服务商名称（排序后）。
func GetProviders() []string {
	if ModelList == nil {
		LoadModelList()
	}
	providers := make([]string, 0, len(ModelList))
	for p := range ModelList {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// SaveModelList 将 ModelList 持久化到 models.json
func SaveModelList() error {
	if ModelList == nil {
		return fmt.Errorf("模型列表未加载")
	}
	p := ModelsPath()
	data, err := json.MarshalIndent(ModelList, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化模型列表失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	return os.WriteFile(p, data, 0o644)
}

// AddProvider 添加新服务商及模型列表
func AddProvider(name string, models []string) error {
	if ModelList == nil {
		LoadModelList()
	}
	if _, exists := ModelList[name]; exists {
		return fmt.Errorf("服务商 %q 已存在", name)
	}
	if name == "" {
		return fmt.Errorf("服务商名称不能为空")
	}
	ModelList[name] = models
	return SaveModelList()
}

// RemoveProvider 删除服务商
func RemoveProvider(name string) error {
	if ModelList == nil {
		LoadModelList()
	}
	if _, exists := ModelList[name]; !exists {
		return fmt.Errorf("服务商 %q 不存在", name)
	}
	delete(ModelList, name)
	return SaveModelList()
}

// UpdateProviderModels 更新指定服务商的模型列表
func UpdateProviderModels(name string, models []string) error {
	if ModelList == nil {
		LoadModelList()
	}
	if _, exists := ModelList[name]; !exists {
		return fmt.Errorf("服务商 %q 不存在", name)
	}
	ModelList[name] = models
	return SaveModelList()
}

// RenameProvider 重命名服务商（同时更新 settings 中引用的 provider 名）
func RenameProvider(oldName, newName string) error {
	if ModelList == nil {
		LoadModelList()
	}
	models, exists := ModelList[oldName]
	if !exists {
		return fmt.Errorf("服务商 %q 不存在", oldName)
	}
	if _, exists := ModelList[newName]; exists {
		return fmt.Errorf("服务商 %q 已存在", newName)
	}
	if newName == "" {
		return fmt.Errorf("服务商名称不能为空")
	}
	delete(ModelList, oldName)
	ModelList[newName] = models
	// 更新 settings 中的 provider 引用
	if Settings.Provider == oldName {
		Settings.Provider = newName
	}
	if Settings.CompressProvider == oldName {
		Settings.CompressProvider = newName
	}
	Save()
	return SaveModelList()
}

// WriteDefaultModels 在安装目录下写入内置默认 models.json（仅在文件不存在时）。
func WriteDefaultModels() error {
	p := ModelsPath()
	// 如果已存在则跳过
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	// 确保目录存在
	_ = os.MkdirAll(filepath.Dir(p), 0o755)

	data, err := json.MarshalIndent(defaultModels, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// EnsureModelList 确保模型列表已加载，若文件不存在则写入默认文件。
func EnsureModelList() {
	WriteDefaultModels()
	LoadModelList()
}
