<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { usersApi } from '@/api/users'
import { packagesApi } from '@/api/packages'
import { domainsApi } from '@/api/domains'
import { useRoute, useRouter } from 'vue-router'
import { useToast, useConfirm } from '@/notify'
import client from '@/api/client'

const route = useRoute()
const router = useRouter()
const { success: toastSuccess, error: toastError } = useToast()
const { confirm } = useConfirm()

const user = ref<any>(null)
const packages = ref<any[]>([])
const usage = ref<any>(null)
const domains = ref<any[]>([])
const ftpAccounts = ref<any[]>([])
const form = ref<any>({})
const loading = ref(false)
const saved = ref(false)
const confirmDelete = ref(false)
const suspendLoading = ref(false)

const uid = Number(route.params.id)

onMounted(async () => {
  const [userRes, pkgRes] = await Promise.all([
    usersApi.get(uid),
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

  // Load usage, domains, FTP in parallel (best-effort)
  const [usageRes, domainsRes, ftpRes] = await Promise.allSettled([
    usersApi.getUsage(uid),
    domainsApi.list({ user_id: uid }),
    client.get('/ftp', { params: { user_id: uid } }),
  ])
  if (usageRes.status === 'fulfilled') usage.value = usageRes.value.data.usage
  if (domainsRes.status === 'fulfilled') domains.value = domainsRes.value.data.data || []
  if (ftpRes.status === 'fulfilled') ftpAccounts.value = ftpRes.value.data.data || []
})

function formatBytes(bytes: number) {
  if (!bytes) return '0 B'
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  return (bytes / 1024).toFixed(0) + ' KB'
}

function usagePct(used: number, max: number) {
  if (!max) return 0
  return Math.min(100, Math.round((used / max) * 100))
}

function barColor(pct: number) {
  if (pct >= 90) return 'bg-red-500'
  if (pct >= 70) return 'bg-amber-500'
  return 'bg-indigo-500'
}

// Soft enforcement warnings — existing resources over new package limits.
// We block new creates but don't remove existing ones (Option A).
const overLimitWarnings = computed(() => {
  if (!usage.value) return []
  const warnings: string[] = []
  const d = usage.value.domains
  const db = usage.value.databases
  if (d?.max > 0 && d?.used > d?.max)
    warnings.push(`Domains: ${d.used} active but package allows ${d.max}. New domains blocked until below limit.`)
  if (db?.max > 0 && db?.used > db?.max)
    warnings.push(`Databases: ${db.used} active but package allows ${db.max}. New databases blocked until below limit.`)
  if (usage.value.disk?.max > 0 && usage.value.disk?.used > usage.value.disk?.max)
    warnings.push(`Disk: using ${formatBytes(usage.value.disk.used)} but package limit is ${formatBytes(usage.value.disk.max)}. Writes will fail (EDQUOT).`)
  return warnings
})

async function save() {
  loading.value = true
  try {
    await usersApi.update(uid, form.value)
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } finally {
    loading.value = false
  }
}

async function deleteUser() {
  await usersApi.delete(uid)
  router.push('/users')
}

async function loginAs() {
  const res = await usersApi.impersonate(uid)
  const token = res.data.token
  window.open(`${window.location.origin}/#impersonate=${encodeURIComponent(token)}`, '_blank')
}

async function suspendUser() {
  const ok = await confirm(`Suspend ${user.value.username}? This will disable all websites, FTP, and revoke all active sessions immediately.`)
  if (!ok) return
  suspendLoading.value = true
  try {
    await usersApi.suspend(uid)
    user.value = { ...user.value, status: 'suspended' }
    toastSuccess('User suspended — all sessions revoked')
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to suspend')
  } finally {
    suspendLoading.value = false
  }
}

async function unsuspendUser() {
  suspendLoading.value = true
  try {
    await usersApi.unsuspend(uid)
    user.value = { ...user.value, status: 'active' }
    toastSuccess('User unsuspended — services restored')
  } catch (e: any) {
    toastError(e?.response?.data?.error || 'Failed to unsuspend')
  } finally {
    suspendLoading.value = false
  }
}

function pkgName(id: any) {
  const pkg = packages.value.find(p => p.id === id || p.id === id?.Int64 || p.id === Number(id))
  return pkg?.name ?? '—'
}
</script>

<template>
  <div v-if="user" class="space-y-5 max-w-4xl">

    <!-- Header -->
    <div class="flex items-center gap-3 flex-wrap">
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
      <span class="text-xs text-gray-400">{{ user.email }}</span>
      <div class="ml-auto flex gap-2 flex-wrap">
        <button @click="loginAs"
          class="text-xs text-purple-600 border border-purple-200 px-3 py-1.5 rounded-md hover:bg-purple-50">
          Login as User
        </button>
        <button v-if="user.status === 'active'" @click="suspendUser" :disabled="suspendLoading"
          class="text-xs text-orange-600 border border-orange-200 px-3 py-1.5 rounded-md hover:bg-orange-50 disabled:opacity-50">
          {{ suspendLoading ? '...' : 'Suspend' }}
        </button>
        <button v-else @click="unsuspendUser" :disabled="suspendLoading"
          class="text-xs text-green-600 border border-green-200 px-3 py-1.5 rounded-md hover:bg-green-50 disabled:opacity-50">
          {{ suspendLoading ? '...' : 'Unsuspend' }}
        </button>
        <button @click="confirmDelete = true"
          class="text-xs text-red-600 border border-red-200 px-3 py-1.5 rounded-md hover:bg-red-50">
          Delete
        </button>
      </div>
    </div>

    <!-- Suspended banner -->
    <div v-if="user.status === 'suspended'"
      class="bg-red-50 border border-red-200 rounded-lg px-4 py-3 flex items-center gap-3">
      <svg class="w-4 h-4 text-red-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
      </svg>
      <p class="text-sm text-red-700">Account suspended — all websites, FTP, and sessions are disabled.</p>
    </div>

    <!-- Over-limit warnings (soft enforcement) -->
    <div v-if="overLimitWarnings.length"
      class="bg-amber-50 border border-amber-200 rounded-lg px-4 py-3 space-y-1">
      <div class="flex items-center gap-2 mb-1">
        <svg class="w-4 h-4 text-amber-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <p class="text-xs font-semibold text-amber-800">Over package limits — existing resources kept, new ones blocked</p>
      </div>
      <ul class="space-y-0.5">
        <li v-for="w in overLimitWarnings" :key="w" class="text-xs text-amber-700 pl-6">· {{ w }}</li>
      </ul>
    </div>

    <!-- Resource usage -->
    <div v-if="usage" class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <!-- CPU -->
      <div class="bg-white border border-gray-200 rounded-lg p-3">
        <p class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">CPU</p>
        <p class="text-lg font-semibold text-gray-800">{{ usage.cpu?.used?.toFixed(1) ?? 0 }}%</p>
        <div class="mt-1.5 h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <div class="h-full rounded-full transition-all" :class="barColor(usage.cpu?.used ?? 0)"
            :style="`width:${Math.min(100, usage.cpu?.used ?? 0)}%`"></div>
        </div>
      </div>
      <!-- RAM -->
      <div class="bg-white border border-gray-200 rounded-lg p-3">
        <p class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">RAM</p>
        <p class="text-lg font-semibold text-gray-800">{{ formatBytes(usage.ram?.used ?? 0) }}</p>
        <p class="text-[10px] text-gray-400">of {{ formatBytes(usage.ram?.max ?? 0) }}</p>
        <div class="mt-1.5 h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <div class="h-full rounded-full transition-all" :class="barColor(usagePct(usage.ram?.used, usage.ram?.max))"
            :style="`width:${usagePct(usage.ram?.used, usage.ram?.max)}%`"></div>
        </div>
      </div>
      <!-- Disk -->
      <div class="bg-white border border-gray-200 rounded-lg p-3">
        <p class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Disk</p>
        <p class="text-lg font-semibold text-gray-800">{{ formatBytes(usage.disk?.used ?? 0) }}</p>
        <p class="text-[10px] text-gray-400">of {{ formatBytes(usage.disk?.max ?? 0) }}</p>
        <div class="mt-1.5 h-1.5 bg-gray-100 rounded-full overflow-hidden">
          <div class="h-full rounded-full transition-all" :class="barColor(usagePct(usage.disk?.used, usage.disk?.max))"
            :style="`width:${usagePct(usage.disk?.used, usage.disk?.max)}%`"></div>
        </div>
      </div>
      <!-- Domains / DBs -->
      <div class="bg-white border border-gray-200 rounded-lg p-3">
        <p class="text-[10px] text-gray-400 uppercase tracking-wide mb-1">Resources</p>
        <div class="space-y-1 text-xs text-gray-600">
          <div class="flex justify-between">
            <span>Domains</span>
            <span class="font-medium">{{ usage.domains?.used ?? 0 }} / {{ usage.domains?.max ?? '∞' }}</span>
          </div>
          <div class="flex justify-between">
            <span>Databases</span>
            <span class="font-medium">{{ usage.databases?.used ?? 0 }} / {{ usage.databases?.max ?? '∞' }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">

      <!-- User info + edit -->
      <div class="bg-white border border-gray-200 rounded-lg p-5 space-y-4">
        <h2 class="text-sm font-semibold text-gray-800">Account Settings</h2>
        <div class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Username</label>
              <input :value="user.username" disabled
                class="w-full border border-gray-100 rounded-md px-3 py-2 text-sm bg-gray-50 text-gray-400" />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-600 mb-1">Linux UID</label>
              <input :value="user.linux_uid" disabled
                class="w-full border border-gray-100 rounded-md px-3 py-2 text-sm bg-gray-50 text-gray-400" />
            </div>
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
          <div class="flex items-center gap-4 pt-1">
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.terminal_enabled" class="rounded border-gray-300 text-indigo-600" />
              Terminal
            </label>
            <label class="flex items-center gap-2 cursor-pointer text-xs text-gray-600">
              <input type="checkbox" v-model="form.backup_enabled" class="rounded border-gray-300 text-indigo-600" />
              Backup
            </label>
          </div>
        </div>
        <div class="flex items-center gap-3 pt-1">
          <button @click="save" :disabled="loading"
            class="bg-indigo-600 text-white text-sm px-4 py-2 rounded-md hover:bg-indigo-700 disabled:opacity-50">
            {{ loading ? 'Saving...' : 'Save Changes' }}
          </button>
          <span v-if="saved" class="text-green-600 text-xs">Saved!</span>
        </div>

        <!-- Meta info -->
        <div class="border-t border-gray-100 pt-3 space-y-1 text-xs text-gray-500">
          <div class="flex justify-between">
            <span>Role</span><span class="font-medium text-gray-700 capitalize">{{ user.role }}</span>
          </div>
          <div class="flex justify-between">
            <span>PHP Version</span><span class="font-medium text-gray-700">{{ user.php_version || '—' }}</span>
          </div>
          <div class="flex justify-between">
            <span>2FA</span>
            <span :class="user.totp_enabled ? 'text-green-600 font-medium' : 'text-gray-400'">
              {{ user.totp_enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>
          <div class="flex justify-between">
            <span>Created</span><span class="font-medium text-gray-700">{{ new Date(user.created_at).toLocaleDateString() }}</span>
          </div>
        </div>
      </div>

      <!-- Domains -->
      <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-gray-800">Domains</h2>
          <span class="text-xs text-gray-400">{{ domains.length }} total</span>
        </div>
        <div v-if="!domains.length" class="px-4 py-6 text-center text-xs text-gray-400">No domains</div>
        <div v-else class="divide-y divide-gray-50 max-h-64 overflow-y-auto">
          <div v-for="d in domains" :key="d.id" class="px-4 py-2.5 flex items-center justify-between">
            <div>
              <p class="text-xs font-medium text-gray-700">{{ d.domain }}</p>
              <p class="text-[10px] text-gray-400">PHP {{ d.php_version }} · {{ d.ssl_type === 'none' ? 'No SSL' : d.ssl_type }}</p>
            </div>
            <span class="text-[10px] px-2 py-0.5 rounded font-medium"
              :class="d.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'">
              {{ d.status }}
            </span>
          </div>
        </div>
      </div>

      <!-- FTP Accounts -->
      <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-gray-800">FTP Accounts</h2>
          <span class="text-xs text-gray-400">{{ ftpAccounts.length }} total</span>
        </div>
        <div v-if="!ftpAccounts.length" class="px-4 py-6 text-center text-xs text-gray-400">No FTP accounts</div>
        <div v-else class="divide-y divide-gray-50 max-h-48 overflow-y-auto">
          <div v-for="a in ftpAccounts" :key="a.id" class="px-4 py-2.5 flex items-center justify-between">
            <div>
              <p class="text-xs font-mono font-medium text-gray-700">{{ a.ftp_username }}</p>
              <p class="text-[10px] text-gray-400 truncate max-w-[200px]">{{ a.home_dir }}</p>
            </div>
            <span class="text-[10px] px-2 py-0.5 rounded font-medium"
              :class="a.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
              {{ a.enabled ? 'Active' : 'Disabled' }}
            </span>
          </div>
        </div>
      </div>

      <!-- Disk breakdown -->
      <div v-if="usage?.disk" class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="text-sm font-semibold text-gray-800 mb-3">Disk Breakdown</h2>
        <div class="space-y-2 text-xs">
          <div class="flex justify-between text-gray-600">
            <span>Files</span><span class="font-medium text-gray-800">{{ formatBytes(usage.disk.files ?? 0) }}</span>
          </div>
          <div class="flex justify-between text-gray-600">
            <span>Databases</span><span class="font-medium text-gray-800">{{ formatBytes(usage.disk.db ?? 0) }}</span>
          </div>
          <div class="flex justify-between text-gray-600 border-t border-gray-100 pt-2">
            <span class="font-medium">Total</span>
            <span class="font-semibold text-gray-800">
              {{ formatBytes(usage.disk.used ?? 0) }}
              <span class="text-gray-400 font-normal"> / {{ formatBytes(usage.disk.max ?? 0) }}</span>
            </span>
          </div>
        </div>
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
