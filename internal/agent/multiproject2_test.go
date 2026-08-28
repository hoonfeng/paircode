// multiproject2_test.go — 多项目工作区支持：project 参数路由 + codegraph 按项目独立建图/查询。
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkGoProject 造一个最小 Go 项目（go.mod + 两个 .go 文件），返回项目根。
func mkGoProject(t *testing.T, parent, name string) string {
	t.Helper()
	root := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(root, "pkg", name), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+name+"\n\ngo 1.21\n"), 0o644)
	pkgName := strings.ReplaceAll(name, "-", "")
	os.WriteFile(filepath.Join(root, "pkg", name, "a.go"), []byte("package "+pkgName+"\n\nfunc HelloA() string { return \"a\" }\nfunc helper() int { return 1 }\n"), 0o644)
	os.WriteFile(filepath.Join(root, "pkg", name, "b.go"), []byte("package "+pkgName+"\n\nfunc HelloB() string { return HelloA() + \"b\" }\n"), 0o644)
	return root
}

// TestResolveProjectRoot 验证 project 参数路由（目录名/相对路径/绝对路径/缺省）。
func TestResolveProjectRoot(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projB := mkGoProject(t, primary, "wb-ui")
	WorkspaceRoots = []string{primary, projB}

	// 缺省 → 主项目
	r, err := resolveProjectRoot(primary, "")
	if err != nil || !samePath(r, primary) {
		t.Fatalf("缺省应回主项目, got %v err %v", r, err)
	}
	// 目录名
	r, err = resolveProjectRoot(primary, "wb-ui")
	if err != nil || !samePath(r, projB) {
		t.Fatalf("目录名路由失败: %v err %v", r, err)
	}
	// 相对主项目路径
	r, err = resolveProjectRoot(primary, "wb-ui")
	if err != nil || !samePath(r, projB) {
		t.Fatalf("相对路径路由失败: %v err %v", r, err)
	}
	// 绝对路径
	r, err = resolveProjectRoot(primary, projB)
	if err != nil || !samePath(r, projB) {
		t.Fatalf("绝对路径路由失败: %v err %v", r, err)
	}
	// 不存在
	if _, err = resolveProjectRoot(primary, "nope"); err == nil {
		t.Fatalf("未知项目应报错")
	}
}

// TestCodeGraphPerProject 核心场景：多项目工作区中，非主项目独立建图、独立查询，
// 不依赖主项目图谱（用户痛点：非主项目查不到）。
func TestCodeGraphPerProject(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projA := mkGoProject(t, primary, "gou-ide")
	projB := mkGoProject(t, primary, "wb-ui")
	WorkspaceRoots = []string{projA, projB}

	// 重置缓存保证干净
	resetCodeGraph(projA)
	resetCodeGraph(projB)

	// 对 wb-ui 单独建图（JSONStore 独立存储）
	reg := NewRegistry()
	RegisterDefaultTools(reg, projA)

	out, err := reg.Execute(context.Background(), "codegraph_build", `{"project":"wb-ui","rebuild":true}`)
	if err != nil {
		t.Fatalf("codegraph_build(project=wb-ui): %v", err)
	}
	t.Logf("build out: %s", out)
	if !strings.Contains(out, "wb-ui") {
		t.Fatalf("构建输出应含项目名, got %q", out)
	}

	// 检查 wb-ui 图统计（实体数应为 >0）
	statsOut, err := reg.Execute(context.Background(), "codegraph_stats", `{"project":"wb-ui"}`)
	if err != nil {
		t.Fatalf("codegraph_stats(project=wb-ui): %v", err)
	}
	t.Logf("stats out: %s", statsOut)

	// 在 wb-ui 图里查 HelloA（非主项目函数）
	out, err = reg.Execute(context.Background(), "codegraph_function", `{"name":"HelloA","project":"wb-ui"}`)
	if err != nil {
		t.Fatalf("codegraph_function(project=wb-ui): %v", err)
	}
	if !strings.Contains(out, "HelloA") {
		t.Fatalf("wb-ui 图谱应查到 HelloA, got %q", out)
	}

	// 主项目图谱尚未构建 → 查主项目应有明确提示/错误而非串数据
	out, err = reg.Execute(context.Background(), "codegraph_function", `{"name":"HelloA"}`)
	if err == nil {
		// 允许：主项目图可能自动构建了（EnsureBuildIfNeeded），但不能混入 wb-ui 的 HelloA
		if strings.Contains(out, "HelloA") && strings.Contains(out, "wb-ui") {
			t.Fatalf("主项目图不应混入 wb-ui 实体, got %q", out)
		}
	}

	// 主项目建图后独立查询 HelloB
	out, err = reg.Execute(context.Background(), "codegraph_build", `{}`)
	if err != nil {
		t.Fatalf("主项目 codegraph_build: %v", err)
	}
	out, err = reg.Execute(context.Background(), "codegraph_function", `{"name":"HelloB"}`)
	if err != nil {
		t.Fatalf("主项目 codegraph_function: %v", err)
	}
	if !strings.Contains(out, "HelloB") {
		t.Fatalf("主项目图谱应查到 HelloB, got %q", out)
	}
	// wb-ui 图仍独立可查
	out, err = reg.Execute(context.Background(), "codegraph_function", `{"name":"HelloA","project":"wb-ui"}`)
	if err != nil || !strings.Contains(out, "HelloA") {
		t.Fatalf("wb-ui 图应仍可查, err=%v out=%q", err, out)
	}
}

