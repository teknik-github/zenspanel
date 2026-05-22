<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ipAllowlistApi } from '@/api/ipAllowlist'
import { useToast, useConfirm } from '@/notify'

const { success: toastSuccess, error: toastError } = useToast()
const { confirm } = useConfirm()

const entries = ref<any[]>([])
const loaded = ref(false)
const showModal = ref(false)
const loading = ref(false)
const form = ref({ ip_cidr: '', note: '' })

// Detect current admin IP to pre-fill
const currentIP = ref('')

onMounted(async () => {
  try {
    const res = await ipAllowlistApi.list()
    entries.value = res.data.data || []
  } finally {
    loaded.value = true
  }
  // Fetch current IP via public API
  try {
    const r = await fetch('https://api.ipify.org?format=json')
    const d = await r.json()
    currentIP.value = d.ip || ''
  } catch { /* ignore */ }
})

async function create() {
  if (!form.value.ip_cidr) return
  loading.value = true
  try {
    await ipAllowlistApi.create(form.value.ip_cidr, form.value.note)
    toastSuccess(`${form.value.ip_cidr} added to allowlist`)
    showModal.value = false
    form.value = { ip_cidr: '', note: '' }
    const res = await ipAllowlistApi.list()
    entries.value = res.data.data || []
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to add IP')
  } finally {
    loading.value = false
  }
}

async function addCurrentIP() {
  if (!currentIP.value) return
  form.value.ip_cidr = currentIP.value
  form.value.note = 'My IP (auto-detected)'
  showModal.value = true
}

async function remove(entry: any) {
  const ok = await confirm(
    `Remove ${entry.ip_cidr} from allowlist? If this is your only allowed IP and the list is not empty, you will lose admin access.`
  )
  if (!ok) return
  try {
    await ipAllowlistApi.delete(entry.id)
    toastSuccess(`${entry.ip_cidr} removed`)
    const res = await ipAllowlistApi.list()
    entries.value = res.data.data || []
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to remove IP')
  }
}
</script>

<template>
  <div class="space-y-4 max-w-2xl">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Admin IP Allowlist</h1>
        <p class="text-xs text-gray-400 mt-0.5">Restrict access to the admin panel by IP address or CIDR range</p>
      </div>
      <button @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add IP
      </button>
    </div>

    <!-- Info banner -->
    <div class="bg-amber-50 border border-amber-200 rounded-lg px-4 py-3 flex gap-3">
      <svg class="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
      </svg>
      <div class="text-xs text-amber-800 space-y-1">
        <p class="font-medium">Empty list = allow all IPs (default)</p>
        <p>Once you add an IP, only listed IPs can access <span class="font-mono">/admin/</span>. Make sure to add your own IP first before enabling restrictions.</p>
        <p v-if="currentIP">Your current IP: <span class="font-mono font-medium">{{ currentIP }}</span>
          <button @click="addCurrentIP" class="ml-2 text-indigo-600 hover:underline">Add this IP</button>
        </p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 2" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!entries.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-10 text-center px-4">
      <div class="w-10 h-10 rounded-full bg-green-50 text-green-500 flex items-center justify-center mb-3">
        <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No restrictions</p>
      <p class="text-xs text-gray-400 mt-1">All IPs can access the admin panel. Add an IP to start restricting access.</p>
    </div>

    <!-- List -->
    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-2.5 bg-red-50 border-b border-red-100 flex items-center gap-2">
        <svg class="w-3.5 h-3.5 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
        <p class="text-xs text-red-700 font-medium">Restrictions active — only listed IPs can access /admin/</p>
      </div>
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-2.5 font-medium">IP / CIDR</th>
            <th class="text-left px-4 py-2.5 font-medium">Note</th>
            <th class="text-left px-4 py-2.5 font-medium">Added</th>
            <th class="px-4 py-2.5"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in entries" :key="e.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-2.5 font-mono font-medium text-gray-800">
              {{ e.ip_cidr }}
              <span v-if="currentIP && e.ip_cidr === currentIP"
                class="ml-2 text-[10px] bg-indigo-100 text-indigo-600 px-1.5 py-0.5 rounded">You</span>
            </td>
            <td class="px-4 py-2.5 text-gray-500">{{ e.note || '—' }}</td>
            <td class="px-4 py-2.5 text-gray-400">{{ new Date(e.created_at).toLocaleDateString() }}</td>
            <td class="px-4 py-2.5 text-right">
              <button @click="remove(e)" class="text-red-400 hover:text-red-600 transition-colors">
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                </svg>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add modal -->
    <Transition name="modal">
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="modal-panel bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Add IP / CIDR</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">IP Address or CIDR</label>
            <input v-model="form.ip_cidr" type="text" placeholder="e.g. 103.150.92.61 or 192.168.1.0/24"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Note <span class="text-gray-400">(optional)</span></label>
            <input v-model="form.note" type="text" placeholder="e.g. Office, Home, VPN"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false; form = { ip_cidr: '', note: '' }"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="create" :disabled="loading || !form.ip_cidr"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Adding...' : 'Add' }}
          </button>
        </div>
      </div>
    </div>
    </Transition>
  </div>
</template>
