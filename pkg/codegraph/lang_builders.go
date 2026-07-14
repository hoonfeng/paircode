package codegraph

// lang_builders.go — 多语言正则解析器。
// 覆盖 Tree-sitter 支持的主流编程语言，提取函数/类/方法/变量/导入等实体。
// 每种语言使用正则匹配提取顶层结构（函数、类、变量、导入），
// 辅以缩进/花括号分析确定方法归属。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ── 语言解析器注册表 ──────────────────────────────────

// getLangParser 根据文件扩展名返回对应的语言解析函数。
func getLangParser(ext string) func(b *LangBuilder, filePath string, source []byte) (string, error) {
	switch ext {
	case ".rs":
		return parseRustFile
	case ".java":
		return parseJavaFile
	case ".c", ".h":
		return parseCFile
	case ".cpp", ".hpp", ".cc", ".cxx", ".hh", ".hxx":
		return parseCppFile
	case ".cs":
		return parseCsharpFile
	case ".rb":
		return parseRubyFile
	case ".php":
		return parsePhpFile
	case ".swift":
		return parseSwiftFile
	case ".kt", ".kts":
		return parseKotlinFile
	case ".dart":
		return parseDartFile
	case ".lua":
		return parseLuaFile
	case ".sh", ".bash", ".zsh":
		return parseBashFile
	case ".sql":
		return parseSqlFile
	case ".vue":
		return parseVueFile
	// Go/JS/Python 已由各自的 builder 处理
	default:
		return nil
	}
}

// getLangExtensions 返回语言支持的扩展名列表。
func getLangExtensions(lang string) []string {
	switch lang {
	case "rust":
		return []string{".rs"}
	case "java":
		return []string{".java"}
	case "c":
		return []string{".c", ".h"}
	case "cpp":
		return []string{".cpp", ".hpp", ".cc", ".cxx", ".hh", ".hxx"}
	case "csharp":
		return []string{".cs"}
	case "ruby":
		return []string{".rb"}
	case "php":
		return []string{".php"}
	case "swift":
		return []string{".swift"}
	case "kotlin":
		return []string{".kt", ".kts"}
	case "dart":
		return []string{".dart"}
	case "lua":
		return []string{".lua"}
	case "bash":
		return []string{".sh", ".bash", ".zsh"}
	case "sql":
		return []string{".sql"}
	case "vue":
		return []string{".vue"}
	}
	return nil
}

// isLangSupportedExtra 检查扩展名是否由 lang_builders 支持。
func isLangSupportedExtra(ext string) bool {
	return getLangParser(ext) != nil
}

// ── 通用正则 ──────────────────────────────────────────

var (
	// 通用标识符（用于变量名/函数名）
	reIdent = `[a-zA-Z_]\w*`
	// 通用类型名（首字母大写标识符）
	reTypeIdent = `[A-Z]\w*`

	// 注释行（各语言通用）
	reLineComment = regexp.MustCompile(`^\s*(//|#|--|%|;)`)
)

// ── LangBuilder ───────────────────────────────────────

// LangBuilder 通用多语言解析器。
// 所有语言共用此结构，通过 ParseFile 方法路由到具体语言解析函数。
type LangBuilder struct {
	ModuleName string
	root       string
	graph      *Graph
}

func NewLangBuilder(root, moduleName string) *LangBuilder {
	return &LangBuilder{ModuleName: moduleName, root: root, graph: NewGraph()}
}
func (b *LangBuilder) Graph() *Graph    { return b.graph }
func (b *LangBuilder) Reset()           { b.graph = NewGraph() }
func (b *LangBuilder) SetGraph(g *Graph) { b.graph = g }

// ParseFile 根据文件扩展名路由到具体语言解析器。
func (b *LangBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	parser := getLangParser(ext)
	if parser == nil {
		return "", fmt.Errorf("不支持的语言: %s", ext)
	}

	return parser(b, filePath, source)
}

// ── 辅助函数 ──────────────────────────────────────────

// addEntity 快捷添加实体的辅助方法。
func (b *LangBuilder) addEntity(kind EntityKind, name, fqn, filePath, sig string, line int) string {
	id := EntityID(kind, "", name)
	b.graph.AddEntity(&Entity{
		ID: id, Kind: kind, Name: name,
		FQN: fqn, FilePath: filePath,
		Line: line, Signature: sig,
	})
	return id
}

