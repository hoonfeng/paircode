package agent

import (
	"fmt"
	"strings"
)

// ─── 文件编辑匹配器 ──────────────────────────────────────────
// 解决 edit_file/multi_edit「找不到原文本」问题：
//   1. CRLF vs LF：LLM 产出的 old_string 通常只有 \n，而 Windows 文件含 \r\n，字节级匹配失败。
//   2. 空白/缩进差异：LLM 漏掉/多加空格、tab vs space，字节级不匹配。
//   3. 要求逐字节复述原文，违反 LLM tokenizer 特性。
//
// 匹配策略（按优先级）：
//   行号定位 → 精确匹配 → CRLF 归一化匹配 → 空白折叠匹配 → 失败诊断
//
// 关键：匹配阶段可归一化，但替换阶段保留文件原有换行风格（CRLF 文件替换后仍 CRLF）。

// EditOptions 文本编辑参数。
type EditOptions struct {
	OldString string // 待替换原文（与 LineStart/LineEnd 二选一）
	NewString string // 替换后新文
	LineStart int    // 1 基行号，>0 时启用行号定位模式
	LineEnd   int    // 1 基行号（含）；0 或 < LineStart 时只替换 LineStart 一行
}

// ApplyEdit 在 content 上应用一次编辑，返回编辑后的完整内容。
// 全部匹配策略失败时返回带行号上下文的诊断 error，帮助 LLM 下一轮纠正。
func ApplyEdit(content string, opts EditOptions) (string, error) {
	// ── 行号定位模式 ──
	if opts.LineStart > 0 {
		return applyEditByLine(content, opts)
	}

	old := opts.OldString
	if old == "" {
		return "", fmt.Errorf("old_string 不能为空（或改用 line_start/line_end 行号定位）")
	}

	// 1. 精确匹配（原始字节）——最快、零风险
	if n := strings.Count(content, old); n == 1 {
		out := strings.Replace(content, old, opts.NewString, 1)
		return restoreNewlines(out, content), nil // new_string 的换行符对齐文件风格
	} else if n > 1 {
		return "", diagnoseMultiple(content, old)
	}

	// 2. CRLF→LF 归一化匹配——解决 Windows 文件 \r\n 与 LLM 给的 \n 不匹配
	normContent := normalizeNewlines(content)
	normOld := normalizeNewlines(old)
	if n := strings.Count(normContent, normOld); n == 1 {
		normNew := normalizeNewlines(opts.NewString)
		out := strings.Replace(normContent, normOld, normNew, 1)
		return restoreNewlines(out, content), nil
	} else if n > 1 {
		return "", diagnoseMultiple(normContent, normOld)
	}

	// 3. 空白折叠匹配——解决缩进/行尾空白/tab vs space 差异
	if out, ok := matchWhitespaceFold(normContent, normOld, normalizeNewlines(opts.NewString)); ok {
		return restoreNewlines(out, content), nil
	}

	// 4. 全部失败：诊断（带行号上下文，帮 LLM 纠正）
	return "", diagnoseNotFound(content, old)
}

// applyEditByLine 行号定位模式：按 line_start..line_end 替换整段。
// 若同时提供 old_string，校验该行段是否（空白容忍）匹配 old_string。
func applyEditByLine(content string, opts EditOptions) (string, error) {
	normContent := normalizeNewlines(content)
	lines := strings.Split(normContent, "\n")

	start := opts.LineStart - 1 // 转 0 基
	if start < 0 {
		start = 0
	}
	end := start // 默认只替换一行
	if opts.LineEnd > start {
		end = opts.LineEnd - 1
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}
	if start >= len(lines) {
		return "", fmt.Errorf("line_start %d 超出文件行数 %d", opts.LineStart, len(lines))
	}

	// 若提供 old_string，校验该行段是否（空白容忍）匹配
	if opts.OldString != "" {
		actual := strings.Join(lines[start:end+1], "\n")
		if !foldEqualMulti(actual, opts.OldString) {
			return "", fmt.Errorf("line %d-%d 内容与 old_string 不匹配（空白折叠后比较）\n期望: %s\n实际: %s",
				opts.LineStart, end+1, foldPreview(opts.OldString), foldPreview(actual))
		}
	}

	normNew := normalizeNewlines(opts.NewString)
	// 用 \n 拼接，前后各保留换行；首行/末行边界处理
	var b strings.Builder
	if start > 0 {
		b.WriteString(strings.Join(lines[:start], "\n"))
		b.WriteString("\n")
	}
	b.WriteString(normNew)
	if end+1 < len(lines) {
		b.WriteString("\n")
		b.WriteString(strings.Join(lines[end+1:], "\n"))
	}
	return restoreNewlines(b.String(), content), nil
}

