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
	h, err := CreateLoop(LoopOpts{MaxIterations: 11, System: "s"})
	if err != nil {
		t.Fatalf("CreateLoop(默认): %v", err)
	}
	loop := h.Loop()
	if loop.MaxIterations != 11 {
		t.Fatalf("默认工厂 MaxIterations = %d, want 11", loop.MaxIterations)
	}
	if loop.System != "s" {
		t.Fatalf("默认工厂 System = %q", loop.System)
	}
	_ = restoreDefault

	// 替换工厂：自定义 LoopFactory 生效
	replaced := &fakeLoopFactory{tag: "custom"}
	restore := ReplaceLoopFactory(replaced)
	defer restore()

	h2, err := CreateLoop(LoopOpts{MaxIterations: 22})
	if err != nil {
		t.Fatalf("CreateLoop(替换): %v", err)
	}
	l2 := h2.Loop()
	if l2.MaxIterations != 22 {
		t.Fatalf("自定义工厂 MaxIterations = %d, want 22", l2.MaxIterations)
	}
	if l2.System != "custom" {
		t.Fatalf("自定义工厂 System = %q, want custom", l2.System)
	}

	// restore 还原后走默认
	restore()
	h3, err := CreateLoop(LoopOpts{MaxIterations: 33})
	if err != nil {
		t.Fatalf("CreateLoop(还原): %v", err)
	}
	if h3.Loop().MaxIterations != 33 {
		t.Fatalf("还原后 MaxIterations = %d, want 33", h3.Loop().MaxIterations)
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
      maxIterations: 99,
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
	h, err := CreateLoop(LoopOpts{System: "base", MaxIterations: 5})
	if err != nil {
		t.Fatalf("CreateLoop(JS 装配): %v", err)
	}
	loop := h.Loop()
	if loop.MaxIterations != 99 {
		t.Fatalf("装配后 MaxIterations = %d, want 99", loop.MaxIterations)
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
	h2, err := CreateLoop(LoopOpts{MaxIterations: 7, System: "plain"})
	if err != nil {
		t.Fatalf("CreateLoop(还原后): %v", err)
	}
	if h2.Loop().MaxIterations != 7 || h2.Loop().System != "plain" {
		t.Fatalf("还原后装配参数残留: iters=%d system=%q", h2.Loop().MaxIterations, h2.Loop().System)
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

	h, err := CreateLoop(LoopOpts{System: "keep", MaxIterations: 3})
	if err != nil {
		t.Fatalf("CreateLoop: %v", err)
	}
	if h.Loop().System != "keep" || h.Loop().MaxIterations != 3 {
		t.Fatalf("null 装配器不应改动: system=%q iters=%d", h.Loop().System, h.Loop().MaxIterations)
	}
}

var _ = context.Background // 保留 context import（后续断言扩展）
