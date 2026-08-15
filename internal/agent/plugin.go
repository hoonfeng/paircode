// ═══════════════════════════════════════════════════════════════
// plugin.go — 插件框架（对齐 deepseek-harness 的 Cordis 插件体系）
//
// harness 的「一切皆插件」：功能 = 插件 { name, inject, apply(ctx) }，
// apply(ctx) 里通过 ctx 服务（ctx.tools / ctx.systemPrompt / ctx.on /
// ctx.provide ...）贡献工具、提示词、服务与事件监听；启动时经
// cordis.patch.yml 装配进应用。
//
// 本文件提供 Go 侧等价物：
//   - Plugin / GoPlugin            → { name, apply(ctx) }
//   - PluginContext                → Cordis Context（服务注入 + 事件 + 生命周期）
//   - PluginHost                   → 插件装载器（Use/Load/Unload/List/Inspect）
//   - EventBus                     → ctx.on / ctx.emit
//   - PromptSection                → ctx.systemPrompt.section（系统提示片段）
//
// JS 动态插件（goja 沙箱）见 plugin_js.go；
// 模型可用的 cordis_* 工具（inspect/define/run/stop/undefine）见 plugin_tools.go。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// ─── EventBus ─────────────────────────────────────────────

// EventListener 事件监听器（ctx.on 注册的回调）。
type EventListener func(payload any)

// EventBus 进程内事件总线（对齐 Cordis ctx.on / ctx.emit）。
type EventBus struct {
	mu         sync.RWMutex
	listeners  map[string][]EventListener
	clientHook func(name string, payload any) // 可选：事件转发到浏览器（插件 client 半消费）
}

// NewEventBus 创建事件总线。
func NewEventBus() *EventBus {
	return &EventBus{listeners: map[string][]EventListener{}}
}

// SetClientHook 设置浏览器转发钩子（host→client 事件桥）。
// 由 PluginHost 在创建根上下文时注入：事件经此钩子进入 client 事件队列，
// 浏览器侧插件 client 半轮询 /api/plugins/client-events 消费。
func (b *EventBus) SetClientHook(fn func(name string, payload any)) {
	b.mu.Lock()
	b.clientHook = fn
	b.mu.Unlock()
}

// On 注册监听器，返回取消函数（插件停止时由 PluginContext 统一收集调用）。
func (b *EventBus) On(name string, fn EventListener) func() {
	b.mu.Lock()
	b.listeners[name] = append(b.listeners[name], fn)
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		ls := b.listeners[name]
		ptr := reflect.ValueOf(fn).Pointer()
		for i, l := range ls {
			if reflect.ValueOf(l).Pointer() == ptr {
				b.listeners[name] = append(ls[:i], ls[i+1:]...)
				break
			}
		}
	}
}

// Emit 广播事件（监听器在锁外调用，支持重入/并发）。
// 同时把事件经 clientHook 转发给浏览器（若设置且事件名以 ui:/client: 前缀开头）。
func (b *EventBus) Emit(name string, payload any) {
	b.mu.RLock()
	ls := append([]EventListener(nil), b.listeners[name]...)
	hook := b.clientHook
	b.mu.RUnlock()
	for _, l := range ls {
		l(payload)
	}
	if hook != nil && (strings.HasPrefix(name, "ui:") || strings.HasPrefix(name, "client:")) {
		hook(name, payload)
	}
}

// ListenerCount 指定事件的监听器数量。
func (b *EventBus) ListenerCount(name string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.listeners[name])
}