// matchWhitespaceFold 在 normContent 中用空白折叠方式找 normOld 的唯一连续匹配，替换为 normNew。
// 折叠规则：每行 TrimSpace + 行内连续空白（含 tab）折叠为单空格。
// 命中且唯一时返回替换后的内容；不唯一或未命中返回 ok=false。
func matchWhitespaceFold(normContent, normOld, normNew string) (string, bool) {
	contentLines := strings.Split(normContent, "\n")
	oldLines := strings.Split(normOld, "\n")
	if len(oldLines) == 0 || len(oldLines) > len(contentLines) {
		return "", false
	}

	foldContent := make([]string, len(contentLines))
	for i, l := range contentLines {
		foldContent[i] = foldWS(l)
	}
	foldOld := make([]string, len(oldLines))
	for i, l := range oldLines {
		foldOld[i] = foldWS(l)
	}
	// old 全为空行时无法可靠匹配，跳过
	allEmpty := true
	for _, f := range foldOld {
		if f != "" {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		return "", false
	}

	// 在 foldContent 中找 foldOld 的连续子序列匹配
	start, count := -1, 0
	for i := 0; i+len(oldLines) <= len(contentLines); i++ {
		match := true
		for j := range oldLines {
			if foldContent[i+j] != foldOld[j] {
				match = false
				break
			}
		}
		if match {
			count++
			if count == 1 {
				start = i
			}
			if count > 1 {
				return "", false // 不唯一，放弃
			}
		}
	}
	if count != 1 {
		return "", false
	}

	// 替换 contentLines[start:start+len(oldLines)] 为 normNew 的行
	end := start + len(oldLines)
	newLines := strings.Split(normNew, "\n")
	result := make([]string, 0, len(contentLines)-len(oldLines)+len(newLines))
	result = append(result, contentLines[:start]...)
	result = append(result, newLines...)
	result = append(result, contentLines[end:]...)
	return strings.Join(result, "\n"), true
}

// ─── 换行符工具 ─────────────────────────────────────────────

// normalizeNewlines 把 \r\n 和孤立 \r 统一为 \n（仅匹配阶段用）。
func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// restoreNewlines 把 out 的换行符恢复为 ref 的风格（ref 含 \r\n 则转 CRLF，否则保持 LF）。
// 保证替换后文件换行风格不变。
func restoreNewlines(out, ref string) string {
	if strings.Contains(ref, "\r\n") {
		// ref 是 CRLF：先把 out 里已有的 \r\n 降级为 \n，再统一升为 \r\n
		out = strings.ReplaceAll(out, "\r\n", "\n")
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return out
}

// ─── 空白折叠工具 ───────────────────────────────────────────

// foldWS 折叠行内空白：TrimSpace + 连续空白（空格/tab）→单空格。用于空白容忍比较。
func foldWS(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

// foldEqualMulti 多行空白折叠比较：a 和 b 逐行 foldWS 后完全相等。
func foldEqualMulti(a, b string) bool {
	aLines := strings.Split(normalizeNewlines(a), "\n")
	bLines := strings.Split(normalizeNewlines(b), "\n")
	if len(aLines) != len(bLines) {
		return false
	}
	for i := range aLines {
		if foldWS(aLines[i]) != foldWS(bLines[i]) {
			return false
		}
	}
	return true
}

// foldPreview 折叠后预览（截断到 80 字符），用于诊断信息。
func foldPreview(s string) string {
	f := foldWS(s)
	if len(f) > 80 {
		return f[:80] + "…"
	}
	return f
}

// ─── 诊断信息 ───────────────────────────────────────────────

// diagnoseNotFound 生成「未找到 old_string」诊断：含 old 首行 + 文件中相似行（带行号）。
// 帮助 LLM 下一轮纠正（无需重新 read_file 全文）。
func diagnoseNotFound(content, old string) error {
	normContent := normalizeNewlines(content)
	normOld := normalizeNewlines(old)
	contentLines := strings.Split(normContent, "\n")
	oldLines := strings.Split(normOld, "\n")
	oldFirst := ""
	if len(oldLines) > 0 {
		oldFirst = strings.TrimSpace(oldLines[0])
	}

	var b strings.Builder
	b.WriteString("未找到 old_string（精确/CRLF归一化/空白折叠 均未命中）。\n")
	if oldFirst != "" {
		b.WriteString("old_string 首行: " + oldFirst + "\n")
	}
	b.WriteString("文件中相似行（折叠后包含 old 首行关键词）:\n")

	found := 0
	oldKey := foldWS(oldFirst)
	for i, l := range contentLines {
		if oldKey != "" && strings.Contains(foldWS(l), oldKey) {
			fmt.Fprintf(&b, "  L%d: %s\n", i+1, strings.TrimSpace(l))
			found++
			if found >= 10 {
				break
			}
		}
	}
	if found == 0 {
		// 兜底：显示文件前 10 行带行号，帮 LLM 定位
		b.WriteString("（无相似行，文件前 10 行:）\n")
		limit := 10
		if limit > len(contentLines) {
			limit = len(contentLines)
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(&b, "  L%d: %s\n", i+1, strings.TrimSpace(contentLines[i]))
		}
	}
	b.WriteString("\n建议：1) 用 read_file 重新获取最新内容并逐字复制 old_string；2) 改用 line_start/line_end 行号定位（更可靠）；3) 检查缩进 tab/空格、行尾空白是否一致。")
	return fmt.Errorf("%s", b.String())
}

// diagnoseMultiple 生成「old_string 出现多次」诊断：列出所有命中起始行号。
func diagnoseMultiple(content, old string) error {
	normContent := normalizeNewlines(content)
	normOld := normalizeNewlines(old)

	var positions []int
	searchFrom := 0
	for {
		idx := strings.Index(normContent[searchFrom:], normOld)
		if idx < 0 {
			break
		}
		absIdx := searchFrom + idx
		lineNo := strings.Count(normContent[:absIdx], "\n") + 1
		positions = append(positions, lineNo)
		searchFrom = absIdx + len(normOld)
		if len(positions) >= 20 {
			break
		}
	}
	return fmt.Errorf("old_string 出现 %d 次（不唯一），命中起始行号: %v。请提供更长上下文（多包含前后行）使其唯一，或改用 line_start/line_end 行号定位", len(positions), positions)
}
