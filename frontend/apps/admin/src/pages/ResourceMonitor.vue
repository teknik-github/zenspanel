<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { systemApi, type SystemStats } from '@/api/system'

const stats = ref<SystemStats | null>(null)
const userMetrics = ref<any[]>([])
const error = ref('')
let timer: ReturnType<typeof setInterval> | null = null

const ramPercent = computed(() => {
  if (!stats.value || !stats.value.ram_total) return 0
  return (stats.value.ram_used / stats.value.ram_total) * 100
})

const ramLabel = computed(() => {
  if (!stats.value) return '— / —'
  const used = stats.value.ram_used / 1073741824
  const total = stats.value.ram_total / 1073741824
  return `${used.toFixed(2)} / ${total.toFixed(2)} GB`
})

const uptimeLabel = computed(() => {
  if (!stats.value) return '—'
  let s = stats.value.uptime_seconds
  const days = Math.floor(s / 86400); s %= 86400
  const hours = Math.floor(s / 3600); s %= 3600
  const mins = Math.floor(s / 60)
  return `${days}d ${hours}h ${mins}m`
})

// Sort users by CPU desc so abusers float to top
const sortedUsers = computed(() =>
  [...userMetrics.value].sort((a, b) => b.cpu_pct - a.cpu_pct)
)

function pct(used: number, max: number) {
  if (!max) return 0
  return Math.min(100, (used / max) * 100)
}

function fmtBytes(b: number) {
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  if (b >= 1048576) return (b / 1048576).toFixed(0) + ' MB'
  return b + ' B'
}

function barColor(pct: number) {
  if (pct >= 90) return 'bg-red-500'
  if (pct >= 70) return 'bg-amber-500'
  return 'bg-indigo-500'
}

async function load() {
  try {
    const [statsRes, metricsRes] = await Promise.all([
      systemApi.stats(),
      systemApi.userMetrics(),
    ])
    stats.value = statsRes.data
    userMetrics.value = metricsRes.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load stats'
  }
}

