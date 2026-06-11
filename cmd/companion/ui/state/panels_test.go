package state

import "testing"

// TestPanelsMove 面板互换：默认排布 + 移动=两区互换 + 保持一区一组（排列）。
func TestPanelsMove(t *testing.T) {
	p := DefaultPanels() // 左 files / 右 chat / 底 terminal
	if p.PanelIn(ZoneLeft) != "files" || p.PanelIn(ZoneRight) != "chat" || p.PanelIn(ZoneBottom) != "terminal" {
		t.Fatalf("默认排布错：左=%s 右=%s 底=%s", p.PanelIn(ZoneLeft), p.PanelIn(ZoneRight), p.PanelIn(ZoneBottom))
	}
	if p.ZoneOf("chat") != ZoneRight {
		t.Error("ZoneOf chat 应为右")
	}

	p.Move("chat", ZoneLeft) // chat→左：与左原 files 互换
	if p.PanelIn(ZoneLeft) != "chat" || p.PanelIn(ZoneRight) != "files" {
		t.Errorf("chat→左后：左=%s 右=%s（期望 chat/files）", p.PanelIn(ZoneLeft), p.PanelIn(ZoneRight))
	}
	if p.PanelIn(ZoneBottom) != "terminal" {
		t.Error("底部不应受影响")
	}
	// 仍是排列：三组各占一区、无重复
	seen := map[string]bool{p.LeftPanel: true, p.RightPanel: true, p.BotPanel: true}
	if len(seen) != 3 {
		t.Errorf("非排列（有重复）：%+v", seen)
	}

	p.Move("chat", ZoneLeft) // 已在左 → 无操作
	if p.PanelIn(ZoneLeft) != "chat" {
		t.Error("移到同区应无操作")
	}
}
