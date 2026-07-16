package agent

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hoonfeng/paircode/pkg/codegraph"
)

// registerExtraCodeGraphTools 注册额外的代码图谱工具（#29-#30）。
// 在 RegisterDefaultTools 中调用。
func registerExtraCodeGraphTools(r *Registry, root string) {

		// ---- 19. codegraph_find_entry_points ----
	r.Register(&Tool{
		Name: "codegraph_find_entry_points", Description: "发现应用程序入口点和执行起点。",
		Parameters: objSchema(props{"entryType": strProp("可选：main/http_handler/cli_command/all，默认 all"), "limit": intProp("可选：最大返回数（默认 50）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entryType := argStr(args, "entry_type"); limit := argInt(args, "limit", 50)
			g, err := getCodeGraph(root); if err != nil { return "", err }
			type ep struct{ name, kind, file string; line int }; var entries []ep
			if entryType == "" || entryType == "all" || entryType == "main" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) { if fn.Name == "main" && fn.FilePath != "" { entries = append(entries, ep{fn.Name, "main", fn.FilePath, fn.Line}) } }
			}
			if entryType == "" || entryType == "all" || entryType == "http_handler" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range []string{"HandleFunc", "Handle", "router.GET", "echo.GET", "gin.GET", "http.Handle"} { if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) { entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line}); break } }
				}
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityMethod) { if strings.Contains(fn.Signature, "ServeHTTP") { entries = append(entries, ep{fn.Name, "http_handler", fn.FilePath, fn.Line}) } }
			}
			if entryType == "" || entryType == "all" || entryType == "cli_command" {
				for _, fn := range g.GetEntitiesByKind(codegraph.EntityFunction) {
					for _, p := range []string{"cobra", "Execute", "RunE", "flag.Parse"} { if strings.Contains(fn.Signature, p) || strings.Contains(fn.Name, p) { entries = append(entries, ep{fn.Name, "cli_command", fn.FilePath, fn.Line}); break } }
				}
			}
			seen := map[string]bool{}; var uniq []ep
			for _, e := range entries { k := fmt.Sprintf("%s:%d:%s", e.file, e.line, e.name); if !seen[k] { seen[k] = true; uniq = append(uniq, e) } }
			sort.Slice(uniq, func(i, j int) bool { if uniq[i].kind != uniq[j].kind { return uniq[i].kind < uniq[j].kind }; return uniq[i].file < uniq[j].file })
			if len(uniq) > limit { uniq = uniq[:limit] }
			if len(uniq) == 0 { return "未发现入口点。", nil }
			var b strings.Builder; b.WriteString(fmt.Sprintf("发现 %d 个入口点：\n\n", len(uniq)))
			for _, e := range uniq { b.WriteString(fmt.Sprintf("  [%s] %s (%s:%d)\n", e.kind, e.name, e.file, e.line)) }
			return b.String(), nil
		},
	})

	// ---- 20. codegraph_find_hot_paths ----
	r.Register(&Tool{
		Name: "codegraph_find_hot_paths", Description: "查找最常被调用的函数。",
		Parameters: objSchema(props{"limit": intProp("可选：最大返回数（默认 20）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			limit := argInt(args, "limit", 20); if limit <= 0 || limit > 100 { limit = 20 }
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			type hf struct{ name, file string; callers int }; var list []hf
			for _, fn := range all { n := len(g.GetPredecessors(fn.ID, codegraph.RelCalls)); if n > 0 { list = append(list, hf{fn.Name, fn.FilePath, n}) } }
			sort.Slice(list, func(i, j int) bool { return list[i].callers > list[j].callers })
			if len(list) > limit { list = list[:limit] }
			if len(list) == 0 { return "未发现被调用的函数。", nil }
			var b strings.Builder; b.WriteString(fmt.Sprintf("最热函数（共 %d 条）：\n\n", len(list)))
			for i, h := range list { b.WriteString(fmt.Sprintf("%d. %s (%s) - %d 个调用者\n", i+1, h.name, h.file, h.callers)) }
			return b.String(), nil
		},
	})

	// ---- 21. codegraph_find_by_imports ----
	r.Register(&Tool{
		Name: "codegraph_find_by_imports", Description: "查找所有导入指定模块的文件。",
		Parameters: objSchema(props{"moduleName": strProp("模块/包名"), "matchMode": strProp("可选：exact/prefix/contains/fuzzy，默认 contains"), "limit": intProp("可选：最大返回数（默认 50）")}, "moduleName"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			mn := strings.TrimSpace(argStr(args, "module_name")); mm := argStr(args, "match_mode"); limit := argInt(args, "limit", 50)
			if mn == "" { return "", fmt.Errorf("moduleName 不能为空") }; if mm == "" { mm = "contains" }
			fm := map[string][]string{}
			for _, fe := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if fe.FilePath == "" { continue }
				for _, imp := range g.GetSuccessors(fe.ID, codegraph.RelImports) {
					im := imp.Name; if im == "" { im = imp.FQN }; m := false
					switch mm { case "exact": m = strings.EqualFold(im, mn); case "prefix": m = strings.HasPrefix(strings.ToLower(im), strings.ToLower(mn)); default: m = strings.Contains(strings.ToLower(im), strings.ToLower(mn)) }
					if m { fm[fe.FilePath] = append(fm[fe.FilePath], im) }
				}
			}
			if len(fm) == 0 { return fmt.Sprintf("未找到导入「%s」的文件。", mn), nil }
			fs := make([]string, 0, len(fm)); for f := range fm { fs = append(fs, f) }; sort.Strings(fs)
			if len(fs) > limit { fs = fs[:limit] }
			var b strings.Builder; b.WriteString(fmt.Sprintf("导入「%s」的文件（共 %d 个）：\n\n", mn, len(fm)))
			for _, f := range fs { b.WriteString(fmt.Sprintf("  %s\n    - %s\n", f, fm[f][0])) }
			if len(fs) < len(fm) { b.WriteString(fmt.Sprintf("\n... 还有 %d 个文件未显示。", len(fm)-len(fs))) }
			return b.String(), nil
		},
	})

	// ---- 22. codegraph_get_detailed_symbol ----
	r.Register(&Tool{
		Name: "codegraph_get_detailed_symbol", Description: "获取符号详细上下文（源码+调用者+被调用者）。",
		Parameters: objSchema(props{"query": strProp("符号名"), "includeSource": boolProp("可选：包含源码（默认 true）")}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(argStr(args, "query"))
			if query == "" { return "", fmt.Errorf("query 不能为空") }
			g, err := getCodeGraph(root); if err != nil { return "", err }
			qe := codegraph.NewQueryEngine(g)
			entities := g.SearchEntities(query)
			if len(entities) == 0 { return fmt.Sprintf("未找到符号: %s", query), nil }
			if len(entities) > 5 { entities = entities[:5] }
			var b strings.Builder
			for i, e := range entities {
				if i > 0 { b.WriteString("\n" + strings.Repeat("-", 60) + "\n\n") }
				b.WriteString(fmt.Sprintf("## %s (%s)\n文件: %s:%d-%d\n", e.Name, string(e.Kind), e.FilePath, e.Line, e.EndLine))
				if e.Signature != "" { b.WriteString(fmt.Sprintf("签名: %s\n", e.Signature)) }
				callers := qe.GetCallers(e.Name); b.WriteString(fmt.Sprintf("调用者: %d\n", len(callers)))
				for j := 0; j < len(callers) && j < 10; j++ { b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", callers[j].CallerName, callers[j].CallerFile, callers[j].CallerLine)) }
				callees := qe.GetCallees(e.Name); b.WriteString(fmt.Sprintf("被调用者: %d\n", len(callees)))
				for j := 0; j < len(callees) && j < 10; j++ { b.WriteString(fmt.Sprintf("  - %s (%s:%d)\n", callees[j].CalleeName, callees[j].CallerFile, callees[j].CallerLine)) }
				if argBool(args, "include_source") && e.FilePath != "" {
					d2, rErr := os.ReadFile(filepath.Join(root, e.FilePath))
					if rErr == nil {
						lines := strings.Split(string(d2), "\n")
						start, end := max(0, e.Line-1), e.EndLine
						if end <= start { end = start + 20 }; end = min(len(lines), min(start+80, end))
						if start < end { b.WriteString(fmt.Sprintf("\n### 源码\n")); for ln := start; ln < end; ln++ { b.WriteString(fmt.Sprintf("%d\t%s\n", ln+1, lines[ln])) } }
					}
				}
			}
			return b.String(), nil
		},
	})

	// ---- 23. codegraph_find_dead_imports ----
	r.Register(&Tool{
		Name: "codegraph_find_dead_imports", Description: "查找已导入但从未使用的模块。",
		Parameters: objSchema(props{"file": strProp("可选：指定文件"), "limit": intProp("可选：最大返回数（默认 50）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			fileFilter := strings.TrimSpace(argStr(args, "file")); limit := argInt(args, "limit", 50)
			type di struct{ file, imp string }; var dead []di
			for _, fe := range g.GetEntitiesByKind(codegraph.EntityFile) {
				if fe.FilePath == "" || (fileFilter != "" && !strings.Contains(fe.FilePath, fileFilter)) { continue }
				imps := g.GetSuccessors(fe.ID, codegraph.RelImports)
				if len(imps) == 0 { continue }
				fEnts := g.GetEntitiesByFile(fe.FilePath)
				for _, imp := range imps {
					im := imp.Name; if im == "" { im = imp.FQN }; ls := im
					if i := strings.LastIndex(im, "/"); i >= 0 { ls = im[i+1:] }; if ls == "" { ls = im }
					used := false
					for _, fe2 := range fEnts {
						if (fe2.Kind == codegraph.EntityFunction || fe2.Kind == codegraph.EntityMethod) && (strings.Contains(fe2.FQN, ls) || strings.Contains(fe2.Signature, ls)) { used = true; break }
						if (fe2.Kind == codegraph.EntityType || fe2.Kind == codegraph.EntityStruct) && strings.Contains(fe2.Signature, ls) { used = true; break }
					}
					if !used { dead = append(dead, di{fe.FilePath, im}) }
				}
			}
			if len(dead) == 0 { return "未发现死导入。", nil }
			sort.Slice(dead, func(i, j int) bool { if dead[i].file != dead[j].file { return dead[i].file < dead[j].file }; return dead[i].imp < dead[j].imp })
			if len(dead) > limit { dead = dead[:limit] }
			byFile := map[string][]string{}
			for _, d := range dead { byFile[d.file] = append(byFile[d.file], d.imp) }
			var b strings.Builder; b.WriteString(fmt.Sprintf("发现 %d 个可能未使用的导入：\n\n", len(dead)))
			for f, imps := range byFile { b.WriteString(fmt.Sprintf("  %s\n", f)); for _, imp := range imps { b.WriteString(fmt.Sprintf("    - %s\n", imp)) } }
			return b.String(), nil
		},
	})

	// ---- 24. codegraph_search_by_error ----
	r.Register(&Tool{
		Name: "codegraph_search_by_error", Description: "查找抛出或处理错误的函数。",
		Parameters: objSchema(props{"mode": strProp("可选：throws/catches/any，默认 any"), "errorType": strProp("可选：错误类型过滤"), "limit": intProp("可选：最大返回数（默认 50）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			mode := argStr(args, "mode"); eType := strings.TrimSpace(argStr(args, "error_type")); limit := argInt(args, "limit", 50)
			all := g.GetEntitiesByKind(codegraph.EntityFunction)
			all = append(all, g.GetEntitiesByKind(codegraph.EntityMethod)...)
			type ht struct{ name, file string; line int; pats []string }; var res []ht
			for _, fn := range all {
				var m []string; sig := fn.Signature + " " + fn.Doc
				if mode == "" || mode == "any" || mode == "throws" {
					for _, p := range []string{"errors.New", "fmt.Errorf", "panic(", "return.*err"} { if ok, _ := regexp.MatchString(p, sig); ok { m = append(m, p) } }
				}
				if mode == "" || mode == "any" || mode == "catches" {
					for _, p := range []string{"if err != nil", "if err == nil", "catch(", "except "} { if strings.Contains(sig, p) { m = append(m, p) } }
				}
				if eType != "" { if !strings.Contains(fn.Signature, eType) && !strings.Contains(fn.FQN, eType) { continue }; m = append(m, "err:"+eType) }
				if len(m) > 0 { res = append(res, ht{fn.Name, fn.FilePath, fn.Line, m}) }
			}
			if len(res) == 0 { return "未找到错误匹配。", nil }
			sort.Slice(res, func(i, j int) bool { if res[i].file != res[j].file { return res[i].file < res[j].file }; return res[i].name < res[j].name })
			if len(res) > limit { res = res[:limit] }
			var b strings.Builder; b.WriteString(fmt.Sprintf("匹配 %d 个函数：\n\n", len(res)))
			for _, h := range res { b.WriteString(fmt.Sprintf("  %s (%s:%d) %s\n", h.name, h.file, h.line, strings.Join(h.pats, ", "))) }
			return b.String(), nil
		},
	})

	// ---- 25. codegraph_index_markdown ----
	r.Register(&Tool{
		Name: "codegraph_index_markdown", Description: "索引 Markdown 文档，按标题分段。有 ONNX 则计算嵌入向量。",
		Parameters: objSchema(props{"path": strProp("可选：文件路径；省略则扫描全部 .md 文件")}),
		ReadOnly: false,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			mdPath := strings.TrimSpace(argStr(args, "path")); var mds []string
			if mdPath != "" { mds = append(mds, mdPath) } else {
				filepath.WalkDir(root, func(p string, d fs.DirEntry, werr error) error {
					if werr != nil { return nil }
					if d.IsDir() { if d.Name() == ".git" || d.Name() == "node_modules" { return fs.SkipDir }; return nil }
					if strings.HasSuffix(strings.ToLower(p), ".md") { rel, _ := filepath.Rel(root, p); mds = append(mds, rel) }
					return nil
				})
			}
			embedCache := LoadEmbeddingCache(root); backend := GetEmbeddingBackend(root); idxd := 0
			for _, mf := range mds {
				d2, rErr := os.ReadFile(filepath.Join(root, mf)); if rErr != nil { continue }
				sections := splitMarkdownSections(string(d2))
				for i, sec := range sections {
					dk := fmt.Sprintf("doc:%s#%d", mf, i)
					g.AddEntity(&codegraph.Entity{ID: dk, Kind: codegraph.EntityDocSection, Name: fmt.Sprintf("%s - %s", mf, sec.heading), FilePath: mf, Line: sec.line, Doc: sec.body})
					if backend.Available() { if vec, eErr := backend.Embed(sec.heading + "\n" + sec.body); eErr == nil && len(vec) > 0 { embedCache.Set(dk, vec) } }
				}
				idxd++
			}
			if backend.Available() { embedCache.Save(); return fmt.Sprintf("已索引 %d 个文件（含向量），共 %d 个节。", idxd, len(mds)), nil }
			return fmt.Sprintf("已索引 %d 个文件（无向量），共 %d 个节。", idxd, len(mds)), nil
		},
	})

	// ---- 26. codegraph_search_docs ----
	r.Register(&Tool{
		Name: "codegraph_search_docs", Description: "搜索已索引文档。优先向量语义搜索，回退关键词。",
		Parameters: objSchema(props{"query": strProp("搜索关键词"), "limit": intProp("可选：最大返回数（默认 5）")}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			g, err := getCodeGraph(root); if err != nil { return "", err }
			query := strings.TrimSpace(argStr(args, "query")); limit := argInt(args, "limit", 5)
			if query == "" { return "", fmt.Errorf("query 不能为空") }
			sections := g.GetEntitiesByKind(codegraph.EntityDocSection)
			if len(sections) == 0 { return "未索引文档。请先运行 codegraph_index_markdown。", nil }
			backend := GetEmbeddingBackend(root)
			if backend.Available() {
				embedCache := LoadEmbeddingCache(root)
				if qv, qErr := backend.Embed(query); qErr == nil && len(qv) > 0 {
					type vm struct{ s *codegraph.Entity; sim float64 }; var vms []vm
					for _, sec := range sections { if vec := embedCache.Get(sec.ID); vec != nil { vms = append(vms, vm{sec, CosineSimilarity(qv, vec)}) } }
					if len(vms) > 0 {
						sort.Slice(vms, func(i, j int) bool { return vms[i].sim > vms[j].sim })
						if len(vms) > limit { vms = vms[:limit] }
						var b strings.Builder; b.WriteString(fmt.Sprintf("【向量搜索】找到 %d 个节：\n\n", len(vms)))
						for _, m := range vms { b.WriteString(fmt.Sprintf("### %s\n相似度: %.3f\n\n%s\n\n---\n\n", m.s.Name, m.sim, m.s.Doc)) }
						return b.String(), nil
					}
				}
			}
			type km struct{ s *codegraph.Entity; score int }; var kms []km; q := strings.ToLower(query)
			for _, sec := range sections {
				score := 0; name := strings.ToLower(sec.Name); doc := strings.ToLower(sec.Doc)
				for _, w := range strings.Fields(q) { if len(w) >= 2 { if strings.Contains(name, w) { score += 3 }; if strings.Contains(doc, w) { score++ } } }
				if score > 0 { kms = append(kms, km{sec, score}) }
			}
			sort.Slice(kms, func(i, j int) bool { return kms[i].score > kms[j].score })
			if len(kms) > limit { kms = kms[:limit] }; if len(kms) == 0 { return "未找到匹配。", nil }
			var b strings.Builder; b.WriteString(fmt.Sprintf("【关键词搜索】找到 %d 个节：\n\n", len(kms)))
			for _, m := range kms { b.WriteString(fmt.Sprintf("### %s\n%s\n\n---\n\n", m.s.Name, m.s.Doc)) }
			return b.String(), nil
		},
	})

	// ---- 27. codegraph_verify_design ----
	r.Register(&Tool{
		Name: "codegraph_verify_design", Description: "检查设计文档中的代码引用是否存在。",
		Parameters: objSchema(props{"docFile": strProp("设计文档路径")}, "docFile"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			docFile := strings.TrimSpace(argStr(args, "doc_file"))
			if docFile == "" { return "", fmt.Errorf("docFile 不能为空") }
			d2, rErr := os.ReadFile(filepath.Join(root, docFile)); if rErr != nil { return "", fmt.Errorf("读取失败: %w", rErr) }
			re := regexp.MustCompile("[`]([^`]+)[`]")
			idents := re.FindAllStringSubmatch(string(d2), -1)
			uniq := map[string]bool{}; var syms []string
			for _, m := range idents { s := strings.TrimSpace(m[1]); if s != "" && len(s) > 1 && !uniq[s] { uniq[s] = true; syms = append(syms, s) } }
			if len(syms) == 0 { return "文档中未发现代码标识符。", nil }
			g, err := getCodeGraph(root); if err != nil { return "", err }
			var found, nf []string
			for _, sym := range syms { if len(g.SearchEntities(sym)) > 0 { found = append(found, sym) } else { nf = append(nf, sym) } }
			pct := 0; if len(syms) > 0 { pct = len(found) * 100 / len(syms) }
			return fmt.Sprintf("共 %d 个标识符，存在 %d 个 (%d%%)，缺失 %d 个", len(syms), len(found), pct, len(nf)), nil
		},
	})

	// ---- 28. codegraph_pr_context ----
	r.Register(&Tool{
		Name: "codegraph_pr_context", Description: "分析分支变更影响范围。",
		Parameters: objSchema(props{"baseBranch": strProp("可选：基准分支（默认 main）")}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			bb := strings.TrimSpace(argStr(args, "base_branch"))
			if bb == "" { bb = "main" }
			diff, dErr := runGit(ctx, root, "diff", "--stat", bb+"...HEAD")
			if dErr != nil { return "", fmt.Errorf("git diff: %w", dErr) }
			if strings.TrimSpace(diff) == "" || diff == "(无输出)" { return "无差异。", nil }
			no, _ := runGit(ctx, root, "diff", "--name-only", bb+"...HEAD")
			files := strings.Fields(strings.TrimSpace(no))
			g, err := getCodeGraph(root)
			if err != nil { return diff, nil }
			var b strings.Builder; b.WriteString(fmt.Sprintf("## PR 影响分析\n### 变更文件\n%s\n\n", diff))
			for _, f := range files {
				f = strings.TrimSpace(f); if f == "" { continue }
				fe := g.GetEntitiesByFile(f)
				if len(fe) > 0 {
					cs := map[string]bool{}
					for _, e := range fe {
						if e.Kind == codegraph.EntityFunction || e.Kind == codegraph.EntityMethod {
							for _, c := range codegraph.NewQueryEngine(g).GetCallers(e.Name) { cs[c.CallerName+"@"+c.CallerFile] = true }
						}
					}
					b.WriteString(fmt.Sprintf("- **%s** 影响 %d 个调用者\n", f, len(cs)))
				} else { b.WriteString(fmt.Sprintf("- %s\n", f)) }
			}
			return b.String(), nil
		},
	})

// ── 29. codegraph_find_by_signature — 按签名查找函数 ──
	r.Register(&Tool{
		Name: "codegraph_find_by_signature",
		Description: "按结构特征（参数数、返回类型、名称模式）查找函数。",
		Parameters: objSchema(props{
			"namePattern": strProp("可选：函数名通配模式，如 'get*'、'*Handler'"),
			"paramCount":  intProp("可选：精确参数个数"),
			"minParams":   intProp("可选：最少参数个数"),
			"maxParams":   intProp("可选：最多参数个数"),
			"returnType":  strProp("可选：返回类型，如 'error'、'string'"),
			"limit":       intProp("可选：最大返回数（默认 50）"),
		}),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			namePattern := strings.TrimSpace(argStr(args, "name_pattern"))
			paramCount := argInt(args, "param_count", 0)
			minParams := argInt(args, "min_params", 0)
			maxParams := argInt(args, "max_params", 0)
			returnType := strings.TrimSpace(argStr(args, "return_type"))
			limit := argInt(args, "limit", 50)
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
			type sm struct{ n, k, f, s, r string; l, c int }
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

	// ── 30. codegraph_explore — 自然语言→源码 ──
	r.Register(&Tool{
		Name: "codegraph_explore",
		Description: "一站式代码理解工具。用自然语言或符号名探索代码，返回相关源码和位置。分析代码的首选工具。",
		Parameters: objSchema(props{"query": strProp("自然语言问题或符号名"), "maxFiles": intProp("可选：最大返回文件数（默认 8）")}, "query"),
		ReadOnly: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			query := strings.TrimSpace(argStr(args, "query"))
			mx := argInt(args, "max_files", 8)
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
			type se struct{ e *codegraph.Entity; s float64 }
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
