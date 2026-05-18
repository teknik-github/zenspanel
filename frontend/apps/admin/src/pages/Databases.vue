<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { databasesApi } from '@/api/databases'

const databases = ref<any[]>([])
const search = ref('')
const loading = ref(false)
const error = ref('')

async function load() {
  loading.value = true
  try {
    const res = await databasesApi.list()
    databases.value = res.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  } finally {
    loading.value = false
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
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Databases</h1>
      <span class="text-xs text-gray-400">{{ filtered.length }} of {{ databases.length }} databases</span>
    </div>

    <input v-model="search" type="text" placeholder="Search database name or user..."
      class="border border-gray-200 rounded-md px-3 py-2 text-xs w-80 focus:outline-none focus:ring-2 focus:ring-indigo-500" />

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Database</th>
            <th class="text-left px-4 py-3 font-medium">User</th>
            <th class="text-left px-4 py-3 font-medium">Owner ID</th>
            <th class="text-left px-4 py-3 font-medium">Created</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in filtered" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ d.db_name }}</td>
            <td class="px-4 py-3 text-gray-500">{{ d.db_user }}</td>
            <td class="px-4 py-3 text-gray-500">{{ d.user_id }}</td>
            <td class="px-4 py-3 text-gray-500">{{ fmtDate(d.created_at) }}</td>
          </tr>
          <tr v-if="!filtered.length && !loading">
            <td colspan="4" class="px-4 py-8 text-center text-gray-400">No databases found</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
