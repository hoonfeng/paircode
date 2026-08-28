// ═══════════════════════════════════════════════════════════
// builtin_stdlib.go — goja 插件内置依赖库注册表
//
// goja 沙箱无 Node API（无 require/process/Buffer/globalThis 的全局原语），
// 插件 `import path from 'path'` 等依赖在 esbuild bundle 时命中本表 →
// 内置 namespace（builtin-lib）直接把纯 JS 实现内联进 bundle：
//   - path（posix + win32 常用 API）
//   - events（EventEmitter mini）
//   - util（format/inspect mini）
//
// ★ npm 依赖支持：外置包（lodash/dayjs 等纯 JS）经 node_modules 解析
//   直接 bundle（tscompile.go 的 onResolve 管线）；Node API 依赖的包
//   （http/fs 等）运行期无能力 → 使用时应优先 ctx.fs/ctx.web/ctx.bash。
// ═══════════════════════════════════════════════════════════
package agent

import "strings"

// builtinLibSource 内置库源码（纯 ES2020 JS，goja 可执行；CJS 导出）。
var builtinLibSource = map[string]string{
	// ── path（mini：posix 语义 + win32 基本 API）──
	"path": `
var win32 = {
  sep: '\\', delimiter: ';',
  normalize: normalize, join: join, resolve: resolve,
  basename: basename, dirname: dirname, extname: extname,
  isAbsolute: function(p){ return /^[A-Za-z]:[\\/]/.test(p) || /^[\\/]/.test(p); },
  relative: relative,
};
function splitPath(p) {
  p = String(p || '');
  var m = /^(?:[^?]+)?([A-Za-z]:[\\/]+)?(.*)$/.exec(p);
  return m ? p : p;
}
function normalizeArray(parts) {
  var out = [];
  for (var i = 0; i < parts.length; i++) {
    var p = parts[i];
    if (!p || p === '.') continue;
    if (p === '..') {
      if (out.length && out[out.length-1] !== '..') out.pop();
      else out.push('..');
    } else out.push(p);
  }
  return out;
}
function normalize(p) {
  p = String(p || '');
  if (!p) return '.';
  var isAbs = p[0] === '/' || p[1] === ':';
  var trailing = p[p.length-1] === '/';
  var parts = normalizeArray(p.split(/[\\/]+/));
  var res = parts.join('/');
  if (isAbs) res = '/' + res;
  else if (p.slice(0,2) === '..' || p[0] === '.') res = res || '.';
  if (trailing && res.length > 1) res += '/';
  if (res === '' && !isAbs) res = '.';
  return res;
}
function join() {
  var args = Array.prototype.slice.call(arguments);
  var p = '';
  for (var i = 0; i < args.length; i++) {
    if (!args[i] && args[i] !== 0) continue;
    if (typeof args[i] === 'object') args[i] = String(args[i]);
    if (p) p += '/';
    p += String(args[i]);
  }
  return normalize(p);
}
function resolve() {
  var args = Array.prototype.slice.call(arguments);
  var res = '', abs = false;
  for (var i = args.length - 1; i >= -1 && !abs; i--) {
    var p = i >= 0 ? String(args[i] || '') : processCwd();
    if (!p) continue;
    var full = p;
    if (/^[A-Za-z]:/.test(full)) { res = full; abs = true; }
    else if (full[0] === '/' || full[0] === '\\\\') { res = full; abs = true; }
    else { res = full + '/' + res; }
  }
  return normalize(res);
}
function processCwd() { return ''; }
function basename(p, ext) {
  p = String(p || '');
  var f = p.split(/[\\/]/).pop() || '';
  if (ext && f.slice(-ext.length) === ext) f = f.slice(0, -ext.length);
  return f;
}
function dirname(p) {
  p = String(p || '');
  var i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'));
  if (i < 0) return '.';
  if (i === 0) return p[0];
  var d = p.slice(0, i);
  return d || '.';
}
function extname(p) {
  p = String(p || '');
  var b = basename(p);
  var i = b.lastIndexOf('.');
  if (i <= 0) return '';
  return b.slice(i);
}
function relative(from, to) {
  function parts(p) { return normalize(p).split('/').filter(Boolean); }
  var a = parts(from), b = parts(to), i = 0;
  while (i < a.length && i < b.length && a[i] === b[i]) i++;
  var out = a.slice(i).map(function(){ return '..'; }).concat(b.slice(i));
  return out.join('/') || '.';
}
module.exports = {
  sep: '/', delimiter: ':',
  normalize: normalize, join: join, resolve: resolve,
  basename: basename, dirname: dirname, extname: extname,
  isAbsolute: function(p){ return /^[\\/]/.test(p); },
  relative: relative,
  posix: null, win32: win32,
};
`,
	// ── events（EventEmitter mini）──
	"events": `
function EventEmitter() { this._events = {}; }
EventEmitter.prototype.on = function(type, fn) {
  if (!this._events[type]) this._events[type] = [];
  this._events[type].push(fn);
  return this;
};
EventEmitter.prototype.addListener = EventEmitter.prototype.on;
EventEmitter.prototype.once = function(type, fn) {
  var self = this;
  function w() { self.off(type, w); fn.apply(self, arguments); }
  w.fn = fn;
  return this.on(type, w);
};
EventEmitter.prototype.off = function(type, fn) {
  var list = this._events[type] || [];
  for (var i = list.length - 1; i >= 0; i--) {
    if (list[i] === fn || (list[i].fn === fn && list[i].fn)) list.splice(i, 1);
  }
  return this;
};
EventEmitter.prototype.removeListener = EventEmitter.prototype.off;
EventEmitter.prototype.removeAllListeners = function(type) {
  if (type) delete this._events[type];
  else this._events = {};
  return this;
};
EventEmitter.prototype.emit = function(type) {
  var list = (this._events[type] || []).slice();
  var args = Array.prototype.slice.call(arguments, 1);
  for (var i = 0; i < list.length; i++) {
    try { list[i].apply(this, args); } catch (e) { /* 监听器异常不中断 */ }
  }
  return true;
};
EventEmitter.prototype.listenerCount = function(type) {
  return (this._events[type] || []).length;
};
EventEmitter.prototype.listeners = function(type) { return (this._events[type] || []).slice(); };
EventEmitter.prototype.eventNames = function() { return Object.keys(this._events).filter(function(k){ return (this._events[k]||[]).length > 0; }, this); };
module.exports = { EventEmitter: EventEmitter };
`,
	// ── util（format/inherits/TextEncoder 兜底）──
	"util": `
function format(f) {
  var args = Array.prototype.slice.call(arguments, 1);
  if (typeof f !== 'string') {
    return args.length ? args.map(function(a){ return inspect(a); }).join(' ') : inspect(f);
  }
  var i = 0;
  return String(f).replace(/%[sdj%]/g, function(m) {
    if (m === '%%') return '%';
    if (i >= args.length) return m;
    var v = args[i++];
    if (m === '%s') return typeof v === 'string' ? v : String(v);
    if (m === '%d') return Number(v);
    if (m === '%j') { try { return JSON.stringify(v); } catch (e) { return '[circular]'; } }
    return m;
  });
}
function inspect(v) {
  if (v === null) return 'null';
  if (v === undefined) return 'undefined';
  if (typeof v === 'string') return JSON.stringify(v);
  if (typeof v !== 'object') return String(v);
  try { return JSON.stringify(v, null, 2); } catch (e) { return String(v); }
}
function inherits(c, p) {
  c.prototype = Object.create(p.prototype);
  c.prototype.constructor = c;
}
function isArray(a) { return Array.isArray(a); }
function isObject(o) { return o !== null && typeof o === 'object'; }
function isString(s) { return typeof s === 'string'; }
function isNumber(n) { return typeof n === 'number'; }
function isFunction(f) { return typeof f === 'function'; }
module.exports = {
  format: format, inspect: inspect, inherits: inherits,
  isArray: isArray, isObject: isObject, isString: isString,
  isNumber: isNumber, isFunction: isFunction,
};
`,
}

// builtinLibAliases 内置库别名表（npm 风格名 → 实现名）。
var builtinLibAliases = map[string]string{
	"path":       "path",
	"path/posix": "path",
	"path/win32": "path",
	"node:path":  "path",
	"events":     "events",
	"node:events": "events",
	"util":       "util",
	"node:util":  "util",
}

// builtinLibFor 查询内置库：返回（实现名, 源码, 命中）。
func builtinLibFor(name string) (string, string, bool) {
	impl, ok := builtinLibAliases[name]
	if !ok {
		return "", "", false
	}
	src, ok := builtinLibSource[impl]
	if !ok {
		return "", "", false
	}
	return impl, src, true
}

// stripNodePrefix 把 "node:fs" → "fs"（npm 包名归一化）。
func stripNodePrefix(name string) string {
	return strings.TrimPrefix(name, "node:")
}
