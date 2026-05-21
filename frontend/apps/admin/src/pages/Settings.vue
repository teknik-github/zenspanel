<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { systemApi, type SystemStats } from '@/api/system'
import { phpVersionsApi } from '@/api/phpVersions'

const stats = ref<SystemStats | null>(null)
const phpVersions = ref<any[]>([])
const currentVersion = ref('—')

async function loadStats() {
  const s = await systemApi.stats()
  stats.value = s.data
}

async function loadPhp() {
  const p = await phpVersionsApi.list()
  phpVersions.value = p.data.data || []
}

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

onMounted(async () => {
  await Promise.all([loadStats(), loadPhp()])
  runMaintenance('service_status')
  try {
    const r = await systemApi.version()
    currentVersion.value = r.data.version
  } catch { /* ignore */ }
})
</script>

<template>
  <div class="space-y-5">
    <div>
      <h1 class="text-lg font-semibold text-gray-800">Settings</h1>
      <p class="text-xs text-gray-400 mt-0.5">Server health and services</p>
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
          <dd class="text-gray-800 font-medium font-mono">{{ currentVersion }}</dd>
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

      <div class="border-t border-gray-100 pt-3 space-y-2">
        <p class="text-xs font-medium text-gray-600">Optional Tools</p>
        <p class="text-xs text-gray-400">Required for: Antivirus Realtime (inotify-tools) · S3 Backups (rclone)</p>
        <div class="flex items-center gap-3 flex-wrap">
          <div class="flex gap-2 text-[10px]">
            <span class="flex items-center gap-1">
              <span class="w-2 h-2 rounded-full"
                :class="serviceStatuses['inotify-tools'] === 'active' ? 'bg-green-500' : 'bg-red-400'"></span>
              inotify-tools
            </span>
            <span class="flex items-center gap-1">
              <span class="w-2 h-2 rounded-full"
                :class="serviceStatuses['rclone'] === 'active' ? 'bg-green-500' : 'bg-red-400'"></span>
              rclone
            </span>
          </div>
          <button @click="runMaintenance('install_tools')" :disabled="!!maintRunning"
            class="text-xs border border-indigo-200 text-indigo-600 px-3 py-1.5 rounded-md hover:bg-indigo-50 disabled:opacity-50">
            {{ maintRunning === 'install_tools' ? 'Installing...' : 'Install Missing Tools' }}
          </button>
        </div>
        <div v-if="maintResult['install_tools']"
          class="bg-gray-900 rounded p-2 font-mono text-xs max-h-24 overflow-y-auto">
          <div v-if="maintResult['install_tools']?.error" class="text-red-400">{{ maintResult['install_tools'].error }}</div>
          <div v-else class="text-green-400">Tools installed successfully</div>
        </div>
      </div>
    </div>
  </div>
</template>
