// ═══════════════════════════════════════════════════════════
// node_bridge_conflict.go — DSH/npm 桥插件与 repo 移植版插件「并存 + 可切换」
//
// 背景（2026-08-31）：此前桥插件装载会自动停用同源 repo 移植版 goja 插件
// （takeoverConflictingPlugin），goja 侧 claimTool 也对桥插件「让位」——结果
// 「装了 外部版就看不见 repo 版」：插件面板只剩一个 agent-teams，用户既无法
// 对比两个来源的差异，也无法回退。
//
// 现语义（取消同名工具覆盖冲突）：
//   - 两版并存：桥插件并入 Inspect 输出（source=node-bridge），与 goja 插件
//     （source=js）同列插件面板/cordis_inspect，来源与版本各自标注；
//   - 同名工具默认 repo 版优先生效，桥侧同名工具「挂起」（实例保留在 b.tools）；
//   - 生效方可切换：SetBridgeToolPreference(ph, tool, "bridge"|"repo")
//     （HTTP：POST /api/plugins/prefer）；
//   - repo 版插件停用（面板「停止插件」）→ 挂起的桥工具自动恢复生效。
//
// 数据面：冲突记录挂在 nodeBridge（受 toolsMu 保护），桥未启动时全部函数
// nil 安全（返回空、无操作）。
// ═══════════════════════════════════════════════════════════
package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 生效方取值（ToolConflictInfo.Active）。
const (
	ToolImplRepo   = "repo"   // repo 移植版（goja 插件）生效
	ToolImplBridge = "bridge" // DSH/npm 桥插件生效
)

// bridgeToolConflict 单个同名工具的并存记录（桥侧 ↔ repo 侧）。
type bridgeToolConflict struct {
	tool       string // 工具名
	bridgePkg  string // 桥插件包名（如 @nanmicoder/dsh-agent-teams）
	repoPlugin string // repo 侧插件名（如 agent-teams）
	active     string // 当前生效方：ToolImplRepo | ToolImplBridge
	repoTool   *Tool  // repo 侧工具实例（active=bridge 期间保存，供切回）
}

// noteConflict 记录/更新一条并存关系（桥侧内部，自持 toolsMu）。
func (b *nodeBridge) noteConflict(tool, bridgePkg, repoPlugin, active string, repoTool *Tool) {
	if b == nil || tool == "" {
		return
	}
	b.toolsMu.Lock()
	defer b.toolsMu.Unlock()
	if b.conflicts == nil {
		b.conflicts = map[string]*bridgeToolConflict{}
	}
	c := b.conflicts[tool]
	if c == nil {
		c = &bridgeToolConflict{tool: tool}
		b.conflicts[tool] = c
	}
	if bridgePkg != "" {
		c.bridgePkg = bridgePkg
	}
	if repoPlugin != "" {
		c.repoPlugin = repoPlugin
	}
	if active != "" {
		c.active = active
	}
	if repoTool != nil {
		c.repoTool = repoTool
	}
}

// noteLoadedSpecs 记录 ready 消息上报的已装载插件 spec（pkg → spec）。
// ready 是每次桥启动的全量清单 → 全量替换（卸载/失败的包不残留 running 状态）。
func (b *nodeBridge) noteLoadedSpecs(specs []string) {
	if b == nil {
		return
	}
	next := make(map[string]string, len(specs))
	for _, s := range specs {
		pkg, _ := splitNpmSpec(s)
		if pkg == "" {
			continue
		}
		next[pkg] = s
	}
	b.toolsMu.Lock()
	b.specs = next
	b.toolsMu.Unlock()
}

// conflictSnapshot 冲突快照（按工具名排序，只读副本）。
func (b *nodeBridge) conflictSnapshot() []ToolConflictInfo {
	if b == nil {
		return nil
	}
	b.toolsMu.Lock()
	out := make([]ToolConflictInfo, 0, len(b.conflicts))
	for _, c := range b.conflicts {
		active := c.active
		if active == "" {
			active = ToolImplRepo
		}
		out = append(out, ToolConflictInfo{Tool: c.tool, Repo: c.repoPlugin, Bridge: c.bridgePkg, Active: active})
	}
	b.toolsMu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Tool < out[j].Tool })
	return out
}

