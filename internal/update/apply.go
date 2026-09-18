// apply.go — 安装目录替换：保护名单过滤、主程序备份/在线替换、失败回滚、重启脚本。
//
// ★ 替换策略（见 docs/online-update-design.md §7）：
//   - 按更新包条目「覆盖式」写入（不做整目录删除，避免误删用户文件）；
//   - 保护名单内的用户数据/配置一律跳过；
//   - 主程序用「同卷 rename 运行中 exe」技巧在线替换（old → 备份，new → 正式名），
//     rename 不可用时降级：先放 pair.exe.new，由重启脚本在进程退出后 move 到位；
//   - 任一环节失败 → 恢复备份主程序（旧版本仍可启动），其余包内资源保持新版本
//     （资源文件新旧混存向后兼容），失败清单写入 last-apply.json 供排查。
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// protectedPrefixes 保护名单前缀（相对安装目录，正斜杠）：用户配置与运行数据永不覆盖。
var protectedPrefixes = []string{
	"config/",      // 用户配置（含 plugins 设置、ai-presets、updates 自身）
	"logs/",        // 运行日志
	"screenshots/", // 截图
	"_temp/",       // 临时产物
	"release/",     // 打包产物
	".pair/toolsets/",
	".pair/memory/",
	".pair/tasks/",
	".pair/skills/",
	".pair/project-info/",
}

// protectedExact 保护名单精确路径。
var protectedExact = []string{
	"config",
	"config/settings.json",
	".pair/project.md",
	".pair/mcp.json",
}

// IsProtected 判断相对路径是否受保护（不覆盖用户数据/配置）。
func IsProtected(rel string) bool {
	r := strings.TrimPrefix(path.Clean(strings.ReplaceAll(rel, "\\", "/")), "./")
	if r == "" || r == "." {
		return true
	}
	for _, p := range protectedPrefixes {
		if strings.HasPrefix(r, p) {
			return true
		}
	}
	for _, e := range protectedExact {
		if strings.EqualFold(r, e) {
			return true
		}
	}
	// .pair 根层 json（用户级配置：如 .pair/xxx.json）
	if strings.HasPrefix(r, ".pair/") {
		rest := r[len(".pair/"):]
		if !strings.Contains(rest, "/") && strings.HasSuffix(strings.ToLower(rest), ".json") {
			return true
		}
	}
	return false
}

// stagedFile 暂存目录内的一个文件。
type stagedFile struct {
	Rel  string
	Size int64
}

