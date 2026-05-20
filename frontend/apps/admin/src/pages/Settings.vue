<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { systemApi, type SystemStats, type UpdateInfo, type UpdateStatus } from '@/api/system'
import { phpVersionsApi } from '@/api/phpVersions'

const stats = ref<SystemStats | null>(null)
const phpVersions = ref<any[]>([])

// Update flow state. updateInfo holds the result of "Check for updates";
// updateStatus is what we poll while a run is in flight. Null means
// "haven't checked yet" / "no run in flight" — the template uses these
// to decide which buttons to show.
const updateInfo = ref<UpdateInfo | null>(null)
const updateStatus = ref<UpdateStatus | null>(null)
const checking = ref(false)
const starting = ref(false)
const updateError = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const isRunning = computed(() => {
  if (!updateStatus.value) return false
  if (updateStatus.value.done) return false
  return updateStatus.value.phase !== '' && updateStatus.value.phase !== 'idle'
})

const phaseLabel: Record<string, string> = {
  idle: 'Idle',
  starting: 'Starting',
  // Build-from-source phases (legacy fallback)
  pulling: 'Pulling latest source',
  building_api: 'Building API binary',
  building_agent: 'Building agent binary',
  building_cli: 'Building CLI binary',
  building_frontend: 'Building frontend',
  // Download-release phases (preferred)
  downloading: 'Downloading release tarball',
  extracting: 'Extracting tarball',
  deploying_binaries: 'Deploying binaries',
  pulling_source: 'Updating source tree',
  // Shared
  deploying_frontend: 'Deploying frontend',
  restarting: 'Restarting API service',
  done: 'Update complete',
  failed: 'Update failed',
}

async function loadStats() {
  const s = await systemApi.stats()
  stats.value = s.data
}

async function loadPhp() {
  const p = await phpVersionsApi.list()
  phpVersions.value = p.data.data || []
}

async function loadStatus() {
  try {
    const r = await systemApi.updateStatus()
    updateStatus.value = r.data
    if (r.data.done) {
      stopPolling()
      if (r.data.phase === 'done') {
        // Services just restarted — wait a few seconds, then reload so
        // the user sees the new build.
        setTimeout(() => location.reload(), 5000)
      }
    }
  } catch {
    // status endpoint may briefly 502 mid-restart; keep polling.
  }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(loadStatus, 3000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function checkForUpdates() {
  checking.value = true
  updateError.value = ''
  try {
    const r = await systemApi.checkUpdate()
    updateInfo.value = r.data
  } catch (e: any) {
    updateError.value = e.response?.data?.error || 'Failed to check for updates'
  } finally {
    checking.value = false
  }
}

async function applyUpdate() {
  const usingDownload = !!updateInfo.value?.download_url
  const msg = usingDownload
    ? `Download release ${updateInfo.value?.release_tag} and restart? Services will be down for a few seconds.`
    : 'No pre-built release available — fall back to build-from-source? Build can use 1-2 GB RAM and may OOM small VPS hosts. Continue?'
  if (!confirm(msg)) return
  starting.value = true
  updateError.value = ''
  try {
    await systemApi.runUpdate(updateInfo.value?.download_url || '')
    await loadStatus()
    startPolling()
  } catch (e: any) {
    updateError.value = e.response?.data?.error || 'Failed to start update'
  } finally {
    starting.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadStats(), loadPhp(), loadStatus()])
  // Auto-fetch update info on mount so the About card can show the
  // current SHA without the user clicking "Check for updates" first.
  // Failure is non-fatal — the card just falls back to "—".
  try {
    const r = await systemApi.checkUpdate()
    updateInfo.value = r.data
  } catch {
    // ignore — user can click "Check for updates" manually
  }
  // If there's already a run in flight (eg. user reloaded mid-update),
  // resume polling instead of starting from scratch.
  if (updateStatus.value && !updateStatus.value.done && updateStatus.value.phase !== '' && updateStatus.value.phase !== 'idle') {
    startPolling()
  }
  // Load service statuses for maintenance panel.
  runMaintenance('service_status')
})

