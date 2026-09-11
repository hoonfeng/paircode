package agent

// 任务-会话绑定回归测试（★ 2026-09-12 修复）
//
// 缺陷背景：`.pair/tasks/` 是工作区级共享目录（所有会话任务混放），
// 但 update_tasks 执行器从不写 Task.ConvID → 前端 `GET /api/tasks?convId=X`
// 按会话过滤后恒为空 → 「刷新页面/切换对话后任务面板空白」（运行中仅靠 WS
// 事件兜住表象）。同时 ReplaceAll 的清理逻辑会删除所有不在新列表中的任务
// → 多会话并行时互相清空。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readTaskFiles 读取工作区 .pair/tasks 下全部任务（测试辅助）。
func readTaskFiles(t *testing.T, ws string) []Task {
	t.Helper()
	dir := filepath.Join(ws, ".pair", "tasks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取任务目录失败: %v", err)
	}
	var out []Task
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取任务文件失败: %v", err)
		}
		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			t.Fatalf("解析任务文件失败: %v", err)
		}
		out = append(out, task)
	}
	return out
}

// TestUpdateTasksBindsConvIDArgs 核心回归：执行器经 args._convID 注入的会话
// 必须落到任务文件的 convId 字段（前端按会话查询的前提）。
func TestUpdateTasksBindsConvIDArgs(t *testing.T) {
	ws := t.TempDir()
	r := NewRegistry()
	registerTaskTools(r, "")

	out, err := r.Execute(context.Background(), "update_tasks",
		`{"tasks":[{"subject":"绑定会话的任务","status":"pending"}],
		  "_wsRoot":"`+filepath.ToSlash(ws)+`",
		  "_convID":"conv-bind-1"}`)
	if err != nil {
		t.Fatalf("update_tasks 失败: %v", err)
	}
	if out == "" {
		t.Fatal("update_tasks 应返回任务摘要")
	}

	tasks := readTaskFiles(t, ws)
	if len(tasks) != 1 {
		t.Fatalf("应落盘 1 个任务，实际 %d", len(tasks))
	}
	if tasks[0].ConvID != "conv-bind-1" {
		t.Fatalf("任务 convId 应为 conv-bind-1，实际 %q（前端按会话过滤将拿不到任务）", tasks[0].ConvID)
	}

	// 会话维度可读回：本会话 1 条，其它会话 0 条
	tm := UseTaskManager(ws)
	if got := tm.ListByConvID("conv-bind-1"); len(got) != 1 {
		t.Fatalf("ListByConvID(conv-bind-1) 应为 1 条，实际 %d", len(got))
	}
	if got := tm.ListByConvID("conv-other"); len(got) != 0 {
		t.Fatalf("ListByConvID(conv-other) 应为 0 条，实际 %d", len(got))
	}
}

// TestUpdateTasksBindsConvIDFromCtx 会话 ID 也可来自 ctx（SessionManager.Start
// 注入的 runCtx），覆盖宿主 agent 工具面直调路径。
func TestUpdateTasksBindsConvIDFromCtx(t *testing.T) {
	ws := t.TempDir()
	r := NewRegistry()
	registerTaskTools(r, "")

	ctx := WithSessionConvID(context.Background(), "conv-ctx-9")
	_, err := r.Execute(ctx, "update_tasks",
		`{"tasks":[{"subject":"ctx 会话任务","status":"in_progress"}],
		  "_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("update_tasks 失败: %v", err)
	}

	tasks := readTaskFiles(t, ws)
	if len(tasks) != 1 || tasks[0].ConvID != "conv-ctx-9" {
		t.Fatalf("任务应从 ctx 绑定 conv-ctx-9，实际 %+v", tasks)
	}
}

// TestUpdateTasksNoConvKeepsGlobal 无会话上下文时保持旧行为（全局任务，
// convId 为空），不因修复丢失向后兼容。
func TestUpdateTasksNoConvKeepsGlobal(t *testing.T) {
	ws := t.TempDir()
	r := NewRegistry()
	registerTaskTools(r, "")

	_, err := r.Execute(context.Background(), "update_tasks",
		`{"tasks":[{"subject":"全局任务","status":"pending"}],
		  "_wsRoot":"`+filepath.ToSlash(ws)+`"}`)
	if err != nil {
		t.Fatalf("update_tasks 失败: %v", err)
	}
	tasks := readTaskFiles(t, ws)
	if len(tasks) != 1 || tasks[0].ConvID != "" {
		t.Fatalf("无会话时应保持空 convId，实际 %+v", tasks)
	}
}

// TestReplaceAllForConvIsolatesConversations 多会话隔离：一个会话的全量替换
// 不得删除其它会话的任务（原 ReplaceAll 会清掉）。
func TestReplaceAllForConvIsolatesConversations(t *testing.T) {
	ws := t.TempDir()
	tm := NewTaskManager(ws)

	if err := tm.ReplaceAllForConv([]Task{{Subject: "A1"}, {Subject: "A2"}}, "conv-A"); err != nil {
		t.Fatalf("A 会话写入失败: %v", err)
	}
	if err := tm.ReplaceAllForConv([]Task{{Subject: "B1"}}, "conv-B"); err != nil {
		t.Fatalf("B 会话写入失败: %v", err)
	}

	if got := tm.ListByConvID("conv-A"); len(got) != 2 {
		t.Fatalf("B 会话替换后 A 任务应保留 2 条，实际 %d（跨会话误删）", len(got))
	}
	if got := tm.ListByConvID("conv-B"); len(got) != 1 {
		t.Fatalf("B 会话应 1 条，实际 %d", len(got))
	}

	// A 会话再次全量替换（只剩 1 条）→ 只清理 A 自己的旧任务
	if err := tm.ReplaceAllForConv([]Task{{Subject: "A1"}}, "conv-A"); err != nil {
		t.Fatalf("A 会话二次替换失败: %v", err)
	}
	if got := tm.ListByConvID("conv-A"); len(got) != 1 {
		t.Fatalf("A 会话应剩 1 条，实际 %d", len(got))
	}
	if got := tm.ListByConvID("conv-B"); len(got) != 1 {
		t.Fatalf("A 会话二次替换不应影响 B，实际 B=%d 条", len(got))
	}

	// 兼容入口：ReplaceAll（convID 空）只动「未绑定会话」的任务
	if err := tm.ReplaceAll([]Task{{Subject: "G1"}}); err != nil {
		t.Fatalf("ReplaceAll 失败: %v", err)
	}
	if got := tm.ListByConvID("conv-A"); len(got) != 1 {
		t.Fatalf("ReplaceAll 不应影响会话内任务，实际 %d", len(got))
	}
	// 注：ListByConvID("") 语义为「不过滤」（返回全部），故直接按文件统计
	global := 0
	for _, task := range readTaskFiles(t, ws) {
		if task.ConvID == "" {
			global++
		}
	}
	if global != 1 {
		t.Fatalf("未绑定会话的任务应 1 条，实际 %d", global)
	}
}
