package agent

// tool_landing_test.go — 磁盘工具插件「落地」矩阵验证（2026-08-27）。
//
// 背景：工具已全部迁移为磁盘插件（.pair/plugins/tool-*），execute 分三类：
//   1. ctx.hostTool.exec（tool-system）→ 宿主 Go 存档（RegisterHostFrameworkTools
//      + SetSessionBridge 存档）必须存在，否则执行报「宿主执行器不存在」
//   2. ctx.binary.exec（tool-codegraph/office/screenshot 等）→ 插件 bin/*.exe 已移除，
//      回退内嵌内核（embedded_tools.go embeddedToolRegistrars）必须覆盖该工具，
//      否则执行报「插件二进制不存在」
//   3. JS 原生（ctx.fs/ctx.bash/ctx.process/ctx.web）→ 无外部落地点
//
//   ★ 生产装载时序：AgentBase.New = RegisterHostFrameworkTools(registry) →
//     NewPluginHost(registry) → LoadAllToolsets（磁盘插件 claimTool 时存档）
//     → SetGlobalPluginHost；SetSessionBridge 另存档 ask_user/task_create 路由版。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// toolPluginModes 插件执行模式表（静态事实，与插件源码一致）：
//
//	hostTool : 全部工具走 ctx.hostTool.exec（宿主 Go 存档）→ 强断言
//	binary   : 全部工具走 ctx.binary.exec（内嵌内核覆盖）→ 强断言
//	mixed    : 部分原生 + 二进制回退分支（内核覆盖全量=设计事实）→ 强断言
//	             （原生实现工具不在内嵌内核——toolHarnessAliases 声明跳过，
//	             如 harness 的 read/write/edit/… 与 tool-web 的 web_fetch/web_search）
//	native   : 全 JS 原生实现（ctx.fs/bash/process/web）
//
// ★ 修改插件实现模式时必须同步本表（测试是模式漂移守卫）。
// ★ 2026-09 Round3 ③.4 插件瘦身合并：tool-shell（→tool-harness）、
//
//	tool-screenshot/tool-web-debug（→tool-web）、tool-goal（→tool-system）、
//	tool-subagent（→tool-workflow）、tool-bridge（已删除）不再独立存在，
//	模式表同步删除对应行。
var toolPluginModes = map[string]string{
	"tool-system":    "hostTool",
	"tool-codegraph": "binary",
	"tool-binary":    "binary",
	// ★ 2026-09 Round4.5：tool-debug 已移除（纯命令行包装壳、无组合逻辑）
	"tool-office":       "mixed",
	"tool-harness":      "mixed",  // ★ 07-23 合并 tool-shell；2026-09 后台 4 件套移交 tool-exec；Round5 apply_patch 统一编辑面（read/write/apply_patch/glob/grep 仍原生）
	"tool-exec":         "native", // ★ 2026-09 工具重构：exec_command/write_stdin/kill_process（ctx.process 会话式）
	"tool-memory":       "native",
	"tool-project-info": "native",
	"tool-bug":          "native",
	"tool-web":          "mixed", // ★ ③.4 合并 tool-screenshot/tool-web-debug；2026-09-12 并入 tool-vision（read_image JS 原生，别名）——screenshot/web_debug 走 binary 内核
	// ★ 2026-09 t1 T1 闭环：7 组孤儿工具迁移为磁盘插件（tool_plugin_gen.go 生成，
	//   execute 走 ctx.hostTool.exec 复用宿主存档能力——legacy_host_tools.go）
	// ★ 2026-09-12 codex 精简轮：tool-git/tool-entryconfig 已删除；tool-vision→tool-web、
	//   tool-snapshot→tool-harness、tool-asset→tool-resource（仅剩 tool-resource 独立行）
	"tool-resource": "hostTool", // ★ 2026-09-04 tool-verify 并入；2026-09-12 tool-asset（asset_delete/evolution_*）并入
	// ★ Round3 ③/⑤：goal/workflow 宿主机制工具面（execute → ctx.hostTool）；
	//   ★ 2026-09：subagent 系列六工具随子 Agent 实现删除，tool-workflow 仅剩 workflow。
	"tool-workflow": "mixed", // workflow→hostTool 执行器（不在内嵌内核，别名声明）
	// ★ 2026-09-11 场景创造（创造模式）：/创造 按需激活；scenario_scan/scenario_create
	//   execute → ctx.hostTool（宿主 internal/agent/scenario_tools.go）。
	"tool-scenario": "hostTool",
	// ★ 2026-09 L1（人声插件试点）：nodeBridge = Node 桥轨插件（package.json 声明了
	//   运行期 npm 依赖，如 @audio/*）——由 bridge.js 装载，工具只进桥注册表，
	//   不进 goja 轨工具面，故本矩阵不适用（其装载由 node_bridge_*_test.go 覆盖）。
	"tool-voice": "nodeBridge",
	// ★ 2026-09 L2（音乐插件）：native = 全 JS 原生实现。tool-music 的 5 个工具
	//   （project/edit/import/export/verify）全部在 goja 沙箱内用 ctx.fs 完成，
	//   SMF/ABC/MusicXML/SVG 均为自实现（MIDI 二进制走 readFileBase64/writeFileBase64），
	//   不经 ctx.hostTool / ctx.binary。
	"tool-music": "native",
	// ★ 2026-09 L2（矢量画板插件）：native = 全 JS 原生实现。tool-art 的 5 个工具
	//   （project/edit/import/export/verify）全部在 goja 沙箱内用 ctx.fs 完成：
	//   SVG 生成/解析、矩阵变换、包围盒与 WCAG 对比度均为自实现，
	//   不经 ctx.hostTool / ctx.binary（PNG 光栅化不在沙箱内，由面板浏览器侧完成）。
	"tool-art": "native",
	// ★ 2026-09 L2（UI 设计插件）：native = 全 JS 原生实现。tool-design 的 5 个工具
	//   （tokens/project/edit/export/verify）全部在 goja 沙箱内用 ctx.fs 完成：
	//   令牌与界面工程双文本真相源、确定性 flex 布局引擎、HTML/CSS/mermaid 生成、
	//   9 项判据（含 WCAG 对比度、4px 网格、字号阶梯、触达区、图标白名单）均为自实现，
	//   不经 ctx.hostTool / ctx.binary。
	"tool-design": "native",
	// ★ 2026-09 L2（2D 角色插件）：native = 全 JS 原生实现。tool-rig 的 5 个工具
	//   （model/import/edit/export/verify）全走 ctx.fs —— 沙箱无 Buffer，PSD 二进制经
	//   ctx.fs.readFileBase64 + 自研 base64 解码；PSD 结构解析、命名约定自动绑定、
	//   canvas 渲染器（预览 HTML 内联）与 9 项判据均为自研，零 npm 依赖，
	//   不经 ctx.hostTool / ctx.binary。
	"tool-rig": "native",
	// ★ 2026-09 L2（3D/CAD 插件）：native = 全 JS 原生实现。tool-model 的 5 个工具
	//   （doc/add/edit/export/verify）全走 ctx.fs —— 表达式求值器、三角网格内核、
	//   BSP 实体布尔、IEEE754/base64 自研编码（binary STL / GLB 经 ctx.fs.writeFileBase64）、
	//   glTF 2.0 导出（z-up→y-up 转换）、WebGL 预览（预览 HTML 内联渲染器）与 9 项判据
	//   均为自研，零 npm 依赖，不经 ctx.hostTool / ctx.binary。
	"tool-model": "native",
	// ★ 2026-09-26 本机通用插件打包器（tool-packager）：mixed = 打包主流程经
	//   ctx.binary.exec 调用插件自带 exe（bin/tool-packager.exe，stdin/stdout 一行 JSON），
	//   其余（说明生成/回验/zip 组装）走 ctx.fs 原生——exe 为本插件自带、非内嵌内核，
	//   工具名在 toolHarnessAliases 声明跳过内嵌断言。
	"tool-packager": "mixed",
}

