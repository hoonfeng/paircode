import { createApp, h, reactive, ref, render } from 'vue'
import App from './App.vue'

console.log('A')
var app = createApp(App)
console.log('B')

// Test mount first
var root = app.mount('#app')
console.log('D root=' + (typeof root))

var el = document.getElementById('app')
console.log('E children=' + el.childNodes.length + ' html=[' + el.innerHTML.substring(0,50) + ']')
console.log('F __vue_app__=' + (el.__vue_app__ ? 'yes' : 'no'))
