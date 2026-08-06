// Handler 实现 — 工作区 + 设置
package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hoonfeng/paircode/internal/core"
)

// HandleWorkspace 返回工作区信息（GET）或创建工作区（POST）
func HandleWorkspace(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		jsonResp(w, map[string]any{
			"root":    core.Root(),
			"folders": core.Folders,
			"loaded":  core.Loaded,
		})
	case "POST":
		HandleWorkspacePost(w, r)
	}
}

// HandleWorkspacePost 创建工作区或添加文件夹
func HandleWorkspacePost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action      string   `json:"action"`
		Root        string   `json:"root"`
		Folders     []string `json:"folders"`
		Name        string   `json:"name"`
		Path        string   `json:"path"`
		ParentDir   string   `json:"parentDir"`
		Lang        string   `json:"lang"`
		DeleteFiles bool     `json:"deleteFiles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	switch req.Action {
	case "create":
		if req.Name == "" && req.Root == "" {
			jsonErr(w, "需要 name 或 root 参数")
			return
		}
		root := req.Root
		if root == "" {
			home, _ := os.UserHomeDir()
			root = filepath.Join(home, "paircode-workspaces", req.Name)
		}
		if err := os.MkdirAll(root, 0755); err != nil {
			jsonErr(w, "创建工作区失败: "+err.Error())
			return
		}
		core.Folders = []string{root}
		core.Settings.LastProject = root
		core.Settings.WorkspaceFolders = core.Folders
		core.Loaded = true
		core.Save()
		if core.OnSyncWorkspace != nil {
			core.OnSyncWorkspace(true)
		}
		jsonResp(w, map[string]any{"ok": true, "root": root})

	case "add-folder":
		if req.Path == "" {
			jsonErr(w, "需要 path 参数")
			return
		}
		for _, f := range core.Folders {
			if f == req.Path {
				jsonErr(w, "文件夹已在工作区中")
				return
			}
		}
		core.Folders = append(core.Folders, req.Path)
		if len(core.Folders) == 1 {
			core.Settings.LastProject = req.Path
		}
		core.Settings.WorkspaceFolders = core.Folders
		core.Save()
		if core.OnSyncWorkspace != nil {
			core.OnSyncWorkspace(true)
		}
		jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

	case "remove-folder":
		if req.Path == "" {
			jsonErr(w, "需要 path 参数")
			return
		}
		newFolders := make([]string, 0, len(core.Folders))
		for _, f := range core.Folders {
			if f != req.Path {
				newFolders = append(newFolders, f)
			}
		}
		if len(newFolders) == len(core.Folders) {
			jsonErr(w, "文件夹不在工作区中")
			return
		}
		core.Folders = newFolders
		if len(core.Folders) > 0 {
			core.Settings.LastProject = core.Folders[0]
		}
		core.Settings.WorkspaceFolders = core.Folders
		core.Save()
		if core.OnSyncWorkspace != nil {
			core.OnSyncWorkspace(len(newFolders) > 0)
		}
		jsonResp(w, map[string]any{"ok": true, "folders": core.Folders})

	case "switch":
		// 切换主工作区（前端 FileExplorer 切换工作区列表项发 action:"switch"，
		// root=新主根，folders=附加文件夹；此前缺失此 case 落入 default 报
		// "未知操作: switch" → 前端 apiPost 抛异常 → 点击切换无效果）。
		if req.Root == "" {
			jsonErr(w, "需要 root 参数")
			return
		}
		newFolders := []string{req.Root}
		for _, f := range req.Folders {
			if f != "" && f != req.Root {
				newFolders = append(newFolders, f)
			}
		}
		core.Folders = newFolders
		core.Settings.LastProject = req.Root
		core.Settings.WorkspaceFolders = append([]string{}, newFolders...)
		core.Loaded = true
		core.Save()
		if core.OnSyncWorkspace != nil {
			core.OnSyncWorkspace(true)
		}
		jsonResp(w, map[string]any{"ok": true, "root": req.Root, "folders": newFolders})

	case "open":
		if req.Root == "" {
			jsonErr(w, "需要 root 参数")
			return
		}
		core.Folders = []string{req.Root}
		core.Settings.LastProject = req.Root
		core.Settings.WorkspaceFolders = core.Folders
		core.Loaded = true
		core.Save()
		if core.OnSyncWorkspace != nil {
			core.OnSyncWorkspace(true)
		}
		jsonResp(w, map[string]any{"ok": true, "root": req.Root})

	default:
		jsonErr(w, "未知操作: "+req.Action)
	}
}

// HandleSettings 返回设置
func HandleSettings(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{
		"settings": core.Settings,
		"loaded":   core.Loaded,
	})
}

// HandleSettingsPut 保存设置
func HandleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Settings *core.AppSettings `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Settings != nil {
		core.Settings = *req.Settings
	}
	core.Save()
	jsonResp(w, map[string]any{"ok": true})
}
