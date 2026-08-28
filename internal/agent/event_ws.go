// ═══════════════════════════════════════════════════════════════
// event_ws.go — 全局事件流 WebSocket 端点（框架层）
//
// 从 cmd/companion/websocket_handler.go 下沉（2026-08-16）：WebSocket
// 能力由框架层（本包）处理——帧实现 wsconn.go、路由注册表 ext_ws.go、
// 宿主事件流端点本文件。main 只保留一行路由注册。
//
// ServeGlobalEventStreamWS：订阅 SessionManager 全局事件流，将所有
// 会话事件以 JSON 推送到 WebSocket。连接建立时先发送 {type:"status",
// runningConvs:[...]} 同步初始状态。心跳 30s ping + 70s 读超时。
// ═══════════════════════════════════════════════════════════════

package agent

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ServeGlobalEventStreamWS 是 /ws 事件流端点处理器：升级连接后订阅
// mgr 的全局事件流，事件以 JSON 帧推送。每条消息格式：
// {convId, type, content, tool, args, callId, usage, doneReason}
func ServeGlobalEventStreamWS(w http.ResponseWriter, r *http.Request, mgr *SessionManager) {
	wsc, err := UpgradeWS(w, r)
	if err != nil {
		return // UpgradeWS 已写过 HTTP 错误响应
	}
	defer wsc.Close()

	// 订阅全局事件流
	ch := mgr.SubscribeAll()
	defer mgr.UnsubscribeAll(ch)

	// 发送初始状态：当前运行中的所有会话 + 按工作区分组的运行计数
	wsc.WriteTextFrame(buildStatusPayload(mgr))

	// ★ 2026-08-21 WS 断线补偿：对新连接推送 running 会话的流式生成快照。
	//   客户端重连时，断线期间丢失的 content/thinking/tool 事件由快照弥补，
	//   前端据此重建占位消息，避免「断线后 assistant 内容截断/缺失」。
	//   无快照的会话（尚未产出内容）跳过——后续事件流会正常到达。
	if payload := buildSnapshotPayload(mgr); payload != nil {
		wsc.WriteTextFrame(payload)
	}

	// 心跳定时器（每 30 秒发一次 ping）
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// 后台读取客户端消息（处理 close 帧、忽略其他）。
	// ★ 2026-08-22 读超时由 ReadFrame 内部管理（每帧刷新）：仅文本 ping 时
	//   客户端不回帧 → 读循环 70s 超时必触发；因此心跳同时发协议层 Ping
	//   （浏览器自动回 Pong，每帧刷新 deadline → 连接永不因超时断开；
	//   客户端真正失联时 Pong 停发 → 70s 超时正常判死）。
	wsc.SetReadTimeout(70 * time.Second)
	clientClosed := make(chan struct{}, 1)
	go func() {
		defer func() {
			select {
			case clientClosed <- struct{}{}:
			default:
			}
		}()
		for {
			if _, _, e := wsc.ReadFrame(); e != nil {
				return
			}
			// 忽略客户端发来的文本消息（当前不支持命令通道）
		}
	}()

	// 主循环：从全局 channel 读取事件，推送到 WebSocket
	// 当事件类型为 done（会话结束）时，额外推送一次 status 消息，
	// 让前端的"工作区/对话列表"运行状态计数保持同步。
	for {
		select {
		case <-clientClosed:
			return
		case <-heartbeat.C:
			// ★ 2026-08-17 修复「发送消息后 agent 无响应」根因：
			//   心跳必须用文本帧 {type:"ping"}，不能用协议层 Ping 帧！
			//   浏览器 WebSocket 的 onmessage 只触发于数据帧；协议层 Ping 帧由
			//   浏览器自动回复 Pong，不触发 onmessage → 前端「45s 无消息」定时器
			//   永不重置 → agent 思考/LLM 重试期间无业务事件时，前端误判连接
			//   断开 → 反复重连 → 事件全部丢失（用户看到无任何响应）。
			//   文本帧 ping 触发 onmessage（重置超时器），实现真正的连接健康检测。
			// ★ 2026-08-22 修复「连接 70s 必断」：
			//   仅文本帧 ping 时，客户端不会回任何帧 → 后台读循环 70s 读超时
			//   必然触发（判定"客户端未回复"）→ 连接被服务端关闭。
			//   因此必须**同时**发协议层 Ping 帧：浏览器自动回 Pong（RFC6455），
			//   读循环收到 Pong 重置读超时 → 连接永不因超时断开；客户端真正
			//   失联（网络断开/进程退出）时 Pong 停发 → 70s 超时正常判死。
			if err := wsc.WritePingFrame(); err != nil {
				return
			}
			if err := wsc.WriteTextFrame([]byte(`{"type":"ping"}`)); err != nil {
				return
			}
		case ge, ok := <-ch:
			if !ok {
				// channel 被关闭（不应发生，但兜底）
				return
			}
			payload := buildWSPayload(ge)
			if err := wsc.WriteTextFrame(payload); err != nil {
				return
			}
			// done/error 事件表示会话运行集将发生变化，追加 status 更新。
			// 注意：EventDone 发射时 session_manager 的 Running 标志可能尚未置 false
			// （loop.Run 返回 → defer 设置 Running=false → close(Events)），
			// 因此短暂等待 50ms 让 Running 状态落地后再查询，避免刚结束的会话仍出现在 running 列表。
			if ge.Event.Type == EventDone || ge.Event.Type == EventError {
				time.Sleep(50 * time.Millisecond)
				if err := wsc.WriteTextFrame(buildStatusPayload(mgr)); err != nil {
					return
				}
			}
		}
	}
}

