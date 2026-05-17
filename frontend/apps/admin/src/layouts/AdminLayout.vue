<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push('/login')
}

const navGroups = [
  {
    label: 'Overview',
    items: [
      { path: '/dashboard', label: 'Dashboard', icon: 'grid' },
      { path: '/resource-monitor', label: 'Resource Monitor', icon: 'activity' },
    ],
  },
  {
    label: 'Management',
    items: [
      { path: '/users', label: 'Users', icon: 'users' },
      { path: '/packages', label: 'Packages', icon: 'package' },
      { path: '/domains', label: 'Domains', icon: 'globe' },
      { path: '/databases', label: 'Databases', icon: 'database' },
    ],
  },
  {
    label: 'Server',
    items: [
      { path: '/php-versions', label: 'PHP Versions', icon: 'code' },
      { path: '/ssl-manager', label: 'SSL Manager', icon: 'lock' },
      { path: '/backups', label: 'Backups', icon: 'upload-cloud' },
      { path: '/api-keys', label: 'API Keys', icon: 'key' },
    ],
  },
  {
    label: 'System',
    items: [
      { path: '/audit-logs', label: 'Audit Logs', icon: 'file-text' },
      { path: '/settings', label: 'Settings', icon: 'settings' },
    ],
  },
]
</script>

<template>
  <div class="flex h-screen bg-gray-50 font-sans text-sm">
    <!-- Sidebar -->
    <aside class="w-[200px] bg-white border-r border-gray-200 flex flex-col flex-shrink-0">
      <!-- Logo -->
      <div class="px-4 py-4 border-b border-gray-200">
        <div class="flex items-center gap-2 text-indigo-600 font-bold text-base">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          ZensPanel
        </div>
        <div class="text-xs text-gray-400 mt-0.5 pl-7">Admin Area</div>
      </div>

      <!-- Nav -->
      <nav class="flex-1 py-2 overflow-y-auto">
        <template v-for="group in navGroups" :key="group.label">
          <div class="px-3 pt-3 pb-1 text-[10px] uppercase tracking-widest text-gray-400">
            {{ group.label }}
          </div>
          <router-link
            v-for="item in group.items"
            :key="item.path"
            :to="item.path"
            class="flex items-center gap-2 px-4 py-2 text-gray-500 border-l-[3px] border-transparent hover:bg-gray-50 hover:text-gray-700 transition-colors"
            :class="route.path.startsWith(item.path) ? 'bg-indigo-50 border-indigo-600 text-indigo-600 font-medium' : ''"
          >
            <!-- Grid icon -->
            <svg v-if="item.icon === 'grid'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>
            </svg>
            <svg v-else-if="item.icon === 'activity'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
            <svg v-else-if="item.icon === 'users'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            <svg v-else-if="item.icon === 'package'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
            </svg>
            <svg v-else-if="item.icon === 'globe'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
            </svg>
            <svg v-else-if="item.icon === 'database'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
            </svg>
            <svg v-else-if="item.icon === 'code'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
            </svg>
            <svg v-else-if="item.icon === 'lock'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <svg v-else-if="item.icon === 'upload-cloud'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
            </svg>
            <svg v-else-if="item.icon === 'key'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
            </svg>
            <svg v-else-if="item.icon === 'file-text'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/>
            </svg>
            <svg v-else-if="item.icon === 'settings'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14M4.93 4.93a10 10 0 0 0 0 14.14"/>
            </svg>
            {{ item.label }}
          </router-link>
        </template>
      </nav>

      <!-- User -->
      <div class="px-4 py-3 border-t border-gray-200 flex items-center gap-2">
        <div class="w-7 h-7 bg-indigo-600 rounded-full flex items-center justify-center text-white text-xs font-bold flex-shrink-0">
          {{ auth.user?.username?.[0]?.toUpperCase() || 'A' }}
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-gray-800 text-xs font-medium truncate">{{ auth.user?.username }}</div>
          <div class="text-gray-400 text-[10px]">Super Admin</div>
        </div>
        <button @click="logout" class="text-gray-400 hover:text-gray-600">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
          </svg>
        </button>
      </div>
    </aside>

    <!-- Main -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Top bar -->
      <header class="bg-white border-b border-gray-200 px-5 py-2.5 flex items-center gap-3 flex-shrink-0">
        <div class="flex-1 max-w-sm bg-gray-50 border border-gray-200 rounded-md px-3 py-1.5 flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <span class="text-gray-400 text-xs">Search users, domains, databases...</span>
          <span class="ml-auto bg-gray-200 text-gray-400 text-[10px] px-1.5 py-0.5 rounded">⌘K</span>
        </div>
        <div class="ml-auto flex items-center gap-3">
          <button class="relative text-gray-400 hover:text-gray-600">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/>
            </svg>
          </button>
          <span class="text-gray-400 text-[10px] border border-gray-200 px-1.5 py-0.5 rounded">v1.0.0</span>
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1 overflow-y-auto p-5">
        <router-view />
      </main>
    </div>
  </div>
</template>
