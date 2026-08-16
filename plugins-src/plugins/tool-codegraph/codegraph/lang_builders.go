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
	// 原有语言
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
	// 新增 16 种语言
	case ".html", ".htm":
		return parseHTMLFile
	case ".css", ".scss", ".sass", ".less", ".styl":
		return parseCSSFile
	case ".json":
		return parseJSONFile
	case ".yaml", ".yml":
		return parseYAMLFile
	case ".md", ".markdown", ".mdown", ".mdwn":
		return parseMarkdownFile
	case "Dockerfile", ".dockerfile":
		return parseDockerfileFile
	case "Makefile", "makefile", ".mk":
		return parseMakefileFile
	case ".toml":
		return parseTOMLFile
	case ".tf", ".hcl":
		return parseHCLFile
	case ".ps1", ".psm1", ".psd1":
		return parsePSFile
	case ".zig":
		return parseZigFile
	case ".scala":
		return parseScalaFile
	case ".ex", ".exs":
		return parseElixirFile
	case ".r", ".R":
		return parseRFile
	case ".graphql", ".gql":
		return parseGraphQLFile
	case "CMakeLists.txt", ".cmake":
		return parseCMakeFile
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
	case "html":
		return []string{".html", ".htm"}
	case "css":
		return []string{".css", ".scss", ".sass", ".less", ".styl"}
	case "json":
		return []string{".json"}
	case "yaml":
		return []string{".yaml", ".yml"}
	case "markdown":
		return []string{".md", ".markdown", ".mdown", ".mdwn"}
	case "dockerfile":
		return []string{"Dockerfile", ".dockerfile"}
	case "makefile":
		return []string{"Makefile", "makefile", ".mk"}
	case "toml":
		return []string{".toml"}
	case "hcl":
		return []string{".tf", ".hcl"}
	case "powershell":
		return []string{".ps1", ".psm1", ".psd1"}
	case "zig":
		return []string{".zig"}
	case "scala":
		return []string{".scala"}
	case "elixir":
		return []string{".ex", ".exs"}
	case "r":
		return []string{".r", ".R"}
	case "graphql":
		return []string{".graphql", ".gql"}
	case "cmake":
		return []string{"CMakeLists.txt", ".cmake"}
	}
	return nil
}

// isLangSupportedExtra 检查文件是否由 lang_builders 支持。
// 需要同时检查文件扩展名和完整文件名（如 Dockerfile/Makefile）。
func isLangSupportedExtra(name string) bool {
	// 先检查扩展名
	ext := strings.ToLower(filepath.Ext(name))
	if getLangParser(ext) != nil {
		return true
	}
	// 再检查完整文件名
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "dockerfile", "makefile", "cmakelists.txt":
		return true
	}
	return false
}

// getLangParserByFile 根据文件名获取解析器（同时检查扩展名和完整文件名）。
func getLangParserByFile(name string) func(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(name))
	if parser := getLangParser(ext); parser != nil {
		return parser
	}
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "dockerfile":
		return parseDockerfileFile
	case "makefile":
		return parseMakefileFile
	case "cmakelists.txt":
		return parseCMakeFile
	}
	return nil
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
func (b *LangBuilder) Graph() *Graph     { return b.graph }
func (b *LangBuilder) Reset()            { b.graph = NewGraph() }
func (b *LangBuilder) SetGraph(g *Graph) { b.graph = g }

