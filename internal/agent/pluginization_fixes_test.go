// pluginization_fixes_test.go — t2 实施验证（t1 报告高优先级缺口闭环的单元验证）
//
// 覆盖：
//   - C1/C2：角色提示磁盘优先 loader（config/roles/<name>.md 覆盖内置）
//   - T1：孤儿工具组宿主能力存档 + 磁盘插件（tool-asset）装载与 hostTool 执行
//   - S1：Provider 实现级插件槽位（RegisterProviderImpl/CreateProvider 路由 + JS 桥）
//   - G1：审核配置单源迁移（.pair/tools.json → settings.json）
//   - L2：钩子系统接线（配置钩子 PreToolUse 拦截 + 插件钩子拦截）
package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/internal/core"
)

// ─── C1/C2：角色提示统一提示词资产（★ 2026-09 升级：默认资产迁移至插件包，
// 漂移守卫见 prompt_assets_test.go 的 TestPluginPromptAssetsSync）───────────────

func TestRolePromptDiskOverride(t *testing.T) {
	dir := t.TempDir()
	SetRolePromptBaseDirForTest(dir)
	defer SetRolePromptBaseDirForTest("")

	// ① 无磁盘覆盖 → 回退内置
	if got := DefaultReviewerPrompt(); got != reviewerSystemPrompt {
		t.Fatalf("无磁盘覆盖时应回退内置 reviewer 提示")
	}
	if got := DefaultPlannerPrompt(); got != plannerSystemPrompt {
		t.Fatalf("无磁盘覆盖时应回退内置 planner 提示")
	}
	if got := DefaultJudgePrompt(); got != judgeSystemPrompt {
		t.Fatalf("无磁盘覆盖时应回退内置 judge 提示")
	}

	// ② 磁盘覆盖 → 磁盘优先（不重编译即可换角色）
	override := "# 自定义审核角色\n你是自定义 Reviewer，铁律：只放行无风险操作。"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reviewer.md"), []byte(override), 0o644); err != nil {
		t.Fatal(err)
	}
	// 缓存语义：角色提示静态缓存（改文件需重启）；测试中显式复位模拟重启
	LoadRolePromptReset()
	if got := DefaultReviewerPrompt(); got != override {
		t.Fatalf("磁盘覆盖未生效：got %q want %q", got, override)
	}
	// ③ 空白文件视为无覆盖
	if err := os.WriteFile(filepath.Join(dir, "planner.md"), []byte("  \n\n  "), 0o644); err != nil {
		t.Fatal(err)
	}
	LoadRolePromptReset()
	if got := DefaultPlannerPrompt(); got != plannerSystemPrompt {
		t.Fatalf("空白文件应回退内置 planner 提示")
	}
}

// ─── T1：孤儿工具宿主存档 + 磁盘插件装载执行 ────────────────

