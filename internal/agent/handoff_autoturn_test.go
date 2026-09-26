package agent

// handoff_autoturn_test.go — 自主续轮边界（监督续轮 / goal 自动续轮）**强制折叠**单测
// （2026-09-27）。
//
// 背景：监督续轮与 goal 续轮此前以 loop.Run(ctx, msg, nil) 唤醒工作 agent（nil = 复用
// loop.History 累计全量时间线），每轮续跑都全量重喂历史 → 上下文无界增长。改为每轮
// 续跑前经 HandoffAutoTurnView 强制折叠（视图 = [交接提交消息] + 最近 Keep 条原文）。
//
// 覆盖：
//  1. 强制折叠：历史**未达**触发阈值也折叠，视图深度 = 1 + Keep，Keep 可配；
//  2. 关闭开关：PAIR_HANDOFF_AUTO_TURN=0 → 不启用（调用方回落全量历史原行为）；
//  3. 复用与归一：增量未达刷新阈值 → 复用交接文本（不重生成）；旧记录深度归一为 Keep；
//  4. JS 优先：注册 onAutoTurn 后宿主委托 JS（args.color 契约正确），失败回退 Go 默认。

import (
	"context"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// autoTurnLoop 构造自主续轮场景的 Loop（History = 该轮结束时的完整时间线）。
func autoTurnLoop(hist []Message) *Loop {
	return &Loop{History: hist, MaxContextTokens: 0}
}

// TestHandoffAutoTurn_ForceCollapse 未达阈值也折叠：视图 = [交接] + 最近 Keep 条原文。
func TestHandoffAutoTurn_ForceCollapse(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "0")                     // 强制 Go 默认实现（隔离其它测试注册的 JS 实现）
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "")               // 默认 Keep
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100000000") // 阈值拉满 → 证明强制路径不看阈值

	store := NewMessageStore(t.TempDir())
	hist := handoffHist(12, "auto") // 12 条，远未达任何阈值
	loop := autoTurnLoop(hist)

	// 前置断言：常规边界（按阈值）此时不应启用——证明续轮走的是强制路径
	if _, ok := BuildHandoffView(context.Background(), nil, nil, store, "conv_auto_pre",
		hist, "继续", 0); ok {
		t.Fatal("未达阈值时常规边界不应启用交接（前置断言失败）")
	}

	view, ok, _ := HandoffAutoTurnView(context.Background(), loop, store, "conv_auto", "继续推进")
	if !ok {
		t.Fatal("自主续轮边界应强制折叠（未达阈值也启用）")
	}
	want := 1 + handoffAutoTurnKeepDefault
	if len(view) != want {
		t.Fatalf("视图应为 [交接]+最近 %d 条 = %d 条，实际 %d 条",
			handoffAutoTurnKeepDefault, want, len(view))
	}
	if !strings.HasPrefix(view[0].Content, handoffTitle) || view[0].Role != RoleUser {
		t.Fatalf("视图首条应为 user 角色的交接提交消息，实际 role=%s content=%q",
			view[0].Role, truncRunesAgent(view[0].Content, 40))
	}
	// 保留段应为历史尾部原文（不丢最近执行细节）
	tail := stripSystemMsgs(hist)
	tail = tail[len(tail)-handoffAutoTurnKeepDefault:]
	for i, m := range tail {
		if view[1+i].Content != m.Content {
			t.Fatalf("保留段第 %d 条应与历史逐字节一致：\nwant %q\ngot  %q",
				i, truncRunesAgent(m.Content, 30), truncRunesAgent(view[1+i].Content, 30))
		}
	}
	// 记录落盘：Keep 与视图深度一致（跨轮锚点精确复原的前提）
	rec, err := store.LoadHandoff("conv_auto")
	if err != nil || rec == nil {
		t.Fatalf("交接记录应已落盘: rec=%+v err=%v", rec, err)
	}
	if rec.Keep != handoffAutoTurnKeepDefault {
		t.Fatalf("记录 Keep 应为 %d，实际 %d", handoffAutoTurnKeepDefault, rec.Keep)
	}
	if rec.Anchor == "" || rec.MsgCount == 0 {
		t.Fatalf("记录锚点/条数应齐备: %+v", rec)
	}
}

// TestHandoffAutoTurn_KeepConfigurableAndDisabled Keep 可配（含上限/非法值）+ 关闭开关。
func TestHandoffAutoTurn_KeepConfigurableAndDisabled(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "0")
	store := NewMessageStore(t.TempDir())
	hist := handoffHist(20, "cfg")

	// Keep=3 → 视图 4 条
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "3")
	view, ok, _ := HandoffAutoTurnView(context.Background(), autoTurnLoop(hist), store, "conv_cfg3", "继续")
	if !ok || len(view) != 4 {
		t.Fatalf("PAIR_HANDOFF_AUTO_KEEP=3 → 视图应为 4 条，实际 ok=%v len=%d", ok, len(view))
	}
	if rec, _ := store.LoadHandoff("conv_cfg3"); rec == nil || rec.Keep != 3 {
		t.Fatalf("记录 Keep 应为 3，实际 %+v", rec)
	}

	// 超上限 → 收敛到上限；非法值 → 回落默认
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "999")
	if got := handoffAutoTurnKeep(); got != handoffAutoTurnKeepMax {
		t.Fatalf("超上限应收敛到 %d，实际 %d", handoffAutoTurnKeepMax, got)
	}
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "abc")
	if got := handoffAutoTurnKeep(); got != handoffAutoTurnKeepDefault {
		t.Fatalf("非法值应回落默认 %d，实际 %d", handoffAutoTurnKeepDefault, got)
	}

	// 关闭开关 → 不启用（调用方保持「携带全量历史」原行为）
	t.Setenv("PAIR_HANDOFF_AUTO_TURN", "0")
	if view, ok, _ := HandoffAutoTurnView(context.Background(), autoTurnLoop(hist), store,
		"conv_off", "继续"); ok || view != nil {
		t.Fatalf("PAIR_HANDOFF_AUTO_TURN=0 时不应启用: ok=%v view=%+v", ok, view)
	}
	// nil Loop 保护
	if _, ok, _ := HandoffAutoTurnView(context.Background(), nil, store, "conv_nil", "继续"); ok {
		t.Fatal("nil Loop 不应启用交接")
	}
}

