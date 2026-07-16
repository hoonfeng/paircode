package agent

// find_symbol 工具：基于 Go AST 解析项目源文件，搜索所有符号（函数、类型、变量、常量等）
// 的定义位置。接受 symbol（必须）和可选 scope（包路径或文件路径）参数。
// 返回文件路径、行号、符号类型和简短签名。
//
// 使用 go/parser + go/ast + go/token 实现纯静态分析，无需 LSP 语言服务器。

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── 工具注册 ──

// registerFindSymbolTool 注册 find_symbol 工具。
func registerFindSymbolTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "find_symbol",
		Description: "基于 Go AST 在项目中搜索符号（函数、类型、结构体、接口、常量、变量等）的定义位置。" +
			"支持按名称搜索（精确匹配或子串匹配），可通过 scope 限定搜索范围为指定包路径或文件路径。" +
			"返回每个匹配符号的文件路径、行号、符号类型和简短签名。" +
			"纯静态分析，无需语言服务器。当前仅支持 Go 语言。",
		Parameters: objSchema(props{
			"symbol": strProp("要搜索的符号名称（精确匹配；支持子串匹配用 find_any=true）"),
			"scope":  strProp("可选：限定搜索范围。可以是包路径（如 './internal/lsp'）或文件路径（如 'pkg/widget/button.go'）"),
		}, "symbol"),
		ReadOnly: true,
		Handler:  findSymbolHandler(root),
	})
}

// ── 符号模型 ──

// astSymbol 表示一个从 AST 中提取的符号。
type astSymbol struct {
	Name      string // 符号名
	Kind      string // 种类：function, method, struct, interface, type, constant, variable
	FilePath  string // 文件路径（项目内相对路径）
	Line      int    // 行号（1 基）
	Signature string // 简短签名
}

// ── 工具处理函数 ──

func findSymbolHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		symbolName := argStr(args, "symbol")
		if symbolName == "" {
			return "", fmt.Errorf("symbol 不能为空")
		}
		scope := argStr(args, "scope")

		// 确定搜索目录：scope 可以是文件路径或包路径
		searchDir := root
		scopeFileOnly := "" // 如果 scope 是单个文件，这里存解析后的绝对路径
		if scope != "" {
			absScope, err := resolvePath(root, scope)
			if err != nil {
				return "", fmt.Errorf("scope 路径无效: %w", err)
			}
			fi, err := os.Stat(absScope)
			if err != nil {
				return "", fmt.Errorf("scope 路径不可访问: %w", err)
			}
			if fi.IsDir() {
				searchDir = absScope
			} else {
				// scope 是单个文件
				if filepath.Ext(absScope) != ".go" {
					return "", fmt.Errorf("scope 文件不是 Go 源文件（.go）：%s", scope)
				}
				scopeFileOnly = absScope
			}
		}

		// 扫描所有匹配的符号
		var results []astSymbol

		if scopeFileOnly != "" {
			// 只扫描单个文件
			rel, _ := filepath.Rel(root, scopeFileOnly)
			syms, err := scanGoFileSymbols(scopeFileOnly, filepath.ToSlash(rel))
			if err == nil {
				for _, s := range syms {
					if matchSymbol(s, symbolName) {
						results = append(results, s)
					}
				}
			}
		} else {
			// 扫描目录中的所有 Go 文件
			err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".go" {
					return nil
				}
				// 跳过测试文件？（不过滤，让用户用 scope 控制）
				rel, _ := filepath.Rel(root, path)
				syms, err := scanGoFileSymbols(path, filepath.ToSlash(rel))
				if err != nil {
					return nil // 跳过不可读或解析失败的文件
				}
				for _, s := range syms {
					if matchSymbol(s, symbolName) {
						results = append(results, s)
					}
				}
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("扫描目录失败: %w", err)
			}
		}

		if len(results) == 0 {
			if scope != "" {
				return fmt.Sprintf("在 %s 中未找到符号 %q", scope, symbolName), nil
			}
			return fmt.Sprintf("项目中未找到符号 %q", symbolName), nil
		}

		// 按文件路径排序
		sort.Slice(results, func(i, j int) bool {
			if results[i].FilePath != results[j].FilePath {
				return results[i].FilePath < results[j].FilePath
			}
			return results[i].Line < results[j].Line
		})

		// 格式化输出
		var b strings.Builder
		fmt.Fprintf(&b, "符号 %q 在项目中共找到 %d 处定义：\n\n", symbolName, len(results))

		// 按文件分组
		type fileGroup struct {
			filePath string
			symbols  []astSymbol
		}
		groups := map[string]*fileGroup{}
		var groupOrder []string
		for _, s := range results {
			if _, ok := groups[s.FilePath]; !ok {
				groups[s.FilePath] = &fileGroup{filePath: s.FilePath}
				groupOrder = append(groupOrder, s.FilePath)
			}
			groups[s.FilePath].symbols = append(groups[s.FilePath].symbols, s)
		}

		for _, fp := range groupOrder {
			g := groups[fp]
			fmt.Fprintf(&b, "  %s:\n", g.filePath)
			for _, s := range g.symbols {
				sig := strings.TrimSpace(s.Signature)
				if sig != "" {
					fmt.Fprintf(&b, "    %s %s → %d  %s\n", s.Kind, s.Name, s.Line, sig)
				} else {
					fmt.Fprintf(&b, "    %s %s → %d\n", s.Kind, s.Name, s.Line)
				}
			}
			b.WriteString("\n")
		}

		return b.String(), nil
	}
}

