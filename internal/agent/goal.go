// ═══════════════════════════════════════════════════════════════
// goal.go — 宿主 goal 机制（Round3 ③.1，对齐 DSH harness goal 语义）
//
// 原则：机制（状态机/会话编排）在宿主 Loop 层，工具面（schema/描述）在
// 插件层（.pair/plugins/tool-goal）。无 goal / 无插件时零行为变化。
//
// 语义对齐 DSH：
//   - create_goal：直接接收 objective（不做 LLM 推断，减少不确定面）
//   - get_goal：返回 goal_id/revision/objective/phase/rounds/roundLimit/
//     blockerReason/armed
//   - update_goal：action ∈ {edit,pause,resume,complete,blocked}；
//     revision 冲突拒绝（乐观锁）
//   - 自动续轮：会话 Run 结束后，goal Armed && phase 非终态 &&
//     Rounds < RoundLimit → 自动发起下一轮（continuation 消息）
//   - 同一阻塞条件连续 ≥3 轮 → 自动 blocked（blocked_reason 记录）
//
// 持久化：<wsRoot>/.pair/goals/<convID>.json（防宿主重启丢目标）。
// 完整实施记录见 docs/plugin-round3-plan.md §10（Round3 t2）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ─── Goal 状态机 ──────────────────────────────────────────

// Goal 阶段（phase）。
const (
	GoalPhaseActive    = "active"    // 推进中（默认）
	GoalPhaseCompleted = "completed" // 已完成（update_goal action=complete）
	GoalPhaseBlocked   = "blocked"   // 阻塞（update_goal action=blocked / 自动连续阻塞）
)

// Goal 一个会话目标（会话级状态机 + 磁盘持久化）。
type Goal struct {
	ID            string `json:"id"`
	Revision      int    `json:"revision"`
	Objective     string `json:"objective"`
	Phase         string `json:"phase"` // active/completed/blocked
	Rounds        int    `json:"rounds"`
	RoundLimit    int    `json:"roundLimit"` // 0=不限（默认 create 给 3，防无限续轮）
	BlockerReason string `json:"blockerReason,omitempty"`
	Armed         bool   `json:"armed"` // pause=false 停续轮；resume=true 重挂

	// 运行时（不持久化）：阻塞连续计数
	lastErr    string `json:"-"`
	errStreak  int    `json:"-"`
}

// Active 是否处于推进态（非终态且武装）。
func (g *Goal) Active() bool {
	if g == nil {
		return false
	}
	return g.Armed && g.Phase == GoalPhaseActive
}

// Terminal 是否终态（completed/blocked）。
func (g *Goal) Terminal() bool {
	if g == nil {
		return true
	}
	return g.Phase == GoalPhaseCompleted || g.Phase == GoalPhaseBlocked
}

// ContinueMessage 返回下一轮 continuation 消息；空串 = 不续轮。
// 条件：Armed && 非终态 && （RoundLimit<=0 || Rounds < RoundLimit）。
func (g *Goal) ContinueMessage() string {
	if !g.Active() {
		return ""
	}
	if g.RoundLimit > 0 && g.Rounds >= g.RoundLimit {
		return ""
	}
	limitTxt := fmt.Sprintf("%d", g.RoundLimit)
	if g.RoundLimit <= 0 {
		limitTxt = "不限"
	}
	return fmt.Sprintf("（goal 自动续轮 %d/%s）继续推进目标：%s。当前阶段：%s。"+
		"请持续推进直到目标完成；完成后调用 update_goal 提交 complete，遇到不可克服障碍用 blocked 说明。",
		g.Rounds, limitTxt, g.Objective, g.Phase)
}

// goalSystemMarker 系统提示注入标记（幂等：已注入不重复追加）。
const goalSystemMarker = "【当前目标】"

// goalSystemSection 目标上下文段（注入系统提示，对齐 DSH「同会话完成目标」语义）。
func goalSystemSection(g *Goal) string {
	limitTxt := fmt.Sprintf("%d", g.RoundLimit)
	if g.RoundLimit <= 0 {
		limitTxt = "不限"
	}
	return fmt.Sprintf("%s\n目标：%s\n阶段：%s\n已进行轮次：%d/%s（自动续轮中，完成任务后调用 update_goal action=complete）",
		goalSystemMarker, g.Objective, g.Phase, g.Rounds, limitTxt)
}

// ─── GoalManager（会话级持有 + 持久化） ─────────────────────

// GoalManager goal 状态管理器：按 (wsRoot, convID) 路由，内存优先、磁盘持久化。
// ★ 包级单例（与 sessionBridge 同构）：SessionManager 续轮与 goal 工具执行器共用。
type GoalManager struct {
	mu    sync.Mutex
	goals map[string]*Goal // key: wsRoot + "\x00" + convID
}

var goalManager = NewGoalManager()

// NewGoalManager 创建空管理器（测试可直接构造独立实例）。
func NewGoalManager() *GoalManager {
	return &GoalManager{goals: map[string]*Goal{}}
}

func goalKey(wsRoot, convID string) string { return wsRoot + "\x00" + convID }

