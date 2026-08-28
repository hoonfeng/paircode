package impl

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/tool-codegraph-extra/codegraph"
	. "github.com/hoonfeng/paircode/plugins-src/plugins/tool-codegraph-extra/toolbin"
)

// registerExtraCodeGraphTools 注册额外的代码图谱工具（#29-#30）。
// 在 RegisterDefaultTools 中调用。
func Register(r *Registry, root string) {

	// ---- 19. codegraph_find_entry_points ----
	r.Register(&Tool{
		Name: "codegraph_find_entry_points", Description: "发现应用程序入口点和执行起点。",
		UsageGuide: "发现应用程序入口点（main 函数、HTTP handler、CLI 命令）。新项目先调此工具了解从哪启动。",
		Parameters: ObjSchema(Props{"entryType": StrProp("可选：main/http_handler/cli_command/all，默认 all"), "limit": IntProp("可选：最大返回数（默认 50）")}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entryType := ArgStr(args, "entry_type")
			limit := ArgInt(args, "limit", 50)
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			type ep struct {
				name, kind, file string
				line             int
			}
			var entries []ep
			if entryType == "" || entryType == "all" || entryType == "main" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					if fn.Name == "main" && fn.FilePath != "" {
						entries = append(entries, ep{fn.Name, "main", fn.FilePath, fn.Line})
					}
				}
			}
			if entryType == "" || entryType == "all" || entryType == "http_handler" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range []string{"HandleFunc", "Handle", "router.GET", "echo.GET", "gin.GET", "http.Handle"} {
						if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) {
							entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line})
							break
						}
					}
				}
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityMethod) {
					if strings.Contains(fn.Signature, "ServeHTTP") {
						entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line})
					}
				}
			}
			if entryType == "" || entryType == "all" || entryType == "cli_command" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range []string{"cobra", "Execute", "RunE", "flag.Parse"} {
						if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) {
							entries = append(entries, ep{fn.Name, "cli_command", fn.FilePath, fn.Line})
							break
						}
					}
				}
			}
			seen := map[string]bool{}
			var uniq []ep
			for _, e := range entries {
				k := fmt.Sprintf("%s:%d:%s", e.file, e.line, e.name)
				if !seen[k] {
					seen[k] = true
					uniq = append(uniq, e)
				}
			}
			sort.Slice(uniq, func(i, j int) bool {
				if uniq[i].kind != uniq[j].kind {
					return uniq[i].kind < uniq[j].kind
				}
				return uniq[i].file < uniq[j].file
			})
			if len(uniq) > limit {
				uniq = uniq[:limit]
			}
			if len(uniq) == 0 {
				return "未发现入口点。", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("发现 %d 个入口点：\n\n", len(uniq)))
			for _, e := range uniq {
				b.WriteString(fmt.Sprintf("  [%s] %s (%s:%d)\n", e.kind, e.name, e.file, e.line))
			}
			return b.String(), nil
		},
	})

	// ---- 20. codegraph_find_hot_paths ----
	r.Register(&Tool{
		Name: "codegraph_find_hot_paths", Description: "查找最常被调用的函数。",
		UsageGuide: "查找最常被调用的函数（按调用者数量排序）。了解核心热点代码，优化优先考虑高频路径。",
		Parameters: ObjSchema(Props{"limit": IntProp("可选：最大返回数（默认 20）")}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			limit := ArgInt(args, "limit", 20)
			if limit <= 0 || limit > 100 {
				limit = 20
			}
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			type hf struct {
				name, file string
				callers    int
			}
			var list []hf
			for _, fn := range all {
				n := len(g.GetPredecessors(fn.ID, codegraph.RelCalls))
				if n > 0 {
					list = append(list, hf{fn.Name, fn.FilePath, n})
				}
			}
			sort.Slice(list, func(i, j int) bool { return list[i].callers > list[j].callers })
			if len(list) > limit {
				list = list[:limit]
			}
			if len(list) == 0 {
				return "未发现被调用的函数。", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("最热函数（共 %d 条）：\n\n", len(list)))
			for i, h := range list {
				b.WriteString(fmt.Sprintf("%d. %s (%s) - %d 个调用者\n", i+1, h.name, h.file, h.callers))
			}
			return b.String(), nil
		},
	})

	// ---- 21. codegraph_find_by_imports ----
	r.Register(&Tool{
		Name: "codegraph_find_by_imports", Description: "查找所有导入指定模块的文件。",
		UsageGuide: "查找所有导入指定模块的文件。想了解某包被哪些文件引用时用。比 grep 搜索 import 语句更精确（基于解析的 import 关系）。",
		Parameters: ObjSchema(Props{"moduleName": StrProp("模块/包名"), "matchMode": StrProp("可选：exact/prefix/contains/fuzzy，默认 contains"), "limit": IntProp("可选：最大返回数（默认 50）")}, "moduleName"),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			mn := strings.TrimSpace(ArgStr(args, "module_name"))
			mm := ArgStr(args, "match_mode")
			limit := ArgInt(args, "limit", 50)
			if mn == "" {
				return "", fmt.Errorf("moduleName 不能为空")
			}
			if mm == "" {
				mm = "contains"
			}
			fm := map[string][]string{}
			for _, fe := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if fe.FilePath == "" {
					continue
				}
				for _, imp := range g.GetSuccessors(fe.ID, codegraph.RelImports) {
					im := imp.Name
					if im == "" {
						im = imp.FQN
					}
					m := false
					switch mm {
					case "exact":
						m = strings.EqualFold(im, mn)
					case "prefix":
						m = strings.HasPrefix(strings.ToLower(im), strings.ToLower(mn))
					default:
						m = strings.Contains(strings.ToLower(im), strings.ToLower(mn))
					}
					if m {
						fm[fe.FilePath] = append(fm[fe.FilePath], im)
					}
				}
			}
			if len(fm) == 0 {
				return fmt.Sprintf("未找到导入「%s」的文件。", mn), nil
			}
			fs := make([]string, 0, len(fm))
			for f := range fm {
				fs = append(fs, f)
			}
			sort.Strings(fs)
			if len(fs) > limit {
				fs = fs[:limit]
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("导入「%s」的文件（共 %d 个）：\n\n", mn, len(fm)))
			for _, f := range fs {
				b.WriteString(fmt.Sprintf("  %s\n    - %s\n", f, fm[f][0]))
			}
			if len(fs) < len(fm) {
				b.WriteString(fmt.Sprintf("\n... 还有 %d 个文件未显示。", len(fm)-len(fs)))
			}
			return b.String(), nil
		},
	})

	// ---- 22. codegraph_get_detailed_symbol ----
	r.Register(&Tool{
		Name: "codegraph_get_detailed_symbol", Description: "获取符号详细上下文（源码+调用者+被调用者）。",
		UsageGuide: "获取某符号的完整上下文：源码+调用者+被调用者。比分别调 codegraph_callers/callees 更省 token（一站式）。",
		Parameters: ObjSchema(Props{"query": StrProp("符号名"), "includeSource": BoolProp("可选：包含源码（默认 true）")}, "query"),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(ArgStr(args, "query"))
			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			qe := codegraph.NewQueryEngine(g)
			entities := g.SearchEntities(query)
			if len(entities) == 0 {
				return fmt.Sprintf("未找到符号: %s", query), nil
			}
			if len(entities) > 5 {
				entities = entities[:5]
			}
			var b strings.Builder
			for i, e := range entities {
				if i > 0 {
					b.WriteString("\n" + strings.Repeat("-", 60) + "\n\n")
				}
				b.WriteString(fmt.Sprintf("## %s (%s)\n文件: %s:%d-%d\n", e.Name, string(e.Kind), e.FilePath, e.Line, e.EndLine))
				if e.Signature != "" {
					b.WriteString(fmt.Sprintf("签名: %s\n", e.Signature))
				}
				callers := qe.GetCallers(e.Name)
				b.WriteString(fmt.Sprintf("调用者: %d\n", len(callers)))
				for j := 0; j < len(callers) && j < 10; j++ {
					b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", callers[j].CallerName, callers[j].CallerFile, callers[j].CallerLine))
				}
				callees := qe.GetCallees(e.Name)
				b.WriteString(fmt.Sprintf("被调用者: %d\n", len(callees)))
				for j := 0; j < len(callees) && j < 10; j++ {
					b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", callees[j].CalleeName, callees[j].CallerFile, callees[j].CallerLine))
				}
				if ArgBool(args, "include_source") && e.FilePath != "" {
					d2, rErr := os.ReadFile(filepath.Join(root, e.FilePath))
					if rErr == nil {
						lines := strings.Split(string(d2), "\n")
						start, end := max(0, e.Line-1), e.EndLine
						if end <= start {
							end = start + 20
						}
						end = min(len(lines), min(start+80, end))
						if start < end {
							b.WriteString(fmt.Sprintf("\n### 源码\n"))
							for ln := start; ln < end; ln++ {
								b.WriteString(fmt.Sprintf("%d\t%s\n", ln+1, lines[ln]))
							}
						}
					}
				}
			}
			return b.String(), nil
		},
	})

	// ---- 23. codegraph_find_dead_imports ----
	r.Register(&Tool{
		Name: "codegraph_find_dead_imports", Description: "查找已导入但从未使用的模块。",
		UsageGuide: "查找已导入但从未使用的模块。改完代码后运行可发现残留 import。比 goimports 更灵活（指定文件或全量扫描）。",
		Parameters: ObjSchema(Props{"file": StrProp("可选：指定文件"), "limit": IntProp("可选：最大返回数（默认 50）")}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			fileFilter := strings.TrimSpace(ArgStr(args, "file"))
			limit := ArgInt(args, "limit", 50)
			type di struct{ file, imp string }
			var dead []di
			for _, fe := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if fe.FilePath == "" || (fileFilter != "" && !strings.Contains(fe.FilePath, fileFilter)) {
					continue
				}
				imps := g.GetSuccessors(fe.ID, codegraph.RelImports)
				if len(imps) == 0 {
					continue
				}
				fEnts := g.GetEntitiesByFile(fe.FilePath)
				for _, imp := range imps {
					im := imp.Name
					if im == "" {
						im = imp.FQN
					}
					ls := im
					if i := strings.LastIndex(im, "/"); i >= 0 {
						ls = im[i+1:]
					}
					if ls == "" {
						ls = im
					}
					used := false
					for _, fe2 := range fEnts {
						if (fe2.Kind == codegraph.EntityFunction || fe2.Kind == codegraph.EntityMethod) && (strings.Contains(fe2.FQN, ls) || strings.Contains(fe2.Signature, ls)) {
							used = true
							break
						}
						if (fe2.Kind == codegraph.EntityType || fe2.Kind == codegraph.EntityStruct) && strings.Contains(fe2.Signature, ls) {
							used = true
							break
						}
					}
					if !used {
						dead = append(dead, di{fe.FilePath, im})
					}
				}
			}
			if len(dead) == 0 {
				return "未发现死导入。", nil
			}
			sort.Slice(dead, func(i, j int) bool {
				if dead[i].file != dead[j].file {
					return dead[i].file < dead[j].file
				}
				return dead[i].imp < dead[j].imp
			})
			if len(dead) > limit {
				dead = dead[:limit]
			}
			byFile := map[string][]string{}
			for _, d := range dead {
				byFile[d.file] = append(byFile[d.file], d.imp)
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("发现 %d 个可能未使用的导入：\n\n", len(dead)))
			for f, imps := range byFile {
				b.WriteString(fmt.Sprintf("  %s\n", f))
				for _, imp := range imps {
					b.WriteString(fmt.Sprintf("    - %s\n", imp))
				}
			}
			return b.String(), nil
		},
	})

	// ---- 24. codegraph_search_by_error ----
	r.Register(&Tool{
		Name: "codegraph_search_by_error", Description: "查找抛出或处理错误的函数。",
		UsageGuide: "查找抛出或处理特定错误的函数。mode=throws 找谁抛了错误，catches 找谁处理了。错误分析定位根因时用。",
		Parameters: ObjSchema(Props{"mode": StrProp("可选：throws/catches/any，默认 any"), "errorType": StrProp("可选：错误类型过滤"), "limit": IntProp("可选：最大返回数（默认 50）")}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			mode := ArgStr(args, "mode")
			eType := strings.TrimSpace(ArgStr(args, "error_type"))
			limit := ArgInt(args, "limit", 50)
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			type ht struct {
				name, file string
				line       int
				pats       []string
			}
			var res []ht
			for _, fn := range all {
				var m []string
				sig := fn.Signature + " " + fn.Doc
				if mode == "" || mode == "any" || mode == "throws" {
					for _, p := range []string{"errors.New", "fmt.Errorf", "panic(", "return.*err"} {
						if ok, _ := regexp.MatchString(p, sig); ok {
							m = append(m, p)
						}
					}
				}
				if mode == "" || mode == "any" || mode == "catches" {
					for _, p := range []string{"if err != nil", "if err == nil", "catch(", "except "} {
						if strings.Contains(sig, p) {
							m = append(m, p)
						}
					}
				}
				if eType != "" {
					if !strings.Contains(fn.Signature, eType) && !strings.Contains(fn.FQN, eType) {
						continue
					}
					m = append(m, "err:"+eType)
				}
				if len(m) > 0 {
					res = append(res, ht{fn.Name, fn.FilePath, fn.Line, m})
				}
			}
			if len(res) == 0 {
				return "未找到错误匹配。", nil
			}
			sort.Slice(res, func(i, j int) bool {
				if res[i].file != res[j].file {
					return res[i].file < res[j].file
				}
				return res[i].name < res[j].name
			})
			if len(res) > limit {
				res = res[:limit]
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("匹配 %d 个函数：\n\n", len(res)))
			for _, h := range res {
				b.WriteString(fmt.Sprintf("  %s (%s:%d) %s\n", h.name, h.file, h.line, strings.Join(h.pats, ", ")))
			}
			return b.String(), nil
		},
	})

	// ---- 25. codegraph_index_markdown ----
	r.Register(&Tool{
		Name: "codegraph_index_markdown", Description: "索引 Markdown 文档，按标题分段。有 ONNX 则计算嵌入向量。",
		UsageGuide: "索引 Markdown 文档到知识库。之后可用 codegraph_search_docs 语义搜索。新加文档后需重新索引才能搜到。",
		Parameters: ObjSchema(Props{"path": StrProp("可选：文件路径；省略则扫描全部 .md 文件")}),
		ReadOnly:   false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			mdPath := strings.TrimSpace(ArgStr(args, "path"))
			var mds []string
			if mdPath != "" {
				mds = append(mds, mdPath)
			} else {
				filepath.WalkDir(root, func(p string, d fs.DirEntry, werr error) error {
					if werr != nil {
						return nil
					}
					if d.IsDir() {
						if d.Name() == ".git" || d.Name() == "node_modules" {
							return fs.SkipDir
						}
						return nil
					}
					if strings.HasSuffix(strings.ToLower(p), ".md") {
						rel, _ := filepath.Rel(root, p)
						mds = append(mds, rel)
					}
					return nil
				})
			}
			embedCache := LoadEmbeddingCache(root)
			backend := GetEmbeddingBackend(root)
			idxd := 0
			for _, mf := range mds {
				d2, rErr := os.ReadFile(filepath.Join(root, mf))
				if rErr != nil {
					continue
				}
				sections := splitMarkdownSections(string(d2))
				for i, sec := range sections {
					dk := fmt.Sprintf("doc:%s#%d", mf, i)
					g.AddEntity(&codegraph.Entity{ID: dk, Kind: codegraph.EntityDocSection, Name: fmt.Sprintf("%s - %s", mf, sec.heading), FilePath: mf, Line: sec.line, Doc: sec.body})
					if backend.Available() {
						if vec, eErr := backend.Embed(sec.heading + "\n" + sec.body); eErr == nil && len(vec) > 0 {
							embedCache.Set(dk, vec)
						}
					}
				}
				idxd++
			}
			if backend.Available() {
				embedCache.Save()
				return fmt.Sprintf("已索引 %d 个文件（含向量），共 %d 个节。", idxd, len(mds)), nil
			}
			return fmt.Sprintf("已索引 %d 个文件（无向量），共 %d 个节。", idxd, len(mds)), nil
		},
	})

	// ---- 26. codegraph_search_docs ----
	r.Register(&Tool{
		Name: "codegraph_search_docs", Description: "搜索已索引文档。优先向量语义搜索，回退关键词。",
		UsageGuide: "搜索已索引的 Markdown 文档。有 ONNX 时做语义搜索（理解意图），否则关键词回退。比全文搜索更智能。",
		Parameters: ObjSchema(Props{"query": StrProp("搜索关键词"), "limit": IntProp("可选：最大返回数（默认 5）")}, "query"),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			query := strings.TrimSpace(ArgStr(args, "query"))
			limit := ArgInt(args, "limit", 5)
			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			sections := g.GetEntitiesByKind(codegraph.EntityDocSection)
			if len(sections) == 0 {
				return "未索引文档。请先运行 codegraph_index_markdown。", nil
			}
			backend := GetEmbeddingBackend(root)
			if backend.Available() {
				embedCache := LoadEmbeddingCache(root)
				if qv, qErr := backend.Embed(query); qErr == nil && len(qv) > 0 {
					type vm struct {
						s   *codegraph.Entity
						sim float64
					}
					var vms []vm
					for _, sec := range sections {
						if vec := embedCache.Get(sec.ID); vec != nil {
							vms = append(vms, vm{sec, CosineSimilarity(qv, vec)})
						}
					}
					if len(vms) > 0 {
						sort.Slice(vms, func(i, j int) bool { return vms[i].sim > vms[j].sim })
						if len(vms) > limit {
							vms = vms[:limit]
						}
						var b strings.Builder
						b.WriteString(fmt.Sprintf("【向量搜索】找到 %d 个节：\n\n", len(vms)))
						for _, m := range vms {
							b.WriteString(fmt.Sprintf("### %s\n相似度: %.3f\n\n%s\n\n---\n\n", m.s.Name, m.sim, m.s.Doc))
						}
						return b.String(), nil
					}
				}
			}
			type km struct {
				s     *codegraph.Entity
				score int
			}
			var kms []km
			q := strings.ToLower(query)
			for _, sec := range sections {
				score := 0
				name := strings.ToLower(sec.Name)
				doc := strings.ToLower(sec.Doc)
				for _, w := range strings.Fields(q) {
					if len(w) >= 2 {
						if strings.Contains(name, w) {
							score += 3
						}
						if strings.Contains(doc, w) {
							score++
						}
					}
				}
				if score > 0 {
					kms = append(kms, km{sec, score})
				}
			}
			sort.Slice(kms, func(i, j int) bool { return kms[i].score > kms[j].score })
			if len(kms) > limit {
				kms = kms[:limit]
			}
			if len(kms) == 0 {
				return "未找到匹配。", nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("【关键词搜索】找到 %d 个节：\n\n", len(kms)))
			for _, m := range kms {
				b.WriteString(fmt.Sprintf("### %s\n%s\n\n---\n\n", m.s.Name, m.s.Doc))
			}
			return b.String(), nil
		},
	})

	// ---- 27. codegraph_verify_design ----
	r.Register(&Tool{
		Name: "codegraph_verify_design", Description: "检查设计文档中的代码引用是否存在。",
		UsageGuide: "检查设计文档中的代码引用是否仍然有效。重构后运行可发现过期的文档引用。",
		Parameters: ObjSchema(Props{"docFile": StrProp("设计文档路径")}, "docFile"),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			docFile := strings.TrimSpace(ArgStr(args, "doc_file"))
			if docFile == "" {
				return "", fmt.Errorf("docFile 不能为空")
			}
			d2, rErr := os.ReadFile(filepath.Join(root, docFile))
			if rErr != nil {
				return "", fmt.Errorf("读取失败: %w", rErr)
			}
			re := regexp.MustCompile("[`]([^`]+)[`]")
			idents := re.FindAllStringSubmatch(string(d2), -1)
			uniq := map[string]bool{}
			var syms []string
			for _, m := range idents {
				s := strings.TrimSpace(m[1])
				if s != "" && len(s) > 1 && !uniq[s] {
					uniq[s] = true
					syms = append(syms, s)
				}
			}
			if len(syms) == 0 {
				return "文档中未发现代码标识符。", nil
			}
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			var found, nf []string
			for _, sym := range syms {
				if len(g.SearchEntities(sym)) > 0 {
					found = append(found, sym)
				} else {
					nf = append(nf, sym)
				}
			}
			pct := 0
			if len(syms) > 0 {
				pct = len(found) * 100 / len(syms)
			}
			return fmt.Sprintf("共 %d 个标识符，存在 %d 个 (%d%%)，缺失 %d 个", len(syms), len(found), pct, len(nf)), nil
		},
	})

	// ---- 28. codegraph_pr_context ----
	r.Register(&Tool{
		Name: "codegraph_pr_context", Description: "分析分支变更影响范围。",
		UsageGuide: "分析当前分支与 baseBranch 的变更影响范围。提交 PR 前运行可了解变更波及哪些文件/函数。",
		Parameters: ObjSchema(Props{"baseBranch": StrProp("可选：基准分支（默认 main）")}),
		ReadOnly:   true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			bb := strings.TrimSpace(ArgStr(args, "base_branch"))
			if bb == "" {
				bb = "main"
			}
			diff, dErr := runGit(ctx, root, "diff", "--stat", bb+"...HEAD")
			if dErr != nil {
				return "", fmt.Errorf("git diff: %w", dErr)
			}
			if strings.TrimSpace(diff) == "" || diff == "(无输出)" {
				return "无差异。", nil
			}
			no, _ := runGit(ctx, root, "diff", "--name-only", bb+"...HEAD")
			files := strings.Fields(strings.TrimSpace(no))
			g, err := getCodeGraph(root)
			if err != nil {
				return diff, nil
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("## PR 影响分析\n### 变更文件\n%s\n\n", diff))
			for _, f := range files {
				f = strings.TrimSpace(f)
				if f == "" {
					continue
				}
				fe := g.GetEntitiesByFile(f)
				if len(fe) > 0 {
					cs := map[string]bool{}
					for _, e := range fe {
						if e.Kind == codegraph.EntityFunction || e.Kind == codegraph.EntityMethod {
							for _, c := range codegraph.NewQueryEngine(g).GetCallers(e.Name) {
								cs[c.CallerName+"@"+c.CallerFile] = true
							}
						}
					}
					b.WriteString(fmt.Sprintf("- **%s** 影响 %d 个调用者\n", f, len(cs)))
				} else {
					b.WriteString(fmt.Sprintf("- %s\n", f))
				}
			}
			return b.String(), nil
		},
	})

	// ── 29. codegraph_find_by_signature — 按签名查找函数 ──
	r.Register(&Tool{
		Name:        "codegraph_find_by_signature",
		UsageGuide:  "按结构特征查找函数：参数个数/返回类型/名称模式。想找「接收 string 返回 error」的函数时用。比 grep 更原子化（基于签名匹配）。",
		Description: "按结构特征（参数数、返回类型、名称模式）查找函数。",
		Parameters: ObjSchema(Props{
			"namePattern": StrProp("可选：函数名通配模式，如 'get*'、'*Handler'"),
			"paramCount":  IntProp("可选：精确参数个数"),
			"minParams":   IntProp("可选：最少参数个数"),
			"maxParams":   IntProp("可选：最多参数个数"),
			"returnType":  StrProp("可选：返回类型，如 'error'、'string'"),
			"limit":       IntProp("可选：最大返回数（默认 50）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			namePattern := strings.TrimSpace(ArgStr(args, "name_pattern"))
			paramCount := ArgInt(args, "param_count", 0)
			minParams := ArgInt(args, "min_params", 0)
			maxParams := ArgInt(args, "max_params", 0)
			returnType := strings.TrimSpace(ArgStr(args, "return_type"))
			limit := ArgInt(args, "limit", 50)
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			var nameRe *regexp.Regexp
			if namePattern != "" {
				nameRe, _ = regexp.Compile("^" + strings.ReplaceAll(regexp.QuoteMeta(namePattern), "\\*", ".*") + "$")
			}
			type sm struct {
				n, k, f, s, r string
				l, c          int
			}
			var res []sm
			for _, fn := range all {
				if nameRe != nil && !nameRe.MatchString(fn.Name) {
					continue
				}
				c := countParams(fn.Signature)
				if paramCount > 0 && c != paramCount {
					continue
				}
				if minParams > 0 && c < minParams {
					continue
				}
				if maxParams > 0 && c > maxParams {
					continue
				}
				r := extractReturnType(fn.Signature)
				if returnType != "" && !strings.Contains(strings.ToLower(r), strings.ToLower(returnType)) {
					continue
				}
				res = append(res, sm{fn.Name, string(fn.Kind), fn.FilePath, fn.Signature, r, fn.Line, c})
			}
			if len(res) == 0 {
				return "未找到匹配签名的函数。", nil
			}
			sort.Slice(res, func(i, j int) bool {
				if res[i].f != res[j].f {
					return res[i].f < res[j].f
				}
				return res[i].n < res[j].n
			})
			if len(res) > limit {
				res = res[:limit]
			}
			var b strings.Builder
			b.WriteString(fmt.Sprintf("匹配 %d 个函数：\n\n", len(res)))
			for _, r := range res {
				b.WriteString(fmt.Sprintf("  %s (%s:%d) 参数:%d 返回:%s\n", r.n, r.f, r.l, r.c, r.r))
			}
			return b.String(), nil
		},
	})

	// ── 29.5. codegraph_semantic_search — 语义搜索代码 ──
	r.Register(&Tool{
		Name:        "codegraph_semantic_search",
		UsageGuide:  "基于语义理解搜索代码（需 ONNX 嵌入模型）。支持自然语言查询如「读取配置文件」「处理 HTTP 请求」。比关键词搜索更智能。",
		Description: "基于语义理解搜索代码（需 ONNX 嵌入模型）。支持自然语言查询，如「读取文件的函数」「处理错误的逻辑」。结果按语义相似度排序，比关键词搜索更准确。",
		Parameters: ObjSchema(Props{
			"query":   StrProp("自然语言查询，如「读取配置文件」「处理 HTTP 请求」"),
			"limit":   IntProp("可选：最大返回数（默认 10）"),
			"reindex": BoolProp("可选：强制重新索引代码实体（默认 false）"),
		}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(ArgStr(args, "query"))
			limit := ArgInt(args, "limit", 10)
			reindex := ArgBool(args, "reindex")
			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			if limit <= 0 || limit > 50 {
				limit = 10
			}

			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}

			backend := GetEmbeddingBackend(root)
			if !backend.Available() {
				return "嵌入模型不可用。需要 CGo（安装 GCC/MinGW-w64 后构建）且模型文件在 models/bge-small-zh-v1.5/ 或 .pair/embeddings/bge-small-zh-v1.5/ 中。", nil
			}

			cache := LoadEmbeddingCache(root)

			// 惰性索引：首次调用或强制重索引时计算所有代码实体的嵌入向量
			indexedCount := 0
			if reindex || len(cache.Vectors) == 0 {
				entities := g.GetEntitiesByKind(codegraph.EntityFunction)
				entities = append(entities, g.GetEntitiesByKind(codegraph.EntityMethod)...)
				entities = append(entities, g.GetEntitiesByKind(codegraph.EntityStruct)...)
				entities = append(entities, g.GetEntitiesByKind(codegraph.EntityInterface)...)
				entities = append(entities, g.GetEntitiesByKind(codegraph.EntityType)...)

				for _, e := range entities {
					key := "code:" + e.ID
					if !reindex && cache.Get(key) != nil {
						continue // 已有向量且非强制重索引
					}
					// 构建文本：类型 + 名称 + 签名 + 文档注释
					var text strings.Builder
					text.WriteString(string(e.Kind))
					text.WriteString(": ")
					text.WriteString(e.Name)
					text.WriteString("\n")
					if e.Signature != "" {
						text.WriteString(e.Signature)
						text.WriteString("\n")
					}
					if e.Doc != "" {
						text.WriteString(e.Doc)
					}
					vec, err := backend.Embed(text.String())
					if err == nil && len(vec) > 0 {
						cache.Set(key, vec)
						indexedCount++
					}
				}
				if indexedCount > 0 {
					cache.Save()
				}
			}

			// 算查询向量
			qv, err := backend.Embed(query)
			if err != nil || len(qv) == 0 {
				return "", fmt.Errorf("查询向量计算失败: %w", err)
			}

			// 遍历缓存，计算余弦相似度
			type se struct {
				entity *codegraph.Entity
				sim    float64
			}
			var results []se
			for key, vec := range cache.Vectors {
				if !strings.HasPrefix(key, "code:") {
					continue
				}
				entityID := strings.TrimPrefix(key, "code:")
				e := g.GetEntity(entityID)
				if e == nil {
					continue
				}
				sim := CosineSimilarity(qv, vec)
				if sim > 0.3 {
					results = append(results, se{e, sim})
				}
			}

			// 按相似度降序
			sort.Slice(results, func(i, j int) bool {
				return results[i].sim > results[j].sim
			})
			if len(results) > limit {
				results = results[:limit]
			}

			if len(results) == 0 {
				return fmt.Sprintf("未找到与「%s」语义相关的代码。（共 %d 个已索引实体）", query, indexedCount), nil
			}

			var b strings.Builder
			b.WriteString(fmt.Sprintf("## 语义搜索结果\n查询: `%s` | 找到 %d 个结果\n\n", query, len(results)))
			for i, r := range results {
				sig := r.entity.Signature
				if len(sig) > 100 {
					sig = sig[:100] + "…"
				}
				b.WriteString(fmt.Sprintf("### %d. %s (`%s`)\n", i+1, r.entity.Name, string(r.entity.Kind)))
				b.WriteString(fmt.Sprintf("- 相似度: **%.3f**\n", r.sim))
				b.WriteString(fmt.Sprintf("- 文件: `%s:%d`\n", r.entity.FilePath, r.entity.Line))
				if r.entity.Doc != "" {
					doc := r.entity.Doc
					if len(doc) > 200 {
						doc = doc[:200] + "…"
					}
					b.WriteString(fmt.Sprintf("- 说明: %s\n", doc))
				}
				if sig != "" {
					b.WriteString(fmt.Sprintf("- 签名: `%s`\n", sig))
				}
				b.WriteString("\n")
			}
			// 统计缓存的代码实体总数
			totalCached := 0
			for k := range cache.Vectors {
				if strings.HasPrefix(k, "code:") {
					totalCached++
				}
			}
			b.WriteString(fmt.Sprintf("---\n已索引 %d 个代码实体（共 %d 个缓存），相似度阈值 0.3", indexedCount, totalCached))
			return b.String(), nil
		},
	})

	// ── 30. codegraph_explore — 自然语言→源码 ──
	r.Register(&Tool{
		Name:        "codegraph_explore",
		UsageGuide:  "一站式代码理解工具。用自然语言或符号名探索代码，返回相关源码和位置。新接触项目时用此工具了解代码比逐个 read 更高效。",
		Description: "一站式代码理解工具。用自然语言或符号名探索代码，返回相关源码和位置。分析代码的首选工具。",
		Parameters:  ObjSchema(Props{"query": StrProp("自然语言问题或符号名"), "maxFiles": IntProp("可选：最大返回文件数（默认 8）")}, "query"),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(ArgStr(args, "query"))
			mx := ArgInt(args, "max_files", 8)
			if query == "" {
				return "", fmt.Errorf("query 不能为空")
			}
			if mx <= 0 || mx > 20 {
				mx = 8
			}
			g, err := getCodeGraph(root)
			if err != nil {
				return "", err
			}
			stop := map[string]bool{"the": true, "a": true, "in": true, "of": true, "to": true, "is": true, "be": true, "do": true, "如何": true, "什么": true}
			var kw []string
			for _, w := range strings.FieldsFunc(query, func(r rune) bool { return r == ' ' || r == ',' || r == '?' }) {
				l := strings.ToLower(strings.TrimSpace(w))
				if len(l) >= 2 && !stop[l] {
					kw = append(kw, l)
				}
			}
			sc := map[string]float64{}
			for _, q := range append([]string{query}, kw...) {
				for _, e := range g.SearchEntities(q) {
					if e.Kind == codegraph.EntityFile || e.Kind == codegraph.EntityPackage {
						continue
					}
					s := sc[e.ID]
					if strings.EqualFold(e.Name, q) {
						s += 3
					} else if strings.Contains(strings.ToLower(e.Name), q) {
						s += 1.5
					} else {
						s += 0.3
					}
					if e.Kind == codegraph.EntityFunction || e.Kind == codegraph.EntityMethod {
						s += 0.5
					}
					sc[e.ID] = s
				}
			}
			if len(sc) == 0 {
				return fmt.Sprintf("未找到与「%s」相关的代码。", query), nil
			}
			type se struct {
				e *codegraph.Entity
				s float64
			}
			var sl []se
			for id, s := range sc {
				if en := g.GetEntity(id); en != nil && en.FilePath != "" {
					sl = append(sl, se{en, s})
				}
			}
			sort.Slice(sl, func(i, j int) bool { return sl[i].s > sl[j].s })
			fm := map[string]*struct{ es []se }{}
			for _, s := range sl {
				if f, ok := fm[s.e.FilePath]; ok {
					f.es = append(f.es, s)
				} else {
					fm[s.e.FilePath] = &struct{ es []se }{[]se{s}}
				}
			}
			var gs []struct{ f string }
			for f := range fm {
				gs = append(gs, struct{ f string }{f})
			}
			sort.Slice(gs, func(i, j int) bool {
				return len(fm[gs[i].f].es) > len(fm[gs[j].f].es)
			})
			if len(gs) > mx {
				gs = gs[:mx]
			}
			const mc = 16000
			var b strings.Builder
			b.WriteString(fmt.Sprintf("## 代码探索\n查询: `%s` | %d 个符号 / %d 个文件\n\n", query, len(sl), len(gs)))
			for _, g2 := range gs {
				fp := filepath.Join(root, g2.f)
				data, rErr := os.ReadFile(fp)
				if rErr != nil {
					continue
				}
				lines := strings.Split(string(data), "\n")
				b.WriteString(fmt.Sprintf("### **`%s`** — %d 个匹配\n", g2.f, len(fm[g2.f].es)))
				for _, s2 := range fm[g2.f].es {
					sig := s2.e.Signature
					if len(sig) > 80 {
						sig = sig[:80] + "…"
					}
					b.WriteString(fmt.Sprintf("  · `%s` (:%d) — %s\n", s2.e.Name, s2.e.Line, sig))
				}
				mi, ma := 1000000, 0
				for _, s3 := range fm[g2.f].es {
					if s3.e.Line < mi {
						mi = s3.e.Line
					}
					if s3.e.EndLine > ma {
						ma = s3.e.EndLine
					}
				}
				st := max(0, mi-3)
				en := min(len(lines), ma+3)
				if en-st > 60 {
					en = st + 60
				}
				b.WriteString("\n```go\n")
				for ln := st; ln < en; ln++ {
					b.WriteString(fmt.Sprintf("%d\t%s\n", ln+1, lines[ln]))
				}
				b.WriteString("```\n\n")
				if b.Len() > mc {
					break
				}
			}
			return b.String(), nil
		},
	})
}

