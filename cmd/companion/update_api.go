// update_api.go — 在线更新接口（/api/update/*）：引擎单例、配置装配、HTTP 处理器。
//
// 架构（与项目「接口插件化」一致）：
//   - 本文件实现处理器（Go 内核能力）；
//   - 路由在 cmd/companion/kernel_register.go 注册进内核路由表；
//   - 挂载由 .pair/plugins/core-api/ 的 ROUTES 清单完成（缺则该接口不挂载）；
//   - 设置项由 .pair/plugins/app-update/ 注册 schema（存 pluginSettings.app-update）。
//
// 更新引擎本身在 internal/update（可脱离宿主单测），本文件只做「宿主适配」：
// 配置来源（core.InstallDir/ConfigDir + 插件设置 + 环境变量）、HTTP 出入参、自动检查。
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/update"
)

// updateSettingKey 更新设置段 key（.pair/plugins/app-update/index.js 注册）。
const updateSettingKey = "app-update"

// 环境变量覆盖（离线源 / 测试 / 内网镜像；优先级高于插件设置）。
const (
	envUpdateFeed   = "PAIRCODE_UPDATE_FEED"   // 自定义清单 URL 或本地路径（→ feedType=custom）
	envUpdateRepo   = "PAIRCODE_UPDATE_REPO"   // owner/repo 覆盖
	envUpdateMirror = "PAIRCODE_UPDATE_MIRROR" // 镜像前缀覆盖
	envUpdateDir    = "PAIRCODE_UPDATE_DIR"    // 下载/暂存目录覆盖
)

// updateEngine 进程内更新引擎单例。
var updateEngine = update.NewEngine(update.Config{CurrentVersion: version})

// buildUpdateConfig 装配引擎配置（每次请求前刷新 → 设置改动即时生效）。
func buildUpdateConfig() update.Config {
	cfg := update.DefaultConfig()
	cfg.CurrentVersion = version
	cfg.GOOS = runtime.GOOS
	cfg.GOARCH = runtime.GOARCH
	cfg.InstallDir = core.InstallDir()
	cfg.DownloadDir = filepath.Join(core.ConfigDir(), "updates")
	cfg.UserAgent = "PairCode-Updater/" + update.NormalizeVersion(version)

	if st := core.Settings.PluginSettings[updateSettingKey]; st != nil {
		if v := updStr(st, "feedType"); v != "" {
			cfg.FeedType = v
		}
		cfg.Repo = updStrDefault(st, "repo", cfg.Repo)
		cfg.FeedURL = updStr(st, "feedURL")
		cfg.MirrorPrefix = updStr(st, "mirrorPrefix")
		if v := updStr(st, "channel"); v != "" {
			cfg.Channel = v
		}
		cfg.RequireSHA256 = updBool(st, "requireSha256", true)
		cfg.KeepBackup = updBool(st, "keepBackup", true)
	}

	// 环境变量优先（测试/离线源）
	if v := strings.TrimSpace(os.Getenv(envUpdateFeed)); v != "" {
		cfg.FeedType = "custom"
		cfg.FeedURL = v
	}
	if v := strings.TrimSpace(os.Getenv(envUpdateRepo)); v != "" {
		cfg.Repo = v
	}
	if v := strings.TrimSpace(os.Getenv(envUpdateMirror)); v != "" {
		cfg.MirrorPrefix = v
	}
	if v := strings.TrimSpace(os.Getenv(envUpdateDir)); v != "" {
		cfg.DownloadDir = v
	}
	// 兜底：custom 但无地址 → 回落到 GitHub 源（避免配置错误导致功能不可用）
	if strings.EqualFold(cfg.FeedType, "custom") && strings.TrimSpace(cfg.FeedURL) == "" {
		cfg.FeedType = "github"
	}
	return cfg
}

// refreshUpdateEngine 刷新引擎配置（只改配置，不动状态）。
func refreshUpdateEngine() {
	updateEngine.SetConfig(buildUpdateConfig())
}

