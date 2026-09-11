package agent

// handoff_js_test.go — 会话交接插件化链路测试（2026-09-12）。
//
// 覆盖：
//  1. JS 实现优先：注册 registerHandoff 后，宿主两个边界入口委托 JS（不走 Go 默认）；
//  2. 真实 agentloop 插件链路：装载磁盘插件 → 触发 → 生成 → 复用，且视图/记录正确；
//  3. 口径同源（记录互认）：JS 实现写入的 .handoff.json 能被 Go 默认实现读取并复用
//     （锚点指纹与阈值判定两侧一致 —— 缓存前缀稳定的前提）；
//  4. 可回退：JS 实现抛错 → 回退 Go 默认（功能不丢）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// regFakeJSHandoff 注册一个假 JS 交接实现（直接构造 impl，绕过插件装载），
// 返回清理函数。用于验证「优先权 / 失败回退」两条语义。
func regFakeJSHandoff(t *testing.T, onUserTurn, onSegment string) *jsHandoffImpl {
	t.Helper()
	vm := goja.New()
	impl := &jsHandoffImpl{id: "fake-handoff", vm: vm}
	if onUserTurn != "" {
		v, err := vm.RunString(onUserTurn)
		if err != nil {
			t.Fatalf("构造 onUserTurn 失败: %v", err)
		}
		fn, ok := goja.AssertFunction(v)
		if !ok {
			t.Fatal("onUserTurn 不是函数")
		}
		impl.onUserTurn = fn
	}
	if onSegment != "" {
		v, err := vm.RunString(onSegment)
		if err != nil {
			t.Fatalf("构造 onSegment 失败: %v", err)
		}
		fn, ok := goja.AssertFunction(v)
		if !ok {
			t.Fatal("onSegment 不是函数")
		}
		impl.onSegment = fn
	}
	restore := RegisterJSHandoff(impl)
	t.Cleanup(restore)
	return impl
}

// TestHandoffJS_Precedence 注册后宿主边界入口委托 JS（返回 JS 给的视图）。
func TestHandoffJS_Precedence(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	regFakeJSHandoff(t,
		`(function(args){ return { view: [{role:'user',content:'JS 交接视图'}], applied: true, notice: '来自 JS 的提示' } })`,
		`(function(args){ return { view: [{role:'user',content:'JS 段边界视图'}], applied: true } })`)

	root := t.TempDir()
	store := NewMessageStore(root)
	hist := handoffHist(120, "prec") // ≥100 条 → 达门槛

	view, ok, notice := HandoffUserTurnView(context.Background(), nil, nil, store,
		"conv_prec", root, hist, "继续", 0)
	if !ok || len(view) != 1 || view[0].Content != "JS 交接视图" {
		t.Fatalf("用户输入边界应委托 JS 实现: ok=%v view=%+v", ok, view)
	}
	if notice != "来自 JS 的提示" {
		t.Fatalf("notice 应透传 JS 返回值: %q", notice)
	}

	// applied=false → 不启用，且不回退 Go 默认（避免重复判定）
	regFakeJSHandoff(t, `(function(args){ return { applied: false } })`, "")
	if view, ok, _ := HandoffUserTurnView(context.Background(), nil, nil, store,
		"conv_prec2", root, hist, "继续", 0); ok || view != nil {
		t.Fatalf("applied=false 应不启用交接: ok=%v view=%+v", ok, view)
	}
}

// TestHandoffJS_SegmentView 段边界入口委托 JS（judge 恒为 null）。
func TestHandoffJS_SegmentView(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	regFakeJSHandoff(t, "",
		`(function(args){
			if (args.judge !== null) throw new Error('段边界 judge 必须为 null');
			if (args.kind !== 'segment') throw new Error('kind 应为 segment');
			return { view: [{role:'user',content:'段边界视图'}], applied: true };
		})`)
	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "seg"), MaxContextTokens: 0}
	view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_seg", "继续")
	if !ok || len(view) != 1 || view[0].Content != "段边界视图" {
		t.Fatalf("段边界应委托 JS 实现: ok=%v view=%+v", ok, view)
	}
}

