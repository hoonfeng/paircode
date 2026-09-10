// apply_patch.go — 自由格式文件补丁（对齐 codex apply_patch 语法）。
//
// ★ 2026-09 工具重构（Phase B）：引入 codex 式 apply_patch 作为主编辑工具，
// 取代 edit/multi_edit（旧 JSON old_string 模式 + 多级模糊匹配补丁链）。
//
// 语法（对齐 codex-rs/core/assets/tools/apply_patch.lark）：
//
//	*** Begin Patch
//	*** Add File: path            ← 新文件（+ 行）
//	+line
//	*** Update File: path         ← 修改现有文件
//	*** Move to: newpath          ← 可选：更新后移动
//	@@ 可选上下文说明
//	 context line
//	-removed line
//	+added line
//	*** Delete File: path         ← 删除文件
//	*** End Patch
//
// 行为：
//   - 严格 Begin/End 包裹；hunk 按序应用（多 hunk 文件级原子性：逐个应用，失败即止）；
//   - Update 的 @@ 段按序定位：从上一段结束处向后搜索（顺序敏感，对齐 codex）；
//   - 匹配容差：行尾空白（含 \r）忽略；找不到给出行号级诊断；
//   - Move to：先应用变更再移动（对齐 codex）；Delete 不存在文件报错；
//   - 逐文件写前快照 + 变更回调（与 write/edit 行为一致）；
//   - 换行风格保留（CRLF 文件应用后仍 CRLF）。
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ─── 数据类型 ────────────────────────────────────────────────

// patchLine 补丁中的一行（op: ' ' 上下文 / '-' 删除 / '+' 添加）。
type patchLine struct {
	op   byte
	text string
}

// patchSection Update 的单个 @@ 分段。
type patchSection struct {
	ctx   string // "@@ 说明"（可空）
	lines []patchLine
}

// patchHunk 一个补丁块（add/delete/update）。
type patchHunk struct {
	kind     string // add | delete | update
	path     string
	moveTo   string         // 仅 update：*** Move to:
	addLines []string       // 仅 add
	sections []patchSection // 仅 update
}

// ─── 解析 ────────────────────────────────────────────────────

