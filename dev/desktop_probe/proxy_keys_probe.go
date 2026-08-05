// proxy_keys_probe 验证 jsc 引擎的 Proxy ownKeys / for-in / Object.keys 行为
//（Vue 3 的 setupState 是 proxyRefs 返回的 Proxy，组件渲染/依赖收集依赖它）
package main

import (
	"fmt"
	"log"

	"wb-ui/jsc"
)

func main() {
	log.SetFlags(0)
	in := jsc.NewInterpreter()
	in.SetupGlobal(nil)
	v, err := in.RunJS(`(function(){
		var out = {};
		var target = { a: 1, b: 2 };
		var p = new Proxy(target, {
			get: function(t, k, r) { return Reflect.get(t, k, r); },
			ownKeys: function(t) { return Reflect.ownKeys(t); },
			getOwnPropertyDescriptor: function(t, k) { return Object.getOwnPropertyDescriptor(t, k); },
		});
		out.keys = Object.keys(p).join(',');
		var fi = [];
		for (var k in p) fi.push(k);
		out.forin = fi.join(',');
		out.hasA = ('a' in p);
		out.getA = p.a;
		out.getB = p.b;
		// 模拟 proxyRefs（Vue 3）：get trap 解 ref
		var refLike = { __v_isRef: true, value: 'refval' };
		var state2 = { currentConvId: 'conv_A', obj: refLike };
		var pr = new Proxy(state2, {
			get: function(t, k, r) {
				var v = Reflect.get(t, k, r);
				return v && v.__v_isRef ? v.value : v;
			},
			ownKeys: function(t) { return Reflect.ownKeys(t); },
		});
		out.prKeys = Object.keys(pr).join(',');
		var prfi = [];
		for (var k in pr) prfi.push(k);
		out.prForin = prfi.join(',');
		out.prGet = pr.currentConvId;
		return JSON.stringify(out);
	})()`)
	if err != nil {
		log.Fatalf("RunJS error: %v", err)
	}
	fmt.Printf("RESULT: %s\n", v.ToString())
}
