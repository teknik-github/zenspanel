import { defineStore } from 'pinia'
import { ref } from 'vue'
import { usageApi } from '@/api/usage'
import { useAuthStore } from './auth'

type Quota = { used: number; max: number }
type Usage = {
  domains: Quota
  databases: Quota
  disk: Quota
  ram: Quota
  cpu: Quota
}

const emptyUsage = (): Usage => ({
  domains: { used: 0, max: 0 },
  databases: { used: 0, max: 0 },
  disk: { used: 0, max: 0 },
  ram: { used: 0, max: 0 },
  cpu: { used: 0, max: 100 },
})

export const useUsageStore = defineStore('usage', () => {
  const usage = ref<Usage>(emptyUsage())

  async function fetch() {
    const auth = useAuthStore()
    if (!auth.user) return
    try {
      const res = await usageApi.get(auth.user.id)
      const incoming = res.data?.usage
      if (incoming && typeof incoming === 'object') {
        // Merge field by field so a missing key falls back to {used:0,max:0}
        // instead of leaving the slot undefined and crashing the Dashboard.
        const next = emptyUsage()
        for (const k of ['domains', 'databases', 'disk', 'ram', 'cpu'] as const) {
          const v = incoming[k]
          if (v && typeof v === 'object') {
            next[k] = { used: Number(v.used) || 0, max: Number(v.max) || 0 }
          }
        }
        usage.value = next
      }
    } catch {
      usage.value = emptyUsage()
    }
  }

  return { usage, fetch }
})
