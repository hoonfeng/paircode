// Command conv_click_probe reproduces the "conversation list click does not
// select" issue: it loads the real dist, finds .conv-item geometry, then
// simulates host.handleClick's exact flow:
//
//	el := HitTest(rv, x, y, "onclick")        // Vue @click is NOT an onclick attr
//	if el == nil { deepest := HitTest(rv, x, y, ""); deepest.DispatchEvent(click) }
//
// and finally inspects whether the Vue @click handler ran (conv-item.active
// class toggled / currentConvId changed).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/dom"
	"wb-ui/jsc"
	"wb-ui/layout"
	"wb-ui/platform/graphics"
	"wb-ui/rendering"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/desktopbridge"
)

func main() {
	log.SetFlags(log.Ltime)
	wd, _ := os.Getwd()
	distDir := filepath.Join(wd, "cmd", "companion", "web-ui", "dist")
	absDist, _ := filepath.Abs(distDir)
	htmlData, err := os.ReadFile(filepath.Join(absDist, "index.html"))
	if err != nil {
		log.Fatalf("read index.html: %v", err)
	}
	if graphics.GetFontManager() == nil {
		_ = graphics.InitFontManager("")
		if mgr := graphics.GetFontManager(); mgr != nil {
			mgr.LoadSystemFonts()
		}
	}
	layout.MeasureTextFunc = func(family string, size float64, weight int, style2, text string) float64 {
		return graphics.MeasureText(graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}, text)
	}
	layout.FontMetricsFunc = func(family string, size float64, weight int, style2 string) (float64, float64, float64) {
		f := graphics.Font{Family: family, Size: size, Weight: weight, Style: style2}
		return graphics.GlobalFontAscent(f), graphics.GlobalFontDescent(f), graphics.GlobalFontLineGap(f)
	}
	wv := webkit.NewWebView()
	setupLoaders(wv, distDir)
	_ = wv.JSInterpreter()
	desktopbridge.Init(wv)
	// ★ 注入 hook：在 app.js 执行后、微任务 flush 前 hook RightPanel.update，捕获 patch 前的 prevTree
	hookJS := `<script>
(function(){
	try {
		var root = document.querySelector('#app');
		if (!root || !root._vnode) { window.__hookLog = [{ev:'no-root'}]; return; }
		function findRp(v, d) {
			if (!v || d > 14) return null;
			if (v.component) {
				var c = v.component;
				var nm = (c.type && (c.type.name || c.type.__name)) || '';
				if (nm === 'RightPanel') return c;
			}
			var subs = [];
			if (v.component && v.component.subTree) subs.push(v.component.subTree);
			if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
			if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
			for (var k=0;k<subs.length;k++) { var r = findRp(subs[k], d+1); if (r) return r; }
			return null;
		}
		var rp = findRp(root._vnode, 0);
		if (!rp) { window.__hookLog = [{ev:'no-rp'}]; return; }
		window.__hookRp = rp;
		var orig = rp.update;
		// ★ block 隔离检查：Root 与 RightPanel 的 dynamicChildren 是否同一数组？
		var blockCheck = {};
		try {
			var rootComp = root._vnode.component;
			var rSub = rootComp && rootComp.subTree;
			var rpSub = rp.subTree;
			blockCheck.rootSubType = rSub && typeof rSub.type === 'string' ? rSub.type : String(rSub && rSub.type);
			blockCheck.rootDynLen = rSub && rSub.dynamicChildren ? rSub.dynamicChildren.length : -1;
			blockCheck.rpDynLen = rpSub && rpSub.dynamicChildren ? rpSub.dynamicChildren.length : -1;
			blockCheck.sameArray = !!(rSub && rpSub && rSub.dynamicChildren === rpSub.dynamicChildren);
			blockCheck.rootDynTypes = (rSub && rSub.dynamicChildren ? rSub.dynamicChildren : []).slice(0, 12).map(function(d){ var t = d && d.type; return typeof t === 'string' ? t : ((t && (t.__name || t.name)) || String(t)); });
			blockCheck.rpDynTypes = (rpSub && rpSub.dynamicChildren ? rpSub.dynamicChildren : []).slice(0, 12).map(function(d){ var t = d && d.type; return typeof t === 'string' ? t : ((t && (t.__name || t.name)) || String(t)); });
		} catch(e) { blockCheck.err = String(e).slice(0, 150); }
		window.__hookLog = [{
			ev: 'hooked',
			compName: rp.type && (rp.type.name || rp.type.__name),
			subEl: !!rp.subTree.el,
			subDyn: (rp.subTree.dynamicChildren || []).length,
			dynEls: (rp.subTree.dynamicChildren || []).slice(0, 40).map(function(d){ return !!(d && d.el); }),
			childrenEls: Array.isArray(rp.subTree.children) ? rp.subTree.children.map(function(c){ return !!(c && c.el); }) : [],
			blockCheck: blockCheck
		}];
		window.__hookErr = [];
		var oe = window.onerror || function(){};
		window.onerror = function(msg, src, ln, col, err){
			try { window.__hookErr.push(String(msg).slice(0, 200) + ' @' + ln); } catch(e){}
			try { return oe.apply(this, arguments); } catch(e){}
		};
		rp.update = function() {
			var self = this;
			try {
				var pt = self.subTree;
				var arr = pt && pt.dynamicChildren ? pt.dynamicChildren : [];
				window.__hookLog.push({
					ev: 'update',
					prevSubEl: !!(pt && pt.el),
					prevDynLen: arr.length,
					prevDynEls: arr.slice(0, 40).map(function(d){ return !!(d && d.el); }),
					prevDynTypes: arr.slice(0, 40).map(function(d){ var t = d && d.type; return typeof t === 'string' ? t : ((t && (t.__name || t.name)) || String(t)); }),
					prevChildrenEls: (pt && Array.isArray(pt.children)) ? pt.children.map(function(c){ return !!(c && c.el); }) : []
				});
				// ★ 扫描 prevTree 整棵树：找 el 为 undefined/null 的 element vnode（patch 失败点）
				if (pt) {
					var badEls = [];
					(function scan(n, path){
						if (!n || typeof n !== 'object') return;
						if (typeof n.type === 'string') {
							var e = n.el;
							if (e === undefined || e === null) {
								badEls.push({ path: path, t: n.type, pf: n.patchFlag, elKind: e === undefined ? 'undef' : 'null' });
							}
						}
						if (n.component && n.component.subTree) scan(n.component.subTree, path + '>c');
						if (Array.isArray(n.children)) for (var i = 0; i < n.children.length; i++) scan(n.children[i], path + '[' + i + ']');
					})(pt, 'root');
					window.__hookLog[window.__hookLog.length - 1].badEls = badEls.slice(0, 20);
					window.__hookLog[window.__hookLog.length - 1].badElCount = badEls.length;
				}
				// ★★★ 第一次 update 的 n1/n2 dynamicChildren 对齐性检查：新 render 产物与旧树对比
				try {
					var nt2 = self.render(self.proxy, self.renderCache);
					var ndyn = nt2 && nt2.dynamicChildren ? nt2.dynamicChildren : [];
					var diff = [];
					var n = Math.min(arr.length, ndyn.length);
					for (var k2 = 0; k2 < n; k2++) {
						var a = arr[k2], b = ndyn[k2];
						var ta = a ? (typeof a.type === 'string' ? a.type : ((a.type && (a.type.__name || a.type.name)) || String(a.type))) : '?';
						var tb = b ? (typeof b.type === 'string' ? b.type : ((b.type && (b.type.__name || b.type.name)) || String(b.type))) : '?';
						if (ta !== tb) diff.push({ i: k2, old: ta, neu: tb, oldEl: a && !!a.el, newEl: b && !!b.el });
					}
					window.__hookLog[window.__hookLog.length - 1].dynAlign = {
						oldLen: arr.length,
						newLen: ndyn.length,
						mismatches: diff.slice(0, 12),
						firstMismatch: diff.length ? diff[0] : null
					};
				} catch(e) { window.__hookLog[window.__hookLog.length - 1].alignErr = String(e).slice(0,150); }
				// ★★★★ 孤儿字段 dump：prevDyn[7..] 的 shapeFlag/patchFlag + children 树同型节点
				try {
					var orphInfo = [];
					var treeTypes = {};
					(function walk(n, d){
						if (!n || typeof n !== 'object' || d > 22) return;
						var t = typeof n.type === 'string' ? n.type : String(n.type);
						if (!treeTypes[t]) treeTypes[t] = 0;
						treeTypes[t]++;
						if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], d+1);
						if (n.component && n.component.subTree) walk(n.component.subTree, d+1);
					})(pt, 0);
					for (var j2 = 7; j2 < Math.min(arr.length, 33); j2++) {
						var d2 = arr[j2];
						if (!d2) continue;
						var t2 = typeof d2.type === 'string' ? d2.type : String(d2.type);
						orphInfo.push({
							i: j2,
							t: t2.slice(0, 14),
							pf: d2.patchFlag,
							sf: d2.shapeFlag,
							key: d2.key === undefined || d2.key === null ? '' : String(d2.key),
							el: d2.el === null ? 'null' : (d2.el === undefined ? 'undef' : 'obj'),
							treeHasType: treeTypes[t2] || 0,
							ctx: (d2.ctx && d2.ctx.type && (d2.ctx.type.name || d2.ctx.type.__name)) || String(d2.ctx && d2.ctx.type).slice(0,10) || '?',
							props: d2.props ? Object.keys(d2.props).slice(0, 6).join(',') : '',
							comp: !!(d2.component),
							subOf: (d2.parent && d2.parent.type && (d2.parent.type.name || typeof d2.parent.type === 'string' ? d2.parent.type : '?')) || ''
						});
					}
					window.__hookLog[window.__hookLog.length - 1].orphFields = orphInfo;
				} catch(e) { window.__hookLog[window.__hookLog.length - 1].orphErr = String(e).slice(0,150); }
			} catch(e) { window.__hookLog.push({ev:'snapErr', err: String(e).slice(0,150)}); }
			var r;
			try { r = orig.apply(self, arguments); }
			catch(e) {
				var info = {ev:'updateErr', err: String(e).slice(0,200)};
				try { info.stack = String(e.stack).slice(0, 800); } catch(e2){}
				window.__hookLog.push(info);
				return undefined;
			}
			// ★ update 后树状态：subTree 是否被替换？dynamicChildren 是否更新？
			try {
				var nt = self.subTree;
				var nArr = nt && nt.dynamicChildren ? nt.dynamicChildren : [];
				window.__hookLog.push({
					ev: 'afterUpdate',
					subEl: !!(nt && nt.el),
					subChildrenLen: nt && Array.isArray(nt.children) ? nt.children.length : -1,
					newDynLen: nArr.length,
					newDynEls: nArr.slice(0, 40).map(function(d){ return !!(d && d.el); }),
					badElCount: (function(){
						if (!nt) return -1;
						var c = 0;
						(function scan(n, d){
							if (!n || typeof n !== 'object' || d > 25) return;
							if (typeof n.type === 'string' && (n.el === undefined || n.el === null)) c++;
							if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) scan(n.children[i], d+1);
						})(nt, 0);
						return c;
					})()
				});
			} catch(e) { window.__hookLog.push({ev:'afterSnapErr', err: String(e).slice(0,150)}); }
			return r;
		};
	} catch(e) { window.__hookLog = [{ev:'outerErr', err: String(e).slice(0,150)}]; }
})();
</script>`
	htmlStr := string(htmlData)
	if !strings.Contains(htmlStr, "__hookLog") {
		// ★ pre-hook：注入到 module script 之前，拦截微任务序列，抓 mount 后第一次 update 前的树状态
		preHookJS := `<script>
(function(){
	try {
		var origThen = Promise.prototype.then;
		window.__mtLog = [];
		window.__errLog = [];
		// ★ 拦截 console.error，抓 Vue 内部错误（default errorHandler 输出到 console）
		if (window.console && console.error) {
			var oe = console.error.bind(console);
			console.error = function() {
				try {
					var args2 = Array.prototype.slice.call(arguments);
					var msg2 = args2.map(String).join(' ').slice(0, 300);
					var stk = '';
					try { var er = args2[0]; if (er && er.stack) stk = String(er.stack).slice(0, 700); } catch(e2){}
					window.__errLog.push({ msg: msg2, at: window.__mtLog.length, stack: stk });
				} catch(e){}
				return oe.apply(console, arguments);
			};
		}
		window.__errLog.push('console hooked');
		var snapCount = 0;
		// ★ 无污染 pidOf：用外部 Map，绝不往 Vue 对象上写属性（写属性可能触发 goja 对象结构变化）
		window.__pidSeq = window.__pidSeq || 0;
		window.__pidMap = window.__pidMap || new Map();
		function pidOf(v) {
			if (!v || typeof v !== 'object') return -1;
			if (window.__pidMap.has(v)) return window.__pidMap.get(v);
			var p = ++window.__pidSeq;
			window.__pidMap.set(v, p);
			return p;
		}
		// ★ subTree 引用追踪：保存每个 subTree 对象的存活状态
		window.__subRefs = window.__subRefs || [];
		function trackSub(st, tag) {
			try {
				if (!st) return;
				var dyn = st.dynamicChildren;
				window.__subRefs.push({
					pid: pidOf(st),
					tag: tag,
					dynArrPid: dyn ? pidOf(dyn) : -1,
					el: st.el === undefined ? 'u' : (st.el === null ? 'n' : 'o'),
					dynLen: dyn ? dyn.length : -1,
					dynEls: dyn ? dyn.slice(0, 10).map(function(d){ return !!(d && d.el) ? '1' : '0'; }).join('') : '?'
				});
				if (window.__subRefs.length > 40) window.__subRefs.shift();
			} catch(e) {}
		}
		// ★ 无污染 wrapped 标记
		window.__wrappedSet = window.__wrappedSet || new Map();
		function elKind(el) { if (el === undefined) return 'u'; if (el === null) return 'n'; return 'o'; }
		function readRpState() {
			try {
				var rpi = window.__findRp(document.querySelector('#app')._vnode, 0);
				if (!rpi) return 'no-rp';
				var st = rpi.subTree;
				function tn(v) { if (!v) return 'nil'; var t = v.type; return typeof t === 'string' ? t : ((t && (t.__name || t.name)) || String(t)).slice(0, 18); }
				var ch = st && Array.isArray(st.children) ? st.children : [];
				return 'pid=' + pidOf(rpi) + ' sub=' + pidOf(st) + ' subT=' + tn(st) + ' subPF=' + st.patchFlag + ' subEl=' + elKind(st && st.el) + ' chN=' + ch.length + ' ch0t=' + tn(ch[0]) + ' ch1t=' + tn(ch[1]) + ' ch=' + ch.map(function(c){ return elKind(c && c.el); }).join('') + ' chp=' + ch.map(function(c){ return pidOf(c); }).join(',').slice(0,20) + ' dyn0-12=' + (st && st.dynamicChildren ? st.dynamicChildren.slice(0, 12).map(function(d){ return !!(d && d.el) ? '1' : '0'; }).join('') : '?');
			} catch(e) { return 'err:' + String(e).slice(0, 60); }
		}
		function snapRoot(tag) {
			window.__findRp = window.__findRp || function(v, d) {
				if (!v || d > 18) return null;
				if (v.component) {
					var c = v.component;
					var nm = (c.type && (c.type.name || c.type.__name)) || '';
					if (nm === 'RightPanel') return c;
				}
				var subs = [];
				if (v.component && v.component.subTree) subs.push(v.component.subTree);
				if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
				if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
				for (var k=0;k<subs.length;k++) { var r = window.__findRp(subs[k], d+1); if (r) return r; }
				return null;
			};
			// ★ 找树里所有 RightPanel 实例（区分是否有多个实例）
			window.__findRpAll = window.__findRpAll || function(v, d) {
				var out = [];
				(function walk(v, d) {
					if (!v || d > 22) return;
					if (v.component) {
						var nm = (v.component.type && (v.component.type.name || v.component.type.__name)) || '';
						if (nm === 'RightPanel') out.push(v.component);
					}
					if (v.component && v.component.subTree) walk(v.component.subTree, d + 1);
					if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) walk(v.children[i], d + 1);
					if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) walk(v.dynamicChildren[j], d + 1);
				})(v, d);
				return out;
			};
			try {
				var root = document.querySelector('#app');
				if (!root || !root._vnode) { window.__mtLog.push({tag:tag, st:'no-vnode'}); return; }
				var rp = window.__findRp(root._vnode, 0);
				if (!rp) { window.__mtLog.push({tag:tag, st:'no-rp'}); return; }
				if (snapCount >= 300) return;
				snapCount++;
				// ★ 完整树扫描：el 为 null 的 element vnode 计数（与 hookJS 的 badEls 扫描一致）
				var badEls = [];
				(function scan(n, path) {
					if (!n || typeof n !== 'object') return;
					if (typeof n.type === 'string') {
						var e = n.el;
						if (e === undefined || e === null) badEls.push({ path: path, t: n.type, pf: n.patchFlag, k: e === undefined ? 'u' : 'n' });
					}
					if (n.component && n.component.subTree) scan(n.component.subTree, path + '>c');
					if (Array.isArray(n.children)) for (var i = 0; i < n.children.length; i++) scan(n.children[i], path + '[' + i + ']');
				})(rp.subTree, 'root');
				// ★ fail-fast：树首次变坏时记录完整现场（微任务名 + 栈 + 坏节点）
				if (badEls.length > 0 && !window.__firstBad) {
					window.__firstBad = {
						tag: tag,
						count: badEls.length,
						sample: badEls.slice(0, 12),
						stack: String(new Error('firstBad@' + tag).stack).slice(0, 900),
						childEls: Array.isArray(rp.subTree.children) ? rp.subTree.children.map(function(c){ return !!(c && c.el); }) : []
					};
				}
				// ★ 树里所有 RightPanel 实例的 pid + 是否已包装
				var allPids = [];
				var wrappedFlags = [];
				var allInsts = window.__findRpAll(root._vnode, 0);
				for (var ai = 0; ai < allInsts.length; ai++) {
					allPids.push(pidOf(allInsts[ai]));
					wrappedFlags.push(!!(allInsts[ai].update && window.__wrappedSet.get(allInsts[ai])));
				}
				window.__mtLog.push({
					tag: tag,
					rpPid: pidOf(rp),
					rpWrapped: !!(rp.update && window.__wrappedSet.get(rp)),
					rpUpdateName: (rp.update && rp.update.name) || String(rp.update).slice(0, 40),
					allRpPids: allPids.join(','),
					allRpWrapped: wrappedFlags.join(','),
					subPid: pidOf(rp.subTree),
					subEl: !!rp.subTree.el,
					dynLen: (rp.subTree.dynamicChildren || []).length,
					dynEls: (rp.subTree.dynamicChildren || []).slice(0, 40).map(function(d){ return !!(d && d.el); }),
					childEls: Array.isArray(rp.subTree.children) ? rp.subTree.children.map(function(c){ return !!(c && c.el); }) : [],
					chKinds: Array.isArray(rp.subTree.children) ? rp.subTree.children.map(function(c){ return elKind(c && c.el); }) : [],
					badElCount: badEls.length,
					badElSample: badEls.slice(0, 6).map(function(b){ return b.path + ':' + b.t + ':' + b.k; }),
					errN: (window.__errLog && window.__errLog.length) || 0,
					// ★ subTree.el 的 DOM 实际结构
					subElDom: (function(){
						var se = rp.subTree.el;
						var pr = 't=' + typeof se;
						if (se && typeof se === 'object') {
							pr += ' nt=' + se.nodeType;
							pr += ' childNodes=' + (se.childNodes ? se.childNodes.length : '?');
							pr += ' inDoc=' + (se.ownerDocument && se.ownerDocument.contains ? se.ownerDocument.contains(se) : '?');
							pr += ' fc=' + (se.firstChild ? (se.firstChild.tagName || se.firstChild.nodeName || '?') : 'none');
							pr += ' parentTag=' + (se.parentNode ? (se.parentNode.tagName || se.parentNode.nodeName || '?') : 'none');
						} else { pr += ' v=' + String(se); }
						return pr;
					})(),
					// ★ rp.subTree.dynamicChildren 前 12 个 el 状态（锁定坏树时刻）
					rpDyn: (function(){
						try {
							var rpd = rp.subTree && rp.subTree.dynamicChildren;
							if (!rpd) return 'none';
							var s = '';
							for (var i = 0; i < 12 && i < rpd.length; i++) { s += (rpd[i] && rpd[i].el) ? '1' : '0'; }
							return s + '/len' + rpd.length;
						} catch(e) { return 'err'; }
					})(),
					// ★ 树中 vnode 对象重复引用检测（共享 vnode → el 污染根源）
					sharedCheck: (function(){
						try {
							var cnt = {};
							var dups = [];
							(function walk(n, path){
								if (!n || typeof n !== 'object' || !('type' in n) || !('el' in n)) return;
								var key;
								if (!window.__idSeq) window.__idSeq = 1;
								try { key = 'o' + (window.__idSeq++); } catch(e) { return; }
								cnt[key] = n;
								var found = false;
								for (var k in cnt) { if (k !== key && cnt[k] === n) { found = true; break; } }
								if (found) {
									dups.push(path + ':' + (typeof n.type === 'string' ? n.type : 'x') + ':pf' + n.patchFlag + ':el' + (n.el ? 'y' : 'n'));
									if (dups.length >= 3) throw 'enough';
								}
								if (n.component && n.component.subTree) walk(n.component.subTree, path + '>c');
								if (Array.isArray(n.children)) for (var i = 0; i < n.children.length; i++) walk(n.children[i], path + '[' + i + ']');
							})(rp.subTree, 'root');
							return dups.length ? dups.join(' | ') : 'no-dup';
						} catch(e) { return e === 'enough' ? 'many' : 'err'; }
					})()
				});
			} catch(e) { window.__mtLog.push({tag:tag, err:String(e).slice(0,100)}); }
		}
		// ★ 包装所有组件实例的 update，记录组件更新顺序（定位 flushJobs 内部先执行哪个组件）
		window.__wrapAllUpdates = function() {
			try {
				var root = document.querySelector('#app');
				if (!root || !root._vnode) return;
				var insts = [];
				(function collect(v, d) {
					if (!v || d > 24) return;
					if (v.component) insts.push(v.component);
					if (v.component && v.component.subTree) collect(v.component.subTree, d + 1);
					if (Array.isArray(v.children)) for (var i = 0; i < v.children.length; i++) collect(v.children[i], d + 1);
				})(root._vnode, 0);
				insts.forEach(function(inst) {
					if (inst.update && !window.__wrappedSet.get(inst)) {
						window.__wrappedSet.set(inst, true);
						var u0 = inst.update;
						inst.update = function() {
							var nm = (inst.type && (inst.type.name || inst.type.__name)) || 'anon';
							var before = readRpState();
							// ★ 记录 prev subTree children 引用 + el（判断 update 后是否复用）
							var prevSt = inst.subTree;
							trackSub(prevSt, 'before:' + nm);
							var prevCh = prevSt && Array.isArray(prevSt.children) ? prevSt.children.slice(0, 6) : [];
							var prevChEl = prevCh.map(function(c){ return !!(c && c.el); });
							var prevChDyn = prevCh.map(function(c){ return c && c.dynamicChildren ? c.dynamicChildren.length : 0; });
							var stk = '';
							try { stk = String(new Error('upd-' + nm).stack).slice(0, 350); } catch(e2){}
							var r;
							try { r = u0.apply(this, arguments); }
							catch(e) {
								try { window.__mtLog.push({ tag: 'UPDATE-ERR:' + nm + ':' + (window.__mtLog.length), ipid: pidOf(inst), msg: String(e).slice(0, 200), stack: String(e.stack).slice(0, 400) }); } catch(e3){}
								throw e;
							}
							var after = readRpState();
							trackSub(inst.subTree, 'after:' + nm);
							// ★ 对比 prev/next children：对象身份 + el + 是否被 patch
							var newCh = inst.subTree && Array.isArray(inst.subTree.children) ? inst.subTree.children.slice(0, 6) : [];
							var cmp = [];
							for (var ci = 0; ci < Math.max(prevCh.length, newCh.length); ci++) {
								var pc = prevCh[ci], nc = newCh[ci];
								cmp.push({ i: ci, same: pc === nc, prevEl: !!(pc && pc.el), newEl: !!(nc && nc.el), prevT: pc ? (typeof pc.type === 'string' ? pc.type : (pc.type && pc.type.__name) || '?') : '-', newT: nc ? (typeof nc.type === 'string' ? nc.type : (nc.type && nc.type.__name) || '?') : '-' });
							}
							window.__mtLog.push({ tag: 'UPDATE:' + nm + ':' + (window.__mtLog.length), ipid: pidOf(inst), before: before, after: after, chSame: cmp, stk: stk });
							// ★ 更新后立即快照（精确定位 subTree 替换时刻）
							try { snapRoot('post-upd:' + nm); } catch(e4){}
							return r;
						};
						window.__wrappedSet.set(inst.update, true);
					}
				});
			} catch(e) { try { window.__mtLog.push({tag:'WRAP-ERR:'+String(e).slice(0,120)}); } catch(e2){} }
		};
		Promise.prototype.then = function(cb, eb) {
			var self = this;
			if (typeof cb !== 'function') {
				// ★ 无参 then 调用：原样转发（标准语义：不执行任何回调），记录来源 stack
				try {
					if (window.__noCbCount === undefined) window.__noCbCount = 0;
					if (window.__noCbCount < 20) {
						window.__noCbCount++;
						var nst = new Error('nocb').stack || '';
						window.__mtLog.push({ tag: 'THEN-NO-CB:' + (window.__mtLog.length), stack: String(nst).slice(0, 400) });
					}
				} catch(e) {}
				return origThen.call(self, cb, eb);
			}
			return origThen.call(self, function() {
				window.__wrapAllUpdates();
				snapRoot('pre:' + (cb && cb.name) + ':' + (window.__mtLog.length));
				var r;
				try {
					r = cb.apply(this, arguments);
				} catch(e) {
					try { window.__mtLog.push({tag:'MT-ERR', msg:String(e).slice(0,300), stack:String(e.stack).slice(0,600)}); } catch(e2){}
					throw e;
				}
				snapRoot('post:' + (cb && cb.name) + ':' + (window.__mtLog.length));
				return r;
			}, eb);
		};
		// 同步代码执行后也拍一次（module script 完成后）
		setTimeout(function(){ snapRoot('sync-later'); }, 5);
	} catch(e) { window.__mtLog = [{err: String(e).slice(0,100)}]; }
})();
</script>`
		htmlStr = strings.Replace(htmlStr, "<script type=\"module\"", preHookJS+"<script type=\"module\"", 1)
		htmlStr = strings.Replace(htmlStr, "</body>", hookJS+"</body>", 1)
	}
	wv.LoadHTML(htmlStr)
	// ★ 初始快照：mount 后立即检查 RightPanel.subTree 完整性 + 首次 update 测试
	if wv.JSInterpreter() != nil {
		is, _ := wv.JSInterpreter().RunJS(`(function(){
			var root = document.querySelector('#app');
			if (!root || !root._vnode) return {err:'no vnode'};
			function findRp(v, d) {
				if (!v || d > 14) return null;
				if (v.component) {
					var c = v.component;
					var nm = (c.type && (c.type.name || c.type.__name)) || '';
					if (nm === 'RightPanel') return c;
				}
				var subs = [];
				if (v.component && v.component.subTree) subs.push(v.component.subTree);
				if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
				if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
				for (var k=0;k<subs.length;k++) { var r = findRp(subs[k], d+1); if (r) return r; }
				return null;
			}
			var rp = findRp(root._vnode, 0);
			if (!rp) return {err:'no RightPanel'};
			var sub = rp.subTree;
			var dyn = sub && sub.dynamicChildren ? sub.dynamicChildren : [];
			// ★ 深度扫描 subTree：找 el=null 节点的完整路径
			function treeScan(v, path, depth, out) {
				if (!v || depth > 4) return;
				var t = typeof v.type === 'string' ? v.type : ((v.type && (v.type.__name || v.type.name)) || String(v.type));
				var entry = { p: path, t: t, el: !!v.el, pf: v.patchFlag, sf: v.shapeFlag, dynC: v.dynamicChildren ? v.dynamicChildren.length : 0 };
				if (v.component) entry.comp = (v.component.type && (v.component.type.name || v.component.type.__name)) || '?';
				out.push(entry);
				if (Array.isArray(v.children)) {
					for (var i = 0; i < v.children.length; i++) {
						treeScan(v.children[i], path + '.' + i, depth + 1, out);
					}
				}
				if (v.dynamicChildren && !(Array.isArray(v.children))) {
					for (var j = 0; j < v.dynamicChildren.length; j++) {
						treeScan(v.dynamicChildren[j], path + '[d' + j + ']', depth + 1, out);
					}
				}
			}
			var tree = [];
			treeScan(sub, 'root', 0, tree);
			var out = {
				subEl: !!sub.el,
				subPF: sub.patchFlag,
				subShape: sub.shapeFlag,
				subDynCount: dyn.length,
				tree: tree.slice(0, 80),
				subChildren: (function(){
					var chs = sub && sub.children;
					if (!Array.isArray(chs)) return 'not-array:' + (typeof chs);
					return chs.map(function(c, i){
						var t = typeof c.type === 'string' ? c.type : ((c.type && (c.type.__name || c.type.name)) || String(c.type));
						return { i: i, t: t, el: !!c.el, dyn: !!(c.dynamicChildren && c.dynamicChildren.length) };
					});
				})(),
				dyn: dyn.slice(0, 36).map(function(d, i){
					var t = typeof d.type === 'string' ? d.type : ((d.type && (d.type.__name || d.type.name)) || String(d.type));
					return { i: i, t: t, el: !!d.el, pf: d.patchFlag };
				}),
				updateTest: 'skip'
			};
			// ★★★★★ 终极三连：state.messages vs vnode 树消息 vs DOM 消息
			try {
				var msgState = (window.__state || {}).messages || [];
				// 从 subTree 树统计消息（fragment#cN 数 + msg-item 数）
				var treeMsgs = [];
				(function walk(n, path, d){
					if (!n || typeof n !== 'object' || d > 22) return;
					var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || '');
					var keyS = n.key !== undefined && n.key !== null ? String(n.key) : '';
					if (nm === 'div' && keyS !== '' && n.props && typeof n.props.class === 'string' && n.props.class.indexOf('msg-item') >= 0) {
						treeMsgs.push({ path: path, el: n.el === null ? 'null' : 'obj', cls: n.props.class.slice(0, 30) });
					}
					if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], path + '[' + i + ']', d+1);
				})(rp.subTree, 'root', 0);
				// c0/c1 fragment 的消息文本（第一个 msg-bubble 的文本）
				function bubbleText(v, d) {
					if (!v || typeof v !== 'object' || d > 12) return '';
					if (v.children && typeof v.children === 'string') return v.children.slice(0, 60);
					if (Array.isArray(v.children)) {
						for (var i=0;i<v.children.length;i++) { var t = bubbleText(v.children[i], d+1); if (t) return t; }
					}
					return '';
				}
				var cFrags = [];
				(function walkF(n, path, d){
					if (!n || typeof n !== 'object' || d > 22) return;
					var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || '');
					if (!nm && n.key !== undefined && n.key !== null && String(n.key).indexOf('c') === 0 && n.dynamicChildren) {
						cFrags.push({ key: String(n.key), el: n.el === null ? 'null' : 'obj', path: path, sample: bubbleText(n, 0).slice(0, 80) });
					}
					if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walkF(n.children[i], path + '[' + i + ']', d+1);
				})(rp.subTree, 'root', 0);
				out.triple = {
					stateMsgs: msgState.length,
					stateFirst: msgState[0] ? JSON.stringify(msgState[0]).slice(0, 150) : '',
					treeMsgItems: treeMsgs.slice(0, 10),
					treeMsgCount: treeMsgs.length,
					cFrags: cFrags.slice(0, 6),
					domMsgItems: document.querySelectorAll('.msg-item').length,
					domChatMsgs: !!document.querySelector('.chat-messages')
				};
			} catch(e) { out.tripleErr = String(e).slice(0, 150); }
			// 包装 update：捕获 patch 前的 prevTree 状态
			window.__patchErr = '';
			window.__prevDyn = [];
			var origUp = rp.update;
			rp.update = function() {
				try {
					var pd = this.subTree && this.subTree.dynamicChildren ? this.subTree.dynamicChildren : [];
					window.__prevDyn = pd.slice(0, 36).map(function(d, i){
						var t = typeof d.type === 'string' ? d.type : ((d.type && (d.type.__name || d.type.name)) || String(d.type));
						return { i: i, t: t, el: !!d.el, pf: d.patchFlag };
					});
				} catch(e) {}
				try { return origUp.apply(this, arguments); }
				catch(e) {
					window.__patchErr = String(e);
					window.__patchStack = String(e.stack || '').slice(0, 600);
					return undefined;
				}
			};
			try { rp.update(); out.updateTest = window.__patchErr ? 'ERR:' + window.__patchErr.slice(0, 120) : 'ok'; }
			catch(e) { out.updateTest = 'outerERR:' + String(e).slice(0, 120); }
			out.prevDyn = window.__prevDyn;
			out.patchStack = window.__patchStack || '';
			// ★ 手动 render 生成全新树 T3，检查 dynamicChildren 是否混入带 el 的旧 vnode
			try {
				var T3 = rp.render(rp.proxy, rp.renderCache);
				if (T3 && T3.dynamicChildren) {
					out.freshDyn = T3.dynamicChildren.slice(0, 36).map(function(d, i){
						var t = typeof d.type === 'string' ? d.type : ((d.type && (d.type.__name || d.type.name)) || String(d.type));
						return { i: i, t: t, el: !!d.el, pf: d.patchFlag, sameAsOld: !!(sub.dynamicChildren && d === sub.dynamicChildren[i]) };
					});
				} else {
					out.freshDyn = 'no-dyn:' + (T3 ? typeof T3 : 'null');
				}
			} catch(e) { out.freshErr = String(e).slice(0, 200); }
			// ★ 数组身份实验：多次 render 的 dynamicChildren 是否同一对象？
			try {
				var t1 = rp.render(rp.proxy, rp.renderCache);
				var t2 = rp.render(rp.proxy, rp.renderCache);
				var t3 = rp.render(rp.proxy, rp.renderCache);
				out.arrIdentity = {
					t1t2: t1.dynamicChildren === t2.dynamicChildren,
					t2t3: t2.dynamicChildren === t3.dynamicChildren,
					t1Sub: t1.dynamicChildren === sub.dynamicChildren,
					len1: t1.dynamicChildren ? t1.dynamicChildren.length : -1,
					lenSub: sub.dynamicChildren ? sub.dynamicChildren.length : -1
				};
				// ★★ 核心实验：render 产物 t1 的树结构与 dynamicChildren 的孤儿确认
				out.renderOrphans = (function(){
					// 收集 t1.children 树所有节点
					var treeSet = [];
					(function walk(n, d){
						if (!n || typeof n !== 'object' || d > 30) return;
						treeSet.push(n);
						if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], d+1);
						if (n.component && n.component.subTree) walk(n.component.subTree, d+1);
					})(t1, 0);
					var dyn1 = t1.dynamicChildren || [];
					var orphans = [];
					for (var i=0;i<dyn1.length;i++) {
						if (treeSet.indexOf(dyn1[i]) < 0) {
							var v = dyn1[i];
							orphans.push({ i: i, t: typeof v.type === 'string' ? v.type : ((v.type && (v.type.__name || v.type.name)) || String(v.type).slice(0,10)), el: v.el === null ? 'null' : (v.el === undefined ? 'undef' : 'obj'), key: v.key !== undefined ? String(v.key) : '' });
						}
					}
					// children 数组共享检测（对象引用级）
					var arrById = [];
					var arrMap = {};  // objId -> {count, paths, len}
					var nextId = 1;
					var sharedArrs = [];
					(function walk2(n, path){
						if (!n || typeof n !== 'object' || path.length > 60) return;
						if (Array.isArray(n.children)) {
							var id = arrById.indexOf(n.children);
							if (id < 0) { id = arrById.length; arrById.push(n.children); arrMap[id] = { count: 0, paths: [], len: n.children.length }; }
							arrMap[id].count++;
							arrMap[id].paths.push(path);
							for (var i=0;i<n.children.length;i++) walk2(n.children[i], path + '[' + i + ']');
						}
						if (n.component && n.component.subTree) walk2(n.component.subTree, path + '>c');
					})(t1, 'root');
					for (var id in arrMap) if (arrMap[id].count > 1) sharedArrs.push({ id: id, len: arrMap[id].len, count: arrMap[id].count, paths: arrMap[id].paths.slice(0,4) });
					return { dynCount: dyn1.length, treeSize: treeSet.length, orphans: orphans.slice(0, 30), sharedArrs: sharedArrs.slice(0,8) };
				})();
				// ★★ 决定性实验：subTree 与 render 产物 t1 的同一节点引用对比（树是否被替换）
				out.treeReplaceCheck = (function(){
					var sub = rp.subTree;
					var t1 = rp.render(rp.proxy, rp.renderCache);
					function findConvSidebar(root) {
						var res = [];
						(function walk(n, d){
							if (!n || typeof n !== 'object' || d > 20) return;
							if (n.type && (n.type.name === 'ConvSidebar' || n.type.__name === 'ConvSidebar')) { res.push(n); return; }
							if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], d+1);
							if (n.dynamicChildren) for (var j=0;j<n.dynamicChildren.length;j++) walk(n.dynamicChildren[j], d+1);
						})(root, 0);
						return res;
					}
					var sCS = findConvSidebar(sub);
					var tCS = findConvSidebar(t1);
					// 对比 root children 数组引用
					var sameRootChildren = sub.children === t1.children;
					// 对比 root dynamicChildren 数组引用
					var sameDynArr = sub.dynamicChildren === t1.dynamicChildren;
					// 对比「树的节点数」与「是否同集合」
					var subSet = [], t1Set = [];
					(function walk3(n){ if (!n || typeof n !== 'object') return; subSet.push(n); if (Array.isArray(n.children)) n.children.forEach(walk3); })(sub);
					(function walk4(n){ if (!n || typeof n !== 'object') return; t1Set.push(n); if (Array.isArray(n.children)) n.children.forEach(walk4); })(t1);
					// 从 subTree 树取 root[1]（body）与 t1 树 root[1] 对比
					var overlap = 0;
					for (var i=0;i<subSet.length;i++) if (t1Set.indexOf(subSet[i]) >= 0) overlap++;
					return {
						sameRootChildrenArr: sameRootChildren,
						sameDynArr: sameDynArr,
						subSize: subSet.length,
						t1Size: t1Set.length,
						overlap: overlap,
						subCS: sCS.map(function(v){ return { el: v.el === null ? 'null' : (v.el === undefined ? 'undef' : 'obj'), key: String(v.key), compMounted: !!(v.component && v.component.isMounted), compSubEl: !!(v.component && v.component.subTree && v.component.subTree.el) }; }),
						t1CS: tCS.map(function(v){ return { el: v.el === null ? 'null' : (v.el === undefined ? 'undef' : 'obj'), key: String(v.key), compMounted: !!(v.component && v.component.isMounted), compSubEl: !!(v.component && v.component.subTree && v.component.subTree.el) }; }),
						sameCSRef: !!(sCS.length && tCS.length && sCS[0] === tCS[0])
					};
				})();
				// ★★★ 树结构异常 dump：subTree 树里 ConvSidebar/重复结构的路径
				out.subTreeDup = (function(){
					var sub = rp.subTree;
					var found = [];
					(function walk(n, path, d){
						if (!n || typeof n !== 'object' || d > 25) return;
						var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || '');
						var keyS = n.key !== undefined && n.key !== null ? String(n.key) : '';
						var marker = nm + (keyS ? '#' + keyS : '');
						if (marker) {
							if (!found[marker]) found[marker] = [];
							if (found[marker].length < 4) found[marker].push({ path: path, el: n.el === null ? 'null' : (n.el === undefined ? 'undef' : 'obj'), dyn: !!(n.dynamicChildren) });
						}
						if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], path + '[' + i + ']', d+1);
					})(sub, 'root', 0);
					var dups = {};
					for (var m in found) if (found[m].length > 1) dups[m] = found[m];
					// body 区（root[1]）结构概要
					var body = sub.children && sub.children[1];
					var bodyShape = [];
					(function walkB(n, path, d){
						if (!n || typeof n !== 'object' || d > 8) return;
						var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || '?');
						var keyS = n.key !== undefined && n.key !== null ? String(n.key) : '';
						bodyShape.push(path + ':' + nm + (keyS ? '#' + keyS : ''));
						if (Array.isArray(n.children)) for (var i=0;i<n.children.length && i<8;i++) walkB(n.children[i], path + '[' + i + ']', d+1);
					})(body, 'root[1]', 0);
					return { dupMarkers: dups, bodyShape: bodyShape.slice(0, 40) };
				})();
				// ★★★★ #c0/#c1 双分支内容 dump：判定是哪个 v-for / 分支
				out.dupBranchContent = (function(){
					var sub = rp.subTree;
					var body = sub.children && sub.children[1];
					function shape(n, d) {
						if (!n || typeof n !== 'object' || d > 4) return '';
						var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || String(n.type).slice(0, 12));
						var cls = '';
						if (n.props) cls = n.props.class || '';
						if (typeof cls === 'object') cls = JSON.stringify(cls).slice(0, 40);
						var txt = '';
						if (n.children && typeof n.children === 'string') txt = n.children.slice(0, 30);
						var keyS = n.key !== undefined && n.key !== null ? '#' + String(n.key) : '';
						var elS = n.el === null ? 'E0' : (n.el === undefined ? 'EU' : 'E1');
						var out2 = nm + keyS + '[' + elS + (cls ? '|' + cls : '') + (txt ? '|' + txt : '') + ']';
						if (Array.isArray(n.children)) {
							out2 += '(';
							for (var i=0;i<n.children.length && i<6;i++) out2 += shape(n.children[i], d+1);
							out2 += ')';
						}
						return out2;
					}
					// 找 c0/c1：root[1] 下所有「带 key 的 Symbol」
					var res = {};
					(function walk(n, path, d){
						if (!n || typeof n !== 'object' || d > 22) return;
						var nm = typeof n.type === 'string' ? n.type : ((n.type && (n.type.name || n.type.__name)) || '');
						if (!nm && n.key !== undefined && n.key !== null) {
							var ks = String(n.key);
							if (!res[ks]) res[ks] = [];
							if (res[ks].length < 2) res[ks].push({ path: path, shape: shape(n, 0).slice(0, 400) });
						}
						if (Array.isArray(n.children)) for (var i=0;i<n.children.length;i++) walk(n.children[i], path + '[' + i + ']', d+1);
					})(sub, 'root', 0);
					return res;
				})();
			} catch(e) { out.arrIdentityErr = String(e).slice(0, 200); }
			// ★ 决定性实验：children 树里的 vnode 与 dynamicChildren 里的 vnode 是否同一对象？
			try {
				var allChildren = [];
				function collectRefs(v) {
					if (!v) return;
					allChildren.push(v);
					if (Array.isArray(v.children)) v.children.forEach(collectRefs);
					if (v.component && v.component.subTree) collectRefs(v.component.subTree);
					if (v.dynamicChildren) v.dynamicChildren.forEach(collectRefs);
				}
				collectRefs(sub);
				out.dynInChildren = dyn.slice(0, 36).map(function(d, i){
					return allChildren.indexOf(d) >= 0;
				});
				out.subChildrenInTree = sub.children.map(function(c){ return allChildren.indexOf(c) >= 0; });
			} catch(e) { out.refErr = String(e).slice(0, 200); }
			// ★ 看 render 编译产物的 _cache 用法（缓存 vnode 共享嫌疑）
			try {
				var rsrc = String(rp.render);
				out.cachePatterns = (rsrc.match(/_cache\[[^\]]*\]/g) || []).slice(0, 24);
				out.renderSrcHead = rsrc.slice(0, 500);
			} catch(e) { out.cacheErr = String(e).slice(0, 120); }
			// ★ 决定性实验：JSC 的 Promise.then 是同步还是微任务？
			try {
				var order = [];
				new Promise(function(res){ order.push('exec'); res(); }).then(function(){ order.push('then'); });
				order.push('after');
				out.promiseOrder = order.join(',');
				var torder = [];
				setTimeout(function(){ torder.push('timeout'); }, 0);
				torder.push('t-after');
				out.timeoutOrder = torder.join(',');
			} catch(e) { out.promiseErr = String(e).slice(0, 120); }
			// ★ 读注入 hook 的日志
			try {
				out.hookLog = JSON.stringify(window.__hookLog || 'no-hook');
				out.hookErr = JSON.stringify(window.__hookErr || 'none');
				out.mtLog = JSON.stringify(window.__mtLog || 'no-mt');
				out.errLog = JSON.stringify(window.__errLog || 'no-err');
				out.firstBad = JSON.stringify(window.__firstBad || 'none');
				out.subRefs = JSON.stringify(window.__subRefs || 'no-subs');
			} catch(e) { out.hookErr = String(e).slice(0, 120); }
			// ★ 无缺陷引用检查：只沿 children 树（不碰 dynamicChildren）收集对象，检查 dyn[i] 是否在其中
			try {
				var childSet = [];
				function collectCh(v) {
					if (!v) return;
					childSet.push(v);
					if (v.component && v.component.subTree) collectCh(v.component.subTree);
					if (Array.isArray(v.children)) v.children.forEach(collectCh);
				}
				collectCh(sub);
				out.dynInChildTree = dyn.slice(0, 36).map(function(d){ return childSet.indexOf(d) >= 0; });
				out.childTreeSize = childSet.length;
			} catch(e) { out.childTreeErr = String(e).slice(0, 120); }
			// ★ el 真实性检查：dyn[i].el 是否真的是 DOM 节点？DOM 里实际有什么？
			try {
				out.dynElDetail = dyn.slice(0, 20).map(function(d){
					var e = d && d.el;
					var info = { t: typeof e };
					if (e && typeof e === 'object') {
						info.nt = e.nodeType;
						info.tag = e.tagName || e.nodeName || '';
					}
					return info;
				});
				out.domProbe = {
					rpHeader: !!document.querySelector('.rp-header, [class*="rp-header"]'),
					rpBody: !!document.querySelector('.rp-body, [class*="rp-body"]'),
					chatMsgs: !!document.querySelector('.chat-messages, [class*="chat-messages"]'),
					convItems: document.querySelectorAll('.conv-item').length,
					bodyChildren: (document.body ? document.body.childNodes.length : -1),
					appChildren: (document.querySelector('#app') ? document.querySelector('#app').childNodes.length : -1)
				};
				// ★ dyn 7+ 的 el 对象身份检查（是否共享同一对象）
				var firstEl = dyn[7] && dyn[7].el;
				if (firstEl && typeof firstEl === 'object') {
					var same = true;
					for (var ei = 8; ei < Math.min(dyn.length, 33); ei++) { if (dyn[ei].el !== firstEl) { same = false; break; } }
					out.elShared = { same: same, keys: Object.keys(firstEl).slice(0, 12), str: String(firstEl).slice(0, 80) };
				} else {
				out.elShared = 'el-not-obj:' + (firstEl === undefined ? 'undef' : (firstEl === null ? 'null' : typeof firstEl));
			}
			// ★ dyn 7+ 在树中的精确路径定位
			try {
				out.dynPaths = dyn.slice(7, 34).map(function(d, ii){
					var foundPath = null;
					var depth = 0;
					(function walk(n, path){
						if (foundPath || depth > 40 || !n || typeof n !== 'object') return;
						depth++;
						if (n === d) { foundPath = path; return; }
						if (n.component && n.component.subTree) walk(n.component.subTree, path + '>' + (n.component.type && (n.component.type.__name || n.component.type.name) || 'c'));
						if (Array.isArray(n.children)) for (var j = 0; j < n.children.length; j++) walk(n.children[j], path + '[' + j + ']');
					})(sub, 'root');
					var t = typeof d.type === 'string' ? d.type : ((d.type && (d.type.__name || d.type.name)) || String(d.type).slice(0,12));
					return { i: ii + 7, t: t, path: foundPath || 'NOT-IN-TREE', el: d.el === null ? 'null' : (d.el === undefined ? 'undef' : 'obj'), key: d.key !== undefined ? String(d.key) : '' };
				});
			} catch(e) { out.dynPathsErr = String(e).slice(0, 120); }
			// ★ 完整 render 源码（找 dyn 7+ 的创建代码 + _cache 使用）
			try {
				out.renderFull = String(rp.type && rp.type.render).slice(0, 3000);
			} catch(e) { out.renderFullErr = String(e).slice(0, 80); }
			// ★ dyn 7+ vnode 是否与 children 树里的 vnode 同一对象
			try {
				out.dynVsTree = (function(){
					var cnt = 0, same = 0;
					(function walk(n){
						if (!n || typeof n !== 'object') return;
						cnt++;
						for (var i = 7; i < Math.min(dyn.length, 33); i++) {
							if (n === dyn[i]) { same++; }
						}
						if (Array.isArray(n.children)) { for (var j = 0; j < n.children.length; j++) walk(n.children[j]); }
					})(rp.subTree);
					return { cnt: cnt, same: same };
				})();
			} catch(e) { out.dynVsTreeErr = String(e).slice(0, 80); }
			} catch(e) { out.elDetailErr = String(e).slice(0, 120); }
			return JSON.stringify(out);
		})()`)
		fmt.Printf("[conv] INIT snap: %s\n", is.ToString())
	}
	// 等 Vue mount + 异步数据
	for i := 0; i < 10; i++ {
		_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 300); })`)
		time.Sleep(300 * time.Millisecond)
	}
	_, _ = wv.JSInterpreter().RunJS(`new Promise(function(res){ setTimeout(res, 800); })`)
	time.Sleep(800 * time.Millisecond)
	// ★ 若 conversations 为空（API 未配置/无工作区），注入假数据并手动触发渲染
	inj, _ := wv.JSInterpreter().RunJS(`(function(){
		try {
			var st = window.__state;
			if (!st) return 'no-store';
			if (document.querySelectorAll('.conv-item').length >= 2) return 'already';
			var convs = [];
			for (var i=1;i<=6;i++) {
				convs.push({ id:'conv-'+i, title:'对话 '+i, updatedAt:'2026-08-05T12:00:0'+i+'Z', lastMessage:'测试消息 '+i, unread:0, messageCount:i*10 });
			}
			try {
				st.conversations.splice(0, st.conversations.length);
				convs.forEach(function(c){ st.conversations.push(c); });
			} catch(e2) { st.conversations = convs; }
			if (!st.currentConvId) st.currentConvId = 'conv-1';
			// 手动触发 RightPanel update 让 ConvSidebar 拿到新数据
			var el0 = document.querySelector('#app');
			var vnode = el0._vnode;
			function walkV(v, d) {
				if (!v || d > 12) return null;
				if (v.component) {
					var nm = v.component.type && (v.component.type.name || v.component.type.__name);
					if (nm === 'RightPanel') return v.component;
				}
				var subs = [];
				if (v.component && v.component.subTree) subs.push(v.component.subTree);
				if (Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
				if (v.dynamicChildren) v.dynamicChildren.forEach(function(c){ if (c) subs.push(c); });
				for (var k=0;k<subs.length;k++) { var r = walkV(subs[k], d+1); if (r) return r; }
				return null;
			}
			var rp2 = walkV(vnode, 0);
			var upd = 'no-rp';
			if (rp2) { try { rp2.update(); upd = 'ok'; } catch(e3) { upd = 'ERR:' + String(e3).slice(0,120); } }
			return JSON.stringify({convCount: st.conversations.length, cur: st.currentConvId, upd: upd});
		} catch(e) { return 'err:' + String(e).slice(0,150); }
	})()`)
	fmt.Printf("[conv] inject convs: %s\n", inj.ToString())
	// flush + 重新布局
	if wv.JSInterpreter() != nil {
		for i := 0; i < 8; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
	}
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	rv := wv.RenderView()
	if rv == nil {
		log.Fatal("no render view")
	}

	// 1) 收集 .conv-item 几何 + 初始 active 状态
	var items []convItem
	var findConv func(o rendering.RenderObject)
	findConv = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			lb := o.LayoutBox()
			if strings.Contains(cn, "conv-item") && lb != nil && rv.LayoutState() != nil {
				g := rv.LayoutState().GeometryForBox(lb)
				items = append(items, convItem{
					el:     el,
					x:      g.Left(),
					y:      g.Top(),
					w:      g.BorderBoxWidth(),
					h:      g.BorderBoxHeight(),
					active: strings.Contains(cn, "active"),
				})
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findConv(c)
		}
	}
	findConv(rendering.RenderObject(rv))
	fmt.Printf("[conv] found %d conv-items\n", len(items))
	for i, it := range items {
		fmt.Printf("[conv]   #%d %q x=%.0f y=%.0f w=%.0f h=%.0f active=%v\n",
			i, it.el.TextContent()[:min(18, len(it.el.TextContent()))], it.x, it.y, it.w, it.h, it.active)
	}
	if len(items) < 2 {
		log.Fatal("need >=2 conv-items")
	}

	// 记录点击前的 active 状态
	beforeActive := -1
	for i, it := range items {
		if it.active {
			beforeActive = i
		}
	}
	fmt.Printf("[conv] before: active=#%d\n", beforeActive)

	// 点击前读 store 基准
	st0, _ := wv.JSInterpreter().RunJS(`(function(){
		try {
			var st = window.__state;
			if (!st) return JSON.stringify({hasStore: false});
			var c = st.currentConvId;
			var idx = -1;
			if (Array.isArray(st.conversations)) for (var i=0;i<st.conversations.length;i++) if (st.conversations[i].id === c) { idx = i; break; }
			return JSON.stringify({hasStore: true, currentConvId: c, convIdx: idx, convCount: st.conversations.length, conv0Title: st.conversations[0] ? st.conversations[0].title : ''});
		} catch(e) { return JSON.stringify({hasStore: true, err: String(e).slice(0,120)}); }
	})()`)
	fmt.Printf("[conv] before store: %s\n", st0.ToString())

	// 2) 模拟 host.handleClick 对第 2 个 item 的完整流程
	target := items[1]
	cx := target.x + target.w/2
	cy := target.y + target.h/2

	// 2.5) 诊断：conv-item 是否注册了 click listener？冒泡路径？
	for _, it := range items[:3] {
		fmt.Printf("[conv] #%d HasEventListener(click)=%v\n", idx(items, it), it.el.HasEventListener("click"))
	}
	// 从命中元素（conv-title）向上打印祖先链
	deep := rendering.HitTest(rv, cx, cy, "")
	if deep != nil {
		var chain []string
		for n := dom.Node(deep); n != nil; n = n.ParentNode() {
			if e, ok := n.(*dom.Element); ok {
				chain = append(chain, e.LocalName()+"."+e.GetAttribute("class"))
			} else {
				chain = append(chain, n.NodeName())
			}
		}
		fmt.Printf("[conv] ancestor chain: %v\n", chain)
	}
	// 注入探针监听 conv-item#1 的 click，验证 dispatch 是否到达
	_, _ = wv.JSInterpreter().RunJS(`
		window.__convProbe = [];
		var it = document.querySelectorAll('.conv-item')[1];
		it.addEventListener('click', function(e){
			window.__convProbe.push('bubble:' + e.type + ' target=' + (e.target.className||e.target.nodeName));
		}, false);
		it.addEventListener('click', function(e){
			window.__convProbe.push('capture:' + e.type);
		}, true);
		// ★ Proxy / reactive 能力探测（Vue3 响应式依赖）
		window.__proxyTest = (function(){
			var out = { typeOf: typeof Proxy, hasProxy: !!window.Proxy,
				// ★ Symbol.for 语义测试（Vue isSameVNodeType 依赖 Symbol.for 全局单例）
				symFor: (function(){
					var r = {};
					try {
						r.typeofSym = typeof Symbol;
						r.hasFor = typeof Symbol.for === 'function';
						var a = Symbol.for('v-fgt');
						var b = Symbol.for('v-fgt');
						r.same = (a === b);
						r.sameTwice = (Symbol.for('v-fgt') === Symbol.for('v-fgt'));
						var v1 = { type: a, __v_isVNode: true };
						var v2 = { type: b, __v_isVNode: true };
						r.sameTypeVnode = (v1.type === v2.type);
						r.descA = String(a);
						r.descB = String(b);
					} catch(e) { r.err = String(e).slice(0, 100); }
					return r;
				})()
			};
			try {
				var target = { a: 1 };
				var p = new Proxy(target, { set: function(t, k, v){ t[k] = v; window.__proxySetCalled = (window.__proxySetCalled||0)+1; return true; } });
				p.a = 42;
				out.setWorked = (target.a === 42);
				out.setTrapCalled = window.__proxySetCalled || 0;
				out.getWorked = (p.a === 42);
				out.proxyCreated = true;
			} catch(e) {
				out.proxyCreated = false;
				out.proxyErr = String(e).slice(0,100);
			}
			return JSON.stringify(out);
		})();
		// 检查 Vue app 实例与组件树（生产构建 __vueParentComponent 可能被移除）
		window.__vueAppInfo = (function(){
			var el = document.querySelector('#app');
			var out = { keys: Object.keys(el).filter(function(k){return k.indexOf('__')>=0;}) };
			var app = el.__vue_app__;
			if (app) {
				out.hasApp = true;
				out.appKeys = Object.keys(app).slice(0,15).join(',');
				var inst = app._instance;
				if (inst) {
					out.hasRootInstance = true;
					out.rootType = inst.type && (inst.type.name || inst.type.__name || 'anon');
				} else out.hasRootInstance = false;
			} else out.hasApp = false;
			// ★ 从 #app._vnode 遍历组件树（绕过 app._instance=null）
			try {
				var vnode = el._vnode;
				out.hasContainerVnode = !!vnode;
				if (vnode && vnode.component) {
					out.rootComp = vnode.component.type && (vnode.component.type.name || vnode.component.type.__name || 'anon');
					// ★ 打印 App 实例的所有键 + setupState 详情
					try {
						var ai = vnode.component;
						out.appInstKeys = Object.keys(ai).slice(0,40).join(',');
						out.appSetupType = typeof ai.setupState;
						// ★ 深读 setupState：JSON 内容 / Reflect.ownKeys / 直接属性
						try {
							var ssJson = JSON.stringify(ai.setupState);
							out.appSetupJson = ssJson ? ssJson.slice(0, 200) : '(empty json)';
						} catch(e) { out.appSetupJsonErr = String(e).slice(0,80); }
						try {
							out.appOwnKeys = Reflect.ownKeys(ai.setupState).slice(0,40).join(',');
						} catch(e) { out.appOwnKeysErr = String(e).slice(0,80); }
						out.appStateDirect = ai.setupState && ai.setupState.state ? 'yes' : 'no';
						out.appCtxKeys = ai.ctx ? Object.keys(ai.ctx).slice(0,30).join(',') : 'none';
						out.appHasProxy = !!ai.proxy;
						out.appHasRender = typeof ai.render;
						out.appDataType = typeof ai.data;
						// ★ 通过 ctx 找 state（render 上下文里 state 可通过代理访问）
						try {
							if (ai.ctx) { out.appCtxState = ai.ctx.state ? 'yes' : 'no'; }
						} catch(e) { out.appCtxErr = String(e).slice(0,80); }
						// ★ dist 环境 Proxy 行为测试（模拟 Vue proxyRefs + reactive 嵌套）
						try {
							var raw = { state: { name: 'x', items: [1,2,3] }, showSettings: { _v_isRef: true, value: false } };
							var pr = new Proxy(raw, {
								get: function(t, k, r) {
									var v = Reflect.get(t, k, r);
									return (v && v._v_isRef) ? v.value : v;
								}
							});
							out.envPrKeys = Object.keys(pr).join(',');
							out.envPrJson = JSON.stringify(pr);
							out.envPrState = String(pr.state.name);
							// 嵌套 Proxy（reactive 里套 proxyRefs）
							var inner = new Proxy({a:1,b:2}, { get: function(t,k){ return Reflect.get(t,k); } });
							var outer = new Proxy({ convs: inner }, { get: function(t,k,r){ return Reflect.get(t,k,r); } });
							var fk = []; for (var k0 in outer) fk.push(k0);
							var fk2 = []; for (var k1 in outer.convs) fk2.push(k1);
							out.envNestedForIn = fk.join(',') + '|' + fk2.join(',');
						} catch(e) { out.envErr = String(e).slice(0,120); }
						// ★ ai.proxy（公共渲染代理）能否访问 state
						try {
							out.proxyStateType = typeof ai.proxy.state;
							out.proxyStateKeys = ai.proxy.state ? Object.keys(ai.proxy.state).slice(0,10).join(',') : 'none';
							out.proxySetKeys = ai.proxy.setupState ? Object.keys(ai.proxy.setupState).slice(0,10).join(',') : 'none';
						} catch(e) { out.proxyErr = String(e).slice(0,120); }
						// ★ app._instance 为何 null？（Vue3 mount 后 _instance 本为 null，正常）
						try {
							out.rootInstDetail = String(app._instance);
							// ★★ 决定性实验：手动调用 App 的 setup()，看返回值/异常
							var rc = app._component;
							out.rootCompHasSetup = typeof rc.setup;
							out.rootCompKeys = rc ? Object.keys(rc).slice(0,15).join(',') : 'none';
							if (rc && rc.setup) {
								out.rootSetupSrc = String(rc.setup).slice(0, 200);
								var fakeInst = { uid: 999, vnode: {}, type: rc, parent: null, appContext: app._context, provides: app._context.provides, emit: function(){}, isMounted: false, isUnmounted: false };
								var fakeCtx = { attrs: {}, slots: {}, emit: function(){}, expose: function(){} };
								var sr = null;
								try { sr = rc.setup(fakeInst, fakeCtx); out.manualSetupResult = typeof sr; } catch(e) { out.manualSetupErr = String(e).slice(0, 200); }
								if (sr && typeof sr === 'object') {
									try { out.manualSetupKeys = Object.keys(sr).slice(0, 20).join(','); } catch(e) { out.manualSetupKeysErr = String(e).slice(0,100); }
								}
								// ★ 看 setup 源码结尾的 return 语句
								try {
									var ssrc = String(rc.setup);
									var ri = ssrc.lastIndexOf('return ');
									out.rootSetupTail = ssrc.slice(Math.max(0, ri-150), Math.min(ssrc.length, ri+250));
									out.rootSetupLen = ssrc.length;
								} catch(e) { out.rootSetupTailErr = String(e).slice(0,100); }
							}
						} catch(e) { out.rootInstErr = String(e).slice(0,120); }
					} catch(e) { out.appSetupErr = String(e).slice(0,120); }
					// ★ 打印组件树：所有组件 name + props 键 + setupState 键数
					var tree = [];
					var rpInfo = null;
					function walkTree(v, d, tag) {
						if (!v || d > 10) return;
						if (v.component) {
							var c = v.component;
							var nm = (c.type && (c.type.name || c.type.__name)) || 'anon';
							var pkeys = c.props ? Object.keys(c.props).slice(0,10).join(',') : '';
							var skeys = c.setupState ? (function(){ var a=[]; try { for (var k in c.setupState) a.push(k); } catch(e){} return a.slice(0,8).join(','); })() : '';
							var st = c.setupState && c.setupState.state;
							// ★ 找 RightPanel 实例：检查其 subTree 是否存在（vnode 树 vs DOM 一致性）
							if (nm === 'RightPanel' && !rpInfo) {
								var sub = c.subTree;
								rpInfo = {
									subTreeType: sub ? String((sub.type && (sub.type.name || sub.type.__name)) || (sub.tag || '?')) : 'none',
									subTreeHasComponent: sub && sub.component ? 'yes' : 'no',
									subTreeChildCount: sub && sub.children && Array.isArray(sub.children) ? sub.children.length : -1,
									renderSrcHasCur: String(c.render).indexOf('currentConvId') >= 0,
									renderSrcHasState: String(c.render).indexOf('state$') >= 0 || String(c.render).indexOf('state.') >= 0,
									renderSrcHead: String(c.render).slice(0, 120),
								};
							}
							// ★ 找 ConvSidebar：记录其 props
							if (nm === 'ConvSidebar' && v.props && !window.__csInfo) {
								window.__csInfo = {
									propsKeys: Object.keys(v.props).slice(0,20).join(','),
									curProp: v.props.currentConvId !== undefined ? v.props.currentConvId : (v.props['current-conv-id'] !== undefined ? v.props['current-conv-id'] : 'none'),
									convPropLen: v.props.conversations ? v.props.conversations.length : -1,
								};
							}
							tree.push({ d: d, n: nm, tag: tag, sk: skeys, pk: pkeys, hasState: !!st, cur: st ? st.currentConvId : undefined });
						}
						var subs = [];
						if (v.children && Array.isArray(v.children)) { v.children.forEach(function(x){ if (x) subs.push(x); }); }
						if (v.dynamicChildren) { subs = subs.concat(v.dynamicChildren); }
						if (v.component && v.component.subTree) subs.push(v.component.subTree);
						subs.forEach(function(s){ walkTree(s, d+1, tag + '.' + (s.type && (s.type.name || s.type.__name || '?') || (s.tag||'frag'))); });
					}
					walkTree(vnode, 0, 'root');
					out.rpInfo = rpInfo ? JSON.stringify(rpInfo) : 'none';
					out.csInfo = window.__csInfo ? JSON.stringify(window.__csInfo) : 'none';
					out.specialTree = JSON.stringify(tree.filter(function(t){ return t.n === 'RightPanel' || t.n === 'ConvSidebar' || t.n === 'Sidebar' || t.n === 'EditorArea'; }).slice(0,6));
					out.treeCount = tree.length;
					out.tree = JSON.stringify(tree.slice(0, 25));
				}
			} catch(e) { out.walkErr = String(e).slice(0,150); }
			return JSON.stringify(out);
		})();
		// 记录 conv-item 上 Vue 注册的 listener（_vei 是 Vue 3 的 invoker 缓存，Symbol 键）
		window.__veiInfo = (function(){
			var el = document.querySelectorAll('.conv-item')[1];
			var out = {};
			// 拦截 console.log 以观察 switchConv 是否执行
			window.__capturedLogs = [];
			var origLog = console.log.bind(console);
			console.log = function(){
				try { window.__capturedLogs.push(Array.prototype.slice.call(arguments).join(' ').slice(0,200)); } catch(e){}
				return origLog.apply(null, arguments);
			};
			// Symbol 键（Vue 3 用 Symbol("_vei")）
			var syms = Object.getOwnPropertySymbols(el);
			syms.forEach(function(s){ out['sym:' + String(s)] = typeof el[s]; });
			// 找到 Vue invoker 并手动调用
			var invokers = null;
			for (var i=0;i<syms.length;i++) {
				var v = el[syms[i]];
				if (v && typeof v === 'object' && v.onClick) { invokers = v; break; }
			}
			if (invokers && invokers.onClick) {
				out['hasInvoker'] = 'yes';
				var ev = { type:'click', target: el, currentTarget: el, bubbles: true, cancelable: true };
				try { invokers.onClick(ev); out['invokeResult'] = 'ok'; }
				catch(e) { out['invokeResult'] = 'ERR:' + e.message; }
			} else {
				out['hasInvoker'] = 'no';
			}
			return JSON.stringify(out);
		})();
	`)

	// ★ 点击前保存 RightPanel.subTree（旧树）dynamicChildren 快照
	if wv.JSInterpreter() != nil {
		wv.JSInterpreter().RunJS(`(function(){
			window.__preSnap = (function(){
				var root = document.querySelector('#app');
				if (!root || !root._vnode) return {err:'no vnode'};
				function findRp(v, d) {
					if (!v || d > 14) return null;
					if (v.component) {
						var c = v.component;
						var nm = (c.type && (c.type.name || c.type.__name)) || '';
						if (nm === 'RightPanel') return c;
					}
					var subs = [];
					if (v.component && v.component.subTree) subs.push(v.component.subTree);
					if (Array.isArray(v.children)) for (var i=0;i<v.children.length;i++) subs.push(v.children[i]);
					if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
					for (var k=0;k<subs.length;k++) { var r = findRp(subs[k], d+1); if (r) return r; }
					return null;
				}
				var rp = findRp(root._vnode, 0);
				if (!rp) return {err:'no RightPanel'};
				var sub = rp.subTree;
				var dyn = sub && sub.dynamicChildren ? sub.dynamicChildren : [];
				return {
					subEl: !!sub.el,
					dynCount: dyn.length,
					dyn: dyn.slice(0, 36).map(function(d, i){
						var t = typeof d.type === 'string' ? d.type : ((d.type && (d.type.__name || d.type.name)) || 'c');
						var t2 = '';
						try { if (d.type && d.type.__name) t2 = String(d.type.__name); } catch(e) {}
						var pk = '';
						try { if (d.props) { var a = []; for (var k in d.props) { a.push(k); if (a.length >= 5) break; } pk = a.join(','); } } catch(e) {}
						return { i: i, t: t, t2: t2, el: !!d.el, pf: d.patchFlag, key: d.key !== undefined ? String(d.key) : '', pk: pk };
					}),
					firstBad: (function(){
						for (var bi=0; bi<dyn.length; bi++) {
							if (!dyn[bi].el) {
								var d2 = dyn[bi];
								var info = { i: bi, t2: '', typeSrc: '', hasComp: !!d2.component };
								try { if (d2.type && d2.type.__name) info.t2 = String(d2.type.__name); } catch(e) {}
								try { info.typeSrc = String(d2.type).slice(0, 160); } catch(e) {}
								try {
									if (d2.component && d2.component.subTree) {
										var cs = d2.component.subTree;
										var cd = cs.dynamicChildren || [];
										info.compSubType = typeof cs.type === 'string' ? cs.type : (cs.type && (cs.type.__name || cs.type.name)) || '?';
										info.compSubEl = !!cs.el;
										info.compDynCount = cd.length;
										info.compDyn = cd.slice(0, 8).map(function(d3, j){
											var t3 = typeof d3.type === 'string' ? d3.type : ((d3.type && (d3.type.__name || d3.type.name)) || 'c');
											return { j: j, t: t3, el: !!d3.el, pf: d3.patchFlag };
										});
									}
								} catch(e) { info.compErr = String(e).slice(0,100); }
								return info;
							}
						}
						return null;
					})()
				};
			})();
		})();`)
	}

	el := rendering.HitTest(rv, cx, cy, "onclick")
	desc := "<nil>"
	if el != nil {
		desc = el.LocalName() + "." + el.GetAttribute("class") + " onclick=" + el.GetAttribute("onclick")
	}
	fmt.Printf("[conv] HitTest(x=%.0f,y=%.0f,onclick) -> %s\n", cx, cy, desc)

	if el == nil {
		deepest := rendering.HitTest(rv, cx, cy, "")
		dd := "<nil>"
		if deepest != nil {
			dd = deepest.LocalName() + "." + deepest.GetAttribute("class")
		}
		fmt.Printf("[conv] HitTest(x=%.0f,y=%.0f,\"\") -> %s\n", cx, cy, dd)
		if deepest != nil {
			deepest.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
		}
	} else {
		el.DispatchEvent(dom.NewMouseEvent(dom.EventClick, true, true, false))
	}

	// 3) flush JS 微任务（同 host.handleClick 的多轮 RunJobs + sleep）
	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
		for i := 0; i < 8; i++ {
			time.Sleep(15 * time.Millisecond)
			process(wv)
			wv.JSInterpreter().RunJobs()
		}
	}
	wv.RebuildRenderTree()

	// 4) 读点击后的 active 状态（DOM + store 状态层 + ConvSidebar props 层）
	v, _ := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var out = [];
		for (var i=0;i<items.length && i<20;i++) {
			out.push((items[i].className.indexOf('active')>=0?'A':'-') + ':' + items[i].textContent.replace(/\s+/g,' ').slice(0,16));
		}
		var act = '';
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) { act = items[i].textContent.slice(0,20); break; }
		// ★ store 状态层：currentConvId 是否已切换（优先 window.__state，回退 ai.proxy.state）
		var storeInfo = { hasStore: false };
		try {
			var st = null;
			if (typeof window !== 'undefined' && window.__state) st = window.__state;
			else if (typeof ai !== 'undefined' && ai.proxy && ai.proxy.state) st = ai.proxy.state;
			if (st) {
				storeInfo.hasStore = true;
				storeInfo.currentConvId = st.currentConvId;
				storeInfo.convCount = Array.isArray(st.conversations) ? st.conversations.length : -1;
				var c0 = st.conversations && st.conversations[0];
				storeInfo.conv0Title = c0 ? c0.title : '';
				storeInfo.messagesByConvLen = st.messagesByConv ? Object.keys(st.messagesByConv).length : -1;
			}
		} catch(e) { storeInfo.storeErr = String(e).slice(0,100); }
		// ★ props 层：ConvSidebar 实例的 props.currentConvId
		var propInfo = { found: false };
		try {
			var root = document.querySelector('#app');
			if (root && root._vnode) {
				function walkCs(v, d) {
					if (!v || d > 14) return null;
					if (v.component && v.component.type && (v.component.type.__name || v.component.type.name) === 'ConvSidebar') return v.component;
					var subs = [];
					if (v.component && v.component.subTree) subs.push(v.component.subTree);
					if (v.dynamicChildren) for (var j=0;j<v.dynamicChildren.length;j++) subs.push(v.dynamicChildren[j]);
					if (Array.isArray(v.children)) for (var k=0;k<v.children.length;k++) subs.push(v.children[k]);
					for (var m=0;m<subs.length;m++) { var r = walkCs(subs[m], d+1); if (r) return r; }
					return null;
				}
				var cs = walkCs(root._vnode, 0);
				if (cs) {
					propInfo.found = true;
					propInfo.propsCurrentConvId = cs.props ? cs.props.currentConvId : 'no-props';
					propInfo.setupCurrentConvId = cs.setupState ? cs.setupState.currentConvId : 'no-setup';
				}
			}
		} catch(e) { propInfo.propErr = String(e).slice(0,100); }
		return JSON.stringify({list: out, activeTitle: act, items: items.length, store: storeInfo, prop: propInfo});
	})()`)
	fmt.Printf("[conv] after: %s\n", v.ToString())
	// ★ 决定性实验：点击后手动调用 RightPanel 的 instance.update()（render effect），看 DOM 是否更新
	vu, _ := wv.JSInterpreter().RunJS(`(function(){
		var el = document.querySelector('#app');
		var vnode = el._vnode;
		var found = null;
		function walkV(v, d) {
			if (found || d > 8 || !v) return;
			if (v.component) {
				var nm = v.component.type && (v.component.type.name || v.component.type.__name);
				if (nm === 'RightPanel') { found = v.component; return; }
				if (v.component.subTree) walkV(v.component.subTree, d+1);
			}
			var subs = [];
			if (v.children && Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
			if (v.dynamicChildren) subs = subs.concat(v.dynamicChildren);
			subs.forEach(function(s){ walkV(s, d+1); });
		}
		walkV(vnode, 0);
		if (!found) return JSON.stringify({found: false});
		var out = { found: true };
		out.hasUpdate = typeof found.update;
		out.renderHead = String(found.render).slice(0, 250);
		// ★ 检查 subTree 的 dynamicChildren 各 vnode 的 el 完整性
		try {
			var sub = found.subTree;
			var dcs = sub && sub.dynamicChildren ? sub.dynamicChildren : null;
			out.subType = sub ? String(sub.type) : 'none';
			out.subDynCount = dcs ? dcs.length : -1;
			if (dcs) {
				out.subDynEls = dcs.slice(0, 12).map(function(d, i) {
					return { i: i, hasEl: !!d.el, t: typeof d.type === 'string' ? d.type : (d.type && (d.type.__name || d.type.name) || 'c'), hasDyn: !!(d.dynamicChildren && d.dynamicChildren.length) };
				});
			}
			// children 结构
			var chs = sub && sub.children;
			out.subChildrenType = Array.isArray(chs) ? 'array(' + chs.length + ')' : (typeof chs === 'string' ? 'string' : 'none');
			if (Array.isArray(chs)) {
				out.subChildren = chs.slice(0, 8).map(function(c, i) {
					return { i: i, t: typeof c.type === 'string' ? c.type : (c.type && (c.type.__name || c.type.name)) || '?', hasEl: !!c.el };
				});
			}
		} catch(e) { out.subErr = String(e).slice(0,120); }
		// 记录调用前的 active
		var items0 = document.querySelectorAll('.conv-item');
		var act0 = '';
		for (var i=0;i<items0.length;i++) if (items0[i].className.indexOf('active')>=0) { act0 = items0[i].textContent.slice(0,20); break; }
		out.activeBefore = act0;
		// ★ 读点击前快照（旧树）
		if (window.__preSnap) out.preSnap = window.__preSnap;
		// 手动调用 render effect
		try { found.update(); out.updateOk = 'yes'; } catch(e) { out.updateOk = 'ERR:' + String(e).slice(0,120); out.updateStack = String(e.stack || 'no-stack').slice(0, 800); }
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[conv] manual update: %s\n", vu.ToString())
	// flush 后检查 active 是否变化
	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
	}
	wv.RebuildRenderTree()
	vu2, _ := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var act = '';
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) { act = items[i].textContent.slice(0,20); break; }
		return JSON.stringify({activeAfterUpdate: act});
	})()`)
	fmt.Printf("[conv] after update: %s\n", vu2.ToString())
	// ★ 决定性实验：手动调 RightPanel.render 看新树中 ConvSidebar props；直接调 ConvSidebar.update()
	cse, _ := wv.JSInterpreter().RunJS(`(function(){
		var out = { };
		var el0 = document.querySelector('#app');
		var vnode = el0._vnode;
		var rp = null;
		function walkV(v, d) {
			if (rp || d > 12 || !v) return;
			if (v.component) {
				var nm = v.component.type && (v.component.type.name || v.component.type.__name);
				if (nm === 'RightPanel') { rp = v.component; return; }
			}
			var subs = [];
			if (v.children && Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
			if (v.dynamicChildren) subs = subs.concat(v.dynamicChildren);
			if (v.component && v.component.subTree) subs.push(v.component.subTree);
			subs.forEach(function(s){ walkV(s, d+1); });
		}
		walkV(vnode, 0);
		if (!rp) return JSON.stringify({found: false});
		out.found = true;
		// 1) 旧树（rp.subTree）中 ConvSidebar vnode 的 props
		try {
			var csOld = null;
			function findCS(v, d) {
				if (csOld || d > 8 || !v) return;
				if (v.type && (v.type.__name || v.type.name) === 'ConvSidebar') { csOld = v; return; }
				var subs = [];
				if (v.children && Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
				if (v.dynamicChildren) subs = subs.concat(v.dynamicChildren);
				subs.forEach(function(s){ findCS(s, d+1); });
			}
			findCS(rp.subTree, 0);
			if (csOld) {
				out.oldCid = csOld.props ? String(csOld.props['current-conv-id']) : 'none';
				out.oldEl = !!csOld.el;
				out.oldKey = String(csOld.key);
			} else out.oldCid = 'not-found';
		} catch(e) { out.oldErr = String(e).slice(0,100); }
		// 2) 手动 render 出新树，看 ConvSidebar props
		try {
			var tree = rp.render(rp.proxy, rp.renderCache);
			out.renderRetType = typeof tree;
			out.renderCacheKeys = rp.renderCache ? Object.keys(rp.renderCache).join(',') : 'none';
			var csNew = null;
			function findCS2(v, d) {
				if (csNew || d > 8 || !v) return;
				if (v.type && (v.type.__name || v.type.name) === 'ConvSidebar') { csNew = v; return; }
				var subs = [];
				if (v.children && Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
				if (v.dynamicChildren) subs = subs.concat(v.dynamicChildren);
				subs.forEach(function(s){ findCS2(s, d+1); });
			}
			findCS2(tree, 0);
			if (csNew) {
				out.newCid = csNew.props ? String(csNew.props['current-conv-id']) : 'none';
				out.newFlag = csNew.patchFlag;
				out.newDynProps = csNew.dynamicProps ? csNew.dynamicProps.join(',') : 'none';
				out.newEl = !!csNew.el;
			} else out.newCid = 'not-found';
		} catch(e) { out.renderErr = String(e).slice(0,150); out.renderStack = String(e.stack||'').slice(0,500); }
		// 3) 直接调 ConvSidebar 组件实例 update（若存在）
		try {
			var csInst = null;
			function findCSI(v, d) {
				if (csInst || d > 8 || !v) return;
				if (v.component) {
					var nm2 = v.component.type && (v.component.type.name || v.component.type.__name);
					if (nm2 === 'ConvSidebar') { csInst = v.component; return; }
				}
				var subs = [];
				if (v.children && Array.isArray(v.children)) v.children.forEach(function(c){ if (c) subs.push(c); });
				if (v.dynamicChildren) subs = subs.concat(v.dynamicChildren);
				if (v.component && v.component.subTree) subs.push(v.component.subTree);
				subs.forEach(function(s){ findCSI(s, d+1); });
			}
			findCSI(vnode, 0);
			if (csInst) {
				out.csInst = true;
				out.csProps = csInst.props ? Object.keys(csInst.props).join(',') : 'none';
				out.csCid = csInst.props ? String(csInst.props['current-conv-id']) : 'n/a';
				out.csCidRaw = csInst.props && csInst.props['current-conv-id'] && csInst.props['current-conv-id'].value !== undefined ? String(csInst.props['current-conv-id'].value) : 'plain';
				try { csInst.update(); out.csUpdateOk = 'yes'; } catch(e2) { out.csUpdateOk = 'ERR:' + String(e2).slice(0,120); }
			} else out.csInst = false;
		} catch(e3) { out.csInstErr = String(e3).slice(0,100); }
		return JSON.stringify(out);
	})()`)
	fmt.Printf("[conv] convsidebar-exp: %s\n", cse.ToString())
	// flush 后读 active
	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
	}
	wv.RebuildRenderTree()
	cse2, _ := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var act = '';
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) { act = items[i].textContent.slice(0,20); break; }
		return JSON.stringify({activeAfterCSUpdate: act, n: items.length});
	})()`)
	fmt.Printf("[conv] after convsidebar update: %s\n", cse2.ToString())
	pv, _ := wv.JSInterpreter().RunJS(`JSON.stringify(window.__convProbe)`)
	fmt.Printf("[conv] probe log: %s\n", pv.ToString())
	vi, _ := wv.JSInterpreter().RunJS(`window.__veiInfo || 'none'`)
	fmt.Printf("[conv] vei info: %s\n", vi.ToString())
	mi, _ := wv.JSInterpreter().RunJS(`window.__manualInvoke || 'none'`)
	fmt.Printf("[conv] manual invoke: %s\n", mi.ToString())
	// 手动调用 invoker 后再看 active 状态 + 捕获的日志
	if wv.JSInterpreter() != nil {
		for i := 0; i < 10; i++ {
			wv.JSInterpreter().RunJobs()
			process(wv)
		}
	}
	wv.RebuildRenderTree()
	v2, _ := wv.JSInterpreter().RunJS(`(function(){
		var items = document.querySelectorAll('.conv-item');
		var act = '';
		for (var i=0;i<items.length;i++) if (items[i].className.indexOf('active')>=0) { act = items[i].textContent.slice(0,20); break; }
		return JSON.stringify({activeTitle: act});
	})()`)
	fmt.Printf("[conv] after manual invoke: %s\n", v2.ToString())
	cl, _ := wv.JSInterpreter().RunJS(`JSON.stringify(window.__capturedLogs || [])`)
	fmt.Printf("[conv] captured logs: %s\n", cl.ToString())
	// ★ 点击后组件树状态（绕过 app._instance=null，从 #app._vnode 遍历）
	vs, _ := wv.JSInterpreter().RunJS(`(function(){
		var el = document.querySelector('#app');
		var vnode = el._vnode;
		var found = null;
		var foundTag = '';
		function walkV(v, d, tag) {
			if (found || d > 10 || !v) return;
			if (v.component) {
				var st = v.component.setupState;
				if (st && st.state && st.state.currentConvId !== undefined) { found = v.component; foundTag = tag; return; }
			}
			var subs = [];
			if (v.children && Array.isArray(v.children)) { v.children.forEach(function(c){ if (c) subs.push(c); }); }
			if (v.dynamicChildren) { subs = subs.concat(v.dynamicChildren); }
			if (v.component && v.component.subTree) subs.push(v.component.subTree);
			subs.forEach(function(s){ walkV(s, d+1, tag + '.' + (s.type && (s.type.name || s.type.__name || '?'))); });
		}
		walkV(vnode, 0, 'root');
		if (!found) return JSON.stringify({found: false});
		var st = found.setupState.state;
		return JSON.stringify({
			found: true,
			comp: foundTag,
			currentConvId: st.currentConvId,
			convs: (st.conversations||[]).length,
			convsFirst: st.conversations && st.conversations[0] && st.conversations[0].id,
			convsSecond: st.conversations && st.conversations[1] && st.conversations[1].id,
			msgLen: (st.messages||[]).length,
		});
	})()`)
	fmt.Printf("[conv] state after: %s\n", vs.ToString())
	pt, _ := wv.JSInterpreter().RunJS(`window.__proxyTest || 'none'`)
	fmt.Printf("[conv] proxy test: %s\n", pt.ToString())
	va, _ := wv.JSInterpreter().RunJS(`window.__vueAppInfo || 'none'`)
	fmt.Printf("[conv] vue app info: %s\n", va.ToString())
	// ★ 决定性实验：手动修改 JS DOM className，看 wb-ui 渲染是否反映
	mc, _ := wv.JSInterpreter().RunJS(`(function(){
		var el = document.querySelectorAll('.conv-item')[1];
		el.className = 'conv-item active';
		return el.className;
	})()`)
	fmt.Printf("[conv] manual class set: %s\n", mc.ToString())
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	mc2, _ := wv.JSInterpreter().RunJS(`document.querySelectorAll('.conv-item')[1].className`)
	fmt.Printf("[conv] manual class read-back: %s\n", mc2.ToString())
	// 从渲染树读回 active 状态
	var rbItems []convItem
	var findConvRB func(o rendering.RenderObject)
	findConvRB = func(o rendering.RenderObject) {
		if el, ok := o.Node().(*dom.Element); ok {
			cn := el.GetAttribute("class")
			lb := o.LayoutBox()
			if strings.Contains(cn, "conv-item") && lb != nil && rv.LayoutState() != nil {
				rbItems = append(rbItems, convItem{el: el, active: strings.Contains(cn, "active")})
			}
		}
		for c := o.FirstChild(); c != nil; c = c.NextSibling() {
			findConvRB(c)
		}
	}
	findConvRB(rendering.RenderObject(rv))
	for i, it := range rbItems[:min(3, len(rbItems))] {
		fmt.Printf("[conv]   RB#%d active=%v class=%q\n", i, it.active, it.el.GetAttribute("class"))
	}
	// ★ 决定性证据 dump：mtLog 关键条目（firstBad / UPDATE / 错误）+ errLog
	mld, _ := wv.JSInterpreter().RunJS(`(function(){
		var ml = window.__mtLog || [];
		var sel = [];
		for (var i=0;i<ml.length;i++) {
			var e = ml[i] || {};
			var t = String(e.tag || '');
			if (t.indexOf('firstBad')>=0 || t.indexOf('UPDATE')>=0 || t.indexOf('ERR')>=0 || t.indexOf('WRAP')>=0) {
				sel.push({i:i, tag:t, badElCount:e.badElCount, err:e.err, msg:e.msg, stk:String(e.stack||'').slice(0,260)});
			}
		}
		var fb = null;
		for (var i=0;i<ml.length;i++) { if (ml[i] && ml[i].firstBad) { fb = ml[i].firstBad; break; } }
		// hookLog 关键条目（render 对齐/badEls/orphFields）
		var hl = window.__hookLog || [];
		var hsel = [];
		for (var i=0;i<hl.length;i++) {
			var h = hl[i] || {};
			var ht = String(h.tag || '');
			if (ht.indexOf('dynAlign')>=0 || ht.indexOf('badEls')>=0 || ht.indexOf('orph')>=0 || ht.indexOf('patch')>=0 || ht.indexOf('mount')>=0) {
				hsel.push({i:i, tag:ht, msg:String(h.msg||'').slice(0,220)});
			}
		}
		return JSON.stringify({mtN:ml.length, sel:sel.slice(0,50), firstBad:fb, errs:(window.__errLog||[]).slice(0,12), hookSel:hsel.slice(0,30), hookN:hl.length});
	})()`)
	fmt.Printf("[conv] mtlog-summary: %s\n", mld.ToString())
	// ★ 点击后手动修改 DOM className 再读回（验证 wb-ui 是否保留手动改动）
	mc3, _ := wv.JSInterpreter().RunJS(`(function(){
		var el = document.querySelectorAll('.conv-item')[1];
		el.className = 'conv-item active';
		return el.className;
	})()`)
	fmt.Printf("[conv] manual class set2: %s\n", mc3.ToString())
	wv.RebuildRenderTree()
	wv.EnsureLayout()
	mc4, _ := wv.JSInterpreter().RunJS(`document.querySelectorAll('.conv-item')[1].className`)
	fmt.Printf("[conv] manual class read-back2: %s\n", mc4.ToString())
}

func idx(items []convItem, it convItem) int {
	for i := range items {
		if items[i].el == it.el {
			return i
		}
	}
	return -1
}

type convItem struct {
	el     *dom.Element
	x, y   float64
	w, h   float64
	active bool
}

func process(wv *webkit.WebView) {
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.MarkRenderTreeDirty()
		}
	}
}

func setupLoaders(wv *webkit.WebView, distDir string) {
	absDist, _ := filepath.Abs(distDir)
	if mf := wv.MainFrame(); mf != nil {
		if fr := mf.Frame(); fr != nil {
			fr.ScriptLoader = func(src string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(src, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				return string(data), err
			}
			fr.StyleSheetLoader = func(href string) (string, error) {
				clean := strings.TrimPrefix(strings.TrimPrefix(href, "file://"), "./")
				data, err := os.ReadFile(filepath.Join(absDist, clean))
				if err != nil {
					return "", err
				}
				return string(data), nil
			}
		}
	}
}

// silence unused import if jsc not needed
var _ = jsc.Interpreter{}
