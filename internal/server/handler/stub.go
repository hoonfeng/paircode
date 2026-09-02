// Handler — 对话/Agent（从 web_server.go 迁移）
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/pkg/memory"
)

// HandleChatSend 启动 agent 会话
func HandleChatSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message       string            `json:"message"`
		SessionID     string            `json:"sessionId"`
		Autonomous    bool              `json:"autonomous"`
		ConvID        string            `json:"convId"`
		WorkspaceRoot string            `json:"workspaceRoot"`
		Images        []agent.ImagePart `json:"images,omitempty"` // ★ 2026-08-21 多模态：前端结构化发送的图片
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Message == "" {
		jsonErr(w, "消息不能为空")
		return
	}
	const maxMsgLen = 50000
	if len(req.Message) > maxMsgLen {
		req.Message = req.Message[:maxMsgLen] + "\n\n…（消息过长，已截断至 " + fmt.Sprint(maxMsgLen) + " 字符）"
	}
	if req.ConvID == "" {
		req.ConvID = fmt.Sprintf("conv_%d", time.Now().UnixNano())
	}
	if req.WorkspaceRoot == "" {
		req.WorkspaceRoot = core.Root()
	}
	if !agent.ConfiguredProvider() {
		jsonErr(w, "未配置 API key。请在设置面板中配置 API Key 和模型。")
		return
	}
	if len(req.Images) > 0 {
		// ★ 2026-08-21 多模态：带图片的用户消息落盘
		if err := AgentMgr.AppendPersistedUserMessageWithImages(req.ConvID, req.Message, req.Images); err != nil {
			log.Printf("[chat] AppendPersistedUserMessageWithImages 失败 conv=%s err=%v", req.ConvID, err)
			jsonErr(w, "写入用户消息失败: "+err.Error())
			return
		}
	} else {
		if err := AgentMgr.AppendPersistedUserMessage(req.ConvID, req.Message); err != nil {
			log.Printf("[chat] AppendPersistedUserMessage 失败 conv=%s err=%v", req.ConvID, err)
			jsonErr(w, "写入用户消息失败: "+err.Error())
			return
		}
	}
	if store := AgentMgr.Store(); store != nil {
		if count, err := store.Count(req.ConvID); err == nil && count > 0 {
			if tr := agent.GetTracker(); tr != nil {
				tr.SetCurrentMsg(req.ConvID, count-1)
			}
		}
	}
	opts := BuildLoopOpts(req.ConvID, req.Message, req.Autonomous)
	opts.WorkspaceRoot = req.WorkspaceRoot
	opts.ReviewMode = core.Settings.ReviewMode
	// ★ 配置消费插件化：ReviewProvider 参数统一经装配点解析。
	cur := agent.ResolveProviderParams()
	if core.Settings.ReviewMode == "auto" && cur.ReviewModel != "" {
		pm := strings.TrimSpace(cur.PlanModel)
		if pm != "" && cur.BaseURL != "" && cur.APIKey != "" {
			// ★ t1 S1：实现级插件槽位（插件注册的 Provider 实现对新协议生效）
			rp := cur
			rp.Model = pm
			rp.Temperature = -1
			rp.ThinkingMode = "non-thinking"
			rp.MaxTokens = 0
			rp.Multimodal = false
			opts.ReviewProvider = agent.CreateProvider(rp)
		}
	}

	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()
	if err := AgentMgr.Start(setupCtx, req.ConvID, req.Message, opts); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "convId": req.ConvID})
}

func HandleChatStop(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	if convID == "" {
		convID = r.URL.Query().Get("sessionId")
	}
	if convID == "" {
		jsonErr(w, "缺少 convId 参数")
		return
	}
	AgentMgr.Stop(convID)
	jsonResp(w, map[string]any{"ok": true})
}

func HandleChatAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConvID string `json:"convId"`
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" || req.Answer == "" {
		jsonErr(w, "convId 和 answer 必填")
		return
	}
	if err := AgentMgr.SendAnswer(req.ConvID, req.Answer); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleChatApprove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConvID   string `json:"convId"`
		Approved bool   `json:"approved"`
		Reply    string `json:"reply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := AgentMgr.Approve(req.ConvID, req.Approved, req.Reply); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleChatFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConvID  string `json:"convId"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" || req.Content == "" {
		jsonErr(w, "convId 和 content 必填")
		return
	}
	AgentMgr.SendFeedback(req.ConvID, req.Content)
	jsonResp(w, map[string]any{"ok": true})
}

func HandleChatRollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConvID string `json:"convId"`
		MsgIdx int    `json:"msgIdx"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.ConvID == "" {
		jsonErr(w, "convId 必填")
		return
	}
	root := core.Root()
	if root == "" {
		jsonErr(w, "工作区未设置")
		return
	}
	var store agent.ConversationStore
	if AgentMgr != nil {
		store = AgentMgr.Store()
	}
	if err := agent.RollbackToMsg(root, req.ConvID, req.MsgIdx, store); err != nil {
		jsonErr(w, err.Error())
		return
	}
	AgentMgr.Stop(req.ConvID)
	jsonResp(w, map[string]any{"ok": true, "msgIdx": req.MsgIdx})
}

// ─── 对话列表 ──────────────────────────────────────────────

func HandleConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		wsFilter := r.URL.Query().Get("workspace")
		store := AgentMgr.Store()
		if store == nil {
			jsonResp(w, []agent.ConversationMeta{})
			return
		}
		metas, err := store.ListConversations(wsFilter)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		if metas == nil {
			metas = []agent.ConversationMeta{}
		}
		jsonResp(w, metas)
		return
	}
	if r.Method == "POST" {
		var req struct {
			ID            string `json:"id"`
			Title         string `json:"title"`
			WorkspaceRoot string `json:"workspaceRoot"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		store := AgentMgr.Store()
		if store == nil {
			jsonErr(w, "消息存储未初始化")
			return
		}
		id := req.ID
		if id == "" {
			id = fmt.Sprintf("conv_%d", time.Now().UnixNano())
		}
		wsRoot := req.WorkspaceRoot
		if wsRoot == "" {
			wsRoot = core.Root()
		}
		title := req.Title
		if title == "" {
			title = "新对话 " + time.Now().Format("15:04")
		}
		if err := store.CreateConversation(id, title, wsRoot); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "id": id, "title": title})
		return
	}
	jsonErr(w, "不支持的方法")
}

func HandleConversationCreate(w http.ResponseWriter, r *http.Request) {
	HandleConversations(w, r)
}

func HandleConversationByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/conversations/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		jsonErr(w, "缺少对话 ID")
		return
	}
	id := parts[0]
	sub := ""
	if len(parts) >= 2 {
		sub = parts[1]
	}
	if AgentMgr == nil {
		jsonErr(w, "会话管理器未初始化")
		return
	}
	store := AgentMgr.Store()
	if store == nil {
		jsonErr(w, "消息存储未初始化")
		return
	}
	switch r.Method {
	case "GET":
		switch sub {
		case "messages":
			limit := 50
			if l := r.URL.Query().Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}
			beforeStr := r.URL.Query().Get("before")
			if beforeStr != "" {
				var before int
				fmt.Sscanf(beforeStr, "%d", &before)
				msgs, err := store.LoadBefore(id, before, limit)
				if err != nil {
					jsonErr(w, err.Error())
					return
				}
				if msgs == nil {
					msgs = []agent.StoredMessage{}
				}
				msgs = agent.MergeConsecutiveAssistants(msgs)
				total, _ := store.Count(id)
				jsonResp(w, map[string]any{"messages": msgs, "total": total})
				return
			}
			msgs, total, err := store.LoadLatest(id, limit)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if msgs == nil {
				msgs = []agent.StoredMessage{}
			}
			msgs = agent.MergeConsecutiveAssistants(msgs)
			jsonResp(w, map[string]any{"messages": msgs, "total": total})
		case "token-stats":
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil || meta.CtxStats == nil {
				jsonResp(w, map[string]any{
					"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
					"cacheHitTokens": 0, "cacheMissTokens": 0,
				})
				return
			}
			cs := meta.CtxStats
			m := map[string]any{
				"promptTokens":     cs.PromptTokens,
				"completionTokens": cs.CompletionTokens,
				"totalTokens":      cs.TotalTokens,
				"cacheHitTokens":   cs.PromptCacheHitTokens,
				"cacheMissTokens":  cs.PromptCacheMissTokens,
			}
			if cs.PromptBreakdown.SystemTokens > 0 || cs.PromptBreakdown.SkillsTokens > 0 ||
				cs.PromptBreakdown.MCPTokens > 0 || cs.PromptBreakdown.ToolTokens > 0 {
				m["systemTokens"] = cs.SystemTokens
				m["skillsTokens"] = cs.SkillsTokens
				m["mcpTokens"] = cs.MCPTokens
				m["toolTokens"] = cs.ToolTokens
				m["historyTokens"] = cs.HistoryTokens
				m["otherTokens"] = cs.OtherTokens
			}
			jsonResp(w, m)
		default:
			meta, err := store.GetConversation(id)
			if err != nil {
				jsonErr(w, err.Error())
				return
			}
			if meta == nil {
				jsonErr(w, "对话不存在")
				return
			}
			// 注入 agent 运行状态
			resp := map[string]any{
				"id":        meta.ID,
				"title":     meta.Title,
				"createdAt": meta.CreatedAt,
				"updatedAt": meta.UpdatedAt,
				"msgCount":  meta.MsgCount,
				"summary":   meta.Summary,
				"summaryAt": meta.SummaryAt,
			}
			if meta.WorkspaceRoot != "" {
				resp["workspaceRoot"] = meta.WorkspaceRoot
			}
			if meta.CtxStats != nil {
				resp["ctxStats"] = meta.CtxStats
			}
			if status := AgentMgr.GetStatus(id); status != nil {
				resp["running"] = status.Running
				resp["stopped"] = status.Stopped
				resp["startedAt"] = status.StartedAt
			}
			jsonResp(w, resp)
		}
	case "PUT":
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if req.Title != "" {
			if err := store.UpdateTitle(id, req.Title); err != nil {
				jsonErr(w, err.Error())
				return
			}
		}
		jsonResp(w, map[string]any{"ok": true})
	case "POST":
		var msg struct {
			Role     string          `json:"role"`
			Content  string          `json:"content"`
			Segments []agent.Segment `json:"segments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if msg.Role == "" {
			msg.Role = "user"
		}
		aMsg := agent.Message{Role: agent.Role(msg.Role), Content: msg.Content}
		if err := store.AppendMessage(id, aMsg, msg.Segments); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	case "DELETE":
		if err := store.DeleteConversation(id); err != nil {
			jsonErr(w, err.Error())
			return
		}
		memory.Delete(id)
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "不支持的方法")
	}
}

// ─── Tasks ────────────────────────────────────────────────

func HandleTasks(w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("convId")
	root := core.Root()
	tm := agent.UseTaskManager(root)
	tasks := tm.ListByConvID(convID)
	summary := tm.GetSummary()
	if tasks == nil {
		tasks = []*agent.Task{}
	}
	jsonResp(w, map[string]any{
		"tasks":   tasks,
		"summary": summary,
	})
}

// ★ 2026-08-31：HandleTaskPlan（/api/taskplan 规划文档）已随 plan 体系移除。


// ─── 模型 / 指令 / 思想 ──────────────────────────────────

func HandleModels(w http.ResponseWriter, r *http.Request) {
	// POST/PUT：全量保存服务商与模型列表 → 落盘到安装目录 config/models.json
	// body: { "providers": { "<name>": { "baseURL": "...", "models": ["..."] } } }
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		var req struct {
			Providers map[string]core.ProviderEntry `json:"providers"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "无效 JSON: "+err.Error())
			return
		}
		if req.Providers == nil {
			jsonErr(w, "providers 不能为空")
			return
		}
		core.SetModelList(req.Providers)
		if err := core.SaveModelList(); err != nil {
			jsonErr(w, "保存失败: "+err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "saved": len(req.Providers)})
		return
	}
	providers := core.GetProviders()
	modelMap := make(map[string][]string)
	for _, p := range providers {
		modelMap[p] = core.GetModels(p)
	}
	jsonResp(w, map[string]any{
		"providers":        providers,
		"models":           modelMap,
		"providerBaseURLs": core.GetProviderBaseURLs(),
		"providerKeys":     core.GetProviderAPIKeys(),          // ★ 服务商独立 API Key（切服务商自动带出）
		"providerContexts": core.GetProviderContextMaxTokens(), // ★ 服务商级默认上下文窗口（模型级可覆盖）
		"providerProtocols": core.GetProviderProtocols(),       // ★ 2026-09-02 服务商 LLM 协议（前端联动下拉）
	})
}

