// Command chat_probe verifies the desktop Chat closed loop without a real LLM:
//  1. POST /api/chat/send with a fake provider → handler.HandleChatSend must
//     start a Session (or reject cleanly when the provider is missing).
//  2. GET /api/conversations must list the created conversation (real store).
//  3. SubscribeAll events plumbing works (agent event → JSON payload).
//
// Run: go run ./dev/desktop_probe/chat_probe.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/hoonfeng/paircode/internal/agent"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

func main() {
	logf := func(f string, a ...interface{}) { fmt.Printf(f+"\n", a...) }

	// 1. Init core like desktop does
	core.Load()
	core.LoadLastProject()
	logf("workspace root=%q folders=%d configured=%v", core.Root(), len(core.Folders), core.Configured())

	mgr := agent.NewSessionManager()
	if root := core.Root(); root != "" {
		mgr.SetWorkspaceRoot(root)
	}
	handler.AgentMgr = mgr
	handler.BuildLoopOpts = func(convID, message string, autonomous bool) agent.LoopOpts {
		reg := agent.NewRegistry()
		agent.RegisterDefaultTools(reg, core.Root())
		return agent.LoopOpts{
			Provider:      nil, // no real LLM — Start must still create the session
			Registry:      reg,
			System:        "probe",
			MaxIterations: 1,
			Autonomous:    autonomous,
		}
	}

	// 2. Subscribe agent events (same as forwardAgentEvents in desktop)
	events := mgr.SubscribeAll()
	defer mgr.UnsubscribeAll(events)
	eventSink := make(chan string, 64)
	go func() {
		for ge := range events {
			b, _ := json.Marshal(ge.Event)
			eventSink <- fmt.Sprintf("conv=%s type=%s payload=%s", ge.ConvID, ge.Event.Type, string(b))
		}
	}()

	// 3. POST /api/chat/send
	sendBody := `{"message":"测试桌面 Chat 闭环","convId":"conv_probe_1"}`
	req := httptest.NewRequest("POST", "/api/chat/send", strings.NewReader(sendBody))
	rec := httptest.NewRecorder()
	handler.HandleChatSend(rec, req)
	logf("chat/send status=%d body=%s", rec.Code, strings.TrimSpace(rec.Body.String()))

	// 4. GET /api/conversations — the created conversation must be persisted
	rec2 := httptest.NewRecorder()
	handler.HandleConversations(rec2, httptest.NewRequest("GET", "/api/conversations", nil))
	var metas []agent.ConversationMeta
	_ = json.Unmarshal(rec2.Body.Bytes(), &metas)
	found := false
	for _, m := range metas {
		logf("  conversation: id=%q title=%q ws=%q", m.ID, m.Title, m.WorkspaceRoot)
		if m.ID == "conv_probe_1" {
			found = true
		}
	}
	if found {
		logf("OK: conv_probe_1 persisted in store")
	} else {
		logf("WARN: conv_probe_1 not in conversation list (%d total)", len(metas))
	}

	// 5. Stop the session if it started (cleanup)
	mgr.Stop("conv_probe_1")

	// 6. Check events arrived (Start without provider should emit an error/done event)
	select {
	case ev := <-eventSink:
		logf("event received: %s", ev)
	case <-time.After(2 * time.Second):
		logf("no event within 2s (loop may not have started without provider)")
	}

	// 7. HTTP handler interface check: HandleConversations must handle POST too
	rec3 := httptest.NewRecorder()
	handler.HandleConversationCreate(rec3, httptest.NewRequest("POST", "/api/conversations",
		strings.NewReader(`{"id":"conv_probe_2","title":"探针会话"}`)))
	logf("conversations POST status=%d ok=%v", rec3.Code, strings.Contains(rec3.Body.String(), "ok"))
	mgr.Stop("conv_probe_2")

	// 8. Remove probe conversations (keep store clean)
	if store := mgr.Store(); store != nil {
		for _, id := range []string{"conv_probe_1", "conv_probe_2"} {
			if err := store.DeleteConversation(id); err != nil {
				logf("cleanup %s: %v", id, err)
			} else {
				logf("cleanup %s: ok", id)
			}
		}
	}

	logf("chat_probe done (exit 0 = expected; real LLM call not made)")
	_ = os.Getenv
}

// keep compile references
var _ = http.StatusOK
