package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// evalLike 模拟 evalJSPlugin 的求值方式（async 包 + __resolve 回传）。
func evalLike(t *testing.T, js string) (*goja.Object, error) {
	t.Helper()
	vm := goja.New()
	var resolved *goja.Object
	var resolveErr error
	done := make(chan struct{})
	vm.Set("__resolve", func(call goja.FunctionCall) goja.Value {
		defer close(done)
		v := call.Argument(0)
		if o, ok := v.(*goja.Object); ok {
			resolved = o
		} else {
			resolveErr = fmt.Errorf("插件求值未返回对象")
		}
		return goja.Undefined()
	})
	src := "(async () => {\n" + js + "\n})().then(__resolve, e => __resolve({ __error: String(e) }))"
	if _, err := vm.RunString(src); err != nil {
		return nil, err
	}
	<-done
	if resolveErr != nil {
		return nil, resolveErr
	}
	return resolved, nil
}

// 验证内置库（path/events/util）bundle 后可被 goja 执行。
func TestBuiltinLibBundle(t *testing.T) {
	src := `
import path from 'path';
import { EventEmitter } from 'events';
import { format } from 'util';
export default {
  name: 'builtin-lib-test',
  apply(ctx) {
    const j = path.join('a', 'b', 'c.txt');
    const ee = new EventEmitter();
    let got = '';
    ee.on('x', (v) => { got = v; });
    ee.emit('x', 'ok');
    const f = format('%s-%d', 'v', 3);
    return { j, got, f, base: path.basename('/x/y/z.md'), ext: path.extname('a.tar.gz') };
  }
};
`
	js, err := compilePluginSource(src, "js", "builtin-test.ts", "")
	if err != nil {
		t.Fatalf("bundle 失败: %v", err)
	}
	res, err := evalLike(t, js)
	if err != nil {
		t.Fatalf("goja 执行失败: %v", err)
	}
	if e := res.Get("__error"); e != nil && !goja.IsUndefined(e) && e.String() != "" {
		t.Fatalf("求值失败: %s", e.String())
	}
	apply := res.Get("apply")
	if apply == nil || goja.IsUndefined(apply) {
		t.Fatalf("apply 缺失")
	}
	fn, ok := goja.AssertFunction(apply)
	if !ok {
		t.Fatal("apply 不是函数")
	}
	ctx := vmGet(res).NewObject()
	out, err := fn(goja.Undefined(), ctx)
	if err != nil {
		t.Fatalf("apply 执行失败: %v", err)
	}
	o := out.ToObject(vmGet(res))
	for k, want := range map[string]string{
		"j": "a/b/c.txt", "got": "ok", "f": "v-3",
		"base": "z.md", "ext": ".gz",
	} {
		if o.Get(k).String() != want {
			t.Errorf("%s 错误: got=%v want=%s", k, o.Get(k).String(), want)
		}
	}
}

// vmGet 从 __resolve 的对象拿 runtime（对象自带）。
func vmGet(o *goja.Object) *goja.Runtime { return o.Runtime() }

// 验证相对多文件 import + 内置库混合。
func TestBuiltinLibRelativeImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "helper.js"), []byte("module.exports = { double: (n) => n * 2 };"), 0644); err != nil {
		t.Fatal(err)
	}
	src := `
import path from 'path';
import { double } from './helper.js';
export default {
  name: 'multi-test',
  apply(ctx) {
    return { v: double(21), j: path.join('x', 'y') };
  }
};
`
	js, err := compilePluginSource(src, "js", "multi-test.ts", dir)
	if err != nil {
		t.Fatalf("bundle 失败: %v", err)
	}
	if !strings.Contains(js, "double") {
		t.Fatal("相对导入未内联")
	}
	res, err := evalLike(t, js)
	if err != nil {
		t.Fatalf("goja 执行失败: %v", err)
	}
	if e := res.Get("__error"); e != nil && !goja.IsUndefined(e) && e.String() != "" {
		t.Fatalf("求值失败: %s", e.String())
	}
	fn, ok := goja.AssertFunction(res.Get("apply"))
	if !ok {
		t.Fatal("apply 缺失")
	}
	out, err := fn(goja.Undefined(), vmGet(res).NewObject())
	if err != nil {
		t.Fatalf("apply 失败: %v", err)
	}
	o := out.ToObject(vmGet(res))
	if o.Get("v").String() != "42" {
		t.Errorf("相对导入结果错误: %v", o.Get("v").String())
	}
	if o.Get("j").String() != "x/y" {
		t.Errorf("path.join 错误: %v", o.Get("j").String())
	}
}

// 验证 locateNodeModule 探测。
func TestLocateNodeModule(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "fake-lib")
	if err := os.MkdirAll(nm, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "package.json"), []byte(`{"name":"fake-lib","main":"index.js"}`), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "a", "b")
	os.MkdirAll(sub, 0755)
	if _, ok := locateNodeModule("fake-lib", sub); !ok {
		t.Fatal("未向上探测到 node_modules")
	}
	if _, ok := locateNodeModule("no-such-pkg", sub); ok {
		t.Fatal("幽灵包不应命中")
	}
	if _, ok := locateNodeModule("@scope/nope", sub); ok {
		t.Fatal("作用域包不应命中")
	}
}
