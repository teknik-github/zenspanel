import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    { path: '/login', component: () => import('@/pages/Login.vue') },
    {
      path: '/',
      component: () => import('@/layouts/AdminLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('@/pages/Dashboard.vue') },
        { path: 'resource-monitor', component: () => import('@/pages/ResourceMonitor.vue') },
        { path: 'users', component: () => import('@/pages/users/UserList.vue') },
        { path: 'users/:id', component: () => import('@/pages/users/UserDetail.vue') },
        { path: 'packages', component: () => import('@/pages/Packages.vue') },
        { path: 'domains', component: () => import('@/pages/Domains.vue') },
        { path: 'databases', component: () => import('@/pages/Databases.vue') },
        { path: 'php-versions', component: () => import('@/pages/PhpVersions.vue') },
        { path: 'ssl-manager', component: () => import('@/pages/SSLManager.vue') },
        { path: 'backups', component: () => import('@/pages/Backups.vue') },
        { path: 'api-keys', component: () => import('@/pages/ApiKeys.vue') },
        { path: 'audit-logs', component: () => import('@/pages/AuditLogs.vue') },
        { path: 'settings', component: () => import('@/pages/Settings.vue') },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.token) return '/login'
})

export default router
