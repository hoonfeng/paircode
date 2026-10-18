//go:build stub

// PairCode 单文件启动器（自解压 stub）。
//
// 分发形态：发布包只有一个可执行文件（Windows: PairCode.exe，Linux/macOS: PairCode）。
// 它内嵌真正的 GUI 主程序、Skia 原生库与插件/前端资源，首次启动时释放到
// 用户级运行时目录，之后直接复用该目录启动。
//
// 为什么必须由外层进程解压：Windows PE 在进程启动前就解析导入表，
// libSkiaSharp.dll 必须在主程序启动时已存在于可搜索路径；Go 的 init/main
// 在导入表解析之后才执行，无法自救。因此解压只能放在一个不依赖 Skia 的
// 外层可执行文件里。
//
// 运行时目录（按平台）：
//   - Windows: %LOCALAPPDATA%\PairCode\runtime\<version>\
//   - Linux:   $XDG_DATA_HOME/PairCode/runtime/<version>/（缺省 ~/.local/share）
//   - macOS:   ~/Library/Application Support/PairCode/runtime/<version>/
//
// 进程模型：stub 释放并启动主程序后退出（健康检查窗口内确认存活），
// 任务管理器里只留下主程序一个进程，不产生常驻的父进程。
package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// version 由构建时注入：-ldflags "-X main.version=1.6.21"。
var version = "dev"

// healthWindow 启动后确认主程序存活的观察窗口；窗口内退出视为启动失败。
const healthWindow = 1500 * time.Millisecond

// payload 由 build-stub 步骤在构建前填充（GUI 主程序 + Skia 库 + 插件/前端资源）。
// all: 前缀用于把 .pair 这类点开头的目录也纳入 embed。
//
//go:embed all:payload
var payloadFS embed.FS

func main() {
	rtDir, err := runtimeDir()
	if err != nil {
		fatal("定位运行时目录失败: " + err.Error())
		return
	}
	if err := ensurePayload(rtDir); err != nil {
		fatal("释放运行时文件失败: " + err.Error())
		return
	}
	migrateUserData(rtDir)

	exePath := filepath.Join(rtDir, mainExeName())
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Dir = rtDir
	// 主程序日志自行落盘（logs/paircode.log）；stub 在 Windows 上是 GUI 子系统，
	// 没有控制台，继承的 stdout/stderr 可能无效，写失败会被 os.File 吞掉。
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "PAIRCODE_STUB=1")
	if err := cmd.Start(); err != nil {
		fatal(fmt.Sprintf("启动主程序失败: %v\n%s", err, exePath))
		return
	}

	exit := make(chan error, 1)
	go func() { exit <- cmd.Wait() }()
	select {
	case err := <-exit:
		// 健康窗口内就退出：大概率是启动失败（缺库、权限、崩溃）。
		if err != nil {
			fatal(fmt.Sprintf("主程序启动后立即退出: %v", err))
			return
		}
	case <-time.After(healthWindow):
		// 主程序已稳定运行，stub 退出，不留下常驻父进程。
	}
}

// runtimeDir 返回本版本的主程序释放目录（不存在时创建）。
func runtimeDir() (string, error) {
	base, err := userDataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "runtime", version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// userDataDir 返回用户级 PairCode 数据根目录。
func userDataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "PairCode"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Local", "PairCode"), nil
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "PairCode"), nil
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return filepath.Join(v, "PairCode"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "PairCode"), nil
	}
}

// ensurePayload 把内嵌的 payload 树释放到运行时目录。
// 同尺寸文件直接跳过：版本目录按 version 隔离，正常只在首次启动时写盘。
func ensurePayload(dir string) error {
	return fs.WalkDir(payloadFS, "payload", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, "payload"), "/")
		if rel == "" {
			return nil
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if st, err := os.Stat(dst); err == nil && st.Size() == info.Size() {
			return nil
		}
		data, err := payloadFS.ReadFile(p)
		if err != nil {
			return err
		}
		return writeExecutable(dst, data, isExecutable(rel))
	})
}

// isExecutable 判断 payload 内哪些文件需要可执行权限。
func isExecutable(rel string) bool {
	base := filepath.Base(rel)
	switch base {
	case "pair.exe", "pair":
		return true
	}
	// Linux 上库的 SONAME 带版本后缀（libSkiaSharp.so.119.0.0），按前缀判断。
	if strings.HasPrefix(base, "libSkiaSharp") {
		return true
	}
	return false
}

// writeExecutable 先写临时文件再原子重命名，避免中断留下半个文件。
func writeExecutable(dst string, data []byte, exec bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if exec {
		mode = 0o755
	}
	tmp := dst + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// migrateUserData 升级时把上一个版本目录里的用户数据复制到新版本目录，
// 避免换版本后丢失工作区、会话与配置。
// 只复制用户数据（config、.pair 下的会话/记忆/任务/工具集），不复制程序资源。
func migrateUserData(rtDir string) {
	base := filepath.Dir(rtDir) // .../runtime
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	var prev string
	var prevMod time.Time
	for _, e := range entries {
		if !e.IsDir() || e.Name() == version {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(prevMod) {
			prev, prevMod = filepath.Join(base, e.Name()), info.ModTime()
		}
	}
	if prev == "" {
		return
	}
	if _, err := os.Stat(filepath.Join(rtDir, "config")); err == nil {
		return // 已有配置，不做覆盖
	}
	for _, rel := range []string{
		"config",
		filepath.Join(".pair", "memory"),
		filepath.Join(".pair", "conversations"),
		filepath.Join(".pair", "tasks"),
		filepath.Join(".pair", "skills"),
		filepath.Join(".pair", "toolsets"),
		filepath.Join(".pair", "pair.db"),
	} {
		src := filepath.Join(prev, rel)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := copyTree(src, filepath.Join(rtDir, rel)); err != nil {
			log.Printf("[stub] 迁移 %s 失败（忽略）: %v", rel, err)
		}
	}
	log.Printf("[stub] 已从旧版本目录迁移用户数据: %s", prev)
}

// copyTree 递归复制文件或目录（不存在时忽略）。
func copyTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return copyFile(src, dst)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
