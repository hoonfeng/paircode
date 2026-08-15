// toolsets.go — 工具集 REST（前端工具集管理面板）。
//
// 与 agent 工具集工具（toolset_build/list/export/import/remove）同源，
// 提供浏览器 UI 直调通道：
//   - GET  /api/toolsets              列表（工作区 + 全局）
//   - POST /api/toolsets/build        动态构建 + 固化 + 装载
//   - GET  /api/toolsets/export       导出发布 JSON（?name=）
//   - POST /api/toolsets/import       导入（{json|file, scope}）
//   - POST /api/toolsets/remove       删除（{name, scope}）
package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
)

// workspaceRoot 主工作区根（工具集操作默认目标）。
func workspaceRoot() string {
	if r := core.Root(); r != "" {
		return r
	}
	if len(agent.WorkspaceRoots) > 0 {
		return agent.WorkspaceRoots[0]
	}
	return ""
}

// HandleToolsetsList GET /api/toolsets：工具集列表。
func HandleToolsetsList(w http.ResponseWriter, r *http.Request) {
	root := workspaceRoot()
	if root == "" {
		jsonResp(w, []any{})
		return
	}
	jsonResp(w, agent.ListAllToolsetsPublic(root))
}

// HandleToolsetBuild POST /api/toolsets/build：动态构建 + 固化 + 装载。
// body: { name?, description?, requirement?, project?, overwrite? }
func HandleToolsetBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	root := workspaceRoot()
	if root == "" {
		jsonErr(w, "工作区未就绪")
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Requirement string `json:"requirement"`
		Project     string `json:"project"`
		Overwrite   bool   `json:"overwrite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	projectDir := root
	if req.Project != "" {
		dir, err := agent.ResolveWorkspaceProjectPublic(root, req.Project)
		if err != nil {
			jsonErr(w, err.Error())
			return
		}
		projectDir = dir
	}
	name := req.Name
	if name == "" {
		name = "default"
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if !agent.ValidToolsetName(name) {
		jsonErr(w, "工具集名只能含小写字母/数字/-/_")
		return
	}
	// 已固化检查（不覆盖时拒绝）
	if !req.Overwrite {
		for _, scope := range []string{"project", "global"} {
			if _, err := os.Stat(agent.ToolsetPath(projectDir, scope, name)); err == nil {
				jsonErr(w, "工具集 "+name+" 已固化（"+scope+"）；如需重建请勾选覆盖")
				return
			}
		}
	}
	// 旧插件先卸载
	if ts, err := agent.LoadToolsetPublic(projectDir, "", name); err == nil {
		for _, p := range ts.Plugins {
			if _, ok := ph.Get(p.Name); ok {
				_ = ph.Unload(p.Name)
				_ = ph.Undefine(p.Name)
			}
		}
	}
	ts, err := agent.BuildToolset(ph, projectDir, name, req.Description, req.Requirement)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := agent.SaveToolsetPublic(projectDir, "project", ts); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{
		"ok": true, "name": ts.Name, "project": ts.Project,
		"description": ts.Description, "pluginCount": len(ts.Plugins),
		"plugins": ts.Plugins,
	})
}

// HandleToolsetExport GET /api/toolsets/export?name=：导出发布 JSON。
func HandleToolsetExport(w http.ResponseWriter, r *http.Request) {
	root := workspaceRoot()
	if root == "" {
		jsonErr(w, "工作区未就绪")
		return
	}
	name := r.URL.Query().Get("name")
	ts, err := agent.LoadToolsetPublic(root, "", name)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	content, err := agent.ExportToolsetJSON(ts, nil, "", "")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	// 下载响应
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename="+ts.Name+".toolset.json")
	_, _ = w.Write([]byte(content))
}

// HandleToolsetImport POST /api/toolsets/import：导入（{json|file, scope}）。
func HandleToolsetImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	root := workspaceRoot()
	if root == "" {
		jsonErr(w, "工作区未就绪")
		return
	}
	var req struct {
		JSON  string `json:"json"`
		File  string `json:"file"`
		Scope string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	content := req.JSON
	if content == "" {
		if req.File == "" {
			jsonErr(w, "需要 json 或 file")
			return
		}
		data, err := os.ReadFile(req.File)
		if err != nil {
			jsonErr(w, "读取导入文件失败: "+err.Error())
			return
		}
		content = string(data)
	}
	ts, err := agent.ParseToolsetPublish(content)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	scope := "project"
	if req.Scope == "user" {
		scope = "global"
	}
	if err := agent.SaveToolsetPublic(root, scope, ts); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := agent.InstallToolsetPublic(ph, ts); err != nil {
		jsonErr(w, "工具集已固化但装载失败: "+err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "name": ts.Name, "scope": scope, "pluginCount": len(ts.Plugins)})
}

// HandleToolsetRemove POST /api/toolsets/remove：删除（{name, scope}）。
func HandleToolsetRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "仅 POST")
		return
	}
	ph, ok := getPluginHost()
	if !ok {
		jsonErr(w, "插件系统未初始化")
		return
	}
	root := workspaceRoot()
	if root == "" {
		jsonErr(w, "工作区未就绪")
		return
	}
	var req struct {
		Name  string `json:"name"`
		Scope string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "请求体解析失败: "+err.Error())
		return
	}
	if req.Name == "" {
		jsonErr(w, "缺少 name")
		return
	}
	// 卸载已装载插件
	if ts, err := agent.LoadToolsetPublic(root, req.Scope, req.Name); err == nil {
		for _, p := range ts.Plugins {
			if _, ok := ph.Get(p.Name); ok {
				_ = ph.Unload(p.Name)
				_ = ph.Undefine(p.Name)
			}
		}
	}
	if err := agent.RemoveToolsetPublic(root, req.Scope, req.Name); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "name": req.Name})
}

// filepath 引用（ResolveWorkspaceProject 相对路径解析用）。
var _ = filepath.Join
