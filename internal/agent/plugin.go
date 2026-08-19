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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
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
//
//	冲突返回明确错误（含占用方与处理建议）。
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
		if t, exists := h.ctx.Tools.Get(toolName); exists {
			// ★ 插件接管宿主内置工具（2026-08-16 迁移）：原 Go 实现存档到
			//   hostExecutors（供 ctx.hostTool 调用），Registry 交给插件工具
			//   （同名覆盖）。对齐 harness「工具编排在插件、能力在宿主 seam」。
			ArchiveHostTool(t)
		}
	}
	h.toolOwner[toolName] = plugin
	return nil
}

// IsPluginTool 判断工具是否由插件注册（宿主工具豁免 harness 对齐过滤——
// 插件注册的工具是「内容」而非 pair 独有编码能力）。
func (h *PluginHost) IsPluginTool(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, taken := h.toolOwner[name]
	return taken
}

// MergePluginTools 把插件宿主中由插件注册的业务工具合并进目标注册表
// （会话级 reg 独立新建，需同步插件工具——Node 桥工具 + goja 插件
// ctx.tools.register）。★ 同名接管（2026-08-16 迁移）：插件工具覆盖会话
// 内置工具——磁盘工具插件（tool-*）注册的同名工具接管 agent 可见面
// （宿主 Go 实现已存档 hostExecutors，经 ctx.hostTool 调用）。
func MergePluginTools(reg *Registry, ph *PluginHost) {
	if ph == nil {
		return
	}
	for _, meta := range ph.Context().Tools.AllToolMeta() {
		if !ph.IsPluginTool(meta.Name) {
			continue
		}
		t, ok := ph.Context().Tools.Get(meta.Name)
		if !ok {
			continue
		}
		// 插件工具接管：覆盖会话内置实现，且默认启用（插件是「内容」，
		// 豁免 harness 过滤；用户装载插件即期望工具可用）。
		t.Enabled = true
		reg.Register(t) // 同名覆盖（插件优先）
	}
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

// PluginState 插件运行状态（对齐 harness CordisRunStatus 的进程内简化 6 态）：
//
//	running   正在运行（apply 已执行）
//	stopped   已停止（定义保留，可再 run）
//	waiting   等待服务（inject 声明的服务未就绪，服务出现后自动激活）
//	rejected  装载被拒绝（求值/形态/schema 错误——定义期即可发现的问题）
//	failed    apply 失败（已执行但运行期报错）
//	cancelled 已取消（undefine 或用户中止）
type PluginState int

// PluginState 取值。
const (
	PluginStopped PluginState = iota
	PluginRunning
	PluginWaiting
	PluginRejected
	PluginFailed
	PluginCancelled
)

func (s PluginState) String() string {
	switch s {
	case PluginRunning:
		return "running"
	case PluginWaiting:
		return "waiting"
	case PluginRejected:
		return "rejected"
	case PluginFailed:
		return "failed"
	case PluginCancelled:
		return "cancelled"
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
	Scope      string       `json:"scope,omitempty"` // 生效作用域（JS 动态插件）："global"=全局插件（UI 类，跨工作区）/""或"project"=项目
	State      string       `json:"state"`
	Provides   []string     `json:"provides,omitempty"`
	Tools      []string     `json:"tools,omitempty"`
	Sections   []string     `json:"sections,omitempty"`
	Version    string       `json:"version,omitempty"`
	Purpose    string       `json:"purpose,omitempty"`    // 用途说明（JS 动态插件）
	HasClient  bool         `json:"hasClient,omitempty"`  // 是否有 client 半（浏览器端）
	ClientCode string       `json:"clientCode,omitempty"` // client 半源码（供浏览器装载；列表接口可能省略）
	// ClientApproved client 半是否已获激活批准（cordis_run 经审批门后为 true；
	// 浏览器仅装载已批准的 client 半）
	ClientApproved bool     `json:"clientApproved,omitempty"`
	DefID          string   `json:"defId,omitempty"`      // JS 动态插件定义 id（dyn-<n>）
	PluginID       string   `json:"pluginId,omitempty"`   // 稳定插件身份（跨版本；默认=首次定义 id）
	PkgID          string   `json:"pkgId,omitempty"`      // 当前版本 package id（pkg-<n>，不可变）
	Versions       int      `json:"versions,omitempty"`   // 该插件累计版本数（含历史）
	WaitingFor     []string `json:"waitingFor,omitempty"` // state=waiting 时缺的服务
	LastError      string   `json:"lastError,omitempty"`  // 最近一次装载失败原因（诊断）
	Diag           []string `json:"diag,omitempty"`       // 运行诊断（阶段记录，最新在后）
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

	// ★ 版本化 package 模型（对齐 harness registry）：
	//   pluginId（稳定身份）→ 版本链（package 列表，最新在尾）。define 时
	//   传 pluginId=existing 追加版本；cordis_run 传 pluginId 解析到最新版本。
	pluginVersions map[string][]*jsPluginDef
	// 等待中的定义（inject 声明服务缺失 → waiting；服务提供后自动重试激活）
	waiting map[string]*jsPluginDef

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

	// ★ inspect provider 注册表（对齐 harness hostInspectProviders/clientInspectProviders）：
	//   platform → providerID → InspectProvider。cordisInspectQuery 走注册表路由，
	//   第三方（内置代码/Go 插件）可 RegisterInspectProvider 扩展自定义诊断接口。
	inspectMu sync.RWMutex
	inspect   map[string]map[string]*InspectProvider

	// ★ approvedGlobalDir global 批准文件目录覆盖（默认空=core.InstallDir()）。
	//   测试隔离用：go test 开发态 InstallDir()=cwd=包目录，直接写会污染源码树
	//   （internal/agent/.pair/cordis-approved.json 残留导致测试误判）。
	approvedGlobalDir string

	// ★ 已废弃（2026-08-19）：client 半激活审批机制整体取消（参考项目无此机制）。
	//   以下字段与 load/save 函数仅为兼容保留，不再产生效果（IsClientApproved 恒
	//   true、cordis_run 不再触发审批门）；未来如需恢复审批可复用。
	approveMu       sync.RWMutex
	approvedClients map[string]bool // 合并视图（global + project）[废弃]
	approvedGlobal  map[string]bool // 全局（UI 类）批准：安装目录持久化 [废弃]
	approvedProject map[string]bool // 项目批准：工作区持久化 [废弃]
	root            string          // 工作区根（project 批准持久化用）

	// ★ 工具对 cordis 可见性（2026-08-19）：插件面板「插件内工具」对勾控制的是
	//   「JS 插件运行时（ctx.tools.list）能否看到该工具」，与 agent 可见性
	//   （工作区工具集 Enabled）完全解耦。默认全部对 cordis 可见（缺省 true）。
	cordisMu        sync.RWMutex
	toolCordisVisible map[string]bool
}

// ─── 工具对 cordis 可见性（对勾语义：控制 JS 插件运行时能否看到工具）───

// SetToolCordisVisible 设置工具对 cordis（ctx.tools.list）的可见性。
// 插件面板工具对勾（/api/plugins/tool）调用；不影响 agent 可见性
// （agent 只由工作区工具集 Enabled 决定）。
func (h *PluginHost) SetToolCordisVisible(name string, visible bool) {
	if h == nil {
		return
	}
	h.cordisMu.Lock()
	if h.toolCordisVisible == nil {
		h.toolCordisVisible = map[string]bool{}
	}
	h.toolCordisVisible[name] = visible
	h.cordisMu.Unlock()
}

// IsToolCordisVisible 工具是否对 cordis 可见（缺省 true——未显式设置即可见）。
func (h *PluginHost) IsToolCordisVisible(name string) bool {
	if h == nil {
		return true
	}
	h.cordisMu.RLock()
	v, ok := h.toolCordisVisible[name]
	h.cordisMu.RUnlock()
	if !ok {
		return true
	}
	return v
}

// ToolCordisVisibility 全部工具对 cordis 可见性快照（前端插件面板展示用）。
func (h *PluginHost) ToolCordisVisibility() map[string]bool {
	out := map[string]bool{}
	if h == nil {
		return out
	}
	h.cordisMu.RLock()
	for k, v := range h.toolCordisVisible {
		out[k] = v
	}
	h.cordisMu.RUnlock()
	return out
}

// InspectMethod 一个 inspect provider 方法（对齐参考 manifest.methods 的单个条目）。
type InspectMethod struct {
	Name        string // 方法名（小写，如 "listService"/"getService"）
	Description string
	// Query 实现查询；host 为插件宿主，input 为工具输入参数。
	Query func(h *PluginHost, input map[string]any) (string, error)
}

// InspectProvider 一个 inspect provider（对齐参考 manifest+query）：
// ID=provider 名（service/tool/event/plugin/...），Methods=方法表。
type InspectProvider struct {
	ID          string
	Description string
	Methods     map[string]InspectMethod
}

// RegisterInspectProvider 注册 inspect provider。
// platform：host（宿主侧）/ client（浏览器 client 半侧）；已存在同 ID 则整体覆盖。
func (h *PluginHost) RegisterInspectProvider(platform string, p *InspectProvider) error {
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "host" && platform != "client" {
		return fmt.Errorf("inspect provider 平台必须是 host 或 client，收到 %q", platform)
	}
	if p == nil || strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("inspect provider ID 不能为空")
	}
	p.ID = strings.ToLower(strings.TrimSpace(p.ID))
	h.inspectMu.Lock()
	defer h.inspectMu.Unlock()
	if h.inspect == nil {
		h.inspect = map[string]map[string]*InspectProvider{}
	}
	if h.inspect[platform] == nil {
		h.inspect[platform] = map[string]*InspectProvider{}
	}
	h.inspect[platform][p.ID] = p
	return nil
}

// InspectProviderLookup 查询已注册的 inspect provider（未注册返回 nil）。
func (h *PluginHost) InspectProviderLookup(platform, provider string) *InspectProvider {
	platform = strings.ToLower(strings.TrimSpace(platform))
	provider = strings.ToLower(strings.TrimSpace(provider))
	h.inspectMu.RLock()
	defer h.inspectMu.RUnlock()
	if h.inspect == nil {
		return nil
	}
	if pm := h.inspect[platform]; pm != nil {
		return pm[provider]
	}
	return nil
}

// InspectPlatforms 列出已注册的 inspect 平台（host/client）。
func (h *PluginHost) InspectPlatforms() []string {
	h.inspectMu.RLock()
	defer h.inspectMu.RUnlock()
	var out []string
	for p := range h.inspect {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// InspectProviders 列出某平台已注册的 provider ID（排序）。
func (h *PluginHost) InspectProviders(platform string) []string {
	h.inspectMu.RLock()
	defer h.inspectMu.RUnlock()
	var out []string
	for id := range h.inspect[platform] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// ─── D11：client 半 → host 半 调用链（invoke RPC + 失败上报）───

// InvokeClientMethod 浏览器 client 半远程调用 host 半注册的方法（对齐 harness
// @Remote('invoke')）。plugin 可为插件名或 defId；method 为 harness.handle /
// ctx 注册的处理器名；args 为 JSON 参数。返回 (结果, 错误)；错误语义：
//   - 插件未运行（未找到 running 实例）
//   - 方法未注册（handlers 中不存在）
//   - 处理器执行异常/超时
func (h *PluginHost) InvokeClientMethod(plugin, method string, args any) (any, error) {
	h.mu.RLock()
	adapter := h.findRunningJSAdapter(plugin)
	h.mu.RUnlock()
	if adapter == nil {
		return nil, fmt.Errorf("插件 %q 未在运行（无法 invoke %s）", plugin, method)
	}
	adapter.mu.Lock()
	fn, ok := adapter.handlers[method]
	adapter.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("插件 %q 未注册 host 方法 %q（可用 harness.handle 或 ctx.registerClientMethod 注册）", plugin, method)
	}
	out, err := fn(args)
	if err != nil {
		// 对齐 harness steerHostHandlerFailure：记诊断供 Agent 修复
		adapter.def.addDiag(fmt.Sprintf("client invoke %s 失败: %v", method, err))
		return nil, err
	}
	return out, nil
}

// findRunningJSAdapter 按插件名或 defId 找运行中的 JS 插件适配器（h.mu 读锁外调用）。
func (h *PluginHost) findRunningJSAdapter(nameOrID string) *jsPluginAdapter {
	// 直接按名字/defId 匹配运行中插件
	if p, ok := h.plugins[nameOrID]; ok {
		if a, ok := p.(*jsPluginAdapter); ok && a.def.status == PluginRunning {
			return a
		}
	}
	// 按插件名匹配（defs 里 name；注册键 = 插件名）
	for _, d := range h.defs {
		if d.name == nameOrID && d.status == PluginRunning {
			if p, ok := h.plugins[d.name]; ok {
				if a, ok := p.(*jsPluginAdapter); ok {
					return a
				}
			}
		}
	}
	return nil
}

// ReportClientFailure 浏览器 client 半失败上报（对齐 harness
// reportRenderFailure/reportClientGuardFailure）：记入定义诊断，供 Agent
// 经 cordis_inspect 发现并修复。不改变插件运行状态（host 半不受影响）。
func (h *PluginHost) ReportClientFailure(plugin, phase, message string) error {
	if plugin == "" {
		return fmt.Errorf("plugin 不能为空")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	d := h.findDefByNameOrID(plugin)
	if d == nil {
		return fmt.Errorf("插件 %q 未定义", plugin)
	}
	phaseText := "render"
	if phase == "guard" {
		phaseText = "guard"
	} else if phase == "boot" {
		phaseText = "boot"
	}
	d.lastError = fmt.Sprintf("client 半 %s 失败: %s", phaseText, message)
	d.addDiag(fmt.Sprintf("client 半 %s 失败: %s", phaseText, message))
	return nil
}

// findDefByNameOrID 按插件名或 defId 找定义（h.mu 锁内调用）。
func (h *PluginHost) findDefByNameOrID(nameOrID string) *jsPluginDef {
	if d, ok := h.defs[nameOrID]; ok {
		return d
	}
	for _, d := range h.defs {
		if d.name == nameOrID {
			return d
		}
	}
	return nil
}

// NewPluginHost 创建插件宿主。
// registry：工具注册表（ctx.tools）；store：会话存储（ctx.session）；root：工作区根。
func NewPluginHost(registry *Registry, store ConversationStore, root string) *PluginHost {
	h := &PluginHost{
		plugins:         map[string]Plugin{},
		states:          map[string]PluginState{},
		sources:         map[string]PluginSource{},
		defs:            map[string]*jsPluginDef{},
		pluginVersions:  map[string][]*jsPluginDef{},
		waiting:         map[string]*jsPluginDef{},
		pluginTools:     map[string][]string{},
		pluginSections:  map[string][]*PromptSection{},
		pluginVars:      map[string][]*PromptVariable{},
		toolOwner:       map[string]string{},
		templates:       map[string]*ToolsetTemplate{},
		approvedClients: map[string]bool{},
		approvedGlobal:  map[string]bool{},
		approvedProject: map[string]bool{},
		root:            root,
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
	// 内置 inspect provider（host: service/tool/event/plugin；client: plugin/event/service/tool）
	registerBuiltinInspectProviders(h)
	// 恢复跨重启的 client 半批准记录（.pair/cordis-approved.json；文件缺失/损坏不致命）
	h.loadApprovedClients()

	// ── 框架能力内联（宿主固有，不经过插件体系：不可启停、不出现在插件列表）──
	// workspaceRoot 服务：宿主向插件生态暴露自身工作区根（原 sysinfo 插件——
	// 提供者就是宿主自身，分离无意义；JS 插件走 app.workspaceRoot 注入，
	// Go/JS 插件经 ctx.Get("workspaceRoot") 取）。
	h.ctx.Provide("workspaceRoot", root)
	// 内置工具集构建模板（原 toolset-tpl-core 插件）：toolset_build 的动态组合
	// 数据源——Generate 逻辑内嵌宿主（toolset_templates.go），随宿主合理；
	// 市场/用户插件仍可经 RegisterTemplate / ctx.toolset.registerTemplate 追加。
	registerBuiltinTemplates(h)
	return h
}

// ─── client 半激活批准（已废弃）────────────────────────────
// ★ 2026-08-19：client 半激活审批机制整体取消（参考项目 deepseek-harness 无
//   approvedClientPackages 机制，属自行添加）→ IsClientApproved 恒 true，
//   cordis_run 不再触发审批门。以下 load/save 函数保留仅为兼容，不再产生效果；
//   未来如需恢复审批可复用。

// approvedFilePath 按作用域返回 .pair/cordis-approved.json 绝对路径：
//   - global：安装目录 <InstallDir>/.pair/（UI 类插件跨工作区生效，随安装包
//     发布——UI 插件与工作区无关；发布版打开任意工作区都不丢批准）
//   - project：工作区 <root>/.pair/（工具插件按项目隔离）
//
// 未打开工作区时 project 退回安装目录（无工作区也能记录项目级批准）。
func (h *PluginHost) approvedFilePath(scope string) string {
	if scope == "global" {
		base := h.approvedGlobalDir
		if base == "" {
			base = core.InstallDir()
		}
		return filepath.Join(base, ".pair", "cordis-approved.json")
	}
	if h.root != "" {
		return filepath.Join(h.root, ".pair", "cordis-approved.json")
	}
	return filepath.Join(core.InstallDir(), ".pair", "cordis-approved.json")
}

// SetApprovedGlobalDir 覆盖 global 批准文件目录（测试隔离用；生产代码不要调用）。
func (h *PluginHost) SetApprovedGlobalDir(dir string) {
	h.approveMu.Lock()
	defer h.approveMu.Unlock()
	h.approvedGlobalDir = dir
}

// loadApprovedClients 恢复批准记录：global（安装目录）+ project（工作区）分别
// 加载（缺文件/坏 JSON 静默），合并到 approvedClients 视图。
// ★ 发布版用户电脑：安装目录文件随安装包存在（含全部 UI 插件批准），即使
//
//	用户打开了全新工作区（工作区文件缺失）也保持已批准——UI 插件与工作区无关。
func (h *PluginHost) loadApprovedClients() {
	h.loadApprovedFile(h.approvedFilePath("global"), h.approvedGlobal)
	h.loadApprovedFile(h.approvedFilePath("project"), h.approvedProject)
	h.approveMu.Lock()
	for n := range h.approvedGlobal {
		h.approvedClients[n] = true
	}
	for n := range h.approvedProject {
		h.approvedClients[n] = true
	}
	h.approveMu.Unlock()
}

// loadApprovedFile 从指定文件加载批准名单到目标 map（缺文件/坏 JSON 静默）。
func (h *PluginHost) loadApprovedFile(p string, target map[string]bool) {
	if p == "" {
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return
	}
	h.approveMu.Lock()
	defer h.approveMu.Unlock()
	for _, n := range names {
		if n != "" {
			target[n] = true
		}
	}
}

// saveApprovedClients 持久化批准记录：global → 安装目录；project → 工作区
// （失败静默——批准生效以内存为准，重启后重批即可）。
func (h *PluginHost) saveApprovedClients() {
	h.saveApprovedFile(h.approvedFilePath("global"), h.approvedGlobal)
	h.saveApprovedFile(h.approvedFilePath("project"), h.approvedProject)
}

// saveApprovedFile 把批准名单写入指定文件（失败静默）。
func (h *PluginHost) saveApprovedFile(p string, src map[string]bool) {
	if p == "" {
		return
	}
	h.approveMu.RLock()
	names := make([]string, 0, len(src))
	for n := range src {
		names = append(names, n)
	}
	h.approveMu.RUnlock()
	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

// IsClientApproved 该插件的 client 半是否已获激活批准。
// ★ 2026-08-19：client 半激活审批机制整体取消（参考项目 deepseek-harness 无
//   approvedClientPackages 机制，属自行添加）→ 恒 true：所有带 client 半的插件
//   视为已批准，浏览器直接装载，无需 cordis_run 审批门、无需手动维护
//   .pair/cordis-approved.json。保留签名兼容调用点；load/save 批准文件不再产生效果。
func (h *PluginHost) IsClientApproved(pluginID string) bool {
	return true
}

// MarkClientApproved 批准插件 client 半激活（cordis_run 经审批门执行成功后调用，
// 传插件稳定身份 pluginId 与作用域 scope），按作用域持久化：
// global（UI 类）→ 安装目录；project → 工作区。批准覆盖该插件后续版本。
func (h *PluginHost) MarkClientApproved(pluginID, scope string) {
	if pluginID == "" {
		return
	}
	h.approveMu.Lock()
	h.approvedClients[pluginID] = true
	if scope == "global" {
		h.approvedGlobal[pluginID] = true
	} else {
		h.approvedProject[pluginID] = true
	}
	h.approveMu.Unlock()
	if scope == "global" {
		h.saveApprovedFile(h.approvedFilePath("global"), h.approvedGlobal)
	} else {
		h.saveApprovedFile(h.approvedFilePath("project"), h.approvedProject)
	}
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
	Status  string   `json:"status"`            // loaded | error
	Panels  []string `json:"panels,omitempty"`  // 注册的面板 id
	Events  []string `json:"events,omitempty"`  // 监听的事件名（ui.on）
	Version string   `json:"version,omitempty"` // client 半版本（同 host 定义版本）
	Error   string   `json:"error,omitempty"`   // 装载失败原因
}

// ClientRuntimeSnapshot 浏览器 client 半运行时整体快照（浏览器周期上报）。
type ClientRuntimeSnapshot struct {
	Connected  bool                   `json:"connected"`  // 页面在线且上报过
	ReportedAt int64                  `json:"reportedAt"` // 本次上报时间（Unix 秒）
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
// 宿主内联注册内置通用模板（NewPluginHost），市场/用户插件可注册专属模板）。
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
	// JS 动态插件：定义状态复位（与宿主状态表一致；版本链不再显示 running）
	if adapter, ok := h.plugins[name].(*jsPluginAdapter); ok {
		adapter.def.setStatus(PluginStopped, nil)
	}
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

// UndefinePermanent 删除插件定义并同步删除磁盘插件包（前端「删除定义」按钮 /
// cordis_undefine 用）。与 Undefine 的区别：Undefine 只删进程内存（defs/plugins/
// sources/order），磁盘插件包目录 <InstallDir>/.pair/plugins/<name>/ 保留——
// 重启 LoadGlobalPlugins 扫描目录重新装配，插件「复活」。Permanent 复用
// RemoveJSDef（解析 def → 删版本链 → 删磁盘包），彻底移除。
// ★ 工具集装卸等内部路径仍用 Undefine（纯内存语义，不误删磁盘包）。
func (h *PluginHost) UndefinePermanent(name string) error {
	// 解析 def（dyn id / pluginId / 插件名）——defs 的 key 是 dyn id，不能直接
	// 按 name 调 Undefine（未装载时 defs[name]/plugins[name] 都查不到）。
	def, err := h.resolveJSDef(name)
	if err != nil {
		// 非 JS 插件（Go 插件等）：回退普通 Undefine（无磁盘包概念）
		return h.Undefine(name)
	}
	return h.RemoveJSDef(def.pluginId)
}

// removeGlobalPluginPackage 删除全局插件包目录（仅限 globalPluginsDir 内的目录，
// 且含 package.json 才删——防误删非插件目录；目录不存在则静默跳过）。
// 供 UndefinePermanent / RemoveJSDef 复用。
func removeGlobalPluginPackage(dir string) error {
	if dir == "" {
		return nil
	}
	base := filepath.Clean(globalPluginsDir())
	dirC := filepath.Clean(dir)
	if !strings.HasPrefix(dirC, base+string(os.PathSeparator)) {
		// 目录不在全局插件目录内（如工具集路径/测试临时目录）→ 不删磁盘
		return nil
	}
	if _, err := os.Stat(filepath.Join(dirC, "package.json")); err != nil {
		return nil // 非插件包目录 / 已不存在 → 无需处理
	}
	if err := os.RemoveAll(dirC); err != nil {
		return fmt.Errorf("删除插件包目录 %s 失败: %w", dirC, err)
	}
	return nil
}

// Get 取插件（未注册返回 nil,false）。
func (h *PluginHost) Get(name string) (Plugin, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[name]
	return p, ok
}

// ─── inject 等待语义（D3：对齐 harness lifecycle 的 waiting）──────

// waitForServices 把 def 登记为等待状态（inject 声明服务缺失时调用）。
// 插件进入 waiting：不装载、不 apply；服务出现后经 retryWaiting 自动激活。
func (h *PluginHost) waitForServices(def *jsPluginDef, missing []string) {
	h.mu.Lock()
	h.waiting[def.id] = def
	h.mu.Unlock()
	def.setStatus(PluginWaiting, missing)
}

// RetryWaiting 在服务提供后尝试激活全部等待该服务的插件（ctx.provide 触发）。
func (h *PluginHost) retryWaiting(serviceName string) {
	h.mu.Lock()
	var retry []*jsPluginDef
	for id, def := range h.waiting {
		if strInSlice(def.waitingFor, serviceName) {
			retry = append(retry, def)
			delete(h.waiting, id)
		}
	}
	h.mu.Unlock()
	for _, def := range retry {
		if err := h.LoadJSDynamic(def); err != nil {
			// 重试失败：若非等待类错误，记录诊断并回到可重试状态
			log.Printf("[cordis-waiting] 插件 %s 服务 %s 就绪后重试装载失败: %v", def.id, serviceName, err)
		}
	}
}

// resolveJSDef 把 cordis_run/stop/undefine 的 id 解析为 JS 定义：
//   - 精确 dyn id（pkg-xxx 的 def）→ 该版本
//   - pluginId（稳定身份）→ 版本链最新版
//   - 插件名 → 匹配该名插件的最新版本
func (h *PluginHost) resolveJSDef(idOrName string) (*jsPluginDef, error) {
	// ★ pluginId（稳定身份）优先解析到最新版本（对齐 cordis_run 语义）：
	//   注意 pluginId 恒等于首次 dyn id（如 dyn-1），而该 id 同时也是 v1 的精确 id——
	//   必须先查 pluginVersions 链，否则追加版本后传 pluginId 会错误命中旧版本 v1。
	h.mu.RLock()
	if chain := h.pluginVersions[idOrName]; len(chain) > 0 {
		d := chain[len(chain)-1]
		h.mu.RUnlock()
		return d, nil
	}
	h.mu.RUnlock()
	if def, ok := h.GetJSDef(idOrName); ok {
		return def, nil
	}
	// 按插件名匹配（同名插件取最新版本）
	h.mu.RLock()
	var best *jsPluginDef
	for _, d := range h.defs {
		if d.name == idOrName && (best == nil || d.createdAt.After(best.createdAt)) {
			best = d
		}
	}
	h.mu.RUnlock()
	if best != nil {
		return best, nil
	}
	return nil, fmt.Errorf("插件定义不存在: %s（定义只活在进程内存，跨重启不存续）", idOrName)
}

// PluginVersions 返回某 pluginId 的版本链（最新在尾；不存在返回 nil）。
func (h *PluginHost) PluginVersions(pluginId string) []*jsPluginDef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]*jsPluginDef(nil), h.pluginVersions[pluginId]...)
}

// PluginIds 全部已知 pluginId（含单版本插件）。
func (h *PluginHost) PluginIds() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.pluginVersions))
	for id := range h.pluginVersions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// waitingDefs 全部等待中的定义（排序；供 cordis_inspect 报告）。
func (h *PluginHost) waitingDefs() []*jsPluginDef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*jsPluginDef, 0, len(h.waiting))
	for _, d := range h.waiting {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
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
				rec.Scope = d.scope
				rec.Version = d.version
				rec.Provides = d.provides
				rec.Purpose = d.purpose
				rec.HasClient = strings.TrimSpace(d.clientCode) != ""
				rec.ClientCode = d.clientCode
				rec.DefID = d.id
				rec.PluginID = d.pluginId
				rec.PkgID = d.packageId
				if chain := h.pluginVersions[d.pluginId]; len(chain) > 0 {
					rec.Versions = len(chain)
				} else {
					rec.Versions = 1
				}
				rec.WaitingFor = d.waitingFor
				rec.LastError = d.lastError
				rec.Diag = d.diag
				if d.status == PluginWaiting {
					rec.State = "waiting"
				}
				// client 半激活批准状态（浏览器仅装载已批准；cordis_run 经审批门后批准）
				if rec.HasClient {
					rec.ClientApproved = h.IsClientApproved(d.name)
				}
			}
		}
		rec.Tools = append([]string(nil), h.pluginTools[name]...)
		// ★ 内置 Go 插件：工具经 Registry.Register 直接注册（不经 addPluginTool，
		//   pluginTools 为空）——补静态派生工具清单，cordis_inspect/插件面板可见
		//   （agent「通过插件列表看见被过滤工具」的通道）。
		if len(rec.Tools) == 0 && isBuiltinPluginName(name) {
			rec.Tools = append([]string(nil), builtinPluginToolGroups()[name]...)
		}
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
				rec.PluginID = d.pluginId
				rec.PkgID = d.packageId
				if chain := h.pluginVersions[d.pluginId]; len(chain) > 0 {
					rec.Versions = len(chain)
				} else {
					rec.Versions = 1
				}
				rec.WaitingFor = d.waitingFor
				rec.LastError = d.lastError
				rec.Diag = d.diag
				if d.status == PluginWaiting {
					rec.State = "waiting"
				}
				// client 半激活批准状态（浏览器仅装载已批准；cordis_run 经审批门后批准）
				if rec.HasClient {
					rec.ClientApproved = h.IsClientApproved(d.name)
				}
			}
		}
		rec.Tools = append([]string(nil), h.pluginTools[n]...)
		if len(rec.Tools) == 0 && isBuiltinPluginName(n) {
			rec.Tools = append([]string(nil), builtinPluginToolGroups()[n]...)
		}
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

