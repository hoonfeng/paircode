//go:build linux || darwin

package pty

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	detectShellsOnce sync.Once
	detectedShells   []Shell
)

// DetectShells 探测 Unix（Linux/macOS）可用解释器。
// 有些系统没有 bash（如精简容器/Alpine 默认 ash、部分发行版），故不能写死 bash——
// 逐个探测：$SHELL（用户默认）优先，再扫常见解释器的常见路径，再走 PATH，取真实存在的。
// 结果缓存，仅首次调用时探测（shell 列表运行时不会变化）。
func DetectShells() []Shell {
	detectShellsOnce.Do(func() {
		detectedShells = detectShellsUncached()
	})
	return detectedShells
}

func detectShellsUncached() []Shell {
	var out []Shell
	seen := map[string]bool{}
	add := func(name, path string, args ...string) {
		if path == "" || seen[path] {
			return
		}
		if fi, err := os.Stat(path); err != nil || fi.IsDir() {
			return // 不存在/非文件→跳过（这就是「探测可用解释器」）
		}
		seen[path] = true
		out = append(out, Shell{Name: name, Path: path, Args: args})
	}

	// 1) 用户默认 $SHELL 最优先
	if s := os.Getenv("SHELL"); s != "" {
		add(filepath.Base(s), s, "-i")
	}
	// 2) 常见解释器的常见绝对路径（覆盖 $SHELL 未设/被精简的系统）
	for _, c := range []struct{ name, path string }{
		{"bash", "/bin/bash"}, {"bash", "/usr/bin/bash"}, {"bash", "/usr/local/bin/bash"},
		{"zsh", "/bin/zsh"}, {"zsh", "/usr/bin/zsh"}, {"zsh", "/usr/local/bin/zsh"},
		{"fish", "/usr/bin/fish"}, {"fish", "/usr/local/bin/fish"},
		{"ash", "/bin/ash"}, {"dash", "/bin/dash"},
		{"sh", "/bin/sh"}, {"sh", "/usr/bin/sh"},
	} {
		add(c.name, c.path, "-i")
	}
	// 3) PATH 兜底（自定义安装位置）
	for _, name := range []string{"bash", "zsh", "fish", "ash", "dash", "sh"} {
		if p, err := exec.LookPath(name); err == nil {
			add(name, p, "-i")
		}
	}
	if len(out) == 0 { // 极端兜底：POSIX 必有 /bin/sh
		out = append(out, Shell{Name: "sh", Path: "/bin/sh", Args: []string{"-i"}})
	}
	return out
}

func fallbackShell() Shell { return Shell{Name: "sh", Path: "/bin/sh", Args: []string{"-i"}} }
