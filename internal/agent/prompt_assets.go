// prompt_assets.go —— 提示词插件化：提示词资产（Prompt Asset）注册表
//
// 「一切皆插件」延伸（2026-09）：所有提示词（系统提示/角色提示/评测提示等）
// 统一为「提示词资产」，由相关插件承载，形态二选一：
//   · 插件内置：插件包内 prompts/<name>.md（随包发布；启动扫描注册为磁盘资产）
//   · 插件+插件配置：package.json config.prompts 映射（装载时注册）
//     或运行时 ctx.prompts.provide({name, text})（JS 插件）/ Node 桥 prompts.provide
//
// 查找顺序（LoadPrompt 统一入口，先到先得）：
//   ① 运行时注册（插件配置 config.prompts 装载时注册；ctx.prompts.provide 运行时注册）
//   ② 磁盘插件资产（<插件包>/prompts/<name>.md，插件内置）
//   ③ 旧式覆盖（config/roles/<name>.md，向后兼容既有部署）
//   ④ ""（调用方用 Go 内置回退；内置提示安全网，插件资产缺席时逐字节一致）
//
// 与插件生命周期联动：插件卸载（Unload）时按来源清理其注册资产
// （见 plugin.go 的清理点）；磁盘资产随包存在（卸载 JS 定义不删磁盘文件，
// 重启后重新扫描注册——与「磁盘插件复活」语义一致）。
//
// 模板变量：资产文本支持 {{NAME}} 占位符（仅替换已知变量，未知变量原样保留，
// 宽松语义——资产是用户内容，不应因未知变量报错）。

package agent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// promptAsset 提示词资产条目。
type promptAsset struct {
	Name   string // 资产名（如 reviewer / planner / judge / system-harness / system-full）
	Source string // 来源标识（js:<plugin> / node:<plugin> / <plugin>:config / <plugin>:disk）
	Text   string // 资产文本
}

// 提示词资产注册表（进程级；分两层：运行时注册 + 磁盘插件资产）。
var (
	promptAssetsMu    sync.RWMutex
	promptRuntime     = map[string]promptAsset{} // ① 运行时注册（插件配置 / ctx.prompts.provide）
	promptDiskAssets  = map[string]promptAsset{} // ② 磁盘插件资产（插件包 prompts/ 目录）
	promptDiskPlugins = map[string]bool{}        // 已扫描的插件包目录（防重复扫描）
)

// ProvidePrompt 注册/覆盖提示词资产（运行时）。text 为空白时视为移除（防脏注册）。
// source 标识资产来源（插件卸载时按来源清理）。
func ProvidePrompt(name, text, source string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	promptAssetsMu.Lock()
	defer promptAssetsMu.Unlock()
	if strings.TrimSpace(text) == "" {
		delete(promptRuntime, name)
		return
	}
	promptRuntime[name] = promptAsset{Name: name, Source: source, Text: text}
}

// RemovePrompt 移除指定资产名的运行时注册（磁盘资产与旧式覆盖不受影响）。
func RemovePrompt(name string) {
	promptAssetsMu.Lock()
	delete(promptRuntime, name)
	promptAssetsMu.Unlock()
}

// RemovePromptSource 按来源移除全部运行时注册（插件卸载清理用）。
func RemovePromptSource(source string) {
	if source == "" {
		return
	}
	promptAssetsMu.Lock()
	for name, a := range promptRuntime {
		if a.Source == source {
			delete(promptRuntime, name)
		}
	}
	promptAssetsMu.Unlock()
}

