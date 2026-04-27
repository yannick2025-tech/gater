import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'ProtocolTest',
    component: () => import('@/views/ProtocolTest.vue'),
  },
  {
    // 会话详情页（支持直接通过URL访问，如 /session/C1FF6550-9610-45）
    path: '/session/:sessionId',
    name: 'SessionDetail',
    component: () => import('@/views/ProtocolTest.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFound.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
