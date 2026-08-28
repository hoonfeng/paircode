// ═══════════════════════════════════════════════════════════════
// session_bridge.go — 会话桥（ask_user/task_create 插件化路由）
//
// 背景（2026-08-16）：ask_user/task_create 原为会话级注册工具（Handler 闭包
// 捕获 sess.askCh / sess.ConvID），依赖会话状态无法直接外置。但「会话绑定」
// 本身不构成障碍——Loop 上下文（runCtx）携带 convID 后，插件工具执行时可经
// jsToolToGo 注入 `_convID` 内部参数，再经 ctx.hostTool 路由回宿主执行器：
//   插件（schema 编排）→ ctx.hostTool.exec('ask_user', {...,_convID})
//     → 路由执行器 → 会话桥（SessionBridge）→ SessionManager 按 convID 路由
// 由此 ask_user/task_create 与其余 17 个 hostTool 插件同构（编排在插件、
// 能力在宿主），会话级状态经桥访问，多会话并发不串（按 _convID 精确路由）。
//
// 本文件提供：
//   - 会话上下文 key：WithSessionConvID / SessionConvID（Loop ctx 链传递）
//   - SessionBridge：web 层注入的会话能力（按 convID 路由）
//   - archiveSessionTools：ask_user/task_create 路由执行器存档（hostTool 索引）
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"encoding/json"
	"fmt"
)

// ─── 会话上下文（ctx 链传递 convID） ──────────────────────────

type sessionCtxKey struct{}
type sessionWsCtxKey struct{}

// WithSessionConvID 向 ctx 注入会话 ID（SessionManager.Start 的 runCtx 设置，
// Loop 内 Registry.Execute 的 ctx 同源，JS 工具包装时可提取）。
func WithSessionConvID(ctx context.Context, convID string) context.Context {
	return context.WithValue(ctx, sessionCtxKey{}, convID)
}

