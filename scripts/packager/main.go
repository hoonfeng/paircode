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
	Vars        map[string]string `json:"vars,omitempty"` // 自定义变量 {{varName}}
	Pipeline    []StepDef         `json:"pipeline"`       // 步骤定义（顺序执行）
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

	// 以下字段供 build-go / build-go-gui 覆盖 vars 中的同名配置。
	Pkg    string `json:"pkg,omitempty"`    // 主包（覆盖 vars.mainPkg）
	Output string `json:"output,omitempty"` // 产物名（覆盖 vars.output）
	Tags   string `json:"tags,omitempty"`   // -tags（build-go-gui 用，如 "gui"）
	GoWork string `json:"gowork,omitempty"` // GOWORK 文件（build-go-gui 用，如 "go.work.gui"）
}

type DistConfig struct {
	OutputDir string      `json:"outputDir"`        // 发布根目录（默认 "release"）
	DirName   string      `json:"dirName"`          // 包内目录名
	Format    string      `json:"format,omitempty"` // "zip"（默认）或 "dir"
	Include   []DistEntry `json:"include,omitempty"`
	// Editions 发行版拆分（web / gui …）：每个 edition 独立产出「平台 × 版本」zip。
	// 为空时退回旧的单包逻辑（dirName + vars.output），保持向后兼容。
	Editions []DistEdition `json:"editions,omitempty"`
}

// DistEdition 一种独立发行版（如 web = 纯服务端、gui = 桌面单文件）。
//
// 拆分的直接原因：旧逻辑把「GUI 单文件启动器」与「pair-server」同时塞进
// 同一个 zip（两个 exe 互相重复约 40MB），且发布件无法区分用途。
// 现在每个 edition 只放自己的主程序与所需资源：
//   - web: release/pair-server-* → 包内 pair-server[.exe]
//   - gui: release/PairCode[*]   → 包内 PairCode[.exe]（单文件启动器）
type DistEdition struct {
	Name       string      `json:"name"`                 // web / gui（用于日志与校验）
	DirName    string      `json:"dirName"`              // 包内顶层目录 & zip 名前缀
	MainSrc    string      `json:"mainSrc"`              // release/ 下主程序基名（按目标平台加后缀）
	MainDst    string      `json:"mainDst"`              // 包内主程序基名（windows 自动补 .exe）
	NativeOnly bool        `json:"nativeOnly,omitempty"` // true = 只打本机平台（GUI 需 cgo）
	Include    []DistEntry `json:"include,omitempty"`    // 该 edition 专属资源（与 dist.include 合并）
}

type DistEntry struct {
	Src       string   `json:"src"`
	Dst       string   `json:"dst"`
	Optional  bool     `json:"optional,omitempty"`
	Recursive bool     `json:"recursive,omitempty"`
	OS        string   `json:"os,omitempty"`      // 平台过滤："windows"/"linux"/"darwin"（空=全部平台）
	Exclude   []string `json:"exclude,omitempty"` // 相对 Src 的路径（目录/文件），递归拷贝时跳过
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
	targetsOverride := ""
	versionOnly := false

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
		case "--targets":
			if i+1 < len(os.Args) {
				targetsOverride = os.Args[i+1]
				i++
			}
		case "--version-only":
			versionOnly = true
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
	if versionOnly {
		// 只改版本号，不执行任何 pipeline 步骤（CI 的 tag 版本对齐用）。
		fmt.Printf("  仅版本处理完成: %s v%s\n", cfg.AppName, cfg.Version)
		return
	}

	fmt.Printf("Packager: %s v%s\n", cfg.AppName, cfg.Version)
	fmt.Printf("  工作区: %s\n", root)
	fmt.Println(strings.Repeat("─", 60))

	// 准备变量
	vars := buildVars(root, cfg)

	// --targets 覆盖（如: --targets windows/amd64,linux/amd64）
	if targetsOverride != "" {
		vars.custom["targets"] = targetsOverride
	}

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
			err = execBuildGo(root, step, cfg, vars)
		case "build-go-gui":
			err = execBuildGoGUI(root, step, cfg, vars)
		case "build-stub":
			err = execBuildStub(root, step, cfg)
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

// ─── 目标平台 ────────────────────────────────────────────

// parseTargets 解析 vars.targets（"windows/amd64,linux/amd64,darwin/arm64"）；
// 空 → 当前平台单目标。
func parseTargets(spec string) [][2]string {
	if strings.TrimSpace(spec) == "" {
		return [][2]string{{runtime.GOOS, runtime.GOARCH}}
	}
	var out [][2]string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		gg := strings.SplitN(part, "/", 2)
		goos, goarch := gg[0], "amd64"
		if len(gg) > 1 && gg[1] != "" {
			goarch = gg[1]
		}
		out = append(out, [2]string{goos, goarch})
	}
	return out
}

