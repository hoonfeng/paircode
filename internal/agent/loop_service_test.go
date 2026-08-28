// loop_service_test.go — loop 服务面（一切皆插件：ctx.get('loop')）验证。
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"wb-ui/goja"
)

// loopSvcProvider 固定返回 read_file 工具调用的 mock Provider（计数调用次数+时间戳）。
type loopSvcProvider struct {
	n    atomic.Int32
	last atomic.Int64 // 上次 Chat 调用 UnixNano
}

func (p *loopSvcProvider) Name() string { return "svc" }
func (p *loopSvcProvider) Chat(ctx context.Context, m []Message, td []ToolDefinition, oc func(Chunk)) (Message, error) {
	p.n.Add(1)
	p.last.Store(time.Now().UnixNano())
	return Message{Role: RoleAssistant, ToolCalls: []ToolCall{
		{ID: "x", Type: "function", Function: FunctionCall{Name: "read_file", Arguments: `{"path":"x.txt"}`}},
	}}, nil
}

// withLoopSvcHost 设置全局插件宿主（测试专用；结束恢复原宿主）。
func withLoopSvcHost(t *testing.T) *PluginHost {
	t.Helper()
	prev := GetGlobalPluginHost()
	ph := NewPluginHost(NewRegistry(), nil, t.TempDir())
	SetGlobalPluginHost(ph)
	t.Cleanup(func() { SetGlobalPluginHost(prev) })
	return ph
}

// loopSvcEnv 造环境：临时目录 + 默认工具 + mock provider。
func loopSvcEnv(t *testing.T) (string, *loopSvcProvider) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	prov := &loopSvcProvider{}
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	return dir, prov
}

// 插件侧取 loop 服务（模拟 ctx.get('loop')）。
func loopSvcOf(ph *PluginHost) *LoopService {
	v := ph.Context().Get("loop")
	if v == nil {
		return nil
	}
	svc, _ := v.(*LoopService)
	return svc
}

