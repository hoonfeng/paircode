package codegraph

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ── Go Builder ────────────────────────────────────────

// GoBuilder 使用 Go AST 解析源文件，构建语法结构层图谱。
// 只解析 Go 文件（.go），提取包、文件、类型、函数、方法、变量。
type GoBuilder struct {
	ModuleName string // go.mod 中的模块名（用于生成 FQN）
	root       string // 工作区根目录
	graph      *Graph
}

// NewGoBuilder 创建新的 Go AST 构建器。
func NewGoBuilder(root, moduleName string) *GoBuilder {
	return &GoBuilder{
		ModuleName: moduleName,
		root:       root,
		graph:      NewGraph(),
	}
}

// Graph 返回当前构建的图。
func (b *GoBuilder) Graph() *Graph { return b.graph }

// Reset 重置构建器（清空图）。
func (b *GoBuilder) Reset() { b.graph = NewGraph() }

// ── 文件解析 ──────────────────────────────────────────

// ParseFile 解析单个 Go 文件，提取其中所有实体和关系并加入图。
// filePath 为工作区相对路径（如 "pkg/codegraph/graph.go"）。
// 返回该文件根的实体 ID。
func (b *GoBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("解析文件 %s 失败: %w", filePath, err)
	}

	// ── 1. 包实体 ──
	pkgName := f.Name.Name
	pkgPath := b.packagePathFromFile(filePath)
	pkgID := EntityID(EntityPackage, "", pkgPath)

	b.graph.AddEntity(&Entity{
		ID:       pkgID,
		Kind:     EntityPackage,
		Name:     pkgName,
		FQN:      b.fqn(pkgPath),
		FilePath: filePath,
	})

	// ── 2. 文件实体 ──
	fileName := filepath.Base(filePath)
	fileID := EntityID(EntityFile, pkgPath, fileName)
	fileEntity := &Entity{
		ID:       fileID,
		Kind:     EntityFile,
		Name:     fileName,
		FQN:      b.fqn(pkgPath + "." + fileName),
		FilePath: filePath,
		Line:     1,
	}
	b.graph.AddEntity(fileEntity)

	// 包包含文件
	b.graph.AddRelation(&Relation{
		SourceID: pkgID,
		TargetID: fileID,
		Kind:     RelContains,
		File:     filePath,
	})

	// ── 3. 导入实体 ──
	for _, imp := range f.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		impName := impPath
		if imp.Name != nil {
			impName = imp.Name.Name + " " + impPath
		}

		impID := EntityID(EntityImport, filePath, impPath)
		b.graph.AddEntity(&Entity{
			ID:       impID,
			Kind:     EntityImport,
			Name:     impName,
			FQN:      impPath,
			FilePath: filePath,
			Line:     fset.Position(imp.Pos()).Line,
		})

		// 文件导入包
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: impID,
			Kind:     RelImports,
			File:     filePath,
			Line:     fset.Position(imp.Pos()).Line,
		})
	}

	// ── 4. AST 遍历提取实体 ──
	// 收集文档注释（预留语义分析使用）
	for _, cg := range f.Comments {
		if cg != nil {
			pos := fset.Position(cg.Pos())
			_ = pos
			// 将注释组关联到后续节点
			for _, comment := range cg.List {
				text := strings.TrimSpace(comment.Text)
				if strings.HasPrefix(text, "//") {
					text = strings.TrimSpace(text[2:])
				} else if strings.HasPrefix(text, "/*") {
					text = strings.TrimSpace(text[2 : len(text)-2])
				}
				// 简化：只保留紧挨文档注释
			}
		}
	}

	// 使用 go/doc 提取文档注释（更可靠）
	docPkg, _ := doc.NewFromFiles(fset, []*ast.File{f}, pkgPath,
		doc.AllDecls|doc.PreserveAST)
	_ = docPkg

	// 遍历顶层声明
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			b.parseGenDecl(d, fset, filePath, pkgPath, fileID)
		case *ast.FuncDecl:
			b.parseFuncDecl(d, fset, filePath, pkgPath, fileID)
		}
	}

	return fileID, nil
}

// ── 通用声明解析（import/type/var/const） ──────────

