package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ProviderEntry 一个服务商的信息，含 API 基地址和可用模型列表。
type ProviderEntry struct {
	BaseURL          string   `json:"baseURL"`
	Models           []string `json:"models"`
	APIKey           string   `json:"apiKey,omitempty"`           // ★ 2026-08-20 服务商独立 API Key（切换服务商自动带出）
	ContextMaxTokens int      `json:"contextMaxTokens,omitempty"` // ★ 2026-08-20 服务商级默认上下文窗口（Token；0=不限制/未配置，模型级可覆盖）
	// ★ 2026-09-02 LLM 协议：openai-completions（默认，OpenAI 兼容 /chat/completions）/
	//   openai-responses（OpenAI Responses /responses）/ anthropic-messages（Anthropic /messages）。
	//   BaseURL 此时为「基础地址」（不含协议路径，如 https://api.deepseek.com/v1），
	//   完整请求端点由内部按协议拼接；兼容旧数据：BaseURL 已含协议路径后缀时直接使用。
	Protocol string `json:"protocol,omitempty"`
}

// ModelListMap 按服务商分组，key=服务商名，value=ProviderEntry。
// 加载自 config/models.json（安装目录），运行时通过 GetModels / GetProviderBaseURL 查询。
type ModelListMap map[string]ProviderEntry

var (
	// ModelList 当前生效的模型列表（加载自安装目录或内置默认）。
	ModelList ModelListMap

	// defaultModels 仅在 models.json 不存在或解析失败时使用的兜底列表。
	// ★ 2026-08-20 与安装版 config/models.json 对齐（用户实际在用）：6 个服务商含「基元律动」网关。
	// ★ 2026-08-27 BaseURL 语义调整（2026-09-02 定稿）：值为「基础地址」（不含协议路径），
	// 完整端点（/chat/completions、/responses、/messages）由内部按 Protocol 拼接；
	// 已有数据若 BaseURL 已含协议路径后缀则直接使用（兼容，不重复拼接）。
	// ★ 2026-09-02 Anthropic 改用原生 messages 协议（旧配置曾错误写成 /v1/chat/completions）。
	defaultModels = ModelListMap{
		"anthropic":         {BaseURL: "https://api.anthropic.com/v1", Protocol: "anthropic-messages", Models: []string{"claude-3-5-sonnet-20241022", "claude-3-5-haiku-20241022", "claude-4-sonnet-20250514", "claude-4-haiku-latest"}},
		"custom":            {BaseURL: "", Models: []string{"custom"}},
		"deepseek":          {BaseURL: "https://api.deepseek.com/v1", Models: []string{"deepseek-v4-pro", "deepseek-v4-flash"}},
		"基元律动":              {BaseURL: "https://tokenrhythm.studio/v1", Models: []string{"deepseek-v4-pro-0813", "deepseek-v4-flash-0731"}},
		"kimi":              {BaseURL: "https://api.moonshot.cn/v1", Models: []string{"kimi-k3"}},
		"openai-compatible": {BaseURL: "", Models: []string{"custom"}},
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
		useDefaultModels()
		return
	}

	var fileModels ModelListMap
	if err := json.Unmarshal(data, &fileModels); err != nil {
		useDefaultModels()
		return
	}
	// ★ 2026-09 Round3（G4）：服务商条目未知键告警（不阻断）——配置模板漂移
	//   在启动日志可见（服务商名为自由键，只查条目内部字段）。
	if raw, ok := parseRawProviders(data); ok {
		known := structJSONKeys(ProviderEntry{})
		for prov, fields := range raw {
			warnUnknownKeys(p+" [服务商 "+prov+"]", fields, known)
		}
	}

	ModelList = fileModels
}

// parseRawProviders 把 models.json 解析为 服务商 → 字段 → 原始值（未知键告警用）。
func parseRawProviders(data []byte) (map[string]map[string]json.RawMessage, bool) {
	var m map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}
	return m, true
}

func useDefaultModels() {
	ModelList = make(ModelListMap, len(defaultModels))
	for k, v := range defaultModels {
		entry := ProviderEntry{BaseURL: v.BaseURL, Protocol: v.Protocol}
		entry.Models = make([]string, len(v.Models))
		copy(entry.Models, v.Models)
		ModelList[k] = entry
	}
}

// GetModels 返回指定服务商的可用模型列表。若不存在则返回空切片。
func GetModels(provider string) []string {
	if ModelList == nil {
		LoadModelList()
	}
	entry, ok := ModelList[provider]
	if !ok {
		return nil
	}
	return entry.Models
}