// walkStaged 枚举暂存目录内的全部文件（相对路径用正斜杠）。
func walkStaged(root string) ([]stagedFile, error) {
	out := make([]stagedFile, 0, 256)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		var size int64
		if info, ierr := d.Info(); ierr == nil {
			size = info.Size()
		}
		out = append(out, stagedFile{Rel: filepath.ToSlash(rel), Size: size})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PlanStaged 生成替换计划（不做任何写入）——dryRun 展示与真实替换共用同一份判定。
func (c Config) PlanStaged(staging string) (ApplyPlan, []stagedFile, error) {
	plan := ApplyPlan{InstallDir: c.InstallDir, StagingDir: staging, DryRun: true}
	if strings.TrimSpace(c.InstallDir) == "" {
		return plan, nil, fmt.Errorf("安装目录未知（无法定位替换目标）")
	}
	files, err := walkStaged(staging)
	if err != nil {
		return plan, nil, fmt.Errorf("读取暂存目录失败: %w", err)
	}
	for _, f := range files {
		if IsProtected(f.Rel) {
			plan.Skip = append(plan.Skip, f.Rel)
			continue
		}
		plan.Write = append(plan.Write, f.Rel)
		plan.Bytes += f.Size
	}
	plan.MainProgram = MatchMainProgram(plan.Write, c.GOOS, c.GOARCH)
	sort.Strings(plan.Skip)
	return plan, files, nil
}

// ApplyStaged 把暂存内容替换进安装目录。dryRun=true 时只返回计划，不写盘。
// 成功后返回结果计划（含 BackupPath / Renamed / Deferred），失败时返回已发生的部分 + error。
func (c Config) ApplyStaged(staging string, dryRun bool) (ApplyPlan, error) {
	plan, files, err := c.PlanStaged(staging)
	if err != nil {
		return plan, err
	}
	plan.DryRun = dryRun
	if plan.MainProgram == "" {
		return plan, fmt.Errorf("暂存目录内未找到本平台主程序（期望 %s）", MainProgramName(c.GOOS, c.GOARCH))
	}
	if dryRun {
		plan.Message = "dry-run：未改动任何文件"
		return plan, nil
	}
	if err := probeWritable(c.InstallDir); err != nil {
		return plan, fmt.Errorf("安装目录不可写: %w", err)
	}

	// ① 主程序：先备好 .new（同卷复制），再 rename 旧→备份、新→正式名
	backup := filepath.Join(c.InstallDir, plan.MainProgram+".old")
	stagedMain := filepath.Join(staging, filepath.FromSlash(plan.MainProgram))
	newPath := filepath.Join(c.InstallDir, plan.MainProgram+".new")
	if err := copyFile(stagedMain, newPath, 0o755); err != nil {
		return plan, fmt.Errorf("准备新主程序失败: %w", err)
	}
	plan.BackupPath = backup
	_ = os.Remove(backup)
	renameErr := os.Rename(filepath.Join(c.InstallDir, plan.MainProgram), backup)
	switch {
	case renameErr == nil:
		if err := os.Rename(newPath, filepath.Join(c.InstallDir, plan.MainProgram)); err != nil {
			// 回滚：备份恢复
			_ = os.Rename(backup, filepath.Join(c.InstallDir, plan.MainProgram))
			_ = os.Remove(newPath)
			return plan, fmt.Errorf("替换主程序失败（已回滚）: %w", err)
		}
		plan.Renamed = true
	default:
		// 降级：rename 不可用（权限/杀软/跨卷）→ 保留 .new，由重启脚本在退出后 move
		plan.Deferred = true
		plan.Message = "主程序将在进程退出后由重启脚本替换（在线 rename 不可用）"
	}

	// ② 其余文件：覆盖式写入（跳出保护名单与主程序本身）
	// ★ 资源文件写失败不回滚主程序（新主程序可正常启动，资源缺失仅影响对应功能），
	//   但如实记入 plan.Missing 并写入替换记录，便于排查。
	for _, f := range files {
		if IsProtected(f.Rel) || strings.EqualFold(f.Rel, plan.MainProgram) {
			continue
		}
		src := filepath.Join(staging, filepath.FromSlash(f.Rel))
		dst := filepath.Join(c.InstallDir, filepath.FromSlash(f.Rel))
		if !strings.HasPrefix(dst, c.InstallDir+string(os.PathSeparator)) {
			plan.Missing = append(plan.Missing, f.Rel+"（路径越界已跳过）")
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			plan.Missing = append(plan.Missing, f.Rel)
			continue
		}
		if err := copyFile(src, dst, 0o644); err != nil {
			plan.Missing = append(plan.Missing, f.Rel)
		}
	}
	if len(plan.Missing) > 0 {
		// 资源文件写失败不触发主程序回滚（新主程序可正常运行），但如实上报
		plan.Message = fmt.Sprintf("完成，但有 %d 个文件写入失败（见 Failed）", len(plan.Missing))
		_ = writeLastApply(c, LastApply{
			FinishedAt: time.Now().Format(time.RFC3339),
			Backup:     plan.BackupPath,
			Deferred:   plan.Deferred,
			Failed:     plan.Missing,
			Error:      "部分文件写入失败",
		})
	}
	return plan, nil
}

// probeWritable 写探针：确认安装目录可写（避免替换到一半才发现权限不足）。
func probeWritable(dir string) error {
	p := filepath.Join(dir, ".pair-update-probe")
	if err := os.WriteFile(p, []byte("ok"), 0o644); err != nil {
		return err
	}
	return os.Remove(p)
}

// copyFile 复制文件（自动建父目录由调用方负责）。
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// ─── 替换记录 ────────────────────────────────────────────────

// LastApply 最近一次替换记录（config/updates/last-apply.json）。
type LastApply struct {
	Version    string   `json:"version"`
	AppliedAt  string   `json:"appliedAt,omitempty"`
	FinishedAt string   `json:"finishedAt,omitempty"`
	From       string   `json:"from,omitempty"`
	Files      int      `json:"files,omitempty"`
	Backup     string   `json:"backup,omitempty"`
	Deferred   bool     `json:"deferred,omitempty"`
	Failed     []string `json:"failed,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func (c Config) lastApplyPath() string {
	return filepath.Join(c.DownloadDir, "last-apply.json")
}

func writeLastApply(c Config, rec LastApply) error {
	if err := os.MkdirAll(c.DownloadDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.lastApplyPath(), b, 0o644)
}

// ReadLastApply 读取最近一次替换记录（不存在时返回零值与 nil）。
func (c Config) ReadLastApply() (LastApply, error) {
	b, err := os.ReadFile(c.lastApplyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return LastApply{}, nil
		}
		return LastApply{}, err
	}
	var rec LastApply
	if err := json.Unmarshal(b, &rec); err != nil {
		return LastApply{}, err
	}
	return rec, nil
}

// ─── 重启脚本 ────────────────────────────────────────────────

// safeProgramRe 主程序名白名单（防脚本注入：只允许字母数字与 . _ -）。
var safeProgramRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// WriteRestartScript 生成「等待本进程退出 → 处理主程序 → 启动新程序 → 自删」的重启脚本。
// deferred=true 时脚本负责把 <main>.new move 到正式名（在线 rename 不可用的降级路径）。
func (c Config) WriteRestartScript(main string, deferred bool) (string, error) {
	base := path.Base(strings.ReplaceAll(main, "\\", "/"))
	if !safeProgramRe.MatchString(base) {
		return "", fmt.Errorf("主程序名不合法: %s", main)
	}
	if err := os.MkdirAll(c.DownloadDir, 0o755); err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102-150405")
	install := c.InstallDir
	if c.GOOS == "windows" {
		p := filepath.Join(c.DownloadDir, "restart-"+ts+".bat")
		body := buildWindowsRestart(install, base, deferred)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			return "", err
		}
		return p, nil
	}
	p := filepath.Join(c.DownloadDir, "restart-"+ts+".sh")
	body := buildUnixRestart(install, base, deferred)
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		return "", err
	}
	return p, nil
}

// buildWindowsRestart 生成 Windows 重启脚本（CRLF 换行）。
func buildWindowsRestart(install, main string, deferred bool) string {
	exe := filepath.Join(install, main)
	lines := []string{
		"@echo off",
		"rem PairCode 在线更新重启脚本（由程序生成，等待旧进程退出后启动新程序）",
		"ping -n 4 127.0.0.1 >nul 2>nul",
	}
	if deferred {
		lines = append(lines,
			fmt.Sprintf(`if exist "%s.new" move /y "%s.new" "%s" >nul 2>nul`, exe, exe, exe))
	}
	lines = append(lines,
		fmt.Sprintf(`del /f /q "%s.old" >nul 2>nul`, exe),
		fmt.Sprintf(`start "" "%s"`, exe),
		"timeout /t 5 /nobreak >nul 2>nul",
		`del /f /q "%~f0" >nul 2>nul`,
		"",
	)
	return strings.Join(lines, "\r\n")
}

// buildUnixRestart 生成 POSIX sh 重启脚本。
func buildUnixRestart(install, main string, deferred bool) string {
	exe := path.Join(filepath.ToSlash(install), main)
	lines := []string{
		"#!/bin/sh",
		"# PairCode 在线更新重启脚本（等待旧进程退出后启动新程序）",
		"sleep 3",
	}
	if deferred {
		lines = append(lines, fmt.Sprintf(`[ -f "%s.new" ] && mv -f "%s.new" "%s"`, exe, exe, exe))
	}
	lines = append(lines,
		fmt.Sprintf(`rm -f "%s.old"`, exe),
		fmt.Sprintf(`cd "%s" || exit 1`, filepath.ToSlash(install)),
		fmt.Sprintf(`nohup "./%s" >> "%s/logs/update-restart.log" 2>&1 &`, main, filepath.ToSlash(install)),
		"sleep 3",
		`rm -f "$0"`,
		"",
	)
	return strings.Join(lines, "\n")
}