func (b *GoBuilder) parseGenDecl(d *ast.GenDecl, fset *token.FileSet, filePath, pkgPath, fileID string) {
	switch d.Tok {
	case token.IMPORT:
		// 已在前面处理了 import
	case token.TYPE:
		for _, spec := range d.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			b.parseTypeSpec(ts, fset, filePath, pkgPath, fileID)
		}
	case token.VAR:
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name == "_" {
					continue
				}
				sig := fmt.Sprintf("var %s %s", name.Name, b.exprString(vs.Type))
				if i < len(vs.Values) {
					sig += " = ..."
				}
				varID := EntityID(EntityVariable, pkgPath, name.Name)
				b.graph.AddEntity(&Entity{
					ID:        varID,
					Kind:      EntityVariable,
					Name:      name.Name,
					FQN:       b.fqn(pkgPath + "." + name.Name),
					FilePath:  filePath,
					Line:      fset.Position(name.Pos()).Line,
					EndLine:   fset.Position(name.End()).Line,
					Signature: sig,
				})
				// 文件包含变量
				b.graph.AddRelation(&Relation{
					SourceID: fileID,
					TargetID: varID,
					Kind:     RelContains,
					File:     filePath,
					Line:     fset.Position(name.Pos()).Line,
				})
			}
		}
	case token.CONST:
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if name.Name == "_" {
					continue
				}
				sig := fmt.Sprintf("const %s", name.Name)
				constID := EntityID(EntityConstant, pkgPath, name.Name)
				b.graph.AddEntity(&Entity{
					ID:        constID,
					Kind:      EntityConstant,
					Name:      name.Name,
					FQN:       b.fqn(pkgPath + "." + name.Name),
					FilePath:  filePath,
					Line:      fset.Position(name.Pos()).Line,
					EndLine:   fset.Position(name.End()).Line,
					Signature: sig,
				})
				b.graph.AddRelation(&Relation{
					SourceID: fileID,
					TargetID: constID,
					Kind:     RelContains,
					File:     filePath,
					Line:     fset.Position(name.Pos()).Line,
				})
			}
		}
	}
}

// ── 类型声明解析 ──────────────────────────────────────

func (b *GoBuilder) parseTypeSpec(ts *ast.TypeSpec, fset *token.FileSet, filePath, pkgPath, fileID string) {
	typeName := ts.Name.Name
	sig := b.exprString(ts.Type)

	var kind EntityKind

	switch t := ts.Type.(type) {
	case *ast.StructType:
		kind = EntityStruct
		typeID := EntityID(EntityStruct, pkgPath, typeName)
		b.graph.AddEntity(&Entity{
			ID:        typeID,
			Kind:      kind,
			Name:      typeName,
			FQN:       b.fqn(pkgPath + "." + typeName),
			FilePath:  filePath,
			Line:      fset.Position(ts.Pos()).Line,
			EndLine:   fset.Position(ts.End()).Line,
			Signature: sig,
		})

		// 解析结构体字段
		for _, field := range t.Fields.List {
			for _, fname := range field.Names {
				fieldID := EntityID(EntityField, pkgPath, typeName+"."+fname.Name)
				b.graph.AddEntity(&Entity{
					ID:        fieldID,
					Kind:      EntityField,
					Name:      fname.Name,
					FQN:       b.fqn(pkgPath + "." + typeName + "." + fname.Name),
					FilePath:  filePath,
					Line:      fset.Position(fname.Pos()).Line,
					Signature: fmt.Sprintf("%s %s", fname.Name, b.exprString(field.Type)),
					Metadata: map[string]string{
						"receiver": typeName,
					},
				})
				b.graph.AddRelation(&Relation{
					SourceID: typeID,
					TargetID: fieldID,
					Kind:     RelContains,
					File:     filePath,
					Line:     fset.Position(fname.Pos()).Line,
				})
			}
		}

		// 嵌入类型
		for _, field := range t.Fields.List {
			if len(field.Names) == 0 {
				// 嵌入字段
				embType := b.exprString(field.Type)
				_ = embType
			}
		}

	case *ast.InterfaceType:
		kind = EntityInterface
		typeID := EntityID(EntityInterface, pkgPath, typeName)
		b.graph.AddEntity(&Entity{
			ID:        typeID,
			Kind:      kind,
			Name:      typeName,
			FQN:       b.fqn(pkgPath + "." + typeName),
			FilePath:  filePath,
			Line:      fset.Position(ts.Pos()).Line,
			EndLine:   fset.Position(ts.End()).Line,
			Signature: sig,
		})

		// 解析接口方法
		for _, method := range t.Methods.List {
			for _, mname := range method.Names {
				methodID := EntityID(EntityMethod, pkgPath, typeName+"."+mname.Name)
				b.graph.AddEntity(&Entity{
					ID:        methodID,
					Kind:      EntityMethod,
					Name:      mname.Name,
					FQN:       b.fqn(pkgPath + "." + typeName + "." + mname.Name),
					FilePath:  filePath,
					Line:      fset.Position(mname.Pos()).Line,
					Signature: fmt.Sprintf("(%s) %s", typeName, b.exprString(method.Type)),
					Metadata: map[string]string{
						"receiver": typeName,
					},
				})
				b.graph.AddRelation(&Relation{
					SourceID: typeID,
					TargetID: methodID,
					Kind:     RelContains,
					File:     filePath,
					Line:     fset.Position(mname.Pos()).Line,
				})
			}
		}

	default:
		// 普通类型别名/定义
		kind = EntityType
		typeID := EntityID(EntityType, pkgPath, typeName)
		b.graph.AddEntity(&Entity{
			ID:        typeID,
			Kind:      kind,
			Name:      typeName,
			FQN:       b.fqn(pkgPath + "." + typeName),
			FilePath:  filePath,
			Line:      fset.Position(ts.Pos()).Line,
			EndLine:   fset.Position(ts.End()).Line,
			Signature: sig,
		})
	}

	// 文件包含类型
	typeID := EntityID(kind, pkgPath, typeName)
	b.graph.AddRelation(&Relation{
		SourceID: fileID,
		TargetID: typeID,
		Kind:     RelContains,
		File:     filePath,
		Line:     fset.Position(ts.Pos()).Line,
	})
}