// matchSymbol 判断符号是否匹配搜索的符号名。
// 当前采用大小写敏感的子串匹配（symbolName 完全匹配则优先）。
func matchSymbol(s astSymbol, symbolName string) bool {
	// 精确匹配优先
	if s.Name == symbolName {
		return true
	}
	// 子串匹配
	return strings.Contains(s.Name, symbolName)
}

// scanGoFileSymbols 使用 Go AST 解析单个 Go 文件，提取其中所有符号。
// absPath 为文件绝对路径，relPath 为项目内相对路径（用于输出）。
func scanGoFileSymbols(absPath, relPath string) ([]astSymbol, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", absPath, err)
	}

	var syms []astSymbol

	// 遍历 AST
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// 函数/方法声明
			pos := fset.Position(node.Pos())
			kind := "function"
			sig := buildFuncSignature(node)

			// 检查是否有接收者 → 方法
			if node.Recv != nil && len(node.Recv.List) > 0 {
				kind = "method"
			}

			syms = append(syms, astSymbol{
				Name:      node.Name.Name,
				Kind:      kind,
				FilePath:  relPath,
				Line:      pos.Line,
				Signature: sig,
			})

		case *ast.GenDecl:
			// 通用声明：type、const、var、import
			switch node.Tok {
			case token.TYPE:
				for _, spec := range node.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					pos := fset.Position(ts.Pos())
					kind := "type"
					sig := ""

					switch t := ts.Type.(type) {
					case *ast.StructType:
						kind = "struct"
						sig = buildStructSignature(t)
					case *ast.InterfaceType:
						kind = "interface"
						sig = buildInterfaceSignature(t)
					default:
						// 普通类型别名，如 type MyInt int
						sig = typeExprString(ts.Type)
					}

					syms = append(syms, astSymbol{
						Name:      ts.Name.Name,
						Kind:      kind,
						FilePath:  relPath,
						Line:      pos.Line,
						Signature: sig,
					})
				}

			case token.CONST:
				for _, spec := range node.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vs.Names {
						pos := fset.Position(name.Pos())
						sig := ""
						if i < len(vs.Values) {
							sig = "= " + exprString(vs.Values[i])
						} else if vs.Type != nil {
							sig = typeExprString(vs.Type)
						}
						syms = append(syms, astSymbol{
							Name:      name.Name,
							Kind:      "constant",
							FilePath:  relPath,
							Line:      pos.Line,
							Signature: sig,
						})
					}
				}

			case token.VAR:
				for _, spec := range node.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, name := range vs.Names {
						pos := fset.Position(name.Pos())
						sig := ""
						if vs.Type != nil {
							sig = typeExprString(vs.Type)
						}
						if i < len(vs.Values) {
							if sig != "" {
								sig += " = " + exprString(vs.Values[i])
							} else {
								sig = "= " + exprString(vs.Values[i])
							}
						}
						syms = append(syms, astSymbol{
							Name:      name.Name,
							Kind:      "variable",
							FilePath:  relPath,
							Line:      pos.Line,
							Signature: sig,
						})
					}
				}
			}
		}
		return true
	})

	return syms, nil
}

