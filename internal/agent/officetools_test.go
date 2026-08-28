package agent

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════
// 1. csv_read 测试
// ═══════════════════════════════════════════════════════════

func TestCSVRead(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerCSVRead(reg, dir)
	ctx := context.Background()

	// 准备 CSV 数据
	csvPath := filepath.Join(dir, "test.csv")
	os.WriteFile(csvPath, []byte("姓名,年龄,城市\n张三,28,北京\n李四,35,上海\n王五,42,广州\n"), 0o644)

	// 基本读取
	out, err := reg.Execute(ctx, "csv_read", `{"path":"test.csv"}`)
	if err != nil {
		t.Fatalf("csv_read: %v", err)
	}
	if !strings.Contains(out, "姓名") || !strings.Contains(out, "张三") || !strings.Contains(out, "上海") {
		t.Errorf("csv_read 基础读取 = %q", out)
	}

	// columns 筛选
	out, err = reg.Execute(ctx, "csv_read", `{"path":"test.csv","columns":"0,1"}`)
	if err != nil {
		t.Fatalf("csv_read columns: %v", err)
	}
	if !strings.Contains(out, "姓名") || !strings.Contains(out, "张三") || strings.Contains(out, "北京") {
		t.Errorf("csv_read columns 筛选 = %q", out)
	}

	// limit
	out, err = reg.Execute(ctx, "csv_read", `{"path":"test.csv","limit":2}`)
	if err != nil {
		t.Fatalf("csv_read limit: %v", err)
	}
	if !strings.Contains(out, "显示前 2 行") {
		t.Errorf("csv_read limit 应提示截断 = %q", out)
	}

	// offset
	out, err = reg.Execute(ctx, "csv_read", `{"path":"test.csv","offset":2}`)
	if err != nil {
		t.Fatalf("csv_read offset: %v", err)
	}
	if strings.Contains(out, "张三") {
		t.Errorf("csv_read offset 2 后不应含张三 = %q", out)
	}

	// TSV 格式
	tsvPath := filepath.Join(dir, "test.tsv")
	os.WriteFile(tsvPath, []byte("姓名\t年龄\n测试\t25\n"), 0o644)
	out, err = reg.Execute(ctx, "csv_read", `{"path":"test.tsv","delimiter":"tab"}`)
	if err != nil {
		t.Fatalf("csv_read tsv: %v", err)
	}
	if !strings.Contains(out, "测试") {
		t.Errorf("csv_read TSV = %q", out)
	}

	// 空文件
	emptyPath := filepath.Join(dir, "empty.csv")
	os.WriteFile(emptyPath, []byte{}, 0o644)
	out, err = reg.Execute(ctx, "csv_read", `{"path":"empty.csv"}`)
	if err != nil || !strings.Contains(out, "空文件") {
		t.Errorf("csv_read 空文件应提示空: %v, %q", err, out)
	}

	// 文件不存在
	_, err = reg.Execute(ctx, "csv_read", `{"path":"nope.csv"}`)
	if err == nil {
		t.Error("csv_read 不存在的文件应报错")
	}
}

// ═══════════════════════════════════════════════════════════
// 2. csv_write 测试
// ═══════════════════════════════════════════════════════════

