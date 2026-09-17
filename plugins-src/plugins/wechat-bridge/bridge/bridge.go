// Package bridge 桥主循环：收（getupdates 长轮询）→ 投喂（PairCode）→ 双路等待
// （WS 流式主路 + JSONL 兜底）→ 发（sendmessage）。
//
// 多账号模型：每账号一个 Bridge（独立 goroutine 组与状态），共享一条
// EventStream（按 convId 路由事件）。账号级隔离：单桥 panic/异常不扩散。
//
// 蓝图：M1 Node 原型 temp/wx-bridge/src/bridge.mjs（线上实测版本）。
package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/account"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/config"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/ilink"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/login"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/media"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/paircode"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/store"
	"github.com/hoonfeng/paircode/plugins-src/plugins/wechat-bridge/stream"
)

// logf 日志函数（由宿主注入；桥内所有输出经此）。
type logf func(format string, args ...any)

// -14（token 暂时失效）重试策略：先观察恢复，多次失败后才请求重新登录。
var staleRetryWaits = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second, 60 * time.Second}

const (
	processedKeep   = 500 // 去重集合裁剪保留数
	processedMax    = 800 // 超过则裁剪
	saveDebounceMs  = 500 // 状态持久化防抖
	msgSendGapMs    = 500 // 分片发送间隔
	sendTimeoutSec  = 30  // 投喂超时
	askUserSnippet  = 500 // ask_user 通知截断
	replyMsgTimeout = 30 * time.Minute
)

// State 桥运行状态（accounts/<id>/state.json）。
type State struct {
	GetUpdatesBuf string            `json:"get_updates_buf"`
	ProcessedIDs  []string          `json:"processedIds"`
	ContextTokens map[string]string `json:"context_tokens"`
}

// pendingMsg 待处理消息（enqueue 解析产物）。
type pendingMsg struct {
	from  string
	text  string
	media []ilink.MsgItem // 附带的媒体条目（type 2/3/4/5）
}

// Bridge 单账号桥主循环。
type Bridge struct {
	cfg  config.Config
	info account.Info
	dir  string // 账号数据目录（state.json 位置）
	api  *ilink.Client
	es   *EventStream
	log  logf
	raw  *rawLogger // raw 抓样（默认开；config.RawDump=false 时为 nil）

	onStale func(id string) // -14 持续失败：请求宿主发起重新登录

	mu            sync.Mutex
	getUpdatesBuf string
	processed     map[string]struct{}
	processedQ    []string
	ctxTokens     map[string]string
	typingTickets map[string]string
	activeSession *stream.Session
	draining      bool              // ★ 忙对齐：drain 相位（等待前序任务结束，丢弃其输出）
	drainDone     chan struct{}     // drain 结束信号（endDrain 关闭；幂等）
	replyDoneCh   chan stream.Event // ★ done 快路径：非流式等待期间 done 事件直达（awaitReply 登记）

	queue    chan pendingMsg
	stopCh   chan struct{}
	stopOnce sync.Once
	unwatch  func()

	saveMu    sync.Mutex
	saveTimer *time.Timer
}

// New 创建桥（dir = 账号数据目录；不自动启动）。
func New(cfg config.Config, info account.Info, dir string, creds *login.Credentials, es *EventStream, log logf, onStale func(id string)) *Bridge {
	// 发送节流与限流退避（2026-09-16 新增）：sendmessage 走「最小间隔 +
	// 滑动窗口配额」，ret=-2 触发冷静期（15m 起、上限 30m）后自动重试，
	// 防连发打爆服务端频率限制（详见 ilink.SendThrottleOptions）。
	api := ilink.NewClient(creds.BaseURL, creds.Token)
	api.Logf = log
	api.SetSendThrottle(ilink.SendThrottleOptions{
		MinInterval:     time.Duration(cfg.SendMinIntervalMs) * time.Millisecond,
		WindowMax:       cfg.SendWindowMax,
		WindowDur:       time.Duration(cfg.SendWindowSec) * time.Second,
		LongWindowMax:   cfg.SendLongWindowMax,
		LongWindowDur:   time.Duration(cfg.SendLongWindowSec) * time.Second,
		LimitBase:       time.Duration(cfg.SendLimitBackoffSec) * time.Second,
		LimitMax:        time.Duration(cfg.SendLimitMaxBackoffSec) * time.Second,
		LimitMaxRetries: cfg.SendLimitMaxRetries,
	})
	b := &Bridge{
		cfg:           cfg,
		info:          info,
		dir:           dir,
		api:           api,
		es:            es,
		log:           log,
		onStale:       onStale,
		processed:     map[string]struct{}{},
		ctxTokens:     map[string]string{},
		typingTickets: map[string]string{},
		queue:         make(chan pendingMsg, 100),
		stopCh:        make(chan struct{}),
	}
	var st State
	if store.ReadJSON(filepath.Join(dir, "state.json"), &st) {
		b.getUpdatesBuf = st.GetUpdatesBuf
		for _, id := range st.ProcessedIDs {
			if _, ok := b.processed[id]; !ok {
				b.processed[id] = struct{}{}
				b.processedQ = append(b.processedQ, id)
			}
		}
		for k, v := range st.ContextTokens {
			b.ctxTokens[k] = v
		}
	}
	if cfg.RawDump {
		if rl, err := newRawLogger(cfg.DataDir, info.ID, log); err != nil {
			log("raw 抓样初始化失败: %v", err)
		} else {
			b.raw = rl
		}
	}
	return b
}

