// skill_runtime_fix_test.go — skill/task/mcp 工具运行时根解析回归测试
//
// ★ 2026-09-12 修复回归（重大 BUG）：
//   RegisterManagementTools / registerTaskTools 闭包捕获注册时根，经 tool-system
//   插件接管存档进全局 hostExecutors（启动一次性存档）后永久冻结：
//     1. 启动时未开工作区（root=""）→ skill_write 执行 WriteSkill("") →
//        filepath.Join("", name) 相对路径 → 写到进程 CWD（安装目录根）下；
//     2. 切换工作区后工具仍写启动工作区（多工作区串台）。
//   修复后：执行时运行时解析（args._wsRoot 会话注入 → ctx 会话绑定根 →
//   工作区实时快照 → 注册时 root 兜底），且新增 scope 参数支持全局层级。
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withIsolatedSkillEnv 隔离技能全局变量（快照恢复），并清空工作区快照
// （保证「未开工作区」场景确定性——否则其他测试残留的 WorkspaceRoots /
// core.Folders 会短路回落链）。
func withIsolatedSkillEnv(t *testing.T) {
	t.Helper()
	oldGlobal, oldSystem, oldProject, oldRoots := SkillGlobalDir, SkillSystemDir, SkillProjectDir, WorkspaceRoots
	WorkspaceRoots = nil
	SkillGlobalDir = ""
	SkillSystemDir = ""
	SkillProjectDir = ""
	t.Cleanup(func() {
		SkillGlobalDir, SkillSystemDir, SkillProjectDir, WorkspaceRoots = oldGlobal, oldSystem, oldProject, oldRoots
	})
}

