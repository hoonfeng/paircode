package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrConsecToolError 连续多轮工具执行失败，由 Loop.Run 返回，桥接层应据此终止本轮后续阶段。
var ErrConsecToolError = errors.New("连续 3 轮工具执行失败，已停止")

// ErrMaxIterations 已达最大迭代数仍未 [FINAL]，由 Loop.Run 返回。
var ErrMaxIterations = errors.New("已达最大迭代数，停止")

// EventType 循环对外广播的事件类型（供 UI 流式展示）。
type EventType string

const (
	EventThinking   EventType = "thinking"    // LLM 思考链增量
	EventContent    EventType = "content"     // LLM 正文增量
	EventToolCall   EventType = "tool_call"   // 即将执行某工具
	EventToolResult EventType = "tool_result" // 工具结果回来
	EventFinal      EventType = "final"       // 任务完成（最终答复）
	EventError      EventType = "error"       // 出错/止损
	EventCompacted  EventType = "compacted"   // 上下文已压缩（中段老消息压成摘要；UI 显示一行素色提示）
	EventEvaluation EventType = "evaluation"  // 任务评测评分（完成后评测模型打分；UI 显示评分卡）
	EventCircling   EventType = "circling"    // 检测到重复绕圈，已注入「换思路」提示打破死循环（UI 显示一行提示）
	// EventApproval 等待用户审批某次写类工具调用。由宿主（UI 桥）在 Approve 钩子里 emit，
	// loop 自身不直接发——loop 只通过 Approve 回调阻塞等待裁决（见 agent_bridge.go）。
	EventApproval EventType = "approval"
)

// Event 一条循环事件。
type Event struct {
	Type    EventType
	Content string // thinking/content/final/error/tool_result 的文本
	Tool    string // tool_call/tool_result 的工具名
	Args    string // tool_call 的参数 JSON
	CallID  string
}

const finalMarker = "[FINAL]"

// Loop TAOR 编排器：think(LLM 决策)→act(执行工具)→observe(结果回灌)→repeat。
// 停止：assistant 输出 [FINAL] 或无工具调用（即给出文本答复）/ 连续 3 轮工具全错 / 达最大迭代 / 外部取消。
type Loop struct {
	Provider      Provider
	Registry      *Registry
	System        string // 系统提示词
	MaxIterations int    // 默认 30
	OnEvent       func(Event)
	// Approve 审批钩子（可空）。设置后，每次执行 RequiresApproval 的写类工具前调用它，
	// 返回 (false, feedback) 即拒绝执行——feedback 非空则作为观察回灌（让模型据此改道），空则用默认拒绝语。
	// 只读工具永不经过它。nil = 自动审核（全部放行）。宿主可在此阻塞等用户点「允许/拒绝」(人工审核)，
	// 或调审核模型自动裁决并回灌建议(AI 审核)（见 agent_bridge.go）。
	Approve func(ctx context.Context, tc ToolCall) (approved bool, feedback string)

	// ── 上下文压缩（可空；复刻参考 context/manager.ts，见 compress.go）──
	// MaxContextTokens>0 时启用：每次 LLM 调用前，若 tokens/Max 超阈值，把中段老消息压成一条摘要。
	// Compressor 非空→用它（轻量压缩模型）做 LLM 摘要，否则/失败→规则式摘要。
	Compressor       Provider
	MaxContextTokens int

	lastPromptTokens int // 上一轮 API 实测 prompt_tokens（驱动压缩阈值，比纯估算可信）
	compactCooldown  int // 压缩后冷却剩余轮数（防每轮重复压缩，复刻参考 refreshCooldown）

	recentCalls []toolSig // 最近若干次工具调用签名+成败（绕圈检测，见 circling.go）
}

func (l *Loop) emit(e Event) {
	if l.OnEvent != nil {
		l.OnEvent(e)
	}
}

