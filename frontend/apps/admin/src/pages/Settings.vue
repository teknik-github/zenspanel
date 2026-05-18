<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { systemApi, type SystemStats } from '@/api/system'
import { phpVersionsApi } from '@/api/phpVersions'

const stats = ref<SystemStats | null>(null)
const phpVersions = ref<any[]>([])

onMounted(async () => {
  const [s, p] = await Promise.all([systemApi.stats(), phpVersionsApi.list()])
  stats.value = s.data
  phpVersions.value = p.data.data || []
})
</script>

<template>
  <div class="space-y-5">
    <h1 class="text-lg font-semibold text-gray-800">Settings</h1>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">Server</h2>
      <dl class="grid grid-cols-2 gap-x-6 gap-y-2 text-xs">
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Total RAM</dt>
          <dd class="text-gray-800 font-medium">
            {{ stats ? (stats.ram_total / 1073741824).toFixed(2) + ' GB' : '—' }}
          </dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Uptime</dt>
          <dd class="text-gray-800 font-medium">
            {{ stats ? Math.floor(stats.uptime_seconds / 86400) + ' days' : '—' }}
          </dd>
        </div>
      </dl>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">Services</h2>
      <dl class="grid grid-cols-3 gap-3 text-xs">
        <div v-for="(status, svc) in stats?.services ?? {}" :key="svc"
          class="flex items-center justify-between border border-gray-100 rounded-md px-3 py-2">
          <span class="text-gray-600 capitalize">{{ svc }}</span>
          <span class="px-2 py-0.5 rounded text-[10px] font-medium"
            :class="status === 'active' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'">{{ status }}</span>
        </div>
      </dl>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-3">
      <h2 class="font-semibold text-gray-800 text-sm">PHP Versions</h2>
      <p class="text-xs text-gray-500">Manage which PHP versions are available to users in the PHP Versions page.</p>
      <div class="flex flex-wrap gap-2 text-xs">
        <span v-for="v in phpVersions" :key="v.id"
          class="px-3 py-1 rounded-md border"
          :class="v.enabled ? 'border-indigo-200 bg-indigo-50 text-indigo-700' : 'border-gray-200 bg-gray-50 text-gray-400'">
          PHP {{ v.version }}{{ v.enabled ? '' : ' (disabled)' }}
        </span>
      </div>
    </div>

    <div class="bg-white border border-gray-200 rounded-lg p-4 space-y-2">
      <h2 class="font-semibold text-gray-800 text-sm">About</h2>
      <dl class="grid grid-cols-2 gap-x-6 gap-y-1 text-xs">
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Panel version</dt>
          <dd class="text-gray-800 font-medium">v1.0.0</dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">License</dt>
          <dd class="text-gray-800 font-medium">MIT</dd>
        </div>
        <div class="flex justify-between border-b border-gray-50 py-1">
          <dt class="text-gray-500">Docs</dt>
          <dd>
            <a href="https://github.com/teknik-github/zenspanel" target="_blank"
              class="text-indigo-600 hover:text-indigo-800">GitHub</a>
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>
