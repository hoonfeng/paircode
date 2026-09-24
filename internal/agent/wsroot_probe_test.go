package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// ─────────────────────────────────────────────────────────────
// 插件 ctx 服务工作区根解析回归测试（2026-09-20 修复配套）
//
// 根因（用户报告「在非默认工作区对话时，插件的文件输出到默认工作区」）：
//   - 插件宿主根只在 NewPluginHost 时快照一次，主工作区切换后不更新
//     → 插件 ctx 服务永远按「IDE 启动时那个工作区」解析路径；
//   - cordis(op=run) 装载插件时（宿主工具执行）不绑定会话根，apply 期间
//     直接回落到快照根；
//   - buildFSService 自己维护一份手写根解析副本，只认「工具调用根 / UI 根」两档。
//
// 修复后的根优先级（jsPluginAdapter.ctxServiceRoot）：
//
//	工具调用会话根 ＞ UI invoke 根 ＞ 装载期会话根 ＞ 插件上下文根
//	（随宿主主工作区实时更新）＞ 全局主根；全空 → 显式报错（不静默写别的工作区）。
//
// ─────────────────────────────────────────────────────────────

// TestPluginCtxRootApplyPhaseBinding 装载阶段（apply）根绑定：
//   - A) 无装载根的装载（启动期磁盘插件装配等）：apply 期间落插件上下文根；
//   - B) LoadJSDynamicRoot(def, 会话根)（cordis(op=run) 路径）：apply 期间落会话根
//     ——这正是 agent 自建/装载插件写错工作区的修复点。
func TestPluginCtxRootApplyPhaseBinding(t *testing.T) {
	hostRoot := t.TempDir() // 宿主创建时的工作区（=「默认工作区」）
	sessRoot := t.TempDir() // 发起装载的会话工作区（非默认工作区）

	ph := NewPluginHost(NewRegistry(), nil, hostRoot)

	// ── A) 无装载根：apply 落插件上下文根（宿主根） ──
	codeA := `
return {
  name: 'ctxroot-apply-plain',
  inject: ['fs'],
  apply(ctx) {
    ctx.fs.writeFile('apply-plain.txt', 'apply-phase')
  }
}`
	defA, err := ph.DefineJSCodeVersioned(codeA, "js", "probe: apply 阶段根解析（无装载根）", "", "", "")
	if err != nil {
		t.Fatalf("define A: %v", err)
	}
	dA, _ := ph.GetJSDef(defA)
	if err := ph.LoadJSDynamic(dA); err != nil {
		t.Fatalf("load A: %v", err)
	}
	_, errHost := os.Stat(filepath.Join(hostRoot, "apply-plain.txt"))
	_, errSess := os.Stat(filepath.Join(sessRoot, "apply-plain.txt"))
	t.Logf("[A] 无装载根：宿主根/apply-plain.txt=%v ; 会话根/apply-plain.txt=%v", errHost == nil, errSess == nil)
	if errHost != nil {
		t.Errorf("[A] 无装载根时 apply 期间 ctx.fs 写入应落宿主根 %s", hostRoot)
	}
	if errSess == nil {
		t.Errorf("[A] 无装载根时 apply 期间 ctx.fs 写入不应落会话根 %s", sessRoot)
	}

	// ── B) 带装载会话根：apply 落会话根（★ 修复点） ──
	codeB := `
return {
  name: 'ctxroot-apply-session',
  inject: ['fs'],
  apply(ctx) {
    ctx.fs.writeFile('apply-session.txt', 'apply-phase')
  }
}`
	defB, err := ph.DefineJSCodeVersioned(codeB, "js", "probe: apply 阶段根解析（带装载会话根）", "", "", "")
	if err != nil {
		t.Fatalf("define B: %v", err)
	}
	dB, _ := ph.GetJSDef(defB)
	if err := ph.LoadJSDynamicRoot(dB, sessRoot); err != nil {
		t.Fatalf("load B: %v", err)
	}
	_, errHostB := os.Stat(filepath.Join(hostRoot, "apply-session.txt"))
	_, errSessB := os.Stat(filepath.Join(sessRoot, "apply-session.txt"))
	t.Logf("[B] 带装载根：宿主根/apply-session.txt=%v ; 会话根/apply-session.txt=%v", errHostB == nil, errSessB == nil)
	if errSessB != nil {
		t.Errorf("[B] 绑定装载会话根后，apply 期间 ctx.fs 写入应落会话根 %s", sessRoot)
	}
	if errHostB == nil {
		t.Errorf("[B] 绑定装载会话根后，apply 期间 ctx.fs 写入不应落宿主根 %s", hostRoot)
	}
}