// addRel 快捷添加关系的辅助方法。
func (b *LangBuilder) addRel(sourceID, targetID string, kind RelationKind, filePath string, line int) {
	b.graph.AddRelation(&Relation{
		SourceID: sourceID, TargetID: targetID,
		Kind: kind, File: filePath, Line: line,
	})
}

// ParseDir 批量解析目录下所有支持的文件。
func (b *LangBuilder) ParseDir(dirPath string) (int, []error) {
	var errors []error
	count := 0
	absDir := filepath.Join(b.root, dirPath)
	filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" ||
				name == "vendor" || name == "__pycache__" ||
				name == "venv" || name == ".venv" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !isLangSupportedExtra(ext) {
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
	return count, errors
}

// ──────────────────────────────────────────────────────
// 各语言解析器
// ──────────────────────────────────────────────────────

// entityExtract 共享的结构体，将行级实体提取逻辑封装。
type entityExtract struct {
	lines   []string
	filePath string
	fileID   string
	graph    *Graph
}

func newExtract(source, filePath, fileID string, graph *Graph) *entityExtract {
	return &entityExtract{
		lines:    strings.Split(source, "\n"),
		filePath: filePath,
		fileID:   fileID,
		graph:    graph,
	}
}

func (e *entityExtract) addEntity(kind EntityKind, name, sig string, lineNo int) string {
	id := EntityID(kind, "", name)
	e.graph.AddEntity(&Entity{
		ID: id, Kind: kind, Name: name,
		FQN: name, FilePath: e.filePath,
		Line: lineNo, Signature: sig,
	})
	return id
}

func (e *entityExtract) addRel(src, dst string, kind RelationKind, lineNo int) {
	e.graph.AddRelation(&Relation{
		SourceID: src, TargetID: dst,
		Kind: kind, File: e.filePath, Line: lineNo,
	})
}

// ── Rust ──────────────────────────────────────────────

var (
	reRustFn    = regexp.MustCompile(`^\s*(?:pub\s+)?(?:unsafe\s+)?fn\s+(` + reIdent + `)\s*\(`)
	reRustStruct = regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+(` + reTypeIdent + `)`)
	reRustEnum   = regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+(` + reTypeIdent + `)`)
	reRustTrait  = regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+(` + reTypeIdent + `)`)
	reRustImpl   = regexp.MustCompile(`^\s*(?:pub\s+)?impl\s+(?:<[^>]+>\s*)?(` + reIdent + `(?:<[^>]+>)?)\s*(?:for\s+(` + reIdent + `))?`)
	reRustUse    = regexp.MustCompile(`^\s*(?:pub\s+)?use\s+(\S+);`)
	reRustConst  = regexp.MustCompile(`^\s*(?:pub\s+)?(?:const|static)\s+(` + reIdent + `)\s*:`)
)

func parseRustFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	implType := ""
	braceDepth := 0

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		ln := lineNo + 1

		// 跟踪花括号深度（用于 impl 方法归属）
		braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

		switch {
		case reRustFn.MatchString(trimmed):
			m := reRustFn.FindStringSubmatch(trimmed)
			name := m[1]
			if implType != "" {
				// 方法
				id := ext.addEntity(EntityMethod, implType+"."+name, fmt.Sprintf("%s.%s(...)", implType, name), ln)
				if clsID := ext.addEntity(EntityStruct, implType, "", 0); clsID != "" {
					ext.addRel(clsID, id, RelDefines, ln)
				}
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("fn %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reRustStruct.MatchString(trimmed):
			m := reRustStruct.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], fmt.Sprintf("struct %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)
			implType = "" // struct 定义不改变 impl 上下文

		case reRustEnum.MatchString(trimmed):
			m := reRustEnum.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], fmt.Sprintf("enum %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRustTrait.MatchString(trimmed):
			m := reRustTrait.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityInterface, m[1], fmt.Sprintf("trait %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRustImpl.MatchString(trimmed):
			m := reRustImpl.FindStringSubmatch(trimmed)
			implType = m[1]

		case reRustUse.MatchString(trimmed):
			m := reRustUse.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)

		case reRustConst.MatchString(trimmed):
			m := reRustConst.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityConstant, m[1], fmt.Sprintf("const %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)
		}

		// impl 块结束后重置
		if braceDepth == 0 && implType != "" {
			// 检查是否已离开 impl 块
		}
	}
	return fileID, nil
}

// ── Java ──────────────────────────────────────────────

var (
	reJavaClass = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:abstract\s+|final\s+|static\s+)*(?:class|interface|enum|@interface)\s+(` + reTypeIdent + `)`)
	reJavaMethod = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:static\s+|abstract\s+|final\s+|synchronized\s+)*(?:<[^>]+>\s*)?(` + reTypeIdent + `|void|int|long|double|float|boolean|char|byte|short|String|` + reTypeIdent + `\[])\s+(` + reIdent + `)\s*\(`)
	reJavaFn     = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:static\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\(`)
	reJavaImport = regexp.MustCompile(`^\s*import\s+(?:static\s+)?(\S+);`)
	reJavaField  = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:static\s+|final\s+)*(` + reIdent + `)\s+(` + reIdent + `)\s*(?:=|;)`)
)

func parseJavaFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reJavaClass.MatchString(trimmed):
			m := reJavaClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			kind := EntityStruct
			if strings.Contains(trimmed, "interface") {
				kind = EntityInterface
			} else if strings.Contains(trimmed, "enum") {
				kind = EntityType
			}
			id := ext.addEntity(kind, currentClass, fmt.Sprintf("class %s", currentClass), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reJavaMethod.MatchString(trimmed):
			m := reJavaMethod.FindStringSubmatch(trimmed)
			retType, methodName := m[1], m[2]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+methodName,
					fmt.Sprintf("%s %s.%s(...)", retType, currentClass, methodName), ln)
				if clsID := EntityID(EntityStruct, "", currentClass); b.graph.GetEntity(clsID) != nil {
					ext.addRel(clsID, id, RelDefines, ln)
				}
			} else {
				id := ext.addEntity(EntityFunction, methodName,
					fmt.Sprintf("%s %s(...)", retType, methodName), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reJavaImport.MatchString(trimmed):
			m := reJavaImport.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── C ─────────────────────────────────────────────────

var (
	reCFunc     = regexp.MustCompile(`^\s*(?:static\s+|inline\s+|extern\s+)?(?:const\s+)?(` + reIdent + `\s*\*?)\s+(` + reIdent + `)\s*\(`)
	reCStruct   = regexp.MustCompile(`^\s*(?:typedef\s+)?struct\s+(` + reIdent + `)\s*\{`)
	reCInclude  = regexp.MustCompile(`^\s*#\s*include\s+[<"](.+)[>"]`)
	reCTypedef  = regexp.MustCompile(`^\s*typedef\s+(?:const\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*;`)
	reCMacro    = regexp.MustCompile(`^\s*#\s*define\s+(` + reIdent + `)\s+`)
	reCEnum     = regexp.MustCompile(`^\s*(?:typedef\s+)?enum\s+(` + reIdent + `)\s*\{`)
)

func parseCFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	return parseCFamily(b, filePath, source)
}

func parseCppFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	return parseCFamily(b, filePath, source)
}

func parseCFamily(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentStruct := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reCStruct.MatchString(trimmed):
			m := reCStruct.FindStringSubmatch(trimmed)
			currentStruct = m[1]
			id := ext.addEntity(EntityStruct, currentStruct, fmt.Sprintf("struct %s", currentStruct), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCEnum.MatchString(trimmed):
			m := reCEnum.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], fmt.Sprintf("enum %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCFunc.MatchString(trimmed):
			m := reCFunc.FindStringSubmatch(trimmed)
			retType, funcName := m[1], m[2]
			_ = retType
			id := ext.addEntity(EntityFunction, funcName, fmt.Sprintf("%s %s(...)", retType, funcName), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCInclude.MatchString(trimmed):
			m := reCInclude.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)

		case reCTypedef.MatchString(trimmed):
			m := reCTypedef.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[2], fmt.Sprintf("typedef %s %s", m[1], m[2]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMacro.MatchString(trimmed):
			m := reCMacro.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityConstant, m[1], fmt.Sprintf("#define %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── C# ────────────────────────────────────────────────

var (
	reCsClass   = regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|abstract|sealed|partial)\s+(?:class|struct|interface|record|enum)\s+(` + reTypeIdent + `)`)
	reCsMethod  = regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|virtual|override|abstract|async|unsafe)\s+(?:` + reIdent + `\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\(`)
	reCsUsing   = regexp.MustCompile(`^\s*using\s+(?:static\s+)?(\S+)\s*;`)
	reCsProp    = regexp.MustCompile(`^\s*(?:public|private|protected|internal)\s+(?:static\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\{`)
	reCsNs      = regexp.MustCompile(`^\s*namespace\s+(\S+)`)
)

func parseCsharpFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reCsClass.MatchString(trimmed):
			m := reCsClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCsMethod.MatchString(trimmed):
			m := reCsMethod.FindStringSubmatch(trimmed)
			retType, methodName := m[1], m[2]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+methodName,
					fmt.Sprintf("%s %s.%s(...)", retType, currentClass, methodName), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, methodName,
					fmt.Sprintf("%s %s(...)", retType, methodName), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reCsUsing.MatchString(trimmed):
			m := reCsUsing.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── Ruby ──────────────────────────────────────────────

var (
	reRubyClass = regexp.MustCompile(`^\s*(?:class\s+)(` + reTypeIdent + `(?:\s*<\s*` + reTypeIdent + `)?)`)
	reRubyModule = regexp.MustCompile(`^\s*(?:module\s+)(` + reTypeIdent + `)`)
	reRubyDef   = regexp.MustCompile(`^\s*(?:public|private|protected|static)?\s*def\s+(?:self\.)?(` + reIdent + `)`)
	reRubyRequire = regexp.MustCompile(`^\s*(?:require|require_relative|load|autoload)\s+['"](\S+)['"]`)
	reRubyAttr  = regexp.MustCompile(`^\s*(?:attr_accessor|attr_reader|attr_writer)\s+(?::(` + reIdent + `))`)
)

func parseRubyFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reRubyClass.MatchString(trimmed):
			m := reRubyClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, fmt.Sprintf("class %s", currentClass), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRubyModule.MatchString(trimmed):
			m := reRubyModule.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityType, currentClass, fmt.Sprintf("module %s", currentClass), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRubyDef.MatchString(trimmed):
			m := reRubyDef.FindStringSubmatch(trimmed)
			name := m[1]
		if currentClass != "" {
			id := ext.addEntity(EntityMethod, currentClass+"."+name,
				fmt.Sprintf("%s.%s(...)", currentClass, name), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("def %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reRubyRequire.MatchString(trimmed):
			m := reRubyRequire.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)
		}
	}
	_ = ext
	return fileID, nil
}

// ── PHP ───────────────────────────────────────────────

var (
	rePhpClass  = regexp.MustCompile(`^\s*(?:abstract\s+|final\s+)?class\s+(` + reTypeIdent + `)`)
	rePhpInterface = regexp.MustCompile(`^\s*interface\s+(` + reTypeIdent + `)`)
	rePhpTrait  = regexp.MustCompile(`^\s*trait\s+(` + reTypeIdent + `)`)
	rePhpFunc   = regexp.MustCompile(`^\s*(?:public|private|protected|static|abstract|final)?\s*(?:function)\s+(` + reIdent + `)\s*\(`)
	rePhpUse    = regexp.MustCompile(`^\s*use\s+(.+);$`)
	rePhpConst  = regexp.MustCompile(`^\s*(?:const|define)\s*\(?\s*['"]?(` + reIdent + `)`)
)

func parsePhpFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	// 需要处理 <?php 标签
	sourceStr := string(source)
	if idx := strings.Index(sourceStr, "<?php"); idx >= 0 {
		sourceStr = sourceStr[idx+5:]
	}

	lines := strings.Split(sourceStr, "\n")
	ext.lines = lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case rePhpClass.MatchString(trimmed):
			m := rePhpClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, fmt.Sprintf("class %s", currentClass), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case rePhpInterface.MatchString(trimmed):
			m := rePhpInterface.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityInterface, m[1], fmt.Sprintf("interface %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case rePhpTrait.MatchString(trimmed):
			m := rePhpTrait.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], fmt.Sprintf("trait %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case rePhpFunc.MatchString(trimmed):
			m := rePhpFunc.FindStringSubmatch(trimmed)
			name := m[1]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+name,
					fmt.Sprintf("%s::%s(...)", currentClass, name), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("function %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}
		}
	}
	return fileID, nil
}

