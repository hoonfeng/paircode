import { createApp, ref } from 'vue'
import App from './App.vue'

window.__MOUNT_START__ = 1
const app = createApp(App)
window.__APP_CREATED__ = 1
app.mount('#app')
window.__MOUNT_DONE__ = 1
