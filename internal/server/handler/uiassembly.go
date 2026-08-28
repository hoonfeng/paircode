// UI 装配状态磁盘持久化：GET/PUT /api/ui-assembly。
//
// 背景：UI 槽位装配状态（slotOwner/slotOverlay/slotUIEnabled——即「哪个插件占哪个
// 区域、list 勾选、插件级 UI 开关」）原本只存在浏览器 localStorage——用户无法直接
// 修改文件；且插件面板（PluginPanel）由 ui-sidebar 插件承载，若停用该插件面板入口
// 即消失（死锁）。本接口把装配状态落到 .pair/ui-assembly.json：
//   - 用户可直接编辑该文件控制 UI 装配（面板被停用/锁死时的逃生通道）；
//   - 前端启动时 merge 进 localStorage（文件优先），变更时防抖写回。
package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/hoonfeng/paircode/internal/core"
)

var uiAssemblyMu sync.Mutex

// HandleUIAssembly GET/PUT /api/ui-assembly：UI 装配状态磁盘持久化。
func HandleUIAssembly(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	switch r.Method {
	case "GET":
		uiAssemblyMu.Lock()
		defer uiAssemblyMu.Unlock()
		// 空工作区（未打开文件夹）：不存在装配文件，语义上就是空状态——
		// 返回 {} 而非 400（早期实现返回"未设置工作区"400，网络面板出现
		// 错误响应，且与「新建工作区弹窗」等前端缺陷混淆排查）。
		if root == "" {
			jsonResp(w, map[string]any{})
			return
		}
		path := filepath.Join(root, ".pair", "ui-assembly.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				jsonResp(w, map[string]any{})
				return
			}
			jsonErr(w, "读取失败: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
	case "PUT":
		var raw map[string]any
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			jsonErr(w, "无效 JSON: "+err.Error())
			return
		}
		if root == "" {
			// 空工作区无落盘位置：静默成功（装配状态本就只存 localStorage）
			jsonResp(w, map[string]string{"status": "ok"})
			return
		}
		path := filepath.Join(root, ".pair", "ui-assembly.json")
		uiAssemblyMu.Lock()
		defer uiAssemblyMu.Unlock()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			jsonErr(w, "创建目录失败: "+err.Error())
			return
		}
		data, _ := json.MarshalIndent(raw, "", "  ")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			jsonErr(w, "保存失败: "+err.Error())
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
	default:
		jsonErr(w, "仅 GET/PUT")
	}
}
