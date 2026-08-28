// Git Handler 实现（从 web_server.go 迁移）
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hoonfeng/paircode/pkg/executil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/core"
)

// ─── Git 辅助 ──────────────────────────────────────────────

type gitDirCtxKey struct{}

type gitStatusResult struct {
	Branch    string           `json:"branch"`
	Ahead     int              `json:"ahead"`
	Behind    int              `json:"behind"`
	IsRepo    bool             `json:"isRepo"`
	Staged    []gitChangeEntry `json:"staged"`
	Conflict  []gitChangeEntry `json:"conflict"`
	Modified  []gitChangeEntry `json:"modified"`
	Untracked []gitChangeEntry `json:"untracked"`
	Branches  []string         `json:"brances"`
	Error     string           `json:"error,omitempty"`
}

type gitChangeEntry struct {
	Path string `json:"path"`
	X    string `json:"x"`
	Y    string `json:"y"`
}

func withGitDir(r *http.Request) *http.Request {
	if p := r.URL.Query().Get("path"); p != "" {
		clean := filepath.Clean(p)
		ctx := context.WithValue(r.Context(), gitDirCtxKey{}, clean)
		return r.WithContext(ctx)
	}
	return r
}

func gitRoot(ctx context.Context) string {
	if ctx != nil {
		if dir, ok := ctx.Value(gitDirCtxKey{}).(string); ok && dir != "" {
			return filepath.Clean(dir)
		}
	}
	return core.Root()
}

func runGit(ctx context.Context, args ...string) (string, error) {
	dir := gitRoot(ctx)
	if dir == "" {
		return "", fmt.Errorf("未设置工作区")
	}
	// 30s 超时防止阻塞
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	fullArgs := append([]string{"-C", dir, "-c", "core.quotepath=false"}, args...)
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	if runtime.GOOS == "windows" {
		executil.HideWindow(cmd)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		if out != "" {
			return out, nil
		}
		return "", fmt.Errorf("%s", errMsg)
	}
	return out, nil
}

// ─── Handler ──────────────────────────────────────────────

func HandleGitStatus(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	res := gitStatusResult{}

	out, err := runGit(r.Context(), "rev-parse", "--is-inside-work-tree")
	if err != nil || out != "true" {
		res.Error = "非 Git 仓库或未设置工作区"
		jsonResp(w, res)
		return
	}
	res.IsRepo = true
	if b, err := runGit(r.Context(), "branch", "--show-current"); err == nil {
		res.Branch = b
	}
	if ab, err := runGit(r.Context(), "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		fmt.Sscanf(ab, "%d\t%d", &res.Ahead, &res.Behind)
	}
	statusOut, err := runGit(r.Context(), "status", "--porcelain")
	if err == nil {
		for _, line := range strings.Split(statusOut, "\n") {
			if len(line) < 4 {
				continue
			}
			x, y := string(line[0]), string(line[1])
			p := strings.TrimSpace(line[3:])
			if i := strings.Index(p, " -> "); i >= 0 {
				p = p[i+4:]
			}
			e := gitChangeEntry{Path: p, X: x, Y: y}
			switch {
			case x == "?" && y == "?":
				res.Untracked = append(res.Untracked, e)
			case x == "U" || y == "U" || (x == "D" && y == "D") || (x == "A" && y == "A"):
				res.Conflict = append(res.Conflict, e)
			default:
				if x != " " && x != "?" {
					res.Staged = append(res.Staged, e)
				}
				if y != " " && y != "?" {
					res.Modified = append(res.Modified, e)
				}
			}
		}
	}
	if branches, err := runGit(r.Context(), "branch", "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(branches, "\n") {
			if b = strings.TrimSpace(b); b != "" {
				res.Branches = append(res.Branches, b)
			}
		}
	}
	jsonResp(w, res)
}

func HandleGitInit(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGit(r.Context(), "init")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": out})
}

func HandleGitDiff(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	p := r.URL.Query().Get("path")
	args := []string{"diff", "--no-color"}
	if p != "" {
		args = append(args, "--", filepath.Clean(p))
	}
	out, err := runGit(r.Context(), args...)
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, out)
}

func HandleGitAdd(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"add"}
	if req.Path != "" {
		args = append(args, filepath.Clean(req.Path))
	} else {
		args = append(args, ".")
	}
	if _, err := runGit(r.Context(), args...); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitReset(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"reset"}
	if req.Path != "" {
		args = append(args, "--", filepath.Clean(req.Path))
	}
	if _, err := runGit(r.Context(), args...); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitCommit(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if req.Message == "" {
		jsonErr(w, "提交信息不能为空")
		return
	}
	if _, err := runGit(r.Context(), "commit", "-m", req.Message); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitLog(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	n := r.URL.Query().Get("count")
	if n == "" {
		n = "50"
	}
	out, err := runGit(r.Context(), "log", "--oneline", "--decorate", "-"+n,
		"--pretty=format:%h|%an|%ar|%s")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	type entry struct {
		Hash    string `json:"hash"`
		Author  string `json:"author"`
		Date    string `json:"date"`
		Message string `json:"message"`
	}
	var list []entry
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			list = append(list, entry{parts[0], parts[1], parts[2], parts[3]})
		}
	}
	jsonResp(w, list)
}

func HandleGitBranch(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Name   string `json:"name"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	switch req.Action {
	case "create":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		if _, err := runGit(r.Context(), "branch", req.Name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true, "branch": req.Name})
	case "delete":
		if req.Name == "" {
			jsonErr(w, "分支名不能为空")
			return
		}
		if _, err := runGit(r.Context(), "branch", "-D", req.Name); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
	default:
		jsonErr(w, "未知操作: "+req.Action)
	}
}

func HandleGitCheckout(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Branch string `json:"branch"`
		Create bool   `json:"create"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"checkout"}
	if req.Create {
		args = append(args, "-b")
	}
	args = append(args, req.Branch)
	if _, err := runGit(r.Context(), args...); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitStash(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	args := []string{"stash"}
	switch req.Action {
	case "", "push":
		args = append(args, "push")
	case "pop":
		args = append(args, "pop")
	case "drop":
		args = append(args, "drop")
	default:
		jsonErr(w, "未知 stash 操作: "+req.Action)
		return
	}
	if _, err := runGit(r.Context(), args...); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitStashList(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGit(r.Context(), "stash", "list")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, out)
}

func HandleGitIgnore(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	gitDir := gitRoot(r.Context())
	p := filepath.Join(gitDir, ".gitignore")
	if r.Method == "POST" {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, err.Error())
			return
		}
		if err := os.WriteFile(p, []byte(req.Content), 0644); err != nil {
			jsonErr(w, err.Error())
			return
		}
		jsonResp(w, map[string]any{"ok": true})
		return
	}
	data, _ := os.ReadFile(p)
	jsonResp(w, string(data))
}

func HandleGitDiscard(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, err.Error())
		return
	}
	if _, err := runGit(r.Context(), "checkout", "--", filepath.Clean(req.Path)); err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true})
}

func HandleGitPush(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGit(r.Context(), "push")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": out})
}

func HandleGitPull(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGit(r.Context(), "pull")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	jsonResp(w, map[string]any{"ok": true, "message": out})
}

func HandleGitRemote(w http.ResponseWriter, r *http.Request) {
	r = withGitDir(r)
	out, err := runGit(r.Context(), "remote", "-v")
	if err != nil {
		jsonErr(w, err.Error())
		return
	}
	type remote struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	var list []remote
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			list = append(list, remote{Name: parts[0], URL: parts[1]})
		}
	}
	jsonResp(w, list)
}