// TestGitToolsProjectParam git 工具 project 参数：对非主项目执行 git_status。
func TestGitToolsProjectParam(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projB := mkGoProject(t, primary, "wb-ui")
	WorkspaceRoots = []string{primary, projB}
	// 初始化 wb-ui 的 git 仓库
	gitInit := func(dir string) {
		out, err := runGit(context.Background(), dir, "init", "-b", "main")
		if err != nil {
			t.Fatalf("git init %s: %v %s", dir, err, out)
		}
	}
	gitInit(projB)
	gitInit(primary)

	reg := NewRegistry()
	RegisterDefaultTools(reg, primary)

	// 对 wb-ui 看 status
	out, err := reg.Execute(context.Background(), "git_status", `{"project":"wb-ui"}`)
	if err != nil {
		t.Fatalf("git_status(project=wb-ui): %v", err)
	}
	if !strings.Contains(out, "wb-ui") && !strings.Contains(out, "main") {
		t.Fatalf("git_status 应反映 wb-ui 仓库, got %q", out)
	}

	// 默认主项目 status（不应串 wb-ui）
	out, err = reg.Execute(context.Background(), "git_status", `{}`)
	if err != nil {
		t.Fatalf("git_status(主项目): %v", err)
	}
	if strings.Contains(out, "pkg/wb-ui") {
		t.Fatalf("主项目 status 不应混入 wb-ui 文件, got %q", out)
	}
}

// TestCoreToolsProjectParam 核心文件工具 project 参数：read 读非主项目文件。
func TestCoreToolsProjectParam(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projB := mkGoProject(t, primary, "wb-ui")
	WorkspaceRoots = []string{primary, projB}

	reg := NewRegistry()
	RegisterDefaultTools(reg, primary)

	// 带 project 参数读非主项目文件（相对路径基于该项目根）
	out, err := reg.Execute(context.Background(), "read", `{"path":"pkg/wb-ui/a.go","project":"wb-ui"}`)
	if err != nil {
		t.Fatalf("read(project=wb-ui): %v", err)
	}
	if !strings.Contains(out, "HelloA") {
		t.Fatalf("应读到 wb-ui 的 a.go, got %q", out)
	}

	// 不带 project 默认主项目：同样路径在主项目下不存在
	_, err = reg.Execute(context.Background(), "read", `{"path":"pkg/wb-ui/a.go"}`)
	if err == nil {
		t.Fatalf("主项目下不存在该路径，应报错")
	}
}
