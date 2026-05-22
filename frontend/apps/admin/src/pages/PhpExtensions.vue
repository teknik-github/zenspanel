<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { phpExtensionsApi } from '@/api/phpExtensions'
import { useToast, useConfirm } from '@/notify'

const { success: toastSuccess, error: toastError } = useToast()
const { confirm } = useConfirm()

const exts = ref<any[]>([])
const loading = ref(false)
const saving = ref<number | null>(null)
const seeding = ref(false)
const phpVersionFilter = ref('')
const showModal = ref(false)
const createLoading = ref(false)
const form = ref({ name: '', php_version: '', enabled: true })

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
    toastSuccess('Default extensions seeded')
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

async function createExt() {
  if (!form.value.name || !form.value.php_version) return
  createLoading.value = true
  try {
    await phpExtensionsApi.adminCreate(form.value.name.trim().toLowerCase(), form.value.php_version, form.value.enabled)
    toastSuccess(`${form.value.name} added to PHP ${form.value.php_version}`)
    showModal.value = false
    form.value = { name: '', php_version: '', enabled: true }
    await fetchExts()
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to add extension')
  } finally {
    createLoading.value = false
  }
}

async function deleteExt(ext: any) {
  const ok = await confirm(`Remove ${ext.name} from PHP ${ext.php_version} catalog? Users who have it enabled will lose it.`)
  if (!ok) return
  try {
    await phpExtensionsApi.adminDelete(ext.id)
    toastSuccess(`${ext.name} removed`)
    await fetchExts()
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to delete extension')
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-3 flex-wrap">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">PHP Extensions</h1>
        <p class="text-xs text-gray-400 mt-0.5">Manage available PHP extensions per version</p>
      </div>
      <div class="flex items-center gap-2 flex-wrap">
        <button @click="seed" :disabled="seeding"
          class="text-xs border border-gray-200 text-gray-600 px-3 py-1.5 rounded-md hover:bg-gray-50 disabled:opacity-50">
          {{ seeding ? 'Seeding...' : 'Seed Defaults' }}
        </button>
        <select v-model="phpVersionFilter"
          class="border border-gray-200 rounded-md px-3 py-1.5 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
          <option value="">All versions</option>
          <option v-for="v in phpVersions" :key="v" :value="v">PHP {{ v }}</option>
        </select>
        <button @click="showModal = true"
          class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-1.5 rounded-md hover:bg-indigo-700">
          <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          Add Extension
        </button>
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
        <p class="text-xs text-gray-400 mt-1">Seed defaults or add extensions manually.</p>
        <button @click="seed" :disabled="seeding"
          class="mt-4 text-xs bg-indigo-600 text-white px-3 py-1.5 rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {{ seeding ? 'Seeding...' : 'Seed Default Extensions' }}
        </button>
      </div>

      <div v-for="(group, ver) in grouped" :key="ver" class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-2.5 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-xs font-semibold text-gray-700">PHP {{ ver }}</span>
            <span class="text-xs text-gray-400">{{ group.filter((e:any) => e.enabled).length }}/{{ group.length }} enabled</span>
          </div>
        </div>
        <div class="overflow-x-auto">
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
                  <div class="flex items-center justify-end gap-2">
                    <button @click="toggle(ext)" :disabled="saving === ext.id"
                      class="text-xs px-3 py-1 rounded border transition-colors disabled:opacity-50"
                      :class="ext.enabled
                        ? 'text-red-600 border-red-200 hover:bg-red-50'
                        : 'text-green-600 border-green-200 hover:bg-green-50'">
                      {{ saving === ext.id ? '...' : ext.enabled ? 'Disable' : 'Enable' }}
                    </button>
                    <button @click="deleteExt(ext)" title="Remove from catalog"
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
    </template>

    <!-- Add Extension Modal -->
    <Transition name="modal">
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="modal-panel bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-4">Add PHP Extension</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Extension Name</label>
            <input v-model="form.name" type="text" placeholder="e.g. redis, imagick, gd"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            <p class="text-[10px] text-gray-400 mt-0.5">Lowercase, no php_ prefix needed</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">PHP Version</label>
            <select v-model="form.php_version"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
              <option value="">Select version</option>
              <option v-for="v in phpVersions" :key="v" :value="v">PHP {{ v }}</option>
              <option value="8.3">PHP 8.3</option>
              <option value="8.2">PHP 8.2</option>
              <option value="8.1">PHP 8.1</option>
            </select>
          </div>
          <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
            <input type="checkbox" v-model="form.enabled" class="rounded border-gray-300 text-indigo-600" />
            Enabled by default
          </label>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">Cancel</button>
          <button @click="createExt" :disabled="createLoading || !form.name || !form.php_version"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ createLoading ? 'Adding...' : 'Add Extension' }}
          </button>
        </div>
      </div>
    </div>
    </Transition>
  </div>
</template>
