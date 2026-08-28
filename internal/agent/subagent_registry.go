// ═══════════════════════════════════════════════════════════
// subagent_registry.go — 可续聊子 Agent（成员会话）注册表
//
// 背景（2026-08-28）：多智能体团队插件（agent-teams，参考 dsh-agent-teams
// 移植）需要「队长派生成员、成员各自续聊、成员空闲后自动续领任务」的能力。
// 本项目的「会话」本身就是可续聊 Agent（SessionManager 持久化历史 + 可重复
// Start 续跑），因此成员 = 一个独立会话（convID）：
//
//   队长（当前会话 convID）
//     └── 成员会话 conv_team_<team>_<member>_<rand>（独立历史、独立 persona）
//
// 本文件提供三件事：
//  1. 注册表：成员会话的身份/归属/状态（内存态，可按需由插件带 convID 重建）
//  2. Spawner 注入点：真正启动一轮会话的能力由 web 层注入（复用 /api/chat/send
//     的 LoopOpts 构造链：Provider/Registry/工具集白名单/审核配置）——
//     agent 包不反向依赖 web 层（与 SessionBridge 同构的注入模式）
//  3. 轮次队列 + 空闲事件：一个子会话同一时刻只跑一轮，后到消息排队；
//     轮次结束（EventDone/EventError）触发 EmitHostEvent("subagent/idle")，
//     JS 插件据此驱动任务调度（成员空闲 → 续领下一项就绪任务）
// ═══════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ─── 规格与记录 ─────────────────────────────────────────────

// SubAgentSpec 成员会话启动规格（插件侧 ctx.agents.start 的参数映射）。
type SubAgentSpec struct {
	ConvID     string   // 会话 ID（空=自动生成）
	ParentConv string   // 父会话（队长）ID
	Label      string   // 展示名（面板/事件用，如 team:member）
	Team       string   // 团队 ID（归属检索用，可空）
	Member     string   // 成员名（归属检索用，可空）
	Task       string   // 本轮输入（用户消息文本）
	System     string   // persona：追加到系统提示（空=宿主默认）
	Model      string   // 模型覆盖（空=会话默认模型）
	Provider   string   // 服务商覆盖（空=当前服务商；跨商按 models.json 解析端点与 Key）
	WsRoot     string   // 工作区根（空=父会话/全局主根）
	DenyTools  []string // 工具黑名单（成员不可用的工具，如队长专属工具）
	MaxIter    int      // 单轮最大迭代（<=0 用宿主默认）
}

// SubAgentRecord 成员会话记录（内存态）。
type SubAgentRecord struct {
	ConvID     string   `json:"convId"`
	ParentConv string   `json:"parentConvId"`
	Label      string   `json:"label"`
	Team       string   `json:"team"`
	Member     string   `json:"member"`
	System     string   `json:"system"`
	Model      string   `json:"model"`
	Provider   string   `json:"provider"`
	WsRoot     string   `json:"wsRoot"`
	DenyTools  []string `json:"denyTools"`
	CreatedAt  int64    `json:"createdAt"`
	LastActive int64    `json:"lastActiveAt"`
	Turns      int      `json:"turns"`     // 已发起轮次数
	State      string   `json:"state"`     // running | idle | stopped
	LastError  string   `json:"lastError"` // 最近一轮错误（空=无）
	Pending    int      `json:"pending"`   // 排队中的消息条数
}

// SubAgentSpawner 会话启动能力（web 层注入；agent 包只持接口）。
type SubAgentSpawner struct {
	// Start 启动一轮会话（异步：内部落盘用户消息 + SessionManager.Start）。
	Start func(spec SubAgentSpec) error
	// Stop 取消会话当前轮次。
	Stop func(convID string)
	// Running 会话是否正在跑。
	Running func(convID string) bool
	// LastAssistant 取会话最近一条助手正文（汇总/回报用；空=无）。
	LastAssistant func(convID string, wsRoot string) string
	// Models 模型目录（[{provider, model, label, isDefault}]）。
	Models func() []map[string]any
	// Current 当前主模型（{provider, model}）。
	Current func() map[string]any
}

var (
	subAgentMu       sync.Mutex
	subAgents        = map[string]*SubAgentRecord{} // convID → 记录
	subAgentQueue    = map[string][]string{}        // convID → 待发消息队列（FIFO）
	subAgentSpawner  *SubAgentSpawner
	subAgentBridgeOn bool
)