// Run 跑一轮任务。history 为先前对话（可空，不含本次 task）。
// 返回在 history 基础上追加了 system(首轮)/user/assistant/tool 等本轮全部消息的完整对话。
func (l *Loop) Run(ctx context.Context, task string, history []Message) ([]Message, error) {
	max := l.MaxIterations
	if max <= 0 {
		max = 30
	}
	msgs := make([]Message, 0, len(history)+4)
	if l.System != "" && !hasSystem(history) {
		msgs = append(msgs, Message{Role: RoleSystem, Content: l.System})
	}
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: RoleUser, Content: task})

	tools := l.Registry.Definitions()
	consecErr := 0

	for iter := 0; iter < max; iter++ {
		if err := ctx.Err(); err != nil {
			return msgs, err // 外部取消
		}

		msgs = l.maybeCompact(ctx, msgs) // 超窗口阈值则把中段老消息压成摘要（见 compress.go）

		// ── THINK：LLM 决策（流式 thinking/content 经事件外发）──
		assistant, err := l.Provider.Chat(ctx, msgs, tools, func(c Chunk) {
			if c.Reasoning != "" {
				l.emit(Event{Type: EventThinking, Content: c.Reasoning})
			}
			if c.Content != "" {
				l.emit(Event{Type: EventContent, Content: c.Content})
			}
			if c.Usage != nil && c.Usage.PromptTokens > 0 {
				l.lastPromptTokens = c.Usage.PromptTokens // 实测用量驱动下轮压缩判定
			}
		})
		if err != nil {
			l.emit(Event{Type: EventError, Content: err.Error()})
			return msgs, err
		}
		msgs = append(msgs, assistant)

		// ── 完成判定：[FINAL] 标记，或无工具调用（视作已给出文本答复）──
		if strings.Contains(assistant.Content, finalMarker) || len(assistant.ToolCalls) == 0 {
			l.emit(Event{Type: EventFinal, Content: stripFinal(assistant.Content)})
			return msgs, nil
		}

		// ── ACT + OBSERVE：依次执行工具，结果作 role=tool 消息回灌 ──
		iterErr := false
		for _, tc := range assistant.ToolCalls {
			l.emit(Event{Type: EventToolCall, Tool: tc.Function.Name, Args: tc.Function.Arguments, CallID: tc.ID})

			// 审批门：写类工具（RequiresApproval）在手动审核下需用户批准。被拒则不执行，
			// 把拒绝作为观察回灌（让模型改道，而非当成工具错误计入连续失败）。
			if l.Approve != nil {
				if tool, ok := l.Registry.Get(tc.Function.Name); ok && tool.RequiresApproval {
					if approved, feedback := l.Approve(ctx, tc); !approved {
						rej := strings.TrimSpace(feedback)
						if rej == "" {
							rej = "用户拒绝了此操作。请勿重试该操作；改用其他方式达成目标，或先向用户说明你为何需要它。"
						}
						l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: rej, CallID: tc.ID})
						msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: rej})
						l.trackCall(tc.Function.Name, tc.Function.Arguments, true) // 被拒也算一次未成、计入绕圈检测
						continue
					}
				}
			}

			result, terr := l.Registry.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if terr != nil {
				result = "Error: " + terr.Error()
				iterErr = true
			}
			l.emit(Event{Type: EventToolResult, Tool: tc.Function.Name, Content: result, CallID: tc.ID})
			msgs = append(msgs, Message{Role: RoleTool, ToolCallID: tc.ID, Name: tc.Function.Name, Content: result})
			l.trackCall(tc.Function.Name, tc.Function.Arguments, terr != nil || strings.HasPrefix(strings.TrimSpace(result), "Error:"))
		}

		// 绕圈检测：同一操作反复失败/反复执行 → 注入「换思路」提示打破死循环（见 circling.go）。
		if nudge := l.detectCircling(); nudge != "" {
			l.emit(Event{Type: EventCircling, Content: "检测到重复操作/反复失败，已提示 Agent 换思路打破死循环"})
			msgs = append(msgs, Message{Role: RoleUser, Content: nudge})
			l.recentCalls = nil // 提示后清零，给新思路一个干净起点
		}

		// 连续 3 轮工具全有错 → 止损停（复刻参考源 3-consecutive-error）。
		// 返回 sentinel 错误供桥接层判断，避免误以为正常完成而继续验证/评测阶段。
		if iterErr {
			if consecErr++; consecErr >= 3 {
				l.emit(Event{Type: EventError, Content: ErrConsecToolError.Error()})
				return msgs, ErrConsecToolError
			}
		} else {
			consecErr = 0
		}
	}
	l.emit(Event{Type: EventError, Content: ErrMaxIterations.Error()})
	return msgs, ErrMaxIterations
}

func hasSystem(msgs []Message) bool {
	for _, m := range msgs {
		if m.Role == RoleSystem {
			return true
		}
	}
	return false
}

func stripFinal(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, finalMarker, ""))
}