// parsePatchText 解析补丁文本；严格遵守 Begin/End 包裹与 hunk 头格式。
func parsePatchText(patch string) ([]patchHunk, error) {
	norm := strings.ReplaceAll(patch, "\r\n", "\n")
	lines := strings.Split(norm, "\n")
	i := 0
	// 跳过前导空行
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) || strings.TrimSpace(lines[i]) != "*** Begin Patch" {
		first := ""
		if i < len(lines) {
			first = lines[i]
		}
		return nil, fmt.Errorf("补丁必须以 \"*** Begin Patch\" 开始（实际: %q）", first)
	}
	i++

	var hunks []patchHunk
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			i++
			continue
		}
		if trimmed == "*** End Patch" {
			return hunks, nil
		}
		switch {
		case strings.HasPrefix(trimmed, "*** Add File:"):
			path := strings.TrimSpace(strings.TrimPrefix(trimmed, "*** Add File:"))
			if path == "" {
				return nil, fmt.Errorf("Add File 缺少路径（第 %d 行）", i+1)
			}
			h := patchHunk{kind: "add", path: path}
			i++
			for i < len(lines) {
				l := lines[i]
				t := strings.TrimSpace(l)
				if t == "*** End Patch" || strings.HasPrefix(t, "*** Add File:") ||
					strings.HasPrefix(t, "*** Update File:") || strings.HasPrefix(t, "*** Delete File:") {
					break
				}
				if l == "" {
					// Add 块内空行：视为空内容行（宽容）
					h.addLines = append(h.addLines, "")
					i++
					continue
				}
				if !strings.HasPrefix(l, "+") {
					return nil, fmt.Errorf("Add File 行必须以 \"+\" 开头（第 %d 行: %q）", i+1, l)
				}
				h.addLines = append(h.addLines, l[1:])
				i++
			}
			hunks = append(hunks, h)

		case strings.HasPrefix(trimmed, "*** Delete File:"):
			path := strings.TrimSpace(strings.TrimPrefix(trimmed, "*** Delete File:"))
			if path == "" {
				return nil, fmt.Errorf("Delete File 缺少路径（第 %d 行）", i+1)
			}
			hunks = append(hunks, patchHunk{kind: "delete", path: path})
			i++

		case strings.HasPrefix(trimmed, "*** Update File:"):
			path := strings.TrimSpace(strings.TrimPrefix(trimmed, "*** Update File:"))
			if path == "" {
				return nil, fmt.Errorf("Update File 缺少路径（第 %d 行）", i+1)
			}
			h := patchHunk{kind: "update", path: path}
			i++
			// 可选 Move to（紧跟 Update File 行）
			if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "*** Move to:") {
				h.moveTo = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "*** Move to:"))
				i++
			}
			// 收集 change 段（"@@"/" "/"-"/"+"/"*** End of File"）
			cur := patchSection{}
			flush := func() {
				if len(cur.lines) > 0 || cur.ctx != "" {
					h.sections = append(h.sections, cur)
					cur = patchSection{}
				}
			}
			for i < len(lines) {
				l := lines[i]
				t := strings.TrimSpace(l)
				if t == "*** End Patch" || strings.HasPrefix(t, "*** Add File:") ||
					strings.HasPrefix(t, "*** Update File:") || strings.HasPrefix(t, "*** Delete File:") {
					break
				}
				if t == "*** Move to:" || strings.HasPrefix(t, "*** Move to:") {
					// Move 出现在 change 之后（少见）：接受
					h.moveTo = strings.TrimSpace(strings.TrimPrefix(t, "*** Move to:"))
					i++
					continue
				}
				if strings.HasPrefix(t, "@@") {
					flush()
					cur.ctx = strings.TrimSpace(strings.TrimPrefix(t, "@@"))
					i++
					continue
				}
				if t == "*** End of File" {
					// 段尾标记：仅提示块位于文件末尾，忽略（匹配算法不依赖）
					i++
					continue
				}
				if l == "" {
					// 空行：在 change 中按"空上下文行"处理（宽容），段外跳过
					if len(cur.lines) > 0 {
						cur.lines = append(cur.lines, patchLine{op: ' ', text: ""})
					}
					i++
					continue
				}
				switch l[0] {
				case ' ':
					cur.lines = append(cur.lines, patchLine{op: ' ', text: l[1:]})
				case '-':
					cur.lines = append(cur.lines, patchLine{op: '-', text: l[1:]})
				case '+':
					cur.lines = append(cur.lines, patchLine{op: '+', text: l[1:]})
				default:
					return nil, fmt.Errorf("Update 行必须以 \" \"、\"-\"、\"+\" 或 \"@@\" 开头（第 %d 行: %q）", i+1, l)
				}
				i++
			}
			flush()
			if len(h.sections) == 0 && h.moveTo == "" {
				return nil, fmt.Errorf("Update File %s 没有任何变更行（且无 Move to）", path)
			}
			hunks = append(hunks, h)

		default:
			return nil, fmt.Errorf("无法识别的补丁行（第 %d 行: %q）——hunk 头须为 *** Add/Update/Delete File:", i+1, line)
		}
	}
	return nil, fmt.Errorf("补丁缺少 \"*** End Patch\" 结尾")
}

// ─── 应用 ────────────────────────────────────────────────────

// ApplyPatchText 应用补丁到工作区（root 解析相对路径，含越界拦截）。
// 返回人类可读摘要；任一 hunk 失败即中止并返回错误（已应用的 hunk 不回滚——
// 与 codex 一致：补丁按序应用，失败提示模型重新检查）。
func ApplyPatchText(root, patchText string) (string, error) {
	hunks, err := parsePatchText(patchText)
	if err != nil {
		return "", err
	}
	if len(hunks) == 0 {
		return "", fmt.Errorf("补丁为空（没有任何 Add/Update/Delete 块）")
	}
	var summary []string
	for _, h := range hunks {
		switch h.kind {
		case "add":
			p, err := resolvePath(root, h.path)
			if err != nil {
				return "", err
			}
			SnapshotBeforeWriteWithTracking(root, p)
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return "", err
			}
			content := ""
			if len(h.addLines) > 0 {
				content = strings.Join(h.addLines, "\n") + "\n"
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return "", err
			}
			if FileChangeCallback != nil {
				FileChangeCallback(h.path)
			}
			summary = append(summary, "A "+h.path)

		case "delete":
			p, err := resolvePath(root, h.path)
			if err != nil {
				return "", err
			}
			if _, statErr := os.Stat(p); statErr != nil {
				return "", fmt.Errorf("Delete File 失败：%s 不存在", h.path)
			}
			SnapshotBeforeWriteWithTracking(root, p)
			if err := os.Remove(p); err != nil {
				return "", err
			}
			if FileChangeCallback != nil {
				FileChangeCallback("(删除) " + h.path)
			}
			summary = append(summary, "D "+h.path)

		case "update":
			p, err := resolvePath(root, h.path)
			if err != nil {
				return "", err
			}
			orig, err := os.ReadFile(p)
			if err != nil {
				return "", fmt.Errorf("Update File 失败：%v", err)
			}
			origStr := string(orig)
			newStr, stat, err := applyUpdateHunks(origStr, h.sections)
			if err != nil {
				return "", fmt.Errorf("%s: %w", h.path, err)
			}
			SnapshotBeforeWriteWithTracking(root, p)
			if err := os.WriteFile(p, []byte(newStr), 0o644); err != nil {
				return "", err
			}
			if h.moveTo != "" {
				np, err := resolvePath(root, h.moveTo)
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Dir(np), 0o755); err != nil {
					return "", err
				}
				if err := os.Rename(p, np); err != nil {
					return "", err
				}
				summary = append(summary, fmt.Sprintf("M %s → %s (+%d -%d)", h.path, h.moveTo, stat.added, stat.removed))
			} else {
				summary = append(summary, fmt.Sprintf("M %s (+%d -%d)", h.path, stat.added, stat.removed))
			}
			if FileChangeCallback != nil {
				FileChangeCallback(h.path)
			}
		}
	}
	return "补丁已应用（" + fmt.Sprint(len(summary)) + " 个文件）:\n" + strings.Join(summary, "\n"), nil
}

