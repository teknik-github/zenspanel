import { defineStore } from 'pinia'
import { ref } from 'vue'
import { packagesApi } from '@/api/packages'

export const usePackagesStore = defineStore('packages', () => {
  const packages = ref<any[]>([])

  async function fetch() {
    const res = await packagesApi.list()
    packages.value = res.data.data || []
  }

  return { packages, fetch }
})
