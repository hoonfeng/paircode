package agent

import (
	"context"
	"fmt"
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
