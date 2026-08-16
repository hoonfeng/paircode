// toolset.go — 工具集（Toolset）：命名插件包 + 动态构建/固化/导出/导入。
//
// 背景：工作区工具能力 = 内置工具 + 插件贡献。工具集把「为某个
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
	Code    string `json:"code,omitempty"`     // host 半：async 函数体（return { name, apply(ctx) }）
	Client  string `json:"client,omitempty"`   // client 半：(ui) => void
	Dir     string `json:"-"`                  // ★ 插件目录（磁盘插件包装载时注入，供 ctx.binary 定位 bin/assets）
	// Scope 插件生效作用域（cordis 动态插件条目）："global"=全局插件（UI 类，
	// 跨工作区生效，存程序目录）/""或"project"=项目插件。★ 存储统一在程序目录
	// <InstallDir>/.pair/plugins/（插件是程序的扩展，不属于工作区）；
	// scope 仅用于记录与前端徽标。
	Scope string `json:"scope,omitempty"`
	// Config 插件配置（package.json "config" 字段，apply(ctx, config) 第二参）。
	// ★ 磁盘插件配置通道（2026-08-16）：agentloop 等插件从 package.json 读装配
	//   参数（模型/迭代上限/追加提示词等），无需重新编译 Go。
	Config map[string]any `json:"config,omitempty"`
	// ★ 内置工具包条目（无 Code）：引用宿主内置 Go 工具组（core/git/codegraph/… 或
	//   system/plugin-mgmt/toolset-mgmt）。装载=对 Tools 清单内已注册工具
	//   SetToolEnabled(true)（工具对 agent 可见）；卸载=恢复默认状态
	//   （ToolDefaultEnabled——harness 保留清单内保持启用，其余禁用）。
	//   这是「被过滤工具进插件面板」的载体：内置组默认不加入工作区，
	//   用户/agent 用 toolset_edit add_builtin 选择加入，add_builtin_all 强制全部。
	Builtin string   `json:"builtin,omitempty"` // 内置分组名（如 "core" / "system"）
	Tools   []string `json:"tools,omitempty"`   // 内置条目：本组宿主内置工具名清单（快照）
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
)

