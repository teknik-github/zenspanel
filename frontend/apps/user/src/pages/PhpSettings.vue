<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { domainsApi } from '@/api/domains'
import { phpVersionsApi } from '@/api/phpVersions'
import { phpExtensionsApi } from '@/api/phpExtensions'
import { usersApi } from '@/api/users'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const domains = ref<any[]>([])
const phpVersions = ref<any[]>([])
const saving = ref<number | null>(null)
const saved = ref<number | null>(null)

const shellPHP = ref<string>('8.3')
const shellSaving = ref(false)
const shellSaved = ref(false)

// Extensions
const extensions = ref<any[]>([])
const extLoading = ref(false)
const extSaving = ref<string | null>(null) // "<name>-<ver>"

onMounted(async () => {
  const [domainsRes, phpRes] = await Promise.all([
    domainsApi.list(),
    phpVersionsApi.listEnabled(),
  ])
  domains.value = domainsRes.data.data || []
  phpVersions.value = phpRes.data.data || []
  shellPHP.value = auth.user?.php_version || '8.3'
  await loadExtensions(shellPHP.value)
})

async function loadExtensions(ver: string) {
  extLoading.value = true
  try {
    const res = await phpExtensionsApi.userList(ver)
    extensions.value = res.data.data || []
  } finally {
    extLoading.value = false
  }
}

// Reload extensions when shell PHP version changes
watch(shellPHP, (ver) => {
  loadExtensions(ver)
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

async function saveShellPHP() {
  if (!auth.user) return
  shellSaving.value = true
  try {
    await usersApi.update(auth.user.id, { php_version: shellPHP.value })
    await auth.fetchMe()
    shellSaved.value = true
    setTimeout(() => { shellSaved.value = false }, 2000)
  } finally {
    shellSaving.value = false
  }
}

async function toggleExtension(ext: any) {
  // V20: cannot enable admin-disabled extension
  if (!ext.admin_enabled && !ext.user_enabled) return
  const key = `${ext.name}-${ext.php_version}`
  extSaving.value = key
  try {
    await phpExtensionsApi.userUpdate(ext.name, ext.php_version, !ext.user_enabled)
    ext.user_enabled = !ext.user_enabled
  } finally {
    extSaving.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <h1 class="text-lg font-semibold text-gray-800">PHP Settings</h1>

    <!-- Shell PHP Version -->
    <div class="bg-white border border-gray-200 rounded-lg p-4">
      <div class="flex items-start justify-between gap-4">
        <div>
          <h2 class="text-sm font-semibold text-gray-800">Shell PHP Version</h2>
          <p class="text-xs text-gray-500 mt-1">
            Used by the terminal — <code class="px-1 bg-gray-100 rounded">php</code>,
            <code class="px-1 bg-gray-100 rounded">composer</code>, and
            <code class="px-1 bg-gray-100 rounded">artisan</code> resolve to this version.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <select v-model="shellPHP"
            class="border border-gray-200 rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-indigo-500">
            <option v-for="v in phpVersions" :key="v.version" :value="v.version">PHP {{ v.version }}</option>
          </select>
          <button @click="saveShellPHP" :disabled="shellSaving"
            class="text-xs bg-indigo-600 text-white px-3 py-1 rounded hover:bg-indigo-700 disabled:opacity-50">
            {{ shellSaving ? 'Saving...' : 'Save' }}
          </button>
          <span v-if="shellSaved" class="text-green-600 text-xs">Saved!</span>
        </div>
      </div>
    </div>

    <!-- PHP Extensions -->
    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200">
        <h2 class="text-sm font-semibold text-gray-800">Extensions <span class="text-gray-400 font-normal">— PHP {{ shellPHP }}</span></h2>
        <p class="text-xs text-gray-500 mt-0.5">Toggle extensions for your PHP {{ shellPHP }} environment. Admin-disabled extensions cannot be enabled.</p>
      </div>

      <div v-if="extLoading" class="p-4 space-y-2">
        <div v-for="i in 6" :key="i" class="h-8 bg-gray-50 rounded animate-pulse"></div>
      </div>

      <div v-else-if="!extensions.length" class="px-4 py-8 text-center text-xs text-gray-400">
        No extensions found for PHP {{ shellPHP }}.
      </div>

      <table v-else class="w-full text-xs">
        <tbody>
          <tr v-for="ext in extensions" :key="ext.id"
            class="border-b border-gray-50 last:border-0 hover:bg-gray-50"
            :class="{ 'opacity-50': !ext.admin_enabled }">
            <td class="px-4 py-2.5 font-mono text-gray-700">{{ ext.name }}</td>
            <td class="px-4 py-2.5">
              <span v-if="!ext.admin_enabled" class="text-[10px] text-gray-400 italic">disabled by admin</span>
              <span v-else class="px-2 py-0.5 rounded text-[10px] font-medium"
                :class="ext.user_enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">
                {{ ext.user_enabled ? 'Enabled' : 'Disabled' }}
              </span>
            </td>
            <td class="px-4 py-2.5 text-right">
              <button
                v-if="ext.admin_enabled"
                @click="toggleExtension(ext)"
                :disabled="extSaving === `${ext.name}-${ext.php_version}`"
                class="text-xs px-3 py-1 rounded border transition-colors disabled:opacity-50"
                :class="ext.user_enabled
                  ? 'text-red-600 border-red-200 hover:bg-red-50'
                  : 'text-green-600 border-green-200 hover:bg-green-50'">
                {{ extSaving === `${ext.name}-${ext.php_version}` ? '...' : ext.user_enabled ? 'Disable' : 'Enable' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Per-domain PHP version -->
    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200">
        <h2 class="text-sm font-semibold text-gray-800">Domain PHP Version</h2>
      </div>
      <table class="w-full text-xs">
        <thead class="border-b border-gray-200">
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