func TestCSVWrite(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerCSVWrite(reg, dir)
	ctx := context.Background()

	// JSON 二维数组写入
	out, err := reg.Execute(ctx, "csv_write", `{"path":"out.csv","data":"[[\"a\",\"b\"],[\"1\",\"2\"],[\"3\",\"4\"]]"}`)
	if err != nil {
		t.Fatalf("csv_write JSON: %v", err)
	}
	if !strings.Contains(out, "已写入") {
		t.Errorf("csv_write 应返回成功: %q", out)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "out.csv"))
	if !strings.Contains(string(data), "a,b") || !strings.Contains(string(data), "1,2") {
		t.Errorf("csv_write 内容 = %q", string(data))
	}

	// 自定义表头
	out, err = reg.Execute(ctx, "csv_write", `{"path":"head.csv","header":"[\"x\",\"y\"]","data":"[[\"1\",\"2\"]]"}`)
	if err != nil {
		t.Fatalf("csv_write header: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "head.csv"))
	if !strings.Contains(string(data), "x,y") {
		t.Errorf("csv_write 表头 = %q", string(data))
	}

	// TSV 格式
	out, err = reg.Execute(ctx, "csv_write", `{"path":"out.tsv","delimiter":"tab","data":"[[\"a\",\"b\"],[\"1\",\"2\"]]"}`)
	if err != nil {
		t.Fatalf("csv_write tsv: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(dir, "out.tsv"))
	if !strings.Contains(string(data), "a\tb") {
		t.Errorf("csv_write TSV = %q", string(data))
	}

	// 无效 data
	_, err = reg.Execute(ctx, "csv_write", `{"path":"bad.csv","data":"not valid"}`)
	if err == nil {
		t.Error("csv_write 无效 data 应报错")
	}

	// 路径穿越
	_, err = reg.Execute(ctx, "csv_write", `{"path":"../../escape.csv","data":"[[\"x\"]]"}`)
	if err == nil {
		t.Error("csv_write 路径穿越应被拒")
	}
}

// ═══════════════════════════════════════════════════════════
// 3. json_to_table 测试
// ═══════════════════════════════════════════════════════════

func TestJSONToTable(t *testing.T) {
	reg := NewRegistry()
	registerJSONToTable(reg)
	ctx := context.Background()

	// 基本转换
	out, err := reg.Execute(ctx, "json_to_table", `{"json":"[{\"name\":\"张三\",\"age\":30},{\"name\":\"李四\",\"age\":25}]"}`)
	if err != nil {
		t.Fatalf("json_to_table: %v", err)
	}
	if !strings.Contains(out, "张三") || !strings.Contains(out, "30") || !strings.Contains(out, "2 条记录") {
		t.Errorf("json_to_table = %q", out)
	}

	// 指定列序
	out, err = reg.Execute(ctx, "json_to_table", `{"json":"[{\"a\":1,\"b\":2}]","columns":"b,a"}`)
	if err != nil {
		t.Fatalf("json_to_table columns: %v", err)
	}
	// b 应在 a 之前
	bIdx := strings.Index(out, "b")
	aIdx := strings.Index(out, "a")
	if bIdx < 0 || aIdx < 0 || bIdx > aIdx {
		t.Errorf("json_to_table 列序应 b 在前: %q", out)
	}

	// title
	out, err = reg.Execute(ctx, "json_to_table", `{"json":"[{\"x\":1}]","title":"测试表"}`)
	if err != nil {
		t.Fatalf("json_to_table title: %v", err)
	}
	if !strings.Contains(out, "测试表") {
		t.Errorf("json_to_table 应有标题: %q", out)
	}

	// limit
	out, err = reg.Execute(ctx, "json_to_table", `{"json":"[{\"x\":1},{\"x\":2},{\"x\":3}]","limit":2}`)
	if err != nil {
		t.Fatalf("json_to_table limit: %v", err)
	}
	if !strings.Contains(out, "显示前 2 条") {
		t.Errorf("json_to_table limit 应提示: %q", out)
	}

	// 空数组
	out, err = reg.Execute(ctx, "json_to_table", `{"json":"[]"}`)
	if err != nil || !strings.Contains(out, "空数组") {
		t.Errorf("json_to_table 空数组: %q, %v", out, err)
	}

	// 无效 JSON
	_, err = reg.Execute(ctx, "json_to_table", `{"json":"not json"}`)
	if err == nil {
		t.Error("json_to_table 无效 JSON 应报错")
	}
}

// ═══════════════════════════════════════════════════════════
// 4. table_stats 测试
// ═══════════════════════════════════════════════════════════

func TestTableStats(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerTableStats(reg, dir)
	ctx := context.Background()

	// CSV 文本统计
	out, err := reg.Execute(ctx, "table_stats", `{"data":"名称,数量\nA,10\nB,20\nC,30","format":"csv"}`)
	if err != nil {
		t.Fatalf("table_stats csv: %v", err)
	}
	if !strings.Contains(out, "60") || !strings.Contains(out, "30") || !strings.Contains(out, "10") {
		t.Errorf("table_stats 求和/最大/最小应有值: %q", out)
	}

	// JSON 格式统计
	out, err = reg.Execute(ctx, "table_stats", `{"data":"[{\"item\":\"A\",\"val\":100},{\"item\":\"B\",\"val\":200}]","format":"json"}`)
	if err != nil {
		t.Fatalf("table_stats json: %v", err)
	}
	if !strings.Contains(out, "300") || !strings.Contains(out, "150") {
		t.Errorf("table_stats JSON 应计算求和/均值: %q", out)
	}

	// 文件格式统计
	csvPath := filepath.Join(dir, "data.csv")
	os.WriteFile(csvPath, []byte("x,y\n1,2\n3,4\n5,6\n"), 0o644)
	out, err = reg.Execute(ctx, "table_stats", `{"data":"data.csv","format":"file"}`)
	if err != nil {
		t.Fatalf("table_stats file: %v", err)
	}
	if !strings.Contains(out, "9") || !strings.Contains(out, "12") {
		t.Errorf("table_stats file = %q", out)
	}

	// 分组统计
	out, err = reg.Execute(ctx, "table_stats", `{"data":"部门,金额\n销售,100\n销售,200\n技术,300\n技术,400","format":"csv","group_by":"部门"}`)
	if err != nil {
		t.Fatalf("table_stats group: %v", err)
	}
	if !strings.Contains(out, "销售") || !strings.Contains(out, "技术") || !strings.Contains(out, "300") || !strings.Contains(out, "700") {
		t.Errorf("table_stats 分组 = %q", out)
	}

	// 只有表头无数据
	out, err = reg.Execute(ctx, "table_stats", `{"data":"a,b","format":"csv"}`)
	if err != nil || !strings.Contains(out, "数据不足") {
		t.Errorf("table_stats 无数据应提示: %q, %v", out, err)
	}
}

// ═══════════════════════════════════════════════════════════
// 5. text_report 测试
// ═══════════════════════════════════════════════════════════

func TestTextReport(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerTextReport(reg, dir)
	ctx := context.Background()

	// 创建测试文件
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte("// comment\npackage util\n\nconst X = 1\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub", "c.js"), []byte("function hello() {\n  return 1;\n}\n"), 0o644)

	// 默认按扩展名分组
	dirEsc := strings.ReplaceAll(dir, "\\", "/")
	out, err := reg.Execute(ctx, "text_report", `{"path":"`+dirEsc+`"}`)
	if err != nil {
		t.Fatalf("text_report: %v", err)
	}
	if !strings.Contains(out, "代码统计报告") {
		t.Errorf("text_report 应有标题: %q", out)
	}
	if !strings.Contains(out, ".go") {
		t.Errorf("text_report 应有 .go 统计: %q", out)
	}

	// 限定扩展名
	dirEsc = strings.ReplaceAll(dir, "\\", "/")
	out, err = reg.Execute(ctx, "text_report", `{"path":"`+dirEsc+`","extensions":".js"}`)
	if err != nil {
		t.Fatalf("text_report ext: %v", err)
	}
	if strings.Contains(out, ".go") {
		t.Errorf("text_report 限定 .js 后不应含 .go: %q", out)
	}
	if !strings.Contains(out, ".js") {
		t.Errorf("text_report 应有 .js: %q", out)
	}

	// 按目录分组
	out, err = reg.Execute(ctx, "text_report", `{"path":"`+dirEsc+`","group_by":"dir"}`)
	if err != nil {
		t.Fatalf("text_report dir: %v", err)
	}
	if !strings.Contains(out, "(根目录)") {
		t.Errorf("text_report 按目录应有根目录: %q", out)
	}
}

// ═══════════════════════════════════════════════════════════
// 6. word_write + word_read 联合测试
// ═══════════════════════════════════════════════════════════

func TestWordWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerWordWrite(reg, dir)
	registerWordRead(reg, dir)
	ctx := context.Background()

	// 写入 docx
	out, err := reg.Execute(ctx, "word_write", `{"path":"test.docx","content":"# 标题\n\n这是正文段落\n\n- 列表项1\n- 列表项2\n\n| 列1 | 列2 |\n|-----|-----|\n| A | B |","title":"测试文档"}`)
	if err != nil {
		t.Fatalf("word_write: %v", err)
	}
	if !strings.Contains(out, "已生成") {
		t.Errorf("word_write 应返回成功: %q", out)
	}

	// 验证生成的 docx 是有效的 ZIP
	docxPath := filepath.Join(dir, "test.docx")
	zr, err := zip.OpenReader(docxPath)
	if err != nil {
		t.Fatalf("生成的 docx 不是有效 ZIP: %v", err)
	}
	defer zr.Close()
	hasDoc := false
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			hasDoc = true
			break
		}
	}
	if !hasDoc {
		t.Error("docx 缺少 word/document.xml")
	}

	// 读取 docx
	out, err = reg.Execute(ctx, "word_read", `{"path":"test.docx"}`)
	if err != nil {
		t.Fatalf("word_read: %v", err)
	}
	if !strings.Contains(out, "标题") || !strings.Contains(out, "正文段落") {
		t.Errorf("word_read 应含写入内容: %q", out)
	}

	// 读取不存在的文件
	_, err = reg.Execute(ctx, "word_read", `{"path":"no.docx"}`)
	if err == nil {
		t.Error("word_read 不存在的文件应报错")
	}

	// 写入空目录穿越
	_, err = reg.Execute(ctx, "word_write", `{"path":"../../esc.docx","content":"hi"}`)
	if err == nil {
		t.Error("word_write 路径穿越应被拒")
	}
}

