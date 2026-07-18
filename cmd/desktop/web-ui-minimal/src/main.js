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
try { app.mount('#app'); window.__S7__='OK' } catch(e) { window.__S7__=''+e }
