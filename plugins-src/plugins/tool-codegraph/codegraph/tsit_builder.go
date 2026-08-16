package codegraph

// tsit_builder.go — 基于 Tree-sitter (tsit) 的多语言文件解析器。
// 替换旧的 builder_go.go（仅支持 Go 的 go/parser 实现）。
//
// 使用 tsit.DetectLanguage + Language.ParseFile 解析任意支持的语言，
// 将源码解析为 Tree-sitter 兼容的语法树，再遍历树提取 Entity/Relation。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/plugins-src/plugins/tool-codegraph/codegraph/tsit"
)

// TsitBuilder 基于 tsit 的多语言 AST 解析构建器。
// 支持 Go/JavaScript/TypeScript（Go 使用 go/parser 适配，其他语言后续扩展）。
type TsitBuilder struct {
	ModuleName string // go.mod 中的模块名
	root       string // 工作区根
	graph      *Graph
}

// NewTsitBuilder 创建新的 tsit 构建器。
func NewTsitBuilder(root, moduleName string) *TsitBuilder {
	return &TsitBuilder{
		ModuleName: moduleName,
		root:       root,
		graph:      NewGraph(),
	}
}

// Graph 返回当前构建的图。
func (b *TsitBuilder) Graph() *Graph { return b.graph }

// Reset 重置构建器。
func (b *TsitBuilder) Reset() { b.graph = NewGraph() }

// ParseFile 解析单个源文件，提取实体和关系加入图。
// 根据文件扩展名自动选择语言解析器。
// filePath 为工作区相对路径。
func (b *TsitBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)

	// 读取源码
	source, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	// 检测语言
	lang := tsit.DetectLanguage(filePath)
	if lang == nil {
		return "", fmt.Errorf("不支持的文件类型: %s", filePathExt(filePath))
	}

	// 使用 tsit 解析文件
	tree, err := lang.ParseFile(filePath, source)
	if err != nil {
		return "", fmt.Errorf("tsit 解析 %s 失败: %w", filePath, err)
	}

	rootNode := tree.RootNode()
	if rootNode == nil || rootNode.IsNull() {
		return "", fmt.Errorf("解析结果为空: %s", filePath)
	}

	// 遍历语法树，提取实体和关系
	fileID := b.extractFromTree(tree, rootNode, filePath, source)

	return fileID, nil
}

// filePathExt 返回文件扩展名（小写，含点）。
func filePathExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return ext
}

// ── 树遍历提取 ────────────────────────────────────────

// extractFromTree 遍历 tsit 树，提取实体和关系加入图。
func (b *TsitBuilder) extractFromTree(tree *tsit.Tree, root *tsit.Node, filePath string, source []byte) string {
	pkgPath := b.packagePathFromFile(filePath)

	// 文件实体
	fileName := filepath.Base(filePath)
	fileID := EntityID(EntityFile, "", filePath)
	b.graph.AddEntity(&Entity{
		ID:       fileID,
		Kind:     EntityFile,
		Name:     fileName,
		FQN:      filePath,
		FilePath: filePath,
		Line:     1,
	})

	// 递归遍历所有节点
	b.walkNode(tree, root, source, filePath, fileID, pkgPath)

	return fileID
}

