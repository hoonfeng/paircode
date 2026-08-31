// 工具配置共享 handler：/api/tools（工具列表）、/api/tools/review（审核配置）。
// web 端（cmd/companion）与桌面端（internal/desktopbridge）共用。
// 工具注册表由各端构建 LoopOpts 时经 SetToolsRegistry 注入，
// 避免两端各自维护 lastReg（此前 web 端独有、桌面端 404 → 工具配置弹窗
// 显示「加载失败，请重试」）。
// ★ 旧版 PUT /api/tools/save（工具开关写 .pair/tools.json）已随「工具集（插件化）
// 手动管理」机制删除：工具启用/禁用由 toolset_edit（DisabledTools）管理。
package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
)

// ToolsRegistry 最近一次构建的工具注册表（工具列表与启用状态查询用）。
var (
	ToolsRegistry   *agent.Registry
	ToolsRegistryMu sync.RWMutex
)

// SetToolsRegistry 由各端（buildWebLoopOpts / buildDesktopLoopOpts /
// 启动初始化）在构建完注册表后调用，供 /api/tools 查询工具列表与状态。
func SetToolsRegistry(reg *agent.Registry) {
	ToolsRegistryMu.Lock()
	ToolsRegistry = reg
	ToolsRegistryMu.Unlock()
}

// HandleTools GET /api/tools：返回注册表中全部工具元信息（含 enabled 状态）。
func HandleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonErr(w, "仅 GET")
		return
	}
	ToolsRegistryMu.RLock()
	reg := ToolsRegistry
	ToolsRegistryMu.RUnlock()
	if reg == nil {
		jsonResp(w, []any{})
		return
	}
	metas := reg.AllToolMeta()
	jsonResp(w, metas)
}

// HandleReviewConfig GET/PUT /api/tools/review：审核配置（模式 + 黑白名单）。
// ★ 2026-08-31 会话级：query 带 convId 时读写**会话级**审核模式
//   （会话元数据持久化 + 运行中 Loop 实时更新；未设置回落工作区配置）；
//   不带 convId 维持原状——工作区级配置（所有会话共享的默认）。
func HandleReviewConfig(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	// ★ 2026-08-31 支持 root 参数（与 /api/conversations 的 workspace 一致）：
	// 多工作区/隔离验证指定存储根；缺省回落当前工作区。
	if rr := r.URL.Query().Get("root"); rr != "" {
		root = rr
	}
	if root == "" {
		jsonErr(w, "未设置工作区")
		return
	}
	convID := r.URL.Query().Get("convId")
	switch r.Method {
	case "GET":
		mode, blacklist, whitelist := agent.LoadWorkspaceReviewConfig(root)
		// 会话级模式优先（黑/白名单仍工作区级——会话仅覆盖模式）
		if convID != "" {
			if cm, err := agent.LookupConvReview(convID, root); err == nil && cm != "" {
				mode = cm
			}
		}
		jsonResp(w, map[string]any{
			"reviewMode":      mode,
			"reviewBlacklist": blacklist,
			"reviewWhitelist": whitelist,
		})
	case "PUT":
		var req struct {
			ReviewMode      string   `json:"reviewMode"`
			ReviewBlacklist []string `json:"reviewBlacklist"`
			ReviewWhitelist []string `json:"reviewWhitelist"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "无效 JSON: "+err.Error())
			return
		}
		if convID != "" {
			// 会话级：持久化到会话元数据 + 实时更新当前 Loop（会话内切换）
			if err := agent.ApplyConvReview(convID, root, req.ReviewMode); err != nil {
				jsonErr(w, "会话级保存失败: "+err.Error())
				return
			}
			jsonResp(w, map[string]string{"status": "ok", "scope": "conversation"})
			return
		}
		if err := agent.SaveWorkspaceReviewConfig(root, req.ReviewMode, req.ReviewBlacklist, req.ReviewWhitelist); err != nil {
			jsonErr(w, "保存失败: "+err.Error())
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
	default:
		jsonErr(w, "仅 GET/PUT")
	}
}