// startUpdateAutoCheck 后台自动检查：启动 20s 后首次检查，之后按
// checkIntervalHours（默认 24h）节流；结果只更新状态，不自动下载。
func startUpdateAutoCheck() {
	go func() {
		time.Sleep(20 * time.Second)
		for {
			enabled, hours := true, 24
			if st := core.Settings.PluginSettings[updateSettingKey]; st != nil {
				enabled = updBool(st, "autoCheck", true)
				hours = updIntDefault(st, "checkIntervalHours", 24)
			}
			if enabled && updateEngine.AutoCheckDue(time.Duration(hours)*time.Hour) {
				refreshUpdateEngine()
				ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
				st, err := updateEngine.Check(ctx)
				cancel()
				if err != nil {
					log.Printf("[update] 自动检查失败: %v", err)
				} else if st.HasUpdate {
					log.Printf("[update] 发现新版本 %s（当前 %s）", st.Latest, st.Current)
				} else {
					log.Printf("[update] 自动检查：已是最新（%s）", st.Current)
				}
			}
			time.Sleep(30 * time.Minute)
		}
	}()
}

// ─── HTTP 处理器 ─────────────────────────────────────────────

// handleUpdateStatus GET /api/update/status — 状态快照（前端轮询）。
func (s *webServer) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"ok": true, "state": updateEngine.Snapshot()})
}

// handleUpdateCheck GET /api/update/check — 拉清单并比较版本。
func (s *webServer) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	refreshUpdateEngine()
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	st, err := updateEngine.Check(ctx)
	jsonResp(w, map[string]any{"ok": err == nil, "state": st, "error": errText(err)})
}

// handleUpdateDownload POST /api/update/download — 启动后台下载（幂等）。
func (s *webServer) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	refreshUpdateEngine()
	st, err := updateEngine.StartDownload()
	jsonResp(w, map[string]any{"ok": err == nil, "state": st, "error": errText(err)})
}

// updateApplyReq 替换请求体。
type updateApplyReq struct {
	DryRun  bool  `json:"dryRun"`  // 只报告将写/跳过/失败清单
	Restart *bool `json:"restart"` // 替换成功后是否立即重启（默认 true）
}

// handleUpdateApply POST /api/update/apply — 替换安装目录文件（默认随后重启）。
func (s *webServer) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	refreshUpdateEngine()
	var req updateApplyReq
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	restart := true
	if req.Restart != nil {
		restart = *req.Restart
	}
	res, err := updateEngine.Apply(req.DryRun, restart)
	jsonResp(w, map[string]any{
		"ok":     err == nil,
		"result": res,
		"error":  errText(err),
		"state":  updateEngine.Snapshot(),
	})
}

// handleUpdateCancel POST /api/update/cancel — 取消进行中的检查/下载。
func (s *webServer) handleUpdateCancel(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"ok": true, "state": updateEngine.Cancel()})
}

// handleUpdateConfig GET /api/update/config — 生效配置（供前端展示更新源）。
func (s *webServer) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	cfg := buildUpdateConfig()
	jsonResp(w, map[string]any{
		"ok":             true,
		"currentVersion": update.NormalizeVersion(cfg.CurrentVersion),
		"goos":           cfg.GOOS,
		"goarch":         cfg.GOARCH,
		"feedType":       cfg.FeedType,
		"repo":           cfg.Repo,
		"feedURL":        cfg.FeedURL,
		"mirror":         cfg.MirrorPrefix,
		"channel":        cfg.Channel,
		"requireSha256":  cfg.RequireSHA256,
		"keepBackup":     cfg.KeepBackup,
		"installDir":     cfg.InstallDir,
		"downloadDir":    cfg.DownloadDir,
		"mainProgram":    update.MainProgramName(cfg.GOOS, cfg.GOARCH),
		"source":         describeUpdateSource(cfg),
	})
}

// describeUpdateSource 更新源的人类可读描述（前端展示用）。
func describeUpdateSource(cfg update.Config) string {
	if strings.EqualFold(cfg.FeedType, "custom") {
		return "自定义源：" + cfg.FeedURL
	}
	s := "GitHub：" + cfg.Repo
	if cfg.Channel == "prerelease" {
		s += "（含预发布）"
	}
	if cfg.MirrorPrefix != "" {
		s += "｜镜像：" + cfg.MirrorPrefix
	}
	return s
}

// ─── 设置读取小工具（pluginSettings 反序列化后数字为 float64）──

func updStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func updStrDefault(m map[string]any, key, def string) string {
	if v := updStr(m, key); v != "" {
		return v
	}
	return def
}

func updBool(m map[string]any, key string, def bool) bool {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return def
}

func updIntDefault(m map[string]any, key string, def int) int {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// errText 把 error 转成可读字符串（nil → 空串）。
func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