// ── 签名构建辅助函数 ──

// buildFuncSignature 构建函数/方法的简短签名。
func buildFuncSignature(fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		b.WriteString("(")
		recv := fn.Recv.List[0]
		b.WriteString(typeExprString(recv.Type))
		b.WriteString(")")
	}
	b.WriteString("(")
	for i, param := range fn.Type.Params.List {
		if i > 0 {
			b.WriteString(", ")
		}
		names := make([]string, 0, len(param.Names))
		for _, n := range param.Names {
			names = append(names, n.Name)
		}
		if len(names) > 0 {
			b.WriteString(strings.Join(names, ", "))
			b.WriteString(" ")
		}
		b.WriteString(typeExprString(param.Type))
	}
	b.WriteString(")")
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		b.WriteString(" ")
		if len(fn.Type.Results.List) == 1 {
			b.WriteString(typeExprString(fn.Type.Results.List[0].Type))
		} else {
			b.WriteString("(")
			for i, result := range fn.Type.Results.List {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(typeExprString(result.Type))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

// buildStructSignature 构建结构体的简短签名（含字段数）。
func buildStructSignature(st *ast.StructType) string {
	if st.Fields == nil {
		return "struct{}"
	}
	count := len(st.Fields.List)
	if count == 0 {
		return "struct{}"
	}
	return fmt.Sprintf("struct{%d fields}", count)
}

// buildInterfaceSignature 构建接口的简短签名（含方法数）。
func buildInterfaceSignature(iface *ast.InterfaceType) string {
	if iface.Methods == nil {
		return "interface{}"
	}
	count := len(iface.Methods.List)
	if count == 0 {
		return "interface{}"
	}
	return fmt.Sprintf("interface{%d methods}", count)
}

// typeExprString 将 AST 类型表达式转为可读字符串。
func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		if t.Dir == ast.SEND {
			return "chan<- " + typeExprString(t.Value)
		} else if t.Dir == ast.RECV {
			return "<-chan " + typeExprString(t.Value)
		}
		return "chan " + typeExprString(t.Value)
	default:
		return exprString(expr)
	}
}

// exprString 将任意 AST 表达式转为简短字符串。
func exprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.BasicLit:
		return t.Value
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.CallExpr:
		return typeExprString(t.Fun) + "(...)"
	case *ast.CompositeLit:
		return typeExprString(t.Type) + "{...}"
	case *ast.BinaryExpr:
		return exprString(t.X) + " " + t.Op.String() + " " + exprString(t.Y)
	case *ast.UnaryExpr:
		return t.Op.String() + exprString(t.X)
	case *ast.IndexExpr:
		return typeExprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.SliceExpr:
		return typeExprString(t.X) + "[...]"
	case *ast.ParenExpr:
		return "(" + exprString(t.X) + ")"
	case *ast.KeyValueExpr:
		return exprString(t.Key) + ": " + exprString(t.Value)
	case *ast.ArrayType:
		return typeExprString(t)
	case *ast.MapType:
		return typeExprString(t)
	case *ast.FuncLit:
		return "func(...) {...}"
	case *ast.TypeAssertExpr:
		return typeExprString(t.X) + ".(type)"
	default:
		return fmt.Sprintf("%T", expr)
	}
}
