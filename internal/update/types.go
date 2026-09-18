// Package update 实现基于 GitHub Releases 的在线更新引擎。
//
// 全链路：清单获取（GitHub Releases API / 自定义 feed）→ 版本比较 → 下载
// （断点续传 + 流式 sha256）→ 校验 → 安全解压 → 安装目录替换（含保护名单与
// 回滚）→ 重启。
//
// ★ 设计约束（见 docs/online-update-design.md）：
//   - 更新内容只来自发布端点，且默认强制 sha256 校验（校验不过一律不落地）；
//   - 只写安装目录内文件，解压防 zip-slip / zip bomb；
//   - 不覆盖用户配置与用户数据（保护名单）；
//   - 替换主程序用「同卷 rename 运行中 exe」技巧，失败降级为退出后由重启脚本复制。
//
// 本包不读全局状态（core.Settings / 环境）——配置由宿主构造 Config 注入，
// 因此可脱离宿主单测。
package update

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// ─── 阶段常量 ────────────────────────────────────────────────

// 引擎阶段（State.Stage）。
const (
	StageIdle        = "idle"        // 未检查
	StageChecking    = "checking"    // 拉取清单中
	StageUpToDate    = "up-to-date"  // 已是最新
	StageAvailable   = "available"   // 有新版本，待下载
	StageDownloading = "downloading" // 下载中
	StageVerifying   = "verifying"   // 校验 sha256
	StageExtracting  = "extracting"  // 解压中
	StageReady       = "ready"       // 已就绪（可安装）
	StageApplying    = "applying"    // 替换文件中
	StageRestarting  = "restarting"  // 即将重启
	StageError       = "error"       // 出错（附 message）
)

// ─── 清单 ────────────────────────────────────────────────────

// AssetRef 一个平台发布包（zip）。
type AssetRef struct {
	ID     int64  `json:"id,omitempty"`     // GitHub asset id（→ API 下载端点）
	Name   string `json:"name"`             // 文件名（PairCode-1.7.0.zip）
	Size   int64  `json:"size"`             // 字节数（0 = 未知）
	SHA256 string `json:"sha256"`           // 十六进制摘要（不含 sha256: 前缀）
	URL    string `json:"url"`              // 直链（browser_download_url 或自定义源）
	APIURL string `json:"apiUrl,omitempty"` // GitHub 资产 API 端点（稳定优先）
	OS     string `json:"os,omitempty"`     // 该资产对应平台（自定义 feed 用）
}

// Manifest 归一化后的更新清单（GitHub 与自定义 feed 共用）。
type Manifest struct {
	Version     string   `json:"version"`     // 规范化版本（不含 v 前缀）：1.7.0
	Tag         string   `json:"tag"`         // 原始 tag：v1.7.0
	Notes       string   `json:"notes"`       // 发布说明（Markdown）
	PublishedAt string   `json:"publishedAt"` // RFC3339（可空）
	Asset       AssetRef `json:"asset"`       // 本平台资产（匹配失败时为零值）
	Source      string   `json:"source"`      // 来源描述：github:owner/repo / custom:URL
}

// ─── 配置 ────────────────────────────────────────────────────

// Config 引擎配置（由宿主从插件设置 + 环境变量构造）。
type Config struct {
	CurrentVersion string // 本地版本（如 v1.6.3 或 1.6.3）
	GOOS           string // 运行平台（默认 runtime.GOOS）
	GOARCH         string // 运行架构（默认 runtime.GOARCH）

	Repo         string // GitHub owner/repo（feedType=github）
	FeedType     string // github | custom
	FeedURL      string // 自定义清单 URL（file:// 或 http(s)://）
	MirrorPrefix string // 镜像前缀（非空时拼接在下载 URL 前）
	Channel      string // stable | prerelease
	Token        string // 可选 GitHub token（提升 API 限额 / 私有仓库）

	RequireSHA256 bool // 强制校验（默认 true）
	KeepBackup    bool // 保留 pair.exe.old（默认 true）

	DownloadDir string // 下载/暂存目录（<ConfigDir>/updates）
	InstallDir  string // 安装目录（替换目标）

	UserAgent string // HTTP User-Agent（默认 PairCode-Updater/<ver>）

	// 测试钩子（零值 = 生产行为：内部自建 *http.Client）
	HTTPClient HTTPDoer // 自定义 HTTP 客户端（单测注入桩）
}

