// Admin page aliases use /admin/<page>. These root paths are kept only for
// legacy redirects when an authenticated admin opens an old URL.
const ADMIN_ALIAS_ROOTS = [
  '/dashboard',
  '/users',
  '/packages',
  '/domains',
  '/databases',
  '/ssl',
  '/backups',
  '/terminal',
  '/settings',
  '/audit-logs',
  '/updates',
  '/ip-allowlist',
  '/firewall',
  '/php-versions',
  '/php-extensions',
  '/backup-targets',
  '/monitor',
  '/api-keys'
]

// Admin-only root routes — users are redirected to /dashboard
const ADMIN_ONLY_ROOTS = [
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
  '/api-keys'
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
  '/logs'
]

function matchesPath(path: string, roots: string[]) {
  return roots.some(root => path === root || path.startsWith(root + '/'))
}

function legacyAdminTarget(path: string) {
  return matchesPath(path, ADMIN_ALIAS_ROOTS) ? `/admin${path}` : null
}

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path === '/admin/login' || to.path === '/login') return

  const auth = useAuth()
  const path = to.path

  if (!auth.user.value) {
    await auth.fetchMe()
  }

  if (!auth.isAuthenticated.value) {
    if (path.startsWith('/admin') || matchesPath(path, ADMIN_ONLY_ROOTS)) {
      return navigateTo('/admin/login')
    }
    return navigateTo('/login')
  }

  const role = auth.user.value?.role

  if (role === 'admin' && path === '/admin') {
    return navigateTo('/admin/dashboard')
  }

  if (role === 'admin') {
    const target = legacyAdminTarget(path)

    if (target) {
      return navigateTo(target)
    }
  }

  // Block client from accessing admin-only routes
  if (role !== 'admin') {
    if (path.startsWith('/admin') || matchesPath(path, ADMIN_ONLY_ROOTS)) {
      return navigateTo('/dashboard')
    }
  }

  // Block admin from accessing user-only routes
  if (role === 'admin') {
    if (matchesPath(path, USER_ROUTES)) {
      return navigateTo('/admin/dashboard')
    }
  }
})
