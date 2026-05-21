<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { systemApi, type UpdateInfo, type UpdateStatus } from '@/api/system'
import { useConfirm, useToast } from '../notify'

const { confirm } = useConfirm()
const { error: toastError } = useToast()

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
  pulling: 'Pulling latest source',
  building_api: 'Building API binary',
  building_agent: 'Building agent binary',
  building_cli: 'Building CLI binary',
  building_frontend: 'Building frontend',
  downloading: 'Downloading release tarball',
  extracting: 'Extracting tarball',
  deploying_binaries: 'Deploying binaries',
  pulling_source: 'Updating source tree',
  deploying_frontend: 'Deploying frontend',
  restarting: 'Restarting API service',
  done: 'Update complete',
  failed: 'Update failed',
}

async function loadStatus() {
  try {
    const r = await systemApi.updateStatus()
    updateStatus.value = r.data
    if (r.data.done) {
      stopPolling()
      if (r.data.phase === 'done' && pollTimer !== null) {
        setTimeout(() => location.reload(), 5000)
      }
    }
  } catch { /* ignore */ }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(loadStatus, 3000)
}

function stopPolling() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
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
  const ok = await confirm({ title: 'Apply Update', message: msg, confirmLabel: 'Apply', danger: !usingDownload })
  if (!ok) return
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
  await loadStatus()
  try {
    const r = await systemApi.checkUpdate()
    updateInfo.value = r.data
  } catch { /* ignore */ }
  if (updateStatus.value && !updateStatus.value.done && updateStatus.value.phase !== '' && updateStatus.value.phase !== 'idle') {
    startPolling()
  }
})

onUnmounted(stopPolling)
</script>

<template>
  <div class="space-y-5 max-w-2xl">
    <div>
      <h1 class="text-lg font-semibold text-gray-800">Updates</h1>
      <p class="text-xs text-gray-400 mt-0.5">Check for and apply panel updates</p>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-gray-800 text-sm">Panel Version</h2>
        <button v-if="!isRunning" @click="checkForUpdates" :disabled="checking"
          class="text-xs border border-gray-200 text-gray-600 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
          {{ checking ? 'Checking...' : 'Check for updates' }}
        </button>
      </div>

      <p v-if="updateError" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">
        {{ updateError }}
      </p>

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
        <div v-else class="text-[11px] text-amber-700 bg-amber-50 border border-amber-100 rounded px-3 py-1.5">
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
  </div>
</template>
