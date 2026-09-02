package agent

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// ★ 2026-09 Round3 ③.4：内置 bash 资源（bin/bash，约 28MB）已移除
//   （打包瘦身）——原 TestBashResourceExists（断言 bin/bash 存在）已删除；
//   bash 服务回退链见 TestDetectBash。

// TestDetectBash 探测链：系统 Git Bash → PATH bash。
func TestDetectBash(t *testing.T) {
	bashPath, msysBin := detectBash()
	if bashPath == "" {
		t.Fatal("detectBash 返回空（本机应至少命中系统 Git Bash 或 PATH 内的 bash）")
	}
	t.Logf("detectBash => %s (msys: %s)", bashPath, msysBin)
	// 可执行性冒烟
	cmd := exec.Command(bashPath, "-c", "echo ok")
	applyBashEnv(cmd, msysBin)
	out, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("bash 冒烟失败: err=%v out=%q", err, string(out))
	}
}

// TestRunShellWithBash 统一入口跑 POSIX 语法 + 管道 + 中文 + git（LLM 常用场景）。
func TestRunShellWithBash(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want string
	}{
		{"echo", "echo hello-bash", "hello-bash"},
		{"管道", "printf 'a\\nb\\nc\\n' | grep b", "b"},
		{"引号", `echo "引号\"测试" && echo '单引号'`, "引号\"测试"},
		{"中文", "echo 中文输出测试", "中文输出测试"},
		{"git", "git --version", "git version"},
		{"多命令", "cd .. && pwd | head -1 && echo done", "done"},
		{"POSIX变量", "x=42; echo value=$x", "value=42"},
	}
	for _, c := range cases {
		out, exitErr := runShellWithTimeout(context.Background(), c.cmd, t.TempDir())
		if exitErr != "" {
			t.Errorf("%s: exitErr=%s out=%q", c.name, exitErr, out)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: 期望包含 %q，实际 %q", c.name, c.want, out)
		}
	}
}

// TestRunShellBacktick 反引号命令替换（cmd 下不可能，bash 原生）。
func TestRunShellBacktick(t *testing.T) {
	out, exitErr := runShellWithTimeout(context.Background(), "echo `echo nested`", t.TempDir())
	if exitErr != "" {
		t.Fatalf("exitErr=%s", exitErr)
	}
	if !strings.Contains(out, "nested") {
		t.Fatalf("期望 nested，实际 %q", out)
	}
}