// bridgeToolConflicts 全局并存快照（桥未运行 → nil）。
func bridgeToolConflicts() []ToolConflictInfo {
	return globalNodeBridge.conflictSnapshot()
}

// BridgeToolConflicts 并存快照（导出：HTTP handler / 校验用）。
func BridgeToolConflicts() []ToolConflictInfo { return bridgeToolConflicts() }

// noteBridgeToolPreempted goja（repo 移植版）插件抢占已由桥注册的同名工具时
// 记录并存关系（claimTool 调用，active=repo）。桥工具实例仍留在 b.tools，
// 切换回 外部版时直接复用。
//
// reg 仅用于取被抢占前的桥工具实例做校验（可为 nil）。
func noteBridgeToolPreempted(tool, bridgeOwner, byPlugin string, reg *Registry) {
	b := globalNodeBridge
	if b == nil || tool == "" {
		return
	}
	pkg := strings.TrimPrefix(bridgeOwner, "node-bridge:")
	b.noteConflict(tool, pkg, byPlugin, ToolImplRepo, nil)
	log.Printf("[node-bridge] 同名工具并存：%q 改由 repo 插件 %s 生效（repo 优先），外部插件 %s 的同名工具挂起——插件面板可切换生效方",
		tool, byPlugin, pkg)
}

// shadowConflictingTool 桥工具注册遇同名占用（占用方为 repo 侧插件）时的处置：
// 不再停用对方、不再覆盖 —— 记录并存关系（active=repo），桥工具挂起。
// 返回是否成功记录（占用方为其他桥插件等异常情况返回 false）。
func (b *nodeBridge) shadowConflictingTool(ph *PluginHost, owner string, tool *Tool) bool {
	if b == nil || ph == nil || tool == nil {
		return false
	}
	pkg := strings.TrimPrefix(owner, "node-bridge:")
	existing := ph.PluginToolOwners()[tool.Name]
	if existing == "" || strings.HasPrefix(existing, "node-bridge:") {
		// 占用方是宿主内置工具或另一个桥插件 → 非「repo 版 ↔ 外部版」并存关系，
		// 保持既有严格语义（不记录、由调用方打印失败日志）。
		return false
	}
	b.noteConflict(tool.Name, pkg, existing, ToolImplRepo, nil)
	log.Printf("[node-bridge] 同名工具并存：%q 由 repo 插件 %s 生效（repo 优先），外部插件 %s 的同名工具挂起——插件面板可切换生效方",
		tool.Name, existing, pkg)
	return true
}

// restoreBridgeToolsFor repo 侧插件停用后，恢复其挂起的桥工具（自动接管）。
// 幂等：无挂起工具时无操作；桥未就绪时只更新记录不注册。
func restoreBridgeToolsFor(h *PluginHost, repoPlugin string) {
	b := globalNodeBridge
	if b == nil || h == nil || repoPlugin == "" {
		return
	}
	type pending struct {
		tool  *Tool
		owner string
	}
	var todo []pending
	b.toolsMu.Lock()
	for name, c := range b.conflicts {
		if c.repoPlugin != repoPlugin || c.active == ToolImplBridge {
			continue
		}
		t := b.tools[name]
		if t == nil {
			continue
		}
		c.active = ToolImplBridge
		c.repoTool = nil // repo 插件已停用，其工具实例由插件重启时重建
		todo = append(todo, pending{tool: t, owner: "node-bridge:" + c.bridgePkg})
	}
	b.toolsMu.Unlock()
	for _, p := range todo {
		if _, _, err := h.SwapToolOwner(p.tool.Name, p.owner, p.tool); err != nil {
			log.Printf("[node-bridge] 恢复挂起工具 %s 失败: %v", p.tool.Name, err)
			continue
		}
		log.Printf("[node-bridge] repo 插件 %s 已停用 → 外部版工具 %s 自动接管生效", repoPlugin, p.tool.Name)
	}
}

