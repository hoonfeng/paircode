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
