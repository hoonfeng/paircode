// launch_retry_test.go 会话忙排队重试的单测（2026-09-16 新增）。
package main

import (
	"errors"
	"testing"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
)

// 忙两次后空闲 → 排队等待后启动成功（start 只在空闲时被调用）。
func TestWaitIdleAndStartWaitsThenSucceeds(t *testing.T) {
	running := true
	var startCalls, sleepCalls int
	err := waitIdleAndStart("c1",
		func(string) bool { return running },
		func() error { startCalls++; return nil },
		func(d time.Duration) bool {
			sleepCalls++
			if sleepCalls >= 2 {
				running = false
			}
			return true
		},
		time.Now().Add(time.Minute),
		func(string, ...any) {},
	)
	if err != nil {
		t.Fatalf("期望成功，得 %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("start 调用次数 = %d，期望 1（空闲后才启动）", startCalls)
	}
	if sleepCalls != 2 {
		t.Fatalf("等待次数 = %d，期望 2", sleepCalls)
	}
}

// 启动竞态：查询空闲后 start 仍返回 ErrSessionRunning → 继续排队，最终成功。
func TestWaitIdleAndStartRetryOnRace(t *testing.T) {
	call := 0
	err := waitIdleAndStart("c2",
		func(string) bool { return false }, // 始终空闲
		func() error {
			call++
			if call == 1 {
				return agent.ErrSessionRunning // 竞态：启动瞬间又被占
			}
			return nil
		},
		func(time.Duration) bool { return true },
		time.Now().Add(time.Minute),
		func(string, ...any) {},
	)
	if err != nil {
		t.Fatalf("期望最终成功，得 %v", err)
	}
	if call != 2 {
		t.Fatalf("start 调用次数 = %d，期望 2（含 1 次竞态重试）", call)
	}
}

// 非忙错误：直接失败，不排队不等待。
func TestWaitIdleAndStartFailsFastOnOtherError(t *testing.T) {
	sentinel := errors.New("创建 agent 循环失败")
	err := waitIdleAndStart("c3",
		func(string) bool { return false },
		func() error { return sentinel },
		func(time.Duration) bool { t.Fatal("非忙错误不应等待"); return true },
		time.Now().Add(time.Minute),
		func(string, ...any) {},
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("期望哨兵错误，得 %v", err)
	}
}

// 超时：持续忙 → 超时错误（忙时不应调用 start）。
func TestWaitIdleAndStartTimeout(t *testing.T) {
	err := waitIdleAndStart("c4",
		func(string) bool { return true }, // 一直忙
		func() error { t.Fatal("忙时不应调用 start"); return nil },
		func(time.Duration) bool { return true },
		time.Now().Add(-time.Second), // deadline 已过
		func(string, ...any) {},
	)
	if err == nil {
		t.Fatal("期望超时错误")
	}
}

// 取消：sleep 返回 false → 取消错误。
func TestWaitIdleAndStartCancel(t *testing.T) {
	err := waitIdleAndStart("c5",
		func(string) bool { return true },
		func() error { return nil },
		func(time.Duration) bool { return false },
		time.Now().Add(time.Minute),
		func(string, ...any) {},
	)
	if err == nil {
		t.Fatal("期望取消错误")
	}
}
