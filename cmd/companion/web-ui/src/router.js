import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('./components/ChatView.vue'),
  },
  {
    path: '/workspace/:wsId',
    name: 'workspace',
    component: () => import('./components/ChatView.vue'),
    props: true,
  },
  {
    path: '/workspace/:wsId/chat/:convId',
    name: 'conversation',
    component: () => import('./components/ChatView.vue'),
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
