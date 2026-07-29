package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wb-ui/bridge"
	"wb-ui/jsc"
	"wb-ui/webkit"

	"github.com/hoonfeng/paircode/internal/agent"
	pairBridge "github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
)

var bridgeRegistry = pairBridge.NewRegistry()
var bridgeSessionManager = agent.NewSessionManager()

func InitDesktopBridge(wv *webkit.WebView) {
	rt := wv.JSInterpreter()
	if rt == nil {
		log.Printf("[Bridge] JSInterpreter 为 nil")
		return
	}
	log.Printf("[Bridge] 初始化桌面端桥接...")
	registerHandlers()

	bridge.Register("/bridge/call", func(args []jsc.JSValue) (jsc.JSValue, error) {
		if len(args) < 3 {
			return jsc.StringValue(`{"status":400,"body":"{\"error\":\"param missing\"}"}`), nil
		}
		method := args[0].ToString()
		path := args[1].ToString()
		bodyJSON := args[2].ToString()
		paramsJSON := ""
		if len(args) > 3 {
			paramsJSON = args[3].ToString()
		}
		return jsc.StringValue(handleBridgeCall(method, path, bodyJSON, paramsJSON)), nil
	})

	bridge.InjectAll(rt)
	injectJSBridge(rt)

	sdk := bridge.InjectSDK()
	if sdk != "" {
		rt.RunJS(sdk)
	}
	log.Printf("[Bridge] 完成, 已注册 %d 个处理器", len(bridgeRegistry.AllRoutes()))
}

func injectJSBridge(rt *jsc.Interpreter) {
	rt.RunJS(`(function(){
		window.__DESKTOP_MODE__ = true;
		window.desktopBridge = {
			call: function(method, path, bodyJSON, paramsJSON) {
				try {
					var r = go.bridge_call(method, path||'', bodyJSON||'', paramsJSON||'');
					return Promise.resolve(r);
				} catch(e) {
					return Promise.reject('[Bridge] ' + (e.message||e));
				}
			},
			onAgentEvent: null,
			onStatus: null
		};
	})()`)
}
// ─── handler registration ──────────────────────────────────

func registerHandlers() {
	bridgeRegistry.Register("GET", "/api/health", hHealth)
	bridgeRegistry.Register("GET", "/api/workspace", hWorkspace)
	bridgeRegistry.Register("GET", "/api/fs/list", hFSList)
	bridgeRegistry.Register("GET", "/api/fs/read", hFSRead)
	bridgeRegistry.Register("POST", "/api/fs/write", hFSWrite)
	bridgeRegistry.Register("GET", "/api/fs/search", hFSSearch)
	bridgeRegistry.Register("POST", "/api/fs/mkdir", hFSMkdir)
	bridgeRegistry.Register("DELETE", "/api/fs/delete", hFSDelete)
	bridgeRegistry.Register("POST", "/api/fs/rename", hFSRename)
	bridgeRegistry.Register("GET", "/api/settings", hSettings)
	bridgeRegistry.Register("GET", "/api/sysinfo", hSysInfo)
	bridgeRegistry.Register("GET", "/api/models", hModels)
	bridgeRegistry.Register("GET", "/api/conversations", hConversations)
	bridgeRegistry.Register("GET", "/api/conversations/{id}/messages", hConvMessages)
	bridgeRegistry.Register("POST", "/api/chat/send", hChatSend)
	bridgeRegistry.Register("POST", "/api/chat/stop", hChatStop)
	bridgeRegistry.Register("GET", "/api/git/status", hGitStatus)
	bridgeRegistry.Register("GET", "/api/git/log", hGitLog)
}

// ─── bridge call dispatch ──────────────────────────────────

func handleBridgeCall(method, path, bodyJSON, paramsJSON string) string {
	bodyReader := strings.NewReader(bodyJSON)
	httpReq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		return errResp(400, "req failed: "+err.Error())
	}
	if bodyJSON != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if paramsJSON != "" {
		var params map[string]string
		if json.Unmarshal([]byte(paramsJSON), &params) == nil {
			q := httpReq.URL.Query()
			for k, v := range params {
				q.Set(k, v)
			}
			httpReq.URL.RawQuery = q.Encode()
		}
	}
	vw := &virtRW{headers: http.Header{}, body: strings.Builder{}, status: 200}
	if !bridgeRegistry.Dispatch(method, path, vw, httpReq) {
		return errResp(404, "no route: "+method+" "+path)
	}
	return okResp(vw.status, vw.body.String())
}

type virtRW struct {
	headers http.Header
	body    strings.Builder
	status  int
}
func (v *virtRW) Header() http.Header         { return v.headers }
func (v *virtRW) Write(b []byte) (int, error) { return v.body.Write(b) }
func (v *virtRW) WriteHeader(s int)            { v.status = s }

