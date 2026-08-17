// builtin_toolset.go — 内置工具包（builtin 虚拟工具集）：被过滤工具进插件面板的载体。
//
// 背景（2026-08-16）：harness 对齐模式下 pair 独有工具（codegraph_*/memory_*/
// project_info_*/git_*/debug_*/binary_*/office 等 130+ 个）被过滤（Enabled=false）。
// 用户要求这些工具「放进插件面板」——以内置插件组形态展示（core/git/codegraph/…），
// 每个组一个开关：打开=该组工具全部对 agent 可见（加入工作区），关闭=回到过滤状态。
// 另提供「强制全部加入」开关（add_builtin_all：全部内置组一次性启用）。
//
// 数据模型（★ 2026-08-17 统一为一套逻辑）：内置工具包是「虚拟」工具集
// （名 builtin，scope=builtin，不落盘），每次访问从当前注册表 + 插件宿主实时派生：
//
//	builtin（内置，不可删除）
//	├── core（内置插件组：read_file/write_file/edit_file/…）
//	├── git / codegraph / memory / project-info / binary / debug / office / …
//	├── plugin-mgmt（cordis_* 插件管理工具）
//	├── toolset-mgmt（toolset_* 工具集管理工具）
//	└── system（其余宿主工具：harness 别名 read/write/edit/…、update_tasks 等）
//
// 加入状态：用户/agent 选择加入的内置组固化为工作区主工具集（.pair/toolsets/
// default.json）的 Builtin 条目（与 JS 插件条目同文件）——内置组与工作区工具集
// 共同组成 agent 可用工具集，一套文件、一套逻辑。旧版独立 builtin.json 由
// MigrateLegacyBuiltinJSON 启动时自动迁移并入 default.json 后删除。
// 装载/过滤：installToolset（Builtin 条目启用组内工具）+ workspaceToolsetVisibleTools
// （白名单）全量覆盖 default.json，无需独立路径。

package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// builtinToolsetName 内置工具集名（虚拟展示 + builtin.json 固化文件名）。
const builtinToolsetName = "builtin"

// builtinToolsetScope 内置工具集作用域标记（列表/详情展示用，不落盘）。
const builtinToolsetScope toolsetScope = "builtin"

// manualBuiltinGroup 手动工具条目标记（builtin.json 内 Builtin 字段值）：
// 工具级开关（前端工具列表/文件浏览器手动添加）持久化到该条目的 Tools 快照。
const manualBuiltinGroup = "_manual"

// ─── 数据模型（前端/API 展示） ─────────────────────────────

// BuiltinToolInfo 内置分组内一个工具（含启用状态）。
type BuiltinToolInfo struct {
	Name    string `json:"name"`
	Desc    string `json:"desc"`
	Enabled bool   `json:"enabled"`
}

// BuiltinGroupInfo 一个内置分组（插件面板「内置插件」区块条目）。
type BuiltinGroupInfo struct {
	Name    string            `json:"name"`    // 分组名（core/git/…/system/plugin-mgmt/toolset-mgmt）
	Title   string            `json:"title"`   // 展示标题
	Desc    string            `json:"desc"`    // 描述
	Source  string            `json:"source"`  // builtin（内置插件组）/ system / plugin-mgmt / toolset-mgmt
	Tools   []BuiltinToolInfo `json:"tools"`   // 工具清单（含启用状态）
	Enabled bool              `json:"enabled"` // 组是否全部启用（全部工具对 agent 可见）
	Partial bool              `json:"partial"` // 部分启用（部分工具可见）
	Joined  bool              `json:"joined"`  // 组是否已加入工作区（固化在 builtin.json）
}

// BuiltinToolsetInfo 内置工具集完整信息（/api/toolsets?name=builtin 返回）。
type BuiltinToolsetInfo struct {
	Name     string             `json:"name"`
	Scope    string             `json:"scope"` // "builtin"
	Desc     string             `json:"desc"`
	Groups   []BuiltinGroupInfo `json:"groups"`
	// Plugins 插件面板中存在工具的插件分组（source=plugin）——工作区工具集管理
	// 弹窗的候选池：按插件展示工具，勾选工具加入/移出工作区工具集
	// （toolsetEdit add_plugin / enable_tool / rm_tool / rm_plugin）。
	// ★ 2026-08-17：管理弹窗不再展示宿主核心自举工具组（plugin-mgmt/toolset-mgmt
	//   ——它们本就是 agent 可用工具，无「加入」语义），改为插件工具列表。
	Plugins  []BuiltinGroupInfo `json:"plugins"`
	Joined   []string           `json:"joined"`      // 已加入工作区（固化在 builtin.json）的分组名
	ManualTools []string        `json:"manualTools"` // 手动添加的工具（builtin.json _manual 条目）
	ToolTotal int               `json:"toolTotal"`   // 全部内置工具数
	EnabledTotal int            `json:"enabledTotal"` // 当前启用（agent 可见）的内置工具数
	// WorkspaceToolsets 工作区工具集（project scope）装配信息——toolset_edit /
	// toolset_build 等对工作区工具集（default 等）的增删改即时反映在此
	// （插件/工具/摘除工具）。★ 修复：/api/plugins/builtin 不再固定——
	// 工作区工具集的动态变化（如单独加入某插件工具）可在本字段看到。
	WorkspaceToolsets []WorkspaceToolsetInfo `json:"workspaceToolsets"`
}

