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
//	             （tool-harness 例外：bash 等原生、run_code 走内核——内核无 bash 属正常）
//	native   : 全 JS 原生实现（ctx.fs/bash/process/web）
//
// ★ 修改插件实现模式时必须同步本表（测试是模式漂移守卫）。
var toolPluginModes = map[string]string{
	"tool-system":          "hostTool",
	"tool-codegraph":       "binary",
	"tool-codegraph-extra": "binary",
	"tool-binary":          "binary",
	"tool-screenshot":      "binary",
	"tool-web-debug":       "binary",
	"tool-debug":           "mixed",
	"tool-office":          "mixed",
	"tool-harness":         "mixed",
	"tool-git":             "native",
	"tool-shell":           "native",
	"tool-core":            "native",
	"tool-memory":          "native",
	"tool-project-info":    "native",
	"tool-verify":          "native",
	"tool-vision":          "native",
	"tool-bug":             "native",
	"tool-web":             "native",
	// ★ 2026-09 t1 T1 闭环：7 组孤儿工具迁移为磁盘插件（tool_plugin_gen.go 生成，
	//   execute 走 ctx.hostTool.exec 复用宿主存档能力——legacy_host_tools.go）
	"tool-asset":       "hostTool",
	"tool-bridge":      "hostTool",
	"tool-entryconfig": "hostTool",
	"tool-evolution":   "hostTool",
	"tool-progress":    "hostTool",
	"tool-resource":    "hostTool",
	"tool-snapshot":    "hostTool",
	// ★ Round3 ③/⑤：goal/workflow 宿主机制工具面（execute → ctx.hostTool）；
	//   tool-subagent 走 ctx.agents 服务（JS 原生编排，非 hostTool）
	"tool-goal":     "hostTool",
	"tool-subagent": "native",
	"tool-workflow": "hostTool",
}

// harnessNativeAliases tool-harness 的 JS 原生别名（模式表注释声明的例外）：
// read/write/edit/glob/grep/bash 由插件内 ctx.fs/ctx.bash 原生实现，
// 无二进制内核回退——混合型内嵌内核断言跳过这 6 个（str_replace_editor/run_code
// 仍由 registerStrReplaceEditor/registerRunCode 覆盖内核，正常断言）。
var harnessNativeAliases = map[string]bool{
	"read": true, "write": true, "edit": true, "glob": true, "grep": true, "bash": true,
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
				if _, ok := embedded.Get(tn); !ok && !harnessNativeAliases[tn] {
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
