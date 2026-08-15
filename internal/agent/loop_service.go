// loop_service.go — 一切皆插件：loop 服务面（ctx.get('loop')）。
//
// ★ 插件在循环运行期间可查询 Loop 状态、请求暂停/继续/停止——
//   对齐参考项目（deepseek-harness）agent-loop 插件包的服务能力：宿主循环对插件可编程。
//   配合 loop:* 事件桥（Loop.emit 广播），插件可完整感知/调控 agentloop。
//
// ★ 生命周期：Loop.Run 开始时注册到全局插件宿主根上下文（ctx.provide('loop')），
//   Run 结束自动撤销——仅循环运行期间可用；未运行时 ctx.get('loop') 返回 nil（插件判空）。
//   并行 Run（delegate 子 Loop）时服务指向最近启动的 Loop（provide 覆盖语义）。
//
// ★ 控制语义：Pause/RequestStop 都是「请求」——在下一轮迭代开始处生效；
//   阻塞中的 LLM 调用不受影响（暂停等待期间可被 ctx 取消唤醒）。
package agent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// LoopState 循环状态快照（JSON 友好，供插件 ctx.get('loop').getState() 读取）。
type LoopState struct {
	Running       bool      `json:"running"`
	Turn          int       `json:"turn"`
	Step          int       `json:"step"`
	Phase         string    `json:"phase"` // 最近事件阶段（thinking/tool_call/…）
	LastTool      string    `json:"lastTool"`
	LastEventTime time.Time `json:"lastEventTime"`
	Paused        bool      `json:"paused"`
	StopRequested bool      `json:"stopRequested"`
	Iterations    int       `json:"iterations"` // 已完成迭代数
}

// LoopService 循环服务（插件 ctx.get('loop') 获取；未运行期间为 nil）。
type LoopService struct {
	Loop *Loop

	pause   atomic.Bool  // 暂停请求
	stop    atomic.Bool  // 停止请求
	pauseMu sync.Mutex   // 保护 pauseCh 重建
	pauseCh chan struct{} // 暂停期间阻塞等待通道（Resume 关闭并重建）

	stateMu sync.Mutex
	lastType  EventType
	lastTool  string
	lastEvtAt time.Time
}

func newLoopService(l *Loop) *LoopService {
	return &LoopService{Loop: l, pauseCh: make(chan struct{})}
}

// GetState 当前循环状态快照。
func (s *LoopService) GetState() LoopState {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	l := s.Loop
	st := LoopState{
		Running:       true,
		Turn:          l.TurnNo,
		Step:          l.StepNo,
		Phase:         string(s.lastType),
		LastTool:      s.lastTool,
		LastEventTime: s.lastEvtAt,
		Paused:        s.pause.Load(),
		StopRequested: s.stop.Load(),
		Iterations:    l.StepNo,
	}
	if st.LastEventTime.IsZero() {
		st.LastEventTime = time.Now()
	}
	return st
}

// Pause 请求暂停（幂等）。下一轮迭代开始处生效；暂停期间发 loop:paused 事件。
func (s *LoopService) Pause() {
	if s.pause.CompareAndSwap(false, true) {
		s.Loop.emit(Event{Type: EventNotice, Content: "插件请求暂停 agentloop（下轮迭代生效）"})
	}
}

// Resume 继续运行（幂等）。
func (s *LoopService) Resume() {
	if s.pause.CompareAndSwap(true, false) {
		s.pauseMu.Lock()
		close(s.pauseCh)
		s.pauseCh = make(chan struct{})
		s.pauseMu.Unlock()
		s.Loop.emit(Event{Type: EventNotice, Content: "插件请求继续 agentloop"})
	}
}

// RequestStop 请求停止（下轮迭代开始处退出 Run；返回的 msgs 以插件停止原因结束）。
func (s *LoopService) RequestStop() {
	if s.stop.CompareAndSwap(false, true) {
		s.Loop.emit(Event{Type: EventNotice, Content: "插件请求停止 agentloop（下轮迭代退出）"})
	}
}

// noteEvent 由 Loop.emit 调用，同步最近事件快照（锁外无并发：emit 在 Loop 主 goroutine）。
func (s *LoopService) noteEvent(e Event) {
	s.stateMu.Lock()
	s.lastType = e.Type
	s.lastTool = e.Tool
	s.lastEvtAt = time.Now()
	s.stateMu.Unlock()
}

// waitIfPaused 由 Loop.Run 每轮迭代开始调用：
//   - 暂停中 → 阻塞等待 Resume / 外部 ctx 取消（阻塞时若插件 RequestStop 也退出）
//   - 返回 false = 应停止（插件停止请求 或 ctx 取消）
func (s *LoopService) waitIfPaused(ctx context.Context) bool {
	if !s.pause.Load() {
		return !s.stop.Load()
	}
	s.pauseMu.Lock()
	ch := s.pauseCh
	s.pauseMu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-ch: // 已 Resume
			return !s.stop.Load()
		default:
			if s.stop.Load() || !s.pause.Load() {
				return !s.stop.Load()
			}
			select {
			case <-ctx.Done():
				return false
			case <-ch:
				return !s.stop.Load()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

// shouldStop 停止请求检查（Run 每轮调用；返回停止原因）。
func (s *LoopService) shouldStop() string {
	if s.stop.Load() {
		return "插件请求停止（ctx.get('loop').requestStop()）"
	}
	return ""
}

// loopStopError 插件请求停止时的 Run 返回错误（标记性；调用方通常忽略 msgs 差异）。
func loopStopError(reason string) error {
	return fmt.Errorf("loop stopped: %s", reason)
}