// ═══════════════════════════════════════════════════════════
// 7. write_xlsx + read_xlsx 联合测试
// ═══════════════════════════════════════════════════════════

func TestXLSXWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerXLSXWrite(reg, dir)
	registerXLSXRead(reg, dir)
	ctx := context.Background()

	// 写入 xlsx
	out, err := reg.Execute(ctx, "write_xlsx", `{"path":"test.xlsx","data":"[[\"名称\",\"值\"],[\"A\",\"1\"],[\"B\",\"2\"]]"}`)
	if err != nil {
		t.Fatalf("write_xlsx: %v", err)
	}
	if !strings.Contains(out, "已生成") {
		t.Errorf("write_xlsx 应返回成功: %q", out)
	}

	// 验证 ZIP
	xlsxPath := filepath.Join(dir, "test.xlsx")
	zr, err := zip.OpenReader(xlsxPath)
	if err != nil {
		t.Fatalf("生成的 xlsx 不是有效 ZIP: %v", err)
	}
	defer zr.Close()
	hasSheet := false
	for _, f := range zr.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			hasSheet = true
			break
		}
	}
	if !hasSheet {
		t.Error("xlsx 缺少 xl/worksheets/sheet1.xml")
	}

	// 读取 xlsx
	out, err = reg.Execute(ctx, "read_xlsx", `{"path":"test.xlsx"}`)
	if err != nil {
		t.Fatalf("read_xlsx: %v", err)
	}
	if !strings.Contains(out, "名称") || !strings.Contains(out, "A") {
		t.Errorf("read_xlsx 应含写入内容: %q", out)
	}

	// 指定 sheet 名称
	out, err = reg.Execute(ctx, "write_xlsx", `{"path":"named.xlsx","data":"[[\"x\"],[\"1\"]]","sheet":"数据表"}`)
	if err != nil {
		t.Fatalf("write_xlsx named: %v", err)
	}
	out, err = reg.Execute(ctx, "read_xlsx", `{"path":"named.xlsx","sheet":"数据表"}`)
	if err != nil {
		t.Fatalf("read_xlsx named: %v", err)
	}
	if !strings.Contains(out, "数据表") {
		t.Errorf("read_xlsx 应有 sheet 名: %q", out)
	}

	// 写入路径穿越
	_, err = reg.Execute(ctx, "write_xlsx", `{"path":"../../bad.xlsx","data":"[[\"x\"]]"}`)
	if err == nil {
		t.Error("write_xlsx 路径穿越应被拒")
	}
	// 读取路径穿越
	_, err = reg.Execute(ctx, "read_xlsx", `{"path":"../../bad.xlsx"}`)
	if err == nil {
		t.Error("read_xlsx 路径穿越应被拒")
	}
}

