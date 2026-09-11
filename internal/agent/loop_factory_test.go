package agent

import (
	"context"
	"strings"
	"testing"
)

// ─── LoopFactory 单槽位 ────────────────────────────────────

func TestLoopFactoryDefaultAndReplace(t *testing.T) {
	// 默认工厂：CreateLoop 返回 *Loop 内核
	restoreDefault := func() {
		// 确保测试结束还原（防污染其他测试）
	}
	// ★ 2026-09-12：「最大迭代数」已移除，改用 ToolCallBudget 作为装配参数载体
	h, err := CreateLoop(LoopOpts{ToolCallBudget: 11, System: "s"})
	if err != nil {
		t.Fatalf("CreateLoop(默认): %v", err)
	}
	loop := h.Loop()
	if loop.ToolCallBudget != 11 {
		t.Fatalf("默认工厂 ToolCallBudget = %d, want 11", loop.ToolCallBudget)
	}
	if loop.System != "s" {
		t.Fatalf("默认工厂 System = %q", loop.System)
	}
	_ = restoreDefault

	// 替换工厂：自定义 LoopFactory 生效
	replaced := &fakeLoopFactory{tag: "custom"}
	restore := ReplaceLoopFactory(replaced)
	defer restore()

	h2, err := CreateLoop(LoopOpts{ToolCallBudget: 22})
	if err != nil {
		t.Fatalf("CreateLoop(替换): %v", err)
	}
	l2 := h2.Loop()
	if l2.ToolCallBudget != 22 {
		t.Fatalf("自定义工厂 ToolCallBudget = %d, want 22", l2.ToolCallBudget)
	}
	if l2.System != "custom" {
		t.Fatalf("自定义工厂 System = %q, want custom", l2.System)
	}

	// restore 还原后走默认
	restore()
	h3, err := CreateLoop(LoopOpts{ToolCallBudget: 33})
	if err != nil {
		t.Fatalf("CreateLoop(还原): %v", err)
	}
	if h3.Loop().ToolCallBudget != 33 {
		t.Fatalf("还原后 ToolCallBudget = %d, want 33", h3.Loop().ToolCallBudget)
	}
	if h3.Loop().System != "" {
		t.Fatalf("还原后 System = %q, want 空", h3.Loop().System)
	}
}

// fakeLoopFactory 测试用自定义工厂：修改 System 标记装配来源。
type fakeLoopFactory struct{ tag string }

func (f *fakeLoopFactory) Create(opts LoopOpts) (LoopHandle, error) {
	opts.System = f.tag
	return goLoopFactory{}.Create(opts)
}

// ─── JS 装配器（ctx.loopFactory.register） ─────────────────

const demoLoopAssemblerPlugin = `
return {
  name: 'loop-assembler',
  apply(ctx) {
    ctx.loopFactory.register((opts) => ({
      toolCallBudget: 99,
      maxContextTokens: 5000,
      autonomous: true,
      system: (opts.system || '') + '\n\n## 装配器追加规则\n- 由插件装配\n'
    }))
  }
}`

func TestJSLoopFactoryAssembler(t *testing.T) {
	// 前置：确保全局工厂是默认（防其他测试污染）
	restore := func() {}
	_ = restore
	// 若当前工厂已被占用则还原（防御）
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过避免污染", LoopFactoryNow())
	}

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(demoLoopAssemblerPlugin, "loop assembler demo")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	// 插件装载后：全局工厂应为 JS 桥（装配器已注册）
	if _, ok := LoopFactoryNow().(*jsLoopFactoryBridge); !ok {
		t.Fatalf("注册后工厂 = %T, want *jsLoopFactoryBridge", LoopFactoryNow())
	}

	// CreateLoop 走 JS 装配器：overrides 生效
	h, err := CreateLoop(LoopOpts{System: "base", ToolCallBudget: 5})
	if err != nil {
		t.Fatalf("CreateLoop(JS 装配): %v", err)
	}
	loop := h.Loop()
	if loop.ToolCallBudget != 99 {
		t.Fatalf("装配后 ToolCallBudget = %d, want 99", loop.ToolCallBudget)
	}
	if loop.MaxContextTokens != 5000 {
		t.Fatalf("装配后 MaxContextTokens = %d, want 5000", loop.MaxContextTokens)
	}
	if !loop.Autonomous {
		t.Fatalf("装配后 Autonomous = false, want true")
	}
	if !strings.Contains(loop.System, "base") || !strings.Contains(loop.System, "装配器追加规则") {
		t.Fatalf("装配后 System = %q, 应含 base + 追加规则", loop.System)
	}

	// 插件卸载 → 工厂还原默认
	if err := host.Unload("loop-assembler"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Fatalf("卸载后工厂 = %T, want goLoopFactory（还原默认）", LoopFactoryNow())
	}
	h2, err := CreateLoop(LoopOpts{ToolCallBudget: 7, System: "plain"})
	if err != nil {
		t.Fatalf("CreateLoop(还原后): %v", err)
	}
	if h2.Loop().ToolCallBudget != 7 || h2.Loop().System != "plain" {
		t.Fatalf("还原后装配参数残留: budget=%d system=%q", h2.Loop().ToolCallBudget, h2.Loop().System)
	}
}

