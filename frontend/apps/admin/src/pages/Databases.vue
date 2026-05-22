<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { databasesApi } from '@/api/databases'
import { usersApi } from '@/api/users'

const databases = ref<any[]>([])
const users = ref<Map<number, string>>(new Map())
const search = ref('')
const loaded = ref(false)
const error = ref('')

async function load() {
  try {
    const [dRes, uRes] = await Promise.all([databasesApi.list(), usersApi.list()])
    databases.value = dRes.data.data || []
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
  if (!search.value) return databases.value
  const q = search.value.toLowerCase()
  return databases.value.filter(d =>
    d.db_name.toLowerCase().includes(q) || d.db_user.toLowerCase().includes(q),
  )
})

function fmtDate(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleDateString()
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Databases</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">All MySQL databases across all users</p>
      </div>
      <span v-if="loaded" class="text-xs text-gray-400">{{ filtered.length }} of {{ databases.length }}</span>
    </div>

    <div class="relative w-72">
      <svg class="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
      </svg>
      <input v-model="search" type="text" placeholder="Search database or user..."
        class="w-full border border-gray-200 rounded-md pl-8 pr-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500" />
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-3 py-2">{{ error }}</p>

    <!-- Skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 4" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!databases.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No databases yet</p>
      <p class="text-xs text-gray-400 mt-1">Databases will appear here once users create them</p>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[560px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Database</th>
              <th class="text-left px-4 py-3 font-medium">DB User</th>
              <th class="text-left px-4 py-3 font-medium">Owner</th>
              <th class="text-left px-4 py-3 font-medium">Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!filtered.length">
              <td colspan="4" class="px-4 py-8 text-center text-gray-400">No databases match your search</td>
            </tr>
            <tr v-for="d in filtered" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700 font-mono">{{ d.db_name }}</td>
              <td class="px-4 py-3 text-gray-500 font-mono">{{ d.db_user }}</td>
              <td class="px-4 py-3 text-gray-500">
                <router-link :to="`/users/${d.user_id}`" class="hover:text-indigo-600 transition-colors">
                  {{ users.get(d.user_id) || `#${d.user_id}` }}
                </router-link>
              </td>
              <td class="px-4 py-3 text-gray-400">{{ fmtDate(d.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