func TestArchiveHostLegacyToolsAndPluginLoad(t *testing.T) {
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, t.TempDir())

	// ① 宿主能力已存档（hostTool seam：能力在宿主）
	for _, name := range []string{"asset_list", "asset_search", "asset_delete",
		"find_entry_points", "evolution_status", "progress_checker",
		"resource_list", "list_snapshots", "bridge_status"} {
		if _, ok := HostToolMeta(name); !ok {
			t.Fatalf("孤儿工具 %s 未存档为宿主能力", name)
		}
	}
	// ② 未注册进任何 Registry（agent 可见面由插件决定）
	for _, name := range []string{"asset_list", "progress_checker", "list_snapshots"} {
		if _, ok := reg.Get(name); ok {
			t.Fatalf("%s 不应直接注册进宿主 Registry（可见面应由插件决定）", name)
		}
	}

	// ③ 磁盘插件（tool-asset，生成器产物）装载 → 插件工具接管 + hostTool 执行
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	code, err := os.ReadFile(filepath.Join(repoRoot, ".pair", "plugins", "tool-asset", "index.js"))
	if err != nil {
		t.Skipf("tool-asset 插件未生成（先跑 go run -tags toolsgen ./dev/tool_plugin_gen）: %v", err)
	}
	id, err := host.DefineJSCodeFull(string(code), "js", "tool-asset 装载测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	for _, name := range []string{"asset_list", "asset_search", "asset_delete"} {
		tool, ok := reg.Get(name)
		if !ok {
			t.Fatalf("%s 未注册（插件未接管）", name)
		}
		if strings.TrimSpace(tool.Description) == "" {
			t.Fatalf("%s 描述为空", name)
		}
	}
	// ④ 执行走 ctx.hostTool.exec → 宿主 Go 实现（seam 闭环）
	out, err := reg.Execute(context.Background(), "asset_list", `{}`)
	if err != nil {
		t.Fatalf("asset_list 执行失败: %v", err)
	}
	if !strings.Contains(out, "智能资产") && !strings.Contains(out, "资产") {
		t.Fatalf("asset_list 输出异常: %q", out)
	}
	// ⑤ 全部 7 组插件包均生成（package.json 齐全，LoadGlobalPlugins 可扫描）
	for _, name := range []string{"tool-asset", "tool-bridge", "tool-entryconfig",
		"tool-evolution", "tool-progress", "tool-resource", "tool-snapshot"} {
		if _, err := os.Stat(filepath.Join(repoRoot, ".pair", "plugins", name, "package.json")); err != nil {
			t.Fatalf("插件包 %s 缺 package.json: %v", name, err)
		}
	}
}

// ─── S1：Provider 实现级插件槽位 ───────────────────────────

func TestCreateProviderRouting(t *testing.T) {
	// ① 未注册服务商 → 回退 OpenAI 兼容实现
	p := CreateProvider(ProviderParams{Provider: "deepseek", BaseURL: "http://x", APIKey: "k", Model: "m"})
	if _, ok := p.(*OpenAIProvider); !ok {
		t.Fatalf("未注册服务商应回退 OpenAIProvider，got %T", p)
	}
	// ② 注册实现 → 按服务商名路由（大小写不敏感）
	restore := RegisterProviderImpl("custom-proto", func(params ProviderParams) Provider {
		return &MockProvider{Responses: []Message{{Role: RoleAssistant, Content: "custom:" + params.Model}}}
	})
	defer restore()
	p2 := CreateProvider(ProviderParams{Provider: "Custom-Proto", Model: "m2"})
	mp, ok := p2.(*MockProvider)
	if !ok {
		t.Fatalf("应路由到注册实现，got %T", p2)
	}
	msg, err := mp.Chat(context.Background(), nil, nil, nil)
	if err != nil || msg.Content != "custom:m2" {
		t.Fatalf("自定义实现 Chat 异常: %q %v", msg.Content, err)
	}
	// ③ 还原后回退
	restore()
	p3 := CreateProvider(ProviderParams{Provider: "custom-proto", BaseURL: "http://x", APIKey: "k", Model: "m"})
	if _, ok := p3.(*OpenAIProvider); !ok {
		t.Fatalf("还原后应回退 OpenAIProvider，got %T", p3)
	}
}

func TestJSProviderImplBridge(t *testing.T) {
	// JS 插件经 ctx.provider.register 注册实现；CreateProvider 按服务商名路由；
	// Chat 走 JS 实现（非流式契约，一次性 Done chunk）。
	src := `
const impl = {
  chat(params, messages, tools) {
    const last = messages[messages.length - 1] || {};
    return { content: 'js-echo:' + last.content + ':' + params.model,
             reasoning: 'js-reasoning', toolCalls: [] };
  }
};
return {
  name: 'js-provider-test',
  apply(ctx) { ctx.provider.register('js-test-proto', impl); },
}`
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, t.TempDir())
	id, err := host.DefineJSCodeFull(src, "js", "js provider 测试", "", "")
	if err != nil {
		t.Fatalf("define 失败: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("装载失败: %v", err)
	}
	defer host.Unload("js-provider-test") // 卸载 → 实现注册自动还原

	p := CreateProvider(ProviderParams{Provider: "js-test-proto", Model: "m9"})
	if p == nil {
		t.Fatal("CreateProvider 应返回 JS 实现")
	}
	if p.Name() != "js-test-proto" {
		t.Fatalf("Name 异常: %s", p.Name())
	}
	msg, err := p.Chat(context.Background(), []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: "hi"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("JS Chat 失败: %v", err)
	}
	if msg.Content != "js-echo:hi:m9" {
		t.Fatalf("JS Chat 内容异常: %q", msg.Content)
	}
	if msg.Reasoning != "js-reasoning" {
		t.Fatalf("JS Chat reasoning 异常: %q", msg.Reasoning)
	}
	// 卸载后还原：回退 OpenAI 实现
	_ = host.Unload("js-provider-test")
	p2 := CreateProvider(ProviderParams{Provider: "js-test-proto", BaseURL: "http://x", APIKey: "k", Model: "m"})
	if _, ok := p2.(*OpenAIProvider); !ok {
		t.Fatalf("插件卸载后应回退 OpenAIProvider，got %T", p2)
	}
}