// SetBridgeToolPreference 切换同名工具的生效实现（插件面板「生效方」切换）。
// impl: "bridge"（DSH/npm 桥插件生效）| "repo"（repo 移植版 goja 插件生效）。
// 两侧插件运行状态均不变——只换 Registry 生效面（并存语义）。
func SetBridgeToolPreference(ph *PluginHost, tool, impl string) error {
	if ph == nil {
		return fmt.Errorf("插件系统未初始化")
	}
	b := globalNodeBridge
	if b == nil {
		return fmt.Errorf("Node 桥未运行（无 DSH/npm 桥插件工具）")
	}
	if impl != ToolImplRepo && impl != ToolImplBridge {
		return fmt.Errorf("impl 必须是 %s 或 %s，收到 %q", ToolImplRepo, ToolImplBridge, impl)
	}
	b.toolsMu.Lock()
	c := b.conflicts[tool]
	if c == nil {
		b.toolsMu.Unlock()
		return fmt.Errorf("工具 %q 无同名并存记录（无需切换）", tool)
	}
	cur := c.active
	if cur == "" {
		cur = ToolImplRepo
	}
	bridgeTool := b.tools[tool]
	repoTool := c.repoTool
	repoPlugin, bridgePkg := c.repoPlugin, c.bridgePkg
	b.toolsMu.Unlock()
	if cur == impl {
		return nil // 幂等：已是目标生效方
	}
	switch impl {
	case ToolImplBridge:
		if bridgeTool == nil {
			return fmt.Errorf("桥插件 %s 未注册工具 %q（桥可能未就绪）", bridgePkg, tool)
		}
		old, oldOwner, err := ph.SwapToolOwner(tool, "node-bridge:"+bridgePkg, bridgeTool)
		if err != nil {
			return err
		}
		b.noteConflict(tool, bridgePkg, repoPlugin, ToolImplBridge, old)
		log.Printf("[node-bridge] 工具 %q 生效方切换：%s（repo）→ %s（外部桥）", tool, oldOwner, bridgePkg)
	case ToolImplRepo:
		if repoTool == nil {
			return fmt.Errorf("repo 插件 %s 的工具 %q 实例已不在（请在插件面板重启该插件后再切回）", repoPlugin, tool)
		}
		if _, _, err := ph.SwapToolOwner(tool, repoPlugin, repoTool); err != nil {
			return err
		}
		b.noteConflict(tool, bridgePkg, repoPlugin, ToolImplRepo, nil)
		log.Printf("[node-bridge] 工具 %q 生效方切换：%s（外部桥）→ %s（repo）", tool, bridgePkg, repoPlugin)
	}
	return nil
}

