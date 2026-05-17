<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push('/login')
}

const navGroups = computed(() => [
  {
    label: 'Overview',
    items: [
      { path: '/dashboard', label: 'Dashboard', icon: 'grid', show: true },
    ],
  },
  {
    label: 'Websites',
    items: [
      { path: '/domains', label: 'Domains', icon: 'globe', show: true },
      { path: '/ssl-manager', label: 'SSL Manager', icon: 'lock', show: true },
      { path: '/php-settings', label: 'PHP Settings', icon: 'code', show: true },
    ],
  },
  {
    label: 'Database',
    items: [
      { path: '/databases', label: 'Databases', icon: 'database', show: true },
    ],
  },
  {
    label: 'Files & Tools',
    items: [
      { path: '/file-manager', label: 'File Manager', icon: 'folder', show: true },
      { path: '/terminal', label: 'Terminal', icon: 'terminal', show: auth.user?.terminal_enabled },
      { path: '/backups', label: 'Backups', icon: 'upload-cloud', show: auth.user?.backup_enabled },
    ],
  },
])
</script>

<template>
  <div class="flex h-screen bg-gray-50 font-sans text-sm">
    <!-- Sidebar -->
    <aside class="w-[200px] bg-white border-r border-gray-200 flex flex-col flex-shrink-0">
      <div class="px-4 py-4 border-b border-gray-200">
        <div class="flex items-center gap-2 text-indigo-600 font-bold text-base">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          ZensPanel
        </div>
        <div class="text-xs text-gray-400 mt-0.5 pl-7">User Area</div>
      </div>

      <nav class="flex-1 py-2 overflow-y-auto">
        <template v-for="group in navGroups" :key="group.label">
          <div class="px-3 pt-3 pb-1 text-[10px] uppercase tracking-widest text-gray-400">
            {{ group.label }}
          </div>
          <template v-for="item in group.items" :key="item.path">
            <router-link
              v-if="item.show"
              :to="item.path"
              class="flex items-center gap-2 px-4 py-2 text-gray-500 border-l-[3px] border-transparent hover:bg-gray-50 hover:text-gray-700 transition-colors"
              :class="route.path === item.path ? 'bg-indigo-50 border-indigo-600 text-indigo-600 font-medium' : ''"
            >
              <svg v-if="item.icon === 'grid'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>
              </svg>
              <svg v-else-if="item.icon === 'globe'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
              </svg>
              <svg v-else-if="item.icon === 'lock'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
              <svg v-else-if="item.icon === 'code'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>
              </svg>
              <svg v-else-if="item.icon === 'database'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
              </svg>
              <svg v-else-if="item.icon === 'folder'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <svg v-else-if="item.icon === 'terminal'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              <svg v-else-if="item.icon === 'upload-cloud'" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
              </svg>
              {{ item.label }}
            </router-link>
          </template>
        </template>
      </nav>

      <div class="px-4 py-3 border-t border-gray-200 flex items-center gap-2">
        <div class="w-7 h-7 bg-indigo-600 rounded-full flex items-center justify-center text-white text-xs font-bold flex-shrink-0">
          {{ auth.user?.username?.[0]?.toUpperCase() || 'U' }}
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-gray-800 text-xs font-medium truncate">{{ auth.user?.username }}</div>
          <div class="text-gray-400 text-[10px]">User</div>
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
      <header class="bg-white border-b border-gray-200 px-5 py-2.5 flex items-center gap-3 flex-shrink-0">
        <div class="flex-1 max-w-sm bg-gray-50 border border-gray-200 rounded-md px-3 py-1.5 flex items-center gap-2">
          <svg class="w-3.5 h-3.5 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <span class="text-gray-400 text-xs">Search domains, databases...</span>
          <span class="ml-auto bg-gray-200 text-gray-400 text-[10px] px-1.5 py-0.5 rounded">⌘K</span>
        </div>
      </header>
      <main class="flex-1 overflow-y-auto p-5">
        <router-view />
      </main>
    </div>
  </div>
</template>