onMounted(async () => {
  await load()
  timer = setInterval(load, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function svcClass(s?: string) {
  if (s === 'active') return 'bg-green-100 text-green-700'
  if (s === 'inactive' || s === 'failed') return 'bg-red-100 text-red-600'
  return 'bg-gray-100 text-gray-500'
}
</script>

<template>
  <div class="space-y-5">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Resource Monitor</h1>
      <span class="text-xs text-gray-400">auto-refreshes every 5s</span>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <!-- Server overview -->
    <div class="grid grid-cols-3 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-5">
        <div class="text-xs text-gray-400 uppercase tracking-wide">CPU</div>
        <div class="text-3xl font-bold text-gray-800 mt-2">{{ stats ? stats.cpu_percent.toFixed(1) : '—' }}%</div>
        <div class="mt-3 bg-gray-100 rounded-full h-2">
          <div class="bg-indigo-500 h-2 rounded-full transition-all"
            :style="{ width: (stats?.cpu_percent ?? 0) + '%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-5">
        <div class="text-xs text-gray-400 uppercase tracking-wide">RAM</div>
        <div class="text-3xl font-bold text-gray-800 mt-2">{{ ramPercent.toFixed(0) }}%</div>
        <div class="text-xs text-gray-500 mt-1">{{ ramLabel }}</div>
        <div class="mt-3 bg-gray-100 rounded-full h-2">
          <div class="bg-amber-500 h-2 rounded-full transition-all"
            :style="{ width: ramPercent + '%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-5">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Uptime</div>
        <div class="text-3xl font-bold text-gray-800 mt-2">{{ uptimeLabel }}</div>
        <div class="text-xs text-gray-500 mt-3">since last reboot</div>
      </div>
    </div>

    <div class="grid grid-cols-[1fr_280px] gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Panel Inventory</h2>
        <div class="grid grid-cols-3 gap-3 text-xs">
          <div>
            <div class="text-gray-400">Users</div>
            <div class="text-xl font-bold text-gray-800 mt-1">{{ stats?.users.total ?? '—' }}</div>
            <div class="text-[10px] text-gray-500">
              {{ stats?.users.active ?? 0 }} active · {{ stats?.users.suspended ?? 0 }} suspended
            </div>
          </div>
          <div>
            <div class="text-gray-400">Domains</div>
            <div class="text-xl font-bold text-gray-800 mt-1">{{ stats?.domains.total ?? '—' }}</div>
            <div class="text-[10px] text-gray-500">{{ stats?.domains.active ?? 0 }} active</div>
          </div>
          <div>
            <div class="text-gray-400">Databases</div>
            <div class="text-xl font-bold text-gray-800 mt-1">{{ stats?.databases.total ?? '—' }}</div>
            <div class="text-[10px] text-gray-500">across all users</div>
          </div>
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Services</h2>
        <div class="space-y-2 text-xs">
          <div v-for="(status, svc) in stats?.services ?? { nginx: 'unknown', mysql: 'unknown', redis: 'unknown' }" :key="svc"
            class="flex items-center justify-between">
            <span class="text-gray-600 capitalize">{{ svc }}</span>
            <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="svcClass(status)">{{ status }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Per-user resource usage -->
    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-gray-800">Per-User Resource Usage</h2>
          <p class="text-xs text-gray-400 mt-0.5">Sorted by CPU — high usage floats to top. Red = over 90% of quota.</p>
        </div>
        <span class="text-xs text-gray-400">{{ userMetrics.length }} active users</span>
      </div>

      <div v-if="!userMetrics.length" class="px-4 py-8 text-center text-xs text-gray-400">
        No active users or cgroup metrics unavailable.
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full text-xs min-w-[700px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-2.5 font-medium">User</th>
              <th class="text-left px-4 py-2.5 font-medium w-40">CPU</th>
              <th class="text-left px-4 py-2.5 font-medium w-48">RAM</th>
              <th class="text-left px-4 py-2.5 font-medium w-48">Disk</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in sortedUsers" :key="u.id"
              class="border-b border-gray-50 hover:bg-gray-50"
              :class="{ 'bg-red-50': u.cpu_pct >= 90 || pct(u.ram_used, u.ram_max) >= 90 || pct(u.disk_used, u.disk_max) >= 90 }">
              <td class="px-4 py-2.5">
                <div class="font-medium text-gray-800">{{ u.username }}</div>
                <div v-if="u.cpu_pct >= 90 || pct(u.ram_used, u.ram_max) >= 90 || pct(u.disk_used, u.disk_max) >= 90"
                  class="flex items-center gap-1 text-[10px] text-red-500 font-medium mt-0.5">
                  <svg class="w-3 h-3 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
                  </svg>
                  High usage
                </div>
              </td>
              <td class="px-4 py-2.5">
                <div class="flex items-center gap-2">
                  <div class="flex-1 bg-gray-100 rounded-full h-1.5">
                    <div class="h-1.5 rounded-full transition-all" :class="barColor(u.cpu_pct)"
                      :style="{ width: u.cpu_pct + '%' }"></div>
                  </div>
                  <span class="text-gray-600 w-10 text-right">{{ u.cpu_pct.toFixed(1) }}%</span>
                </div>
              </td>
              <td class="px-4 py-2.5">
                <div class="flex items-center gap-2">
                  <div class="flex-1 bg-gray-100 rounded-full h-1.5">
                    <div class="h-1.5 rounded-full transition-all" :class="barColor(pct(u.ram_used, u.ram_max))"
                      :style="{ width: pct(u.ram_used, u.ram_max) + '%' }"></div>
                  </div>
                  <span class="text-gray-600 w-20 text-right">
                    {{ fmtBytes(u.ram_used) }}<span v-if="u.ram_max"> / {{ fmtBytes(u.ram_max) }}</span>
                  </span>
                </div>
              </td>
              <td class="px-4 py-2.5">
                <div class="flex items-center gap-2">
                  <div class="flex-1 bg-gray-100 rounded-full h-1.5">
                    <div class="h-1.5 rounded-full transition-all" :class="barColor(pct(u.disk_used, u.disk_max))"
                      :style="{ width: pct(u.disk_used, u.disk_max) + '%' }"></div>
                  </div>
                  <span class="text-gray-600 w-20 text-right">
                    {{ fmtBytes(u.disk_used) }}<span v-if="u.disk_max"> / {{ fmtBytes(u.disk_max) }}</span>
                  </span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