// HandleAiPresets AI 配置预设管理（GET 查询 / POST 保存-删除-应用 / PUT 全量保存）。
// ★ 2026-08-20 AI 配置预设：把「一份完整 AI 配置」命名保存为预设，对话面板快速切换。
//
//	body (POST): { "action": "save", "name": "预设名", "preset": {...} }   —— 保存/覆盖
//	             { "action": "apply", "name": "预设名" }                    —— 应用（只写 settings.preset，装配时按 preset 展开）
//	             { "action": "delete", "name": "预设名" }                   —— 删除
//	body (PUT):  { "presets": { "<预设名>": {...} } }                       —— 全量替换
func HandleAiPresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Action  string         `json:"action"`
			Name    string         `json:"name"`
			Preset  core.AiPreset  `json:"preset"`
			Presets core.AiPresets `json:"presets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "无效 JSON: "+err.Error())
			return
		}
		switch req.Action {
		case "save":
			if req.Name == "" {
				jsonErr(w, "预设名不能为空")
				return
			}
			// 未传 preset 时抓取当前 settings 快照
			p := req.Preset
			if p.Provider == "" && p.ExecuteModel == "" && p.BaseURL == "" && p.APIKey == "" {
				p = core.AiPresetFromSettings()
			}
			core.GetAiPresets()
			core.PresetList[req.Name] = p
			if err := core.SaveAiPresets(); err != nil {
				jsonErr(w, "保存失败: "+err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true, "name": req.Name, "presets": core.GetAiPresets()})
		case "apply":
			if req.Name == "" {
				jsonErr(w, "预设名不能为空")
				return
			}
			p := core.GetPreset(req.Name)
			if p.Provider == "" && p.ExecuteModel == "" {
				jsonErr(w, "预设不存在: "+req.Name)
				return
			}
			// 应用：预设字段写回 settings（provider/baseURL/apiKey/模型/参数）
			core.ApplyPreset(req.Name, p)
			jsonResp(w, map[string]any{"ok": true, "name": req.Name, "settings": core.Settings})
		case "delete":
			if req.Name == "" {
				jsonErr(w, "预设名不能为空")
				return
			}
			core.GetAiPresets()
			if _, ok := core.PresetList[req.Name]; ok {
				delete(core.PresetList, req.Name)
				if err := core.SaveAiPresets(); err != nil {
					jsonErr(w, "保存失败: "+err.Error())
					return
				}
			}
			jsonResp(w, map[string]any{"ok": true, "presets": core.GetAiPresets()})
		case "rename":
			if req.Name == "" || req.Presets == nil {
				jsonErr(w, "rename 需要 name 与 presets（改名后全量）")
				return
			}
			core.SetAiPresets(req.Presets)
			if err := core.SaveAiPresets(); err != nil {
				jsonErr(w, "保存失败: "+err.Error())
				return
			}
			jsonResp(w, map[string]any{"ok": true, "presets": core.GetAiPresets()})
		default:
			jsonErr(w, "未知 action: "+req.Action+"（save/apply/delete/rename）")
		}
		return
	case http.MethodPut:
		var req struct {
			Presets core.AiPresets `json:"presets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "无效 JSON: "+err.Error())
			return
		}
		if req.Presets == nil {
			jsonErr(w, "presets 不能为空")
			return
		}
		core.SetAiPresets(req.Presets)
		if err := core.SaveAiPresets(); err != nil {
			jsonErr(w, "保存失败: "+err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "saved": len(req.Presets)})
		return
	}
	jsonResp(w, map[string]any{
		"presets": core.GetAiPresets(),
	})
}

func HandleInstructions(w http.ResponseWriter, r *http.Request) { jsonResp(w, "") }
func HandleInstructionsPut(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}

// ─── MCP / Skills ────────────────────────────────────────

func HandleMCPList(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func HandleMCPSave(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}
func HandleSkillsList(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func HandleSkillsRead(w http.ResponseWriter, r *http.Request) { jsonResp(w, "") }
func HandleSkillsSave(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}
func HandleSkillsDelete(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}

// ─── Token / Debug ───────────────────────────────────────

// HandleTokensStats 返回工作区级 token 统计（agent 自闭环持久化数据，
// 与 web 端 web_server.go handleTokensStats 同源：.pair/token-stats.json）。
func HandleTokensStats(w http.ResponseWriter, r *http.Request) {
	wsRoot := r.URL.Query().Get("workspaceRoot")
	if wsRoot == "" {
		wsRoot = core.Root()
	}
	stats := agent.ReadTokenStatsForRoot(wsRoot)
	if stats == nil {
		jsonResp(w, map[string]any{
			"promptTokens": 0, "completionTokens": 0, "totalTokens": 0,
			"cacheHitTokens": 0, "cacheMissTokens": 0,
			"systemTokens": 0, "skillsTokens": 0, "mcpTokens": 0,
			"toolTokens": 0, "historyTokens": 0, "otherTokens": 0,
		})
		return
	}
	jsonResp(w, map[string]any{
		"promptTokens":     stats.PromptTokens,
		"completionTokens": stats.CompletionTokens,
		"totalTokens":      stats.TotalTokens,
		"cacheHitTokens":   stats.CacheHitTokens,
		"cacheMissTokens":  stats.CacheMissTokens,
		"systemTokens":     stats.SystemTokens,
		"skillsTokens":     stats.SkillsTokens,
		"mcpTokens":        stats.MCPTokens,
		"toolTokens":       stats.ToolTokens,
		"historyTokens":    stats.HistoryTokens,
		"otherTokens":      stats.OtherTokens,
	})
}
func HandleDebugLogs(w http.ResponseWriter, r *http.Request)    { jsonResp(w, []string{}) }
func HandleDebugLogByID(w http.ResponseWriter, r *http.Request) { jsonResp(w, "") }

// ─── 市场 / 记忆 ─────────────────────────────────────────

func HandleMarketplaceSearch(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func HandleMarketplaceInstall(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}
func HandleMarketplaceRefresh(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}
func HandleMemorySearch(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }
func HandleMemoryList(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func HandleMemoryRebuild(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]string{"status": "ok"})
}
