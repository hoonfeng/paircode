// Command webkit_timer_probe verifies setTimeout through the full webkit
// WebView (LoadHTML + RegisterDOMBindings) with manual ProcessTasks driving,
// to isolate whether the event loop works outside app.Host.
package main

import (
	"fmt"
	"time"

	"wb-ui/jsc"
	"wb-ui/webkit"
)

func main() {
	wv := webkit.NewWebView()
	rt := wv.JSInterpreter()
	el := rt.GetEventLoop()
	if el == nil {
		el = jsc.NewEventLoop(rt)
	}

	err := wv.LoadHTML(`<!DOCTYPE html><html><body><div id="r">no</div></body></html>`)
	if err != nil {
		fmt.Println("LoadHTML err:", err)
		return
	}
	if _, err := rt.RunJS(`(function(){
		__fired = '';
		__sid = setTimeout(function(){ __fired = 'timeout'; var d=document.getElementById('r'); if(d) d.textContent='YES'; }, 300);
	})()`); err != nil {
		fmt.Printf("inject err: %v\n", err)
		return
	}
	st, _ := rt.RunJS(`typeof setTimeout`)
	sid, _ := rt.RunJS(`__sid`)
	fmt.Printf("after inject: pending=%d typeof setTimeout=%s sid=%s\n", el.PendingTasks(), st.ToString(), sid.ToString())
	// 直接检查 SetTimeout 是否注册（返回 ID）
	cb, cbErr := rt.RunJS(`function(){ __probe = 1; }`)
	fmt.Printf("cbErr=%v callable=%v\n", cbErr, cb.IsCallable())
	id := el.SetTimeout(cb, 50)
	fmt.Printf("direct SetTimeout id=%d pending=%d\n", id, el.PendingTasks())

	for frame := 0; frame < 60; frame++ {
		time.Sleep(16 * time.Millisecond)
		el.ProcessTasks(int64(frame * 16))
		if frame%10 == 0 {
			v, _ := rt.RunJS(`__fired`)
			fmt.Printf("frame=%d pending=%d fired=%q\n", frame, el.PendingTasks(), v.ToString())
		}
		if v, _ := rt.RunJS(`__fired`); v.ToString() == "timeout" {
			div, _ := rt.RunJS(`document.getElementById('r').textContent`)
			fmt.Printf("OK: setTimeout fired; #r=%s\n", div.ToString())
			return
		}
	}
	fmt.Println("FAIL: setTimeout not fired in webkit env")
}
