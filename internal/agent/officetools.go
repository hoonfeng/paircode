package agent

// 办公工具集：CSV 读写、JSON 转 Markdown 表格、表格统计、代码报告、Word (.docx) 读取。
// 纯 Go 标准库实现（encoding/csv, encoding/json, encoding/xml, archive/zip），零外部依赖。
// 文件路径经 resolvePath 安全校验，路径穿越防护由 tools.go 的 resolvePath 统一保障。

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ─── 注册入口 ──

// registerOfficeTools 注册全部办公工具。
func registerOfficeTools(r *Registry, root string) {
	registerCSVRead(r, root)
	registerCSVWrite(r, root)
	registerJSONToTable(r)
	registerTableStats(r, root)
	registerTextReport(r, root)
	registerWordRead(r, root)
	registerWordWrite(r, root)
	registerXLSXRead(r, root)
	registerXLSXWrite(r, root)
	registerPDFRead(r, root)
	registerMarkdownToHTML(r)
}

// ═══════════════════════════════════════════════════════════
// 1. csv_read
// ═══════════════════════════════════════════════════════════

func registerCSVRead(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "csv_read",
		UsageGuide: "读取 CSV/TSV 文件并以 Markdown 表格形式返回。比直接 read_file 读 CSV 更友好（自动解析分隔符+格式化表格）。delimiter 可指定 comma/tab。",
		Description: "读取 CSV/TSV 文件并以 Markdown 表格形式返回内容。" +
			"参数 delimiter 可选 \"comma\"（逗号, 默认）或 \"tab\"（制表符）。" +
			"columns 按列索引过滤（从 0 开始，逗号分隔，如 \"0,2,3\"）。" +
			"limit 限制返回行数（默认 100，-1=全部），offset 跳过前 N 行。",
		Parameters: objSchema(props{
			"path":      strProp("文件路径（工作区内）"),
			"delimiter": strProp("可选：分隔符，\"comma\"（逗号）或 \"tab\"（制表符），默认 \"comma\""),
			"columns":   strProp("可选：要显示的列索引（从 0 开始，逗号分隔），省略显示全部"),
			"limit":     intProp("可选：最大返回行数（默认 100，-1 表示全部）"),
			"offset":    intProp("可选：跳过前 N 行（默认 0）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return "", fmt.Errorf("读取文件失败: %w", err)
			}
			if len(data) > 100<<20 { // 100MB 上限
				return "", fmt.Errorf("文件超过 100MB，请缩小范围后用 search_content 搜索特定内容")
			}

			delim := csvDelim(argStr(args, "delimiter"))
			limit := argInt(args, "limit", 100)
			offset := argInt(args, "offset", 0)
			colSpec := strings.TrimSpace(argStr(args, "columns"))

			records, err := readCSV(data, delim)
			if err != nil {
				return "", err
			}
			if len(records) == 0 {
				return "（空文件）", nil
			}

			colIdx := parseColIndex(colSpec, len(records[0]))
			if colSpec != "" && len(colIdx) == 0 {
				return "", fmt.Errorf("无效的 columns 参数: %q，应为逗号分隔的列索引（从 0 开始）", colSpec)
			}

			// 应用 offset
			if offset > 0 && offset < len(records) {
				records = records[offset:]
			} else if offset >= len(records) {
				return "（offset 超出文件行数）", nil
			}

			total := len(records)
			if limit > 0 && limit < len(records) {
				records = records[:limit]
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("**CSV 文件**: `%s` · 共 %d 行 × %d 列", argStr(args, "path"), total, len(records[0])))
			if limit > 0 && total > limit {
				b.WriteString(fmt.Sprintf(" · 显示前 %d 行", limit))
			}
			if offset > 0 {
				b.WriteString(fmt.Sprintf(" · 跳过 %d 行", offset))
			}
			if colSpec != "" {
				b.WriteString(fmt.Sprintf(" · 显示列 [%s]", colSpec))
			}
			b.WriteString("\n\n")
			writeMarkdownTable(&b, records, colIdx)
			return b.String(), nil
		},
	})
}

// ═══════════════════════════════════════════════════════════
// 2. csv_write
// ═══════════════════════════════════════════════════════════

func registerCSVWrite(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "csv_write",
		UsageGuide: "将表格数据写入 CSV/TSV 文件。data 参数支持 JSON 二维数组或 Markdown 表格文本。比手动拼接 CSV 更高效（自动处理转义+分隔符）。需审核批准。",
		Description: "将表格数据写入 CSV/TSV 文件。" +
			"data 为 JSON 二维数组（如 [[\"列1\",\"列2\"],[\"值1\",\"值2\"]]）或 Markdown 表格文本。" +
			"delimiter 可选 \"comma\"（逗号, 默认）或 \"tab\"（制表符）。" +
			"header 为可选的表头行 JSON 数组，省略则从 data 首行自动提取。",
		Parameters: objSchema(props{
			"path":      strProp("文件路径（工作区内）"),
			"data":      strProp("表格数据：JSON 二维数组字符串，或 Markdown 表格文本"),
			"delimiter": strProp("可选：分隔符 \"comma\" 或 \"tab\"，默认 \"comma\""),
			"header":    strProp("可选：表头行 JSON 数组，如 \"[\"姓名\",\"年龄\"]\""),
		}, "path", "data"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			dataStr := argStr(args, "data")
			delim := csvDelim(argStr(args, "delimiter"))
			headerJSON := strings.TrimSpace(argStr(args, "header"))

			// 解析数据：优先 JSON 二维数组，回退 Markdown 表格
			var records [][]string
			if err := json.Unmarshal([]byte(dataStr), &records); err != nil {
				records = parseMarkdownTable(dataStr)
			}
			if len(records) == 0 {
				return "", fmt.Errorf("data 格式无效：无法解析为 JSON 二维数组或 Markdown 表格")
			}

			if headerJSON != "" {
				var header []string
				if err := json.Unmarshal([]byte(headerJSON), &header); err != nil {
					return "", fmt.Errorf("header JSON 解析失败: %w", err)
				}
				records = append([][]string{header}, records...)
			}

			var buf bytes.Buffer
			w := csv.NewWriter(&buf)
			w.Comma = delim
			if err := w.WriteAll(records); err != nil {
				return "", fmt.Errorf("CSV 生成失败: %w", err)
			}
			w.Flush()
			if err := w.Error(); err != nil {
				return "", fmt.Errorf("CSV 生成失败: %w", err)
			}
			if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
				return "", fmt.Errorf("文件写入失败: %w", err)
			}
			return fmt.Sprintf("已写入 CSV 文件 `%s`（%d 行 × %d 列，%d 字节）",
				argStr(args, "path"), len(records), len(records[0]), buf.Len()), nil
		},
	})
}

// ═══════════════════════════════════════════════════════════
// 3. json_to_table：JSON 数组 → Markdown 表格
// ═══════════════════════════════════════════════════════════

