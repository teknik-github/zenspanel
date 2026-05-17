import { defineStore } from 'pinia'
import { ref } from 'vue'
import { databasesApi } from '@/api/databases'

export const useDatabasesStore = defineStore('databases', () => {
  const databases = ref<any[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    try {
      const res = await databasesApi.list()
      databases.value = res.data.data || []
    } finally {
      loading.value = false
    }
  }

  return { databases, loading, fetch }
})
