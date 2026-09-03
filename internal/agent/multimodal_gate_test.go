package agent

import (
	"context"
	"testing"
)

// gateProvider 可配置多模态能力的测试 Provider（实现 Multimodal() bool 探测接口）。
type gateProvider struct{ multimodal bool }

func (m *gateProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, onChunk func(Chunk)) (Message, error) {
	return Message{}, nil
}

func (m *gateProvider) Name() string { return "gate-provider" }

func (m *gateProvider) Multimodal() bool { return m.multimodal }

func registerGateTool(t *testing.T, reg *Registry, name string, system bool) {
	t.Helper()
	reg.Register(&Tool{
		Name:        name,
		Description: "gate test",
		SystemTool:  system,
		Enabled:     true,
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			return "", nil
		},
	})
}

// TestApplyMultimodalToolGate 多模态门控：非视觉 Provider 禁用视觉依赖工具
// （含 SystemTool 恒可用项——门控在白名单收敛之后执行，须能覆盖），视觉
// Provider 恢复启用（防上一会话禁用残留）；普通工具不受影响；未注册工具
// 与 nil 参数安全跳过。
func TestApplyMultimodalToolGate(t *testing.T) {
	reg := NewRegistry()
	for _, n := range visionDependentTools {
		registerGateTool(t, reg, n, true) // SystemTool=true：模拟白名单恒可用
	}
	registerGateTool(t, reg, "read", false)

	// 非视觉 Provider：视觉依赖工具全部禁用
	ApplyMultimodalToolGate(reg, &gateProvider{multimodal: false})
	for _, n := range visionDependentTools {
		if reg.IsEnabled(n) {
			t.Errorf("非视觉模型下 %s 应被禁用", n)
		}
	}
	if !reg.IsEnabled("read") {
		t.Errorf("普通工具不应受门控影响")
	}

	// 视觉 Provider：全部恢复启用（防残留）
	ApplyMultimodalToolGate(reg, &gateProvider{multimodal: true})
	for _, n := range visionDependentTools {
		if !reg.IsEnabled(n) {
			t.Errorf("视觉模型下 %s 应恢复启用", n)
		}
	}

	// nil Provider → 一律禁用（保守：无能力证明就不给视觉工具）
	ApplyMultimodalToolGate(reg, nil)
	for _, n := range visionDependentTools {
		if reg.IsEnabled(n) {
			t.Errorf("nil Provider 下 %s 应被禁用", n)
		}
	}

	// 未注册工具 / nil Registry：安全不 panic
	reg2 := NewRegistry()
	ApplyMultimodalToolGate(reg2, &gateProvider{multimodal: false})
	ApplyMultimodalToolGate(nil, &gateProvider{multimodal: true})
}