func errResp(status int, msg string) string {
	b, _ := json.Marshal(map[string]interface{}{"status": status, "body": `{"error":"` + msg + `"}`})
	return string(b)
}
func okResp(status int, body string) string {
	b, _ := json.Marshal(map[string]interface{}{"status": status, "body": body})
	return string(b)
}
func jsonStr(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ─── handler implementations ───────────────────────────────

func hHealth(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, map[string]interface{}{"status": "ok", "time": time.Now().UnixMilli()})
}

func hWorkspace(w http.ResponseWriter, r *http.Request) {
	root := core.Root()
	if root == "" {
		root, _ = os.Getwd()
	}
	jsonStr(w, map[string]interface{}{
		"name":          filepath.Base(root),
		"path":          root,
		"workspaceRoot": root,
		"workspacePaths": core.Folders,
	})
}

func hFSList(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("path")
	if dir == "" {
		dir = r.URL.Query().Get("dir")
	}
	if dir == "" {
		dir = core.Root()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
		return
	}
	type item struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"isDir"`
		Size    int64  `json:"size"`
		ModTime int64  `json:"modTime"`
	}
	var items []item
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, ".") || n == "node_modules" {
			continue
		}
		info, _ := e.Info()
		sz, mt := int64(0), int64(0)
		if info != nil {
			sz = info.Size()
			mt = info.ModTime().UnixMilli()
		}
		items = append(items, item{Name: n, IsDir: e.IsDir(), Size: sz, ModTime: mt})
	}
	jsonStr(w, items)
}

func hFSRead(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		var b struct{ Path string `json:"path"` }
		json.NewDecoder(r.Body).Decode(&b)
		p = b.Path
	}
	if p == "" {
		http.Error(w, `{"error":"path required"}`, 400)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), 500)
		return
	}
	w.Write(data)
}

func hFSWrite(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if b.Path == "" {
		http.Error(w, `{"error":"path required"}`, 400)
		return
	}
	os.MkdirAll(filepath.Dir(b.Path), 0755)
	os.WriteFile(b.Path, []byte(b.Content), 0644)
	jsonStr(w, map[string]string{"status": "ok"})
}

func hFSSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	root := r.URL.Query().Get("root")
	if root == "" {
		root = core.Root()
	}
	if q == "" {
		jsonStr(w, []interface{}{})
		return
	}
	type result struct {
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}
	var res []result
	filepath.WalkDir(root, func(fp string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(q)) {
			res = append(res, result{Path: fp, Line: 1, Content: d.Name()})
		}
		if len(res) >= 50 {
			return filepath.SkipAll
		}
		return nil
	})
	jsonStr(w, res)
}

func hFSMkdir(w http.ResponseWriter, r *http.Request) {
	var b struct{ Path string `json:"path"` }
	json.NewDecoder(r.Body).Decode(&b)
	if b.Path == "" {
		http.Error(w, `{"error":"path required"}`, 400)
		return
	}
	os.MkdirAll(b.Path, 0755)
	jsonStr(w, map[string]string{"status": "ok"})
}

func hFSDelete(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	if p == "" {
		var b struct{ Path string `json:"path"` }
		json.NewDecoder(r.Body).Decode(&b)
		p = b.Path
	}
	if p == "" {
		http.Error(w, `{"error":"path required"}`, 400)
		return
	}
	os.RemoveAll(p)
	jsonStr(w, map[string]string{"status": "ok"})
}

func hFSRename(w http.ResponseWriter, r *http.Request) {
	var b struct {
		OldPath string `json:"oldPath"`
		NewPath string `json:"newPath"`
	}
	json.NewDecoder(r.Body).Decode(&b)
	if b.OldPath == "" || b.NewPath == "" {
		http.Error(w, `{"error":"oldPath and newPath required"}`, 400)
		return
	}
	os.MkdirAll(filepath.Dir(b.NewPath), 0755)
	os.Rename(b.OldPath, b.NewPath)
	jsonStr(w, map[string]string{"status": "ok"})
}

func hSettings(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, map[string]interface{}{
		"autoCommit": false, "reviewMode": "off",
		"maxTokens": 8192, "temperature": 0.7,
	})
}

func hSysInfo(w http.ResponseWriter, r *http.Request) {
	wd, _ := os.Getwd()
	jsonStr(w, map[string]interface{}{
		"platform": "windows (desktop)", "version": "1.1.5-desktop",
		"cwd": wd, "goos": "windows",
	})
}

func hModels(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, []map[string]interface{}{
		{"id": "gpt-4o", "name": "GPT-4o", "provider": "openai"},
		{"id": "deepseek-chat", "name": "DeepSeek Chat", "provider": "deepseek"},
	})
}

func hConversations(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, []interface{}{})
}

func hConvMessages(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, []interface{}{})
}

func hChatSend(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, map[string]string{"status": "ok", "convId": "desktop_conv"})
}

func hChatStop(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, map[string]string{"status": "ok"})
}

func hGitStatus(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, map[string]interface{}{
		"branch": "master", "changed": []interface{}{},
	})
}

func hGitLog(w http.ResponseWriter, r *http.Request) {
	jsonStr(w, []interface{}{})
}