// TestHandoffJS_FailureFallsBack 注册但执行抛错 → 回退 Go 默认实现（功能不丢）。
func TestHandoffJS_FailureFallsBack(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	regFakeJSHandoff(t, `(function(args){ throw new Error('boom') })`,
		`(function(args){ throw new Error('boom-seg') })`)

	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_fb"
	hist := handoffHist(120, "fb")

	view, ok, _ := HandoffUserTurnView(context.Background(), nil, nil, store, conv, root, hist, "继续", 0)
	if !ok || len(view) == 0 {
		t.Fatal("JS 实现失败时应回退 Go 默认实现并启用交接")
	}
	if !isHandoffText(view[0].Content) {
		t.Fatalf("回退视图首条应为交接消息: %q", view[0].Content)
	}
	// 回退路径写入的记录可供后续复用（同一 store/conv）
	if rec, err := store.LoadHandoff(conv); err != nil || rec == nil || rec.Anchor == "" {
		t.Fatalf("回退路径应写入交接记录: rec=%+v err=%v", rec, err)
	}
}

// TestHandoffJS_RealAgentloopPlugin 真实 agentloop 插件链路：
// 装载磁盘插件 → registerHandoff 生效 → 生成 → 复用 → 与 Go 默认实现记录互认。
func TestHandoffJS_RealAgentloopPlugin(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if _, err := os.Stat(filepath.Join("..", "..", ".pair", "plugins", "agentloop", "index.js")); err != nil {
		t.Skipf("无 agentloop 磁盘插件源码: %v", err)
	}
	loadRealAgentloop(t) // 同时注册 registerLoop 与 registerHandoff
	if CurrentJSHandoff() == nil {
		t.Fatal("agentloop 插件装载后应注册 JS 交接实现（registerHandoff）")
	}

	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_js_real"
	llmCalls := 0
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		llmCalls++
		body := "## 与当前任务的相关性\n高：同一任务的延续，继续验证插件化交接链路。\n" +
			strings.Repeat("插件化交接要点内容。", 8)
		return Message{Role: RoleAssistant, Content: body}, nil
	}}

	hist := handoffHist(120, "jsreal") // ≥100 条 → 达门槛

	// ① 首次：JS 实现生成交接（LLM 经 provider 桥接调用一次）
	view, ok, _ := HandoffUserTurnView(context.Background(), prov, nil, store, conv, root, hist, "继续", 0)
	if !ok || len(view) == 0 {
		t.Fatal("JS 实现应启用交接视图")
	}
	if !isHandoffText(view[0].Content) {
		t.Fatalf("视图首条应为交接消息: %q", truncRunesAgent(view[0].Content, 60))
	}
	if llmCalls != 1 {
		t.Fatalf("整理应经 provider 桥接调用一次 LLM: %d", llmCalls)
	}
	rec, err := store.LoadHandoff(conv)
	if err != nil || rec == nil {
		t.Fatalf("JS 实现应写入交接记录: rec=%+v err=%v", rec, err)
	}
	if rec.Anchor == "" || rec.MsgCount != 120 || rec.Relevance != handoffRelHigh {
		t.Fatalf("记录字段异常（锚点/条数/相关性）: %+v", rec)
	}
	if isHandoffText(strings.Join([]string{rec.Text}, "")) == false {
		t.Fatal("交接文本应带 marker+标题（注入识别标记）")
	}
	// 视图 = [交接消息] + 保留段（高相关性 16 条，基点非 tool）
	if len(view) != 1+handoffKeepRelHigh {
		t.Fatalf("视图应为 1+%d 条: %d", handoffKeepRelHigh, len(view))
	}

	// ② 追加 1 条（增量 ~1 条 tokens ≪ 刷新阈值）→ 复用同一交接文本，不再调 LLM，
	//    视图只追加增量（前缀单调延展）。
	hist2 := append(append([]Message{}, hist...), handoffHist(1, "small")...)
	view2, ok2, _ := HandoffUserTurnView(context.Background(), prov, nil, store, conv, root, hist2, "继续", 0)
	if !ok2 {
		t.Fatal("② 应启用交接")
	}
	if llmCalls != 1 {
		t.Fatalf("复用期不应再调 LLM: %d", llmCalls)
	}
	if view2[0].Content != view[0].Content {
		t.Fatal("复用期交接文本应逐字节不变（KV 前缀稳定前提）")
	}
	if len(view2) != len(view)+1 {
		t.Fatalf("复用期视图应只追加增量: %d → %d", len(view), len(view2))
	}
	// 前缀单调：view2 的前 len(view) 条与 view 逐字节一致
	for i := range view {
		if view2[i].Content != view[i].Content || view2[i].Role != view[i].Role {
			t.Fatalf("复用期视图前缀应逐字节稳定（第 %d 条）", i)
		}
	}

	// ③ 口径同源（记录互认）：注销 JS 实现 → Go 默认实现读同一记录 → 复用同一文本
	UnregisterJSHandoff(CurrentJSHandoff())
	view3, ok3 := BuildHandoffView(context.Background(), prov, nil, store, conv, hist2, "继续", 0)
	if !ok3 {
		t.Fatal("③ Go 默认实现应启用交接")
	}
	if view3[0].Content != view[0].Content {
		t.Fatal("Go 默认实现应复用 JS 写入的交接文本（锚点指纹口径同源）")
	}
	if llmCalls != 1 {
		t.Fatalf("③ 复用期不应再调 LLM: %d", llmCalls)
	}
	if len(view3) != len(view2) {
		t.Fatalf("两侧保留段应一致（同一锚点定位）: %d vs %d", len(view3), len(view2))
	}
}