// WorkspaceToolsetInfo 工作区工具集简要信息（/api/plugins/builtin 附加返回，
// 不含插件代码，避免响应臃肿）。
type WorkspaceToolsetInfo struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Plugins     []WorkspacePluginInfo `json:"plugins"`
}

// WorkspacePluginInfo 工作区工具集内一个插件（工具 + 摘除工具）。
type WorkspacePluginInfo struct {
	Name          string   `json:"name"`
	Purpose       string   `json:"purpose"`
	Builtin       string   `json:"builtin,omitempty"` // 内置分组名（引用内置组时非空）
	Tools         []string `json:"tools,omitempty"`
	DisabledTools []string `json:"disabledTools,omitempty"`
}

// ─── 分组派生 ──────────────────────────────────────────────

// builtinPluginMeta 内置插件组元信息（名+描述；来自 builtinPluginSpecs 规格表）。
type builtinPluginMeta struct {
	name string
	desc string
}

// builtinPluginToolGroups 内置插件组 → 工具名清单（静态派生，缓存）。
// 原理：内置插件工具经 Registry.Register 直接注册（不经 PluginHost.addPluginTool，
// pluginTools 为空；且 RegisterDefaultTools 预注册使运行时 diff 失效），
// 故在独立临时 Registry 上按 spec 顺序逐个 apply 取「新增工具」差异——工具名
// 与 root 无关（root 只进 handler 闭包），可安全缓存。
var (
	builtinGroupsOnce sync.Once
	builtinGroupsMap  map[string][]string
)

func builtinPluginToolGroups() map[string][]string {
	builtinGroupsOnce.Do(func() {
		tmp := NewRegistry()
		builtinGroupsMap = map[string][]string{}
		for _, s := range builtinPluginSpecs("") {
			before := make(map[string]bool)
			for _, n := range tmp.Names() {
				before[n] = true
			}
			s.apply(&PluginContext{Tools: tmp})
			for _, n := range tmp.Names() {
				if !before[n] {
					builtinGroupsMap[s.name] = append(builtinGroupsMap[s.name], n)
				}
			}
		}
		reassignBuiltinToolGroups(builtinGroupsMap)
	})
	return builtinGroupsMap
}

// reassignBuiltinToolGroups 归属修正：registerCoreTools 历史聚合调用
// registerCodeGraphTools/registerExtraCodeGraphTools
// （tools.go 末尾），导致 core 组 diff 抢注了 codegraph 工具、独立组
// （codegraph/codegraph-extra）diff 为空。按精确名/前缀重新归属，
// 保证各内置插件组工具清单独立准确（前端分组开关 + 内置工具集加入按组生效）。
func reassignBuiltinToolGroups(m map[string][]string) {
	reassign := []struct {
		match func(string) bool
		group string
	}{
		{func(n string) bool { return n == "codegraph_find_by_signature" || n == "codegraph_explore" }, "codegraph-extra"},
		{func(n string) bool { return strings.HasPrefix(n, "codegraph_") }, "codegraph"},
	}
	if core, ok := m["core"]; ok {
		var keep []string
		for _, n := range core {
			moved := false
			for _, r := range reassign {
				if r.match(n) {
					m[r.group] = append(m[r.group], n)
					moved = true
					break
				}
			}
			if !moved {
				keep = append(keep, n)
			}
		}
		m["core"] = keep
	}
	for _, list := range m {
		sort.Strings(list)
	}
}

// builtinPluginMetas 内置插件组元信息全表（顺序即 builtinPluginSpecs 装配顺序）。
func builtinPluginMetas() []builtinPluginMeta {
	var out []builtinPluginMeta
	for _, s := range builtinPluginSpecs("") {
		out = append(out, builtinPluginMeta{name: s.name, desc: s.desc})
	}
	return out
}

// isBuiltinPluginName 判断插件名是否为内置插件组（core/git/codegraph/…）。
func isBuiltinPluginName(name string) bool {
	for _, m := range builtinPluginMetas() {
		if m.name == name {
			return true
		}
	}
	return false
}