// ScanPluginPromptAssets 扫描插件包内 prompts/ 目录注册磁盘资产（插件内置形态）。
// 约定：<pkgDir>/prompts/<name>.md → 资产名 <name>（不含 .md；一级目录，不递归）。
// 同资产名多插件贡献：先扫描者优先（后到不覆盖，避免装载顺序影响结果）。
// 返回注册成功的资产数。
func ScanPluginPromptAssets(pkgDir, plugin string) int {
	if pkgDir == "" || plugin == "" {
		return 0
	}
	promptsDir := filepath.Join(pkgDir, "prompts")
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return 0
	}
	promptAssetsMu.Lock()
	defer promptAssetsMu.Unlock()
	if promptDiskPlugins[plugin] {
		return 0 // 已扫描过（防重复注册）
	}
	promptDiskPlugins[plugin] = true
	n := 0
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		if strings.TrimSpace(name) == "" {
			continue
		}
		key := "disk:" + plugin + ":" + name // 磁盘资产内部键（按插件+名唯一）
		if _, exists := promptDiskAssets[key]; exists {
			continue
		}
		b, err := os.ReadFile(filepath.Join(promptsDir, e.Name()))
		if err != nil {
			continue
		}
		// ★ 保留原文（含尾部换行等）——系统提示资产必须与内置文本逐字节一致
		//   （默认环境零变化；仅 TrimSpace 判空防脏文件）。
		if strings.TrimSpace(string(b)) == "" {
			continue
		}
		promptDiskAssets[key] = promptAsset{Name: name, Source: plugin + ":disk", Text: string(b)}
		n++
	}
	return n
}

// LoadPrompt 统一提示词加载入口：运行时注册 > 磁盘插件资产 > config/roles 旧式覆盖 > ""。
func LoadPrompt(name string) string {
	a, _ := LoadPromptAsset(name)
	return a.Text
}

// LoadPromptAsset 同 LoadPrompt 但返回完整资产（含来源；诊断用）。
func LoadPromptAsset(name string) (promptAsset, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return promptAsset{}, false
	}
	promptAssetsMu.RLock()
	if a, ok := promptRuntime[name]; ok {
		promptAssetsMu.RUnlock()
		return a, true
	}
	// 磁盘资产：same-name 先到先得（按插件名排序保证确定性）
	var keys []string
	for k, a := range promptDiskAssets {
		if a.Name == name {
			keys = append(keys, k)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
		a := promptDiskAssets[keys[0]]
		promptAssetsMu.RUnlock()
		return a, true
	}
	promptAssetsMu.RUnlock()
	// 旧式覆盖（config/roles/<name>.md；向后兼容）
	if f := rolePromptFilePath(name); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			if s := strings.TrimSpace(string(b)); s != "" {
				return promptAsset{Name: name, Source: "config/roles:" + name, Text: s}, true
			}
		}
	}
	return promptAsset{}, false
}

// ResolvePromptVars 资产模板变量插值：仅替换已知变量，未知变量原样保留（宽松）。
func ResolvePromptVars(text string, vars map[string]string) string {
	if text == "" || len(vars) == 0 {
		return text
	}
	return promptVarRe.ReplaceAllStringFunc(text, func(m string) string {
		key := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(m, "{{"), "}}"))
		if v, ok := vars[key]; ok {
			return v
		}
		return m
	})
}

// PromptAssetsSnapshot 当前注册表快照（诊断/测试用）：
// 返回 {name, source, len} 列表，按 name 排序。
func PromptAssetsSnapshot() []struct {
	Name   string
	Source string
	Length int
} {
	promptAssetsMu.RLock()
	defer promptAssetsMu.RUnlock()
	out := make([]struct {
		Name   string
		Source string
		Length int
	}, 0, len(promptRuntime)+len(promptDiskAssets))
	for _, a := range promptRuntime {
		out = append(out, struct {
			Name   string
			Source string
			Length int
		}{a.Name, a.Source, len(a.Text)})
	}
	for _, a := range promptDiskAssets {
		out = append(out, struct {
			Name   string
			Source string
			Length int
		}{a.Name, a.Source, len(a.Text)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResetPromptAssetsForTest 清空注册表（测试隔离用）。
func ResetPromptAssetsForTest() {
	promptAssetsMu.Lock()
	promptRuntime = map[string]promptAsset{}
	promptDiskAssets = map[string]promptAsset{}
	promptDiskPlugins = map[string]bool{}
	promptAssetsMu.Unlock()
}

// registerConfigPrompts 从插件配置提取 prompts 映射并注册（插件+插件配置形态）。
// config 形如 {"prompts": {"reviewer": "...", "planner": "..."}}。
func registerConfigPrompts(config map[string]any, plugin string) int {
	if config == nil {
		return 0
	}
	raw, ok := config["prompts"]
	if !ok {
		return 0
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return 0
	}
	n := 0
	for name, v := range m {
		text, ok := v.(string)
		if !ok || strings.TrimSpace(name) == "" {
			continue
		}
		ProvidePrompt(name, text, plugin+":config")
		n++
	}
	return n
}