// Start 启动桥循环（pollLoop + workLoop + 事件订阅）。
func (b *Bridge) Start() {
	b.unwatch = b.es.Watch(b.info.ConvID, b.onStreamEvent)
	go b.guard("pollLoop", b.pollLoop)
	go b.guard("workLoop", b.workLoop)
	if b.raw != nil {
		b.log("raw 抓样 → %s", b.raw.path)
	}
	b.log("桥循环已启动（conv=%s workspace=%s）", b.info.ConvID, b.resolveWorkspace())
}

// Stop 停止桥循环（幂等；不阻塞等待在网络 I/O 上的 goroutine 退出）。
func (b *Bridge) Stop() {
	b.stopOnce.Do(func() {
		close(b.stopCh)
		// 中断等待中的发送节流/冷静期睡眠（不阻塞桥退出）
		b.api.Close()
		if b.unwatch != nil {
			b.unwatch()
		}
		b.saveNow()
		if b.raw != nil {
			b.raw.close()
		}
		b.log("桥循环已停止（id=%s）", b.info.ID)
	})
}

// guard panic 隔离（单桥异常不扩散）。
func (b *Bridge) guard(name string, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			b.log("[%s] panic（已隔离）: %v", name, p)
		}
	}()
	fn()
}

func (b *Bridge) stopping() bool {
	select {
	case <-b.stopCh:
		return true
	default:
		return false
	}
}

// resolveWorkspace 会话工作区：账号指定 > 全局配置 > 从 data-dir 推断。
func (b *Bridge) resolveWorkspace() string {
	if b.info.WorkspaceRoot != "" {
		return b.info.WorkspaceRoot
	}
	if b.cfg.WorkspaceRoot != "" {
		return b.cfg.WorkspaceRoot
	}
	// data-dir 通常为 <workspace>/.pair/wechat-bridge → 上溯两级
	if b.dir != "" {
		p := filepath.Dir(filepath.Dir(b.dir))
		if p != "." && p != string(filepath.Separator) {
			return p
		}
	}
	return b.dir
}

// ── 收消息（长轮询）────────────────────────────────────────────────

func (b *Bridge) pollLoop() {
	failures := 0
	staleRetries := 0
	for !b.stopping() {
		resp, err := b.api.GetUpdates(b.getUpdatesBuf, b.cfg.LongPollTimeoutSec)
		if err != nil {
			if b.stopping() {
				break
			}
			failures++
			b.log("getUpdates 异常（%d/3）: %v", failures, err)
			if failures >= 3 {
				time.Sleep(30 * time.Second)
				failures = 0
			} else {
				time.Sleep(2 * time.Second)
			}
			continue
		}
		if resp.Ret != 0 || resp.Errcode != 0 {
			if resp.Ret == -14 || resp.Errcode == -14 {
				staleRetries++
				if staleRetries <= len(staleRetryWaits) {
					wait := staleRetryWaits[staleRetries-1]
					b.log("会话失效（-14，第 %d/%d 次），%s 后自动重试（观察 token 是否恢复）…",
						staleRetries, len(staleRetryWaits), wait)
					if b.sleepInterruptible(wait) {
						break
					}
					continue
				}
				b.log("会话持续失效（-14 重试 %d 次仍未恢复），请求重新登录…", len(staleRetryWaits))
				if b.onStale != nil {
					b.onStale(b.info.ID)
				}
				return // 停止本桥循环；重新登录成功后宿主会用新凭据重建
			}
			failures++
			b.log("getUpdates 错误 ret=%d errcode=%d errmsg=%s（%d/3）",
				resp.Ret, resp.Errcode, resp.Errmsg, failures)
			if b.sleepInterruptible(sleepOr(failures >= 3, 30*time.Second, 2*time.Second)) {
				break
			}
			if failures >= 3 {
				failures = 0
			}
			continue
		}
		failures = 0
		staleRetries = 0
		if resp.GetUpdatesBuf != "" {
			b.mu.Lock()
			b.getUpdatesBuf = resp.GetUpdatesBuf
			b.mu.Unlock()
			b.queueSave()
		}
		for _, raw := range resp.Msgs {
			b.enqueue(raw)
		}
	}
	b.log("pollLoop 退出")
}

func sleepOr(cond bool, a, b time.Duration) time.Duration {
	if cond {
		return a
	}
	return b
}

func (b *Bridge) sleepInterruptible(d time.Duration) bool {
	select {
	case <-b.stopCh:
		return true
	case <-time.After(d):
		return false
	}
}

