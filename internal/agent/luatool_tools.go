package agent

// Lua 自定义工具管理工具 — agent 通过以下工具创建/查看/更新/删除 Lua 工具：
//   - lua_tool_list   列出所有 Lua 工具
//   - lua_tool_create 创建新 Lua 工具
//   - lua_tool_update 更新现有 Lua 工具
//   - lua_tool_delete 删除 Lua 工具

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func registerLuaToolTools(r *Registry, root string) {
	toolsDir := filepath.Join(root, ".pair", "tools")

	// ── lua_tool_list ──
	r.Register(&Tool{
		Name:        "lua_tool_list",
		UsageGuide:  "列出所有 Lua 自定义工具（.pair/tools/ 下的 .lua 脚本）。先查已有哪些自定义工具再决定是否需要新建。",
		Description: "列出所有 Lua 自定义工具（工作区 .pair/tools/ 目录下的 .lua 脚本）。显示名称、描述和参数概要。",
		Parameters:  objSchema(props{}),
		ReadOnly:    true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			entries, err := os.ReadDir(toolsDir)
			if err != nil {
				if os.IsNotExist(err) {
					return "暂无 Lua 自定义工具（.pair/tools/ 目录不存在）。", nil
				}
				return "", fmt.Errorf("读取工具目录失败: %w", err)
			}
			var tools []*Tool
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
					continue
				}
				src, err := os.ReadFile(filepath.Join(toolsDir, e.Name()))
				if err != nil {
					continue
				}
				tool, err := buildLuaTool(string(src), e.Name())
				if err != nil {
					continue
				}
				tools = append(tools, tool)
			}
			if len(tools) == 0 {
				return ".pair/tools/ 目录下没有可用的 Lua 自定义工具。", nil
			}
			sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
			var b strings.Builder
			fmt.Fprintf(&b, "## Lua 自定义工具（共 %d 个）\n\n", len(tools))
			b.WriteString("| 名称 | 描述 | 参数 |\n")
			b.WriteString("|------|------|------|\n")
			for _, t := range tools {
				paramNames := paramKeys(t.Parameters)
				paramStr := "无"
				if len(paramNames) > 0 {
					paramStr = "`" + strings.Join(paramNames, "`, `") + "`"
				}
				fmt.Fprintf(&b, "| `%s` | %s | %s |\n", t.Name, strings.TrimSuffix(t.Description, "（Lua 自定义工具）"), paramStr)
			}
			b.WriteString("\n---\n")
			b.WriteString("[提示] 用 `lua_tool_create` 创建新工具，`lua_tool_update` 更新，`lua_tool_delete` 删除。\n")
			return b.String(), nil
		},
	})

	// ── lua_tool_create ──
	r.Register(&Tool{
		Name: "lua_tool_create",
		UsageGuide: "创建新的 Lua 自定义工具。写入 .pair/tools/{name}.lua，下次消息即热加载生效。用于扩展 agent 能力（自定义脚本）。需审核批准。",
		Description: "创建一个新的 Lua 自定义工具。写入 .pair/tools/{name}.lua，下次发送即热加载生效。" +
			"沙箱仅开 string/table/math 库，无文件/系统访问，单次 10s 超时。脚本 run 函数可通过 agent.run_command({command=..., cwd=...}) 执行 shell 命令。",
		Parameters: objSchema(props{
			"name":        strProp("工具名称（仅英文数字下划线，用作文件名，如 \"word_count\"）"),
			"description": strProp("工具描述，说明功能和用法，如 \"统计文本字数\"。由 agent 推断并填写"),
			"parameters":  strProp("参数 JSON Schema，JSON 对象字符串，如 {\"type\":\"object\",\"properties\":{\"text\":{\"type\":\"string\",\"description\":\"文本\"}},\"required\":[\"text\"]}。由 agent 根据任务需求推断，无参数则省略。"),
			"code":        strProp("run 函数的 Lua 代码（不含 function(args) 外层包装）。通过 args.参数名 访问参数，return 结果字符串。可调用 agent.run_command({command=...}) 执行 shell。"),
		}, "name", "description", "code"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			desc := argStr(args, "description")
			code := argStr(args, "code")
			paramsStr := argStr(args, "parameters")

			if name == "" {
				return "", fmt.Errorf("name 不能为空")
			}
			if desc == "" {
				return "", fmt.Errorf("description 不能为空")
			}
			if code == "" {
				return "", fmt.Errorf("code 不能为空")
			}

			// 解析参数 Schema（JSON → Go map）
			var params map[string]any
			if paramsStr != "" {
				if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
					return "", fmt.Errorf("parameters 格式错误（需 JSON Schema 对象）: %w", err)
				}
			} else {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}

			// Sanitize filename
			fileName := sanitizeLuaName(name)
			if fileName == "" {
				return "", fmt.Errorf("name 经 sanitize 后为空，请使用有效的英文/数字/下划线")
			}
			fileName += ".lua"
			targetPath := filepath.Join(toolsDir, fileName)

			// 生成 Lua 脚本
			script := generateLuaScript(name, desc, params, code)

			// 用 buildLuaTool 验证脚本语法正确
			if _, err := buildLuaTool(script, fileName); err != nil {
				return "", fmt.Errorf("生成的 Lua 脚本验证失败（请检查 code 语法）: %w", err)
			}

			// 确保目录存在并写入
			if err := os.MkdirAll(toolsDir, 0o755); err != nil {
				return "", fmt.Errorf("创建工具目录失败: %w", err)
			}
			if _, err := os.Stat(targetPath); err == nil {
				return "", fmt.Errorf("工具 %q 已存在（.pair/tools/%s），如需修改请用 `lua_tool_update`", name, fileName)
			}
			if err := os.WriteFile(targetPath, []byte(script), 0o644); err != nil {
				return "", fmt.Errorf("写入工具文件失败: %w", err)
			}

			return fmt.Sprintf("已创建 Lua 工具 %q（.pair/tools/%s）。\n\n"+
				"下次发送消息时将自动热加载生效。\n"+
				"用 `lua_tool_list` 查看所有工具，`lua_tool_update` 更新，`lua_tool_delete` 删除。", name, fileName), nil
		},
	})

	// ── lua_tool_update ──
	r.Register(&Tool{
		Name: "lua_tool_update",
		UsageGuide: "更新现有 Lua 自定义工具。按 name 查找并更新其描述/参数/代码。需审核批准。",
		Description: "更新现有 Lua 自定义工具。按 name 查找并更新指定字段（description/parameters/code），" +
			"未提供的字段保持原值。不传 code 则保留原有代码体。",
		Parameters: objSchema(props{
			"name":        strProp("要更新的工具名称"),
			"description": strProp("可选：新的描述"),
			"parameters":  strProp("可选：新的 JSON Schema 字符串"),
			"code":        strProp("可选：新的 run 函数 Lua 代码体（不含 function(args) 包装），不传则保留原有代码"),
		}, "name"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			if name == "" {
				return "", fmt.Errorf("name 不能为空")
			}

			// 根据 tool name 查找 .lua 文件
			filePath, err := findLuaFileByToolName(toolsDir, name)
			if err != nil {
				return "", err
			}

			// 读取当前文件内容
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("读取工具文件失败: %w", err)
			}

			// 解析现有工具元信息
			existingTool, err := buildLuaTool(string(content), filepath.Base(filePath))
			if err != nil {
				return "", fmt.Errorf("解析现有工具失败: %w", err)
			}

			// 从源码中提取现有的 run 代码体
			existingCode := extractLuaCode(string(content))

			// 合并更新值
			newDesc := existingTool.Description
			newParams := existingTool.Parameters
			newCode := existingCode

			if v := argStr(args, "description"); v != "" {
				newDesc = v
			}
			if v := argStr(args, "parameters"); v != "" {
				var p map[string]any
				if err := json.Unmarshal([]byte(v), &p); err != nil {
					return "", fmt.Errorf("parameters 格式错误（需 JSON Schema 字符串）: %w", err)
				}
				newParams = p
			}
			if v := argStr(args, "code"); v != "" {
				newCode = v
			}

			// 生成新的脚本
			script := generateLuaScript(name, newDesc, newParams, newCode)

			// 验证
			if _, err := buildLuaTool(script, filepath.Base(filePath)); err != nil {
				return "", fmt.Errorf("更新后的 Lua 脚本验证失败: %w", err)
			}

			if err := os.WriteFile(filePath, []byte(script), 0o644); err != nil {
				return "", fmt.Errorf("写入工具文件失败: %w", err)
			}

			return fmt.Sprintf("已更新 Lua 工具 %q（%s）。下次发送消息时将自动热加载生效。",
				name, filepath.Base(filePath)), nil
		},
	})

	// ── lua_tool_delete ──
	r.Register(&Tool{
		Name:             "lua_tool_delete",
		UsageGuide:       "删除一个 Lua 自定义工具。按 name 查找 .pair/tools/ 下对应 .lua 文件并删除。需审核批准。",
		Description:      "删除一个 Lua 自定义工具。按 name 查找 .pair/tools/ 下对应的 .lua 文件并删除。此操作不可逆。",
		Parameters:       objSchema(props{"name": strProp("要删除的工具名称")}, "name"),
		RequiresApproval: true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argStr(args, "name")
			if name == "" {
				return "", fmt.Errorf("name 不能为空")
			}

			filePath, err := findLuaFileByToolName(toolsDir, name)
			if err != nil {
				return "", err
			}

			if err := os.Remove(filePath); err != nil {
				return "", fmt.Errorf("删除工具文件失败: %w", err)
			}

			return fmt.Sprintf("已删除 Lua 工具 %q（%s）。热加载后此工具将不可用。",
				name, filepath.Base(filePath)), nil
		},
	})
}

