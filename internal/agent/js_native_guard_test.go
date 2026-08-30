// js_native_guard_test.go —— JS 超时看门狗「原生调用感知」验证
//
// 覆盖回归点（2026-09-01「handler runCommand 执行超时（疑似死循环）」误判）：
//   - 原生阻塞调用（长命令/长 IO）期间不计时 → 不再误判超时
//   - 纯 JS 死循环仍被 Interrupt 中断（护栏未丢）
//   - guard 包装保持错误/返回值语义（异常可被 JS try/catch 捕获、返回值不变）
//   - 嵌套原生调用（原生内再调原生）深度计数正确
package agent

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/goja"
)

// TestNativeGuardBlockingNotTimeout 长时原生调用不被误判超时（核心回归）。
func TestNativeGuardBlockingNotTimeout(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	obj := vm.NewObject()
	// 模拟 ctx.bash.exec：Go 侧阻塞 1.2s（远超下方 300ms 超时）
	obj.Set("exec", func(call goja.FunctionCall) goja.Value {
		time.Sleep(1200 * time.Millisecond)
		return vm.ToValue("done")
	})
	vm.Set("svc", obj)
	wrapNativeGuards(vm, obj, 1)

	start := time.Now()
	err := runJSWithTimeout(vm, 300*time.Millisecond, func() error {
		_, e := vm.RunString(`svc.exec()`)
		return e
	})
	if err != nil {
		t.Fatalf("原生阻塞调用不应超时（native-aware 看门狗暂停计时）: %v", err)
	}
	if el := time.Since(start); el < time.Second {
		t.Fatalf("原生调用应真实执行完（耗时 %v）", el)
	}
}

// TestNativeGuardBlockingWithoutGuardWouldTimeout 对照组：不包装时同样场景会被
// 中断——证明 guard 是修复关键（且 flag 在原生返回后才生效，正是历史误判成因）。
func TestNativeGuardBlockingWithoutGuardWouldTimeout(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	obj := vm.NewObject()
	obj.Set("exec", func(call goja.FunctionCall) goja.Value {
		time.Sleep(600 * time.Millisecond)
		return vm.ToValue("done")
	})
	vm.Set("svc", obj)
	// 故意不包装
	err := runJSWithTimeout(vm, 200*time.Millisecond, func() error {
		// 原生调用后还有 JS 指令 → 撞上 interrupt flag
		_, e := vm.RunString(`svc.exec(); var x = 1; x`)
		return e
	})
	if err == nil || !isJSTimeout(err) {
		t.Fatalf("未包装时应复现历史误判（原生返回后撞 flag）：err=%v", err)
	}
}

// TestNativeGuardPureJSLoopStillInterrupted 纯 JS 死循环护栏保留。
func TestNativeGuardPureJSLoopStillInterrupted(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	obj := vm.NewObject()
	obj.Set("noop", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	vm.Set("svc", obj)
	wrapNativeGuards(vm, obj, 1)

	start := time.Now()
	err := runJSWithTimeout(vm, 400*time.Millisecond, func() error {
		_, e := vm.RunString(`while (true) { }`)
		return e
	})
	if err == nil || !isJSTimeout(err) {
		t.Fatalf("纯 JS 死循环应被中断：err=%v", err)
	}
	if el := time.Since(start); el > 5*time.Second {
		t.Fatalf("死循环中断过慢：%v", el)
	}
}

// TestNativeGuardLoopCallingNativeStillInterrupted 死循环中反复调用原生函数
// （每次都很快）——纯 JS 时间仍在累计，应被中断（防「靠调原生绕过护栏」）。
func TestNativeGuardLoopCallingNativeStillInterrupted(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	obj := vm.NewObject()
	obj.Set("tick", func(call goja.FunctionCall) goja.Value { return vm.ToValue(1) })
	vm.Set("svc", obj)
	wrapNativeGuards(vm, obj, 1)

	err := runJSWithTimeout(vm, 400*time.Millisecond, func() error {
		_, e := vm.RunString(`var n = 0; while (true) { n += svc.tick(); }`)
		return e
	})
	if err == nil || !isJSTimeout(err) {
		t.Fatalf("快速原生调用的死循环仍应被中断：err=%v", err)
	}
}

// TestNativeGuardPreservesSemantics guard 包装不改变返回值/异常语义。
func TestNativeGuardPreservesSemantics(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	obj := vm.NewObject()
	obj.Set("echo", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(call.Argument(0).String() + "!")
	})
	obj.Set("boom", func(call goja.FunctionCall) goja.Value {
		panic(vm.NewGoError(errTestNativeBoom))
	})
	obj.Set("nested", vm.NewObject()) // 二层对象递归包装
	nested := obj.Get("nested").(*goja.Object)
	nested.Set("deep", func(call goja.FunctionCall) goja.Value { return vm.ToValue("deep-ok") })
	vm.Set("svc", obj)
	wrapNativeGuards(vm, obj, 1)

	v, err := vm.RunString(`svc.echo("hi")`)
	if err != nil || v.String() != "hi!" {
		t.Fatalf("返回值语义被破坏：v=%v err=%v", v, err)
	}
	v, err = vm.RunString(`svc.nested.deep()`)
	if err != nil || v.String() != "deep-ok" {
		t.Fatalf("嵌套对象包装异常：v=%v err=%v", v, err)
	}
	// 异常可被 JS try/catch 捕获且消息保留
	v, err = vm.RunString(`(function(){ try { svc.boom(); return 'no-throw' } catch (e) { return 'caught:' + e.message } })()`)
	if err != nil {
		t.Fatalf("异常传播异常：%v", err)
	}
	if !strings.Contains(v.String(), "caught:") || !strings.Contains(v.String(), "原生炸裂") {
		t.Fatalf("异常语义被破坏：%q", v.String())
	}
}

