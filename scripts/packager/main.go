// PairCode Release Packager
// 读取 packager.json 配置，执行：构建前端 → 生成图标 → 编译 Go 二进制 → 打包发布目录
// 产品编译打包工具，不含脱敏/代码发布功能。

package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Version          string     `json:"version"`
	AppName          string     `json:"appName"`
	Description      string     `json:"description"`
	Company          string     `json:"company"`
	Copyright        string     `json:"copyright"`
	Icon             string     `json:"icon"`
	FrontendDir      string     `json:"frontendDir"`
	SkipBuildFrontend bool      `json:"skipBuildFrontend"`
	MainPkg          string     `json:"mainPkg"`
	Output           string     `json:"output"`
	Secrets          SecretsCfg `json:"secrets"`
	Dist             DistConfig `json:"dist"`
	Tools            Tools      `json:"tools"`
}

type SecretsCfg struct {
	Fields []string `json:"fields"`
	Files  []string `json:"files"`
}

type DistConfig struct {
	OutputDir string      `json:"outputDir"`
	DirName   string      `json:"dirName"`
	Include   []DistEntry `json:"include"`
}

type DistEntry struct {
	Src       string `json:"src"`
	Dst       string `json:"dst"`
	Optional  bool   `json:"optional"`
	Recursive bool   `json:"recursive"`
}

type Tools struct {
	Windres string `json:"windres"`
	FFmpeg  string `json:"ffmpeg"`
	Go      string `json:"go"`
	Npm     string `json:"npm"`
}

func main() {
	root, _ := os.Getwd()
	cfgPath := filepath.Join(root, "packager.json")
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg := loadConfig(cfgPath)
	fmt.Printf("PairCode Packager v%s\n", cfg.Version)
	fmt.Printf("  工作区: %s\n", root)
	fmt.Println(strings.Repeat("-", 60))

	// Step 1: 构建前端
	if !cfg.SkipBuildFrontend {
		step("1/4", "构建前端...")
		if err := buildFrontend(root, cfg); err != nil {
			fatalf("前端构建失败: %v", err)
		}
		ok("前端构建完成")
	} else {
		fmt.Println("  跳过前端构建")
	}

	// Step 2: 生成图标
	step("2/4", "生成 Windows 图标...")
	if err := generateIcon(root, cfg); err != nil {
		fmt.Printf("  图标生成失败（可跳过）: %v\n", err)
	} else {
		ok("图标生成完成")
	}

	// Step 3: 编译资源
	step("3/4", "编译资源文件...")
	if err := compileResource(cfg); err != nil {
		fmt.Printf("  资源编译失败（可跳过）: %v\n", err)
	} else {
		ok("资源编译完成")
	}

	// Step 4: 编译 Go 二进制
	step("4/4", "编译 companion.exe...")
	exePath, err := buildGo(cfg)
	if err != nil {
		fatalf("Go 编译失败: %v", err)
	}
	ok(fmt.Sprintf("编译完成: %s", exePath))

	// 打包发布目录
	distDir, zipPath, err := packageDist(root, exePath, cfg)
	if err != nil {
		fmt.Printf("  打包失败: %v\n", err)
	} else {
		ok(fmt.Sprintf("发布目录: %s", distDir))
		if zipPath != "" {
			info, _ := os.Stat(zipPath)
			sizeMB := float64(info.Size()) / 1024 / 1024
			ok(fmt.Sprintf("ZIP 包: %s (%.1f MB)", zipPath, sizeMB))
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("打包完成!")
}

func step(label, msg string) { fmt.Printf("\n%s %s\n", label, msg) }
func ok(msg string)           { fmt.Printf("  OK %s\n", msg) }
func fatalf(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, "  FAIL %s\n", fmt.Sprintf(msg, a...))
	os.Exit(1)
}

func loadConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("读取配置失败 %s: %v", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fatalf("解析配置失败: %v", err)
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Output == "" {
		cfg.Output = "companion.exe"
	}
	if cfg.Dist.OutputDir == "" {
		cfg.Dist.OutputDir = "release"
	}
	if cfg.Dist.DirName == "" {
		cfg.Dist.DirName = "PairCode"
	}
	if cfg.Tools.Go == "" {
		cfg.Tools.Go = "go"
	}
	return &cfg
}

// ─── 1. 构建前端 ───────────────────────────────────────────

func buildFrontend(root string, cfg *Config) error {
	frontendDir := filepath.Join(root, cfg.FrontendDir)
	if _, err := os.Stat(filepath.Join(frontendDir, "package.json")); os.IsNotExist(err) {
		return fmt.Errorf("前端目录不存在: %s", frontendDir)
	}
	cmd := exec.Command(cfg.Tools.Npm, "run", "build")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ─── 2. 生成图标 ───────────────────────────────────────────

func generateIcon(root string, cfg *Config) error {
	if cfg.Icon == "" {
		return nil
	}
	iconSrc := filepath.Join(root, cfg.Icon)
	if _, err := os.Stat(iconSrc); os.IsNotExist(err) {
		return fmt.Errorf("图标文件不存在: %s", iconSrc)
	}

	sizes := []int{16, 32, 48, 64, 128, 256}
	pngFiles := make([]string, 0, len(sizes))
	iconDir := filepath.Join(root, filepath.Dir(cfg.MainPkg))

	for _, s := range sizes {
		out := filepath.Join(iconDir, fmt.Sprintf("icon_%d.png", s))
		cmd := exec.Command(cfg.Tools.FFmpeg, "-y", "-i", iconSrc, "-s", fmt.Sprintf("%dx%d", s, s), out)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("缩放 %dpx: %v\n%s", s, err, string(output))
		}
		pngFiles = append(pngFiles, out)
	}

	icoPath := filepath.Join(iconDir, "icon.ico")
	packICO(icoPath, pngFiles)
	fmt.Printf("  图标: %s (%d sizes)\n", icoPath, len(sizes))
	return nil
}

func packICO(path string, pngFiles []string) error {
	type entry struct{ w, h byte; size, off uint32 }
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	entries := make([]entry, len(pngFiles))
	offset := uint32(6 + len(pngFiles)*16)
	for i, p := range pngFiles {
		data, _ := os.ReadFile(p)
		var w int
		fmt.Sscanf(filepath.Base(p), "icon_%d.png", &w)
		ww := byte(w)
		if w > 255 {
			ww = 0
		}
		entries[i] = entry{ww, ww, uint32(len(data)), offset}
		offset += uint32(len(data))
		f.Write(data)
	}
	f.Seek(0, 0)
	f.Write([]byte{0, 0, 1, 0, byte(len(pngFiles)), 0})
	for _, e := range entries {
		f.Write([]byte{e.w, e.h, 0, 0, 1, 0, 32, 0,
			byte(e.size), byte(e.size >> 8), byte(e.size >> 16), byte(e.size >> 24),
			byte(e.off), byte(e.off >> 8), byte(e.off >> 16), byte(e.off >> 24)})
	}
	return nil
}

// ─── 3. 编译资源 ───────────────────────────────────────────

func compileResource(cfg *Config) error {
	rcFile := filepath.Join(filepath.Dir(cfg.MainPkg), "companion.rc")
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		return nil
	}
	sysoFile := filepath.Join(filepath.Dir(cfg.MainPkg), "companion.syso")
	cmd := exec.Command(cfg.Tools.Windres, "-i", rcFile, "-o", sysoFile, "-O", "coff")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("windres: %v\n%s", err, string(output))
	}
	return nil
}