// ─── G1：审核配置单源迁移 ──────────────────────────────────

func TestReviewConfigSingleSourceMigration(t *testing.T) {
	// 测试隔离：全量套件中其他测试可能已改动 core.Settings——开始时重置为干净默认，
	// 结束时还原（防止顺序依赖）
	savedSettings := core.Settings
	core.Settings = core.Default()
	defer func() { core.Settings = savedSettings }()
	// core.Save() 在测试环境写 <pkg>/config/settings.json（ConfigDir 回退 wd/config），
	// 记录原状态并在测试后还原/删除，避免污染源码树。
	cfgPath := filepath.Join("config", "settings.json")
	hadCfg := false
	var savedCfg []byte
	if b, err := os.ReadFile(cfgPath); err == nil {
		hadCfg = true
		savedCfg = b
	}
	t.Cleanup(func() {
		if hadCfg {
			_ = os.WriteFile(cfgPath, savedCfg, 0o644)
		} else {
			_ = os.Remove(cfgPath)
		}
	})

	root := t.TempDir()
	pairDir := filepath.Join(root, ".pair")
	if err := os.MkdirAll(pairDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 遗留 tools.json（旧双源之一）
	legacy := `{"reviewMode":"off","reviewBlacklist":["delete_file"],"reviewWhitelist":["read_file"]}`
	if err := os.WriteFile(filepath.Join(pairDir, "tools.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	// 读取触发一次性迁移 → settings 顶层成为唯一来源
	mode, black, white := LoadWorkspaceReviewConfig(root)
	if mode != "off" {
		t.Fatalf("迁移后 reviewMode=%q want off", mode)
	}
	if len(black) != 1 || black[0] != "delete_file" {
		t.Fatalf("迁移后 blacklist=%v", black)
	}
	if len(white) != 1 || white[0] != "read_file" {
		t.Fatalf("迁移后 whitelist=%v", white)
	}
	if _, err := os.Stat(filepath.Join(pairDir, "tools.json")); !os.IsNotExist(err) {
		t.Fatalf("迁移后 tools.json 应删除，err=%v", err)
	}
	if core.Settings.ReviewMode != "off" || len(core.Settings.ReviewBlacklist) != 1 {
		t.Fatalf("core.Settings 未更新: mode=%q black=%v", core.Settings.ReviewMode, core.Settings.ReviewBlacklist)
	}
	// 保存路径也走 settings（单源）：写回后再次读取一致
	if err := SaveWorkspaceReviewConfig(root, "manual", []string{"run_command"}, nil); err != nil {
		t.Fatal(err)
	}
	mode2, black2, _ := LoadWorkspaceReviewConfig(root)
	if mode2 != "manual" || len(black2) != 1 || black2[0] != "run_command" {
		t.Fatalf("Save 后读取不一致: mode=%q black=%v", mode2, black2)
	}
}

// ─── L2：钩子系统接线 ──────────────────────────────────────

// hookTestTool 注册一个简单工具供钩子门测试。
func hookTestTool(reg *Registry, name string) {
	reg.Register(&Tool{Name: name, Description: "测试工具", Parameters: objSchema(props{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "ok", nil }})
}

func TestLoopHookConfigBlock(t *testing.T) {
	root := t.TempDir()
	pairDir := filepath.Join(root, ".pair")
	if err := os.MkdirAll(pairDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 配置钩子：PreToolUse 恒 exit 2（阻塞门事件）
	cfg := map[string]any{"hooks": map[string]any{
		"PreToolUse": []map[string]any{{"match": ".*", "command": "exit 2", "description": "测试拦截"}},
	}}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(pairDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// ★ 信任门控（t4 M2 修复）：项目钩子需显式信任才装载——测试走受信任路径
	SetLoopHookRootsTrusted(root, "", true)
	defer SetLoopHookRoots("", "") // 还原（清空配置钩子）

	reg := NewRegistry()
	hookTestTool(reg, "hook_gated_tool")
	out, err := reg.Execute(context.Background(), "hook_gated_tool", `{}`)
	if err == nil || !strings.Contains(err.Error(), "钩子拦截") {
		t.Fatalf("PreToolUse 配置钩子应拦截（受信任路径）：out=%q err=%v", out, err)
	}
}

// TestLoopHookProjectTrustGate 信任门控（t4 M2 修复）：默认（不信任）时项目钩子
// 不装载——打开含恶意 .pair/settings.json 的工作区不会自动执行钩子 shell 命令。
func TestLoopHookProjectTrustGate(t *testing.T) {
	root := t.TempDir()
	pairDir := filepath.Join(root, ".pair")
	if err := os.MkdirAll(pairDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{"hooks": map[string]any{
		"PreToolUse": []map[string]any{{"match": ".*", "command": "exit 2", "description": "恶意拦截"}},
	}}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(pairDir, "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// 默认入口（不信任）：项目钩子不装载 → 工具正常执行
	SetLoopHookRoots(root, "")
	defer SetLoopHookRoots("", "")
	if st := LoopHooksStatus(); st["configHooks"].(int) != 0 {
		t.Fatalf("不信任时不应装载项目钩子：%v", st)
	}
	reg := NewRegistry()
	hookTestTool(reg, "hook_trusted_tool")
	if _, err := reg.Execute(context.Background(), "hook_trusted_tool", `{}`); err != nil {
		t.Fatalf("不信任时项目钩子不应拦截：%v", err)
	}
	// 显式信任后：同一工作区项目钩子装载并拦截
	SetLoopHookRootsTrusted(root, "", true)
	if st := LoopHooksStatus(); st["configHooks"].(int) != 1 {
		t.Fatalf("信任后应装载 1 个配置钩子：%v", st)
	}
	out, err := reg.Execute(context.Background(), "hook_trusted_tool", `{}`)
	if err == nil || !strings.Contains(err.Error(), "钩子拦截") {
		t.Fatalf("信任后项目钩子应拦截：out=%q err=%v", out, err)
	}
}

func TestLoopHookPluginBlock(t *testing.T) {
	defer SetLoopHookRoots("", "")
	restore := RegisterLoopHook("PreToolUse", func(payload map[string]any) (bool, string) {
		if payload["toolName"] == "hook_plugin_tool" {
			return true, "插件钩子禁止调用"
		}
		return false, ""
	})
	defer restore()

	reg := NewRegistry()
	hookTestTool(reg, "hook_plugin_tool")
	out, err := reg.Execute(context.Background(), "hook_plugin_tool", `{}`)
	if err == nil || !strings.Contains(err.Error(), "插件钩子禁止调用") {
		t.Fatalf("插件钩子应拦截：out=%q err=%v", out, err)
	}
	// 未命中钩子的工具正常执行
	hookTestTool(reg, "hook_other_tool")
	if _, err := reg.Execute(context.Background(), "hook_other_tool", `{}`); err != nil {
		t.Fatalf("未命中钩子的工具不应被拦截: %v", err)
	}
	// 还原后不再拦截
	restore()
	if _, err := reg.Execute(context.Background(), "hook_plugin_tool", `{}`); err != nil {
		t.Fatalf("还原后不应再拦截: %v", err)
	}
}