// enqueue 过滤 + 入队（只处理 USER 消息；去重；维护 context_token）。
func (b *Bridge) enqueue(raw json.RawMessage) {
	b.raw.write(raw) // raw 抓样（默认开；非 USER/BOT 也落，供协议排查）
	var m ilink.RawMsg
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	if m.MessageType != 1 { // 仅 USER（忽略 BOT 等）
		return
	}

	msgID := string(m.MessageID)
	if msgID == "" {
		msgID = string(m.Seq)
	}
	if msgID != "" {
		b.mu.Lock()
		if _, dup := b.processed[msgID]; dup {
			b.mu.Unlock()
			return
		}
		b.processed[msgID] = struct{}{}
		b.processedQ = append(b.processedQ, msgID)
		if len(b.processedQ) > processedMax {
			cut := len(b.processedQ) - processedKeep
			for _, id := range b.processedQ[:cut] {
				delete(b.processed, id)
			}
			b.processedQ = append([]string(nil), b.processedQ[cut:]...)
		}
		b.mu.Unlock()
	}

	from := m.FromUserID
	if from != "" && m.ContextToken != "" {
		b.mu.Lock()
		if b.ctxTokens[from] != m.ContextToken {
			b.ctxTokens[from] = m.ContextToken
		}
		b.mu.Unlock()
	}

	// 解析 item：文本（含引用前缀）/ 媒体
	var textSB strings.Builder
	hasMedia := false
	var mediaItems []ilink.MsgItem
	for _, item := range m.ItemList {
		switch item.Type {
		case 1:
			if item.TextItem != nil {
				textSB.WriteString(item.TextItem.Text)
			}
			if item.RefMsg != nil {
				var parts []string
				if item.RefMsg.Title != "" {
					parts = append(parts, item.RefMsg.Title)
				}
				if rmi := item.RefMsg.MessageItem; rmi != nil && rmi.Type == 1 && rmi.TextItem != nil {
					if rb := rmi.TextItem.Text; rb != "" {
						parts = append(parts, rb)
					}
				}
				if len(parts) > 0 {
					textSB.Reset()
					textSB.WriteString("[引用: " + strings.Join(parts, " | ") + "]\n" + textSB.String())
				}
			}
		case 2, 3, 4, 5:
			hasMedia = true
			mediaItems = append(mediaItems, item)
		}
	}
	text := strings.TrimSpace(textSB.String())

	if text == "" && hasMedia {
		b.push(pendingMsg{from: from, media: mediaItems})
		b.queueSave()
		return
	}
	if text == "" || from == "" {
		return
	}
	b.log("📥 收到消息 from=%s：%s", from, truncateRunes(text, 80))
	b.push(pendingMsg{from: from, text: text, media: mediaItems})
	b.queueSave()
}

// push 入队（队列满时阻塞；消息仍在服务器游标之后，不丢）。
func (b *Bridge) push(m pendingMsg) {
	select {
	case b.queue <- m:
	case <-b.stopCh:
	}
}

// ── 处理队列（串行）────────────────────────────────────────────────

func (b *Bridge) workLoop() {
	for {
		select {
		case <-b.stopCh:
			b.log("workLoop 退出")
			return
		case item := <-b.queue:
			b.handleOne(item)
		}
	}
}

