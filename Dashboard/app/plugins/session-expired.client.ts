function isLoginRequest(request: RequestInfo | URL) {
  const url = String(request)

  return url.includes('/api/v1/auth/login')
    || url.includes('/api/v1/auth/2fa/verify')
    || url.includes('/api/v1/auth/2fa/recover')
}

export default defineNuxtPlugin(() => {
  const authFetch = $fetch.create({
    async onResponseError({ request, response }) {
      if (response.status !== 401 || isLoginRequest(request)) {
        return
      }

      const auth = useAuth()
      await auth.handleSessionExpired()
    }
  })

  globalThis.$fetch = authFetch
})
