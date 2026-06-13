package agent

// project_index 工具：通过递归扫描项目目录，构建文件树结构。
//
// project_index 递归扫描项目目录，排除 .git、node_modules、vendor 等常见目录，
// 输出 JSON 格式的目录层级（包含文件大小和修改时间）。
// 支持通过 path 限定子目录、depth 限制深度、filter 按名称模式过滤。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── 跳过目录 ──

// skipDirsForProjectIndex 在递归扫描时跳过的目录（与 entryconfig.go 保持一致）。
var skipDirsForProjectIndex = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"out":          true,
	".idea":        true,
	".vscode":      true,
	"third_party":  true,
	"thirdparty":   true,
	"coverage":     true,
	".cache":       true,
	".pair":        true,
}

// ── 树节点类型 ──

// treeNode 树中的一个节点（文件或目录）。
type treeNode struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`               // "file" 或 "directory"
	Size     int64      `json:"size,omitempty"`     // 文件字节数（目录无此字段）
	Modified string     `json:"modified,omitempty"` // 修改时间 RFC3339（目录无此字段）
	Children []treeNode `json:"children,omitempty"` // 子节点（仅目录有）
}

// projectIndexOutput 整体输出结构。
type projectIndexOutput struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Size     int64      `json:"size,omitempty"`
	Modified string     `json:"modified,omitempty"`
	Children []treeNode `json:"children,omitempty"`
}

// ── 扫描逻辑 ──

// buildTree 递归构建文件树。
// 参数：
//
//	root    — 扫描的绝对根目录
//	current — 当前要扫描的绝对路径
//	depth   — 剩余深度（-1 表示不限）
//	filter  — 文件名模式匹配（filepath.Match），空字符串表示不过滤
//
// 返回：(树节点, 遇到的错误)
func buildTree(root, current string, depth int, filter string) (treeNode, error) {
	info, err := os.Stat(current)
	if err != nil {
		return treeNode{}, fmt.Errorf("stat %s: %w", current, err)
	}

	node := treeNode{
		Name:     filepath.Base(current),
		Modified: info.ModTime().UTC().Format(time.RFC3339),
	}

	if info.IsDir() {
		node.Type = "directory"
		if depth == 0 {
			// 达到深度限制，不再展开子目录
			return node, nil
		}

		entries, err := os.ReadDir(current)
		if err != nil {
			return node, nil // 跳过不可读的目录
		}

		// 按名称排序（目录在前）
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})

		nextDepth := depth - 1
		if depth < 0 {
			nextDepth = -1
		}

		for _, e := range entries {
			name := e.Name()

			// 跳过隐藏文件和跳过目录
			if e.IsDir() {
				if skipDirsForProjectIndex[name] {
					continue
				}
				// 跳过以 . 开头的隐藏目录（但保留 .github 等有意义目录）
				if strings.HasPrefix(name, ".") && name != "." && name != ".." {
					// 保留部分以 . 开头的常见有意义目录
					meaningful := map[string]bool{
						".github": true,
						".vscode": true,
						".idea":   true,
						".gitlab": true,
						".husky":  true,
					}
					if !meaningful[name] && !skipDirsForProjectIndex[name] {
						continue // 跳过未知隐藏目录
					}
				}
			} else {
				// 跳过隐藏文件
				if strings.HasPrefix(name, ".") {
					continue
				}
			}

			childPath := filepath.Join(current, name)
			child, err := buildTree(root, childPath, nextDepth, filter)
			if err != nil {
				continue // 跳过出错的子节点
			}

			// 应用 filter（仅对文件，目录总是显示）
			if filter != "" && child.Type == "file" {
				if ok, _ := filepath.Match(filter, child.Name); !ok {
					continue
				}
			}

			node.Children = append(node.Children, child)
		}
	} else {
		node.Type = "file"
		node.Size = info.Size()
	}

	return node, nil
}

// ── 工具处理函数 ──

// projectIndexHandler 创建 project_index 工具的处理函数。
func projectIndexHandler(root string) ToolHandler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		searchRoot := root
		if sub := argStr(args, "path"); sub != "" {
			var err error
			searchRoot, err = resolvePath(root, sub)
			if err != nil {
				return "", err
			}
		}

		depth := argInt(args, "depth", -1) // -1 表示不限深度
		if depth < -1 {
			depth = -1
		}
		if depth > 20 {
			depth = 20 // 防止过深递归撑爆
		}

		filter := argStr(args, "filter") // 可选 glob 过滤

		tree, err := buildTree(root, searchRoot, depth, filter)
		if err != nil {
			return "", fmt.Errorf("扫描目录失败: %w", err)
		}

		// 顶层节点：用相对路径或项目名
		rel, _ := filepath.Rel(root, searchRoot)
		if rel == "." || rel == "" {
			tree.Name = filepath.Base(root)
		} else {
			tree.Name = rel
		}

		// 输出 JSON（美化缩进）
		out, err := json.MarshalIndent(tree, "", "  ")
		if err != nil {
			return "", fmt.Errorf("JSON 序列化失败: %w", err)
		}

		return string(out), nil
	}
}

// ── 工具注册 ──

// registerProjectIndexTool 注册 project_index 工具。
func registerProjectIndexTool(r *Registry, root string) {
	r.Register(&Tool{
		Name: "project_index",
		Description: "通过递归扫描项目目录，构建文件树结构（JSON 格式），" +
			"排除 .git、node_modules、vendor 等常见非源码目录。" +
			"返回的 JSON 包含：name（文件名）、type（file/directory）、" +
			"size（文件字节数）、modified（修改时间 RFC3339）、children（子节点列表）。" +
			"可选参数：path 限定子目录，depth 限制递归深度，filter 按 glob 模式过滤文件名。",
		Parameters: objSchema(props{
			"path":   strProp("可选：限定扫描的子目录路径，留空则扫描整个项目"),
			"depth":  intProp("可选：递归深度（1=仅当前目录，2=当前+子目录，-1=不限），默认 -1"),
			"filter": strProp("可选：文件名过滤 glob 模式（如 \"*.go\"），仅对文件生效"),
		}),
		ReadOnly: true,
		Handler:  projectIndexHandler(root),
	})
}
