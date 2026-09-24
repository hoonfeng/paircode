//go:build toolsgen

// ═══════════════════════════════════════════════════════════════
// tool_plugin_gen.go — 磁盘工具插件生成器（build tag: toolsgen）
//
// 用法（工作区根）：
//   go run -tags toolsgen ./dev/tool_plugin_gen
//
// 作用：遍历尚未外置的复杂内置工具组（git/memory/verify/task/
// project-info/binary（含逆向，2026-08-16 并入 binary-re）/debug/
// screenshot/web-debug/bug/office/lsp/codegraph/codegraph-extra），把每组
// 注册的工具定义（name/description/usageGuide/category/readOnly/
// requiresApproval/parameters）完整导出为 .pair/plugins/tool-<组>/index.js
// 磁盘插件。
//
// 生成产物与 tool-core 等手工迁移插件同构：api 声明在插件、execute
// 调 ctx.hostTool.exec 复用宿主 Go 执行器（seam 编排在插件/能力在宿主）。
// Go 侧工具描述变更后重跑生成器即可同步（幂等，仅重写有差异的文件）。
// SystemTool 内部工具不迁移（不暴露给 LLM，无外置意义）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// genToolGroup 一个待生成的工具组。
type genToolGroup struct {
	plugin string                         // 磁盘插件目录名（tool-<组>）
	desc   string                         // 插件用途（purpose/头注释）
	apply  func(r *Registry, root string) // 组注册函数
	names  []string                       // 可选白名单：只挑选这些工具（nil=全部非 SystemTool 工具）
	// binary 指定 execute 调度形态（2026-08-16 第三轮：全部独立二进制）：
	//   ""     → ctx.hostTool.exec（宿主内执行器，会话/宿主状态依赖——tool-system）
	//   "self" → ctx.binary.exec（本插件目录 bin/<插件名>.exe，独立二进制；
	//            源码 plugins-src/plugins/tool-<组>/，import agent 复用内置组实现）
	binary string
}

