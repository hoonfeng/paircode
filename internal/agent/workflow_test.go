package agent

// ═══════════════════════════════════════════════════════════════
// workflow_test.go — Round3 ③.3：workflow 运行器测试
//
// 覆盖：pipeline（逐项过阶段/失败项 null）、parallel（barrier/抛错 null）、
// phase/log 进度记录、args 注入、参数校验。
//
// ★ 2026-09 子 Agent 实现删除：agent(prompt, opts?) 钩子与其测试
//   （TestWorkflowRunner_Agent / TestWorkflowRunner_Cancel）一并移除——
//   脚本内不再有宿主子 Agent 委托能力。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestWorkflowRunner_Pipeline pipeline：逐项过阶段、阶段抛错 → null、多阶段链式。
func TestWorkflowRunner_Pipeline(t *testing.T) {
	out, err := RunWorkflow(context.Background(), `
const rs = pipeline([1, 2, 3], (prev, item, idx) => item * 10 + idx,
                    (prev, item) => prev + 1);
return rs;
`, nil)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	var res map[string]any
	_ = json.Unmarshal([]byte(out), &res)
	arr, _ := res["output"].([]any)
	// 阶段1: 1*10+0=10 → +1=11；2*10+1=21 → 22；3*10+2=32 → 33
	want := []float64{11, 22, 33}
	if len(arr) != 3 {
		t.Fatalf("pipeline 结果数异常: %v", arr)
	}
	for i := range want {
		if arr[i].(float64) != want[i] {
			t.Errorf("pipeline[%d] = %v, want %v", i, arr[i], want[i])
		}
	}

	// 阶段抛错 → 该项 null，其余项继续
	out, err = RunWorkflow(context.Background(), `
const rs = pipeline([1, 2, 3], (prev, item) => {
  if (item === 2) { throw new Error('boom'); }
  return item * 2;
});
return rs;
`, nil)
	if err != nil {
		t.Fatalf("RunWorkflow(throw): %v", err)
	}
	_ = json.Unmarshal([]byte(out), &res)
	arr, _ = res["output"].([]any)
	if len(arr) != 3 || arr[0].(float64) != 2 || arr[1] != nil || arr[2].(float64) != 6 {
		t.Errorf("pipeline 抛错项应为 null: %v", arr)
	}
}

// TestWorkflowRunner_Parallel parallel：barrier 等待、thunk 抛错 → null、结果序保持。
func TestWorkflowRunner_Parallel(t *testing.T) {
	out, err := RunWorkflow(context.Background(), `
const ps = parallel([
  () => 'a',
  () => 42,
  () => { throw new Error('thunk boom'); },
  () => true,
]);
return ps;
`, nil)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	var res map[string]any
	_ = json.Unmarshal([]byte(out), &res)
	arr, _ := res["output"].([]any)
	if len(arr) != 4 || arr[0] != "a" || arr[1].(float64) != 42 || arr[2] != nil || arr[3] != true {
		t.Errorf("parallel 结果异常（顺序保持、抛错 null）: %v", arr)
	}
}

// TestWorkflowRunner_PhaseLogArgs phase/log 进度记录 + args 注入。
func TestWorkflowRunner_PhaseLogArgs(t *testing.T) {
	out, err := RunWorkflow(context.Background(), `
phase('调研');
log('开始调研阶段');
const x = args.multiplier * 2;
phase('实施');
log('调研完成，x=' + x);
return x;
`, map[string]any{"multiplier": 21})
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	var res map[string]any
	_ = json.Unmarshal([]byte(out), &res)
	if res["output"].(float64) != 42 {
		t.Errorf("args 注入异常: %v", res["output"])
	}
	phases, _ := res["phases"].([]any)
	if len(phases) != 2 || phases[0] != "调研" || phases[1] != "实施" {
		t.Errorf("phases 记录异常: %v", phases)
	}
	logs, _ := res["logs"].([]any)
	if len(logs) < 2 || !strings.Contains(logs[1].(string), "开始调研") {
		t.Errorf("logs 记录异常: %v", logs)
	}
}

// TestWorkflowRunner_Errors 参数校验：空脚本 / agent 钩子已删除。
func TestWorkflowRunner_Errors(t *testing.T) {
	if _, err := RunWorkflow(context.Background(), "", nil); err == nil {
		t.Error("空 script 应报错")
	}
	// ★ 2026-09：agent() 钩子已随子 Agent 实现删除 → 脚本内调用应报「未定义」
	if _, err := RunWorkflow(context.Background(), `const a = agent('x'); return a;`, nil); err == nil {
		t.Error("agent() 钩子已删除，脚本内调用应报错（ReferenceError）")
	}
}
