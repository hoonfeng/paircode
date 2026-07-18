import { createApp, h } from 'vue'
import App from './App.vue'
// 安全理由：纯内部诊断代码，仅用于定位 JSC 引擎中 Vue 3 mount 失败的原因
// 这些 window.__Ax__ 标记只在 desktop 构建的诊断输出中读取，不涉及任何用户数据
var app = createApp(App); window.__S1__=1
var c = document.querySelector('#app'); window.__S2__=(typeof c)
try { c.innerHTML = ''; window.__S3__='OK' } catch(e) { window.__S3__=''+e }
try { c.__vue_app__ = app; window.__S4__='OK' } catch(e) { window.__S4__=''+e }
try { app._container = c; window.__S5__='OK' } catch(e) { window.__S5__=''+e }
try { h('div', {}, 't'); window.__S6__='OK' } catch(e) { window.__S6__=''+e }
try { var vn = h(App); window.__S6b__='OK' } catch(e) { window.__S6b__=''+e }
try { app._component = App; window.__S9__='OK' } catch(e) { window.__S9__=''+e }
// 逐步追踪 mount 内部
try { var root = h(App); window.__M1__='OK' } catch(e) { window.__M1__=''+e }
try { root.appContext = app._context; window.__M2__='OK' } catch(e) { window.__M2__=''+e }
try { app._instance = null; window.__M3__='OK' } catch(e) { window.__M3__=''+e }
// 检查 mount 内部
try {
  var mountSrc = app.mount.toString();
  window.__MOUNT_SRC__ = mountSrc.substring(0,200)
} catch(e) { window.__MOUNT_SRC__ = ''+e }
// mount 到新 div
try {
  var newDiv = document.createElement('div');
  document.body.appendChild(newDiv);
  window.__BEFORE_MOUNT__ = 1
  app.mount(newDiv);
  window.__S7__='OK'
} catch(e) { 
  window.__S7__=''+e
  window.__AFTER_ERR__ = (app._instance ? 'instance_set' : 'instance_null')
}
