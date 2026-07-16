package debugger

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// ─── 调试会话生命周期 ─────────────────────────────────────────────

// Start 启动调试会话：查找空闲端口，启动 dlv dap，建立连接并初始化。
// ctx 用于控制超时。timeout 建议 30s（dlv 编译+启动耗时）。
func (s *DebugSession) Start(ctx context.Context, program string) error {
	s.mu.Lock()
	if s.state != StateIdle {
		s.mu.Unlock()
		return fmt.Errorf("调试会话已启动（状态: %s）", s.state)
	}
	s.state = StateInitialized
	s.program = program
	s.mu.Unlock()

	// 1. 分配端口
	port, err := findFreePort()
	if err != nil {
		s.setState(StateError)
		return fmt.Errorf("分配端口失败: %w", err)
	}

	// 2. 启动 dlv dap
	args := []string{"dap", "--listen", fmt.Sprintf("127.0.0.1:%d", port)}
	if program != "" {
		args = append(args, "--", program)
	}
	cmd := exec.CommandContext(ctx, s.dlvCmd, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.setState(StateError)
		return fmt.Errorf("创建 dlv stdout 管道失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.setState(StateError)
		return fmt.Errorf("创建 dlv stderr 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		s.setState(StateError)
		return fmt.Errorf("启动 dlv dap 失败（请确认已安装 dlv）: %w", err)
	}

	// 3. 等待 dlv 输出 "DAP server listening"
	dlvReady := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		var output strings.Builder
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
				if strings.Contains(output.String(), "DAP server listening") {
					dlvReady <- nil
					return
				}
			}
			if err != nil {
				dlvReady <- fmt.Errorf("dlv 输出错误: %w（输出: %s）", err, output.String())
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 4096)
		var output strings.Builder
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	select {
	case err := <-dlvReady:
		if err != nil {
			cmd.Process.Kill()
			s.setState(StateError)
			return err
		}
	case <-ctx.Done():
		cmd.Process.Kill()
		s.setState(StateError)
		return fmt.Errorf("等待 dlv dap 启动超时")
	}

	// 4. 建立 DAP 连接
	conn, err := newDAPConn("127.0.0.1", port)
	if err != nil {
		cmd.Process.Kill()
		s.setState(StateError)
		return fmt.Errorf("连接 dlv dap 失败: %w", err)
	}
	s.conn = conn
	s.port = port

	// 5. 发送 initialize 请求
	initArgs := InitializeRequest{
		AdapterID:             "gou-ide",
		ClientID:              "gou-ide",
		ClientName:            "gou-ide Debugger",
		LinesStartAt1:         true,
		ColumnsStartAt1:       true,
		PathFormat:            "path",
		SupportsVariableType:  true,
		SupportsVariablePaging: true,
	}
	resp, err := conn.send("initialize", initArgs)
	if err != nil {
		cmd.Process.Kill()
		conn.close()
		s.setState(StateError)
		return fmt.Errorf("DAP initialize 失败: %w", err)
	}
	if !resp.Success {
		cmd.Process.Kill()
		conn.close()
		s.setState(StateError)
		return fmt.Errorf("DAP initialize 返回错误: %s", resp.Message)
	}

	// 6. 发送 initialized 事件确认（DAP 协议要求）
	// dlv dap 收到 initialize 后会自动发送 initialized 事件，不需要客户端发

	// 7. 发送 launch 请求
	launchArgs := LaunchRequest{
		Program:     program,
		StopOnEntry: false,
	}
	resp, err = conn.send("launch", launchArgs)
	if err != nil {
		cmd.Process.Kill()
		conn.close()
		s.setState(StateError)
		return fmt.Errorf("DAP launch 失败: %w", err)
	}
	if !resp.Success {
		cmd.Process.Kill()
		conn.close()
		s.setState(StateError)
		return fmt.Errorf("DAP launch 返回错误: %s", resp.Message)
	}

	s.setState(StateRunning)
	return nil
}

// Stop 停止调试会话，关闭 dlv 和连接。
func (s *DebugSession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateIdle {
		return nil
	}

	// 发送 disconnect 请求
	if s.conn != nil {
		s.conn.send("disconnect", nil)
		s.conn.close()
		s.conn = nil
	}

	s.state = StateExited
	s.breakpoints = nil
	return nil
}

// ─── 断点管理 ────────────────────────────────────────────────────

// SetBreakpoints 在指定源文件的指定行设置断点。
// 返回实际设置的断点列表（含 verified 状态）。
func (s *DebugSession) SetBreakpoints(sourcePath string, lines []int) ([]Breakpoint, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	bps := make([]Breakpoint, len(lines))
	for i, line := range lines {
		bps[i] = Breakpoint{Line: line}
	}

	req := SetBreakpointsRequest{
		Source:      Source{Path: sourcePath},
		Lines:       lines,
		Breakpoints: bps,
	}

	resp, err := conn.send("setBreakpoints", req)
	if err != nil {
		return nil, fmt.Errorf("设置断点失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("设置断点返回错误: %s", resp.Message)
	}

	// 解析响应
	var body struct {
		Breakpoints []Breakpoint `json:"breakpoints"`
	}
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}

	s.mu.Lock()
	s.breakpoints = append(s.breakpoints, body.Breakpoints...)
	s.mu.Unlock()

	return body.Breakpoints, nil
}

// ClearBreakpoints 清除指定文件的所有断点。
func (s *DebugSession) ClearBreakpoints(sourcePath string) error {
	_, err := s.SetBreakpoints(sourcePath, []int{})
	return err
}

// ─── 执行控制 ────────────────────────────────────────────────────

// Continue 继续执行（从暂停状态恢复）。
func (s *DebugSession) Continue(threadID int) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("调试会话未启动")
	}

	req := ContinueRequest{ThreadID: threadID}
	resp, err := conn.send("continue", req)
	if err != nil {
		return fmt.Errorf("continue 失败: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("continue 返回错误: %s", resp.Message)
	}

	s.setState(StateRunning)
	return nil
}