// bridgePluginRecords Node 桥插件记录（供 Inspect 合并；桥未运行时仍按
// plugins.json 声明清单输出 stopped 记录——面板能看见「装了但没跑起来」）。
func bridgePluginRecords(conflicts []ToolConflictInfo) []PluginRecord {
	dir := nodeBridgeDir()
	declared, err := readNodePluginsFile(filepath.Join(dir, "plugins.json"))
	if err != nil || declared == nil {
		declared = &nodePluginsFile{}
	}
	purposes := bridgePluginPurposes(dir)

	b := globalNodeBridge
	ready := false
	loaded := map[string]bool{}
	toolsByPkg := map[string][]string{}
	if b != nil {
		ready = b.isReady()
		b.toolsMu.Lock()
		for pkg := range b.specs {
			loaded[pkg] = true
		}
		for tool, owner := range b.toolOwner {
			pkg := strings.TrimPrefix(owner, "node-bridge:")
			toolsByPkg[pkg] = append(toolsByPkg[pkg], tool)
		}
		b.toolsMu.Unlock()
	}

	// 声明清单（plugins.json）+ 运行时已装载（ready 消息）合并去重
	type entry struct {
		spec    string
		runtime string
	}
	order := make([]string, 0, len(declared.Plugins))
	byPkg := map[string]entry{}
	add := func(spec, runtime string) {
		pkg, ver := splitNpmSpec(spec)
		if pkg == "" {
			return
		}
		if _, seen := byPkg[pkg]; !seen {
			order = append(order, pkg)
		}
		e := byPkg[pkg]
		if e.spec == "" || ver != "" {
			e.spec = spec
		}
		if runtime != "" {
			e.runtime = runtime
		}
		byPkg[pkg] = e
	}
	for _, d := range declared.Plugins {
		add(d.Spec, d.Runtime)
	}
	if b != nil {
		b.toolsMu.Lock()
		for pkg, spec := range b.specs {
			_ = pkg
			add(spec, "")
		}
		b.toolsMu.Unlock()
	}
	sort.Strings(order)

	recs := make([]PluginRecord, 0, len(order))
	for _, pkg := range order {
		e := byPkg[pkg]
		_, ver := splitNpmSpec(e.spec)
		state := "stopped"
		switch {
		case ready && loaded[pkg]:
			state = "running"
		case ready && !loaded[pkg]:
			state = "error" // 桥已就绪但该包未装载成功（见 [bridge] 日志）
		}
		tools := append([]string(nil), toolsByPkg[pkg]...)
		sort.Strings(tools)
		rec := PluginRecord{
			Name:    pkg,
			Source:  PluginSourceBridge,
			State:   state,
			Version: ver,
			Spec:    e.spec,
			Runtime: e.runtime,
			Purpose: purposes[e.spec],
			Tools:   tools,
		}
		if rec.Runtime == "" {
			rec.Runtime = "node"
		}
		if rec.Purpose == "" {
			rec.Purpose = bridgeRuntimeLabel(rec.Runtime)
		}
		if state == "error" {
			rec.LastError = "Node 桥已就绪但该包未装载成功（node 侧 import/apply 失败，见 [bridge] 日志）"
		}
		for _, c := range conflicts {
			if c.Bridge == pkg {
				rec.Conflicts = append(rec.Conflicts, c)
			}
		}
		recs = append(recs, rec)
	}
	return recs
}

// bridgeRuntimeLabel 运行时轨的人话说明（无 purpose 时兜底展示）。
func bridgeRuntimeLabel(runtime string) string {
	if runtime == "dsh" {
		return "外部插件（Node 桥装载，cordis4 + 外部服务面）"
	}
	return "npm 插件（Node 桥装载，cordis3）"
}

// bridgePluginPurposes 从 .pair/cordis.patch.json 取桥插件用途说明（key=npm spec）。
func bridgePluginPurposes(bridgeDir string) map[string]string {
	out := map[string]string{}
	root := npmPluginProjectRoot()
	if root == "" {
		if bridgeDir == "" {
			return out
		}
		// .pair/cordis/node → 回推工作区根
		root = filepath.Dir(filepath.Dir(filepath.Dir(bridgeDir)))
	}
	path := filepath.Join(root, ".pair", "cordis.patch.json")
	if _, err := os.Stat(path); err != nil {
		return out
	}
	doc, err := readCordisPatch(path)
	if err != nil || doc == nil {
		return out
	}
	for _, p := range doc.Plugins {
		spec, _ := p.Config["npm"].(string)
		if spec != "" && strings.TrimSpace(p.Purpose) != "" {
			out[spec] = p.Purpose
		}
	}
	return out
}

// splitNpmSpec 拆分 npm spec（"@scope/pkg@1.2.3" → "@scope/pkg", "1.2.3"）。
func splitNpmSpec(spec string) (string, string) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", ""
	}
	if at := strings.LastIndex(s, "@"); at > 0 {
		return s[:at], s[at+1:]
	}
	return s, ""
}
