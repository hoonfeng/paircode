package agent

// 搜索/导航工具（Round3 改名基座）：grep（正则内容搜索，原 search_content）+
// glob 递归查找（原 search_files）+ glob 目录列举（原 list_files，与查找合并）。
// 复刻参考源 src/agent 的 search_content / search_files。全部只读、免审批、
// 限定工作区内，自动跳过 .git/node_modules 等目录与二进制/超大文件
// （防把 LLM 上下文撑爆）。注册入口已并入 registerCoreTools（core 组），
// 本文件仅保留 handler 构造与辅助函数；生产语义以 tool-harness JS 插件为准，
// Go 侧仅测试/归档基座。

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	maxSearchFileSize = 10 << 20 // 10MB：超过则跳过（不读进内存搜索）
	searchSniffBytes  = 8000     // 二进制嗅探：读前 N 字节查空字节
)

// defaultSkipDirs 内置基线：搜索/探索时跳过的依赖库/模块库/构建产物/缓存/VCS 目录（跨生态，全包共用）。
// 仍可显式把 path 指进某个被跳目录来搜它（跳过只作用于自动递归下降，不挡显式起点）。
// 用户可经 SetExtraSkipDirs 追加（全局设置 + 项目级，companion 注入）。
var defaultSkipDirs = map[string]bool{
	// VCS / 编辑器
	".git": true, ".svn": true, ".hg": true, ".idea": true, ".vscode": true,
	// 依赖库 / 模块库
	"node_modules": true, "bower_components": true, "jspm_packages": true, "vendor": true, "Pods": true,
	"venv": true, ".venv": true, "__pycache__": true, ".pytest_cache": true, ".mypy_cache": true, ".tox": true,
	// 构建产物
	"dist": true, "build": true, "out": true, "target": true,
	".next": true, ".nuxt": true, ".svelte-kit": true, ".output": true,
	// 缓存 / 覆盖率 / 基建
	".gradle": true, ".cache": true, ".turbo": true, "coverage": true, ".nyc_output": true, ".terraform": true,
	// 本项目自身数据 / 备份
	".pair": true, "源码备份": true,
}

// extraSkipDirs 用户配置的额外忽略目录（全局设置 + 项目级 .pair/ignore，由 companion 合并后注入）。
var extraSkipDirs = map[string]bool{}

// SetExtraSkipDirs 设置额外忽略目录名（覆盖上次）。companion 合并 全局+项目 配置后调用。
func SetExtraSkipDirs(dirs []string) {
	m := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		if d = strings.TrimSpace(d); d != "" {
			m[d] = true
		}
	}
	extraSkipDirs = m
}

// isSkipDir 该目录名是否跳过（内置基线 ∪ 用户额外）。
func isSkipDir(name string) bool { return defaultSkipDirs[name] || extraSkipDirs[name] }

// listFilesHandler 目录列举 handler（原 list_files，Round3 并入 glob：
// glob 无 pattern 时走本分支）。
func listFilesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		rel := argStr(args, "path")
		p := root
		if rel != "" {
			var err error
			if p, err = resolvePathFor(root, args, rel); err != nil {
				return "", err
			}
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			// ★ 路径不存在给明确提示（Windows 原生错误晦涩，LLM 难恢复）
			if os.IsNotExist(err) {
				return "", fmt.Errorf("目录不存在: %s（请确认路径在工作区内且拼写正确；可用 str_replace_editor view 列目录探查）", argStr(args, "path"))
			}
			return "", err
		}
		pattern := argStr(args, "pattern")
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})
		var b strings.Builder
		for _, e := range entries {
			if pattern != "" && !e.IsDir() {
				if ok, _ := filepath.Match(pattern, e.Name()); !ok {
					continue
				}
			}
			if e.IsDir() {
				b.WriteString(e.Name() + "/\n")
			} else {
				sz := int64(-1)
				if fi, err := e.Info(); err == nil {
					sz = fi.Size()
				}
				fmt.Fprintf(&b, "%s\t%d\n", e.Name(), sz)
			}
		}
		if b.Len() == 0 {
			return "（空目录或无匹配）", nil
		}
		return b.String(), nil
	}
}

func searchContentHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		projRoot, err := projRootFromArgs(root, args)
		if err != nil {
			return "", err
		}
		pattern := strings.TrimSpace(argStr(args, "pattern"))
		if pattern == "" {
			return "", fmt.Errorf("pattern 不能为空")
		}
		prefix := ""
		if argBool(args, "case_insensitive") {
			prefix = "(?i)"
		}
		re, err := regexp.Compile(prefix + pattern)
		if err != nil {
			return "", fmt.Errorf("正则编译失败: %w", err)
		}
		base, err := searchRoot(projRoot, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		glob := strings.TrimSpace(argStr(args, "glob"))
		max := clampInt(argInt(args, "max_results", 200), 200, 1, 2000)

		var lines []string
		fileHits := map[string]int{} // 相对路径 → 命中数（供结果统计）
		count := 0
		truncated := false
		walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil // 跳过无法访问的项
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				if p != base && isSkipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if glob != "" {
				if !matchGlobFilter(glob, d.Name(), relSlash(projRoot, p)) {
					return nil
				}
			}
			if info, e := d.Info(); e == nil && info.Size() > maxSearchFileSize {
				return nil
			}
			data, e := os.ReadFile(p)
			if e != nil || isBinary(data) {
				return nil
			}
			rel := relSlash(projRoot, p)
			for i, line := range strings.Split(string(data), "\n") {
				if re.MatchString(line) {
					lines = append(lines, fmt.Sprintf("%s:%d: %s", rel, i+1, trimLine(line)))
					fileHits[rel]++
					if count++; count >= max {
						truncated = true
						return fs.SkipAll
					}
				}
			}
			return nil
		})
		if walkErr != nil && ctx.Err() != nil {
			return "", ctx.Err()
		}
		if count == 0 {
			return "（未找到匹配）\n提示：无结果≠不存在，建议补搜：① 换关键词/同义词 ② 加 (?i) 忽略大小写 ③ 换 path/glob 范围 ④ 检查正则写法。不要就此断言不存在。", nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "（命中 %d 处，覆盖 %d 个文件）\n", count, len(fileHits))
		for _, l := range lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
		res := b.String()
		if truncated {
			res += fmt.Sprintf("[已达上限 %d 条，可能还有更多匹配——请缩小 pattern 或 path]\n", max)
		}
		return capOutput(res, 16000), nil
	}
}

func searchFilesHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		projRoot, err := projRootFromArgs(root, args)
		if err != nil {
			return "", err
		}
		pattern := strings.TrimSpace(argStr(args, "pattern"))
		if pattern == "" {
			return "", fmt.Errorf("pattern 不能为空")
		}
		base, err := searchRoot(projRoot, argStr(args, "path"))
		if err != nil {
			return "", err
		}
		max := clampInt(argInt(args, "max_results", 500), 500, 1, 5000)
		langFilter := strings.TrimSpace(argStr(args, "language"))

		var matches []string
		truncated := false
		walkErr := filepath.WalkDir(base, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				if p != base && isSkipDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if langFilter != "" {
				ext := strings.ToLower(filepath.Ext(p))
				detectedLang := extLangMap[ext]
				if detectedLang == "" {
					detectedLang = strings.TrimPrefix(ext, ".")
				}
				if !strings.EqualFold(detectedLang, langFilter) {
					return nil
				}
			}
			if matchFile(pattern, d.Name(), relSlash(projRoot, p)) {
				matches = append(matches, relSlash(projRoot, p))
				if len(matches) >= max {
					truncated = true
					return fs.SkipAll
				}
			}
			return nil
		})
		if walkErr != nil && ctx.Err() != nil {
			return "", ctx.Err()
		}
		if len(matches) == 0 {
			return "（未找到匹配文件）\n提示：无结果≠不存在，建议补搜：① 换文件名通配（如 *关键字*）② 换 path 范围 ③ 换 language 过滤 ④ 改用 grep 搜内容。不要就此断言不存在。", nil
		}
		sort.Strings(matches)
		res := fmt.Sprintf("（找到 %d 个文件）\n", len(matches)) + strings.Join(matches, "\n")
		if truncated {
			res += fmt.Sprintf("\n[已达上限 %d 个，可能还有更多——可缩小 path 或加 language 过滤]", max)
		}
		return capOutput(res, 16000), nil
	}
}

// ─── 辅助 ────────────────────────────────────────────────────

// searchRoot 解析搜索起点目录（省略=工作区根，限定工作区内）。
func searchRoot(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return root, nil
	}
	return resolvePath(root, rel)
}

// matchFile 通配匹配（支持 ** 递归）：pattern 含 / 或 ** 时按相对路径匹配，否则按文件名匹配。
func matchFile(pattern, base, rel string) bool {
	return matchGlobFilter(pattern, base, rel)
}

// relSlash 取相对工作区根的 slash 路径（给 LLM 看的稳定相对路径）。
func relSlash(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// isBinary 嗅探前若干字节是否含空字节（含=视作二进制，跳过文本搜索）。
func isBinary(data []byte) bool {
	n := min(len(data), searchSniffBytes)
	for i := range n {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// trimLine 去首尾空白并按 rune 截断过长行（结果行预览，避免单行撑爆）。
func trimLine(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return s
}

// clampInt 取值约束：v<=0 或越界则回退 def，并夹到 [lo, hi]。
func clampInt(v, def, lo, hi int) int {
	if v <= 0 {
		v = def
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}
