// PairCode Packager — 通用打包执行引擎
//
// 全配置化设计：所有步骤由 packager.json 定义，引擎只做变量替换和顺序执行。
// 特殊步骤 type=build-go / type=package 由引擎内置逻辑处理，其余作为 shell 命令。
//
// 安全设计：
//   - 使用简单字符串替换（strings.ReplaceAll），无模板引擎表达式执行风险
//   - 工作目录始终限于项目根下（filepath.Join 防止路径逃逸）
//   - 命令由项目用户在 packager.json 中定义（用户本来就能在本机执行任意命令）
//
// 用法:
//   go run ./scripts/packager               完整打包
//   go run ./scripts/packager --bump patch  自动递增版本号
//   go run ./scripts/packager --step package 只执行某一步骤

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
	"time"
)

// ─── 配置结构（全配置化）─────────────────────────────────

type PackagerConfig struct {
	Version     string            `json:"version"`
	AppName     string            `json:"appName"`
	Description string            `json:"description,omitempty"`
	Company     string            `json:"company,omitempty"`
	Copyright   string            `json:"copyright,omitempty"`
	Vars        map[string]string `json:"vars,omitempty"`    // 自定义变量 {{varName}}
	Pipeline    []StepDef         `json:"pipeline"`           // 步骤定义（顺序执行）
	Dist        *DistConfig       `json:"dist,omitempty"`

	// 密钥脱敏：打包前从指定 JSON 文件中清除密钥字段
	StripSecrets *StripSecretsConfig `json:"stripSecrets,omitempty"`
}

type StepDef struct {
	Name        string            `json:"name"`                  // 步骤名
	Command     string            `json:"command,omitempty"`     // shell 命令（支持 {{key}} 变量）
	Dir         string            `json:"dir,omitempty"`         // 工作目录（相对项目根）
	OS          string            `json:"os,omitempty"`          // 平台过滤："windows"/"linux"/"darwin"
	IgnoreError bool              `json:"ignoreError,omitempty"` // true=失败继续
	Env         map[string]string `json:"env,omitempty"`         // 环境变量
	Type        string            `json:"type,omitempty"`        // "build-go" | "package"（内置逻辑）
}

type DistConfig struct {
	OutputDir string      `json:"outputDir"`         // 发布根目录（默认 "release"）
	DirName   string      `json:"dirName"`           // 包内目录名
	Format    string      `json:"format,omitempty"`  // "zip"（默认）或 "dir"
	Include   []DistEntry `json:"include,omitempty"`
}

