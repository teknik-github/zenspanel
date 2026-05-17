import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('@/pages/Login.vue') },
    {
      path: '/',
      component: () => import('@/layouts/UserLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('@/pages/Dashboard.vue') },
        { path: 'domains', component: () => import('@/pages/Domains.vue') },
        { path: 'ssl-manager', component: () => import('@/pages/SSLManager.vue') },
        { path: 'php-settings', component: () => import('@/pages/PhpSettings.vue') },
        { path: 'databases', component: () => import('@/pages/Databases.vue') },
        { path: 'file-manager', component: () => import('@/pages/FileManager.vue') },
        { path: 'terminal', component: () => import('@/pages/Terminal.vue') },
        { path: 'backups', component: () => import('@/pages/Backups.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) return '/login'
})

export default router
