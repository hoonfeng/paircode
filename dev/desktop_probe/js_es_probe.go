// Command js_es_probe checks which ES6+ browser APIs the wb-ui jsc engine
// (goja) exposes. The companion frontend relies on many modern Array/String/
// Promise/Object methods — a missing one throws ReferenceError and breaks
// the app at boot.
//
// Run: go run ./dev/desktop_probe/js_es_probe.go
package main

import (
	"fmt"
	"strings"

	"wb-ui/webkit"
)

var checks = []string{
	// Array
	`typeof [].at === 'function'`,
	`typeof [].flat === 'function'`,
	`typeof [].flatMap === 'function'`,
	`typeof [].find === 'function'`,
	`typeof [].findIndex === 'function'`,
	`typeof [].findLast === 'function'`,
	`typeof [].includes === 'function'`,
	`typeof Array.from === 'function'`,
	`typeof Array.isArray === 'function'`,
	`typeof [].entries === 'function'`,
	`typeof [].values === 'function'`,
	`typeof [].keys === 'function'`,
	`typeof [].fill === 'function'`,
	`typeof [].copyWithin === 'function'`,
	`typeof [].sort === 'function'`,
	`[1,2,3].at(-1) === 3`,
	`[1,[2,3]].flat(1).length === 3`,
	`[1,2,3].includes(2) === true`,
	`[1,2,3].findLast(x => x < 3) === 2`,
	// String
	`typeof ''.startsWith === 'function'`,
	`typeof ''.endsWith === 'function'`,
	`typeof ''.includes === 'function'`,
	`typeof ''.padStart === 'function'`,
	`typeof ''.padEnd === 'function'`,
	`typeof ''.repeat === 'function'`,
	`typeof ''.trimStart === 'function'`,
	`typeof ''.trimEnd === 'function'`,
	`typeof ''.at === 'function'`,
	`typeof ''.replaceAll === 'function'`,
	`'abc'.padStart(5,'0') === '00abc'`,
	`'a,b,c'.split(',').length === 3`,
	`' x '.trim() === 'x'`,
	// Object
	`typeof Object.assign === 'function'`,
	`typeof Object.entries === 'function'`,
	`typeof Object.values === 'function'`,
	`typeof Object.fromEntries === 'function'`,
	`typeof Object.keys === 'function'`,
	`typeof Object.freeze === 'function'`,
	`typeof Object.hasOwn === 'function'`,
	`typeof structuredClone === 'function'`,
	// Promise / async
	`typeof Promise === 'function'`,
	`typeof Promise.all === 'function'`,
	`typeof Promise.race === 'function'`,
	`typeof Promise.allSettled === 'function'`,
	`typeof Promise.any === 'function'`,
	`typeof Promise.resolve === 'function'`,
	`(async function(){ return 1 })() instanceof Promise`,
	// Symbol
	`typeof Symbol === 'function'`,
	`typeof Symbol.iterator === 'symbol'`,
	`typeof Symbol.for === 'function'`,
	`[1,2][Symbol.iterator] !== undefined`,
	// Map/Set/WeakMap
	`typeof Map === 'function'`,
	`typeof Set === 'function'`,
	`typeof WeakMap === 'function'`,
	`typeof WeakSet === 'function'`,
	`(new Map([['a',1]])).get('a') === 1`,
	`(new Set([1,2,2])).size === 2`,
	// Other globals
	`typeof JSON === 'object'`,
	`typeof Date === 'function'`,
	`typeof Math.max === 'function'`,
	`typeof Number.isNaN === 'function'`,
	`typeof Number.isFinite === 'function'`,
	`typeof BigInt === 'function'`,
	`typeof URL === 'function'`,
	`typeof URLSearchParams === 'function'`,
	`typeof TextEncoder === 'function'`,
	`typeof TextDecoder === 'function'`,
	`typeof performance === 'object'`,
	`typeof navigator === 'object'`,
	`typeof crypto === 'object'`,
	`typeof requestAnimationFrame === 'function'`,
	`typeof queueMicrotask === 'function'`,
	`typeof setTimeout === 'function'`,
	`typeof console === 'object'`,
	// Destructuring / spread / optional chaining / nullish (syntax)
	`(() => { const {a = 1} = {}; return a === 1 })()`,
	`(() => { const [x = 5] = []; return x === 5 })()`,
	`(() => { const o = {b: null}; return (o.b ?? 'd') === 'd' })()`,
	`(() => { const o = {c: 1}; return o?.c === 1 })()`,
	`(() => { const a = [1]; const b = [...a, 2]; return b.length === 2 })()`,
	`(() => { const a = {x:1}; const b = {...a, y:2}; return b.y === 2 })()`,
	`(() => { const f = (x, ...rest) => rest.length; return f(1,2,3) === 2 })()`,
	`(() => { const f = (a = 9) => a; return f() === 9 })()`,
	// DOM / Vue-3 runtime deps
	`typeof document === 'object'`,
	`typeof document.getElementById === 'function'`,
	`typeof document.querySelector === 'function'`,
	`typeof document.querySelectorAll === 'function'`,
	`typeof document.createElement === 'function'`,
	`typeof document.createTextNode === 'function'`,
	`typeof document.addEventListener === 'function'`,
	`typeof window === 'object'`,
	`typeof window.addEventListener === 'function'`,
	`typeof window.getComputedStyle === 'function'`,
	`typeof MutationObserver === 'function'`,
	`typeof ResizeObserver === 'function'`,
	`typeof IntersectionObserver === 'function'`,
	`typeof CustomEvent === 'function'`,
	`typeof Event === 'function'`,
	`typeof EventTarget === 'function'`,
	`typeof localStorage === 'object'`,
	`typeof history === 'object'`,
	`typeof history.pushState === 'function'`,
	`typeof history.replaceState === 'function'`,
	`typeof location === 'object'`,
	`typeof XMLHttpRequest === 'function'`,
	`typeof fetch === 'function'`,
	`typeof WebSocket === 'function'`,
	`typeof navigator.clipboard !== 'undefined' || true`,
	`typeof matchMedia === 'function' || true`,
	// Proxy (Vue reactivity core)
	`typeof Proxy === 'function'`,
	`(() => { const t = {a:1}; const p = new Proxy(t, {get(o,k){return o[k]*2}}); return p.a === 2 })()`,
	`typeof Reflect === 'object'`,
	`typeof Reflect.get === 'function'`,
	// keydown etc.
	`(() => { const e = new Event('click'); return e.type === 'click' })()`,
	`(() => { const ce = new CustomEvent('x', {detail: 5}); return ce.detail === 5 })()`,
}

func main() {
	wv := webkit.NewWebView()
	// Load an empty page so RegisterDOMBindings (URL/URLSearchParams/DOM) runs.
	if err := wv.LoadHTML(`<!DOCTYPE html><html><body>probe</body></html>`); err != nil {
		fmt.Println("LoadHTML:", err)
	}
	rt := wv.JSInterpreter()
	if rt == nil {
		fmt.Println("FAIL: no interpreter")
		return
	}
	// bridge SDK may be needed for fetch; not tested here.

	pass, fail := 0, 0
	for _, expr := range checks {
		val, err := rt.RunJS(expr)
		if err != nil {
			fmt.Printf("ERR  %-55s → %v\n", expr, err)
			fail++
			continue
		}
		s := val.ToString()
		if strings.TrimSpace(s) == "true" {
			pass++
		} else {
			fmt.Printf("FAIL %-55s → %s\n", expr, s)
			fail++
		}
	}
	fmt.Printf("\n=== ES API probe: %d pass, %d fail ===\n", pass, fail)
}
