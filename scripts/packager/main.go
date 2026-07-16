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
	"strconv"
	"strings"
)

type Hooks struct {
	PreBuild     []string `json:"preBuild"`     // 所有步骤之前
	PreFrontend  []string `json:"preFrontend"`  // 构建前端前
	PostFrontend []string `json:"postFrontend"` // 构建前端后
	PostIcon     []string `json:"postIcon"`     // 图标生成后
	PostResource []string `json:"postResource"` // 资源编译后
	PreGoBuild   []string `json:"preGoBuild"`   // Go 编译前
	PostBuild    []string `json:"postBuild"`    // Go 编译完成，打包前
	PostPackage  []string `json:"postPackage"`  // 全部完成后
}

type Config struct {
	Version          string     `json:"version"`
	AppName          string     `json:"appName"`
	Description      string     `json:"description"`
	Company          string     `json:"company"`
	Copyright        string     `json:"copyright"`
	Icon             string     `json:"icon"`
	FrontendDir      string     `json:"frontendDir"`
	SkipBuildFrontend bool      `json:"skipBuildFrontend,omitempty"`
	MainPkg          string     `json:"mainPkg"`
	Output           string     `json:"output"`
	Console          bool       `json:"console,omitempty"`
	Pipeline         []string   `json:"pipeline,omitempty"`
	Hooks            Hooks      `json:"hooks,omitempty"`
	Secrets          SecretsCfg `json:"secrets"`
	Dist             DistConfig `json:"dist"`
	Tools            Tools      `json:"tools"`
}

type SecretsCfg struct {
	Fields []string `json:"fields,omitempty"`
	Files  []string `json:"files,omitempty"`
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

// 预定义的步骤名
const (
	StepFrontend   = "build-frontend"
	StepIcon       = "generate-icon"
	StepResource   = "compile-resource"
	StepGoBuild    = "build-go"
	StepPackage    = "package"
)

// 默认 pipeline（向后兼容）
var defaultPipeline = []string{
	StepFrontend,
	StepIcon,
	StepResource,
	StepGoBuild,
	StepPackage,
}

func main() {
	root, _ := os.Getwd()
	cfgPath := filepath.Join(root, "packager.json")
	bump := ""
	setVersion := ""
	skipFrontend := false

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--bump":
			if i+1 < len(os.Args) {
				bump = os.Args[i+1]
				i++
			}
		case "--version":
			if i+1 < len(os.Args) {
				setVersion = os.Args[i+1]
				i++
			}
		case "--skip-frontend":
			skipFrontend = true
		case "-h", "--help":
			fmt.Println("用法: packager [--bump patch|minor|major] [--version x.y.z] [--skip-frontend]")
			return
		}
	}

	cfg := loadConfig(cfgPath)

	// 版本号迭代
	if setVersion != "" {
		cfg.Version = setVersion
	} else if bump != "" {
		oldVer := cfg.Version
		cfg.Version = bumpVersion(cfg.Version, bump)
		fmt.Printf("  bump: %s (%s → %s)\n", bump, oldVer, cfg.Version)
	}
	if setVersion != "" || bump != "" {
		saveConfig(cfgPath, cfg)
		updateRCVersion(cfg)
		fmt.Println("  版本已更新: packager.json + companion.rc")
	}

	fmt.Printf("PairCode Packager v%s\n", cfg.Version)
	fmt.Printf("  工作区: %s\n", root)
	fmt.Println(strings.Repeat("-", 60))

	// 前置钩子（所有步骤之前）
	runHooks(root, "preBuild", cfg.Hooks.PreBuild)

	// 确定 pipeline（配置了就用配置的，否则用默认）
	pipeline := cfg.Pipeline
	if len(pipeline) == 0 {
		pipeline = defaultPipeline
	}

	total := len(pipeline)
	for i, stepName := range pipeline {
		stepLabel := fmt.Sprintf("%d/%d", i+1, total)

		switch stepName {
		case StepFrontend:
			runHooks(root, "preFrontend", cfg.Hooks.PreFrontend)
			if !cfg.SkipBuildFrontend && !skipFrontend {
				step(stepLabel, "构建前端...")
				if err := buildFrontend(root, cfg); err != nil {
					fatalf("前端构建失败: %v", err)
				}
				ok("前端构建完成")
			} else {
				fmt.Println("  跳过前端构建")
			}
			runHooks(root, "postFrontend", cfg.Hooks.PostFrontend)

		case StepIcon:
			step(stepLabel, "生成 Windows 图标...")
			if err := generateIcon(root, cfg); err != nil {
				fmt.Printf("  图标生成失败（可跳过）: %v\n", err)
			} else {
				ok("图标生成完成")
			}
			runHooks(root, "postIcon", cfg.Hooks.PostIcon)

		case StepResource:
			step(stepLabel, "编译资源文件...")
			if err := compileResource(cfg); err != nil {
				fmt.Printf("  资源编译失败（可跳过）: %v\n", err)
			} else {
				ok("资源编译完成")
			}
			runHooks(root, "postResource", cfg.Hooks.PostResource)

		case StepGoBuild:
			runHooks(root, "preGoBuild", cfg.Hooks.PreGoBuild)
			step(stepLabel, "编译 companion.exe...")
			exePath, err := buildGo(cfg)
			if err != nil {
				fatalf("Go 编译失败: %v", err)
			}
			ok(fmt.Sprintf("编译完成: %s", exePath))
			runHooks(root, "postBuild", cfg.Hooks.PostBuild)

		case StepPackage:
			distDir, zipPath, err := packageDist(root, cfg)
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

		default:
			// 未知的步骤名当作自定义命令执行
			runHooks(root, stepName, []string{stepName})
		}
	}

	// 后置钩子（全部完成后）
	runHooks(root, "postPackage", cfg.Hooks.PostPackage)

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
	if len(cfg.Pipeline) == 0 {
		cfg.Pipeline = defaultPipeline
	}
	return &cfg
}