// ─── 装配器返回 null：不改动 ────────────────────────────────

const demoLoopAssemblerBudgetPlugin = `
return {
  name: 'loop-assembler-budget',
  apply(ctx) {
    ctx.loopFactory.register((opts) => ({
      toolCallBudget: 7,
      maxToolBudgetSegments: 3
    }))
  }
}`

// TestJSLoopFactoryBudgetOverride 配置化闭环（2026-09-12）：插件注册的
// toolCallBudget / maxToolBudgetSegments 经 ctx.loopFactory.register 的 overrides
// 透传到 Loop；插件卸载后回到宿主装配参数（未配置 → 默认 120 次 / 20 段）。
func TestJSLoopFactoryBudgetOverride(t *testing.T) {
	if !gojaOk() {
		t.Skip("goja 不可用")
	}
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过避免污染", LoopFactoryNow())
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(demoLoopAssemblerBudgetPlugin, "loop assembler budget demo")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}

	// 宿主未配置（0）→ 装配器覆盖生效
	h, err := CreateLoop(LoopOpts{})
	if err != nil {
		t.Fatalf("CreateLoop(装配): %v", err)
	}
	loop := h.Loop()
	if got := loop.ToolCallBudgetOrDefault(); got != 7 {
		t.Errorf("装配后生效预算 = %d，期望 7", got)
	}
	if got := loop.MaxToolBudgetSegmentsOrDefault(); got != 3 {
		t.Errorf("装配后生效段数上限 = %d，期望 3", got)
	}

	// 卸载 → 回到宿主装配参数（未配置 0 → 默认 120 次 / 20 段）
	if err := host.Unload("loop-assembler-budget"); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	h2, err := CreateLoop(LoopOpts{})
	if err != nil {
		t.Fatalf("CreateLoop(还原): %v", err)
	}
	l2 := h2.Loop()
	if got := l2.ToolCallBudgetOrDefault(); got != DefaultToolCallBudget {
		t.Errorf("卸载后生效预算 = %d，期望默认 %d", got, DefaultToolCallBudget)
	}
	if got := l2.MaxToolBudgetSegmentsOrDefault(); got != MaxToolBudgetSegments {
		t.Errorf("卸载后生效段数上限 = %d，期望默认 %d", got, MaxToolBudgetSegments)
	}
}

// ─── 装配器返回 null：不改动 ────────────────────────────────

const demoLoopAssemblerNullPlugin = `
return {
  name: 'loop-assembler-null',
  apply(ctx) {
    ctx.loopFactory.register(() => null)
  }
}`

func TestJSLoopFactoryAssemblerNull(t *testing.T) {
	if _, ok := LoopFactoryNow().(goLoopFactory); !ok {
		t.Skipf("当前 LoopFactory 非默认（%T），跳过", LoopFactoryNow())
	}
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(demoLoopAssemblerNullPlugin, "loop assembler null")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	defer host.Unload("loop-assembler-null")

	h, err := CreateLoop(LoopOpts{System: "keep", ToolCallBudget: 3})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	if h.Loop().System != "keep" || h.Loop().ToolCallBudget != 3 {
		t.Fatalf("null 装配器不应改动: system=%q budget=%d", h.Loop().System, h.Loop().ToolCallBudget)
	}
}

var _ = context.Background // 保留 context import（后续断言扩展）