func (b *Bridge) handleOne(item pendingMsg) {
	b.log("■ 开始处理（队列剩余 %d 条）", len(b.queue))

	// 媒体预处理：下载/解密/落盘 → 描述并入投喂文本
	if len(item.media) > 0 {
		desc, okCount := b.processMedia(item.media)
		switch {
		case okCount > 0:
			prefix := "【微信媒体】（已自动保存到本机，供你读取）\n" + desc
			if item.text != "" {
				item.text = prefix + "\n【用户留言】\n" + item.text
			} else {
				item.text = prefix
			}
		case item.text != "":
			item.text = "【微信媒体】接收失败（下载/解密异常），未能取得内容：\n" + desc + "\n【用户留言】\n" + item.text
		default:
			b.sendToWeixin(item.from, "（微信桥）媒体接收失败，未取得可用内容，请重试或改发文本。")
			return
		}
	}

	// 0)「正在输入」占位（即时反馈；失败静默忽略）
	typing := b.beginTyping(item.from)
	defer typing.Stop()

	var askOnce sync.Once
	notifyAskUser := func(q string) {
		askOnce.Do(func() {
			snippet := truncateRunes(q, askUserSnippet)
			msg := "（微信桥）任务需要你的确认/输入：\n" + orDefault(snippet, "(需要交互输入)") +
				"\n\n请到 PairCode 界面打开「微信桥」会话回复；完成后最终结果会发回这里。"
			go b.sendToWeixin(item.from, msg)
		})
	}

	// 1) 基线 idx
	baseline := (&paircode.Waiter{WorkspaceRoot: b.resolveWorkspace(), ConvID: b.info.ConvID}).BaselineIdx()

	// 1.5) ★ 忙对齐（2026-09-17）：投喂前若会话已有任务在跑（如 PC 端触发/历史
	//      遗留的运行），本消息将在宿主侧排队——先进入 drain 相位：丢弃该任务
	//      的输出，等它结束（done / 状态校正）后再把后续输出当作本消息的回复，
	//      避免把别的任务的输出误当回复发回微信（「微信端收到 PC 端内容」根因）。
	busy := b.es.ConvActive(b.info.ConvID)
	var drainCh chan struct{}
	if busy {
		drainCh = make(chan struct{})
		b.mu.Lock()
		b.draining = true
		b.drainDone = drainCh
		b.mu.Unlock()
		// 复核：检测到忙之后、drain 建立之前，若前序任务恰已结束（done 已被处理、
		// 状态机已校正空闲），立即结束 drain——避免把本消息（即将投喂/启动）的输出误丢。
		if !b.es.ConvActive(b.info.ConvID) {
			b.endDrain("建立后复核已空闲")
		}
		b.log("[stream] 会话忙（有任务运行中）：先等其结束，再接收本消息回复（排队对齐）")
		defer b.endDrain("handleOne 退出兜底")
	}

	// 2) 挂载流式会话（必须在投喂之前）
	var session *stream.Session
	if b.cfg.StreamEnabled && b.es.Connected() {
		session = stream.New(stream.Options{
			FirstMinChars:  b.cfg.StreamFirstMinChars,
			FirstMaxWaitMs: b.cfg.StreamFirstMaxWaitMs,
			NextMinChars:   b.cfg.StreamNextMinChars,
			NextMaxWaitMs:  b.cfg.StreamNextMaxWaitMs,
			SoftLimit:      b.cfg.StreamSoftLimit,
			OnFlush: func(text string, meta stream.FlushMeta) {
				if !b.sendToWeixin(item.from, text) {
					b.log("[stream] 第 %d 片发送失败（%d 字符）", meta.N, meta.Len)
				}
			},
			OnToolCall: func(tool, args string) {
				if tool == "ask_user" {
					notifyAskUser(extractAskQuestion(args))
				}
			},
			OnDone: func(full string, s stream.Summary) {
				b.log("[stream] 流式完成（%d 片 / %d 字符 / toolCalls=%d）",
					s.Flushes, runeLen(s.Text), s.ToolCalls)
			},
			Logf: b.log,
		})
		b.mu.Lock()
		b.activeSession = session
		b.mu.Unlock()
		defer func() {
			b.mu.Lock()
			b.activeSession = nil
			b.mu.Unlock()
		}()
	}

	// 2.5) ★ done 即发快路径登记（2026-09-17）：非流式模式下 done 事件自带最终
	//      文本（自然终止时 loop 发射 Content=assistant.Content，与 JSONL 候选
	//      同源、条件等价），收到即可直接发送，省去 JSONL 路的轮询发现 +
	//      静默确认（实测 ~16s）。必须在投喂之前登记：回合极快结束时 done 也
	//      不会错过；handleOne 退出兜底清理（覆盖投喂失败/超时等全部出口）。
	//      流式模式不参与：onStreamEvent 对 activeSession!=nil 不分发快路径。
	doneFastCh := make(chan stream.Event, 1)
	b.mu.Lock()
	b.replyDoneCh = doneFastCh
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		if b.replyDoneCh == doneFastCh {
			b.replyDoneCh = nil
		}
		b.mu.Unlock()
	}()

	// 3) 投喂 PairCode
	if err := paircode.Send(b.cfg.PairCodeURL, b.info.ConvID, b.resolveWorkspace(),
		item.text, sendTimeoutSec*time.Second); err != nil {
		b.log("投喂失败: %v", err)
		b.sendToWeixin(item.from, "（微信桥）消息投喂失败："+truncateRunes(err.Error(), 300))
		if session != nil {
			session.Abort()
		}
		return
	}
	b.log("已投喂 PairCode（baseline idx=%d，流式=%v）", baseline, session != nil)

	// 4) 双路等待（WS 流式主路 + JSONL 兜底）
	b.awaitReply(item, baseline, session, notifyAskUser, drainCh, doneFastCh)
}

// onStreamEvent 事件流分发（按 convId 路由后进入活动会话）。
// ★ 忙对齐（2026-09-17）：drain 相位丢弃「前序任务」的全部输出（不发微信），
// 直到 done（或状态校正为空闲）后才开始接收本消息的回复。
func (b *Bridge) onStreamEvent(ev stream.Event) {
	b.mu.Lock()
	draining := b.draining
	s := b.activeSession
	doneCh := b.replyDoneCh
	b.mu.Unlock()
	if draining {
		if ev.Type == "done" {
			b.endDrain("done 事件")
		}
		return
	}
	if s != nil {
		// 流式模式：done 交由流式会话收尾（其内容已由 OnFlush 逐片发出，快路径不参与防重复）
		s.Feed(ev)
		return
	}
	// ★ done 即发快路径（2026-09-17）：非流式模式下 done 事件自带最终文本——
	// 直达等待者即刻发送，省去 JSONL 路的轮询发现 + 静默确认（实测 ~16s）。
	// 无等待者（未登记/已消耗）时忽略：常规 JSONL 兜底路仍会送达。
	if ev.Type == "done" && doneCh != nil {
		select {
		case doneCh <- ev:
		default:
		}
	}
}

