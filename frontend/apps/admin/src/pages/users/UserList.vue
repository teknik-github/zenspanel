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
