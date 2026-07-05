//go:build windows

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hoonfeng/paircode/cmd/companion/codetypes"
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
	// 压缩模型
	CompressEnabled      bool   `json:"compressEnabled"`
	CompressProvider     string `json:"compressProvider"`
	CompressAPIKey       string `json:"compressApiKey"`
	CompressBaseURL      string `json:"compressBaseURL"`
	CompressModel        string `json:"compressModel"`
	CompressThinkingMode string `json:"compressThinkingMode"`
	// 工作区
	LastProject      string   `json:"lastProject"`
	WorkspaceFolders []string `json:"workspaceFolders"`
	RecentProjects   []string `json:"recentProjects"`
	// Agent 行为
	AutoReview         bool   `json:"autoReview"`
	Autonomous         bool   `json:"autonomous"`
	AutoCollapse       bool   `json:"autoCollapse"`
	MaxIterations      int    `json:"maxIterations"`
	MaxParallel        int    `json:"maxParallelAgents"`
	ReviewRetries      int    `json:"maxReviewRetries"`
	AutoIterate        bool   `json:"autoIterateOnRejection"`
	RequireApproval    bool   `json:"requireHumanApprovalForDestructive"`
	AIReview           bool   `json:"aiReview"`
	LuaTools           bool   `json:"luaTools"`
	Benchmark          bool   `json:"enableBenchmarking"`
	SystemInstructions string `json:"systemInstructions"`
	SearxngURL         string `json:"searxngUrl"`
	IgnoreDirs         []string `json:"ignoreDirs"`
	// 终端
	DefaultShell string `json:"defaultShell"`
	TermFontSize int    `json:"termFontSize"`
	TermEncoding string `json:"termEncoding"`
	// 外观
	Theme              string `json:"theme"`
	FontFamily         string `json:"fontFamily"`
	EditorFontSize     int    `json:"editorFontSize"`
	EditorFontBold      bool `json:"editorFontBold"`
	EditorFontItalic    bool `json:"editorFontItalic"`
	EditorFontUnderline bool `json:"editorFontUnderline"`
	UIFontFamily    string `json:"uiFontFamily"`
	UIFontBold      bool   `json:"uiFontBold"`
	UIFontItalic    bool   `json:"uiFontItalic"`
	UIFontUnderline bool   `json:"uiFontUnderline"`
	HideMinimap     bool   `json:"hideMinimap"`
	// 思想
	PhilosophyEnabled  bool              `json:"philosophyEnabled"`
	PhilosophySelected []string          `json:"philosophySelected"`
	PhilosophyRoles    map[string]string `json:"philosophyRoles"`
	// MCP / Skills
	AutoConnectMCP        bool            `json:"autoConnectMCP"`
	MCPEnabledOverrides   map[string]bool `json:"mcpEnabledOverrides"`
	SkillEnabledOverrides map[string]bool `json:"skillEnabledOverrides"`
	// 自定义语言提供者
	CustomProviders []codetypes.CustomProviderConfig `json:"customProviders"`
}

var (
	Settings AppSettings
	Loaded   bool
)

// ConfigDir 全局配置目录：安装目录（exe 所在）下的 config/ 子区。
func ConfigDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		low := strings.ToLower(dir)
		if !strings.Contains(low, "go-build") && !strings.Contains(low, `\temp\`) && !strings.Contains(low, "/tmp/") {
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
		Temperature: "1.0", ThinkingMode: "thinking", MaxTokens: 131072, ContextMaxTokens: 1000000,
		CompressEnabled: true, CompressProvider: "deepseek", CompressBaseURL: "https://api.deepseek.com/v1",
		CompressModel: "deepseek-v4-flash", CompressThinkingMode: "non-thinking",
		MaxIterations: 50, MaxParallel: 3, ReviewRetries: 3, AutoIterate: true, RequireApproval: true, Benchmark: true, LuaTools: true,
		DefaultShell: "auto", TermFontSize: 13, TermEncoding: "auto",
		Theme: "dark", EditorFontSize: 14, FontFamily: "'Cascadia Code', 'Fira Code', Consolas, monospace",
		PhilosophySelected: []string{"tao-te-ching", "huangdi-yinfu-jing", "sunzi-bingfa"},
		AutoConnectMCP:     true,
	}
}

// Load 读 settings.json 进 Settings。同时确保 models.json 存在并加载模型列表。
func Load() bool {
	Settings = Default()
	loaded := false
	if data, err := os.ReadFile(SettingsPath()); err == nil {
		_ = json.Unmarshal(data, &Settings)
		loaded = true
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