// HTTPDoer 最小 HTTP 客户端接口（单测注入桩用；*http.Client 天然满足）。
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DefaultConfig 返回带默认值的配置（宿主只需覆盖 Repo/版本/目录）。
func DefaultConfig() Config {
	return Config{
		Repo:          "hoonfeng/paircode",
		FeedType:      "github",
		Channel:       "stable",
		RequireSHA256: true,
		KeepBackup:    true,
	}
}

// ─── 状态 ────────────────────────────────────────────────────

// Progress 下载进度。
type Progress struct {
	Downloaded int64   `json:"downloaded"` // 已下载字节
	Total      int64   `json:"total"`      // 总字节（0 = 未知）
	Percent    float64 `json:"percent"`    // 0..100
	SpeedBps   int64   `json:"speedBps"`   // 瞬时速率（字节/秒）
	Resumed    bool    `json:"resumed"`    // 本次是否走断点续传
}

// ApplyPlan 替换计划/结果（dryRun 时只报告不落地）。
type ApplyPlan struct {
	InstallDir  string   `json:"installDir"`
	StagingDir  string   `json:"stagingDir"`
	MainProgram string   `json:"mainProgram"` // 包内主程序名（pair.exe / pair-linux-amd64）
	BackupPath  string   `json:"backupPath"`  // 备份后的旧主程序路径（pair.exe.old）
	DryRun      bool     `json:"dryRun"`
	Deferred    bool     `json:"deferred"` // 已降级为「退出后由重启脚本复制」
	Renamed     bool     `json:"renamed"`  // 主程序是否已在线替换成功
	Write       []string `json:"write"`    // 将写/已写（相对安装目录）
	Skip        []string `json:"skip"`     // 保护名单跳过
	Missing     []string `json:"missing"`  // 写失败
	Bytes       int64    `json:"bytes"`    // 将写/已写字节数
	Message     string   `json:"message"`
}

// State 引擎状态快照（/api/update/status 返回体）。
type State struct {
	Stage       string     `json:"stage"`
	Current     string     `json:"current"`
	Latest      string     `json:"latest"`
	HasUpdate   bool       `json:"hasUpdate"`
	Notes       string     `json:"notes"`
	PublishedAt string     `json:"publishedAt"`
	Asset       *AssetRef  `json:"asset,omitempty"`
	Progress    *Progress  `json:"progress,omitempty"`
	Plan        *ApplyPlan `json:"plan,omitempty"`
	Error       string     `json:"error"`
	Message     string     `json:"message"`
	CheckedAt   string     `json:"checkedAt"`
	Source      string     `json:"source"`
	RestartHint string     `json:"restartHint,omitempty"` // 重启提示文案（如手工回滚说明）
}

// ─── 版本比较 ────────────────────────────────────────────────

// NormalizeVersion 归一化版本字符串：去空白、去 refs/tags/ 与 v/V 前缀。
func NormalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "refs/tags/")
	s = strings.TrimPrefix(s, "refs/heads/")
	if len(s) > 1 && (s[0] == 'v' || s[0] == 'V') && s[1] >= '0' && s[1] <= '9' {
		s = s[1:]
	}
	return s
}

// CompareVersions 比较两个版本号：a < b 返回 -1，相等 0，a > b 返回 1。
// 容错：忽略 v 前缀；不可解析时退化为字符串比较（保证不 panic）。
func CompareVersions(a, b string) int {
	av, aerr := semver.NewVersion(NormalizeVersion(a))
	bv, berr := semver.NewVersion(NormalizeVersion(b))
	if aerr != nil || berr != nil {
		return strings.Compare(NormalizeVersion(a), NormalizeVersion(b))
	}
	return av.Compare(bv)
}

