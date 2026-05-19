<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { domainsApi } from '@/api/domains'
import { logsApi } from '@/api/logs'

const domains = ref<any[]>([])
const selectedDomain = ref<number | null>(null)
const logType = ref('nginx-error')
const lineCount = ref(100)
const lines = ref<string[]>([])
const loading = ref(false)
const autoRefresh = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const logTypes = [
  { value: 'nginx-error',  label: 'Nginx Error' },
  { value: 'nginx-access', label: 'Nginx Access' },
  { value: 'fpm',          label: 'PHP-FPM' },
]

const lineCounts = [50, 100, 200, 500]

onMounted(async () => {
  const res = await domainsApi.list()
  domains.value = res.data.data || []
  if (domains.value.length) {
    selectedDomain.value = domains.value[0].id
    await fetchLogs()
  }
})

onUnmounted(() => {
  stopRefresh()
})

watch([selectedDomain, logType, lineCount], async () => {
  await fetchLogs()
})

watch(autoRefresh, (v) => {
  if (v) startRefresh()
  else stopRefresh()
})

async function fetchLogs() {
  if (!selectedDomain.value) return
  loading.value = true
  try {
    const res = await logsApi.domain(selectedDomain.value, logType.value, lineCount.value)
    lines.value = res.data.lines || []
  } catch {
    lines.value = []
  } finally {
    loading.value = false
  }
}

function startRefresh() {
  stopRefresh()
  refreshTimer = setInterval(fetchLogs, 5000)
}

function stopRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// Scroll log container to bottom when lines update
function scrollToBottom(el: HTMLElement | null) {
  if (el) el.scrollTop = el.scrollHeight
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">Error Logs</h1>

    <!-- Controls -->
    <div class="bg-white border border-gray-200 rounded-lg p-3 flex flex-wrap items-center gap-3">
      <div class="flex items-center gap-2">
        <label class="text-xs text-gray-500 font-medium">Domain</label>
        <select v-model="selectedDomain"
          class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
          <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.domain }}</option>
        </select>
      </div>
      <div class="flex items-center gap-2">
        <label class="text-xs text-gray-500 font-medium">Log</label>
        <select v-model="logType"
          class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
          <option v-for="t in logTypes" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
      </div>
      <div class="flex items-center gap-2">
        <label class="text-xs text-gray-500 font-medium">Lines</label>
        <select v-model="lineCount"
          class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
          <option v-for="n in lineCounts" :key="n" :value="n">{{ n }}</option>
        </select>
      </div>
      <div class="flex items-center gap-2 ml-auto">
        <label class="flex items-center gap-1.5 cursor-pointer text-xs text-gray-600">
          <input type="checkbox" v-model="autoRefresh" class="rounded border-gray-300 text-indigo-600" />
          Auto-refresh (5s)
        </label>
        <button @click="fetchLogs" :disabled="loading"
          class="text-xs border border-gray-200 px-3 py-1 rounded hover:bg-gray-50 disabled:opacity-50 flex items-center gap-1">
          <svg class="w-3 h-3" :class="loading ? 'animate-spin' : ''" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
          </svg>
          Refresh
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!domains.length"
      class="flex flex-col items-center justify-center py-16 bg-white border border-gray-200 rounded-lg text-center">
      <div class="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mb-3">
        <svg class="w-6 h-6 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No domains found</p>
      <p class="text-xs text-gray-400 mt-1">Add a domain first to view its logs.</p>
    </div>

    <!-- Log output -->
    <div v-else class="bg-gray-900 rounded-lg overflow-hidden">
      <div class="px-3 py-2 bg-gray-800 flex items-center justify-between">
        <span class="text-xs text-gray-400 font-mono">
          {{ logTypes.find(t => t.value === logType)?.label }} —
          {{ domains.find(d => d.id === selectedDomain)?.domain }}
        </span>
        <span class="text-xs text-gray-500">{{ lines.length }} lines</span>
      </div>
      <div
        :ref="(el) => scrollToBottom(el as HTMLElement)"
        class="overflow-y-auto max-h-[60vh] p-3 font-mono text-xs leading-5">
        <div v-if="loading && !lines.length" class="text-gray-500 animate-pulse">Loading...</div>
        <div v-else-if="!lines.length" class="text-gray-500">No log entries found.</div>
        <div v-else>
          <div v-for="(line, i) in lines" :key="i"
            class="whitespace-pre-wrap break-all"
            :class="{
              'text-red-400':    line.includes('[error]') || line.includes('[crit]') || line.includes('[emerg]'),
              'text-yellow-400': line.includes('[warn]'),
              'text-gray-300':   !line.includes('[error]') && !line.includes('[crit]') && !line.includes('[emerg]') && !line.includes('[warn]'),
            }">{{ line }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
