// 工具配置共享 handler：/api/tools、/api/tools/save、/api/tools/review
// web 端（cmd/companion）与桌面端（internal/desktopbridge）共用。
// 工具注册表由各端构建 LoopOpts 时经 SetToolsRegistry 注入，
// 避免两端各自维护 lastReg（此前 web 端独有、桌面端 404 → 工具配置弹窗
// 显示「加载失败，请重试」）。
package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/agenttools"
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

// HandleToolsSave PUT /api/tools/save：保存工具开关到 .pair/tools.json 并立即生效。
func HandleToolsSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		jsonErr(w, "仅 PUT")
		return
	}
	var req struct {
		Tools map[string]bool `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "无效 JSON: "+err.Error())
		return
	}

	root := core.Root()
	if root == "" {
		jsonErr(w, "未设置工作区")
		return
	}
	cfgPath := filepath.Join(root, ".pair", "tools.json")

	// 读取现有配置并合并：只更新 Tools，保留 reviewMode/reviewBlacklist/reviewWhitelist，
	// 避免保存工具开关时把审核配置覆盖丢失。
	cfg := agent.WorkspaceToolConfig{Tools: map[string]agent.ToolConfigItem{}}
	if data, err := os.ReadFile(cfgPath); err == nil {
		var existing agent.WorkspaceToolConfig
		if err := json.Unmarshal(data, &existing); err == nil {
			cfg = existing
		}
	}
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]agent.ToolConfigItem, len(req.Tools))
	}
	for name, enabled := range req.Tools {
		e := enabled
		cfg.Tools[name] = agent.ToolConfigItem{Enabled: &e}
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		jsonErr(w, "写入失败: "+err.Error())
		return
	}

	// 立即应用到当前注册表
	ToolsRegistryMu.RLock()
	reg := ToolsRegistry
	ToolsRegistryMu.RUnlock()
	if reg == nil {
		// 注册表尚未初始化（启动时 root 为空 / 从未 run 过）→ 重建参考注册表，
		// 保证 GET /tools 有数据、保存的开关立即反映。
		reg = agent.NewRegistry()
		agent.RegisterDefaultTools(reg, root)
		agent.RegisterCommitMessageTool(reg)
		agenttools.RegisterManagementTools(reg, root)
		SetToolsRegistry(reg)
	}
	agent.LoadWorkspaceToolConfig(reg, root)

	jsonResp(w, map[string]string{"status": "ok"})
}

// HandleReviewConfig GET/PUT /api/tools/review：工作区级审核配置（模式 + 黑白名单）。
func HandleReviewConfig(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	if root == "" {
		jsonErr(w, "未设置工作区")
		return
	}
	switch r.Method {
	case "GET":
		mode, blacklist, whitelist := agent.LoadWorkspaceReviewConfig(root)
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
		if err := agent.SaveWorkspaceReviewConfig(root, req.ReviewMode, req.ReviewBlacklist, req.ReviewWhitelist); err != nil {
			jsonErr(w, "保存失败: "+err.Error())
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
	default:
		jsonErr(w, "仅 GET/PUT")
	}
}
