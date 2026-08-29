package agent

// ═══════════════════════════════════════════════════════════════
// nodeapi_mini.go — 插件沙箱 Node.js 兼容 API 子集（候选 A，2026-08-29）
//
// 背景：goja 沙箱纪律对齐 harness（无 require/网络/定时器）。候选 A 此前
// 降级 P2 的理由是「无需求」——本轮按「创造需求」原则落地：仓库自有插件
// 存在真实痛点（手工 d+'/'+name 拼路径、无 base64/hex 编解码、单文件
// 无法拆 helper 模块），mini 层直接解决。
//
// 提供的 API（原创实现，语义参考 Node.js 与 goja_nodejs（MIT），未复制
// 其代码；见 THIRD_PARTY_NOTICES.md）：
//   - require(id)：内置 mini 模块 fs/path/buffer/events/util + 相对文件模块
//     （./x.js，相对插件根解析，单文件 CommonJS，结果缓存）
//   - fs（同步）：readFileSync/writeFileSync/existsSync/readdirSync/statSync/
//     mkdirSync/unlinkSync/rmSync —— ★ 全部路径经 root 受限（越界抛错）
//   - path：join/resolve/basename/dirname/extname/relative/isAbsolute/
//     normalize/sep/delimiter —— 输出统一 "/" 分隔（Node 语义）
//   - Buffer（全局 + require('buffer')）：from/alloc/isBuffer/toString
//     （utf8/base64/hex）/length —— 编解码主力
//   - events：EventEmitter（on/once/emit/off/removeAllListeners）
//   - util：format（%s/%d/%j/%o）
//
// 沙箱纪律保持：网络/定时器仍走 ctx 服务（trap 保留）；process 保持只读
// shim；fs 越界抛错（不静默放行）。
// ═══════════════════════════════════════════════════════════════

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	goja "github.com/hoonfeng/paircode/goja"
)