// ═══════════════════════════════════════════════════════════
// 8. read_pdf 文本提取测试
// ═══════════════════════════════════════════════════════════

func TestPDFRead(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	registerPDFRead(reg, dir)
	ctx := context.Background()

	// 构建一个最小 PDF（含一行文本 "Hello World"）
	// 最小 PDF 结构：header → objects → xref → trailer
	pdfContent := "%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n" +
		"4 0 obj\n<< /Length 44 >>\nstream\nBT /F1 12 Tf 100 700 Td (Hello World) Tj ET\nendstream\nendobj\n" +
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n" +
		"xref\n0 6\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \n0000000266 00000 n \n0000000365 00000 n \n" +
		"trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n434\n%%%%EOF"

	pdfPath := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(pdfPath, []byte(pdfContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 读取文本
	out, err := reg.Execute(ctx, "read_pdf", `{"path":"test.pdf"}`)
	if err != nil {
		t.Fatalf("read_pdf: %v", err)
	}
	if !strings.Contains(out, "Hello World") {
		t.Errorf("read_pdf 应提取到 Hello World: %q", out)
	}

	// 按页读取
	out, err = reg.Execute(ctx, "read_pdf", `{"path":"test.pdf","page":1}`)
	if err != nil {
		t.Fatalf("read_pdf page: %v", err)
	}
	if !strings.Contains(out, "Hello World") {
		t.Errorf("read_pdf page=1 应提取: %q", out)
	}

	// 不存在的文件
	_, err = reg.Execute(ctx, "read_pdf", `{"path":"nope.pdf"}`)
	if err == nil {
		t.Error("read_pdf 不存在的文件应报错")
	}

	// 空文本内容（图片 PDF 模拟——无 BT/ET 但文件存在 → 应返回无嵌入文本提示）
	emptyPDF := "%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n" +
		"xref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \n" +
		"trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n200\n%%%%EOF"
	os.WriteFile(filepath.Join(dir, "img.pdf"), []byte(emptyPDF), 0o644)
	out, err = reg.Execute(ctx, "read_pdf", `{"path":"img.pdf"}`)
	// OCR 已移除（2026-08-22）——应返回成功但含「未提取到嵌入文本」提示
	if err != nil {
		t.Errorf("read_pdf 空文本应返回成功含提示, 但返回错误: %v", err)
	}
	if !strings.Contains(out, "未提取到嵌入文本") {
		t.Errorf("read_pdf 空文本应含未提取提示: %q", out)
	}
}

// ═══════════════════════════════════════════════════════════
// 9. markdown_to_html 测试
// ═══════════════════════════════════════════════════════════

func TestMarkdownToHTML(t *testing.T) {
	reg := NewRegistry()
	registerMarkdownToHTML(reg)
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"标题", "# H1\n## H2\n### H3", []string{"<h1>", "<h2>", "<h3>"}},
		{"粗体斜体", "**粗** *斜*", []string{"<strong>", "<em>"}},
		{"代码", "`code`", []string{"<code>code</code>"}},
		{"代码块", "```\ncode\n```", []string{"<pre>", "<code>"}},
		{"链接", "[文本](url)", []string{"<a href="}},
		{"图片", "![alt](img.png)", []string{"<img src="}},
		{"引用", "> 引用", []string{"<blockquote>"}},
		{"无序列表", "- 项1\n- 项2", []string{"<li>"}},
		{"表格", "| A | B |\n|---|---|\n| 1 | 2 |", []string{"<table>", "<th>", "<td>"}},
		{"水平线", "---", []string{"<hr>"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := reg.Execute(ctx, "markdown_to_html", `{"markdown":"`+escapeTestJSON(tt.input)+`"}`)
			if err != nil {
				t.Fatalf("markdown_to_html %s: %v", tt.name, err)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("markdown_to_html %s 应含 %s: %q", tt.name, w, out)
				}
			}
		})
	}

	// full_html 模式
	out, err := reg.Execute(ctx, "markdown_to_html", `{"markdown":"hello","full_html":true,"title":"测试"}`)
	if err != nil {
		t.Fatalf("markdown_to_html full: %v", err)
	}
	if !strings.Contains(out, "<!DOCTYPE html>") || !strings.Contains(out, "<title>测试</title>") {
		t.Errorf("markdown_to_html full 应含 DOCTYPE+title: %q", out)
	}
}

