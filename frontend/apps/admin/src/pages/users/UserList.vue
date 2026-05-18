<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { usersApi } from '@/api/users'
import { packagesApi } from '@/api/packages'
import { useRouter } from 'vue-router'

const router = useRouter()
const users = ref<any[]>([])
const total = ref(0)
const packages = ref<any[]>([])
const search = ref('')
const statusFilter = ref('')
const packageFilter = ref('')
const page = ref(1)
const loading = ref(false)
const confirmDelete = ref<number | null>(null)

const showCreate = ref(false)
const creating = ref(false)
const createError = ref('')
const createWarnings = ref<string[]>([])
const form = ref({
  username: '',
  email: '',
  password: '',
  package_id: '' as string | number,
  terminal_enabled: false,
  backup_enabled: false,
})

function resetForm() {
  form.value = {
    username: '',
    email: '',
    password: '',
    package_id: '',
    terminal_enabled: false,
    backup_enabled: false,
  }
  createError.value = ''
  createWarnings.value = []
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

async function fetchUsers() {
  loading.value = true
  try {
    const res = await usersApi.list({
      search: search.value,
      status: statusFilter.value,
      package_id: packageFilter.value,
      page: page.value,
      limit: 20,
    })
    users.value = res.data.data || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function createUser() {
  createError.value = ''
  createWarnings.value = []
  if (form.value.password.length < 8) {
    createError.value = 'Password must be at least 8 characters'
    return
  }
  creating.value = true
  try {
    const payload: Record<string, unknown> = {
      username: form.value.username,
      email: form.value.email,
      password: form.value.password,
      terminal_enabled: form.value.terminal_enabled,
      backup_enabled: form.value.backup_enabled,
    }
    if (form.value.package_id) payload.package_id = Number(form.value.package_id)
    const res = await usersApi.create(payload)
    if (res.data?.warnings?.length) {
      createWarnings.value = res.data.warnings
    } else {
      showCreate.value = false
    }
    await fetchUsers()
  } catch (e: any) {
    createError.value = e.response?.data?.error || 'Failed to create user'
  } finally {
    creating.value = false
  }
}

onMounted(async () => {
  await fetchUsers()
  const res = await packagesApi.list()
  packages.value = res.data.data || []
})

async function suspend(id: number) {
  await usersApi.suspend(id)
  await fetchUsers()
}

async function unsuspend(id: number) {
  await usersApi.unsuspend(id)
  await fetchUsers()
}

async function deleteUser(id: number) {
  await usersApi.delete(id)
  confirmDelete.value = null
  await fetchUsers()
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Users</h1>
      <button @click="openCreate"
        class="flex items-center gap-1.5 bg-indigo-600 text-white text-xs px-3 py-2 rounded-md hover:bg-indigo-700">
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        Add User
      </button>
    </div>

    <!-- Filters -->
    <div class="flex gap-3">
      <input v-model="search" @input="fetchUsers" type="text" placeholder="Search username or email..."
        class="border border-gray-200 rounded-md px-3 py-2 text-xs w-64 focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      <select v-model="statusFilter" @change="fetchUsers"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Status</option>
        <option value="active">Active</option>
        <option value="suspended">Suspended</option>
      </select>
      <select v-model="packageFilter" @change="fetchUsers"
        class="border border-gray-200 rounded-md px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-indigo-500">
        <option value="">All Packages</option>
        <option v-for="p in packages" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">Username</th>
            <th class="text-left px-4 py-3 font-medium">Email</th>
            <th class="text-left px-4 py-3 font-medium">Package</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
            <th class="text-left px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium text-gray-700">{{ u.username }}</td>
            <td class="px-4 py-3 text-gray-500">{{ u.email }}</td>
            <td class="px-4 py-3 text-gray-500">{{ u.package_id || '—' }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="u.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'">
                {{ u.status }}
              </span>
            </td>
            <td class="px-4 py-3 flex items-center gap-2">
              <button @click="router.push(`/users/${u.id}`)"
                class="text-xs text-indigo-600 border border-indigo-200 px-2 py-1 rounded hover:bg-indigo-50">View</button>
              <button v-if="u.status === 'active'" @click="suspend(u.id)"
                class="text-xs text-amber-600 border border-amber-200 px-2 py-1 rounded hover:bg-amber-50">Suspend</button>
              <button v-else @click="unsuspend(u.id)"
                class="text-xs text-green-600 border border-green-200 px-2 py-1 rounded hover:bg-green-50">Unsuspend</button>
              <button @click="confirmDelete = u.id" class="text-red-400 hover:text-red-600">
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                </svg>
              </button>
            </td>
          </tr>
          <tr v-if="!users.length">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No users found.</td>
          </tr>
        </tbody>
      </table>
      <div class="px-4 py-3 border-t border-gray-100 text-xs text-gray-400">
        {{ total }} total users
      </div>
    </div>

    <!-- Add User modal -->
    <div v-if="showCreate" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-md shadow-xl max-h-[90vh] overflow-y-auto">
        <h2 class="font-semibold text-gray-800 mb-4">Add User</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Username</label>
            <input v-model="form.username" type="text" placeholder="lowercase, 3-32 chars, starts with letter"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Email</label>
            <input v-model="form.email" type="email" placeholder="user@example.com"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Password</label>
            <input v-model="form.password" type="password" placeholder="minimum 8 characters"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-600 mb-1">Package</label>
            <select v-model="form.package_id"
              class="w-full border border-gray-200 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500">
              <option value="">— No package —</option>
              <option v-for="p in packages" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>
          <div class="flex items-center gap-6 pt-1">
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.terminal_enabled" class="rounded border-gray-300 text-indigo-600" />
              Terminal Enabled
            </label>
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.backup_enabled" class="rounded border-gray-300 text-indigo-600" />
              Backup Enabled
            </label>
          </div>
          <p v-if="createError" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">
            {{ createError }}
          </p>
          <div v-if="createWarnings.length" class="text-xs text-amber-700 bg-amber-50 border border-amber-100 rounded px-2 py-1.5 space-y-0.5">
            <p class="font-medium">User created with warnings:</p>
            <p v-for="w in createWarnings" :key="w">• {{ w }}</p>
          </div>
        </div>
        <div class="flex gap-2 mt-5">
          <button @click="showCreate = false"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Close</button>
          <button @click="createUser"
            :disabled="creating || !form.username || !form.email || !form.password"
            class="flex-1 bg-indigo-600 text-white rounded-md py-2 text-sm hover:bg-indigo-700 disabled:opacity-50">
            {{ creating ? 'Creating...' : 'Create User' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Confirm Delete -->
    <div v-if="confirmDelete" class="fixed inset-0 bg-black/30 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl">
        <h2 class="font-semibold text-gray-800 mb-2">Delete User?</h2>
        <p class="text-sm text-gray-500 mb-4">This will permanently delete the user and all their data.</p>
        <div class="flex gap-2">
          <button @click="confirmDelete = null"
            class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm">Cancel</button>
          <button @click="deleteUser(confirmDelete!)"
            class="flex-1 bg-red-600 text-white rounded-md py-2 text-sm hover:bg-red-700">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>