// countParams 从函数签名中统计参数个数。
func countParams(sig string) int {
	start := strings.Index(sig, "(")
	if start < 0 {
		return 0
	}
	end := strings.Index(sig[start:], ")")
	if end < 0 {
		return 0
	}
	params := strings.TrimSpace(sig[start+1 : start+end])
	if params == "" || params == "..." {
		return 0
	}
	// 按逗号分割，但注意处理嵌套泛型中的逗号
	depth := 0
	count := 1
	for _, ch := range params {
		switch ch {
		case '<', '[':
			depth++
		case '>', ']':
			depth--
		case ',':
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// extractReturnType 从函数签名中提取返回类型。
func extractReturnType(sig string) string {
	// 找最后一个 ) 之后的内容
	lastParen := strings.LastIndex(sig, ")")
	if lastParen < 0 || lastParen >= len(sig)-1 {
		return ""
	}
	ret := strings.TrimSpace(sig[lastParen+1:])
	return ret
}

// 将 codegraph 包的构建、查询、搜索、影响分析、Git 历史等功能
//

type mdSection struct {
	heading string
	body    string
	line    int
}

// ─── 支撑：图谱状态（迁移自 codegraph_tools.go 头部）───
type cgEntry struct {
	graph     *codegraph.Graph
	lastCheck time.Time // 上次变更检测时间
}

var (
	cgEntriesMu sync.Mutex
	cgEntries   = map[string]*cgEntry{} // key: normRoot(root)

	// cgSharedDB 主项目共享 SQLite 连接（web_server 对主项目 pair.db）。
	// 非主项目使用各自的 JSONStore（.pair/codegraph/graph.json，天然按根隔离）。
	cgSharedDB   *sql.DB
	cgSharedRoot string // 主项目根（SetCodeGraphRoot 设置）

	// ★ 自动增量更新缓存
	cgCheckMu sync.Mutex                           // 增量检测互斥锁
	cgSrcDirs = []string{"cmd", "internal", "pkg"} // 主要源文件目录
)

// normRoot 规范化项目根（Windows 大小写不敏感）。
func normRoot(root string) string {
	return strings.ToLower(filepath.Clean(root))
}

// SetCodeGraphDB 设置主项目共享数据库连接。
// 由 web_server.go / AgentBase.Init 调用。
func SetCodeGraphDB(db *sql.DB) {
	cgSharedDB = db
}

// SetCodeGraphRoot 记录主项目根（共享 DB 归属判定用）。
func SetCodeGraphRoot(root string) {
	cgSharedRoot = filepath.Clean(root)
}

// cgStoreFor 返回项目图谱存储：主项目（且共享 DB 可用）用 SQLiteStore
// （增量写入 pair.db）；其余项目用 JSONStore（各自 .pair/codegraph/graph.json）。
func cgStoreFor(root string) codegraph.GraphStore {
	if cgSharedDB != nil && SamePath(root, cgSharedRoot) {
		return codegraph.NewSQLiteStore(root, cgSharedDB)
	}
	return codegraph.NewStore(root)
}

// ensureCodeGraph 确保指定项目图谱已初始化。首次调用时自动加载或构建。
func ensureCodeGraph(root string) (*codegraph.Graph, error) {
	key := normRoot(root)
	cgEntriesMu.Lock()
	defer cgEntriesMu.Unlock()
	if e, ok := cgEntries[key]; ok && e.graph != nil {
		return e.graph, nil
	}
	e := &cgEntry{}
	// 带共享 DB 的项目用 SQLiteStore，否则 JSONStore
	if cgSharedDB != nil && SamePath(root, cgSharedRoot) {
		var loadErr error
		e.graph, _, loadErr = ensureCodeGraphStore(root, codegraph.NewSQLiteStore(root, cgSharedDB))
		if loadErr != nil {
			return nil, loadErr
		}
	} else {
		var loadErr error
		e.graph, _, loadErr = ensureCodeGraphStore(root, codegraph.NewStore(root))
		if loadErr != nil {
			return nil, loadErr
		}
	}
	cgEntries[key] = e
	return e.graph, nil
}

// ensureCodeGraphStore 在指定存储上确保图谱构建完成（复用 EnsureBuildIfNeeded 的
// 空图检测语义：持久化空图视同未构建自动重建）。
func ensureCodeGraphStore(root string, store codegraph.GraphStore) (*codegraph.Graph, bool, error) {
	if store.Exists() {
		graph, err := store.Load()
		if err != nil {
			return nil, false, err
		}
		if graph.Stats().EntityCount > 0 {
			return graph, false, nil
		}
	}
	moduleName := codegraph.DetectModuleName(root)
	config := codegraph.DefaultBuildConfig(root)
	config.ModuleName = moduleName
	config.AutoSave = true
	builder := codegraph.NewBuilder(config)
	builder.SetStore(store)
	_, err := builder.BuildFull()
	if err != nil {
		return nil, false, err
	}
	return builder.Graph(), true, nil
}

// EnsureCodeGraph 公开包装器，供 web_server.go 调用。
func EnsureCodeGraph(root string) (*codegraph.Graph, error) {
	return ensureCodeGraph(root)
}

// getCodeGraph 获取指定项目图谱实例（确保已初始化）。
// ★ 自动检测文件变更，需要时触发增量构建。
func getCodeGraph(root string) (*codegraph.Graph, error) {
	key := normRoot(root)
	cgEntriesMu.Lock()
	e := cgEntries[key]
	cgEntriesMu.Unlock()
	if e != nil && e.graph != nil {
		// ★ 每 30 秒检测一次源文件变更
		cgCheckMu.Lock()
		if time.Since(e.lastCheck) > 30*time.Second {
			e.lastCheck = time.Now()
			cgCheckMu.Unlock()
			tryIncrementalBuild(root)
		} else {
			cgCheckMu.Unlock()
		}
		cgEntriesMu.Lock()
		graph := cgEntries[key].graph
		cgEntriesMu.Unlock()
		return graph, nil
	}
	return ensureCodeGraph(root)
}

// resetCodeGraph 重置指定项目图谱（下次调用时重新构建）。
func resetCodeGraph(root string) {
	cgEntriesMu.Lock()
	delete(cgEntries, normRoot(root))
	cgEntriesMu.Unlock()
}

// needRebuild 轻量检测：检查是否有 .go 源文件比 graph.json 更新。
// 只扫描主要源目录（cmd/ internal/ pkg/），不反序列化图谱文件。
// 主项目共享 SQLiteStore 模式时检查 file_index 表是否有记录。
func needRebuild(root string) bool {
	// 主项目共享 SQLiteStore 模式：检查 file_index 表是否有记录即可
	if cgSharedDB != nil && SamePath(root, cgSharedRoot) {
		var count int
		err := cgSharedDB.QueryRow(`SELECT COUNT(*) FROM file_index`).Scan(&count)
		if err != nil || count == 0 {
			return true // 需要初始构建
		}
		return false // 有索引，让 IncrementalBuild 内部检测变更
	}

	graphPath := filepath.Join(root, ".pair", "codegraph", "graph.json")
	graphInfo, err := os.Stat(graphPath)
	if err != nil {
		return true // 文件不存在，需要重新构建
	}
	graphMtime := graphInfo.ModTime()

	// 快速扫描主要源目录
	for _, dir := range cgSrcDirs {
		srcDir := filepath.Join(root, dir)
		if fi, err := os.Stat(srcDir); err != nil || !fi.IsDir() {
			continue
		}
		hasNewer := false
		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(info.Name(), ".go") && info.ModTime().After(graphMtime) {
				hasNewer = true
				return filepath.SkipAll
			}
			return nil
		})
		if hasNewer {
			return true
		}
	}
	return false
}

// tryIncrementalBuild 尝试增量构建图谱，只在检测到文件变更时执行。
func tryIncrementalBuild(root string) {
	if !needRebuild(root) {
		return // 没有变更
	}

	moduleName := codegraph.DetectModuleName(root)
	config := codegraph.DefaultBuildConfig(root)
	config.ModuleName = moduleName
	config.AutoSave = true

	builder := codegraph.NewBuilder(config)
	builder.SetStore(cgStoreFor(root))

	result, err := builder.IncrementalBuild()
	if err != nil {
		log.Printf("[codegraph] 自动增量构建失败: %v", err)
		return
	}
	if result.FilesParsed == 0 {
		return // 没有实际变更
	}

	// 更新缓存（按项目）
	cgEntriesMu.Lock()
	cgEntries[normRoot(root)] = &cgEntry{graph: builder.Graph(), lastCheck: time.Now()}
	cgEntriesMu.Unlock()

	log.Printf("[codegraph] 自动增量完成: %d 文件变更, %d 新实体, %d 新关系",
		result.FilesParsed, result.EntitiesAdded, result.RelationsAdded)
}

// ── 工具注册 ──────────────────────────────────────────

// registerCodeGraphTools 注册所有代码知识图谱相关工具。

// ─── 支撑：嵌入缓存（迁移自 embedding.go）───
type EmbeddingBackend interface {
	// Embed 计算单条文本的嵌入向量。
	Embed(text string) ([]float32, error)
	// Available 后端是否可用。
	Available() bool
	// Dim 返回嵌入向量维度（0 表示未知）。
	Dim() int
}

// noopBackend 空实现——始终返回不可用。
type noopBackend struct{}

func (n *noopBackend) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("无可用嵌入后端")
}
func (n *noopBackend) Available() bool { return false }
func (n *noopBackend) Dim() int        { return 0 }

