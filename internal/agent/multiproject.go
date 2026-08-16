// ═══════════════════════════════════════════════════════════════
// multiproject.go — 多项目工作区支持（工具级 project 参数路由）
//
// 工作区可含多个项目根（WorkspaceRoots：如 gou-ide + wb-ui + ref）。
// 工具通过 project 参数（项目目录名/相对路径/绝对路径）路由到对应项目根；
// 缺省 = 主项目（primaryRoot）。路径解析仍以 resolvePath 的多根检查兜底。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"fmt"
	"path/filepath"
	"strings"
)

// resolveProjectRoot 解析 project 参数 → 项目根目录。
// project 为空 → primaryRoot；否则按 项目目录名（basename）/相对 primary 路径/
// 绝对路径 在 orderedRoots 中匹配；未匹配返回带候选列表的错误。
func resolveProjectRoot(primaryRoot, project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return primaryRoot, nil
	}
	proj := filepath.Clean(project)
	roots := orderedRoots(primaryRoot)
	for _, wr := range roots {
		if samePath(wr, proj) || strings.EqualFold(filepath.Base(wr), proj) {
			return wr, nil
		}
	}
	if !filepath.IsAbs(proj) {
		full := filepath.Join(primaryRoot, proj)
		for _, wr := range roots {
			if samePath(wr, full) {
				return wr, nil
			}
		}
	}
	return "", fmt.Errorf("未找到项目 %q（工作区项目：%v）。project 应为项目目录名（如 wb-ui）或完整路径。",
		project, workspaceRootNames(primaryRoot))
}

// projectSchemaProp 生成工具参数里标准 project 字段（多项目路由用）。
// 所有支持多项目的工具统一使用此描述，保证模型行为一致。
func projectSchemaProp() map[string]any {
	return strProp("可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区：gou-ide、wb-ui、ref 等。")
}

// projRootFromArgs 从工具参数解析项目根（project 缺省 = primaryRoot）。
// 便捷封装：工具 Handler 第一行调用。
func projRootFromArgs(primaryRoot string, args map[string]any) (string, error) {
	return resolveProjectRoot(primaryRoot, argStr(args, "project"))
}

// resolvePathFor 便捷封装：按 args 中 project 参数路由到目标项目根后，
// 再走 resolvePath 的多根安全解析（相对路径相对该项目根，绝对路径越界拦截）。
// 用于核心文件工具（read/write/edit/move/delete/run_command 的 cwd）。
func resolvePathFor(primaryRoot string, args map[string]any, p string) (string, error) {
	projRoot, err := projRootFromArgs(primaryRoot, args)
	if err != nil {
		return "", err
	}
	return resolvePath(projRoot, p)
}

// workspaceRootNames 列出工作区各项目根目录名（诊断/错误提示用）。
func workspaceRootNames(primaryRoot string) []string {
	var names []string
	for _, wr := range orderedRoots(primaryRoot) {
		names = append(names, filepath.Base(wr))
	}
	return names
}

// samePath 比较两个路径是否指向同一位置（Windows 不区分大小写）。
func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
