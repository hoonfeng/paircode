package agent

// ═══════════════════════════════════════════════════════════════
// subagent_fork_test.go — Round3 ③.2：fork 机制测试
//
// 覆盖：TestAgentsForkSpec（spec 映射：ForkOf/seed 透传/记录字段）、
// TestJSPluginAgentsFork（ctx.agents.fork/report 桥端到端）。
// ═══════════════════════════════════════════════════════════════

import (
	"context"
	"strings"
	"testing"
)

// fakeForkSpawner 记录 fork 调用的假 Spawner。
type fakeForkSpawner struct {
	forked  []SubAgentSpec
	seeds   [][]Message
	reports map[string]string
}

func newFakeForkSpawner() *fakeForkSpawner {
	return &fakeForkSpawner{reports: map[string]string{}}
}

func (f *fakeForkSpawner) install(t *testing.T) {
	t.Helper()
	SetSubAgentSpawner(&SubAgentSpawner{
		Start: func(spec SubAgentSpec) error { return nil },
		Stop:  func(convID string) {},
		Running: func(convID string) bool { return false },
		LastAssistant: func(convID, wsRoot string) string { return "" },
		Models:        func() []map[string]any { return nil },
		Current:       func() map[string]any { return nil },
		ForkSeed: func(sourceConvID string) []Message {
			return []Message{{Role: RoleUser, Content: "seed:" + sourceConvID}}
		},
		Fork: func(spec SubAgentSpec, seed []Message) error {
			f.forked = append(f.forked, spec)
			f.seeds = append(f.seeds, seed)
			return nil
		},
	})
	t.Cleanup(func() { SetSubAgentSpawner(nil) })
}

// TestAgentsForkSpec spec 映射：ForkOf/任务/persona/模型/黑名单 + seed 透传 + 记录字段。
func TestAgentsForkSpec(t *testing.T) {
	sp := newFakeForkSpawner()
	sp.install(t)

	rec, err := ForkSubAgent(SubAgentSpec{
		ParentConv:      "conv-captain",
		Label:           "team-x/analyst",
		Team:            "team-x",
		Member:          "analyst",
		Task:            "fork 任务",
		System:          "你是分析师",
		Model:           "deepseek-v4-flash",
		Provider:        "deepseek-official",
		ReasoningEffort: "max",
		WsRoot:          "C:/ws",
		DenyTools:       []string{"update_plan"},
		ForkOf:          "conv-src-1",
	})
	if err != nil {
		t.Fatalf("ForkSubAgent: %v", err)
	}
	if rec.State != "running" || rec.ForkOf != "conv-src-1" {
		t.Errorf("记录字段异常: %+v", rec)
	}
	if len(sp.forked) != 1 {
		t.Fatalf("fork 应调用 1 次，得 %d", len(sp.forked))
	}
	spec := sp.forked[0]
	if spec.ForkOf != "conv-src-1" || spec.Task != "fork 任务" || spec.System != "你是分析师" ||
		spec.Model != "deepseek-v4-flash" || spec.Provider != "deepseek-official" ||
		spec.ReasoningEffort != "max" || len(spec.DenyTools) != 1 {
		t.Errorf("fork spec 映射异常: %+v", spec)
	}
	// seed = 源会话快照（ForkSeed 结果）
	if len(sp.seeds) != 1 || len(sp.seeds[0]) != 1 || !strings.Contains(sp.seeds[0][0].Content, "conv-src-1") {
		t.Errorf("fork seed 异常: %+v", sp.seeds)
	}
	// convID 自动生成 + 记录可查
	if spec.ConvID == "" || SubAgentInfo(spec.ConvID) == nil {
		t.Error("fork 后记录应可查")
	}

	// 参数校验：缺 task / 缺 forkFrom
	if _, err := ForkSubAgent(SubAgentSpec{Task: "x"}); err == nil {
		t.Error("缺 forkFrom 应报错")
	}
	if _, err := ForkSubAgent(SubAgentSpec{ForkOf: "src"}); err == nil {
		t.Error("缺 task 应报错")
	}

	// ReportSubAgent：写入记录，status/list 可读
	if err := ReportSubAgent(spec.ConvID, "进度 50%"); err != nil {
		t.Fatalf("report: %v", err)
	}
	info := SubAgentInfo(spec.ConvID)
	if info == nil || info.Report != "进度 50%" {
		t.Errorf("report 未写入记录: %+v", info)
	}
	listed := ListSubAgents("conv-captain", "")
	if len(listed) != 1 || listed[0].Report != "进度 50%" || listed[0].ForkOf != "conv-src-1" {
		t.Errorf("list 应含 report/forkOf: %+v", listed)
	}
}

// TestJSPluginAgentsFork ctx.agents.fork/report 桥端到端（JS 插件 execute）。
func TestJSPluginAgentsFork(t *testing.T) {
	sp := newFakeForkSpawner()
	sp.install(t)

	reg := NewRegistry()
	host := NewPluginHost(reg, nil, `C:\ws`)
	id, err := host.DefineJS(`
	return {
		name: 'fork-bridge-test',
		inject: ['agents'],
		apply(ctx) {
			ctx.tools.register({
				name: 'my_fork',
				description: 'fork bridge',
				parameters: { type: 'object', properties: {}, required: [] },
				execute: (args) => JSON.stringify(ctx.agents.fork({
					task: (args && args.task) || '',
					forkFrom: (args && args.forkFrom) || '',
					label: 'fork-bridge',
				})),
			})
			ctx.tools.register({
				name: 'my_report',
				description: 'report bridge',
				parameters: { type: 'object', properties: {}, required: [] },
				execute: (args) => JSON.stringify(ctx.agents.report((args && args._convID) || '', (args && args.text) || '')),
			})
		},
	}`, "fork-bridge-test")
	if err != nil {
		t.Fatalf("DefineJS: %v", err)
	}
	def, _ := host.GetJSDef(id)
	if err := host.LoadJSDynamic(def); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}

	// my_fork → ctx.agents.fork
	out, err := reg.Execute(context.Background(), "my_fork", `{"task":"JS fork 任务","forkFrom":"conv-src-js"}`)
	if err != nil {
		t.Fatalf("my_fork: %v", err)
	}
	if !strings.Contains(out, "conv") || !strings.Contains(out, "running") {
		t.Errorf("my_fork 输出异常: %s", out)
	}
	if len(sp.forked) != 1 || sp.forked[0].ForkOf != "conv-src-js" || sp.forked[0].Task != "JS fork 任务" {
		t.Errorf("ctx.agents.fork 桥参数异常: %+v", sp.forked)
	}

	// my_report → ctx.agents.report（_convID 注入）
	forkedConv := sp.forked[0].ConvID
	out, err = reg.Execute(context.Background(), "my_report", `{"text":"JS 报告","_convID":"`+forkedConv+`"}`)
	if err != nil {
		t.Fatalf("my_report: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("my_report 输出异常: %s", out)
	}
	if info := SubAgentInfo(forkedConv); info == nil || info.Report != "JS 报告" {
		t.Errorf("report 桥未写入: %+v", info)
	}
}