// endDrain 结束 drain 相位（幂等；done 事件、状态校正或 handleOne 退出兜底触发）。
func (b *Bridge) endDrain(reason string) {
	b.mu.Lock()
	if !b.draining {
		b.mu.Unlock()
		return
	}
	b.draining = false
	if b.drainDone != nil {
		close(b.drainDone)
		b.drainDone = nil
	}
	b.mu.Unlock()
	b.log("[stream] 前序任务结束（%s）：开始接收本消息的回复", reason)
}

type waitResult struct {
	src string // jsonl | stream | timeout | stopped
	r   paircode.Result
	s   stream.Summary
}

// waitDrain 等待 drain（前序任务）结束。返回 true 可继续正式等待；false 放弃。
// 出口：① drainCh 关闭（done 事件经 endDrain）；② 每 1.5s 检查事件状态机，
// 被 status 快照校正为空闲（兜底 done 丢失/断线）；③ deadline 超时（给用户
// 提示后返回 false，绝不无界阻塞）；④ 桥停止。
func (b *Bridge) waitDrain(item pendingMsg, drainCh chan struct{}, deadline time.Time) bool {
	ticker := time.NewTicker(1500 * time.Millisecond)
	deadlineTimer := time.NewTimer(time.Until(deadline))
	defer ticker.Stop()
	defer deadlineTimer.Stop()
	for {
		select {
		case <-drainCh:
			return true
		case <-ticker.C:
			if !b.es.ConvActive(b.info.ConvID) {
				b.endDrain("状态校正为空闲")
				return true
			}
		case <-deadlineTimer.C:
			b.log("等待前序任务结束超时（drain）")
			b.sendToWeixin(item.from, "（微信桥）该会话有任务运行较久仍未结束，本消息暂未处理。请稍后重试，或到 PairCode 界面查看进度。")
			return false
		case <-b.stopCh:
			return false
		}
	}
}