// TestNativeGuardNestedDepth 嵌套原生调用深度计数：内层结束后外层仍算原生忙。
func TestNativeGuardNestedDepth(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	if jsNativeBusy(vm) {
		t.Fatal("初始不应为忙")
	}
	r1 := jsEnterNative(vm)
	r2 := jsEnterNative(vm)
	if !jsNativeBusy(vm) {
		t.Fatal("进入原生调用后应为忙")
	}
	r2()
	if !jsNativeBusy(vm) {
		t.Fatal("内层退出后外层仍在原生调用中")
	}
	r1()
	if jsNativeBusy(vm) {
		t.Fatal("全部退出后应为空闲")
	}
	jsForgetNative(vm)
	if jsNativeBusy(vm) {
		t.Fatal("清理后应为空闲")
	}
}

// TestApplyNativeGuardsServiceList applyNativeGuards 按服务名单生效（未在名单
// 的注册面不包装——保留死循环检测灵敏度）。
func TestApplyNativeGuardsServiceList(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	ctxObj := vm.NewObject()
	bash := vm.NewObject()
	bashCalled := false
	bash.Set("exec", func(call goja.FunctionCall) goja.Value {
		bashCalled = jsNativeBusy(vm) // 包装生效时调用期间应为忙
		return goja.Undefined()
	})
	ctxObj.Set("bash", bash)
	events := vm.NewObject()
	eventsBusy := false
	events.Set("emit", func(call goja.FunctionCall) goja.Value {
		eventsBusy = jsNativeBusy(vm)
		return goja.Undefined()
	})
	ctxObj.Set("events", events) // 不在名单
	vm.Set("ctx", ctxObj)
	applyNativeGuards(vm, ctxObj)

	if _, err := vm.RunString(`ctx.bash.exec(); ctx.events.emit()`); err != nil {
		t.Fatal(err)
	}
	if !bashCalled {
		t.Fatal("bash 服务应被包装（调用期间标记原生忙）")
	}
	if eventsBusy {
		t.Fatal("events（注册面）不应被包装")
	}
}

// errTestNativeBoom 测试用原生错误。
var errTestNativeBoom = errors.New("原生炸裂")

// TestWrapNativeGlobals 顶层全局函数包装：workflow agent() 阻塞期间计数忙，
// 退出后恢复空闲；不存在的全局名不报错。
func TestWrapNativeGlobals(t *testing.T) {
	vm := goja.New()
	defer jsForgetNative(vm)
	busyDuringCall := false
	_ = vm.Set("agent", func(call goja.FunctionCall) goja.Value {
		busyDuringCall = jsNativeBusy(vm)
		return goja.Undefined()
	})
	wrapNativeGlobals(vm, "agent", "no_such_global")

	if _, err := vm.RunString(`agent()`); err != nil {
		t.Fatal(err)
	}
	if !busyDuringCall {
		t.Fatal("agent 应被包装（调用期间标记原生忙）")
	}
	if jsNativeBusy(vm) {
		t.Fatal("返回后应为空闲")
	}
}
