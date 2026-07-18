import { createApp, ref, h } from 'vue'
import App from './App.vue'

window.__VUE_STEPS__ = []

function logStep(name, ok) {
  window.__VUE_STEPS__.push(name + '=' + (ok ? 'OK' : 'FAIL'))
  console.log('VSTEP_' + name + ': ' + (ok ? 'OK' : 'FAIL'))
}

window.__MOUNT_START__ = 1

// Step 1: createApp
var app
try {
  app = createApp(App)
  window.__APP_CREATED__ = 1
  logStep('createApp', true)
} catch(e) {
  logStep('createApp', false)
  window.__APP_ERR__ = '' + e
}

// Step 2: get container
var container
try {
  container = document.querySelector('#app')
  logStep('querySelector', container !== null)
} catch(e) {
  logStep('querySelector', false)
  window.__APP_ERR__ = 'qs:' + e
}

// Step 3: mount (try-catch around the whole thing)
try {
  // Vue 3 mount internally does:
  // 1. container = querySelector
  // 2. container.__vue_app__ = app
  // 3. app._container = container
  // 4. Render root component
  // 5. patch(container._vnode, vnode)
  
  // Let's try setup the container manually first
  container.__vue_app__ = app
  logStep('set_vue_app', true)
  
  app._container = container
  logStep('set_container', true)
  
  // Now call mount - this should work
  var result = app.mount('#app')
  window.__MOUNT_DONE__ = 1
  logStep('mount_call', result !== null && result !== undefined)
} catch(e) {
  logStep('mount_call', false)
  window.__APP_ERR__ = 'mount:' + (e && e.message ? e.message : e)
  window.__APP_ERR_STACK__ = (e && e.stack ? e.stack : 'no stack')
}
