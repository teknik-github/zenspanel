<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { backupsApi } from '@/api/backups'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const backups = ref<any[]>([])
const showModal = ref(false)
const backupType = ref<'full' | 'db' | 'files'>('full')
const loading = ref(false)
const confirmRestore = ref<number | null>(null)
const confirmDelete = ref<number | null>(null)

const hasPending = computed(() => backups.value.some(b => ['pending', 'running', 'restoring'].includes(b.status)))

let pollInterval: ReturnType<typeof setInterval> | null = null

async function fetchBackups() {
  const res = await backupsApi.list()
  backups.value = res.data.data || []
}

onMounted(async () => {
  if (!auth.user?.backup_enabled) return
  await fetchBackups()
  pollInterval = setInterval(() => { if (hasPending.value) fetchBackups() }, 5000)
})

import { onUnmounted } from 'vue'
onUnmounted(() => { if (pollInterval) clearInterval(pollInterval) })

async function createBackup() {
  loading.value = true
  try {
    await backupsApi.create(backupType.value)
    showModal.value = false
    await fetchBackups()
  } finally {
    loading.value = false
  }
}

async function downloadBackup(id: number, filename: string) {
  const res = await backupsApi.download(id)
  const url = URL.createObjectURL(res.data)
  const a = document.createElement('a')
  a.href = url
  a.download = filename || `backup-${id}.tar.gz`
  a.click()
  URL.revokeObjectURL(url)
}

async function restoreBackup(id: number) {
  await backupsApi.restore(id)
  confirmRestore.value = null
  await fetchBackups()
}

async function deleteBackup(id: number) {
  await backupsApi.delete(id)
  confirmDelete.value = null
  await fetchBackups()
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
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Backups</h1>
      <button v-if="auth.user?.backup_enabled" @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Create Backup
      </button>
    </div>

    <div v-if="!auth.user?.backup_enabled"
      class="flex items-center justify-center bg-white border border-gray-200 rounded-lg py-16">
      <div class="text-center text-gray-400">
        <svg class="w-10 h-10 mx-auto mb-2 opacity-30" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
        </svg>
        <p class="text-sm">Backups are disabled for your account.</p>
      </div>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
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
            <td class="px-4 py-3 flex items-center gap-2">
              <button v-if="b.status === 'done'" @click="downloadBackup(b.id, b.file_path?.split('/').pop())"
                class="text-xs text-indigo-600 hover:text-indigo-800 border border-indigo-200 px-2 py-1 rounded">Download</button>
              <button v-if="b.status === 'done'" @click="confirmRestore = b.id"
                class="text-xs text-amber-600 hover:text-amber-800 border border-amber-200 px-2 py-1 rounded">Restore</button>
              <button @click="confirmDelete = b.id" class="text-red-400 hover:text-red-600">
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="!backups.length">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No backups yet.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
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
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="createBackup" :disabled="loading"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Restore -->
    <div v-if="confirmRestore" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Restore Backup?</h2>
        <p class="text-sm text-gray-500 mb-4">This will overwrite current files/databases with the backup.</p>
        <div class="flex gap-2">
          <button @click="confirmRestore = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="restoreBackup(confirmRestore!)"
            class="flex-1 bg-amber-600 text-white rounded-md py-2 text-sm hover:bg-amber-700">Restore</button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete Backup?</h2>
        <p class="text-sm text-gray-500 mb-4">This will permanently delete the backup file.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteBackup(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
