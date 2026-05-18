<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { backupsApi } from '@/api/backups'

const backups = ref<any[]>([])
const error = ref('')
const confirmRestore = ref<any | null>(null)
const restoring = ref(false)
const actionError = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null
const hasInflight = computed(() =>
  backups.value.some(b => ['pending', 'running', 'restoring'].includes(b.status)),
)

async function load() {
  try {
    const res = await backupsApi.list()
    backups.value = res.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  }
}

async function doRestore() {
  if (!confirmRestore.value) return
  restoring.value = true
  actionError.value = ''
  try {
    await backupsApi.restore(confirmRestore.value.id)
    confirmRestore.value = null
    await load()
  } catch (e: any) {
    actionError.value = e.response?.data?.error || 'Restore failed to start'
  } finally {
    restoring.value = false
  }
}

function fmtSize(n?: number) {
  if (!n) return '—'
  if (n < 1048576) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1073741824) return (n / 1048576).toFixed(1) + ' MB'
  return (n / 1073741824).toFixed(2) + ' GB'
}

function fmtDate(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleString()
}

function statusClass(s: string) {
  if (s === 'done') return 'bg-green-100 text-green-700'
  if (s === 'failed' || s === 'restore_failed') return 'bg-red-100 text-red-600'
  if (s === 'restoring') return 'bg-indigo-100 text-indigo-700'
  if (s === 'running' || s === 'pending') return 'bg-amber-100 text-amber-700'
  return 'bg-gray-100 text-gray-500'
}

onMounted(async () => {
  await load()
  // Poll every 5s while there's a running/restoring row so the admin sees
  // status transitions without a manual refresh.
  pollTimer = setInterval(() => {
    if (hasInflight.value) load()
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Backups</h1>
      <span class="text-xs text-gray-400">{{ backups.length }} backup{{ backups.length !== 1 ? 's' : '' }}</span>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">User</th>
            <th class="text-left px-4 py-3 font-medium">Type</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
            <th class="text-left px-4 py-3 font-medium">Size</th>
            <th class="text-left px-4 py-3 font-medium">Created</th>
            <th class="text-left px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in backups" :key="b.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 text-gray-500">{{ b.user_id }}</td>
            <td class="px-4 py-3 font-medium text-gray-700">{{ b.type }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(b.status)">{{ b.status }}</span>
            </td>
            <td class="px-4 py-3 text-gray-500">{{ fmtSize(b.size_bytes?.Int64 || b.size_bytes) }}</td>
            <td class="px-4 py-3 text-gray-500">{{ fmtDate(b.created_at) }}</td>
            <td class="px-4 py-3">
              <button v-if="b.status === 'done'" @click="confirmRestore = b"
                class="text-xs text-indigo-600 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50">
                Restore
              </button>
            </td>
          </tr>
          <tr v-if="!backups.length">
            <td colspan="6" class="px-4 py-8 text-center text-gray-400">No backups yet</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Confirm Restore -->
    <div v-if="confirmRestore" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Restore "{{ confirmRestore.type }}" backup?</h2>
        <p class="text-sm text-gray-500 mb-2">
          User <span class="font-medium text-gray-700">#{{ confirmRestore.user_id }}</span> will have:
        </p>
        <ul class="text-xs text-gray-500 list-disc list-inside space-y-0.5 mb-3">
          <li v-if="confirmRestore.type === 'files' || confirmRestore.type === 'full'">
            Home directory wiped and restored from this archive
          </li>
          <li v-if="confirmRestore.type === 'db' || confirmRestore.type === 'full'">
            All databases overwritten with the dump in this archive
          </li>
        </ul>
        <p class="text-xs text-amber-700 bg-amber-50 border border-amber-100 rounded px-2 py-1.5 mb-3">
          This is destructive and cannot be undone. Active sessions and live data will be replaced.
        </p>
        <p v-if="actionError" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5 mb-3">
          {{ actionError }}
        </p>
        <div class="flex gap-2">
          <button @click="confirmRestore = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="doRestore" :disabled="restoring"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700 disabled:opacity-50">
            {{ restoring ? 'Starting...' : 'Restore' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
