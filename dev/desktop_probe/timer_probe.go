// Command timer_probe verifies setTimeout fires after the event-loop time
// fix: SetTimeout uses el.nowMs() (EventLoop-relative), ProcessTasks must
// compare against the same clock, not the host's animStart elapsedMs.
package main

import (
	"fmt"
	"time"

	"wb-ui/jsc"
)

func main() {
	rt := jsc.NewInterpreter()
	elStart := time.Now().UnixMilli()
	el := jsc.NewEventLoop(rt)
	rt.SetupGlobal(nil)
	rt.InjectBrowserEnv() // setTimeout/setInterval/requestAnimationFrame 注册

	fired := make(chan string, 4)
	// 注入一个辅助 JS：setTimeout 存标记 + setInterval 计次
	if _, err := rt.RunJS(`(function(){
		__fired = '';
		__intv = 0;
		setTimeout(function(){ __fired = 'timeout'; }, 300);
		setInterval(function(){ __intv++; }, 100);
	})()`); err != nil {
		fmt.Printf("RUNJS ERR: %v\n", err)
		return
	}
	fmt.Printf("after inject: pending=%d\n", el.PendingTasks())

	// 模拟 host 逐帧驱动（每 16ms 一帧，传一个任意 elapsedMs——应被忽略）
	for frame := 0; frame < 60; frame++ {
		time.Sleep(16 * time.Millisecond)
		// 故意传错误基准（比如从 0 开始的小值），验证 ProcessTasks 用内部时钟
		el.ProcessTasks(int64(frame * 16))
		if frame%10 == 0 {
			pending := el.PendingTasks()
			now := time.Now().UnixMilli() - elStart
			fl, _ := rt.RunJS(`typeof __fired + ':' + __fired`)
			iv, _ := rt.RunJS(`__intv`)
			fmt.Printf("frame=%d pending=%d hostElapsed=%dms elNow≈%dms fired=%s intv=%s\n", frame, pending, frame*16, now, fl.ToString(), iv.ToString())
		}
		if v, err := rt.RunJS(`__fired`); err == nil && v.ToString() == "timeout" {
			// 验证 interval 至少触发 2 次
			iv, _ := rt.RunJS(`__intv`)
			fmt.Printf("OK: setTimeout fired (frame=%d) interval=%s\n", frame, iv.ToString())
			fired <- "ok"
			break
		}
	}
	select {
	case <-fired:
	default:
		fmt.Println("FAIL: setTimeout did not fire within 60 frames")
	}
}
