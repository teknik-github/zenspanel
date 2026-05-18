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

onMounted(async () => {
  await Promise.all([
    auth.fetchMe(),
    domainsStore.fetch(),
    databasesStore.fetch(),
    usageStore.fetch(),
  ])
})

const usage = usageStore.usage
</script>

<template>
  <div class="space-y-5">
    <h1 class="text-lg font-semibold text-gray-800">Dashboard</h1>

    <!-- Resource usage -->
    <div class="grid grid-cols-5 gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Domains</div>
        <div class="text-xl font-bold text-gray-800 mt-1">
          {{ usageStore.usage.domains.used }}
          <span class="text-sm font-normal text-gray-400">/ {{ usageStore.usage.domains.max }}</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-indigo-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.domains.max ? (usageStore.usage.domains.used / usageStore.usage.domains.max * 100) + '%' : '0%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Databases</div>
        <div class="text-xl font-bold text-gray-800 mt-1">
          {{ usageStore.usage.databases.used }}
          <span class="text-sm font-normal text-gray-400">/ {{ usageStore.usage.databases.max }}</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-indigo-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.databases.max ? (usageStore.usage.databases.used / usageStore.usage.databases.max * 100) + '%' : '0%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">CPU Usage</div>
        <div class="text-xl font-bold text-gray-800 mt-1">
          {{ usageStore.usage.cpu.used.toFixed(1) }}<span class="text-sm font-normal text-gray-400">%</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-purple-500 h-1.5 rounded-full transition-all"
            :style="{ width: Math.min(usageStore.usage.cpu.used, 100) + '%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">Disk Usage</div>
        <div class="text-xl font-bold text-gray-800 mt-1">
          {{ (usageStore.usage.disk.used / 1073741824).toFixed(1) }}
          <span class="text-sm font-normal text-gray-400">/ {{ (usageStore.usage.disk.max / 1073741824).toFixed(0) }} GB</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-emerald-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.disk.max ? (usageStore.usage.disk.used / usageStore.usage.disk.max * 100) + '%' : '0%' }"></div>
        </div>
      </div>
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="text-xs text-gray-400 uppercase tracking-wide">RAM Usage</div>
        <div class="text-xl font-bold text-gray-800 mt-1">
          {{ (usageStore.usage.ram.used / 1073741824).toFixed(1) }}
          <span class="text-sm font-normal text-gray-400">/ {{ (usageStore.usage.ram.max / 1073741824).toFixed(0) }} GB</span>
        </div>
        <div class="mt-2 bg-gray-100 rounded-full h-1.5">
          <div class="bg-amber-500 h-1.5 rounded-full transition-all"
            :style="{ width: usageStore.usage.ram.max ? (usageStore.usage.ram.used / usageStore.usage.ram.max * 100) + '%' : '0%' }"></div>
        </div>
      </div>
    </div>

    <!-- Domains table + Quick Actions -->
    <div class="grid grid-cols-[1fr_200px] gap-4">
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <div class="flex items-center justify-between mb-3">
          <h2 class="font-semibold text-gray-800 text-sm">My Domains</h2>
          <router-link to="/domains"
            class="flex items-center gap-1 bg-indigo-600 text-white text-xs px-3 py-1.5 rounded-md hover:bg-indigo-700">
            <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            Add Domain
          </router-link>
        </div>
        <table class="w-full text-xs">
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
            <tr v-if="!domainsStore.domains.length">
              <td colspan="4" class="py-4 text-center text-gray-400">No domains yet</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Quick Actions -->
      <div class="bg-white border border-gray-200 rounded-lg p-4">
        <h2 class="font-semibold text-gray-800 text-sm mb-3">Quick Actions</h2>
        <div class="space-y-2">
          <router-link to="/databases"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
            </svg>
            New Database
          </router-link>
          <router-link v-if="auth.user?.terminal_enabled" to="/terminal"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
            Open Terminal
          </router-link>
          <router-link v-if="auth.user?.backup_enabled" to="/backups"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
            Create Backup
          </router-link>
          <router-link to="/file-manager"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 transition-colors">
            <svg class="w-3.5 h-3.5 text-indigo-500 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            File Manager
          </router-link>
          <router-link to="/ssl-manager"
            class="flex items-center gap-2 w-full bg-gray-50 border border-gray-200 rounded-md px-3 py-2 text-xs text-gray-600 hover:bg-gray-100 transition-colors">
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
