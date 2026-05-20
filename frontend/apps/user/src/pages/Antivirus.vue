<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { antivirusApi } from '@/api/antivirus'

const daemonRunning = ref<boolean | null>(null)
const scanning = ref(false)
const scanPath = ref('')
const jobId = ref('')
const scanResult = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  try {
    const res = await antivirusApi.status()
    daemonRunning.value = res.data.running ?? false
  } catch {
    daemonRunning.value = false
  }
})

onUnmounted(() => stopPoll())

async function startScan() {
  scanning.value = true
  scanResult.value = null
  jobId.value = ''
  try {
    const res = await antivirusApi.scan(scanPath.value || undefined)
    jobId.value = res.data.job_id
    startPoll()
  } catch (e: any) {
    scanning.value = false
    scanResult.value = { error: e.response?.data?.error || 'Scan failed to start' }
  }
}

function startPoll() {
  stopPoll()
  pollTimer = setInterval(async () => {
    try {
      const res = await antivirusApi.scanStatus(jobId.value)
      scanResult.value = res.data
      if (res.data.done) {
        stopPoll()
        scanning.value = false
      }
    } catch {
      stopPoll()
      scanning.value = false
    }
  }, 2000)
}

function stopPoll() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}
</script>

<template>
  <div class="space-y-4 max-w-2xl">
    <h1 class="text-lg font-semibold text-gray-800">Antivirus Scanner</h1>

    <!-- Daemon status -->
    <div class="bg-white border border-gray-200 rounded-lg p-4 flex items-center justify-between">
      <div>
        <h2 class="text-sm font-semibold text-gray-800">ClamAV Status</h2>
        <p class="text-xs text-gray-500 mt-0.5">Real-time virus definition updates via freshclam.</p>
      </div>
      <span class="flex items-center gap-1.5 text-xs font-medium"
        :class="daemonRunning === null ? 'text-gray-400' : daemonRunning ? 'text-green-600' : 'text-red-500'">
        <span class="w-2 h-2 rounded-full"
          :class="daemonRunning === null ? 'bg-gray-300' : daemonRunning ? 'bg-green-500' : 'bg-red-400'"></span>
        {{ daemonRunning === null ? 'Checking...' : daemonRunning ? 'Running' : 'Not running' }}
      </span>
    </div>

    <!-- Scan form -->
    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="text-sm font-semibold text-gray-800">Scan Files</h2>
      <p class="text-xs text-gray-500">Scan your home directory for malware. Leave path empty to scan everything.</p>
      <div class="flex gap-2">
        <input v-model="scanPath" type="text" placeholder="public_html/  (empty = full scan)"
          :disabled="scanning"
          class="flex-1 border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-gray-50" />
        <button @click="startScan" :disabled="scanning || daemonRunning === false"
          class="bg-indigo-600 text-white text-sm px-4 py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50 flex items-center gap-2">
          <svg v-if="scanning" class="w-4 h-4 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          {{ scanning ? 'Scanning...' : 'Scan' }}
        </button>
      </div>
      <p v-if="daemonRunning === false" class="text-xs text-red-500">
        ClamAV daemon is not running. Contact your administrator.
      </p>
    </div>

    <!-- Scan progress -->
    <div v-if="scanning && !scanResult?.done" class="bg-white border border-gray-200 rounded-lg p-4">
      <div class="flex items-center gap-3">
        <svg class="w-5 h-5 text-indigo-500 animate-spin flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
        </svg>
        <div>
          <p class="text-sm font-medium text-gray-800">Scanning in progress...</p>
          <p class="text-xs text-gray-400 mt-0.5">{{ scanResult?.scanned ?? 0 }} files scanned</p>
        </div>
      </div>
    </div>

    <!-- Scan results -->
    <div v-if="scanResult?.done || scanResult?.error" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 border-b border-gray-200"
        :class="scanResult?.error ? 'bg-red-50' : scanResult?.infected?.length ? 'bg-red-50' : 'bg-green-50'">
        <div class="flex items-center gap-2">
          <svg v-if="!scanResult?.error && !scanResult?.infected?.length" class="w-5 h-5 text-green-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
          <svg v-else class="w-5 h-5 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          <div>
            <p class="text-sm font-semibold"
              :class="scanResult?.error ? 'text-red-800' : scanResult?.infected?.length ? 'text-red-800' : 'text-green-800'">
              <template v-if="scanResult?.error">Scan error</template>
              <template v-else-if="scanResult?.infected?.length">{{ scanResult.infected.length }} infected file(s) found</template>
              <template v-else>No threats found</template>
            </p>
            <p class="text-xs mt-0.5"
              :class="scanResult?.error ? 'text-red-600' : 'text-gray-500'">
              <template v-if="scanResult?.error">{{ scanResult.error }}</template>
              <template v-else>{{ scanResult?.scanned ?? 0 }} files scanned</template>
            </p>
          </div>
        </div>
      </div>

      <div v-if="scanResult?.infected?.length" class="divide-y divide-gray-50">
        <div v-for="file in scanResult.infected" :key="file"
          class="px-4 py-2.5 flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-red-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          </svg>
          <span class="text-xs font-mono text-gray-700 break-all">{{ file }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
