<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { sslApi } from '@/api/ssl'

const domains = ref<any[]>([])
const showUploadModal = ref(false)
const selectedDomain = ref<any>(null)
const certPEM = ref('')
const keyPEM = ref('')
const loaded = ref(false)
// Per-row pending-state map keyed by domain ID. The previous
// implementation had a single shared `loading` flag, which disabled every
// SSL action across every row whenever any one of them was in flight.
const pending = ref<Record<number, string>>({})
const confirmRemove = ref<any>(null)
const uploading = ref(false)

async function refresh() {
  const res = await domainsApi.list()
  domains.value = res.data.data || []
}

onMounted(async () => {
  try {
    await refresh()
  } finally {
    loaded.value = true
  }
})

async function issueLetsEncrypt(domainId: number) {
  pending.value[domainId] = 'issuing'
  try {
    await sslApi.issueLetsEncrypt(domainId)
    await refresh()
  } catch (e: any) {
    alert(e?.response?.data?.error || 'Failed to issue certificate')
  } finally {
    delete pending.value[domainId]
  }
}

async function uploadCustomSSL() {
  if (!selectedDomain.value) return
  uploading.value = true
  try {
    await sslApi.uploadCustom(selectedDomain.value.id, certPEM.value, keyPEM.value)
    showUploadModal.value = false
    certPEM.value = ''
    keyPEM.value = ''
    await refresh()
  } catch (e: any) {
    alert(e?.response?.data?.error || 'Failed to upload certificate')
  } finally {
    uploading.value = false
  }
}

async function removeSSL(domainId: number) {
  pending.value[domainId] = 'removing'
  try {
    await sslApi.remove(domainId)
    confirmRemove.value = null
    await refresh()
  } catch (e: any) {
    alert(e?.response?.data?.error || 'Failed to remove certificate')
  } finally {
    delete pending.value[domainId]
  }
}

function isExpiringSoon(expiresAt: string) {
  if (!expiresAt) return false
  return new Date(expiresAt).getTime() - Date.now() < 30 * 24 * 60 * 60 * 1000
}
</script>

<template>
  <div class="space-y-4">
    <div>
      <h1 class="text-lg font-semibold text-gray-800">SSL Manager</h1>
      <p class="text-xs text-gray-400 mt-0.5 hidden sm:block">Issue Let's Encrypt or upload custom certificates</p>
    </div>

    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 3" :key="i" class="h-8 bg-gray-50 rounded animate-pulse" />
    </div>

    <div v-else-if="!domains.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-emerald-50 text-emerald-500 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No domains found</p>
      <p class="text-xs text-gray-400 mt-1">Add a domain first to issue an SSL certificate</p>
      <router-link to="/domains"
        class="mt-4 flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 transition-colors">
        Go to Domains
      </router-link>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[640px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Domain</th>
              <th class="text-left px-4 py-3 font-medium">SSL Type</th>
              <th class="text-left px-4 py-3 font-medium">Expires</th>
              <th class="text-left px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in domains" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="d.ssl_type === 'none' ? 'bg-gray-100 text-gray-500' : d.ssl_type === 'letsencrypt' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'">
                  {{ d.ssl_type === 'none' ? 'No SSL' : d.ssl_type }}
                </span>
              </td>
              <td class="px-4 py-3">
                <span v-if="d.ssl_expires_at"
                  class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="isExpiringSoon(d.ssl_expires_at) ? 'bg-amber-100 text-amber-700' : 'text-gray-400'">
                  {{ new Date(d.ssl_expires_at).toLocaleDateString() }}
                  {{ isExpiringSoon(d.ssl_expires_at) ? '⚠ Expiring soon' : '' }}
                </span>
                <span v-else class="text-gray-400">—</span>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2 flex-wrap">
                  <button @click="issueLetsEncrypt(d.id)" :disabled="!!pending[d.id]"
                    class="text-xs text-green-600 border border-green-200 px-2 py-1 rounded hover:bg-green-50 disabled:opacity-50 transition-colors">
                    {{ pending[d.id] === 'issuing' ? 'Issuing…' : "Let's Encrypt" }}
                  </button>
                  <button @click="selectedDomain = d; showUploadModal = true" :disabled="!!pending[d.id]"
                    class="text-xs text-blue-600 border border-blue-200 px-2 py-1 rounded hover:bg-blue-50 disabled:opacity-50 transition-colors">
                    Upload Custom
                  </button>
                  <button v-if="d.ssl_type !== 'none'" @click="confirmRemove = d" :disabled="!!pending[d.id]"
                    class="text-xs text-red-500 border border-red-200 px-2 py-1 rounded hover:bg-red-50 disabled:opacity-50 transition-colors">
                    {{ pending[d.id] === 'removing' ? 'Removing…' : 'Remove' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Upload Custom SSL Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 class="font-semibold text-gray-800 mb-1">Upload Custom SSL</h2>
        <p class="text-xs text-gray-400 mb-4">{{ selectedDomain?.domain }}</p>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Certificate (PEM)</label>
            <textarea v-model="certPEM" rows="5" placeholder="-----BEGIN CERTIFICATE-----"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"></textarea>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Private Key (PEM)</label>
            <textarea v-model="keyPEM" rows="5" placeholder="-----BEGIN PRIVATE KEY-----"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"></textarea>
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showUploadModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="uploadCustomSSL" :disabled="uploading || !certPEM || !keyPEM"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ uploading ? 'Uploading...' : 'Upload' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Remove SSL confirm -->
    <div v-if="confirmRemove" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Remove SSL?</h2>
        <p class="text-sm text-gray-500 mb-4">This will remove the SSL certificate from <span class="font-mono">{{ confirmRemove.domain }}</span>. The site will fall back to HTTP only.</p>
        <div class="flex gap-2">
          <button @click="confirmRemove = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="removeSSL(confirmRemove.id)" :disabled="!!pending[confirmRemove.id]"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700 disabled:opacity-50">
            {{ pending[confirmRemove.id] === 'removing' ? 'Removing…' : 'Remove' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
