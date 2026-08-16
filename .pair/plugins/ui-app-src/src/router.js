import { createRouter, createWebHashHistory } from 'vue-router'

// 注意：App.vue 直接使用 <RightPanel /> 渲染消息，未使用 <router-view>；
// 此处路由保留以兼容可能的路由跳转，ChatView.vue 已废弃删除，改用 RightPanel。
const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('./components/RightPanel.vue'),
  },
  {
    path: '/workspace/:wsId',
    name: 'workspace',
    component: () => import('./components/RightPanel.vue'),
    props: true,
  },
  {
    path: '/workspace/:wsId/chat/:convId',
    name: 'conversation',
    component: () => import('./components/RightPanel.vue'),
    props: true,
  },
  {
    path: '/workspace/:wsId/plan/:planId',
    name: 'plan',
    component: () => import('./components/PlanView.vue'),
    props: true,
  },
  {
    path: '/tasks',
    name: 'tasks',
    component: () => import('./components/TaskBoard.vue'),
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