// ─── 辅助 ──────────────────────────────────────────────────

// sanitizeLuaName 将工具名 sanitize 为安全的文件名（小写字母数字下划线）。
func sanitizeLuaName(name string) string {
	name = strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9_]`)
	name = re.ReplaceAllString(name, "_")
	re = regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")
	return strings.Trim(name, "_")
}

// generateLuaScript 根据元信息和代码体生成完整的 Lua 工具脚本。
func generateLuaScript(name, description string, params map[string]any, code string) string {
	var b strings.Builder

	b.WriteString("-- " + name + ".lua")
	b.WriteString("\n-- 由 lua_tool_create 自动生成\n")
	b.WriteString("\nreturn {\n")
	fmt.Fprintf(&b, "  name = %q,\n", name)
	fmt.Fprintf(&b, "  description = %q,\n", description)
	b.WriteString("\n  parameters = ")
	b.WriteString(jsonToLuaTable(params, 2))
	b.WriteString(",\n")
	b.WriteString("\n  run = function(args)\n")
	b.WriteString(indentLuaCode(code, 4))
	b.WriteString("\n  end,\n")
	b.WriteString("}\n")

	return b.String()
}

// jsonToLuaTable 将 Go 值递归转为 Lua 表文本（indentLevel 为当前缩进层级*2空格）。
func jsonToLuaTable(v any, indentLevel int) string {
	indent := strings.Repeat("  ", indentLevel)
	innerIndent := strings.Repeat("  ", indentLevel+1)

	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		var b strings.Builder
		b.WriteString("{\n")
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "%s%s = %s,\n", innerIndent, k, jsonToLuaTable(val[k], indentLevel+1))
		}
		b.WriteString(indent + "}")
		return b.String()
	case []any:
		if len(val) == 0 {
			return "{}"
		}
		// 判断是否全部为简单值（单行）
		allSimple := true
		for _, e := range val {
			switch e.(type) {
			case string, float64, bool, nil:
				continue
			default:
				allSimple = false
			}
		}
		if allSimple {
			var b strings.Builder
			b.WriteString("{ ")
			for i, e := range val {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(jsonToLuaTable(e, indentLevel+1))
			}
			b.WriteString(" }")
			return b.String()
		}
		// 复杂值多行
		var b strings.Builder
		b.WriteString("{\n")
		for _, e := range val {
			fmt.Fprintf(&b, "%s%s,\n", innerIndent, jsonToLuaTable(e, indentLevel+1))
		}
		b.WriteString(indent + "}")
		return b.String()
	case string:
		// 用 %q 确保正确转义
		return strconv.Quote(val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "nil"
	default:
		return "nil"
	}
}

// indentLuaCode 给 code 每行前加 prefix 个空格。
func indentLuaCode(code string, prefix int) string {
	pad := strings.Repeat(" ", prefix)
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = pad + line
		}
	}
	return strings.Join(lines, "\n")
}

// findLuaFileByToolName 扫描 tools 目录，逐个解析 .lua 文件匹配 name 字段。
func findLuaFileByToolName(toolsDir, toolName string) (string, error) {
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("工具目录 .pair/tools/ 不存在")
		}
		return "", fmt.Errorf("读取工具目录失败: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".lua") {
			continue
		}
		path := filepath.Join(toolsDir, e.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		tool, err := buildLuaTool(string(src), e.Name())
		if err != nil {
			continue
		}
		if tool.Name == toolName {
			return path, nil
		}
	}
	return "", fmt.Errorf("未找到名为 %q 的 Lua 工具。用 `lua_tool_list` 查看所有可用工具。", toolName)
}

// extractLuaCode 从 Lua 脚本源码中提取 run 函数的代码体。
// 匹配 run = function(args) ... end 之间的内容。
func extractLuaCode(src string) string {
	re := regexp.MustCompile(`(?s)run\s*=\s*function\s*\(args\)\s*\n(.*?)end\s*[,}\]]`)
	matches := re.FindStringSubmatch(src)
	if len(matches) >= 2 {
		code := matches[1]
		// 去掉每行前 4 格的缩进
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			if len(line) >= 4 && line[:4] == "    " {
				lines[i] = line[4:]
			}
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return ""
}

// paramKeys 从 parameters JSON Schema 中提取属性名列表。
func paramKeys(params map[string]any) []string {
	if params == nil {
		return nil
	}
	props, _ := params["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
