<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { backupsApi } from '@/api/backups'
import { usersApi } from '@/api/users'

const backups = ref<any[]>([])
const users = ref<Map<number, string>>(new Map())
const loaded = ref(false)
const error = ref('')
const search = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const confirmRestore = ref<any | null>(null)
const restoring = ref(false)
const actionError = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null
const hasInflight = computed(() =>
  backups.value.some(b => ['pending', 'running', 'restoring'].includes(b.status)),
)

const filtered = computed(() => {
  return backups.value.filter(b => {
    if (typeFilter.value && b.type !== typeFilter.value) return false
    if (statusFilter.value && b.status !== statusFilter.value) return false
    const username = users.value.get(b.user_id) || ''
    if (search.value && !username.toLowerCase().includes(search.value.toLowerCase())) return false
    return true
  })
})

async function load() {
  try {
    const [bRes, uRes] = await Promise.all([backupsApi.list(), usersApi.list()])
    backups.value = bRes.data.data || []
    const uMap = new Map<number, string>()
    for (const u of (uRes.data.data || [])) uMap.set(u.id, u.username)
    users.value = uMap
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  } finally {
    loaded.value = true
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
  pollTimer = setInterval(() => { if (hasInflight.value) load() }, 5000)
})
onUnmounted(() => { if (pollTimer) clearInterval(pollTimer) })
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Backups</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">All backups across all users</p>
      </div>
      <span v-if="loaded" class="text-xs text-gray-400">{{ filtered.length }} of {{ backups.length }}</span>
    </div>

    <div class="flex gap-2 flex-wrap">
      <div class="relative">
        <svg class="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input v-model="search" type="text" placeholder="Search user..."
          class="border border-gray-200 rounded-md pl-8 pr-3 py-2 text-xs w-44 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      </div>
      <select v-model="typeFilter"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Types</option>
        <option value="full">Full</option>
        <option value="files">Files</option>
        <option value="db">Database</option>
        <option value="domain">Domain</option>
      </select>
      <select v-model="statusFilter"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Status</option>
        <option value="done">Done</option>
        <option value="running">Running</option>
        <option value="pending">Pending</option>
        <option value="failed">Failed</option>
      </select>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-3 py-2">{{ error }}</p>

    <!-- Skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 4" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!backups.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No backups yet</p>
      <p class="text-xs text-gray-400 mt-1">Backups will appear here once users create them</p>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[640px]">
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
            <tr v-if="!filtered.length">
              <td colspan="6" class="px-4 py-8 text-center text-gray-400">No backups match your filter</td>
            </tr>
            <tr v-for="b in filtered" :key="b.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 text-gray-500">
                <router-link :to="`/users/${b.user_id}`" class="hover:text-indigo-600 transition-colors">
                  {{ users.get(b.user_id) || `#${b.user_id}` }}
                </router-link>
              </td>
              <td class="px-4 py-3 font-medium text-gray-700 capitalize">{{ b.type }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(b.status)">
                  {{ b.status }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-500">{{ fmtSize(b.size_bytes?.Int64 || b.size_bytes) }}</td>
              <td class="px-4 py-3 text-gray-400">{{ fmtDate(b.created_at) }}</td>
              <td class="px-4 py-3">
                <button v-if="b.status === 'done'" @click="confirmRestore = b"
                  class="text-xs text-indigo-600 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50 transition-colors">
                  Restore
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Confirm Restore -->
    <div v-if="confirmRestore" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Restore "{{ confirmRestore.type }}" backup?</h2>
        <p class="text-sm text-gray-500 mb-2">
          User <span class="font-medium text-gray-700">{{ users.get(confirmRestore.user_id) || `#${confirmRestore.user_id}` }}</span> will have:
        </p>
        <ul class="text-xs text-gray-500 list-disc list-inside space-y-0.5 mb-3">
          <li v-if="confirmRestore.type === 'files' || confirmRestore.type === 'full'">Home directory wiped and restored from this archive</li>
          <li v-if="confirmRestore.type === 'db' || confirmRestore.type === 'full'">All databases overwritten with the dump in this archive</li>
        </ul>
        <p class="text-xs text-amber-700 bg-amber-50 border border-amber-100 rounded px-2 py-1.5 mb-3">
          This is destructive and cannot be undone.
        </p>
        <p v-if="actionError" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5 mb-3">{{ actionError }}</p>
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