// genToolGroups 待生成工具组全表（5 组：tool-binary/tool-bug/tool-office/tool-codegraph/
// tool-resource；★ 2026-09 工具面合并：tool-memory/tool-project-info/tool-system
// 已 JS 原生/手改（单工具 op 分派 + 双执行器路由），不在此列（重跑生成会覆盖手改版）；
// core/fs-search/web/shell 已手工迁移为 tool-core/tool-search/tool-web/tool-shell，
// 不在此列；★ 2026-09-04 合并：tool-verify→tool-resource、
// tool-codegraph-extra→tool-codegraph、tool-evolution→tool-asset、
// tool-progress→tool-system，磁盘插件目录已合并/删除）。tool-system 内的
// SystemTool 工具（update_tasks/tool_stats/history_*）对 LLM 可见但前端 UI 隐藏，
// 同样外置可更换；ask_user 已插件化（2026-08-16 会话桥机制，
// 见 session_bridge.go：JS 包装经 _convID 路由回宿主 SessionBridge，
// 非「不可外置」）；task_create 工具面 2026-09-12 已移除（收敛 update_tasks，
// 会话桥路由能力保留）。
// ★ 2026-09 Round3 ③.4 插件瘦身合并：tool-screenshot/tool-web-debug 已并入
//
//	tool-web（binary 型，execute 经 ctx.binary.exec → 内嵌内核
//	registerScreenshotTools/registerWebDebugTool 回退）；tool-bridge（桌面桥接
//	5 工具）已删除（桌面版已移除，见 desktop-architecture）。
func genToolGroups() []genToolGroup {
	return []genToolGroup{
		// ★ 2026-09-12 codex 精简轮：tool-git 已移除（git 操作走 exec_command，codex 无 git 工具面）——
		//   registerGitTools 内核实现保留（builtinPluginSpecs "git" 组，需恢复时重生成插件）。
		// ★ 2026-09 工具面合并：tool-memory 已 JS 原生化为单工具 memory(op=write/read/search/list/delete)，
		//   完整实现在插件内（.pair/plugins/tool-memory/index.js）——从生成器移除：重跑会把插件
		//   覆盖回 binary 形态的 5 工具声明（memory_write/read/list/search/delete），丢合并。
		//   Go 内核 registerMemoryTools 实现保留（hostTool 执行器存档，供插件路由/恢复）。
		// ★ 2026-09-04 合并：tool-verify 已并入 tool-resource（JS 原生迁移，见 tool_plugin_gen.go
		//   顶部工具清单注释与 .pair/plugins/tool-resource/index.js；registerVerifyTools 实现保留：
		//   embedded 内嵌内核 + legacy_host_tools.go 宿主存档供 hostTool 承载）。
		// ★ 2026-09 工具面合并：tool-project-info 已 JS 原生化为单工具 project_info(op=write/read/list/tree/search/delete/explore)，
		//   完整实现在插件内——从生成器移除（重跑会覆盖回 7 工具声明，丢合并）；
		//   registerProjectInfoTools 内核实现保留（hostTool 执行器存档）。
		{"tool-binary", "二进制读写 + 逆向分析（inspect_binary/write_binary/binary_strings/find/patch/info/hash/entropy，含 2026-08-16 并入的 tool-binary-re 逆向 6 工具）", registerBinaryTools, nil, "self"},
		// ★ 2026-09 Round4.5：tool-debug 已移除——纯命令行包装壳（api 声明 + ctx.binary.exec
		//   直通内嵌内核），无组合编排逻辑，浪费上下文。registerDebugTools 内核实现保留
		//   （builtinPluginSpecs/独立二进制复用），需恢复时重新生成插件即可。
		// ★ 2026-09 ③.4 已并入 tool-web（磁盘插件删除；内嵌内核保留供 binary 回退）
		{"tool-bug", "BUG 检测与修复（bug_detect/bug_analyze/bug_fix）", RegisterBugTools, nil, "self"},
		{"tool-office", "办公文档（csv_read/csv_write/json_to_table/table_stats/text_report/word_read）", registerOfficeTools, nil, "self"},
		{"tool-codegraph", "代码知识图谱（codegraph_build/search/relations/…）", registerCodeGraphTools, nil, "self"},
		// ★ 2026-09-04 合并：tool-codegraph-extra（图谱扩展 13 工具）已并入 tool-codegraph；
		//   registerExtraCodeGraphTools 实现保留（embedded_tools.go 内嵌内核供
		//   ctx.binary.exec 回退承载 extra 工具）。
		// tool-system：SystemTool 内部工具 + Skills/MCP
		// （ask_user 经会话桥插件化，见 session_bridge.go；task_create 工具面
		//  2026-09-12 已移除；marketplace_search/install 已迁至 marketplace 插件，2026-08-20）
		// ★ 2026-09 工具面合并：tool-system 已大幅手改（单工具 goal(op=create/get/update)、
		//   load_skill 双执行器路由、追加 ask_user/progress_checker 声明）——生成模板
		//   （白名单 12 项 + hostTool.exec）无法表达这些声明，重跑会丢掉它们（回退）。
		//   故从生成器移除；插件声明见 .pair/plugins/tool-system/index.js（15 工具），
		//   Go 内核 RegisterManagementTools/registerToolStatsTool/registerTaskTools 实现保留。
		// ★ 2026-09 第二轮外置（t1 报告 T1 缺口闭环）：7 组「孤儿工具」注册函数
		//   （有实现、零调用点、Agent 永不可用）迁移为磁盘插件。Go 实现经
		//   ArchiveHostLegacyTools 存档为宿主能力（hostExecutors），插件 execute
		//   走 ctx.hostTool.exec 复用——对齐 harness seam：编排在插件、能力在宿主。
		//   ★ 2026-09 ③.4：tool-bridge（桌面桥接）已删除——桌面版已移除
		//   （desktop-architecture：web-only 运行时），bridge_* 工具零消费方。
		// ★ 2026-09-12 codex 精简轮：tool-asset 已并入 tool-resource（下方 resource 条目合并注册）；
		//   tool-entryconfig 已移除（能力被 glob/codegraph 覆盖，codex 无此类工具）。
		// ★ 2026-09-04 合并：tool-evolution（进化 3 工具）已并入 tool-asset（同名资产存储）；
		//   tool-progress（progress_checker）已并入 tool-system。registerEvolutionTools /
		//   registerProgressChecker 实现保留（legacy_host_tools.go 宿主存档供 hostTool 承载）。
		// ⚠️ 本组重跑生成器会整文件重写（verify_* 从「JS 原生实现」回退为 hostTool.exec 形态）——
		//   如无必要不要重跑；如需同步 resource_* 描述，改 Go 后手工同步 JS。
		{"tool-resource", "资源管理与资产/进化（resource_list/search/stats + memory_verify/project_info_verify + asset_delete/evolution_save_capsule/save_gene/status；tool-verify 2026-09-04、tool-asset 2026-09-12 并入）",
			func(r *Registry, root string) { registerResourceTools(r, root); registerAssetTools(r, root) }, nil, ""},
		// ★ 2026-09-21：文件快照能力整体移除——snapshot.go / rollback.go 删除，
		//   restore_snapshot / list_snapshots 声明与「回退到消息」功能一并下线
		//   （tool-snapshot 曾于 2026-09-12 并入 tool-harness，声明已同步清理）。
	}
}

