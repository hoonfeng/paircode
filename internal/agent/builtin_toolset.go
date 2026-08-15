// builtin_toolset.go — 内置工具集（builtin）：被过滤工具进插件面板的载体。
//
// 背景（2026-08-16）：harness 对齐模式下 pair 独有工具（codegraph_*/memory_*/
// project_info_*/git_*/debug_*/binary_*/office 等 130+ 个）被过滤（Enabled=false）。
// 用户要求这些工具「放进插件面板」——以内置插件组形态展示（core/git/codegraph/…），
// 每个组一个开关：打开=该组工具全部对 agent 可见（加入工作区），关闭=回到过滤状态。
// 另提供「强制全部加入」开关（add_builtin_all：全部内置组一次性启用）。
//
// 数据模型：内置工具集是「虚拟」工具集（名 builtin，scope=builtin，不落盘），
// 每次访问从当前注册表 + 插件宿主实时派生：
//
//	builtin（内置，不可删除）
//	├── core（内置插件组：read_file/write_file/edit_file/…）
//	├── git / codegraph / memory / project-info / binary / debug / office / …
//	├── plugin-mgmt（cordis_* 插件管理工具）
//	├── toolset-mgmt（toolset_* 工具集管理工具）
//	└── system（其余宿主工具：harness 别名 read/write/edit/…、update_tasks 等）
//
// 持久化：用户/agent 选择加入的组固化到工作区 .pair/toolsets/builtin.json
// （普通 Toolset 结构，Plugins=已加入的内置条目）——启动 LoadAllToolsets 自动装载
// （applyToolsetPlugin 启用组内工具）；会话级注册表构建时 ApplyToolsetBuiltinState 应用。
// builtin.json 是持久化载体，不作为普通工具集列表展示（listToolsets 跳过）。

package agent

import (
	"fmt"
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
}