// targetExeName 按目标平台生成产物名：windows 保持 output（pair.exe）；
// 其他平台 pair-linux-amd64 / pair-darwin-arm64。
func targetExeName(output, goos, goarch string) string {
	if goos == "windows" {
		return output
	}
	base := strings.TrimSuffix(output, ".exe")
	return base + "-" + goos + "-" + goarch
}

// ─── 内置步骤：Go 编译 ───────────────────────────────────

func execBuildGo(root string, step StepDef, cfg *PackagerConfig, vars *Vars) error {
	mainPkg := vars.custom["mainPkg"]
	if step.Pkg != "" {
		mainPkg = step.Pkg
	}
	if mainPkg == "" {
		return fmt.Errorf("vars.mainPkg 未配置")
	}
	outputName := vars.custom["output"]
	if step.Output != "" {
		outputName = step.Output
	}
	if outputName == "" {
		outputName = "app" + exeSuffix()
	}
	targets := parseTargets(vars.custom["targets"])
	if len(targets) == 0 {
		return fmt.Errorf("vars.targets 解析为空")
	}

	for _, t := range targets {
		goos, goarch := t[0], t[1]
		local := goos == runtime.GOOS && goarch == runtime.GOARCH
		ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
		if extra := vars.custom["extraLdflags"]; extra != "" {
			ldflags += " " + extra
		}
		// Windows GUI 应用默认隐藏控制台（仅本机 Windows 编译）
		if local && goos == "windows" && vars.custom["console"] != "true" {
			ldflags += " -H windowsgui"
		}

		outName := targetExeName(outputName, goos, goarch)
		outputPath := filepath.Join(root, "release", outName)
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
		if !local {
			cgo = "0" // 交叉编译无 cgo 工具链（Linux/mac 目标语义搜索降级 noop）
		}
		env := []string{"GOOS=" + goos, "GOARCH=" + goarch, "CGO_ENABLED=" + cgo}
		cmd.Env = append(os.Environ(), env...)

		fmt.Printf("  → go build [%s/%s cgo=%s] -ldflags=\"%s\" -o %s %s\n", goos, goarch, cgo, ldflags, outName, mainPkg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("编译 %s/%s 失败: %w", goos, goarch, err)
		}
	}
	return nil
}

// ─── 内置步骤：打包发布目录 ──────────────────────────────

