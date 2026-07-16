// PairCode Release Packager
// 读取 packager.json 配置，执行：脱敏 → 构建前端 → 生成图标 → 编译 Go 二进制 → 打包发布目录
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

// ─── 配置 ──────────────────────────────────────────────────

type Config struct {
	Version     string     `json:"version"`
	AppName     string     `json:"appName"`
	Description string     `json:"description"`
	Company     string     `json:"company"`
	Copyright   string     `json:"copyright"`
	Icon        string     `json:"icon"`
	FrontendDir string     `json:"frontendDir"`
	MainPkg     string     `json:"mainPkg"`
	Output      string     `json:"output"`
	SkipBuildFrontend bool `json:"skipBuildFrontend"`
	SkipGoModTidy     bool `json:"skipGoModTidy"`
	Sanitize    Sanitize   `json:"sanitize"`
	Dist        DistConfig `json:"dist"`
	Tools       Tools      `json:"tools"`
}

type Sanitize struct {
	Enabled      bool       `json:"enabled"`
	OldModule    string     `json:"oldModule"`
	NewModule    string     `json:"newModule"`
	Replacements [][2]string `json:"replacements"`
	SkipFiles    []string   `json:"skipFiles"`
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

// ─── 主流程 ────────────────────────────────────────────────

func main() {
	root, _ := os.Getwd()
	cfgPath := filepath.Join(root, "packager.json")
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	cfg := loadConfig(cfgPath)
	fmt.Printf("PairCode Release Packager v%s\n", cfg.Version)
	fmt.Printf("  工作区: %s\n", root)
	fmt.Println(strings.Repeat("-", 60))

	// Step 1: 构建前端
	step("1/6", "构建前端...")
	if err := buildFrontend(root, cfg); err != nil {
		fatalf("前端构建失败: %v", err)
	}
	ok("前端构建完成")

	// Step 2: 创建脱敏源码副本
	var buildDir string
	if cfg.Sanitize.Enabled {
		step("2/6", "创建脱敏源码副本...")
		var err error
		buildDir, err = createSanitizedCopy(root, cfg)
		if err != nil {
			fatalf("脱敏复制失败: %v", err)
		}
		fmt.Printf("  脱敏副本: %s\n", buildDir)
		ok("脱敏完成")
	} else {
		buildDir = root
	}

	// Step 3: 生成图标
	step("3/6", "生成 Windows 图标...")
	if err := generateIcon(root, buildDir, cfg); err != nil {
		fmt.Printf("  WARNING 图标生成失败（可跳过）: %v\n", err)
	} else {
		ok("图标生成完成")
	}

	// Step 4: 编译资源
	step("4/6", "编译资源文件...")
	if err := compileResource(root, buildDir, cfg); err != nil {
		fmt.Printf("  WARNING 资源编译失败（可跳过）: %v\n", err)
	} else {
		ok("资源编译完成")
	}

	// Step 5: 编译 Go
	step("5/6", "编译 companion.exe...")
	exePath, err := buildGo(root, buildDir, cfg)
	if err != nil {
		fatalf("Go 编译失败: %v", err)
	}
	ok(fmt.Sprintf("编译完成: %s", exePath))

	// Step 6: 打包
	step("6/6", "打包发布目录...")
	distDir, zipPath, err := packageDist(root, exePath, cfg)
	if err != nil {
		fatalf("打包失败: %v", err)
	}
	ok(fmt.Sprintf("发布目录: %s", distDir))
	if zipPath != "" {
		info, _ := os.Stat(zipPath)
		sizeMB := float64(info.Size()) / 1024 / 1024
		ok(fmt.Sprintf("ZIP 包: %s (%.1f MB)", zipPath, sizeMB))
	}

	if buildDir != root {
		os.RemoveAll(buildDir)
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("发布包制作完成!")
}

func step(label, msg string) {
	fmt.Printf("\n%s %s\n", label, msg)
}

func ok(msg string) {
	fmt.Printf("  OK %s\n", msg)
}

func fatalf(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "  FAIL %s\n", fmt.Sprintf(msg, args...))
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

// ─── 2. 脱敏复制 ───────────────────────────────────────────

func createSanitizedCopy(root string, cfg *Config) (string, error) {
	buildDir := filepath.Join(root, ".pair", "build-tmp")
	os.RemoveAll(buildDir)

	skipSet := make(map[string]bool)
	for _, s := range cfg.Sanitize.SkipFiles {
		skipSet[filepath.ToSlash(s)] = true
	}

	err := filepath.Walk(root, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, srcPath)
		relSlash := filepath.ToSlash(rel)
		if relSlash == "." {
			return nil
		}
		for skip := range skipSet {
			if relSlash == skip || strings.HasPrefix(relSlash, skip+"/") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		dstPath := filepath.Join(buildDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		ext := filepath.Ext(info.Name())
		isTextFile := ext == ".go" || ext == ".mod" || ext == ".sum" || ext == ".json" ||
			ext == ".yaml" || ext == ".yml" || ext == ".md" || ext == ".txt" ||
			ext == ".rc" || ext == ".bat" || ext == ".ps1" || ext == ".sh" ||
			ext == ".vue" || ext == ".js" || ext == ".ts" || ext == ".css" || ext == ".html"

		if !isTextFile {
			return copyFile(dstPath, srcPath)
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		content := string(data)
		content = strings.ReplaceAll(content, cfg.Sanitize.OldModule, cfg.Sanitize.NewModule)
		for _, pair := range cfg.Sanitize.Replacements {
			content = strings.ReplaceAll(content, pair[0], pair[1])
		}
		return os.WriteFile(dstPath, []byte(content), info.Mode())
	})

	if err == nil {
		// 重建 go.sum（模块路径变更后旧 sum 失效）
		if !cfg.SkipGoModTidy {
			tidy := exec.Command(cfg.Tools.Go, "mod", "tidy")
			tidy.Dir = buildDir
			tidy.Stdout = os.Stdout
			tidy.Stderr = os.Stderr
			if tidyErr := tidy.Run(); tidyErr != nil {
				fmt.Printf("  WARNING go mod tidy 失败（继续构建）: %v\n", tidyErr)
			}
		}
	}
	return buildDir, err
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

// ─── 3. 生成图标 ───────────────────────────────────────────

func generateIcon(root, buildDir string, cfg *Config) error {
	if cfg.Icon == "" {
		return nil
	}
	iconSrc := filepath.Join(root, cfg.Icon)
	if _, err := os.Stat(iconSrc); os.IsNotExist(err) {
		return fmt.Errorf("图标文件不存在: %s", iconSrc)
	}
	if _, err := exec.LookPath(cfg.Tools.FFmpeg); err != nil {
		return fmt.Errorf("ffmpeg 未安装，跳过图标生成")
	}

	sizes := []int{16, 32, 48, 64, 128, 256}
	pngFiles := make([]string, 0, len(sizes))
	for _, s := range sizes {
		out := filepath.Join(buildDir, cfg.FrontendDir, fmt.Sprintf("icon_%d.png", s))
		cmd := exec.Command(cfg.Tools.FFmpeg, "-y", "-i", iconSrc, "-s", fmt.Sprintf("%dx%d", s, s), out)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("缩放图标 %dpx 失败: %v\n%s", s, err, string(output))
		}
		pngFiles = append(pngFiles, out)
	}

	icoPath := filepath.Join(buildDir, cfg.FrontendDir, "icon.ico")
	packICO(icoPath, pngFiles)

	dstIco := filepath.Join(buildDir, filepath.Dir(cfg.MainPkg), "icon.ico")
	if err := copyFile(dstIco, icoPath); err != nil {
		return err
	}
	fmt.Printf("  图标: %s (%d sizes)\n", icoPath, len(sizes))
	return nil
}

func packICO(path string, pngFiles []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	type entry struct {
		w, h  byte
		size  uint32
		off   uint32
	}
	entries := make([]entry, len(pngFiles))
	offset := uint32(6 + len(pngFiles)*16)

	for i, pngPath := range pngFiles {
		data, err := os.ReadFile(pngPath)
		if err != nil {
			return err
		}
		var w, h int
		fmt.Sscanf(filepath.Base(pngPath), "icon_%d.png", &w)
		h = w
		ww, hh := byte(w), byte(h)
		if w > 255 {
			ww, hh = 0, 0
		}
		entries[i] = entry{ww, hh, uint32(len(data)), offset}
		offset += uint32(len(data))
		if _, err := f.Write(data); err != nil {
			return err
		}
	}

	// 写 header + directory
	header := []byte{0, 0, 1, 0, byte(len(pngFiles)), 0}
	f.Seek(0, 0)
	f.Write(header)
	for _, e := range entries {
		dir := []byte{
			e.w, e.h, 0, 0,
			1, 0, // planes
			32, 0, // bpp
			byte(e.size), byte(e.size >> 8), byte(e.size >> 16), byte(e.size >> 24),
			byte(e.off), byte(e.off >> 8), byte(e.off >> 16), byte(e.off >> 24),
		}
		f.Write(dir)
	}
	return nil
}

// ─── 4. 编译资源 ───────────────────────────────────────────

func compileResource(root, buildDir string, cfg *Config) error {
	rcDir := filepath.Join(buildDir, filepath.Dir(cfg.MainPkg))
	rcFile := filepath.Join(rcDir, "companion.rc")
	if _, err := os.Stat(rcFile); os.IsNotExist(err) {
		return nil
	}

	// 更新版本号
	rcData, err := os.ReadFile(rcFile)
	if err != nil {
		return err
	}
	verComma := strings.ReplaceAll(cfg.Version, ".", ",")
	lines := strings.Split(string(rcData), "\n")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "FILEVERSION"):
			lines[i] = "FILEVERSION     " + verComma
		case strings.HasPrefix(t, "PRODUCTVERSION"):
			lines[i] = "PRODUCTVERSION  " + verComma
		case strings.Contains(t, `"FileVersion"`):
			lines[i] = fmt.Sprintf(`            VALUE "FileVersion",        "%s"`, cfg.Version)
		case strings.Contains(t, `"ProductVersion"`):
			lines[i] = fmt.Sprintf(`            VALUE "ProductVersion",     "%s"`, cfg.Version)
		case strings.Contains(t, `"FileDescription"`) && cfg.Description != "":
			lines[i] = fmt.Sprintf(`            VALUE "FileDescription",    "%s"`, cfg.Description)
		case strings.Contains(t, `"CompanyName"`) && cfg.Company != "":
			lines[i] = fmt.Sprintf(`            VALUE "CompanyName",        "%s"`, cfg.Company)
		case strings.Contains(t, `"LegalCopyright"`) && cfg.Copyright != "":
			lines[i] = fmt.Sprintf(`            VALUE "LegalCopyright",     "%s"`, cfg.Copyright)
		case strings.Contains(t, `"ProductName"`) && cfg.AppName != "":
			lines[i] = fmt.Sprintf(`            VALUE "ProductName",        "%s"`, cfg.AppName)
		}
	}
	os.WriteFile(rcFile, []byte(strings.Join(lines, "\n")), 0644)

	sysoFile := filepath.Join(rcDir, "companion.syso")
	cmd := exec.Command(cfg.Tools.Windres, "-i", rcFile, "-o", sysoFile, "-O", "coff")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("windres 失败: %v\n%s", err, string(output))
	}
	return nil
}

