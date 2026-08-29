// ═══════════════════════════════════════════════════════════════
// workflow.go — 宿主 workflow 运行器（Round3 ③.3，P2 最小可用）
//
// 对齐 DSH harness workflow 语义（goja 执行 workflow 脚本）：
//   - 脚本体（plain JS，top-level return 结果）+ meta{name,description,phases}
//   - 钩子：
//       agent(prompt, opts?) → 后台委托子 Agent 并同步等待其完成，返回最终正文
//                              （SpawnSubAgent + 轮询 subagent/idle + LastText）
//       pipeline(items, ...stages) → 逐项顺序过阶段（无 barrier），
//                                    阶段抛错 → 该项结果 null、跳过其余阶段
//       parallel(thunks) → 全部执行并等待（barrier）；thunk 抛错 → null
//                          ★ goja 单线程：JS 侧顺序调度，并发性由宿主子 Agent
//                          侧承载（天然满足「上限 4」防成员风暴）
//       phase(title) / log(msg) → 进度记录（随结果 JSON 返回）
//       args → workflow 输入参数（脚本内全局可读）
//   - 取消：ctx 取消时 agent 等待立即中止（错误返回）
//
// 范围声明（后续演进）：不做跨会话恢复/持久化队列（单次执行、结果即返回）。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// workflowAgentTimeout 单个 agent 钩子的最长等待（超时报错，防脚本悬挂）。
const workflowAgentTimeout = 20 * time.Minute

// workflowWallClockLimit 整个 workflow 脚本的墙钟上限（t4 F4：CPU 看门狗）。
// 只拦纯死循环/失控脚本；agent 等待在 Go 侧通道阻塞，Interrupt 不打断等待本身。
const workflowWallClockLimit = 90 * time.Minute

// workflowRunner 一次 workflow 执行的运行态。
type workflowRunner struct {
	vm     *goja.Runtime
	ctx    context.Context
	args   map[string]any
	logs   []string
	phases []string
}

// ─── workflow 工具路由执行器（插件 execute → ctx.hostTool.exec） ────

// archiveWorkflowTool 将 workflow 工具的路由执行器存档到 hostTool 索引
// （编排在插件、能力在宿主，与 goal/ask_user 同构）。
func archiveWorkflowTool() {
	ArchiveHostTool(&Tool{
		Name:       "workflow",
		SystemTool: true,
		Description: "执行一个 workflow 编排脚本（对齐 DSH harness workflow）。script 为 JS 函数体（末尾 return 结果）；" +
			"钩子：agent(prompt, opts?) 后台委托子 Agent 并等待完成返回其最终正文；pipeline(items, ...stages) 逐项过阶段；" +
			"parallel(thunks) 批量执行（宿主侧并发）；phase(title)/log(msg) 记录进度；args 为脚本内可读输入。返回 JSON {ok, output, logs, phases}。",
		Parameters: objSchema(props{
			"script": strProp("workflow 脚本体（JS，末尾 return 结果；可用 agent/pipeline/parallel/phase/log/args）"),
			"meta":   strProp("可选：{name, description, phases} 元信息（记录用）"),
			"args":   strProp("可选：输入参数 JSON 对象（脚本内 args 全局可读）"),
		}, "script"),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			script := argStr(args, "script")
			if strings.TrimSpace(script) == "" {
				return "", fmt.Errorf("workflow：script 不能为空")
			}
			var wfArgs map[string]any
			if a := argStr(args, "args"); strings.TrimSpace(a) != "" {
				if err := json.Unmarshal([]byte(a), &wfArgs); err != nil {
					return "", fmt.Errorf("workflow：args 不是合法 JSON 对象: %v", err)
				}
			}
			return RunWorkflow(ctx, script, wfArgs)
		},
	})
}

// RunWorkflow 执行 workflow 脚本（script 为函数体，末尾 return 结果）。
// 返回 JSON：{ok, output, logs, phases}（output 为脚本 return 值）。
func RunWorkflow(ctx context.Context, script string, args map[string]any) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("workflow：script 不能为空")
	}
	vm := goja.New()
	if args == nil {
		args = map[string]any{} // 脚本内 args 恒为对象（null 上读属性会抛错）
	}
	r := &workflowRunner{vm: vm, ctx: ctx, args: args}
	r.installGlobals()

	var outVal goja.Value
	// ★ 脚本体按函数体执行（DSH workflow 语义：脚本以 return <value> 结尾）——
	// goja 顶层不允许 return，包一层立即执行函数。
	wrapped := "(function(){\n" + script + "\n})()"
	// ★ t4 F4：CPU 看门狗——纯死循环 JS 脚本会挂死执行 goroutine，需 Interrupt 兜底。
	//   agent() 等待在 Go 侧通道阻塞（JS 不执行），Interrupt 只在 JS 真正执行时生效；
	//   上限取 90 分钟墙钟（远大于多 agent 串联等待预算 20min×N），只拦死循环不误伤长任务。
	runErr := runJSWithTimeout(vm, workflowWallClockLimit, func() error {
		var err error
		outVal, err = vm.RunString(wrapped)
		return err
	})

	result := map[string]any{
		"ok":     runErr == nil,
		"logs":   r.logs,
		"phases": r.phases,
	}
	if runErr != nil {
		result["error"] = runErr.Error()
	} else if outVal != nil && !goja.IsUndefined(outVal) && !goja.IsNull(outVal) {
		result["output"] = outVal.Export()
	} else {
		result["output"] = nil
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	if runErr != nil {
		return string(b), runErr
	}
	return string(b), nil
}

