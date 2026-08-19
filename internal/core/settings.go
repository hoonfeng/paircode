//go:build windows

package core

import (
	"encoding/json"

	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AppSettings 持久化设置 —— 字段对齐参考 settings.ts（扁平存储；分组注释）。
type AppSettings struct {
	Provider         string `json:"provider"`
	BaseURL          string `json:"baseURL"`
	APIKey           string `json:"apiKey"`
	Model            string `json:"model"` // 兼容旧单模型字段（迁移→ExecuteModel）
	PlanModel        string `json:"planModel"`
	ExecuteModel     string `json:"executeModel"`
	ReviewModel      string `json:"reviewModel"`
	Temperature      string `json:"temperature"`
	ThinkingMode     string `json:"thinkingMode"`
	MaxTokens        int    `json:"maxTokens"`
	ContextMaxTokens int    `json:"contextMaxTokens"`
	// 工作区
	LastProject        string              `json:"lastProject"`
	WorkspaceFolders   []string            `json:"workspaceFolders"`
	WorkspaceFolderLists map[string][]string `json:"workspaceFolderLists"` // 每工作区的文件夹列表，key=工作区根目录
	RecentProjects     []string            `json:"recentProjects"`
	// Agent 行为
	ReviewMode         string   `json:"reviewMode"`    // 审核模式："auto"=AI审核, "manual"=手动审批, "off"=全部放行
	ReviewBlacklist    []string `json:"reviewBlacklist"`    // 审核黑名单：命中此列表的工具需要审核（为空=全部审核）
	ReviewWhitelist    []string `json:"reviewWhitelist"`    // 审核白名单：命中此列表的工具跳过审核（黑名单优先）
	Autonomous         bool     `json:"autonomous"`
	MaxIterations      int    `json:"maxIterations"`
	AutoIterate        bool   `json:"autoIterateOnRejection"`
	SystemInstructions string `json:"systemInstructions"`
	IgnoreDirs         []string `json:"ignoreDirs"`
	// 外观
	Theme   string `json:"theme"`
	FontSize int   `json:"fontSize"`
	TabSize  int   `json:"tabSize"`
	// MCP / Skills
	SkillEnabledOverrides map[string]bool   `json:"skillEnabledOverrides"`
	SkillStatusOverrides  map[string]string `json:"skillStatusOverrides"`
	// 模型级参数（★ 2026-08-20）：每个模型独立配置生成参数，key=服务商 → 模型 → 参数
	ModelParams map[string]map[string]ModelParamEntry `json:"modelParams,omitempty"`
	// 插件配置（插件通过 ctx.registerSettings 注册命名空间，值存这里）
	PluginSettings map[string]map[string]any `json:"pluginSettings,omitempty"`
}

// ModelParamEntry 单个模型的独立生成参数（模型级配置，覆盖全局默认）。
type ModelParamEntry struct {
	Temperature      string `json:"temperature,omitempty"`      // 随机性（"0"~"2.0"，空=不覆盖）
	ThinkingMode     string `json:"thinkingMode,omitempty"`     // thinking/non-thinking，空=不覆盖
	MaxTokens        int    `json:"maxTokens,omitempty"`        // 最大输出 token，0=不覆盖
	ContextMaxTokens int    `json:"contextMaxTokens,omitempty"` // 上下文窗口，0=不覆盖
}

var (
	Settings AppSettings
	Loaded   bool
)

// ConfigDir 全局配置目录：安装目录（exe 所在）下的 config/ 子区。
// ★ exe 位于 bin/ 子目录时（如 bin/desktop.exe）回退到上级目录的 config/，
//   与根目录运行（companion.exe → ./config）共用同一份配置——否则桌面版
//   会读到 bin/config/settings.json 旧配置（工作区列表不全，只显示一个）。
func ConfigDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		low := strings.ToLower(dir)
		if !strings.Contains(low, "go-build") && !strings.Contains(low, `\temp\`) && !strings.Contains(low, "/tmp/") {
			if strings.EqualFold(filepath.Base(dir), "bin") {
				return filepath.Join(filepath.Dir(dir), "config")
			}
			return filepath.Join(dir, "config")
		}
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, "config")
}

// InstallDir 返回 exe 所在安装目录。
func InstallDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		low := strings.ToLower(dir)
		if !strings.Contains(low, "go-build") && !strings.Contains(low, `\temp\`) && !strings.Contains(low, "/tmp/") {
			return dir
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// SettingsPath settings.json 路径。
func SettingsPath() string { return filepath.Join(ConfigDir(), "settings.json") }

// Default 默认值。
func Default() AppSettings {
	return AppSettings{
		Provider: "deepseek", BaseURL: "https://api.deepseek.com/v1",
		PlanModel: "deepseek-v4-pro", ExecuteModel: "deepseek-v4-flash", ReviewModel: "deepseek-v4-pro",
		Temperature: "0.3", ThinkingMode: "thinking", MaxTokens: 131072, ContextMaxTokens: 64000,
		MaxIterations: 50, AutoIterate: true, ReviewMode: "auto",
		Theme: "dark", FontSize: 14, TabSize: 2,
	}
}

// Load 读 settings.json 进 Settings。同时确保 models.json 存在并加载模型列表。
func Load() bool {
	Settings = Default()
	loaded := false
	if data, err := os.ReadFile(SettingsPath()); err == nil {
		_ = json.Unmarshal(data, &Settings)
		loaded = true
		// ★ 迁移旧版 autoReview 字段 → reviewMode
		// 旧版：autoReview=true → "auto", autoReview=false → "manual"（原非自主模式=手动审批）
		// 检查逻辑：如果 JSON 中有 autoReview 字段但无 reviewMode 字段，才执行迁移
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			_, hasNew := raw["reviewMode"]
			oldVal, hasOld := raw["autoReview"]
			if !hasNew && hasOld {
				var oldBool bool
				if json.Unmarshal(oldVal, &oldBool) == nil {
					if oldBool {
						Settings.ReviewMode = "auto"
					} else {
						Settings.ReviewMode = "manual"
					}
				}
			}
		}
		// ★ 迁移旧版 editorFontSize 字段 → fontSize（2026-08-19 字段名对齐前端消费）
		if _, hasNew := raw["fontSize"]; !hasNew {
			if v, hasOld := raw["editorFontSize"]; hasOld {
				var n int
				if json.Unmarshal(v, &n) == nil && n > 0 {
					Settings.FontSize = n
				}
			}
		}
	}
	if Settings.ExecuteModel == "" && Settings.Model != "" {
		Settings.ExecuteModel = Settings.Model
	}

	Loaded = loaded
	// 确保模型列表已加载（models.json 不存在则自动写入默认）
	EnsureModelList()
	return loaded
}

// Save 把 Settings 存盘。
func Save() {
	
	p := SettingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if data, err := json.MarshalIndent(Settings, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o600)
	}
}

// MainModel 主循环用的模型：执行模型优先，回退旧 Model 字段。
func MainModel() string {
	if Settings.ExecuteModel != "" {
		return Settings.ExecuteModel
	}
	return Settings.Model
}

// Configured 是否已配好可用 Provider。
func Configured() bool {
	return Settings.APIKey != "" && Settings.BaseURL != "" && MainModel() != ""
}

// Temperature 解析温度：留空/非法→-1。
func Temperature() float64 {
	s := strings.TrimSpace(Settings.Temperature)
	if s == "" {
		return -1
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	return -1
}

// FirstFontFamily 从 CSS 字体栈取首个具体族名。
func FirstFontFamily(stack string) string {
	for _, part := range strings.Split(stack, ",") {
		f := strings.TrimSpace(strings.Trim(strings.TrimSpace(part), "'\""))
		switch strings.ToLower(f) {
		case "", "monospace", "serif", "sans-serif", "system-ui", "ui-monospace", "cursive", "fantasy":
			continue
		}
		return f
	}
	return ""
}

// ParseTempOr 解析温度字符串，失败返回默认值。
func ParseTempOr(s string, def float64) float64 {
	if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return v
	}
	return def
}
