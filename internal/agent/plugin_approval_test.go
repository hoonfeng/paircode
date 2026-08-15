// plugin_approval_test.go — client 半激活人工批准（合并现有审批门）。
//
// 对齐 deepseek-harness「per Plugin, human-approved Client activation」：
//   - cordis_run 装载带 client 半的插件时，DynamicApproval 判定 → 自动进现有审批门
//     （ReviewMode manual=人工审批 / auto=AI 审核 / off=放行）
//   - 工具执行成功（=已过审批门）→ MarkClientApproved(pluginId) 记录，覆盖后续版本
//   - 浏览器仅装载已批准的 client 半；批准记录持久化 .pair/cordis-approved.json
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const approvalClientCode = `(ui) => { ui.registerPanel({ id: 'ap-panel', title: 'AP' }); }`

// TestClientApprovalDynamic cordis_run 的 DynamicApproval 判定：
// 带 client 半未批准 → true；纯 host 插件 → false；已批准 → false（覆盖后续版本）。
func TestClientApprovalDynamic(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	RegisterCordisTools(reg, host, dir)

	runTool, ok := reg.Get("cordis_run")
	if !ok {
		t.Fatal("cordis_run 未注册")
	}
	if runTool.DynamicApproval == nil {
		t.Fatal("cordis_run 应配 DynamicApproval（client 激活动态审批）")
	}

	// 带 client 半、未批准 → 应触发审批
	id1, err := host.DefineJSCodeFull(`return { name: 'ap-with-ui', apply(ctx) {} };`, "", "", "", approvalClientCode)
	if err != nil {
		t.Fatalf("define(带 client): %v", err)
	}
	tc1 := ToolCall{Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"` + id1 + `"}`}}
	if !runTool.DynamicApproval(tc1) {
		t.Fatal("带 client 半未批准 → DynamicApproval 应为 true（触发审批）")
	}

	// 纯 host 插件 → 不触发
	id2, err := host.DefineJS(`return { name: 'ap-plain', apply(ctx) {} };`, "plain")
	if err != nil {
		t.Fatalf("define(纯 host): %v", err)
	}
	tc2 := ToolCall{Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"` + id2 + `"}`}}
	if runTool.DynamicApproval(tc2) {
		t.Fatal("纯 host 插件不应触发 client 审批")
	}

	// 已批准 → 不再触发（批准覆盖后续版本）
	def1 := mustDef(t, host, id1)
	host.MarkClientApproved(def1.pluginId)
	if runTool.DynamicApproval(tc1) {
		t.Fatal("已批准插件再次 run 不应再触发审批")
	}

	// 非法参数/未知 id → 不触发（不拦截无关调用）
	if runTool.DynamicApproval(ToolCall{Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"dyn-999"}`}}) {
		t.Fatal("未知 id 不应触发审批")
	}
	if runTool.DynamicApproval(ToolCall{Function: FunctionCall{Name: "cordis_run", Arguments: `not-json`}}) {
		t.Fatal("非法 JSON 参数不应触发审批")
	}
}

// TestClientApprovalGateLoop 审批门集成（Loop 真跑）：manual 模式下拒绝 → cordis_run
// 不执行、不批准；批准 → 执行并批准（MarkClientApproved）。
func TestClientApprovalGateLoop(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	RegisterCordisTools(reg, host, dir)

	id, err := host.DefineJSCodeFull(`return { name: 'ap-gate', apply(ctx) {} };`, "", "", "", approvalClientCode)
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	pluginID := mustDef(t, host, id).pluginId

	// ── 拒绝场景：Approve 返回 false → cordis_run 不执行 → 不批准 ──
	mockRej := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"` + id + `"}`}}}},
		{Content: "放弃装载"},
	}}
	loopRej := &Loop{Provider: mockRej, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { return false, "不需要浏览器 UI" }}
	if _, err := loopRej.Run(context.Background(), "装载插件", nil); err != nil {
		t.Fatalf("Run(拒绝): %v", err)
	}
	if host.IsClientApproved(pluginID) {
		t.Fatal("审批拒绝后不应批准 client 半")
	}
	// 插件 host 半也未装载（工具未执行）
	if rec := host.InspectDetail("ap-gate"); rec != nil && rec.State == "running" {
		t.Fatal("被拒绝的 cordis_run 不应装载插件")
	}

	// ── 批准场景：Approve 返回 true → 工具执行 → 批准 ──
	mockApp := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c2", Type: "function", Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"` + id + `"}`}}}},
		{Content: "装载完成"},
	}}
	loopApp := &Loop{Provider: mockApp, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { return true, "" }}
	if _, err := loopApp.Run(context.Background(), "装载插件", nil); err != nil {
		t.Fatalf("Run(批准): %v", err)
	}
	if !host.IsClientApproved(pluginID) {
		t.Fatal("审批通过后应批准 client 半")
	}
	// 插件已装载
	if rec := host.InspectDetail("ap-gate"); rec == nil || rec.State != "running" {
		t.Fatalf("批准后插件应 running，得 %+v", rec)
	}
	// 批准后再次 run 不再触发审批（Approve 不应再被 cordis_run 调用）
	mockAgain := &MockProvider{Responses: []Message{
		{ToolCalls: []ToolCall{{ID: "c3", Type: "function", Function: FunctionCall{Name: "cordis_run", Arguments: `{"id":"` + id + `"}`}}}},
		{Content: "重载完成"},
	}}
	approveCalls := 0
	loopAgain := &Loop{Provider: mockAgain, Registry: reg, MaxIterations: 5,
		Approve: func(ctx context.Context, tc ToolCall) (bool, string) { approveCalls++; return true, "" }}
	if _, err := loopAgain.Run(context.Background(), "重载插件", nil); err != nil {
		t.Fatalf("Run(重载): %v", err)
	}
	if approveCalls != 0 {
		t.Fatalf("已批准插件重载不应再触发审批，Approve 被调 %d 次", approveCalls)
	}
}

