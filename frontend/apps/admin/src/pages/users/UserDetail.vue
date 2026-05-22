<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { usersApi } from '@/api/users'
import { packagesApi } from '@/api/packages'
import { useRoute, useRouter } from 'vue-router'
import { useToast, useConfirm } from '@/notify'

const route = useRoute()
const router = useRouter()
const { success: toastSuccess, error: toastError } = useToast()
const { confirm } = useConfirm()

const user = ref<any>(null)
const packages = ref<any[]>([])
const form = ref<any>({})
const loading = ref(false)
const saved = ref(false)
const confirmDelete = ref(false)
const suspendLoading = ref(false)

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

async function loginAs() {
  const res = await usersApi.impersonate(user.value.id)
  const token = res.data.token
  const url = `${window.location.origin}/#impersonate=${encodeURIComponent(token)}`
  window.open(url, '_blank')
}

async function suspendUser() {
  const ok = await confirm(
    `Suspend ${user.value.username}? This will disable all their websites, FTP access, and revoke all active sessions immediately.`
  )
  if (!ok) return
  suspendLoading.value = true
  try {
    await usersApi.suspend(user.value.id)
    user.value = { ...user.value, status: 'suspended' }
    toastSuccess('User suspended — all sessions revoked')
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to suspend user')
  } finally {
    suspendLoading.value = false
  }
}

async function unsuspendUser() {
  suspendLoading.value = true
  try {
    await usersApi.unsuspend(user.value.id)
    user.value = { ...user.value, status: 'active' }
    toastSuccess('User unsuspended — services restored')
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to unsuspend user')
  } finally {
    suspendLoading.value = false
  }
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
        :class="user.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'">
        {{ user.status }}
      </span>
    </div>

    <!-- Suspend/Unsuspend banner -->
    <div v-if="user.status === 'suspended'"
      class="bg-red-50 border border-red-200 rounded-lg px-4 py-3 flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-red-800">Account suspended</p>
        <p class="text-xs text-red-600 mt-0.5">All websites, FTP, and sessions are disabled.</p>
      </div>
      <button @click="unsuspendUser" :disabled="suspendLoading"
        class="text-xs bg-green-600 text-white px-3 py-1.5 rounded-md hover:bg-green-700 disabled:opacity-50">
        {{ suspendLoading ? 'Restoring...' : 'Unsuspend' }}
      </button>
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

      <div class="flex items-center gap-3 pt-2 flex-wrap">
        <button @click="save" :disabled="loading"
          class="bg-indigo-600 text-white text-sm px-4 py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
          {{ loading ? 'Saving...' : 'Save Changes' }}
        </button>
        <span v-if="saved" class="text-green-600 text-xs">Saved!</span>
        <button @click="loginAs"
          class="text-sm text-purple-600 border border-purple-200 px-4 py-2 rounded-md hover:bg-purple-50">
          Login as User
        </button>
        <button v-if="user.status === 'active'" @click="suspendUser" :disabled="suspendLoading"
          class="text-sm text-orange-600 border border-orange-200 px-4 py-2 rounded-md hover:bg-orange-50 disabled:opacity-50">
          {{ suspendLoading ? 'Suspending...' : 'Suspend User' }}
        </button>
        <button v-else @click="unsuspendUser" :disabled="suspendLoading"
          class="text-sm text-green-600 border border-green-200 px-4 py-2 rounded-md hover:bg-green-50 disabled:opacity-50">
          {{ suspendLoading ? 'Restoring...' : 'Unsuspend User' }}
        </button>
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
