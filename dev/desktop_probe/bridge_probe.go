// 模拟 desktop 的 bridge_call 链路，验证 /api/conversations/{id}/messages 返回
// 是否完整（是否被截断）。用法：go run dev/desktop_probe/bridge_probe.go <convID>
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hoonfeng/paircode/internal/agent"
	pairBridge "github.com/hoonfeng/paircode/internal/bridge"
	"github.com/hoonfeng/paircode/internal/core"
	"github.com/hoonfeng/paircode/internal/server/handler"
)

func main() {
	convID := "conv_1785776272867467500"
	if len(os.Args) > 1 {
		convID = os.Args[1]
	}

	core.Load()
	core.LoadLastProject()
	fmt.Printf("workspace root: %s\n", core.Root())

	sm := agent.NewSessionManager()
	sm.SetWorkspaceRoot(core.Root())

	handler.AgentMgr = sm
	handler.BuildLoopOpts = func(convID, message string, autonomous bool) agent.LoopOpts {
		return agent.LoopOpts{}
	}
	reg := pairBridge.NewRegistry()
	router := handler.NewRouter(nil, reg)
	handler.RegisterAll(router)

	// 模拟 handleBridgeCall
	path := "/api/conversations/" + convID + "/messages?limit=50"
	method := "GET"
	bodyReader := strings.NewReader("")
	httpReq, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		fmt.Println("req err:", err)
		return
	}
	vw := &vw{headers: http.Header{}, body: strings.Builder{}, status: 200}
	dispatchPath := path
	if qIdx := strings.IndexByte(dispatchPath, '?'); qIdx >= 0 {
		dispatchPath = dispatchPath[:qIdx]
	}
	ok := reg.Dispatch(method, dispatchPath, vw, httpReq)
	fmt.Printf("dispatch: %v status=%d bodyLen=%d\n", ok, vw.status, vw.body.Len())

	body := vw.body.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		fmt.Println("BODY JSON PARSE ERR:", err)
		fmt.Println("body head:", body[:min(300, len(body))])
		return
	}
	msgs, _ := parsed["messages"].([]any)
	total, _ := parsed["total"].(float64)
	fmt.Printf("parsed messages=%d total=%.0f\n", len(msgs), total)
	for i, m := range msgs {
		mm := m.(map[string]any)
		role := ""
		if msgObj, ok := mm["message"].(map[string]any); ok {
			role, _ = msgObj["role"].(string)
		} else if r, ok := mm["role"].(string); ok {
			role = r
		}
		fmt.Printf("  [%d] idx=%v role=%s\n", i, mm["idx"], role)
		if i >= 15 {
			fmt.Printf("  ... (%d more)\n", len(msgs)-16)
			break
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type vw struct {
	headers http.Header
	body    strings.Builder
	status  int
}

func (v *vw) Header() http.Header         { return v.headers }
func (v *vw) Write(b []byte) (int, error) { return v.body.Write(b) }
func (v *vw) WriteHeader(s int)            { v.status = s }
