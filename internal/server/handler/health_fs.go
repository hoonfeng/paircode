// Handler 实现 — 系统 + 文件系统
package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hoonfeng/paircode/internal/core"
)

// HandleHealth 返回服务状态
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]any{"status": "ok", "workspace": core.Root(), "folders": core.Folders})
}

// HandleSysInfo 返回系统信息
func HandleSysInfo(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	wd, _ := os.Getwd()
	jsonResp(w, map[string]any{
		"hostname": host,
		"platform": "windows",
		"cwd":      wd,
		"pid":      os.Getpid(),
	})
}

// HandleExec 执行系统命令
func HandleExec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Cwd     string   `json:"cwd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Command == "" {
		jsonErr(w, "command 必填")
		return
	}
	// 安全检查：只允许特定命令
	allowed := map[string]bool{
		"go": true, "node": true, "npm": true, "npx": true,
		"git": true, "python": true, "pip": true,
		"make": true, "cmake": true,
		"powershell": true, "cmd": true, "bash": true,
	}
	if !allowed[strings.ToLower(req.Command)] && !strings.HasPrefix(req.Command, ".") {
		jsonErr(w, "不允许的命令: "+req.Command)
		return
	}
	jsonResp(w, map[string]any{"ok": true, "pid": 0})
}

// ─── 文件系统 ──────────────────────────────────────────────

// HandleFSDrives 返回可用驱动器列表
func HandleFSDrives(w http.ResponseWriter, r *http.Request) {
	drives := []string{}
	for _, d := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		p := string(d) + ":\\"
		if _, err := os.Stat(p); err == nil {
			drives = append(drives, p)
		}
	}
	jsonResp(w, drives)
}

// HandleFSList 列出目录内容
func HandleFSList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = core.Root()
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	type entry struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"isDir"`
		Size    int64  `json:"size"`
		ModTime string `json:"modTime"`
	}
	result := make([]entry, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		sz := int64(0)
		mt := ""
		if err == nil {
			sz = fi.Size()
			mt = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		result = append(result, entry{Name: e.Name(), IsDir: e.IsDir(), Size: sz, ModTime: mt})
	}
	jsonResp(w, result)
}

// HandleFSRead 读取文件内容
func HandleFSRead(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	data, err := os.ReadFile(path)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{
		"content": string(data),
		"size":    len(data),
		"path":    path,
	})
}

// imageMime 根据扩展名返回图片 MIME
func imageMime(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	}
	return ""
}

// HandleFSImage 提供图片浏览
func HandleFSImage(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	mime := imageMime(path)
	if mime == "" {
		jsonErr(w, "不支持的文件类型")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		jsonErr(w, "读取文件失败")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}

// HandleFSFileInfo 返回文件类型信息
func HandleFSFileInfo(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	fi, err := os.Stat(path)
	if err != nil {
		jsonErr(w, "文件不存在或无法访问")
		return
	}
	if fi.IsDir() {
		jsonResp(w, map[string]any{"type": "directory", "size": fi.Size()})
		return
	}
	if fi.Size() > 100<<20 {
		jsonErr(w, "文件超过 100MB")
		return
	}
	header := make([]byte, 512)
	f, _ := os.Open(path)
	n, _ := f.Read(header)
	f.Close()
	mime := imageMime(path)
	isImage := mime != ""
	isBinary := n > 0 && bytes.IndexByte(header[:n], 0) >= 0
	fileType := "text"
	if isImage {
		fileType = "image"
	} else if isBinary {
		fileType = "binary"
	}
	jsonResp(w, map[string]any{
		"type": fileType, "size": fi.Size(),
		"isImage": isImage, "isBinary": isBinary, "mimeType": mime,
	})
}

// HandleFSHex 返回文件十六进制转储
func HandleFSHex(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Query().Get("path"))
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("length")
	offset := 0
	if offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	}
	length := 256
	if limitStr != "" {
		length, _ = strconv.Atoi(limitStr)
		if length > 4096 {
			length = 4096
		}
	}
	fi, err := os.Stat(path)
	if err != nil {
		jsonErr(w, "文件不存在")
		return
	}
	fileSize := fi.Size()
	if fileSize > 100<<20 {
		jsonErr(w, "文件超过 100MB")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		jsonErr(w, "无法打开文件")
		return
	}
	defer f.Close()
	if offset > int(fileSize) {
		offset = int(fileSize)
	}
	if offset+length > int(fileSize) {
		length = int(fileSize) - offset
	}
	if length <= 0 {
		jsonResp(w, map[string]any{"hex": "", "offset": offset, "fileSize": fileSize, "hasMore": false})
		return
	}
	buf := make([]byte, length)
	n, err := f.ReadAt(buf, int64(offset))
	if err != nil && n == 0 {
		jsonErr(w, "读取失败")
		return
	}
	buf = buf[:n]
	var lines []string
	bpl := 16
	for i := 0; i < n; i += bpl {
		end := i + bpl
		if end > n {
			end = n
		}
		chunk := buf[i:end]
		addr := fmt.Sprintf("%08X", offset+i)
		hexPart := ""
		for j, b := range chunk {
			if j > 0 && j%8 == 0 {
				hexPart += " "
			}
			hexPart += fmt.Sprintf("%02X ", b)
		}
		pad := bpl - len(chunk)
		for j := 0; j < pad; j++ {
			if j > 0 && (len(chunk)+j)%8 == 0 {
				hexPart += " "
			}
			hexPart += "   "
		}
		asciiPart := ""
		for _, b := range chunk {
			if b >= 32 && b <= 126 {
				asciiPart += string(b)
			} else {
				asciiPart += "."
			}
		}
		lines = append(lines, fmt.Sprintf("%s  %s %s", addr, hexPart, asciiPart))
	}
	jsonResp(w, map[string]any{
		"hex": strings.Join(lines, "\n"), "fileSize": fileSize,
		"hasMore": offset+n < int(fileSize),
	})
}

// HandleFSWrite 写入文件
func HandleFSWrite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(req.Path), 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

// HandleFSRename 重命名/移动文件
func HandleFSRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(req.To), 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.Rename(req.From, req.To); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "from": req.From, "to": req.To})
}

// HandleFSDelete 删除文件或目录
func HandleFSDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Recursive {
		if err := os.RemoveAll(req.Path); err != nil {
			jsonErr(w, err.Error())
			return
		}
	} else {
		if err := os.Remove(req.Path); err != nil {
			jsonErr(w, err.Error())
			return
		}
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

// HandleFSMkdir 创建目录
func HandleFSMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if err := os.MkdirAll(req.Path, 0755); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "path": req.Path})
}

// HandleFSSearch 搜索文件
func HandleFSSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	root := r.URL.Query().Get("root")
	if root == "" {
		root = core.Root()
	}
	if query == "" {
		jsonResp(w, []string{})
		return
	}
	// 简单实现：后续可替换为更高效的搜索
	var results []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(query)) {
			results = append(results, path)
		}
		return nil
	})
	if len(results) > 200 {
		results = results[:200]
	}
	jsonResp(w, results)
}