// ═══════════════════════════════════════════════════════════
// 10. 辅助函数测试
// ═══════════════════════════════════════════════════════════

func TestCSVDelim(t *testing.T) {
	if csvDelim("tab") != '\t' {
		t.Error("tab 应返回制表符")
	}
	if csvDelim("Tab") != '\t' {
		t.Error("Tab 应返回制表符")
	}
	if csvDelim("comma") != ',' {
		t.Error("comma 应返回逗号")
	}
	if csvDelim("") != ',' {
		t.Error("空值应返回逗号")
	}
	if csvDelim("xxx") != ',' {
		t.Error("未知值应返回逗号")
	}
}

func TestPadRight(t *testing.T) {
	if padRight("abc", 5) != "abc  " {
		t.Errorf("padRight 结果=%q", padRight("abc", 5))
	}
	if padRight("abcde", 3) != "abcde" {
		t.Errorf("padRight 不应截断=%q", padRight("abcde", 3))
	}
	if padRight("", 4) != "    " {
		t.Errorf("padRight 空串=%q", padRight("", 4))
	}
}

func TestParseColIndex(t *testing.T) {
	idx := parseColIndex("0,2,4", 5)
	if len(idx) != 3 || idx[0] != 0 || idx[1] != 2 || idx[2] != 4 {
		t.Errorf("parseColIndex 结果=%v", idx)
	}
	// 越界被跳过
	idx = parseColIndex("0,99", 3)
	if len(idx) != 1 || idx[0] != 0 {
		t.Errorf("parseColIndex 越界应跳过=%v", idx)
	}
	// 空字符串
	if parseColIndex("", 5) != nil {
		t.Error("parseColIndex 空串应为 nil")
	}
	// 非数字
	idx = parseColIndex("abc", 3)
	if len(idx) != 0 {
		t.Error("parseColIndex 非数字应跳过")
	}
}

func TestParseMarkdownTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |"
	rows := parseMarkdownTable(md)
	if len(rows) != 3 {
		t.Fatalf("parseMarkdownTable 应有 3 行: %d", len(rows))
	}
	if rows[1][0] != "1" || rows[1][1] != "2" {
		t.Errorf("parseMarkdownTable 数据行=%v", rows[1])
	}

	// 无表格
	if rows := parseMarkdownTable("普通文本"); len(rows) != 0 {
		t.Error("parseMarkdownTable 无表格应返回空")
	}

	// 多表格（只取第一个）
	md2 := "| X |\n|---|\n| 1 |\n\n其他\n\n| Y |\n|---|\n| 2 |"
	rows2 := parseMarkdownTable(md2)
	if len(rows2) != 2 {
		t.Errorf("parseMarkdownTable 只取第一个表格: %d", len(rows2))
	}
}

func TestEscapeXML(t *testing.T) {
	if escapeXML(`a&b<c>d"e`) != `a&amp;b&lt;c&gt;d&quot;e` {
		t.Errorf("escapeXML=%q", escapeXML(`a&b<c>d"e`))
	}
}

func TestWriteMarkdownTable(t *testing.T) {
	var b strings.Builder
	records := [][]string{{"Name", "Age"}, {"Alice", "30"}, {"Bob", "25"}}
	writeMarkdownTable(&b, records, nil)
	out := b.String()
	if !strings.Contains(out, "| Name ") || !strings.Contains(out, "| Alice ") {
		t.Errorf("writeMarkdownTable=%q", out)
	}

	// 带 colIdx 过滤
	var b2 strings.Builder
	writeMarkdownTable(&b2, records, []int{1}) // 只显示 Age
	out2 := b2.String()
	if strings.Contains(out2, "Name") {
		t.Errorf("colIdx 过滤后不应有 Name: %q", out2)
	}
	if !strings.Contains(out2, "Age") || !strings.Contains(out2, "30") {
		t.Errorf("colIdx 过滤后应有 Age: %q", out2)
	}
}

// ═══════════════════════════════════════════════════════════
// 辅助：JSON 转义（测试用）
// ═══════════════════════════════════════════════════════════

func escapeTestJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