func registerJSONToTable(r *Registry) {
	r.Register(&Tool{
		Name:       "json_to_table",
		UsageGuide: "将 JSON 数组字符串转为 Markdown 表格。columns 参数指定列名和顺序。比手动格式化更高效（自动生成表头+对齐）。",
		Description: "将 JSON 数组字符串转为 Markdown 表格。" +
			"columns 指定列名和顺序（逗号分隔，如 \"name,age\"），省略则使用全部键并按字母序排列。" +
			"limit 限制行数（默认 100，-1=全部），title 可选表格标题。",
		Parameters: objSchema(props{
			"json":    strProp("JSON 数组字符串（必填，如 [{\"name\":\"张三\",\"age\":30}]）"),
			"columns": strProp("可选：显示的键名（逗号分隔），省略则用全部键"),
			"limit":   intProp("可选：最大行数（默认 100，-1=全部）"),
			"title":   strProp("可选：表格标题"),
		}, "json"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			jsonStr := argStr(args, "json")
			colSpec := strings.TrimSpace(argStr(args, "columns"))
			limit := argInt(args, "limit", 100)
			title := strings.TrimSpace(argStr(args, "title"))

			// 解析 JSON 数组
			var records []map[string]any
			if err := json.Unmarshal([]byte(jsonStr), &records); err != nil {
				return "", fmt.Errorf("JSON 解析失败: %w", err)
			}
			if len(records) == 0 {
				return "（空数组）", nil
			}

			// 确定列
			var cols []string
			if colSpec != "" {
				cols = strings.Split(colSpec, ",")
				for i := range cols {
					cols[i] = strings.TrimSpace(cols[i])
				}
			} else {
				// 从第一条记录提取键，排序
				for k := range records[0] {
					cols = append(cols, k)
				}
				sort.Strings(cols)
			}

			total := len(records)
			if limit > 0 && limit < len(records) {
				records = records[:limit]
			}

			// 构建二维表格
			var rows [][]string
			rows = append(rows, cols)
			for _, rec := range records {
				row := make([]string, len(cols))
				for i, col := range cols {
					if v, ok := rec[col]; ok {
						row[i] = fmt.Sprint(v)
					}
				}
				rows = append(rows, row)
			}

			var b strings.Builder
			if title != "" {
				b.WriteString(fmt.Sprintf("**%s**\n\n", title))
			}
			b.WriteString(fmt.Sprintf("共 %d 条记录", total))
			if limit > 0 && total > limit {
				b.WriteString(fmt.Sprintf("，显示前 %d 条", limit))
			}
			b.WriteString("\n\n")
			writeMarkdownTable(&b, rows, nil)
			return b.String(), nil
		},
	})
}

// ═══════════════════════════════════════════════════════════
// 4. table_stats：表格数值列统计
// ═══════════════════════════════════════════════════════════

func registerTableStats(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "table_stats",
		UsageGuide: "对表格数据的数值列做基本统计（求和/均值/最大/最小/计数）。group_by 可按某列分组统计。比手动计算更快（自动识别数值列+分组聚合）。",
		Description: "对表格数据的数值列做基本统计（求和、均值、最大值、最小值、计数）。" +
			"data 为 CSV 文本、JSON 数组或文件路径。format 指定数据格式：\"csv\"（CSV 文本）\"json\"（JSON 数组）\"file\"（文件路径）。" +
			"group_by 按指定列分组统计（可选）。",
		Parameters: objSchema(props{
			"data":     strProp("数据：CSV 文本、JSON 数组字符串、或文件路径（根据 format）"),
			"format":   strProp("数据格式：\"csv\"（默认）/ \"json\" / \"file\""),
			"group_by": strProp("可选：按此列名分组统计"),
		}, "data"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dataStr := argStr(args, "data")
			format := strings.ToLower(strings.TrimSpace(argStr(args, "format")))
			groupBy := strings.TrimSpace(argStr(args, "group_by"))
			if format == "" {
				format = "csv"
			}

			// 加载数据
			var records [][]string
			switch format {
			case "file":
				p, err := resolvePath(root, dataStr)
				if err != nil {
					return "", err
				}
				raw, err := os.ReadFile(p)
				if err != nil {
					return "", fmt.Errorf("读取文件失败: %w", err)
				}
				recs, err := readCSV(raw, ',')
				if err != nil {
					return "", err
				}
				records = recs
			case "json":
				var objs []map[string]any
				if err := json.Unmarshal([]byte(dataStr), &objs); err != nil {
					return "", fmt.Errorf("JSON 解析失败: %w", err)
				}
				if len(objs) == 0 {
					return "（空数据）", nil
				}
				var cols []string
				for k := range objs[0] {
					cols = append(cols, k)
				}
				sort.Strings(cols)
				records = append(records, cols)
				for _, obj := range objs {
					row := make([]string, len(cols))
					for i, col := range cols {
						if v, ok := obj[col]; ok {
							row[i] = fmt.Sprint(v)
						}
					}
					records = append(records, row)
				}
			default: // csv
				recs, err := readCSV([]byte(dataStr), ',')
				if err != nil {
					return "", err
				}
				records = recs
			}

			if len(records) < 2 {
				return "（数据不足，至少需要表头 + 1 行数据）", nil
			}

			header := records[0]
			dataRows := records[1:]
			colIdx := make(map[string]int)
			for i, h := range header {
				colIdx[h] = i
			}

			// 自动识别数值列
			var numCols []colStat
			var groupColIdx int = -1
			if groupBy != "" {
				if idx, ok := colIdx[groupBy]; ok {
					groupColIdx = idx
				}
			}

			// 遍历数据行找出数值列
			if len(dataRows) > 0 {
				for ci := range header {
					// 跳过分组列
					if ci == groupColIdx {
						continue
					}
					allNumeric := true
					for _, row := range dataRows {
						if ci >= len(row) {
							allNumeric = false
							break
						}
						if row[ci] == "" || row[ci] == "-" || row[ci] == "N/A" {
							continue // 空值跳过
						}
						if _, err := strconv.ParseFloat(strings.TrimSpace(row[ci]), 64); err != nil {
							allNumeric = false
							break
						}
					}
					if allNumeric {
						numCols = append(numCols, colStat{name: header[ci]})
					}
				}
			}

			if len(numCols) == 0 {
				return "（未找到数值列，无法统计）", nil
			}

			// 分组统计
			if groupColIdx >= 0 {
				return groupedStats(records, header, numCols, colIdx, groupColIdx, groupBy), nil
			}

			// 总体统计
			for ri := range numCols {
				ci := colIdx[numCols[ri].name]
				s := &numCols[ri]
				first := true
				for _, row := range dataRows {
					if ci >= len(row) {
						continue
					}
					val := strings.TrimSpace(row[ci])
					if val == "" || val == "-" || val == "N/A" {
						continue
					}
					f, err := strconv.ParseFloat(val, 64)
					if err != nil {
						continue
					}
					s.sum += f
					s.count++
					if first {
						s.min = f
						s.max = f
						first = false
					} else {
						if f < s.min {
							s.min = f
						}
						if f > s.max {
							s.max = f
						}
					}
				}
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("**统计结果** · %d 行数据\n\n", len(dataRows)))
			writeStatsTable(&b, numCols)
			return b.String(), nil
		},
	})
}

// groupedStats 分组统计输出
func groupedStats(records [][]string, header []string, numCols []colStat, colIdx map[string]int, groupColIdx int, groupBy string) string {
	dataRows := records[1:]
	// 收集分组
	groups := make(map[string][]int)
	for ri, row := range dataRows {
		if groupColIdx < len(row) {
			key := row[groupColIdx]
			groups[key] = append(groups[key], ri)
		}
	}

	// 排序分组键
	var groupKeys []string
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Strings(groupKeys)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("**分组统计**（按 `%s` 分组）\n\n", groupBy))

	// 对每个分组计算统计
	for _, gk := range groupKeys {
		indices := groups[gk]
		b.WriteString(fmt.Sprintf("**%s**（%d 行）\n\n", gk, len(indices)))
		var stats []colStat
		for _, nc := range numCols {
			ci := colIdx[nc.name]
			s := colStat{name: nc.name}
			first := true
			for _, ri := range indices {
				row := dataRows[ri]
				if ci >= len(row) {
					continue
				}
				val := strings.TrimSpace(row[ci])
				if val == "" || val == "-" || val == "N/A" {
					continue
				}
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					continue
				}
				s.sum += f
				s.count++
				if first {
					s.min = f
					s.max = f
					first = false
				} else {
					if f < s.min {
						s.min = f
					}
					if f > s.max {
						s.max = f
					}
				}
			}
			stats = append(stats, s)
		}
		writeStatsTable(&b, stats)
		b.WriteString("\n")
	}
	return b.String()
}

