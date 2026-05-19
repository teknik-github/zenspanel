<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { cronJobsApi } from '@/api/cronJobs'

const jobs = ref<any[]>([])
const loading = ref(false)
const loaded = ref(false)
const showModal = ref(false)
const editingJob = ref<any>(null)
const saving = ref(false)
const error = ref('')
const confirmDelete = ref<number | null>(null)

const form = ref({ expression: '', command: '', enabled: true })

// Common cron presets for the expression helper
const presets = [
  { label: 'Every minute',    value: '* * * * *' },
  { label: 'Every 5 minutes', value: '*/5 * * * *' },
  { label: 'Every hour',      value: '0 * * * *' },
  { label: 'Every day at midnight', value: '0 0 * * *' },
  { label: 'Every week (Sunday)', value: '0 0 * * 0' },
  { label: 'Every month',     value: '0 0 1 * *' },
]

onMounted(async () => {
  await fetchJobs()
  loaded.value = true
})

async function fetchJobs() {
  loading.value = true
  try {
    const res = await cronJobsApi.list()
    jobs.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingJob.value = null
  form.value = { expression: '0 * * * *', command: '', enabled: true }
  error.value = ''
  showModal.value = true
}

function openEdit(job: any) {
  editingJob.value = job
  form.value = { expression: job.expression, command: job.command, enabled: job.enabled }
  error.value = ''
  showModal.value = true
}

async function save() {
  error.value = ''
  saving.value = true
  try {
    if (editingJob.value) {
      await cronJobsApi.update(editingJob.value.id, {
        expression: form.value.expression,
        command: form.value.command,
        enabled: form.value.enabled,
      })
    } else {
      await cronJobsApi.create(form.value)
    }
    showModal.value = false
    await fetchJobs()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to save'
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(job: any) {
  await cronJobsApi.update(job.id, { enabled: !job.enabled })
  job.enabled = !job.enabled
}

async function deleteJob(id: number) {
  await cronJobsApi.delete(id)
  confirmDelete.value = null
  await fetchJobs()
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Cron Jobs</h1>
      <button @click="openCreate"
        class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700 flex items-center gap-1.5">
        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Cron Job
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading && !loaded" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div v-for="i in 3" :key="i" class="px-4 py-3 border-b border-gray-50 animate-pulse flex gap-4">
        <div class="h-4 bg-gray-100 rounded w-32"></div>
        <div class="h-4 bg-gray-100 rounded flex-1"></div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else-if="!jobs.length"
      class="flex flex-col items-center justify-center py-16 bg-white border border-gray-200 rounded-lg text-center">
      <div class="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mb-3">
        <svg class="w-6 h-6 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No cron jobs yet</p>
      <p class="text-xs text-gray-400 mt-1">Schedule recurring tasks for your applications.</p>
      <button @click="openCreate"
        class="mt-4 text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700">
        Add Cron Job
      </button>
    </div>

    <!-- Jobs table -->
    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[600px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Expression</th>
              <th class="text-left px-4 py-3 font-medium">Command</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in jobs" :key="job.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-mono text-indigo-700">{{ job.expression }}</td>
              <td class="px-4 py-3 text-gray-600 max-w-xs truncate" :title="job.command">{{ job.command }}</td>
              <td class="px-4 py-3">
                <button @click="toggleEnabled(job)"
                  class="px-2 py-0.5 rounded text-[10px] font-medium transition-colors"
                  :class="job.enabled ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'">
                  {{ job.enabled ? 'Active' : 'Disabled' }}
                </button>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <button @click="openEdit(job)" title="Edit"
                    class="text-indigo-600 hover:text-indigo-800 transition-colors">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                  </button>
                  <button @click="confirmDelete = job.id" title="Delete"
                    class="text-red-400 hover:text-red-600 transition-colors">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"/>
                      <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add/Edit modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">{{ editingJob ? 'Edit Cron Job' : 'Add Cron Job' }}</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Expression</label>
            <div class="flex gap-2 mb-2">
              <select @change="(e) => form.expression = (e.target as HTMLSelectElement).value"
                class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
                <option value="">— presets —</option>
                <option v-for="p in presets" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </div>
            <input v-model="form.expression" type="text" placeholder="* * * * *"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-1">minute hour day month weekday — or @daily, @hourly, @weekly, @monthly</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Command</label>
            <input v-model="form.command" type="text" placeholder="/usr/bin/php /home/user/artisan schedule:run"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-1">Use absolute paths. Shell metacharacters (; & | &gt; &lt; ` $ ) are not allowed.</p>
          </div>
          <label class="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" v-model="form.enabled" class="rounded border-gray-300 text-indigo-600" />
            <span class="text-xs text-gray-600">Enabled</span>
          </label>
          <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
          <div class="flex gap-2 pt-2">
            <button @click="save" :disabled="saving"
              class="flex-1 bg-indigo-600 text-white text-sm py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
            <button @click="showModal = false"
              class="flex-1 border border-gray-200 text-gray-600 text-sm py-2 rounded-md hover:bg-gray-50">
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete cron job?</h2>
        <p class="text-sm text-gray-500 mb-4">This will remove the job and update your crontab immediately.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteJob(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
