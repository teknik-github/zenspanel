<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useDatabasesStore } from '@/stores/databases'
import { databasesApi } from '@/api/databases'
import { useToast } from '../notify'

const databasesStore = useDatabasesStore()
const { error: toastError, success: toastSuccess } = useToast()
const showModal = ref(false)
const newDBName = ref('')
const newDBUser = ref('')
const newDBPassword = ref(generatePassword())
const loading = ref(false)
const confirmDelete = ref<number | null>(null)
const createdCreds = ref<{ db_name: string; db_user: string; db_password: string } | null>(null)
const loaded = ref(false)

// Reset password state
const resetResult = ref<{ db_user: string; new_password: string } | null>(null)
const resetting = ref<number | null>(null)

function generatePassword() {
  const chars = 'ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$'
  return Array.from({ length: 16 }, () => chars[Math.floor(Math.random() * chars.length)]).join('')
}

onMounted(async () => {
  await databasesStore.fetch()
  loaded.value = true
})

async function createDatabase() {
  loading.value = true
  try {
    await databasesApi.create(newDBName.value, newDBUser.value, newDBPassword.value)
    createdCreds.value = { db_name: newDBName.value, db_user: newDBUser.value, db_password: newDBPassword.value }
    showModal.value = false
    await databasesStore.fetch()
  } finally {
    loading.value = false
  }
}

async function openPHPMyAdmin(id: number) {
  // Launch endpoint resets the MySQL password, mints a one-time token,
  // and returns a redeem URL. We open the redeem URL in a new tab; that
  // page auto-submits a form into phpMyAdmin's cookie-auth login.
  try {
    const res = await databasesApi.launchPHPMyAdmin(id)
    if (res.data.url) {
      window.open(res.data.url, '_blank')
    }
  } catch (e: any) {
    toastError(e.response?.data?.error || 'Failed to open phpMyAdmin')
  }
}

async function deleteDatabase(id: number) {
  await databasesApi.drop(id)
  confirmDelete.value = null
  await databasesStore.fetch()
}

async function resetPassword(id: number) {
  resetting.value = id
  try {
    const res = await databasesApi.resetPassword(id)
    resetResult.value = res.data
  } catch (e: any) {
    toastError(e.response?.data?.error || 'Failed to reset password')
  } finally {
    resetting.value = null
  }
}

function copyToClipboard(text: string) {
  // navigator.clipboard is undefined in non-secure contexts (plain HTTP).
  // Fall back to the legacy execCommand-based copy so this works during
  // local testing before TLS is set up.
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text)
    return
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  try { document.execCommand('copy') } catch {}
  document.body.removeChild(ta)
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Databases</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">MySQL databases for your applications</p>
      </div>
      <button @click="showModal = true; newDBPassword = generatePassword()"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Database
      </button>
    </div>

    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <div v-else-if="!databasesStore.databases.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-blue-50 text-blue-500 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No databases yet</p>
      <p class="text-xs text-gray-400 mt-1">Create your first MySQL database for your application</p>
      <button @click="showModal = true; newDBPassword = generatePassword()"
        class="mt-4 flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Database
      </button>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[480px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Database</th>
              <th class="text-left px-4 py-3 font-medium">User</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="db in databasesStore.databases" :key="db.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700">{{ db.db_name }}</td>
              <td class="px-4 py-3 text-gray-500">{{ db.db_user }}</td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <button @click="openPHPMyAdmin(db.id)"
                    title="Open in phpMyAdmin"
                    class="text-xs text-indigo-600 hover:text-indigo-800 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50 transition-colors">
                    phpMyAdmin
                  </button>
                  <button @click="resetPassword(db.id)" :disabled="resetting === db.id"
                    title="Reset database password"
                    class="text-xs text-amber-600 hover:text-amber-800 border border-amber-200 px-2 py-1 rounded hover:bg-amber-50 transition-colors disabled:opacity-50">
                    {{ resetting === db.id ? '...' : 'Reset PW' }}
                  </button>
                  <button @click="confirmDelete = db.id" title="Drop database" class="text-red-400 hover:text-red-600 transition-colors">
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
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Create Database</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Database Name</label>
            <input v-model="newDBName" type="text" placeholder="myapp_db"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Database User</label>
            <input v-model="newDBUser" type="text" placeholder="myapp_user"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Password (auto-generated)</label>
            <div class="flex gap-2">
              <input :value="newDBPassword" readonly
                class="flex-1 border border-gray-200 rounded-md px-3 py-2 text-sm bg-gray-50 font-mono" />
              <button @click="copyToClipboard(newDBPassword)"
                class="border border-gray-200 px-3 py-2 rounded-md text-xs text-gray-600 hover:bg-gray-50">Copy</button>
            </div>
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="createDatabase" :disabled="loading || !newDBName || !newDBUser"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Show credentials after create -->
    <div v-if="createdCreds" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-1">Database Created</h2>
        <p class="text-xs text-amber-600 mb-4">Save these credentials — the password will not be shown again.</p>
        <div class="space-y-2 text-xs font-mono bg-gray-50 rounded-lg p-3">
          <div><span class="text-gray-400">Database:</span> {{ createdCreds.db_name }}</div>
          <div><span class="text-gray-400">User:</span> {{ createdCreds.db_user }}</div>
          <div><span class="text-gray-400">Password:</span> {{ createdCreds.db_password }}</div>
        </div>
        <button @click="createdCreds = null"
          class="w-full mt-4 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700">Done</button>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Drop Database?</h2>
        <p class="text-sm text-gray-500 mb-4">This will permanently delete the database and MySQL user.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteDatabase(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Drop</button>
        </div>
      </div>
    </div>

    <!-- Reset password result — one-time display (V56) -->
    <div v-if="resetResult" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-1">New Database Password</h2>
        <p class="text-xs text-amber-600 bg-amber-50 border border-amber-100 rounded px-3 py-2 mb-4">
          Copy this password now — it will not be shown again.
        </p>
        <div class="space-y-2 text-xs mb-4">
          <div>
            <span class="text-gray-500">DB User:</span>
            <span class="font-mono ml-2 text-gray-800">{{ resetResult.db_user }}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-gray-500">Password:</span>
            <span class="font-mono text-gray-800 bg-gray-50 border border-gray-200 rounded px-2 py-1 flex-1 select-all">
              {{ resetResult.new_password }}
            </span>
            <button @click="copyToClipboard(resetResult!.new_password)"
              class="text-indigo-600 hover:text-indigo-800 border border-indigo-200 px-2 py-1 rounded text-xs">
              Copy
            </button>
          </div>
        </div>
        <button @click="resetResult = null"
          class="w-full bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700">
          I've saved the password
        </button>
      </div>
    </div>
  </div>
</template>
