// launch_retry.go 会话忙时的启动排队（2026-09-16 新增）。
//
// 背景：handleChatSend 的同步段先落盘用户消息再立即返回 ok，Start 在
// launchConvRun 后台执行；若该会话已有运行中的任务（ErrSessionRunning
// ——如同一用户连发多条消息、长任务尚未结束），Start 被拒后仅经 WS 推
// error 事件，消息陷入「投喂成功但无人处理」，用户侧表现为回复丢失
// （2026-09-16 微信桥实测：桥投喂图片/语音连发场景）。
//
// 对策：ErrSessionRunning 不再直接报错，改为排队——轮询等待会话空闲
// （IsRunning=false）后用同一 opts 重试 Start；其他错误与超时仍走
// PushStartError。桥侧（wechat-bridge stream.go）对忙错误同样「不终结、
// 继续等待」，两侧配合保证消息最终被处理、流式回复照常发回。
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

const (
	// busyQueueMaxWait 排队等待上限：与微信桥的回复等待（30 分钟）对齐，
	// 超过则放弃并推错误事件。
	busyQueueMaxWait = 30 * time.Minute
	// busyQueuePollMin/Max 轮询间隔（逐次增长，降低空转）。
	busyQueuePollMin = 2 * time.Second
	busyQueuePollMax = 10 * time.Second
)

// waitIdleAndStart 排队等待 convID 空闲后重试启动（依赖注入，便于单测）。
//   - isRunning：查询会话是否运行中；
//   - start：执行一次 Start 尝试（内部自行处理 ctx 超时）；
//   - sleep：一次等待（返回 false = 被取消）；
//   - deadline：等待截止时刻；
//   - logf：日志。
//
// 返回 nil = 启动成功；否则为失败原因（非忙错误原样返回 / 超时 / 取消）。
func waitIdleAndStart(
	convID string,
	isRunning func(string) bool,
	start func() error,
	sleep func(time.Duration) bool,
	deadline time.Time,
	logf func(format string, args ...any),
) error {
	interval := busyQueuePollMin
	for attempt := 1; ; attempt++ {
		if !isRunning(convID) {
			err := start()
			if err == nil {
				logf("[chat] 排队重试启动成功 conv=%s（第 %d 次尝试）", convID, attempt)
				return nil
			}
			if !errors.Is(err, agent.ErrSessionRunning) {
				return err // 非忙错误：直接失败（交由调用方推送）
			}
			// 查询空闲但启动时又被占（竞态）：继续排队
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("排队等待超时（会话持续忙碌 ≥ %s）", busyQueueMaxWait.Round(time.Minute))
		}
		if !sleep(interval) {
			return errors.New("排队等待被取消")
		}
		if interval < busyQueuePollMax {
			interval += busyQueuePollMin
			if interval > busyQueuePollMax {
				interval = busyQueuePollMax
			}
		}
	}
}