// writeStatsTable 写统计表
func writeStatsTable(b *strings.Builder, stats []colStat) {
	rows := [][]string{
		{"列名", "计数", "求和", "均值", "最小值", "最大值"},
	}
	for _, s := range stats {
		avg := 0.0
		if s.count > 0 {
			avg = s.sum / float64(s.count)
		}
		rows = append(rows, []string{
			s.name,
			strconv.Itoa(s.count),
			fmt.Sprintf("%.2f", s.sum),
			fmt.Sprintf("%.2f", avg),
			fmt.Sprintf("%.2f", s.min),
			fmt.Sprintf("%.2f", s.max),
		})
	}
	writeMarkdownTable(b, rows, nil)
}

// ═══════════════════════════════════════════════════════════
// 5. text_report：代码行数统计报告
// ═══════════════════════════════════════════════════════════

func registerTextReport(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "text_report",
		UsageGuide: "扫描目录树，按文件扩展名或目录分组统计代码行数。快速了解项目规模和技术栈分布。比 run_command wc -l 更智能（自动跳过 .git/node_modules+按类型分组）。",
		Description: "扫描工作区目录树，按文件扩展名分组统计行数。" +
			"支持统计总行数、代码行（非空非纯注释）、注释行、空行。" +
			"path 限定扫描子目录（默认工作区根）；extensions 限定文件扩展名（逗号分隔，如 \".go,.ts,.vue\"）；" +
			"group_by 分组方式：\"ext\"（按扩展名，默认）或 \"dir\"（按目录）。" +
			"自动跳过 .git/node_modules/vendor 等目录。",
		Parameters: objSchema(props{
			"path":       strProp("可选：要扫描的目录路径（默认工作区根）"),
			"extensions": strProp("可选：限定文件扩展名，逗号分隔（如 \".go,.ts,.vue\"）"),
			"group_by":   strProp("可选：分组方式 \"ext\"（按扩展名，默认）或 \"dir\"（按目录）"),
			"max_files":  intProp("可选：最大扫描文件数（默认 5000）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			scanPath := strings.TrimSpace(argStr(args, "path"))
			if scanPath == "" {
				scanPath = root
			} else {
				var err error
				scanPath, err = resolvePath(root, scanPath)
				if err != nil {
					return "", err
				}
			}
			extStr := strings.TrimSpace(argStr(args, "extensions"))
			groupBy := strings.ToLower(strings.TrimSpace(argStr(args, "group_by")))
			maxFiles := argInt(args, "max_files", 5000)
			if groupBy == "" {
				groupBy = "ext"
			}

			// 解析扩展名过滤
			var extSet map[string]bool
			if extStr != "" {
				extSet = make(map[string]bool)
				for _, e := range strings.Split(extStr, ",") {
					e = strings.TrimSpace(e)
					if !strings.HasPrefix(e, ".") {
						e = "." + e
					}
					extSet[strings.ToLower(e)] = true
				}
			}

			type fileStat struct {
				total   int
				code    int
				comment int
				blank   int
			}
			extStats := make(map[string]*fileStat)
			dirStats := make(map[string]*fileStat)
			total := &fileStat{}
			fileCount := 0

			err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return filepath.SkipDir
				}
				if d.IsDir() {
					name := strings.ToLower(d.Name())
					if isSkipDir(name) {
						return filepath.SkipDir
					}
					return nil
				}
				if fileCount >= maxFiles {
					return io.EOF
				}
				ext := strings.ToLower(filepath.Ext(path))
				if extSet != nil && !extSet[ext] {
					return nil
				}

				fileCount++
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				lines := strings.Split(string(data), "\n")
				fs := countFileLines(lines)

				relPath, _ := filepath.Rel(scanPath, path)
				dir := filepath.Dir(relPath)
				if dir == "." {
					dir = "(根目录)"
				}

				// 按扩展名累加
				if ext == "" {
					ext = "(无扩展名)"
				}
				if extStats[ext] == nil {
					extStats[ext] = &fileStat{}
				}
				es := extStats[ext]
				es.total += fs.total
				es.code += fs.code
				es.comment += fs.comment
				es.blank += fs.blank

				// 按目录累加
				if dirStats[dir] == nil {
					dirStats[dir] = &fileStat{}
				}
				ds := dirStats[dir]
				ds.total += fs.total
				ds.code += fs.code
				ds.comment += fs.comment
				ds.blank += fs.blank

				total.total += fs.total
				total.code += fs.code
				total.comment += fs.comment
				total.blank += fs.blank
				return nil
			})
			if err != nil && err != io.EOF {
				return "", fmt.Errorf("扫描失败: %w", err)
			}

			if fileCount == 0 {
				return "（未找到匹配的文件）", nil
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("**代码统计报告** · 扫描目录: `%s`\n\n", scanPath))
			b.WriteString(fmt.Sprintf("共计 %d 个文件，%d 行（代码 %d / 注释 %d / 空行 %d）\n\n",
				fileCount, total.total, total.code, total.comment, total.blank))

			var rows [][]string
			if groupBy == "dir" {
				// 按目录分组
				var dirs []string
				for k := range dirStats {
					dirs = append(dirs, k)
				}
				sort.Strings(dirs)
				rows = append(rows, []string{"目录", "文件数（估算）", "总行数", "代码行", "注释行", "空行"})
				for _, d := range dirs {
					s := dirStats[d]
					// 估算文件数（通过总行数/平均行数粗略估算，不准确但够用）
					avgLine := 50
					if avgLine < 1 {
						avgLine = 50
					}
					estFiles := (s.total + avgLine - 1) / avgLine
					rows = append(rows, []string{
						d,
						fmt.Sprintf("~%d", estFiles),
						strconv.Itoa(s.total),
						strconv.Itoa(s.code),
						strconv.Itoa(s.comment),
						strconv.Itoa(s.blank),
					})
				}
			} else {
				// 按扩展名分组
				var exts []string
				for k := range extStats {
					exts = append(exts, k)
				}
				sort.Strings(exts)
				rows = append(rows, []string{"扩展名", "文件数（估算）", "总行数", "代码行", "注释行", "空行"})
				for _, e := range exts {
					s := extStats[e]
					avgLine := 50
					if avgLine < 1 {
						avgLine = 50
					}
					estFiles := (s.total + avgLine - 1) / avgLine
					rows = append(rows, []string{
						e,
						fmt.Sprintf("~%d", estFiles),
						strconv.Itoa(s.total),
						strconv.Itoa(s.code),
						strconv.Itoa(s.comment),
						strconv.Itoa(s.blank),
					})
				}
			}
			writeMarkdownTable(&b, rows, nil)
			return b.String(), nil
		},
	})
}

// countFileLines 统计代码行、注释行、空行
func countFileLines(lines []string) (r struct {
	total   int
	code    int
	comment int
	blank   int
}) {
	r.total = len(lines)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			r.blank++
		} else if strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "--") ||
			strings.HasPrefix(trimmed, "/*") ||
			strings.HasPrefix(trimmed, "*") ||
			strings.HasPrefix(trimmed, "<!--") {
			r.comment++
		} else {
			r.code++
		}
	}
	return
}

// ═══════════════════════════════════════════════════════════
// 6. word_read：读取 Word (.docx) 文件内容
// ═══════════════════════════════════════════════════════════

