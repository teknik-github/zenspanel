<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'

const domains = ref<any[]>([])
const filterMode = ref<'all' | 'expiring' | 'none'>('all')
const error = ref('')

async function load() {
  try {
    const res = await domainsApi.list()
    domains.value = res.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  }
}

function daysUntil(s: string | null | undefined): number | null {
  if (!s) return null
  const exp = new Date(s).getTime()
  if (isNaN(exp)) return null
  return Math.floor((exp - Date.now()) / 86400000)
}

const rows = computed(() => {
  return domains.value
    .map(d => {
      const expiry = d.ssl_expires_at?.Time || d.ssl_expires_at
      return { ...d, _days: daysUntil(typeof expiry === 'string' ? expiry : null) }
    })
    .filter(d => {
      if (filterMode.value === 'all') return true
      if (filterMode.value === 'none') return d.ssl_type === 'none'
      if (filterMode.value === 'expiring') return d._days !== null && d._days <= 30
      return true
    })
})

const expiringCount = computed(() =>
  domains.value.filter(d => {
    const days = daysUntil(typeof d.ssl_expires_at === 'string' ? d.ssl_expires_at : d.ssl_expires_at?.Time)
    return days !== null && days <= 30
  }).length,
)

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">SSL Manager</h1>
      <div v-if="expiringCount" class="text-xs bg-amber-50 border border-amber-200 text-amber-700 px-3 py-1 rounded-md">
        {{ expiringCount }} certificate{{ expiringCount > 1 ? 's' : '' }} expiring within 30 days
      </div>
    </div>

    <div class="flex gap-2 text-xs">
      <button @click="filterMode = 'all'"
        :class="filterMode === 'all' ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-600'"
        class="px-3 py-1.5 rounded-md">All ({{ domains.length }})</button>
      <button @click="filterMode = 'expiring'"
        :class="filterMode === 'expiring' ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-600'"
        class="px-3 py-1.5 rounded-md">Expiring soon ({{ expiringCount }})</button>
      <button @click="filterMode = 'none'"
        :class="filterMode === 'none' ? 'bg-indigo-600 text-white' : 'bg-gray-100 text-gray-600'"
        class="px-3 py-1.5 rounded-md">No SSL</button>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Domain</th>
            <th class="text-left px-4 py-3 font-medium">User</th>
            <th class="text-left px-4 py-3 font-medium">SSL Type</th>
            <th class="text-left px-4 py-3 font-medium">Expires</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in rows" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
            <td class="px-4 py-3 text-gray-500">{{ d.user_id }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="d.ssl_type === 'none' ? 'bg-gray-100 text-gray-500' : 'bg-green-100 text-green-700'">
                {{ d.ssl_type }}
              </span>
            </td>
            <td class="px-4 py-3"
              :class="d._days !== null && d._days <= 30 ? 'text-amber-600 font-medium' : 'text-gray-500'">
              <template v-if="d._days === null">—</template>
              <template v-else-if="d._days < 0">expired</template>
              <template v-else>in {{ d._days }} days</template>
            </td>
          </tr>
          <tr v-if="!rows.length">
            <td colspan="4" class="px-4 py-8 text-center text-gray-400">No domains match filter</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
