package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goja "wb-ui/goja"
)

// TestNodeAPIMiniFS mini fs：工作区根内读写 + 越界拒绝。
func TestNodeAPIMiniFS(t *testing.T) {
	root := t.TempDir()
	vm := goja.New()
	installNodeAPIMini(vm, root)

	code := `
const fs = require('fs');
fs.writeFileSync('a.txt', 'hello');
fs.mkdirSync('sub/deep');
fs.writeFileSync('sub/deep/b.txt', 'world');
const s = fs.readFileSync('a.txt', 'utf8');
const b64 = fs.readFileSync('sub/deep/b.txt', 'base64');
const exists = fs.existsSync('sub/deep/b.txt');
const names = fs.readdirSync('sub');
const st = fs.statSync('a.txt');
globalThis.__r = JSON.stringify({s, b64, exists, names, isFile: st.isFile, size: st.size});
`
	if _, err := vm.RunString(code); err != nil {
		t.Fatalf("mini fs 执行失败: %v", err)
	}
	r := vm.Get("__r").String()
	for _, want := range []string{`"s":"hello"`, `"b64":"d29ybGQ="`, `"exists":true`, `"isFile":true`, `"size":5`} {
		if !strings.Contains(r, want) {
			t.Fatalf("结果缺少 %s: %s", want, r)
		}
	}

	// 越界拒绝（相对 ../ 逃逸 + Windows 绝对路径逃逸）
	for _, bad := range []string{
		`require('fs').readFileSync('../../secret.txt')`,
		`require('fs').writeFileSync('Z:\\out-of-root\\evil.txt', 'x')`,
	} {
		if _, err := vm.RunString(bad); err == nil {
			t.Fatalf("越界应报错: %s", bad)
		} else if !strings.Contains(err.Error(), "超出工作区根") {
			t.Fatalf("越界错误信息不符: %v", err)
		}
	}
}

// TestNodeAPIMiniPathAndBuffer path join 统一 "/" + Buffer 编解码。
func TestNodeAPIMiniPathAndBuffer(t *testing.T) {
	vm := goja.New()
	installNodeAPIMini(vm, t.TempDir())
	code := `
const path = require('path');
const j = path.join('a', 'b', 'c.txt');
const ext = path.extname('x.tar.gz');
const base = path.basename('/a/b/x.js', '.js');
const b1 = Buffer.from('aGk=', 'base64').toString();
const b2 = Buffer.from('hello').toString('base64');
const b3 = Buffer.from('ff00', 'hex').length;
const isBuf = Buffer.isBuffer(Buffer.alloc(3));
const f = require('util').format('%s:%d', 'x', 3);
const e = require('events');
const em = new e.EventEmitter();
let got = '';
em.on('ping', (v) => { got = v; });
em.emit('ping', 'pong');
globalThis.__r = JSON.stringify({j, ext, base, b1, b2, b3, isBuf, f, got});
`
	if _, err := vm.RunString(code); err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	r := vm.Get("__r").String()
	for _, want := range []string{`"j":"a/b/c.txt"`, `"ext":".gz"`, `"base":"x"`, `"b1":"hi"`, `"b2":"aGVsbG8="`, `"b3":2`, `"isBuf":true`, `"f":"x:3"`, `"got":"pong"`} {
		if !strings.Contains(r, want) {
			t.Fatalf("结果缺少 %s: %s", want, r)
		}
	}
}

// TestNodeAPIMiniRelativeRequire 相对文件模块（单文件 CommonJS，缓存）。
func TestNodeAPIMiniRelativeRequire(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "helper.js"), []byte(`
module.exports = { double: (n) => n * 2, tag: 'helper' };
`), 0o644); err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	installNodeAPIMini(vm, root)
	if _, err := vm.RunString(`
const h = require('./helper.js');
globalThis.__r = JSON.stringify({ tag: h.tag, double: h.double(21) });
`); err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	r := vm.Get("__r").String()
	if !strings.Contains(r, `"double":42`) || !strings.Contains(r, `"tag":"helper"`) {
		t.Fatalf("相对模块结果不符: %s", r)
	}
}

// TestNodeAPIMiniInSandbox 真实沙箱（LoadCordisPatch 走 newJSSandbox）：
// 插件代码使用 require('fs')/require('path') 落地工作区读写。
func TestNodeAPIMiniInSandbox(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	// mini fs 受限根 = npmPluginProjectRoot()（无工作区时 = cwd）；插件在
	// 沙箱内用相对路径写插件自带目录（通过 path.join 与 __dirname 无关，
	// 这里直接验证 require 面可用 + fs 相对根可用）。
	content := `{
  "plugins": [
    {
      "purpose": "mini node api 沙箱验证",
      "code": "return { name: 'mini-nodeapi-test', apply(ctx, config) { const fs = require('fs'); const path = require('path'); const exists = fs.existsSync('jsplugin.go'); const sep = path.sep; ctx.tools.register({ name: 'mini_probe', description: 'probe', execute: () => ({ text: exists + '|' + sep }) }) } }"
    }
  ]
}`
	patch := filepath.Join(dir, "cordis.patch.json")
	if err := os.WriteFile(patch, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := host.LoadCordisPatch(patch); err != nil {
		t.Fatalf("LoadCordisPatch: %v", err)
	}
	if host.State("mini-nodeapi-test") != PluginRunning {
		t.Fatalf("插件应 running")
	}
	out, err := reg.Execute(context.Background(), "mini_probe", `{}`)
	if err != nil {
		t.Fatalf("mini_probe: %v", err)
	}
	if !strings.HasPrefix(out, "true|") {
		t.Fatalf("mini_probe 应读到 jsplugin.go（cwd=internal/agent）: %q", out)
	}
}
