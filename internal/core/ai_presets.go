// ai_presets.go — AI 配置预设（AI Preset）存储
//
// ★ 2026-08-20 AI 配置预设：用户把「一份完整 AI 配置」命名保存为预设，
//   供对话面板快速切换（多套配置，选中整套生效）。
//   每条预设 = 完整配置快照（provider/baseURL/apiKey/执行-规划-审核模型/
//   温度/思考档位/输出上限/上下文窗口）。
//
// 存储：config/ai-presets.json（安装目录）
//   { "<预设名>": { provider, baseURL, apiKey, executeModel, planModel,
//                   reviewModel, temperature, thinkingMode, maxTokens,
//                   contextMaxTokens } }
//
// 与 model-groups（组内挂实例）不同：预设是「整份配置的命名快照」，
// 保存时从当前 settings 抓取全部 AI 业务字段；应用时整套写回 settings。

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// AiPreset 一条 AI 配置预设（完整配置快照）。
type AiPreset struct {
	Provider         string `json:"provider,omitempty"`         // 服务商名（models.json 条目名）
	BaseURL          string `json:"baseURL,omitempty"`          // API 端点
	APIKey           string `json:"apiKey,omitempty"`           // API Key（预设自带，应用时写入 settings）
	ExecuteModel     string `json:"executeModel,omitempty"`     // 执行模型
	PlanModel        string `json:"planModel,omitempty"`        // 规划模型
	ReviewModel      string `json:"reviewModel,omitempty"`      // 审核模型
	Temperature      string `json:"temperature,omitempty"`      // 随机性（字符串保留原格式）
	ThinkingMode     string `json:"thinkingMode,omitempty"`     // 思考档位
	MaxTokens        int    `json:"maxTokens,omitempty"`        // 输出上限
	ContextMaxTokens int    `json:"contextMaxTokens,omitempty"` // 上下文窗口
}

// AiPresets AI 配置预设集合：预设名 → 完整配置。
type AiPresets map[string]AiPreset

var (
	// PresetList 当前生效的 AI 配置预设（加载自安装目录或空）。
	PresetList AiPresets
)

// AiPresetsPath 返回 ai-presets.json 文件路径（安装目录 config/ 下）。
func AiPresetsPath() string {
	return filepath.Join(InstallDir(), "config", "ai-presets.json")
}

// LoadAiPresets 加载 ai-presets.json。
//   - 文件存在且有效 → 使用文件内容
//   - 文件不存在/格式错误 → 空映射（不报错，预设为可选功能）
func LoadAiPresets() {
	p := AiPresetsPath()
	data, err := os.ReadFile(p)
	if err != nil {
		PresetList = AiPresets{}
		return
	}
	var g AiPresets
	if err := json.Unmarshal(data, &g); err != nil {
		PresetList = AiPresets{}
		return
	}
	if g == nil {
		g = AiPresets{}
	}
	PresetList = g
}

// GetAiPresets 返回 AI 配置预设定义（懒加载）。
func GetAiPresets() AiPresets {
	if PresetList == nil {
		LoadAiPresets()
	}
	return PresetList
}

// GetPresetNames 返回全部预设名（排序）。
func GetPresetNames() []string {
	g := GetAiPresets()
	names := make([]string, 0, len(g))
	for n := range g {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// GetPreset 返回指定预设（不存在返回空值）。
func GetPreset(name string) AiPreset {
	g := GetAiPresets()
	return g[name]
}

// SetAiPresets 全量替换预设定义。
func SetAiPresets(g AiPresets) {
	PresetList = g
}

// SaveAiPresets 持久化预设定义到 ai-presets.json。
func SaveAiPresets() error {
	p := AiPresetsPath()
	data, err := json.MarshalIndent(PresetList, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// EnsureAiPresets 确保预设已加载（不存在则建空映射，不落盘——用户未配置预设时保持默认行为）。
func EnsureAiPresets() {
	LoadAiPresets()
}

// AiPresetFromSettings 从当前 settings 抓取完整 AI 配置快照（保存预设时用）。
func AiPresetFromSettings() AiPreset {
	return AiPreset{
		Provider:         Settings.Provider,
		BaseURL:          Settings.BaseURL,
		APIKey:           Settings.APIKey,
		ExecuteModel:     Settings.ExecuteModel,
		PlanModel:        Settings.PlanModel,
		ReviewModel:      Settings.ReviewModel,
		Temperature:      Settings.Temperature,
		ThinkingMode:     Settings.ThinkingMode,
		MaxTokens:        Settings.MaxTokens,
		ContextMaxTokens: Settings.ContextMaxTokens,
	}
}

// ApplyPreset 应用预设：整份配置写回 settings（并记录当前预设名，供 UI 高亮）。
func ApplyPreset(name string, p AiPreset) {
	Settings.Preset = name
	if p.Provider != "" {
		Settings.Provider = p.Provider
	}
	if p.BaseURL != "" {
		Settings.BaseURL = p.BaseURL
	}
	if p.APIKey != "" {
		Settings.APIKey = p.APIKey
	}
	if p.ExecuteModel != "" {
		Settings.ExecuteModel = p.ExecuteModel
	}
	if p.PlanModel != "" {
		Settings.PlanModel = p.PlanModel
	}
	if p.ReviewModel != "" {
		Settings.ReviewModel = p.ReviewModel
	}
	if p.Temperature != "" {
		Settings.Temperature = p.Temperature
	}
	if p.ThinkingMode != "" {
		Settings.ThinkingMode = p.ThinkingMode
	}
	if p.MaxTokens > 0 {
		Settings.MaxTokens = p.MaxTokens
	}
	if p.ContextMaxTokens > 0 {
		Settings.ContextMaxTokens = p.ContextMaxTokens
	}
	Save()
}