// toolHarnessAliases 混合型插件的 JS 原生实现工具（不在内嵌内核，断言跳过）：
// tool-harness：read/write/apply_patch/glob/grep（ctx.fs 原生；run_code 仍由 registerRunCode
//
//	覆盖内核，正常断言）；★ 2026-09 工具重构：后台 4 件套已移交 tool-exec（native）。
//
// tool-web：web_fetch/web_search（ctx.web 原生，③.4 后为 mixed 型）。
// （str_replace_editor 已移除（Round4）；screenshot_*/web_debug 同理内核覆盖。）
var toolHarnessAliases = map[string]bool{
	"read": true, "write": true, "apply_patch": true, "glob": true, "grep": true,
	"web_fetch": true, "web_search": true,
	"pack_plugin": true, // tool-packager：打包 exe 为插件自带（非内嵌内核），断言跳过
	// ★ 2026-09-12 并入声明：tool-vision→tool-web（read_image JS 原生）；
	// screenshot 三合一（调度直通内核原名 screenshot_desktop/window/area）
	// ★ 2026-09-21：文件快照能力移除，list/restore_snapshot 别名声明同步删除。
	"read_image": true,
	"screenshot": true,
	// tool-workflow（2026-09 ③.4 后为 mixed）：workflow→宿主 goja 运行器（hostTool），
	// 非内嵌内核，别名声明跳过。
	// ★ 2026-09 子 Agent 实现删除：subagent / subagent_fork / report /
	//   list_agents / interrupt_agent / send_message 六个工具已不存在，声明同步删除。
	"workflow": true,
}

