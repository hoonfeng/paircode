// TestCordisApiGlobal：内置 cordis 运行时（CordisApi 全局）端到端验证。
// 插件代码可直接 new CordisApi.api.Context() 建真 cordis app，跑 cordis 生态
// 多插件协作（app.plugin 挂子插件 → app.start 触发 ready → 子插件 ctx.provide 服务）。
package agent

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestCordisApiGlobal(t *testing.T) {
	dir := t.TempDir()
	reg := NewRegistry()
	host := NewPluginHost(reg, nil, dir)
	RegisterCordisTools(reg, host, dir)
	ctx := context.Background()

	// 插件代码：顶层用内置真 cordis 建 app，挂 cordis 生态子插件并 start。
	// evalJSPlugin 经 RunString 求值（goja 返回前 drain 微任务队列），
	// 顶层 await app.start() 让 ready 生命周期完整跑完 → 子插件服务就绪。
	// ★ cordis 3.18 服务 API：set(name, value) 可 get；provide 已 @deprecated（调用不报错但 get 不到）。
	pluginSrc := `
const app = new CordisApi.api.Context()
let readyFlag = false
app.on('ready', () => { readyFlag = true })
function subPlugin(subctx) {
  subctx.set('probeInner', 'hello-from-cordis-sub')
}
app.plugin(subPlugin)
await app.start()
return {
  name: 'cordis-api-probe',
  apply(pluginCtx) {
    const inner = app.get('probeInner')
    pluginCtx.provide('probe', {
      ready: readyFlag,
      inner: inner ?? null,
      hasInner: typeof inner !== 'undefined',
      appExists: typeof CordisApi !== 'undefined'
    })
  }
}`

	defOut, err := reg.Execute(ctx, "cordis_define", `{"code":`+strconv.Quote(pluginSrc)+`,"purpose":"CordisApi 全局验证"}`)
	if err != nil {
		t.Fatalf("cordis_define: %v", err)
	}
	id := regexp.MustCompile(`dyn-\d+`).FindString(defOut)
	if id == "" {
		t.Fatalf("无法提取 dyn id: %s", defOut)
	}

	if _, err := reg.Execute(ctx, "cordis_run", `{"id":"`+id+`"}`); err != nil {
		t.Fatalf("cordis_run: %v", err)
	}
	if host.State("cordis-api-probe") != PluginRunning {
		t.Fatalf("cordis-api-probe 应 running")
	}

	// 验证宿主服务：子插件服务可见 + ready 事件已触发 + CordisApi 全局存在
	v := host.Context().Get("probe")
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("probe 服务类型 = %T, want map", v)
	}
	if m["ready"] != true {
		t.Errorf("cordis app ready 事件应已触发，实际 %v", m["ready"])
	}
	if m["inner"] != "hello-from-cordis-sub" {
		t.Errorf("cordis 子插件 provide 服务应可见，实际 %v", m["inner"])
	}
	if m["hasInner"] != true {
		t.Errorf("app.get('probeInner') 应命中子插件服务，实际 %v", m["hasInner"])
	}
	if m["appExists"] != true {
		t.Errorf("CordisApi 全局应存在，实际 %v", m["appExists"])
	}

	// cordis_stop 回收
	if _, err := reg.Execute(ctx, "cordis_stop", `{"id":"`+id+`"}`); err != nil {
		t.Fatalf("cordis_stop: %v", err)
	}
	if host.State("cordis-api-probe") != PluginStopped {
		t.Fatalf("cordis-api-probe 应 stopped")
	}
	if v := host.Context().Get("probe"); v != nil {
		t.Errorf("cordis_stop 后 probe 服务应移除，实际 %v", v)
	}
	if !strings.Contains(defOut, "cordis-api-probe") && !strings.Contains(defOut, "dyn-") {
		t.Errorf("define 输出异常: %s", defOut)
	}
}
