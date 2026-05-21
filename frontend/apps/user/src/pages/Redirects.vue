<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { redirectsApi } from '@/api/redirects'

const domains = ref<any[]>([])
const selectedDomain = ref<number | null>(null)
const redirects = ref<any[]>([])
const loading = ref(false)
const showModal = ref(false)
const editingRedirect = ref<any>(null)
const saving = ref(false)
const error = ref('')
const confirmDelete = ref<number | null>(null)

const form = ref({ source_path: '', dest_url: '', type: '301', enabled: true })

onMounted(async () => {
  const res = await domainsApi.list()
  domains.value = res.data.data || []
  if (domains.value.length) {
    selectedDomain.value = domains.value[0].id
    await fetchRedirects()
  }
})

async function fetchRedirects() {
  if (!selectedDomain.value) return
  loading.value = true
  try {
    const res = await redirectsApi.list(selectedDomain.value)
    redirects.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function changeDomain() {
  await fetchRedirects()
}

function openCreate() {
  editingRedirect.value = null
  form.value = { source_path: '/', dest_url: '', type: '301', enabled: true }
  error.value = ''
  showModal.value = true
}

function openEdit(r: any) {
  editingRedirect.value = r
  form.value = { source_path: r.source_path, dest_url: r.dest_url, type: r.type, enabled: r.enabled }
  error.value = ''
  showModal.value = true
}

async function save() {
  if (!selectedDomain.value) return
  error.value = ''
  saving.value = true
  try {
    if (editingRedirect.value) {
      await redirectsApi.update(selectedDomain.value, editingRedirect.value.id, form.value)
    } else {
      await redirectsApi.create(selectedDomain.value, form.value)
    }
    showModal.value = false
    await fetchRedirects()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to save'
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(r: any) {
  if (!selectedDomain.value) return
  await redirectsApi.update(selectedDomain.value, r.id, { enabled: !r.enabled })
  r.enabled = !r.enabled
}

async function deleteRedirect(id: number) {
  if (!selectedDomain.value) return
  await redirectsApi.delete(selectedDomain.value, id)
  confirmDelete.value = null
  await fetchRedirects()
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between flex-wrap gap-2">
      <h1 class="text-lg font-semibold text-gray-800">Redirect Manager</h1>
      <div class="flex items-center gap-2">
        <select v-model="selectedDomain" @change="changeDomain"
          class="border border-gray-200 rounded-md px-2 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
          <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.domain }}</option>
        </select>
        <button @click="openCreate"
          class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700 flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Add Redirect
        </button>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="!loading && !redirects.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center">
      <svg class="w-8 h-8 text-gray-300 mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="17 1 21 5 17 9"/><path d="M3 11V9a4 4 0 0 1 4-4h14"/><polyline points="7 23 3 19 7 15"/><path d="M21 13v2a4 4 0 0 1-4 4H3"/>
      </svg>
      <p class="text-sm font-medium text-gray-700">No redirects yet</p>
      <p class="text-xs text-gray-400 mt-1">Add 301/302 redirects for this domain.</p>
    </div>

    <!-- Redirects table -->
    <div v-else-if="redirects.length" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[600px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Source Path</th>
              <th class="text-left px-4 py-3 font-medium">Destination</th>
              <th class="text-left px-4 py-3 font-medium">Type</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in redirects" :key="r.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-mono text-gray-700">{{ r.source_path }}</td>
              <td class="px-4 py-3 text-gray-500 max-w-xs truncate" :title="r.dest_url">{{ r.dest_url }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="r.type === '301' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700'">
                  {{ r.type }}
                </span>
              </td>
              <td class="px-4 py-3">
                <button @click="toggleEnabled(r)"
                  class="px-2 py-0.5 rounded text-[10px] font-medium transition-colors"
                  :class="r.enabled ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'">
                  {{ r.enabled ? 'Active' : 'Disabled' }}
                </button>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <button @click="openEdit(r)" class="text-indigo-600 hover:text-indigo-800">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                    </svg>
                  </button>
                  <button @click="confirmDelete = r.id" class="text-red-400 hover:text-red-600">
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

    <!-- Add/Edit modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">{{ editingRedirect ? 'Edit' : 'Add' }} Redirect</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Source Path</label>
            <input v-model="form.source_path" type="text" placeholder="/old-page"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-0.5">Must start with /</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Destination URL</label>
            <input v-model="form.dest_url" type="text" placeholder="https://example.com/new-page"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Type</label>
              <select v-model="form.type"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
                <option value="301">301 — Permanent</option>
                <option value="302">302 — Temporary</option>
              </select>
            </div>
            <div class="flex items-end pb-2">
              <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
                <input type="checkbox" v-model="form.enabled" class="rounded border-gray-300 text-indigo-600" />
                Enabled
              </label>
            </div>
          </div>
          <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
          <div class="flex gap-2 pt-1">
            <button @click="save" :disabled="saving || !form.source_path || !form.dest_url"
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
        <h2 class="font-semibold text-gray-800 mb-2">Delete redirect?</h2>
        <p class="text-sm text-gray-500 mb-4">This will remove the redirect and reload nginx.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteRedirect(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
