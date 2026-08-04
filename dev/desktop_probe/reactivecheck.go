// Command reactivecheck verifies Vue-3-style reactive (WeakMap + Reflect +
// effect scheduling) support in the wb-ui jsc engine. If Vue's reactive
// fails to trigger re-render here, the desktop app's conv-switch click
// (state.currentConvId = id → DOM class update) will silently no-op.
package main

import (
	"fmt"

	"wb-ui/jsc"
)

func main() {
	rt := jsc.NewInterpreter()
	logger := &jsc.BufferLogger{}
	rt.SetupGlobal(logger)
	_ = rt.GetEventLoop()

	script := `
// ===== Vue 3 runtime.xxx 核心（简化但同构） =====
var targetMap = new WeakMap();
var activeEffect = null;
var effectStack = [];

function track(target, type, key) {
  if (activeEffect == null) return;
  var depsMap = targetMap.get(target);
  if (!depsMap) { depsMap = new Map(); targetMap.set(target, depsMap); }
  var dep = depsMap.get(key);
  if (!dep) { dep = new Set(); depsMap.set(key, dep); }
  if (!dep.has(activeEffect)) dep.add(activeEffect);
}
function trigger(target, type, key) {
  var depsMap = targetMap.get(target);
  if (!depsMap) return;
  var dep = depsMap.get(key);
  if (dep) {
    var effects = Array.from(dep);
    for (var i = 0; i < effects.length; i++) effects[i]();
  }
}
function reactive(target) {
  return new Proxy(target, {
    get(t, key, receiver) {
      var res = Reflect.get(t, key, receiver);
      track(t, 'get', key);
      return res;
    },
    set(t, key, value, receiver) {
      var old = t[key];
      var result = Reflect.set(t, key, value, receiver);
      if (old !== value) trigger(t, 'set', key);
      return result;
    },
    has(t, key) { track(t, 'has', key); return Reflect.has(t, key); },
    deleteProperty(t, key) {
      var had = Reflect.has(t, key);
      var result = Reflect.deleteProperty(t, key);
      if (had) trigger(t, 'delete', key);
      return result;
    }
  });
}
function effect(fn) {
  if (fn.effect) { fn = fn.effect.fn; }
  var wrapped = function() {
    try { effectStack.push(wrapped); activeEffect = wrapped; return fn(); }
    finally { effectStack.pop(); activeEffect = effectStack[effectStack.length - 1]; }
  };
  wrapped.fn = fn;
  wrapped();
  return wrapped;
}

// ===== 模拟 Vue 组件渲染 =====
var state = reactive({ currentConvId: 'A', messages: [] });
var renders = [];
function renderSidebar() {
  renders.push('sidebar current=' + state.currentConvId);
}
effect(renderSidebar);
console.log('R1=' + JSON.stringify(renders));

// 模拟点击切换对话
state.currentConvId = 'B';
console.log('R2=' + JSON.stringify(renders));

// 模拟嵌套对象 + 数组
state.messages.push('hello');
console.log('R3=' + JSON.stringify(renders) + ' msgs=' + state.messages.length);
`
	_, err := rt.RunJS(script)
	if err != nil {
		fmt.Println("RunJS error:", err)
	}
	fmt.Println("--- console ---")
	fmt.Println(logger.String())
	fmt.Println("--- done ---")
}
