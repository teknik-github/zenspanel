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
  } | null>(null)

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
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

  return { token, user, login, fetchMe, logout }
})