// toolDispatchAliases 分派型工具 → 内部路由名（工具面合并后的单工具：
// 工具名本身可能无 hostTool 存档，execute 按参数分派到内部路由名执行器）。
// 断言验证内部路由名均已存档（2026-09：goal → create/get/update_goal；
// load_skill → load_skill_resource（path 分支，L3 技能子资源加载））。
var toolDispatchAliases = map[string][]string{
	"goal":       {"create_goal", "get_goal", "update_goal"},
	"load_skill": {"load_skill_resource"},
}

// loadDiskPluginForTestFramed 装载磁盘插件，且宿主注册表预注册框架工具
// （对齐生产时序：claimTool 时框架工具已注册 → ArchiveHostTool 存档生效）。
// 返回 host + 共享 reg（与 loadDiskPluginForTest 同形）。
func loadDiskPluginForTestFramed(t *testing.T, name string) (*PluginHost, *Registry) {
	t.Helper()
	root := jsNativeWorkspace
	// ★ 2026-09-17 双源定位（.pair/plugins 基线 + plugins-dist 独立发布插件）
	dir := diskPluginDir(root, name)
	code, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if err != nil {
		t.Fatalf("读取 %s/index.js: %v", name, err)
	}
	reg := NewRegistry()
	RegisterHostFrameworkTools(reg, root)
	// ★ 2026-09-11 对齐生产时序：场景创造工具（scenario_scan/scenario_create）在
	//   宿主框架面预注册——tool-scenario 装载时 claimTool 存档 hostExecutors
	//   （同 web_server initReg 路径）。
	RegisterScenarioTools(reg, root, nil)
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJSCodeFull(string(code), "", "落地验证", dir, "")
	if err != nil {
		t.Fatalf("DefineJSCodeFull: %v", err)
	}
	if err := host.LoadJSDynamic(mustDef(t, host, id)); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	return host, reg
}