// TestSkillWriteRuntimeRootWithWsRoot 核心回归：args._wsRoot 会话注入优先——
// 注册时闭包根为空（模拟启动未开工作区），执行时必须落会话工作区。
func TestSkillWriteRuntimeRootWithWsRoot(t *testing.T) {
	withIsolatedSkillEnv(t)
	ws := t.TempDir()
	r := NewRegistry()
	RegisterManagementTools(r, "") // 注册根为空（模拟启动时未开工作区）

	out, err := r.Execute(context.Background(), "skill_write",
		`{"name":"ws-skill","description":"会话根测试","content":"# 正文","_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("skill_write 失败: %v", err)
	}
	if !strings.Contains(out, "工作区级") {
		t.Fatalf("返回应标注工作区级，got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".pair", "skills", "ws-skill", "SKILL.md")); statErr != nil {
		t.Fatalf("技能应写到会话工作区 .pair/skills/ 下: %v", statErr)
	}
}

// TestSkillWriteSessionCtxRoot ctx 会话绑定根（WithSessionWorkspaceRoot）路径。
func TestSkillWriteSessionCtxRoot(t *testing.T) {
	withIsolatedSkillEnv(t)
	ws := t.TempDir()
	r := NewRegistry()
	RegisterManagementTools(r, "") // 注册根为空
	ctx := WithSessionWorkspaceRoot(context.Background(), ws)
	if _, err := r.Execute(ctx, "skill_write",
		`{"name":"ctx-skill","content":"# 正文"}`); err != nil {
		t.Fatalf("skill_write 失败: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".pair", "skills", "ctx-skill", "SKILL.md")); statErr != nil {
		t.Fatalf("技能应写到 ctx 会话根 .pair/skills/ 下: %v", statErr)
	}
}

// TestSkillWriteScopeGlobal scope=global → 写 SkillGlobalDir（跨工作区）。
func TestSkillWriteScopeGlobal(t *testing.T) {
	withIsolatedSkillEnv(t)
	tmp := t.TempDir()
	SkillGlobalDir = tmp
	ws := t.TempDir() // 存在工作区根也不该被用到
	r := NewRegistry()
	RegisterManagementTools(r, "")

	out, err := r.Execute(context.Background(), "skill_write",
		`{"name":"global-skill","content":"# 全局","_wsRoot":"`+filepath.ToSlash(ws)+`","scope":"global"}`)
	if err != nil {
		t.Fatalf("skill_write(global) 失败: %v", err)
	}
	if !strings.Contains(out, "全局") {
		t.Fatalf("返回应标注全局层级，got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "global-skill", "SKILL.md")); statErr != nil {
		t.Fatalf("技能应写到 SkillGlobalDir: %v", statErr)
	}
	// 工作区根下不应出现
	if _, statErr := os.Stat(filepath.Join(ws, ".pair", "skills", "global-skill")); statErr == nil {
		t.Fatal("scope=global 不应写工作区根")
	}
}

// TestSkillWriteNoWorkspaceFallsBackGlobal 原 BUG 场景：启动未开工作区
// （注册根=""、无会话上下文、无工作区快照）→ 必须显式落全局技能目录，
// 绝不能再走相对路径（filepath.Join("", name) → 进程 CWD 安装目录根）。
func TestSkillWriteNoWorkspaceFallsBackGlobal(t *testing.T) {
	withIsolatedSkillEnv(t)
	tmp := t.TempDir()
	SkillGlobalDir = tmp
	r := NewRegistry()
	RegisterManagementTools(r, "") // 注册根为空、无 _wsRoot、无 ctx 会话根、快照已清空

	out, err := r.Execute(context.Background(), "skill_write",
		`{"name":"fallback-skill","content":"# 回落"}`)
	if err != nil {
		t.Fatalf("skill_write 失败: %v", err)
	}
	if !strings.Contains(out, "回落") && !strings.Contains(out, "全局") {
		t.Fatalf("返回应标注全局回落，got %q", out)
	}
	// 必须落在 SkillGlobalDir（不再相对 CWD）
	if _, statErr := os.Stat(filepath.Join(tmp, "fallback-skill", "SKILL.md")); statErr != nil {
		t.Fatalf("未开工作区时应回落全局技能目录: %v", statErr)
	}
}

// TestSkillDeleteScopeGlobal 全局技能删除路径。
func TestSkillDeleteScopeGlobal(t *testing.T) {
	withIsolatedSkillEnv(t)
	tmp := t.TempDir()
	SkillGlobalDir = tmp
	if err := WriteSkill(tmp, Skill{Name: "del-me", Body: "# x"}); err != nil {
		t.Fatalf("预置全局技能失败: %v", err)
	}
	r := NewRegistry()
	RegisterManagementTools(r, "")
	out, err := r.Execute(context.Background(), "skill_delete", `{"name":"del-me","scope":"global"}`)
	if err != nil {
		t.Fatalf("skill_delete(global) 失败: %v", err)
	}
	if !strings.Contains(out, "全局") {
		t.Fatalf("返回应标注全局，got %q", out)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "del-me")); !os.IsNotExist(statErr) {
		t.Fatalf("全局技能目录应被删除，statErr=%v", statErr)
	}
}

// TestSkillDeleteRuntimeWsRoot 工作区级删除按运行时根解析（不依赖注册根）。
func TestSkillDeleteRuntimeWsRoot(t *testing.T) {
	withIsolatedSkillEnv(t)
	ws := t.TempDir()
	if err := WriteSkill(filepath.Join(ws, ".pair", "skills"), Skill{Name: "ws-del", Body: "# x"}); err != nil {
		t.Fatalf("预置工作区技能失败: %v", err)
	}
	r := NewRegistry()
	RegisterManagementTools(r, "") // 注册根为空
	if _, err := r.Execute(context.Background(), "skill_delete",
		`{"name":"ws-del","_wsRoot":"`+filepath.ToSlash(ws)+`"}`); err != nil {
		t.Fatalf("skill_delete 失败: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".pair", "skills", "ws-del")); !os.IsNotExist(statErr) {
		t.Fatalf("工作区技能目录应被删除，statErr=%v", statErr)
	}
}

// TestSkillListShowsGlobalLevel skill_list 应包含全局层级技能并标注「全局」。
func TestSkillListShowsGlobalLevel(t *testing.T) {
	withIsolatedSkillEnv(t)
	tmp := t.TempDir()
	SkillGlobalDir = tmp
	if err := WriteSkill(tmp, Skill{Name: "g-only", Description: "仅全局", Body: "# x"}); err != nil {
		t.Fatalf("预置失败: %v", err)
	}
	r := NewRegistry()
	RegisterManagementTools(r, "")
	out, err := r.Execute(context.Background(), "skill_list", ``)
	if err != nil {
		t.Fatalf("skill_list 失败: %v", err)
	}
	if !strings.Contains(out, "[全局] g-only") {
		t.Fatalf("skill_list 应列出全局层级技能，got:\n%s", out)
	}
}

// TestUseTaskManagerPerRoot 回归：UseTaskManager 按 root 缓存（原 sync.Once
// 首调冻结根 → 任务落启动工作区/安装目录根）。
func TestUseTaskManagerPerRoot(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	ta, tb := UseTaskManager(a), UseTaskManager(b)
	if ta == tb {
		t.Fatal("不同 root 应返回独立 TaskManager 实例（任务目录隔离）")
	}
	if UseTaskManager(a) != ta {
		t.Fatal("同 root 应复用实例")
	}
	ta.Create("A任务", "", nil, "conv-a")
	if _, statErr := os.Stat(filepath.Join(a, ".pair", "tasks")); statErr != nil {
		t.Fatalf("A 任务应落 A 根 .pair/tasks: %v", statErr)
	}
	// B 根可以有 NewTaskManager 预建的空目录（构造副作用），但不应有任务文件
	if ents, _ := os.ReadDir(filepath.Join(b, ".pair", "tasks")); len(ents) > 0 {
		t.Fatalf("A 根任务不应出现在 B 根（文件数=%d）", len(ents))
	}
}

// TestUpdateTasksRuntimeRoot 回归核心：update_tasks 执行器按 _wsRoot 落盘，
// 注册根为空（启动未开工作区场景）不再落安装目录根。
func TestUpdateTasksRuntimeRoot(t *testing.T) {
	withIsolatedSkillEnv(t)
	ws := t.TempDir()
	r := NewRegistry()
	registerTaskTools(r, "") // 注册根为空（模拟启动未开工作区）
	out, err := r.Execute(context.Background(), "update_tasks",
		`{"tasks":[{"subject":"运行时根任务","status":"pending"}],"_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("update_tasks 失败: %v", err)
	}
	if !strings.Contains(out, "运行时根任务") {
		t.Fatalf("返回应含任务，got %q", out)
	}
	ents, readErr := os.ReadDir(filepath.Join(ws, ".pair", "tasks"))
	if readErr != nil || len(ents) == 0 {
		t.Fatalf("任务文件应落会话工作区 .pair/tasks（ents=%v err=%v）", ents, readErr)
	}
}

// TestLoadSkillRuntimeRoot 读路径回归（2026-09-12 修复盲区）：load_skill /
// skill_list 执行器必须按 args._wsRoot（会话注入）定位工作区技能——
// 修复前闭包捕获启动根（空）→ LoadAllSkillsFromRoot("") 只扫 system
// 级 →「未找到技能」（用户实测：工作区 .pair/skills 存在但 load 失败）。
func TestLoadSkillRuntimeRoot(t *testing.T) {
	withIsolatedSkillEnv(t)
	ws := t.TempDir()
	if err := WriteSkill(filepath.Join(ws, ".pair", "skills"),
		Skill{Name: "ws-skill", Description: "d", Body: "# 正文"}); err != nil {
		t.Fatalf("预置工作区技能失败: %v", err)
	}
	r := NewRegistry()
	RegisterManagementTools(r, "") // 注册根为空（模拟启动未开工作区）

	// ① load_skill 经 _wsRoot 找到工作区技能
	out, err := r.Execute(context.Background(), "load_skill",
		`{"name":"ws-skill","_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("load_skill(_wsRoot) 失败: %v", err)
	}
	if !strings.Contains(out, "正文") {
		t.Fatalf("应返回技能正文，got %q", out)
	}

	// ② skill_list 经 _wsRoot 列出工作区技能
	out2, err := r.Execute(context.Background(), "skill_list",
		`{"_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("skill_list(_wsRoot) 失败: %v", err)
	}
	if !strings.Contains(out2, "ws-skill") {
		t.Fatalf("列表应含工作区技能，got %q", out2)
	}

	// ③ 无会话根（复现原 BUG 场景）：工作区技能不可见（system 目录为空 → 无技能）
	if _, err := r.Execute(context.Background(), "load_skill", `{"name":"ws-skill"}`); err == nil {
		t.Fatal("无会话根时不应找到工作区技能（应报未找到）")
	}
}

// TestMCPRemoveScopeProject mcp_remove scope=project / user 分层删除。
func TestMCPRemoveScopeProject(t *testing.T) {
	oldProjectPath, oldUserPath := MCPProjectConfigPath, MCPUserConfigPath
	t.Cleanup(func() { MCPProjectConfigPath, MCPUserConfigPath = oldProjectPath, oldUserPath })
	proj, user := t.TempDir(), t.TempDir()
	MCPProjectConfigPath = filepath.Join(proj, ".pair", "mcp.json")
	MCPUserConfigPath = filepath.Join(user, "mcp.json")

	if err := MCPUpsert(MCPLevelProject, MCPEntry{Name: "p-srv", Command: "echo"}); err != nil {
		t.Fatalf("预置工作区级 MCP 失败: %v", err)
	}
	r := NewRegistry()
	RegisterManagementTools(r, "")
	// 默认（user）：工作区级服务器不应被删
	if _, err := r.Execute(context.Background(), "mcp_remove", `{"name":"p-srv"}`); err == nil {
		t.Fatal("默认 scope=user 删工作区级服务器应报错")
	}
	// scope=project：删除成功
	if _, err := r.Execute(context.Background(), "mcp_remove", `{"name":"p-srv","scope":"project"}`); err != nil {
		t.Fatalf("scope=project 删除失败: %v", err)
	}
	if err := MCPDelete(MCPLevelProject, "p-srv"); !os.IsNotExist(err) {
		t.Fatalf("工作区级服务器应已删除，err=%v", err)
	}
}