// SetSubAgentSpawner 注入会话启动能力（web 层启动时调用；重复注入覆盖）。
func SetSubAgentSpawner(s *SubAgentSpawner) {
	subAgentMu.Lock()
	subAgentSpawner = s
	subAgentMu.Unlock()
}

// SubAgentSpawnerReady 报告 spawner 是否已注入（插件可据此给出明确错误）。
func SubAgentSpawnerReady() bool {
	subAgentMu.Lock()
	defer subAgentMu.Unlock()
	return subAgentSpawner != nil && subAgentSpawner.Start != nil
}

// ─── convID 生成 ───────────────────────────────────────────

var subAgentIDSanitize = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// sanitizeIDPart 归一化 ID 片段（团队名/成员名可能是中文或含空格）。
func sanitizeIDPart(s string) string {
	s = subAgentIDSanitize.ReplaceAllString(strings.TrimSpace(s), "-")
	s = strings.Trim(s, "-")
	if len(s) > 24 {
		s = s[:24]
	}
	return s
}

// newSubAgentConvID 生成成员会话 ID（前缀 conv_ 与主会话同构，便于前端识别）。
func newSubAgentConvID(team, member string) string {
	parts := []string{"conv", "sub"}
	if p := sanitizeIDPart(team); p != "" {
		parts = append(parts, p)
	}
	if p := sanitizeIDPart(member); p != "" {
		parts = append(parts, p)
	}
	return fmt.Sprintf("%s_%d%03d", strings.Join(parts, "_"), time.Now().UnixNano(), rand.Intn(1000))
}

// ─── 对外 API（jsplugin ctx.agents 消费） ────────────────────

// SpawnSubAgent 创建成员会话并发起第一轮（异步执行，立即返回记录）。
func SpawnSubAgent(spec SubAgentSpec) (*SubAgentRecord, error) {
	if strings.TrimSpace(spec.Task) == "" {
		return nil, fmt.Errorf("子 Agent 启动失败：task（首轮输入）不能为空")
	}
	subAgentMu.Lock()
	spawner := subAgentSpawner
	if spawner == nil || spawner.Start == nil {
		subAgentMu.Unlock()
		return nil, fmt.Errorf("子 Agent 能力未就绪：会话启动器未注入（web 层未调用 SetSubAgentSpawner）")
	}
	convID := strings.TrimSpace(spec.ConvID)
	if convID == "" {
		convID = newSubAgentConvID(spec.Team, spec.Member)
	}
	spec.ConvID = convID
	now := time.Now().UnixMilli()
	rec := subAgents[convID]
	if rec == nil {
		rec = &SubAgentRecord{ConvID: convID, CreatedAt: now}
		subAgents[convID] = rec
	}
	rec.ParentConv = spec.ParentConv
	rec.Label = spec.Label
	rec.Team = spec.Team
	rec.Member = spec.Member
	rec.System = spec.System
	rec.Model = spec.Model
	rec.Provider = spec.Provider
	rec.WsRoot = spec.WsRoot
	rec.DenyTools = spec.DenyTools
	rec.State = "running"
	rec.LastActive = now
	rec.Turns++
	rec.LastError = ""
	subAgentMu.Unlock()

	ensureSubAgentEventBridge()

	if err := spawner.Start(spec); err != nil {
		subAgentMu.Lock()
		rec.State = "idle"
		rec.LastError = err.Error()
		subAgentMu.Unlock()
		return rec, err
	}
	log.Printf("[subagent] 已启动成员会话 conv=%s label=%s model=%s deny=%v", convID, spec.Label, spec.Model, spec.DenyTools)
	return snapshotSubAgent(rec), nil
}

