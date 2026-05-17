<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { sslApi } from '@/api/ssl'

const domains = ref<any[]>([])
const showUploadModal = ref(false)
const selectedDomain = ref<any>(null)
const certPEM = ref('')
const keyPEM = ref('')
const loading = ref(false)

onMounted(async () => {
  const res = await domainsApi.list()
  domains.value = res.data.data || []
})

async function issueLetsEncrypt(domainId: number) {
  loading.value = true
  try {
    await sslApi.issueLetsEncrypt(domainId)
    const res = await domainsApi.list()
    domains.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function uploadCustomSSL() {
  loading.value = true
  try {
    await sslApi.uploadCustom(selectedDomain.value.id, certPEM.value, keyPEM.value)
    showUploadModal.value = false
    certPEM.value = ''
    keyPEM.value = ''
    const res = await domainsApi.list()
    domains.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function removeSSL(domainId: number) {
  await sslApi.remove(domainId)
  const res = await domainsApi.list()
  domains.value = res.data.data || []
}

function isExpiringSoon(expiresAt: string) {
  if (!expiresAt) return false
  return new Date(expiresAt).getTime() - Date.now() < 30 * 24 * 60 * 60 * 1000
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">SSL Manager</h1>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
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
            <td class="px-4 py-3 flex items-center gap-2">
              <button @click="issueLetsEncrypt(d.id)" :disabled="loading"
                class="text-xs text-green-600 border border-green-200 px-2 py-1 rounded hover:bg-green-50 disabled:opacity-50">
                Let's Encrypt
              </button>
              <button @click="selectedDomain = d; showUploadModal = true"
                class="text-xs text-blue-600 border border-blue-200 px-2 py-1 rounded hover:bg-blue-50">
                Upload Custom
              </button>
              <button v-if="d.ssl_type !== 'none'" @click="removeSSL(d.id)"
                class="text-xs text-red-500 border border-red-200 px-2 py-1 rounded hover:bg-red-50">
                Remove
              </button>
            </td>
          </tr>
          <tr v-if="!domains.length">
            <td colspan="4" class="px-4 py-8 text-center text-gray-400">No domains found.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Upload Custom SSL Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl">
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
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="uploadCustomSSL" :disabled="loading || !certPEM || !keyPEM"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Uploading...' : 'Upload' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
