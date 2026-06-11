// core 的设置数据层 —— appSettings 类型 + 生效设置 + 加载/保存 + 纯数据助手。
// 分层:数据在此(core),apply 逻辑(applyAgentSettings/applyTheme,碰面板/ui)与设置 UI(settingsBodyState/tab)
// 留 main，读 core.Settings。editingSettings(编辑缓冲)作 UI 态留 main(类型 core.AppSettings)。
//
//go:build windows

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/goui/internal/widget"
)

// AppSettings 持久化设置 —— 字段对齐参考 settings.ts（扁平存储；分组注释）。API Key 敏感，存安装目录 config/，不入库。
type AppSettings struct {
	// 主模型（model）
	Provider         string `json:"provider"`
	BaseURL          string `json:"baseURL"`
	APIKey           string `json:"apiKey"`
	Model            string `json:"model"` // 兼容旧单模型字段（迁移→ExecuteModel）
	PlanModel        string `json:"planModel"`
	ExecuteModel     string `json:"executeModel"`
	ReviewModel      string `json:"reviewModel"`
	Temperature      string `json:"temperature"`      // 字符串：留空=用服务端默认（区分显式 0）
	ThinkingMode     string `json:"thinkingMode"`     // non-thinking / thinking / thinking_max
	MaxTokens        int    `json:"maxTokens"`        // 0=不下发
	ContextMaxTokens int    `json:"contextMaxTokens"` // 上下文窗口上限
	// 压缩模型（compressModel）
	CompressEnabled      bool   `json:"compressEnabled"`
	CompressProvider     string `json:"compressProvider"`
	CompressAPIKey       string `json:"compressApiKey"`
	CompressBaseURL      string `json:"compressBaseURL"`
	CompressModel        string `json:"compressModel"`
	CompressThinkingMode string `json:"compressThinkingMode"`
	// 工作区
	LastProject      string   `json:"lastProject"`
	WorkspaceFolders []string `json:"workspaceFolders"`
	RecentProjects   []string `json:"recentProjects"` // 最近打开的文件夹（标题栏下拉用，最新在前、去重、上限 12）
	// Agent 行为
	AutoReview         bool   `json:"autoReview"`
	Autonomous         bool   `json:"autonomous"`
	AutoCollapse       bool   `json:"autoCollapse"`
	MaxIterations      int    `json:"maxIterations"`
	MaxParallel        int    `json:"maxParallelAgents"`
	ReviewRetries      int    `json:"maxReviewRetries"`
	AutoIterate        bool   `json:"autoIterateOnRejection"`
	RequireApproval    bool   `json:"requireHumanApprovalForDestructive"`
	AIReview           bool   `json:"aiReview"` // AI 审核：用审核模型自动把关写操作（驳回回灌建议，多 Agent 编排）
	LuaTools           bool   `json:"luaTools"` // Lua 自定义工具：.pair/tools/*.lua 沙箱热加载（动态增减/优化工具集）
	Benchmark          bool   `json:"enableBenchmarking"`
	SystemInstructions string `json:"systemInstructions"`
	SearxngURL         string `json:"searxngUrl"`
	// 全局额外忽略目录（搜索/知识库探索跳过，防上下文爆炸）；与项目级 .pair/ignore 合并叠加。
	IgnoreDirs []string `json:"ignoreDirs"`
	// 终端
	DefaultShell string `json:"defaultShell"` // auto / cmd / powershell / git-bash
	TermFontSize int    `json:"termFontSize"` // 0=默认 13
	TermEncoding string `json:"termEncoding"` // auto / utf-8 / gbk
	// 外观
	Theme          string `json:"theme"` // dark / light / high-contrast / solarized-light / dracula
	FontFamily     string `json:"fontFamily"`
	EditorFontSize int    `json:"editorFontSize"` // 0=默认 14
	// 编辑器字体样式（整体加粗/斜体/下划线）
	EditorFontBold      bool `json:"editorFontBold"`
	EditorFontItalic    bool `json:"editorFontItalic"`
	EditorFontUnderline bool `json:"editorFontUnderline"`
	// UI 字体：界面文字（label）的字体族 + 样式（字号仍各处自定，空族=默认）
	UIFontFamily    string `json:"uiFontFamily"`
	UIFontBold      bool   `json:"uiFontBold"`
	UIFontItalic    bool   `json:"uiFontItalic"`
	UIFontUnderline bool   `json:"uiFontUnderline"`
	HideMinimap     bool   `json:"hideMinimap"` // 反向存：零值=显示 minimap
	// 思想（philosophy）
	PhilosophyEnabled  bool              `json:"philosophyEnabled"`
	PhilosophySelected []string          `json:"philosophySelected"`
	PhilosophyRoles    map[string]string `json:"philosophyRoles"`
	// MCP / Skills
	AutoConnectMCP        bool            `json:"autoConnectMCP"`
	MCPEnabledOverrides   map[string]bool `json:"mcpEnabledOverrides"`   // 键 level::name → 启用态（覆盖默认）
	SkillEnabledOverrides map[string]bool `json:"skillEnabledOverrides"` // 同上，Skills 用

	// 自定义语言提供者（结构化编辑器用）：把新语言扩展名映射到已有 provider。
	CustomProviders []widget.CustomProviderConfig `json:"customProviders"`
}

var (
	Settings AppSettings // 生效设置
	Loaded   bool        // settings.json 是否存在（决定启动时是否覆盖 chat 内置默认）
)

// ConfigDir 全局配置目录：安装目录（exe 所在）下的 config/ 子区。go run 的临时 exe→回退 cwd/config。
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

// SettingsPath settings.json 路径。
func SettingsPath() string { return filepath.Join(ConfigDir(), "settings.json") }

// Default 默认值 —— 对齐参考 settings.ts 的 DEFAULTS。
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

// Load 读 settings.json 进 Settings（铺默认值再覆盖；迁移旧单模型字段）。返回是否有存档。
// apply（applyAgentSettings/applyFontFamily/SetSearxngURL/LoadCustomProviders）由 main 在调用后做（碰面板/ui）。
func Load() bool {
	Settings = Default()
	loaded := false
	if data, err := os.ReadFile(SettingsPath()); err == nil {
		_ = json.Unmarshal(data, &Settings)
		loaded = true
	}
	if Settings.ExecuteModel == "" && Settings.Model != "" { // 迁移旧单模型字段
		Settings.ExecuteModel = Settings.Model
	}
	Loaded = loaded
	return loaded
}

// Save 把 Settings 存盘（含 API Key → 仅本人可读）。
func Save() {
	p := SettingsPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if data, err := json.MarshalIndent(Settings, "", "  "); err == nil {
		_ = os.WriteFile(p, data, 0o600)
	}
}

// MainModel 主循环用的模型：执行模型优先（参考的 executeModel），回退旧 Model 字段。
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

// Temperature 解析温度：留空/非法→-1（不下发，用服务端默认）。
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

// FirstFontFamily 从 CSS 字体栈取首个具体族名（去引号/空白、跳过通用关键字）。
func FirstFontFamily(stack string) string {
	for _, part := range strings.Split(stack, ",") {
		f := strings.TrimSpace(strings.Trim(strings.TrimSpace(part), "'\""))
		switch strings.ToLower(f) {
		case "", "monospace", "serif", "sans-serif", "system-ui", "ui-monospace", "cursive", "fantasy":
			continue // 通用关键字 → 取下一个具体族
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