// installNodeAPIMini 在插件沙箱安装 mini Node API（require + 内置模块）。
// root 为 fs 受限根（空 → 当前工作目录）。在 Node API trap 循环之后调用
// （覆盖 require 的 trap 条目）。
func installNodeAPIMini(vm *goja.Runtime, root string) {
	if root == "" {
		root, _ = os.Getwd()
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	m := &miniNodeModules{vm: vm, root: root, cache: map[string]*goja.Object{}}

	vm.Set("require", func(call goja.FunctionCall) goja.Value {
		id := call.Argument(0).String()
		mod, err := m.require(id)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return mod
	})
	// Buffer 全局（与 require('buffer') 同一实现）
	bufMod := m.module("buffer", m.buildBuffer)
	if b := bufMod.Get("Buffer"); b != nil {
		vm.Set("Buffer", b)
	}
	// process.argv 补充（只读占位）
	if pv := vm.Get("process"); pv != nil {
		if po := pv.ToObject(vm); po != nil {
			po.Set("argv", vm.ToValue([]string{"node"}))
		}
	}
}

// miniNodeModules mini 模块注册表（内置模块 + 相对文件模块缓存）。
type miniNodeModules struct {
	vm    *goja.Runtime
	root  string
	mu    sync.Mutex
	cache map[string]*goja.Object
}

func (m *miniNodeModules) module(name string, build func(*goja.Object)) *goja.Object {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cached, ok := m.cache[name]; ok {
		return cached
	}
	exports := m.vm.NewObject()
	build(exports)
	m.cache[name] = exports
	return exports
}

// require 解析模块：内置 mini 模块或相对文件模块（./x.js，相对 root 解析）。
func (m *miniNodeModules) require(id string) (goja.Value, error) {
	switch id {
	case "fs":
		return m.module("fs", m.buildFS), nil
	case "path":
		return m.module("path", m.buildPath), nil
	case "buffer":
		return m.module("buffer", m.buildBuffer), nil
	case "events":
		return m.module("events", m.buildEvents), nil
	case "util":
		return m.module("util", m.buildUtil), nil
	}
	if strings.HasPrefix(id, "./") || strings.HasPrefix(id, "../") || filepath.IsAbs(id) {
		return m.loadFileModule(id)
	}
	return nil, fmt.Errorf("require: 模块 %q 不可用——沙箱仅支持内置 mini 模块（fs/path/buffer/events/util）与相对文件模块（./x.js）；npm 依赖请走 Node 桥插件", id)
}

// loadFileModule 加载相对文件模块（单文件 CommonJS：module/exports/require 闭包，
// 结果缓存；路径经 fsPath 受限）。
func (m *miniNodeModules) loadFileModule(id string) (goja.Value, error) {
	full, err := m.fsPath(id)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(full, ".js") {
		full += ".js"
	}
	src, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("require: 加载 %s 失败: %v", id, err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cached, ok := m.cache[full]; ok {
		return cached, nil
	}
	exports := m.vm.NewObject()
	moduleObj := m.vm.NewObject()
	moduleObj.Set("exports", exports)
	requireVal := m.vm.Get("require")
	wrapper := "(function(module, exports, require) {\n" + string(src) + "\n})"
	fnVal, err := m.vm.RunString(wrapper)
	if err != nil {
		return nil, fmt.Errorf("require: %s 编译失败: %v", id, err)
	}
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return nil, fmt.Errorf("require: %s 不是可执行模块", id)
	}
	if _, err := fn(goja.Undefined(), moduleObj, exports, requireVal); err != nil {
		return nil, fmt.Errorf("require: %s 执行失败: %v", id, err)
	}
	// ★ module.exports 可能被整体替换（CommonJS 语义）——以最终值为准
	final := moduleObj.Get("exports")
	if final == nil || final == goja.Undefined() {
		final = exports
	}
	m.cache[full] = final.ToObject(m.vm)
	return final, nil
}

// fsPath 路径受限解析：绝对/相对路径统一归一到 root 内，越界抛错。
func (m *miniNodeModules) fsPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("fs: 路径为空")
	}
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(m.root, full)
	}
	full = filepath.Clean(full)
	rel, err := filepath.Rel(m.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("fs: 路径 %q 超出工作区根 %s（沙箱限制，请使用相对路径）", p, m.root)
	}
	return full, nil
}

// ── fs（同步，root 受限）─────────────────────────────────────

func (m *miniNodeModules) buildFS(ex *goja.Object) {
	v := m.vm
	read := func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		data, err := os.ReadFile(full)
		if err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.readFileSync: %v", err)))
		}
		enc := "utf8"
		if len(call.Arguments) > 1 && call.Argument(1) != goja.Undefined() && call.Argument(1) != goja.Null() {
			enc = call.Argument(1).String()
		}
		switch enc {
		case "base64":
			return v.ToValue(base64.StdEncoding.EncodeToString(data))
		case "hex":
			return v.ToValue(hex.EncodeToString(data))
		default:
			return v.ToValue(string(data))
		}
	}
	ex.Set("readFileSync", read)
	ex.Set("writeFileSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		data := []byte(call.Argument(1).String())
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.writeFileSync: %v", err)))
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.writeFileSync: %v", err)))
		}
		return goja.Undefined()
	})
	ex.Set("existsSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			return v.ToValue(false)
		}
		_, statErr := os.Stat(full)
		return v.ToValue(statErr == nil)
	})
	ex.Set("readdirSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		names, err := os.ReadDir(full)
		if err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.readdirSync: %v", err)))
		}
		out := make([]string, 0, len(names))
		for _, e := range names {
			out = append(out, e.Name())
		}
		return v.ToValue(out)
	})
	ex.Set("statSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		st, err := os.Stat(full)
		if err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.statSync: %v", err)))
		}
		o := v.NewObject()
		o.Set("size", st.Size())
		o.Set("mtimeMs", float64(st.ModTime().UnixNano())/1e6)
		o.Set("isFile", st.Mode().IsRegular())
		o.Set("isDirectory", st.IsDir())
		return o
	})
	ex.Set("mkdirSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		if err := os.MkdirAll(full, 0o755); err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.mkdirSync: %v", err)))
		}
		return goja.Undefined()
	})
	ex.Set("unlinkSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		if err := os.Remove(full); err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.unlinkSync: %v", err)))
		}
		return goja.Undefined()
	})
	ex.Set("rmSync", func(call goja.FunctionCall) goja.Value {
		full, err := m.fsPath(call.Argument(0).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		if err := os.RemoveAll(full); err != nil {
			panic(v.NewGoError(fmt.Errorf("fs.rmSync: %v", err)))
		}
		return goja.Undefined()
	})
}