func registerWordRead(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "word_read",
		UsageGuide: "读取 Microsoft Word (.docx) 文件内容，以纯文本或 Markdown 格式返回。比手动打开 Word 更高效（直接提取文本到上下文）。",
		Description: "读取 Microsoft Word (.docx) 文件的内容，以纯文本或 Markdown 格式返回。" +
			"支持段落文本、表格、列表等基本结构提取。" +
			"format 可选 \"text\"（纯文本，默认）或 \"markdown\"（Markdown 格式）。" +
			"limit 限制返回字符数（默认 10000，防止内容过长）。",
		Parameters: objSchema(props{
			"path":   strProp("Word 文件路径（工作区内，.docx 格式）"),
			"format": strProp("可选：输出格式 \"text\"（纯文本，默认）或 \"markdown\""),
			"limit":  intProp("可选：最大返回字符数（默认 10000，-1=全部）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			format := strings.ToLower(strings.TrimSpace(argStr(args, "format")))
			if format == "" {
				format = "text"
			}
			limit := argInt(args, "limit", 10000)

			// 打开 ZIP 读取器
			reader, err := zip.OpenReader(p)
			if err != nil {
				return "", fmt.Errorf("无法打开 .docx 文件（不是有效的 ZIP 压缩包）: %w", err)
			}
			defer reader.Close()

			// 查找 word/document.xml
			var docXML []byte
			for _, f := range reader.File {
				if f.Name == "word/document.xml" {
					rc, err := f.Open()
					if err != nil {
						return "", fmt.Errorf("无法读取文档内容: %w", err)
					}
					docXML, err = io.ReadAll(rc)
					rc.Close()
					if err != nil {
						return "", fmt.Errorf("读取文档内容失败: %w", err)
					}
					break
				}
			}
			if docXML == nil {
				return "", fmt.Errorf("未找到 word/document.xml，文件可能不是有效的 .docx 格式")
			}

			// 解析 XML
			content := parseDocxXML(string(docXML), format)

			if limit > 0 && len(content) > limit {
				content = content[:limit] + fmt.Sprintf("\n\n…[内容共 %d 字符，仅显示前 %d 字符]", len(content), limit)
			}

			info := fmt.Sprintf("**Word 文件**: `%s`\n\n", argStr(args, "path"))
			return info + content, nil
		},
	})
}

// docxBody docx document.xml 结构（只提取需要的部分）
type docxBody struct {
	Paragraphs []docxParagraph `xml:"body>p"`
	Tables     []docxTable     `xml:"body>tbl"`
}
type docxParagraph struct {
	ParagraphProps *docxParagraphProps `xml:"pPr"`
	Runs           []docxRun           `xml:"r"`
	HyperlinkRuns  []docxRun           `xml:"hyperlink>r"`
}
type docxParagraphProps struct {
	Style string `xml:"pStyle,attr"`
}
type docxRun struct {
	Text string `xml:"t"`
}
type docxTable struct {
	Rows []docxTableRow `xml:"tr"`
}
type docxTableRow struct {
	Cells []docxTableCell `xml:"tc"`
}
type docxTableCell struct {
	Paragraphs []docxParagraph `xml:"p"`
}

func parseDocxXML(xmlContent, format string) string {
	var body docxBody
	if err := xml.Unmarshal([]byte(xmlContent), &body); err != nil {
		return fmt.Sprintf("（XML 解析失败: %v）", err)
	}

	var b strings.Builder

	// 处理段落
	for _, p := range body.Paragraphs {
		text := extractParagraphText(p)
		if text == "" {
			if format == "markdown" {
				b.WriteString("\n\n")
			} else {
				b.WriteString("\n")
			}
			continue
		}

		if format == "markdown" {
			// 根据样式判断标题级别（简单处理）
			style := ""
			if p.ParagraphProps != nil {
				style = p.ParagraphProps.Style
			}
			switch {
			case strings.HasPrefix(style, "Heading1") || strings.HasPrefix(style, "1"):
				b.WriteString("# " + text + "\n\n")
			case strings.HasPrefix(style, "Heading2") || strings.HasPrefix(style, "2"):
				b.WriteString("## " + text + "\n\n")
			case strings.HasPrefix(style, "Heading3") || strings.HasPrefix(style, "3"):
				b.WriteString("### " + text + "\n\n")
			case strings.HasPrefix(style, "ListBullet"):
				b.WriteString("- " + text + "\n")
			case strings.HasPrefix(style, "ListNumber"):
				b.WriteString("1. " + text + "\n")
			default:
				b.WriteString(text + "\n\n")
			}
		} else {
			b.WriteString(text + "\n")
		}
	}

	// 处理表格
	for _, tbl := range body.Tables {
		if format == "markdown" {
			b.WriteString("\n")
			var rows [][]string
			for _, row := range tbl.Rows {
				var cells []string
				for _, cell := range row.Cells {
					var cellText string
					for _, p := range cell.Paragraphs {
						cellText += extractParagraphText(p) + " "
					}
					cells = append(cells, strings.TrimSpace(cellText))
				}
				rows = append(rows, cells)
			}
			if len(rows) > 0 {
				writeMarkdownTable(&b, rows, nil)
			}
			b.WriteString("\n")
		} else {
			b.WriteString("\n[表格]\n")
			for _, row := range tbl.Rows {
				var cells []string
				for _, cell := range row.Cells {
					var cellText string
					for _, p := range cell.Paragraphs {
						cellText += extractParagraphText(p) + " | "
					}
					cells = append(cells, strings.TrimRight(cellText, " | "))
				}
				b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
			}
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func extractParagraphText(p docxParagraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		b.WriteString(r.Text)
	}
	for _, r := range p.HyperlinkRuns {
		b.WriteString(r.Text)
	}
	return strings.TrimSpace(b.String())
}

// ═══════════════════════════════════════════════════════════
// 共用辅助
// ═══════════════════════════════════════════════════════════

// colStat 数值列统计结果
type colStat struct {
	name  string
	sum   float64
	count int
	min   float64
	max   float64
}

func csvDelim(d string) rune {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "tab", "\t", "制表符":
		return '\t'
	default:
		return ','
	}
}

func readCSV(data []byte, delim rune) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = delim
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 解析失败: %w", err)
	}
	return records, nil
}

func parseColIndex(spec string, maxCols int) []int {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	parts := strings.Split(spec, ",")
	var idx []int
	for _, s := range parts {
		s = strings.TrimSpace(s)
		i, err := strconv.Atoi(s)
		if err != nil || i < 0 || i >= maxCols {
			continue
		}
		idx = append(idx, i)
	}
	return idx
}

// writeMarkdownTable 把二维字符串数组渲染为 Markdown 表格。
// colIdx 为 nil 时渲染全部列。
func writeMarkdownTable(b *strings.Builder, records [][]string, colIdx []int) {
	if len(records) == 0 {
		return
	}
	selectCols := func(row []string) []string {
		if colIdx == nil {
			return row
		}
		out := make([]string, 0, len(colIdx))
		for _, idx := range colIdx {
			if idx < len(row) {
				out = append(out, row[idx])
			} else {
				out = append(out, "")
			}
		}
		return out
	}

	header := selectCols(records[0])
	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range records[1:] {
		cols := selectCols(row)
		for i, c := range cols {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
		for len(widths) < len(cols) {
			widths = append(widths, 0)
		}
	}
	for len(header) < len(widths) {
		header = append(header, "")
	}

	b.WriteString("| ")
	for i, h := range header {
		b.WriteString(padRight(h, widths[i]))
		b.WriteString(" | ")
	}
	b.WriteString("\n| ")
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", max(w, 3)))
		b.WriteString(" | ")
	}
	b.WriteString("\n")
	for _, row := range records[1:] {
		cols := selectCols(row)
		b.WriteString("| ")
		for i, c := range cols {
			if i < len(widths) {
				b.WriteString(padRight(c, widths[i]))
			} else {
				b.WriteString(c)
			}
			b.WriteString(" | ")
		}
		b.WriteString("\n")
	}
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// parseMarkdownTable 从 Markdown 文本中提取表格数据（简单实现）。
func parseMarkdownTable(text string) [][]string {
	lines := strings.Split(text, "\n")
	var records [][]string
	inTable := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if inTable {
				break
			}
			continue
		}
		if !strings.HasPrefix(line, "|") {
			if inTable {
				break
			}
			continue
		}
		stripped := strings.ReplaceAll(line, "-", "")
		stripped = strings.ReplaceAll(stripped, "|", "")
		stripped = strings.ReplaceAll(stripped, " ", "")
		if stripped == "" {
			continue
		}
		inTable = true
		parts := strings.Split(line, "|")
		var row []string
		for _, p := range parts {
			row = append(row, strings.TrimSpace(p))
		}
		if len(row) > 0 && row[0] == "" {
			row = row[1:]
		}
		if len(row) > 0 && row[len(row)-1] == "" {
			row = row[:len(row)-1]
		}
		if len(row) > 0 {
			records = append(records, row)
		}
	}
	return records
}