// SessionConvID 从 ctx 提取会话 ID（无则空串）。ctx 可为 nil（测试/无会话上下文）。
func SessionConvID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(sessionCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// WithSessionWorkspaceRoot 向 ctx 注入会话绑定的工作区根路径。
// ★ 2026-08-23 工作区隔离（重大 BUG）：工具执行必须绑定「会话启动时的工作区」，
//   而不能读全局当前工作区（core.Folders/WorkspaceRoots——切换工作区时被覆写，
//   正在执行的对话会因此把工具跑进新工作区，造成文件读写/命令执行串台）。
//   SessionManager.Start 把 opts.WorkspaceRoot 注入 runCtx，与 convID 平行；
//   Loop → Registry.Execute → 插件工具包装（jsToolToGo）沿 ctx 链提取。
func WithSessionWorkspaceRoot(ctx context.Context, wsRoot string) context.Context {
	return context.WithValue(ctx, sessionWsCtxKey{}, wsRoot)
}

// SessionWorkspaceRoot 从 ctx 提取会话绑定的工作区根（无则空串）。
// ctx 可为 nil（测试/无会话上下文）；返回空串时调用方回落全局/装载快照。
func SessionWorkspaceRoot(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(sessionWsCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// sessionRootsOrGlobal 返回工具调用场景的工作区根列表：会话绑定的工作区根（单元素）
// 优先（工作区隔离），无会话上下文时回落全局 WorkspaceRoots。
func sessionRootsOrGlobal(ctx context.Context) []string {
	if ctx != nil {
		if r := SessionWorkspaceRoot(ctx); r != "" {
			return []string{r}
		}
	}
	return WorkspaceRoots
}

// ─── 会话桥（web 层注入，agent 包不依赖 web 层实例） ──────────

// SessionBridge 宿主会话能力（按 convID 路由，多会话安全）。
// web 层（cmd/companion/web_server.go）在启动时注入；nil 时路由执行器报错
// （提示未注入），插件工具仍可见但执行返回明确错误——不会静默失效。
type SessionBridge struct {
	// WaitAnswer 等待用户回答（ask_user）：从 convID 对应会话的 askCh 读，
	// 前端 /api/answer → SendAnswer 写入。阻塞直到回答/超时/ctx 取消。
	WaitAnswer func(ctx context.Context, convID string) (string, error)
	// WaitAnswers 等待用户回答数组（Round3 ⑤ 多问题；单问题=单元素数组）。
	WaitAnswers func(ctx context.Context, convID string) ([]AskAnswer, error)
	// GetWorkspaceRoot 取会话工作区根（task_create 持久化用；无会话返回空串）。
	GetWorkspaceRoot func(convID string) string
}

var sessionBridge *SessionBridge

// init：包加载即存档 ask_user/task_create 路由执行器（幂等）——
// 不依赖 web 层注入顺序；SetSessionBridge 只注入桥函数，handler 在桥为
// nil 时明确报错（防静默失效）。
func init() {
	archiveSessionTools()
	archiveGoalTools()      // Round3 ③.1：goal 工具路由执行器（hostTool 索引）
	archiveWorkflowTool()   // Round3 ③.3：workflow 工具路由执行器（hostTool 索引）
}

// SetSessionBridge 注入会话桥（web 层启动时调用；重复注入覆盖）。
func SetSessionBridge(b *SessionBridge) {
	sessionBridge = b
	archiveSessionTools() // 幂等：路由执行器同构重复存档覆盖
}

// ─── 路由执行器存档 ──────────────────────────────────────────

// archiveSessionTools 将 ask_user/task_create 的「路由版」执行器存档到
// hostTool 索引。与 SessionManager.Start 的会话级注册（闭包 sess）不同：
// 路由版不闭包任何会话，从 args["_convID"] 取会话 ID 后经 SessionBridge
// 路由到目标会话——插件工具 execute 可安全复用，多会话并发不串。
//
// ★ 优先级：插件注册同名工具（tool-system）接管 agent 可见面后，
// SessionManager.Start 检测 reg 中已存在则不再注册会话级版本（注册条件化），
// 插件工具 execute → ctx.hostTool.exec('ask_user'/'task_create') → 本路由版。
// 插件停用/未装载时 Start 仍注册会话级版本兜底，行为与旧版一致。
func archiveSessionTools() {
	ArchiveHostTool(&Tool{
		Name:       "ask_user",
		SystemTool: true,
		Description: "向用户提问并等待回答（用于关键决策、歧义澄清，别滥用）。" +
			"question 必填（或 questions 数组多问题）；askType 可选(text/single/multi/single-with-input)，默认 text 纯文本输入；" +
			"★ options 当 askType 为 single/multi/single-with-input 时必须提供（至少 2 个，如 [\"方案A\",\"方案B\"]），text 时可省略。" +
			"多问题：questions:[{id, question, options?, multi_select?}]（questions 优先，缺省回落单问题）。调用会阻塞直到用户回答。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{"type": "string", "description": "向用户提出的问题（单问题路径；与 questions 二选一）"},
				"askType":  map[string]any{"type": "string", "enum": []string{"text", "single", "multi", "single-with-input"}, "description": "提问类型：text(纯文本)/single(单选)/multi(多选)/single-with-input(单选+自由输入)"},
				"options":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "选择类问题用：可选项列表"},
				"questions": map[string]any{
					"type":        "array",
					"description": "多问题数组（与 question 二选一；questions 优先）。每项 {id, question, options?, multi_select?}",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":           map[string]any{"type": "string", "description": "问题 ID（回答回灌时对应）"},
							"question":     map[string]any{"type": "string", "description": "问题文本"},
							"options":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "选项（选择类问题）"},
							"multi_select": map[string]any{"type": "boolean", "description": "是否多选（默认 false 单选）"},
						},
						"required": []string{"id", "question"},
					},
				},
			},
			"required": []string{},
		},
		RequiresApproval: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "_convID")
			if convID == "" {
				return "", fmt.Errorf("ask_user：缺少会话标识（_convID 未注入）——插件工具须经宿主工具执行链调用")
			}
			if sessionBridge == nil {
				return "", fmt.Errorf("ask_user：会话桥未注入（web 层未调用 SetSessionBridge）")
			}
			// ★ Round3 ⑤：多问题路径（questions 优先；回答数组 JSON 回灌）
			if qs := askQuestionsFromArgs(args); len(qs) > 0 {
				if sessionBridge.WaitAnswers == nil {
					return "", fmt.Errorf("ask_user(多问题)：会话桥未注入 WaitAnswers（web 层未升级）")
				}
				answers, err := sessionBridge.WaitAnswers(ctx, convID)
				if err != nil {
					return "", err
				}
				b, _ := json.MarshalIndent(map[string]any{"answers": answers}, "", "  ")
				return string(b), nil
			}
			if sessionBridge.WaitAnswer == nil {
				return "", fmt.Errorf("ask_user：会话桥未注入 WaitAnswer（web 层未调用 SetSessionBridge）")
			}
			return sessionBridge.WaitAnswer(ctx, convID)
		},
	})

	ArchiveHostTool(&Tool{
		Name:       "task_create",
		SystemTool: true,
		UsageGuide: "创建子任务并追踪执行进度。复杂任务（3+ 步）必须拆解为子任务，每完成一项更新状态（in_progress→completed）。依赖项用 dependencies 参数关联。比手动记清单更可靠（持久化到磁盘+状态自动管理）。",
		Description: "创建新的子任务。创建后必须立即执行该任务：先调用 task_update 标记为 in_progress 开始执行，" +
			"执行完成后调用 task_update 标记为 completed 并说明结果。重复此流程直到所有子任务完成。",
		Parameters: objSchema(props{
			"subject":      strProp("任务标题，用祈使句（如\"修复登录超时\"）"),
			"description":  strProp("详细描述：做什么、涉及哪些文件。不要包含文件原始内容，只写摘要。"),
			"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "依赖的任务 ID 列表"},
		}, "subject", "description"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			convID := argStr(args, "_convID")
			if convID == "" {
				return "", fmt.Errorf("task_create：缺少会话标识（_convID 未注入）——插件工具须经宿主工具执行链调用")
			}
			if sessionBridge == nil || sessionBridge.GetWorkspaceRoot == nil {
				return "", fmt.Errorf("task_create：会话桥未注入（web 层未调用 SetSessionBridge）")
			}
			root := sessionBridge.GetWorkspaceRoot(convID)
			tm := UseTaskManager(root)
			task := tm.Create(argStr(args, "subject"), argStr(args, "description"), argStrSlice(args, "dependencies"), convID)
			return fmt.Sprintf("✅ 已创建任务 [%s] %s\n> %s\n\n状态: ⏳ 待执行\nID: `%s`", task.ID, task.Subject, task.Description, task.ID), nil
		},
	})
}