// ── 函数声明解析 ──────────────────────────────────────

func (b *GoBuilder) parseFuncDecl(d *ast.FuncDecl, fset *token.FileSet, filePath, pkgPath, fileID string) {
	funcName := d.Name.Name
	sig := b.formatFuncSignature(d)

	if d.Recv != nil && len(d.Recv.List) > 0 {
		// 方法：解析接收者类型
		recvType := b.exprString(d.Recv.List[0].Type)
		// 去除指针 *
		recvType = strings.TrimPrefix(recvType, "*")

		methodID := EntityID(EntityMethod, pkgPath, recvType+"."+funcName)
		b.graph.AddEntity(&Entity{
			ID:        methodID,
			Kind:      EntityMethod,
			Name:      funcName,
			FQN:       b.fqn(pkgPath + "." + recvType + "." + funcName),
			FilePath:  filePath,
			Line:      fset.Position(d.Pos()).Line,
			EndLine:   fset.Position(d.End()).Line,
			Signature: sig,
			Metadata: map[string]string{
				"receiver": recvType,
			},
		})

		// 接收者类型定义此方法
		typeID := EntityID(EntityStruct, pkgPath, recvType)
		if b.graph.GetEntity(typeID) == nil {
			typeID = EntityID(EntityInterface, pkgPath, recvType)
		}
		if b.graph.GetEntity(typeID) == nil {
			typeID = EntityID(EntityType, pkgPath, recvType)
		}
		b.graph.AddRelation(&Relation{
			SourceID: typeID,
			TargetID: methodID,
			Kind:     RelDefines,
			File:     filePath,
			Line:     fset.Position(d.Pos()).Line,
		})

		// 方法也包含在文件中
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: methodID,
			Kind:     RelContains,
			File:     filePath,
			Line:     fset.Position(d.Pos()).Line,
		})

		// 解析函数体内的调用
		if d.Body != nil {
			b.parseCallExprs(d.Body, fset, filePath, methodID)
		}
	} else {
		// 普通函数
		funcID := EntityID(EntityFunction, pkgPath, funcName)
		b.graph.AddEntity(&Entity{
			ID:        funcID,
			Kind:      EntityFunction,
			Name:      funcName,
			FQN:       b.fqn(pkgPath + "." + funcName),
			FilePath:  filePath,
			Line:      fset.Position(d.Pos()).Line,
			EndLine:   fset.Position(d.End()).Line,
			Signature: sig,
		})

		// 文件包含函数
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: funcID,
			Kind:     RelContains,
			File:     filePath,
			Line:     fset.Position(d.Pos()).Line,
		})

		// 解析函数体内的调用
		if d.Body != nil {
			b.parseCallExprs(d.Body, fset, filePath, funcID)
		}
	}
}

// ── 调用表达式解析 ────────────────────────────────────

// parseCallExprs 遍历 AST 提取函数调用关系。
func (b *GoBuilder) parseCallExprs(body *ast.BlockStmt, fset *token.FileSet, filePath, callerID string) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		_ = call

		// 提取被调用函数名
		var calleeName string
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			calleeName = fun.Name
		case *ast.SelectorExpr:
			// pkg.Func 或 obj.Method
			if x, ok := fun.X.(*ast.Ident); ok {
				calleeName = x.Name + "." + fun.Sel.Name
			} else {
				calleeName = fun.Sel.Name
			}
		default:
			return true
		}

		callLine := fset.Position(call.Pos()).Line

		// 被调用的函数/方法实体（可能跨包，暂存为引用标识）
		callSiteID := fmt.Sprintf("%s_call_%s_%d", callerID, calleeName, callLine)

		callSiteEntity := &Entity{
			ID:       callSiteID,
			Kind:     EntityCallSite,
			Name:     calleeName,
			FilePath: filePath,
			Line:     callLine,
			Metadata: map[string]string{
				"caller":  callerID,
				"callee":  calleeName,
			},
		}
		b.graph.AddEntity(callSiteEntity)

		b.graph.AddRelation(&Relation{
			SourceID: callerID,
			TargetID: callSiteID,
			Kind:     RelCalls,
			File:     filePath,
			Line:     callLine,
			Metadata: map[string]string{
				"callee": calleeName,
			},
		})

		return true
	})
}