// ─── 4. 编译 Go ────────────────────────────────────────────

func buildGo(cfg *Config) (string, error) {
	ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
	if runtime.GOOS == "windows" {
		ldflags += " -H windowsgui"
	}
	outputPath := filepath.Join("release", cfg.Output)
	os.MkdirAll(filepath.Dir(outputPath), 0755)
	cmd := exec.Command(cfg.Tools.Go, "build",
		"-ldflags="+ldflags,
		"-o", outputPath,
		cfg.MainPkg,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()
	env = append(env, "CGO_ENABLED=1")
	cmd.Env = env
	return outputPath, cmd.Run()
}

// ─── 打包发布目录 ──────────────────────────────────────────

func packageDist(root, exePath string, cfg *Config) (string, string, error) {
	distDir := filepath.Join(root, cfg.Dist.OutputDir, cfg.Dist.DirName)
	os.RemoveAll(distDir)
	os.MkdirAll(distDir, 0755)

	// 复制主程序
	exeName := filepath.Base(cfg.Output)
	if err := copyFile(filepath.Join(distDir, exeName), exePath); err != nil {
		return "", "", fmt.Errorf("复制主程序: %w", err)
	}

	// 复制资源文件
	for _, entry := range cfg.Dist.Include {
		srcPath := filepath.Join(root, entry.Src)
		dstPath := filepath.Join(distDir, entry.Dst)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			if entry.Optional {
				continue
			}
			return "", "", fmt.Errorf("必需文件缺失: %s", entry.Src)
		}
		if entry.Recursive {
			filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(srcPath, path)
				if rel == "." {
					return nil
				}
				target := filepath.Join(dstPath, rel)
				if info.IsDir() {
					return os.MkdirAll(target, 0755)
				}
				return copyFile(target, path)
			})
		} else {
			os.MkdirAll(filepath.Dir(dstPath), 0755)
			copyFile(dstPath, srcPath)
		}
	}

	// 清除密钥
	if len(cfg.Secrets.Fields) > 0 && len(cfg.Secrets.Files) > 0 {
		stripSecrets(distDir, cfg.Secrets.Fields, cfg.Secrets.Files)
	}

	// ZIP
	zipName := fmt.Sprintf("%s-%s.zip", cfg.Dist.DirName, cfg.Version)
	zipPath := filepath.Join(root, cfg.Dist.OutputDir, zipName)
	os.Remove(zipPath)
	f, _ := os.Create(zipPath)
	w := zip.NewWriter(f)
	filepath.Walk(distDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(distDir, path)
		wr, _ := w.Create(filepath.ToSlash(rel))
		data, _ := os.ReadFile(path)
		wr.Write(data)
		return nil
	})
	w.Close()
	f.Close()

	return distDir, zipPath, nil
}

func copyFile(dst, src string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}

// stripSecrets 从发布目录的指定 JSON 文件中删除密钥字段。
func stripSecrets(distDir string, fields, files []string) {
	for _, rel := range files {
		path := filepath.Join(distDir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		changed := false
		for _, f := range fields {
			if _, ok := m[f]; ok {
				delete(m, f)
				changed = true
			}
		}
		if changed {
			cleaned, _ := json.MarshalIndent(m, "", "  ")
			os.WriteFile(path, cleaned, 0644)
			fmt.Printf("  密钥已清除: %s\n", rel)
		}
	}
}