var (
	globalEmbedBackend   EmbeddingBackend
	embedBackendInitOnce sync.Once
)

// GetEmbeddingBackend 获取全局嵌入后端（懒初始化）。
// 优先 ONNX Runtime，失败时返回 noopBackend。
func GetEmbeddingBackend(root string) EmbeddingBackend {
	embedBackendInitOnce.Do(func() {
		// 尝试 ONNX Runtime
		onnx, err := NewONNXBackend(root)
		if err == nil {
			globalEmbedBackend = onnx
			return
		}
		// ONNX 不可用 → noop
		globalEmbedBackend = &noopBackend{}
	})
	return globalEmbedBackend
}

// ── 向量运算 ──────────────────────────────────────

// CosineSimilarity 计算两个向量的余弦相似度。
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		va, vb := float64(a[i]), float64(b[i])
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Normalize 原地归一化向量为单位向量。
func Normalize(vec []float32) {
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm == 0 {
		return
	}
	norm = math.Sqrt(norm)
	for i := range vec {
		vec[i] = float32(float64(vec[i]) / norm)
	}
}

// ── 嵌入缓存 ──────────────────────────────────────

// EmbeddingCache 嵌入向量的磁盘缓存（.pair/doc_embeddings.json）。
type EmbeddingCache struct {
	path    string
	mu      sync.RWMutex
	Vectors map[string][]float32 `json:"vectors"`
	Version int                  `json:"version"`
}

