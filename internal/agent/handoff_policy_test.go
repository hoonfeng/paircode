package agent

// handoff_policy_test.go — 整理「时机策略」插件化测试（2026-09-30）。
//
// 背景：此前「是否整理」由宿主内置 + 环境变量决定（PAIR_HANDOFF_SEGMENT /
// PAIR_HANDOFF_AUTO_TURN），插件只能提供**整理算法**——时机不可变。用户要求时机
// 也必须插件可变 → registerHandoff 新增 policy 回调：宿主三边界在调用整理**之前**
// 先问插件，未表态（null / 无 enabled 字段）才退回环境变量与内置默认。
//
// 覆盖：
//  1. 插件可**开启**宿主默认停用的边界（segment：enabled=true → 未设环境变量也整理）；
//  2. 插件可**停用**宿主默认启用的边界（userTurn：enabled=false → 达阈值也不整理）；
//  3. 未表态（null）→ 退回环境变量与内置默认（环境变量仍可兜底）；
//  4. 策略抛错 → 回退宿主默认（插件故障不改变行为）；
//  5. autoTurn 的保留深度（keep）与强制折叠可被策略覆写；
//  6. 运维总闸 PAIR_HANDOFF=0 优先于插件策略；
//  7. 未注册 policy 的老插件行为逐字不变（向后兼容）。

import (
	"context"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// regFakeJSHandoffPolicy 注册带时机策略的假 JS 实现（policy 必填，整理回调可选）。
// 用于验证「插件表态优先 / 未表态回退 / 执行失败回退」三条语义。
func regFakeJSHandoffPolicy(t *testing.T, policy, onUserTurn, onSegment, onAutoTurn string) *jsHandoffImpl {
	t.Helper()
	vm := goja.New()
	impl := &jsHandoffImpl{id: "fake-policy", vm: vm}
	bind := func(name, src string) goja.Callable {
		if src == "" {
			return nil
		}
		v, err := vm.RunString(src)
		if err != nil {
			t.Fatalf("构造 %s 失败: %v", name, err)
		}
		fn, ok := goja.AssertFunction(v)
		if !ok {
			t.Fatalf("%s 不是函数", name)
		}
		return fn
	}
	impl.policy = bind("policy", policy)
	impl.onUserTurn = bind("onUserTurn", onUserTurn)
	impl.onSegment = bind("onSegment", onSegment)
	impl.onAutoTurn = bind("onAutoTurn", onAutoTurn)
	restore := RegisterJSHandoff(impl)
	t.Cleanup(restore)
	return impl
}

// TestHandoffPolicy_PluginEnablesSegment 插件可开启宿主默认停用的段边界：
// 未设 PAIR_HANDOFF_SEGMENT（生产默认停用）时，policy 返回 enabled=true 即整理。
func TestHandoffPolicy_PluginEnablesSegment(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")                // 允许委托 JS
	t.Setenv("PAIR_HANDOFF_SEGMENT", "")           // 生产默认：停用
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100") // 阈值很小 → 整理内容本身可生成

	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "polseg"), MaxContextTokens: 0}

	// 前置：未注册策略时保持默认停用（对照组）
	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_seg_off", "继续"); ok || view != nil {
		t.Fatalf("未注册 policy 时段边界应保持默认停用，实际 ok=%v len=%d", ok, len(view))
	}

	// 只注册策略（不带 onSegment）→ 表态后整理走 Go 默认实现
	regFakeJSHandoffPolicy(t, `(function(args){
		if (args.kind !== 'segment') throw new Error('kind 应为 segment: ' + args.kind);
		if (args.defaults.segment !== false) throw new Error('defaults.segment 应为 false（宿主默认停用）');
		return { enabled: true, reason: '测试策略开启段边界' };
	})`, "", "", "")

	view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_seg_on", "继续")
	if !ok || len(view) == 0 {
		t.Fatalf("插件 policy 表态 enabled=true 后段边界应整理（时机插件可变），实际 ok=%v len=%d", ok, len(view))
	}
	if !strings.HasPrefix(view[0].Content, handoffTitle) {
		t.Fatalf("视图首条应为交接提交消息，实际 %q", truncRunesAgent(view[0].Content, 40))
	}
}

