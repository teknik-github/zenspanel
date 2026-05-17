<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { domainsApi } from '@/api/domains'
import { phpVersionsApi } from '@/api/phpVersions'

const domains = ref<any[]>([])
const phpVersions = ref<any[]>([])
const saving = ref<number | null>(null)
const saved = ref<number | null>(null)

onMounted(async () => {
  const [domainsRes, phpRes] = await Promise.all([
    domainsApi.list(),
    phpVersionsApi.listEnabled(),
  ])
  domains.value = domainsRes.data.data || []
  phpVersions.value = phpRes.data.data || []
})

async function savePHP(domain: any) {
  saving.value = domain.id
  try {
    await domainsApi.updatePHPVersion(domain.id, domain.php_version)
    saved.value = domain.id
    setTimeout(() => { saved.value = null }, 2000)
  } finally {
    saving.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">PHP Settings</h1>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Domain</th>
            <th class="text-left px-4 py-3 font-medium">PHP Version</th>
            <th class="text-left px-4 py-3 font-medium">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="d in domains" :key="d.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ d.domain }}</td>
            <td class="px-4 py-3">
              <select v-model="d.php_version"
                class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
                <option v-for="v in phpVersions" :key="v.version" :value="v.version">PHP {{ v.version }}</option>
              </select>
            </td>
            <td class="px-4 py-3 flex items-center gap-2">
              <button @click="savePHP(d)" :disabled="saving === d.id"
                class="text-xs bg-indigo-600 text-white px-3 py-1 rounded hover:bg-indigo-700 disabled:opacity-50">
                {{ saving === d.id ? 'Saving...' : 'Save' }}
              </button>
              <span v-if="saved === d.id" class="text-green-600 text-xs">Saved!</span>
            </td>
          </tr>
          <tr v-if="!domains.length">
            <td colspan="3" class="px-4 py-8 text-center text-gray-400">No domains found.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
