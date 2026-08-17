// log_file.go — stdout/stderr 日志落盘（用户侧排查「无响应」等问题用）
//
// 发布版用户通常双击运行 pair.exe，控制台窗口不常看；本模块把
//   - log 包输出（log.Printf/Println，默认写 os.Stderr）
//   - fmt.Printf 输出（写 os.Stdout）
// 同时写入 <安装目录>/logs/paircode.log，用户可直接把该文件发给开发者。
//
// 实现：log 包用 MultiWriter(原stderr, 文件)；os.Stdout 用 os.Pipe 替换，
// 读协程 tee 到（原stdout, 文件）。并发写管道 ≤64KB 原子，日志行远小于此。

package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/internal/core"
)

var (
	origStdout *os.File
	origStderr *os.File
	logFile    *os.File
)

// initLogFile 初始化日志落盘。幂等：重复调用直接返回。
// 在任何 log.Printf / fmt.Printf 之前调用（main 最前面）。
func initLogFile() {
	if logFile != nil {
		return
	}
	dir := filepath.Join(core.InstallDir(), "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[log] 创建日志目录失败: %v", err)
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "paircode.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[log] 打开日志文件失败: %v", err)
		return
	}
	logFile = f

	// log 包 → 原 stderr + 文件
	origStderr = os.Stderr
	log.SetOutput(io.MultiWriter(origStderr, f))

	// fmt.Printf（os.Stdout）→ 原 stdout + 文件（管道 tee）
	origStdout = os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("[log] 创建输出管道失败: %v", err)
		return
	}
	os.Stdout = w
	go func() {
		buf := make([]byte, 8192)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				origStdout.Write(buf[:n])
				f.Write(buf[:n])
			}
			if rerr != nil {
				return
			}
		}
	}()
	log.Printf("[log] 日志已落盘: %s", filepath.Join(dir, "paircode.log"))
}
