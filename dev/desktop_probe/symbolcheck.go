// Command symbolcheck verifies ES6 Symbol support in the wb-ui jsc engine,
// which Vue 3's reactive implementation depends on (ReactiveFlags.IS_REACTIVE
// / RAW use Symbols as internal keys).
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
var out = [];
// Symbol 基本支持
try {
  var s1 = Symbol('a');
  var s2 = Symbol('b');
  out.push('typeof=' + typeof s1 + ' desc=' + String(s1) + ' eq=' + (s1 === s2) + ' same=' + (s1 === s1));
} catch(e) { out.push('Symbol-ERR: ' + e.message); }

// Symbol 作为对象属性键（Vue ReactiveFlags 用法）
try {
  var obj = {};
  var k1 = Symbol('IS_REACTIVE');
  obj[k1] = true;
  out.push('symKey=' + obj[k1] + ' has=' + (k1 in obj) + ' keys=' + Object.keys(obj).length + ' symKeys=' + Object.getOwnPropertySymbols(obj).length);
} catch(e) { out.push('symKey-ERR: ' + e.message); }

// WeakMap + Symbol 组合（Vue targetMap 用法）
try {
  var wm = new WeakMap();
  var target = { x: 1 };
  var flag = Symbol('RAW');
  wm.set(target, flag);
  out.push('weakmap=' + (wm.get(target) === flag) + ' has=' + wm.has(target));
} catch(e) { out.push('weakmap-ERR: ' + e.message); }

// 完整模拟 Vue 3 reactive：Symbol 内部键 + WeakMap + Reflect + 惰性代理
try {
  var targetMap = new WeakMap();
  var IS_REACTIVE = Symbol('IS_REACTIVE');
  var RAW = Symbol('RAW');
  var activeEffect = null;
  function track(t, k) {
    if (!activeEffect) return;
    var dm = targetMap.get(t);
    if (!dm) { dm = new Map(); targetMap.set(t, dm); }
    var dep = dm.get(k);
    if (!dep) { dep = new Set(); dm.set(k, dep); }
    if (!dep.has(activeEffect)) dep.add(activeEffect);
  }
  function trigger(t, k) {
    var dm = targetMap.get(t);
    if (!dm) return;
    var dep = dm.get(k);
    if (!dep) return;
    var es = Array.from(dep);
    for (var i=0;i<es.length;i++) es[i]();
  }
  function reactive(target) {
    var proxy = new Proxy(target, {
      get(t, k, r) {
        if (k === RAW) return t;
        if (k === IS_REACTIVE) return true;
        var res = Reflect.get(t, k, r);
        track(t, k);
        return res;
      },
      set(t, k, v, r) {
        var old = t[k];
        var ok = Reflect.set(t, k, v, r);
        if (old !== v) trigger(t, k);
        return ok;
      }
    });
    return proxy;
  }
  function effect(fn) {
    var w = function(){ var p=activeEffect; activeEffect=w; try{fn();}finally{activeEffect=p;} };
    w();
    return w;
  }
  var state = reactive({ currentConvId: 'A' });
  var isRx = state[IS_REACTIVE];
  var raw = state[RAW];
  out.push('isReactive=' + isRx + ' rawSame=' + (raw === raw) + ' rawType=' + typeof raw);
  var renders = [];
  effect(function(){ renders.push('render:' + state.currentConvId); });
  state.currentConvId = 'B';
  out.push('renders=' + JSON.stringify(renders) + ' cur=' + state.currentConvId);
} catch(e) { out.push('reactive-ERR: ' + e.message); }

console.log(out.join('\n'));
`
	_, err := rt.RunJS(script)
	if err != nil {
		fmt.Println("RunJS error:", err)
	}
	fmt.Println("--- console ---")
	fmt.Println(logger.String())
	fmt.Println("--- done ---")
}