// TestHandoffJS_RecordInterop_GoToJS 反向互认：Go 默认实现写的记录，JS 实现应能复用。
func TestHandoffJS_RecordInterop_GoToJS(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_interop"
	llmCalls := 0
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		llmCalls++
		return Message{Role: RoleAssistant, Content: "## 与当前任务的相关性\n高：沿用 Go 侧生成的交接要点。" + strings.Repeat("内容", 30)}, nil
	}}
	hist := handoffHist(120, "interop")

	// Go 默认实现生成
	goView, ok := BuildHandoffView(context.Background(), prov, nil, store, conv, hist, "继续", 0)
	if !ok || len(goView) == 0 {
		t.Fatal("Go 默认实现应启用交接")
	}
	if llmCalls != 1 {
		t.Fatalf("应调用一次 LLM: %d", llmCalls)
	}

	// 装载真实插件（注册 JS 实现）→ 复用 Go 写的记录（不再调 LLM）
	loadRealAgentloop(t)
	if CurrentJSHandoff() == nil {
		t.Fatal("agentloop 插件应注册 JS 交接实现")
	}
	jsView, ok2, _ := HandoffUserTurnView(context.Background(), prov, nil, store, conv, root, hist, "继续", 0)
	if !ok2 || len(jsView) == 0 {
		t.Fatal("JS 实现应启用交接")
	}
	if jsView[0].Content != goView[0].Content {
		t.Fatal("JS 实现应复用 Go 写入的交接文本（记录互认）")
	}
	if llmCalls != 1 {
		t.Fatalf("复用期不应再调 LLM: %d", llmCalls)
	}
	if len(jsView) != len(goView) {
		t.Fatalf("两侧保留段应一致: %d vs %d", len(jsView), len(goView))
	}
	_ = fmt.Sprintf // 保留 fmt（调试用）
}

