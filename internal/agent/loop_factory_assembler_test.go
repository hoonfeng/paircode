package agent

// loop_factory_assembler_test.go — 装配器链与「子 agent 跳过装配」回归（2026-09-21 修复）
//
// 覆盖两处修复：
//  1. **装配器链**：多插件装配器按注册顺序**叠加**（旧「单槽位后注册覆盖」语义会让后装载
//     插件把先装载插件的装配参数整体吃掉——实测 autopilot 装载后 agentloop 的
//     systemAppend / 分段预算 / 审核模式全部失效）。
//  2. **子 agent 跳过装配**：LoopOpts.SkipPluginAssembly=true 时任何装配器都不参与。
//     子 agent 由插件 JS 回调栈内的宿主能力创建（decide → ctx.subagent.run），调用装配器
//     会取所属插件 VM 锁——正是当前持锁插件时即自死锁（实测会话永久卡死，见 subagent.go）。

import "testing"

func TestLoopAssemblerChain_OrderStackAndSkip(t *testing.T) {
	restoreA := RegisterLoopAssembler("test-asm-a", func(o LoopOpts) LoopOpts {
		o.System += "|A"
		o.StepBudget = 10
		return o
	})
	defer restoreA()
	restoreB := RegisterLoopAssembler("test-asm-b", func(o LoopOpts) LoopOpts {
		o.System += "|B"
		o.StepBudget += 5 // 基于 A 的结果 → 证明按注册顺序叠加
		return o
	})
	defer restoreB()

	if names := LoopAssemblerNames(); len(names) < 2 {
		t.Fatalf("装配器链 = %v，期望含 test-asm-a / test-asm-b", names)
	}

	h, err := CreateLoop(LoopOpts{System: "base"})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	l := h.Loop()
	if l.System != "base|A|B" {
		t.Fatalf("System = %q，期望 base|A|B（按注册顺序叠加，不被后者覆盖）", l.System)
	}
	if l.StepBudget != 15 {
		t.Fatalf("StepBudget = %d，期望 15（B 在 A 的结果上累加）", l.StepBudget)
	}

	// ★ 子 agent 路径：跳过装配器链（防插件回调栈内取同一把 VM 锁自死锁）
	h2, err := CreateLoop(LoopOpts{System: "raw", SkipPluginAssembly: true})
	if err != nil {
		t.Fatalf("CreateLoop(SkipPluginAssembly): %v", err)
	}
	if got := h2.Loop().System; got != "raw" {
		t.Fatalf("SkipPluginAssembly 后 System = %q，期望 raw（装配器不应参与）", got)
	}
	if got := h2.Loop().StepBudget; got != 0 {
		t.Fatalf("SkipPluginAssembly 后 StepBudget = %d，期望 0（未被装配器改写）", got)
	}

	// 还原：注册器移除后不再影响装配
	restoreA()
	restoreB()
	h3, err := CreateLoop(LoopOpts{System: "clean"})
	if err != nil {
		t.Fatalf("CreateLoop(还原后): %v", err)
	}
	if got := h3.Loop().System; got != "clean" {
		t.Fatalf("还原后 System = %q，期望 clean", got)
	}
}

func TestLoopAssembler_SameNameReplaces(t *testing.T) {
	restore := RegisterLoopAssembler("test-asm-dup", func(o LoopOpts) LoopOpts {
		o.System = "first"
		return o
	})
	defer restore()
	restore2 := RegisterLoopAssembler("test-asm-dup", func(o LoopOpts) LoopOpts {
		o.System = "second"
		return o
	})
	defer restore2()

	h, err := CreateLoop(LoopOpts{System: "base"})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	if got := h.Loop().System; got != "second" {
		t.Fatalf("同名装配器应替换（不叠加），System = %q，期望 second", got)
	}
}