// FollowupSubAgent 向已有成员会话投递一轮输入：空闲则立即跑，忙则排队（FIFO）。
// 返回 queued=true 表示已入队等待当前轮结束。
func FollowupSubAgent(convID, text string) (queued bool, err error) {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return false, fmt.Errorf("followup 失败：convId 不能为空")
	}
	if strings.TrimSpace(text) == "" {
		return false, fmt.Errorf("followup 失败：消息内容不能为空")
	}
	subAgentMu.Lock()
	spawner := subAgentSpawner
	if spawner == nil || spawner.Start == nil {
		subAgentMu.Unlock()
		return false, fmt.Errorf("子 Agent 能力未就绪：会话启动器未注入")
	}
	rec := subAgents[convID]
	if rec == nil {
		// 宿主重启后插件带着 team.json 里的 convID 回来续聊：按需重建记录。
		rec = &SubAgentRecord{ConvID: convID, CreatedAt: time.Now().UnixMilli(), State: "idle"}
		subAgents[convID] = rec
	}
	running := rec.State == "running"
	if !running && spawner.Running != nil && spawner.Running(convID) {
		running = true
		rec.State = "running"
	}
	if running {
		subAgentQueue[convID] = append(subAgentQueue[convID], text)
		rec.Pending = len(subAgentQueue[convID])
		subAgentMu.Unlock()
		ensureSubAgentEventBridge()
		return true, nil
	}
	spec := specFromRecord(rec, text)
	rec.State = "running"
	rec.Turns++
	rec.LastActive = time.Now().UnixMilli()
	subAgentMu.Unlock()

	ensureSubAgentEventBridge()
	if err := spawner.Start(spec); err != nil {
		subAgentMu.Lock()
		rec.State = "idle"
		rec.LastError = err.Error()
		subAgentMu.Unlock()
		return false, err
	}
	return false, nil
}

// specFromRecord 由记录还原启动规格（续聊沿用 persona/模型/工作区/黑名单）。
func specFromRecord(rec *SubAgentRecord, task string) SubAgentSpec {
	return SubAgentSpec{
		ConvID:     rec.ConvID,
		ParentConv: rec.ParentConv,
		Label:      rec.Label,
		Team:       rec.Team,
		Member:     rec.Member,
		Task:       task,
		System:     rec.System,
		Model:      rec.Model,
		Provider:   rec.Provider,
		WsRoot:     rec.WsRoot,
		DenyTools:  rec.DenyTools,
	}
}

// StopSubAgent 中断成员会话当前轮次并清空其排队消息。
func StopSubAgent(convID string) error {
	subAgentMu.Lock()
	spawner := subAgentSpawner
	rec := subAgents[convID]
	delete(subAgentQueue, convID)
	if rec != nil {
		rec.Pending = 0
		rec.State = "idle"
	}
	subAgentMu.Unlock()
	if spawner == nil || spawner.Stop == nil {
		return fmt.Errorf("子 Agent 能力未就绪：会话启动器未注入")
	}
	spawner.Stop(convID)
	return nil
}

// SubAgentInfo 查成员会话状态（实时校正 running 标志）。返回 nil 表示未登记。
func SubAgentInfo(convID string) *SubAgentRecord {
	subAgentMu.Lock()
	rec := subAgents[convID]
	spawner := subAgentSpawner
	subAgentMu.Unlock()
	if rec == nil {
		return nil
	}
	if spawner != nil && spawner.Running != nil {
		live := spawner.Running(convID)
		subAgentMu.Lock()
		if live {
			rec.State = "running"
		} else if rec.State == "running" {
			rec.State = "idle"
		}
		subAgentMu.Unlock()
	}
	subAgentMu.Lock()
	defer subAgentMu.Unlock()
	return snapshotSubAgent(rec)
}

// ListSubAgents 列出成员会话（parent 非空按队长过滤，team 非空按团队过滤）。
func ListSubAgents(parentConv, team string) []*SubAgentRecord {
	subAgentMu.Lock()
	spawner := subAgentSpawner
	out := make([]*SubAgentRecord, 0, len(subAgents))
	for _, rec := range subAgents {
		if parentConv != "" && rec.ParentConv != parentConv {
			continue
		}
		if team != "" && rec.Team != team {
			continue
		}
		out = append(out, rec)
	}
	subAgentMu.Unlock()
	res := make([]*SubAgentRecord, 0, len(out))
	for _, rec := range out {
		if spawner != nil && spawner.Running != nil {
			live := spawner.Running(rec.ConvID)
			subAgentMu.Lock()
			if live {
				rec.State = "running"
			} else if rec.State == "running" {
				rec.State = "idle"
			}
			subAgentMu.Unlock()
		}
		subAgentMu.Lock()
		res = append(res, snapshotSubAgent(rec))
		subAgentMu.Unlock()
	}
	return res
}

// SubAgentLastText 取成员会话最近助手正文（队长汇总用）。
func SubAgentLastText(convID string) string {
	subAgentMu.Lock()
	spawner := subAgentSpawner
	rec := subAgents[convID]
	wsRoot := ""
	if rec != nil {
		wsRoot = rec.WsRoot
	}
	subAgentMu.Unlock()
	if spawner == nil || spawner.LastAssistant == nil {
		return ""
	}
	return spawner.LastAssistant(convID, wsRoot)
}

