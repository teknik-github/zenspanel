<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { antivirusApi } from '@/api/antivirus'
import { packagesApi } from '@/api/packages'
import { useToast } from '../notify'
import { useAuthStore } from '@/stores/auth'

const { error: toastError } = useToast()
const auth = useAuthStore()
const antivirusAllowed = ref<boolean | null>(null) // null = loading

const daemonRunning = ref<boolean | null>(null)
const scanning = ref(false)
const scanPath = ref('')
const jobId = ref('')
const scanResult = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

// Realtime watch state
const watching = ref(false)
const watchId = ref('')
const realtimeAlerts = ref<any[]>([])
const storedAlerts = ref<any[]>([])
let alertPollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  // Check if antivirus is enabled in the user's package (V57).
  if (auth.user?.package_id) {
    try {
      const pkgRes = await packagesApi.get(auth.user.package_id)
      antivirusAllowed.value = pkgRes.data.antivirus_enabled ?? true
    } catch {
      antivirusAllowed.value = true // fail-open
    }
  } else {
    antivirusAllowed.value = true // no package = no restriction
  }

  try {
    const res = await antivirusApi.status()
    daemonRunning.value = res.data.running ?? false
  } catch {
    daemonRunning.value = false
  }
  // Load stored alerts from DB
  try {
    const res = await antivirusApi.alerts()
    storedAlerts.value = res.data.data || []
  } catch { /* ignore */ }
})

onUnmounted(() => {
  stopPoll()
  stopAlertPoll()
  if (watchId.value) antivirusApi.watchStop(watchId.value).catch(() => {})
})

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

async function toggleWatch() {
  if (watching.value) {
    // Stop watching
    if (watchId.value) {
      await antivirusApi.watchStop(watchId.value).catch(() => {})
      watchId.value = ''
    }
    stopAlertPoll()
    watching.value = false
  } else {
    // Start watching
    try {
      const res = await antivirusApi.watchStart()
      watchId.value = res.data.watch_id || ''
      watching.value = true
      startAlertPoll()
    } catch (e: any) {
      toastError(e.response?.data?.error || 'Failed to start realtime watch')
    }
  }
}

function startAlertPoll() {
  stopAlertPoll()
  alertPollTimer = setInterval(async () => {
    try {
      const res = await antivirusApi.poll()
      if (res.data.new_alerts > 0) {
        realtimeAlerts.value = [...res.data.alerts, ...realtimeAlerts.value].slice(0, 100)
        storedAlerts.value = [...res.data.alerts, ...storedAlerts.value].slice(0, 50)
      }
    } catch { /* ignore */ }
  }, 5000)
}

function stopAlertPoll() {
  if (alertPollTimer) { clearInterval(alertPollTimer); alertPollTimer = null }
}
</script>

<template>
  <div class="space-y-4 max-w-2xl">
    <h1 class="text-lg font-semibold text-gray-800">Antivirus Scanner</h1>

    <!-- Package gate (V57) -->
    <div v-if="antivirusAllowed === false"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <svg class="w-10 h-10 text-gray-300 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
      </svg>
      <p class="text-sm font-medium text-gray-700">Antivirus not available</p>
      <p class="text-xs text-gray-400 mt-1">This feature is not included in your current plan. Contact your administrator to upgrade.</p>
    </div>

    <template v-else-if="antivirusAllowed === true">

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

    <!-- Realtime protection -->
    <div class="bg-white border border-gray-200 rounded-lg p-4">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-sm font-semibold text-gray-800">Realtime Protection</h2>
          <p class="text-xs text-gray-500 mt-0.5">
            Monitors your files for threats as they are created or modified.
          </p>
        </div>
        <button @click="toggleWatch" :disabled="daemonRunning === false"
          class="text-xs px-3 py-1.5 rounded-md border transition-colors disabled:opacity-50"
          :class="watching
            ? 'bg-red-50 text-red-600 border-red-200 hover:bg-red-100'
            : 'bg-green-50 text-green-600 border-green-200 hover:bg-green-100'">
          {{ watching ? 'Stop Protection' : 'Start Protection' }}
        </button>
      </div>
      <div v-if="watching" class="mt-2 flex items-center gap-1.5 text-xs text-green-600">
        <span class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></span>
        Monitoring active — scanning new and modified files
      </div>

      <!-- Realtime alerts -->
      <div v-if="realtimeAlerts.length" class="mt-3 space-y-1">
        <p class="text-xs font-medium text-red-600">{{ realtimeAlerts.length }} threat(s) detected this session:</p>
        <div v-for="(a, i) in realtimeAlerts" :key="i"
          class="flex items-start gap-2 bg-red-50 border border-red-100 rounded px-3 py-2">
          <svg class="w-3.5 h-3.5 text-red-500 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          <div class="min-w-0">
            <p class="text-xs font-mono text-red-700 truncate">{{ a.path }}</p>
            <p class="text-[10px] text-red-500">{{ a.threat }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Manual scan -->
    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="text-sm font-semibold text-gray-800">Manual Scan</h2>
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
          {{ scanning ? 'Scanning...' : 'Scan' }}
        </button>
      </div>
    </div>

    <!-- Scan progress/result -->
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
            <p class="text-xs mt-0.5" :class="scanResult?.error ? 'text-red-600' : 'text-gray-500'">
              <template v-if="scanResult?.error">{{ scanResult.error }}</template>
              <template v-else>{{ scanResult?.scanned ?? 0 }} files scanned</template>
            </p>
          </div>
        </div>
      </div>
      <div v-if="scanResult?.infected?.length" class="divide-y divide-gray-50">
        <div v-for="file in scanResult.infected" :key="file" class="px-4 py-2.5 flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-red-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          </svg>
          <span class="text-xs font-mono text-gray-700 break-all">{{ file }}</span>
        </div>
      </div>
    </div>

    <!-- Alert history -->
    <div v-if="storedAlerts.length" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200">
        <h2 class="text-sm font-semibold text-gray-800">Alert History</h2>
      </div>
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-2 font-medium">File</th>
            <th class="text-left px-4 py-2 font-medium">Threat</th>
            <th class="text-left px-4 py-2 font-medium">Detected</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in storedAlerts" :key="a.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-2 font-mono text-gray-700 max-w-xs truncate" :title="a.path">{{ a.path }}</td>
            <td class="px-4 py-2 text-red-600">{{ a.threat }}</td>
            <td class="px-4 py-2 text-gray-400">{{ new Date(a.detected_at).toLocaleString() }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </template>
  </div>
</template>
