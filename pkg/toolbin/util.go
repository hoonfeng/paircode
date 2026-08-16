// ═══════════════════════════════════════════════════════════════
// util.go — 插件独立二进制的公共辅助（多项目路由/编码探测/输出截断）
//
// ★ 2026-08-16 第四轮：从 internal/agent 抽出的通用工具辅助，供各插件
//   impl 包复用（迁移自 multiproject.go/encoding_detect.go/tools.go/
//   search.go 的相关函数）。
// ═══════════════════════════════════════════════════════════════
package toolbin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// ─── 多项目工作区路由 ──────────────────────────────────────────

// orderedRoots 主项目 + 全局工作区根（有序，主项目优先）。
func orderedRoots(primaryRoot string) []string {
	roots := []string{primaryRoot}
	for _, wr := range WorkspaceRoots {
		if !SamePath(wr, primaryRoot) {
			roots = append(roots, wr)
		}
	}
	return roots
}

// WorkspaceRoots 工作区所有根目录（多根工作区支持，宿主启动时设置）。
var WorkspaceRoots []string

// SamePath 是否同一路径（Windows 大小写不敏感）。
func SamePath(a, b string) bool {
	absA, _ := filepath.Abs(a)
	absB, _ := filepath.Abs(b)
	return strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
}

// workspaceRootNames 列出工作区各项目根目录名（诊断/错误提示用）。
func workspaceRootNames(primaryRoot string) []string {
	var names []string
	for _, wr := range orderedRoots(primaryRoot) {
		names = append(names, filepath.Base(wr))
	}
	return names
}

// ResolveProjectRoot 解析 project 参数 → 项目根目录。
// project 为空 → primaryRoot；否则按 项目目录名（basename）/相对 primary 路径/
// 绝对路径 在 orderedRoots 中匹配；未匹配返回带候选列表的错误。
func ResolveProjectRoot(primaryRoot, project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return primaryRoot, nil
	}
	proj := filepath.Clean(project)
	roots := orderedRoots(primaryRoot)
	for _, wr := range roots {
		if SamePath(wr, proj) || strings.EqualFold(filepath.Base(wr), proj) {
			return wr, nil
		}
	}
	if !filepath.IsAbs(proj) {
		full := filepath.Join(primaryRoot, proj)
		for _, wr := range roots {
			if SamePath(wr, full) {
				return wr, nil
			}
		}
	}
	return "", fmt.Errorf("未找到项目 %q（工作区项目：%v）。project 应为项目目录名（如 wb-ui）或完整路径。",
		project, workspaceRootNames(primaryRoot))
}

// ProjectSchemaProp 生成工具参数里标准 project 字段（多项目路由用）。
func ProjectSchemaProp() map[string]any {
	return StrProp("可选：目标项目（工作区项目目录名如 wb-ui，或相对主项目的路径/绝对路径）。省略 = 主项目。多项目工作区多项目路由用。")
}

// ProjRootFromArgs 从工具参数解析项目根（project 缺省 = primaryRoot）。
func ProjRootFromArgs(primaryRoot string, args map[string]any) (string, error) {
	return ResolveProjectRoot(primaryRoot, ArgStr(args, "project"))
}

// ─── 命令行输出编码探测（Windows 防乱码）──────────────────────

// DecodeCmdOutput 命令行输出字节流 → 正确编码字符串（UTF-8 优先，GBK 兜底）。
func DecodeCmdOutput(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	if s, err := simplifiedchinese.GBK.NewDecoder().String(string(b)); err == nil {
		return s
	}
	return string(b) // 解码失败回退原始字节（至少不更糟）
}

// ─── 输出截断 ──────────────────────────────────────────────────

// CapOutput 截断过长输出（保头 3/4 + 尾 1/4），防工具结果撑爆上下文。
func CapOutput(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	head := limit * 3 / 4
	tail := limit - head
	return s[:head] + "\n...[输出截断 " + fmt.Sprint(len(s)-limit) + " 字节]...\n" + s[len(s)-tail:]
}

// ClampInt 取值约束：v<=0 或越界则回退 def，并夹到 [lo, hi]。
func ClampInt(v, def, lo, hi int) int {
	if v <= 0 {
		v = def
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}

// ─── 路径解析（工作区安全底线）─────────────────────────────────

// ResolvePath 把相对/绝对路径解析为工作区内的绝对路径，越界则报错（安全底线）。
// 先检查路径是否在 primary root 下；若不在，再查是否在 WorkspaceRoots（工作区
// 其他根目录）下。相对路径一律相对于 primary root 解析。
func ResolvePath(root, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, full)
	}
	full = filepath.Clean(full)

	roots := append([]string{root}, WorkspaceRoots...)
	for _, r := range roots {
		rel, err := filepath.Rel(r, full)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if pathExists(full) || parentDirExists(r, full) {
				return full, nil
			}
		}
	}
	return "", fmt.Errorf("路径 %q 超出工作区范围（root: %s）", p, root)
}

// ResolvePathFor 便捷封装：按 args 中 project 参数路由到目标项目根后，
// 再走 ResolvePath 的多根安全解析（相对路径相对该项目根，绝对路径越界拦截）。
func ResolvePathFor(primaryRoot string, args map[string]any, p string) (string, error) {
	projRoot, err := ProjRootFromArgs(primaryRoot, args)
	if err != nil {
		return "", err
	}
	return ResolvePath(projRoot, p)
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// parentDirExists 检查相对于根目录的父级目录是否存在（新建文件时确认目标目录归宿）。
func parentDirExists(root, full string) bool {
	rel, err := filepath.Rel(root, full)
	if err != nil {
		return false
	}
	parent := filepath.Dir(rel)
	if parent == "." {
		return true // 直接在根目录创建文件
	}
	absParent := filepath.Join(root, parent)
	fi, err := os.Stat(absParent)
	return err == nil && fi.IsDir()
}