// awaitReply 双路等待：
//   - WS 流式 done 先到：内容已由 OnFlush 全部发出 → 直接完成（JSONL 路后台校验一致性）；
//   - done 即发（非流式快路径）：done 事件自带最终文本 → 直接发送，省轮询+静默；
//   - JSONL 先到（WS 半死/断流）：中止流式，按前缀比对补发未发出的部分；
//   - 超时：提示用户到界面查看。
//   - ★ 忙对齐（2026-09-17）：drainCh 非 nil（投喂时会话忙）时，先等前序任务
//     结束并重锚基线，再做正式双路等待——前序任务的输出不会进入本消息回复。
func (b *Bridge) awaitReply(item pendingMsg, baseline int64, session *stream.Session, notifyAskUser func(string), drainCh chan struct{}, doneFastCh chan stream.Event) {
	timeout := time.Duration(b.cfg.ReplyTimeoutMs) * time.Millisecond

	// ★ 忙对齐：先等前序任务结束（drain），再重锚基线、进入正式双路等待。
	// drain 等待与正式等待共享同一超时上限 timeout；超时由 waitDrain 内部
	// deadline 计时器触发（给用户提示并返回 false），不会无界阻塞。
	if drainCh != nil {
		if !b.waitDrain(item, drainCh, time.Now().Add(timeout)) {
			if session != nil {
				session.Abort()
			}
			return
		}
		// 重锚：done 后短暂等待（前序任务收尾落盘可能稍滞后），再取新基线。
		// 本消息若已开始落盘，其行 idx 只会更大（更早的行不会被误扫），不会漏。
		time.Sleep(500 * time.Millisecond)
		baseline = (&paircode.Waiter{WorkspaceRoot: b.resolveWorkspace(), ConvID: b.info.ConvID}).BaselineIdx()
		b.log("[stream] 前序任务已结束，重锚基线 idx=%d（其输出不会进入回复）", baseline)
	}

	// ★ 快路径窗开启：清掉此前（drain / 投喂前）可能误入的陈旧 done 事件——
	//   只接受本回合的 done（本回合的 done 只能在其运行结束后到达）。
	select {
	case <-doneFastCh:
	default:
	}

	ch := make(chan waitResult, 3)

	waiter := &paircode.Waiter{
		WorkspaceRoot: b.resolveWorkspace(),
		ConvID:        b.info.ConvID,
		PollMs:        b.cfg.ReplyPollMs,
		QuietMs:       b.cfg.ReplyQuietMs,
		TimeoutMs:     b.cfg.ReplyTimeoutMs,
		Logf:          b.log,
	}
	go func() {
		r := waiter.WaitForReply(baseline, notifyAskUser)
		ch <- waitResult{src: "jsonl", r: r}
	}()
	if session != nil {
		go func() {
			s := <-session.Done()
			ch <- waitResult{src: "stream", s: s}
		}()
	}

	waitStart := time.Now()
	var w waitResult
	zeroRetry := 0 // 「0 片完成」转等 JSONL 的计数上限（防异常路径循环；实测至多 1 次）
	for {
		select {
		case w = <-ch:
			// 0 片完成 = 未发出任何内容（如误报 error 终结的现场：2026-09-16 忙错误
			// 导致 WS=0 而 JSONL 已有 1413 字符全文）——不能宣称「已全部发出」，
			// 转等 JSONL 兜底路；达上限按超时语义收口（提示用户到界面查看）。
			// Summary.Flushes 由 flushLocked/finalizeLocked 累加、Aborted 仅在
			// Abort 路径为 true（stream 包契约），此处判定字段可靠。
			if w.src == "stream" && w.s.Flushes == 0 && runeLen(w.s.Text) == 0 && !w.s.Aborted {
				if zeroRetry < 2 {
					zeroRetry++
					b.log("[stream] 流式 0 片完成（未发出内容），转等 JSONL 兜底（第 %d 次）", zeroRetry)
					continue
				}
				w = waitResult{src: "timeout"}
			}
		case ev := <-doneFastCh:
			// ★ done 即发（2026-09-17）：done 事件自带最终文本 → 直接发送，
			//   省去 JSONL 路的轮询发现 + 15s 静默确认（实测 ~16s）。
			//   Content 为空（tool_budget/blocked 等）不发送，转常规兜底路。
			if txt := strings.TrimSpace(ev.Content); txt != "" {
				b.log("[stream] done 即发快路径触发（%d 字符）", runeLen(txt))
				if b.sendToWeixin(item.from, txt) {
					b.log("↩ 回复已发送（done 即发），共 %d 字符", runeLen(txt))
					return
				}
				b.log("[stream] done 即发发送失败，转常规兜底路")
			}
		case <-time.After(timeout - time.Since(waitStart)):
			w = waitResult{src: "timeout"}
		case <-b.stopCh:
			w = waitResult{src: "stopped"}
		}
		break
	}

	switch w.src {
	case "stream":
		b.log("↩ 回复已通过流式全部发出（%d 字符 / %d 片）", runeLen(w.s.Text), w.s.Flushes)
		// JSONL 路后台一致性校验（仅记录）
		go func() {
			r := <-ch
			if r.src != "jsonl" || r.r.Text == "" {
				return
			}
			if r.r.Text != w.s.Text {
				b.log("[stream] 一致性校验：WS=%d 字符 vs JSONL=%d 字符（不一致，仅记录）",
					runeLen(w.s.Text), runeLen(r.r.Text))
			} else {
				b.log("[stream] 一致性校验通过（WS 与 JSONL 文本一致）")
			}
		}()
	case "jsonl":
		if session != nil {
			session.Abort()
		}
		if w.r.Timeout {
			b.sendToWeixin(item.from, "（微信桥）处理已超过 30 分钟仍未完成。请到 PairCode 界面查看进度；完成后可在界面继续对话。")
			b.log("等待回复超时（30 分钟）")
			return
		}
		text := w.r.Text
		if text == "" {
			b.log("JSONL 兜底未取得文本（流式亦未完成）")
			return
		}
		sent := ""
		if session != nil {
			sent = session.SentText()
		}
		toSend := text
		switch {
		case sent == "":
			// 无流式已发内容：全量发送
		case strings.HasPrefix(text, sent):
			toSend = text[len(sent):]
			if toSend != "" {
				b.log("[stream] JSONL 兜底补发 %d 字符（已流式发出 %d）", runeLen(toSend), runeLen(sent))
			} else {
				b.log("[stream] JSONL 兜底：内容与流式发送一致，无需补发")
			}
		default:
			b.log("[stream] JSONL 文本不以已发内容为前缀（sent=%d jsonl=%d），全量补发",
				runeLen(sent), runeLen(text))
		}
		if toSend != "" {
			ok := b.sendToWeixin(item.from, toSend)
			b.log("↩ 回复已发送（兜底，%s），共 %d 字符", map[bool]string{true: "成功", false: "失败"}[ok], runeLen(toSend))
		}
	case "timeout":
		if session != nil {
			session.Abort()
		}
		b.sendToWeixin(item.from, "（微信桥）处理已超过 30 分钟仍未完成。请到 PairCode 界面查看进度；完成后可在界面继续对话。")
		b.log("等待回复超时（30 分钟）")
	case "stopped":
		if session != nil {
			session.Abort()
		}
	}
}

// ── 发送 ─────────────────────────────────────────────────────────────

// processMedia 下载/解密/保存媒体条目（收方向），返回（投喂描述, 成功条数）。
func (b *Bridge) processMedia(items []ilink.MsgItem) (string, int) {
	dir := filepath.Join(b.cfg.DataDir, "inbound")
	var lines []string
	okCount := 0
	for i := range items {
		path, note, err := media.Inbound(dir, &items[i], b.cfg.CDNBaseURL)
		if err != nil {
			b.log("媒体处理失败（type=%d）: %v", items[i].Type, err)
		} else {
			okCount++
			b.log("📥 媒体已保存：%s", path)
		}
		if note == "" {
			note = "（无描述）"
		}
		lines = append(lines, "- "+note)
	}
	return strings.Join(lines, "\n"), okCount
}

// SendTo 主动向指定用户发送文本（Agent 工具 wechat_send / 壳经本地 HTTP 调用）。
// 需要该联系人已与桥交互过（存在 context_token，即曾给桥发过消息）。
func (b *Bridge) SendTo(userID, text string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("收件人（to）不能为空")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("消息文本不能为空")
	}
	if !b.sendToWeixin(userID, text) {
		return fmt.Errorf("发送失败：该联系人暂无有效会话（需先给桥发过消息以建立 context_token）")
	}
	return nil
}

