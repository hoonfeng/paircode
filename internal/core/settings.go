package core

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// AppSettings 持久化设置 —— 字段对齐参考 settings.ts（扁平存储；分组注释）。
// ★ 2026-08-21 AI 业务字段（provider/baseURL/apiKey/模型）加 omitempty：
//
//	settings 不再存 key/模型（唯一来源 ai-presets.json），Save 时不把空字段写回文件。
type AppSettings struct {
	Provider         string `json:"provider,omitempty"`
	Preset           string `json:"preset"` // ★ 2026-08-20 当前 AI 配置预设名（对话面板选预设时记录，UI 高亮用）
	BaseURL          string `json:"baseURL,omitempty"`
	APIKey           string `json:"apiKey,omitempty"`
	Model            string `json:"model,omitempty"` // 兼容旧单模型字段（迁移→ExecuteModel）
	PlanModel        string `json:"planModel,omitempty"`
	ExecuteModel     string `json:"executeModel,omitempty"`
	ReviewModel      string `json:"reviewModel,omitempty"`
	Temperature      string `json:"temperature"`
	ThinkingMode     string `json:"thinkingMode"`
	MaxTokens        int    `json:"maxTokens"`
	ContextMaxTokens int    `json:"contextMaxTokens"`
	// 工作区
	LastProject          string              `json:"lastProject"`
	WorkspaceFolders     []string            `json:"workspaceFolders"`
	WorkspaceFolderLists map[string][]string `json:"workspaceFolderLists"` // 每工作区的文件夹列表，key=工作区根目录
	RecentProjects       []string            `json:"recentProjects"`
	// Agent 行为
	ReviewMode         string   `json:"reviewMode"`      // 审核模式："auto"=AI审核, "manual"=手动审批, "off"=全部放行
	ReviewBlacklist    []string `json:"reviewBlacklist"` // 审核黑名单：命中此列表的工具需要审核（为空=全部审核）
	ReviewWhitelist    []string `json:"reviewWhitelist"` // 审核白名单：命中此列表的工具跳过审核（黑名单优先）
	Autonomous         bool     `json:"autonomous"`
	MaxIterations      int      `json:"maxIterations"`
	AutoIterate        bool     `json:"autoIterateOnRejection"`
	SystemInstructions string   `json:"systemInstructions"`
	IgnoreDirs         []string `json:"ignoreDirs"`
	// 外观
	Theme    string `json:"theme"`
	FontSize int    `json:"fontSize"`
	TabSize  int    `json:"tabSize"`
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
	ThinkingMode     string `json:"thinkingMode,omitempty"`     // OpenAI 思考档位 none/minimal/low/medium/high/xhigh/max，空=不覆盖
	MaxTokens        int    `json:"maxTokens,omitempty"`        // 最大输出 token，0=不覆盖
	ContextMaxTokens int    `json:"contextMaxTokens,omitempty"` // 上下文窗口，0=不覆盖
	Multimodal       bool   `json:"multimodal,omitempty"`       // ★ 2026-08-21 多模态：该模型支持图片输入（对话可直接粘贴/拖拽图片）
}

var (
	Settings AppSettings
	Loaded   bool
)

// ConfigDir 全局配置目录：安装目录（exe 所在）下的 config/ 子区。
// ★ exe 位于 bin/ 子目录时（如 bin/desktop.exe）回退到上级目录的 config/，
//
//	与根目录运行（companion.exe → ./config）共用同一份配置——否则桌面版
//	会读到 bin/config/settings.json 旧配置（工作区列表不全，只显示一个）。
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
// ★ 2026-09-09 修复：exe 位于 bin/ 子目录时回退到上级目录（与 ConfigDir 对齐）——
// 此前 bin 下的 exe 会把 .pair/（工具集/插件/资产）写到 bin\.pair，启动时又从
// bin\.pair 读旧盘数据，导致工具集编辑不生效、插件面与安装根不一致。
func InstallDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		low := strings.ToLower(dir)
		if !strings.Contains(low, "go-build") && !strings.Contains(low, `\temp\`) && !strings.Contains(low, "/tmp/") {
			if strings.EqualFold(filepath.Base(dir), "bin") {
				return filepath.Dir(dir)
			}
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
		// ★ 2026-08-21 AI 业务字段不再设默认（配置来源收敛到 ai-presets.json：
		//   装配按 settings.preset 展开；无预设时 models.json 服务商 key/baseURL 兜底。
		//   全局参数（温度/思考/输出/上下文）保留默认作为装配兜底。
		Provider: "", BaseURL: "", APIKey: "",
		PlanModel: "", ExecuteModel: "", ReviewModel: "",
		Temperature: "0.3", ThinkingMode: "high", MaxTokens: 131072, ContextMaxTokens: 64000,
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
			// ★ 2026-09 Round3（G4）：未知顶层键告警（不阻断）——配置模板漂移/拼写
			//   错误在启动日志可见，避免静默忽略。
			warnUnknownKeys(SettingsPath(), raw, settingsKnownKeys())
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
	// 确保 AI 配置预设已加载（ai-presets.json 不存在则建空映射）
	EnsureAiPresets()
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

// SyncWorkspaceFolderList 同步某个工作区的文件夹列表快照到 workspaceFolderLists。
// ★ 2026-08-22 修复「添加项目刷新后消失」：后端工作区变更（add-folder/
//
//	remove-folder/new-project/create/switch）后同步该字段——前端刷新页面时
//	loadWsList 从 workspaceFolderLists 恢复每个工作区条目的 folders，
//	此前只有前端 saveWsList 在明确事件时写它，add-folder 后不写导致
//	刷新后新项目丢失（folders 退化为根目录）。
func SyncWorkspaceFolderList(wsRoot string, folders []string) {
	if wsRoot == "" {
		return
	}
	if Settings.WorkspaceFolderLists == nil {
		Settings.WorkspaceFolderLists = map[string][]string{}
	}
	Settings.WorkspaceFolderLists[wsRoot] = append([]string(nil), folders...)
	// 兜底：确保工作区根也在 recentProjects（否则 loadWsList 不会列出该条目）
	found := false
	for _, p := range Settings.RecentProjects {
		if p == wsRoot {
			found = true
			break
		}
	}
	if !found {
		Settings.RecentProjects = append([]string{wsRoot}, Settings.RecentProjects...)
		if len(Settings.RecentProjects) > 20 {
			Settings.RecentProjects = Settings.RecentProjects[:20]
		}
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

// ─── 配置模板对齐 / 未知键告警（2026-09 Round3 G4）──────────────────

// structJSONKeys 反射取结构体全部 json tag 名（含 omitempty 字段）。
func structJSONKeys(v any) map[string]bool {
	t := reflect.TypeOf(v)
	known := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			known[name] = true
		}
	}
	return known
}

// settingsKnownKeys AppSettings 的全部 json 键（模板对齐测试与 Load 告警共用）。
func settingsKnownKeys() map[string]bool { return structJSONKeys(AppSettings{}) }

// warnUnknownKeys 输出 JSON 顶层键中不属于 known（且非已知遗留迁移键）的告警。
// log.Warn 级、不阻断加载——配置模板漂移/拼写错误在启动日志可见。
func warnUnknownKeys(path string, raw map[string]json.RawMessage, known map[string]bool) {
	// 已知遗留迁移键：Load 显式处理，不算未知
	legacy := map[string]bool{"autoReview": true, "editorFontSize": true}
	for k := range raw {
		if known[k] || legacy[k] {
			continue
		}
		log.Printf("[config] ⚠️ 未知配置键 %q（%s）——可能已改名/废弃，将被忽略", k, path)
	}
}
