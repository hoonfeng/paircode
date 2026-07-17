// Handler 实现 — Git（桩实现，后续从 web_server.go 迁移完整逻辑）
package handler

import "net/http"

func HandleGitStatus(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func HandleGitInit(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitDiff(w http.ResponseWriter, r *http.Request)       { jsonResp(w, "") }
func HandleGitAdd(w http.ResponseWriter, r *http.Request)        { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitReset(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitCommit(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitLog(w http.ResponseWriter, r *http.Request)        { jsonResp(w, []string{}) }
func HandleGitBranch(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
func HandleGitCheckout(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitStash(w http.ResponseWriter, r *http.Request)      { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitStashList(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func HandleGitIgnore(w http.ResponseWriter, r *http.Request)     { jsonResp(w, "") }
func HandleGitDiscard(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitPush(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitPull(w http.ResponseWriter, r *http.Request)       { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleGitRemote(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }
