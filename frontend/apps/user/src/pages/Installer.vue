<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { installerApi } from '@/api/installer'

const domains = ref<any[]>([])
const apps = ref<any[]>([])
const selectedApp = ref<any>(null)
const selectedDomain = ref<number | null>(null)
const dbName = ref('')
const dbUser = ref('')
const dbPass = ref('')
const overwrite = ref(false)
const installing = ref(false)
const jobId = ref('')
const jobStatus = ref<any>(null)
const error = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  const [domainsRes, appsRes] = await Promise.all([
    domainsApi.list(),
    installerApi.listApps(),
  ])
  domains.value = domainsRes.data.data || []
  apps.value = appsRes.data.data || []
  if (domains.value.length) selectedDomain.value = domains.value[0].id
})

onUnmounted(() => stopPoll())

function selectApp(app: any) {
  selectedApp.value = app
  error.value = ''
  jobId.value = ''
  jobStatus.value = null
}

async function install() {
  if (!selectedApp.value || !selectedDomain.value) return
  error.value = ''
  installing.value = true
  jobStatus.value = null
  try {
    const payload: any = {
      app_id: selectedApp.value.id,
      domain_id: selectedDomain.value,
      overwrite: overwrite.value,
    }
    if (needsDB(selectedApp.value.id)) {
      payload.db_name = dbName.value
      payload.db_user = dbUser.value
      payload.db_pass = dbPass.value
    }
    const res = await installerApi.install(payload)
    jobId.value = res.data.job_id
    startPoll()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Installation failed'
    installing.value = false
  }
}

function startPoll() {
  stopPoll()
  pollTimer = setInterval(async () => {
    try {
      const res = await installerApi.status(jobId.value)
      jobStatus.value = res.data
      if (res.data.done) {
        stopPoll()
        installing.value = false
      }
    } catch {
      stopPoll()
      installing.value = false
    }
  }, 2000)
}

function stopPoll() {
  if (pollTimer) { clearInterval(pollTimer); pollTimer = null }
}

function reset() {
  selectedApp.value = null
  jobId.value = ''
  jobStatus.value = null
  error.value = ''
  dbName.value = ''
  dbUser.value = ''
  dbPass.value = ''
  overwrite.value = false
}

const appIcons: Record<string, string> = {
  wordpress:   `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><circle cx="12" cy="12" r="10"/><path d="M2 12h4m12 0h4M12 2v4m0 12v4"/><circle cx="12" cy="12" r="3"/></svg>`,
  joomla:      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>`,
  drupal:      `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M12 2c-2 2-6 4-6 9a6 6 0 0 0 12 0c0-5-4-7-6-9z"/><path d="M9 17c0 2 1.5 3 3 3s3-1 3-3"/></svg>`,
  prestashop:  `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M6 2h12l2 6-8 3-8-3 2-6z"/><path d="M3 8l9 4 9-4"/><path d="M12 12v10"/><path d="M5 19h14"/></svg>`,
  codeigniter: `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M12 2a7 7 0 0 1 7 7c0 5-7 13-7 13S5 14 5 9a7 7 0 0 1 7-7z"/><circle cx="12" cy="9" r="2.5"/></svg>`,
  laravel:     `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>`,
  html:        `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="15" y2="17"/></svg>`,
  default:     `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="w-8 h-8"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/></svg>`,
}

function needsDB(appId: string) {
  const app = apps.value.find((a: any) => a.id === appId)
  return app?.requires_db ?? (appId === 'wordpress' || appId === 'laravel' || appId === 'joomla' || appId === 'drupal' || appId === 'prestashop')
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">Website Installer</h1>

    <!-- App selection -->
    <div v-if="!selectedApp" class="grid grid-cols-1 sm:grid-cols-3 lg:grid-cols-4 gap-3">
      <button v-for="app in apps" :key="app.id" @click="selectApp(app)"
        class="bg-white border border-gray-200 rounded-lg p-4 text-left hover:border-indigo-400 hover:shadow-sm transition-all">
        <div class="text-indigo-500 mb-2" v-html="appIcons[app.id] || appIcons.default"></div>
        <div class="text-sm font-semibold text-gray-800">{{ app.name }}</div>
        <div class="text-xs text-gray-400 mt-0.5" v-if="app.version !== '—'">v{{ app.version }}</div>
        <div class="text-xs text-gray-500 mt-2">{{ app.description }}</div>
        <div class="mt-2 flex gap-1">
          <span v-if="app.requires_db" class="text-[10px] bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded">DB required</span>
          <span v-else class="text-[10px] bg-gray-50 text-gray-400 px-1.5 py-0.5 rounded">No DB</span>
        </div>
      </button>
    </div>

    <!-- Install form -->
    <div v-else-if="!jobId" class="bg-white border border-gray-200 rounded-lg p-5 space-y-4 max-w-lg">
      <div class="flex items-center gap-3">
        <div class="text-indigo-500 flex-shrink-0" v-html="appIcons[selectedApp.id] || appIcons.default"></div>
        <div>
          <h2 class="text-sm font-semibold text-gray-800">Install {{ selectedApp.name }}</h2>
          <p class="text-xs text-gray-400">{{ selectedApp.description }}</p>
        </div>
      </div>

      <div>
        <label class="block text-xs font-medium text-gray-600 mb-1">Install to domain</label>
        <select v-model="selectedDomain"
          class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
          <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.domain }}</option>
        </select>
      </div>

      <template v-if="needsDB(selectedApp.id)">
        <div class="border-t border-gray-100 pt-3">
          <p class="text-xs font-medium text-gray-600 mb-2">Database (must exist)</p>
          <div class="space-y-2">
            <input v-model="dbName" type="text" placeholder="Database name"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <input v-model="dbUser" type="text" placeholder="Database user"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <input v-model="dbPass" type="password" placeholder="Database password"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
        </div>
      </template>

      <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
        <input type="checkbox" v-model="overwrite" class="rounded border-gray-300 text-indigo-600" />
        Overwrite existing files in document root
      </label>

      <p v-if="error" class="text-xs text-red-600">{{ error }}</p>

      <div class="flex gap-2 pt-1">
        <button @click="install" :disabled="installing"
          class="flex-1 bg-indigo-600 text-white text-sm py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {{ installing ? 'Installing...' : 'Install' }}
        </button>
        <button @click="reset"
          class="flex-1 border border-gray-200 text-gray-600 text-sm py-2 rounded-md hover:bg-gray-50">
          Cancel
        </button>
      </div>
    </div>

    <!-- Progress log -->
    <div v-else class="space-y-3 max-w-lg">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-semibold text-gray-800">
            Installing {{ selectedApp.name }}
          </span>
          <span class="text-xs px-2 py-0.5 rounded font-medium"
            :class="{
              'bg-indigo-100 text-indigo-700': !jobStatus?.done && !jobStatus?.error,
              'bg-green-100 text-green-700':   jobStatus?.done && !jobStatus?.error,
              'bg-red-100 text-red-700':        jobStatus?.error,
            }">
            {{ jobStatus?.phase || 'starting' }}
          </span>
        </div>

        <!-- Progress log terminal -->
        <div class="bg-gray-900 rounded p-3 font-mono text-xs text-gray-300 max-h-64 overflow-y-auto space-y-0.5">
          <div v-for="(line, i) in (jobStatus?.log || [])" :key="i"
            :class="line.includes('ERROR') ? 'text-red-400' : line.includes('Phase:') ? 'text-indigo-400' : 'text-gray-300'">
            {{ line }}
          </div>
          <div v-if="!jobStatus?.log?.length" class="text-gray-500 animate-pulse">Waiting for agent...</div>
        </div>
      </div>

      <div v-if="jobStatus?.done && !jobStatus?.error"
        class="bg-green-50 border border-green-200 rounded-lg p-4 text-center">
        <svg class="w-8 h-8 text-green-500 mx-auto mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>
        </svg>
        <p class="text-sm font-semibold text-green-800">Installation complete!</p>
        <p class="text-xs text-green-600 mt-1">
          Visit <a :href="'http://' + domains.find(d => d.id === selectedDomain)?.domain"
            target="_blank" class="underline">
            {{ domains.find(d => d.id === selectedDomain)?.domain }}
          </a> to complete setup.
        </p>
        <button @click="reset" class="mt-3 text-xs text-indigo-600 hover:underline">Install another app</button>
      </div>

      <div v-if="jobStatus?.error"
        class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-sm font-semibold text-red-800">Installation failed</p>
        <p class="text-xs text-red-600 mt-1">{{ jobStatus.error }}</p>
        <button @click="reset" class="mt-3 text-xs text-indigo-600 hover:underline">Try again</button>
      </div>
    </div>
  </div>
</template>