// buildStatusPayload 构造一条 status 消息，包含：
//   - runningConvs: 当前所有运行中的 convID 列表
//   - runningByWorkspace: 按工作区根路径分组的运行计数 {wsRoot: count}
//
// 前端据此更新工作区列表的脉冲点+计数，以及对话列表的"运行中"标签。
// 在 WebSocket 连接建立时和每次 done 事件后推送。
func buildStatusPayload(mgr *SessionManager) []byte {
	running := mgr.ListRunning()
	counts := make(map[string]int, 8)
	for _, id := range running {
		ws := mgr.GetWorkspaceRoot(id)
		if ws == "" {
			continue
		}
		counts[ws]++
	}
	msg := map[string]any{
		"type":               "status",
		"runningConvs":       running,
		"runningByWorkspace": counts,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] status JSON encode error: %v", err)
		return []byte(`{"type":"status","runningConvs":[],"runningByWorkspace":{}}`)
	}
	return data
}

// buildSnapshotPayload 构造 WS 断线补偿快照消息（支持多条：running 会话各有快照），
// 无任何快照时返回 nil（跳过发送）。
func buildSnapshotPayload(mgr *SessionManager) []byte {
	running := mgr.ListRunning()
	if len(running) == 0 {
		return nil
	}
	var out []map[string]any
	for _, id := range running {
		ev := mgr.LiveSnapshotEvent(id)
		if ev == nil {
			continue
		}
		out = append(out, map[string]any{
			"convId":       id,
			"type":         "snapshot",
			"content":      ev.Content,
			"reasoning":    ev.Reasoning,
			"toolSegments": ev.ToolSegments,
			"events":       ev.LiveEvents,
		})
	}
	if len(out) == 0 {
		return nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		log.Printf("[WS] snapshot JSON encode error: %v", err)
		return nil
	}
	return data
}

// buildWSPayload 将 GlobalEvent 编码为 WebSocket JSON 消息。
func buildWSPayload(ge GlobalEvent) []byte {
	e := ge.Event
	msg := map[string]any{
		"convId":     ge.ConvID,
		"type":       string(e.Type),
		"content":    e.Content,
		"tool":       e.Tool,
		"args":       e.Args,
		"callId":     e.CallID,
		"doneReason": e.DoneReason,
	}
	if e.Reasoning != "" {
		msg["reasoning"] = e.Reasoning
	}
	if len(e.ToolSegments) > 0 {
		msg["toolSegments"] = e.ToolSegments
	}
	if e.Usage != nil {
		msg["usage"] = e.Usage
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WS] JSON encode error: %v", err)
		return []byte(`{"type":"error","content":"JSON encode failed"}`)
	}
	return data
}
