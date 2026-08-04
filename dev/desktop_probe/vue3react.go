// Command vue3react checks Vue-3-style reactive with async scheduler:
// does a state mutation scheduled via Promise.then(flushJobs) actually
// propagate to a DOM className update in the wb-ui jsc engine?
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
// ===== Vue 3 scheduler（Promise.then 驱动，与真实 Vue 同构） =====
var queue = [];
var flushing = false;
function queueJob(job) {
  if (queue.indexOf(job) === -1) queue.push(job);
  queueFlush();
}
function queueFlush() {
  if (!flushing) {
    flushing = true;
    Promise.resolve().then(flushJobs);
  }
}
function flushJobs() {
  var jobs = queue.slice();
  queue = [];
  flushing = false;
  for (var i = 0; i < jobs.length; i++) jobs[i]();
  // 嵌套 job（组件更新内再触发更新）继续处理
  if (queue.length) queueFlush();
}

// ===== Vue 3 reactive（WeakMap + Reflect + track/trigger） =====
var targetMap = new WeakMap();
var activeEffect = null;
function track(target, key) {
  if (!activeEffect) return;
  var depsMap = targetMap.get(target);
  if (!depsMap) { depsMap = new Map(); targetMap.set(target, depsMap); }
  var dep = depsMap.get(key);
  if (!dep) { dep = new Set(); depsMap.set(key, dep); }
  if (!dep.has(activeEffect)) dep.add(activeEffect);
}
function trigger(target, key) {
  var depsMap = targetMap.get(target);
  if (!depsMap) return;
  var dep = depsMap.get(key);
  if (!dep) return;
  var effects = Array.from(dep);
  for (var i = 0; i < effects.length; i++) effects[i]();
}
function reactive(obj) {
  return new Proxy(obj, {
    get(t, k, r) { track(t, k); return Reflect.get(t, k, r); },
    set(t, k, v, r) {
      var old = t[k];
      var ok = Reflect.set(t, k, v, r);
      if (old !== v) trigger(t, k);
      return ok;
    }
  });
}
function effect(fn) {
  var wrapped = function() {
    var prev = activeEffect;
    activeEffect = wrapped;
    try { fn(); } finally { activeEffect = prev; }
  };
  wrapped.fn = fn;
  wrapped();
  return wrapped;
}

// ===== 模拟组件：读 state.currentConvId 渲染 active 类 =====
var state = reactive({ currentConvId: 'A' });
var domItems = [{ id: 'A', className: 'conv-item active' }, { id: 'B', className: 'conv-item' }, { id: 'C', className: 'conv-item' }];
var renders = [];
function renderSidebar() {
  for (var i = 0; i < domItems.length; i++) {
    var isActive = domItems[i].id === state.currentConvId;
    domItems[i].className = 'conv-item' + (isActive ? ' active' : '');
  }
  renders.push('render current=' + state.currentConvId + ' classes=' + domItems.map(function(d){return d.className}).join('|'));
}
effect(renderSidebar);
console.log('INIT: ' + renders[renders.length-1]);

// 模拟点击切换对话（同步更新 state，调度异步 flush）
state.currentConvId = 'B';
console.log('AFTER-SET: ' + renders[renders.length-1]);
console.log('CLASSES-NOW: ' + domItems.map(function(d){return d.className}).join('|'));
`
	_, err := rt.RunJS(script)
	if err != nil {
		fmt.Println("RunJS error:", err)
	}
	// 手动 flush goja 微任务（对应 handleClick 里的 RunJobs）
	rt.RunJobs()
	fmt.Println("--- console ---")
	fmt.Println(logger.String())
	fmt.Println("--- done ---")
}
