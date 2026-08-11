package agent

// 回归测试：两个已知问题的修复验证
//  1. 后台进程跨轮次存活：Registry 每轮对话重建（web_server.go buildWebLoopOpts），
//     bgRegistry 若随之重建则 run_background 的进程在下一轮丢失 → 已改为全局单例 globalBG。
//  2. 多项目 Lua 工具加载：工作区多项目时各自 .pair/tools/*.lua 都应注册（primary 同名覆盖）。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestBGCrossRegistry 模拟 web 端两轮对话（两个独立 Registry），
// 第一轮 run_background 启动的进程，第二轮 read_output/kill_process 应仍可访问。
func TestBGCrossRegistry(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	r1 := NewRegistry()
	RegisterDefaultTools(r1, root)
	out, err := r1.Execute(ctx, "run_background", `{"command":"echo cross_round_ok"}`)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`id=(\d+)`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("run_background 应返回进程 id，得 %q", out)
	}
	idArg := `{"id":` + m[1] + `}`

	// 模拟下一轮对话：全新 Registry（每轮 buildWebLoopOpts 都会新建）
	r2 := NewRegistry()
	RegisterDefaultTools(r2, root)

	var ro string
	for i := 0; i < 200; i++ { // 轮询至结束
		ro, err = r2.Execute(ctx, "read_output", idArg)
		if err == nil && strings.Contains(ro, "已结束") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || !strings.Contains(ro, "cross_round_ok") {
		t.Fatalf("跨轮次读输出失败: err=%v ro=%q", err, ro)
	}
	// kill 也应能找到进程（幂等）
	if _, err := r2.Execute(ctx, "kill_process", idArg); err != nil {
		t.Fatalf("跨轮次 kill_process 失败: %v", err)
	}
}

// TestLoadAllProjectLuaTools 多项目工作区：每个项目的 .pair/tools/*.lua 都注册，
// 同名工具 primary 项目覆盖其他项目。
func TestLoadAllProjectLuaTools(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projB := t.TempDir()
	WorkspaceRoots = []string{primary, projB}

	mkTool := func(dir, file, name, desc string) {
		os.MkdirAll(filepath.Join(dir, ".pair", "tools"), 0o755)
		script := fmt.Sprintf(`return {name=%q, description=%q, parameters={type="object",properties={}}, run=function(args) return "ok" end}`, name, desc)
		if err := os.WriteFile(filepath.Join(dir, ".pair", "tools", file), []byte(script), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mkTool(primary, "tool_a.lua", "tool_a", "主项目工具")
	mkTool(projB, "tool_b.lua", "tool_b", "B项目工具")
	// 同名冲突：primary 应覆盖 projB
	mkTool(projB, "same.lua", "same_tool", "来自B项目")
	mkTool(primary, "same.lua", "same_tool", "来自主项目")

	reg := NewRegistry()
	loaded := LoadAllProjectLuaTools(reg, primary)
	_ = loaded

	if _, ok := reg.Get("tool_a"); !ok {
		t.Error("主项目 Lua 工具 tool_a 未注册")
	}
	if _, ok := reg.Get("tool_b"); !ok {
		t.Error("其他项目 Lua 工具 tool_b 未注册（多项目支持缺失）")
	}
	if same, ok := reg.Get("same_tool"); !ok {
		t.Error("同名工具 same_tool 未注册")
	} else if !strings.Contains(same.Description, "主项目") {
		t.Errorf("同名工具应 primary 覆盖，描述=%q", same.Description)
	}
}

// TestResolveLuaToolsDir project 参数解析：项目名/路径/默认主项目。
func TestResolveLuaToolsDir(t *testing.T) {
	old := WorkspaceRoots
	defer func() { WorkspaceRoots = old }()

	primary := t.TempDir()
	projB := t.TempDir()
	WorkspaceRoots = []string{primary, projB}

	// 默认 → 主项目
	d, err := resolveLuaToolsDir(primary, "")
	if err != nil || d != filepath.Join(primary, ".pair", "tools") {
		t.Errorf("默认应解析到主项目: %q err=%v", d, err)
	}
	// 项目名（basename）
	d, err = resolveLuaToolsDir(primary, filepath.Base(projB))
	if err != nil || d != filepath.Join(projB, ".pair", "tools") {
		t.Errorf("项目名应解析到 B 项目: %q err=%v", d, err)
	}
	// 项目路径
	d, err = resolveLuaToolsDir(primary, projB)
	if err != nil || d != filepath.Join(projB, ".pair", "tools") {
		t.Errorf("项目路径应解析到 B 项目: %q err=%v", d, err)
	}
	// 未知项目 → 报错
	if _, err = resolveLuaToolsDir(primary, "not_exist_proj"); err == nil {
		t.Error("未知项目应报错")
	}
}
