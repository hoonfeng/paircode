package agent

// diskplugin_jsnative_test.go — 磁盘插件 JS 原生化验证（2026-08-22）。
// 加载真实 .pair/plugins/tool-*/index.js 到 PluginHost，经 Registry.Execute
// 验证工具行为（对比原二进制实现：schema/CLI 参数组装/输出形态）。

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const jsNativeWorkspace = `F:\syproject\gou-ide`

// loadDiskPluginForTest 读取 .pair/plugins/<name>/index.js 并装载到新 host。
// 返回 host + registry（注册工具可直接 Execute）。
// diskPluginDir 定位磁盘插件包目录（★ 2026-09-17 双源）：IDE 基线插件在
// .pair/plugins/，独立发布插件（市场分发、不进 IDE 发布包）真源在 plugins-dist/。
// 优先基线（同名并存时以基线为准，与 publish-official-plugins.listPlugins 同口径），
// 其次 plugins-dist；两者都无 index.js 时返回基线路径（保留原有报错形态）。
func diskPluginDir(root, name string) string {
	baseline := filepath.Join(root, ".pair", "plugins", name)
	if _, err := os.Stat(filepath.Join(baseline, "index.js")); err == nil {
		return baseline
	}
	dist := filepath.Join(root, "plugins-dist", name)
	if _, err := os.Stat(filepath.Join(dist, "index.js")); err == nil {
		return dist
	}
	return baseline
}

func loadDiskPluginForTest(t *testing.T, name string) (*PluginHost, *Registry) {
	t.Helper()
	dir := diskPluginDir(jsNativeWorkspace, name)
	code, err := os.ReadFile(filepath.Join(dir, "index.js"))
	if err != nil {
		t.Fatalf("读取 %s/index.js: %v", name, err)
	}
	return loadJSCodeForTest(t, string(code), dir)
}

func loadJSCodeForTest(t *testing.T, code string, dirs ...string) (*PluginHost, *Registry) {
	t.Helper()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, jsNativeWorkspace)
	dir := ""
	if len(dirs) > 0 {
		dir = dirs[0]
	}
	id, err := host.DefineJSCodeFull(code, "", "js 原生化验证", dir, "")
	if err != nil {
		t.Fatalf("DefineJSCodeFull: %v", err)
	}
	if err := host.LoadJSDynamic(mustDef(t, host, id)); err != nil {
		t.Fatalf("LoadJSDynamic: %v", err)
	}
	return host, reg
}

// execJSTool 经 registry 执行工具（Json 参数）并返回文本。
func execJSTool(t *testing.T, reg *Registry, name, argsJSON string) string {
	t.Helper()
	out, err := reg.Execute(context.Background(), name, argsJSON)
	if err != nil {
		t.Fatalf("%s 执行失败: %v", name, err)
	}
	return out
}

// ★ 2026-09-12 codex 精简轮：tool-git（10 工具）已移除——git 操作统一走
// exec_command（codex 无 git 工具面；前端 Git 面板走独立 git-api 插件不受影响）。
// 原 TestToolGitJSNative / TestToolGitJSNativeMultiProject 随之删除
// （插件备份于 _temp/removed-plugins-202609/codex-slim-20260912/）。

// TestToolWebJSNative tool-web JS 原生化：注册 + web_fetch 真实抓取。
func TestToolWebJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-web")
	for _, name := range []string{"web_fetch", "web_search"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		reg.SetToolEnabled(name, true)
	}
	out := execJSTool(t, reg, "web_fetch", `{"url":"https://example.com"}`)
	if !strings.Contains(out, "HTTP 200") || len(out) < 20 {
		t.Fatalf("web_fetch 输出异常: %q", out)
	}
}

