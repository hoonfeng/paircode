// engine.go — 更新引擎：状态机编排（检查 → 下载 → 校验 → 解压 → 替换 → 重启）。
//
// 并发模型：同一时刻只允许一个变更中任务（busy）；状态快照供 HTTP 轮询读取。
// 引擎不读全局状态，配置由宿主注入，可脱离宿主单测。
package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Engine 更新引擎（进程内单例，宿主持有）。
type Engine struct {
	mu     sync.Mutex
	cfg    Config
	state  State
	cancel context.CancelFunc
	busy   bool      // 有进行中的异步任务（下载/替换）
	lastCh time.Time // 上次检查时间（自动检查节流）
}

// NewEngine 创建引擎。
func NewEngine(cfg Config) *Engine {
	e := &Engine{cfg: cfg}
	e.state = State{
		Stage:   StageIdle,
		Current: NormalizeVersion(cfg.CurrentVersion),
		Message: "尚未检查更新",
	}
	return e
}

// SetConfig 热更新配置（设置保存后调用；不清空已有状态）。
func (e *Engine) SetConfig(cfg Config) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state.Current == "" {
		e.state.Current = NormalizeVersion(cfg.CurrentVersion)
	}
	e.cfg = cfg
}

// Config 返回当前配置副本。
func (e *Engine) Config() Config {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cfg
}

// Snapshot 返回状态快照（指针字段深拷贝，避免调用方读到竞态数据）。
func (e *Engine) Snapshot() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *Engine) snapshotLocked() State {
	s := e.state
	if e.state.Asset != nil {
		a := *e.state.Asset
		s.Asset = &a
	}
	if e.state.Progress != nil {
		p := *e.state.Progress
		s.Progress = &p
	}
	if e.state.Plan != nil {
		pl := *e.state.Plan
		s.Plan = &pl
	}
	return s
}

// set 在锁内修改状态。
func (e *Engine) set(fn func(*State)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fn(&e.state)
}

// fail 置为错误态（可从任何阶段进入）。
func (e *Engine) fail(msg string) {
	e.set(func(s *State) {
		s.Stage = StageError
		s.Error = msg
		s.Message = msg
		if s.Progress != nil {
			s.Progress = nil
		}
	})
}

// StagingPath 版本对应的暂存目录。
func (e *Engine) StagingPath(ver string) string {
	return filepath.Join(e.Config().DownloadDir, "staging-"+NormalizeVersion(ver))
}

// AutoCheckDue 自动检查是否到期（启动后首查或超过间隔）。
func (e *Engine) AutoCheckDue(interval time.Duration) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if interval <= 0 {
		return e.lastCh.IsZero()
	}
	return time.Since(e.lastCh) >= interval
}

// MarkChecked 记录检查时间（自动检查节流用）。
func (e *Engine) MarkChecked() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastCh = time.Now()
}

// ─── 检查 ────────────────────────────────────────────────────

// Check 拉取清单并比较版本（同步；调用方用 context 限时）。
func (e *Engine) Check(ctx context.Context) (State, error) {
	e.mu.Lock()
	if e.busy {
		st := e.snapshotLocked()
		e.mu.Unlock()
		return st, fmt.Errorf("有任务进行中（%s），请稍候", st.Stage)
	}
	cfg := e.cfg
	e.state.Stage = StageChecking
	e.state.Error = ""
	e.state.Message = "正在检查更新…"
	e.mu.Unlock()

	m, err := cfg.FetchManifest(ctx)
	if err != nil {
		e.fail("检查更新失败：" + err.Error())
		return e.Snapshot(), err
	}

	has := IsNewer(m.Version, cfg.CurrentVersion)
	now := time.Now().Format(time.RFC3339)
	e.set(func(s *State) {
		s.Current = NormalizeVersion(cfg.CurrentVersion)
		s.Latest = m.Version
		s.Notes = m.Notes
		s.PublishedAt = m.PublishedAt
		s.Source = m.Source
		s.CheckedAt = now
		s.Error = ""
		s.HasUpdate = has
		s.Progress = nil
		s.Plan = nil
		asset := m.Asset
		s.Asset = &asset
		if has {
			s.Stage = StageAvailable
			s.Message = fmt.Sprintf("发现新版本 %s（当前 %s）", m.Version, NormalizeVersion(cfg.CurrentVersion))
		} else {
			s.Stage = StageUpToDate
			s.Message = fmt.Sprintf("已是最新版本（%s）", NormalizeVersion(cfg.CurrentVersion))
		}
	})
	e.MarkChecked()
	return e.Snapshot(), nil
}

