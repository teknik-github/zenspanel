<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { phpVersionsApi } from '@/api/phpVersions'

const versions = ref<any[]>([])
const loading = ref(false)
const confirmToggle = ref<{ id: number; enabled: boolean } | null>(null)

onMounted(async () => {
  const res = await phpVersionsApi.list()
  versions.value = res.data.data || []
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
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">PHP Versions</h1>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Version</th>
            <th class="text-left px-4 py-3 font-medium">FPM Socket</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
            <th class="text-left px-4 py-3 font-medium">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in versions" :key="v.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">PHP {{ v.version }}</td>
            <td class="px-4 py-3 text-gray-400 font-mono">{{ v.fpm_socket }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="v.enabled ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-600'">
                {{ v.enabled ? 'enabled' : 'disabled' }}
              </span>
            </td>
            <td class="px-4 py-3">
              <button @click="confirmToggle = { id: v.id, enabled: v.enabled }"
                class="text-xs border px-3 py-1 rounded transition-colors"
                :class="v.enabled ? 'border-red-200 text-red-600 hover:bg-red-50' : 'border-green-200 text-green-600 hover:bg-green-50'">
                {{ v.enabled ? 'Disable' : 'Enable' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Confirm Toggle -->
    <div v-if="confirmToggle" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">
          {{ confirmToggle.enabled ? 'Disable' : 'Enable' }} PHP Version?
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
            {{ confirmToggle.enabled ? 'Disable' : 'Enable' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
