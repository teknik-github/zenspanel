<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useUsersStore } from '@/stores/users'
import { usersApi } from '@/api/users'

const usersStore = useUsersStore()
const serverStatus = ref<Record<string, string>>({
  nginx: 'unknown',
  mysql: 'unknown',
  redis: 'unknown',
})
const stats = ref({ total_users: 0, active_domains: 0, cpu_usage: 0, ram_gb: 0 })

onMounted(async () => {
  await usersStore.fetch({ limit: 5, sort: 'created_at', order: 'desc' })
})
</script>

<template>
  <div class="space-y-5">
    <h1 class="text-lg font-semibold text-gray-800">Dashboard</h1>

    <!-- Stats -->
    <div class="grid grid-cols-4 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Total Users</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">{{ usersStore.total }}</div>
        <div class="text-xs text-emerald-600 mt-1">Active accounts</div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Active Domains</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">—</div>
        <div class="text-xs text-gray-400 mt-1">Across all users</div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">CPU Usage</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">—%</div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-indigo-500 h-1.5 rounded-full" style="width: 0%"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">RAM Usage</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">— GB</div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-amber-500 h-1.5 rounded-full" style="width: 0%"></div>
        </div>
      </div>
    </div>

    <!-- Recent users + server status -->
    <div class="grid grid-cols-[1fr_240px] gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Recent Users</h2>
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-400 border-b border-gray-100">
              <th class="text-left pb-2 font-medium">Username</th>
              <th class="text-left pb-2 font-medium">Package</th>
              <th class="text-left pb-2 font-medium">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in usersStore.users" :key="u.id" class="border-b border-gray-50">
              <td class="py-2 font-medium text-gray-700">{{ u.username }}</td>
              <td class="py-2 text-gray-500">{{ u.package_id || '—' }}</td>
              <td class="py-2">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="u.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'">
                  {{ u.status }}
                </span>
              </td>
            </tr>
            <tr v-if="!usersStore.users.length">
              <td colspan="3" class="py-4 text-center text-gray-400">No users yet</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Server Status</h2>
        <div class="space-y-2 text-xs">
          <div v-for="(status, service) in serverStatus" :key="service"
            class="flex items-center justify-between">
            <span class="text-gray-600 capitalize">{{ service }}</span>
            <span class="px-2 py-0.5 rounded text-[10px] font-medium"
              :class="status === 'running' ? 'bg-green-100 text-green-700' : status === 'disabled' ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-500'">
              {{ status }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
