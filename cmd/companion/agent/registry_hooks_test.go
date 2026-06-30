package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegistryHooks_BeforeShortCircuit(t *testing.T) {
	reg := NewRegistry()
	called := false
	reg.Register(&Tool{
		Name:    "ping",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { called = true; return "pong", nil },
	})
	// Before 短路：不执行 handler，返回 override
	reg.BeforeTool = func(ctx context.Context, name string, args map[string]any) (bool, string, error) {
		return false, "short-circuited", nil
	}
	out, err := reg.Execute(context.Background(), "ping", "")
	if err != nil {
		t.Fatalf("未预期错误: %v", err)
	}
	if out != "short-circuited" {
		t.Errorf("应返回 override 结果，got %q", out)
	}
	if called {
		t.Error("短路时 handler 不应被调用")
	}
}

func TestRegistryHooks_BeforeProceed(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:    "ping",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "pong", nil },
	})
	reg.BeforeTool = func(ctx context.Context, name string, args map[string]any) (bool, string, error) {
		return true, "", nil // 放行
	}
	out, err := reg.Execute(context.Background(), "ping", "")
	if err != nil || out != "pong" {
		t.Errorf("放行应执行 handler，got %q err=%v", out, err)
	}
}

func TestRegistryHooks_AfterObserved(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:    "ping",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "pong", nil },
	})
	var observedName, observedResult string
	var observedErr error
	var observedDur time.Duration
	reg.AfterTool = func(ctx context.Context, name string, args map[string]any, result string, err error, dur time.Duration) {
		observedName, observedResult, observedErr, observedDur = name, result, err, dur
	}
	reg.Execute(context.Background(), "ping", "")
	if observedName != "ping" || observedResult != "pong" || observedErr != nil {
		t.Errorf("AfterTool 观察值错误: name=%q result=%q err=%v", observedName, observedResult, observedErr)
	}
	if observedDur < 0 {
		t.Errorf("duration 不应为负: %v", observedDur)
	}
}

func TestRegistryHooks_OnToolErrorEnhance(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:    "fail",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", errors.New("原始错误") },
	})
	// OnToolError 增强错误信息
	reg.OnToolError = func(ctx context.Context, name string, args map[string]any, err error) (string, error) {
		return "", errors.New("增强: " + err.Error())
	}
	_, err := reg.Execute(context.Background(), "fail", "")
	if err == nil || !strings.Contains(err.Error(), "增强") {
		t.Errorf("应返回增强后的错误，got %v", err)
	}
}

func TestRegistryHooks_OnToolErrorSwallow(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:    "fail",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", errors.New("可恢复错误") },
	})
	// OnToolError 吞掉错误，转为成功结果
	reg.OnToolError = func(ctx context.Context, name string, args map[string]any, err error) (string, error) {
		return "已自动恢复", nil
	}
	out, err := reg.Execute(context.Background(), "fail", "")
	if err != nil {
		t.Errorf("应吞掉错误，got err=%v", err)
	}
	if out != "已自动恢复" {
		t.Errorf("应返回恢复结果，got %q", out)
	}
}

func TestRegistryHooks_AfterOnError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Tool{
		Name:    "fail",
		Handler: func(ctx context.Context, args map[string]any) (string, error) { return "", errors.New("boom") },
	})
	var observedErr error
	reg.AfterTool = func(ctx context.Context, name string, args map[string]any, result string, err error, dur time.Duration) {
		observedErr = err
	}
	reg.Execute(context.Background(), "fail", "")
	if observedErr == nil {
		t.Error("AfterTool 在出错时应观察到非 nil err")
	}
}
