import { createApp, ref, h } from 'vue'
import App from './App.vue'

window.__VUE_STEPS__ = []

function logStep(name, ok) {
  window.__VUE_STEPS__.push(name + '=' + (ok ? 'OK' : 'FAIL'))
  console.log('VSTEP_' + name + ': ' + (ok ? 'OK' : 'FAIL'))
}

window.__MOUNT_START__ = 1
window.__BUILD_ID__ = 'build003'

var app
try {
  app = createApp(App)
  window.__APP_CREATED__ = 1
  logStep('createApp', true)
} catch(e) {
  logStep('createApp', false)
  window.__APP_ERR__ = '' + e
}

var container
try {
  container = document.querySelector('#app')
  logStep('querySelector', container !== null)
} catch(e) {
  logStep('querySelector', false)
  window.__APP_ERR__ = 'qs:' + e
}

if (container) {
  var _setupFailed = false
  try { container.innerHTML = '' } catch(e) { window.__STEP_ERR_1__=''+e; _setupFailed = true }
  if(!_setupFailed) { try { container.__vue_app__ = app } catch(e) { window.__STEP_ERR_2__=''+e; _setupFailed = true } }
  if(!_setupFailed) { try { app._container = container } catch(e) { window.__STEP_ERR_3__=''+e; _setupFailed = true } }
  logStep('setupContainer', !_setupFailed)
  if(_setupFailed) { throw new Error('setupContainer failed') }
} else {
  logStep('setupContainer', false)
  window.__APP_ERR__ = 'container_null'
}

try {
  var testVNode = h('div', {id: 'test'}, 'hello')
  logStep('h_simple', testVNode !== null)
} catch(e) {
  logStep('h_simple', false)
  window.__APP_ERR__ = 'h_simple:' + e
}

try {
  window.__MOUNT_STEP__ = 'app_mount'
  var result = app.mount('#app')
  window.__MOUNT_DONE__ = 1
  logStep('mount', result !== null)
} catch(e) {
  logStep('mount', false)
  window.__APP_ERR__ = 'mount:' + (e && e.message ? e.message : e)
  window.__APP_ERR_STACK__ = (e && e.stack ? e.stack : 'none')
  window.__MOUNT_STEP_FAIL__ = window.__MOUNT_STEP__
}