// TestHandoffPolicy_PluginDisablesUserTurn 插件可停用宿主默认启用的用户输入边界：
// 历史达阈值（宿主默认会整理）时，policy 返回 enabled=false → 不整理。
func TestHandoffPolicy_PluginDisablesUserTurn(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100") // 阈值很小 → 宿主默认会整理

	store := NewMessageStore(t.TempDir())
	hist := handoffHist(120, "poluser")

	// 前置：无策略时用户输入边界按阈值整理（对照组）
	pre, okPre, _ := HandoffUserTurnView(context.Background(), nil, nil, store,
		"conv_pol_u_pre", t.TempDir(), hist, "继续", 0)
	if !okPre || len(pre) == 0 {
		t.Fatal("前置断言失败：达阈值时用户输入边界应整理")
	}

	regFakeJSHandoffPolicy(t, `(function(args){
		if (args.kind !== 'userTurn') throw new Error('kind 应为 userTurn: ' + args.kind);
		return { enabled: false, reason: '测试策略停用' };
	})`, "", "", "")

	view, ok, _ := HandoffUserTurnView(context.Background(), nil, nil, store,
		"conv_pol_u_off", t.TempDir(), hist, "继续", 0)
	if ok || view != nil {
		t.Fatalf("插件 policy 表态 enabled=false 后用户输入边界不应整理，实际 ok=%v len=%d", ok, len(view))
	}
}

// TestHandoffPolicy_UnstatedFallsBack 插件未表态（返回 null / 无 enabled）→
// 退回环境变量与内置默认（环境变量仍能兜底）。
func TestHandoffPolicy_UnstatedFallsBack(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_SEGMENT", "") // 默认停用
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")

	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "polnone"), MaxContextTokens: 0}

	regFakeJSHandoffPolicy(t, `(function(args){ return null })`, "", "", "")
	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_none", "继续"); ok || view != nil {
		t.Fatalf("policy 未表态应退回内置默认（段边界停用），实际 ok=%v len=%d", ok, len(view))
	}

	// 环境变量兜底仍生效（插件未表态时）
	t.Setenv("PAIR_HANDOFF_SEGMENT", "1")
	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_env", "继续"); !ok || len(view) == 0 {
		t.Fatalf("policy 未表态时环境变量 PAIR_HANDOFF_SEGMENT=1 应仍生效，实际 ok=%v", ok)
	}
}

// TestHandoffPolicy_ErrorFallsBack 策略执行抛错 → 回退宿主默认（插件故障不改变行为）。
func TestHandoffPolicy_ErrorFallsBack(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_SEGMENT", "1") // 宿主默认：整理
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")

	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "polerr"), MaxContextTokens: 0}
	regFakeJSHandoffPolicy(t, `(function(args){ throw new Error('policy boom') })`, "", "", "")

	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_err", "继续"); !ok || len(view) == 0 {
		t.Fatalf("策略抛错应回退宿主默认（PAIR_HANDOFF_SEGMENT=1 → 整理），实际 ok=%v len=%d", ok, len(view))
	}
}

// TestHandoffPolicy_AutoTurnKeepOverride 续轮的保留深度可被插件策略覆写
// （enabled=true + force=true + keep=3 → 视图 = [交接] + 3 条原文）。
func TestHandoffPolicy_AutoTurnKeepOverride(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_AUTO_TURN", "") // 宿主默认停用 → 由插件策略开启
	t.Setenv("PAIR_HANDOFF_AUTO_KEEP", "")

	store := NewMessageStore(t.TempDir())
	hist := handoffHist(12, "polkeep")
	loop := autoTurnLoop(hist)

	// 前置：宿主默认停用（对照组）
	if view, ok, _ := HandoffAutoTurnView(context.Background(), loop, store, "conv_pol_keep_off", "继续"); ok || view != nil {
		t.Fatalf("宿主默认应停用自主续轮整理，实际 ok=%v", ok)
	}

	regFakeJSHandoffPolicy(t, `(function(args){
		if (args.kind !== 'autoTurn') throw new Error('kind 应为 autoTurn: ' + args.kind);
		if (Number(args.keepDefault) <= 0) throw new Error('keepDefault 应传入宿主默认保留深度');
		if (Number(args.historyMsgs) !== args.historyTail.length && args.historyMsgs <= 8) {
			throw new Error('historyMsgs 应为完整历史条数');
		}
		return { enabled: true, force: true, keep: 3 };
	})`, "", "", "")

	view, ok, _ := HandoffAutoTurnView(context.Background(), loop, store, "conv_pol_keep_on", "继续")
	if !ok {
		t.Fatal("插件 policy 表态 enabled=true 后自主续轮应整理")
	}
	if len(view) != 4 {
		t.Fatalf("keep=3 时视图应为 [交接]+3 条 = 4 条，实际 %d 条", len(view))
	}
	if !strings.HasPrefix(view[0].Content, handoffTitle) {
		t.Fatalf("视图首条应为交接提交消息，实际 %q", truncRunesAgent(view[0].Content, 40))
	}
}