// installGlobals 注入 workflow 钩子（agent/pipeline/parallel/phase/log/args）。
func (r *workflowRunner) installGlobals() {
	vm := r.vm
	// console.log → 进度日志
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, fmt.Sprintf("%v", a.Export()))
		}
		r.logs = append(r.logs, strings.Join(parts, " "))
		return goja.Undefined()
	})
	vm.Set("console", console)

	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, fmt.Sprintf("%v", a.Export()))
		}
		r.logs = append(r.logs, strings.Join(parts, " "))
		return goja.Undefined()
	})

	vm.Set("phase", func(call goja.FunctionCall) goja.Value {
		title := call.Argument(0).String()
		r.phases = append(r.phases, title)
		r.logs = append(r.logs, "[phase] "+title)
		return goja.Undefined()
	})

	vm.Set("args", r.args)

	// agent(prompt, opts?)：后台委托子 Agent 并同步等待完成，返回最终正文。
	vm.Set("agent", func(call goja.FunctionCall) goja.Value {
		prompt := call.Argument(0).String()
		if strings.TrimSpace(prompt) == "" {
			panic(vm.NewTypeError("workflow agent：prompt 不能为空"))
		}
		spec := SubAgentSpec{Task: prompt}
		if a := call.Argument(1); a != nil && !goja.IsUndefined(a) && !goja.IsNull(a) {
			if obj, ok := a.Export().(map[string]any); ok {
				spec.Label = mapStr(obj, "label")
				spec.Team = mapStr(obj, "team")
				spec.Member = mapStr(obj, "member")
				spec.System = mapStr(obj, "system")
				spec.Model = mapStr(obj, "model")
				spec.Provider = mapStr(obj, "provider")
				spec.ReasoningEffort = mapStr(obj, "reasoningEffort")
				spec.WsRoot = mapStr(obj, "wsRoot")
				spec.DenyTools = mapStrSlice(obj, "denyTools")
			}
		}
		text, err := r.awaitAgent(spec)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("workflow agent 失败: %v", err)))
		}
		return vm.ToValue(text)
	})

	// pipeline(items, ...stages)：逐项顺序过阶段；阶段抛错 → 该项 null。
	vm.Set("pipeline", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("workflow pipeline：需要 items 与至少一个 stage"))
		}
		itemsVal := call.Argument(0).Export()
		items, ok := itemsVal.([]any)
		if !ok {
			panic(vm.NewTypeError("workflow pipeline：items 必须是数组"))
		}
		stages := call.Arguments[1:]
		results := make([]any, 0, len(items))
		for _, item := range items {
			prev := item
			failed := false
			for _, st := range stages {
				fn, ok := goja.AssertFunction(st)
				if !ok {
					panic(vm.NewTypeError("workflow pipeline：stage 必须是函数"))
				}
				v, err := fn(goja.Undefined(), vm.ToValue(prev), vm.ToValue(item), vm.ToValue(len(results)))
				if err != nil {
					failed = true
					break
				}
				prev = v.Export()
			}
			if failed {
				results = append(results, nil) // 该项失败 → null，继续下一项
			} else {
				results = append(results, prev)
			}
		}
		return vm.ToValue(results)
	})

	// parallel(thunks)：全部执行并等待（barrier）；thunk 抛错 → null。
	// ★ goja 单线程模型：JS 侧顺序调度（每个 thunk 完整执行完再下一个）；
	//   并发性由 thunk 内 agent() 委托的宿主子 Agent 承载（宿主多会话并行），
	//   天然满足「并行上限 4」的防成员风暴约束（顺序调度不可能超发）。
	vm.Set("parallel", func(call goja.FunctionCall) goja.Value {
		arrVal := call.Argument(0)
		if arrVal == nil || goja.IsUndefined(arrVal) || goja.IsNull(arrVal) {
			panic(vm.NewTypeError("workflow parallel：需要一个 thunk 函数数组"))
		}
		arrObj := arrVal.ToObject(vm)
		lengthVal := arrObj.Get("length")
		n := int(lengthVal.ToInteger())
		results := make([]any, 0, n)
		for i := 0; i < n; i++ {
			item := arrObj.Get(fmt.Sprintf("%d", i))
			fn, ok := goja.AssertFunction(item)
			if !ok {
				results = append(results, nil)
				continue
			}
			v, err := fn(goja.Undefined())
			if err != nil {
				results = append(results, nil) // thunk 抛错 → null
				continue
			}
			results = append(results, v.Export())
		}
		return vm.ToValue(results)
	})
}

// awaitAgent 委托子 Agent 并等待完成：SpawnSubAgent → 轮询状态到 idle →
// SubAgentLastText 取最终正文。ctx 取消 / 超时中止。
func (r *workflowRunner) awaitAgent(spec SubAgentSpec) (string, error) {
	if !SubAgentSpawnerReady() {
		return "", fmt.Errorf("子 Agent 能力未就绪（会话启动器未注入）")
	}
	rec, err := SpawnSubAgent(spec)
	if err != nil {
		return "", err
	}
	convID := rec.ConvID
	deadline := time.Now().Add(workflowAgentTimeout)
	for {
		select {
		case <-r.ctx.Done():
			StopSubAgent(convID)
			return "", r.ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			StopSubAgent(convID)
			return "", fmt.Errorf("workflow agent 等待超时（%s）conv=%s", workflowAgentTimeout, convID)
		}
		info := SubAgentInfo(convID)
		if info == nil {
			continue
		}
		if info.State != "running" {
			if info.LastError != "" {
				return "", fmt.Errorf("workflow agent 出错 conv=%s: %s", convID, info.LastError)
			}
			text := SubAgentLastText(convID)
			if strings.TrimSpace(text) == "" {
				text = "(子 Agent 无正文输出)"
			}
			log.Printf("[workflow] agent 完成 conv=%s text=%d 字符", convID, len(text))
			return text, nil
		}
	}
}