// execBuildGoGUI 构建带 GUI 的主程序（-tags gui）。
//
// 与 execBuildGo 的区别：
//   - 只在**本机平台**构建：GUI 依赖 cgo（Skia/GLFW/WebKit），交叉编译需要
//     目标平台的 C 工具链与系统头文件（Linux 的 X11、macOS 的 SDK），无法在
//     Windows 上一键产出其它平台产物。其它平台请在对应系统（或 CI 原生
//     runner）上运行本程序，各自产出本平台的发布包。
//   - CGO_ENABLED=1（GUI 渲染必须）。
//   - 可指定 GOWORK（如 go.work.gui：把兄弟仓库 wb-ui 纳入 workspace）。
func execBuildGoGUI(root string, step StepDef, cfg *PackagerConfig, vars *Vars) error {
	pkg := step.Pkg
	if pkg == "" {
		pkg = vars.custom["guiMainPkg"]
	}
	if pkg == "" {
		pkg = vars.custom["mainPkg"]
	}
	if pkg == "" {
		return fmt.Errorf("build-go-gui: pkg 未配置")
	}
	outputName := step.Output
	if outputName == "" {
		outputName = vars.custom["output"]
	}
	if outputName == "" {
		outputName = "pair.exe"
	}
	tags := step.Tags
	if tags == "" {
		tags = vars.custom["guiTags"]
	}
	if tags == "" {
		tags = "gui"
	}

	goos, goarch := runtime.GOOS, runtime.GOARCH
	outName := targetExeName(outputName, goos, goarch)
	outPath := filepath.Join(root, "release", outName)

	// 清理历史遗留的「非本机平台」GUI 产物：GUI 无法交叉编译，这些文件只可能
	// 来自旧版本构建；留在 release/ 会被 package 步骤误打进其它平台的包
	// （历史事故：v1.6.20 本地打包时把 Oct 17 的旧 pair-linux-amd64 塞进了
	// Linux 包）。目标平台各自原生构建时，本机产物不受影响。
	for _, t := range [][2]string{{"windows", "amd64"}, {"linux", "amd64"}, {"darwin", "arm64"}} {
		if t[0] == goos && t[1] == goarch {
			continue
		}
		_ = os.Remove(filepath.Join(root, "release", targetExeName(outputName, t[0], t[1])))
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
	if goos == "windows" {
		// GUI 子系统：双击启动不弹控制台窗口（服务版 pair-server.exe 保留控制台）。
		ldflags += " -H windowsgui"
	}
	if extra := vars.custom["extraLdflags"]; extra != "" {
		ldflags += " " + extra
	}
	args := []string{"build", "-tags", tags, "-ldflags=" + ldflags, "-o", outPath, pkg}
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := append(os.Environ(), "CGO_ENABLED=1", "GOOS="+goos, "GOARCH="+goarch)
	if step.GoWork != "" {
		p := step.GoWork
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		env = append(env, "GOWORK="+p)
	}
	cmd.Env = env

	fmt.Printf("  → go build -tags %s [%s/%s cgo=1] -o %s %s\n", tags, goos, goarch, outName, pkg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("编译 GUI 主程序 %s/%s 失败: %w", goos, goarch, err)
	}
	return nil
}

// execBuildStub 构建单文件自解压启动器（-tags stub）。
//
// 把本机平台的 GUI 主程序与 Skia 原生库复制到 cmd/paircode-stub/payload/，
// 再编译成 release/PairCode[.exe]。发布包因此只需一个可执行文件：首次运行
// 由 stub 释放到用户级 runtime 目录，之后复用；Windows PE 在进程启动前解析
// 导入表，所以解压只能在不通��� Skia 的外层进程里完成。
func execBuildStub(root string, step StepDef, cfg *PackagerConfig) error {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	payloadDir := filepath.Join(root, step.Dir)
	if step.Dir == "" {
		payloadDir = filepath.Join(root, "cmd", "paircode-stub", "payload")
	}
	mainInPayload := "pair"
	if goos == "windows" {
		mainInPayload = "pair.exe"
	}
	skiaLib := map[string]string{
		"windows": "libSkiaSharp.dll",
		"linux":   "libSkiaSharp.so",
		"darwin":  "libSkiaSharp.dylib",
	}[goos]
	if skiaLib == "" {
		return fmt.Errorf("build-stub: 不支持的平台 %s/%s", goos, goarch)
	}
	// 每次重建 payload：避免上一次构建的平台产物/已删除插件残留进 embed。
	if err := os.RemoveAll(payloadDir); err != nil {
		return err
	}
	if err := os.MkdirAll(payloadDir, 0o755); err != nil {
		return err
	}
	mainSrc := filepath.Join(root, "release", targetExeName("pair.exe", goos, goarch))
	if err := copyFile(filepath.Join(payloadDir, mainInPayload), mainSrc); err != nil {
		return fmt.Errorf("build-stub: 复制 GUI 主程序（需先执行 build-go-gui）: %w", err)
	}
	// Linux 的 DT_NEEDED 记录的是库的 SONAME（如 libSkiaSharp.so.119.0.0），而
	// goskia module 里只有不带版本号的 libSkiaSharp.so——stage-gui-runtime 已把
	// SONAME 别名一并放进 release/。这里按前缀把所有 libSkiaSharp* 带进 payload，
	// 否则解压后的 GUI 会因找不到 SONAME 对应文件而启动失败。
	libs, _ := filepath.Glob(filepath.Join(root, "release", "libSkiaSharp*"))
	if len(libs) == 0 {
		return fmt.Errorf("build-stub: 复制 Skia 原生库（需先执行 stage-gui-runtime）: %s 不存在",
			filepath.Join(root, "release", skiaLib))
	}
	for _, src := range libs {
		if err := copyFile(filepath.Join(payloadDir, filepath.Base(src)), src); err != nil {
			return fmt.Errorf("build-stub: 复制 Skia 原生库 %s: %w", filepath.Base(src), err)
		}
	}
	// 插件与前端资源是 IDE 界面的组成部分：只释放主程序+Skia 库时，插件宿主为 0、
	// 侧边栏与面板全部显示「未装配」。随 stub 一起内嵌，空目录首次启动即可完整。
	for _, res := range []string{".pair/plugins", ".pair/assets", "plugins-src/ui-app"} {
		src := filepath.Join(root, filepath.FromSlash(res))
		if _, err := os.Stat(src); err != nil {
			continue // 资源目录可缺省（精简仓库）
		}
		dst := filepath.Join(payloadDir, filepath.FromSlash(res))
		entry := DistEntry{Src: res, Dst: res, Recursive: true}
		if err := copyDistEntry(src, dst, entry); err != nil {
			return fmt.Errorf("build-stub: 复制 %s: %w", res, err)
		}
	}

	// 与 package 步骤的 targetExeName 保持一致：windows 为 PairCode.exe，
	// 其它平台带平台后缀（PairCode-linux-amd64 / PairCode-darwin-arm64）。
	outName := targetExeName("PairCode.exe", goos, goarch)
	outPath := filepath.Join(root, "release", outName)
	ldflags := fmt.Sprintf("-s -w -X main.version=%s", cfg.Version)
	if goos == "windows" {
		// 单文件启动器同样走 GUI 子系统，双击不弹控制台。
		ldflags += " -H windowsgui"
	}
	fmt.Printf("  → go build -tags stub [%s/%s] -o %s ./cmd/paircode-stub\n", goos, goarch, outName)
	cmd := exec.Command("go", "build", "-tags", "stub", "-ldflags="+ldflags, "-o", outPath, "./cmd/paircode-stub")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// stub 是纯 Go（无 cgo），不依赖 wb-ui，显式 GOWORK=off 避免误用默认 workspace。
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("编译单文件启动器失败: %w", err)
	}
	return nil
}