// ─── 下载（异步）─────────────────────────────────────────────

// StartDownload 启动后台下载（幂等：已有任务进行中则返回当前状态）。
// 前置：先完成 Check 且 HasUpdate=true。
func (e *Engine) StartDownload() (State, error) {
	e.mu.Lock()
	if e.busy {
		st := e.snapshotLocked()
		e.mu.Unlock()
		return st, nil
	}
	if e.state.Asset == nil {
		e.mu.Unlock()
		return e.state, fmt.Errorf("尚未检查更新（请先检查）")
	}
	if !e.state.HasUpdate {
		st := e.snapshotLocked()
		e.mu.Unlock()
		return st, fmt.Errorf("当前已是最新版本")
	}
	asset := *e.state.Asset
	ver := e.state.Latest
	cfg := e.cfg
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.busy = true
	e.state.Stage = StageDownloading
	e.state.Error = ""
	e.state.Message = "下载中…"
	e.state.Progress = &Progress{Total: asset.Size}
	e.mu.Unlock()

	go e.runDownload(ctx, cfg, asset, ver)
	return e.Snapshot(), nil
}

// runDownload 后台任务：下载 → 校验 → 解压 → 就绪。
func (e *Engine) runDownload(ctx context.Context, cfg Config, a AssetRef, ver string) {
	defer func() {
		e.mu.Lock()
		e.busy = false
		e.cancel = nil
		e.mu.Unlock()
	}()

	var winBytes, winStart, winSpeed int64
	winStart = time.Now().UnixNano()
	winBytes = 0
	onProgress := func(n int64) {
		now := time.Now().UnixNano()
		el := float64(now-winStart) / float64(time.Second)
		if el >= 0.5 { // 半秒窗口速率
			winSpeed = int64(float64(n-winBytes) / el)
			winBytes, winStart = n, now
		}
		e.set(func(s *State) {
			total := a.Size
			if total == 0 {
				total = n
			}
			pct := 0.0
			if total > 0 {
				pct = float64(n) / float64(total) * 100
				if pct > 100 {
					pct = 100
				}
			}
			s.Progress = &Progress{Downloaded: n, Total: total, Percent: pct, SpeedBps: winSpeed}
			s.Message = fmt.Sprintf("下载中… %.1f%%", pct)
		})
	}

	res, err := cfg.Download(ctx, a, onProgress)
	if err != nil {
		e.fail(err.Error())
		return
	}
	e.set(func(s *State) {
		s.Stage = StageVerifying
		s.Message = "校验通过（sha256 一致）"
		if s.Progress != nil {
			s.Progress.Percent = 100
			s.Progress.Downloaded = res.Bytes
			s.Progress.Total = res.Bytes
		}
	})

	staging := e.StagingPath(ver)
	files, err := Extract(res.Path, staging)
	if err != nil {
		e.fail("解压失败：" + err.Error())
		return
	}
	main, err := SmokeCheck(staging, files, cfg.GOOS, cfg.GOARCH)
	if err != nil {
		e.fail(err.Error())
		return
	}
	e.set(func(s *State) {
		s.Stage = StageReady
		s.Message = fmt.Sprintf("更新包已就绪（%s，含 %d 个文件，主程序 %s）", ver, len(files), main)
		s.RestartHint = "点击「立即安装并重启」完成更新；进程退出后由重启脚本启动新版本"
	})
}

// Cancel 取消进行中的下载/检查。
func (e *Engine) Cancel() State {
	e.mu.Lock()
	cancel := e.cancel
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	e.set(func(s *State) {
		if s.Stage == StageDownloading || s.Stage == StageChecking {
			s.Stage = StageAvailable
			s.Error = ""
			s.Message = "已取消"
			s.Progress = nil
		}
	})
	return e.Snapshot()
}

// ─── 替换 / 重启 ─────────────────────────────────────────────