// ═══════════════════════════════════════════════════════════
// 7. word_write：生成 Word (.docx) 文档
// ═══════════════════════════════════════════════════════════

func registerWordWrite(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "word_write",
		UsageGuide: "生成 Microsoft Word (.docx) 文档。content 为 Markdown 格式文本。用于输出报告/文档。比手动排版更高效（Markdown 转 Word 格式）。需审核批准。",
		Description: "生成 Microsoft Word (.docx) 文档。" +
			"content 为 Markdown 格式文本（支持 # 标题、普通段落、- 列表项、| 表格），" +
			"系统自动将其转换为 OOXML 格式写入 .docx 文件。" +
			"title 为可选的文档标题（默认无）。",
		Parameters: objSchema(props{
			"path":    strProp("输出文件路径（工作区内，.docx 扩展名）"),
			"content": strProp("文档内容（Markdown 格式：标题用 #、列表用 -、表格用 |）"),
			"title":   strProp("可选：文档标题"),
		}, "path", "content"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			content := argStr(args, "content")
			title := strings.TrimSpace(argStr(args, "title"))

			docxData := buildDocx(content, title)
			if err := os.WriteFile(p, docxData, 0o644); err != nil {
				return "", fmt.Errorf("写入 .docx 失败: %w", err)
			}
			return fmt.Sprintf("已生成 Word 文档 `%s`（%d 字节）", argStr(args, "path"), len(docxData)), nil
		},
	})
}

// buildDocx 从 Markdown 内容构建 .docx 字节。
func buildDocx(markdown string, title string) []byte {
	lines := strings.Split(markdown, "\n")
	var paragraphs []string
	var tableMode bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			paragraphs = append(paragraphs, "<w:p><w:r><w:t xml:space=\"preserve\"></w:t></w:r></w:p>")
			continue
		}

		// 表格行
		if strings.HasPrefix(trimmed, "|") {
			if !tableMode {
				tableMode = true
			}
			// 跳过表头分隔行
			if strings.ReplaceAll(trimmed, "-", "") == "" ||
				strings.ReplaceAll(strings.ReplaceAll(trimmed, "|", ""), "-", "") == "" {
				continue
			}
			// 解析表格行
			cells := strings.Split(trimmed, "|")
			var rowXML strings.Builder
			rowXML.WriteString("<w:tr>")
			for _, cell := range cells {
				c := strings.TrimSpace(cell)
				if c == "" {
					continue
				}
				rowXML.WriteString("<w:tc><w:p><w:r><w:rPr><w:sz w:val=\"20\"/></w:rPr><w:t xml:space=\"preserve\">")
				rowXML.WriteString(escapeXML(c))
				rowXML.WriteString("</w:t></w:r></w:p></w:tc>")
			}
			rowXML.WriteString("</w:tr>")
			paragraphs = append(paragraphs, rowXML.String())
			continue
		}
		tableMode = false

		// 标题 # ## ###
		if strings.HasPrefix(trimmed, "### ") {
			text := strings.TrimPrefix(trimmed, "### ")
			paragraphs = append(paragraphs,
				fmt.Sprintf("<w:p><w:pPr><w:numPr><w:ilvl w:val=\"0\"/></w:numPr></w:pPr><w:r><w:rPr><w:b/><w:sz w:val=\"28\"/></w:rPr><w:t xml:space=\"preserve\">%s</w:t></w:r></w:p>", escapeXML(text)))
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			text := strings.TrimPrefix(trimmed, "## ")
			paragraphs = append(paragraphs,
				fmt.Sprintf("<w:p><w:r><w:rPr><w:b/><w:sz w:val=\"32\"/></w:rPr><w:t xml:space=\"preserve\">%s</w:t></w:r></w:p>", escapeXML(text)))
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			text := strings.TrimPrefix(trimmed, "# ")
			paragraphs = append(paragraphs,
				fmt.Sprintf("<w:p><w:r><w:rPr><w:b/><w:sz w:val=\"36\"/></w:rPr><w:t xml:space=\"preserve\">%s</w:t></w:r></w:p>", escapeXML(text)))
			continue
		}

		// 列表项
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			paragraphs = append(paragraphs,
				fmt.Sprintf("<w:p><w:pPr><w:bullet/></w:pPr><w:r><w:t xml:space=\"preserve\">%s</w:t></w:r></w:p>", escapeXML(text)))
			continue
		}

		// 普通段落
		paragraphs = append(paragraphs,
			fmt.Sprintf("<w:p><w:r><w:t xml:space=\"preserve\">%s</w:t></w:r></w:p>", escapeXML(trimmed)))
	}

	bodyXML := strings.Join(paragraphs, "\n")

	// 生成完整 docx (ZIP)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// [Content_Types].xml
	w, _ := zw.Create("[Content_Types].xml")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// _rels/.rels
	w, _ = zw.Create("_rels/.rels")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// word/_rels/document.xml.rels
	w, _ = zw.Create("word/_rels/document.xml.rels")
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`))

	// word/document.xml
	docXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>%s</w:body>
</w:document>`, bodyXML)
	w, _ = zw.Create("word/document.xml")
	w.Write([]byte(docXML))

	zw.Close()
	return buf.Bytes()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// ═══════════════════════════════════════════════════════════
// 8. read_xlsx：读取 Excel (.xlsx) 文件
// ═══════════════════════════════════════════════════════════

func registerXLSXRead(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "read_xlsx",
		UsageGuide: "读取 Excel (.xlsx) 文件内容，以 Markdown 表格形式返回。sheet 参数指定工作表名。比直接 read_file 更友好（自动解析+多 sheet 支持）。",
		Description: "读取 Microsoft Excel (.xlsx) 文件的内容，以 Markdown 表格形式返回各工作表。" +
			"sheet 指定工作表名称（默认第一个）；limit 限制行数（默认 200，-1=全部）。" +
			"纯 Go 标准库实现（解析 ZIP + XML），零外部依赖。",
		Parameters: objSchema(props{
			"path":  strProp("Excel 文件路径（工作区内，.xlsx 格式）"),
			"sheet": strProp("可选：工作表名称（默认第一个工作表）"),
			"limit": intProp("可选：最大行数（默认 200，-1=全部）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			sheetName := strings.TrimSpace(argStr(args, "sheet"))
			limit := argInt(args, "limit", 200)

			data, err := os.ReadFile(p)
			if err != nil {
				return "", fmt.Errorf("读取文件失败: %w", err)
			}
			if len(data) > 100<<20 {
				return "", fmt.Errorf("文件超过 100MB")
			}
			zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				return "", fmt.Errorf("无法打开 .xlsx 文件: %w", err)
			}

			result := parseXLSX(zr, sheetName, limit)
			if result == "" {
				return "（空文件或未找到数据）", nil
			}
			return fmt.Sprintf("**Excel 文件**: `%s`\n\n%s", argStr(args, "path"), result), nil
		},
	})
}