// TestToolMemoryJSNative tool-memory JS 原生化：写入→读→搜索→删除（自清理）。
// ★ 2026-09 工具面合并：5 工具合并为单工具 memory(op=…)，含 dynamicApproval
// （write/delete 需批准，read/search/list 不需要）。
func TestToolMemoryJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-memory")
	tool, ok := reg.Get("memory")
	if !ok {
		t.Fatalf("工具 memory 未注册")
	}
	reg.SetToolEnabled("memory", true)
	// dynamicApproval：write/delete 需批准；只读 op 不需要
	if tool.DynamicApproval == nil {
		t.Fatal("memory 应声明 dynamicApproval")
	}
	if !tool.DynamicApproval(ToolCall{Function: FunctionCall{Name: "memory", Arguments: `{"op":"write"}`}}) {
		t.Error("op=write 应需批准")
	}
	if tool.DynamicApproval(ToolCall{Function: FunctionCall{Name: "memory", Arguments: `{"op":"read"}`}}) {
		t.Error("op=read 不应需批准")
	}
	const name = "jsnative测试条目"
	out := execJSTool(t, reg, "memory", `{"op":"write","name":"`+name+`","type":"project","description":"JS 原生化验证条目","content":"验证 memory 工具的 JS 实现读写。","project":"gou-ide"}`)
	if !strings.Contains(out, "已新建记忆") && !strings.Contains(out, "已记忆") {
		t.Fatalf("memory(op=write) 输出异常: %q", out)
	}
	defer execJSTool(t, reg, "memory", `{"op":"delete","name":"`+name+`"}`)
	// 读回
	out = execJSTool(t, reg, "memory", `{"op":"read","name":"`+name+`"}`)
	if !strings.Contains(out, "JS 原生化验证条目") {
		t.Fatalf("memory(op=read) 输出异常: %q", out)
	}
	// 搜索
	out = execJSTool(t, reg, "memory", `{"op":"search","query":"jsnative"}`)
	if !strings.Contains(out, name) {
		t.Fatalf("memory(op=search) 未命中: %q", out)
	}
}

// TestToolProjectInfoJSNative tool-project-info JS 原生化：写入→读→树→删除（自清理）。
// ★ 2026-09 工具面合并：7 工具合并为单工具 project_info(op=…)。
func TestToolProjectInfoJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-project-info")
	if _, ok := reg.Get("project_info"); !ok {
		t.Fatalf("工具 project_info 未注册")
	}
	reg.SetToolEnabled("project_info", true)
	const p = "实现/jsnative验证"
	out := execJSTool(t, reg, "project_info", `{"op":"write","path":"`+p+`","content":"# JS 原生验证\n\n内容正文。","project":"gou-ide"}`)
	if !strings.Contains(out, "已写入知识库") && !strings.Contains(out, "已更新知识库") {
		t.Fatalf("project_info(op=write) 输出异常: %q", out)
	}
	defer execJSTool(t, reg, "project_info", `{"op":"delete","path":"`+p+`"}`)
	out = execJSTool(t, reg, "project_info", `{"op":"read","path":"`+p+`"}`)
	if !strings.Contains(out, "JS 原生验证") {
		t.Fatalf("project_info(op=read) 输出异常: %q", out)
	}
	// list/tree 应包含该条目
	out = execJSTool(t, reg, "project_info", `{"op":"tree"}`)
	if !strings.Contains(out, "jsnative验证") {
		t.Fatalf("project_info(op=tree) 未含条目: %q", out)
	}
	// explore
	out = execJSTool(t, reg, "project_info", `{"op":"explore"}`)
	if !strings.Contains(out, "项目结构概览") {
		t.Fatalf("project_info(op=explore) 输出异常: %q", out)
	}
}

// TestToolVerifyJSNative tool-resource 的 verify 工具 JS 原生化：注册 + 知识库验证可跑（只读）。
// ★ 2026-09-04 合并：tool-verify 已并入 tool-resource（JS 实现整体搬迁，行为不变）。
func TestToolVerifyJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-resource")
	for _, name := range []string{"memory_verify", "project_info_verify"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		reg.SetToolEnabled(name, true)
	}
	out := execJSTool(t, reg, "project_info_verify", `{}`)
	if !strings.Contains(out, "验证报告") {
		t.Fatalf("project_info_verify 输出异常: %q", out)
	}
}

