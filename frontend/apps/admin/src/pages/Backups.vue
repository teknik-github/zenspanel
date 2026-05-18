<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { backupsApi } from '@/api/backups'

const backups = ref<any[]>([])
const error = ref('')

async function load() {
  try {
    const res = await backupsApi.list()
    backups.value = res.data.data || []
    error.value = ''
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to load'
  }
}

function fmtSize(n?: number) {
  if (!n) return '—'
  if (n < 1048576) return (n / 1024).toFixed(1) + ' KB'
  if (n < 1073741824) return (n / 1048576).toFixed(1) + ' MB'
  return (n / 1073741824).toFixed(2) + ' GB'
}

function fmtDate(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleString()
}

function statusClass(s: string) {
  if (s === 'done') return 'bg-green-100 text-green-700'
  if (s === 'failed') return 'bg-red-100 text-red-600'
  if (s === 'running' || s === 'pending') return 'bg-amber-100 text-amber-700'
  return 'bg-gray-100 text-gray-500'
}

onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-lg font-semibold text-gray-800">Backups</h1>
      <span class="text-xs text-gray-400">{{ backups.length }} backup{{ backups.length !== 1 ? 's' : '' }}</span>
    </div>

    <p v-if="error" class="text-xs text-red-600 bg-red-50 border border-red-100 rounded px-2 py-1.5">{{ error }}</p>

    <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
      <table class="w-full text-xs">
        <thead class="bg-gray-50 border-b border-gray-200">
          <tr class="text-gray-500">
            <th class="text-left px-4 py-3 font-medium">User</th>
            <th class="text-left px-4 py-3 font-medium">Type</th>
            <th class="text-left px-4 py-3 font-medium">Status</th>
            <th class="text-left px-4 py-3 font-medium">Size</th>
            <th class="text-left px-4 py-3 font-medium">Created</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in backups" :key="b.id" class="border-b border-gray-50 hover:bg-gray-50">
            <td class="px-4 py-3 text-gray-500">{{ b.user_id }}</td>
            <td class="px-4 py-3 font-medium text-gray-700">{{ b.type }}</td>
            <td class="px-4 py-3">
              <span class="px-2 py-0.5 rounded text-[10px] font-medium" :class="statusClass(b.status)">{{ b.status }}</span>
            </td>
            <td class="px-4 py-3 text-gray-500">{{ fmtSize(b.size_bytes?.Int64 || b.size_bytes) }}</td>
            <td class="px-4 py-3 text-gray-500">{{ fmtDate(b.created_at) }}</td>
          </tr>
          <tr v-if="!backups.length">
            <td colspan="5" class="px-4 py-8 text-center text-gray-400">No backups yet</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
