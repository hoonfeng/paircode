package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBashResourceExists 内置 bash 资源必须随仓库分发（bin/bash/usr/bin/bash.exe）。
func TestBashResourceExists(t *testing.T) {
	cand := filepath.Join("..", "..", "bin", "bash", "usr", "bin", "bash.exe")
	if _, err := os.Stat(cand); err != nil {
		t.Fatalf("内置 bash 资源缺失 %s: %v", cand, err)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "bin", "bash", "tmp")); err != nil {
		t.Fatalf("内置 bash 缺 tmp 目录（/tmp 警告）: %v", err)
	}
	// exe 同目录命中：companion.exe 在项目根时，内置路径必须精确命中
	exe := filepath.Join("..", "..", "companion.exe")
	got := bashCandidate(exe)
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("companion.exe 同目录内置 bash 未命中: %s (%v)", got, err)
	}
	t.Logf("companion 运行时将使用内置 bash: %s", got)
}

// TestDetectBash 探测链：内置资源 > 系统 Git Bash > PATH bash。
func TestDetectBash(t *testing.T) {
	bashPath, msysBin := detectBash()
	if bashPath == "" {
		t.Fatal("detectBash 返回空（本机应至少命中系统 Git Bash 或内置资源）")
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