// builtinGroupDesc 取内置分组描述（未知组返回空）。
func builtinGroupDesc(name string) string {
	for _, m := range builtinPluginMetas() {
		if m.name == name {
			return m.desc
		}
	}
	return ""
}

// isCordisMgmtTool 判断工具是否为插件管理工具（cordis_*，plugin-mgmt 组）。
func isCordisMgmtTool(name string) bool {
	return strings.HasPrefix(name, "cordis_")
}

// isToolsetMgmtTool 判断工具是否为工具集管理工具（toolset_*，toolset-mgmt 组）。
func isToolsetMgmtTool(name string) bool {
	return strings.HasPrefix(name, "toolset_")
}

// BuiltinGroupsOf 派生内置分组（含工具与启用状态）。
// reg 为工具注册表（通常 ph.Context().Tools）；ph 提供 JS 插件工具归属（排除用）。
// 分组构成：
//  1. 已加入工作区的内置组（工作区工具集 Builtin 条目；未加入的不再展示——
//     工具已全部由磁盘插件 .pair/plugins/tool-* 承载，插件面板可见可管理）
//  2. plugin-mgmt：cordis_* 工具
//  3. toolset-mgmt：toolset_* 工具
//  4. system：其余无插件归属的宿主工具
func BuiltinGroupsOf(reg *Registry, ph *PluginHost) []BuiltinGroupInfo {
	if reg == nil {
		return nil
	}
	var groups []BuiltinGroupInfo
	covered := map[string]bool{} // 已归组工具名

	// 1. 已加入工作区的内置组（工作区工具集 *.json 的 Builtin 条目 → 工具清单）。
	//    ★ 2026-08-16 第三轮：不再从 builtinPluginSpecs 静态派生全部内置组
	//      （宿主不再注册内置工具，静态清单会与磁盘插件重复展示）
	//    ★ 2026-08-17：builtin.json 独立机制废除，加入状态就是工作区工具集条目，
	//      扫描全部工具集收集（default.json 等）。
	joinedEntries := map[string][]string{}
	if ph != nil && ph.Context() != nil && ph.Context().WorkspaceRoot != "" {
		for _, m := range listToolsets(ph.Context().WorkspaceRoot, toolsetProject) {
			ts, err := loadToolset(ph.Context().WorkspaceRoot, toolsetProject, m.Name)
			if err != nil {
				continue
			}
			for _, p := range ts.Plugins {
				if p.Builtin != "" && p.Builtin != manualBuiltinGroup {
					joinedEntries[p.Builtin] = append([]string(nil), p.Tools...)
				}
			}
		}
	}
	var joinedNames []string
	for n := range joinedEntries {
		joinedNames = append(joinedNames, n)
	}
	sort.Strings(joinedNames)
	for _, name := range joinedNames {
		tools := joinedEntries[name]
		if len(tools) == 0 {
			continue
		}
		sort.Strings(tools)
		g := BuiltinGroupInfo{Name: name, Title: name, Desc: builtinGroupDesc(name), Source: "builtin"}
		all := true
		anyOn := false
		for _, tn := range tools {
			en := reg.IsEnabled(tn)
			g.Tools = append(g.Tools, BuiltinToolInfo{Name: tn, Desc: toolShortDesc(reg, tn), Enabled: en})
			covered[tn] = true
			if en {
				anyOn = true
			} else {
				all = false
			}
		}
		g.Enabled = all && anyOn
		g.Partial = anyOn && !all
		groups = append(groups, g)
	}

	// 2/3/4. 剩余工具按名字/归属分组
	var sysTools []string
	var mgmtTools []string
	var tsTools []string
	owners := map[string]string{}
	if ph != nil {
		owners = ph.PluginToolOwners()
	}
	for _, meta := range reg.AllToolMeta() {
		if covered[meta.Name] {
			continue
		}
		if _, isPlugin := owners[meta.Name]; isPlugin {
			continue // JS 动态插件注册的工具（非内置，单独在插件面板）
		}
		switch {
		case isCordisMgmtTool(meta.Name):
			mgmtTools = append(mgmtTools, meta.Name)
		case isToolsetMgmtTool(meta.Name):
			tsTools = append(tsTools, meta.Name)
		default:
			sysTools = append(sysTools, meta.Name)
		}
	}
	sort.Strings(sysTools)
	sort.Strings(mgmtTools)
	sort.Strings(tsTools)

	groupFrom := func(name, title, desc, source string, tools []string) BuiltinGroupInfo {
		g := BuiltinGroupInfo{Name: name, Title: title, Desc: desc, Source: source}
		all := true
		anyOn := false
		for _, tn := range tools {
			en := reg.IsEnabled(tn)
			g.Tools = append(g.Tools, BuiltinToolInfo{Name: tn, Desc: toolShortDesc(reg, tn), Enabled: en})
			if en {
				anyOn = true
			} else {
				all = false
			}
		}
		g.Enabled = all && anyOn && len(tools) > 0
		g.Partial = anyOn && !all
		return g
	}
	if len(mgmtTools) > 0 {
		groups = append(groups, groupFrom("plugin-mgmt", "插件管理", "cordis_* 插件管理工具（登记/装载/停止/回收/查看）", "plugin-mgmt", mgmtTools))
	}
	if len(tsTools) > 0 {
		groups = append(groups, groupFrom("toolset-mgmt", "工具集管理", "toolset_* 工具集管理工具（构建/列表/编辑/导出/导入）", "toolset-mgmt", tsTools))
	}
	if len(sysTools) > 0 {
		// ★ system 组可能已在步骤 1 展示（已加入的 builtin:system 快照）——
		//   剩余宿主工具合并进该组，避免同组双展示（否则 builtinEntryOfGroup 取首个
		//   会漏掉剩余工具，add_builtin_all 不全量启用）
		merged := false
		for i := range groups {
			if groups[i].Name == "system" {
				groups[i] = mergeBuiltinGroupTools(groups[i], reg, sysTools)
				merged = true
				break
			}
		}
		if !merged {
			groups = append(groups, groupFrom("system", "系统工具", "其余宿主工具（harness 别名/任务追踪/提问/提交标记等）", "system", sysTools))
		}
	}
	return groups
}

