import { createApp, ref, reactive, computed, onMounted, h } from 'vue'

// 暴露 Vue 的 h 函数供诊断
window.__VUE_H__ = h

const App = {
  setup() {
    const count = ref(0)
    const message = ref('Hello from Vue 3!')
    const items = reactive(['Vue', 'Goja', 'WebKit', 'PairCode'])
    const doubleCount = computed(() => count.value * 2)

    onMounted(() => {
      console.log('Vue 3 app mounted successfully!')
      console.log('Initial count:', count.value)
    })

    function increment() {
      count.value++
      console.log('Count incremented to:', count.value)
    }

    function addItem() {
      items.push('Item ' + (items.length + 1))
    }

    return { count, message, items, doubleCount, increment, addItem }
  },
  render() {
    console.log('App.render() called')
    try {
      const vnode = h('div', { class: 'app-layout' }, [
        h('div', { class: 'activity-bar' }, [
          h('div', { class: 'activity-item active' }, '\uD83D\uDCC1'),
          h('div', { class: 'activity-item' }, '\uD83D\uDD0D'),
          h('div', { class: 'activity-item' }, '\u2699'),
        ]),
        h('div', { class: 'sidebar' }, [
          h('div', { class: 'sidebar-header' }, 'EXPLORER'),
          h('div', { class: 'sidebar-item' }, '\uD83D\uDCC4 index.html'),
          h('div', { class: 'sidebar-item' }, '\uD83D\uDCC4 main.js'),
          h('div', { class: 'sidebar-item' }, '\uD83D\uDCC4 App.vue'),
          h('div', { class: 'sidebar-divider' }),
          h('div', { class: 'sidebar-header' }, 'COMPUTED'),
          h('div', { class: 'sidebar-item' }, 'Double count: ' + this.doubleCount),
        ]),
        h('div', { class: 'main-content' }, [
          h('div', { class: 'tab-bar' }, [
            h('div', { class: 'tab active' }, 'main.js'),
            h('div', { class: 'tab' }, 'App.vue'),
          ]),
          h('div', { class: 'editor' }, [
            h('div', { class: 'editor-title' }, this.message),
            h('div', { class: 'editor-content' }, [
              h('h1', 'PairCode IDE'),
              h('p', ['Count: ', h('span', { class: 'status-badge' }, String(this.count))]),
              h('p', ['Double: ', h('span', { class: 'status-badge' }, String(this.doubleCount))]),
              h('button', { onClick: this.increment }, '+1'),
              h('button', { onClick: this.addItem }, '+ Item'),
              h('ul', this.items.map((item, idx) => h('li', { key: idx }, item))),
            ]),
          ]),
        ]),
        h('div', { class: 'status-bar' }, [
          h('div', { class: 'status-left' }, 'Vue 3 + Goja'),
          h('div', { class: 'status-right' }, 'ES6+ Supported'),
        ]),
      ])
      console.log('App.render() created vnode:', vnode ? vnode.type : 'null')
      return vnode
    } catch(e) {
      console.error('App.render() error:', e)
      return h('div', 'Render Error: ' + e.message)
    }
  }
}

console.log('Creating Vue app...')
const app = createApp(App)
console.log('Vue app created')

app.config.errorHandler = (err, vm, info) => {
  console.error('Vue Error:', err, info)
}

console.log('Mounting Vue app...')
const vm = app.mount('#app')
console.log('Vue app mounted, vm:', !!vm)

// 暴露 Vue 引用供外部诊断
window.__VUE_APP__ = app
window.__VUE_VM__ = vm