// Next 单步跳过（Step Over）。
func (s *DebugSession) Next(threadID int) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("调试会话未启动")
	}

	req := NextRequest{ThreadID: threadID}
	resp, err := conn.send("next", req)
	if err != nil {
		return fmt.Errorf("next 失败: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("next 返回错误: %s", resp.Message)
	}

	s.setState(StateRunning)
	return nil
}

// StepIn 单步进入（Step Into）。
func (s *DebugSession) StepIn(threadID int) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("调试会话未启动")
	}

	req := NextRequest{ThreadID: threadID}
	resp, err := conn.send("stepIn", req)
	if err != nil {
		return fmt.Errorf("stepIn 失败: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stepIn 返回错误: %s", resp.Message)
	}

	s.setState(StateRunning)
	return nil
}

// StepOut 单步跳出（Step Out）。
func (s *DebugSession) StepOut(threadID int) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("调试会话未启动")
	}

	req := NextRequest{ThreadID: threadID}
	resp, err := conn.send("stepOut", req)
	if err != nil {
		return fmt.Errorf("stepOut 失败: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("stepOut 返回错误: %s", resp.Message)
	}

	s.setState(StateRunning)
	return nil
}

// Pause 暂停执行。
func (s *DebugSession) Pause(threadID int) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("调试会话未启动")
	}

	req := map[string]int{"threadId": threadID}
	resp, err := conn.send("pause", req)
	if err != nil {
		return fmt.Errorf("pause 失败: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("pause 返回错误: %s", resp.Message)
	}

	return nil
}

