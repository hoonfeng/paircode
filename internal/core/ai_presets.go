// ai_presets.go — AI 配置预设（AI Preset）存储
//
// ★ 2026-08-20 AI 配置预设：用户把「一份完整 AI 配置」命名保存为预设，
//   供对话面板快速切换（多套配置，选中整套生效）。
//   每条预设 = 完整配置快照（provider/baseURL/apiKey/执行模型/温度/思考档位/
//   输出上限/上下文窗口）。
//
// ★ 2026-08-21 配置来源收敛 + 统一模型：
//   · settings 不再冗余存 key/模型——只存 preset（当前激活预设名）；
//     装配时按 preset 从本文件展开整套配置（ai-presets.json 是唯一配置来源）。
//   · 不再拆分 规划/审核 模型，统一用一个模型（executeModel）。
//
// ★ 2026-09-01 Key 回归 AI 配置：预设携带 API Key（用户习惯在 AI 配置里填 Key，
//   服务商只维护 地址/模型/参数）。装配时预设 Key 优先，服务商级 Key 仅兜底。
//
// 存储：config/ai-presets.json（安装目录）
//   { "<预设名>": { provider, baseURL, apiKey, executeModel, temperature,
//                   thinkingMode, maxTokens, contextMaxTokens } }
//
// 与 model-groups（组内挂实例）不同：预设是「整份配置的命名快照」，
// 保存时从表单/当前配置抓取全部 AI 业务字段；应用时只记录 preset 名。

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
	BaseURL          string `json:"baseURL,omitempty"`          // API 端点（基础地址；协议路径由内部按 Protocol 拼接）
	APIKey           string `json:"apiKey,omitempty"`           // API Key（预设自带，应用时写入 settings）
	ExecuteModel     string `json:"executeModel,omitempty"`     // 模型（统一模型：规划/审核/执行共用，2026-08-21 不再拆分）
	PlanModel        string `json:"planModel,omitempty"`        // ★ 旧字段保留兼容（历史数据）；新数据不写，装配统一用 ExecuteModel
	ReviewModel      string `json:"reviewModel,omitempty"`      // ★ 旧字段保留兼容（历史数据）；新数据不写，装配统一用 ExecuteModel
	Temperature      string `json:"temperature,omitempty"`      // 随机性（字符串保留原格式）
	ThinkingMode     string `json:"thinkingMode,omitempty"`     // 思考档位
	MaxTokens        int    `json:"maxTokens,omitempty"`        // 输出上限
	ContextMaxTokens int    `json:"contextMaxTokens,omitempty"` // 上下文窗口
	// ★ 2026-09-02 LLM 协议（空=继承服务商）：openai-completions / openai-responses / anthropic-messages。
	Protocol string `json:"protocol,omitempty"`
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

// GetPresetAPIKeyForProvider 返回指定服务商下任一预设携带的 API Key
// （会话级切换服务商时，Key 以该服务商预设中的 Key 为准；无则返回空串）。
// ★ 2026-09-01 Key 回归 AI 配置：对话切换服务商 → 用该服务商预设里的 Key。
func GetPresetAPIKeyForProvider(provider string) string {
	if provider == "" {
		return ""
	}
	for _, p := range GetAiPresets() {
		if p.Provider == provider && p.APIKey != "" {
			return p.APIKey
		}
	}
	return ""
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

// AiPresetFromSettings 从当前 settings 抓取 AI 配置快照（保存预设时用；未传 preset 的兜底）。
// ★ 2026-08-21 统一模型：不再拆分 规划/审核 模型，plan/review 与执行模型一致。
func AiPresetFromSettings() AiPreset {
	return AiPreset{
		Provider:         Settings.Provider,
		BaseURL:          Settings.BaseURL,
		APIKey:           Settings.APIKey,
		ExecuteModel:     Settings.ExecuteModel,
		PlanModel:        Settings.ExecuteModel,
		ReviewModel:      Settings.ExecuteModel,
		Temperature:      Settings.Temperature,
		ThinkingMode:     Settings.ThinkingMode,
		MaxTokens:        Settings.MaxTokens,
		ContextMaxTokens: Settings.ContextMaxTokens,
	}
}

// ApplyPreset 应用预设：只记录当前预设名（settings 不再冗余存 key/模型/参数，
// 装配时按 settings.preset 从 ai-presets.json 读整套配置）。
func ApplyPreset(name string, p AiPreset) {
	Settings.Preset = name
	Save()
}