// LoadEmbeddingCache 从文件加载嵌入缓存。
func LoadEmbeddingCache(root string) *EmbeddingCache {
	path := filepath.Join(root, ".pair", "doc_embeddings.json")
	cache := &EmbeddingCache{
		path:    path,
		Vectors: make(map[string][]float32),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cache
	}
	_ = json.Unmarshal(data, &cache.Vectors)
	return cache
}

// Save 保存嵌入缓存到磁盘。
func (ec *EmbeddingCache) Save() error {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	dir := filepath.Dir(ec.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ec.Vectors)
	if err != nil {
		return err
	}
	return os.WriteFile(ec.path, data, 0o644)
}

// Get 获取指定键的嵌入向量。
func (ec *EmbeddingCache) Get(key string) []float32 {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.Vectors[key]
}

// Set 设置指定键的嵌入向量。
func (ec *EmbeddingCache) Set(key string, vec []float32) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Vectors[key] = vec
}

// MeanPool 对 BERT 输出的 last_hidden_state 做均值池化。
// hidden: [seqLen, dim] 的二维 flatten 数组。
// attentionMask: [seqLen] 的掩码（1=有效，0=填充）。
// 返回 [dim] 的池化向量。
func MeanPool(hidden []float32, attentionMask []int64, dim int) []float32 {
	seqLen := len(attentionMask)
	if seqLen == 0 || dim == 0 {
		return nil
	}
	result := make([]float32, dim)
	var maskSum float64
	for i := 0; i < seqLen; i++ {
		if attentionMask[i] == 0 {
			continue
		}
		maskSum++
		base := i * dim
		for j := 0; j < dim; j++ {
			result[j] += hidden[base+j]
		}
	}
	if maskSum > 0 {
		for j := 0; j < dim; j++ {
			result[j] = float32(float64(result[j]) / maskSum)
		}
	}
	return result
}