// IsNewer 返回 remote 是否比 local 新。
func IsNewer(remote, local string) bool {
	return CompareVersions(remote, local) > 0
}

// ─── 平台 / 资产映射 ─────────────────────────────────────────

// PlatformSuffix 资产名中的平台中缀（windows 的包名不含平台词 → 空串）。
func PlatformSuffix(goos string) string {
	switch strings.ToLower(goos) {
	case "windows":
		return ""
	case "linux":
		return "linux"
	case "darwin":
		return "darwin"
	}
	return strings.ToLower(goos)
}

// AssetNameForPlatform 生成期望的资产名：PairCode-<ver>.zip / PairCode-<os>-<ver>.zip。
func AssetNameForPlatform(ver, goos string) string {
	s := PlatformSuffix(goos)
	v := strings.TrimPrefix(strings.TrimSpace(ver), "v")
	if s == "" {
		return fmt.Sprintf("PairCode-%s.zip", v)
	}
	return fmt.Sprintf("PairCode-%s-%s.zip", s, v)
}

// MatchAsset 从资产名列表中挑出本平台该版本对应的 zip（宽容匹配）。
// 顺序：精确（含/不含 v 前缀）→ 含版本号 + 平台词 → 返回空串表示无可用包。
func MatchAsset(names []string, ver, goos string) string {
	zips := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasSuffix(strings.ToLower(n), ".zip") {
			zips = append(zips, n)
		}
	}
	if len(zips) == 0 {
		return ""
	}
	want := AssetNameForPlatform(ver, goos)
	for _, n := range zips {
		if strings.EqualFold(n, want) {
			return n
		}
	}
	// 版本号带 v 的写法
	wantV := AssetNameForPlatform("v"+strings.TrimPrefix(strings.TrimSpace(ver), "v"), goos)
	for _, n := range zips {
		if strings.EqualFold(n, wantV) {
			return n
		}
	}
	// 模糊：名字含版本号 + 平台词
	v := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ver), "v"))
	s := PlatformSuffix(goos)
	for _, n := range zips {
		ln := strings.ToLower(n)
		if !strings.Contains(ln, v) {
			continue
		}
		if s == "" {
			// windows 包名不含平台词：排除明确属于其它平台的资产
			if strings.Contains(ln, "linux") || strings.Contains(ln, "darwin") || strings.Contains(ln, "macos") {
				continue
			}
			return n
		}
		if strings.Contains(ln, s) || (s == "darwin" && strings.Contains(ln, "macos")) {
			return n
		}
	}
	return ""
}

// MainProgramName 期望的主程序名：windows 为 pair.exe，其它平台 pair-<goos>-<arch>。
func MainProgramName(goos, goarch string) string {
	if strings.EqualFold(goos, "windows") {
		return "pair.exe"
	}
	return fmt.Sprintf("pair-%s-%s", strings.ToLower(goos), strings.ToLower(goarch))
}

// MatchMainProgram 在包内条目里找主程序名（宽容：精确 → pair-<goos>-* → pair*.exe / pair）。
// files 为相对路径清单（可含子目录，只看根层与任意深度的主程序）。
func MatchMainProgram(files []string, goos, goarch string) string {
	exact := MainProgramName(goos, goarch)
	var fallback, winExe, plain string
	plat := strings.ToLower(goos)
	for _, f := range files {
		base := f
		if i := strings.LastIndexAny(f, `/\`); i >= 0 {
			base = f[i+1:]
		}
		if strings.EqualFold(base, exact) {
			return base
		}
		lb := strings.ToLower(base)
		if strings.EqualFold(goos, "windows") {
			if winExe == "" && strings.HasSuffix(lb, ".exe") && strings.Contains(lb, "pair") {
				winExe = base
			}
			continue
		}
		if fallback == "" && strings.HasPrefix(lb, "pair-"+plat) {
			fallback = base
		}
		if plain == "" && (lb == "pair" || lb == "paircode") {
			plain = base
		}
	}
	if fallback != "" {
		return fallback
	}
	if winExe != "" {
		return winExe
	}
	return plain
}