// TestToolLandingMatrix 全部磁盘工具插件落地矩阵。
func TestToolLandingMatrix(t *testing.T) {
	if _, err := os.Stat(filepath.Join(jsNativeWorkspace, ".pair", "plugins", "tool-system", "index.js")); err != nil {
		t.Skipf("磁盘插件不存在（非开发环境）: %v", err)
	}
	root := jsNativeWorkspace
	SetSessionBridge(&SessionBridge{
		WaitAnswer:       func(ctx context.Context, convID string) (string, error) { return "", nil },
		GetWorkspaceRoot: func(convID string) string { return root },
	})
	defer SetSessionBridge(&SessionBridge{})

	// ★ 2026-09：重置内嵌内核单例——前序测试（harness 的 run_code 等）可能已用
	// TempDir root 初始化（单例首次 root 被缓存），不重置会污染本测试的 root 解析。
	embeddedToolRegistry = nil
	embedded := InitEmbeddedToolRegistry(root)
	hostNames := map[string]bool{}
	for _, n := range HostToolNames() {
		hostNames[n] = true
	}

	// ★ 2026-09-17 双源扫描：独立发布插件真源已迁出 .pair/plugins（→ plugins-dist/，
	//   避免随 IDE 发布包出厂）。只扫基线目录会让这些插件的模式表守卫静默失效。
	//   注：本地开发态的 junction 挂载在 lstat 语义下 IsDir()==false，不会重复计入。
	var dirs []os.DirEntry
	for _, base := range []string{filepath.Join(root, ".pair", "plugins"), filepath.Join(root, "plugins-dist")} {
		sub, err := os.ReadDir(base)
		if err != nil {
			continue // 该来源不存在 = 无插件
		}
		dirs = append(dirs, sub...)
	}
	if len(dirs) == 0 {
		t.Fatalf("读取插件目录失败：.pair/plugins 与 plugins-dist 均不可读")
	}
	var problems []string
	total := 0
	byKind := map[string]int{}
	for _, d := range dirs {
		if !d.IsDir() || !strings.HasPrefix(d.Name(), "tool-") {
			continue
		}
		mode, known := toolPluginModes[d.Name()]
		if !known {
			problems = append(problems, "[未知模式表] "+d.Name()+"——需在 toolPluginModes 声明")
			continue
		}
		// ★ 2026-09 L1：Node 桥轨插件不经 goja 装载（LoadGlobalPlugins 已按运行时轨跳过），
		//   其工具在 bridge.js 装载后进桥注册表 —— 本矩阵的「磁盘插件落地」语义不适用。
		if mode == "nodeBridge" {
			continue
		}
		// hostTool 型用框架预注册装载；其余用普通装载（无副作用）
		var reg2 *Registry
		var host2 *PluginHost
		if mode == "hostTool" {
			host2, reg2 = loadDiskPluginForTestFramed(t, d.Name())
		} else {
			_, reg2 = loadDiskPluginForTest(t, d.Name())
		}
		// ★ hostTool 存档发生在装载期间（claimTool / ArchiveHostLegacyTools），
		//   每次装载后刷新宿主执行器清单再断言（自包含，不依赖测试执行顺序）
		for _, n := range HostToolNames() {
			hostNames[n] = true
		}
		// ★ hostTool 断言只覆盖「本插件注册的工具」（PluginToolOwners 归属本插件的
		//   工具）——宿主预注册的框架工具（skill_*/mcp_*/history_* 等）未被本插件
		//   claim 时不存档属正常（t1 T1 新增插件只 claim 自己的业务工具）
		owned := map[string]bool{}
		if host2 != nil {
			for k, owner := range host2.PluginToolOwners() {
				if owner == d.Name() {
					owned[k] = true
				}
			}
		}
		for _, tn := range reg2.Names() {
			total++
			byKind[mode]++
			switch mode {
			case "hostTool":
				if !owned[tn] {
					continue // 宿主预注册框架工具（未被本插件 claim），跳过
				}
				if !hostNames[tn] {
					if _, ok := toolDispatchAliases[tn]; !ok {
						problems = append(problems, "[hostTool 未存档] "+d.Name()+"/"+tn)
					}
				}
				// ★ 2026-09 工具面合并：分派目标不论工具自身是否在档，都须在档
				//   （load_skill → load_skill_resource（path 分支路由）；goal → create/get/update_goal）
				if targets, ok := toolDispatchAliases[tn]; ok {
					for _, tg := range targets {
						if !hostNames[tg] {
							problems = append(problems, "[分派目标未存档] "+d.Name()+"/"+tn+" → "+tg)
						}
					}
				}
			case "binary", "mixed":
				if _, ok := embedded.Get(tn); !ok && !toolHarnessAliases[tn] {
					problems = append(problems, "[内嵌内核缺失] "+d.Name()+"/"+tn)
				}
			}
		}
	}
	t.Logf("落地矩阵: 共 %d 工具 %v", total, byKind)
	for _, p := range problems {
		t.Errorf("未落地: %s", p)
	}
}

// TestToolLandingSpotCheck 代表性工具真实执行（只读/无副作用）：
//   - hostTool 型：skill_list / history_count / tool_stats（框架存档链路）
//   - binary 型：binary_hash（内嵌内核链路）
//   - native 型：project_info(op=list)（ctx.fs 链路；2026-09 单工具合并）
func TestToolLandingSpotCheck(t *testing.T) {
	root := jsNativeWorkspace
	// ★ 2026-09：重置内嵌内核单例（防前序测试 TempDir root 污染；同 Matrix 测试）
	embeddedToolRegistry = nil
	SetSessionBridge(&SessionBridge{
		WaitAnswer:       func(ctx context.Context, convID string) (string, error) { return "", nil },
		GetWorkspaceRoot: func(convID string) string { return root },
	})
	defer SetSessionBridge(&SessionBridge{})

	cases := []struct {
		plugin string
		tool   string
		args   string
	}{
		{"tool-system", "skill_list", `{}`},
		{"tool-system", "history_count", `{}`},
		{"tool-system", "tool_stats", `{}`},
		{"tool-binary", "binary_hash", `{"path":"go.mod"}`},
		{"tool-project-info", "project_info", `{"op":"list"}`},
	}
	for _, c := range cases {
		var reg *Registry
		if c.plugin == "tool-system" {
			_, reg = loadDiskPluginForTestFramed(t, c.plugin)
		} else {
			_, reg = loadDiskPluginForTest(t, c.plugin)
		}
		// 磁盘插件工具默认未入工具集（disabled）——落地验证启用后执行
		reg.SetToolEnabled(c.tool, true)
		out, err := reg.Execute(context.Background(), c.tool, c.args)
		if err != nil {
			t.Errorf("%s/%s 执行失败（未落地）: %v", c.plugin, c.tool, err)
			continue
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("%s/%s 返回空（落地但无输出？）", c.plugin, c.tool)
		}
	}
}