// walkNode 递归遍历 tsit 节点，提取实体。
func (b *TsitBuilder) walkNode(tree *tsit.Tree, node *tsit.Node, source []byte, filePath, fileID, pkgPath string) {
	if node == nil || node.IsNull() {
		return
	}

	typeName := node.NodeType()
	content := node.Content(source)

	// 根据节点类型提取实体
	switch typeName {
	case "source_file":
		// 根节点，遍历子节点
	case "package_clause":
		// 提取包名
		pkgName := extractFirstChildText(node, source)
		pkgID := EntityID(EntityPackage, "", pkgPath)
		b.graph.AddEntity(&Entity{
			ID:       pkgID,
			Kind:     EntityPackage,
			Name:     pkgName,
			FQN:      pkgPath,
			FilePath: filePath,
			Line:     int(node.StartPoint().Row + 1),
		})
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: pkgID,
			Kind:     RelContains,
			File:     filePath,
		})

	case "function_declaration":
		funcName := extractFirstChildText(node, source)
		if funcName == "" {
			funcName = content
		}
		funcID := EntityID(EntityFunction, pkgPath, funcName)
		b.graph.AddEntity(&Entity{
			ID:        funcID,
			Kind:      EntityFunction,
			Name:      funcName,
			FQN:       b.fqn(pkgPath + "." + funcName),
			FilePath:  filePath,
			Line:      int(node.StartPoint().Row + 1),
			EndLine:   int(node.EndPoint().Row + 1),
			Signature: "func " + content,
		})
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: funcID,
			Kind:     RelContains,
			File:     filePath,
		})

	case "method_declaration":
		methodName := extractFirstChildText(node, source)
		if methodName == "" {
			methodName = content
		}
		// 尝试提取接收者
		receiver := extractReceiver(tree, node, source)
		methodID := EntityID(EntityMethod, pkgPath, receiver+"."+methodName)
		b.graph.AddEntity(&Entity{
			ID:        methodID,
			Kind:      EntityMethod,
			Name:      methodName,
			FQN:       b.fqn(pkgPath + "." + receiver + "." + methodName),
			FilePath:  filePath,
			Line:      int(node.StartPoint().Row + 1),
			EndLine:   int(node.EndPoint().Row + 1),
			Signature: "func " + content,
			Metadata: map[string]string{
				"receiver": receiver,
			},
		})
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: methodID,
			Kind:     RelContains,
			File:     filePath,
		})

	case "type_declaration", "type_spec":
		typeNameStr := extractFirstChildText(node, source)
		if typeNameStr == "" {
			typeNameStr = content
		}
		typeID := EntityID(EntityType, pkgPath, typeNameStr)

		// 判断是 struct 还是 interface
		subType := detectSubType(node, source)

		kind := EntityType
		switch subType {
		case "struct_type":
			kind = EntityStruct
		case "interface_type":
			kind = EntityInterface
		}

		b.graph.AddEntity(&Entity{
			ID:        typeID,
			Kind:      kind,
			Name:      typeNameStr,
			FQN:       b.fqn(pkgPath + "." + typeNameStr),
			FilePath:  filePath,
			Line:      int(node.StartPoint().Row + 1),
			EndLine:   int(node.EndPoint().Row + 1),
			Signature: content,
		})
		b.graph.AddRelation(&Relation{
			SourceID: fileID,
			TargetID: typeID,
			Kind:     RelContains,
			File:     filePath,
		})

	case "var_declaration":
		varName := extractFirstChildText(node, source)
		if varName != "" && varName != "_" {
			varID := EntityID(EntityVariable, pkgPath, varName)
			b.graph.AddEntity(&Entity{
				ID:        varID,
				Kind:      EntityVariable,
				Name:      varName,
				FQN:       b.fqn(pkgPath + "." + varName),
				FilePath:  filePath,
				Line:      int(node.StartPoint().Row + 1),
				Signature: content,
			})
			b.graph.AddRelation(&Relation{
				SourceID: fileID,
				TargetID: varID,
				Kind:     RelContains,
				File:     filePath,
			})
		}

	case "const_declaration":
		constName := extractFirstChildText(node, source)
		if constName != "" {
			constID := EntityID(EntityConstant, pkgPath, constName)
			b.graph.AddEntity(&Entity{
				ID:        constID,
				Kind:      EntityConstant,
				Name:      constName,
				FQN:       b.fqn(pkgPath + "." + constName),
				FilePath:  filePath,
				Line:      int(node.StartPoint().Row + 1),
				Signature: content,
			})
			b.graph.AddRelation(&Relation{
				SourceID: fileID,
				TargetID: constID,
				Kind:     RelContains,
				File:     filePath,
			})
		}

	case "comment":
		// 注释作为额外节点，暂不提取
	}

	// 递归遍历子节点
	for i := uint32(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(int(i))
		if child != nil {
			b.walkNode(tree, child, source, filePath, fileID, pkgPath)
		}
	}
}

// ── 辅助 ──────────────────────────────────────────────

// extractFirstChildText 提取节点第一个命名子节点的文本内容。
func extractFirstChildText(node *tsit.Node, source []byte) string {
	if node.NamedChildCount() == 0 {
		return ""
	}
	child := node.NamedChild(0)
	if child == nil {
		return ""
	}
	return child.Content(source)
}

// extractReceiver 从方法声明中提取接收者类型名。
func extractReceiver(tree *tsit.Tree, node *tsit.Node, source []byte) string {
	// 方法声明的第一个子节点通常是接收者
	if node.ChildCount() == 0 {
		return ""
	}
	firstChild := node.Child(0)
	if firstChild == nil {
		return ""
	}
	// 接收者通常是一个参数列表，取其中类型部分
	recvContent := firstChild.Content(source)
	recvContent = strings.TrimSpace(recvContent)
	// 去掉括号和指针
	recvContent = strings.TrimPrefix(recvContent, "(")
	recvContent = strings.TrimSuffix(recvContent, ")")
	recvContent = strings.TrimPrefix(recvContent, "*")
	recvContent = strings.TrimSpace(recvContent)
	// 取最后一个单词（类型名）
	parts := strings.Fields(recvContent)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return recvContent
}

// detectSubType 检测类型声明的子类型（struct/interface/普通）。
func detectSubType(node *tsit.Node, source []byte) string {
	for i := uint32(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(int(i))
		if child == nil {
			continue
		}
		t := child.NodeType()
		if t == "struct_type" || t == "interface_type" {
			return t
		}
		// 递归检查子节点
		if sub := detectSubType(child, source); sub != "" {
			return sub
		}
	}
	return ""
}

// packagePathFromFile 从文件路径推断包路径。
func (b *TsitBuilder) packagePathFromFile(filePath string) string {
	dir := filepath.Dir(filePath)
	dir = strings.ReplaceAll(dir, string(filepath.Separator), "/")
	if dir == "." {
		return b.ModuleName
	}
	if b.ModuleName != "" {
		return b.ModuleName + "/" + dir
	}
	return dir
}

// fqn 生成完全限定名。
func (b *TsitBuilder) fqn(local string) string {
	if b.ModuleName != "" && !strings.HasPrefix(local, b.ModuleName) {
		return b.ModuleName + "." + local
	}
	return local
}

// ── 批量解析 ──────────────────────────────────────────

// ParseDir 递归解析目录下所有支持的文件。
func (b *TsitBuilder) ParseDir(dirPath string) (int, []error) {
	var errors []error
	count := 0

	absDir := filepath.Join(b.root, dirPath)
	err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "vendor" ||
				info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查是否支持的文件类型
		ext := filePathExt(info.Name())
		switch ext {
		case ".go", ".js", ".jsx", ".mjs", ".ts", ".tsx":
			// 支持
		default:
			return nil
		}

		relPath, _ := filepath.Rel(b.root, path)
		relPath = filepath.ToSlash(relPath)

		if _, err := b.ParseFile(relPath); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", relPath, err))
			return nil
		}
		count++
		return nil
	})

	if err != nil {
		errors = append(errors, err)
	}
	return count, errors
}