type DistEntry struct {
	Src       string `json:"src"`
	Dst       string `json:"dst"`
	Optional  bool   `json:"optional,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

type StripSecretsConfig struct {
	Fields []string `json:"fields,omitempty"`
	Files  []string `json:"files,omitempty"`
}

// ─── 模板变量 ─────────────────────────────────────────────

// Vars 所有可用于命令中的 {{key}} 替换变量。
// 不使用 text/template 执行，仅做字符串替换，无表达式注入风险。
type Vars struct {
	Version     string
	AppName     string
	Description string
	Company     string
	Copyright   string
	Timestamp   string
	Date        string
	Year        string
	Root        string
	OS          string
	Arch        string

	// 来自配置 vars 段的自定义变量
	custom map[string]string
}

func (v *Vars) resolve(tmpl string) string {
	s := tmpl
	pairs := map[string]string{
		"{{version}}": v.Version, "{{appName}}": v.AppName,
		"{{description}}": v.Description, "{{company}}": v.Company,
		"{{copyright}}": v.Copyright, "{{timestamp}}": v.Timestamp,
		"{{date}}": v.Date, "{{year}}": v.Year,
		"{{root}}": v.Root, "{{os}}": v.OS, "{{arch}}": v.Arch,
	}
	for k, val := range pairs {
		s = strings.ReplaceAll(s, k, val)
	}
	for k, val := range v.custom {
		s = strings.ReplaceAll(s, "{{"+k+"}}", val)
	}
	return s
}

// ─── 主流程 ───────────────────────────────────────────────

func main() {
	root, _ := os.Getwd()
	cfgPath := filepath.Join(root, "packager.json")
	bump := ""
	setVersion := ""
	filterStep := ""

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
		case "--step":
			if i+1 < len(os.Args) {
				filterStep = os.Args[i+1]
				i++
			}
		case "-h", "--help":
			printHelp()
			return
		}
	}

	cfg := loadConfig(cfgPath)

	// 版本号处理
	if setVersion != "" {
		cfg.Version = setVersion
	} else if bump != "" {
		oldVer := cfg.Version
		cfg.Version = bumpVersion(cfg.Version, bump)
		fmt.Printf("  bump: %s (%s → %s)\n", bump, oldVer, cfg.Version)
	}
	if setVersion != "" || bump != "" {
		saveConfig(cfgPath, cfg)
		fmt.Println("  版本已更新: packager.json")
	}

	fmt.Printf("Packager: %s v%s\n", cfg.AppName, cfg.Version)
	fmt.Printf("  工作区: %s\n", root)
	fmt.Println(strings.Repeat("─", 60))

	// 准备变量
	vars := buildVars(root, cfg)

	// 执行 pipeline
	total := len(cfg.Pipeline)
	executed := 0
	for i, step := range cfg.Pipeline {
		if filterStep != "" && step.Name != filterStep {
			continue
		}
		if step.OS != "" && step.OS != runtime.GOOS {
			fmt.Printf("  ⏭ [%d/%d] %s (跳过: %s)\n", i+1, total, step.Name, step.OS)
			continue
		}

		label := fmt.Sprintf("%d/%d", i+1, total)
		fmt.Printf("\n  ▶ [%s] %s\n", label, step.Name)
		executed++

		// 工作目录（始终限在项目根下）
		workDir := root
		if step.Dir != "" {
			workDir = filepath.Join(root, step.Dir)
		}

		var err error
		switch step.Type {
		case "build-go":
			err = execBuildGo(root, cfg, vars)
		case "package":
			err = execPackage(root, cfg, vars)
		default:
			command := vars.resolve(step.Command)
			if command != "" {
				err = execShell(command, workDir, step.Env)
			}
		}

		if err != nil {
			if step.IgnoreError {
				fmt.Printf("  ⚠ 失败但已忽略: %v\n", err)
			} else {
				fatalf("[%s] %s 失败: %v", label, step.Name, err)
			}
		} else {
			fmt.Printf("  ✔ [%s] %s 完成\n", label, step.Name)
		}
	}

	if executed == 0 {
		fmt.Println("\n  （无步骤被执行）")
	}
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("打包完成!")
}

// ─── 内置步骤：Go 编译 ───────────────────────────────────

func execBuildGo(root string, cfg *PackagerConfig, vars *Vars) error {
	mainPkg := vars.custom["mainPkg"]
	if mainPkg == "" {
		return fmt.Errorf("vars.mainPkg 未配置")
	}
	outputName := vars.custom["output"]
	if outputName == "" {
		outputName = "app" + exeSuffix()
	}

	ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
	if extra := vars.custom["extraLdflags"]; extra != "" {
		ldflags += " " + extra
	}
	// Windows GUI 应用默认隐藏控制台
	if runtime.GOOS == "windows" && vars.custom["console"] != "true" {
		ldflags += " -H windowsgui"
	}

	outputPath := filepath.Join(root, "release", outputName)
	os.MkdirAll(filepath.Dir(outputPath), 0755)

	args := []string{"build", "-ldflags=" + ldflags, "-o", outputPath}
	if tags := vars.custom["buildTags"]; tags != "" {
		args = append(args, "-tags", tags)
	}
	args = append(args, mainPkg)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = root

	cgo := vars.custom["cgo"]
	if cgo == "" {
		cgo = "0"
	}
	cmd.Env = append(os.Environ(), "CGO_ENABLED="+cgo)

	fmt.Printf("  → go build -ldflags=\"%s\" -o %s %s\n", ldflags, outputPath, mainPkg)
	return cmd.Run()
}

// ─── 内置步骤：打包发布目录 ──────────────────────────────

func execPackage(root string, cfg *PackagerConfig, vars *Vars) error {
	if cfg.Dist == nil {
		return nil
	}
	outputName := vars.custom["output"]
	if outputName == "" {
		outputName = "app" + exeSuffix()
	}
	exePath := filepath.Join(root, "release", outputName)
	distDir := filepath.Join(root, cfg.Dist.OutputDir, cfg.Dist.DirName)

	os.RemoveAll(distDir)
	os.MkdirAll(distDir, 0755)

	// 复制主程序
	if err := copyFile(filepath.Join(distDir, outputName), exePath); err != nil {
		return fmt.Errorf("复制主程序: %w", err)
	}
	fmt.Printf("  → %s\n", outputName)

	// 复制附加资源
	for _, entry := range cfg.Dist.Include {
		srcPath := filepath.Join(root, entry.Src)
		dstPath := filepath.Join(distDir, entry.Dst)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			if entry.Optional {
				continue
			}
			return fmt.Errorf("必需文件缺失: %s", entry.Src)
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
				return fmt.Errorf("复制 %s: %w", entry.Src, err)
			}
		}
		fmt.Printf("  → %s\n", entry.Dst)
	}

	// 密钥脱敏
	if cfg.StripSecrets != nil {
		stripSecrets(distDir, cfg.StripSecrets.Fields, cfg.StripSecrets.Files)
	}

	// ZIP 打包
	if cfg.Dist.Format != "dir" {
		zipName := fmt.Sprintf("%s-%s.zip", cfg.Dist.DirName, cfg.Version)
		zipPath := filepath.Join(root, cfg.Dist.OutputDir, zipName)
		if err := createZIP(zipPath, distDir); err != nil {
			return fmt.Errorf("ZIP 打包: %w", err)
		}
		zi, _ := os.Stat(zipPath)
		fmt.Printf("  ✔ ZIP: %s (%.1f MB)\n", zipName, float64(zi.Size())/1024/1024)
	}

	return nil
}

// ─── 辅助函数 ─────────────────────────────────────────────

func buildVars(root string, cfg *PackagerConfig) *Vars {
	now := time.Now()
	v := &Vars{
		Version:     cfg.Version,
		AppName:     cfg.AppName,
		Description: cfg.Description,
		Company:     cfg.Company,
		Copyright:   cfg.Copyright,
		Timestamp:   now.Format("20060102-150405"),
		Date:        now.Format("20060102"),
		Year:        now.Format("2006"),
		Root:        root,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		custom:      cfg.Vars,
	}
	if v.custom == nil {
		v.custom = map[string]string{}
	}
	return v
}

func execShell(command, dir string, extraEnv map[string]string) error {
	fmt.Printf("  $ %s\n", command)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if len(extraEnv) > 0 {
		env := os.Environ()
		for k, v := range extraEnv {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	return cmd.Run()
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func createZIP(zipPath, srcDir string) error {
	os.Remove(zipPath)
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	defer w.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		wr, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = wr.Write(data)
		return err
	})
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

func stripSecrets(distDir string, fields, files []string) {
	if len(fields) == 0 || len(files) == 0 {
		return
	}
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
			fmt.Printf("  🔒 密钥已清除: %s\n", rel)
		}
	}
}

func loadConfig(path string) *PackagerConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("读取配置失败 %s: %v", path, err)
	}
	var cfg PackagerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fatalf("解析配置失败: %v", err)
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.Dist != nil && cfg.Dist.OutputDir == "" {
		cfg.Dist.OutputDir = "release"
	}
	if cfg.Dist != nil && cfg.Dist.DirName == "" {
		cfg.Dist.DirName = cfg.AppName
	}
	return &cfg
}

func saveConfig(path string, cfg *PackagerConfig) {
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
	default:
		patch++
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func printHelp() {
	fmt.Print(`
Packager — 通用打包执行引擎

全配置化设计：所有步骤由 packager.json 定义，引擎只做变量替换和顺序执行。

用法:
  packager [选项...]

选项:
  --bump <level>    自动递增版本号: patch|minor|major
  --version <x.y.z> 手动指定版本号
  --step <name>     只执行指定步骤
  -h, --help        显示此帮助

配置 (packager.json):

  基础字段:
    version      版本号 (如 "1.0.3")
    appName      应用名
    vars         自定义变量，命令中 {{key}} 引用

  pipeline 步骤数组（每步）:
    name         步骤名
    command      命令模板（含 {{key}} 变量）
    dir          工作目录（相对项目根）
    os           平台过滤: windows/linux/darwin
    ignoreError  true=失败继续
    type         内置类型: "build-go"|"package"
    env          环境变量

  内置类型:
    type=build-go  →  Go 编译（需 vars.mainPkg, vars.output）
    type=package   →  打包发布目录 + ZIP（需 dist 配置）

  dist 发布配置:
    outputDir    发布根目录（默认 release）
    dirName      包内目录名
    format       "zip" 或 "dir"
    include[]    附加文件列表

  stripSecrets 密钥脱敏:
    fields[]     要清除的字段名
    files[]      要处理的 JSON 文件

模板变量: {{version}} {{appName}} {{description}}
  {{company}} {{copyright}} {{timestamp}} {{date}}
  {{year}} {{root}} {{os}} {{arch}}
  以及 vars 中自定义的 {{key}}
`)
}

func fatalf(msg string, a ...any) {
	fmt.Fprintf(os.Stderr, "\n  ✘ "+msg+"\n", a...)
	os.Exit(1)
}