// TestToolOfficeJSNative tool-office JS 原生化：csv 读写 + json_to_table（临时文件自清理）。
func TestToolOfficeJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-office")
	for _, name := range []string{"csv_read", "csv_write", "json_to_table", "table_stats", "text_report"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		reg.SetToolEnabled(name, true)
	}
	const file = "_temp/jsnative_test.csv"
	ctxPath := filepath.Join(jsNativeWorkspace, file)
	os.WriteFile(ctxPath, []byte("name,age\n张三,30\n李四,25\n"), 0o644)
	defer os.Remove(ctxPath)

	// csv_read：表格渲染 + 列过滤
	out := execJSTool(t, reg, "csv_read", `{"path":"`+file+`"}`)
	// ★ 2026-09 Round2：Markdown 表格按列宽 padRight 填充（"| name   | "），
	//   原断言 "| name |" 陈旧——改为验证表头/数据/分隔行（对宽度不敏感）。
	if !strings.Contains(out, "张三") || !strings.Contains(out, "| name") || !strings.Contains(out, "---") {
		t.Fatalf("csv_read 输出异常: %q", out)
	}
	out = execJSTool(t, reg, "csv_read", `{"path":"`+file+`","columns":"0"}`)
	if !strings.Contains(out, "张三") || strings.Contains(out, "30") {
		t.Fatalf("csv_read columns 过滤异常: %q", out)
	}
	// csv_write：JSON → CSV 文件
	out = execJSTool(t, reg, "csv_write", `{"path":"`+file+`","data":"[[\"a\",\"b\"],[\"1\",\"2\"]]"}`)
	if !strings.Contains(out, "已写入") {
		t.Fatalf("csv_write 输出异常: %q", out)
	}
	data, _ := os.ReadFile(ctxPath)
	if !strings.Contains(string(data), "a,b") {
		t.Fatalf("csv_write 文件内容异常: %q", string(data))
	}
	// json_to_table：JSON → Markdown 表格
	out = execJSTool(t, reg, "json_to_table", `{"json":"[{\"name\":\"x\",\"age\":1}]"}`)
	if !strings.Contains(out, "name") || !strings.Contains(out, "x") {
		t.Fatalf("json_to_table 输出异常: %q", out)
	}
	// table_stats：CSV 文本数值统计
	out = execJSTool(t, reg, "table_stats", `{"data":"age\n30\n25\n","format":"csv"}`)
	if !strings.Contains(out, "求和") || !strings.Contains(out, "55.00") {
		t.Fatalf("table_stats 输出异常: %q", out)
	}
	// text_report：扫描单文件
	out = execJSTool(t, reg, "text_report", `{"path":"`+file+`","group_by":"ext"}`)
	if !strings.Contains(out, "代码统计报告") {
		t.Fatalf("text_report 输出异常: %q", out)
	}

	// ── word_read：构造最小 .docx（word/document.xml 段落+表格）──
	reg.SetToolEnabled("word_read", true)
	docx := "_temp/jsnative_test.docx"
	docxPath := filepath.Join(jsNativeWorkspace, docx)
	writeTestZip(t, docxPath, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types/>`,
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
			`<w:body>` +
			`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>标题一</w:t></w:r></w:p>` +
			`<w:p><w:r><w:t>正文段落</w:t></w:r></w:p>` +
			`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>列A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>列B</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
			`</w:body></w:document>`,
	})
	defer os.Remove(docxPath)
	// ★ 2026-09 Round2：word_read markdown 渲染 Heading1 为纯文本「标题一」
	//   （无 "# " 前缀）——原断言 "# 标题一" 陈旧，改为对渲染细节不敏感的子串。
	out = execJSTool(t, reg, "word_read", `{"path":"`+docx+`","format":"markdown"}`)
	if !strings.Contains(out, "标题一") || !strings.Contains(out, "正文段落") || !strings.Contains(out, "列A") {
		t.Fatalf("word_read 输出异常: %q", out)
	}

	// ── read_xlsx：构造最小 .xlsx（sharedStrings + sheet1）──
	reg.SetToolEnabled("read_xlsx", true)
	xlsx := "_temp/jsnative_test.xlsx"
	xlsxPath := filepath.Join(jsNativeWorkspace, xlsx)
	writeTestZip(t, xlsxPath, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types/>`,
		"xl/sharedStrings.xml": `<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<si><t>名字</t></si><si><t>年龄</t></si><si><t>张三</t></si><si><t>30</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<sheetData>` +
			`<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>` +
			`<row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2" t="s"><v>3</v></c></row>` +
			`</sheetData></worksheet>`,
	})
	defer os.Remove(xlsxPath)
	out = execJSTool(t, reg, "read_xlsx", `{"path":"`+xlsx+`","sheet":"Sheet1"}`)
	if !strings.Contains(out, "名字") || !strings.Contains(out, "张三") || !strings.Contains(out, "| 名字") {
		t.Fatalf("read_xlsx 输出异常: %q", out)
	}
}

