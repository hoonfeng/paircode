package agent

// ═══════════════════════════════════════════════════════════════
// workflow_test.go — Round3 ③.3：workflow 运行器测试
//
// 覆盖：agent 钩子（后台委托 + 结果回收）、pipeline（逐项过阶段/失败项 null）、
// parallel（barrier/抛错 null）、phase/log 进度记录、args 注入、ctx 取消。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// installWorkflowFakeSpawner 假 Spawner：Start 只记录；Running 恒 false
// （首轮轮询即 idle）；LastAssistant 返回「已完成: <label>」。
func installWorkflowFakeSpawner(t *testing.T) *[]SubAgentSpec {
	t.Helper()
	started := &[]SubAgentSpec{}
	SetSubAgentSpawner(&SubAgentSpawner{
		Start: func(spec SubAgentSpec) error {
			*started = append(*started, spec)
			return nil
		},
		Stop: func(convID string) {},
		Running: func(convID string) bool { return false },
		LastAssistant: func(convID, wsRoot string) string {
			return "已完成: " + convID
		},
		Models:  func() []map[string]any { return nil },
		Current: func() map[string]any { return nil },
	})
	t.Cleanup(func() { SetSubAgentSpawner(nil) })
	return started
}

// TestWorkflowRunner_Agent agent 钩子：后台委托 + 等待 + 最终正文回收。
func TestWorkflowRunner_Agent(t *testing.T) {
	started := installWorkflowFakeSpawner(t)

	out, err := RunWorkflow(context.Background(), `
const a = agent('分析模块结构', {label: 'wf-analyst', system: '你是分析师'});
return {answer: a, label: 'done'};
`, nil)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("结果非 JSON: %v（%s）", err, out)
	}
	if res["ok"] != true {
		t.Fatalf("ok 应为 true: %s", out)
	}
	output, _ := res["output"].(map[string]any)
	answer, _ := output["answer"].(string)
	if !strings.Contains(answer, "已完成") {
		t.Errorf("agent 结果回收异常: %q", answer)
	}
	if len(*started) != 1 || (*started)[0].Task != "分析模块结构" || (*started)[0].Label != "wf-analyst" || (*started)[0].System != "你是分析师" {
		t.Errorf("agent 委托 spec 异常: %+v", *started)
	}
}

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

// TestWorkflowRunner_Cancel ctx 取消：agent 等待立即中止。
func TestWorkflowRunner_Cancel(t *testing.T) {
	installWorkflowFakeSpawner(t)
	ctx, cancel := context.WithCancel(context.Background())

	type wfResult struct {
		out string
		err error
	}
	ch := make(chan wfResult, 1)
	go func() {
		out, err := RunWorkflow(ctx, `const a = agent('长任务'); return a;`, nil)
		ch <- wfResult{out, err}
	}()

	time.Sleep(150 * time.Millisecond) // 等 agent 委托发生
	cancel()
	select {
	case r := <-ch:
		if r.err == nil || !strings.Contains(r.err.Error(), "canceled") {
			t.Errorf("取消后应返回 canceled 错误: %v（%s）", r.err, r.out)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("取消后 workflow 未退出")
	}
}

// TestWorkflowRunner_Errors 参数校验：空脚本 / agent 空 prompt / 未注入 spawner。
func TestWorkflowRunner_Errors(t *testing.T) {
	if _, err := RunWorkflow(context.Background(), "", nil); err == nil {
		t.Error("空 script 应报错")
	}
	// 未注入 spawner → agent 明确报错
	SetSubAgentSpawner(nil)
	if _, err := RunWorkflow(context.Background(), `const a = agent('x'); return a;`, nil); err == nil {
		t.Error("spawner 未注入时 agent 应报错")
	}
}
