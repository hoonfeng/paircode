// preset_toolsets.go — 预置工具集模式（全局通用集合的种子）。
//
// ★ 2026-09-04 工具集模式改造：工具集从「工作区级」改为「全局通用集合」——
// 跨工作区共享、由对话面板按会话选择。本文件提供开箱即用的多个模式
// （计划讨论/全栈开发/办公/调试/基础/全功能），针对不同类型任务预组合好插件：
//
//	<InstallDir>/.pair/toolsets/<name>.json   全局通用集合（唯一存储位置）
//
// ★ 2026-09-06 v3 中文化：预置模式名从英文（planning/fullstack/office/debug/
// default/full）改为中文（计划讨论/全栈开发/办公/调试/基础/全功能）——
// 展示与文件层统一中文名；旧英文名经 presetNameAliases 兼容解析（会话/配置
// 里存的旧名仍指向新模式），迁移时旧文件 rename + 改写 Name（保留用户改动）。
//
// 播种策略（seedPresetToolsets）：
//  1. 首次启动（无 .preset-seeded 标记）时补全全部预置模式；
//     已有同名工具集（如旧工作区迁移来的 default.json）跳过不覆盖；
//  2. 补全后写 .preset-seeded 标记（内容=预设分类版本号）——此后用户删除/
//     新增集合不再被种子复活（用户管理结果优先，预置只是开箱体验）；
//  3. 版本升级（presetSeedVersion 递增）时执行对应阶梯迁移：只动预设名，
//     用户自定义工具集不受影响。
//
// 模式 = 磁盘插件条目（Name 引用，无 Code——磁盘包 <InstallDir>/.pair/plugins/
// tool-* 是装载载体，条目为 name-only 白名单声明）+ 内置组条目（builtin:xxx，
// Tools 快照由 builtinGroupEntries 派生）。与现有 .pair/toolsets/default.json
// 的条目形态一致（磁盘插件名下：tool-harness 只有 name+purpose）。

package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ── 预置模式中文化名（v3）──
// 展示与文件层统一用中文名；代码内引用一律用常量（勿写裸字符串）。
const (
	presetNamePlanning  = "计划讨论"
	presetNameFullstack = "全栈开发"
	presetNameOffice    = "办公"
	presetNameDebug     = "调试"
	presetNameDefault   = "基础"
	presetNameFull      = "全功能"
)

// presetNameAliases 旧英文名（v1/v2）→ 中文名（v3）。
// ★ 兼容解析：会话里存的旧名、外部引用旧名的调用在 loadToolset 入口统一解析。
var presetNameAliases = map[string]string{
	"planning":  presetNamePlanning,
	"fullstack": presetNameFullstack,
	"office":    presetNameOffice,
	"debug":     presetNameDebug,
	"default":   presetNameDefault,
	"full":      presetNameFull,
}

// resolvePresetName 工具集名解析：预设旧英文名 → 中文名；其余原样返回。
func resolvePresetName(name string) string {
	if n, ok := presetNameAliases[name]; ok {
		return n
	}
	return name
}

// isPresetName 是否预置模式名（中文名白名单——validToolsetName 对预设名放行，
// 使 toolset_edit 等可编辑预置模式；新建仍要求 ASCII 名）。
func isPresetName(name string) bool {
	for _, m := range presetModes {
		if m.Name == name {
			return true
		}
	}
	return false
}

// presetMode 预置模式定义。
type presetMode struct {
	Name     string   // 模式名（工具集名，中文）
	Desc     string   // 描述
	Plugins  []string // 磁盘插件名（Name 引用；空=全部 tool-* 磁盘插件，全功能用）
}

// presetModes 预置模式全表（按展示顺序）。
// 每个模式统一追加内框架工具组条目（system/plugin-mgmt/toolset-mgmt），
// 见 seedPresetToolsets 的 builtinGroupEntries 组装。
// ★ 2026-09-06 分类整理：按任务场景重设计预设（计划讨论/全栈开发/办公/调试 +
// 基础/全功能兜底），删除旧的 dev/test/docs（分别被 全栈开发/办公 取代）；
// v3 起模式名中文化（原 planning/fullstack/office/debug/default/full）。
// 顺序即前端展示顺序（计划讨论 → 全栈开发 → 办公 → 调试 → 基础 → 全功能）。
var presetModes = []presetMode{
	{
		Name: presetNamePlanning, Desc: "计划讨论模式——需求讨论/方案规划/资料检索/记忆与知识库（不含写代码工具）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-memory", "tool-project-info",
			"tool-web", "tool-workflow", "tool-snapshot", "tool-vision",
		},
	},
	{
		Name: presetNameFullstack, Desc: "全栈开发模式——版本控制/代码图谱/缺陷修复/调试/网页/系统/工作流（日常开发）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-web", "tool-vision", "tool-snapshot",
			"tool-project-info", "tool-memory", "tool-git", "tool-codegraph", "tool-bug",
			"tool-debug", "tool-binary", "tool-resource", "tool-entryconfig",
			"tool-workflow", "tool-system",
			"tool-asset",
		},
	},
	{
		Name: presetNameOffice, Desc: "办公模式——文档/表格/知识库/网页/快照（文档与资料整理）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-office", "tool-memory",
			"tool-project-info", "tool-web", "tool-snapshot", "tool-vision",
		},
	},
	{
		Name: presetNameDebug, Desc: "调试排错模式——调试/缺陷/二进制/代码图谱/截图/网页验证（排查问题）",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-debug", "tool-bug",
			"tool-binary", "tool-codegraph",
			"tool-snapshot", "tool-vision", "tool-web",
		},
	},
	{
		Name: presetNameDefault, Desc: "基础工具集——极简核心 + 框架本身提供的工具；插件工具按需加入",
		Plugins: []string{
			"tool-harness", "tool-core", "tool-web", "tool-vision", "tool-snapshot", "tool-project-info", "tool-memory",
		},
	},
	{
		Name: presetNameFull, Desc: "全功能工具集——全部磁盘插件与内置工具组全部对 agent 可见（最全）",
		// Plugins 为空 = 全部 tool-* 磁盘插件（动态扫描）
	},
}