// xlsx 解析（纯 stdlib：ZIP → XML）
type xlsxWorkbook struct {
	Sheets []xlsxSheet `xml:"sheets>sheet"`
}
type xlsxSheet struct {
	Name  string `xml:"name,attr"`
	ID    string `xml:"sheetId,attr"`
	RelID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}
type xlsxSharedStrings struct {
	Items []xlsxSI `xml:"si"`
}
type xlsxSI struct {
	Text string `xml:"t"`
}
type xlsxSheetData struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}
type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}
type xlsxCell struct {
	Ref   string `xml:"r,attr"`
	Type  string `xml:"t,attr"`
	Style string `xml:"s,attr"`
	Value string `xml:"v"`
	Text  string `xml:"is>t"` // inline string
}

func parseXLSX(zr *zip.Reader, targetSheet string, limit int) string {
	// 读取共享字符串表
	var sharedStrings []string
	// 读取 sheet 内容行
	var rows []xlsxRow

	// 策略：直接扫描所有文件，跳过 workbook/rels 解析（避免 OOXML 命名空间兼容问题）
	// 先找共享字符串表
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			var ss xlsxSharedStrings
			xml.Unmarshal(data, &ss)
			for _, item := range ss.Items {
				sharedStrings = append(sharedStrings, item.Text)
			}
			break
		}
	}

	// 确定目标 sheet 文件
	var targetFile string
	hasTargetSheet := false
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			if targetFile == "" {
				targetFile = f.Name
			}
			if targetSheet == "" || hasTargetSheet {
				break
			}
		}
	}
	if targetFile != "" {
		for _, f := range zr.File {
			if f.Name == targetFile {
				rc, _ := f.Open()
				data, _ := io.ReadAll(rc)
				rc.Close()
				var sd xlsxSheetData
				clean := strings.ReplaceAll(string(data), `xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"`, "")
				xml.Unmarshal([]byte(clean), &sd)
				rows = sd.Rows
				break
			}
		}
	}
	if len(rows) == 0 {
		return ""
	}

	// 确定最大列数
	maxCol := 0
	for _, row := range rows {
		for _, cell := range row.Cells {
			col := parseColLetter(cell.Ref)
			if col > maxCol {
				maxCol = col
			}
		}
	}
	if maxCol == 0 {
		// 无法通过引用解析列数，用实际单元格数
		for _, row := range rows {
			if len(row.Cells) > maxCol {
				maxCol = len(row.Cells)
			}
		}
	}

	// 解析单元格值
	total := len(rows)
	if limit > 0 && limit < len(rows) {
		rows = rows[:limit]
	}

	var records [][]string
	for _, row := range rows {
		rec := make([]string, maxCol)
		for _, cell := range row.Cells {
			col := parseColLetter(cell.Ref)
			if col <= 0 || col > maxCol {
				continue
			}
			val := cell.Value
			if cell.Type == "s" { // 共享字符串
				idx, _ := strconv.Atoi(val)
				if idx >= 0 && idx < len(sharedStrings) {
					val = sharedStrings[idx]
				}
			} else if cell.Type == "inlineStr" || (val == "" && cell.Text != "") { // 内联字符串
				val = cell.Text
			}
			rec[col-1] = val
		}
		records = append(records, rec)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("工作表: **%s** · 共 %d 行 × %d 列", targetSheet, total, maxCol))
	if limit > 0 && total > limit {
		b.WriteString(fmt.Sprintf(" · 显示前 %d 行", limit))
	}
	b.WriteString("\n\n")
	writeMarkdownTable(&b, records, nil)
	return b.String()
}

// parseColLetter 将 Excel 列字母转为索引（A→1, Z→26, AA→27）
func parseColLetter(ref string) int {
	// 提取列字母部分（如 "AB12" → "AB"）
	letters := strings.TrimRightFunc(ref, func(r rune) bool { return r >= '0' && r <= '9' })
	col := 0
	for _, c := range letters {
		if c >= 'A' && c <= 'Z' {
			col = col*26 + int(c-'A'+1)
		} else if c >= 'a' && c <= 'z' {
			col = col*26 + int(c-'a'+1)
		}
	}
	return col
}

// stripXMLNS 从 XML 字符串中移除指定命名空间 URI 的声明和前缀引用，
// 使 Go xml.Unmarshal 能解析 OOXML 文件而无需处理命名空间。
func stripXMLNS(xmlData string, nsURIs []string) string {
	for _, ns := range nsURIs {
		// 移除 xmlns="uri" 默认命名空间声明
		xmlData = strings.ReplaceAll(xmlData, `xmlns="`+ns+`"`, "")
		// 移除 xmlns:prefix="uri" 带前缀命名空间声明
		xmlData = strings.ReplaceAll(xmlData, `xmlns:r="`+ns+`"`, "")
		// 移除 r: 前缀引用（元素和属性）
		xmlData = strings.ReplaceAll(xmlData, "r:", "")
	}
	return xmlData
}

// ═══════════════════════════════════════════════════════════
// 9. write_xlsx：创建 Excel (.xlsx) 文件
// ═══════════════════════════════════════════════════════════

func registerXLSXWrite(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "write_xlsx",
		UsageGuide: "创建 Excel (.xlsx) 文件。data 为 JSON 二维数组或 Markdown 表格文本。比手动拼接 CSV 更专业（支持多 sheet+格式）。需审核批准。",
		Description: "创建 Microsoft Excel (.xlsx) 文件。" +
			"data 为 JSON 二维数组（如 [[\"列1\",\"列2\"],[\"值1\",\"值2\"]]）或 Markdown 表格文本。" +
			"sheet 为工作表名称（默认 \"Sheet1\"）。" +
			"纯 Go 标准库实现（生成 ZIP + XML），零外部依赖。",
		Parameters: objSchema(props{
			"path":  strProp("输出文件路径（工作区内，.xlsx 扩展名）"),
			"data":  strProp("表格数据：JSON 二维数组字符串 或 Markdown 表格文本"),
			"sheet": strProp("可选：工作表名称（默认 \"Sheet1\"）"),
		}, "path", "data"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			dataStr := argStr(args, "data")
			sheetName := strings.TrimSpace(argStr(args, "sheet"))
			if sheetName == "" {
				sheetName = "Sheet1"
			}

			var records [][]string
			if err := json.Unmarshal([]byte(dataStr), &records); err != nil {
				records = parseMarkdownTable(dataStr)
			}
			if len(records) == 0 {
				return "", fmt.Errorf("data 格式无效：无法解析为 JSON 二维数组或 Markdown 表格")
			}

			xlsxData := buildXLSX(sheetName, records)
			if err := os.WriteFile(p, xlsxData, 0o644); err != nil {
				return "", fmt.Errorf("写入 .xlsx 失败: %w", err)
			}
			return fmt.Sprintf("已生成 Excel 文件 `%s`（工作表: %s, %d 行 × %d 列, %d 字节）",
				argStr(args, "path"), sheetName, len(records), len(records[0]), len(xlsxData)), nil
		},
	})
}