// ─── 5. 编译 Go ────────────────────────────────────────────

func buildGo(root, buildDir string, cfg *Config) (string, error) {
	ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
	if runtime.GOOS == "windows" {
		ldflags += " -H windowsgui"
	}

	outputPath := filepath.Join(root, "release", cfg.Output)
	os.MkdirAll(filepath.Dir(outputPath), 0755)

	cmd := exec.Command(cfg.Tools.Go, "build",
		"-ldflags="+ldflags,
		"-o", outputPath,
		filepath.ToSlash(cfg.MainPkg),
	)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()
	env = append(env, "CGO_ENABLED=1")
	cmd.Env = env
	return outputPath, cmd.Run()
}

// ─── 6. 打包发布目录 ───────────────────────────────────────

func packageDist(root, exePath string, cfg *Config) (string, string, error) {
	distDir := filepath.Join(root, cfg.Dist.OutputDir, cfg.Dist.DirName)
	os.RemoveAll(distDir)
	os.MkdirAll(distDir, 0755)

	exeName := filepath.Base(cfg.Output)
	if err := copyFile(filepath.Join(distDir, exeName), exePath); err != nil {
		return "", "", fmt.Errorf("复制主程序失败: %w", err)
	}

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
			if err := copyFile(dstPath, srcPath); err != nil {
				return "", "", fmt.Errorf("复制失败 %s: %w", entry.Src, err)
			}
		}
	}

	zipName := fmt.Sprintf("%s-%s.zip", cfg.Dist.DirName, cfg.Version)
	zipPath := filepath.Join(root, cfg.Dist.OutputDir, zipName)
	os.Remove(zipPath)

	f, err := os.Create(zipPath)
	if err != nil {
		return distDir, "", err
	}
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