func execPackage(root string, cfg *PackagerConfig, vars *Vars) error {
	if cfg.Dist == nil {
		return nil
	}
	targets := parseTargets(vars.custom["targets"])
	if len(targets) == 0 {
		return fmt.Errorf("vars.targets 解析为空")
	}

	editions := cfg.Dist.Editions
	if len(editions) == 0 {
		// 向后兼容：旧配置没有 editions 时，维持「一个包 = vars.output 主程序 + include」。
		outputName := vars.custom["output"]
		if outputName == "" {
			outputName = "app" + exeSuffix()
		}
		editions = []DistEdition{{
			Name:    "default",
			DirName: cfg.Dist.DirName,
			MainSrc: outputName,
			MainDst: outputName,
			Include: cfg.Dist.Include,
		}}
	}

	for _, ed := range editions {
		for _, t := range targets {
			if err := packageEdition(root, cfg, ed, t[0], t[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

// packageEdition 打一个「发行版 × 目标平台」的包。
//
// 主程序按 edition.MainSrc 在 release/ 下解析（带平台后缀），包内改名为
// edition.MainDst[.exe]；资源 = dist.include（公共）+ edition.include（专属）。
func packageEdition(root string, cfg *PackagerConfig, ed DistEdition, goos, goarch string) error {
	// GUI 主程序只在本机平台构建（交叉编译 cgo 需要目标平台的 C 工具链与
	// 系统头文件）。edition 声明 nativeOnly 时，非本机目标直接跳过。
	if ed.NativeOnly && !(goos == runtime.GOOS && goarch == runtime.GOARCH) {
		fmt.Printf("  ⏭ [%s/%s] %s 需原生构建（cgo），跳过\n", goos, goarch, ed.Name)
		return nil
	}

	mainSrcName := targetExeName(ed.MainSrc, goos, goarch)
	mainSrcPath := filepath.Join(root, "release", mainSrcName)
	if _, statErr := os.Stat(mainSrcPath); statErr != nil {
		fmt.Printf("  ⏭ [%s/%s] %s: release/%s 不存在，跳过\n", goos, goarch, ed.Name, mainSrcName)
		return nil
	}

	dirName := ed.DirName
	if dirName == "" {
		dirName = cfg.Dist.DirName
	}
	distDir := filepath.Join(root, cfg.Dist.OutputDir, dirName)
	os.RemoveAll(distDir)
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return err
	}

	// 包内主程序名：windows 补 .exe，其它平台保持基名。
	mainDstName := ed.MainDst
	if mainDstName == "" {
		mainDstName = ed.MainSrc
	}
	if goos == "windows" && !strings.HasSuffix(strings.ToLower(mainDstName), ".exe") {
		mainDstName += ".exe"
	}
	if goos != "windows" {
		mainDstName = strings.TrimSuffix(mainDstName, ".exe")
	}
	if err := copyFile(filepath.Join(distDir, mainDstName), mainSrcPath); err != nil {
		return fmt.Errorf("复制主程序 %s: %w", mainSrcName, err)
	}
	fmt.Printf("  → [%s/%s] %s → %s\n", goos, goarch, mainSrcName, mainDstName)

	entries := make([]DistEntry, 0, len(cfg.Dist.Include)+len(ed.Include))
	entries = append(entries, cfg.Dist.Include...)
	entries = append(entries, ed.Include...)
	for _, entry := range entries {
		if entry.OS != "" && entry.OS != goos {
			continue // 非本平台条目跳过
		}
		srcPath := filepath.Join(root, entry.Src)
		dstPath := filepath.Join(distDir, entry.Dst)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			if entry.Optional {
				continue
			}
			return fmt.Errorf("必需文件缺失: %s", entry.Src)
		}
		if err := copyDistEntry(srcPath, dstPath, entry); err != nil {
			return fmt.Errorf("复制 %s: %w", entry.Src, err)
		}
		fmt.Printf("  → %s\n", entry.Dst)
	}

	if cfg.StripSecrets != nil {
		stripSecrets(distDir, cfg.StripSecrets.Fields, cfg.StripSecrets.Files)
	}

	if cfg.Dist.Format != "dir" {
		// 命名：<dirName>-<version>-<goos>.zip（web/gui 与平台一目了然）
		zipName := fmt.Sprintf("%s-%s-%s.zip", dirName, cfg.Version, goos)
		zipPath := filepath.Join(root, cfg.Dist.OutputDir, zipName)
		if err := createZIP(zipPath, distDir); err != nil {
			return fmt.Errorf("ZIP 打包 %s: %w", zipName, err)
		}
		zi, _ := os.Stat(zipPath)
		fmt.Printf("  ✔ ZIP: %s (%.1f MB)\n", zipName, float64(zi.Size())/1024/1024)
	}
	return nil
}

// distIgnoredNames 打包硬编码忽略集——任意层级生效。
//
// 为什么不只靠 DistEntry.Exclude：exclude 只做「相对 Src 的前缀匹配」
// （见下方 copyDistEntry），无法命中 tool-flow/.smoke 这类嵌套路径；
// packager.json 里给 .pair/plugins 加 exclude 也过滤不到，历史事故：
// v1.6.18 三平台包各混入 98 个 tool-flow/.smoke/**（含 26 个嵌套 .git
// 与任务快照）。故在遍历层硬编码，与配置无关、对所有 dist.include 生效。
//
//	.smoke       插件测试夹具（smoke-test.js 以 __dirname/.smoke 落地，
//	             内含嵌套 .git、.pair/tasks 测试快照与运行时锁）
//	.git         VCS 元数据（插件目录内不应出现）
//	node_modules 依赖目录/junction
var distIgnoredNames = map[string]bool{
	".smoke":       true,
	".git":         true,
	"node_modules": true,
}

// copyDistEntry 按 DistEntry 配置复制单个资源（递归目录 / 单文件）。
func copyDistEntry(srcPath, dstPath string, entry DistEntry) error {
	if entry.Recursive {
		return filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if info != nil && info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// ★ 硬编码忽略集：任意层级跳过 .smoke/.git/node_modules
			//   （不依赖 packager.json 的 Src 前缀 exclude）
			if distIgnoredNames[info.Name()] {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// ★ 符号链接/junction 不复制（Windows junction 无文件内容，
			//   copyFile 会失败并中断整个 Walk）
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			rel, _ := filepath.Rel(srcPath, path)
			if rel == "." {
				return nil
			}
			for _, ex := range entry.Exclude {
				if rel == ex || strings.HasPrefix(rel, ex+string(filepath.Separator)) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			target := filepath.Join(dstPath, rel)
			if info.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			return copyFile(target, path)
		})
	}
	// 单文件入口同样受忽略集约束（防止配置误指向夹具文件）
	if distIgnoredNames[filepath.Base(srcPath)] {
		return nil
	}
	os.MkdirAll(filepath.Dir(dstPath), 0755)
	return copyFile(dstPath, srcPath)
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

// stripSecretsRecursive 递归遍历 JSON 树，删除任意层级匹配敏感字段名的键。
// 返回是否发生变更。
func stripSecretsRecursive(v any, fieldSet map[string]bool) bool {
	changed := false
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if fieldSet[k] {
				// 占位符（<...>）或空值保留——模板提示无害；真实密钥（非空且非占位符）删除
				if s, ok := child.(string); ok {
					if s == "" || (strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">")) {
						continue
					}
				}
				delete(t, k)
				changed = true
				continue
			}
			if stripSecretsRecursive(child, fieldSet) {
				changed = true
			}
		}
	case []any:
		for i := range t {
			if stripSecretsRecursive(t[i], fieldSet) {
				changed = true
			}
		}
	}
	return changed
}

func stripSecrets(distDir string, fields, files []string) {
	if len(fields) == 0 || len(files) == 0 {
		return
	}
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[f] = true
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
		if stripSecretsRecursive(m, fieldSet) {
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
		cfg.Version = "1.0.7"
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
  --version-only    只写版本号到 packager.json，不执行 pipeline（配 --version 用）
  --step <name>     只执行指定步骤
  --targets <spec>  覆盖目标平台（如 --targets windows/amd64,linux/amd64,darwin/arm64）
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
