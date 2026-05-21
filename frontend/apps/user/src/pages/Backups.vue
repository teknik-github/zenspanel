<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { backupsApi } from '@/api/backups'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '../notify'

const auth = useAuthStore()
const { error: toastError } = useToast()
const backups = ref<any[]>([])
const showModal = ref(false)
const backupType = ref<'full' | 'db' | 'files'>('full')
const loading = ref(false)
const loaded = ref(false)
const confirmRestore = ref<number | null>(null)
const confirmDelete = ref<number | null>(null)

const hasPending = computed(() => backups.value.some(b => ['pending', 'running', 'restoring'].includes(b.status)))

let pollInterval: ReturnType<typeof setInterval> | null = null

async function fetchBackups() {
  const res = await backupsApi.list()
  backups.value = res.data.data || []
}

onMounted(async () => {
  if (!auth.user?.backup_enabled) {
    loaded.value = true
    return
  }
  try {
    await fetchBackups()
  } finally {
    loaded.value = true
  }
  pollInterval = setInterval(() => { if (hasPending.value) fetchBackups() }, 5000)
})

onUnmounted(() => { if (pollInterval) clearInterval(pollInterval) })

async function createBackup() {
  loading.value = true
  try {
    await backupsApi.create(backupType.value)
    showModal.value = false
    await fetchBackups()
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to create backup')
  } finally {
    loading.value = false
  }
}

async function downloadBackup(id: number, filename: string) {
  try {
    const res = await backupsApi.download(id)
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = filename || `backup-${id}.tar.gz`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to download backup')
  }
}

async function restoreBackup(id: number) {
  try {
    await backupsApi.restore(id)
    confirmRestore.value = null
    await fetchBackups()
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to restore backup')
  }
}

async function deleteBackup(id: number) {
  try {
    await backupsApi.delete(id)
    confirmDelete.value = null
    await fetchBackups()
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to delete backup')
  }
}

// A backup row that's in flight on the server can't be safely removed —
// the worker still has open file handles and will recreate the file mid-
// delete. Surface this in the UI as a disabled trash icon.
function isBusy(status: string) {
  return ['pending', 'running', 'restoring'].includes(status)
}

function statusClass(status: string) {
  return {
    pending: 'bg-yellow-100 text-yellow-700',
    running: 'bg-blue-100 text-blue-700',
    restoring: 'bg-indigo-100 text-indigo-700',
    done: 'bg-green-100 text-green-700',
    failed: 'bg-red-100 text-red-600',
    restore_failed: 'bg-red-100 text-red-600',
  }[status] || 'bg-gray-100 text-gray-500'
}

function formatSize(bytes: number) {
  if (!bytes) return '—'
  if (bytes > 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  return (bytes / 1048576).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Backups</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">Snapshot of your files and databases</p>
      </div>
      <button v-if="auth.user?.backup_enabled" @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Create Backup
      </button>
    </div>

    <div v-if="!auth.user?.backup_enabled"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-16 px-4 text-center">
      <div class="w-12 h-12 rounded-full bg-gray-100 text-gray-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">Backups are disabled</p>
      <p class="text-xs text-gray-400 mt-1">Contact your administrator to enable this feature</p>
    </div>

    <div v-else-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <div v-else-if="!backups.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-500 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No backups yet</p>
      <p class="text-xs text-gray-400 mt-1">Create your first backup to protect your data</p>
      <button @click="showModal = true"
        class="mt-4 flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Create Backup
      </button>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[720px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Type</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-left px-4 py-3 font-medium">Size</th>
              <th class="text-left px-4 py-3 font-medium">Created</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in backups" :key="b.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium bg-indigo-100 text-indigo-700">{{ b.type }}</span>
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(b.status)">{{ b.status }}</span>
              </td>
              <td class="px-4 py-3 text-gray-500">{{ formatSize(b.size_bytes) }}</td>
              <td class="px-4 py-3 text-gray-500">{{ new Date(b.created_at).toLocaleString() }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <button v-if="b.status === 'done'" @click="downloadBackup(b.id, b.file_path?.split('/').pop())"
                    title="Download backup"
                    class="text-xs text-indigo-600 hover:text-indigo-800 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50 transition-colors">Download</button>
                  <button v-if="b.status === 'done'" @click="confirmRestore = b.id"
                    title="Restore from this backup"
                    class="text-xs text-amber-600 hover:text-amber-800 border border-amber-200 px-2 py-1 rounded hover:bg-amber-50 transition-colors">Restore</button>
                  <button @click="confirmDelete = b.id" :disabled="isBusy(b.status)"
                    :title="isBusy(b.status) ? 'Cannot delete while in progress' : 'Delete backup'"
                    class="text-red-400 hover:text-red-600 disabled:opacity-30 disabled:cursor-not-allowed transition-colors">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Create Backup</h2>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Backup Type</label>
          <select v-model="backupType"
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
            <option value="full">Full (files + databases)</option>
            <option value="db">Databases only</option>
            <option value="files">Files only</option>
          </select>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="createBackup" :disabled="loading"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Restore -->
    <div v-if="confirmRestore" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Restore Backup?</h2>
        <p class="text-sm text-gray-500 mb-4">This will overwrite current files/databases with the backup.</p>
        <div class="flex gap-2">
          <button @click="confirmRestore = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="restoreBackup(confirmRestore!)"
            class="flex-1 bg-amber-600 text-white rounded-md py-2 text-sm hover:bg-amber-700">Restore</button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete Backup?</h2>
        <p class="text-sm text-gray-500 mb-4">This will permanently delete the backup file.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="deleteBackup(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
