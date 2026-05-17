import { defineStore } from 'pinia'
import { ref } from 'vue'
import { usageApi } from '@/api/usage'
import { useAuthStore } from './auth'

export const useUsageStore = defineStore('usage', () => {
  const usage = ref({
    domains: { used: 0, max: 0 },
    databases: { used: 0, max: 0 },
    disk: { used: 0, max: 0 },
    ram: { used: 0, max: 0 },
  })

  async function fetch() {
    const auth = useAuthStore()
    if (!auth.user) return
    const res = await usageApi.get(auth.user.id)
    usage.value = res.data.usage || usage.value
  }

  return { usage, fetch }
})
