<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { phpExtensionsApi } from '@/api/phpExtensions'

const exts = ref<any[]>([])
const loading = ref(false)
const saving = ref<number | null>(null)
const seeding = ref(false)
const phpVersionFilter = ref('')

onMounted(async () => {
  await fetchExts()
})

async function fetchExts() {
  loading.value = true
  try {
    const res = await phpExtensionsApi.adminList()
    exts.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function seed() {
  seeding.value = true
  try {
    await phpExtensionsApi.adminSeed()
    await fetchExts()
  } finally {
    seeding.value = false
  }
}

const phpVersions = computed(() => {
  const vers = new Set(exts.value.map((e: any) => e.php_version))
  return Array.from(vers).sort()
})

const filtered = computed(() => {
  if (!phpVersionFilter.value) return exts.value
  return exts.value.filter((e: any) => e.php_version === phpVersionFilter.value)
})

// Group by php_version for display
const grouped = computed(() => {
  const map: Record<string, any[]> = {}
  for (const e of filtered.value) {
    if (!map[e.php_version]) map[e.php_version] = []
    map[e.php_version].push(e)
  }
  return map
})

async function toggle(ext: any) {
  saving.value = ext.id
  try {
    await phpExtensionsApi.adminUpdate(ext.id, !ext.enabled)
    ext.enabled = !ext.enabled
  } finally {
    saving.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">PHP Extensions</h1>
      <div class="flex items-center gap-2">
        <button v-if="!exts.length" @click="seed" :disabled="seeding"
          class="text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {{ seeding ? 'Seeding...' : 'Seed Default Extensions' }}
        </button>
        <select v-model="phpVersionFilter"
          class="border border-gray-200 rounded-md px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
          <option value="">All versions</option>
          <option v-for="v in phpVersions" :key="v" :value="v">PHP {{ v }}</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="bg-white border border-gray-200 rounded-lg p-4 animate-pulse">
        <div class="h-4 bg-gray-100 rounded w-24 mb-3"></div>
        <div class="space-y-2">
          <div v-for="j in 4" :key="j" class="h-8 bg-gray-50 rounded"></div>
        </div>
      </div>
    </div>

    <template v-else>
      <div v-if="!exts.length" class="flex flex-col items-center justify-center py-16 text-center bg-white border border-gray-200 rounded-lg">
        <div class="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mb-3">
          <svg class="w-6 h-6 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
          </svg>
        </div>
        <p class="text-sm font-medium text-gray-700">No extensions found</p>
        <p class="text-xs text-gray-400 mt-1">Run the migration to seed the extension catalog.</p>
      </div>

      <div v-for="(group, ver) in grouped" :key="ver" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-2.5 bg-gray-50 border-b border-gray-200 flex items-center gap-2">
          <span class="text-xs font-semibold text-gray-700">PHP {{ ver }}</span>
          <span class="text-xs text-gray-400">{{ group.filter((e:any) => e.enabled).length }}/{{ group.length }} enabled</span>
        </div>
        <table class="w-full text-xs">
          <tbody>
            <tr v-for="ext in group" :key="ext.id"
              class="border-b border-gray-50 last:border-0 hover:bg-gray-50">
              <td class="px-4 py-2.5 font-mono text-gray-700">{{ ext.name }}</td>
              <td class="px-4 py-2.5">
                <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                  :class="ext.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
                  {{ ext.enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td class="px-4 py-2.5 text-right">
                <button @click="toggle(ext)" :disabled="saving === ext.id"
                  class="text-xs px-3 py-1 rounded border transition-colors disabled:opacity-50"
                  :class="ext.enabled
                    ? 'text-red-600 border-red-200 hover:bg-red-50'
                    : 'text-green-600 border-green-200 hover:bg-green-50'">
                  {{ saving === ext.id ? '...' : ext.enabled ? 'Disable' : 'Enable' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>
