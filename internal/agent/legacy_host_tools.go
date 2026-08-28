// legacy_host_tools.go — 孤儿工具组「宿主能力存档」（t1 报告 T1 缺口闭环）
//
// ★ 背景（2026-09）：registerAssetTools / registerBridgeTools /
//   registerEntryConfigTools / registerEvolutionTools / registerProgressChecker /
//   registerResourceTools / RegisterSnapshotTools 这 7 组约 20 个工具
//   有完整 Go 实现但零调用点——既未插件化也未装配，Agent 永远用不到。
//
// ★ 处置（对齐 harness seam「能力在宿主、编排在插件」）：
//   1. 磁盘插件（.pair/plugins/tool-*，由 tool_plugin_gen.go 生成）声明工具
//      api（name/description/parameters），execute 调 ctx.hostTool.exec 复用
//      宿主 Go 执行器；
//   2. 本文件把这些 Go 实现存档进 hostExecutors（ExecuteHostTool 的索引）——
//      与 tool-system 的宿主框架工具存档（claimTool → ArchiveHostTool）同构，
//      但**不注册进任何 Registry**：agent 可见面完全由插件决定（插件停用
//      即工具消失，不会出现「宿主兜底孤儿工具」）。
//   3. NewPluginHost 构造时调用（宿主能力库随宿主生灭；root 取插件宿主根，
//      与 claimTool 存档宿主框架工具的同根语义一致，幂等覆盖）。

package agent

import "sync"

// legacyToolGroups 孤儿工具组全表（注册函数 → 组说明）。
// 顺序即存档顺序；与 tool_plugin_gen.go 的 genToolGroups 新增组一一对应。
var legacyToolGroups = []struct {
	register func(r *Registry, root string)
	desc     string
}{
	{registerAssetTools, "智能资产管理（asset_list/asset_search/asset_delete）"},
	{registerBridgeTools, "桌面桥接（bridge_status/takeover/release/exec/register_system_tool）"},
	{registerEntryConfigTools, "入口与配置定位（find_entry_points/find_config_files）"},
	{registerEvolutionTools, "进化系统（evolution_save_capsule/search_capsules/save_gene/status）"},
	{registerProgressChecker, "进度检查（progress_checker）"},
	{registerResourceTools, "资源管理（resource_list/search/stats）"},
	{RegisterSnapshotTools, "会话快照（restore_snapshot/list_snapshots）"},
}

var (
	legacyToolsMu   sync.Mutex
	legacyToolsRoot string // 最近一次存档的 root（root 变更时重存，幂等去重）
)

// ArchiveHostLegacyTools 把 7 组孤儿工具的 Go 实现存档为宿主能力
// （hostExecutors，供磁盘插件 ctx.hostTool.exec 调用）。
// 幂等：root 未变化时不做重复存档（map 覆盖，~20 个工具，成本可忽略）。
// 不注册进任何 Registry——agent 可见面由插件决定。
func ArchiveHostLegacyTools(root string) {
	legacyToolsMu.Lock()
	defer legacyToolsMu.Unlock()
	if root == legacyToolsRoot {
		return // 同根已存档（多会话同工作区不重复开销）
	}
	tmp := NewRegistry()
	for _, g := range legacyToolGroups {
		g.register(tmp, root)
	}
	for _, meta := range tmp.AllToolMeta() {
		if t, ok := tmp.Get(meta.Name); ok {
			ArchiveHostTool(t)
		}
	}
	legacyToolsRoot = root
}
