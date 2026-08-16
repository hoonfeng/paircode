package impl

// git 工具：读类(status/diff/log/show/blame，ReadOnly 免审) + 写类(add/commit/branch/checkout/stash，需审批)。
// 读类即便手动审核也能直接查；写类破坏性操作交人确认。全部经 runGit 走 git CLI。

import (
	"context"
	"fmt"
	. "github.com/hoonfeng/paircode/plugins-src/plugins/tool-git/toolbin"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func Register(r *Registry, root string) {
	r.Register(&Tool{
		Name:        "git_status",
		UsageGuide:  "查看工作区 git 状态（分支+已修改/暂存/未跟踪文件）。比 run_command git status 更简洁（porcelain 紧凑格式+自动判断工作区干净）。先调用此工具了解当前变更再决定下一步。",
		Description: "查看 git 工作区状态（当前分支 + 已修改/暂存/未跟踪文件，porcelain 紧凑格式）。",
		Parameters:  ObjSchema(Props{"project": ProjectSchemaProp()}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			out, err := runGit(ctx, projRoot, "status", "--porcelain=v1", "--branch")
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
		UsageGuide:  "查看工作区未暂存改动（或 staged=true 看已暂存改动）。file 参数可限定单个文件。比 run_command git diff 更智能（无改动时自动返回「无改动」而非空输出）。",
		Description: "查看 git 改动。file 可选（限定单个文件）；staged=true 看已暂存(--cached)的改动，否则看工作区未暂存改动。",
		Parameters: ObjSchema(Props{
			"project": ProjectSchemaProp(),
			"file":    StrProp("可选：限定单个文件路径"),
			"staged":  BoolProp("看已暂存(--cached)改动，默认看未暂存"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			gitArgs := []string{"diff"}
			if ArgBool(args, "staged") {
				gitArgs = append(gitArgs, "--cached")
			}
			if f := strings.TrimSpace(ArgStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, projRoot, gitArgs...)
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
		UsageGuide:  "查看最近提交历史（单行格式）。count 限定条数（默认 15）。比 run_command git log --oneline 更省 token（结果精简+自动限制上限 200 条）。",
		Description: "查看最近提交历史（单行格式）。count 限定条数（默认 15）；file 可选（限定某文件的历史）。",
		Parameters: ObjSchema(Props{
			"count": IntProp("条数（默认 15）"),
			"file":  StrProp("可选：限定某文件的提交历史"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			count := ClampInt(ArgInt(args, "count", 15), 15, 1, 200)
			gitArgs := []string{"log", "--oneline", "-n", strconv.Itoa(count)}
			if f := strings.TrimSpace(ArgStr(args, "file")); f != "" {
				gitArgs = append(gitArgs, "--", f)
			}
			out, err := runGit(ctx, projRoot, gitArgs...)
			if err != nil {
				return "", err
			}
			return out, nil
		},
	})

	// ── 读类补充：show / blame ──
	r.Register(&Tool{
		Name:        "git_show",
		UsageGuide:  "查看某次提交的详情与改动。默认 HEAD（最新一次）。比 run_command git show 更安全（自动处理空参数+带 --stat 统计）。",
		Description: "查看某次提交的详情与改动。commit=提交哈希或引用（默认 HEAD）。",
		Parameters:  ObjSchema(Props{"commit": StrProp("提交哈希/引用，默认 HEAD"), "project": ProjectSchemaProp()}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			commit := strings.TrimSpace(ArgStr(args, "commit"))
			if commit == "" {
				commit = "HEAD"
			}
			return runGit(ctx, projRoot, "show", "--stat", commit)
		},
	})

	r.Register(&Tool{
		Name:        "git_blame",
		UsageGuide:  "逐行查看某文件每行的最后修改提交和作者。用 start/end 限定行范围避免输出过多。比 run_command git blame 更方便（自动处理行范围参数格式）。",
		Description: "逐行查看某文件每行的最后修改提交/作者。file 必填；可选 start/end 限定行范围。",
		Parameters:  ObjSchema(Props{"file": StrProp("文件路径"), "start": IntProp("起始行"), "end": IntProp("结束行"), "project": ProjectSchemaProp()}, "file"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			file := strings.TrimSpace(ArgStr(args, "file"))
			if file == "" {
				return "", fmt.Errorf("file 不能为空")
			}
			ga := []string{"blame"}
			if s, e := ArgInt(args, "start", 0), ArgInt(args, "end", 0); s > 0 && e >= s {
				ga = append(ga, "-L", fmt.Sprintf("%d,%d", s, e))
			}
			return runGit(ctx, projRoot, append(ga, "--", file)...)
		},
	})

	// ── 写类：add / commit / branch / checkout / stash（需审批）──
	r.Register(&Tool{
		Name:             "git_add",
		UsageGuide:       "把文件加入暂存区（准备提交）。files 为路径列表；省略则暂存全部改动(-A)。需审核批准。比 run_command git add 更安全（参数自动组装+路径越界拦截）。",
		Description:      "把文件加入暂存区。files 为路径列表；省略则暂存全部改动(-A)。",
		Parameters:       ObjSchema(Props{"files": map[string]any{"type": "array", "description": "文件路径列表（省略=全部）", "items": map[string]any{"type": "string"}}, "project": ProjectSchemaProp()}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			gitArgs := []string{"add"}
			if files := ArgStrSlice(args, "files"); len(files) > 0 {
				gitArgs = append(append(gitArgs, "--"), files...)
			} else {
				gitArgs = append(gitArgs, "-A")
			}
			out, err := runGit(ctx, projRoot, gitArgs...)
			if err != nil {
				return "", err
			}
			return "已暂存。" + out, nil
		},
	})

	r.Register(&Tool{
		Name:             "git_commit",
		UsageGuide:       "提交已暂存的改动。message 必填；all=true 先暂存已跟踪文件再提交(-a)。需审核批准。比 run_command git commit 更安全（message 为空自动拒绝）。",
		Description:      "提交已暂存的改动。message 必填；all=true 先暂存所有已跟踪文件改动再提交(-a)。",
		Parameters:       ObjSchema(Props{"message": StrProp("提交信息"), "all": BoolProp("先 -a 暂存已跟踪改动"), "project": ProjectSchemaProp()}, "message"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			msg := strings.TrimSpace(ArgStr(args, "message"))
			if msg == "" {
				return "", fmt.Errorf("message 不能为空")
			}
			gitArgs := []string{"commit", "-m", msg}
			if ArgBool(args, "all") {
				gitArgs = append(gitArgs, "-a")
			}
			return runGit(ctx, projRoot, gitArgs...)
		},
	})

	r.Register(&Tool{
		Name:             "git_branch",
		UsageGuide:       "分支操作：无 name 列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除。比 run_command git branch 更智能（自动处理三种操作模式的参数差异）。",
		Description:      "分支操作。无 name=列出全部分支；name+checkout=true 创建并切换；name+delete=true 删除；仅 name=创建。",
		Parameters:       ObjSchema(Props{"name": StrProp("分支名（创建/删除时）"), "checkout": BoolProp("创建后切换过去"), "delete": BoolProp("删除该分支"), "project": ProjectSchemaProp()}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			name := strings.TrimSpace(ArgStr(args, "name"))
			switch {
			case name == "":
				return runGit(ctx, projRoot, "branch", "--all")
			case ArgBool(args, "delete"):
				return runGit(ctx, projRoot, "branch", "-D", name)
			case ArgBool(args, "checkout"):
				return runGit(ctx, projRoot, "checkout", "-b", name)
			default:
				return runGit(ctx, projRoot, "branch", name)
			}
		},
	})

	r.Register(&Tool{
		Name:             "git_checkout",
		UsageGuide:       "切换分支或恢复文件到 HEAD。file=true 时 target 为文件路径（丢弃其改动，危险！）。比 run_command git checkout 更安全（分支/文件模式自动判断+参数校验）。",
		Description:      "切换分支，或把文件恢复到 HEAD。target=分支名(切换)；file=true 时 target 为文件路径(丢弃其改动，危险)。",
		Parameters:       ObjSchema(Props{"target": StrProp("分支名或文件路径"), "file": BoolProp("target 是文件(恢复/丢弃改动)"), "project": ProjectSchemaProp()}, "target"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			target := strings.TrimSpace(ArgStr(args, "target"))
			if target == "" {
				return "", fmt.Errorf("target 不能为空")
			}
			if ArgBool(args, "file") {
				return runGit(ctx, projRoot, "checkout", "--", target)
			}
			return runGit(ctx, projRoot, "checkout", target)
		},
	})

	r.Register(&Tool{
		Name:             "git_stash",
		UsageGuide:       "贮藏/恢复工作区改动。action=push(默认贮藏)/pop(弹出恢复)/list(列出)/drop(丢弃)。比 run_command git stash 更方便（自动处理 action+message 组合）。",
		Description:      "贮藏工作区改动。action：push(默认,贮藏) / pop(弹出恢复) / list(列出) / drop(丢弃最近一条)。",
		Parameters:       ObjSchema(Props{"action": StrProp("push/pop/list/drop，默认 push"), "message": StrProp("push 时的备注"), "project": ProjectSchemaProp()}),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			projRoot, err := ProjRootFromArgs(root, args)
			if err != nil {
				return "", err
			}
			action := strings.TrimSpace(ArgStr(args, "action"))
			if action == "" {
				action = "push"
			}
			switch action {
			case "push":
				ga := []string{"stash", "push"}
				if m := strings.TrimSpace(ArgStr(args, "message")); m != "" {
					ga = append(ga, "-m", m)
				}
				return runGit(ctx, projRoot, ga...)
			case "pop", "list", "drop":
				return runGit(ctx, projRoot, "stash", action)
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
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	c.Dir = dir
	out, err := c.CombinedOutput()
	res := CapOutput(DecodeCmdOutput(out), 16000)
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
