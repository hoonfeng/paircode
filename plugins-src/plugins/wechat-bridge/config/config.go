// Package config 集中管理微信桥的配置项。
//
// 优先级：命令行 flag（main 覆盖） > 环境变量（WX_BRIDGE_*，便于测试/多实例） > 默认值。
// 默认值对齐 M1 Node 原型（temp/wx-bridge/src/config.mjs）的实测参数。
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 微信桥运行配置。
type Config struct {
	// DataDir 数据根目录（通常 <workspace>/.pair/wechat-bridge）。
	DataDir string
	// WorkspaceRoot 默认会话工作区（账号级 WorkspaceRoot 优先；
	// 空 = 由桥从 DataDir 上溯推断）。
	WorkspaceRoot string
	// PairCodeURL PairCode 服务地址（投喂 /api/chat/send、事件流 ws://…/ws）。
	PairCodeURL string
	// Port 本地回环 HTTP 端口（0 = 由 localhttp 自动选择，实际端口见 hello 事件）。
	Port int

	// ── 回复等待（JSONL 兜底路）──
	ReplyPollMs    int // 会话 JSONL 轮询间隔
	ReplyQuietMs   int // 静默期：候选回复出现后无新行多久判定完成
	ReplyTimeoutMs int // 单轮等待上限

	// ── 流式回复（PairCode /ws 事件流，主路）──
	StreamEnabled bool

	// ── 流式切片参数（对齐 M1 stream.mjs）──
	StreamFirstMinChars  int // 首片最小字符（达到且遇句末即发）
	StreamFirstMaxWaitMs int // 首块起多久必发首片
	StreamNextMinChars   int // 后续片最小字符
	StreamNextMaxWaitMs  int // 上片起多久必发下一片
	StreamSoftLimit      int // 单片软上限（超限强切）

	// ── 「正在输入」指示 ──
	TypingEnabled     bool
	TypingKeepaliveMs int // 保活间隔（对齐官方 5s）

	// ── ilink 协议 ──
	BaseURL            string // 默认接入点（登录后可被 baseurl/redirect 覆盖）
	LongPollTimeoutSec int    // getupdates 长轮询秒数（超时视为空批）
	TextChunkLimit     int    // 长文本分片上限（sendmessage 单条文本）

	// ── 媒体 CDN ──
	// CDNBaseURL 微信 CDN 基础地址（媒体上传/下载 URL 拼接 fallback；
	// 服务端返回 full_url / upload_full_url 时优先用之）。
	CDNBaseURL string

	// ── 单实例 ──
	LockFile string // 实例锁文件路径（默认 <DataDir>/bridge.lock）

	// ── 调试 ──
	// RawDump raw 抓样：把 getupdates 原始消息按行落盘到 <DataDir>/raw/
	// （默认开；协议字段校准与故障复盘用，WX_BRIDGE_RAW_DUMP=0 可关闭）。
	RawDump bool
}

// Default 返回默认配置（含 WX_BRIDGE_* 环境变量覆盖）。
func Default() Config {
	cfg := Config{
		PairCodeURL: "http://127.0.0.1:9090",
		Port:        9097,

		ReplyPollMs:    1500,
		ReplyQuietMs:   15000,
		ReplyTimeoutMs: 30 * 60 * 1000,

		StreamEnabled: true,

		StreamFirstMinChars:  20,
		StreamFirstMaxWaitMs: 1000,
		StreamNextMinChars:   160,
		StreamNextMaxWaitMs:  2500,
		StreamSoftLimit:      1500,

		TypingEnabled:     true,
		TypingKeepaliveMs: 5000,

		BaseURL:            "https://ilinkai.weixin.qq.com",
		LongPollTimeoutSec: 35,
		TextChunkLimit:     4000,

		CDNBaseURL: "https://novac2c.cdn.weixin.qq.com/c2c",

		RawDump: true,
	}

	if v := os.Getenv("WX_BRIDGE_PAIRCODE_URL"); v != "" {
		cfg.PairCodeURL = v
	}
	if v := os.Getenv("WX_BRIDGE_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		}
	}
	switch strings.ToLower(os.Getenv("WX_BRIDGE_RAW_DUMP")) {
	case "0", "false", "off", "no":
		cfg.RawDump = false
	}
	if v := os.Getenv("WX_BRIDGE_CDN_BASE_URL"); v != "" {
		cfg.CDNBaseURL = v
	}
	return cfg
}

// LockPath 返回实例锁文件路径。
func (c Config) LockPath() string {
	if c.LockFile != "" {
		return c.LockFile
	}
	return filepath.Join(c.DataDir, "bridge.lock")
}