func splitMarkdownSections(content string) []mdSection {
	lines := strings.Split(content, "\n")
	var sections []mdSection
	var cur *mdSection
	headingRe := regexp.MustCompile("^#{1,4}\\s+(.+)$")
	for i, line := range lines {
		m := headingRe.FindStringSubmatch(line)
		if m != nil {
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &mdSection{heading: m[1], line: i + 1}
		} else if cur != nil {
			if cur.body != "" {
				cur.body += "\n"
			}
			cur.body += line
		}
	}
	if cur != nil {
		sections = append(sections, *cur)
	}
	return sections
}
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	full := append([]string{"-c", "core.quotepath=false"}, args...)
	c := exec.CommandContext(cctx, "git", full...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	c.Dir = dir
	out, err := c.CombinedOutput()
	res := CapOutput(DecodeCmdOutput(out), 16000)
	if cctx.Err() == context.DeadlineExceeded {
		return res + "\n[git 超时 30s 已终止]", nil
	}
	if err != nil {
		if strings.TrimSpace(res) == "" {
			return "", fmt.Errorf("git %s 失败: %v", strings.Join(args, " "), err)
		}
		return res, nil // 有输出（如 fatal: not a git repository）→ 回给 agent
	}
	if strings.TrimSpace(res) == "" {
		return "（无输出）", nil
	}
	return res, nil
}

// ── 支撑：ONNX 后端（独立二进制降级为不可用，调用方回退关键词搜索）──

// ── 非 CGo 编译的回退实现 ──
//
// 构建环境无 C 编译器（CGO_ENABLED=0）时使用此文件。
// NewONNXBackend 返回错误，调用者自动回退关键词搜索。

// ONNXBackend 在此构建模式下不可用。
type ONNXBackend struct{}

// NewONNXBackend 返回错误提示缺少 C 编译器。
func NewONNXBackend(root string) (*ONNXBackend, error) {
	return nil, fmt.Errorf("ONNX Runtime 需要 CGo。请安装 GCC 或 MinGW-w64 后重新构建")
}

// Embed 不支持。
func (b *ONNXBackend) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("ONNX 未启用（CGo 不可用）")
}

// Available 返回 false。
func (b *ONNXBackend) Available() bool { return false }

// Dim 返回 0。
func (b *ONNXBackend) Dim() int { return 0 }

// Close 无操作。
func (b *ONNXBackend) Close() error { return nil }
