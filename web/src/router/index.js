import { createRouter, createWebHistory } from 'vue-router'
import Cookies from 'js-cookie'

const routes = [
  { path: '/login', component: () => import('@/views/auth/Login.vue') },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', component: () => import('@/views/dashboard/Index.vue') },
      { path: 'device', component: () => import('@/views/device/List.vue') },
      { path: 'terminal/:deviceId', component: () => import('@/views/device/Terminal.vue') },
      { path: 'files/:deviceId', component: () => import('@/views/device/Files.vue') },
      { path: 'config/:deviceId', component: () => import('@/views/device/Config.vue') },
      { path: 'task/:deviceId', component: () => import('@/views/device/Task.vue') }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = Cookies.get('token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router