// TestHandoffAutoTurn_ReuseAndKeepNormalize 增量小 → 复用交接文本；旧记录深度归一为续轮 Keep。
func TestHandoffAutoTurn_ReuseAndKeepNormalize(t *testing.T) {
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "0")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")    // 常规边界可触发（先生成「相关性档位」记录）
	t.Setenv("PAIR_HANDOFF_REFRESH_TOKENS", "100000") // 刷新阈值拉满 → 只验证复用（不重生成）
	store := NewMessageStore(t.TempDir())
	hist := handoffHist(20, "reuse")

	// ① 常规边界先生成一份记录（未指定 keep → 按相关性档位，默认 16）
	if _, ok := BuildHandoffView(context.Background(), nil, nil, store, "conv_reuse",
		hist, "继续", 0); !ok {
		t.Fatal("达阈值时常规边界应启用交接（前置断言失败）")
	}
	rec0, _ := store.LoadHandoff("conv_reuse")
	if rec0 == nil || rec0.Keep != handoffKeepRecentMsgs {
		t.Fatalf("前置记录 Keep 应为 %d，实际 %+v", handoffKeepRecentMsgs, rec0)
	}

	// ② 自主续轮边界：复用该记录（增量 < 刷新阈值），但保留段深度归一为续轮 Keep
	view, ok, _ := HandoffAutoTurnView(context.Background(), autoTurnLoop(hist), store, "conv_reuse", "继续")
	if !ok {
		t.Fatal("自主续轮边界应启用")
	}
	if len(view) != 1+handoffAutoTurnKeepDefault {
		t.Fatalf("归一后视图应为 %d 条，实际 %d 条", 1+handoffAutoTurnKeepDefault, len(view))
	}
	rec1, _ := store.LoadHandoff("conv_reuse")
	if rec1 == nil || rec1.Keep != handoffAutoTurnKeepDefault {
		t.Fatalf("复用记录深度应归一为 %d，实际 %+v", handoffAutoTurnKeepDefault, rec1)
	}
	if rec1.Text != rec0.Text {
		t.Fatal("增量未达刷新阈值 → 应复用同一交接文本（不重新生成）")
	}
	if rec1.CreatedAt != rec0.CreatedAt {
		t.Fatal("复用不应更新时间戳")
	}
}

// regFakeJSHandoffAuto 注册只带 onAutoTurn 的假 JS 交接实现（验证参数契约与委托语义）。
func regFakeJSHandoffAuto(t *testing.T, script string) {
	t.Helper()
	vm := goja.New()
	v, err := vm.RunString(script)
	if err != nil {
		t.Fatalf("构造 onAutoTurn 失败: %v", err)
	}
	fn, ok := goja.AssertFunction(v)
	if !ok {
		t.Fatal("onAutoTurn 不是函数")
	}
	restore := RegisterJSHandoff(&jsHandoffImpl{id: "fake-autoturn", vm: vm, onAutoTurn: fn})
	t.Cleanup(restore)
}

// TestHandoffAutoTurn_JSInterface 注册 onAutoTurn → 宿主委托 JS（参数契约正确）；抛错回退 Go 默认。
func TestHandoffAutoTurn_JSInterface(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "5")
	regFakeJSHandoffAuto(t, `(function(args){
		if (args.kind !== 'autoTurn') throw new Error('kind 应为 autoTurn，实际 ' + args.kind);
		if (args.force !== true) throw new Error('force 应为 true（强制折叠）');
		if (args.keep !== 5) throw new Error('keep 应为 5，实际 ' + args.keep);
		if (args.judge !== null) throw new Error('续轮边界 judge 必须为 null');
		if (!args.store || typeof args.store.loadRecord !== 'function') throw new Error('store 能力缺失');
		if (!args.history || !args.history.length) throw new Error('history 不应为空');
		return { view: [{role:'user',content:'JS 强制折叠视图'}], applied: true, notice: 'js-notice' };
	})`)
	store := NewMessageStore(t.TempDir())
	loop := autoTurnLoop(handoffHist(30, "jsauto"))

	view, ok, notice := HandoffAutoTurnView(context.Background(), loop, store, "conv_jsauto", "继续")
	if !ok || len(view) != 1 || view[0].Content != "JS 强制折叠视图" {
		t.Fatalf("应委托 JS 实现: ok=%v view=%+v", ok, view)
	}
	if notice != "js-notice" {
		t.Fatalf("notice 应透传 JS 返回值: %q", notice)
	}

	// JS 抛错 → 回退 Go 默认实现（强制折叠仍然生效，功能不丢）
	regFakeJSHandoffAuto(t, `(function(args){ throw new Error('boom-auto') })`)
	view2, ok2, _ := HandoffAutoTurnView(context.Background(), loop,
		NewMessageStore(t.TempDir()), "conv_jsauto2", "继续")
	if !ok2 || len(view2) == 0 {
		t.Fatal("JS 失败应回退 Go 默认实现（强制折叠仍生效）")
	}
	if !strings.HasPrefix(view2[0].Content, handoffTitle) {
		t.Fatalf("回退视图首条应为交接消息，实际 %q", truncRunesAgent(view2[0].Content, 40))
	}
}