// PersonaSection 插件贡献的 persona 槽位段（name==PERSONA_SECTION）。
// 返回第一个命中者（多插件同名时按注册序取先者）；无则返回 nil。
// 组装系统提示时，若此段非空，用它**替换**默认 persona 段（对齐 harness persona slot）。
func (h *PluginHost) PersonaSection() *PromptSection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, secs := range h.pluginSections {
		for _, s := range secs {
			if s != nil && s.Name == PERSONA_SECTION && strings.TrimSpace(s.Text) != "" {
				return s
			}
		}
	}
	return nil
}

// RulesSection 插件贡献的行为准则槽位段（name==RULES_SECTION）。
// 返回第一个命中者；无则返回 nil。
// 组装系统提示时，若此段非空，用它**替换**默认规则段（# 工作区之后的
// 第一铁律/核心规则/调研/搜索/错误恢复/修改纪律等全部行为准则段）。
func (h *PluginHost) RulesSection() *PromptSection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, secs := range h.pluginSections {
		for _, s := range secs {
			if s != nil && s.Name == RULES_SECTION && strings.TrimSpace(s.Text) != "" {
				return s
			}
		}
	}
	return nil
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

// PluginToolsByPlugin 插件名 → 该插件注册的工具名清单（快照；含内置 Go 插件
// 与 JS 动态插件）。内置工具集分组（BuiltinToolsetOf）用它取各内置插件组工具。
func (h *PluginHost) PluginToolsByPlugin() map[string][]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string][]string, len(h.pluginTools))
	for k, v := range h.pluginTools {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// PluginToolOwners 工具名 → 归属插件（快照；toolOwner 反向表）。
func (h *PluginHost) PluginToolOwners() map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make(map[string]string, len(h.toolOwner))
	for k, v := range h.toolOwner {
		out[k] = v
	}
	return out
}

// HasPluginTool 判断工具是否已由某插件注册（claimTool 预检用）。
func (h *PluginHost) HasPluginTool(tool string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, taken := h.toolOwner[tool]
	return taken
}

func (h *PluginHost) addPluginSection(plugin string, s *PromptSection) {
	h.mu.Lock()
	h.pluginSections[plugin] = append(h.pluginSections[plugin], s)
	h.mu.Unlock()
}
