// Command reactive_min_repro2 精确模拟 Vue 3 baseHandlers.set 的 toRaw(receiver) 检查：
//   set(target, key, value, receiver) {
//     const oldValue = target[key]
//     const hadKey = hasOwn(target, key)
//     const result = Reflect.set(target, key, value, receiver)
//     if (target === toRaw(receiver)) { trigger(...) }  // ← goja 的 receiver 传递是否正常？
//   }
package main

import (
	"fmt"
	"log"

	"wb-ui/jsc"
)

const js = `
// ── 精确模拟 Vue 3 reactivity 核心（含 toRaw 检查）──
var targetMap = new WeakMap();
var activeEffect = null;
var RAW = '__v_raw';

function toRaw(observed) {
	var raw = observed && observed[RAW];
	return raw ? raw : observed;
}

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
			if (key === RAW) return t;
			track(t, key);
			var v = Reflect.get(t, key, receiver);
			return v;
		},
		set: function(t, key, value, receiver) {
			var oldValue = t[key];
			var hadKey = Object.prototype.hasOwnProperty.call(t, key);
			var result = Reflect.set(t, key, value, receiver);
			// ★ Vue 3 的关键检查：target === toRaw(receiver)
			var receiverOK = (t === toRaw(receiver));
			__receiverOK = receiverOK;
			if (receiverOK) {
				if (!hadKey) trigger(t, key);
				else if (value !== oldValue) trigger(t, key);
			}
			return result;
		}
	});
	return proxy;
}

var out = [];
var state = reactive({ currentConvId: '' });

function render() {
	activeEffect = render;
	var v = state.currentConvId;
	activeEffect = null;
	out.push('render sees currentConvId=' + v);
}

render();
out.push('--- set currentConvId ---');
state.currentConvId = 'conv_123';
out.push('--- done, receiverOK=' + __receiverOK + ' ---');
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