// DefaultSystemPrompt 复刻参考源核心铁律的系统提示词（中文 lock / 改前 read / 工作区限定 / [FINAL]）。
// roots 为工作区所有根目录（支持多根工作区）；roots[0] 为主根。
func DefaultSystemPrompt(roots []string) string {
	rootInfo := "根目录: " + roots[0]
	if len(roots) > 1 {
		rootInfo += "\n工作区包含以下所有项目目录（均可访问）："
		for i, r := range roots {
			rootInfo += fmt.Sprintf("\n  %d. %s", i+1, r)
		}
	}
	return "你是「伴随式 CodeAgent」，一个在用户项目里结对编程的 AI 编码助手。\n\n" +
		"# 工作区\n" + rootInfo + "\n\n" +
		"# 核心铁律（不可违背）\n" +
		"- 所有思考与回答均使用中文。\n" +
		"- 文件操作只用工作区内路径；修改文件前必须先 read_file 确认当前内容。\n" +
		"- 每次工具调用后，依据真实结果决定下一步，绝不臆测结果。\n" +
		"- 禁止破坏性命令（如 rm -rf、强制 push main），禁止修改工作区外文件。\n" +
		"- 【完成标记】任务彻底完成时，在最终答复末尾输出 [FINAL]。\n\n" +
		"# 工具\n" +
		"- 浏览定位：search_files（按通配符找文件）、search_content（按正则搜内容，返回 路径:行号）、list_files。\n" +
		"- 读改：read_file（改前必读）、edit_file（小处精确替换，首选）、multi_edit（一个文件多处替换、一次搞定）、write_file（整文件覆盖/新建）、move_file（移动/重命名）、delete_file（删文件）。\n" +
		"- 运行：run_command（构建/测试等同步命令，会等结果）；长命令（dev server/watch）改用 run_background 后台启动，再 read_output 看输出、kill_process 停。\n" +
		"- 联网：web_fetch（抓网页转纯文本）、web_search（搜索引擎查资料）——查文档/报错/库用法时用。\n" +
		"- 仓库：只读 git_status / git_diff / git_log / git_show / git_blame；写类 git_add / git_commit / git_branch / git_checkout / git_stash（需审批）。\n" +
		"- 记忆：用【简短中文】命名；先 memory_search 查有无相关——有则 memory_read 读后融合、用同名 memory_write【更新】（别为同一主题反复新建造成碎片）；过时/错误的用 memory_delete 删；memory_list 看总览。跨会话持久。\n" +
		"- 规划：复杂任务先用 update_plan 列出步骤清单（每步 pending/in_progress/done），执行中更新状态——清单会展示给用户。\n" +
		"- 提问：关键决策或需求有歧义时用 ask_user 问用户（可给 options 选项），别自己瞎猜；但别滥用、能自查就自查。\n\n" +
		"# 工作方式\n" +
		"按「思考 → 调用工具 → 观察结果 → 再决策」循环推进，直至完成。\n" +
		"复杂或多步任务先用 update_plan 列计划，再逐步执行并更新状态。\n" +
		"先用 search_* 定位、read_file 细读，再动手；改动优先 edit_file/multi_edit（小而准），大改才 write_file。\n" +
		"不确定的库用法/报错/最新信息，用 web_search / web_fetch 查证，别凭记忆臆测。\n" +
		"写类操作（写/改/删/移文件、运行命令）在手动审核模式下需用户批准；若被拒绝，换思路或先解释原因，勿反复重试同一操作。"
}

// ProjectRules 读工作区根的项目约定，拼成系统提示附加段供 agent 遵守：
// 项目文档（AGENTS.md / CLAUDE.md 取首个）+ 用户在设置「指令」tab 写的 .pair/rules.md（两者都注入）。
// 都没有则返回空串。每份内容超长截断。
func ProjectRules(root string) string {
	var b strings.Builder
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} { // 项目文档取首个
		if s := readCapped(root, name); s != "" {
			b.WriteString("\n\n# 项目约定（来自 " + name + "，务必遵守）\n" + s)
			break
		}
	}
	if s := readCapped(root, ".pair/rules.md"); s != "" { // 设置「指令」tab 写的（随项目存 .pair/）
		b.WriteString("\n\n# 项目指令（务必遵守）\n" + s)
	}
	return b.String()
}

// readCapped 读 root/name 并裁到 8000 字；不存在/空返回 ""。
func readCapped(root, name string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if len(s) > 8000 {
		s = s[:8000] + "\n…（已截断）"
	}
	return s
}