// EventNames 全部已注册事件名（排序）。
func (b *EventBus) EventNames() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	names := make([]string, 0, len(b.listeners))
	for n := range b.listeners {
		if len(b.listeners[n]) > 0 {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// ─── PromptSection ────────────────────────────────────────

// PromptSection 系统提示片段（对齐 Cordis ctx.systemPrompt.section）。
type PromptSection struct {
	Name  string // 唯一标识
	Order int    // 越小越靠前（默认 100）
	Text  string
}

// PromptVariable 提示词变量（对齐 harness ctx.systemPrompt.variable）：
// 段文本中 {{name}} 引用，组装时求值；Provider 返回 "" 表示本次无值（替换为空）。
type PromptVariable struct {
	Name     string
	Provider func() string
}

// ─── Plugin ───────────────────────────────────────────────

// Plugin 对齐 Cordis 插件形态：{ name, apply(ctx) }。
// apply 中通过 ctx 贡献工具/系统提示/服务/事件监听；返回 error 表示启动失败。
type Plugin interface {
	Name() string
	Apply(ctx *PluginContext) error
}

// GoPlugin 结构体插件（Go 侧最简形态）。
type GoPlugin struct {
	NameField string
	ApplyFn   func(ctx *PluginContext) error
}

// Name 实现 Plugin。
func (p *GoPlugin) Name() string { return p.NameField }

// Apply 实现 Plugin。
func (p *GoPlugin) Apply(ctx *PluginContext) error {
	if p.ApplyFn == nil {
		return fmt.Errorf("plugin %q: ApplyFn 为空", p.NameField)
	}
	return p.ApplyFn(ctx)
}

// ─── PluginContext ────────────────────────────────────────

// PluginContext 对齐 Cordis Context：插件可访问的服务与生命周期 API。
// 语义对齐：
//   - ctx.Get / ctx.Provide：动态服务注入（provider/injector 配对）
//   - ctx.On / ctx.Emit：事件（EventBus 透传）
//   - ctx.Effect：注册清理回调，插件停止（Unload）时统一执行
//   - ctx.Tools.Register / ctx.RegisterTool：注册工具
//   - ctx.AddSystemPromptSection：贡献系统提示片段
//
// 每个加载中的插件获得一个带归属（plugin 字段）的上下文：注册的工具/
// 片段/监听器/清理回调都记在该插件名下，Unload 时自动回收。
type PluginContext struct {
	host   *PluginHost
	plugin string // 归属插件名（空=根上下文，仅宿主内部使用）

	// 静态服务（跨插件共享，对齐 ctx.tools / ctx.session 等已知键）
	Tools         *Registry
	Events        *EventBus
	Store         ConversationStore
	WorkspaceRoot string

	// 动态服务（ctx.provide / ctx.get，跨插件共享）
	services   map[string]any
	servicesMu sync.RWMutex

	// 本插件生命周期资源（Unload 时回收）
	effects   []func()
	listeners []func() // On 返回的 cancel，统一取消
}

// forPlugin 返回绑定到指定插件名的上下文（共享 services/Tools/Events）。
func (c *PluginContext) forPlugin(name string) *PluginContext {
	return &PluginContext{
		host:          c.host,
		plugin:        name,
		Tools:         c.Tools,
		Events:        c.Events,
		Store:         c.Store,
		WorkspaceRoot: c.WorkspaceRoot,
		services:      c.services,
	}
}

// PluginName 当前插件名。
func (c *PluginContext) PluginName() string { return c.plugin }

// Get 取动态服务（ctx.get；不存在返回 nil）。
func (c *PluginContext) Get(name string) any {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()
	return c.services[name]
}

// Provide 注册动态服务（ctx.provide），返回撤销函数。
func (c *PluginContext) Provide(name string, v any) func() {
	c.servicesMu.Lock()
	c.services[name] = v
	c.servicesMu.Unlock()
	return func() {
		c.servicesMu.Lock()
		delete(c.services, name)
		c.servicesMu.Unlock()
	}
}

// On 监听事件（ctx.on），返回取消函数并自动记入本插件生命周期。
func (c *PluginContext) On(name string, fn EventListener) func() {
	cancel := c.Events.On(name, fn)
	c.listeners = append(c.listeners, cancel)
	return cancel
}

// Emit 广播事件（ctx.emit）。
func (c *PluginContext) Emit(name string, payload any) { c.Events.Emit(name, payload) }

// Effect 注册清理回调（ctx.effect；插件停止时执行）。
func (c *PluginContext) Effect(fn func()) {
	c.effects = append(c.effects, fn)
}

// RegisterTool 注册工具（ctx.tools.register），记入本插件名下以便 Unload 回收。
// ★ 同名冲突检测（P2）：插件不能静默覆盖宿主内置工具或其他插件的工具；
//   冲突返回明确错误（含占用方与处理建议）。
func (c *PluginContext) RegisterTool(t *Tool) error {
	if t == nil || t.Name == "" {
		return fmt.Errorf("工具名为空")
	}
	if err := c.host.claimTool(c.plugin, t.Name); err != nil {
		return err
	}
	c.host.addPluginTool(c.plugin, t.Name)
	c.Tools.Register(t)
	return nil
}

// claimTool 登记工具归属：同名工具已被其他插件/宿主占用 → 报错（防静默覆盖）。
// 宿主内置工具（Registry 已有但无插件归属）视为宿主占用。
func (h *PluginHost) claimTool(plugin, toolName string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	owner, taken := h.toolOwner[toolName]
	if taken && owner != plugin {
		return fmt.Errorf("工具 %q 已被插件 %s 注册，插件 %s 不能覆盖。请换工具名，或先 cordis_stop %s 再注册",
			toolName, owner, plugin, owner)
	}
	if !taken {
		if _, exists := h.ctx.Tools.Get(toolName); exists {
			return fmt.Errorf("工具 %q 是宿主内置/已占用工具，插件 %s 不能覆盖。请换工具名", toolName, plugin)
		}
	}
	h.toolOwner[toolName] = plugin
	return nil
}

// AddSystemPromptSection 贡献系统提示片段（ctx.systemPrompt.section）。
func (c *PluginContext) AddSystemPromptSection(s *PromptSection) {
	if s == nil {
		return
	}
	c.host.addPluginSection(c.plugin, s)
}

// AddSystemPromptVariable 注册提示词变量（ctx.systemPrompt.variable；{{name}} 组装时求值）。
func (c *PluginContext) AddSystemPromptVariable(v *PromptVariable) {
	if v == nil || strings.TrimSpace(v.Name) == "" {
		return
	}
	c.host.addPluginVariable(c.plugin, v)
}

// cleanup 执行本插件的全部清理回调（Unload 时由 PluginHost 调用）。
func (c *PluginContext) cleanup() {
	for _, l := range c.listeners {
		l()
	}
	c.listeners = nil
	for i := len(c.effects) - 1; i >= 0; i-- {
		func() {
			defer func() { _ = recover() }()
			c.effects[i]()
		}()
	}
	c.effects = nil
}

// ─── PluginState / PluginRecord ──────────────────────────

// PluginState 插件运行状态。
type PluginState int

// PluginState 取值。
const (
	PluginStopped PluginState = iota
	PluginRunning
)

func (s PluginState) String() string {
	if s == PluginRunning {
		return "running"
	}
	return "stopped"
}

// PluginSource 插件来源。
type PluginSource string

// PluginSource 取值。
const (
	PluginSourceGo PluginSource = "go"
	PluginSourceJS PluginSource = "js"
)

// PluginRecord 插件记录（cordis_inspect 报告用）。
type PluginRecord struct {
	Name       string       `json:"name"`
	Source     PluginSource `json:"source"`
	State      string       `json:"state"`
	Provides   []string     `json:"provides,omitempty"`
	Tools      []string     `json:"tools,omitempty"`
	Sections   []string     `json:"sections,omitempty"`
	Version    string       `json:"version,omitempty"`
	Purpose    string       `json:"purpose,omitempty"`     // 用途说明（JS 动态插件）
	HasClient  bool         `json:"hasClient,omitempty"`   // 是否有 client 半（浏览器端）
	ClientCode string       `json:"clientCode,omitempty"`  // client 半源码（供浏览器装载；列表接口可能省略）
	DefID      string       `json:"defId,omitempty"`       // JS 动态插件定义 id（dyn-<n>）
}

// ─── PluginHost ───────────────────────────────────────────

// PluginHost 插件宿主：装载/启动/停止/检查（对齐 Cordis Loader）。
type PluginHost struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	states  map[string]PluginState
	sources map[string]PluginSource
	order   []string
	ctx     *PluginContext

	// JS 动态插件定义（cordis_define 登记，cordis_run 装载）
	defs map[string]*jsPluginDef

	// 插件贡献回收表
	pluginTools    map[string][]string
	pluginSections map[string][]*PromptSection
	pluginVars     map[string][]*PromptVariable
	contexts       map[string]*PluginContext // 每插件 apply 时的上下文（Unload 时 cleanup）

	// 工具名 → 归属插件（同名冲突检测：插件不能静默覆盖宿主/他人工具）
	toolOwner map[string]string

	// host→client 事件桥（浏览器插件 client 半消费）
	clientMu       sync.Mutex
	clientEvents   []ClientEvent
	clientEventSeq int64 // 下一条事件的全局序号

	// 浏览器 client 半运行时上报快照（client inspect provider 的数据源）
	clientStateMu sync.RWMutex
	clientState   ClientRuntimeSnapshot

	// 工具集模板注册表（toolset_build 动态组合；模板本身插件化，可被市场/用户扩展）
	templatesMu sync.RWMutex
	templates   map[string]*ToolsetTemplate
}

// NewPluginHost 创建插件宿主。
// registry：工具注册表（ctx.tools）；store：会话存储（ctx.session）；root：工作区根。
func NewPluginHost(registry *Registry, store ConversationStore, root string) *PluginHost {
	h := &PluginHost{
		plugins:        map[string]Plugin{},
		states:         map[string]PluginState{},
		sources:        map[string]PluginSource{},
		defs:           map[string]*jsPluginDef{},
		pluginTools:    map[string][]string{},
		pluginSections: map[string][]*PromptSection{},
		pluginVars:     map[string][]*PromptVariable{},
		toolOwner:      map[string]string{},
		templates:      map[string]*ToolsetTemplate{},
	}
	h.ctx = &PluginContext{
		host:          h,
		Tools:         registry,
		Events:        NewEventBus(),
		Store:         store,
		WorkspaceRoot: root,
		services:      map[string]any{},
	}
	h.contexts = map[string]*PluginContext{}
	// host→client 事件桥：ui:/client: 前缀事件自动进入 client 事件队列
	h.ctx.Events.SetClientHook(func(name string, payload any) { h.PushClientEvent(name, payload) })
	return h
}

// ─── host→client 事件桥 ──────────────────────────────────

// 全局 PluginHost 引用（市场安装插件/工具集装载用；web 与 AgentBase 初始化时设置）。
var (
	globalPHMu sync.RWMutex
	globalPH   *PluginHost
)

// SetGlobalPluginHost 设置全局插件宿主（web_server / AgentBase.Init 调用）。
func SetGlobalPluginHost(ph *PluginHost) {
	globalPHMu.Lock()
	globalPH = ph
	globalPHMu.Unlock()
}

// GetGlobalPluginHost 取全局插件宿主（未设置返回 nil）。
func GetGlobalPluginHost() *PluginHost {
	globalPHMu.RLock()
	defer globalPHMu.RUnlock()
	return globalPH
}

// ClientEvent 一条转发给浏览器的插件事件（seq 单调递增，轮询游标用）。
type ClientEvent struct {
	Seq     int64  `json:"seq"`
	Name    string `json:"name"`
	Payload any    `json:"payload,omitempty"`
}

// PushClientEvent 入队一条 client 事件（上限 500 条，超限丢弃最旧）。
func (h *PluginHost) PushClientEvent(name string, payload any) {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()
	h.clientEventSeq++
	ev := ClientEvent{Seq: h.clientEventSeq, Name: name, Payload: payload}
	if len(h.clientEvents) >= 500 {
		h.clientEvents = append(h.clientEvents[1:], ev)
	} else {
		h.clientEvents = append(h.clientEvents, ev)
	}
}

// ClientEventsSince 返回 seq 之后（不含 seq）的全部 client 事件。
// seq<=0 表示从最早一条开始（首次轮询）。返回的 LastSeq 供下次轮询使用。
func (h *PluginHost) ClientEventsSince(seq int64) ([]ClientEvent, int64) {
	h.clientMu.Lock()
	defer h.clientMu.Unlock()
	out := make([]ClientEvent, 0, 8)
	for _, ev := range h.clientEvents {
		if ev.Seq > seq {
			out = append(out, ev)
		}
	}
	if len(h.clientEvents) > 0 {
		return out, h.clientEvents[len(h.clientEvents)-1].Seq
	}
	return out, 0
}

// ClientPluginSnapshot 浏览器侧一个 client 半的运行状态（浏览器 plugin-runtime 上报）。
type ClientPluginSnapshot struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`             // loaded | error
	Panels  []string `json:"panels,omitempty"`   // 注册的面板 id
	Events  []string `json:"events,omitempty"`   // 监听的事件名（ui.on）
	Version string   `json:"version,omitempty"`  // client 半版本（同 host 定义版本）
	Error   string   `json:"error,omitempty"`    // 装载失败原因
}

// ClientRuntimeSnapshot 浏览器 client 半运行时整体快照（浏览器周期上报）。
type ClientRuntimeSnapshot struct {
	Connected  bool                   `json:"connected"`   // 页面在线且上报过
	ReportedAt int64                  `json:"reportedAt"`  // 本次上报时间（Unix 秒）
	Plugins    []ClientPluginSnapshot `json:"plugins"`
	Panels     []string               `json:"panels,omitempty"` // 全部已注册面板 id 汇总
}

// SetClientState 浏览器上报 client 半运行时快照。
func (h *PluginHost) SetClientState(snap ClientRuntimeSnapshot) {
	snap.Connected = true
	snap.ReportedAt = time.Now().Unix()
	h.clientStateMu.Lock()
	h.clientState = snap
	h.clientStateMu.Unlock()
}

// ClientState 读取浏览器上报快照；超过 clientStateTTL 未上报视为离线。
func (h *PluginHost) ClientState() ClientRuntimeSnapshot {
	h.clientStateMu.RLock()
	snap := h.clientState
	h.clientStateMu.RUnlock()
	if snap.ReportedAt > 0 && time.Now().Unix()-snap.ReportedAt > clientStateTTL {
		snap.Connected = false
	}
	return snap
}

// clientStateTTL 快照过期秒数（页面关闭/断线后视为离线）。
const clientStateTTL = 30

// EmitHostEvent 由外部（浏览器 client→host 事件桥）把事件发回 EventBus 广播。
// 浏览器侧约定：client→host 事件用 "host:" 前缀（不会被 ui:/client: 转发规则
// 环回浏览器）；host 插件用 ctx.on('host:xxx') 消费。
func (h *PluginHost) EmitHostEvent(name string, payload any) {
	h.ctx.Events.Emit(name, payload)
}

// Context 返回宿主根上下文（服务共享）。
func (h *PluginHost) Context() *PluginContext { return h.ctx }

// ─── 工具集模板注册表（toolset_build 动态组合的数据源）──────

// RegisterTemplate 注册一个工具集构建模板（插件化：模板可由任意插件提供，
// 内置 toolset-tpl-core 提供通用模板，市场/用户插件可注册专属模板）。
func (h *PluginHost) RegisterTemplate(t *ToolsetTemplate) error {
	if t == nil || t.ID == "" {
		return fmt.Errorf("模板 id 不能为空")
	}
	h.templatesMu.Lock()
	if h.templates == nil {
		h.templates = map[string]*ToolsetTemplate{}
	}
	h.templates[t.ID] = t
	h.templatesMu.Unlock()
	return nil
}

// Templates 返回全部已注册模板（按 id 排序）。
func (h *PluginHost) Templates() []*ToolsetTemplate {
	h.templatesMu.RLock()
	defer h.templatesMu.RUnlock()
	out := make([]*ToolsetTemplate, 0, len(h.templates))
	for _, t := range h.templates {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Template 按 id 取模板。
func (h *PluginHost) Template(id string) *ToolsetTemplate {
	h.templatesMu.RLock()
	defer h.templatesMu.RUnlock()
	return h.templates[id]
}

// RemoveTemplate 移除模板（插件卸载时清理）。
func (h *PluginHost) RemoveTemplate(id string) {
	h.templatesMu.Lock()
	delete(h.templates, id)
	h.templatesMu.Unlock()
}

// EventBus 返回共享事件总线。
func (h *PluginHost) EventBus() *EventBus { return h.ctx.Events }

// Use 注册并启动插件（同名报错）。
func (h *PluginHost) Use(p Plugin) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin: name 不能为空")
	}
	h.mu.Lock()
	if _, dup := h.plugins[name]; dup {
		h.mu.Unlock()
		return fmt.Errorf("plugin: 已存在同名插件 %q", name)
	}
	h.plugins[name] = p
	h.sources[name] = PluginSourceGo
	h.order = append(h.order, name)
	h.mu.Unlock()
	return h.Load(name)
}

// Register 注册但不启动（供 cordis_define/run 分两步使用）。
// 返回的 Load 由调用方触发。
func (h *PluginHost) Register(p Plugin, src PluginSource) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin: name 不能为空")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, dup := h.plugins[name]; dup {
		return fmt.Errorf("plugin: 已存在同名插件 %q", name)
	}
	h.plugins[name] = p
	h.sources[name] = src
	h.order = append(h.order, name)
	return nil
}

// Load 启动插件（未注册报错；已运行 no-op）。
func (h *PluginHost) Load(name string) error {
	h.mu.Lock()
	p, ok := h.plugins[name]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("plugin: 未注册 %q", name)
	}
	if h.states[name] == PluginRunning {
		h.mu.Unlock()
		return nil
	}
	h.states[name] = PluginRunning
	h.mu.Unlock()

	pc := h.ctx.forPlugin(name)
	h.mu.Lock()
	h.contexts[name] = pc
	h.mu.Unlock()
	if err := p.Apply(pc); err != nil {
		h.mu.Lock()
		h.states[name] = PluginStopped
		delete(h.contexts, name)
		h.mu.Unlock()
		pc.cleanup()
		return fmt.Errorf("plugin %q apply 失败: %w", name, err)
	}
	return nil
}

// Reload 重新装载插件（先停止回收贡献，再重新 apply；未运行则直接装载）。
// JS 动态插件会按其定义重放（host 半代码重新求值执行）。
func (h *PluginHost) Reload(name string) error {
	h.mu.Lock()
	p, ok := h.plugins[name]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("plugin: 未注册 %q", name)
	}
	wasRunning := h.states[name] == PluginRunning
	h.mu.Unlock()
	if wasRunning {
		if err := h.Unload(name); err != nil {
			return err
		}
	}
	// JS 动态插件：先从注册表移除旧 adapter（Unload 保留 plugins 条目，
	// 而 LoadJSDynamic 会重新 Register，同名会冲突），再按定义重放。
	if adapter, ok := p.(*jsPluginAdapter); ok {
		h.mu.Lock()
		delete(h.plugins, name)
		delete(h.sources, name)
		for i, n := range h.order {
			if n == name {
				h.order = append(h.order[:i], h.order[i+1:]...)
				break
			}
		}
		h.mu.Unlock()
		return h.LoadJSDynamic(adapter.def)
	}
	return h.Load(name)
}

// ResolvePluginName 把 dyn id 或插件名解析为插件名（浏览器 REST 用）。
func (h *PluginHost) ResolvePluginName(idOrName string) (string, error) {
	return h.resolvePluginName(idOrName)
}

// Unload 停止插件并回收其全部贡献（工具/片段/监听器/清理回调）。
func (h *PluginHost) Unload(name string) error {
	h.mu.Lock()
	_, ok := h.plugins[name]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("plugin: 未注册 %q", name)
	}
	if h.states[name] != PluginRunning {
		h.mu.Unlock()
		return nil
	}
	h.states[name] = PluginStopped
	h.mu.Unlock()

	// 回收贡献
	h.mu.Lock()
	for _, tn := range h.pluginTools[name] {
		if h.toolOwner[tn] == name {
			delete(h.toolOwner, tn) // 释放工具归属
		}
		h.ctx.Tools.Unregister(tn)
	}
	delete(h.pluginTools, name)
	delete(h.pluginSections, name)
	delete(h.pluginVars, name)
	pc := h.contexts[name]
	delete(h.contexts, name)
	h.mu.Unlock()
	if pc != nil {
		pc.cleanup() // 触发 effects + 取消 listeners
	}
	return nil
}

// Undefine 删除插件定义（先停止，再忘掉）。
func (h *PluginHost) Undefine(name string) error {
	h.mu.Lock()
	if _, ok := h.defs[name]; !ok {
		if _, ok := h.plugins[name]; !ok {
			h.mu.Unlock()
			return fmt.Errorf("plugin: 未定义 %q", name)
		}
	}
	h.mu.Unlock()
	_ = h.Unload(name)
	h.mu.Lock()
	delete(h.defs, name)
	delete(h.plugins, name)
	delete(h.sources, name)
	for i, n := range h.order {
		if n == name {
			h.order = append(h.order[:i], h.order[i+1:]...)
			break
		}
	}
	h.mu.Unlock()
	return nil
}

// Get 取插件（未注册返回 nil,false）。
func (h *PluginHost) Get(name string) (Plugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[name]
	return p, ok
}

// State 返回插件状态。
func (h *PluginHost) State(name string) PluginState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.states[name]
}

// List 全部插件名（注册顺序）。
func (h *PluginHost) List() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]string(nil), h.order...)
}

// Inspect 全部插件记录（cordis_inspect 报告）。
func (h *PluginHost) Inspect() []PluginRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()
	recs := make([]PluginRecord, 0, len(h.order))
	for _, name := range h.order {
		rec := PluginRecord{
			Name:   name,
			Source: h.sources[name],
			State:  h.states[name].String(),
		}
		if h.sources[name] == PluginSourceJS {
			if d := h.defByNameLocked(name); d != nil {
				rec.Version = d.version
				rec.Provides = d.provides
				rec.Purpose = d.purpose
				rec.HasClient = strings.TrimSpace(d.clientCode) != ""
				rec.ClientCode = d.clientCode
				rec.DefID = d.id
			}
		}
		rec.Tools = append([]string(nil), h.pluginTools[name]...)
		for _, s := range h.pluginSections[name] {
			rec.Sections = append(rec.Sections, s.Name)
		}
		recs = append(recs, rec)
	}
	return recs
}

// InspectDetail 单个插件详情（含 client 半源码；不存在返回 nil）。
func (h *PluginHost) InspectDetail(name string) *PluginRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, n := range h.order {
		if n != name {
			continue
		}
		rec := PluginRecord{
			Name:   n,
			Source: h.sources[n],
			State:  h.states[n].String(),
		}
		if h.sources[n] == PluginSourceJS {
			if d := h.defByNameLocked(n); d != nil {
				rec.Version = d.version
				rec.Provides = d.provides
				rec.Purpose = d.purpose
				rec.HasClient = strings.TrimSpace(d.clientCode) != ""
				rec.ClientCode = d.clientCode
				rec.DefID = d.id
			}
		}
		rec.Tools = append([]string(nil), h.pluginTools[n]...)
		for _, s := range h.pluginSections[n] {
			rec.Sections = append(rec.Sections, s.Name)
		}
		return &rec
	}
	return nil
}

// defByNameLocked 按插件名找 JS 定义（defs 按 id 存储，需要遍历匹配 name）。
// ★ 调用方必须已持有 h.mu 读锁（避免 RWMutex 递归读锁死锁）。
func (h *PluginHost) defByNameLocked(name string) *jsPluginDef {
	for _, d := range h.defs {
		if d.name == name {
			return d
		}
	}
	return nil
}

// Sections 全部插件贡献的系统提示片段（按 Order 排序；供系统提示组装）。
func (h *PluginHost) Sections() []*PromptSection {
	h.mu.RLock()
	var all []*PromptSection
	for _, secs := range h.pluginSections {
		all = append(all, secs...)
	}
	h.mu.RUnlock()
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Order != all[j].Order {
			return all[i].Order < all[j].Order
		}
		return all[i].Name < all[j].Name
	})
	return all
}

// Variables 全部插件注册的提示词变量（组装时求值；对齐 harness systemPrompt.variable）。
func (h *PluginHost) Variables() []*PromptVariable {
	h.mu.RLock()
	var all []*PromptVariable
	for _, vs := range h.pluginVars {
		all = append(all, vs...)
	}
	h.mu.RUnlock()
	return all
}

func (h *PluginHost) addPluginVariable(plugin string, v *PromptVariable) {
	h.mu.Lock()
	h.pluginVars[plugin] = append(h.pluginVars[plugin], v)
	h.mu.Unlock()
}

func (h *PluginHost) addPluginTool(plugin, tool string) {
	h.mu.Lock()
	h.pluginTools[plugin] = append(h.pluginTools[plugin], tool)
	h.mu.Unlock()
}

func (h *PluginHost) addPluginSection(plugin string, s *PromptSection) {
	h.mu.Lock()
	h.pluginSections[plugin] = append(h.pluginSections[plugin], s)
	h.mu.Unlock()
}