// goalPath <wsRoot>/.pair/goals/<convID>.json（wsRoot 空 = 仅内存不持久化）。
func goalPath(wsRoot, convID string) string {
	if wsRoot == "" {
		return ""
	}
	return filepath.Join(wsRoot, ".pair", "goals", convID+".json")
}

func (gm *GoalManager) loadLocked(wsRoot, convID string) *Goal {
	if g, ok := gm.goals[goalKey(wsRoot, convID)]; ok {
		return g
	}
	p := goalPath(wsRoot, convID)
	if p == "" {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var g Goal
	if err := json.Unmarshal(data, &g); err != nil {
		return nil
	}
	gm.goals[goalKey(wsRoot, convID)] = &g
	return &g
}

func (gm *GoalManager) persist(g *Goal, wsRoot string) {
	p := goalPath(wsRoot, g.ID)
	if p == "" {
		return
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

// Get 取会话目标（无则 nil；内存优先，miss 读盘）。
func (gm *GoalManager) Get(wsRoot, convID string) *Goal {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.loadLocked(wsRoot, convID)
}

// Create 创建目标（已存在返回错误）。
func (gm *GoalManager) Create(wsRoot, convID, objective string, roundLimit int) (*Goal, error) {
	objective = strings.TrimSpace(objective)
	if objective == "" {
		return nil, fmt.Errorf("create_goal：objective 不能为空")
	}
	if roundLimit < 0 {
		roundLimit = 0
	}
	gm.mu.Lock()
	defer gm.mu.Unlock()
	if gm.loadLocked(wsRoot, convID) != nil {
		return nil, fmt.Errorf("create_goal：会话 %s 已有目标（可 update_goal edit 修改或 complete 结束）", convID)
	}
	g := &Goal{
		ID:         convID,
		Revision:   1,
		Objective:  objective,
		Phase:      GoalPhaseActive,
		RoundLimit: roundLimit,
		Armed:      true,
	}
	gm.goals[goalKey(wsRoot, convID)] = g
	gm.persist(g, wsRoot)
	return g, nil
}

// Update 更新目标（revision 乐观锁冲突拒绝）。action ∈ edit/pause/resume/complete/blocked。
func (gm *GoalManager) Update(wsRoot, convID string, revision int, action, objective string, maxRounds int, blockedReason string) (*Goal, error) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	g := gm.loadLocked(wsRoot, convID)
	if g == nil {
		return nil, fmt.Errorf("update_goal：会话 %s 无目标（先 create_goal）", convID)
	}
	if revision > 0 && revision != g.Revision {
		return nil, fmt.Errorf("update_goal：revision 冲突（当前 %d，提交 %d）——目标已被其他轮次更新，请先 get_goal 取最新", g.Revision, revision)
	}
	switch action {
	case "edit":
		if strings.TrimSpace(objective) != "" {
			g.Objective = strings.TrimSpace(objective)
		}
		if maxRounds >= 0 {
			g.RoundLimit = maxRounds
		}
	case "pause":
		g.Armed = false
	case "resume":
		if g.Terminal() {
			return nil, fmt.Errorf("update_goal：目标已处于终态（%s），不可 resume", g.Phase)
		}
		g.Armed = true
	case "complete":
		g.Phase = GoalPhaseCompleted
		g.BlockerReason = ""
	case "blocked":
		g.Phase = GoalPhaseBlocked
		g.BlockerReason = strings.TrimSpace(blockedReason)
		if g.BlockerReason == "" {
			g.BlockerReason = "（未说明原因）"
		}
	default:
		return nil, fmt.Errorf("update_goal：未知 action %q（可用 edit/pause/resume/complete/blocked）", action)
	}
	g.Revision++
	gm.persist(g, wsRoot)
	return g, nil
}

// MarkRound 记录一轮结束（Rounds++、阻塞连续计数），返回更新后的 goal。
// 同一阻塞条件连续 ≥3 轮 → 自动 blocked（blocked_reason 记录，对齐 DSH 语义）。
func (gm *GoalManager) MarkRound(wsRoot, convID string, roundErr error) *Goal {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	g := gm.loadLocked(wsRoot, convID)
	if g == nil {
		return nil
	}
	g.Rounds++
	if roundErr != nil {
		reason := roundErr.Error()
		if reason == g.lastErr {
			g.errStreak++
		} else {
			g.lastErr = reason
			g.errStreak = 1
		}
		if g.errStreak >= 3 && g.Phase == GoalPhaseActive {
			g.Phase = GoalPhaseBlocked
			g.BlockerReason = "连续 3 轮同一阻塞条件：" + truncRunesAgent(reason, 200)
			g.Revision++
		}
	} else {
		g.lastErr = ""
		g.errStreak = 0
	}
	gm.persist(g, wsRoot)
	return g
}

// goalWorkspaceRoot 取目标持久化根：args._wsRoot（会话工具链注入）优先，
// 回落会话绑定 ctx，再回落会话桥。
func goalWorkspaceRoot(ctx context.Context, args map[string]any, convID string) string {
	if r := argStr(args, "_wsRoot"); r != "" {
		return r
	}
	if r := SessionWorkspaceRoot(ctx); r != "" {
		return r
	}
	if sessionBridge != nil && sessionBridge.GetWorkspaceRoot != nil {
		return sessionBridge.GetWorkspaceRoot(convID)
	}
	return ""
}

// ─── 路由执行器存档（插件工具 execute → ctx.hostTool.exec） ─────

// archiveGoalTools 将 create_goal/get_goal/update_goal 的路由执行器存档到
// hostTool 索引（与 ask_user 同构：编排在插件、能力在宿主；_convID 路由）。
func archiveGoalTools() {
	ArchiveHostTool(&Tool{
		Name:       "create_goal",
		SystemTool: true,
		Description: "创建同会话完成目标（对齐 DSH harness goal）。objective 必填（直接给出目标，不做推断）；" +
			"max_goal_rounds 可选（自动续轮上限，默认 3）。创建后会话将在每轮结束后自动续轮推进，直到 update_goal complete/blocked 或达轮次上限。",
		Parameters: objSchema(props{
			"objective":       strProp("目标描述（祈使句，直接给出，如「修复登录超时 bug」）"),
			"max_goal_rounds": intProp("可选：自动续轮上限（默认 3；0=不限——慎用，会无限续轮）"),
		}, "objective"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "_convID")
			if convID == "" {
				return "", fmt.Errorf("create_goal：缺少会话标识（_convID 未注入）——插件工具须经宿主工具执行链调用")
			}
			objective := argStr(args, "objective")
			if strings.TrimSpace(objective) == "" {
				return "", fmt.Errorf("create_goal：objective 不能为空")
			}
			limit := argInt(args, "max_goal_rounds", 3) // 缺省 3；显式 0 = 不限（慎用）
			g, err := goalManager.Create(goalWorkspaceRoot(ctx, args, convID), convID, objective, limit)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("✅ 目标已创建：%s\nID: %s（revision %d）｜自动续轮上限：%d 轮｜完成后调用 update_goal action=complete",
				g.Objective, g.ID, g.Revision, g.RoundLimit), nil
		},
	})

	ArchiveHostTool(&Tool{
		Name:       "get_goal",
		SystemTool: true,
		Description: "读取当前会话目标（goal_id/revision/objective/phase/rounds/roundLimit/blockerReason/armed）。无目标返回提示。",
		Parameters: objSchema(props{}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "_convID")
			if convID == "" {
				return "", fmt.Errorf("get_goal：缺少会话标识（_convID 未注入）")
			}
			g := goalManager.Get(goalWorkspaceRoot(ctx, args, convID), convID)
			if g == nil {
				return "（当前会话无目标；可用 create_goal 创建）", nil
			}
			b, _ := json.MarshalIndent(map[string]any{
				"goal_id":       g.ID,
				"revision":      g.Revision,
				"objective":     g.Objective,
				"phase":         g.Phase,
				"rounds":        g.Rounds,
				"roundLimit":    g.RoundLimit,
				"blockerReason": g.BlockerReason,
				"armed":         g.Armed,
			}, "", "  ")
			return string(b), nil
		},
	})

	ArchiveHostTool(&Tool{
		Name:       "update_goal",
		SystemTool: true,
		Description: "更新当前会话目标（对齐 DSH harness goal update）。action ∈ {edit,pause,resume,complete,blocked}；" +
			"revision 必传（乐观锁，冲突拒绝）。edit 可改 objective/max_goal_rounds；pause 停续轮、resume 重挂；" +
			"complete 标记完成；blocked 标记阻塞（blocked_reason 必填说明）。",
		Parameters: objSchema(props{
			"goal_id":        strProp("目标 ID（=会话 ID；get_goal 可查）"),
			"revision":       intProp("当前 revision（get_goal 返回；冲突时拒绝）"),
			"action":         strProp("edit / pause / resume / complete / blocked"),
			"objective":      strProp("edit 用：新目标描述（可选）"),
			"max_goal_rounds": intProp("edit 用：新自动续轮上限（可选）"),
			"blocked_reason": strProp("blocked 用：阻塞原因（必填）"),
		}, "goal_id", "revision", "action"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "_convID")
			if convID == "" {
				return "", fmt.Errorf("update_goal：缺少会话标识（_convID 未注入）")
			}
			action := argStr(args, "action")
			rev := argInt(args, "revision", 0)
			maxRounds := -1
			if v, ok := args["max_goal_rounds"]; ok {
				switch n := v.(type) {
				case float64:
					maxRounds = int(n)
				case int:
					maxRounds = n
				}
			}
			g, err := goalManager.Update(goalWorkspaceRoot(ctx, args, convID), convID, rev, action,
				argStr(args, "objective"), maxRounds, argStr(args, "blocked_reason"))
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("✅ 目标已更新：%s（phase=%s, revision=%d, armed=%v, rounds=%d/%d）",
				g.Objective, g.Phase, g.Revision, g.Armed, g.Rounds, g.RoundLimit), nil
		},
	})
}
