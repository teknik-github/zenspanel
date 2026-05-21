<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useToast } from '../notify'

const { success: toastSuccess, error: toastError } = useToast()

const accounts = ref<any[]>([])
const loaded = ref(false)
const showModal = ref(false)
const loading = ref(false)
const confirmDelete = ref<any | null>(null)

const form = ref({ ftp_username: '', password: '', home_dir: '' })

// FTP server info — shown to user so they know what to connect to
const ftpHost = window.location.hostname
const ftpPort = 21

async function fetchAccounts() {
  try {
    const res = await fetch('/api/v1/ftp', {
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
    })
    const data = await res.json()
    accounts.value = data.data || []
  } catch {
    accounts.value = []
  } finally {
    loaded.value = true
  }
}

onMounted(fetchAccounts)

async function create() {
  if (!form.value.ftp_username || !form.value.password) return
  loading.value = true
  try {
    const res = await fetch('/api/v1/ftp', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(form.value),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to create FTP account')
    toastSuccess('FTP account created')
    showModal.value = false
    form.value = { ftp_username: '', password: '', home_dir: '' }
    await fetchAccounts()
  } catch (e: any) {
    toastError(e.message)
  } finally {
    loading.value = false
  }
}

async function deleteAccount(id: number) {
  try {
    const res = await fetch(`/api/v1/ftp/${id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${localStorage.getItem('token')}` },
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'Failed to delete')
    }
    toastSuccess('FTP account deleted')
    confirmDelete.value = null
    await fetchAccounts()
  } catch (e: any) {
    toastError(e.message)
  }
}
</script>

<template>
  <div class="space-y-4 max-w-2xl">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">FTP Accounts</h1>
        <p class="text-xs text-gray-400 mt-0.5">Manage FTP access to your files</p>
      </div>
      <button @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add FTP Account
      </button>
    </div>

    <!-- Server info card -->
    <div class="bg-white border border-gray-200 rounded-lg p-4">
      <h2 class="text-sm font-semibold text-gray-800 mb-2">Connection Details</h2>
      <div class="grid grid-cols-2 gap-2 text-xs">
        <div class="flex justify-between bg-gray-50 rounded px-3 py-2">
          <span class="text-gray-500">Host</span>
          <span class="font-mono font-medium text-gray-700">{{ ftpHost }}</span>
        </div>
        <div class="flex justify-between bg-gray-50 rounded px-3 py-2">
          <span class="text-gray-500">Port</span>
          <span class="font-mono font-medium text-gray-700">{{ ftpPort }}</span>
        </div>
        <div class="flex justify-between bg-gray-50 rounded px-3 py-2 col-span-2">
          <span class="text-gray-500">Protocol</span>
          <span class="font-mono font-medium text-gray-700">FTP with TLS (FTPS) recommended</span>
        </div>
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 2" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!accounts.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-500 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 4.69 12 19.79 19.79 0 0 1 1.61 3.41 2 2 0 0 1 3.6 1.22h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L7.91 8.96a16 16 0 0 0 6.13 6.13l.96-.96a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0 1 22 16.92z"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No FTP accounts yet</p>
      <p class="text-xs text-gray-400 mt-1">Create an FTP account to upload files via FileZilla or similar clients</p>
      <button @click="showModal = true"
        class="mt-4 flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add FTP Account
      </button>
    </div>

    <!-- Accounts table -->
    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[480px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Username</th>
              <th class="text-left px-4 py-3 font-medium">Home Directory</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-left px-4 py-3 font-medium">Created</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in accounts" :key="a.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-mono font-medium text-gray-700">{{ a.ftp_username }}</td>
              <td class="px-4 py-3 font-mono text-gray-500 max-w-[180px] truncate" :title="a.home_dir">{{ a.home_dir }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="a.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
                  {{ a.enabled ? 'Active' : 'Disabled' }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-400">{{ new Date(a.created_at).toLocaleDateString() }}</td>
              <td class="px-4 py-3">
                <button @click="confirmDelete = a" title="Delete account"
                  class="text-red-400 hover:text-red-600 transition-colors">
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Add FTP Account</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">FTP Username</label>
            <input v-model="form.ftp_username" type="text" placeholder="e.g. alice_ftp"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-0.5">Lowercase letters, numbers, underscores. 3-64 chars.</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Password</label>
            <input v-model="form.password" type="password" placeholder="Min 8 characters"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Home Directory <span class="text-gray-400">(optional)</span></label>
            <input v-model="form.home_dir" type="text" placeholder="Leave empty for your home directory"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="create" :disabled="loading || !form.ftp_username || !form.password"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete FTP Account?</h2>
        <p class="text-sm text-gray-500 mb-4">
          Remove <span class="font-mono">{{ confirmDelete.ftp_username }}</span>? This will revoke FTP access immediately.
        </p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="deleteAccount(confirmDelete.id)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