// genToolDef JS 侧工具定义（对齐 jsToolToGo 的 json 字段，omitempty 保持整洁）。
type genToolDef struct {
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	UsageGuide       string         `json:"usageGuide,omitempty"`
	Category         string         `json:"category,omitempty"`
	Parameters       map[string]any `json:"parameters"`
	ReadOnly         bool           `json:"readOnly,omitempty"`
	RequiresApproval bool           `json:"requiresApproval,omitempty"`
	SystemTool       bool           `json:"systemTool,omitempty"`
}

// collectGroupTools 对一组调用注册函数，收集可外置的工具定义（排除系统内部工具）。
func collectGroupTools(g genToolGroup, root string) ([]genToolDef, error) {
	r := NewRegistry()
	g.apply(r, root)
	want := map[string]bool{}
	for _, n := range g.names {
		want[n] = true
	}
	defs := make([]genToolDef, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		if t == nil {
			continue
		}
		if len(want) > 0 {
			if !want[name] {
				continue // 白名单模式：跳过名单外工具
			}
		} else if t.SystemTool {
			continue // 非白名单模式：跳过系统内部工具（不暴露给 LLM，不迁移）
		}
		defs = append(defs, genToolDef{
			Name:             t.Name,
			Description:      t.Description,
			UsageGuide:       t.UsageGuide,
			Category:         t.Category,
			Parameters:       t.Parameters,
			ReadOnly:         t.ReadOnly,
			RequiresApproval: t.RequiresApproval,
			SystemTool:       t.SystemTool,
		})
	}
	if len(defs) == 0 {
		return nil, fmt.Errorf("组 %s 无可外置工具（全部 SystemTool 或未注册）", g.plugin)
	}
	return defs, nil
}