// GetScopes 获取指定栈帧的作用域列表（用于获取变量的 variablesReference）。
func (s *DebugSession) GetScopes(frameID int) ([]Scope, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	req := ScopesRequest{FrameID: frameID}
	resp, err := conn.send("scopes", req)
	if err != nil {
		return nil, fmt.Errorf("scopes 失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("scopes 返回错误: %s", resp.Message)
	}

	var body struct {
		Scopes []Scope `json:"scopes"`
	}
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}
	return body.Scopes, nil
}

// ─── 信息查询 ────────────────────────────────────────────────────

// StackTrace 获取指定线程的调用栈。
func (s *DebugSession) StackTrace(threadID int, levels int) ([]StackFrame, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	if levels <= 0 {
		levels = 20
	}
	req := StackTraceRequest{ThreadID: threadID, Levels: levels}
	resp, err := conn.send("stackTrace", req)
	if err != nil {
		return nil, fmt.Errorf("stackTrace 失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("stackTrace 返回错误: %s", resp.Message)
	}

	var body struct {
		StackFrames []StackFrame `json:"stackFrames"`
		TotalFrames int          `json:"totalFrames,omitempty"`
	}
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}
	return body.StackFrames, nil
}

// Variables 获取指定变量的子变量。
// variablesReference=0 表示获取所有顶层变量。
func (s *DebugSession) Variables(ref int) ([]Variable, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	req := VariablesRequest{VariablesReference: ref}
	resp, err := conn.send("variables", req)
	if err != nil {
		return nil, fmt.Errorf("variables 失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("variables 返回错误: %s", resp.Message)
	}

	var body struct {
		Variables []Variable `json:"variables"`
	}
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}
	return body.Variables, nil
}

// Evaluate 在指定栈帧上下文中求值表达式。
func (s *DebugSession) Evaluate(expression string, frameID int) (*EvaluateResponse, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	req := EvaluateRequest{
		Expression: expression,
		FrameID:    frameID,
		Context:    "repl",
	}
	resp, err := conn.send("evaluate", req)
	if err != nil {
		return nil, fmt.Errorf("evaluate 失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("evaluate 返回错误: %s", resp.Message)
	}

	var body EvaluateResponse
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}
	return &body, nil
}

// Threads 获取当前线程列表。
func (s *DebugSession) Threads() ([]Thread, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("调试会话未启动")
	}

	resp, err := conn.send("threads", nil)
	if err != nil {
		return nil, fmt.Errorf("threads 失败: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("threads 返回错误: %s", resp.Message)
	}

	var body struct {
		Threads []Thread `json:"threads"`
	}
	if resp.Body != nil {
		json.Unmarshal(resp.Body, &body)
	}
	return body.Threads, nil
}

// WaitStopped 阻塞等待调试事件（如断点命中、单步完成），带超时。
// 返回停止原因。ctx 超时返回 ctx.Err()。
func (s *DebugSession) WaitStopped(ctx context.Context) (StopReason, error) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return "", fmt.Errorf("调试会话未启动")
	}

	select {
	case evt := <-conn.events:
		return s.handleEvent(evt), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// handleEvent 处理 DAP 事件。
func (s *DebugSession) handleEvent(msg Message) StopReason {
	if msg.Type != MsgEvent {
		return ""
	}

	switch msg.Event {
	case "stopped":
		var body StoppedEventBody
		if msg.Body != nil {
			json.Unmarshal(msg.Body, &body)
		}
		s.lastBody = msg.Body
		reason := StopReason(body.Reason)
		s.setState(StatePaused)
		s.stopReason = reason
		return reason

	case "continued":
		s.setState(StateRunning)

	case "exited":
		var body ExitedEventBody
		if msg.Body != nil {
			json.Unmarshal(msg.Body, &body)
		}
		s.setState(StateExited)
		return StopException

	case "terminated":
		s.setState(StateExited)

	case "output":
		// 程序输出，暂不处理
	}

	return ""
}

// ─── 内部辅助 ────────────────────────────────────────────────────

func (s *DebugSession) setState(state SessionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// findFreePort 查找可用 TCP 端口。
func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}
