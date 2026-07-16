package agent

// 规划 Agent（多 Agent 编排之一）—— 忠实复刻参考 prompts/roles/planner.md + agents/planner.ts。
// 用规划模型(planModel, non-thinking)在执行前把任务分解为 3-6 个「大方向目标」步骤（JSON）。
// companion 单循环：规划产出计划清单 → 展示为计划卡 → 注入执行 Agent 作为执行纲领。
//
// 注：参考的并行子 Agent / 审核重试 / 自动迭代设置在参考源里消费处为 0（纯 UI 占位），不实现。

import (
	"context"
	"encoding/json"
	"strings"
)

// PlanStep 规划的一步：大方向目标，非工具指令（Executor 自决用什么工具达成）。
type PlanStep struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Dependencies  []string `json:"dependencies"`
	IsDestructive bool     `json:"isDestructive"`
}

// Plan 规划 Agent 的输出（思路 + 步骤）。
type Plan struct {
	Reasoning string     `json:"reasoning"`
	Steps     []PlanStep `json:"steps"`
}

// Planner 规划 Agent。Provider 用规划模型（建议 non-thinking）。
type Planner struct {
	Provider     Provider
	SystemPrompt string // 角色系统提示（空=用内置默认）；宿主可从 config 加载覆盖（非硬编码）
}

// DefaultPlannerPrompt 内置默认规划角色提示（config/roles/planner.md 缺失时的回退）。
func DefaultPlannerPrompt() string { return plannerSystemPrompt }

// plannerSystemPrompt 复刻参考 prompts/roles/planner.md（角色/原则/步骤规范/输出格式/规则）。
const plannerSystemPrompt = `# 角色
你是规划 Agent（Planner）——将用户任务分解为大方向的步骤。你是内部协调者，不对用户直接输出，最终回答由主 Agent 汇总。

# 核心原则
1. 规划前理解任务：基于已知项目结构和上下文制定计划。
2. 每个步骤是一个任务目标，不是一条工具指令。Executor 会自己决定用什么工具达成目标。
3. 根据用户意图决定计划类型：代码变更/功能实现 → 必须含明确的目标步骤、直接动手；分析/了解/评估/查看 → 纯分析目标即可；模糊任务默认动手优先。

# 步骤编写规范
写目标，不写工具：✅「修复数据库连接失败问题」「更新前端布局」「安装缺失依赖」；❌「用 read_file 读取 X 再 edit_file 改 Y」（太细、限制灵活性）。

# 规则
- 简单问候/闲聊 → steps 为空数组。
- 用户意图是实现/改进 → 至少一个产生变更的步骤。
- 破坏性操作标记 isDestructive: true。
- 所有思考用中文。`

// planUserPrompt 复刻参考 planner.ts 的生成计划提示（含解析失败重试提示）。
func planUserPrompt(task, retryHint string) string {
	p := "[生成计划]\n任务：" + task + `

请输出一个 JSON 格式的执行计划，严格的格式如下（不要加其他文字，只输出 JSON）：
{"reasoning":"对任务的分析思路","steps":[{"id":"step-1","description":"步骤描述（大方向，不指定如何操作）","dependencies":[],"isDestructive":false}]}

要求：
- reasoning: 中文分析
- steps: 3-6 个步骤，只描述大方向和目标
- dependencies: 前置步骤 id 列表
- isDestructive: 是否涉及删除/破坏性操作`
	if retryHint != "" {
		p += "\n\n上次返回的 JSON 解析失败：" + retryHint + "\n请只输出合法的 JSON，不要包含其他文字。"
	}
	return p
}

// Plan 生成执行计划：3 次重试解析（复刻参考），全失败回退合理默认计划。history 为先前对话上下文。
func (p *Planner) Plan(ctx context.Context, task string, history []Message) (Plan, error) {
	var lastErr string
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return Plan{}, err
		}
		msgs := make([]Message, 0, len(history)+2)
		msgs = append(msgs, Message{Role: RoleSystem, Content: orDefault(p.SystemPrompt, plannerSystemPrompt)})
		msgs = append(msgs, history...)
		msgs = append(msgs, Message{Role: RoleUser, Content: planUserPrompt(task, lastErr)})
		resp, err := p.Provider.Chat(ctx, msgs, nil, nil)
		if err != nil {
			return Plan{}, err
		}
		plan, perr := parsePlan(resp.Content)
		if perr == "" {
			return plan, nil // steps 可为空（问候/闲聊），视作成功
		}
		lastErr = perr
	}
	return defaultPlan(task), nil // 3 次均失败 → 默认 5 步计划（复刻参考 planner.ts）
}

// parsePlan 从 LLM 文本抽 JSON 计划（首 { 到末 }，复刻参考 /\{[\s\S]*\}/）。返回错误描述（""=成功）。
func parsePlan(content string) (Plan, string) {
	i, j := strings.IndexByte(content, '{'), strings.LastIndexByte(content, '}')
	if i < 0 || j <= i {
		return Plan{}, "未找到 JSON 片段"
	}
	var plan Plan
	if err := json.Unmarshal([]byte(content[i:j+1]), &plan); err != nil {
		return Plan{}, "JSON 解析异常: " + truncRunesAgent(err.Error(), 100)
	}
	return plan, ""
}

// defaultPlan 3 次生成失败时的默认 5 步计划（复刻参考 planner.ts 默认计划）。
func defaultPlan(task string) Plan {
	t := truncRunesAgent(task, 100)
	return Plan{
		Reasoning: "自动生成的默认计划：" + t,
		Steps: []PlanStep{
			{ID: "step-1", Description: "理解任务需求：" + t, Dependencies: []string{}},
			{ID: "step-2", Description: "搜索和分析相关代码文件", Dependencies: []string{"step-1"}},
			{ID: "step-3", Description: "制定具体方案", Dependencies: []string{"step-2"}},
			{ID: "step-4", Description: "执行修改", Dependencies: []string{"step-3"}},
			{ID: "step-5", Description: "验证修改结果", Dependencies: []string{"step-4"}},
		},
	}
}
