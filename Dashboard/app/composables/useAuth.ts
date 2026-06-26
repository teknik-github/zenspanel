import type { AuthUser } from '~/types/auth'

type LoginSuccess = {
  user: AuthUser
  requires_2fa?: false
}

type LoginRequires2FA = {
  requires_2fa: true
  temp_token: string
}

type AuthError = {
  data?: {
    error?: string
  }
}

function getAuthErrorMessage(error: unknown, fallback: string) {
  const authError = error as AuthError
  return authError.data?.error || fallback
}

const _useAuth = () => {
  const user = useState<AuthUser | null>('auth:user', () => null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const requires2FA = ref(false)
  const tempToken = ref<string | null>(null)
  const sessionExpired = ref(false)

  const isAuthenticated = computed(() => !!user.value)

  async function login(username: string, password: string) {
    loading.value = true
    error.value = null
    requires2FA.value = false
    tempToken.value = null

    try {
      const data = await $fetch<LoginSuccess | LoginRequires2FA>('/api/v1/auth/login', {
        method: 'POST',
        body: { username, password }
      })

      if (data.requires_2fa) {
        requires2FA.value = true
        tempToken.value = data.temp_token
        return { requires2FA: true, tempToken: data.temp_token }
      }

      user.value = data.user as AuthUser
      sessionExpired.value = false
      return { requires2FA: false }
    } catch (e: unknown) {
      error.value = getAuthErrorMessage(e, 'Invalid username or password')
      throw e
    } finally {
      loading.value = false
    }
  }

  async function verify2FA(code: string) {
    if (!tempToken.value) {
      error.value = 'No 2FA session found. Please log in again.'
      throw new Error('No 2FA session')
    }

    loading.value = true
    error.value = null

    try {
      const data = await $fetch<LoginSuccess>('/api/v1/auth/2fa/verify', {
        method: 'POST',
        body: { temp_token: tempToken.value, code }
      })

      user.value = data.user as AuthUser
      requires2FA.value = false
      tempToken.value = null
      sessionExpired.value = false
    } catch (e: unknown) {
      error.value = getAuthErrorMessage(e, 'Invalid verification code')
      throw e
    } finally {
      loading.value = false
    }
  }

  async function recover2FA(recoveryCode: string) {
    if (!tempToken.value) {
      error.value = 'No 2FA session found. Please log in again.'
      throw new Error('No 2FA session')
    }

    loading.value = true
    error.value = null

    try {
      const data = await $fetch<LoginSuccess>('/api/v1/auth/2fa/recover', {
        method: 'POST',
        body: { temp_token: tempToken.value, recovery_code: recoveryCode }
      })

      user.value = data.user as AuthUser
      requires2FA.value = false
      tempToken.value = null
      sessionExpired.value = false
    } catch (e: unknown) {
      error.value = getAuthErrorMessage(e, 'Invalid recovery code')
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    try {
      const headers = import.meta.server ? useRequestHeaders(['cookie']) : undefined

      user.value = await $fetch<AuthUser>('/api/v1/auth/me', { headers })
      sessionExpired.value = false
    } catch {
      user.value = null
    }
  }

  function clearSession() {
    user.value = null
    requires2FA.value = false
    tempToken.value = null
  }

  async function clearRemoteSession() {
    try {
      await $fetch('/api/v1/auth/logout', { method: 'POST' })
    } catch {
      // Local state is still cleared below; logout must not strand the user.
    }
  }

  function loginPath(role?: AuthUser['role']) {
    if (role === 'admin') {
      return '/admin/login'
    }

    const route = useRoute()
    return route.path.startsWith('/admin') ? '/admin/login' : '/login'
  }

  async function redirectToLogin(role?: AuthUser['role']) {
    const route = useRoute()
    const target = loginPath(role)

    if (route.path !== target) {
      await navigateTo(target)
    }
  }

  async function logout() {
    const role = user.value?.role

    sessionExpired.value = false
    await clearRemoteSession()
    clearSession()
    await redirectToLogin(role)
  }

  async function handleSessionExpired() {
    if (sessionExpired.value) {
      return
    }

    const role = user.value?.role

    sessionExpired.value = true
    await clearRemoteSession()
    clearSession()
    error.value = 'Session expired. Please sign in again.'
    await redirectToLogin(role)
  }

  function clearError() {
    error.value = null
  }

  return {
    user,
    loading,
    error,
    requires2FA,
    tempToken,
    isAuthenticated,
    sessionExpired,
    login,
    verify2FA,
    recover2FA,
    fetchMe,
    logout,
    handleSessionExpired,
    clearError
  }
}

export const useAuth = createSharedComposable(_useAuth)