// TestPluginToolCallRootBeatsLoadRoot 工具执行阶段（会话根）优先于装载根与上下文根：
// agent 在「另一个会话工作区」调用同一插件的工具时，写入必须落该会话根。
func TestPluginToolCallRootBeatsLoadRoot(t *testing.T) {
	hostRoot := t.TempDir() // 宿主创建时工作区
	loadRoot := t.TempDir() // 装载该插件的会话工作区
	callRoot := t.TempDir() // 调用该插件工具的会话工作区（第三个）

	reg := NewRegistry()
	ph := NewPluginHost(reg, nil, hostRoot)

	code := `
return {
  name: 'ctxroot-tool-call',
  inject: ['fs'],
  apply(ctx) {
    ctx.tools.register({
      name: 'ctxroot_tool_write',
      description: 'probe：写文件用于验证工具调用阶段的根优先于装载根',
      parameters: { type: 'object', properties: { path: { type: 'string' } } },
      execute: async (args) => {
        ctx.fs.writeFile(args.path, 'tool-phase')
        return 'written:' + args.path
      }
    })
  }
}`
	id, err := ph.DefineJSCodeVersioned(code, "js", "probe: 工具阶段根解析", "", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	def, _ := ph.GetJSDef(id)
	if err := ph.LoadJSDynamicRoot(def, loadRoot); err != nil {
		t.Fatalf("load: %v", err)
	}

	sessCtx := WithSessionWorkspaceRoot(context.Background(), callRoot)
	out, err := reg.Execute(sessCtx, "ctxroot_tool_write", `{"path":"tool.txt"}`)
	if err != nil {
		t.Fatalf("执行插件工具: %v", err)
	}
	t.Logf("[C] 工具阶段返回: %s", out)
	if _, err := os.Stat(filepath.Join(callRoot, "tool.txt")); err != nil {
		t.Errorf("[C] 工具调用期间写入应落调用会话根 %s", callRoot)
	}
	for _, wrong := range []struct{ name, dir string }{{"宿主根", hostRoot}, {"装载根", loadRoot}} {
		if _, err := os.Stat(filepath.Join(wrong.dir, "tool.txt")); err == nil {
			t.Errorf("[C] 工具调用期间写入不应落%s %s", wrong.name, wrong.dir)
		}
	}
}

// TestPluginHostWorkspaceRootSwitch 主工作区切换（PluginHost.SetWorkspaceRoot）：
// 无会话绑定的插件调用（harness.handle 被宿主调用、timer/事件回调等）必须跟随
// 新的主工作区，而不是继续写「宿主创建时那个工作区」。
func TestPluginHostWorkspaceRootSwitch(t *testing.T) {
	oldRoot := t.TempDir() // 宿主创建时工作区（IDE 启动工作区）
	newRoot := t.TempDir() // 用户后来切换到的当前主工作区

	ph := NewPluginHost(NewRegistry(), nil, oldRoot)

	code := `
return {
  name: 'ctxroot-host-switch',
  inject: ['fs'],
  apply(ctx) {
    harness.handle('writeProbe', (args) => {
      ctx.fs.writeFile(args.path, 'handler-phase')
      return ctx.get('workspaceRoot')
    })
  }
}`
	id, err := ph.DefineJSCodeVersioned(code, "js", "probe: 主工作区切换后的根", "", "", "")
	if err != nil {
		t.Fatalf("define: %v", err)
	}
	def, _ := ph.GetJSDef(id)
	if err := ph.LoadJSDynamic(def); err != nil { // 无装载会话根（启动期装配语义）
		t.Fatalf("load: %v", err)
	}
	ad := ph.findRunningJSAdapter("ctxroot-host-switch")
	if ad == nil {
		t.Fatal("适配器未找到")
	}

	// 切换主工作区前：落宿主创建时的工作区
	if _, err := ad.Invoke("writeProbe", map[string]any{"path": "before.txt"}); err != nil {
		t.Fatalf("invoke before: %v", err)
	}
	if _, err := os.Stat(filepath.Join(oldRoot, "before.txt")); err != nil {
		t.Errorf("切换前写入应落初始宿主根 %s", oldRoot)
	}

	// ★ 切换主工作区（主工作区变更回调调用 SetWorkspaceRoot）
	ph.SetWorkspaceRoot(newRoot)
	if got := ph.WorkspaceRoot(); got != newRoot {
		t.Errorf("SetWorkspaceRoot 后 WorkspaceRoot()=%q，期望 %q", got, newRoot)
	}

	wsRootSvc, err := ad.Invoke("writeProbe", map[string]any{"path": "after.txt"})
	if err != nil {
		t.Fatalf("invoke after: %v", err)
	}
	if got, _ := wsRootSvc.(string); got != newRoot {
		t.Errorf("workspaceRoot 服务值=%q，期望 %q（切换主工作区后应同步）", got, newRoot)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "after.txt")); err != nil {
		t.Errorf("切换主工作区后写入应落新主工作区 %s", newRoot)
	}
	if _, err := os.Stat(filepath.Join(oldRoot, "after.txt")); err == nil {
		t.Errorf("切换主工作区后写入不应再落旧工作区 %s", oldRoot)
	}
}
