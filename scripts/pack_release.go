//go:build ignore

// pack_release.go — 打包发布包（仅 runtime 文件，不包含训练工具）
// go run scripts/pack_release.go
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	distDir := "release/PairCode"
	srcDir := "."

	// 安全校验：确保 distDir 以 release/ 开头
	absDist, _ := filepath.Abs(distDir)
	absRelease, _ := filepath.Abs("release")
	if !strings.HasPrefix(absDist, absRelease) {
		fmt.Fprintf(os.Stderr, "[ERROR] distDir %s is outside release/\n", distDir)
		os.Exit(1)
	}

	// 清理旧目录
	if err := os.RemoveAll(distDir); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] failed to clean %s: %v\n", distDir, err)
		os.Exit(1)
	}

	type entry struct {
		src string
		dst string
	}
	var files []entry
	var dirs []string

	// 1. main binary
	files = append(files, entry{"companion.exe", "companion.exe"})

	// 2. tesseract runtime (only 3 exe + all dll + tessdata)
	for _, f := range []string{"tesseract.exe", "tesseract-uninstall.exe", "winpath.exe"} {
		srcPath := filepath.Join(srcDir, "bin/tesseract", f)
		if _, err := os.Stat(srcPath); err == nil {
			files = append(files, entry{srcPath, filepath.Join("bin/tesseract", f)})
		}
	}
	dllEntries, err := os.ReadDir("bin/tesseract")
	if err == nil {
		for _, e := range dllEntries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".dll") {
				files = append(files, entry{
					filepath.Join(srcDir, "bin/tesseract", e.Name()),
					filepath.Join("bin/tesseract", e.Name()),
				})
			}
		}
	}
	if st, err := os.Stat("bin/tesseract/tessdata"); err == nil && st.IsDir() {
		dirs = append(dirs, "bin/tesseract/tessdata")
	}

	// 3. bin/config
	if _, err := os.Stat("bin/config/models.json"); err == nil {
		files = append(files, entry{"bin/config/models.json", "bin/config/models.json"})
	}

	// 4. bin/headless-check.js
	if _, err := os.Stat("bin/headless-check.js"); err == nil {
		files = append(files, entry{"bin/headless-check.js", "bin/headless-check.js"})
	}

	// 5. config
	for _, f := range []string{"config/models.json", "config/settings.json", "config/mcp.json"} {
		if _, err := os.Stat(f); err == nil {
			files = append(files, entry{f, f})
		}
	}
	for _, d := range []string{"config/skills", "config/roles", "config/philosophy"} {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			dirs = append(dirs, d)
		}
	}

	// 6. assets
	for _, f := range []string{"icon.svg", "icon.png", "icon64.png", "icon128.png"} {
		srcPath := filepath.Join(srcDir, "assets", f)
		if _, err := os.Stat(srcPath); err == nil {
			files = append(files, entry{srcPath, filepath.Join("assets", f)})
		}
	}

	// 7. fonts
	fontEntries, err := os.ReadDir("fonts")
	if err == nil {
		for _, e := range fontEntries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".ttf") {
				files = append(files, entry{
					filepath.Join(srcDir, "fonts", e.Name()),
					filepath.Join("fonts", e.Name()),
				})
			}
		}
	}

	// 8. lib
	if _, err := os.Stat("lib/libSkiaSharp.dll"); err == nil {
		files = append(files, entry{"lib/libSkiaSharp.dll", "lib/libSkiaSharp.dll"})
	}

	totalBytes := int64(0)
	fileCount := 0

	// pre-create directories
	for _, d := range dirs {
		dstPath := filepath.Join(distDir, d)
		os.MkdirAll(dstPath, 0755)
	}

	// copy single files
	for _, f := range files {
		dstPath := filepath.Join(distDir, f.dst)
		os.MkdirAll(filepath.Dir(dstPath), 0755)

		srcFile, err := os.Open(f.src)
		if err != nil {
			fmt.Printf("[WARN] skip %s: %v\n", f.src, err)
			continue
		}
		dstFile, err := os.Create(dstPath)
		if err != nil {
			srcFile.Close()
			fmt.Printf("[WARN] skip %s: %v\n", dstPath, err)
			continue
		}
		written, _ := io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()
		totalBytes += written
		fileCount++
	}

	// copy directories recursively
	for _, d := range dirs {
		srcBase := filepath.Join(srcDir, d)
		dstBase := filepath.Join(distDir, d)
		filepath.WalkDir(srcBase, func(p string, info os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(srcBase, p)
			if rel == "." {
				return nil
			}
			dstPath := filepath.Join(dstBase, rel)
			if info.IsDir() {
				os.MkdirAll(dstPath, 0755)
				return nil
			}
			os.MkdirAll(filepath.Dir(dstPath), 0755)
			s, e1 := os.Open(p)
			if e1 != nil {
				return nil
			}
			defer s.Close()
			d, e2 := os.Create(dstPath)
			if e2 != nil {
				s.Close()
				return nil
			}
			w, _ := io.Copy(d, s)
			s.Close()
			d.Close()
			totalBytes += w
			fileCount++
			return nil
		})
	}

	fmt.Printf("[OK] Release package: %s (%d files, %d MB)\n", distDir, fileCount, totalBytes/1024/1024)
}