// SubAgentModels 模型目录（插件 ctx.llm.models）。
func SubAgentModels() []map[string]any {
	subAgentMu.Lock()
	spawner := subAgentSpawner
	subAgentMu.Unlock()
	if spawner == nil || spawner.Models == nil {
		return nil
	}
	return spawner.Models()
}

// SubAgentCurrentModel 当前主模型（插件 ctx.llm.current）。
func SubAgentCurrentModel() map[string]any {
	subAgentMu.Lock()
	spawner := subAgentSpawner
	subAgentMu.Unlock()
	if spawner == nil || spawner.Current == nil {
		return nil
	}
	return spawner.Current()
}

// snapshotSubAgent 拷贝记录（调用方持锁）——避免把内部指针交给 JS 侧。
func snapshotSubAgent(rec *SubAgentRecord) *SubAgentRecord {
	cp := *rec
	if rec.DenyTools != nil {
		cp.DenyTools = append([]string(nil), rec.DenyTools...)
	}
	return &cp
}

// ─── 空闲事件桥（轮次结束 → 队列续发 + 插件事件） ─────────────

// ensureSubAgentEventBridge 惰性启动事件桥（首次有子 Agent 时启动，只启一次）。
func ensureSubAgentEventBridge() {
	subAgentMu.Lock()
	if subAgentBridgeOn {
		subAgentMu.Unlock()
		return
	}
	subAgentBridgeOn = true
	subAgentMu.Unlock()

	mgr := GlobalSessionManager()
	if mgr == nil {
		subAgentMu.Lock()
		subAgentBridgeOn = false
		subAgentMu.Unlock()
		log.Printf("[subagent] 事件桥未启动：SessionManager 未注入")
		return
	}
	ch := mgr.SubscribeAll()
	go func() {
		for ge := range ch {
			if ge.Event.Type != EventDone && ge.Event.Type != EventError {
				continue
			}
			subAgentMu.Lock()
			rec := subAgents[ge.ConvID]
			if rec == nil {
				subAgentMu.Unlock()
				continue
			}
			rec.State = "idle"
			rec.LastActive = time.Now().UnixMilli()
			if ge.Event.Type == EventError {
				rec.LastError = ge.Event.Content
			}
			// 队列续发：本轮结束后立刻投递下一条排队消息。
			var next string
			if q := subAgentQueue[ge.ConvID]; len(q) > 0 {
				next = q[0]
				if len(q) == 1 {
					delete(subAgentQueue, ge.ConvID)
				} else {
					subAgentQueue[ge.ConvID] = q[1:]
				}
				rec.Pending = len(subAgentQueue[ge.ConvID])
			}
			payload := map[string]any{
				"convId":       rec.ConvID,
				"parentConvId": rec.ParentConv,
				"label":        rec.Label,
				"team":         rec.Team,
				"member":       rec.Member,
				"turns":        rec.Turns,
				"pending":      rec.Pending,
				"error":        rec.LastError,
			}
			subAgentMu.Unlock()

			if next != "" {
				if _, err := FollowupSubAgent(ge.ConvID, next); err != nil {
					log.Printf("[subagent] 队列续发失败 conv=%s: %v", ge.ConvID, err)
				}
			}
			// 通知插件：成员一轮结束（调度器据此续领任务）。
			if ph := GetGlobalPluginHost(); ph != nil {
				ph.EmitHostEvent("subagent/idle", payload)
			}
		}
	}()
	log.Printf("[subagent] 事件桥已启动（监听会话轮次结束 → subagent/idle）")
}

// ─── SessionManager 全局引用（web 层注入） ───────────────────

var (
	globalSessionMgrMu sync.RWMutex
	globalSessionMgr   *SessionManager
)

// SetGlobalSessionManager 注入全局会话管理器（web 层启动时调用一次）。
func SetGlobalSessionManager(m *SessionManager) {
	globalSessionMgrMu.Lock()
	globalSessionMgr = m
	globalSessionMgrMu.Unlock()
}

// GlobalSessionManager 取全局会话管理器（未注入返回 nil）。
func GlobalSessionManager() *SessionManager {
	globalSessionMgrMu.RLock()
	defer globalSessionMgrMu.RUnlock()
	return globalSessionMgr
}
