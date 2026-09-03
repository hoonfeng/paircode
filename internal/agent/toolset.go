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
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// ─── 数据模型 ─────────────────────────────────────────────

// Toolset 工具集 = 命名插件包。
type Toolset struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Project     string `json:"project,omitempty"` // 适用项目（basename，多项目区分）
	Version     string `json:"version,omitempty"` // 语义版本
	CreatedAt   string `json:"createdAt,omitempty"`
	// BuiltinsInited 内置工具组是否已初始化进本工具集（★ 2026-08-20：
	// 内置工具默认放入工作区工具集；用户后续可整组/单工具移出。
	// 标记只补一次——防止「移出后重启又补回」；旧工具集无此字段 → 视为未初始化补齐一次）。
	BuiltinsInited bool            `json:"builtinsInited,omitempty"`
	Plugins        []ToolsetPlugin `json:"plugins"`
}

// ToolsetPlugin 工具集内一个插件定义（host/client 双半）。
type ToolsetPlugin struct {
	Name    string `json:"name"`
	Purpose string `json:"purpose"`
	Code    string `json:"code,omitempty"`   // host 半：async 函数体（return { name, apply(ctx) }）
	Client  string `json:"client,omitempty"` // client 半：(ui) => void
	Dir     string `json:"-"`                // ★ 插件目录（磁盘插件包装载时注入，供 ctx.binary 定位 bin/assets）
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
	// HasDshUI ★ 磁盘插件包含 外部兼容 dsh.ui 段（UI 区域/功能包）。装载到 def 后，
	// /api/plugins 列表据此标记 hasClient=true（即使无 client.js），见 Inspect/InspectDetail。
	HasDshUI bool `json:"-"`
}

// ─── 目录解析 ─────────────────────────────────────────────

// toolsetScope 工具集作用域。
type toolsetScope string

const (
	toolsetProject toolsetScope = "project" // ★ 兼容保留：工具集已全局化，任何 scope 都落全局目录
)

// globalToolsetDir 全局工具集（通用集合）目录：<InstallDir>/.pair/toolsets/。
// ★ 2026-09-04 工具集模式改造：工具集从「工作区级」（每工作区 .pair/toolsets/）
//
//	改为「全局通用集合」——跨工作区共享，预置多模式（default/full/dev/debug/
//	test/docs），对话面板按会话选择当前集合。全局插件目录
//	（<InstallDir>/.pair/plugins/）同理，见 globalPluginsDir。
func globalToolsetDir() string {
	if globalToolsetDirOverride != "" {
		return globalToolsetDirOverride
	}
	return filepath.Join(core.InstallDir(), ".pair", "toolsets")
}

// globalToolsetDirOverride 测试隔离/嵌入环境覆盖点（空=默认安装目录）。
var globalToolsetDirOverride string

// SetGlobalToolsetDirForTest 设置全局工具集目录（测试隔离用；传入空恢复默认）。
func SetGlobalToolsetDirForTest(dir string) { globalToolsetDirOverride = dir }

// toolsetDir 返回工具集目录（★ 已全局化：projectRoot/scope 仅作兼容参数保留，
// 一律返回全局通用集合目录 <InstallDir>/.pair/toolsets/）。
func toolsetDir(projectRoot string, scope toolsetScope) string {
	return globalToolsetDir()
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
	Scope       string `json:"scope"` // project | global
	PluginCount int    `json:"pluginCount"`
}

