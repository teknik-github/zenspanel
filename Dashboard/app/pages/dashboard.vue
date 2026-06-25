<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'

const auth = useAuth()
const isAdmin = computed(() => auth.user.value?.role === 'admin')
const userId = computed(() => auth.user.value?.id)

// ── Admin data ──
const { data: stats, refresh: rfStats } = await useFetch('/api/v1/system/stats', { lazy: true })
const { data: metrics, refresh: rfMetrics } = await useFetch('/api/v1/users/metrics', { lazy: true })

// Poll every 5 seconds
useIntervalFn(() => {
  if (isAdmin.value) { rfStats(); rfMetrics() } else { rfUsage() }
}, 5000)

const system = computed(() => stats.value as any)
const users = computed(() => {
  const raw = metrics.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── User data ──
const { data: usageData, refresh: rfUsage } = await useFetch(`/api/v1/users/${userId.value}/usage`, { lazy: true, immediate: false })
const usage = computed(() => {
  const raw = usageData.value as any
  return raw?.usage || null
})

// ── Helpers ──
function fmtBytes(b: number) {
  if (!b) return '0'
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  return (b / 1048576).toFixed(0) + ' MB'
}
function pct(val: number) { return (val || 0).toFixed(1) + '%' }

const adminColumns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'username', header: 'User', cell: ({ row }: any) => h('span', { class: 'font-medium' }, row.original.username) },
  { accessorKey: 'cpu_pct', header: 'CPU', cell: ({ row }: any) => pct(row.original.cpu_pct) },
  { accessorKey: 'ram_used', header: 'RAM', cell: ({ row }: any) => fmtBytes(row.original.ram_used) + ' / ' + fmtBytes(row.original.ram_max) },
  { accessorKey: 'disk_used', header: 'Disk', cell: ({ row }: any) => fmtBytes(row.original.disk_used) + ' / ' + fmtBytes(row.original.disk_max) },
]
</script>

<template>
  <!-- ═══════════════ ADMIN: Resource Monitor ═══════════════ -->
  <UDashboardPanel v-if="isAdmin" id="dashboard">
    <template #header>
      <UDashboardNavbar title="Dashboard">
        <template #leading><UDashboardSidebarCollapse /></template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="space-y-6">
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">CPU Usage</p>
            <p class="text-2xl font-bold mt-1">{{ pct(system?.cpu_pct) }}</p>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">RAM</p>
            <p class="text-2xl font-bold mt-1">{{ fmtBytes(system?.ram_used) }} / {{ fmtBytes(system?.ram_total) }}</p>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Disk</p>
            <p class="text-2xl font-bold mt-1">{{ fmtBytes(system?.disk_used) }} / {{ fmtBytes(system?.disk_total) }}</p>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Load</p>
            <p class="text-2xl font-bold mt-1">{{ system?.load_1?.toFixed(1) }} / {{ system?.load_5?.toFixed(1) }} / {{ system?.load_15?.toFixed(1) }}</p>
          </UCard>
        </div>

        <UCard>
          <template #header><h3 class="font-semibold">Services</h3></template>
          <div class="flex flex-wrap gap-2">
            <UBadge v-for="(status, name) in system?.services || {}" :key="name" variant="subtle"
              :color="status === 'running' ? 'success' : 'error'">{{ name }}: {{ status }}</UBadge>
          </div>
        </UCard>

        <UCard>
          <template #header><h3 class="font-semibold">User Resource Usage</h3></template>
          <UTable :data="users" :columns="adminColumns"
            :ui="{ base: 'table-fixed border-separate border-spacing-0', thead: '[&>tr]:bg-elevated/50', td: 'border-b border-default' }" />
        </UCard>
      </div>
    </template>
  </UDashboardPanel>

  <!-- ═══════════════ USER: Personal Usage ═══════════════ -->
  <UDashboardPanel v-else id="dashboard">
    <template #header>
      <UDashboardNavbar title="Dashboard">
        <template #leading><UDashboardSidebarCollapse /></template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="space-y-6">
        <!-- Resource usage cards -->
        <div class="grid grid-cols-2 lg:grid-cols-3 gap-4">
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">CPU</p>
            <p class="text-2xl font-bold mt-1">{{ pct(usage?.cpu?.used) }}</p>
            <div class="mt-2 w-full bg-elevated rounded-full h-1.5">
              <div class="bg-primary h-1.5 rounded-full" :style="{ width: Math.min(usage?.cpu?.used || 0, 100) + '%' }" />
            </div>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">RAM</p>
            <p class="text-2xl font-bold mt-1">{{ fmtBytes(usage?.ram?.used) }}</p>
            <p class="text-xs text-dimmed mt-0.5">of {{ fmtBytes(usage?.ram?.max) }}</p>
            <div class="mt-2 w-full bg-elevated rounded-full h-1.5">
              <div class="bg-primary h-1.5 rounded-full" :style="{ width: usage?.ram?.max ? Math.min((usage.ram.used / usage.ram.max) * 100, 100) + '%' : '0%' }" />
            </div>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Disk</p>
            <p class="text-2xl font-bold mt-1">{{ fmtBytes(usage?.disk?.used) }}</p>
            <p class="text-xs text-dimmed mt-0.5">of {{ fmtBytes(usage?.disk?.max) }}</p>
            <div class="mt-2 w-full bg-elevated rounded-full h-1.5">
              <div class="bg-primary h-1.5 rounded-full" :style="{ width: usage?.disk?.max ? Math.min((usage.disk.used / usage.disk.max) * 100, 100) + '%' : '0%' }" />
            </div>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Domains</p>
            <p class="text-2xl font-bold mt-1">{{ usage?.domains?.used ?? '—' }}</p>
            <p class="text-xs text-dimmed mt-0.5">of {{ usage?.domains?.max ?? '—' }} max</p>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Databases</p>
            <p class="text-2xl font-bold mt-1">{{ usage?.databases?.used ?? '—' }}</p>
            <p class="text-xs text-dimmed mt-0.5">of {{ usage?.databases?.max ?? '—' }} max</p>
          </UCard>
          <UCard>
            <p class="text-xs text-dimmed uppercase tracking-wider">Disk Breakdown</p>
            <div class="mt-2 space-y-1 text-sm">
              <div class="flex justify-between"><span class="text-dimmed">Files</span><span>{{ fmtBytes(usage?.disk?.files || 0) }}</span></div>
              <div class="flex justify-between"><span class="text-dimmed">Databases</span><span>{{ fmtBytes(usage?.disk?.db || 0) }}</span></div>
            </div>
          </UCard>
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
