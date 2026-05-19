<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useDomainsStore } from '@/stores/domains'
import { useDatabasesStore } from '@/stores/databases'
import { useUsageStore } from '@/stores/usage'

const auth = useAuthStore()
const domainsStore = useDomainsStore()
const databasesStore = useDatabasesStore()
const usageStore = useUsageStore()

// Hide skeleton placeholders once the first round of fetches resolves.
// We don't fail-hard on individual errors — usage/usage_store keeps
// zero values which still render fine.
const loaded = ref(false)

onMounted(async () => {
  await Promise.all([
    auth.fetchMe(),
    domainsStore.fetch(),
    databasesStore.fetch(),
    usageStore.fetch(),
  ])
  loaded.value = true
})
</script>

<template>
  <div class="space-y-5">
    <div>
      <h1 class="text-lg font-semibold text-gray-800">Dashboard</h1>
      <p class="text-xs text-gray-400 mt-0.5">Overview of your hosting account</p>
    </div>

    <!-- Resource usage cards -->
    <div v-if="!loaded" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
      <div v-for="i in 5" :key="i" class="bg-white border border-gray-200 rounded-lg p-4 animate-pulse">
        <div class="h-3 w-16 bg-gray-100 rounded" />
        <div class="h-6 w-20 bg-gray-100 rounded mt-2" />
        <div class="h-1.5 w-full bg-gray-100 rounded-full mt-3" />
      </div>
    </div>

    <div v-else class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-indigo-50 text-indigo-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">Domains</div>
        </div>
        <div class="text-xl font-bold text-gray-800 mt-2">
          {{ usageStore.usage.domains.used }}
          <span class="text-sm font-normal text-gray-400">/ {{ usageStore.usage.domains.max }}</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-indigo-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.domains.max ? Math.min(usageStore.usage.domains.used / usageStore.usage.domains.max * 100, 100) + '%' : '0%' }" />
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-blue-50 text-blue-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">Databases</div>
        </div>
        <div class="text-xl font-bold text-gray-800 mt-2">
          {{ usageStore.usage.databases.used }}
          <span class="text-sm font-normal text-gray-400">/ {{ usageStore.usage.databases.max }}</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-blue-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.databases.max ? Math.min(usageStore.usage.databases.used / usageStore.usage.databases.max * 100, 100) + '%' : '0%' }" />
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-purple-50 text-purple-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">CPU</div>
        </div>
        <div class="text-xl font-bold text-gray-800 mt-2">
          {{ usageStore.usage.cpu.used.toFixed(1) }}<span class="text-sm font-normal text-gray-400">%</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-purple-500 h-1.5 rounded-full transition-all"
            :style="{ width: Math.min(usageStore.usage.cpu.used, 100) + '%' }" />
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-emerald-50 text-emerald-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 12H2"/><path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">Disk</div>
        </div>
        <div class="text-xl font-bold text-gray-800 mt-2">
          {{ (usageStore.usage.disk.used / 1073741824).toFixed(1) }}
          <span class="text-sm font-normal text-gray-400">/ {{ (usageStore.usage.disk.max / 1073741824).toFixed(0) }} GB</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-emerald-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.disk.max ? Math.min(usageStore.usage.disk.used / usageStore.usage.disk.max * 100, 100) + '%' : '0%' }" />
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4 col-span-2 sm:col-span-1">
        <div class="flex items-center gap-2">
          <div class="w-7 h-7 rounded-md bg-amber-50 text-amber-600 flex items-center justify-center flex-shrink-0">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 12h.01M10 12h.01M14 12h.01M18 12h.01"/>
            </svg>
          </div>
          <div class="text-[10px] text-gray-400 uppercase tracking-wide">RAM</div>
        </div>
        <div class="text-xl font-bold text-gray-800 mt-2">
          {{ (usageStore.usage.ram.used / 1073741824).toFixed(1) }}
          <span class="text-sm font-normal text-gray-400">/ {{ (usageStore.usage.ram.max / 1073741824).toFixed(0) }} GB</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5 overflow-hidden">
          <div class="bg-amber-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.ram.max ? Math.min(usageStore.usage.ram.used / usageStore.usage.ram.max * 100, 100) + '%' : '0%' }" />
        </div>
      </div>
    </div>

    <!-- Domains table + Quick Actions. Stacks on small screens. -->
    <div class="grid grid-cols-1 lg:grid-cols-[1fr_220px] gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold text-gray-800 text-sm">My Domains</h2>
          <router-link to="/domains"
            class="flex items-center gap-1 bg-indigo-600 text-white text-xs px-3 py-1.5 rounded-md hover:bg-indigo-700 transition-colors">
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            Add Domain
          </router-link>
        </div>

        <div v-if="!loaded" class="space-y-2">
          <div v-for="i in 3" :key="i" class="h-6 bg-gray-50 rounded animate-pulse" />
        </div>

        <div v-else-if="!domainsStore.domains.length" class="flex flex-col items-center justify-center py-10 text-center">
          <div class="w-10 h-10 rounded-full bg-indigo-50 text-indigo-500 flex items-center justify-center mb-2">
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
          </div>
          <p class="text-xs font-medium text-gray-700">No domains yet</p>
          <p class="text-[11px] text-gray-400 mt-0.5">Add your first domain to get started</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="w-full text-xs min-w-[400px]">
            <thead>
              <tr class="text-gray-400 border-b border-gray-100">
                <th class="text-left pb-2 font-medium">Domain</th>
                <th class="text-left pb-2 font-medium">PHP</th>
                <th class="text-left pb-2 font-medium">SSL</th>
                <th class="text-left pb-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="d in domainsStore.domains" :key="d.id" class="border-b border-gray-50">
                <td class="py-2 font-medium text-gray-700">{{ d.domain }}</td>
                <td class="py-2 text-gray-500">{{ d.php_version }}</td>
                <td class="py-2">
                  <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                    :class="d.ssl_type === 'none' ? 'bg-gray-100 text-gray-500' : 'bg-green-100 text-green-700'">
                    {{ d.ssl_type === 'none' ? 'No SSL' : d.ssl_type }}
                  </span>
                </td>
                <td class="py-2">
                  <span class="px-2 py-0.5 rounded text-[10px] font-medium"
                    :class="d.status === 'active' ? 'bg-green-100 text-green-700' : 'bg-yellow-100 text-yellow-700'">
                    {{ d.status }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Quick Actions</h2>
        <div class="space-y-2">
          <router-link to="/databases"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 hover:border-gray-300 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
            </svg>
            New Database
          </router-link>
          <router-link v-if="auth.user?.terminal_enabled" to="/terminal"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 hover:border-gray-300 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
            Open Terminal
          </router-link>
          <router-link v-if="auth.user?.backup_enabled" to="/backups"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 hover:border-gray-300 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
            Create Backup
          </router-link>
          <a href="/filebrowser/files/" target="_blank" rel="noopener"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 hover:border-gray-300 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            File Manager
          </a>
          <router-link to="/ssl-manager"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 hover:border-gray-300 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            SSL Manager
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