// writeTestZip 构造最小 ZIP 包（docx/xlsx 测试用）。
func writeTestZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip Create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("zip Write %s: %v", name, err)
		}
	}
}

// TestToolBinaryJSNative tool-binary JS 二进制回退（2026-08-22 迁移验证）：
// 插件 exe 已归档（bin/legacy-plugin-bins/），JS 仍调 ctx.binary.exec——
// 宿主内嵌 Go 内核（embedded_tools.go 的 registerBinaryRETools）承接执行。
func TestToolBinaryJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-binary")
	for _, name := range []string{"binary_strings", "inspect_binary", "binary_find", "binary_entropy", "write_binary", "binary_patch", "binary_hash"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		reg.SetToolEnabled(name, true)
	}

	// 构造测试二进制（MZ 头 + 字符串 + 高熵块）
	file := "_temp/jsnative_bin_test.bin"
	_ = os.MkdirAll(filepath.Join(jsNativeWorkspace, "_temp"), 0o755)
	defer os.Remove(filepath.Join(jsNativeWorkspace, file))
	content := []byte{0x4D, 0x5A, 0x90, 0x00, 'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', 0x00, 0x01, 0x02, 0x03}
	if err := os.WriteFile(filepath.Join(jsNativeWorkspace, file), content, 0o644); err != nil {
		t.Fatalf("写测试文件: %v", err)
	}

	// inspect_binary：MZ 魔数
	out := execJSTool(t, reg, "inspect_binary", `{"path":"`+file+`"}`)
	if !strings.Contains(out, "PE 可执行") {
		t.Fatalf("inspect_binary 输出异常: %q", out)
	}

	// binary_strings：提取 hello world
	out = execJSTool(t, reg, "binary_strings", `{"path":"`+file+`","min_length":4,"max_results":20}`)
	if !strings.Contains(out, "hello world") {
		t.Fatalf("binary_strings 输出异常（应含 hello world）: %q", out)
	}

	// binary_find：hex 模式
	out = execJSTool(t, reg, "binary_find", `{"path":"`+file+`","hex":"4d5a"}`)
	if !strings.Contains(out, "0x00000000") {
		t.Fatalf("binary_find 输出异常: %q", out)
	}

	// binary_entropy：整体熵
	out = execJSTool(t, reg, "binary_entropy", `{"path":"`+file+`","chunk_size":256}`)
	if !strings.Contains(out, "整体熵") {
		t.Fatalf("binary_entropy 输出异常: %q", out)
	}

	// write_binary + binary_patch 回读验证
	reg.SetToolEnabled("write_binary", true)
	reg.SetToolEnabled("binary_patch", true)
	writeFile := "_temp/jsnative_write_test.bin"
	defer os.Remove(filepath.Join(jsNativeWorkspace, writeFile))
	b64 := base64.StdEncoding.EncodeToString(content)
	out = execJSTool(t, reg, "write_binary", `{"path":"`+writeFile+`","base64":"`+b64+`"}`)
	if !strings.Contains(out, "已写入") {
		t.Fatalf("write_binary 输出异常: %q", out)
	}
	// binary_hash（走 ctx.fs.fileHash 基础原语）
	out = execJSTool(t, reg, "binary_hash", `{"path":"`+writeFile+`"}`)
	if !strings.Contains(out, "MD5：") || !strings.Contains(out, "SHA256：") {
		t.Fatalf("binary_hash 输出异常: %q", out)
	}
}

// TestToolBugJSNative tool-bug JS 原生化（2026-08-22）：装载真实 index.js，
// 验证 bug_detect/bug_fix 注册（★ Round4 削减：bug_analyze 已删除——
// bug_detect/bug_fix 覆盖构建输出→错误定位链，解析器内置插件内）。
func TestToolBugJSNative(t *testing.T) {
	_, reg := loadDiskPluginForTest(t, "tool-bug")
	for _, name := range []string{"bug_detect", "bug_fix"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("工具 %s 未注册", name)
		}
		reg.SetToolEnabled(name, true)
	}
	// 描述非空冒烟（bug_detect 全量反编译需真实工程，仅断言注册与元数据）
	tool, ok := reg.Get("bug_detect")
	if !ok || strings.TrimSpace(tool.Description) == "" {
		t.Fatal("bug_detect 描述为空")
	}
}
