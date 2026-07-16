package agent

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestGitTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 不在 PATH，跳过 git 工具测试")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Skipf("git init 失败，跳过: %v (%s)", err, out)
	}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()

	mustWrite(t, dir, "new.txt", "hi")

	// git_status：应显示未跟踪文件 new.txt
	out, err := reg.Execute(ctx, "git_status", `{}`)
	if err != nil {
		t.Fatalf("git_status: %v", err)
	}
	if !strings.Contains(out, "new.txt") {
		t.Errorf("git_status 应显示未跟踪 new.txt:\n%s", out)
	}

	// git_diff：未跟踪文件不计入 diff → 无改动
	out, err = reg.Execute(ctx, "git_diff", `{}`)
	if err != nil {
		t.Fatalf("git_diff: %v", err)
	}
	if !strings.Contains(out, "无改动") {
		t.Errorf("未跟踪文件不应进入 diff，预期『无改动』，得:\n%s", out)
	}
}

// TestGitWriteTools 真临时仓：git_add → git_commit → git_log 见提交 → git_branch 创建并切换。
func TestGitWriteTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 不在 PATH")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Skipf("git init 失败: %v (%s)", err, out)
	}
	_ = exec.Command("git", "-C", dir, "config", "user.email", "t@t.dev").Run()
	_ = exec.Command("git", "-C", dir, "config", "user.name", "t").Run()
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	ctx := context.Background()
	mustWrite(t, dir, "a.txt", "hello")

	if _, err := reg.Execute(ctx, "git_add", `{"files":["a.txt"]}`); err != nil {
		t.Fatalf("git_add: %v", err)
	}
	if _, err := reg.Execute(ctx, "git_commit", `{"message":"init commit"}`); err != nil {
		t.Fatalf("git_commit: %v", err)
	}
	if out, err := reg.Execute(ctx, "git_log", `{}`); err != nil || !strings.Contains(out, "init commit") {
		t.Errorf("git_log 应含提交：%v\n%s", err, out)
	}
	if _, err := reg.Execute(ctx, "git_branch", `{"name":"dev","checkout":true}`); err != nil {
		t.Fatalf("git_branch: %v", err)
	}
	if st, _ := reg.Execute(ctx, "git_status", `{}`); !strings.Contains(st, "dev") {
		t.Errorf("应在 dev 分支：\n%s", st)
	}
}

// 非 git 仓库目录：git 输出 fatal 信息，runGit 有输出即连同返回（不报 Go error），
// git_status 不误标「工作区干净」。
func TestGitNotARepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git 不在 PATH")
	}
	dir := t.TempDir() // 未 init → 非 git 仓库
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	out, _ := reg.Execute(context.Background(), "git_status", `{}`)
	if strings.Contains(out, "工作区干净") {
		t.Errorf("非 git 仓库不应标『工作区干净』，得:\n%s", out)
	}
}
