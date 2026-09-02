package agent

// harness_filter.go — 工具集对齐（可选精简）（可选精简 pair 独有工具）
//
// 背景：自举迭代（用 agent 开发 agent）要求 agent 暴露给 LLM 的工具集与
// 工具集对齐。提供 harness 对齐模式——只保留 harness 工具集
// + 对话协议基础设施，其余 pair 独有工具（codegraph_*/memory_*/project_info_*/
// git_*/debug_*/binary_*/office 等 130+ 个）从注册表禁用（Enabled=false，不删除——
// 前端可见可管理，agent 不可见；内置工具集 builtin 可一键恢复）。
//
// ★ 开关（2026-08-16 反转默认）：默认**全量工具集**（插件面板工具默认全勾、
//   全部对 agent 可见——产品默认形态）；需要 harness 对齐精简时显式
//   `WB_HARNESS=1` 开启；旧开关兼容：`WB_FULL_TOOLS=1` 强制全量（关闭过滤）。
// ★ 幂等：可重复调用（工具开关现由工具集 toolset_edit 管理，无 .pair/tools.json
//   依赖；历史注释中的 LoadAllWorkspaceToolConfigs 顺序约束已随旧机制删除）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// HarnessOnlyTools 判断是否处于 harness 对齐模式。
// ★ 默认关闭（全量工具集——插件面板工具默认全勾，全部对 agent 可见）；
//
//	`WB_HARNESS=1` 显式开启（精简工具集）；
//	旧开关兼容：`WB_FULL_TOOLS=1` 强制全量（关闭过滤）。
func HarnessOnlyTools() bool {
	if os.Getenv("WB_HARNESS") == "1" {
		return true
	}
	if os.Getenv("WB_FULL_TOOLS") == "1" {
		return false
	}
	return false
}

// HarnessAlignedToolNames 保留清单（过滤时仅保留以下工具）：
//
//	① harness 原生工具集：read/write/edit（tool-fs）、glob/grep（tool-fs-search）、
//	   bash（tool-bash，内部后台+120s 超时）、web_search/web_fetch（tool-web）、
//	   run_code（code-mode）
//	   ★ Round4：str_replace_editor 已从工具面移除（编辑覆盖链 read/write/edit 完全）
//	② 对话协议基础设施：update_tasks（任务追踪，前端任务面板依赖）、
//	   ask_user（提问）——
//	   属循环协议而非 pair 独有编码能力，保留以维持 agent 循环契约。
//	③ 插件管理工具集（cordis_*）：插件即工具的登记/装载/停止/回收/查看——
//	   自举链路关键能力（用 agent 开发 agent 时需能动态注册新工具），
//	   与 harness 的 "tools are plugins" 哲学一致，保留。
//	④ 工具集管理（toolset_*）：工具集=插件组合的固化单元（.pair/toolsets/*.json）。
//	   与 cordis_* 同级保留——agent 需能自主构建/查看/导出/管理工具集
//	   （需求：工具集由 agent 自主创建，而非仅前端手动创建）。
var HarnessAlignedToolNames = map[string]bool{
	// harness 原生工具集
	"read": true, "write": true, "edit": true,
	"glob": true, "grep": true,
	"web_search":         true, "web_fetch": true,
	"run_code":            true,
	"bash":                true,
	// 对话协议基础设施
	"update_tasks": true,
	"ask_user":     true,
	// 插件管理（cordis_*）：登记/装载/停止/回收/查看 JS 动态插件
	"cordis_inspect":      true,
	"cordis_define":       true,
	"cordis_run":          true,
	"cordis_stop":         true,
	"cordis_undefine":     true,
	"cordis_service_list": true,
	// 工具集管理（toolset_*）：工具集=插件组合的固化单元，agent 自主构建/查看/导出/管理
	"toolset_build":  true,
	"toolset_list":   true,
	"toolset_show":   true,
	"toolset_export": true,
	"toolset_import": true,
	"toolset_remove": true,
	"toolset_edit":   true,
}

// ApplyHarnessToolFilter 把不在保留清单内的工具（pair 独有工具）设为禁用
// （Enabled=false），返回禁用数量。开关关闭（默认全量 / WB_FULL_TOOLS=1）时不做任何事，返回 0。
//
// ★ 语义（2026-08-16 重构）：从「Unregister 删除」改为「SetToolEnabled(false) 禁用」——
//
//	工具保留在注册表（前端 /api/tools 可见、可管理），但 Definitions() 只导出启用工具
//	→ agent 不可见；Execute 拦截禁用工具调用。恢复 = SetToolEnabled(true)（可逆）。
//	这为「内置工具集（builtin）」铺路：被过滤工具以内置插件组形态进插件面板，
//	用户选择加入（启用）或强制全部加入，无需重新注册。
//
// ★ exempt 回调：返回 true 的工具豁免过滤（保持启用；插件注册的工具——插件是内容，
//
//	非 pair 独有编码能力；goja 插件工具与 Node 桥工具都经 PluginHost 注册）。
//
// 使用 SetToolEnabled 而非 Unregister，保留钩子等注册表字段。
func ApplyHarnessToolFilter(r *Registry, exempt func(string) bool) int {
	if !HarnessOnlyTools() {
		return 0
	}
	disabled := 0
	for _, m := range r.AllToolMeta() {
		if !HarnessAlignedToolNames[m.Name] && (exempt == nil || !exempt(m.Name)) {
			if m.Enabled { // 只统计「启用→禁用」（幂等：已禁用的不再计数）
				disabled++
			}
			r.SetToolEnabled(m.Name, false)
		}
	}
	return disabled
}

// ToolDefaultEnabled 工具默认启用状态（harness 对齐模式 WB_HARNESS=1：仅保留清单内启用；
// 默认全量模式：全部启用）。
func ToolDefaultEnabled(name string) bool {
	if !HarnessOnlyTools() {
		return true
	}
	return HarnessAlignedToolNames[name]
}

// ApplyToolsetBuiltinState 把全局工具集中「内置工具包条目」（default.json 等）
// 记录的启用状态应用到目标注册表（会话级 reg 独立实例，需显式应用）：
// 对每个 Builtin 条目的 Tools 清单中已注册工具 SetToolEnabled(true)，
// 并应用 DisabledTools（工具级摘除）。
// ★ 2026-09-04 工具集全局化：读取全局通用集合目录（不再按工作区 root）。
// 幂等；无内置条目时不做事。
func ApplyToolsetBuiltinState(r *Registry, root string) {
	_ = root
	if r == nil {
		return
	}
	// 遍历全部全局工具集（含 builtin.json——listToolsets 会跳过它，这里直接列文件）
	dir := globalToolsetDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
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
			if p.Builtin == "" {
				continue
			}
			for _, tn := range p.Tools {
				if tn == "" {
					continue
				}
				if _, ok := r.Get(tn); ok {
					r.SetToolEnabled(tn, true)
				}
			}
			for _, tn := range p.DisabledTools {
				if tn == "" {
					continue
				}
				if _, ok := r.Get(tn); ok {
					r.SetToolEnabled(tn, false)
				}
			}
		}
	}
}