// GetProviderBaseURL 返回指定服务商的默认 API 地址。
func GetProviderBaseURL(provider string) string {
	if ModelList == nil {
		LoadModelList()
	}
	entry, ok := ModelList[provider]
	if !ok {
		return ""
	}
	return entry.BaseURL
}

// GetProviderProtocol 返回指定服务商的 LLM 协议（空=默认 openai-completions）。
func GetProviderProtocol(provider string) string {
	if ModelList == nil {
		LoadModelList()
	}
	entry, ok := ModelList[provider]
	if !ok {
		return ""
	}
	return entry.Protocol
}

// GetProviderProtocols 返回服务商 → LLM 协议映射（前端联动下拉用）。
func GetProviderProtocols() map[string]string {
	if ModelList == nil {
		LoadModelList()
	}
	out := make(map[string]string, len(ModelList))
	for k, v := range ModelList {
		out[k] = v.Protocol
	}
	return out
}

// GetProviderBaseURLs 返回全部服务商的默认 API 地址映射。
func GetProviderBaseURLs() map[string]string {
	if ModelList == nil {
		LoadModelList()
	}
	out := make(map[string]string, len(ModelList))
	for k, v := range ModelList {
		out[k] = v.BaseURL
	}
	return out
}

// GetProviderAPIKeys 返回服务商 → API Key 映射（切服务商自动带出，Key 按服务商独立保存）。
func GetProviderAPIKeys() map[string]string {
	if ModelList == nil {
		LoadModelList()
	}
	out := make(map[string]string, len(ModelList))
	for k, v := range ModelList {
		out[k] = v.APIKey
	}
	return out
}

// GetProviderContextMaxTokens 返回各服务商默认上下文窗口（Token；0=未配置）。
func GetProviderContextMaxTokens() map[string]int {
	if ModelList == nil {
		LoadModelList()
	}
	out := make(map[string]int, len(ModelList))
	for k, v := range ModelList {
		out[k] = v.ContextMaxTokens
	}
	return out
}

// GetProviderContextMaxToken 返回指定服务商的默认上下文窗口（0=未配置）。
func GetProviderContextMaxToken(provider string) int {
	if ModelList == nil {
		LoadModelList()
	}
	if v, ok := ModelList[provider]; ok {
		return v.ContextMaxTokens
	}
	return 0
}

// GetProviderAPIKey 返回指定服务商的 API Key（空=未配置）。
func GetProviderAPIKey(provider string) string {
	if ModelList == nil {
		LoadModelList()
	}
	if entry, ok := ModelList[provider]; ok {
		return entry.APIKey
	}
	return ""
}

// GetProviderEntry 返回指定服务商的完整条目（懒加载；不存在返回零值）。
// ★ 2026-09-03 插件数据面：JS 装配器经 ctx.models.get(provider) 读取
//   （baseURL/apiKey/protocol/contextMaxTokens/models），装配决策在插件内完成。
func GetProviderEntry(provider string) ProviderEntry {
	if ModelList == nil {
		LoadModelList()
	}
	return ModelList[provider]
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

// SetModelList 全量替换模型列表（面板保存用：providers 全量快照）。
func SetModelList(list ModelListMap) {
	ModelList = list
}

// AddProvider 添加新服务商（含 API 地址和模型列表）
func AddProvider(name, baseURL string, models []string) error {
	if ModelList == nil {
		LoadModelList()
	}
	if _, exists := ModelList[name]; exists {
		return fmt.Errorf("服务商 %q 已存在", name)
	}
	if name == "" {
		return fmt.Errorf("服务商名称不能为空")
	}
	ModelList[name] = ProviderEntry{BaseURL: baseURL, Models: models}
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
func UpdateProviderModels(name string, baseURL string, models []string) error {
	if ModelList == nil {
		LoadModelList()
	}
	entry, exists := ModelList[name]
	if !exists {
		return fmt.Errorf("服务商 %q 不存在", name)
	}
	entry.Models = models
	if baseURL != "" {
		entry.BaseURL = baseURL
	}
	ModelList[name] = entry
	return SaveModelList()
}

// RenameProvider 重命名服务商（同时更新 settings 中引用的 provider 名）
func RenameProvider(oldName, newName string) error {
	if ModelList == nil {
		LoadModelList()
	}
	entry, exists := ModelList[oldName]
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
	ModelList[newName] = entry
	// 更新 settings 中的 provider 引用
	if Settings.Provider == oldName {
		Settings.Provider = newName
	}
	Save()
	return SaveModelList()
}

// WriteDefaultModels 在安装目录下写入内置默认 models.json（仅在文件不存在时）。
func WriteDefaultModels() error {
	p := ModelsPath()
	if _, err := os.Stat(p); err == nil {
		return nil
	}
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