func saveConfig(path string, cfg *Config) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(path, data, 0644)
}

func bumpVersion(current, level string) string {
	parts := strings.SplitN(current, ".", 3)
	if len(parts) < 3 {
		return "1.0.0"
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	switch level {
	case "major":
		major++
		minor, patch = 0, 0
	case "minor":
		minor++
		patch = 0
	default: // patch
		patch++
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func updateRCVersion(cfg *Config) {
	rcFile := filepath.Join(filepath.Dir(cfg.MainPkg), "companion.rc")
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return
	}
	verComma := strings.ReplaceAll(cfg.Version, ".", ",")
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "FILEVERSION") {
			lines[i] = "FILEVERSION     " + verComma
		} else if strings.HasPrefix(t, "PRODUCTVERSION") {
			lines[i] = "PRODUCTVERSION  " + verComma
		} else if strings.Contains(t, `VALUE "FileVersion"`) {
			lines[i] = fmt.Sprintf(`            VALUE "FileVersion",        "%s"`, cfg.Version)
		} else if strings.Contains(t, `VALUE "ProductVersion"`) {
			lines[i] = fmt.Sprintf(`            VALUE "ProductVersion",     "%s"`, cfg.Version)
		}
	}
	os.WriteFile(rcFile, []byte(strings.Join(lines, "\n")), 0644)
}

// ─── 钩子执行 ────────────────────────────────────────────

func runHooks(root, phase string, cmds []string) {
	if len(cmds) == 0 {
		return
	}
	step("⚡", fmt.Sprintf("执行钩子 [%s]...", phase))
	for _, cmdStr := range cmds {
		fmt.Printf("  $ %s\n", cmdStr)
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", cmdStr)
		} else {
			cmd = exec.Command("sh", "-c", cmdStr)
		}
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fatalf("钩子 [%s] 失败: %v", phase, err)
		}
	}
	ok(fmt.Sprintf("钩子 [%s] 完成 (%d 条)", phase, len(cmds)))
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
	if runtime.GOOS == "windows" && !cfg.Console {
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

func packageDist(root string, cfg *Config) (string, string, error) {
	exePath := filepath.Join(root, "release", cfg.Output)
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
