// Admin-only routes — users are redirected to /dashboard
const ADMIN_ROUTES = [
  '/users',
  '/packages',
  '/audit-logs',
  '/updates',
  '/ip-allowlist',
  '/firewall',
  '/php-versions',
  '/php-extensions',
  '/backup-targets',
  '/monitor',
  '/api-keys',
]

// User-only routes — admins are redirected to /dashboard
const USER_ROUTES = [
  '/file-manager',
  '/ftp',
  '/php-settings',
  '/antivirus',
  '/two-factor',
  '/redirects',
  '/cron-jobs',
  '/logs',
]

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path === '/admin/login' || to.path === '/login') return

  const auth = useAuth()

  if (!auth.user.value) {
    await auth.fetchMe()
  }

  if (!auth.isAuthenticated.value) {
    if (to.path.startsWith('/admin')) {
      return navigateTo('/admin/login')
    }
    return navigateTo('/login')
  }

  const role = auth.user.value?.role
  const path = to.path

  // Block client from accessing admin-only routes
  if (role !== 'admin') {
    const isAdminRoute = ADMIN_ROUTES.some(r => path === r || path.startsWith(r + '/'))
    if (isAdminRoute) {
      return navigateTo('/dashboard')
    }
  }

  // Block admin from accessing user-only routes
  if (role === 'admin') {
    const isUserRoute = USER_ROUTES.some(r => path === r || path.startsWith(r + '/'))
    if (isUserRoute) {
      return navigateTo('/dashboard')
    }
  }
})
