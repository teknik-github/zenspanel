<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { systemApi, type SystemStats } from '@/api/system'

const stats = ref<SystemStats | null>(null)
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

async function load() {
  try {
    const res = await systemApi.stats()
    stats.value = res.data
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
  </div>
</template>
