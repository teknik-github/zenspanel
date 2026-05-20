import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<{
    id: number
    username: string
    email: string
    role: string
    terminal_enabled: boolean
    backup_enabled: boolean
    package_id: number | null
    php_version: string
    totp_enabled: boolean
  } | null>(null)

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    if (res.data.requires_2fa) {
      // Return the temp token so the login page can show the TOTP step.
      return { requires_2fa: true, temp_token: res.data.temp_token }
    }
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', res.data.token)
    return { requires_2fa: false }
  }

  async function verifyTOTP(tempToken: string, code: string) {
    const res = await authApi.twofa.verify(tempToken, code)
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', res.data.token)
  }

  async function recoverTOTP(tempToken: string, recoveryCode: string) {
    const res = await authApi.twofa.recover(tempToken, recoveryCode)
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', res.data.token)
  }

  async function fetchMe() {
    const res = await authApi.me()
    user.value = res.data
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, login, verifyTOTP, recoverTOTP, fetchMe, logout }
})