// mergeBuiltinGroupTools 把附加工具合并进已展示的内置组（去重 + 重算启用状态）。
func mergeBuiltinGroupTools(g BuiltinGroupInfo, reg *Registry, extra []string) BuiltinGroupInfo {
	have := map[string]bool{}
	for _, t := range g.Tools {
		have[t.Name] = true
	}
	for _, tn := range extra {
		if tn == "" || have[tn] {
			continue
		}
		have[tn] = true
		g.Tools = append(g.Tools, BuiltinToolInfo{Name: tn, Desc: toolShortDesc(reg, tn), Enabled: reg.IsEnabled(tn)})
	}
	sort.Slice(g.Tools, func(i, j int) bool { return g.Tools[i].Name < g.Tools[j].Name })
	all := len(g.Tools) > 0
	anyOn := false
	for _, t := range g.Tools {
		if t.Enabled {
			anyOn = true
		} else {
			all = false
		}
	}
	g.Enabled = all && anyOn
	g.Partial = anyOn && !all
	return g
}

// toolShortDesc 取工具简短描述（截断，前端展示用）。
func toolShortDesc(reg *Registry, name string) string {
	if reg == nil {
		return ""
	}
	t, ok := reg.Get(name)
	if !ok || t.Description == "" {
		return ""
	}
	d := t.Description
	if len([]rune(d)) > 60 {
		runes := []rune(d)
		return string(runes[:60]) + "…"
	}
	return d
}

// builtinJoinedGroups 已加入工作区（固化在工作区工具集 .pair/toolsets/*.json
// 的 Builtin 条目）的分组名。
func builtinJoinedGroups(root string) []string {
	var out []string
	for _, m := range listToolsets(root, toolsetProject) {
		ts, err := loadToolset(root, toolsetProject, m.Name)
		if err != nil {
			continue
		}
		for _, p := range ts.Plugins {
			if p.Builtin != "" && p.Builtin != manualBuiltinGroup {
				out = append(out, p.Builtin)
			}
		}
	}
	sort.Strings(out)
	return out
}

