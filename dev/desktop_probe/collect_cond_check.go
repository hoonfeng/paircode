// Command collect_cond_test 用 wb-ui jsc 引擎验证 Vue 3.5 收集条件在 goja 下的行为
package main

import (
	"fmt"

	"wb-ui/jsc"
)

const js = `
var out = [];
function check(name, fn) {
	try { out.push(name + '=' + fn()); } catch(e) { out.push(name + '=ERR:' + String(e)); }
}

// 1. 基础收集条件（comment: pf=0 sf=8 → 期望 false）
check('base', function(){
	var vnode = { patchFlag: 0, shapeFlag: 8 };
	var isBlockTreeEnabled = 1, currentBlock = [], isBlockNode = false;
	return (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock && (vnode.patchFlag > 0 || vnode.shapeFlag & 6) && vnode.patchFlag != -2);
});

// 2. 带 Symbol type 的 vnode 对象（孤儿 comment 的样子）
check('symbolType', function(){
	var Comment = Symbol('v-cmt');
	var vnode = { type: Comment, patchFlag: 0, shapeFlag: 8, children: 'x', el: undefined };
	var isBlockTreeEnabled = 1, currentBlock = [], isBlockNode = false;
	return (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock && (vnode.patchFlag > 0 || vnode.shapeFlag & 6) && vnode.patchFlag != -2);
});

// 3. 对象创建后加属性（shape 变化）再读 patchFlag
check('shapeChange', function(){
	var vnode = { type: 'div', patchFlag: 2, shapeFlag: 17 };
	vnode.el = null;            // 后加
	vnode.dynamicChildren = []; // 后加
	vnode.props = {};           // 后加
	var r1 = (vnode.patchFlag > 0 || vnode.shapeFlag & 6);
	var r2 = vnode.patchFlag != -2;
	return r1 + ',' + r2;  // 期望 true,true
});

// 4. shapeFlag & 6 的各种值（位运算）
check('bitop', function(){
	var out2 = [];
	for (var i = 0; i < 32; i++) {
		var r = i & 6;
		if (r !== 0) out2.push(i + '->' + r);
	}
	return out2.join(';') || 'none';
});

// 5. 数组 push 后对象引用
check('arrRef', function(){
	var arr = [];
	var a = { patchFlag: 1, shapeFlag: 1, el: undefined };
	arr.push(a);
	a.el = { tag: 'div' };  // 对象内改，数组元素应看到新值
	return (arr[0].el ? 'hasEl' : 'noEl');
});

// 6. 深层：模拟 openBlock/currentBlock 收集 + 块级 push
check('blockPush', function(){
	var currentBlock = null;
	var stack = [];
	function openBlock() { stack.push(currentBlock); currentBlock = []; return currentBlock; }
	function closeBlock() { var b = currentBlock; currentBlock = stack.pop(); return b; }
	var block = openBlock();
	var v1 = { type: 'div', patchFlag: 2, shapeFlag: 17 };  // 动态元素 → 应收集
	var v2 = { type: Symbol('v-cmt'), patchFlag: 0, shapeFlag: 8, children: 'c' };  // comment → 不应收集
	var isBlockTreeEnabled = 1, isBlockNode = false;
	if (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock && (v1.patchFlag > 0 || v1.shapeFlag & 6) && v1.patchFlag != -2) currentBlock.push(v1);
	if (isBlockTreeEnabled > 0 && !isBlockNode && currentBlock && (v2.patchFlag > 0 || v2.shapeFlag & 6) && v2.patchFlag != -2) currentBlock.push(v2);
	var b = closeBlock();
	return b.length + ':' + b.map(function(x){ return String(x.type); }).join(',');
});

// 7. hoisted 缓存：模块级 const 复用（同一对象）
var _hoisted_1 = { type: 'div', patchFlag: 0, shapeFlag: 9, children: 'static', el: null };
check('hoisted', function(){
	var used = [];
	for (var i = 0; i < 3; i++) {
		// 模拟 mountChildren: el===null → cloneIfMounted
		var child = (_hoisted_1.el === null) ? Object.assign({}, _hoisted_1) : _hoisted_1;
		child.el = { tag: 'div', i: i };
		used.push(child === _hoisted_1 ? 'same' : 'clone');
	}
	return used.join(',');
});

// 8. createBaseVNode 后 patchFlag 再被 -2 覆盖的序列
check('pfAfter', function(){
	var v = { patchFlag: 3, shapeFlag: 17 };
	var collected = (v.patchFlag > 0 || v.shapeFlag & 6);
	v.patchFlag = -2;  // 之后 BAIL
	return collected + ',pf=' + v.patchFlag;
});

console.log(out.join('\n'));
`

func main() {
	rt := jsc.NewInterpreter()
	logger := &jsc.BufferLogger{}
	rt.SetupGlobal(logger)
	_ = rt.GetEventLoop()
	_, err := rt.RunJS(js)
	if err != nil {
		fmt.Println("RunJS error:", err)
	}
	fmt.Println("--- console ---")
	fmt.Println(logger.String())
	fmt.Println("--- done ---")
}