// TestClientApprovalPersist 批准记录持久化：MarkClientApproved 后重建宿主（同 root）
// 仍生效（.pair/cordis-approved.json）。
func TestClientApprovalPersist(t *testing.T) {
	dir := t.TempDir()
	host := NewPluginHost(NewRegistry(), nil, dir)
	id, err := host.DefineJSCodeFull(`return { name: 'ap-persist', apply(ctx) {} };`, "", "", "", approvalClientCode)
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	pluginID := mustDef(t, host, id).pluginId

	if host.IsClientApproved(pluginID) {
		t.Fatal("初始不应已批准")
	}
	host.MarkClientApproved(pluginID)

	// 批准文件已落盘
	p := filepath.Join(dir, ".pair", "cordis-approved.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("批准文件应存在: %v", err)
	}
	if !strings.Contains(string(data), pluginID) {
		t.Fatalf("批准文件应含 pluginId %s: %s", pluginID, data)
	}

	// 重建宿主 → 批准恢复
	host2 := NewPluginHost(NewRegistry(), nil, dir)
	if !host2.IsClientApproved(pluginID) {
		t.Fatal("重建宿主后批准记录应恢复（跨重启存续）")
	}
	// 坏 JSON 文件 → 静默忽略不崩溃
	_ = os.WriteFile(p, []byte("{broken"), 0o644)
	host3 := NewPluginHost(NewRegistry(), nil, dir)
	if host3.IsClientApproved(pluginID) {
		t.Fatal("坏 JSON 应被静默忽略")
	}
}

// TestClientApprovalInspect client 批准状态在 inspect 报告可见（L1 摘要 + 记录）。
func TestClientApprovalInspect(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	RegisterCordisTools(reg, host, dir)

	id, err := host.DefineJSCodeFull(`return { name: 'ap-inspect', apply(ctx) {} };`, "", "", "", approvalClientCode)
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	pluginID := mustDef(t, host, id).pluginId
	// 装载（正式路径经 cordis_run 审批门；此处直接 Load 验证 inspect 字段显示）
	if err := host.LoadJSDynamic(mustDef(t, host, id)); err != nil {
		t.Fatalf("run: %v", err)
	}

	// 未批准：记录 + L1 摘要显示待批准
	rec := host.InspectDetail("ap-inspect")
	if rec == nil || !rec.HasClient || rec.ClientApproved {
		t.Fatalf("未批准记录应 hasClient && !clientApproved: %+v", rec)
	}
	rep, err := cordisInspectReport(host, "", "")
	if err != nil || !strings.Contains(rep, "client=待批准") {
		t.Fatalf("L1 摘要应显示待批准: %v\n%s", err, rep)
	}

	// 批准后：记录 + 摘要更新
	host.MarkClientApproved(pluginID)
	rec = host.InspectDetail("ap-inspect")
	if rec == nil || !rec.ClientApproved {
		t.Fatalf("批准后记录应 clientApproved: %+v", rec)
	}
	rep, _ = cordisInspectReport(host, "", "")
	if !strings.Contains(rep, "client=已批准") {
		t.Fatalf("L1 摘要应显示已批准: %s", rep)
	}
}
