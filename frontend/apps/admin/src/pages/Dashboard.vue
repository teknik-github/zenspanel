<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useUsersStore } from '@/stores/users'
import { systemApi, type SystemStats } from '@/api/system'

const usersStore = useUsersStore()
const stats = ref<SystemStats | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const ramPercent = computed(() => {
  if (!stats.value || !stats.value.ram_total) return 0
  return (stats.value.ram_used / stats.value.ram_total) * 100
})

const ramLabel = computed(() => {
  if (!stats.value) return '—'
  const used = stats.value.ram_used / 1073741824
  const total = stats.value.ram_total / 1073741824
  return `${used.toFixed(1)} / ${total.toFixed(1)} GB`
})

async function loadStats() {
  try {
    const res = await systemApi.stats()
    stats.value = res.data
  } catch {
    // keep last known good values; the dashboard shouldn't disappear
    // because of one transient failure
  }
}

onMounted(async () => {
  await Promise.all([
    usersStore.fetch({ limit: 5, sort: 'created_at', order: 'desc' }),
    loadStats(),
  ])
  // refresh stats every 10s while the dashboard is open
  timer = setInterval(loadStats, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function statusClass(status?: string) {
  if (status === 'active') return 'bg-green-100 text-green-700'
  if (status === 'inactive' || status === 'failed') return 'bg-red-100 text-red-600'
  return 'bg-gray-100 text-gray-500'
}
</script>

<template>
  <div class="space-y-5">
    <h1 class="text-lg font-semibold text-gray-800">Dashboard</h1>

    <!-- Stats -->
    <div class="grid grid-cols-4 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Total Users</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">{{ stats?.users.total ?? usersStore.total }}</div>
        <div class="text-xs text-emerald-600 mt-1">
          {{ stats?.users.active ?? 0 }} active · {{ stats?.users.suspended ?? 0 }} suspended
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Active Domains</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">{{ stats?.domains.active ?? '—' }}</div>
        <div class="text-xs text-gray-400 mt-1">
          of {{ stats?.domains.total ?? 0 }} total
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">CPU Usage</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">{{ stats ? stats.cpu_percent.toFixed(1) : '—' }}%</div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-indigo-500 h-1.5 rounded-full transition-all"
            :style="{ width: (stats?.cpu_percent ?? 0) + '%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">RAM Usage</div>
        <div class="text-2xl font-bold text-gray-800 mt-1">{{ ramLabel }}</div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-amber-500 h-1.5 rounded-full transition-all"
            :style="{ width: ramPercent + '%' }"></div>
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
          <div v-for="(status, service) in stats?.services ?? { nginx: 'unknown', mysql: 'unknown', redis: 'unknown' }" :key="service"
            class="flex items-center justify-between">
            <span class="text-gray-600 capitalize">{{ service }}</span>
            <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(status)">
              {{ status }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
