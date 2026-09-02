// preset_toolsets.go — 预置工具集模式（全局通用集合的种子）。
//
// ★ 2026-09-04 工具集模式改造：工具集从「工作区级」改为「全局通用集合」——
// 跨工作区共享、由对话面板按会话选择。本文件提供开箱即用的多个模式
// （default/full/dev/debug/test/docs），针对不同类型任务预组合好插件：
//
//	<InstallDir>/.pair/toolsets/<name>.json   全局通用集合（唯一存储位置）
//
// 播种策略（seedPresetToolsets）：
//  1. 首次启动（无 .preset-seeded 标记）时补全全部预置模式；
//     已有同名工具集（如旧工作区迁移来的 default.json）跳过不覆盖；
//  2. 补全后写 .preset-seeded 标记——此后用户删除/新增集合不再被种子复活
//     （用户管理结果优先，预置只是开箱体验）。
//
// 模式 = 磁盘插件条目（Name 引用，无 Code——磁盘包 <InstallDir>/.pair/plugins/
// tool-* 是装载载体，条目为 name-only 白名单声明）+ 内置组条目（builtin:xxx，
// Tools 快照由 builtinGroupEntries 派生）。与现有 .pair/toolsets/default.json
// 的条目形态一致（磁盘插件名下：tool-harness 只有 name+purpose）。

package agent

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// presetMode 预置模式定义。
type presetMode struct {
	Name     string   // 模式名（工具集名）
	Desc     string   // 描述
	Plugins  []string // 磁盘插件名（Name 引用；空=全部 tool-* 磁盘插件，full 用）
}

// presetModes 预置模式全表（按展示顺序）。
// 每个模式统一追加内框架工具组条目（system/plugin-mgmt/toolset-mgmt），
// 见 seedPresetToolsets 的 builtinGroupEntries 组装。
var presetModes = []presetMode{
	{
		Name: "default", Desc: "基础工具集——极简核心 + 框架本身提供的工具；插件工具按需加入",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-web", "tool-vision", "tool-snapshot", "tool-project-info", "tool-memory",
		},
	},
	{
		Name: "full", Desc: "全功能工具集——全部磁盘插件与内置工具组全部对 agent 可见（最全）",
		// Plugins 为空 = 全部 tool-* 磁盘插件（动态扫描）
	},
	{
		Name: "dev", Desc: "开发模式——版本控制/代码图谱/缺陷修复/调试/验证（日常开发任务）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-web", "tool-vision", "tool-snapshot",
			"tool-project-info", "tool-memory", "tool-git", "tool-codegraph", "tool-bug",
			"tool-debug", "tool-binary", "tool-resource", "tool-entryconfig",
			"tool-workflow", "tool-system",
			"tool-asset",
		},
	},
	{
		Name: "debug", Desc: "调试排错模式——调试/缺陷/二进制/代码图谱/截图/网页验证（排查问题）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-debug", "tool-bug",
			"tool-binary", "tool-codegraph",
			"tool-snapshot", "tool-vision", "tool-web",
		},
	},
	{
		Name: "test", Desc: "测试验证模式——验证/缺陷/版本控制/办公数据/快照（验证与回归）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-resource", "tool-bug",
			"tool-git", "tool-snapshot", "tool-office", "tool-vision", "tool-web",
		},
	},
	{
		Name: "docs", Desc: "文档办公模式——办公文档/记忆/项目知识库/网页（文档与资料整理）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-office", "tool-memory",
			"tool-project-info", "tool-web", "tool-snapshot", "tool-vision",
		},
	},
}

// presetSeedMarker 预置模式已播种标记（全局工具集目录下）。
const presetSeedMarker = ".preset-seeded"

// seedPresetToolsets 补全预置模式（幂等；首次启动补全后写标记，此后不再补——
// 用户删除的集合不复活）。已有同名工具集跳过（不覆盖用户内容）。
func seedPresetToolsets(ph *PluginHost) {
	dir := globalToolsetDir()
	marker := filepath.Join(dir, presetSeedMarker)
	if _, err := os.Stat(marker); err == nil {
		return // 已播种过：用户管理结果优先
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("[toolset] 预置模式播种失败（目录创建）: %v", err)
		return
	}
	reg := (*Registry)(nil)
	if ph != nil && ph.Context() != nil {
		reg = ph.Context().Tools
	}
	// full 模式 = 全部 tool-* 磁盘插件（动态扫描；排除无工具条目的 UI/系统包）
	allToolPlugins := presetAllToolPlugins()
	for _, m := range presetModes {
		if _, err := os.Stat(toolsetPath("", toolsetProject, m.Name)); err == nil {
			continue // 已有同名工具集（用户/迁移内容）：不覆盖
		}
		ts := &Toolset{
			Name:        m.Name,
			Description: m.Desc,
			Version:     "1.0.0",
			// 内置组条目已含在 Plugins
			BuiltinsInited: true,
		}
		plugins := m.Plugins
		if len(plugins) == 0 {
			plugins = allToolPlugins
		}
		// 磁盘插件条目（name-only 白名单声明；purpose 从磁盘包读取，失败用插件名）
		for _, pn := range plugins {
			ts.Plugins = append(ts.Plugins, ToolsetPlugin{
				Name:    pn,
				Purpose: diskPluginPurpose(pn),
			})
		}
		// 内置组条目（Tools 快照；与 defaultProjectToolset/builtinGroupEntries 同源——
		//  只含框架宿主工具：system（harness 别名 + SystemTool + tool-system 插件工具）/
		//  plugin-mgmt（cordis_*）/ toolset-mgmt（toolset_*）。业务磁盘插件工具
		//  经清单顶部磁盘插件条目声明（name-only），不会混入 system 组。
		ts.Plugins = append(ts.Plugins, builtinGroupEntries(reg, ph)...)
		if err := saveToolset("", toolsetProject, ts); err != nil {
			log.Printf("[toolset] 预置模式 %s 写入失败: %v", m.Name, err)
			continue
		}
		log.Printf("[toolset] 预置模式 %q 已生成（%d 个条目）", m.Name, len(ts.Plugins))
	}
	// 播种标记（无论补全了几个都写——幂等终点）
	if err := os.WriteFile(marker, []byte("1"), 0o644); err != nil {
		log.Printf("[toolset] 预置模式标记写入失败（下次启动会重试播种）: %v", err)
		return
	}
}

// presetAllToolPlugins 扫描全局插件目录，返回全部 tool-* 磁盘插件名（排序）。
// 判定与 LoadGlobalPlugins 一致：目录 + package.json 有效 + main 非空。
func presetAllToolPlugins() []string {
	entries, err := os.ReadDir(globalPluginsDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !strings.HasPrefix(name, "tool-") || !diskPluginCodeAvailable(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// diskPluginPurpose 读磁盘插件包 purpose（失败返回插件名）。
func diskPluginPurpose(name string) string {
	data, err := os.ReadFile(filepath.Join(globalPluginsDir(), name, "package.json"))
	if err != nil {
		return name
	}
	var pkg GlobalPluginPackage
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Purpose == "" {
		return name
	}
	return pkg.Purpose
}
