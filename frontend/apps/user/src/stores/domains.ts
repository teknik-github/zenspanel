import { defineStore } from 'pinia'
import { ref } from 'vue'
import { domainsApi } from '@/api/domains'

export const useDomainsStore = defineStore('domains', () => {
  const domains = ref<any[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    try {
      const res = await domainsApi.list()
      domains.value = res.data.data || []
    } finally {
      loading.value = false
    }
  }

  return { domains, loading, fetch }
})
