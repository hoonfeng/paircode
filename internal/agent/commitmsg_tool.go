package agent

import (
	"context"
	"strings"
)

type generateCommitArgs struct {
	Message string `json:"message"`
}

// RegisterCommitMessageTool 注册 generate_commit_message 工具。
// agent 在任务完成时调用此工具，显式指定自动提交的 commit message。
func RegisterCommitMessageTool(r *Registry) {
	r.Register(DefineTool("generate_commit_message",
		"在任务完成时调用此工具，生成一段简洁的 git commit message。参数 message 应是一段简要描述本次变更的句子，用作自动提交信息。系统提示中的「完成标记」已说明输出完成总结的方式，此工具只负责记录提交信息。",
		func(ctx context.Context, args generateCommitArgs) (string, error) {
			msg := strings.TrimSpace(args.Message)
			if msg == "" {
				return "提交信息不能为空，请重新生成", nil
			}
			if len(msg) > 100 {
				msg = msg[:100]
			}
			r.CommitMessage = msg
			return "提交信息已记录: " + msg, nil
		},
	))
}

// ── finish_task（完成_任务）─────────────────────────────────────

type finishTaskArgs struct {
	Result string `json:"result"`
}

// RegisterFinishTaskTool 注册 finish_task（完成_任务）工具。
// Agent 在任务完成时调用此工具报告最终结果，系统自动结束当前任务。
// 参数 result 为任务完成后的最终结果描述。
// 该工具始终注册（包含子 agent），使 LLM 有标准方式通知系统任务已完成。
func RegisterFinishTaskTool(r *Registry) {
	t := DefineTool("finish_task",
		"任务完成后调用此工具报告结果，系统会自动结束当前任务。参数 result 为任务的最终结果描述。",
		func(ctx context.Context, args finishTaskArgs) (string, error) {
			return strings.TrimSpace(args.Result), nil
		},
	)
	t.UsageGuide = "上报任务完成结果，触发系统自动结束当前任务。所有子任务完成后调用此工具通知系统。参数 result 为任务的最终结果描述。比直接输出文本更正式（系统检测到无工具调用+有正文时才视为完成，但 finish_task 显式通知更可靠）。"
	r.Register(t)
}