// listToolsets 列出指定作用域全部工具集元信息（按名排序）。
// ★ 2026-08-17：builtin.json 独立机制已废除（MigrateLegacyBuiltinJSON 迁移合并），
//
//	工具集文件全部为普通工作区工具集，无需跳过。
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
			continue // 防御：旧版 builtin.json 未迁移前不列入普通工具集 // 内置工具集虚拟展示
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
	sort.Slice(out, func(i, j int) bool {
		// 预设模式按 presetModes 声明顺序（计划讨论→全栈→办公→调试→基础→全功能），
		// 用户自定义工具集排预设之后按名排序。
		oi, oj := presetModeOrder(out[i].Name), presetModeOrder(out[j].Name)
		if oi != oj {
			return oi < oj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// presetModeOrder 预设模式在 presetModes 中的顺序索引；非预设返回大数（排预设之后）。
func presetModeOrder(name string) int {
	for i, m := range presetModes {
		if m.Name == name {
			return i
		}
	}
	return len(presetModes)
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
	// ★ v3 兼容：预设旧英文名（planning/default/…）→ 中文名（会话里存的旧名继续可用）
	name = resolvePresetName(name)
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

// defaultProjectToolset 基础工具集（无工具集时自动生成）：
//  1. 基础工具：极简核心（harness 命名 read/write/edit/glob/grep/bash/
//     run_code）；
//  2. 框架本身提供给 agent 的工具（默认可用）：
//     - system：核心 + SystemTool（任务追踪/提问/提交标记/历史/技能/MCP/
//     市场等宿主自举工具）+ tool-system 插件承载的框架工具
//     - plugin-mgmt：cordis_* 插件管理工具
//     - toolset-mgmt：toolset_* 工具集管理工具
//
// 业务插件工具（tool-git/tool-codegraph 等磁盘插件）不默认加入——用户用
// toolset_edit add_plugin 按需加入。
// ★ 2026-08-17：装载≠可用兜底——新工作区无任何工具集时，agent 默认只有
//
//	基础工具 + 框架本身提供的工具；其余工具对 agent 隐藏，按需加入。
//
// ★ 2026-08-20：内置工具组（system/plugin-mgmt/toolset-mgmt）默认写入
//
//	default.json（ensureBuiltinGroupsInWorkspace 对已有工作区幂等补齐）——
//	内置工具在工作区工具集中可见可管理（工具集面板/插件面板控制启用）。
func defaultProjectToolset(reg *Registry, ph *PluginHost, project string) *Toolset {
	return &Toolset{
		Name:           presetNameDefault,
		Description:    "基础工具集（自动生成）——极简核心 + 框架本身提供的工具；插件工具用 toolset_edit add_plugin 按需加入",
		Project:        project,
		Version:        "1.0.0",
		CreatedAt:      time.Now().Format(time.RFC3339),
		BuiltinsInited: true, // 内置工具组已含在 Plugins（默认放入工作区工具集）
		Plugins:        builtinGroupEntries(reg, ph),
	}
}

// builtinGroupEntries 框架内置工具组条目（system/plugin-mgmt/toolset-mgmt）——
// defaultProjectToolset 与 ensureBuiltinGroupsInWorkspace 共用同一组装逻辑。
func builtinGroupEntries(reg *Registry, ph *PluginHost) []ToolsetPlugin {
	base := []string{"read", "write", "edit", "glob", "grep", "bash", "run_code"}
	sysSet := map[string]bool{}
	for _, t := range base {
		sysSet[t] = true
	}
	// tool-system 插件承载的框架工具（SystemTool 承揽 + Skills/MCP/市场）
	if ph != nil {
		for _, tn := range ph.PluginToolsByPlugin()["tool-system"] {
			if tn != "" {
				sysSet[tn] = true
			}
		}
	}
	var mgmtTools []string
	var tsTools []string
	if reg != nil {
		owners := map[string]bool{}
		if ph != nil {
			for k := range ph.PluginToolOwners() {
				owners[k] = true
			}
		}
		for _, meta := range reg.AllToolMeta() {
			if sysSet[meta.Name] {
				continue
			}
			if meta.SystemTool {
				sysSet[meta.Name] = true // 宿主框架自举工具（恒可用）
				continue
			}
			if owners[meta.Name] {
				continue // 业务插件工具（不默认加入）
			}
			switch {
			case isCordisMgmtTool(meta.Name):
				mgmtTools = append(mgmtTools, meta.Name)
			case isToolsetMgmtTool(meta.Name):
				tsTools = append(tsTools, meta.Name)
			}
			// 其余无 owner 非 SystemTool 工具：内置包残留，默认禁用
		}
	}
	var sysTools []string
	for t := range sysSet {
		sysTools = append(sysTools, t)
	}
	sort.Strings(sysTools)
	sort.Strings(mgmtTools)
	sort.Strings(tsTools)

	plugins := []ToolsetPlugin{
		{
			Name:    "builtin:system",
			Purpose: "system：极简核心 + 框架宿主工具（harness 别名/任务追踪/提问/提交标记等）",
			Builtin: "system",
			Tools:   sysTools,
		},
	}
	if len(mgmtTools) > 0 {
		plugins = append(plugins, ToolsetPlugin{
			Name:    "builtin:plugin-mgmt",
			Purpose: "plugin-mgmt：cordis_* 插件管理工具（登记/装载/停止/回收/查看）",
			Builtin: "plugin-mgmt",
			Tools:   mgmtTools,
		})
	}
	if len(tsTools) > 0 {
		plugins = append(plugins, ToolsetPlugin{
			Name:    "builtin:toolset-mgmt",
			Purpose: "toolset-mgmt：toolset_* 工具集管理工具（构建/列表/编辑/导出/导入）",
			Builtin: "toolset-mgmt",
			Tools:   tsTools,
		})
	}
	return plugins
}

// ensureDefaultWorkspaceToolset 确保全局「default」通用集合存在（无 default 时
// 播种预置模式——default 是首个预置模式）。★ 2026-09-04 工具集全局化：
// 不再生成「工作区」基础工具集——工具集只有一套全局通用集合，default 缺失时
// 由 seedPresetToolsets 补齐（幂等；用户删除后不复活——见 .preset-seeded 标记）。
func ensureDefaultWorkspaceToolset(ph *PluginHost, root string) error {
	_ = root
	seedPresetToolsets(ph)
	return nil
}

// ensureBuiltinGroupsInWorkspace 确保全局 default 集合含全部内置工具组条目
// （system/plugin-mgmt/toolset-mgmt）。★ 2026-08-20 语义保留：内置工具默认放入
// 工具集（agent 默认可用、面板可见可管理）。只补一次：default.json 的
// BuiltinsInited 标记置位后不再自动补（用户移出/增删后保持自己的管理结果）。
// 返回是否补了条目（调用方决定是否重装载）。
func ensureBuiltinGroupsInWorkspace(ph *PluginHost, root string) (bool, error) {
	_ = root
	var reg *Registry
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	ts, err := loadToolset("", toolsetProject, presetNameDefault)
	if err != nil || ts == nil {
		// 无 default：由 seedPresetToolsets 生成（defaultProjectToolset 已含内置组）
		return false, nil
	}
	if ts.BuiltinsInited {
		return false, nil // 已初始化过：保持用户管理结果（含移出后的状态）
	}
	entries := builtinGroupEntries(reg, ph)
	have := map[string]bool{}
	for _, p := range ts.Plugins {
		if p.Builtin != "" {
			have[p.Builtin] = true
		}
	}
	changed := false
	for _, e := range entries {
		if have[e.Builtin] {
			continue
		}
		ts.Plugins = append(ts.Plugins, e)
		have[e.Builtin] = true
		changed = true
	}
	ts.BuiltinsInited = true
	if !changed {
		// 无新增条目也置标记（避免反复扫描）
		if err := saveToolset("", toolsetProject, ts); err != nil {
			return false, err
		}
		return false, nil
	}
	if err := saveToolset("", toolsetProject, ts); err != nil {
		return false, err
	}
	log.Printf("[toolset] 已补齐内置工具组条目到全局工具集 %s（%d 个）", ts.Name, len(entries))
	return true, nil
}

// EnsureWorkspaceToolsetPublic 确保全局 default 通用集合存在（供 handler/前端调用）。
func EnsureWorkspaceToolsetPublic(ph *PluginHost, root string) error {
	return ensureDefaultWorkspaceToolset(ph, root)
}

// MigrateLegacyWorkspaceToolsets 迁移旧工作区工具集（<root>/.pair/toolsets/*.json）
// 到全局通用集合目录（<InstallDir>/.pair/toolsets/）。
// ★ 2026-09-04 工具集模式改造：工作区级工具集被全局通用集合取代——
//
//	启动时把每个工作区的旧工具集一次性搬入全局（同名不覆盖：全局已有同名
//	文件视为更新版本/用户管理结果，工作区旧文件保留在原位但不再读取）。
//	迁移完成（无同名冲突）后工作区 .pair/toolsets/ 目录清空，代码路径不再
//	读工作区目录——「工作区工具集」模式彻底移除。
//
// 返回迁移（移动成功）的文件数。同名冲突的计数含在第二个返回值（跳过数）。
func MigrateLegacyWorkspaceToolsets(ph *PluginHost, root string) int {
	if root == "" {
		return 0
	}
	srcDir := filepath.Join(root, ".pair", "toolsets")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0
	}
	moved := 0
	skipped := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if e.Name() == presetSeedMarker {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(globalToolsetDir(), e.Name())
		if _, err := os.Stat(dst); err == nil {
			skipped++ // 全局已有同名：保留全局（用户管理结果优先）
			continue
		}
		if err := os.MkdirAll(globalToolsetDir(), 0o755); err != nil {
			log.Printf("[toolset] 迁移失败（目录创建）: %v", err)
			break
		}
		if err := os.Rename(src, dst); err != nil {
			log.Printf("[toolset] 迁移 %s 失败: %v", src, err)
			continue
		}
		moved++
		log.Printf("[toolset] 旧工作区工具集已迁移到全局: %s（来自 %s）", e.Name(), root)
	}
	_ = ph
	if moved > 0 || skipped > 0 {
		if _, err := workspaceMainToolset(ph, root); err != nil {
			log.Printf("[toolset] 迁移后确保 default 存在失败: %v", err)
		}
	}
	if moved+skipped > 0 {
		log.Printf("[toolset] 工作区工具集迁移完成：移动 %d，同名跳过 %d（全局保留）", moved, skipped)
	}
	return moved
}

// LoadAllToolsets 装载全局全部工具集（启动时调用；失败不致命）。
// ★ 2026-09-04 工具集模式改造：工具集已全局化（<InstallDir>/.pair/toolsets/，
//
//	通用集合、跨工作区共享）——不再按工作区读取、不再自动生成工作区 default。
//	启动流程：① 迁移旧工作区 .pair/toolsets/（一次性，同名跳过）→ ② 播种预置
//	模式（首次补全 default/full/dev/debug/test/docs）→ ③ 装载全局全部工具集
//	→ ④ 全局插件（UI/工具包）→ ⑤ 清理未暴露条目 → ⑥ 可见性收敛。
//
// ★ 全局插件（UI 类跨工作区）独立于工具集：见 LoadGlobalPlugins（不进工具集列表）。
func LoadAllToolsets(ph *PluginHost, projectRoot string) {
	if ph == nil {
		return
	}
	loaded := 0
	// ★ 2026-09-04：旧工作区工具集一次性迁移到全局（同名跳过；迁移后工作区
	//   目录不再读取——「工作区工具集」模式已被通用集合取代）
	if projectRoot != "" {
		if n := MigrateLegacyWorkspaceToolsets(ph, projectRoot); n > 0 {
			log.Printf("[toolset] 已迁移 %d 个旧工作区工具集到全局通用集合目录", n)
		}
	}
	// ★ 全局插件（UI 类跨工作区生效；不属于任何工具集）——不依赖工作区：
	//   存 <InstallDir>/.pair/plugins/，未打开工作区也必须装载（发布版启动即生效）。
	//   ★ 必须先于 seedPresetToolsets：磁盘插件装载后 toolOwner 认领各自工具，
	//     预置模式的 builtin:system 组快照才不会把业务工具误收进「系统工具」
	//     （否则 default 变全功能——codegraph/git 等工具被 system 组声明）。
	if n := LoadGlobalPlugins(ph); n > 0 {
		loaded += n
	}
	// ★ 2026-09-04：预置模式播种（首次补全 default/full/dev/debug/test/docs；
	//   已被用户删除的集合不复活——.preset-seeded 标记）
	seedPresetToolsets(ph)
	// ★ 全局工具集（通用集合）——不再依赖工作区
	for _, meta := range listToolsets("", toolsetProject) {
		ts, err := loadToolset("", toolsetProject, meta.Name)
		if err != nil {
			continue
		}
		if err := installToolset(ph, ts); err != nil {
			log.Printf("[toolset] %s 装载失败: %v", meta.Name, err)
			continue
		}
		loaded++
	}
	// ★ 2026-08-2x：实装后二次过滤——工具集中「未暴露」的条目/工具清理落盘
	//   （未装载插件条目 / 未注册工具 / 整组未启用的内置组）。已装载插件的工具
	//   保留；清理结果保存到 .pair/toolsets/*.json，下次启动不再出现。
	if n := pruneUnavailableFromToolsets(ph, projectRoot); n > 0 {
		log.Printf("[toolset] 实装后清理：%d 个工具集已移除未暴露条目并保存", n)
	}
	// ★ 2026-08-17：装载 ≠ agent 可用——全部插件照常装载（cordis/前端可见可管理），
	//   但收敛 agent 可见工具 = 全局工具集声明 + 自举管理工具（SystemTool +
	//   cordis_*/toolset_*）。未加入工具集的插件工具对 agent 隐藏（Enabled=false），
	//   恢复 = toolset_edit add_plugin 加入工具集。
	if ph.Context() != nil && ph.Context().Tools != nil {
		if n := ApplyToolsetVisibilityFilter(ph.Context().Tools, ph, projectRoot); n > 0 {
			log.Printf("[toolset] 可见性收敛：%d 个未加入工具集的工具对 agent 隐藏（cordis 仍可见，toolset_edit 可加入）", n)
		}
	}
	if loaded > 0 {
		log.Printf("[toolset] 已装载 %d 个工具集（%d 个插件）", loaded, countAllToolsetPlugins(""))
	}
}

// ─── 全局插件（独立于工具集）────────────────────────────

// ★ 设计：没有「全局工具集」——工具集是工作区级概念。全局生效的是插件
//
//	（UI 类插件，含 client 半，跨工作区装载），存 <InstallDir>/.pair/plugins/，
//	每个插件一个「插件包」目录（package.json + 源码），启动时单独装配，
//	不属于任何工具集（工具集列表/管理不显示）。
func globalPluginsDir() string {
	return filepath.Join(core.InstallDir(), ".pair", "plugins")
}

// GlobalPluginsPath 全局插件目录路径。
func GlobalPluginsPath() string {
	return globalPluginsDir()
}

// GlobalPluginPackage 全局插件包描述（<name>/package.json）。
type GlobalPluginPackage struct {
	Name    string           `json:"name"`              // 插件名（包目录名）
	Purpose string           `json:"purpose,omitempty"` // 用途说明
	Version string           `json:"version"`           // 版本
	Scope   string           `json:"scope,omitempty"`   // "global"（UI 类跨工作区）/ "project"
	Type    string           `json:"type"`              // "plugin"
	Main    string           `json:"main"`              // host 半源码文件（index.js）
	Client  string           `json:"client,omitempty"`  // client 半源码文件（client.js，可选）
	Config  map[string]any   `json:"config,omitempty"`  // 插件配置（透传 apply(ctx, config)）
	Dsh     *GlobalPluginDsh `json:"dsh,omitempty"`     // ★ 外部兼容二段式 manifest 的 dsh.ui 段（UI 区域/功能包声明；新增，旧包无此段仍按 client.js 直载）
}

// diskPluginCodeAvailable 磁盘插件包是否存在且 main 源码非空（R2-8 去重用）。
// 与 LoadGlobalPlugins 的装载判定同源：package.json 有效 + main 文件可读非空。
func diskPluginCodeAvailable(name string) bool {
	if name == "" {
		return false
	}
	dir := filepath.Join(globalPluginsDir(), name)
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Name == "" || pkg.Main == "" {
		return false
	}
	code, err := os.ReadFile(filepath.Join(dir, pkg.Main))
	if err != nil || strings.TrimSpace(string(code)) == "" {
		return false
	}
	return true
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
		// ★ 源码包目录约定：<name>-src 是插件源码（UI 源码工程在项目根 ui-app/，不进 .pair），
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
	// ★ 提示词插件化-插件内置：扫描插件包 prompts/ 目录注册提示词资产
	//   （<name>.md → 资产名 <name>；任何插件包放 prompts/ 即可贡献提示词）。
	if n := ScanPluginPromptAssets(pkgDir, pkg.Name); n > 0 {
		log.Printf("[global-plugin] %s 注册提示词资产 %d 个（prompts/ 目录）", pkg.Name, n)
	}
	return applyGlobalPlugin(ph, &ToolsetPlugin{
		Name: pkg.Name, Purpose: pkg.Purpose,
		Code: string(hostCode), Client: clientCode, Scope: pkg.Scope,
		Dir: pkgDir, Config: pkg.Config, HasDshUI: pkg.Dsh != nil && pkg.Dsh.UI != nil,
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
	id, err := ph.DefineJSCodeFull(p.Code, "", p.Purpose, p.Dir, p.Client)
	if err != nil {
		return err
	}
	def, _ := ph.GetJSDef(id)
	if def != nil {
		def.scope = p.Scope
		if def.scope == "" {
			def.scope = "project"
		}
		def.dir = p.Dir           // ★ 插件目录（ctx.binary 据此定位 bin/<name>.exe 与 assets/）
		def.config = p.Config     // ★ 插件配置（package.json "config"，apply(ctx, config) 第二参）
		def.hasDshUI = p.HasDshUI // ★ 外部兼容 dsh.ui 段：/api/plugins 据此标记 hasClient（见 Inspect）
	}
	// ★ 提示词插件化-插件+插件配置：package.json config.prompts（name → text 映射）
	//   注册为提示词资产（优先级高于插件包 prompts/ 磁盘资产与 config/roles）。
	if n := registerConfigPrompts(p.Config, p.Name); n > 0 {
		log.Printf("[global-plugin] %s 注册提示词资产 %d 个（config.prompts）", p.Name, n)
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
		Config: entry.Config,
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
		name = presetNameDefault
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
	// ★ LLM 现场生成的项目专属插件并入（模板覆盖不到的能力缺口；对齐 
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
		Readme: fmt.Sprintf("# 工具集：%s\n\n%s\n\n## 包含插件\n%s",
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

// SaveToolsetPublic 固化工具集（★ 2026-09-04 全局化：工具集为通用集合，
// 一律存 <InstallDir>/.pair/toolsets/；scope 参数保留兼容，任何值都落全局）。
func SaveToolsetPublic(root, scope string, ts *Toolset) error {
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

// RemoveToolsetPublic 删除工具集（★ 2026-09-04 全局化：通用集合，一律从全局目录删除）。
func RemoveToolsetPublic(root, scope, name string) error {
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

// RemovePluginFromToolsetsPublic 从工作区全部工具集移除「含 code 的同名插件条目」并保存。
// 用途：删除插件定义时联动——工具集条目内嵌 code，重启 installToolset 会重新
// define+load（插件「复活」）；移除条目后删除才彻底。仅移除含 code 的条目
// （纯摘除记录 DisabledTools / 内置组条目不涉及插件定义，保留）。
// 返回移除了条目的工具集数量（0 = 工具集中无该插件条目）。
func RemovePluginFromToolsetsPublic(root, name string) int {
	if root == "" || name == "" {
		return 0
	}
	removed := 0
	for _, meta := range listToolsets(root, toolsetProject) {
		ts, err := loadToolset(root, toolsetProject, meta.Name)
		if err != nil || ts == nil {
			continue
		}
		changed := false
		out := ts.Plugins[:0]
		for _, p := range ts.Plugins {
			if p.Name == name && strings.TrimSpace(p.Code) != "" {
				changed = true
				continue
			}
			out = append(out, p)
		}
		if changed {
			ts.Plugins = out
			if err := saveToolset(root, toolsetProject, ts); err == nil {
				removed++
			}
		}
	}
	return removed
}

// RestorePluginToToolsetsPublic 把磁盘插件包加回工作区 default 工具集（start 联动，
// 与 stop 的 RemovePluginFromToolsetsPublic 对称）：读 <InstallDir>/.pair/plugins/
// <name>/package.json + main 源码 → 作为 JS 条目加入（已有同名条目跳过）。
// ★ 2026-08-2x：stop 移除条目后若不加回，重启时磁盘插件虽被 LoadGlobalPlugins
//
//	装载（running），但工具不在工具集白名单 → agent 不可见（start 后也恢复不了）。
//	调用方需确保该插件有工具（无工具插件不加，避免 UI 类插件占位工具集）。
//
// 返回加入的工具集数（0 = 无包目录/已在工具集/失败）。
func RestorePluginToToolsetsPublic(root, name string) int {
	if root == "" || name == "" {
		return 0
	}
	dir := filepath.Join(globalPluginsDir(), name)
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return 0
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Name == "" || pkg.Main == "" {
		return 0
	}
	code, err := os.ReadFile(filepath.Join(dir, pkg.Main))
	if err != nil {
		return 0
	}
	ts, err := loadToolset(root, toolsetProject, presetNameDefault)
	if err != nil || ts == nil {
		return 0
	}
	for _, p := range ts.Plugins {
		if p.Builtin == "" && p.Name == name {
			return 0 // 已有同名条目
		}
	}
	scope := pkg.Scope
	if scope == "" {
		scope = "project"
	}
	ts.Plugins = append(ts.Plugins, ToolsetPlugin{
		Name:    name,
		Purpose: pkg.Purpose,
		Code:    string(code),
		Scope:   scope,
	})
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return 0
	}
	log.Printf("[toolset] start 联动：插件 %s 已加回工作区工具集 %s", name, ts.Name)
	return 1
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
	// ★ 2026-09 Round2（R2-8）工具集去重：内嵌 code 与磁盘插件双份时，磁盘包
	//   是实现载体（装载时序 installToolset → LoadGlobalPlugins，磁盘同名重定义
	//   最终生效；default.json 内嵌 code 是冗余陈旧拷贝，agent-teams 差异最大）。
	//   此处直接跳过内嵌装载（条目降级为 name-only 白名单声明，工具由磁盘包
	//   注册；DisabledTools 经可见性收敛白名单（−DisabledTools）同样生效）。
	//   磁盘包缺失/无效时回退内嵌 code（兼容仅工具集形态的插件）。
	if strings.TrimSpace(p.Code) != "" && p.Builtin == "" && diskPluginCodeAvailable(p.Name) {
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
	id, err := ph.DefineJSCodeFull(p.Code, "", p.Purpose, p.Dir, p.Client)
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
//
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

// ═══════════════════════════════════════════════════════════════
// 工具可见性收敛（★ 装载 ≠ agent 可用）
//
// 语义（2026-08-17）：全部插件照常装载（cordis 可见、可管理），但 agent
// 执行任务时只能看到「工作区工具集（.pair/toolsets/*.json）声明的工具」+
// 「自举管理工具（SystemTool + cordis_*/toolset_* 等循环协议）」。
// 未加入工具集的插件工具：注册保留（cordis/前端可见可管理），对 agent
// 隐藏（Enabled=false）；恢复 = toolset_edit add_plugin 加入工具集。
// 双入口：① LoadJSDynamic 装载钩子（运行期 cordis_run/全局插件即时生效）；
//   ② ApplyToolsetVisibilityFilter 启动全量兜底（LoadAllToolsets 末尾）。
// harness 对齐模式（WB_HARNESS=1）不干预（走 ApplyHarnessToolFilter）。
// ═══════════════════════════════════════════════════════════════

// isAgentProtocolTool 协议/自举管理工具（不依赖工具集声明，恒对 agent 可见）：
//   - SystemTool（宿主会话绑定：update_tasks/tool_stats/history_*）
//   - cordis_*（插件登记/装载/停止/回收/查看——agent 自举链路）
//   - toolset_*（工具集管理——agent 自主构建/编辑工具集）
//   - ask_user / task_create（循环协议）
func isAgentProtocolTool(name string) bool {
	if HarnessAlignedToolNames[name] {
		return true
	}
	if strings.HasPrefix(name, "cordis_") || strings.HasPrefix(name, "toolset_") {
		return true
	}
	if strings.HasPrefix(name, "history_") {
		return true
	}
	switch name {
	case "tool_stats", "task_create":
		return true
	}
	return false
}

// workspaceToolsetVisibleTools 全局通用集合声明的工具白名单（全部工具集并集）：
//   - 内置工具包条目（Builtin）：Tools 清单（用户选择加入的内置组工具）
//   - JS 插件条目：该插件经 pluginTools 注册的工具（工具集插件 = 声明工具对 agent 可见）
//
// 供可见性收敛使用（装载钩子 + 启动全量兜底）。
// ★ 2026-09-04 工具集全局化：读取全局目录（所有通用集合的并集）
//
//	——任何集合声明的工具都保留注册（会话级按所选集合再收敛）。
func (h *PluginHost) workspaceToolsetVisibleTools() map[string]bool {
	keep := map[string]bool{}
	if h == nil {
		return keep
	}
	dir := globalToolsetDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return keep
	}
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
		reg := (*Registry)(nil)
		if h.ctx != nil {
			reg = h.ctx.Tools
		}
		for _, p := range ts.Plugins {
			// ★ 2026-08-17：DisabledTools 摘除清单内的工具不进白名单——
			//   移除后重启保持禁用（可见性收敛据此排除）。
			disabled := map[string]bool{}
			for _, tn := range p.DisabledTools {
				disabled[tn] = true
			}
			if p.Builtin != "" {
				// ★ 2026-08-2x：内置条目仅「真实注册的工具」进白名单——
				//   声明了但宿主没有的工具（组已废弃/工具被移除）视为未暴露。
				for _, tn := range p.Tools {
					if tn == "" || disabled[tn] {
						continue
					}
					if reg != nil {
						if _, ok := reg.Get(tn); !ok {
							continue
						}
					}
					keep[tn] = true
				}
				continue
			}
			if p.Name == "" {
				continue
			}
			// ★ 2026-08-2x：JS 插件条目仅当插件「已启用（running）」时其工具
			//   才进白名单——插件被停止/未装载 → 整条跳过（工具不暴露给 agent）。
			if h.State(p.Name) != PluginRunning {
				continue
			}
			h.mu.RLock()
			tns := append([]string(nil), h.pluginTools[p.Name]...)
			h.mu.RUnlock()
			for _, tn := range tns {
				if tn != "" && !disabled[tn] {
					keep[tn] = true
				}
			}
		}
	}
	return keep
}

// workspaceToolsetDisabledTools 工作区工具集显式摘除的工具集合（全部条目
// DisabledTools 并集）。
// ★ 2026-08-17：摘除清单对「协议/SystemTool 工具」也生效——用户从管理弹窗
//
//	移出的工具写入 DisabledTools，重启后 ApplyToolsetVisibilityFilter 先排除
//	这些工具再强启协议工具，保证移除持久有效（enable_tool 可恢复）。
func (h *PluginHost) workspaceToolsetDisabledTools() map[string]bool {
	out := map[string]bool{}
	if h == nil {
		return out
	}
	dir := globalToolsetDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
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
		for _, p := range ts.Plugins {
			for _, tn := range p.DisabledTools {
				if tn != "" {
					out[tn] = true
				}
			}
		}
	}
	return out
}

// pruneUnavailableFromToolsets 实装后清理全局工具集中「未暴露」的条目/工具并保存
// （★ 2026-08-2x：插件面板未启用/未装载的插件工具不再留在工具集）：
//   - JS 插件条目：插件未装载（未定义或非 running）→ 整条移除（未启用的插件
//     不给工具集——工具集管理/白名单不再出现）。
//   - builtin 条目：Tools/DisabledTools 中未真实注册的工具移除；组内工具全部
//     未暴露（Tools 为空，或全部被 DisabledTools 摘除 = 面板未启用）→ 整条移除。
//
// 幂等；仅启动实装（LoadAllToolsets）完成后调用一次。返回清理了条目的工具集数量。
func pruneUnavailableFromToolsets(ph *PluginHost, root string) int {
	_ = root
	var reg *Registry
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	cleaned := 0
	for _, meta := range listToolsets("", toolsetProject) {
		ts, err := loadToolset("", toolsetProject, meta.Name)
		if err != nil || ts == nil {
			continue
		}
		out := make([]ToolsetPlugin, 0, len(ts.Plugins))
		changed := false
		for i := range ts.Plugins {
			p := ts.Plugins[i]
			// ── JS 插件条目：插件必须已装载（running）才保留 ──
			if p.Builtin == "" && strings.TrimSpace(p.Code) != "" {
				if p.Name == "" || ph == nil || ph.State(p.Name) != PluginRunning {
					changed = true
					log.Printf("[toolset] %s 清理：插件 %q 未启用（未装载），移除工具集条目", ts.Name, p.Name)
					continue
				}
				out = append(out, p)
				continue
			}
			// ── builtin 条目：清理未注册工具；整组未暴露 → 整条移除 ──
			if p.Builtin != "" && strings.TrimSpace(p.Code) == "" {
				if reg == nil {
					out = append(out, p)
					continue
				}
				tools := make([]string, 0, len(p.Tools))
				for _, tn := range p.Tools {
					if _, ok := reg.Get(tn); ok {
						tools = append(tools, tn)
					} else {
						changed = true
						log.Printf("[toolset] %s 清理：内置组 %s 工具 %s 未注册，移除声明", ts.Name, p.Builtin, tn)
					}
				}
				p.Tools = tools
				dis := make([]string, 0, len(p.DisabledTools))
				for _, tn := range p.DisabledTools {
					if _, ok := reg.Get(tn); ok {
						dis = append(dis, tn)
					} else {
						changed = true
						log.Printf("[toolset] %s 清理：内置组 %s 摘除记录 %s 未注册，移除", ts.Name, p.Builtin, tn)
					}
				}
				p.DisabledTools = dis
				// 整组未暴露：无工具声明，或全部工具被摘除（面板未启用）→ 整条移除
				if len(p.Tools) == 0 {
					changed = true
					log.Printf("[toolset] %s 清理：内置组 %s 无可用工具（未启用），移除工具集条目", ts.Name, p.Builtin)
					continue
				}
				allDisabled := true
				for _, tn := range p.Tools {
					if !slices.Contains(p.DisabledTools, tn) {
						allDisabled = false
						break
					}
				}
				if allDisabled {
					changed = true
					log.Printf("[toolset] %s 清理：内置组 %s 工具全部未启用，移除工具集条目", ts.Name, p.Builtin)
					continue
				}
				out = append(out, p)
				continue
			}
			out = append(out, p)
		}
		if changed {
			ts.Plugins = out
			if err := saveToolset("", toolsetProject, ts); err == nil {
				cleaned++
				log.Printf("[toolset] %s 清理完成：未暴露条目已从全局工具集移除并保存", ts.Name)
			}
		}
	}
	return cleaned
}

// ApplyWorkspaceToolsetWhitelist 兼容包装（★ 2026-09-04 已由会话级
// ApplyConvToolsetWhitelist 取代）：按 root 找会话已不可行——工具集已全局化。
// 语义保留为「应用默认集合（default）」：与旧行为（无工具集时默认集合）对齐。
// 调用方应改用 ApplyConvToolsetWhitelist（会话级选择）。
func ApplyWorkspaceToolsetWhitelist(ph *PluginHost, reg *Registry, root string) {
	_ = root
	ApplyToolsetWhitelistByName(ph, reg, "")
}

// ApplyConvToolsetWhitelist 会话级工具集白名单：按会话元数据选择的通用集合
// （ConversationMeta.Toolset；空=default 集合）收敛 agent 可见工具。
// ★ 2026-09-04：工具集全局化——会话选择哪个集合（模式），agent 只暴露该集合
//
//	声明的工具（builtin 条目 Tools + JS 插件工具 − DisabledTools）+ 恒可用
//	协议/管理工具。未声明工具全部禁用（cordis/前端仍可见可管理）。
//
// 幂等；subagent 成员会话同样按自己的 convID 读取（spawn 时继承源会话集合）。
func ApplyConvToolsetWhitelist(ph *PluginHost, reg *Registry, convID, wsRoot string) {
	if reg == nil {
		return
	}
	name := ""
	// ★ 会话元数据优先：会话显式选择过集合 → 用所选集合；未选择/找不到 →
	//   跨 store 兜底再找（防 workspaceRoot 错位）；仍无 → 空 = default 集合。
	if convID != "" {
		if store := storeForConvLookup(convID, wsRoot); store != nil {
			name = store.ConvToolset(convID)
		}
	}
	ApplyToolsetWhitelistByName(ph, reg, name)
}

// storeForConvLookup 按会话定位消息存储（workspaceRoot 指定优先；未命中时
// 跨已打开 store 兜底——与 SessionManager.FindConversation 语义一致）。
func storeForConvLookup(convID, wsRoot string) ConversationStore {
	if m := GlobalSessionManager(); m != nil {
		if wsRoot != "" {
			if st := m.StoreFor(wsRoot); st != nil {
				if meta, err := st.GetConversation(convID); err == nil && meta != nil {
					return st
				}
			}
		}
		if meta, root := m.FindConversation(convID); meta != nil && root != "" {
			return m.StoreFor(root)
		}
	}
	return nil
}

// ApplyToolsetWhitelistByName 按指定工具集名（全局通用集合）应用白名单：
// name 空 = default 集合；default 不存在时先播种（幂等），仍无 → 全量保留
// （无集合配置不收敛——兼容无工具集环境）。
func ApplyToolsetWhitelistByName(ph *PluginHost, reg *Registry, name string) {
	if reg == nil {
		return
	}
	if name == "" {
		name = presetNameDefault
	}
	ts, err := loadToolset("", toolsetProject, name)
	if err != nil {
		// 缺 default：播种预置模式再试一次
		seedPresetToolsets(ph)
		ts, err = loadToolset("", toolsetProject, name)
		if err != nil {
			log.Printf("[toolset] 白名单应用：工具集 %q 不存在，不收敛（全量保留）", name)
			return
		}
	}
	keep := toolsetVisibleToolsFor(ph, ts)
	// ★ 框架本身提供的工具恒可用（无论工具集是否声明）——agent 自举闭环必需：
	//   SystemTool（任务追踪/提问/提交标记/历史/技能/MCP/市场等）+ cordis_* 插件
	//   管理 + toolset_* 工具集管理 + tool-system 插件承载的框架工具。默认工具集
	//   也会显式声明它们（管理面板一致），旧工具集未声明时靠此兜底。
	if ph != nil {
		for _, tn := range ph.PluginToolsByPlugin()["tool-system"] {
			if tn != "" {
				keep[tn] = true
			}
		}
	}
	for _, meta := range reg.AllToolMeta() {
		if meta.SystemTool || isCordisMgmtTool(meta.Name) || isToolsetMgmtTool(meta.Name) {
			keep[meta.Name] = true
		}
	}
	for _, meta := range reg.AllToolMeta() {
		if keep[meta.Name] {
			reg.SetToolEnabled(meta.Name, true)
		} else {
			reg.SetToolEnabled(meta.Name, false)
		}
	}
}

// toolsetVisibleToolsFor 单个工具集声明的白名单（声明可见的工具名集合）：
// builtin 条目 Tools（−DisabledTools）+ JS 插件注册工具（−DisabledTools）。
func toolsetVisibleToolsFor(ph *PluginHost, ts *Toolset) map[string]bool {
	keep := map[string]bool{}
	if ts == nil {
		return keep
	}
	var reg *Registry
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	for _, p := range ts.Plugins {
		disabled := map[string]bool{}
		for _, tn := range p.DisabledTools {
			disabled[tn] = true
		}
		if p.Builtin != "" {
			// ★ 内置条目仅「真实注册的工具」进白名单（未暴露跳过）。
			for _, tn := range p.Tools {
				if tn == "" || disabled[tn] {
					continue
				}
				if reg != nil {
					if _, ok := reg.Get(tn); !ok {
						continue
					}
				}
				keep[tn] = true
			}
			continue
		}
		if p.Name == "" {
			continue
		}
		// ★ JS/磁盘插件条目仅当插件 running 时其工具才进白名单
		//   （插件未启用/未装载 → 整条跳过，工具不暴露）。
		if ph == nil || ph.State(p.Name) != PluginRunning {
			continue
		}
		var tns []string
		if ph != nil {
			tns = ph.PluginToolsByPlugin()[p.Name]
		}
		for _, tn := range tns {
			if tn != "" && !disabled[tn] {
				keep[tn] = true
			}
		}
	}
	return keep
}

// workspaceToolsetVisibleToolsFor 全部全局工具集并集白名单（自由函数版，
// 兼容原有调用方：ensureDefaultWorkspaceToolset 等旧路径；与
// PluginHost.workspaceToolsetVisibleTools 同逻辑）。
func workspaceToolsetVisibleToolsFor(ph *PluginHost, root string) map[string]bool {
	_ = root
	keep := map[string]bool{}
	dir := globalToolsetDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return keep
	}
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
		for tn := range toolsetVisibleToolsFor(ph, &ts) {
			keep[tn] = true
		}
	}
	return keep
}

// applyPluginToolVisibility 插件装载后应用工具可见性（★ 装载 ≠ agent 可用）：
// 插件注册的工具若不在全局工具集白名单（内置条目 Tools / 工具集 JS 插件声明），
// 对 agent 隐藏（Enabled=false）——cordis/前端仍可见可管理，toolset_edit 加入后恢复。
// 在 LoadJSDynamic 装载成功后调用（工具集插件经 applyToolsetPlugin 也走此路径，
// 其工具在白名单内保持启用；非工具集插件经全局装载/cordis_run 装载即被隐藏）。
func (h *PluginHost) applyPluginToolVisibility(name string) {
	if h == nil || name == "" || !hasWorkspaceToolsets() {
		return
	}
	keep := h.workspaceToolsetVisibleTools()
	h.mu.RLock()
	tns := append([]string(nil), h.pluginTools[name]...)
	h.mu.RUnlock()
	for _, tn := range tns {
		if tn != "" && !keep[tn] {
			h.ctx.Tools.SetToolEnabled(tn, false)
		}
	}
}

// hasWorkspaceToolsets 全局是否存在工具集配置（.pair/toolsets/ 下任意 *.json）。
// ★ 2026-08-17：可见性收敛（装载≠可用）仅在「配置了工具集」时生效——
//
//	无工具集保持默认全量（工具注册即对 agent 可见，旧行为）。用户配置
//	工具集（含预置模式播种）后即开始收敛。
// ★ 2026-09-04：工具集全局化——检查全局目录，不再依赖工作区根。
func hasWorkspaceToolsets() bool {
	entries, err := os.ReadDir(globalToolsetDir())
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			return true
		}
	}
	return false
}

// hideToolIfNotInToolset 工具若不在全局工具集白名单 → 禁用（agent 不可见）。
// 供非 LoadJSDynamic 路径（Node 桥插件 ctx.tools.register 等）注册工具后调用，
// 保持「装载 ≠ 可用」语义：插件照常装载（cordis/前端可见可管理），工具仅对
// 工具集声明 + 协议/管理工具可见。harness 模式（WB_HARNESS=1）或无
// 工具集配置时不干预。
func (h *PluginHost) hideToolIfNotInToolset(name string) {
	if h == nil || name == "" || HarnessOnlyTools() || !hasWorkspaceToolsets() {
		return
	}
	if h.workspaceToolsetVisibleTools()[name] {
		return
	}
	h.ctx.Tools.SetToolEnabled(name, false)
}

// ApplyToolsetVisibilityFilter 收敛 agent 可见工具 = 全局工具集声明 + 协议/管理工具。
// ★ 语义（2026-08-17）：装载 ≠ agent 可用。全部插件照常装载（cordis 可见可管理），
//
//	但 agent 执行任务时只能看到「工具集（.pair/toolsets/*.json）声明的工具」+
//	「自举管理工具（SystemTool + cordis_*/toolset_* 等）」。未加入工具集的工具
//	Enabled=false（注册保留、前端可见），恢复 = toolset_edit add_plugin。
//
// ★ 2026-09-04：工具集全局化——声明集 = 全部全局工具集并集（任何集合声明的
//
//	工具都保留）；会话级收敛（是否只暴露所选集合）由 ApplyConvToolsetWhitelist
//	在发送消息时按会话选择执行。
//
// 幂等；harness 对齐模式（WB_HARNESS=1）不干预（走 ApplyHarnessToolFilter）。
func ApplyToolsetVisibilityFilter(r *Registry, ph *PluginHost, root string) int {
	if r == nil || HarnessOnlyTools() || !hasWorkspaceToolsets() {
		return 0
	}
	keep := map[string]bool{}
	// ① 协议/管理工具（SystemTool + cordis_*/toolset_* + 循环协议）
	//    ★ 2026-08-17：工作区工具集显式摘除（DisabledTools）的工具豁免——
	//      用户从管理弹窗移出的工具重启后保持禁用（否则协议工具被无条件
	//      重新启用，移除无效；enable_tool 恢复时从摘除清单移除即可）。
	explicitlyRemoved := map[string]bool{}
	if ph != nil {
		explicitlyRemoved = ph.workspaceToolsetDisabledTools()
	}
	for _, name := range r.Names() {
		if explicitlyRemoved[name] {
			continue
		}
		if isAgentProtocolTool(name) {
			keep[name] = true
			continue
		}
		if t, ok := r.Get(name); ok && t.SystemTool {
			keep[name] = true
		}
	}
	// ② 工作区工具集声明工具（内置条目 Tools + JS 插件 pluginTools）
	if ph != nil {
		for tn := range ph.workspaceToolsetVisibleTools() {
			keep[tn] = true
		}
	}
	// 应用：白名单内启用，白名单外禁用（统计 启用→禁用 数；幂等）
	disabled := 0
	for _, name := range r.Names() {
		if keep[name] {
			if !r.IsEnabled(name) {
				r.SetToolEnabled(name, true)
			}
		} else if r.IsEnabled(name) {
			disabled++
			r.SetToolEnabled(name, false)
		}
	}
	return disabled
}