// ── 包路径推断 ────────────────────────────────────────

// packagePathFromFile 从文件路径推断包路径（相对于模块根）。
func (b *GoBuilder) packagePathFromFile(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." {
		return b.ModuleName
	}
	// 去除工作区前缀
	dir = strings.TrimPrefix(dir, b.root)
	dir = strings.TrimPrefix(dir, string(filepath.Separator))
	// 转换为 Go 包路径
	pkgPath := strings.ReplaceAll(dir, string(filepath.Separator), "/")
	if b.ModuleName != "" {
		return b.ModuleName + "/" + pkgPath
	}
	return pkgPath
}

// fqn 生成完全限定名。
func (b *GoBuilder) fqn(local string) string {
	if b.ModuleName != "" && !strings.HasPrefix(local, b.ModuleName) {
		return b.ModuleName + "." + local
	}
	return local
}

// exprString 将 AST 表达式转为字符串（用于签名）。
func (b *GoBuilder) exprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + b.exprString(e.X)
	case *ast.SelectorExpr:
		return b.exprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + b.exprString(e.Elt)
		}
		return "[" + b.exprString(e.Len) + "]" + b.exprString(e.Elt)
	case *ast.MapType:
		return "map[" + b.exprString(e.Key) + "]" + b.exprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.Ellipsis:
		return "..." + b.exprString(e.Elt)
	case *ast.BasicLit:
		return e.Value
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// formatFuncSignature 格式化函数/方法签名。
func (b *GoBuilder) formatFuncSignature(d *ast.FuncDecl) string {
	var parts []string
	if d.Recv != nil && len(d.Recv.List) > 0 {
		recv := b.exprString(d.Recv.List[0].Type)
		parts = append(parts, "func ("+recv+")")
	} else {
		parts = append(parts, "func")
	}
	parts = append(parts, d.Name.Name)

	// 参数
	parts = append(parts, "(")
	if d.Type.Params != nil {
		paramStrs := make([]string, 0, len(d.Type.Params.List))
		for _, param := range d.Type.Params.List {
			typeStr := b.exprString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					paramStrs = append(paramStrs, name.Name+" "+typeStr)
				}
			} else {
				paramStrs = append(paramStrs, typeStr)
			}
		}
		parts = append(parts, strings.Join(paramStrs, ", "))
	}
	parts = append(parts, ")")

	// 返回值
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		parts = append(parts, " ")
		if len(d.Type.Results.List) == 1 && d.Type.Results.List[0].Names == nil {
			parts = append(parts, b.exprString(d.Type.Results.List[0].Type))
		} else {
			parts = append(parts, "(")
			retStrs := make([]string, 0, len(d.Type.Results.List))
			for _, ret := range d.Type.Results.List {
				typeStr := b.exprString(ret.Type)
				if len(ret.Names) > 0 {
					for _, name := range ret.Names {
						retStrs = append(retStrs, name.Name+" "+typeStr)
					}
				} else {
					retStrs = append(retStrs, typeStr)
				}
			}
			parts = append(parts, strings.Join(retStrs, ", "))
			parts = append(parts, ")")
		}
	}

	return strings.Join(parts, "")
}

// ── 批量解析 ──────────────────────────────────────────

// ParseDir 递归解析目录下所有 Go 文件。
// 返回解析的文件数和错误列表（非致命错误累积，不中断）。
func (b *GoBuilder) ParseDir(dirPath string) (int, []error) {
	var errors []error
	count := 0

	absDir := filepath.Join(b.root, dirPath)
	err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// 跳过隐藏目录和常见非源码目录
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "vendor" ||
				info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		relPath, _ := filepath.Rel(b.root, path)
		relPath = filepath.ToSlash(relPath)

		if _, err := b.ParseFile(relPath); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", relPath, err))
			return nil // 继续解析其他文件
		}
		count++
		return nil
	})

	if err != nil {
		errors = append(errors, err)
	}
	return count, errors
}
