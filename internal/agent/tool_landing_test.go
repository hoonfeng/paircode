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
//   tool-screenshot/tool-web-debug（→tool-web）、tool-goal（→tool-system）、
//   tool-subagent（→tool-workflow）、tool-bridge（已删除）不再独立存在，
//   模式表同步删除对应行。
var toolPluginModes = map[string]string{
	"tool-system":          "hostTool",
	"tool-codegraph":       "binary",
	"tool-binary":          "binary",
	// ★ 2026-09 Round4.5：tool-debug 已移除（纯命令行包装壳、无组合逻辑）
	"tool-office":          "mixed",
	"tool-harness":         "mixed", // ★ 07-23 合并 tool-shell：6 后台进程工具为 JS 原生（原 tool-shell native 模式）
	"tool-git":             "native",
	"tool-core":            "native",
	"tool-memory":          "native",
	"tool-project-info":    "native",
	"tool-vision":          "native",
	"tool-bug":             "native",
	"tool-web":             "mixed", // ★ ③.4 合并 tool-screenshot/tool-web-debug：screenshot_*/web_debug 走 binary 内核
	// ★ 2026-09 t1 T1 闭环：7 组孤儿工具迁移为磁盘插件（tool_plugin_gen.go 生成，
	//   execute 走 ctx.hostTool.exec 复用宿主存档能力——legacy_host_tools.go）
	"tool-asset":       "hostTool",
	"tool-entryconfig": "hostTool",
	"tool-resource":    "hostTool", // ★ 2026-09-04 tool-verify 并入：verify 2 工具 JS 原生 impl 优先、hostTool 兜底（registerVerifyTools 已加档）
	"tool-snapshot":    "hostTool",
	// ★ Round3 ③/⑤：goal/workflow 宿主机制工具面（execute → ctx.hostTool）；
	//   subagent 系列 2026-09 ③.4 并入 tool-workflow（ctx.agents 服务，JS 原生编排）
	"tool-workflow": "mixed", // workflow→hostTool 执行器；subagent 系列→ctx.agents 服务（均不在内嵌内核，别名声明）
}

// toolHarnessAliases 混合型插件的 JS 原生实现工具（不在内嵌内核，断言跳过）：
// tool-harness：read/write/edit/glob/grep（ctx.fs 原生）+
//   run_background/read_output/kill_process/job_output/job_list/job_kill
//   （ctx.process 原生，③.4 并入自 tool-shell）；
// tool-web：web_fetch/web_search（ctx.web 原生，③.4 后为 mixed 型）。
// （run_code 仍由 registerRunCode 覆盖内核，正常断言；str_replace_editor 已移除
// （Round4）；screenshot_*/web_debug 同理内核覆盖。）
var toolHarnessAliases = map[string]bool{
	"read": true, "write": true, "edit": true, "glob": true, "grep": true,
	"run_background": true, "read_output": true, "kill_process": true,
	"job_output": true, "job_list": true, "job_kill": true,
	"web_fetch": true, "web_search": true,
	// tool-workflow（2026-09 ③.4 后为 mixed）：workflow→宿主 goja 运行器（hostTool），
	// subagent 系列→ctx.agents 宿主服务——二者均非内嵌内核，别名声明跳过
	"workflow": true, "subagent": true, "subagent_fork": true, "report": true,
	"list_agents": true, "interrupt_agent": true, "send_message": true,
}

// loadDiskPluginForTestFramed 装载磁盘插件，且宿主注册表预注册框架工具
// （对齐生产时序：claimTool 时框架工具已注册 → ArchiveHostTool 存档生效）。
// 返回 host + 共享 reg（与 loadDiskPluginForTest 同形）。
func loadDiskPluginForTestFramed(t *testing.T, name string) (*PluginHost, *Registry) {
	t.Helper()
	root := jsNativeWorkspace
	code, err := os.ReadFile(filepath.Join(root, ".pair", "plugins", name, "index.js"))
	if err != nil {
		t.Fatalf("读取 %s/index.js: %v", name, err)
	}
	reg := NewRegistry()
	RegisterHostFrameworkTools(reg, root)
	host := NewPluginHost(reg, nil, root)
	id, err := host.DefineJSCodeFull(string(code), "", "落地验证", filepath.Join(root, ".pair", "plugins", name), "")
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

	embedded := InitEmbeddedToolRegistry(root)
	hostNames := map[string]bool{}
	for _, n := range HostToolNames() {
		hostNames[n] = true
	}

	dirs, err := os.ReadDir(filepath.Join(root, ".pair", "plugins"))
	if err != nil {
		t.Fatalf("读取插件目录: %v", err)
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
					problems = append(problems, "[hostTool 未存档] "+d.Name()+"/"+tn)
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
//   - native 型：project_info_list（ctx.fs 链路）
func TestToolLandingSpotCheck(t *testing.T) {
	root := jsNativeWorkspace
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
		{"tool-project-info", "project_info_list", `{}`},
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
