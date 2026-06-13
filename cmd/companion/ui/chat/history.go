// Package chatpanel 对话面板 - 对话历史持久化。
//
//go:build windows

package chatpanel

import (
	"github.com/hoonfeng/paircode/cmd/companion/core"
)

// historyLoaded 包级标志，跟踪是否已从磁盘加载对话历史（防止重复加载）。
var historyLoaded bool

// SaveHistory 把当前所有对话历史持久化到工作区根。
func (s *ChatState) SaveHistory() {
	s.Store.Save(core.Root())
}

// saveHistory 包内别名（chat.go 中大量调用，保持兼容）。
func (s *ChatState) saveHistory() { s.SaveHistory() }

// LoadHistory 从工作区根加载对话历史。返回是否加载成功。
// 重复调用安全：仅首次实际加载，之后直接返回 true。
// 加载成功后自动触发 SetState 刷新 UI。
func (s *ChatState) LoadHistory() bool {
	if historyLoaded {
		return true
	}
	ok := s.Store.Load(core.Root())
	if ok {
		active := s.Store.Active()
		if active != nil {
		}
		historyLoaded = true
			s.SetState()
		}
	return ok
}
