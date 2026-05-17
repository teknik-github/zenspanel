<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { auditLogsApi } from '@/api/auditLogs'

const logs = ref<any[]>([])
const total = ref(0)
const filters = ref({ user_id: '', action: '', date_from: '', date_to: '', page: 1, limit: 50 })

async function fetchLogs() {
  const res = await auditLogsApi.list(filters.value)
  logs.value = res.data.data || []
  total.value = res.data.total || 0
}

onMounted(fetchLogs)
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">Audit Logs</h1>

    <!-- Filters -->
    <div class="flex gap-3 flex-wrap">
      <input v-model="filters.user_id" @input="fetchLogs" type="text" placeholder="User ID"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-28 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.action" @input="fetchLogs" type="text" placeholder="Action (e.g. domain.create)"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-48 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.date_from" @change="fetchLogs" type="date"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <input v-model="filters.date_to" @change="fetchLogs" type="date"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500" />
    </div>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
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
            <td class="px-4 py-2.5 text-gray-400">{{ new Date(log.created_at).toLocaleString() }}</td>
            <td class="px-4 py-2.5 text-gray-600">{{ log.user_id || '—' }}</td>
            <td class="px-4 py-2.5 font-mono text-indigo-600">{{ log.action }}</td>
            <td class="px-4 py-2.5 text-gray-500">{{ log.resource || '—' }}</td>
            <td class="px-4 py-2.5 text-gray-400 font-mono">{{ log.ip_address }}</td>
          </tr>
          <tr v-if="!logs.length">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No audit logs found.</td>
          </tr>
        </tbody>
      </table>
      <div class="px-4 py-3 border-t border-gray-100 text-xs text-gray-400">{{ total }} total entries</div>
    </div>
  </div>
</template>
