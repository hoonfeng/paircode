package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hoonfeng/paircode/goja"
)

// ─── JS 循环实例池（并行会话隔离）────────────────────────────
//
// 问题（2026-08-30 实测）：agentloop 核心外置后，Loop.Run 的整个循环在插件 VM
// 执行锁（goja Runtime.Lock）内执行——LLM 调用、工具执行全程持锁。JS 循环注册表
// 是单实例（一个插件一个 Runtime），于是并行会话被完全串行化：
//
//	会话 A 跑着 → 会话 B 的 Run 卡在 withLock → 前端 30s 超时（"新开对话必超时"）
//	AgentTeams 成员会话（ctx.agents 派生）同理无法并行工作。
//
// 修复：为并行 Run 提供「循环实例池」。每个实例 = 独立 goja Runtime + 独立 run
// 函数（影子实例），互不持锁；用完归还池复用。
//
// 影子实例的注册面（ctx.provide/on/registerSettings/tools.register/http.register/
// loopFactory.register/providerFactory.register/systemPrompt.section…）在 apply
// 期间全部 no-op（jsPluginAdapter.shadow=true → muteRegistrationAPIs），只捕获
// registerLoop 的循环实现——主实例已完成全部全局注册，影子只承载「跑循环」。
//
// 能力面（ctx.get/getSettings/logger/fs/…）保持可用：JS 循环运行期需要读配置。
//
// 上限：默认 24 个实例（环境变量 PAIR_JSLOOP_POOL_MAX 覆盖）。达上限的会话排队
// 等待空闲实例（ctx 取消即退出等待，不再永久阻塞）。

// defaultJSLoopPoolMax 循环实例上限（并行会话数上界；每实例 ≈ 一个 goja Runtime）。
const defaultJSLoopPoolMax = 24

// maxIdleShadowLoops 空闲影子实例保留数（超出的归还即丢弃，防内存长期占用）。
const maxIdleShadowLoops = 6

// errJSLoopPoolClosed 池已关闭（插件卸载/重装）。
var errJSLoopPoolClosed = errors.New("JS 循环实例池已关闭（插件已卸载或重装）")

// jsLoopPool 一个已注册 JS 循环实现的实例池。
type jsLoopPool struct {
	mu      sync.Mutex
	primary *jsLoopImpl   // 插件装载的主实例（不销毁）
	free    []*jsLoopImpl // 空闲实例栈（含 primary）
	live    int           // 已存在实例数（含 primary 与在用实例）
	max     int           // 实例上限
	waiters []chan *jsLoopImpl
	closed  bool
	spawnN  int // 已派生影子计数（日志/id 标识用）
}

// newJSLoopPool 建池。
// ★ 主实例（插件装载的那个 VM）不参与循环租借：循环一跑就是几分钟到
// 几十分钟的长持锁，而主实例 VM 还要承担短调用：循环装配器（CreateLoop →
// jsLoopFactoryBridge.Create 持主 VM 锁）、配置读写、事件回调、Provider 工厂等。
// 若主实例被某个会话的循环占满，新会话的 CreateLoop 依旧阻塞——所以
// 「长时循环」全部跑影子实例，主实例保留给短调用（只在派生失败时兜底）。
func newJSLoopPool(primary *jsLoopImpl) *jsLoopPool {
	max := defaultJSLoopPoolMax
	if v := strings.TrimSpace(os.Getenv("PAIR_JSLOOP_POOL_MAX")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			max = n
		}
	}
	p := &jsLoopPool{primary: primary, max: max, live: 0}
	if primary != nil {
		primary.pool = p
	}
	return p
}

// acquire 租借一个循环实例：空闲优先 → 未达上限则派生影子 → 否则排队等待。
func (p *jsLoopPool) acquire(ctx context.Context) (*jsLoopImpl, error) {
	if p == nil {
		return nil, errors.New("JS 循环实例池为空")
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errJSLoopPoolClosed
	}
	if n := len(p.free); n > 0 {
		impl := p.free[n-1]
		p.free = p.free[:n-1]
		p.mu.Unlock()
		return impl, nil
	}
	if p.live < p.max {
		p.live++
		p.spawnN++
		seq := p.spawnN
		p.mu.Unlock()
		impl, err := p.spawnShadow(seq)
		if err == nil {
			log.Printf("[loop-js] 已派生循环实例 #%d（并行会话隔离，实例数=%d/%d）", seq, p.liveCount(), p.max)
			return impl, nil
		}
		p.mu.Lock()
		p.live--
		none := p.live == 0
		p.mu.Unlock()
		log.Printf("[loop-js] 循环实例派生失败: %v", err)
		if none {
			// 没有任何可用实例 → 兜底用主实例（退化为旧行为，保可用性）
			log.Printf("[loop-js] 回落主循环实例执行（并行会话可能串行）")
			return p.primary, nil
		}
		return p.wait(ctx)
	}
	p.mu.Unlock()
	return p.wait(ctx)
}

