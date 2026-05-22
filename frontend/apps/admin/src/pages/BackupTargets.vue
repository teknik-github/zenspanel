<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { backupTargetsApi } from '@/api/backupTargets'

const targets = ref<any[]>([])
const loading = ref(false)
const showModal = ref(false)
const editingTarget = ref<any>(null)
const saving = ref(false)
const testing = ref<number | null>(null)
const testResult = ref<Record<number, {ok: boolean, error?: string}>>({})
const confirmDelete = ref<number | null>(null)
const error = ref('')

const defaultForm = () => ({
  name: '',
  type: 's3',
  bucket: '',
  prefix: 'backups/',
  access_key: '',
  secret_key: '',
  region: 'us-east-1',
  endpoint: '',
  enabled: true,
})
const form = ref(defaultForm())

onMounted(fetchTargets)

async function fetchTargets() {
  loading.value = true
  try {
    const res = await backupTargetsApi.list()
    targets.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingTarget.value = null
  form.value = defaultForm()
  error.value = ''
  showModal.value = true
}

function openEdit(t: any) {
  editingTarget.value = t
  form.value = {
    name: t.name,
    type: t.type,
    bucket: t.bucket,
    prefix: t.prefix,
    access_key: t.access_key,
    secret_key: '', // never pre-fill secret
    region: t.region,
    endpoint: t.endpoint,
    enabled: t.enabled,
  }
  error.value = ''
  showModal.value = true
}

async function save() {
  error.value = ''
  saving.value = true
  try {
    if (editingTarget.value) {
      const payload: any = { ...form.value }
      if (!payload.secret_key) delete payload.secret_key // keep existing if empty
      await backupTargetsApi.update(editingTarget.value.id, payload)
    } else {
      await backupTargetsApi.create(form.value)
    }
    showModal.value = false
    await fetchTargets()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to save'
  } finally {
    saving.value = false
  }
}

async function testTarget(id: number) {
  testing.value = id
  testResult.value[id] = { ok: false }
  try {
    const res = await backupTargetsApi.test(id)
    testResult.value[id] = res.data
  } catch (e: any) {
    testResult.value[id] = { ok: false, error: e.response?.data?.error || 'Test failed' }
  } finally {
    testing.value = null
  }
}

async function deleteTarget(id: number) {
  await backupTargetsApi.delete(id)
  confirmDelete.value = null
  await fetchTargets()
}

const typeLabels: Record<string, string> = {
  s3: 'Amazon S3 / S3-compatible',
  b2: 'Backblaze B2',
  gcs: 'Google Cloud Storage',
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-lg font-semibold text-gray-800">Backup Targets</h1>
        <p class="text-xs text-gray-400 mt-0.5">Remote destinations for automatic backup uploads (S3, B2, GCS)</p>
      </div>
      <button @click="openCreate"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add Target
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-2">
      <div v-for="i in 2" :key="i" class="h-16 bg-white border border-gray-200 rounded-lg animate-pulse"></div>
    </div>

    <!-- Empty -->
    <div v-else-if="!targets.length"
      class="bg-white border border-gray-200 rounded-lg flex flex-col items-center justify-center py-12 text-center">
      <svg class="w-8 h-8 text-gray-300 mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/>
        <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
      </svg>
      <p class="text-sm font-medium text-gray-700">No backup targets</p>
      <p class="text-xs text-gray-400 mt-1">Add an S3-compatible target to enable remote backups.</p>
    </div>

    <!-- List -->
    <div v-else class="space-y-2">
      <div v-for="t in targets" :key="t.id"
        class="bg-white border border-gray-200 rounded-lg p-4 flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-sm font-semibold text-gray-800">{{ t.name }}</span>
            <span class="px-2 py-0.5 rounded text-[10px] font-medium"
              :class="t.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
              {{ t.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>
          <p class="text-xs text-gray-500 mt-0.5">
            {{ typeLabels[t.type] || t.type }} · {{ t.bucket }}/{{ t.prefix }} · {{ t.region }}
          </p>
          <div v-if="testResult[t.id]" class="mt-1 flex items-center gap-1">
            <svg v-if="testResult[t.id].ok" class="w-3.5 h-3.5 text-green-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
            <svg v-else class="w-3.5 h-3.5 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
            <span class="text-xs" :class="testResult[t.id].ok ? 'text-green-600' : 'text-red-500'">
              {{ testResult[t.id].ok ? 'Connection OK' : testResult[t.id].error }}
            </span>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button @click="testTarget(t.id)" :disabled="testing === t.id"
            class="text-xs border border-gray-200 px-3 py-1.5 rounded hover:bg-gray-50 disabled:opacity-50">
            {{ testing === t.id ? 'Testing...' : 'Test' }}
          </button>
          <button @click="openEdit(t)" class="text-indigo-600 hover:text-indigo-800">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
            </svg>
          </button>
          <button @click="confirmDelete = t.id" class="text-red-400 hover:text-red-600">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Add/Edit modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-lg shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 class="font-semibold text-gray-800 mb-4">{{ editingTarget ? 'Edit' : 'Add' }} Backup Target</h2>
        <div class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div class="col-span-2">
              <label class="block text-xs font-medium text-gray-600 mb-1">Name</label>
              <input v-model="form.name" type="text" placeholder="My S3 Backup"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Type</label>
              <select v-model="form.type"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
                <option value="s3">Amazon S3 / S3-compatible</option>
                <option value="b2">Backblaze B2</option>
                <option value="gcs">Google Cloud Storage</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Region</label>
              <input v-model="form.region" type="text" placeholder="us-east-1"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Bucket</label>
              <input v-model="form.bucket" type="text" placeholder="my-backup-bucket"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Prefix</label>
              <input v-model="form.prefix" type="text" placeholder="backups/"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div class="col-span-2">
              <label class="block text-xs font-medium text-gray-600 mb-1">Endpoint (S3-compatible only, leave empty for AWS)</label>
              <input v-model="form.endpoint" type="text" placeholder="https://s3.example.com"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Access Key ID</label>
              <input v-model="form.access_key" type="text" autocomplete="off"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">
                Secret Key {{ editingTarget ? '(leave empty to keep)' : '' }}
              </label>
              <input v-model="form.secret_key" type="password" autocomplete="new-password"
                class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500" />
            </div>
          </div>
          <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
            <input type="checkbox" v-model="form.enabled" class="rounded border-gray-300 text-indigo-600" />
            Enabled
          </label>
          <p v-if="error" class="text-xs text-red-600">{{ error }}</p>
          <div class="flex gap-2 pt-1">
            <button @click="save" :disabled="saving"
              class="flex-1 bg-indigo-600 text-white text-sm py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
              {{ saving ? 'Saving...' : 'Save' }}
            </button>
            <button @click="showModal = false"
              class="flex-1 border border-gray-200 text-gray-600 text-sm py-2 rounded-md hover:bg-gray-50">
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete backup target?</h2>
        <p class="text-sm text-gray-500 mb-4">Existing backups in the remote storage will not be deleted.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteTarget(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