// ApplyResult 替换结果（供 API 返回）。
type ApplyResult struct {
	Plan     ApplyPlan `json:"plan"`
	Restart  bool      `json:"restart"`
	Script   string    `json:"script,omitempty"`
	WillExit bool      `json:"willExit"`
}

// Apply 执行替换（dryRun 只报告）；restart=true 时替换成功后启动重启脚本并退出本进程。
func (e *Engine) Apply(dryRun, restart bool) (ApplyResult, error) {
	e.mu.Lock()
	if e.busy {
		e.mu.Unlock()
		return ApplyResult{}, fmt.Errorf("有任务进行中，请稍候")
	}
	st := e.snapshotLocked()
	cfg := e.cfg
	if st.Latest == "" {
		e.mu.Unlock()
		return ApplyResult{}, fmt.Errorf("尚未检查更新")
	}
	// ★ 注意：此处已持锁 —— 不能调用 StagingPath（它内部 Config() 会再次加锁 → 死锁）。
	staging := filepath.Join(cfg.DownloadDir, "staging-"+NormalizeVersion(st.Latest))
	if _, err := os.Stat(staging); err != nil {
		e.mu.Unlock()
		return ApplyResult{}, fmt.Errorf("更新包尚未下载或已清理（%s）", staging)
	}

	if dryRun {
		plan, err := cfg.ApplyStaged(staging, true)
		e.mu.Unlock()
		return ApplyResult{Plan: plan}, err
	}
	e.busy = true
	e.state.Stage = StageApplying
	e.state.Error = ""
	e.state.Message = "正在替换文件…"
	e.mu.Unlock()

	plan, err := cfg.ApplyStaged(staging, false)
	e.set(func(s *State) { s.Plan = &plan })
	if err != nil {
		e.mu.Lock()
		e.busy = false
		e.mu.Unlock()
		e.fail("替换失败：" + err.Error())
		return ApplyResult{Plan: plan}, err
	}

	res := ApplyResult{Plan: plan, Restart: restart}
	rec := LastApply{
		Version:   st.Latest,
		AppliedAt: time.Now().Format(time.RFC3339),
		From:      st.Current,
		Files:     len(plan.Write),
		Backup:    plan.BackupPath,
		Deferred:  plan.Deferred,
		Failed:    plan.Missing,
	}
	if err := writeLastApply(cfg, rec); err != nil {
		e.set(func(s *State) { s.Message = "替换完成，但替换记录写入失败：" + err.Error() })
	}
	e.set(func(s *State) {
		s.Current = st.Latest
		s.HasUpdate = false
	})

	if !restart {
		e.mu.Lock()
		e.busy = false
		e.mu.Unlock()
		e.set(func(s *State) {
			s.Stage = StageReady
			s.Message = "文件已替换，重启后生效"
			s.RestartHint = fmt.Sprintf("请手动重启程序（若主程序被占用，新版位于 %s.new）", plan.MainProgram)
		})
		return res, nil
	}

	script, serr := cfg.WriteRestartScript(plan.MainProgram, plan.Deferred)
	if serr != nil {
		e.mu.Lock()
		e.busy = false
		e.mu.Unlock()
		e.fail("生成重启脚本失败（文件已替换，请手动重启）：" + serr.Error())
		return ApplyResult{Plan: plan, Restart: false}, serr
	}
	if serr := StartDetached(script, cfg); serr != nil {
		e.mu.Lock()
		e.busy = false
		e.mu.Unlock()
		e.fail("启动重启脚本失败（文件已替换，请手动重启）：" + serr.Error())
		return ApplyResult{Plan: plan, Restart: false, Script: script}, serr
	}
	e.set(func(s *State) {
		s.Stage = StageRestarting
		s.Message = "更新完成，正在重启…"
	})
	res.Script = script
	res.WillExit = true
	// 让 HTTP 响应先返回，再退出进程（由重启脚本接管）
	go func() {
		time.Sleep(800 * time.Millisecond)
		os.Exit(0)
	}()
	return res, nil
}

// CleanupStaging 清理暂存目录（替换完成或用户放弃时调用）。
func (e *Engine) CleanupStaging(ver string) error {
	if ver == "" {
		ver = e.Snapshot().Latest
	}
	if ver == "" {
		return nil
	}
	return os.RemoveAll(e.StagingPath(ver))
}