// waitLoopSvc 等 loop 服务注册（Run 内 Provide 在 for 循环前，早于首次 Chat；
// mock 循环毫秒级跑完，等 provider 计数不可靠）。
func waitLoopSvc(t *testing.T, ph *PluginHost) *LoopService {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if svc := loopSvcOf(ph); svc != nil {
			return svc
		}
		if time.Now().After(deadline) {
			t.Fatal("等待 loop 服务注册超时（Run 未执行 Provide？）")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// 查询面：Run 期间服务可用且状态反映 turn/step/phase；Run 结束自动撤销（get 回 nil）。
func TestLoopService_GetState(t *testing.T) {
	ph := withLoopSvcHost(t)
	dir, prov := loopSvcEnv(t)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	loop := &Loop{Provider: prov, Registry: reg, MaxIterations: 100, WorkspaceRoot: dir}

	done := make(chan struct{})
	go func() {
		loop.Run(context.Background(), "svc test", nil)
		close(done)
	}()

	svc := waitLoopSvc(t, ph)
	// 等至少一轮迭代（状态快照有内容）
	deadline := time.Now().Add(3 * time.Second)
	for svc.GetState().Step == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	st := svc.GetState()
	if !st.Running {
		t.Error("运行中 Running 应为 true")
	}
	if st.Turn < 1 || st.Step < 1 {
		t.Errorf("Turn/Step 应 ≥1，得 %d/%d", st.Turn, st.Step)
	}
	if st.Phase == "" {
		t.Error("Phase 应有最近事件阶段")
	}
	<-done
	// Run 结束：服务已撤销（get 回 nil）
	if v := loopSvcOf(ph); v != nil {
		t.Errorf("Run 结束后 get('loop') 应为 nil，得 %T", v)
	}
}

// 控制面·暂停/继续：Pause 后迭代暂停（暂停期间 provider 调用不再增长），Resume 恢复。
func TestLoopService_PauseResume(t *testing.T) {
	ph := withLoopSvcHost(t)
	dir, prov := loopSvcEnv(t)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	loop := &Loop{Provider: prov, Registry: reg, MaxIterations: 100, WorkspaceRoot: dir}

	done := make(chan struct{})
	go func() {
		loop.Run(context.Background(), "pause test", nil)
		close(done)
	}()

	svc := waitLoopSvc(t, ph)
	// 等至少 2 轮迭代（Pause 请求在下轮迭代开始处生效，需有已排队的迭代）
	deadline := time.Now().Add(3 * time.Second)
	for prov.n.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	svc.Pause()
	if !svc.GetState().Paused {
		t.Error("Pause 后 Paused 应为 true")
	}
	before := prov.n.Load()
	time.Sleep(400 * time.Millisecond) // 暂停期间
	after := prov.n.Load()
	if after > before+1 {
		t.Errorf("暂停期间不应继续迭代（before=%d, after=%d）", before, after)
	}
	svc.Resume()
	if svc.GetState().Paused {
		t.Error("Resume 后 Paused 应为 false")
	}
	deadline = time.Now().Add(3 * time.Second)
	for prov.n.Load() < after+2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if prov.n.Load() < after+2 {
		t.Errorf("Resume 后应继续迭代，得 %d", prov.n.Load())
	}
	<-done
}

// 控制面·停止：RequestStop 后 Run 提前退出（CancelCause.kind=plugin）。
func TestLoopService_RequestStop(t *testing.T) {
	ph := withLoopSvcHost(t)
	dir, prov := loopSvcEnv(t)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	loop := &Loop{Provider: prov, Registry: reg, MaxIterations: 100, WorkspaceRoot: dir}

	done := make(chan struct{})
	var runErr error
	go func() {
		_, runErr = loop.Run(context.Background(), "stop test", nil)
		close(done)
	}()

	svc := waitLoopSvc(t, ph)
	deadline := time.Now().Add(3 * time.Second)
	for prov.n.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	svc.RequestStop()
	<-done

	if runErr == nil {
		t.Error("RequestStop 后 Run 应返回错误")
	}
	if loop.CancelCause.Kind != CancelByPlugin {
		t.Errorf("CancelCause.Kind 应为 %q，得 %q", CancelByPlugin, loop.CancelCause.Kind)
	}
	if prov.n.Load() >= 100 {
		t.Errorf("应提前停止（未跑满 100 次），实际 %d", prov.n.Load())
	}
}

// JS 插件视角：goja 沙箱内 ctx.get('loop') 拿到服务对象并可调用 GetState()。
// （模拟插件 apply 里 `const svc = ctx.get('loop'); svc.GetState()` 的运行时形态。）
func TestLoopService_GojaView(t *testing.T) {
	ph := withLoopSvcHost(t)
	dir, prov := loopSvcEnv(t)
	reg := NewRegistry()
	RegisterDefaultTools(reg, dir)
	loop := &Loop{Provider: prov, Registry: reg, MaxIterations: 100, WorkspaceRoot: dir}

	done := make(chan struct{})
	go func() {
		loop.Run(context.Background(), "goja view", nil)
		close(done)
	}()

	svc := waitLoopSvc(t, ph)
	// 等至少一轮迭代（快照有内容）
	deadline := time.Now().Add(3 * time.Second)
	for svc.GetState().Step == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	vm := goja.New()
	_ = vm.Set("SVC", svc)
	val, err := vm.RunString(`(() => {
		const o = SVC.GetState();
		return o.Running + '/' + o.Turn + '/' + o.Step;
	})()`)
	if err != nil {
		t.Fatalf("goja 调用 GetState 失败: %v", err)
	}
	got := val.String()
	if !strings.HasPrefix(got, "true/") {
		t.Errorf("Running 应为 true（goja 视图），得 %q", got)
	}
	<-done
}
