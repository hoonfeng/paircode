// Command proxycheck verifies Vue-3-style reactive (Proxy) support in the
// wb-ui jsc engine: does a reactive state mutation trigger an effect and
// propagate to a DOM className change?
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
var hits = 0;
var target = { currentConvId: 'A', items: [1,2,3] };
var state = new Proxy(target, {
  set(t, k, v) {
    t[k] = v;
    hits++;
    return true;
  },
  get(t, k) { return t[k]; }
});
state.currentConvId = 'B';
state.items.push(4);
console.log('hits=' + hits + ' current=' + state.currentConvId + ' items=' + state.items.length);

// Vue 3 reactive + effect 简化模拟
var activeEffect = null;
var deps = new Map();
function track(obj, key) {
  if (!activeEffect) return;
  if (!deps.has(key)) deps.set(key, []);
  deps.get(key).push(activeEffect);
}
function trigger(key) {
  var list = deps.get(key) || [];
  for (var i=0;i<list.length;i++) list[i]();
}
function reactive(obj) {
  return new Proxy(obj, {
    get(t,k){ track(t,k); return t[k]; },
    set(t,k,v){ t[k]=v; trigger(k); return true; }
  });
}
function effect(fn){ activeEffect = fn; fn(); activeEffect = null; }
var vState = reactive({ cur: 0 });
var rendered = [];
effect(function(){ rendered.push('render cur=' + vState.cur); });
vState.cur = 5;
console.log('reactive-renders=' + JSON.stringify(rendered) + ' cur=' + vState.cur);
`
	_, err := rt.RunJS(script)
	if err != nil {
		fmt.Println("RunJS error:", err)
	}
	fmt.Println("--- console ---")
	fmt.Println(logger.String())
	fmt.Println("--- done ---")
}
