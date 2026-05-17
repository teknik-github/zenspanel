<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { packagesApi } from '@/api/packages'

const packages = ref<any[]>([])
const showModal = ref(false)
const editingPackage = ref<any>(null)
const form = ref<any>({})
const loading = ref(false)
const confirmDelete = ref<number | null>(null)

const defaultForm = () => ({
  name: '',
  cpu_quota: 10000,
  memory_limit: 536870912,
  disk_quota: 10737418240,
  max_domains: 5,
  max_databases: 5,
  php_versions_allowed: '["8.3","8.2","8.1"]',
  terminal_enabled: false,
  backup_enabled: false,
})

onMounted(async () => {
  const res = await packagesApi.list()
  packages.value = res.data.data || []
})

function openCreate() {
  editingPackage.value = null
  form.value = defaultForm()
  showModal.value = true
}

function openEdit(pkg: any) {
  editingPackage.value = pkg
  form.value = { ...pkg }
  showModal.value = true
}

async function save() {
  loading.value = true
  try {
    if (editingPackage.value) {
      await packagesApi.update(editingPackage.value.id, form.value)
    } else {
      await packagesApi.create(form.value)
    }
    showModal.value = false
    const res = await packagesApi.list()
    packages.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function deletePackage(id: number) {
  await packagesApi.delete(id)
  confirmDelete.value = null
  const res = await packagesApi.list()
  packages.value = res.data.data || []
}

function formatBytes(bytes: number) {
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(0) + ' GB'
  return (bytes / 1048576).toFixed(0) + ' MB'
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Packages</h1>
      <button @click="openCreate"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Package
      </button>
    </div>

    <div class="grid grid-cols-3 gap-4">
      <div v-for="pkg in packages" :key="pkg.id"
        class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
        <div class="flex items-start justify-between">
          <h3 class="font-semibold text-gray-800">{{ pkg.name }}</h3>
          <div class="flex gap-1">
            <button @click="openEdit(pkg)" class="text-gray-400 hover:text-indigo-600">
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
            <button @click="confirmDelete = pkg.id" class="text-gray-400 hover:text-red-500">
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
              </svg>
            </button>
          </div>
        </div>
        <div class="text-xs text-gray-500 space-y-1">
          <div class="flex justify-between"><span>CPU</span><span class="font-medium text-gray-700">{{ pkg.cpu_quota / 1000 }}%</span></div>
          <div class="flex justify-between"><span>RAM</span><span class="font-medium text-gray-700">{{ formatBytes(pkg.memory_limit) }}</span></div>
          <div class="flex justify-between"><span>Disk</span><span class="font-medium text-gray-700">{{ formatBytes(pkg.disk_quota) }}</span></div>
          <div class="flex justify-between"><span>Domains</span><span class="font-medium text-gray-700">{{ pkg.max_domains }}</span></div>
          <div class="flex justify-between"><span>Databases</span><span class="font-medium text-gray-700">{{ pkg.max_databases }}</span></div>
        </div>
        <div class="flex gap-2 text-[10px]">
          <span v-if="pkg.terminal_enabled" class="bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded">Terminal</span>
          <span v-if="pkg.backup_enabled" class="bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded">Backup</span>
        </div>
      </div>
      <div v-if="!packages.length" class="col-span-3 py-12 text-center text-gray-400 text-sm">
        No packages yet. Create your first package.
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 class="font-semibold text-gray-800 mb-4">{{ editingPackage ? 'Edit' : 'Create' }} Package</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Name</label>
            <input v-model="form.name" type="text" placeholder="Basic / Pro / Business"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">CPU Quota (microseconds)</label>
              <input v-model.number="form.cpu_quota" type="number"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Memory Limit (bytes)</label>
              <input v-model.number="form.memory_limit" type="number"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Disk Quota (bytes)</label>
              <input v-model.number="form.disk_quota" type="number"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Max Domains</label>
              <input v-model.number="form.max_domains" type="number"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Max Databases</label>
              <input v-model.number="form.max_databases" type="number"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
          </div>
          <div class="flex items-center gap-6">
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.terminal_enabled" class="rounded border-gray-300 text-indigo-600" />
              Terminal Enabled
            </label>
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.backup_enabled" class="rounded border-gray-300 text-indigo-600" />
              Backup Enabled
            </label>
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showModal = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="save" :disabled="loading || !form.name"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Saving...' : 'Save' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete Package?</h2>
        <p class="text-sm text-gray-500 mb-4">Users assigned to this package will have no package.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deletePackage(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
