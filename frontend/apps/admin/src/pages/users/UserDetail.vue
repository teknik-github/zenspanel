<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { usersApi } from '@/api/users'
import { packagesApi } from '@/api/packages'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const user = ref<any>(null)
const packages = ref<any[]>([])
const form = ref<any>({})
const loading = ref(false)
const saved = ref(false)
const confirmDelete = ref(false)

onMounted(async () => {
  const [userRes, pkgRes] = await Promise.all([
    usersApi.get(Number(route.params.id)),
    packagesApi.list(),
  ])
  user.value = userRes.data
  packages.value = pkgRes.data.data || []
  form.value = {
    email: user.value.email,
    package_id: user.value.package_id,
    terminal_enabled: user.value.terminal_enabled,
    backup_enabled: user.value.backup_enabled,
    status: user.value.status,
  }
})

async function save() {
  loading.value = true
  try {
    await usersApi.update(user.value.id, form.value)
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function deleteUser() {
  await usersApi.delete(user.value.id)
  router.push('/users')
}
</script>

<template>
  <div v-if="user" class="space-y-5 max-w-2xl">
    <div class="flex items-center gap-3">
      <button @click="router.push('/users')" class="text-gray-400 hover:text-gray-600">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="15 18 9 12 15 6"/>
        </svg>
      </button>
      <h1 class="text-lg font-semibold text-gray-800">{{ user.username }}</h1>
      <span class="px-2 py-0.5 rounded text-[10px] font-medium"
        :class="user.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'">
        {{ user.status }}
      </span>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-5 space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Username</label>
          <input :value="user.username" disabled
            class="w-full border border-gray-100 rounded-md px-3 py-2 text-sm bg-gray-50 text-gray-400" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Email</label>
          <input v-model="form.email" type="email"
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Package</label>
          <select v-model="form.package_id"
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
            <option :value="null">No Package</option>
            <option v-for="p in packages" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-600 mb-1">Status</label>
          <select v-model="form.status"
            class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
            <option value="active">Active</option>
            <option value="suspended">Suspended</option>
          </select>
        </div>
      </div>

      <div class="flex items-center gap-6 pt-2">
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="checkbox" v-model="form.terminal_enabled" class="rounded border-gray-300 text-indigo-600" />
          <span class="text-xs text-gray-600">Terminal Enabled</span>
        </label>
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="checkbox" v-model="form.backup_enabled" class="rounded border-gray-300 text-indigo-600" />
          <span class="text-xs text-gray-600">Backup Enabled</span>
        </label>
      </div>

      <div class="flex items-center gap-3 pt-2">
        <button @click="save" :disabled="loading"
          class="bg-indigo-600 text-white text-sm px-4 py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {{ loading ? 'Saving...' : 'Save Changes' }}
        </button>
        <span v-if="saved" class="text-green-600 text-xs">Saved!</span>
        <button @click="confirmDelete = true"
          class="ml-auto text-sm text-red-600 border border-red-200 px-4 py-2 rounded-md hover:bg-red-50">
          Delete User
        </button>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete {{ user.username }}?</h2>
        <p class="text-sm text-gray-500 mb-4">This will permanently delete the user and all their data.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteUser"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
  <div v-else class="flex items-center justify-center py-16 text-gray-400 text-sm">Loading...</div>
</template>
