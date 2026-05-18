<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'

const domains = ref<any[]>([])
const search = ref('')
const statusFilter = ref('')
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  try {
    const res = await domainsApi.list()
    domains.value = res.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  return domains.value.filter(d => {
    if (statusFilter.value && d.status !== statusFilter.value) return false
    if (search.value && !d.domain.toLowerCase().includes(search.value.toLowerCase())) return false
    return true
  })
})

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Domains</h1>
      <span class="text-xs text-gray-400">{{ filtered.length }} of {{ domains.length }} domains</span>
    </div>

    <div class="flex gap-3">
      <input v-model="search" type="text" placeholder="Search domain..."
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-64 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <select v-model="statusFilter"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Status</option>
        <option value="active">Active</option>
        <option value="pending">Pending</option>
        <option value="suspended">Suspended</option>
      </select>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Domain</th>
            <th class="text-left px-4 py-3 font-medium">User ID</th>
            <th class="text-left px-4 py-3 font-medium">PHP</th>
            <th class="text-left px-4 py-3 font-medium">SSL</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in filtered" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
            <td class="px-4 py-3 text-gray-500">{{ d.user_id }}</td>
            <td class="px-4 py-3 text-gray-500">{{ d.php_version }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="d.ssl_type === 'none' ? 'bg-gray-100 text-gray-500' : 'bg-green-100 text-green-700'">
                {{ d.ssl_type }}
              </span>
            </td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="d.status === 'active' ? 'bg-green-100 text-green-700' : d.status === 'pending' ? 'bg-amber-100 text-amber-700' : 'bg-yellow-100 text-yellow-700'">
                {{ d.status }}
              </span>
            </td>
          </tr>
          <tr v-if="!filtered.length && !loading">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No domains found</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