// SendMediaTo 主动发送媒体文件（wechat_send 的 file 参数 / 壳经本地 HTTP 调用）。
// 相对路径按会话工作区解析；类型按扩展名推断（图片/视频/其他文件）。
func (b *Bridge) SendMediaTo(userID, filePath, caption string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return fmt.Errorf("收件人（to）不能为空")
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("文件路径（file）不能为空")
	}
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(b.resolveWorkspace(), filePath)
	}
	if fi, err := os.Stat(filePath); err != nil || fi.IsDir() {
		return fmt.Errorf("文件不存在或不可读: %s", filePath)
	}
	kind, mediaType, ok := media.KindFromPath(filePath)
	if !ok {
		return fmt.Errorf("无法按扩展名识别文件类型: %s（支持图片/视频/常见文档）", filePath)
	}
	b.mu.Lock()
	token := b.ctxTokens[userID]
	b.mu.Unlock()
	if token == "" {
		return fmt.Errorf("该联系人暂无有效会话（需先给桥发过消息以建立 context_token）")
	}

	up, err := media.Upload(b.api, filePath, userID, mediaType, b.cfg.CDNBaseURL)
	if err != nil {
		return fmt.Errorf("媒体上传失败: %w", err)
	}
	itemMap, err := media.BuildItem(kind, up, filepath.Base(filePath))
	if err != nil {
		return err
	}
	if strings.TrimSpace(caption) != "" {
		b.sendToWeixin(userID, caption)
	}
	resp, err := b.api.SendItem(userID, itemMap, token)
	if err != nil {
		return fmt.Errorf("媒体发送失败: %w", err)
	}
	if resp.Ret == -14 || resp.Errcode == -14 {
		b.mu.Lock()
		delete(b.ctxTokens, userID)
		b.mu.Unlock()
		b.queueSave()
		return fmt.Errorf("会话已过期（-14），请等待对方重新发消息后重试")
	}
	if resp.Ret != 0 || resp.Errcode != 0 {
		return fmt.Errorf("发送响应异常 ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
	}
	b.log("📤 媒体已发送（%s，%s）", kind, filePath)
	return nil
}

// Contacts 已知联系人列表（有过 context_token 的微信用户；稳定排序）。
func (b *Bridge) Contacts() []map[string]any {
	b.mu.Lock()
	ids := make([]string, 0, len(b.ctxTokens))
	for id := range b.ctxTokens {
		ids = append(ids, id)
	}
	b.mu.Unlock()
	sort.Strings(ids)
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{"id": id, "hasSession": true})
	}
	return out
}

// sendToWeixin 发送文本（按用户 context_token；支持长文本分片）。
func (b *Bridge) sendToWeixin(toUserID, text string) bool {
	b.mu.Lock()
	token := b.ctxTokens[toUserID]
	b.mu.Unlock()
	if token == "" {
		b.log("无 context_token（%s），无法发送", toUserID)
		return false
	}
	chunks := splitText(text, b.cfg.TextChunkLimit)
	for i, chunk := range chunks {
		resp, err := b.api.SendText(toUserID, chunk, token)
		if err != nil {
			b.log("发送失败（分片 %d/%d）: %v", i+1, len(chunks), err)
			return false
		}
		if resp.Ret == -14 || resp.Errcode == -14 {
			b.log("发送返回 -14：会话过期，丢弃 context_token（等下次消息刷新）")
			b.mu.Lock()
			delete(b.ctxTokens, toUserID)
			b.mu.Unlock()
			b.queueSave()
			return false
		}
		if resp.Ret == -2 || resp.Errcode == -2 {
			// -2 已在 ilink 层做过冷静期重试（仍失败）：判定本片未送达
			b.log("发送失败（分片 %d/%d）：服务端限流重试未恢复（ret=-2），该片未送达", i+1, len(chunks))
			return false
		}
		if resp.Ret != 0 || resp.Errcode != 0 {
			b.log("发送响应异常 ret=%d errcode=%d errmsg=%s", resp.Ret, resp.Errcode, resp.Errmsg)
		} else {
			b.log("📤 已发送（%d/%d 片，%d 字符）", i+1, len(chunks), runeLen(chunk))
		}
		if i < len(chunks)-1 {
			time.Sleep(msgSendGapMs * time.Millisecond)
		}
	}
	return true
}

// splitText 长文本分片（优先段落/换行处切分；limit 按 rune 计）。
func splitText(text string, limit int) []string {
	if limit <= 0 {
		limit = 4000
	}
	rs := []rune(text)
	if len(rs) <= limit {
		return []string{text}
	}
	var chunks []string
	rest := rs
	for len(rest) > 0 {
		if len(rest) <= limit {
			chunks = append(chunks, string(rest))
			break
		}
		cut := -1
		for p := limit; p >= 1; p-- { // 段落切分：最后一个双换行的起点（≤limit）
			if p+1 < len(rest) && rest[p] == '\n' && rest[p+1] == '\n' {
				cut = p
				break
			}
		}
		if cut <= 0 {
			for p := limit; p >= 1; p-- { // 退化为单换行
				if rest[p] == '\n' {
					cut = p
					break
				}
			}
		}
		if cut <= 0 {
			cut = limit
		}
		chunks = append(chunks, string(rest[:cut]))
		rest = rest[cut:]
		for len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
	}
	return chunks
}

