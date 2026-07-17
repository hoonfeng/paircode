// Handler 实现 — 对话/Agent（桩实现，后续从 web_server.go 迁移完整逻辑）
package handler

import "net/http"

func HandleChatSend(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleChatStop(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleChatAnswer(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleChatApprove(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleChatFeedback(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleChatRollback(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }

func HandleConversations(w http.ResponseWriter, r *http.Request)        { jsonResp(w, []string{}) }
func HandleConversationCreate(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]any{}) }
func HandleConversationByID(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]any{}) }

func HandleTasks(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func HandleTaskPlan(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]any{}) }

func HandleModels(w http.ResponseWriter, r *http.Request) { jsonResp(w, []string{}) }

func HandleInstructions(w http.ResponseWriter, r *http.Request)      { jsonResp(w, "") }
func HandleInstructionsPut(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func HandlePhilosophy(w http.ResponseWriter, r *http.Request)        { jsonResp(w, "") }
func HandlePhilosophyPut(w http.ResponseWriter, r *http.Request)     { jsonResp(w, map[string]string{"status": "ok"}) }

func HandleMCPList(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func HandleMCPSave(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }

func HandleSkillsList(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func HandleSkillsRead(w http.ResponseWriter, r *http.Request)   { jsonResp(w, "") }
func HandleSkillsSave(w http.ResponseWriter, r *http.Request)   { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleSkillsDelete(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }

func HandleTokensStats(w http.ResponseWriter, r *http.Request)    { jsonResp(w, map[string]any{}) }
func HandleDebugLogs(w http.ResponseWriter, r *http.Request)      { jsonResp(w, []string{}) }
func HandleDebugLogByID(w http.ResponseWriter, r *http.Request)   { jsonResp(w, "") }

func HandleMarketplaceSearch(w http.ResponseWriter, r *http.Request)   { jsonResp(w, []string{}) }
func HandleMarketplaceInstall(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }
func HandleMarketplaceRefresh(w http.ResponseWriter, r *http.Request)  { jsonResp(w, map[string]string{"status": "ok"}) }

func HandleMemorySearch(w http.ResponseWriter, r *http.Request)  { jsonResp(w, []string{}) }
func HandleMemoryList(w http.ResponseWriter, r *http.Request)    { jsonResp(w, []string{}) }
func HandleMemoryRebuild(w http.ResponseWriter, r *http.Request) { jsonResp(w, map[string]string{"status": "ok"}) }
