<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { phpVersionsApi } from '@/api/phpVersions'
import { useToast, useConfirm } from '@/notify'

const { success: toastSuccess, error: toastError } = useToast()
const { confirm } = useConfirm()

const versions = ref<any[]>([])
const loaded = ref(false)
const loading = ref(false)
const confirmToggle = ref<{ id: number; enabled: boolean; version: string } | null>(null)
const showModal = ref(false)
const createLoading = ref(false)
const form = ref({ version: '', fpm_socket: '' })

onMounted(async () => {
  const res = await phpVersionsApi.list()
  versions.value = res.data.data || []
  loaded.value = true
})

async function toggle(id: number, enabled: boolean) {
  loading.value = true
  try {
    if (enabled) {
      await phpVersionsApi.disable(id)
    } else {
      await phpVersionsApi.enable(id)
    }
    const res = await phpVersionsApi.list()
    versions.value = res.data.data || []
  } finally {
    loading.value = false
    confirmToggle.value = null
  }
}

async function createVersion() {
  if (!form.value.version) return
  createLoading.value = true
  try {
    await phpVersionsApi.create(form.value.version.trim(), form.value.fpm_socket.trim() || undefined)
    toastSuccess(`PHP ${form.value.version} added`)
    showModal.value = false
    form.value = { version: '', fpm_socket: '' }
    const res = await phpVersionsApi.list()
    versions.value = res.data.data || []
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to add PHP version')
  } finally {
    createLoading.value = false
  }
}

async function deleteVersion(v: any) {
  const ok = await confirm(`Remove PHP ${v.version}? Users with this version assigned will keep it until changed.`)
  if (!ok) return
  try {
    await phpVersionsApi.delete(v.id)
    toastSuccess(`PHP ${v.version} removed`)
    const res = await phpVersionsApi.list()
    versions.value = res.data.data || []
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to delete PHP version')
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">PHP Versions</h1>
        <p class="text-xs text-gray-400 mt-0.5">Enable or disable PHP versions available to users</p>
      </div>
      <button @click="showModal = true"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700 flex-shrink-0">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add PHP Version
      </button>
    </div>

    <!-- Skeleton -->
    <div v-if="!loaded" class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <div v-for="i in 3" :key="i" class="h-10 bg-gray-50 rounded animate-pulse" />
    </div>

    <!-- Empty state -->
    <div v-else-if="!versions.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center px-4">
      <div class="w-12 h-12 rounded-full bg-indigo-50 text-indigo-400 flex items-center justify-center mb-3">
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
        </svg>
      </div>
      <p class="text-sm font-medium text-gray-700">No PHP versions found</p>
      <p class="text-xs text-gray-400 mt-1">Add a PHP version to get started</p>
    </div>

    <div v-else class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-xs min-w-[480px]">
          <thead class="bg-gray-50 border-b border-gray-200">
            <tr class="text-gray-500">
              <th class="text-left px-4 py-3 font-medium">Version</th>
              <th class="text-left px-4 py-3 font-medium">FPM Socket</th>
              <th class="text-left px-4 py-3 font-medium">Status</th>
              <th class="text-right px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="v in versions" :key="v.id" class="border-b border-gray-50 hover:bg-gray-50">
              <td class="px-4 py-3 font-medium text-gray-700">PHP {{ v.version }}</td>
              <td class="px-4 py-3 text-gray-400 font-mono text-[10px]">{{ v.fpm_socket }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="v.enabled ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600'">
                  {{ v.enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-2">
                  <button @click="confirmToggle = { id: v.id, enabled: v.enabled, version: v.version }"
                    class="text-xs border px-3 py-1 rounded transition-colors"
                    :class="v.enabled
                      ? 'border-red-200 text-red-600 hover:bg-red-50'
                      : 'border-green-200 text-green-600 hover:bg-green-50'">
                    {{ v.enabled ? 'Disable' : 'Enable' }}
                  </button>
                  <button @click="deleteVersion(v)" title="Remove PHP version"
                    class="text-gray-400 hover:text-red-500 transition-colors">
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

    <!-- Add PHP Version Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Add PHP Version</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">PHP Version</label>
            <input v-model="form.version" type="text" placeholder="e.g. 8.4"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-0.5">Format: X.Y (e.g. 8.4, 8.5)</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">FPM Socket <span class="text-gray-400">(optional)</span></label>
            <input v-model="form.fpm_socket" type="text" :placeholder="`/run/php/php${form.version || 'X.Y'}-fpm.sock`"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-0.5">Leave empty to use default path</p>
          </div>
          <div class="bg-amber-50 border border-amber-100 rounded px-3 py-2 text-xs text-amber-700">
            Make sure PHP {{ form.version || 'X.Y' }}-FPM is installed on the server before enabling this version.
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false; form = { version: '', fpm_socket: '' }"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="createVersion" :disabled="createLoading || !form.version"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ createLoading ? 'Adding...' : 'Add Version' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Toggle -->
    <div v-if="confirmToggle" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">
          {{ confirmToggle.enabled ? 'Disable' : 'Enable' }} PHP {{ confirmToggle.version }}?
        </h2>
        <p class="text-sm text-gray-500 mb-4">
          {{ confirmToggle.enabled
            ? 'Existing sites using this version will continue to work, but new sites cannot select it.'
            : 'This PHP version will be available for new and existing sites.' }}
        </p>
        <div class="flex gap-2">
          <button @click="confirmToggle = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="toggle(confirmToggle!.id, confirmToggle!.enabled)" :disabled="loading"
            class="flex-1 rounded-md py-2 text-sm text-white disabled:opacity-50"
            :class="confirmToggle.enabled ? 'bg-red-600 hover:bg-red-700' : 'bg-green-600 hover:bg-green-700'">
            {{ loading ? 'Saving...' : (confirmToggle.enabled ? 'Disable' : 'Enable') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