// BuiltinToolsetInfo 内置工具集完整信息（/api/toolsets?name=builtin 返回）。
type BuiltinToolsetInfo struct {
	Name     string             `json:"name"`
	Scope    string             `json:"scope"` // "builtin"
	Desc     string             `json:"desc"`
	Groups   []BuiltinGroupInfo `json:"groups"`
	Joined   []string           `json:"joined"`   // 已加入工作区（固化在 builtin.json）的分组名
	ToolTotal int               `json:"toolTotal"` // 全部内置工具数
	EnabledTotal int            `json:"enabledTotal"` // 当前启用（agent 可见）的内置工具数
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
// registerCodeGraphTools/registerExtraCodeGraphTools/registerLuaToolTools
// （tools.go 末尾），导致 core 组 diff 抢注了 codegraph/lua 工具、独立组
// （codegraph/codegraph-extra/lua-tools）diff 为空。按精确名/前缀重新归属，
// 保证各内置插件组工具清单独立准确（前端分组开关 + 内置工具集加入按组生效）。
func reassignBuiltinToolGroups(m map[string][]string) {
	reassign := []struct {
		match func(string) bool
		group string
	}{
		{func(n string) bool { return n == "codegraph_find_by_signature" || n == "codegraph_explore" }, "codegraph-extra"},
		{func(n string) bool { return strings.HasPrefix(n, "codegraph_") }, "codegraph"},
		{func(n string) bool { return strings.HasPrefix(n, "lua_tool") }, "lua-tools"},
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

// BuiltinGroupsOf 派生全部内置分组（含工具与启用状态）。
// reg 为工具注册表（通常 ph.Context().Tools）；ph 提供 JS 插件工具归属（排除用）。
// 分组构成：
//  1. 内置插件组（builtinPluginMetas）：工具 = builtinPluginToolGroups()（静态派生）
//  2. plugin-mgmt：cordis_* 工具
//  3. toolset-mgmt：toolset_* 工具
//  4. system：其余无插件归属的宿主工具
func BuiltinGroupsOf(reg *Registry, ph *PluginHost) []BuiltinGroupInfo {
	if reg == nil {
		return nil
	}
	var groups []BuiltinGroupInfo
	covered := map[string]bool{} // 已归组工具名

	// 1. 内置插件组（仅取有工具注册的组）
	groupsByPlugin := builtinPluginToolGroups()
	for _, m := range builtinPluginMetas() {
		tools := groupsByPlugin[m.name]
		if len(tools) == 0 {
			continue // 插件未注册工具（如 lua-tools 无 lua 文件时）
		}
		sort.Strings(tools)
		g := BuiltinGroupInfo{Name: m.name, Title: m.name, Desc: m.desc, Source: "builtin"}
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
		groups = append(groups, groupFrom("system", "系统工具", "其余宿主工具（harness 别名/任务追踪/提问/提交标记等）", "system", sysTools))
	}
	return groups
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

// builtinJoinedGroups 已加入工作区（固化在 .pair/toolsets/builtin.json）的分组名。
func builtinJoinedGroups(root string) []string {
	ts, err := loadToolset(root, toolsetProject, builtinToolsetName)
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range ts.Plugins {
		if p.Builtin != "" {
			out = append(out, p.Builtin)
		}
	}
	sort.Strings(out)
	return out
}

// BuiltinToolsetInfoOf 派生内置工具集完整信息（列表/详情 API 用）。
func BuiltinToolsetInfoOf(reg *Registry, ph *PluginHost, root string) *BuiltinToolsetInfo {
	groups := BuiltinGroupsOf(reg, ph)
	info := &BuiltinToolsetInfo{
		Name:  builtinToolsetName,
		Scope: "builtin",
		Desc:  "内置工具包（默认不加入工作区；分组开关加入，add_builtin_all 强制全部）",
		Groups: groups,
		Joined: builtinJoinedGroups(root),
	}
	for _, g := range groups {
		info.ToolTotal += len(g.Tools)
		for _, t := range g.Tools {
			if t.Enabled {
				info.EnabledTotal++
			}
		}
	}
	return info
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
	// 读取/创建 builtin 工具集（固化文件）
	ts, err := loadToolset(root, toolsetProject, builtinToolsetName)
	if err != nil {
		ts = &Toolset{Name: builtinToolsetName, Description: "内置工具包（用户/agent 选择加入的内置分组）"}
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
		return "✅ 已强制加入全部内置工具组（" + itoaAgent(len(groups)) + " 组，" + itoaAgent(len(ts.Plugins)) + " 个内置条目）→ 组内工具全部对 agent 可见。固化 .pair/toolsets/builtin.json", nil
	}
	return "✅ 内置工具组 " + groupName + " 已加入工作区（组内工具全部对 agent 可见）。固化 .pair/toolsets/builtin.json", nil
}

// removeBuiltinGroupFromToolset 把内置分组移出工作区 builtin 工具集（恢复默认过滤状态）。
func removeBuiltinGroupFromToolset(ph *PluginHost, root string, groupName string) (string, error) {
	ts, err := loadToolset(root, toolsetProject, builtinToolsetName)
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
		// 全移除：删除固化文件（空工具集不落盘）
		_ = os.Remove(toolsetPath(root, toolsetProject, builtinToolsetName))
		return "✅ 内置工具组 " + groupName + " 已移出工作区（组内工具恢复默认过滤状态）", nil
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	return "✅ 内置工具组 " + groupName + " 已移出工作区（组内工具恢复默认过滤状态）。固化 .pair/toolsets/builtin.json", nil
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
	ts, err := loadToolset(root, toolsetProject, builtinToolsetName)
	if err != nil {
		ts = &Toolset{Name: builtinToolsetName, Description: "内置工具包（用户/agent 选择加入的内置分组与手动工具）"}
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
		return "✅ 工具 " + toolName + " 已加入 agent 可用（手动工具）。固化 .pair/toolsets/builtin.json", nil
	}

	// 禁用：恢复默认过滤 + 持久化差集
	if reg != nil {
		reg.SetToolEnabled(toolName, ToolDefaultEnabled(toolName))
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
		_ = os.Remove(toolsetPath(root, toolsetProject, builtinToolsetName))
		return "✅ 工具 " + toolName + " 已从 agent 可用移除（恢复默认过滤状态）", nil
	}
	if err := saveToolset(root, toolsetProject, ts); err != nil {
		return "", err
	}
	return "✅ 工具 " + toolName + " 已从 agent 可用移除（恢复默认过滤状态）。固化 .pair/toolsets/builtin.json", nil
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

// builtinToolsetJSONPath builtin.json 固化路径（供前端展示/清理）。
func builtinToolsetJSONPath(root string) string {
	return filepath.Join(root, ".pair", "toolsets", builtinToolsetName+".json")
}