// presetSeedMarker 预置模式已播种标记（全局工具集目录下）。
// ★ 标记内容 = 预置分类版本号（presetSeedVersion）；版本升级时执行阶梯迁移并重播种。
const presetSeedMarker = ".preset-seeded"

// presetSeedVersion 预置分类版本：预设名单/命名变化时递增，触发 seedPresetToolsets
// 对应阶梯迁移（仅限预设名单，用户自定义工具集不动）并重播种。
// v2：dev/test/docs → 全栈开发/办公；v3：英文名 → 中文名。
const presetSeedVersion = 3

// presetLegacyNames v1 起不再保留的旧预置模式（dev/test/docs 被全栈开发/办公取代；
// v2 已清理；保留定义供版本阶梯迁移 v1→v2 使用）。
var presetLegacyNames = []string{"dev", "test", "docs"}

// seedPresetToolsets 补全预置模式（幂等；首次启动补全后写标记，此后不再补——
// 用户删除的集合不复活）。已有同名工具集跳过（不覆盖用户内容）。
func seedPresetToolsets(ph *PluginHost) {
	dir := globalToolsetDir()
	marker := filepath.Join(dir, presetSeedMarker)
	// ★ 版本迁移：读取已播种版本；达到当前版本则跳过（用户管理结果优先）。
	curVer := 0
	if data, err := os.ReadFile(marker); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &curVer)
	}
	if curVer >= presetSeedVersion {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("[toolset] 预置模式播种失败（目录创建）: %v", err)
		return
	}
	// ── 版本阶梯迁移（每级幂等；只动预设名，用户自定义工具集不动）──
	if curVer < 2 {
		// v1→v2：清理不再保留的旧预置模式（dev/test/docs）
		for _, n := range presetLegacyNames {
			if err := os.Remove(toolsetPath("", toolsetProject, n)); err == nil {
				log.Printf("[toolset] 清理旧预置模式 %q", n)
			}
		}
	}
	if curVer < 3 {
		// v2→v3：旧英文名 → 中文名（rename + 改写 Name，保留用户改动内容）
		migratePresetNamesV3(dir)
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
	// 写版本标记（无论补全了几个都写——幂等终点）
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d", presetSeedVersion)), 0o644); err != nil {
		log.Printf("[toolset] 预置模式标记写入失败（下次启动会重试播种）: %v", err)
		return
	}
}

// migratePresetNamesV3 旧英文名文件 → 中文名文件迁移（v2→v3）：
// rename + 改写 JSON Name（保留用户对预设内容的改动）；
// 新中文名文件已存在时删除旧英文名文件（用户已有新名版本，旧的是预设残留）。
func migratePresetNamesV3(dir string) {
	log.Printf("[toolset] v3 迁移：预置模式名中文化（旧英文名 → 中文名）")
	for oldName, newName := range presetNameAliases {
		oldPath := filepath.Join(dir, oldName+".json")
		newPath := filepath.Join(dir, newName+".json")
		if _, err := os.Stat(oldPath); err != nil {
			continue // 旧文件不存在（未播种/已清理）
		}
		if _, err := os.Stat(newPath); err == nil {
			// 新名文件已存在（用户自建）：旧预设残留删除，保留用户版本
			if err := os.Remove(oldPath); err == nil {
				log.Printf("[toolset] 旧预置 %q 已被中文名版本覆盖，删除残留", oldName)
			}
			continue
		}
		data, err := os.ReadFile(oldPath)
		if err != nil {
			continue
		}
		var ts Toolset
		if err := json.Unmarshal(data, &ts); err != nil || ts.Name == "" {
			log.Printf("[toolset] v3 迁移跳过 %q（解析失败，删除旧文件）", oldName)
			_ = os.Remove(oldPath)
			continue
		}
		ts.Name = newName
		out, err := json.MarshalIndent(ts, "", "  ")
		if err != nil {
			continue
		}
		if err := os.WriteFile(newPath, out, 0o644); err != nil {
			log.Printf("[toolset] v3 迁移 %q → %q 写入失败: %v", oldName, newName, err)
			continue
		}
		_ = os.Remove(oldPath)
		log.Printf("[toolset] 预置模式改名 %s → %s", oldName, newName)
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