// ── Swift ─────────────────────────────────────────────

var (
	reSwiftClass = regexp.MustCompile(`^\s*(?:public|private|internal|open|final)?\s*(?:class|struct|enum|protocol|extension)\s+(` + reTypeIdent + `)`)
	reSwiftFunc  = regexp.MustCompile(`^\s*(?:public|private|internal|fileprivate|static|class|override)?\s*(?:func)\s+(` + reIdent + `)\s*\(`)
	reSwiftImport = regexp.MustCompile(`^\s*import\s+(?:` + reIdent + `\s+)?(\S+)`)
	reSwiftVar   = regexp.MustCompile(`^\s*(?:public|private|internal|static|let|var)\s+(?:let|var)\s+(` + reIdent + `)\s*:`)
)

func parseSwiftFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reSwiftClass.MatchString(trimmed):
			m := reSwiftClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			kind := EntityStruct
			if strings.Contains(trimmed, "protocol") {
				kind = EntityInterface
			} else if strings.Contains(trimmed, "enum") {
				kind = EntityType
			}
			id := ext.addEntity(kind, currentClass, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reSwiftFunc.MatchString(trimmed):
			m := reSwiftFunc.FindStringSubmatch(trimmed)
			name := m[1]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+name,
					fmt.Sprintf("%s.%s(...)", currentClass, name), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("func %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reSwiftImport.MatchString(trimmed):
			m := reSwiftImport.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── Kotlin ────────────────────────────────────────────

var (
	reKtClass  = regexp.MustCompile(`^\s*(?:public|private|internal|abstract|data|sealed|open)?\s*(?:class|interface|enum class|object|annotation class)\s+(` + reTypeIdent + `)`)
	reKtFun    = regexp.MustCompile(`^\s*(?:public|private|internal|protected|override|abstract|open|suspend|inline|tailrec|operator|infix)?\s*(?:fun)\s+(` + reIdent + `)\s*\(`)
	reKtImport = regexp.MustCompile(`^\s*import\s+(\S+)`)
	reKtVal    = regexp.MustCompile(`^\s*(?:public|private|internal|protected)?\s*(?:val|var|const\s+val|const\s+var)\s+(` + reIdent + `)\s*(?::|=)`)
)

func parseKotlinFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reKtClass.MatchString(trimmed):
			m := reKtClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reKtFun.MatchString(trimmed):
			m := reKtFun.FindStringSubmatch(trimmed)
			name := m[1]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+name,
					fmt.Sprintf("%s.%s(...)", currentClass, name), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("fun %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reKtImport.MatchString(trimmed):
			m := reKtImport.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)

		case reKtVal.MatchString(trimmed):
			m := reKtVal.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Dart ──────────────────────────────────────────────

var (
	reDartClass  = regexp.MustCompile(`^\s*(?:abstract\s+)?(?:class|mixin|extension)\s+(` + reTypeIdent + `)`)
	reDartFunc   = regexp.MustCompile(`^\s*(?:Future<` + reIdent + `>\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\(`)
	reDartImport = regexp.MustCompile(`^\s*import\s+(?:'([^']+)'|"([^"]+)"|package:(\S+)|dart:(\S+))`)
	reDartVar    = regexp.MustCompile(`^\s*(?:final|const|var|late\s+final|late\s+var|static\s+const)\s+(` + reIdent + `)\s*=`)
)

func parseDartFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reDartClass.MatchString(trimmed):
			m := reDartClass.FindStringSubmatch(trimmed)
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reDartFunc.MatchString(trimmed):
			m := reDartFunc.FindStringSubmatch(trimmed)
			typeName, funcName := m[1], m[2]
			_ = typeName
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+funcName,
					fmt.Sprintf("%s.%s(...)", currentClass, funcName), ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, funcName,
					fmt.Sprintf("%s %s(...)", typeName, funcName), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reDartImport.MatchString(trimmed):
			m := reDartImport.FindStringSubmatch(trimmed)
			var impPath string
			for _, p := range m[1:] {
				if p != "" {
					impPath = p
					break
				}
			}
			if impPath != "" {
				impID := ext.addEntity(EntityImport, impPath, impPath, ln)
				ext.addRel(fileID, impID, RelImports, ln)
			}

		case reDartVar.MatchString(trimmed):
			m := reDartVar.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Lua ───────────────────────────────────────────────

var (
	reLuaFn      = regexp.MustCompile(`^\s*(?:local\s+)?function\s+(?:` + reIdent + `\.)?(` + reIdent + `)\s*\(`)
	reLuaClass   = regexp.MustCompile(`^\s*(?:` + reTypeIdent + `)\s*=\s*\{`)
	reLuaRequire = regexp.MustCompile(`^\s*(?:require|dofile)\s+['"](.+)['"]`)
	reLuaLocal   = regexp.MustCompile(`^\s*local\s+(` + reIdent + `)\s*=`)
)

func parseLuaFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	currentModule := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reLuaFn.MatchString(trimmed):
			m := reLuaFn.FindStringSubmatch(trimmed)
			name := m[1]
			if strings.Contains(trimmed, ".") || currentModule != "" {
				prefix := currentModule
				if prefix == "" {
					prefix = strings.SplitN(trimmed, ".", 2)[0]
				}
				id := ext.addEntity(EntityMethod, prefix+"."+name,
					fmt.Sprintf("%s.%s(...)", prefix, name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, fmt.Sprintf("function %s(...)", name), ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reLuaRequire.MatchString(trimmed):
			m := reLuaRequire.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)

		case reLuaLocal.MatchString(trimmed):
			m := reLuaLocal.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reLuaClass.MatchString(trimmed) && currentModule == "":
			// 可能有类定义
		}
	}
	return fileID, nil
}

// ── Bash ──────────────────────────────────────────────

var (
	reBashFn     = regexp.MustCompile(`^\s*(?:function\s+)?(` + reIdent + `)\s*\(\s*\)`)
	reBashVar    = regexp.MustCompile(`^\s*(?:export\s+)?(` + reIdent + `)=[\"'`+"`"+`]?`)
	reBashSource = regexp.MustCompile(`^\s*(?:source|\.)\s+(\S+)`)
)

func parseBashFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reBashFn.MatchString(trimmed):
			m := reBashFn.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], fmt.Sprintf("function %s()", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reBashVar.MatchString(trimmed):
			m := reBashVar.FindStringSubmatch(trimmed)
			// 避免匹配函数名
			if !strings.HasSuffix(strings.TrimSpace(trimmed), ")") {
				id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reBashSource.MatchString(trimmed):
			m := reBashSource.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── SQL ───────────────────────────────────────────────

var (
	reSqlCreateTable  = regexp.MustCompile(`(?i)create\s+(?:temp|temporary|global\s+temporary)?\s*table\s+(?:if\s+not\s+exists\s+)?(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateFunc   = regexp.MustCompile(`(?i)create\s+(?:or\s+replace\s+)?(?:function|procedure|trigger|view)\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateIndex  = regexp.MustCompile(`(?i)create\s+(?:unique\s+)?index\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateType   = regexp.MustCompile(`(?i)create\s+(?:or\s+replace\s+)?type\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
)

func parseSqlFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := ext.lines
	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reSqlCreateFunc.MatchString(trimmed):
			m := reSqlCreateFunc.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reSqlCreateTable.MatchString(trimmed):
			m := reSqlCreateTable.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reSqlCreateIndex.MatchString(trimmed):
			// 索引作为类型记录
		}
	}
	return fileID, nil
}

// ── Vue ───────────────────────────────────────────────

// Vue 文件包含 template/script/style。提取 script 中的 JS 实体。
func parseVueFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	sourceStr := string(source)

	// 提取 <script> 和 <script setup> 中的 JS 代码
	scriptRe := regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`)
	matches := scriptRe.FindStringSubmatch(sourceStr)
	if len(matches) < 2 {
		return "", nil
	}
	jsSource := matches[1]

	ext := newExtract(jsSource, filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	// 使用 JS 相同的正则提取函数/变量等
	for lineNo, line := range strings.Split(jsSource, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		ln := lineNo + 1

		// 提取 export default { methods: { ... } } 等略
		// 提取 export function
		if strings.HasPrefix(trimmed, "export function") || strings.HasPrefix(trimmed, "export default function") {
			name := extractNameAfterKeyword(trimmed, "function")
			if name != "" {
				id := ext.addEntity(EntityFunction, name, trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}
		}
		if strings.HasPrefix(trimmed, "export default") && strings.Contains(trimmed, "{") {
			id := ext.addEntity(EntityStruct, "default_export", trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

func extractNameAfterKeyword(s, keyword string) string {
	idx := strings.Index(s, keyword)
	if idx < 0 {
		return ""
	}
	after := strings.TrimSpace(s[idx+len(keyword):])
	parts := strings.Fields(after)
	if len(parts) > 0 {
		name := parts[0]
		if name[0] == '(' {
			return ""
		}
		return name
	}
	return ""
}
