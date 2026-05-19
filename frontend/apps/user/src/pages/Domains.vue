<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDomainsStore } from '@/stores/domains'
import { phpVersionsApi } from '@/api/phpVersions'
import { domainsApi } from '@/api/domains'

const router = useRouter()
const domainsStore = useDomainsStore()
const phpVersions = ref<any[]>([])
const showModal = ref(false)
const newDomain = ref('')
const newPHP = ref('8.3')
const loading = ref(false)
const confirmDelete = ref<number | null>(null)
const loaded = ref(false)

function manageFiles(domain: string) {
  // Open FileBrowser in a new tab at the domain's docroot. Same origin
  // means the panel session cookie travels with the request and the
  // auth_request bridge logs the user in automatically.
  window.open(`/filebrowser/files/public_html/${domain}/`, '_blank', 'noopener')
}

onMounted(async () => {
  await domainsStore.fetch()
  const res = await phpVersionsApi.listEnabled()
  phpVersions.value = res.data.data || []
  if (phpVersions.value.length) newPHP.value = phpVersions.value[0].version
  loaded.value = true
})

async function createDomain() {
  loading.value = true
  try {
    await domainsApi.create(newDomain.value, newPHP.value)
    showModal.value = false
    newDomain.value = ''
    await domainsStore.fetch()
  } finally {
    loading.value = false
  }
}

async function updatePHP(id: number, version: string) {
  await domainsApi.updatePHPVersion(id, version)
  await domainsStore.fetch()
}

async function deleteDomain(id: number) {
  await domainsApi.delete(id)
  confirmDelete.value = null
  await domainsStore.fetch()
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Domains</h1>
        <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">Manage websites hosted on your account</p>
      </div>
      <button @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Domain
      </button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!domainsStore.domains.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-500 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No domains yet</p>
      <p class="text-xs text-gray-400 mt-1">Add your first domain to get a website online</p>
      <button @click="showModal = true"
        class="mt-4 flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Domain
      </button>
    </div>

    <!-- Table -->
    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[640px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Domain</th>
              <th class="text-left px-4 py-3 font-medium">PHP Version</th>
              <th class="text-left px-4 py-3 font-medium">SSL</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in domainsStore.domains" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
              <td class="px-4 py-3">
                <select :value="d.php_version" @change="updatePHP(d.id, ($event.target as HTMLSelectElement).value)"
                  class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
                  <option v-for="v in phpVersions" :key="v.version" :value="v.version">{{ v.version }}</option>
                </select>
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="d.ssl_type === 'none' ? 'bg-gray-100 text-gray-500' : 'bg-green-100 text-green-700'">
                  {{ d.ssl_type === 'none' ? 'No SSL' : d.ssl_type }}
                </span>
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="d.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'">
                  {{ d.status }}
                </span>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <button @click="manageFiles(d.domain)"
                    title="Manage files for this domain"
                    class="text-xs text-indigo-600 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50 inline-flex items-center gap-1 transition-colors">
                    <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                    </svg>
                    <span class="hidden sm:inline">Files</span>
                  </button>
                  <button @click="confirmDelete = d.id" title="Delete domain" class="text-red-400 hover:text-red-600 transition-colors">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M9 6V4h6v2"/>
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add Domain Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Add Domain</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Domain Name</label>
            <input v-model="newDomain" type="text" placeholder="example.com"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">PHP Version</label>
            <select v-model="newPHP"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
              <option v-for="v in phpVersions" :key="v.version" :value="v.version">PHP {{ v.version }}</option>
            </select>
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="createDomain" :disabled="loading || !newDomain"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete Domain?</h2>
        <p class="text-sm text-gray-500 mb-4">This will remove the domain and its nginx configuration.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="deleteDomain(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