// tryAcquire 非阻塞租借：有空闲实例或可派生时返回，否则返回 nil（不等待）。
// 供 delegate 嵌套使用——池满时调用方回落「复用父实例」，避免自死锁。
func (p *jsLoopPool) tryAcquire() *jsLoopImpl {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	if n := len(p.free); n > 0 {
		impl := p.free[n-1]
		p.free = p.free[:n-1]
		p.mu.Unlock()
		return impl
	}
	if p.live >= p.max {
		p.mu.Unlock()
		return nil
	}
	p.live++
	p.spawnN++
	seq := p.spawnN
	p.mu.Unlock()
	impl, err := p.spawnShadow(seq)
	if err != nil {
		p.mu.Lock()
		p.live--
		p.mu.Unlock()
		log.Printf("[loop-js] 影子循环实例派生失败（嵌套非阻塞路径）: %v", err)
		return nil
	}
	return impl
}

// liveCount 当前实例数（诊断用）。
func (p *jsLoopPool) liveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.live
}

// wait 排队等待空闲实例（ctx 取消即退出）。
func (p *jsLoopPool) wait(ctx context.Context) (*jsLoopImpl, error) {
	ch := make(chan *jsLoopImpl, 1)
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errJSLoopPoolClosed
	}
	if n := len(p.free); n > 0 { // 二次检查：等待登记前可能已有归还
		impl := p.free[n-1]
		p.free = p.free[:n-1]
		p.mu.Unlock()
		return impl, nil
	}
	p.waiters = append(p.waiters, ch)
	p.mu.Unlock()
	log.Printf("[loop-js] 循环实例已达上限 %d，会话排队等待空闲实例", p.max)
	select {
	case impl := <-ch:
		if impl == nil {
			return nil, errJSLoopPoolClosed
		}
		return impl, nil
	case <-ctx.Done():
		p.dropWaiter(ch)
		return nil, ctx.Err()
	}
}

// dropWaiter 撤销等待登记；若已被投递则把实例归还池。
func (p *jsLoopPool) dropWaiter(ch chan *jsLoopImpl) {
	p.mu.Lock()
	for i, w := range p.waiters {
		if w == ch {
			p.waiters = append(p.waiters[:i], p.waiters[i+1:]...)
			p.mu.Unlock()
			return
		}
	}
	p.mu.Unlock()
	// 已出队（实例可能已投递）→ 非阻塞取回并归还
	select {
	case impl := <-ch:
		if impl != nil {
			p.release(impl)
		}
	default:
	}
}

// release 归还实例：优先交给等待者 → 否则回空闲栈（影子空闲过多则丢弃）。
// 主实例不属于池（兜底路径才会用到），直接忽略。
func (p *jsLoopPool) release(impl *jsLoopImpl) {
	if p == nil || impl == nil {
		return
	}
	if impl == p.primary {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.live--
		p.mu.Unlock()
		disposeShadowLoop(impl)
		return
	}
	if len(p.waiters) > 0 {
		ch := p.waiters[0]
		p.waiters = p.waiters[1:]
		p.mu.Unlock()
		ch <- impl
		return
	}
	// 空闲影子过多 → 丢弃（primary 永不丢）
	idleShadows := 0
	for _, f := range p.free {
		if f.shadow {
			idleShadows++
		}
	}
	if impl.shadow && idleShadows >= maxIdleShadowLoops {
		p.live--
		p.mu.Unlock()
		disposeShadowLoop(impl)
		return
	}
	p.free = append(p.free, impl)
	p.mu.Unlock()
}

// close 关闭池（插件卸载/重装）：唤醒等待者并丢弃空闲影子。
func (p *jsLoopPool) close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	waiters := p.waiters
	p.waiters = nil
	free := p.free
	p.free = nil
	p.mu.Unlock()
	for _, ch := range waiters {
		ch <- nil // 唤醒：返回池已关闭
	}
	for _, impl := range free {
		disposeShadowLoop(impl)
	}
}

// disposeShadowLoop 回收影子实例的 JS 侧资源（primary 由插件卸载流程回收）。
func disposeShadowLoop(impl *jsLoopImpl) {
	if impl == nil || !impl.shadow || impl.plugin == nil {
		return
	}
	impl.plugin.cleanupJS()
}