// TestHandoffPolicy_MasterSwitchWins 运维总闸 PAIR_HANDOFF=0 优先于插件策略
// （排障/对照开关：插件要求整理也不得绕过）。
func TestHandoffPolicy_MasterSwitchWins(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "0") // 总闸：关闭交接
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_SEGMENT", "1")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")

	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "polgate"), MaxContextTokens: 0}
	regFakeJSHandoffPolicy(t, `(function(args){ return { enabled: true, force: true } })`, "", "", "")

	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_gate_s", "继续"); ok || view != nil {
		t.Fatalf("总闸 PAIR_HANDOFF=0 下段边界不得整理，实际 ok=%v len=%d", ok, len(view))
	}
	if view, ok, _ := HandoffUserTurnView(context.Background(), nil, nil, store, "conv_pol_gate_u",
		t.TempDir(), handoffHist(120, "polgateu"), "继续", 0); ok || view != nil {
		t.Fatalf("总闸 PAIR_HANDOFF=0 下用户输入边界不得整理，实际 ok=%v len=%d", ok, len(view))
	}
	if view, ok, _ := HandoffAutoTurnView(context.Background(), autoTurnLoop(handoffHist(12, "polgatea")),
		store, "conv_pol_gate_a", "继续"); ok || view != nil {
		t.Fatalf("总闸 PAIR_HANDOFF=0 下自主续轮不得整理，实际 ok=%v len=%d", ok, len(view))
	}
}

// TestHandoffPolicy_AbsentPolicyUnchanged 未注册 policy 的老插件行为逐字不变
// （段边界/自主续轮仍停用、用户输入边界仍按阈值整理）—— 向后兼容断言。
func TestHandoffPolicy_AbsentPolicyUnchanged(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	t.Setenv("PAIR_HANDOFF", "")
	t.Setenv("PAIR_HANDOFF_JS", "")
	t.Setenv("PAIR_HANDOFF_SEGMENT", "")
	t.Setenv("PAIR_HANDOFF_AUTO_TURN", "")
	t.Setenv("PAIR_HANDOFF_TRIGGER_TOKENS", "100")

	// 只注册整理实现（无 policy）：等价于 2026-09-27 之后的插件形态
	regFakeJSHandoff(t, `(function(args){ return { applied: false } })`, "")

	store := NewMessageStore(t.TempDir())
	loop := &Loop{History: handoffHist(120, "polabs"), MaxContextTokens: 0}
	if view, ok, _ := HandoffSegmentView(context.Background(), loop, store, "conv_pol_abs_seg", "继续"); ok || view != nil {
		t.Fatalf("未注册 policy 时段边界应保持默认停用，实际 ok=%v len=%d", ok, len(view))
	}
	if view, ok, _ := HandoffAutoTurnView(context.Background(), autoTurnLoop(handoffHist(12, "polabs2")),
		store, "conv_pol_abs_auto", "继续"); ok || view != nil {
		t.Fatalf("未注册 policy 时自主续轮应保持默认停用，实际 ok=%v len=%d", ok, len(view))
	}
	// 用户输入边界仍委托既有 JS 实现（applied:false → 不整理，行为未变）
	if view, ok, _ := HandoffUserTurnView(context.Background(), nil, nil, store, "conv_pol_abs_u",
		t.TempDir(), handoffHist(120, "polabsu"), "继续", 0); ok || view != nil {
		t.Fatalf("未注册 policy 时用户输入边界仍委托既有实现（applied:false），实际 ok=%v", ok)
	}
}
