package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hoonfeng/paircode/goja"
)

// TestExtractImportNames 命名导入提取。
func TestExtractImportNames(t *testing.T) {
	dir := t.TempDir()
	src := `
import { createConnection } from '@cordisjs/core'
import type { Config } from 'schemastery'
import * as ns from 'cordis'
import d from 'my-pkg'
import x, { y as z } from 'other-pkg'
import { a, b as c } from '@cordisjs/plugin-http'
`
	f := filepath.Join(dir, "src.ts")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]string{
		"@cordisjs/core":        {"createConnection"},
		"schemastery":           nil, // type-only import 自动擦除，不需要 mock
		"cordis":                {"ns"},
		"my-pkg":                {"d"},
		"other-pkg":             {"x", "y"},
		"@cordisjs/plugin-http": {"a", "b"},
	}
	for pkg, want := range cases {
		got := extractImportNames(f, pkg)
		if len(got) != len(want) {
			t.Errorf("pkg %s: 期望 %v 实际 %v", pkg, want, got)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("pkg %s: 期望 %v 实际 %v", pkg, want, got)
				break
			}
		}
	}
}

// TestCompileNPMPluginStyle 模拟 npm cordis 插件（ESM + 命名导入）编译通过。
func TestCompileNPMPluginStyle(t *testing.T) {
	dir := t.TempDir()
	src := `
import { createConnection } from '@cordisjs/core'
import { Context } from 'cordis'

export default function (ctx) {
  ctx.on('ready', () => {})
  // 实际使用命名导入（触发 mock 命名导出，运行期 undefined 不抛编译错）
  return { name: 'demo', conn: createConnection }
}
`
	js, err := compilePluginSource(src, "js", "cordis-dyn.ts", dir)
	if err != nil {
		t.Fatalf("编译失败: %v", err)
	}
	// 语法校验（goja 可解析）
	if _, err := goja.Compile("cordis-dyn.js", "(async () => {\n"+js+"\n})()", false); err != nil {
		t.Fatalf("goja 语法错误: %v", err)
	}
	// 命名导入被引用时 mock 导出保留（createConnection 出现在输出中）
	if !strings.Contains(js, "createConnection") {
		t.Fatalf("mock 未导出 createConnection？输出: %.300s", js)
	}
}