// spawnShadow 派生一个影子循环实例：同源码 + 独立 Runtime + 注册面 no-op。
func (p *jsLoopPool) spawnShadow(seq int) (*jsLoopImpl, error) {
	src := p.primary
	if src == nil || src.plugin == nil {
		return nil, errors.New("主循环实例缺失，无法派生影子实例")
	}
	base := src.plugin
	host := base.host
	if host == nil || host.ctx == nil {
		return nil, errors.New("插件宿主上下文缺失，无法派生影子实例")
	}
	tag := fmt.Sprintf("#loop-shadow-%d", seq)
	def := cloneDefForShadowLoop(base.def, tag)
	if def == nil || strings.TrimSpace(def.code) == "" {
		return nil, errors.New("插件源码为空，无法派生影子实例")
	}

	vm, _ := newJSSandbox(def)
	obj, err := evalJSPlugin(vm, def.code, def.id)
	if err != nil {
		return nil, fmt.Errorf("影子实例求值失败: %w", err)
	}
	var applyFn goja.Callable
	if fn, ok := goja.AssertFunction(obj); ok {
		applyFn = fn
	} else {
		fn, ok := goja.AssertFunction(obj.Get("apply"))
		if !ok {
			return nil, errors.New("影子实例：插件缺少 apply 函数")
		}
		applyFn = fn
	}

	shadow := &jsPluginAdapter{
		host:       host,
		def:        def,
		vm:         vm,
		applyFn:    applyFn,
		handlers:   map[string]func(args any) (any, error){},
		handlersUI: map[string]func(args any) (any, error){},
		shadow:     true,
	}
	pc := host.ctx.forPlugin(base.def.name + tag)
	ctxObj, err := shadow.buildContextObject(pc)
	if err != nil {
		return nil, fmt.Errorf("影子实例 ctx 构建失败: %w", err)
	}
	configVal := goja.Undefined()
	if def.config != nil {
		configVal = vm.ToValue(def.config)
	}
	var applyErr error
	shadow.withLock(func() {
		applyErr = runJSWithTimeout(vm, jsApplyTimeout, func() error {
			_, e := applyFn(goja.Undefined(), ctxObj, configVal)
			return e
		})
	})
	if applyErr != nil {
		return nil, fmt.Errorf("影子实例 apply 失败: %v", jsErrorText(applyErr))
	}
	impl := shadow.shadowLoop
	if impl == nil {
		return nil, errors.New("影子实例未注册循环实现（apply 未调用 ctx.loopFactory.registerLoop）")
	}
	impl.shadow = true
	impl.pool = p
	return impl, nil
}

// cloneDefForShadowLoop 复制插件定义供影子实例使用：共享源码/配置内容，
// 但诊断（diag/console/status）独立——避免与主实例并发写同一 slice（无锁字段）。
func cloneDefForShadowLoop(d *jsPluginDef, tag string) *jsPluginDef {
	if d == nil {
		return nil
	}
	c := &jsPluginDef{
		id:        d.id + tag,
		pluginId:  d.pluginId,
		packageId: d.packageId,
		lang:      d.lang,
		name:      d.name + tag,
		purpose:   d.purpose,
		code:      d.code,
		version:   d.version,
		isFunc:    d.isFunc,
		scope:     d.scope,
		dir:       d.dir,
		createdAt: d.createdAt,
	}
	c.inject = append([]string(nil), d.inject...)
	if d.config != nil {
		cfg := make(map[string]any, len(d.config))
		for k, v := range d.config {
			cfg[k] = v
		}
		c.config = cfg
	}
	return c
}

// muteRegistrationAPIs 影子实例的注册面静音：只覆盖「已存在的注册方法」为 no-op，
// 能力面（读配置/日志/文件/网络…）保持不变。
// registerLoop 不在此列——由 attachLoopRegister 在 shadow 分支捕获循环实现。
func (p *jsPluginAdapter) muteRegistrationAPIs(ctxObj *goja.Object) {
	if ctxObj == nil {
		return
	}
	vm := p.vm
	noop := func(call goja.FunctionCall) goja.Value { return goja.Undefined() }
	noopVal := vm.ToValue(noop)

	// 顶层注册面（存在才覆盖）
	for _, key := range []string{"provide", "on", "registerSettings", "setSettings", "registerClientMethod"} {
		if v := ctxObj.Get(key); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
			_ = ctxObj.Set(key, noopVal)
		}
	}
	// 子对象注册方法
	mute := func(objKey string, methods ...string) {
		v := ctxObj.Get(objKey)
		if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
			return
		}
		obj, ok := v.(*goja.Object)
		if !ok {
			return
		}
		for _, m := range methods {
			if mv := obj.Get(m); mv != nil && !goja.IsUndefined(mv) && !goja.IsNull(mv) {
				_ = obj.Set(m, noopVal)
			}
		}
	}
	mute("tools", "register")
	mute("http", "register")
	mute("webServer", "register")
	mute("hooks", "register")
	mute("loopFactory", "register") // 装配器：主实例已注册
	mute("providerFactory", "register")
	mute("provider", "register")
	mute("systemPrompt", "section", "variable")
	mute("commands", "register")
	mute("skill", "register")
}
