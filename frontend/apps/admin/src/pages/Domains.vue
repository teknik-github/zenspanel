<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { usersApi } from '@/api/users'

const domains = ref<any[]>([])
const users = ref<Map<number, string>>(new Map())
const search = ref('')
const statusFilter = ref('')
const loaded = ref(false)
const error = ref('')

async function load() {
  try {
    const [dRes, uRes] = await Promise.all([domainsApi.list(), usersApi.list()])
    domains.value = dRes.data.data || []
    const uMap = new Map<number, string>()
    for (const u of (uRes.data.data || [])) uMap.set(u.id, u.username)
    users.value = uMap
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  } finally {
    loaded.value = true
  }
}

const filtered = computed(() => {
  return domains.value.filter(d => {
    if (statusFilter.value && d.status !== statusFilter.value) return false
    if (search.value && !d.domain.toLowerCase().includes(search.value.toLowerCase())) return false
    return true
  })
})

function sslClass(t: string) {
  if (t === 'none') return 'bg-gray-100 text-gray-500'
  if (t === 'letsencrypt') return 'bg-green-100 text-green-700'
  return 'bg-blue-100 text-blue-700'
}

function statusClass(s: string) {
  if (s === 'active') return 'bg-green-100 text-green-700'
  if (s === 'suspended') return 'bg-red-100 text-red-700'
  return 'bg-amber-100 text-amber-700'
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Domains</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">All domains across all users</p>
      </div>
      <span v-if="loaded" class="text-xs text-gray-400">{{ filtered.length }} of {{ domains.length }}</span>
    </div>

    <div class="flex gap-2 flex-wrap">
      <div class="relative">
        <svg class="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input v-model="search" type="text" placeholder="Search domain..."
          class="border border-gray-200 rounded-md pl-8 pr-3 py-2 text-xs w-56 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      </div>
      <select v-model="statusFilter"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Status</option>
        <option value="active">Active</option>
        <option value="pending">Pending</option>
        <option value="suspended">Suspended</option>
      </select>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-3 py-2">{{ error }}</p>

    <!-- Skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 5" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!domains.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No domains yet</p>
      <p class="text-xs text-gray-400 mt-1">Domains will appear here once users add them</p>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[640px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Domain</th>
              <th class="text-left px-4 py-3 font-medium">User</th>
              <th class="text-left px-4 py-3 font-medium">PHP</th>
              <th class="text-left px-4 py-3 font-medium">SSL</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!filtered.length">
              <td colspan="5" class="px-4 py-8 text-center text-gray-400">No domains match your filter</td>
            </tr>
            <tr v-for="d in filtered" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
              <td class="px-4 py-3 text-gray-500">
                <router-link :to="`/users/${d.user_id}`" class="hover:text-indigo-600 transition-colors">
                  {{ users.get(d.user_id) || `#${d.user_id}` }}
                </router-link>
              </td>
              <td class="px-4 py-3 text-gray-500">{{ d.php_version }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="sslClass(d.ssl_type)">
                  {{ d.ssl_type === 'none' ? 'No SSL' : d.ssl_type }}
                </span>
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(d.status)">
                  {{ d.status }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
