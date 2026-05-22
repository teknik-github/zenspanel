<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useUsersStore } from '@/stores/users'
import { systemApi, type SystemStats } from '@/api/system'

const usersStore = useUsersStore()
const stats = ref<SystemStats | null>(null)
const loaded = ref(false)
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
  loaded.value = true
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
    <div>
      <h1 class="text-lg font-semibold text-gray-800">Dashboard</h1>
      <p class="text-xs text-gray-400 mt-0.5">Server health and panel activity</p>
    </div>

    <!-- Stat cards. Skeleton placeholders match the same grid shape so the
         layout doesn't jump when data arrives. -->
    <div v-if="!loaded" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div v-for="i in 4" :key="i" class="bg-white border border-gray-200 rounded-lg p-4 animate-pulse">
        <div class="h-3 w-20 bg-gray-100 rounded" />
        <div class="h-7 w-24 bg-gray-100 rounded mt-2" />
        <div class="h-3 w-16 bg-gray-100 rounded mt-2" />
      </div>
    </div>

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4 border-l-4 border-l-indigo-500">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-indigo-50 text-indigo-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">Total Users</div>
        </div>
        <div class="text-2xl font-bold text-gray-800 mt-2">{{ stats?.users.total ?? usersStore.total }}</div>
        <div class="text-xs text-emerald-600 mt-1">
          {{ stats?.users.active ?? 0 }} active · {{ stats?.users.suspended ?? 0 }} suspended
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4 border-l-4 border-l-blue-500">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-blue-50 text-blue-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">Active Domains</div>
        </div>
        <div class="text-2xl font-bold text-gray-800 mt-2">{{ stats?.domains.active ?? '—' }}</div>
        <div class="text-xs text-gray-400 mt-1">
          of {{ stats?.domains.total ?? 0 }} total
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4 border-l-4 border-l-purple-500">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-purple-50 text-purple-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">CPU Usage</div>
        </div>
        <div class="text-2xl font-bold text-gray-800 mt-2">{{ stats ? stats.cpu_percent.toFixed(1) : '—' }}<span class="text-base font-normal text-gray-400">%</span></div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-purple-500 h-1.5 rounded-full transition-all"
            :style="{ width: Math.min(stats?.cpu_percent ?? 0, 100) + '%' }" />
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4 border-l-4 border-l-amber-500">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-amber-50 text-amber-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 12h.01M10 12h.01M14 12h.01M18 12h.01"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">RAM Usage</div>
        </div>
        <div class="text-2xl font-bold text-gray-800 mt-2">{{ ramLabel }}</div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-amber-500 h-1.5 rounded-full transition-all"
            :style="{ width: Math.min(ramPercent, 100) + '%' }" />
        </div>
      </div>
    </div>

    <!-- Recent users + server status. Stacks on small screens. -->
    <div class="grid grid-cols-1 lg:grid-cols-[1fr_260px] gap-4 items-start">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold text-gray-800 text-sm">Recent Users</h2>
          <router-link to="/users" class="text-xs text-indigo-600 hover:text-indigo-800 transition-colors">View all →</router-link>
        </div>

        <div v-if="!loaded" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-6 bg-gray-50 rounded animate-pulse" />
        </div>

        <div v-else-if="!usersStore.users.length" class="flex flex-col items-center justify-center py-10 text-center">
          <div class="w-10 h-10 rounded-full bg-indigo-50 text-indigo-500 flex items-center justify-center mb-2">
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>
            </svg>
          </div>
          <p class="text-xs font-medium text-gray-700">No users yet</p>
          <router-link to="/users" class="text-[11px] text-indigo-600 mt-1 hover:text-indigo-800 transition-colors">Create your first user →</router-link>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full text-xs min-w-[400px]">
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
            </tbody>
          </table>
        </div>
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