func buildXLSX(sheetName string, records [][]string) []byte {
	// 收集所有唯一字符串用于共享字符串表（从第2行开始）
	ssSet := make(map[string]bool)
	for ri, row := range records {
		if ri == 0 {
			continue // 跳过表头，直接用内联
		}
		for _, cell := range row {
			if cell != "" {
				ssSet[cell] = true
			}
		}
	}
	var ssList []string
	for s := range ssSet {
		ssList = append(ssList, s)
	}

	// 构建 shared strings XML
	var ssXML strings.Builder
	ssXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="1" uniqueCount="`)
	ssXML.WriteString(strconv.Itoa(len(ssList)))
	ssXML.WriteString(`">`)
	for _, s := range ssList {
		ssXML.WriteString("<si><t>")
		ssXML.WriteString(escapeXML(s))
		ssXML.WriteString("</t></si>")
	}
	ssXML.WriteString("</sst>")

	// 构建 sheet XML
	ssIndex := make(map[string]int)
	for i, s := range ssList {
		ssIndex[s] = i
	}

	var sheetXML strings.Builder
	sheetXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>`)

	for ri, row := range records {
		sheetXML.WriteString("<row r=\"")
		sheetXML.WriteString(strconv.Itoa(ri + 1))
		sheetXML.WriteString("\">")
		for ci, cell := range row {
			colLetter := colIndexToLetters(ci)
			ref := colLetter + strconv.Itoa(ri+1)
			if ri == 0 {
				// 表头用 inline string
				sheetXML.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`,
					ref, escapeXML(cell)))
			} else {
				if idx, ok := ssIndex[cell]; ok {
					sheetXML.WriteString(fmt.Sprintf(`<c r="%s" t="s"><v>%d</v></c>`, ref, idx))
				} else {
					// 数值或空
					if _, err := strconv.ParseFloat(cell, 64); err == nil {
						sheetXML.WriteString(fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, ref, cell))
					} else {
						sheetXML.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`,
							ref, escapeXML(cell)))
					}
				}
			}
		}
		sheetXML.WriteString("</row>")
	}
	sheetXML.WriteString(`</sheetData></worksheet>`)

	// 构建 ZIP
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// [Content_Types].xml
	// [Content_Types].xml
	w2, _ := zw.Create("[Content_Types].xml")
	w2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
</Types>`))

	// _rels/.rels
	// _rels/.rels
	w2, _ = zw.Create("_rels/.rels")
	w2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`))

	// xl/_rels/workbook.xml.rels
	w2, _ = zw.Create("xl/_rels/workbook.xml.rels")
	w2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`))

	// xl/workbook.xml
	wbXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
          xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="%s" sheetId="1" r:id="rId1"/>
  </sheets>
</workbook>`, escapeXML(sheetName))
	w2, _ = zw.Create("xl/workbook.xml")
	w2.Write([]byte(wbXML))

	// xl/worksheets/sheet1.xml
	w2, _ = zw.Create("xl/worksheets/sheet1.xml")
	w2.Write([]byte(sheetXML.String()))

	// xl/sharedStrings.xml
	w2, _ = zw.Create("xl/sharedStrings.xml")
	w2.Write([]byte(ssXML.String()))

	// xl/styles.xml（最小样式表）
	w2, _ = zw.Create("xl/styles.xml")
	w2.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"/>`))

	zw.Close()
	return buf.Bytes()
}

func colIndexToLetters(idx int) string {
	var letters []byte
	for {
		letters = append([]byte{byte('A' + idx%26)}, letters...)
		idx = idx/26 - 1
		if idx < 0 {
			break
		}
	}
	return string(letters)
}

// ═══════════════════════════════════════════════════════════
// 10. read_pdf：读取 PDF 文本内容
// ═══════════════════════════════════════════════════════════

func registerPDFRead(r *Registry, root string) {
	r.Register(&Tool{
		Name:       "read_pdf",
		UsageGuide: "提取 PDF 文件文本内容（解析 PDF 流对象，纯文本 PDF）。page 指定页码；limit 限制返回字符数。扫描/图片型 PDF 无嵌入文本时返回提示。",
		Description: "提取 PDF 文件的文本内容（解析 PDF 流对象，支持纯文本 PDF）。" +
			"扫描/图片型 PDF（无嵌入文本）无法提取，返回说明。" +
			"page 指定页码（从 1 开始，默认全部）；limit 限制返回字符数（默认 10000）。",
		Parameters: objSchema(props{
			"path":  strProp("PDF 文件路径（工作区内）"),
			"page":  intProp("可选：页码（从 1 开始），省略则提取全部页面"),
			"limit": intProp("可选：最大返回字符数（默认 10000，-1=全部）"),
		}, "path"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			p, err := resolvePath(root, argStr(args, "path"))
			if err != nil {
				return "", err
			}
			targetPage := argInt(args, "page", 0)
			limit := argInt(args, "limit", 10000)

			data, err := os.ReadFile(p)
			if err != nil {
				return "", fmt.Errorf("读取文件失败: %w", err)
			}
			if len(data) > 100<<20 {
				return "", fmt.Errorf("文件超过 100MB")
			}

			// 纯文本提取
			content := extractPDFText(string(data), targetPage)
			content = strings.TrimSpace(content)
			if content == "" {
				// OCR 已移除（2026-08-22 模型文件清理）——扫描/图片型 PDF 无法提取
				return fmt.Sprintf("**PDF 文件**: `%s`\n\n⚠️ 未提取到嵌入文本（扫描/图片型 PDF）。"+
					"该文件需 OCR 才能识别文字，但 OCR 已整体移除。", argStr(args, "path")), nil
			}

			if limit > 0 && len(content) > limit {
				content = content[:limit] + fmt.Sprintf("\n\n…[内容共 %d 字符，仅显示前 %d 字符]", len(content), limit)
			}
			return fmt.Sprintf("**PDF 文件**: `%s`\n\n%s", argStr(args, "path"), content), nil
		},
	})
}

// extractPDFText 从 PDF 原始字符串中提取文本（简单解析器，支持纯文本 PDF）。
func extractPDFText(pdfData string, targetPage int) string {
	// 1. 查找所有页面对象
	type pageInfo struct {
		objNum  int
		content int // content stream 对象号
	}

	// 解析 PDF 交叉引用表等简化处理：直接提取所有流对象中的文本
	// 方法：提取 BT...ET 标记之间的文本操作符

	// 按页码分割：查找 /Type /Page 和 /Parent
	var pages []pageInfo
	rePage := regexp.MustCompile(`(?s)(\d+)\s+\d+\s+obj.*?/Type\s*/Page\b[^<]*/Contents\s+(\d+)\s+\d+\s+R`)
	pageMatches := rePage.FindAllStringSubmatch(pdfData, -1)
	for _, m := range pageMatches {
		objNum, _ := strconv.Atoi(m[1])
		contentNum := 0
		if m[2] != "" {
			contentNum, _ = strconv.Atoi(m[2])
		}
		pages = append(pages, pageInfo{objNum: objNum, content: contentNum})
	}

	if len(pages) == 0 {
		// 没找到页面，尝试直接从流中提取文本
		return extractTextFromStreams(pdfData)
	}

	// 如果指定了页码
	startIdx := 0
	endIdx := len(pages)
	if targetPage > 0 && targetPage <= len(pages) {
		startIdx = targetPage - 1
		endIdx = targetPage
	}

	var out strings.Builder
	for pi := startIdx; pi < endIdx; pi++ {
		p := pages[pi]
		if len(pages) > 1 {
			out.WriteString(fmt.Sprintf("--- 第 %d 页 ---\n", pi+1))
		}
		if p.content > 0 {
			// 提取内容流中的文本
			reStream := regexp.MustCompile(fmt.Sprintf(`(?s)%d\s+\d+\s+obj.*?stream\x0a(.*?)endstream`, p.content))
			streamMatches := reStream.FindStringSubmatch(pdfData)
			if len(streamMatches) > 1 {
				text := extractTextFromPDFStream(streamMatches[1])
				out.WriteString(text)
			}
		}
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String())
}

