// Command set_probe 验证 goja 的 Set/Map/WeakMap 基本语义：
// has/forEach/spread 迭代器——Vue 3 依赖收集用 Set 存 effect。
package main

import (
	"fmt"
	"log"

	"wb-ui/jsc"
)

const js = `
var out = {};

// 1. Set.has 对字符串
try {
	var s = new Set();
	s.add('conv_123');
	out.setStrHas = s.has('conv_123');          // 期望 true
	out.setStrHasNew = s.has('conv_456');       // 期望 false
} catch(e) { out.setStrErr = String(e); }

// 2. Set 迭代（forEach / for-of / spread）
try {
	var s2 = new Set();
	var fn1 = function(){ return 1; };
	var fn2 = function(){ return 2; };
	s2.add(fn1); s2.add(fn2);
	var viaForEach = [];
	s2.forEach(function(f){ viaForEach.push(f()); });
	out.setForEach = viaForEach.join(',');
	var viaSpread = [];
	var arr = Array.from(s2);
	for (var i=0;i<arr.length;i++) viaSpread.push(arr[i]());
	out.setArrayFrom = viaSpread.join(',');
	out.setSize = s2.size;
} catch(e) { out.setIterErr = String(e); }

// 3. Map 基本语义
try {
	var m = new Map();
	var t = { a: 1 };
	m.set(t, 'dep1');
	out.mapGetSameRef = m.get(t);         // 期望 dep1
	out.mapHasSameRef = m.has(t);
	var t2 = { a: 1 };
	out.mapGetDiffRef = m.get(t2);        // 期望 undefined（引用语义）
} catch(e) { out.mapErr = String(e); }

// 4. WeakMap 引用语义
try {
	var wm = new WeakMap();
	var k1 = {};
	wm.set(k1, 'v1');
	out.wmGet = wm.get(k1);               // 期望 v1
	var k2 = {};
	out.wmGetDiff = wm.get(k2);           // 期望 undefined
	out.wmHas = wm.has(k1);
} catch(e) { out.wmErr = String(e); }

// 5. 字符串 Set 的 forEach 迭代（模拟 Vue dep 存 effect 的场景）
try {
	var dep = new Set();
	var calls = [];
	dep.add(function(){ calls.push('A'); });
	dep.add(function(){ calls.push('B'); });
	dep.forEach(function(effect){ effect(); });
	out.depCalls = calls.join(',');
} catch(e) { out.depErr = String(e); }

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