// ── path（输出统一 "/" 分隔）─────────────────────────────────

func (m *miniNodeModules) buildPath(ex *goja.Object) {
	v := m.vm
	slash := func(p string) string { return filepath.ToSlash(p) }
	ex.Set("sep", "/")
	ex.Set("delimiter", string(os.PathListSeparator))
	ex.Set("join", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			if a == goja.Undefined() || a == goja.Null() {
				continue
			}
			parts = append(parts, a.String())
		}
		return v.ToValue(slash(filepath.Join(parts...)))
	})
	ex.Set("resolve", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, a := range call.Arguments {
			parts = append(parts, a.String())
		}
		base := m.root
		if len(parts) > 0 && filepath.IsAbs(parts[0]) {
			base = ""
		}
		return v.ToValue(slash(filepath.Clean(filepath.Join(append([]string{base}, parts...)...))))
	})
	ex.Set("basename", func(call goja.FunctionCall) goja.Value {
		p := call.Argument(0).String()
		ext := ""
		if len(call.Arguments) > 1 {
			ext = call.Argument(1).String()
		}
		base := filepath.Base(p)
		if ext != "" && strings.HasSuffix(base, ext) {
			base = strings.TrimSuffix(base, ext)
		}
		return v.ToValue(base)
	})
	ex.Set("dirname", func(call goja.FunctionCall) goja.Value {
		return v.ToValue(slash(filepath.Dir(call.Argument(0).String())))
	})
	ex.Set("extname", func(call goja.FunctionCall) goja.Value {
		return v.ToValue(filepath.Ext(call.Argument(0).String()))
	})
	ex.Set("relative", func(call goja.FunctionCall) goja.Value {
		rel, err := filepath.Rel(call.Argument(0).String(), call.Argument(1).String())
		if err != nil {
			panic(v.NewGoError(err))
		}
		return v.ToValue(slash(rel))
	})
	ex.Set("isAbsolute", func(call goja.FunctionCall) goja.Value {
		return v.ToValue(filepath.IsAbs(call.Argument(0).String()))
	})
	ex.Set("normalize", func(call goja.FunctionCall) goja.Value {
		return v.ToValue(slash(filepath.Clean(call.Argument(0).String())))
	})
}

// ── buffer（Buffer 编解码：utf8/base64/hex）───────────────────

func (m *miniNodeModules) makeBuffer(data []byte) *goja.Object {
	v := m.vm
	o := v.NewObject()
	o.Set("length", len(data))
	o.Set("__miniBuffer", true)
	o.Set("toString", func(call goja.FunctionCall) goja.Value {
		enc := "utf8"
		if len(call.Arguments) > 0 && call.Argument(0) != goja.Undefined() && call.Argument(0) != goja.Null() {
			enc = call.Argument(0).String()
		}
		switch enc {
		case "base64":
			return v.ToValue(base64.StdEncoding.EncodeToString(data))
		case "hex":
			return v.ToValue(hex.EncodeToString(data))
		default:
			return v.ToValue(string(data))
		}
	})
	return o
}

