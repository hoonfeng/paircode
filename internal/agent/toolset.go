// toolset.go — 工具集（Toolset）：命名插件包 + 动态构建/固化/导出/导入。
//
// 背景：工作区工具能力 = 内置工具 + Lua 工具 + 插件贡献。工具集把「为某个
// 项目/需求动态组合的插件集合」固化成可复用、可分享的单元：
//
//	.pair/toolsets/<name>.json     工作区级（项目专属，固化）
//	<installDir>/.pair/toolsets/   全局级（跨项目可用）
//
// 每个工具集是一个 Toolset：{ name, description, project, version, plugins[] }，
// plugins 为 JS 动态插件定义（host 半，与 cordis_define 的 code 参数同形态）。
// 装载走 PluginHost.DefineJSCodeFull + LoadJSDynamic——工具集即插件，插件化闭环。
//
// 动态构建（toolset_build）：无工具集配置时，分析项目（语言/框架/依赖/入口）
// + 要求描述 → 模板组合生成插件代码 → 定义装载 → 固化到工作区。
// 显式调用 toolset_build 可随时重新分析并更新工具集。
//
// 导出/导入：toolset_export 生成可移植 JSON（含全部插件代码），可导入全局
// （跨项目）或发布到市场（GitHub 仓库 / 本地注册表）。

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// ─── 数据模型 ─────────────────────────────────────────────

// Toolset 工具集 = 命名插件包。
type Toolset struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Project     string          `json:"project,omitempty"`  // 适用项目（basename，多项目区分）
	Version     string          `json:"version,omitempty"`  // 语义版本
	CreatedAt   string          `json:"createdAt,omitempty"`
	Plugins     []ToolsetPlugin `json:"plugins"`
}

// ToolsetPlugin 工具集内一个插件定义（host/client 双半）。
type ToolsetPlugin struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Code    string `json:"code"`           // host 半：async 函数体（return { name, apply(ctx) }）
	Client  string `json:"client,omitempty"` // client 半：(ui) => void
	// DisabledTools 插件保留、但被手动摘除的工具（toolset_edit rm_tool）。
	// 装载后应用：Registry.SetToolEnabled(false) → agent 工具列表不可见；
	// 工具仍注册在案（可逆，重新 edit 可恢复）。
	DisabledTools []string `json:"disabledTools,omitempty"`
}

// ─── 目录解析 ─────────────────────────────────────────────

// toolsetDirScope 工具集作用域。
type toolsetScope string

const (
	toolsetProject toolsetScope = "project" // 工作区 .pair/toolsets/
	toolsetGlobal  toolsetScope = "global"  // 安装目录 .pair/toolsets/
)

// toolsetDir 返回指定作用域的工具集目录。
func toolsetDir(projectRoot string, scope toolsetScope) string {
	if scope == toolsetGlobal {
		return filepath.Join(core.InstallDir(), ".pair", "toolsets")
	}
	return filepath.Join(projectRoot, ".pair", "toolsets")
}

// toolsetPath 返回指定工具集的完整文件路径。
func toolsetPath(projectRoot string, scope toolsetScope, name string) string {
	return filepath.Join(toolsetDir(projectRoot, scope), name+".json")
}

// ─── 序列化 ───────────────────────────────────────────────

