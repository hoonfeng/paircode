package agent

import (
	"context"
	"strings"
	"testing"
)

// TestUpdatePlan 验证 update_plan：返回步数+完成数；空计划报错；只读免审。
func TestUpdatePlan(t *testing.T) {
	r := NewRegistry()
	RegisterDefaultTools(r, t.TempDir())
	tool, ok := r.Get("update_plan")
	if !ok {
		t.Fatal("update_plan 未注册")
	}
	if !tool.ReadOnly || tool.RequiresApproval {
		t.Error("update_plan 应只读免审")
	}
	out, err := r.Execute(context.Background(), "update_plan",
		`{"plan":[{"step":"读代码","status":"done"},{"step":"改 bug","status":"in_progress"},{"step":"测试","status":"pending"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "3 步") || !strings.Contains(out, "1 完成") {
		t.Errorf("结果 = %q", out)
	}
	if _, err := r.Execute(context.Background(), "update_plan", `{"plan":[]}`); err == nil {
		t.Error("空计划应报错")
	}
}