// workspaceMainToolset 工作区主工具集（default）——内置组加入/移出/手动工具的
// 统一落点（★ 2026-08-17：内置工具包与工作区工具集合并为一套，无独立 builtin.json）。
// 不存在时创建基础工具集（defaultProjectToolset，dsh 极简核心），与
// ensureDefaultWorkspaceToolset 语义一致。
func workspaceMainToolset(root string) (*Toolset, error) {
	if root == "" {
		return nil, fmt.Errorf("工作区未就绪")
	}
	if ts, err := loadToolset(root, toolsetProject, "default"); err == nil {
		return ts, nil
	}
	ts := defaultProjectToolset(filepath.Base(root))
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// MigrateLegacyBuiltinJSON 迁移旧版独立固化文件 .pair/toolsets/builtin.json：
// 其中的内置组条目（Builtin 非空）并入工作区主工具集（default.json，去重），
// 然后删除旧文件。
// ★ 2026-08-17：内置工具包与工作区工具集统一为「一套逻辑」——内置组的加入状态
//   就是工作区工具集条目（与 JS 插件条目同文件），不再有独立 builtin.json。
// 幂等：无旧文件时不做任何事。启动装配（LoadAllToolsets 前）调用。
func MigrateLegacyBuiltinJSON(root string) error {
	if root == "" {
		return nil
	}
	path := toolsetPath(root, toolsetProject, builtinToolsetName)
	if _, err := os.Stat(path); err != nil {
		return nil // 无旧文件，无需迁移
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var legacy Toolset
	if err := json.Unmarshal(data, &legacy); err != nil || legacy.Name == "" {
		// 无效旧文件直接删除（不阻塞启动）
		_ = os.Remove(path)
		return nil
	}
	ts, err := workspaceMainToolset(root)
	if err != nil {
		return err
	}
	migrated := 0
	for _, p := range legacy.Plugins {
		if p.Builtin == "" {
			continue // 非内置条目（理论上旧文件只存内置条目）不迁移
		}
		dup := false
		for _, e := range ts.Plugins {
			if e.Builtin == p.Builtin {
				dup = true
				break
			}
		}
		if !dup {
			ts.Plugins = append(ts.Plugins, p)
			migrated++
		}
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	log.Printf("[toolset] 已迁移旧版 builtin.json → %s（%d 个内置组条目并入工作区工具集，旧文件删除）", ts.Name, migrated)
	return nil
}

// BuiltinToolsetInfoOf 派生内置工具集完整信息（列表/详情 API 用）。
func BuiltinToolsetInfoOf(reg *Registry, ph *PluginHost, root string) *BuiltinToolsetInfo {
	groups := BuiltinGroupsOf(reg, ph)
	joined := builtinJoinedGroups(root)
	joinedSet := map[string]bool{}
	for _, n := range joined {
		joinedSet[n] = true
	}
	for i := range groups {
		// ★ Joined 只标记内置组（Source="builtin"）——管理组（plugin-mgmt/
		// toolset-mgmt/system）组名可能与内置组同名（如 system），不能误标
		if groups[i].Source == "builtin" && joinedSet[groups[i].Name] {
			groups[i].Joined = true
		}
	}
	info := &BuiltinToolsetInfo{
		Name:         builtinToolsetName,
		Scope:        "builtin",
		Desc:         "内置工具包（默认不加入工作区；分组开关加入，add_builtin_all 强制全部）",
		Groups:       groups,
		Joined:       joined,
		ManualTools:  []string{},
	}
	// _manual 手动条目工具（工具级添加，固化工作区工具集）
	for _, m := range listToolsets(root, toolsetProject) {
		ts, err := loadToolset(root, toolsetProject, m.Name)
		if err != nil {
			continue
		}
		for _, p := range ts.Plugins {
			if p.Builtin == manualBuiltinGroup {
				info.ManualTools = append([]string{}, p.Tools...)
				break
			}
		}
	}
	for _, g := range groups {
		info.ToolTotal += len(g.Tools)
		for _, t := range g.Tools {
			if t.Enabled {
				info.EnabledTotal++
			}
		}
	}
	// 工作区工具集（project scope）：让接口即时反映 toolset_edit / toolset_build
	// 对工具集的动态修改（如单独加入某插件工具）。listToolsets 已跳过 builtin.json。
	if root != "" {
		for _, m := range listToolsets(root, toolsetProject) {
			ts, err := loadToolset(root, toolsetProject, m.Name)
			if err != nil || ts == nil {
				continue
			}
			wi := WorkspaceToolsetInfo{Name: ts.Name, Description: ts.Description}
			for _, p := range ts.Plugins {
				wi.Plugins = append(wi.Plugins, WorkspacePluginInfo{
					Name:          p.Name,
					Purpose:       p.Purpose,
					Builtin:       p.Builtin,
					Tools:         append([]string(nil), p.Tools...),
					DisabledTools: append([]string(nil), p.DisabledTools...),
				})
			}
			info.WorkspaceToolsets = append(info.WorkspaceToolsets, wi)
		}
	}
	// ★ 插件面板中存在工具的插件分组（source=plugin）——工作区工具集管理弹窗
	//   候选池：每个有工具的插件一组，组内是该插件注册的工具（含启用状态）。
	//   数据源 ph.PluginToolsByPlugin（插件名→工具清单）+ reg.IsEnabled（启用状态）。
	//   （2026-08-17：管理弹窗不再展示宿主核心自举组 plugin-mgmt/toolset-mgmt——
	//     它们本就是 agent 可用工具；插件工具才是用户可勾选加入/移出的对象。）
	info.Plugins = pluginGroupsOf(reg, ph, info.WorkspaceToolsets)
	return info
}

// pluginGroupsOf 插件面板中存在工具的插件分组（source=plugin，含工具+启用状态）。
// 过滤条件：插件已注册工具（len(tools)>0）；组名=插件名，标题=插件名（tool-* 语义清晰）。
// Joined：插件是否已加入工作区工具集（任一工具集 JS 插件条目中 Name 匹配）——
// 前端据此选择加入方式（未加入→add_plugin；已加入→enable_tool 恢复）。
func pluginGroupsOf(reg *Registry, ph *PluginHost, workspaceToolsets []WorkspaceToolsetInfo) []BuiltinGroupInfo {
	if ph == nil || reg == nil {
		return nil
	}
	// 已加入工作区工具集的插件名集合（JS 插件条目：Builtin 为空）
	joinedPlugins := map[string]bool{}
	for _, wt := range workspaceToolsets {
		for _, p := range wt.Plugins {
			if p.Builtin == "" && p.Name != "" {
				joinedPlugins[p.Name] = true
			}
		}
	}
	byPlugin := ph.PluginToolsByPlugin()
	// ★ 工具 enabled 状态按「root 工作区工具集声明」计算（插件已加入且工具未被
	// DisabledTools 摘除 → 已加入/可见）。2026-08-17：不再依赖全局 ph 运行时状态
	// （reg.IsEnabled）——不同工作区各自管理自己的 .pair/toolsets/，管理弹窗按
	// 当前工作区隔离展示，互不影响。
	declared := map[string]map[string]bool{} // 插件名 → 工具名 → 声明可用（未摘除）
	for _, wt := range workspaceToolsets {
		for _, p := range wt.Plugins {
			if p.Builtin != "" || p.Name == "" {
				continue
			}
			if declared[p.Name] == nil {
				declared[p.Name] = map[string]bool{}
			}
			for _, tn := range byPlugin[p.Name] {
				declared[p.Name][tn] = true
			}
			for _, tn := range p.DisabledTools {
				declared[p.Name][tn] = false
			}
		}
	}
	names := make([]string, 0, len(byPlugin))
	for n := range byPlugin {
		names = append(names, n)
	}
	sort.Strings(names)
	var groups []BuiltinGroupInfo
	for _, name := range names {
		tools := byPlugin[name]
		if name == "" || len(tools) == 0 {
			continue // 空名/无工具的插件不展示（空 key 为历史残留注册，无操作对象）
		}
		sort.Strings(tools)
		g := BuiltinGroupInfo{Name: name, Title: name, Desc: pluginPurposeOf(ph, name), Source: "plugin"}
		g.Joined = joinedPlugins[name]
		anyOn := false
		all := true
		for _, tn := range tools {
			en := false
			if d, ok := declared[name]; ok {
				en = d[tn]
			}
			g.Tools = append(g.Tools, BuiltinToolInfo{Name: tn, Desc: toolShortDesc(reg, tn), Enabled: en})
			if en {
				anyOn = true
			} else {
				all = false
			}
		}
		g.Enabled = all && anyOn && len(tools) > 0
		g.Partial = anyOn && !all
		groups = append(groups, g)
	}
	return groups
}

// pluginPurposeOf 取插件用途描述（Inspect 记录里的 Purpose；磁盘插件 tool-* 有）。
func pluginPurposeOf(ph *PluginHost, name string) string {
	for _, rec := range ph.Inspect() {
		if rec.Name == name && rec.Purpose != "" {
			return rec.Purpose
		}
	}
	return ""
}


// BuiltinToolsetOf 派生内置工具集（Toolset 形态，scope=builtin 虚拟）。
// 每个分组一个内置条目（Builtin 非空、无 Code）——toolset_show / 导出展示用。
func BuiltinToolsetOf(reg *Registry, ph *PluginHost) *Toolset {
	groups := BuiltinGroupsOf(reg, ph)
	ts := &Toolset{Name: builtinToolsetName, Description: "内置工具包（默认不加入工作区；选择加入后固化到 .pair/toolsets/builtin.json）"}
	for _, g := range groups {
		p := ToolsetPlugin{Name: "builtin:" + g.Name, Purpose: g.Title + "：" + g.Desc, Builtin: g.Name}
		for _, t := range g.Tools {
			p.Tools = append(p.Tools, t.Name)
		}
		ts.Plugins = append(ts.Plugins, p)
	}
	return ts
}

// ─── 加入/移出/强制全部（持久化 + 热装载） ─────────────────

// builtinEntryOfGroup 从内置分组生成工具集条目（含组内工具快照）。
// 数据源统一走 BuiltinGroupsOf（分组派生单一入口），避免规则重复。
func builtinEntryOfGroup(reg *Registry, ph *PluginHost, groupName string) (*ToolsetPlugin, error) {
	// ① 已展示组（已加入/管理组）
	for _, g := range BuiltinGroupsOf(reg, ph) {
		if g.Name != groupName {
			continue
		}
		entry := &ToolsetPlugin{
			Name:    "builtin:" + g.Name,
			Purpose: g.Title + "：" + g.Desc,
			Builtin: g.Name,
		}
		for _, t := range g.Tools {
			entry.Tools = append(entry.Tools, t.Name)
		}
		if len(entry.Tools) == 0 {
			return nil, os.ErrNotExist // 组无工具
		}
		return entry, nil
	}
	// ② ★ 未展示的内置组（迁移磁盘插件后前端不再展示，但加入操作仍可用）：
	//    从组规格静态派生工具清单（builtinPluginSpecs，仅作实现库组规格）
	if tools := builtinPluginToolGroups()[groupName]; len(tools) > 0 {
		entry := &ToolsetPlugin{
			Name:    "builtin:" + groupName,
			Purpose: builtinGroupDesc(groupName),
			Builtin: groupName,
		}
		entry.Tools = append(entry.Tools, tools...)
		return entry, nil
	}
	return nil, os.ErrNotExist
}

// applyBuiltinGroupToToolset 把内置分组加入工作区 builtin 工具集（固化）并热装载。
// 返回加入后的消息文本。
func applyBuiltinGroupToToolset(ph *PluginHost, root string, groupName string, forceAll bool) (string, error) {
	reg := (*Registry)(nil)
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	// 目标分组清单：单个组 或 全部组
	var groups []string
	if forceAll {
		for _, g := range BuiltinGroupsOf(reg, ph) {
			groups = append(groups, g.Name)
		}
	} else {
		if groupName == "" {
			return "", os.ErrInvalid
		}
		groups = []string{groupName}
	}
	// 读取/创建工作区主工具集（default——内置组加入/移出的统一落点，无独立 builtin.json）
	ts, err := workspaceMainToolset(root)
	if err != nil {
		return "", err
	}
	added := 0
	for _, gn := range groups {
		entry, e := builtinEntryOfGroup(reg, ph, gn)
		if e != nil {
			continue // 组无工具，跳过
		}
		// 已存在同组条目：先恢复默认再重加（覆盖最新工具清单）
		for i := range ts.Plugins {
			if ts.Plugins[i].Builtin == gn {
				unloadToolsetPlugin(ph, &ts.Plugins[i])
				ts.Plugins = append(ts.Plugins[:i], ts.Plugins[i+1:]...)
				break
			}
		}
		if err := applyToolsetPlugin(ph, entry); err != nil {
			return "", err
		}
		ts.Plugins = append(ts.Plugins, *entry)
		added++
	}
	if added == 0 {
		return "", os.ErrNotExist
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	if forceAll {
		return "✅ 已强制加入全部内置工具组（" + itoaAgent(len(groups)) + " 组，" + itoaAgent(len(ts.Plugins)) + " 个内置条目）→ 组内工具全部对 agent 可见。固化工作区工具集 " + ts.Name + ".json", nil
	}
	return "✅ 内置工具组 " + groupName + " 已加入工作区（组内工具全部对 agent 可见）。固化工作区工具集 " + ts.Name + ".json", nil
}

// removeBuiltinGroupFromToolset 把内置分组移出工作区工具集（恢复默认过滤状态）。
func removeBuiltinGroupFromToolset(ph *PluginHost, root string, groupName string) (string, error) {
	ts, err := workspaceMainToolset(root)
	if err != nil {
		return "", err
	}
	for i := range ts.Plugins {
		if ts.Plugins[i].Builtin == groupName {
			unloadToolsetPlugin(ph, &ts.Plugins[i])
			ts.Plugins = append(ts.Plugins[:i], ts.Plugins[i+1:]...)
			break
		}
	}
	if len(ts.Plugins) == 0 {
		// 全部移出：重置为基础工具集（default 是工作区工具集主文件，永存——
		// 否则「工作区有工具集」状态丢失，agent 可见工具会回到全量/默认语义）
		base := defaultProjectToolset(filepath.Base(root))
		if err := saveToolset(root, toolsetProject, base); err != nil {
			return "", err
		}
		return "✅ 内置工具组 " + groupName + " 已移出工作区（组内工具恢复默认过滤状态）", nil
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	return "✅ 内置工具组 " + groupName + " 已移出工作区（组内工具恢复默认过滤状态）。工作区工具集 " + ts.Name + ".json 已更新", nil
}

// SetBuiltinGroupEnabled 内置分组开关（启用=组内工具全部对 agent 可见并固化；
// 禁用=移出工作区恢复默认）。返回操作结果文本。
func SetBuiltinGroupEnabled(ph *PluginHost, root string, groupName string, enabled bool) (string, error) {
	if enabled {
		return applyBuiltinGroupToToolset(ph, root, groupName, false)
	}
	return removeBuiltinGroupFromToolset(ph, root, groupName)
}

// EnableAllBuiltin 强制全部内置工具组加入工作区（开关：add_builtin_all）。
func EnableAllBuiltin(ph *PluginHost, root string) (string, error) {
	return applyBuiltinGroupToToolset(ph, root, "", true)
}

// SetBuiltinToolEnabled 内置工具级开关（前端工具列表/文件浏览器「手动添加工具」）。
//   - enabled=true：工具加入 agent 可用（SetToolEnabled(true)），持久化到 builtin.json
//     的 _manual 手动条目（Tools 快照幂等追加）
//   - enabled=false：工具恢复默认状态（ToolDefaultEnabled——harness 保留清单内保持启用），
//     从 _manual 移除；若工具在某已加入组条目快照内，加入该组 DisabledTools 保持禁用。
//   固化文件全空时删除（空工具集不落盘）。
func SetBuiltinToolEnabled(ph *PluginHost, root, toolName string, enabled bool) (string, error) {
	reg := (*Registry)(nil)
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	// 校验工具属于内置工具包（全量工具集合）
	valid := map[string]bool{}
	for _, g := range BuiltinGroupsOf(reg, ph) {
		for _, t := range g.Tools {
			valid[t.Name] = true
		}
	}
	if !valid[toolName] {
		return "", fmt.Errorf("工具 %s 不存在或不属于内置工具包", toolName)
	}
	ts, err := workspaceMainToolset(root)
	if err != nil {
		return "", err
	}

	if enabled {
		idx := -1
		for i := range ts.Plugins {
			if ts.Plugins[i].Builtin == manualBuiltinGroup {
				idx = i
				break
			}
		}
		if idx < 0 {
			ts.Plugins = append(ts.Plugins, ToolsetPlugin{
				Name: "builtin:" + manualBuiltinGroup, Purpose: "手动添加的工具（工具级开关）", Builtin: manualBuiltinGroup,
			})
			idx = len(ts.Plugins) - 1
		}
		has := false
		for _, tn := range ts.Plugins[idx].Tools {
			if tn == toolName {
				has = true
				break
			}
		}
		if !has {
			ts.Plugins[idx].Tools = append(ts.Plugins[idx].Tools, toolName)
		}
		if reg != nil {
			reg.SetToolEnabled(toolName, true)
		}
		if err := saveToolset(root, toolsetProject, ts); err != nil {
			return "", err
		}
		return "✅ 工具 " + toolName + " 已加入 agent 可用（手动工具）。固化工作区工具集 " + ts.Name + ".json", nil
	}

	// 禁用：强制移出 agent 可用集合（SetToolEnabled(false)）+ 持久化差集
	// （2026-08-16：原为恢复 ToolDefaultEnabled——默认全量模式下=全 true，
	//   移出后工具仍可见，前端「移出」无效。改为强制禁用，重启后差集保持一致。）
	if reg != nil {
		reg.SetToolEnabled(toolName, false)
	}
	for i := range ts.Plugins {
		if ts.Plugins[i].Builtin == manualBuiltinGroup {
			removeToolName(&ts.Plugins[i].Tools, toolName)
			break
		}
	}
	for i := range ts.Plugins {
		p := &ts.Plugins[i]
		if p.Builtin == "" || p.Builtin == manualBuiltinGroup {
			continue
		}
		for _, tn := range p.Tools {
			if tn == toolName {
				addToolName(&p.DisabledTools, toolName)
				break
			}
		}
	}
	// 清空空手动条目
	for i := range ts.Plugins {
		if ts.Plugins[i].Builtin == manualBuiltinGroup && len(ts.Plugins[i].Tools) == 0 {
			ts.Plugins = append(ts.Plugins[:i], ts.Plugins[i+1:]...)
			break
		}
	}
	if len(ts.Plugins) == 0 {
		// 全部移出：重置为基础工具集（default 永存，见 removeBuiltinGroupFromToolset）
		base := defaultProjectToolset(filepath.Base(root))
		if err := saveToolset(root, toolsetProject, base); err != nil {
			return "", err
		}
		return "✅ 工具 " + toolName + " 已移出 agent 可用集合（工作区工具集）", nil
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	return "✅ 工具 " + toolName + " 已移出 agent 可用集合（工作区工具集）。工作区工具集 " + ts.Name + ".json 已更新", nil
}

// removeToolName 从字符串切片移除指定元素（无则不动）。
func removeToolName(list *[]string, name string) {
	for i := range *list {
		if (*list)[i] == name {
			*list = append((*list)[:i], (*list)[i+1:]...)
			return
		}
	}
}

// addToolName 向字符串切片幂等追加元素。
func addToolName(list *[]string, name string) {
	for _, n := range *list {
		if n == name {
			return
		}
	}
	*list = append(*list, name)
}

// itoaAgent 简易 int→string（避免 import strconv 噪音）。
func itoaAgent(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}