// toolsetDir 返回工具集目录（★ 工具集是工作区级概念——没有「全局工具集」；
// 全局生效的是插件（UI 类），存 <InstallDir>/.pair/plugins/，见 LoadGlobalPlugins）。
func toolsetDir(projectRoot string, scope toolsetScope) string {
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
// ★ 跳过 builtin.json：内置工具集（builtin）是虚拟展示（见 listAllToolsets），
//   固化文件只是「已加入分组」的持久化载体，不作为普通工具集列在列表中。
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
		if strings.TrimSuffix(e.Name(), ".json") == builtinToolsetName {
			continue // 内置工具集虚拟展示
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

// listAllToolsets 列出工作区全部工具集 + 末尾注入虚拟内置工具集 builtin
// （scope=builtin，不落盘；组数=内置分组数）。
func listAllToolsets(projectRoot string) []ToolsetMeta {
	merged := listToolsets(projectRoot, toolsetProject)
	// ★ 虚拟内置工具集（组数 = 内置分组数；ph 未装配时 0）
	groupCount := 0
	if ph := GetGlobalPluginHost(); ph != nil && ph.Context() != nil {
		groupCount = len(BuiltinGroupsOf(ph.Context().Tools, ph))
	}
	merged = append(merged, ToolsetMeta{
		Name: builtinToolsetName, Description: "内置工具包（core/git/codegraph/…；默认不加入，分组开关加入或强制全部）",
		Scope: string(builtinToolsetScope), PluginCount: groupCount,
	})
	return merged
}

// loadToolset 读取指定工具集（scope 为空时先查工作区再查全局）。
func loadToolset(projectRoot string, scope toolsetScope, name string) (*Toolset, error) {
	if name == "" {
		return nil, fmt.Errorf("工具集名不能为空")
	}
	paths := []string{}
	if scope != "" {
		paths = append(paths, toolsetPath(projectRoot, toolsetProject, name))
	} else {
		paths = append(paths, toolsetPath(projectRoot, toolsetProject, name))
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
	return nil, fmt.Errorf("工具集 %q 未找到", name)
}

// saveToolset 固化工具集到指定作用域（原子写：tmp + rename）。
func saveToolset(projectRoot string, scope toolsetScope, ts *Toolset) error {
	if ts == nil || ts.Name == "" {
		return fmt.Errorf("工具集名不能为空")
	}
	// ★ 允许空插件列表（dynamic 等容器工具集移除最后一个插件后需能保存空状态；
	//   build/import 路径已有各自的非空校验，不会走到这里保存空工具集）
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

// removeToolset 删除工具集文件（工具集仅工作区级）。
func removeToolset(projectRoot string, scope toolsetScope, name string) error {
	if name == "" {
		return fmt.Errorf("工具集名不能为空")
	}
	return os.Remove(toolsetPath(projectRoot, toolsetProject, name))
}

// ─── 装载（启动时自动装配 + 构建后装载）──────────────────

// LoadAllToolsets 装载工作区全部工具集（启动时调用；失败不致命）。
// ★ 内置工具集 builtin.json（listToolsets 虚拟展示跳过，需单独装载——
//   否则用户加入的内置分组重启后不生效）。
// ★ 全局插件（UI 类跨工作区）独立于工具集：见 LoadGlobalPlugins（不进工具集列表）。
func LoadAllToolsets(ph *PluginHost, projectRoot string) {
	if ph == nil {
		return
	}
	loaded := 0
	// ★ 项目工具集（工作区 .pair/toolsets/）——依赖工作区，未打开时跳过
	if projectRoot != "" {
		for _, meta := range listToolsets(projectRoot, toolsetProject) {
			ts, err := loadToolset(projectRoot, toolsetProject, meta.Name)
			if err != nil {
				continue
			}
			if err := installToolset(ph, ts); err != nil {
				log.Printf("[toolset] %s 装载失败: %v", meta.Name, err)
				continue
			}
			loaded++
		}
		// 内置工具包（用户/agent 选择加入的分组）
		if ts, err := loadToolset(projectRoot, toolsetProject, builtinToolsetName); err == nil {
			if err := installToolset(ph, ts); err != nil {
				log.Printf("[toolset] builtin 内置工具包装载失败: %v", err)
			} else {
				loaded++
			}
		}
	}
	// ★ 全局插件（UI 类跨工作区生效；不属于任何工具集）——不依赖工作区：
	//   存 <InstallDir>/.pair/plugins/，未打开工作区也必须装载（发布版启动即生效）
	if n := LoadGlobalPlugins(ph); n > 0 {
		loaded += n
	}
	if loaded > 0 {
		log.Printf("[toolset] 已装载 %d 个工具集（%d 个插件）", loaded, countAllToolsetPlugins(projectRoot))
	}
}

// ─── 全局插件（独立于工具集）────────────────────────────

// ★ 设计：没有「全局工具集」——工具集是工作区级概念。全局生效的是插件
//   （UI 类插件，含 client 半，跨工作区装载），存 <InstallDir>/.pair/plugins/，
//   每个插件一个「插件包」目录（package.json + 源码），启动时单独装配，
//   不属于任何工具集（工具集列表/管理不显示）。
func globalPluginsDir() string {
	return filepath.Join(core.InstallDir(), ".pair", "plugins")
}

// GlobalPluginsPath 全局插件目录路径。
func GlobalPluginsPath() string {
	return globalPluginsDir()
}

// GlobalPluginPackage 全局插件包描述（<name>/package.json）。
type GlobalPluginPackage struct {
	Name    string `json:"name"`              // 插件名（包目录名）
	Purpose string `json:"purpose,omitempty"` // 用途说明
	Version string `json:"version"`           // 版本
	Scope   string `json:"scope,omitempty"`   // "global"（UI 类跨工作区）/ "project"
	Type    string `json:"type"`              // "plugin"
	Main    string `json:"main"`              // host 半源码文件（index.js）
	Client  string `json:"client,omitempty"`  // client 半源码文件（client.js，可选）
	Config  map[string]any `json:"config,omitempty"` // 插件配置（透传 apply(ctx, config)）
}

// LoadGlobalPlugins 装配全部全局插件包（启动时调用；失败不致命）。返回成功装载数。
// ★ 插件包形态：<InstallDir>/.pair/plugins/<name>/package.json + 源码文件。
func LoadGlobalPlugins(ph *PluginHost) int {
	if ph == nil {
		return 0
	}
	entries, err := os.ReadDir(globalPluginsDir())
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules" {
			continue
		}
		// ★ 源码包目录约定：<name>-src 是插件源码（如 ui-app-src 前端 Vite 工程），
		//   不是可装载插件包——跳过（用户可改源码后重新构建进插件包/assets）。
		if strings.HasSuffix(e.Name(), "-src") {
			continue
		}
		// ★ 非插件目录（无 package.json，如 config/ 模型模板、README 等）静默跳过
		if _, err := os.Stat(filepath.Join(globalPluginsDir(), e.Name(), "package.json")); err != nil {
			continue
		}
		if err := applyGlobalPluginDir(ph, filepath.Join(globalPluginsDir(), e.Name())); err != nil {
			log.Printf("[global-plugin] %s 装载失败: %v", e.Name(), err)
			continue
		}
		n++
	}
	return n
}

// applyGlobalPluginDir 装载一个全局插件包目录（读 package.json + 源码 → define+load）。
func applyGlobalPluginDir(ph *PluginHost, pkgDir string) error {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return fmt.Errorf("缺 package.json: %w", err)
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Name == "" || pkg.Main == "" {
		return fmt.Errorf("package.json 无效（缺 name/main）")
	}
	hostCode, err := os.ReadFile(filepath.Join(pkgDir, pkg.Main))
	if err != nil {
		return fmt.Errorf("host 源码读取失败: %w", err)
	}
	clientCode := ""
	if pkg.Client != "" {
		if cb, err := os.ReadFile(filepath.Join(pkgDir, pkg.Client)); err == nil {
			clientCode = string(cb)
		}
	}
	return applyGlobalPlugin(ph, &ToolsetPlugin{
		Name: pkg.Name, Purpose: pkg.Purpose,
		Code: string(hostCode), Client: clientCode, Scope: pkg.Scope,
		Dir: pkgDir, Config: pkg.Config,
	})
}

// applyGlobalPlugin 装载单个全局插件（定义 + 装载；scope 从 package.json 恢复）。
func applyGlobalPlugin(ph *PluginHost, p *ToolsetPlugin) error {
	if p == nil || strings.TrimSpace(p.Code) == "" {
		return nil
	}
	// 已存在同名插件：先卸载再重定义（升级/覆盖场景）
	if _, ok := ph.Get(p.Name); ok {
		_ = ph.Unload(p.Name)
		_ = ph.Undefine(p.Name)
	}
	id, err := ph.DefineJSCodeFull(p.Code, "", p.Purpose, "", p.Client)
	if err != nil {
		return err
	}
	def, _ := ph.GetJSDef(id)
	if def != nil {
		def.scope = p.Scope
		if def.scope == "" {
			def.scope = "project"
		}
		def.dir = p.Dir // ★ 插件目录（ctx.binary 据此定位 bin/<name>.exe 与 assets/）
		def.config = p.Config // ★ 插件配置（package.json "config"，apply(ctx, config) 第二参）
	}
	if err := ph.LoadJSDynamic(def); err != nil {
		return err
	}
	return nil
}

// syncGlobalPlugin 把插件固化为插件包目录（同名更新/追加；★ 插件=包，不是 json）。
// 结构：<InstallDir>/.pair/plugins/<name>/package.json + index.js（host 半）
// + client.js（有 client 半时）。
func syncGlobalPlugin(entry ToolsetPlugin) error {
	if entry.Name == "" || strings.TrimSpace(entry.Code) == "" {
		return fmt.Errorf("全局插件缺 name/code")
	}
	dir := filepath.Join(globalPluginsDir(), entry.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	pkg := GlobalPluginPackage{
		Name: entry.Name, Purpose: entry.Purpose, Version: "1.0.0",
		Scope: entry.Scope, Type: "plugin", Main: "index.js",
	}
	if pkg.Scope == "" {
		pkg.Scope = "project"
	}
	// host 半源码
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte(entry.Code), 0644); err != nil {
		return err
	}
	// client 半源码（有 client 半时写 client.js 并在 package.json 声明）
	if strings.TrimSpace(entry.Client) != "" {
		pkg.Client = "client.js"
		if err := os.WriteFile(filepath.Join(dir, "client.js"), []byte(entry.Client), 0644); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "package.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// SaveToolsetPublic 固化工具集（★ 仅工作区级：scope 非空且≠project 时拒绝——
// 没有「全局工具集」，全局生效的是插件，见 GlobalPluginsPath）。
func SaveToolsetPublic(root, scope string, ts *Toolset) error {
	if scope != "" && scope != "project" {
		return fmt.Errorf("工具集仅工作区级（没有全局工具集）；scope 只支持 project")
	}
	return saveToolset(root, toolsetProject, ts)
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

// UnloadToolsetPublic 卸载工具集全部条目（内置条目恢复默认，JS 插件 Unload+Undefine）。
func UnloadToolsetPublic(ph *PluginHost, ts *Toolset) {
	UnloadToolsetPlugins(ph, ts)
}

// RemoveToolsetPublic 删除工具集（★ 仅工作区级）。
func RemoveToolsetPublic(root, scope, name string) error {
	if scope != "" && scope != "project" {
		return fmt.Errorf("工具集仅工作区级（没有全局工具集）；scope 只支持 project")
	}
	return removeToolset(root, toolsetProject, name)
}

// ─── 内置工具集公共封装（handler REST / 前端面板直调）──────

// BuiltinToolsetNamePublic 内置工具集名（虚拟，scope=builtin）。
func BuiltinToolsetNamePublic() string { return builtinToolsetName }

// BuiltinToolsetInfoPublic 内置工具包完整信息（分组+工具+启用状态+已加入分组）。
func BuiltinToolsetInfoPublic(reg *Registry, ph *PluginHost, root string) *BuiltinToolsetInfo {
	return BuiltinToolsetInfoOf(reg, ph, root)
}

// SetBuiltinGroupEnabledPublic 内置分组开关（enabled=true 加入工作区并启用组内工具；
// false 移出恢复默认过滤）。返回操作结果文本。
func SetBuiltinGroupEnabledPublic(ph *PluginHost, root, groupName string, enabled bool) (string, error) {
	return SetBuiltinGroupEnabled(ph, root, groupName, enabled)
}

// EnableAllBuiltinPublic 强制全部内置工具组加入工作区。
func EnableAllBuiltinPublic(ph *PluginHost, root string) (string, error) {
	return EnableAllBuiltin(ph, root)
}

// SetBuiltinToolEnabledPublic 内置工具级开关（工具列表/手动添加指定工具）。
func SetBuiltinToolEnabledPublic(ph *PluginHost, root, tool string, enabled bool) (string, error) {
	return SetBuiltinToolEnabled(ph, root, tool, enabled)
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

// UnloadToolsetPlugins 卸载工具集全部条目（rm_plugin/remove/覆盖重建前调用）：
// 内置条目恢复默认过滤状态，JS 插件 Unload+Undefine。公共导出（handler 层复用）。
func UnloadToolsetPlugins(ph *PluginHost, ts *Toolset) {
	if ph == nil || ts == nil {
		return
	}
	for i := range ts.Plugins {
		unloadToolsetPlugin(ph, &ts.Plugins[i])
	}
}

// applyToolsetPlugin 装载单个工具集条目：
//   - JS 插件条目（Code 非空）：定义（define 预检）→ 装载（apply 注册工具）
//     → 应用 DisabledTools（工具级摘除：Registry.SetToolEnabled(false)，agent 不可见）。
//     重名插件先卸载再重定义（升级/覆盖场景）。toolset_edit 增删单插件时复用。
//   - 内置工具包条目（Builtin 非空，无 Code）：对 Tools 清单内已注册工具
//     SetToolEnabled(true)（启用——工具对 agent 可见）→ 应用 DisabledTools。
func applyToolsetPlugin(ph *PluginHost, p *ToolsetPlugin) error {
	// ── 内置工具包条目：无 JS 代码，装载=启用组内工具 ──
	if p.Builtin != "" && strings.TrimSpace(p.Code) == "" {
		if ph.Context() != nil && ph.Context().Tools != nil {
			for _, tn := range p.Tools {
				if tn == "" {
					continue
				}
				if _, ok := ph.Context().Tools.Get(tn); ok {
					ph.Context().Tools.SetToolEnabled(tn, true)
				}
			}
			for _, tn := range p.DisabledTools {
				if tn == "" {
					continue
				}
				if _, ok := ph.Context().Tools.Get(tn); ok {
					ph.Context().Tools.SetToolEnabled(tn, false)
				}
			}
		}
		return nil
	}
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

// unloadToolsetPlugin 卸载单个工具集条目（rm_plugin / remove / 覆盖重装时调用）：
//   - 内置工具包条目（Builtin 非空）：不卸载插件（无 JS 插件），而是把组内工具
//     恢复默认状态（ToolDefaultEnabled：harness 保留清单内保持启用，其余禁用）——
//     工具从「已加入」回到「被过滤」。
//   - JS 插件条目：正常 Unload（回收工具/事件/系统提示）+ Undefine。
// 幂等：插件未运行 / 工具未注册时无操作。
func unloadToolsetPlugin(ph *PluginHost, p *ToolsetPlugin) {
	if ph == nil || p == nil {
		return
	}
	if p.Builtin != "" && strings.TrimSpace(p.Code) == "" {
		if ph.Context() != nil && ph.Context().Tools != nil {
			for _, tn := range p.Tools {
				if tn == "" {
					continue
				}
				if _, ok := ph.Context().Tools.Get(tn); ok {
					ph.Context().Tools.SetToolEnabled(tn, ToolDefaultEnabled(tn))
				}
			}
		}
		return
	}
	if p.Name == "" {
		return
	}
	if _, ok := ph.Get(p.Name); ok {
		_ = ph.Unload(p.Name)
		_ = ph.Undefine(p.Name)
	}
}
