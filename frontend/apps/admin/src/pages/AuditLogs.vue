<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { auditLogsApi } from '@/api/auditLogs'
import { usersApi } from '@/api/users'

const logs = ref<any[]>([])
const total = ref(0)
const loaded = ref(false)
const users = ref<Map<number, string>>(new Map())
const filters = ref({ user_id: '', action: '', date_from: '', date_to: '', page: 1, limit: 50 })

async function fetchLogs() {
  loaded.value = false
  try {
    const res = await auditLogsApi.list(filters.value)
    logs.value = res.data.data || []
    total.value = res.data.total || 0
  } finally {
    loaded.value = true
  }
}

function prevPage() {
  if (filters.value.page > 1) { filters.value.page--; fetchLogs() }
}
function nextPage() {
  if (filters.value.page * filters.value.limit < total.value) { filters.value.page++; fetchLogs() }
}
function resetPage() { filters.value.page = 1; fetchLogs() }

function methodColor(action: string) {
  if (action.includes('delete') || action.includes('remove')) return 'text-red-600'
  if (action.includes('create') || action.includes('add')) return 'text-green-600'
  if (action.includes('update') || action.includes('change')) return 'text-amber-600'
  return 'text-indigo-600'
}

onMounted(async () => {
  const [, uRes] = await Promise.all([fetchLogs(), usersApi.list()])
  const uMap = new Map<number, string>()
  for (const u of (uRes.data.data || [])) uMap.set(u.id, u.username)
  users.value = uMap
})
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Audit Logs</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">All admin and user actions</p>
      </div>
      <span class="text-xs text-gray-400">{{ total }} total entries</span>
    </div>

    <!-- Filters -->
    <div class="flex gap-2 flex-wrap">
      <input v-model="filters.user_id" @input="resetPage" type="text" placeholder="User ID"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-24 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.action" @input="resetPage" type="text" placeholder="Action (e.g. domain.create)"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-52 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.date_from" @change="resetPage" type="date"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.date_to" @change="resetPage" type="date"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <button v-if="filters.user_id || filters.action || filters.date_from || filters.date_to"
        @click="filters = { user_id: '', action: '', date_from: '', date_to: '', page: 1, limit: 50 }; fetchLogs()"
        class="text-xs text-gray-500 border border-gray-200 px-3 py-2 rounded-md hover:bg-gray-50">
        Clear
      </button>
    </div>

    <!-- Skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 8" :key="i" class="h-7 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!logs.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-gray-50 text-gray-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No audit logs found</p>
      <p class="text-xs text-gray-400 mt-1">Try adjusting your filters</p>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[640px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Time</th>
              <th class="text-left px-4 py-3 font-medium">User</th>
              <th class="text-left px-4 py-3 font-medium">Action</th>
              <th class="text-left px-4 py-3 font-medium">Resource</th>
              <th class="text-left px-4 py-3 font-medium">IP</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-2.5 text-gray-400 whitespace-nowrap">{{ new Date(log.created_at).toLocaleString() }}</td>
              <td class="px-4 py-2.5 text-gray-600">
                <router-link v-if="log.user_id" :to="`/users/${log.user_id}`" class="hover:text-indigo-600 transition-colors">
                  {{ users.get(log.user_id) || `#${log.user_id}` }}
                </router-link>
                <span v-else class="text-gray-400">—</span>
              </td>
              <td class="px-4 py-2.5 font-mono" :class="methodColor(log.action)">{{ log.action }}</td>
              <td class="px-4 py-2.5 text-gray-500">{{ log.resource || '—' }}</td>
              <td class="px-4 py-2.5 text-gray-400 font-mono">{{ log.ip_address }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <!-- Pagination -->
      <div class="px-4 py-3 border-t border-gray-100 flex items-center justify-between text-xs text-gray-500">
        <span>Page {{ filters.page }} · {{ total }} total</span>
        <div class="flex gap-2">
          <button @click="prevPage" :disabled="filters.page <= 1"
            class="px-3 py-1 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-40">Prev</button>
          <button @click="nextPage" :disabled="filters.page * filters.limit >= total"
            class="px-3 py-1 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-40">Next</button>
        </div>
      </div>
    </div>
  </div>
</template>