// ToolsetMeta 工具集元信息（列表用，不含插件代码）。
type ToolsetMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Project     string `json:"project,omitempty"`
	Version     string `json:"version,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	Scope       string `json:"scope"`             // project | global
	PluginCount int    `json:"pluginCount"`
}

// listToolsets 列出指定作用域全部工具集元信息（按名排序）。
func listToolsets(projectRoot string, scope toolsetScope) []ToolsetMeta {
	dir := toolsetDir(projectRoot, scope)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []ToolsetMeta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var ts Toolset
		if err := json.Unmarshal(data, &ts); err != nil || ts.Name == "" {
			continue
		}
		out = append(out, ToolsetMeta{
			Name: ts.Name, Description: ts.Description, Project: ts.Project,
			Version: ts.Version, CreatedAt: ts.CreatedAt,
			Scope: string(scope), PluginCount: len(ts.Plugins),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// listAllToolsets 列出工作区 + 全局全部工具集（全局在前，工作区同名覆盖标记）。
// 安装目录与工作区相同时（开发环境常见）目录去重，避免重复显示。
func listAllToolsets(projectRoot string) []ToolsetMeta {
	globalDir := toolsetDir(projectRoot, toolsetGlobal)
	projectDir := toolsetDir(projectRoot, toolsetProject)
	merged := listToolsets(projectRoot, toolsetGlobal)
	if filepath.Clean(globalDir) == filepath.Clean(projectDir) {
		return merged
	}
	seen := map[string]bool{}
	for _, g := range merged {
		seen[g.Name] = true
	}
	for _, p := range listToolsets(projectRoot, toolsetProject) {
		if seen[p.Name] {
			p.Name += "（工作区覆盖）"
		}
		merged = append(merged, p)
	}
	return merged
}

// loadToolset 读取指定工具集（scope 为空时先查工作区再查全局）。
func loadToolset(projectRoot string, scope toolsetScope, name string) (*Toolset, error) {
	if name == "" {
		return nil, fmt.Errorf("工具集名不能为空")
	}
	paths := []string{}
	if scope != "" {
		paths = append(paths, toolsetPath(projectRoot, scope, name))
	} else {
		paths = append(paths, toolsetPath(projectRoot, toolsetProject, name))
		paths = append(paths, toolsetPath(projectRoot, toolsetGlobal, name))
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var ts Toolset
		if err := json.Unmarshal(data, &ts); err != nil || ts.Name == "" {
			continue
		}
		return &ts, nil
	}
	return nil, fmt.Errorf("工具集 %q 未找到（工作区与全局均无）", name)
}

// saveToolset 固化工具集到指定作用域（原子写：tmp + rename）。
func saveToolset(projectRoot string, scope toolsetScope, ts *Toolset) error {
	if ts == nil || ts.Name == "" {
		return fmt.Errorf("工具集名不能为空")
	}
	if len(ts.Plugins) == 0 {
		return fmt.Errorf("工具集 %s 没有插件", ts.Name)
	}
	if ts.Version == "" {
		ts.Version = "1.0.0"
	}
	if ts.CreatedAt == "" {
		ts.CreatedAt = time.Now().Format(time.RFC3339)
	}
	dir := toolsetDir(projectRoot, scope)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建工具集目录失败: %w", err)
	}
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return fmt.Errorf("工具集序列化失败: %w", err)
	}
	path := toolsetPath(projectRoot, scope, ts.Name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("写入工具集失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("固化工具集失败: %w", err)
	}
	return nil
}

// removeToolset 删除工具集文件（scope 为空时两个作用域都删）。
func removeToolset(projectRoot string, scope toolsetScope, name string) error {
	if name == "" {
		return fmt.Errorf("工具集名不能为空")
	}
	if scope != "" {
		return os.Remove(toolsetPath(projectRoot, scope, name))
	}
	var errs []string
	for _, s := range []toolsetScope{toolsetProject, toolsetGlobal} {
		p := toolsetPath(projectRoot, s, name)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("删除工具集 %s 失败: %s", name, strings.Join(errs, "; "))
	}
	return nil
}

// ─── 装载（启动时自动装配 + 构建后装载）──────────────────

// LoadAllToolsets 装载工作区 + 全局全部工具集（启动时调用；失败不致命）。
// 先全局后工作区（工作区同名覆盖全局，最后装载生效）。
func LoadAllToolsets(ph *PluginHost, projectRoot string) {
	if ph == nil || projectRoot == "" {
		return
	}
	loaded := 0
	for _, scope := range []toolsetScope{toolsetGlobal, toolsetProject} {
		for _, meta := range listToolsets(projectRoot, scope) {
			ts, err := loadToolset(projectRoot, scope, meta.Name)
			if err != nil {
				continue
			}
			if err := installToolset(ph, ts); err != nil {
				log.Printf("[toolset] %s 装载失败: %v", meta.Name, err)
				continue
			}
			loaded++
		}
	}
	if loaded > 0 {
		log.Printf("[toolset] 已装载 %d 个工具集（%d 个插件）", loaded, countAllToolsetPlugins(projectRoot))
	}
}

// ─── 动态构建（模板驱动）─────────────────────────────────

// BuildToolset 分析项目 + 收集模板 → 匹配 → 生成插件 → 定义装载 → 返回工具集
// （尚未落盘，由调用方 saveToolset 固化）。
// requirement 为可选要求描述（模板 generate 可参考裁剪插件）。
func BuildToolset(ph *PluginHost, projectDir, name, description, requirement string) (*Toolset, error) {
	if ph == nil {
		return nil, fmt.Errorf("插件宿主未初始化")
	}
	if projectDir == "" {
		return nil, fmt.Errorf("项目目录不能为空")
	}
	if name == "" {
		name = "default"
	}
	profile := analyzeProject(projectDir)

	// ★ LLM 项目意图分析（可选）：理解项目「实际要实现的目的」并推荐工具类别。
	// 无 provider / 调用失败 / 解析失败 → 回退纯静态分析，不影响主流程。
	var intent *ProjectIntent
	if prov := toolsetLLMProvider(); prov != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		it, err := llmAnalyzeProject(ctx, prov, projectDir, profile, requirement)
		cancel()
		if err != nil {
			log.Printf("[toolset] LLM 项目分析跳过（回退静态特征）: %v", err)
		} else if it != nil {
			intent = it
			log.Printf("[toolset] LLM 项目分析: %q（推荐 %v）", it.Purpose, it.RecommendedTags)
		}
	}

	var plugins []ToolsetPlugin
	var used []string
	var tplErrs []string
	genReq := requirement
	if intent != nil && strings.TrimSpace(intent.Notes) != "" {
		genReq = strings.TrimSpace(intent.Notes + " " + requirement)
	}
	// 生成 profile：LLM 分析出的真实命令合入（无 LLM 时保持静态探测结果）
	genProfile := *profile
	if intent != nil {
		intent.applyToProfile(&genProfile)
	}
	for _, t := range ph.Templates() {
		if !t.matches(profile) {
			// 静态特征未命中 → 意图标签补充命中（如 LLM 识别出 API 项目）
			if !t.matchesIntent(intent) {
				continue
			}
		}
		gs, err := t.generate(&genProfile, genReq)
		if err != nil {
			tplErrs = append(tplErrs, fmt.Sprintf("%s: %v", t.ID, err))
			continue
		}
		if len(gs) == 0 {
			continue
		}
		plugins = append(plugins, gs...)
		used = append(used, t.ID)
	}
	// ★ LLM 现场生成的项目专属插件并入（模板覆盖不到的能力缺口；对齐 deepseek-harness
	// 「模型所写插件」模式——注册时即校验：define 预检失败剔除并给指导性错误信息，
	// 不因单个 LLM 插件问题阻塞整个工具集）。
	if intent != nil && len(intent.CustomPlugins) > 0 {
		have := map[string]bool{}
		for _, e := range plugins {
			have[e.Name] = true
		}
		for _, cp := range intent.CustomPlugins {
			if have[cp.Name] {
				log.Printf("[toolset] customPlugin %s 与模板产物重名，跳过", cp.Name)
				continue
			}
			if _, err := ph.DefineJSCodeFull(cp.Code, "", cp.Purpose, "", ""); err != nil {
				log.Printf("[toolset] customPlugin %s 预检失败已剔除: %v（插件需为纯 JS：return { name, inject, apply(ctx) }，工具经 ctx.tools.register 注册）", cp.Name, err)
				continue
			}
			plugins = append(plugins, ToolsetPlugin{Name: cp.Name, Purpose: cp.Purpose, Code: cp.Code})
			have[cp.Name] = true
			log.Printf("[toolset] 并入 LLM 现场生成插件: %s（%s）", cp.Name, cp.Purpose)
		}
	}
	if len(plugins) == 0 {
		msg := "没有工具集模板适用于该项目"
		if len(tplErrs) > 0 {
			msg += "；模板错误: " + strings.Join(tplErrs, "; ")
		}
		return nil, fmt.Errorf("%s（检测到语言 %s）", msg, strings.Join(profile.Langs, "/"))
	}
	desc := strings.TrimSpace(description)
	if desc == "" {
		if intent != nil && strings.TrimSpace(intent.Purpose) != "" {
			desc = intent.Purpose // LLM 理解的项目目的作为工具集描述
		} else {
			desc = fmt.Sprintf("为项目 %s 动态构建的工具集（模板: %s）", profile.Name, strings.Join(used, ", "))
		}
	}
	ts := &Toolset{
		Name:        name,
		Description: desc,
		Project:     profile.Name,
		Plugins:     plugins,
	}
	if err := installToolset(ph, ts); err != nil {
		return nil, fmt.Errorf("工具集装载失败: %w", err)
	}
	return ts, nil
}

// ─── 市场发布导出 ─────────────────────────────────────────

// ToolsetPublish 工具集发布包（导出到市场/GitHub 的可移植格式）。
type ToolsetPublish struct {
	SchemaVersion string   `json:"schemaVersion"`
	Kind          string   `json:"kind"` // plugin | toolset
	Toolset       Toolset  `json:"toolset"`
	Tags          []string `json:"tags,omitempty"`
	Author        string   `json:"author,omitempty"`
	Repository    string   `json:"repository,omitempty"` // 发布目标仓库（github:owner/repo）
	Readme        string   `json:"readme,omitempty"`
}

// ExportToolsetJSON 序列化工具集为可移植 JSON（marketplace 发布格式）。
func ExportToolsetJSON(ts *Toolset, tags []string, author, repo string) (string, error) {
	if ts == nil || ts.Name == "" {
		return "", fmt.Errorf("工具集为空")
	}
	pub := ToolsetPublish{
		SchemaVersion: "1.0",
		Kind:          "toolset",
		Toolset:       *ts,
		Tags:          tags,
		Author:        author,
		Repository:    repo,
		Readme:        fmt.Sprintf("# 工具集：%s\n\n%s\n\n## 包含插件\n%s",
			ts.Name, ts.Description, toolsetPluginList(ts)),
	}
	data, err := json.MarshalIndent(pub, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化失败: %w", err)
	}
	return string(data), nil
}

// toolsetPluginList 插件清单文本（发布 README 用）。
func toolsetPluginList(ts *Toolset) string {
	var b strings.Builder
	for _, p := range ts.Plugins {
		fmt.Fprintf(&b, "- `%s`：%s\n", p.Name, p.Purpose)
	}
	return b.String()
}

// ─── 公开封装（handler REST / 前端面板直调）───────────────

// ListAllToolsetsPublic 全部工具集元信息（工作区 + 全局）。
func ListAllToolsetsPublic(root string) []ToolsetMeta {
	out := listAllToolsets(root)
	if out == nil {
		out = []ToolsetMeta{}
	}
	return out
}

// ResolveWorkspaceProjectPublic 解析 project 参数（多项目）。
func ResolveWorkspaceProjectPublic(primaryRoot, project string) (string, error) {
	return resolveWorkspaceProject(primaryRoot, project)
}

// ValidToolsetName 校验工具集名。
func ValidToolsetName(name string) bool { return validToolsetName(name) }

// ToolsetPath 工具集文件路径。
func ToolsetPath(root, scope, name string) string {
	return toolsetPath(root, toolsetScope(scope), name)
}

// LoadToolsetPublic 读取工具集（scope 空=先工作区再全局）。
func LoadToolsetPublic(root, scope, name string) (*Toolset, error) {
	return loadToolset(root, toolsetScope(scope), name)
}

// SaveToolsetPublic 固化工具集。
func SaveToolsetPublic(root, scope string, ts *Toolset) error {
	return saveToolset(root, toolsetScope(scope), ts)
}

// ParseToolsetPublish 解析发布 JSON → 工具集。
func ParseToolsetPublish(content string) (*Toolset, error) {
	var pub ToolsetPublish
	if err := json.Unmarshal([]byte(content), &pub); err != nil {
		return nil, fmt.Errorf("导入 JSON 解析失败（应为 toolset_export 输出格式）: %v", err)
	}
	if pub.Toolset.Name == "" || len(pub.Toolset.Plugins) == 0 {
		return nil, fmt.Errorf("导入内容不是有效工具集（缺 name/plugins）")
	}
	return &pub.Toolset, nil
}

// InstallToolsetPublic 装载工具集全部插件。
func InstallToolsetPublic(ph *PluginHost, ts *Toolset) error {
	return installToolset(ph, ts)
}

// RemoveToolsetPublic 删除工具集。
func RemoveToolsetPublic(root, scope, name string) error {
	return removeToolset(root, toolsetScope(scope), name)
}

// countAllToolsetPlugins 统计全部工具集插件数（列表摘要用）。
func countAllToolsetPlugins(projectRoot string) int {
	n := 0
	for _, meta := range listAllToolsets(projectRoot) {
		ts, err := loadToolset(projectRoot, "", meta.Name)
		if err == nil {
			n += len(ts.Plugins)
		}
	}
	return n
}

// installToolset 把一个工具集的全部插件定义并装载到宿主（重名先卸载）。
func installToolset(ph *PluginHost, ts *Toolset) error {
	var errs []string
	for _, p := range ts.Plugins {
		if err := applyToolsetPlugin(ph, &p); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// applyToolsetPlugin 装载单个工具集插件：定义（define 预检）→ 装载（apply 注册工具）
// → 应用 DisabledTools（工具级摘除：Registry.SetToolEnabled(false)，agent 不可见）。
// 重名插件先卸载再重定义（升级/覆盖场景）。toolset_edit 增删单插件时复用。
func applyToolsetPlugin(ph *PluginHost, p *ToolsetPlugin) error {
	if strings.TrimSpace(p.Code) == "" {
		return nil
	}
	// 已存在同名插件：先卸载再重定义（升级/覆盖场景）
	if _, ok := ph.Get(p.Name); ok {
		_ = ph.Unload(p.Name)
		_ = ph.Undefine(p.Name) // 删除 plugins 注册（defs 按 id 存，孤儿条目无碍）
	}
	id, err := ph.DefineJSCodeFull(p.Code, "", p.Purpose, "", p.Client)
	if err != nil {
		return err
	}
	def, _ := ph.GetJSDef(id)
	if err := ph.LoadJSDynamic(def); err != nil {
		return err
	}
	// 应用工具级摘除（插件保留、指定工具禁用 → agent 不可见）
	for _, tn := range p.DisabledTools {
		if tn == "" {
			continue
		}
		if ph.Context() != nil && ph.Context().Tools != nil {
			ph.Context().Tools.SetToolEnabled(tn, false)
		}
	}
	return nil
}