func (m *miniNodeModules) buildBuffer(ex *goja.Object) {
	v := m.vm
	buf := v.NewObject()
	buf.Set("from", func(call goja.FunctionCall) goja.Value {
		arg := call.Argument(0)
		enc := "utf8"
		if len(call.Arguments) > 1 && call.Argument(1) != goja.Undefined() && call.Argument(1) != goja.Null() {
			enc = call.Argument(1).String()
		}
		var data []byte
		switch enc {
		case "base64":
			var err error
			data, err = base64.StdEncoding.DecodeString(arg.String())
			if err != nil {
				panic(v.NewGoError(fmt.Errorf("Buffer.from(base64): %v", err)))
			}
		case "hex":
			var err error
			data, err = hex.DecodeString(arg.String())
			if err != nil {
				panic(v.NewGoError(fmt.Errorf("Buffer.from(hex): %v", err)))
			}
		default:
			data = []byte(arg.String())
		}
		return m.makeBuffer(data)
	})
	buf.Set("alloc", func(call goja.FunctionCall) goja.Value {
		n := int(call.Argument(0).ToInteger())
		if n < 0 {
			panic(v.NewGoError(fmt.Errorf("Buffer.alloc: 长度不能为负")))
		}
		return m.makeBuffer(make([]byte, n))
	})
	buf.Set("isBuffer", func(call goja.FunctionCall) goja.Value {
		if call.Argument(0) == goja.Undefined() || call.Argument(0) == goja.Null() {
			return v.ToValue(false)
		}
		o := call.Argument(0).ToObject(v)
		return v.ToValue(o.Get("__miniBuffer") != nil && o.Get("__miniBuffer").ToBoolean())
	})
	buf.Set("concat", func(call goja.FunctionCall) goja.Value {
		// 简化：数组元素按 utf8 拼接（全量二进制 concat 属超集需求，见文档）
		var out []byte
		if call.Argument(0) == goja.Undefined() {
			return m.makeBuffer(nil)
		}
		arr := call.Argument(0).Export().([]any)
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, []byte(s)...)
			}
		}
		return m.makeBuffer(out)
	})
	ex.Set("Buffer", buf)
}

// ── events（EventEmitter mini）───────────────────────────────

func (m *miniNodeModules) buildEvents(ex *goja.Object) {
	v := m.vm
	// JS class 实现（goja 对 Go 函数不支持 new 构造；JS class 语义完整且简单）
	src := `
(() => {
  class EventEmitter {
    constructor() { this._h = Object.create(null); }
    on(n, f) { (this._h[n] = this._h[n] || []).push(f); return this; }
    once(n, f) { const g = (...a) => { this.off(n, g); f(...a); }; return this.on(n, g); }
    emit(n, ...a) { const l = this._h[n] || []; for (const f of l) f(...a); return l.length > 0; }
    off(n) { delete this._h[n]; return this; }
    removeAllListeners(n) { if (n) { delete this._h[n]; } else { this._h = Object.create(null); } return this; }
  }
  return EventEmitter;
})()`
	if val, err := v.RunString(src); err == nil {
		ex.Set("EventEmitter", val)
	} else {
		log.Printf("[nodeapi-mini] EventEmitter 定义失败: %v", err)
		ex.Set("EventEmitter", goja.Undefined())
	}
}

// ── util（format：%s/%d/%j/%o/%%）────────────────────────────

func (m *miniNodeModules) buildUtil(ex *goja.Object) {
	v := m.vm
	ex.Set("format", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return v.ToValue("")
		}
		format := call.Argument(0).String()
		args := call.Arguments[1:]
		ai := 0
		var sb strings.Builder
		for i := 0; i < len(format); i++ {
			c := format[i]
			if c != '%' || i+1 >= len(format) {
				sb.WriteByte(c)
				continue
			}
			i++
			switch format[i] {
			case 's':
				if ai < len(args) {
					sb.WriteString(args[ai].String())
				}
				ai++
			case 'd':
				if ai < len(args) {
					sb.WriteString(fmt.Sprintf("%d", args[ai].ToInteger()))
				}
				ai++
			case 'j', 'o':
				if ai < len(args) {
					b, _ := json.Marshal(args[ai].Export())
					sb.Write(b)
				}
				ai++
			case '%':
				sb.WriteByte('%')
			default:
				sb.WriteByte('%')
				sb.WriteByte(format[i])
			}
		}
		return v.ToValue(sb.String())
	})
}