onUnmounted(stopPolling)

// Maintenance
const maintRunning = ref<string | null>(null)
const maintResult = ref<Record<string, any>>({})
const serviceStatuses = ref<Record<string, string>>({})

async function runMaintenance(action: string) {
  maintRunning.value = action
  try {
    const res = await systemApi.maintenance(action)
    maintResult.value[action] = res.data
    if (action === 'service_status') {
      serviceStatuses.value = res.data.services || {}
    }
  } catch (e: any) {
    maintResult.value[action] = { error: e.response?.data?.error || 'Failed' }
  } finally {
    maintRunning.value = null
  }
}
</script>

<template>
  <div class="space-y-5">
    <div>
      <h1 class="text-lg font-semibold text-gray-800">Settings</h1>
      <p class="text-xs text-gray-400 mt-0.5">Server health, services, and panel updates</p>
    </div>

    <!-- Update card -->
    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-gray-800 text-sm">Updates</h2>
        <button v-if="!isRunning" @click="checkForUpdates" :disabled="checking"
          class="text-xs border border-gray-200 text-gray-600 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
          {{ checking ? 'Checking...' : 'Check for updates' }}
        </button>
      </div>

      <p v-if="updateError" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">
        {{ updateError }}
      </p>

      <!-- No check yet -->
      <p v-if="!updateInfo && !isRunning" class="text-xs text-gray-500">
        Click "Check for updates" to see if a newer panel version is available.
      </p>

      <!-- Up to date -->
      <div v-if="updateInfo && updateInfo.behind_by === 0 && !isRunning"
        class="flex items-center gap-2 text-xs text-green-700 bg-green-50 border border-green-100 rounded px-3 py-2">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        Up to date — current commit {{ updateInfo.current_sha.slice(0, 7) }} on {{ updateInfo.current_branch }}
      </div>

      <!-- Update available -->
      <div v-if="updateInfo && updateInfo.behind_by > 0 && !isRunning" class="space-y-2">
        <div class="text-xs text-amber-700 bg-amber-50 border border-amber-100 rounded px-3 py-2 flex items-center justify-between">
          <span>{{ updateInfo.behind_by }} commit{{ updateInfo.behind_by === 1 ? '' : 's' }} behind origin/{{ updateInfo.current_branch }}</span>
          <button @click="applyUpdate" :disabled="starting"
            class="text-xs bg-indigo-600 text-white px-3 py-1 rounded-md hover:bg-indigo-700 disabled:opacity-50">
            {{ starting ? 'Starting...' : 'Apply update' }}
          </button>
        </div>
        <div v-if="updateInfo.download_url"
          class="text-[11px] text-emerald-700 bg-emerald-50 border border-emerald-100 rounded px-3 py-1.5 flex items-center gap-1.5">
          <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          Pre-built release {{ updateInfo.release_tag }} available — fast download, low memory footprint
        </div>
        <div v-else
          class="text-[11px] text-amber-700 bg-amber-50 border border-amber-100 rounded px-3 py-1.5">
          ⚠ No pre-built release for the latest commit. Falling back to build-from-source uses 1-2 GB RAM and may OOM small VPS hosts.
        </div>
        <details v-if="updateInfo.changelog" class="text-xs">
          <summary class="cursor-pointer text-gray-500 hover:text-gray-700">View changelog excerpt</summary>
          <pre class="mt-2 bg-gray-50 border border-gray-100 rounded p-3 text-[11px] text-gray-700 whitespace-pre-wrap overflow-x-auto">{{ updateInfo.changelog }}</pre>
        </details>
        <div class="text-[11px] text-gray-400">
          {{ updateInfo.current_sha.slice(0, 7) }} → {{ updateInfo.latest_sha.slice(0, 7) }}
        </div>
      </div>

      <!-- Update in progress -->
      <div v-if="isRunning && updateStatus" class="space-y-2">
        <div class="flex items-center gap-2 text-xs text-indigo-700">
          <svg class="w-3.5 h-3.5 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          {{ phaseLabel[updateStatus.phase] || updateStatus.phase }}
        </div>
        <pre class="bg-gray-900 text-gray-100 rounded p-3 text-[11px] max-h-48 overflow-y-auto whitespace-pre-wrap">{{ updateStatus.log.slice(-15).join('\n') }}</pre>
      </div>

      <!-- Update finished -->
      <div v-if="updateStatus && updateStatus.done && updateStatus.phase === 'done'"
        class="text-xs text-green-700 bg-green-50 border border-green-100 rounded px-3 py-2">
        Update complete. This page will reload in a moment.
      </div>
      <div v-if="updateStatus && updateStatus.done && updateStatus.phase === 'failed'"
        class="text-xs text-red-700 bg-red-50 border border-red-100 rounded px-3 py-2">
        Update failed: {{ updateStatus.error }}
      </div>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">Server</h2>
      <dl class="grid grid-cols-2 gap-x-6 gap-y-2 text-xs">
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Total RAM</dt>
          <dd class="text-gray-800 font-medium">
            {{ stats ? (stats.ram_total / 1073741824).toFixed(2) + ' GB' : '—' }}
          </dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Uptime</dt>
          <dd class="text-gray-800 font-medium">
            {{ stats ? Math.floor(stats.uptime_seconds / 86400) + ' days' : '—' }}
          </dd>
        </div>
      </dl>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">Services</h2>
      <dl class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 text-xs">
        <div v-for="(status, svc) in stats?.services ?? {}" :key="svc"
          class="flex items-center justify-between border border-gray-100 rounded-md px-3 py-2">
          <span class="text-gray-600 capitalize">{{ svc }}</span>
          <span class="px-2 py-0.5 rounded text-[10px] font-medium"
            :class="status === 'active' ? 'bg-green-100 text-green-700' : status === 'inactive' || status === 'failed' ? 'bg-red-100 text-red-600' : 'bg-gray-100 text-gray-500'">{{ status }}</span>
        </div>
      </dl>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">PHP Versions</h2>
      <p class="text-xs text-gray-500">Manage which PHP versions are available to users in the PHP Versions page.</p>
      <div class="flex flex-wrap gap-2 text-xs">
        <span v-for="v in phpVersions" :key="v.id"
          class="px-3 py-1 rounded-md border"
          :class="v.enabled ? 'border-indigo-200 bg-indigo-50 text-indigo-700' : 'border-gray-200 bg-gray-50 text-gray-400'">
          PHP {{ v.version }}{{ v.enabled ? '' : ' (disabled)' }}
        </span>
      </div>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <h2 class="font-semibold text-gray-800 text-sm">About</h2>
      <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1 text-xs">
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Panel version</dt>
          <dd class="text-gray-800 font-medium font-mono">
            <span v-if="updateInfo?.release_tag">{{ updateInfo.release_tag }}</span>
            <span v-else-if="updateInfo?.current_sha">{{ updateInfo.current_sha.slice(0, 7) }}</span>
            <span v-else class="text-gray-400">—</span>
          </dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">License</dt>
          <dd class="text-gray-800 font-medium">MIT</dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Docs</dt>
          <dd>
            <a href="https://github.com/teknik-github/zenspanel" target="_blank" rel="noopener"
              class="text-indigo-600 hover:text-indigo-800 transition-colors">GitHub</a>
          </dd>
        </div>
      </dl>
    </div>

    <!-- Maintenance card -->
    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="font-semibold text-gray-800 text-sm">Maintenance</h2>
          <p class="text-xs text-gray-400 mt-0.5">Install, update, and restart server components.</p>
        </div>
        <button @click="runMaintenance('service_status')" :disabled="maintRunning === 'service_status'"
          class="text-xs border border-gray-200 text-gray-600 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
          {{ maintRunning === 'service_status' ? 'Checking...' : 'Refresh status' }}
        </button>
      </div>

      <!-- Service status grid -->
      <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        <div v-for="(status, svc) in serviceStatuses" :key="svc"
          class="flex items-center gap-2 bg-gray-50 rounded-md px-3 py-2">
          <span class="w-2 h-2 rounded-full flex-shrink-0"
            :class="status === 'active' ? 'bg-green-500' : 'bg-red-400'"></span>
          <div class="min-w-0">
            <p class="text-xs font-medium text-gray-700 truncate">{{ svc }}</p>
            <p class="text-[10px]" :class="status === 'active' ? 'text-green-600' : 'text-red-500'">{{ status }}</p>
          </div>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="border-t border-gray-100 pt-3 space-y-2">
        <p class="text-xs font-medium text-gray-600">ClamAV Antivirus</p>
        <div class="flex flex-wrap gap-2">
          <button @click="runMaintenance('clamav_install')" :disabled="!!maintRunning"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
            {{ maintRunning === 'clamav_install' ? 'Installing...' : 'Install / Enable' }}
          </button>
          <button @click="runMaintenance('clamav_update')" :disabled="!!maintRunning"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
            {{ maintRunning === 'clamav_update' ? 'Updating...' : 'Update Definitions' }}
          </button>
          <button @click="runMaintenance('clamav_restart')" :disabled="!!maintRunning"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
            {{ maintRunning === 'clamav_restart' ? 'Restarting...' : 'Restart Daemon' }}
          </button>
        </div>
        <div v-if="maintResult['clamav_install'] || maintResult['clamav_update'] || maintResult['clamav_restart']"
          class="bg-gray-900 rounded p-2 font-mono text-xs text-gray-300 max-h-32 overflow-y-auto">
          <div v-if="maintResult['clamav_install']?.error" class="text-red-400">{{ maintResult['clamav_install'].error }}</div>
          <div v-else-if="maintResult['clamav_install']?.output">{{ maintResult['clamav_install'].output }}</div>
          <div v-if="maintResult['clamav_update']?.error" class="text-red-400">{{ maintResult['clamav_update'].error }}</div>
          <div v-else-if="maintResult['clamav_update']?.output">{{ maintResult['clamav_update'].output }}</div>
          <div v-if="maintResult['clamav_restart']?.error" class="text-red-400">{{ maintResult['clamav_restart'].error }}</div>
          <div v-else-if="maintResult['clamav_restart']?.output" class="text-green-400">{{ maintResult['clamav_restart'].output || 'Restarted successfully' }}</div>
        </div>
      </div>

      <div class="border-t border-gray-100 pt-3 space-y-2">
        <p class="text-xs font-medium text-gray-600">Other Services</p>
        <div class="flex flex-wrap gap-2">
          <button @click="runMaintenance('fail2ban_restart')" :disabled="!!maintRunning"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
            {{ maintRunning === 'fail2ban_restart' ? 'Restarting...' : 'Restart fail2ban' }}
          </button>
          <button @click="runMaintenance('nginx_reload')" :disabled="!!maintRunning"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
            {{ maintRunning === 'nginx_reload' ? 'Reloading...' : 'Reload nginx' }}
          </button>
        </div>
        <div v-if="maintResult['fail2ban_restart'] || maintResult['nginx_reload']"
          class="bg-gray-900 rounded p-2 font-mono text-xs max-h-20 overflow-y-auto">
          <div v-if="maintResult['fail2ban_restart']?.error" class="text-red-400">{{ maintResult['fail2ban_restart'].error }}</div>
          <div v-else-if="maintResult['fail2ban_restart'] && !maintResult['fail2ban_restart']?.error" class="text-green-400">fail2ban restarted</div>
          <div v-if="maintResult['nginx_reload']?.error" class="text-red-400">{{ maintResult['nginx_reload'].error }}</div>
          <div v-else-if="maintResult['nginx_reload'] && !maintResult['nginx_reload']?.error" class="text-green-400">nginx reloaded</div>
        </div>
      </div>
    </div>
  </div>
</template>