// patchStat 变更统计。
type patchStat struct{ added, removed int }

// applyUpdateHunks 把各 @@ 段按序应用到 content（保留原换行风格）。
func applyUpdateHunks(content string, sections []patchSection) (string, patchStat, error) {
	// 统一 \n 处理（最后 restoreNewlines 恢复原风格）
	work := normalizeNewlines(content)
	lines := strings.Split(work, "\n")
	// 尾随 \n 产生的最后空元素：保留（写回时 join("\n") 还原）
	var stat patchStat
	from := 0
	for si, sec := range sections {
		newLines, next, err := applyUpdateSection(lines, sec, from)
		if err != nil {
			if sec.ctx != "" {
				return "", stat, fmt.Errorf("第 %d 段（@@ %s）匹配失败: %w", si+1, sec.ctx, err)
			}
			return "", stat, fmt.Errorf("第 %d 段匹配失败: %w", si+1, err)
		}
		// 统计
		for _, pl := range sec.lines {
			switch pl.op {
			case '+':
				stat.added++
			case '-':
				stat.removed++
			}
		}
		lines = newLines
		from = next
	}
	out := strings.Join(lines, "\n")
	return restoreNewlines(out, content), stat, nil
}

// applyUpdateSection 定位并应用一个 change 段；返回新行数组与下一段搜索起点（0 基）。
func applyUpdateSection(lines []string, sec patchSection, from int) ([]string, int, error) {
	// 期望序列（" " 与 "-" 行参与匹配）
	var want []string
	var wantKeep []bool
	for _, pl := range sec.lines {
		if pl.op == '+' {
			continue
		}
		want = append(want, strings.TrimRight(pl.text, " \t"))
		wantKeep = append(wantKeep, pl.op == ' ')
	}
	if len(want) == 0 {
		// 纯插入（无上下文）：在 from 处插入 + 行
		var add []string
		for _, pl := range sec.lines {
			if pl.op == '+' {
				add = append(add, pl.text)
			}
		}
		if len(add) == 0 {
			return lines, from, nil
		}
		if from > len(lines) {
			from = len(lines)
		}
		out := make([]string, 0, len(lines)+len(add))
		out = append(out, lines[:from]...)
		out = append(out, add...)
		out = append(out, lines[from:]...)
		return out, from + len(add), nil
	}
	if from > len(lines) {
		from = len(lines)
	}
	matchAt := -1
	for i := from; i+len(want) <= len(lines); i++ {
		ok := true
		for j := range want {
			if strings.TrimRight(lines[i+j], " \t") != want[j] {
				ok = false
				break
			}
		}
		if ok {
			matchAt = i
			break
		}
	}
	if matchAt < 0 {
		var prev strings.Builder
		for j := range want {
			if j >= 6 {
				prev.WriteString("  …\n")
				break
			}
			mark := " "
			if !wantKeep[j] {
				mark = "-"
			}
			prev.WriteString("  " + mark + want[j] + "\n")
		}
		return nil, 0, fmt.Errorf(
			"上下文未找到（从第 %d 行起搜索，文件共 %d 行）。期望块:\n%s请先用 read 确认文件当前内容后再生成补丁",
			from+1, len(lines), prev.String())
	}
	// 构造替换块：' ' 保留文件中实际行，'-' 跳过，'+' 插入
	var replacement []string
	wi := matchAt
	for _, pl := range sec.lines {
		switch pl.op {
		case ' ':
			replacement = append(replacement, lines[wi])
			wi++
		case '-':
			wi++
		case '+':
			replacement = append(replacement, pl.text)
		}
	}
	out := make([]string, 0, len(lines)-len(want)+len(replacement))
	out = append(out, lines[:matchAt]...)
	out = append(out, replacement...)
	out = append(out, lines[matchAt+len(want):]...)
	return out, matchAt + len(replacement), nil
}
