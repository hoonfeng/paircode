// Command reactive_min_repro 用 goja 引擎最小化复现 Vue 3 响应式链路：
// 模拟 reactive(target) + track/trigger（WeakMap targetMap + Map depsMap + Set dep），
// 验证 set 后 effect 是否被重新调用。
package main

import (
	"fmt"
	"log"

	"wb-ui/jsc"
)

const js = `
// ── 模拟 Vue 3 reactivity 核心 ──
var targetMap = new WeakMap();   // target -> depsMap
var activeEffect = null;

function track(target, key) {
	if (activeEffect == null) return;
	var depsMap = targetMap.get(target);
	if (!depsMap) { depsMap = new Map(); targetMap.set(target, depsMap); }
	var dep = depsMap.get(key);
	if (!dep) { dep = new Set(); depsMap.set(key, dep); }
	dep.add(activeEffect);
}

function trigger(target, key) {
	var depsMap = targetMap.get(target);
	if (!depsMap) return;
	var dep = depsMap.get(key);
	if (!dep) return;
	dep.forEach(function(effect){ effect(); });
}

function reactive(target) {
	var proxy = new Proxy(target, {
		get: function(t, key, receiver) {
			track(t, key);
			var v = Reflect.get(t, key, receiver);
			return v;
		},
		set: function(t, key, value, receiver) {
			var ok = Reflect.set(t, key, value, receiver);
			trigger(t, key);
			return ok;
		}
	});
	return proxy;
}

// ── 模拟 Vue 组件：渲染函数读取 state.currentConvId ──
var out = [];
var state = reactive({ currentConvId: '' });

function render() {
	activeEffect = render;
	var v = state.currentConvId;
	activeEffect = null;
	out.push('render sees currentConvId=' + v);
}

// 初次渲染（建立依赖）
render();
out.push('--- set currentConvId ---');
state.currentConvId = 'conv_123';
out.push('--- done ---');

// 若响应式正常，render 会被 trigger 再次调用
JSON.stringify(out);
`

func main() {
	log.SetFlags(0)
	in := jsc.NewInterpreter()
	in.SetupGlobal(nil)
	v, err := in.RunJS(js)
	if err != nil {
		log.Fatalf("RunJS error: %v", err)
	}
	fmt.Printf("RESULT: %s\n", v.ToString())
}
