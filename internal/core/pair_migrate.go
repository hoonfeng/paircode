package core

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// MigrateLegacyBinPairData 一次性迁移旧版 bin\.pair 数据
// （2026-09-09 InstallDir bin 上跳修复的配套升级路径）。
//
// 背景：旧版 InstallDir() 不做 bin 上跳，bin 下运行的 exe 把 .pair/
// （工具集/插件/进化资产/mcp 配置）写在 bin\.pair；修复后安装根 .pair
// 才是真相源。此函数把 bin\.pair 中安装根缺失的文件「只补缺不覆盖」地
// 拷贝过来（用户编辑过的工具集/资产不丢）。
//
// 幂等：源不存在直接返回；目标已存在的文件一律跳过（保留安装根/用户现状）；
// 不删除源目录（保守策略，确认无误后可手动删除）。
//
// ★ toolsets 子目录跳过框架历史预设名文件（v1/v2 英文名与 v3 中文名均为
// 框架播种产物，非用户数据——迁移会让旧预设以「自定义集合」身份复活，
// 与新版预设并存造成混乱）。用户自建集合（非预设名）正常迁移。
func MigrateLegacyBinPairData() {
	root := InstallDir()
	legacy := filepath.Join(root, "bin", ".pair")
	st, err := os.Stat(legacy)
	if err != nil || !st.IsDir() {
		return
	}
	target := filepath.Join(root, ".pair")
	// 预设旧名（v1/v2 英文 + v3 中文）+ 播种标记：一律不迁
	presetNames := map[string]bool{
		"planning": true, "fullstack": true, "office": true, "debug": true,
		"default": true, "full": true, "dev": true, "test": true, "docs": true,
		"计划讨论": true, "全栈开发": true, "办公": true, "调试": true,
		"基础": true, "全功能": true,
	}
	n := 0
	for _, e := range entriesOf(legacy) {
		name := e.Name()
		low := strings.ToLower(name)
		if e.IsDir() {
			if low == "node_modules" || low == ".git" {
				continue
			}
			skip := map[string]bool{}
			if low == "toolsets" {
				skip = presetNames
			}
			n += migratePairTree(filepath.Join(legacy, name), filepath.Join(target, name), 0, skip)
			continue
		}
		if name == ".preset-seeded" {
			continue
		}
		if copyIfMissing(filepath.Join(legacy, name), filepath.Join(target, name)) {
			n++
		}
	}
	if n > 0 {
		log.Printf("[migrate] 旧 bin\\.pair → 安装根 .pair 迁移完成：补齐 %d 个文件（源目录保留，可手动删除 %s）", n, legacy)
	}
}

func entriesOf(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	return entries
}

// migratePairTree 递归「只补缺」复制（跳过 node_modules/.git/符号链接，防越界与海量拷贝；
// skip 非 nil 时跳过当前层命中名单的文件）。返回拷贝的文件数。
func migratePairTree(srcDir, dstDir string, depth int, skip map[string]bool) int {
	if depth > 8 {
		return 0
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0
	}
	copied := 0
	for _, e := range entries {
		name := e.Name()
		low := strings.ToLower(name)
		if e.IsDir() {
			if low == "node_modules" || low == ".git" {
				continue
			}
			copied += migratePairTree(filepath.Join(srcDir, name), filepath.Join(dstDir, name), depth+1, skip)
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if skip != nil && skip[strings.TrimSuffix(name, filepath.Ext(name))] {
			continue
		}
		if copyIfMissing(filepath.Join(srcDir, name), filepath.Join(dstDir, name)) {
			copied++
		}
	}
	return copied
}

// copyIfMissing 目标已存在则跳过（不覆盖）；不存在则拷贝。
func copyIfMissing(src, dst string) bool {
	if _, err := os.Stat(dst); err == nil {
		return false // 目标已存在：不覆盖
	}
	if err := copyFileSync(src, dst); err != nil {
		log.Printf("[migrate] 迁移 %s 失败（跳过）: %v", src, err)
		return false
	}
	return true
}

func copyFileSync(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
