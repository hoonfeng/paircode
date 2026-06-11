package agent

// git 工具：读类(status/diff/log/show/blame，ReadOnly 免审) + 写类(add/commit/branch/checkout/stash，需审批)。
// 读类即便手动审核也能直接查；写类破坏性操作交人确认。全部经 runGit 走 git CLI。

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func registerGitTools(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "git_status",
		Description: "查看 git 工作区状态（当前分支 + 已修改/暂存/未跟踪文件，porcelain 紧凑格式）。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := runGit(ctx, root, "status", "--porcelain=v1", "--branch")
			if err != nil {
				return "", err
			}
			trimmed := strings.TrimSpace(out)
			// porcelain --branch 首行恒为 "## <branch>"；仅此一行=工作区干净。
			// 非 ## 开头（如 fatal: not a git repository）原样返回，别误标「干净」。
			if strings.HasPrefix(trimmed, "##") && !strings.Contains(trimmed, "\n") {
				return out + "（工作区干净）", nil
			}
			return out, nil
		},
	})

	r.Register(&Tool{
		Name:        "git_diff",
		Description: "查看 git 改动。file 可选（限定单个文件）；staged=true 看已暂存(--cached)的改动，否则看工作区未暂存改动。",
		Parameters: objSchema(props{
			"file":   strProp("可选：限定单个文件路径"),
			"staged": boolProp("看已暂存(--cached)改动，默认看未暂存"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			gitArgs := []string{"diff"}
			if argBool(args, "staged") {
				gitArgs = append(gitArgs, "--cached")
			}
			if f := strings.TrimSpace(argStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, root, gitArgs...)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(out) == "" || out == "（无输出）" {
				return "（无改动）", nil
			}
			return out, nil
		},
	})

	r.Register(&Tool{
		Name:        "git_log",
		Description: "查看最近提交历史（单行格式）。count 限定条数（默认 15）；file 可选（限定某文件的历史）。",
		Parameters: objSchema(props{
			"count": intProp("条数（默认 15）"),
			"file":  strProp("可选：限定某文件的提交历史"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			count := clampInt(argInt(args, "count", 15), 15, 1, 200)
			gitArgs := []string{"log", "--oneline", "-n", strconv.Itoa(count)}
			if f := strings.TrimSpace(argStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, root, gitArgs...)
			if err != nil {
				return "", err
			}
			return out, nil
		},
	})

	// ── 读类补充：show / blame ──
	r.Register(&Tool{
		Name:        "git_show",
		Description: "查看某次提交的详情与改动。commit=提交哈希或引用（默认 HEAD）。",
		Parameters:  objSchema(props{"commit": strProp("提交哈希/引用，默认 HEAD")}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			commit := strings.TrimSpace(argStr(args, "commit"))
			if commit == "" {
				commit = "HEAD"
			}
			return runGit(ctx, root, "show", "--stat", commit)
		},
	})

	r.Register(&Tool{
		Name:        "git_blame",
		Description: "逐行查看某文件每行的最后修改提交/作者。file 必填；可选 start/end 限定行范围。",
		Parameters:  objSchema(props{"file": strProp("文件路径"), "start": intProp("起始行"), "end": intProp("结束行")}, "file"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			file := strings.TrimSpace(argStr(args, "file"))
			if file == "" {
				return "", fmt.Errorf("file 不能为空")
			}
			ga := []string{"blame"}
			if s, e := argInt(args, "start", 0), argInt(args, "end", 0); s > 0 && e >= s {
				ga = append(ga, "-L", fmt.Sprintf("%d,%d", s, e))
			}
			return runGit(ctx, root, append(ga, "--", file)...)
		},
	})

	// ── 写类：add / commit / branch / checkout / stash（需审批）──
	r.Register(&Tool{
		Name:             "git_add",
		Description:      "把文件加入暂存区。files 为路径列表；省略则暂存全部改动(-A)。",
		Parameters:       objSchema(props{"files": map[string]any{"type": "array", "description": "文件路径列表（省略=全部）", "items": map[string]any{"type": "string"}}}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			gitArgs := []string{"add"}
			if files := argStrSlice(args, "files"); len(files) > 0 {
				gitArgs = append(append(gitArgs, "--"), files...)
			} else {
				gitArgs = append(gitArgs, "-A")
			}
			out, err := runGit(ctx, root, gitArgs...)
			if err != nil {
				return "", err
			}
			return "已暂存。" + out, nil
		},
	})

	r.Register(&Tool{
		Name:             "git_commit",
		Description:      "提交已暂存的改动。message 必填；all=true 先暂存所有已跟踪文件改动再提交(-a)。",
		Parameters:       objSchema(props{"message": strProp("提交信息"), "all": boolProp("先 -a 暂存已跟踪改动")}, "message"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			msg := strings.TrimSpace(argStr(args, "message"))
			if msg == "" {
				return "", fmt.Errorf("message 不能为空")
			}
			gitArgs := []string{"commit", "-m", msg}
			if argBool(args, "all") {
				gitArgs = append(gitArgs, "-a")
			}
			return runGit(ctx, root, gitArgs...)
		},
	})

	r.Register(&Tool{
		Name:             "git_branch",
		Description:      "分支操作。无 name=列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除；仅 name=创建。",
		Parameters:       objSchema(props{"name": strProp("分支名（创建/删除时）"), "checkout": boolProp("创建后切换过去"), "delete": boolProp("删除该分支")}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := strings.TrimSpace(argStr(args, "name"))
			switch {
			case name == "":
				return runGit(ctx, root, "branch", "--all")
			case argBool(args, "delete"):
				return runGit(ctx, root, "branch", "-D", name)
			case argBool(args, "checkout"):
				return runGit(ctx, root, "checkout", "-b", name)
			default:
				return runGit(ctx, root, "branch", name)
			}
		},
	})

	r.Register(&Tool{
		Name:             "git_checkout",
		Description:      "切换分支，或把文件恢复到 HEAD。target=分支名(切换)；file=true 时 target 为文件路径(丢弃其改动，危险)。",
		Parameters:       objSchema(props{"target": strProp("分支名或文件路径"), "file": boolProp("target 是文件(恢复/丢弃改动)")}, "target"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			target := strings.TrimSpace(argStr(args, "target"))
			if target == "" {
				return "", fmt.Errorf("target 不能为空")
			}
			if argBool(args, "file") {
				return runGit(ctx, root, "checkout", "--", target)
			}
			return runGit(ctx, root, "checkout", target)
		},
	})

	r.Register(&Tool{
		Name:             "git_stash",
		Description:      "贮藏工作区改动。action：push(默认,贮藏) / pop(弹出恢复) / list(列出) / drop(丢弃最近一条)。",
		Parameters:       objSchema(props{"action": strProp("push/pop/list/drop，默认 push"), "message": strProp("push 时的备注")}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			action := strings.TrimSpace(argStr(args, "action"))
			if action == "" {
				action = "push"
			}
			switch action {
			case "push":
				ga := []string{"stash", "push"}
				if m := strings.TrimSpace(argStr(args, "message")); m != "" {
					ga = append(ga, "-m", m)
				}
				return runGit(ctx, root, ga...)
			case "pop", "list", "drop":
				return runGit(ctx, root, "stash", action)
			default:
				return "", fmt.Errorf("未知 action: %s（push/pop/list/drop）", action)
			}
		},
	})
}

// runGit 在 dir 执行一条 git 子命令（30s 超时）。core.quotepath=false 让非 ASCII 文件名正常显示。
// git 非零退出（如目录非 git 仓库）：有输出则连同返回（让 agent 看到原因），无输出则作 error。
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	full := append([]string{"-c", "core.quotepath=false"}, args...)
	c := exec.CommandContext(cctx, "git", full...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	res := capOutput(string(out), 16000)
	if cctx.Err() == context.DeadlineExceeded {
		return res + "\n[git 超时 30s 已终止]", nil
	}
	if err != nil {
		if strings.TrimSpace(res) == "" {
			return "", fmt.Errorf("git %s 失败: %v", strings.Join(args, " "), err)
		}
		return res, nil // 有输出（如 fatal: not a git repository）→ 回给 agent
	}
	if strings.TrimSpace(res) == "" {
		return "（无输出）", nil
	}
	return res, nil
}
