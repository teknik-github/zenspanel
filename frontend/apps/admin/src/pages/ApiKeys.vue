<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiKeysApi } from '@/api/apiKeys'

const keys = ref<any[]>([])
const showModal = ref(false)
const newKey = ref({ name: '', permissions: [] as string[], expires_at: '' })
const createdKey = ref<string | null>(null)
const loading = ref(false)
const confirmRevoke = ref<number | null>(null)

const allPermissions = [
  'create_user', 'read_user', 'update_user', 'delete_user',
  'suspend_user', 'unsuspend_user', 'assign_package',
  'read_package', 'create_package', 'update_package', 'delete_package',
  'read_usage',
]

onMounted(async () => {
  const res = await apiKeysApi.list()
  keys.value = res.data.data || []
})

async function createKey() {
  loading.value = true
  try {
    const res = await apiKeysApi.create({
      name: newKey.value.name,
      permissions: JSON.stringify(newKey.value.permissions),
      expires_at: newKey.value.expires_at || undefined,
    })
    createdKey.value = res.data.key
    showModal.value = false
    const listRes = await apiKeysApi.list()
    keys.value = listRes.data.data || []
  } finally {
    loading.value = false
  }
}

async function revokeKey(id: number) {
  await apiKeysApi.revoke(id)
  confirmRevoke.value = null
  const res = await apiKeysApi.list()
  keys.value = res.data.data || []
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">API Keys</h1>
      <button @click="showModal = true; newKey = { name: '', permissions: [], expires_at: '' }"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Create API Key
      </button>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Name</th>
            <th class="text-left px-4 py-3 font-medium">Prefix</th>
            <th class="text-left px-4 py-3 font-medium">Last Used</th>
            <th class="text-left px-4 py-3 font-medium">Expires</th>
            <th class="text-left px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="k in keys" :key="k.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ k.name }}</td>
            <td class="px-4 py-3 font-mono text-gray-500">{{ k.key_prefix }}...</td>
            <td class="px-4 py-3 text-gray-400">{{ k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : 'Never' }}</td>
            <td class="px-4 py-3 text-gray-400">{{ k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'Never' }}</td>
            <td class="px-4 py-3">
              <button @click="confirmRevoke = k.id"
                class="text-xs text-red-600 border border-red-200 px-2 py-1 rounded hover:bg-red-50">Revoke</button>
            </td>
          </tr>
          <tr v-if="!keys.length">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No API keys yet.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 class="font-semibold text-gray-800 mb-4">Create API Key</h2>
        <div class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Name</label>
            <input v-model="newKey.name" type="text" placeholder="WHMCS Integration"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-2">Permissions</label>
            <div class="grid grid-cols-2 gap-1.5">
              <label v-for="perm in allPermissions" :key="perm"
                class="flex items-center gap-2 text-xs text-gray-600 cursor-pointer">
                <input type="checkbox" :value="perm" v-model="newKey.permissions"
                  class="rounded border-gray-300 text-indigo-600" />
                {{ perm }}
              </label>
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Expires At (optional)</label>
            <input v-model="newKey.expires_at" type="date"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="createKey" :disabled="loading || !newKey.name || !newKey.permissions.length"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Show created key -->
    <div v-if="createdKey" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-1">API Key Created</h2>
        <p class="text-xs text-amber-600 mb-4">Copy this key now — it will not be shown again.</p>
        <div class="flex gap-2">
          <input :value="createdKey" readonly
            class="flex-1 border border-gray-200 rounded-md px-3 py-2 text-xs font-mono bg-gray-50" />
          <button @click="copyToClipboard(createdKey!)"
            class="border border-gray-200 px-3 py-2 rounded-md text-xs text-gray-600 hover:bg-gray-50">Copy</button>
        </div>
        <button @click="createdKey = null"
          class="w-full mt-4 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700">Done</button>
      </div>
    </div>

    <!-- Confirm Revoke -->
    <div v-if="confirmRevoke" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Revoke API Key?</h2>
        <p class="text-sm text-gray-500 mb-4">Any integrations using this key will stop working immediately.</p>
        <div class="flex gap-2">
          <button @click="confirmRevoke = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="revokeKey(confirmRevoke!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Revoke</button>
        </div>
      </div>
    </div>
  </div>
</template>