// buildPluginJS 组装磁盘插件源码（对齐 tool-core 模板：api 声明 + hostTool 编排）。
func buildPluginJS(g genToolGroup, defs []genToolDef) (string, error) {
	raw, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化 %s 工具定义: %w", g.plugin, err)
	}
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// ═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(&b, "// %s — %s\n//\n", g.plugin, g.desc)
	fmt.Fprintf(&b, "// 生成来源（2026-08-16）：内置 Go 工具组 → 磁盘外置插件（tool_plugin_gen.go\n")
	execLine := "execute: (args) => ctx.hostTool.exec(t.name, args || {}),"
	execDoc := "execute 调 ctx.hostTool 复用宿主 Go 执行器（对齐 harness seam：编排在插件、能力在宿主）。"
	switch g.binary {
	case "self":
		execLine = "execute: (args) => ctx.binary.exec(t.name, args || {}),"
		execDoc = "execute 调 ctx.binary 复用本插件目录 bin/ 下的独立二进制（源码 plugins-src/plugins/<name>/，改实现重编译即更换）。"
	case "tool-binary":
		execLine = "execute: (args) => ctx.binary.exec(t.name, args || {}, {bin: 'tool-binary'}),"
		execDoc = "execute 调 ctx.binary 复用统一宿主二进制（.pair/plugins/tool-binary/bin/，源码 plugins-src/plugins/tool-binary/，承载全部内置工具组实现）。"
	}
	fmt.Fprintf(&b, "// 自动生成，schema 完整外置拷贝）。api 声明在插件，%s\n", execDoc)
	fmt.Fprintf(&b, "// 工具清单：%s\n", strings.Join(names, "、"))
	fmt.Fprintf(&b, "// ═══════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(&b, "const tools = %s;\n\n", raw)
	fmt.Fprintf(&b, "return {\n")
	fmt.Fprintf(&b, "  name: '%s',\n", g.plugin)
	fmt.Fprintf(&b, "  purpose: '%s（自动生成，迁移自内置 Go 工具组）',\n", g.desc)
	fmt.Fprintf(&b, "  apply(ctx) {\n")
	fmt.Fprintf(&b, "    for (const t of tools) {\n")
	fmt.Fprintf(&b, "      ctx.tools.register({\n")
	fmt.Fprintf(&b, "        name: t.name,\n")
	fmt.Fprintf(&b, "        description: t.description,\n")
	fmt.Fprintf(&b, "        usageGuide: t.usageGuide,\n")
	fmt.Fprintf(&b, "        category: t.category,\n")
	fmt.Fprintf(&b, "        readOnly: t.readOnly,\n")
	fmt.Fprintf(&b, "        requiresApproval: t.requiresApproval,\n")
	fmt.Fprintf(&b, "        systemTool: t.systemTool,\n")
	fmt.Fprintf(&b, "        parameters: t.parameters,\n")
	fmt.Fprintf(&b, "        %s\n", execLine)
	fmt.Fprintf(&b, "      })\n")
	fmt.Fprintf(&b, "    }\n")
	fmt.Fprintf(&b, "  },\n")
	fmt.Fprintf(&b, "}\n")
	return b.String(), nil
}

// GenerateToolPlugins 生成全部未外置工具组的磁盘插件（幂等，仅重写有差异文件）。
// outDir 为 .pair/plugins 目录；每个插件包写 index.js + package.json
// （LoadGlobalPlugins 按 package.json 扫描装载）。返回写入的文件列表。
func GenerateToolPlugins(root, outDir string) ([]string, error) {
	var written []string
	for _, g := range genToolGroups() {
		defs, err := collectGroupTools(g, root)
		if err != nil {
			return written, err
		}
		js, err := buildPluginJS(g, defs)
		if err != nil {
			return written, err
		}
		dir := filepath.Join(outDir, g.plugin)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return written, fmt.Errorf("创建 %s: %w", dir, err)
		}
		// ① index.js（host 半源码）
		f := filepath.Join(dir, "index.js")
		if err := os.WriteFile(f, []byte(js), 0o644); err != nil {
			return written, fmt.Errorf("写入 %s: %w", f, err)
		}
		written = append(written, f)
		// ② package.json（装载清单：LoadGlobalPlugins 按此扫描）
		pkg := map[string]any{
			"name":    g.plugin,
			"purpose": g.desc + "（自动生成，迁移自内置 Go 工具组）",
			"version": "1.0.0",
			"scope":   "global",
			"type":    "plugin",
			"main":    "index.js",
		}
		pkgJSON, err := json.MarshalIndent(pkg, "", "  ")
		if err != nil {
			return written, fmt.Errorf("序列化 %s package.json: %w", g.plugin, err)
		}
		pf := filepath.Join(dir, "package.json")
		if err := os.WriteFile(pf, append(pkgJSON, '\n'), 0o644); err != nil {
			return written, fmt.Errorf("写入 %s: %w", pf, err)
		}
		written = append(written, pf)
	}
	return written, nil
}