// ParseFile 根据文件扩展名路由到具体语言解析器。
func (b *LangBuilder) ParseFile(filePath string) (string, error) {
	absPath := filepath.Join(b.root, filePath)
	source, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", filePath, err)
	}

	baseName := filepath.Base(filePath)
	parser := getLangParserByFile(baseName)
	if parser == nil {
		return "", fmt.Errorf("不支持的语言: %s", filepath.Ext(filePath))
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
	lines    []string
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
	reRustFn     = regexp.MustCompile(`^\s*(?:pub\s+)?(?:unsafe\s+)?fn\s+(` + reIdent + `)\s*\(`)
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

	// 使用 brace 栈跟踪 impl 块归属
	type implFrame struct {
		implType   string // 当前 impl 的类型
		braceDepth int    // 进入时的花括号深度
	}
	implStack := []implFrame{}
	braceDepth := 0

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		ln := lineNo + 1

		// 计算本行的 brace 变化（忽略字符串中的括号）
		openBrace := 0
		closeBrace := 0
		for _, ch := range trimmed {
			if ch == '{' {
				openBrace++
			}
			if ch == '}' {
				closeBrace++
			}
		}

		// 检查是否在 impl 块开始时记录
		if reRustImpl.MatchString(trimmed) && braceDepth == 0 {
			m := reRustImpl.FindStringSubmatch(trimmed)
			implType := m[1]
			if len(m) > 2 && m[2] != "" {
				implType = m[2] // impl Trait for Type → 取 Type
			}
			implStack = append(implStack, implFrame{
				implType:   implType,
				braceDepth: braceDepth,
			})
		}

		// 应用 brace 变化
		braceDepth += openBrace - closeBrace

		// 关闭 impl 块
		for len(implStack) > 0 && implStack[len(implStack)-1].braceDepth > braceDepth {
			implStack = implStack[:len(implStack)-1]
		}

		// 获取当前 impl 类型（检查 impl 栈帧的 braceDepth 是否在当前深度范围内）
		currentImpl := ""
		for i := len(implStack) - 1; i >= 0; i-- {
			f := implStack[i]
			if f.braceDepth <= braceDepth-openBrace {
				currentImpl = f.implType
				break
			}
		}
		// 如果当前在 impl 块内但 braceDepth 没更新到，用栈顶
		if currentImpl == "" && len(implStack) > 0 {
			lastFrame := implStack[len(implStack)-1]
			if braceDepth > lastFrame.braceDepth {
				currentImpl = lastFrame.implType
			}
		}

		switch {
		case reRustFn.MatchString(trimmed):
			m := reRustFn.FindStringSubmatch(trimmed)
			name := m[1]
			sig := fmt.Sprintf("fn %s(...)", name)
			if currentImpl != "" {
				id := ext.addEntity(EntityMethod, currentImpl+"."+name, sig, ln)
				if clsID := ext.addEntity(EntityStruct, currentImpl, "", 0); clsID != "" {
					b.graph.AddRelation(&Relation{
						SourceID: clsID, TargetID: id, Kind: RelDefines, File: filePath, Line: ln,
					})
				}
			} else {
				id := ext.addEntity(EntityFunction, name, sig, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reRustStruct.MatchString(trimmed):
			m := reRustStruct.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], fmt.Sprintf("struct %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRustEnum.MatchString(trimmed):
			m := reRustEnum.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], fmt.Sprintf("enum %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRustTrait.MatchString(trimmed):
			m := reRustTrait.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityInterface, m[1], fmt.Sprintf("trait %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRustUse.MatchString(trimmed):
			m := reRustUse.FindStringSubmatch(trimmed)
			impID := ext.addEntity(EntityImport, m[1], m[1], ln)
			ext.addRel(fileID, impID, RelImports, ln)

		case reRustConst.MatchString(trimmed):
			m := reRustConst.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityConstant, m[1], fmt.Sprintf("const %s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Java ──────────────────────────────────────────────

var (
	reJavaClass  = regexp.MustCompile(`^\s*(?:public\s+|private\s+|protected\s+)?(?:abstract\s+|final\s+|static\s+)*(?:class|interface|enum|@interface)\s+(` + reTypeIdent + `)`)
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
	reCFunc    = regexp.MustCompile(`^\s*(?:static\s+|inline\s+|extern\s+)?(?:const\s+)?(` + reIdent + `\s*\*?)\s+(` + reIdent + `)\s*\(`)
	reCStruct  = regexp.MustCompile(`^\s*(?:typedef\s+)?struct\s+(` + reIdent + `)\s*\{`)
	reCInclude = regexp.MustCompile(`^\s*#\s*include\s+[<"](.+)[>"]`)
	reCTypedef = regexp.MustCompile(`^\s*typedef\s+(?:const\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*;`)
	reCMacro   = regexp.MustCompile(`^\s*#\s*define\s+(` + reIdent + `)\s+`)
	reCEnum    = regexp.MustCompile(`^\s*(?:typedef\s+)?enum\s+(` + reIdent + `)\s*\{`)
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
	reCsClass  = regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|abstract|sealed|partial)\s+(?:class|struct|interface|record|enum)\s+(` + reTypeIdent + `)`)
	reCsMethod = regexp.MustCompile(`^\s*(?:public|private|protected|internal|static|virtual|override|abstract|async|unsafe)\s+(?:` + reIdent + `\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\(`)
	reCsUsing  = regexp.MustCompile(`^\s*using\s+(?:static\s+)?(\S+)\s*;`)
	reCsProp   = regexp.MustCompile(`^\s*(?:public|private|protected|internal)\s+(?:static\s+)?(` + reIdent + `)\s+(` + reIdent + `)\s*\{`)
	reCsNs     = regexp.MustCompile(`^\s*namespace\s+(\S+)`)
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
	reRubyClass   = regexp.MustCompile(`^\s*(?:class\s+)(` + reTypeIdent + `(?:\s*<\s*` + reTypeIdent + `)?)`)
	reRubyModule  = regexp.MustCompile(`^\s*(?:module\s+)(` + reTypeIdent + `)`)
	reRubyDef     = regexp.MustCompile(`^\s*(?:public|private|protected|static)?\s*def\s+(?:self\.)?(` + reIdent + `)`)
	reRubyRequire = regexp.MustCompile(`^\s*(?:require|require_relative|load|autoload)\s+['"](\S+)['"]`)
	reRubyAttr    = regexp.MustCompile(`^\s*(?:attr_accessor|attr_reader|attr_writer)\s+(?::(` + reIdent + `))`)
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
	rePhpClass     = regexp.MustCompile(`^\s*(?:abstract\s+|final\s+)?class\s+(` + reTypeIdent + `)`)
	rePhpInterface = regexp.MustCompile(`^\s*interface\s+(` + reTypeIdent + `)`)
	rePhpTrait     = regexp.MustCompile(`^\s*trait\s+(` + reTypeIdent + `)`)
	rePhpFunc      = regexp.MustCompile(`^\s*(?:public|private|protected|static|abstract|final)?\s*(?:function)\s+(` + reIdent + `)\s*\(`)
	rePhpUse       = regexp.MustCompile(`^\s*use\s+(.+);$`)
	rePhpConst     = regexp.MustCompile(`^\s*(?:const|define)\s*\(?\s*['"]?(` + reIdent + `)`)
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
	reSwiftClass  = regexp.MustCompile(`^\s*(?:public|private|internal|open|final)?\s*(?:class|struct|enum|protocol|extension)\s+(` + reTypeIdent + `)`)
	reSwiftFunc   = regexp.MustCompile(`^\s*(?:public|private|internal|fileprivate|static|class|override)?\s*(?:func)\s+(` + reIdent + `)\s*\(`)
	reSwiftImport = regexp.MustCompile(`^\s*import\s+(?:` + reIdent + `\s+)?(\S+)`)
	reSwiftVar    = regexp.MustCompile(`^\s*(?:public|private|internal|static|let|var)\s+(?:let|var)\s+(` + reIdent + `)\s*:`)
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
	reBashVar    = regexp.MustCompile(`^\s*(?:export\s+)?(` + reIdent + `)=[\"'` + "`" + `]?`)
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
	reSqlCreateTable = regexp.MustCompile(`(?i)create\s+(?:temp|temporary|global\s+temporary)?\s*table\s+(?:if\s+not\s+exists\s+)?(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateFunc  = regexp.MustCompile(`(?i)create\s+(?:or\s+replace\s+)?(?:function|procedure|trigger|view)\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateIndex = regexp.MustCompile(`(?i)create\s+(?:unique\s+)?index\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
	reSqlCreateType  = regexp.MustCompile(`(?i)create\s+(?:or\s+replace\s+)?type\s+(?:` + reIdent + `\.)?(` + reIdent + `)`)
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
		if len(name) > 0 && name[0] == '(' {
			return ""
		}
		return name
	}
	return ""
}

// ═══════════════════════════════════════════════════════
// 新增 16 种语言解析器
// ═══════════════════════════════════════════════════════

// ── HTML ──────────────────────────────────────────────

var (
	reHTMLTag       = regexp.MustCompile(`(?i)<(/?)([a-z]\w*)(?:\s+[^>]*)?>`)
	reHTMLClass     = regexp.MustCompile(`(?i)class\s*=\s*["']([^"']+)["']`)
	reHTMLID        = regexp.MustCompile(`(?i)id\s*=\s*["']([^"']+)["']`)
	reHTMLComponent = regexp.MustCompile(`(?i)<([A-Z]\w*)(?:\s+[^>]*)?>`)
)

func parseHTMLFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	sourceStr := string(source)
	tags := make(map[string]bool)
	classes := make(map[string]bool)
	ids := make(map[string]bool)
	components := make(map[string]bool)

	for _, m := range reHTMLTag.FindAllStringSubmatch(sourceStr, -1) {
		if m[1] != "/" { // 非闭合标签
			tags[m[2]] = true
		}
	}
	for _, m := range reHTMLComponent.FindAllStringSubmatch(sourceStr, -1) {
		components[m[1]] = true
	}
	for _, m := range reHTMLClass.FindAllStringSubmatch(sourceStr, -1) {
		for _, cls := range strings.Fields(m[1]) {
			classes[cls] = true
		}
	}
	for _, m := range reHTMLID.FindAllStringSubmatch(sourceStr, -1) {
		ids[m[1]] = true
	}

	for tag := range tags {
		id := ext.addEntity(EntityType, "html."+tag, fmt.Sprintf("<%s>", tag), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for cls := range classes {
		id := ext.addEntity(EntityType, "."+cls, fmt.Sprintf(".%s", cls), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for idName := range ids {
		id := ext.addEntity(EntityStruct, "#"+idName, fmt.Sprintf("#%s", idName), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for comp := range components {
		id := ext.addEntity(EntityStruct, comp, fmt.Sprintf("<%s />", comp), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}

	// 提取 script/style 中的实体
	if idx := strings.Index(sourceStr, "<style"); idx >= 0 {
		end := strings.Index(sourceStr[idx:], "</style>")
		if end > 0 {
			_ = end // CSS 提取由 CSS 解析器处理
		}
	}
	if idx := strings.Index(sourceStr, "<script"); idx >= 0 {
		start := strings.Index(sourceStr[idx:], ">")
		end := strings.Index(sourceStr[idx:], "</script>")
		if start > 0 && end > start {
			jsCode := sourceStr[idx+start+1 : idx+end]
			_ = jsCode // JS 提取由 JSBuilder 处理
		}
	}

	return fileID, nil
}

// ── CSS/SCSS ─────────────────────────────────────────

var (
	reCSSClass     = regexp.MustCompile(`\.([a-zA-Z_]\w*)\s*\{`)
	reCSSID        = regexp.MustCompile(`#([a-zA-Z_]\w*)\s*\{`)
	reCSSKeyframes = regexp.MustCompile(`@keyframes\s+([a-zA-Z_]\w*)`)
	reCSSMedia     = regexp.MustCompile(`@media\s+(.+?)\{`)
	reCSSImport    = regexp.MustCompile(`@import\s+['"](.+?)['"]`)
	reCSSMixin     = regexp.MustCompile(`@mixin\s+([a-zA-Z_]\w*)`)
	reCSSInclude   = regexp.MustCompile(`@include\s+([a-zA-Z_]\w*)`)
	reCSSFunction  = regexp.MustCompile(`@function\s+([a-zA-Z_]\w*)`)
)

func parseCSSFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	sourceStr := string(source)
	for _, m := range reCSSClass.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityType, "."+m[1], fmt.Sprintf(".%s {}", m[1]), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for _, m := range reCSSID.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityStruct, "#"+m[1], fmt.Sprintf("#%s {}", m[1]), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for _, m := range reCSSKeyframes.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityFunction, m[1], fmt.Sprintf("@keyframes %s", m[1]), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for _, m := range reCSSMixin.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityFunction, m[1], fmt.Sprintf("@mixin %s", m[1]), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for _, m := range reCSSFunction.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityFunction, m[1], fmt.Sprintf("@function %s", m[1]), 1)
		ext.addRel(fileID, id, RelContains, 1)
	}
	for _, m := range reCSSImport.FindAllStringSubmatch(sourceStr, -1) {
		id := ext.addEntity(EntityImport, m[1], m[1], 1)
		ext.addRel(fileID, id, RelImports, 1)
	}
	return fileID, nil
}

// ── JSON ──────────────────────────────────────────────

var (
	reJSONKey  = regexp.MustCompile(`^\s*"([a-zA-Z_]\w*)"\s*:`)
	reJSONType = regexp.MustCompile(`^\s*"([a-zA-Z_]\w*)"\s*:\s*(\{|\[|"|\d+|true|false|null)`)
)

func parseJSONFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if m := reJSONKey.FindStringSubmatch(trimmed); m != nil {
			id := ext.addEntity(EntityVariable, m[1], trimmed, lineNo+1)
			ext.addRel(fileID, id, RelContains, lineNo+1)
		}
	}
	return fileID, nil
}

// ── YAML ──────────────────────────────────────────────

var (
	reYAMLKey = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*:`)
)

func parseYAMLFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	seen := make(map[string]bool)
	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "  ") {
			continue
		}
		if m := reYAMLKey.FindStringSubmatch(trimmed); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			id := ext.addEntity(EntityVariable, m[1], trimmed, lineNo+1)
			ext.addRel(fileID, id, RelContains, lineNo+1)
		}
	}
	return fileID, nil
}

// ── Markdown ──────────────────────────────────────────

var (
	reMDHeading   = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	reMDCodeFence = regexp.MustCompile("^`{3,}(\\w*)")
	reMDLink      = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func parseMarkdownFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		ln := lineNo + 1

		if m := reMDHeading.FindStringSubmatch(trimmed); m != nil {
			level := len(m[1])
			title := m[2]
			kind := EntityFunction
			if level >= 2 {
				kind = EntityMethod
			}
			id := ext.addEntity(kind, title, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
		if m := reMDCodeFence.FindStringSubmatch(trimmed); m != nil && m[1] != "" {
			id := ext.addEntity(EntityType, "code:"+m[1], fmt.Sprintf("```%s", m[1]), ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Dockerfile ────────────────────────────────────────

var (
	reDockerFrom   = regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)(?:\s+AS\s+(\S+))?`)
	reDockerRun    = regexp.MustCompile(`(?i)^\s*RUN\s+(.+)`)
	reDockerCopy   = regexp.MustCompile(`(?i)^\s*COPY\s+(.+?)\s+(.+)`)
	reDockerExpose = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(\d+)`)
	reDockerArg    = regexp.MustCompile(`(?i)^\s*ARG\s+(\S+)`)
	reDockerEnv    = regexp.MustCompile(`(?i)^\s*ENV\s+(\S+)(?:=(.+))?`)
	reDockerLabel  = regexp.MustCompile(`(?i)^\s*LABEL\s+(\S+)=`)
)

func parseDockerfileFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reDockerFrom.MatchString(trimmed):
			m := reDockerFrom.FindStringSubmatch(trimmed)
			imageName := m[1]
			impID := ext.addEntity(EntityImport, "docker:"+imageName, trimmed, ln)
			ext.addRel(fileID, impID, RelImports, ln)
			if len(m) > 2 && m[2] != "" {
				stageID := ext.addEntity(EntityStruct, "stage:"+m[2], fmt.Sprintf("FROM %s AS %s", imageName, m[2]), ln)
				ext.addRel(fileID, stageID, RelContains, ln)
			}

		case reDockerArg.MatchString(trimmed):
			m := reDockerArg.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reDockerEnv.MatchString(trimmed):
			m := reDockerEnv.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reDockerLabel.MatchString(trimmed):
			m := reDockerLabel.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityConstant, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Makefile ──────────────────────────────────────────

var (
	reMakeTarget  = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*:`)
	reMakeVar     = regexp.MustCompile(`^([A-Z_]\w*)\s*[\+\?]?=\s*(.*)`)
	reMakeInclude = regexp.MustCompile(`^include\s+(\S+)`)
)

func parseMakefileFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reMakeTarget.MatchString(trimmed):
			m := reMakeTarget.FindStringSubmatch(trimmed)
			if !strings.HasPrefix(m[1], ".") { // 忽略 .PHONY 等
				id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reMakeVar.MatchString(trimmed):
			m := reMakeVar.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reMakeInclude.MatchString(trimmed):
			m := reMakeInclude.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── TOML ──────────────────────────────────────────────

var (
	reTOMLTable = regexp.MustCompile(`^\[([^\]]+)\]`)
	reTOMLArray = regexp.MustCompile(`^\[\[([^\]]+)\]\]`)
	reTOMLKey   = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*=`)
)

func parseTOMLFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reTOMLTable.MatchString(trimmed):
			m := reTOMLTable.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reTOMLKey.MatchString(trimmed):
			m := reTOMLKey.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── HCL / Terraform ───────────────────────────────────

var (
	reHCLResource  = regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"`)
	reHCLData      = regexp.MustCompile(`^\s*data\s+"([^"]+)"\s+"([^"]+)"`)
	reHCLVariable  = regexp.MustCompile(`^\s*variable\s+"([^"]+)"`)
	reHCLOutput    = regexp.MustCompile(`^\s*output\s+"([^"]+)"`)
	reHCLModule    = regexp.MustCompile(`^\s*module\s+"([^"]+)"`)
	reHCLLocals    = regexp.MustCompile(`^\s*locals\s*\{`)
	reHCLProvider  = regexp.MustCompile(`^\s*provider\s+"([^"]+)"`)
	reHCLTerraform = regexp.MustCompile(`^\s*terraform\s*\{`)
)

func parseHCLFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reHCLResource.MatchString(trimmed):
			m := reHCLResource.FindStringSubmatch(trimmed)
			name := m[1] + "." + m[2]
			id := ext.addEntity(EntityStruct, name, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reHCLData.MatchString(trimmed):
			m := reHCLData.FindStringSubmatch(trimmed)
			name := "data." + m[1] + "." + m[2]
			id := ext.addEntity(EntityStruct, name, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reHCLVariable.MatchString(trimmed):
			m := reHCLVariable.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, "var."+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reHCLOutput.MatchString(trimmed):
			m := reHCLOutput.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, "output."+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reHCLModule.MatchString(trimmed):
			m := reHCLModule.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, "module."+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)

		case reHCLProvider.MatchString(trimmed):
			m := reHCLProvider.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, "provider."+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── PowerShell ────────────────────────────────────────

var (
	rePSFunction = regexp.MustCompile(`(?i)^\s*function\s+([a-zA-Z_]\w*)`)
	rePSVariable = regexp.MustCompile(`^\s*\$\{?([a-zA-Z_]\w*)\}?\s*=`)
	rePSModule   = regexp.MustCompile(`(?i)^\s*import-module\s+(\S+)`)
	rePSCmdlet   = regexp.MustCompile(`(?i)^\s*(?:begin|process|end)\s*\{`)
	rePSClass    = regexp.MustCompile(`(?i)^\s*class\s+([a-zA-Z_]\w*)`)
)

func parsePSFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case rePSFunction.MatchString(trimmed):
			m := rePSFunction.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case rePSClass.MatchString(trimmed):
			m := rePSClass.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case rePSVariable.MatchString(trimmed):
			m := rePSVariable.FindStringSubmatch(trimmed)
			if !strings.HasPrefix(m[1], "?") {
				id := ext.addEntity(EntityVariable, "$"+m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case rePSModule.MatchString(trimmed):
			m := rePSModule.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── Zig ───────────────────────────────────────────────

var (
	reZigFn     = regexp.MustCompile(`^\s*(?:pub\s+)?fn\s+([a-zA-Z_]\w*)\s*\(`)
	reZigStruct = regexp.MustCompile(`^\s*(?:pub\s+)?(?:const\s+)?([A-Z]\w*)\s*=\s*(?:struct|union|enum)\s*\{`)
	reZigConst  = regexp.MustCompile(`^\s*(?:pub\s+)?const\s+([a-zA-Z_]\w*)\s*=`)
	reZigVar    = regexp.MustCompile(`^\s*(?:pub\s+)?var\s+([a-zA-Z_]\w*)\s*:`)
	reZigTest   = regexp.MustCompile(`^\s*test\s+"(.+?)"`)
)

func parseZigFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reZigFn.MatchString(trimmed):
			m := reZigFn.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reZigStruct.MatchString(trimmed):
			m := reZigStruct.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reZigConst.MatchString(trimmed):
			m := reZigConst.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityConstant, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reZigVar.MatchString(trimmed):
			m := reZigVar.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reZigTest.MatchString(trimmed):
			m := reZigTest.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, "test."+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Scala ─────────────────────────────────────────────

var (
	reScalaClass   = regexp.MustCompile(`^\s*(?:abstract\s+|sealed\s+|case\s+|final\s+)?(?:class|object|trait|case class|enum)\s+([A-Z]\w*)`)
	reScalaDef     = regexp.MustCompile(`^\s*(?:private|protected|override|abstract|final|implicit|lazy)?\s*def\s+([a-zA-Z_]\w*)\s*\(`)
	reScalaVal     = regexp.MustCompile(`^\s*(?:private|protected|override|implicit|lazy)?\s*(?:val|var|lazy val)\s+([a-zA-Z_]\w*)\s*(?::|=)`)
	reScalaImport  = regexp.MustCompile(`^\s*import\s+(\S+)`)
	reScalaPackage = regexp.MustCompile(`^\s*package\s+(\S+)`)
	reScalaType    = regexp.MustCompile(`^\s*(?:type)\s+([a-zA-Z_]\w*)\s*=`)
	reScalaEnum    = regexp.MustCompile(`^\s*enum\s+([A-Z]\w*)`)
)

func parseScalaFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := strings.Split(string(source), "\n")
	currentClass := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reScalaPackage.MatchString(trimmed):
			m := reScalaPackage.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityPackage, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reScalaClass.MatchString(trimmed) || reScalaEnum.MatchString(trimmed):
			m := reScalaClass.FindStringSubmatch(trimmed)
			if m == nil {
				m = reScalaEnum.FindStringSubmatch(trimmed)
			}
			currentClass = m[1]
			id := ext.addEntity(EntityStruct, currentClass, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reScalaDef.MatchString(trimmed):
			m := reScalaDef.FindStringSubmatch(trimmed)
			name := m[1]
			if currentClass != "" {
				id := ext.addEntity(EntityMethod, currentClass+"."+name, trimmed, ln)
				clsID := EntityID(EntityStruct, "", currentClass)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reScalaVal.MatchString(trimmed):
			m := reScalaVal.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reScalaImport.MatchString(trimmed):
			m := reScalaImport.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)

		case reScalaType.MatchString(trimmed):
			m := reScalaType.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── Elixir ────────────────────────────────────────────

var (
	reElixirModule   = regexp.MustCompile(`^\s*defmodule\s+([A-Z]\w*(?:\.[A-Z]\w+)*)\s+do`)
	reElixirDef      = regexp.MustCompile(`^\s*def\s+([a-zA-Z_]\w*)\s*\(`)
	reElixirDefP     = regexp.MustCompile(`^\s*defp\s+([a-zA-Z_]\w*)\s*\(`)
	reElixirDefMacro = regexp.MustCompile(`^\s*defmacro\s+([a-zA-Z_]\w*)\s*\(`)
	reElixirImport   = regexp.MustCompile(`^\s*(?:import|alias|use|require)\s+([A-Z]\w*(?:\.[A-Z]\w+)*)`)
	reElixirStruct   = regexp.MustCompile(`^\s*defstruct\s+(?:[a-z_]\w*:\s*)?(\w+)`)
)

func parseElixirFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	lines := strings.Split(string(source), "\n")
	currentModule := ""

	for lineNo, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reElixirModule.MatchString(trimmed):
			m := reElixirModule.FindStringSubmatch(trimmed)
			currentModule = m[1]
			id := ext.addEntity(EntityStruct, currentModule, trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reElixirDef.MatchString(trimmed):
			m := reElixirDef.FindStringSubmatch(trimmed)
			name := m[1]
			if currentModule != "" {
				id := ext.addEntity(EntityMethod, currentModule+"."+name, trimmed, ln)
				clsID := EntityID(EntityStruct, "", currentModule)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reElixirDefP.MatchString(trimmed):
			m := reElixirDefP.FindStringSubmatch(trimmed)
			name := m[1]
			if currentModule != "" {
				id := ext.addEntity(EntityMethod, currentModule+"."+name, trimmed, ln)
				clsID := EntityID(EntityStruct, "", currentModule)
				ext.addRel(clsID, id, RelDefines, ln)
			} else {
				id := ext.addEntity(EntityFunction, name, trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reElixirImport.MatchString(trimmed):
			m := reElixirImport.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)
		}
	}
	return fileID, nil
}

// ── R ─────────────────────────────────────────────────

var (
	reRFunc    = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*<-\s*function\s*\(`)
	reRAssign  = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*<-\s+`)
	reRSource  = regexp.MustCompile(`^\s*source\(['"](.+?)['"]\)`)
	reRLibrary = regexp.MustCompile(`^\s*(?:library|require)\(['"]?(.+?)['"]?\)`)
)

func parseRFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reRFunc.MatchString(trimmed):
			m := reRFunc.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reRLibrary.MatchString(trimmed):
			m := reRLibrary.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)

		case reRAssign.MatchString(trimmed):
			m := reRAssign.FindStringSubmatch(trimmed)
			// 排除函数赋值（已处理）
			if !strings.Contains(trimmed, "function") && !strings.HasPrefix(trimmed, ".") {
				id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}
		}
	}
	return fileID, nil
}

// ── GraphQL ───────────────────────────────────────────

var (
	reGQLType      = regexp.MustCompile(`(?i)^\s*(?:type|input|interface|union|enum)\s+([A-Z]\w*)`)
	reGQLExtend    = regexp.MustCompile(`(?i)^\s*extend\s+(?:type|input|interface)\s+([A-Z]\w*)`)
	reGQLQuery     = regexp.MustCompile(`(?i)^\s*(?:type\s+)?query\s*\{`)
	reGQLMutation  = regexp.MustCompile(`(?i)^\s*(?:type\s+)?mutation\s*\{`)
	reGQLSubscript = regexp.MustCompile(`(?i)^\s*(?:type\s+)?subscription\s*\{`)
	reGQLSchema    = regexp.MustCompile(`(?i)^\s*schema\s*\{`)
	reGQLDirective = regexp.MustCompile(`(?i)^\s*directive\s+@([a-zA-Z_]\w*)`)
	reGQLScalar    = regexp.MustCompile(`(?i)^\s*scalar\s+([A-Z]\w*)`)
)

func parseGraphQLFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, `"""`) {
			continue
		}
		ln := lineNo + 1

		switch {
		case reGQLType.MatchString(trimmed):
			m := reGQLType.FindStringSubmatch(trimmed)
			if strings.Contains(trimmed, "interface") {
				id := ext.addEntity(EntityInterface, m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			} else {
				id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
				ext.addRel(fileID, id, RelContains, ln)
			}

		case reGQLScalar.MatchString(trimmed):
			m := reGQLScalar.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityType, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reGQLDirective.MatchString(trimmed):
			m := reGQLDirective.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, "@"+m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)
		}
	}
	return fileID, nil
}

// ── CMake ─────────────────────────────────────────────

var (
	reCMakeFunction = regexp.MustCompile(`(?i)^\s*function\s*\(\s*([a-zA-Z_]\w*)`)
	reCMakeMacro    = regexp.MustCompile(`(?i)^\s*macro\s*\(\s*([a-zA-Z_]\w*)`)
	reCMakeSet      = regexp.MustCompile(`(?i)^\s*set\s*\(\s*([a-zA-Z_]\w*)`)
	reCMakeOption   = regexp.MustCompile(`(?i)^\s*option\s*\(\s*([a-zA-Z_]\w*)`)
	reCMakeProject  = regexp.MustCompile(`(?i)^\s*project\s*\(\s*(\S+)`)
	reCMakeAddSub   = regexp.MustCompile(`(?i)^\s*add_subdirectory\s*\(\s*(\S+)`)
	reCMakeFindPkg  = regexp.MustCompile(`(?i)^\s*find_package\s*\(\s*(\S+)`)
	reCMakeTarget   = regexp.MustCompile(`(?i)^\s*(?:add_executable|add_library|add_test)\s*\(\s*(\S+)`)
)

func parseCMakeFile(b *LangBuilder, filePath string, source []byte) (string, error) {
	ext := newExtract(string(source), filePath, "", b.graph)
	fileName := filepath.Base(filePath)
	fileID := ext.addEntity(EntityFile, fileName, "", 1)
	ext.fileID = fileID

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ln := lineNo + 1

		switch {
		case reCMakeFunction.MatchString(trimmed):
			m := reCMakeFunction.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeMacro.MatchString(trimmed):
			m := reCMakeMacro.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityFunction, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeSet.MatchString(trimmed):
			m := reCMakeSet.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeOption.MatchString(trimmed):
			m := reCMakeOption.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityVariable, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeProject.MatchString(trimmed):
			m := reCMakeProject.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityPackage, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeTarget.MatchString(trimmed):
			m := reCMakeTarget.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityStruct, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelContains, ln)

		case reCMakeFindPkg.MatchString(trimmed):
			m := reCMakeFindPkg.FindStringSubmatch(trimmed)
			id := ext.addEntity(EntityImport, m[1], trimmed, ln)
			ext.addRel(fileID, id, RelImports, ln)
		}
	}
	return fileID, nil
}