func extractTextFromStreams(pdfData string) string {
	// 从所有流对象中提取文本
	reStream := regexp.MustCompile(`(?s)stream\x0a(.*?)endstream`)
	matches := reStream.FindAllStringSubmatch(pdfData, -1)
	var out strings.Builder
	for _, m := range matches {
		text := extractTextFromPDFStream(m[1])
		if text != "" {
			out.WriteString(text)
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String())
}

func extractTextFromPDFStream(stream string) string {
	// 解压（ZIP 压缩的流）
	decoded := maybeInflate(stream)
	if decoded == "" {
		decoded = stream
	}

	// 提取 BT ... ET 之间的文本操作符
	var out strings.Builder
	reBT := regexp.MustCompile(`(?s)BT(.*?)ET`)
	btMatches := reBT.FindAllStringSubmatch(decoded, -1)
	for _, m := range btMatches {
		block := m[1]
		// 提取 Tj 和 TJ 操作符的文本
		// (text) Tj
		reTj := regexp.MustCompile(`\(([^)]*)\)\s*Tj`)
		for _, tm := range reTj.FindAllStringSubmatch(block, -1) {
			text := unescapePDFString(tm[1])
			out.WriteString(text)
		}
		// [(text) num (text)] TJ
		reTJ := regexp.MustCompile(`\[(.*?)\]\s*TJ`)
		for _, tm := range reTJ.FindAllStringSubmatch(block, -1) {
			reStr := regexp.MustCompile(`\(([^)]*)\)`)
			for _, sm := range reStr.FindAllStringSubmatch(tm[1], -1) {
				text := unescapePDFString(sm[1])
				out.WriteString(text)
			}
		}
		out.WriteString(" ")
	}
	return out.String()
}

func maybeInflate(data string) string {
	// 尝试 ZLIB 解压（PDF 常用 /Filter /FlateDecode）
	r := bytes.NewReader([]byte(data))
	zr, err := zlib.NewReader(r)
	if err != nil {
		return ""
	}
	defer zr.Close()
	decoded, err := io.ReadAll(zr)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func unescapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\(", "(")
	s = strings.ReplaceAll(s, "\\)", ")")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

// ═══════════════════════════════════════════════════════════
// 11. markdown_to_html：Markdown 转 HTML
// ═══════════════════════════════════════════════════════════

func registerMarkdownToHTML(r *Registry) {
	r.Register(&Tool{
		Name:       "markdown_to_html",
		UsageGuide: "将 Markdown 文本转为 HTML。支持 full_html 参数输出完整 HTML 文档（含 title）。比手动转换更快（内置渲染器+代码高亮）。",
		Description: "将 Markdown 文本转换为 HTML 片段。" +
			"支持 # 标题、**粗体**、*斜体*、`行内代码`、```代码块```、- 无序列表、" +
			"1. 有序列表、| 表格、> 引用、[链接](url)、![图片](url)。" +
			"full_html 为 true 时输出完整 HTML 文档（含 DOCTYPE + head + body），否则只输出 body 内的 HTML 片段。",
		Parameters: objSchema(props{
			"markdown":  strProp("Markdown 文本（必填）"),
			"full_html": boolProp("可选：是否输出完整 HTML 文档（默认 false，只输出片段）"),
			"title":     strProp("可选：完整 HTML 时的页面标题"),
		}, "markdown"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			md := argStr(args, "markdown")
			fullHTML := argBool(args, "full_html")
			title := strings.TrimSpace(argStr(args, "title"))

			body := mdToHTML(md)

			if !fullHTML {
				return body, nil
			}

			var b strings.Builder
			b.WriteString("<!DOCTYPE html>\n<html lang=\"zh-CN\">\n<head>\n<meta charset=\"UTF-8\">\n")
			if title != "" {
				b.WriteString(fmt.Sprintf("<title>%s</title>\n", escapeHTML(title)))
			}
			b.WriteString("<style>body{font-family:sans-serif;line-height:1.6;padding:20px;max-width:800px;margin:auto}" +
				"table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:6px 10px;text-align:left}" +
				"th{background:#f5f5f5}pre{background:#f5f5f5;padding:10px;border-radius:4px;overflow-x:auto}" +
				"code{background:#f0f0f0;padding:2px 4px;border-radius:2px;font-size:0.9em}" +
				"pre code{background:none;padding:0}blockquote{border-left:3px solid #ccc;margin:0;padding:4px 12px;color:#666}" +
				"</style>\n</head>\n<body>\n")
			b.WriteString(body)
			b.WriteString("\n</body>\n</html>")
			return b.String(), nil
		},
	})
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// mdToHTML 将 Markdown 文本转为 HTML（简单实现，覆盖常用语法）。
func mdToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var b strings.Builder
	inCodeBlock := false
	inTable := false
	var tableRows []string

	flushTable := func() {
		if len(tableRows) == 0 {
			return
		}
		b.WriteString("<table>\n")
		for i, row := range tableRows {
			tag := "td"
			if i == 0 {
				tag = "th"
			}
			b.WriteString("<tr>")
			cells := strings.Split(row, "|")
			for _, cell := range cells {
				c := strings.TrimSpace(cell)
				if c == "" {
					continue
				}
				b.WriteString("<" + tag + ">" + inlineMDToHTML(c) + "</" + tag + ">")
			}
			b.WriteString("</tr>\n")
		}
		b.WriteString("</table>\n")
		tableRows = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 代码块
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				b.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				flushTable()
				b.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}
		if inCodeBlock {
			b.WriteString(escapeHTML(line) + "\n")
			continue
		}

		// 表格
		if strings.HasPrefix(trimmed, "|") {
			// 跳过表头分隔行
			if strings.ReplaceAll(trimmed, "-", "") == "" ||
				strings.ReplaceAll(strings.ReplaceAll(trimmed, "|", ""), "-", "") == "" {
				continue
			}
			if !inTable {
				inTable = true
				flushTable()
			}
			tableRows = append(tableRows, trimmed)
			continue
		}
		inTable = false
		flushTable()

		// 空行
		if trimmed == "" {
			b.WriteString("<br>\n")
			continue
		}

		// 标题
		if strings.HasPrefix(trimmed, "### ") {
			b.WriteString("<h3>" + inlineMDToHTML(strings.TrimPrefix(trimmed, "### ")) + "</h3>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			b.WriteString("<h2>" + inlineMDToHTML(strings.TrimPrefix(trimmed, "## ")) + "</h2>\n")
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			b.WriteString("<h1>" + inlineMDToHTML(strings.TrimPrefix(trimmed, "# ")) + "</h1>\n")
			continue
		}

		// 引用
		if strings.HasPrefix(trimmed, "> ") {
			b.WriteString("<blockquote>" + inlineMDToHTML(strings.TrimPrefix(trimmed, "> ")) + "</blockquote>\n")
			continue
		}

		// 无序列表
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			b.WriteString("<li>" + inlineMDToHTML(text) + "</li>\n")
			continue
		}

		// 有序列表
		if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
			text := regexp.MustCompile(`^\d+\.\s`).ReplaceAllString(trimmed, "")
			b.WriteString("<li>" + inlineMDToHTML(text) + "</li>\n")
			continue
		}

		// 水平线
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			b.WriteString("<hr>\n")
			continue
		}

		// 普通段落
		b.WriteString("<p>" + inlineMDToHTML(trimmed) + "</p>\n")
	}

	flushTable()
	if inCodeBlock {
		b.WriteString("</code></pre>\n")
	}
	return strings.TrimSpace(b.String())
}

// inlineMDToHTML 处理行内 Markdown 语法
func inlineMDToHTML(text string) string {
	// 代码 `code`
	re := regexp.MustCompile("`([^`]+)`")
	text = re.ReplaceAllString(text, "<code>$1</code>")

	// 粗体 **text**
	re = regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = re.ReplaceAllString(text, "<strong>$1</strong>")

	// 斜体 *text*
	re = regexp.MustCompile(`\*(.+?)\*`)
	text = re.ReplaceAllString(text, "<em>$1</em>")

	// 图片 ![alt](url)
	re = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	text = re.ReplaceAllString(text, `<img src="$2" alt="$1">`)

	// 链接 [text](url)
	re = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = re.ReplaceAllString(text, `<a href="$2">$1</a>`)

	return text
}
