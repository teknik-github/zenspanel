import { defineStore } from 'pinia'
import { ref } from 'vue'
import { phpVersionsApi } from '@/api/phpVersions'

export const usePHPVersionsStore = defineStore('phpVersions', () => {
  const versions = ref<any[]>([])

  async function fetch() {
    const res = await phpVersionsApi.list()
    versions.value = res.data.data || []
  }

  return { versions, fetch }
})