// ── 「正在输入」指示 ─────────────────────────────────────────────────

type typingHandle struct {
	stopOnce sync.Once
	stopFn   func()
}

// Stop 停止「正在输入」（发送 status=2；失败忽略）。
func (h *typingHandle) Stop() {
	if h == nil || h.stopFn == nil {
		return
	}
	h.stopOnce.Do(h.stopFn)
}

// beginTyping 开始「正在输入」指示；返回句柄（Stop 幂等）。
// 失败静默忽略（不影响主流程）。
func (b *Bridge) beginTyping(userID string) *typingHandle {
	h := &typingHandle{}
	if !b.cfg.TypingEnabled {
		return h
	}
	var mu sync.Mutex
	ticket := b.getTypingTicket(userID)
	stopped := false

	call := func(status int) {
		mu.Lock()
		st := stopped
		tk := ticket
		mu.Unlock()
		if st && status == 1 {
			return
		}
		if tk == "" {
			b.mu.Lock()
			ctxToken := b.ctxTokens[userID]
			b.mu.Unlock()
			resp, err := b.api.GetConfig(userID, ctxToken)
			if err != nil {
				b.log("typing getConfig 异常（忽略）: %v", err)
				return
			}
			if resp.Ret == 0 {
				tk = resp.TypingTicket
				if tk != "" {
					mu.Lock()
					ticket = tk
					mu.Unlock()
					b.setTypingTicket(userID, tk)
				}
			} else {
				b.log("typing getConfig 失败 ret=%d errcode=%d", resp.Ret, resp.Errcode)
			}
		}
		if tk == "" {
			return
		}
		resp, err := b.api.SendTyping(userID, tk, status)
		if err != nil {
			b.log("typing 调用异常（忽略）: %v", err)
			return
		}
		if resp.Ret != 0 || resp.Errcode != 0 {
			b.log("sendTyping(%d) ret=%d errcode=%d %s", status, resp.Ret, resp.Errcode, resp.Errmsg)
			if status == 1 { // 失效重取
				mu.Lock()
				ticket = ""
				mu.Unlock()
				b.deleteTypingTicket(userID)
			}
		}
	}

	call(1) // 即时反馈（同步；失败静默）
	ticker := time.NewTicker(time.Duration(b.cfg.TypingKeepaliveMs) * time.Millisecond)
	go func() {
		for range ticker.C {
			mu.Lock()
			st := stopped
			mu.Unlock()
			if st {
				return
			}
			call(1)
		}
	}()
	h.stopFn = func() {
		mu.Lock()
		stopped = true
		has := ticket != ""
		mu.Unlock()
		ticker.Stop()
		if !has {
			return
		}
		call(2)
	}
	return h
}

// ── 状态存取辅助 ─────────────────────────────────────────────────────

func (b *Bridge) getTypingTicket(userID string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.typingTickets[userID]
}

func (b *Bridge) setTypingTicket(userID, ticket string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.typingTickets[userID] = ticket
}

func (b *Bridge) deleteTypingTicket(userID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.typingTickets, userID)
}

// queueSave 防抖保存状态（500ms 合并）。
func (b *Bridge) queueSave() {
	b.saveMu.Lock()
	defer b.saveMu.Unlock()
	if b.saveTimer != nil {
		return
	}
	b.saveTimer = time.AfterFunc(saveDebounceMs*time.Millisecond, func() {
		b.saveMu.Lock()
		b.saveTimer = nil
		b.saveMu.Unlock()
		b.saveNow()
	})
}

// saveNow 立即持久化状态（原子写）。
func (b *Bridge) saveNow() {
	b.mu.Lock()
	state := State{
		GetUpdatesBuf: b.getUpdatesBuf,
		ProcessedIDs:  append([]string(nil), b.processedQ...),
		ContextTokens: make(map[string]string, len(b.ctxTokens)),
	}
	for k, v := range b.ctxTokens {
		state.ContextTokens[k] = v
	}
	b.mu.Unlock()
	if err := store.WriteJSONAtomic(filepath.Join(b.dir, "state.json"), state); err != nil {
		b.log("状态保存失败: %v", err)
	}
}

// ── 工具 ─────────────────────────────────────────────────────────────

func runeLen(s string) int { return len([]rune(s)) }

func truncateRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// extractAskQuestion 从 ask_user 工具参数中提取问题文本。
func extractAskQuestion(args string) string {
	var a struct {
		Question  string `json:"question"`
		Questions []struct {
			Question string `json:"question"`
		} `json:"questions"`
	}
	if json.Unmarshal([]byte(args), &a) != nil {
		return ""
	}
	if len(a.Questions) > 0 {
		parts := make([]string, 0, len(a.Questions))
		for _, q := range a.Questions {
			if q.Question != "" {
				parts = append(parts, q.Question)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return a.Question
}

var _ = fmt.Sprintf // 保留 fmt（防御性引用，便于调试扩展）