// TestHandoffJS_SegmentReuseKeepsPrefixStable 段边界复用链路（缓存关键场景）：
// 第 1 段结束生成交接 → 第 2 段以「上段视图 + 本段新增」继续 → 段边界再次整理时，
// 视图前缀必须逐字节稳定（复用同一交接 + 保留段不重排），且只追加增量。
// 覆盖实测暴露的缺陷：视图历史下锚点定位失败 → 兜底「最近 N 条」→ 每段都断裂。
func TestHandoffJS_SegmentReuseKeepsPrefixStable(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	loadRealAgentloop(t)
	if CurrentJSHandoff() == nil {
		t.Fatal("agentloop 插件应注册 JS 交接实现")
	}
	root := t.TempDir()
	store := NewMessageStore(root)
	conv := "conv_seg_reuse"

	seg1 := handoffHist(120, "s1")
	loop1 := &Loop{History: seg1}
	v1, ok1, _ := HandoffSegmentView(context.Background(), loop1, store, conv, "继续")
	if !ok1 {
		t.Fatal("第 1 段边界应启用交接")
	}
	rec, _ := store.LoadHandoff(conv)
	if rec == nil {
		t.Fatal("第 1 段边界应写入交接记录")
	}
	t.Logf("① 视图 %d 条；记录 msgCount=%d keptTokens=%d", len(v1), rec.MsgCount, rec.KeptTokens)

	// 第 2 段：loop.History = 上段注入的视图 + 本段新增
	seg2 := append(cloneMsgs(v1), handoffHist(120, "s2")...)
	loop2 := &Loop{History: seg2}
	v2, ok2, _ := HandoffSegmentView(context.Background(), loop2, store, conv, "继续")
	if !ok2 {
		t.Fatal("第 2 段边界应启用交接")
	}
	t.Logf("② 视图 %d 条（历史 %d 条）", len(v2), len(seg2))
	if len(v2) <= len(v1) {
		t.Fatalf("第 2 段视图应只追加增量（期望 > %d，实际 %d）——锚点定位失败退回兜底",
			len(v1), len(v2))
	}
	for i := range v1 {
		if v2[i].Content != v1[i].Content || v2[i].Role != v1[i].Role {
			t.Fatalf("视图前缀应逐字节稳定（第 %d 条）", i)
		}
	}
}

// TestHandoffJS_EquivalentToGoDefault 迁移等价性：同一历史 / 同一 Provider 输出下，
// JS 实现与 Go 默认实现必须产出**逐字节一致**的视图与记录字段
// （否则交接周期边界会因两侧判定不同而抖动 → 前缀断裂）。
func TestHandoffJS_EquivalentToGoDefault(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	hist := handoffHist(120, "eq")
	llmText := "## 与当前任务的相关性\n高：同一任务的延续。\n## 任务目标\n沿用历史目标\n## 已完成\n多步验证"
	prov := &funcProvider{chat: func(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
		return Message{Role: RoleAssistant, Content: llmText}, nil
	}}

	// ① Go 默认实现（未注册 JS）
	storeGo := NewMessageStore(t.TempDir())
	goView, goOK := BuildHandoffView(context.Background(), prov, nil, storeGo, "conv_eq", hist, "继续", 0)
	if !goOK {
		t.Fatal("Go 默认实现应启用交接")
	}
	goRec, _ := storeGo.LoadHandoff("conv_eq")

	// ② JS 实现（真实插件）
	loadRealAgentloop(t)
	if CurrentJSHandoff() == nil {
		t.Fatal("agentloop 插件应注册 JS 交接实现")
	}
	storeJS := NewMessageStore(t.TempDir())
	jsView, jsOK, _ := HandoffUserTurnView(context.Background(), prov, nil, storeJS,
		"conv_eq", t.TempDir(), hist, "继续", 0)
	if !jsOK {
		t.Fatal("JS 实现应启用交接")
	}
	jsRec, _ := storeJS.LoadHandoff("conv_eq")

	// ③ 视图逐字节一致
	if len(jsView) != len(goView) {
		t.Fatalf("视图长度应一致: JS=%d Go=%d", len(jsView), len(goView))
	}
	for i := range goView {
		if jsView[i].Role != goView[i].Role || jsView[i].Content != goView[i].Content {
			t.Fatalf("视图第 %d 条应逐字节一致:\n  JS=%q\n  Go=%q",
				i, truncRunesAgent(jsView[i].Content, 60), truncRunesAgent(goView[i].Content, 60))
		}
	}
	// ④ 记录字段一致（CreatedAt 为时间戳，不比）
	if goRec == nil || jsRec == nil {
		t.Fatalf("两侧都应写入记录: go=%v js=%v", goRec, jsRec)
	}
	if goRec.Text != jsRec.Text {
		t.Fatal("交接文本应一致")
	}
	if goRec.Anchor != jsRec.Anchor {
		t.Fatalf("锚点指纹应一致（跨实现互认的前提）:\n  JS=%q\n  Go=%q",
			jsRec.Anchor, goRec.Anchor)
	}
	if goRec.MsgCount != jsRec.MsgCount || goRec.Relevance != jsRec.Relevance ||
		goRec.KeptTokens != jsRec.KeptTokens {
		t.Fatalf("记录字段应一致:\n  JS=%+v\n  Go=%+v", jsRec, goRec)
	}
}
