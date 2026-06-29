<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { VisXYContainer, VisArea, VisLine } from '@unovis/vue'

definePageMeta({ alias: '/admin/dashboard' })

const auth = useAuth()
const isAdmin = computed(() => auth.user.value?.role === 'admin')
const userId = computed(() => auth.user.value?.id)

// ── Admin data ──
const { data: stats, refresh: rfStats } = await useFetch('/api/v1/system/stats', { lazy: true })
const { data: metrics, refresh: rfMetrics } = await useFetch('/api/v1/users/metrics', { lazy: true })

const system = computed(() => stats.value as any)
const users = computed(() => {
  const raw = metrics.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Sparkline history (last 30 readings @ 5s = ~2.5 min window) ──
interface HistoryPoint { t: number; cpu: number; ramPct: number; diskPct: number; load1: number; load5: number; load15: number }
const history = ref<HistoryPoint[]>([])

watch(() => stats.value, (s: any) => {
  if (!s) return
  const ramPct  = s.ram_total  ? (s.ram_used  / s.ram_total)  * 100 : 0
  const diskPct = s.disk_total ? (s.disk_used / s.disk_total) * 100 : 0
  history.value = [
    ...history.value.slice(-29),
    { t: Date.now(), cpu: s.cpu_pct ?? 0, ramPct, diskPct, load1: s.load_1 ?? 0, load5: s.load_5 ?? 0, load15: s.load_15 ?? 0 }
  ]
})

// Stable accessor references (avoids Unovis re-render thrash)
const xAcc      = (d: HistoryPoint) => d.t
const cpuAcc    = (d: HistoryPoint) => d.cpu
const ramAcc    = (d: HistoryPoint) => d.ramPct
const diskAcc   = (d: HistoryPoint) => d.diskPct
const load1Acc  = (d: HistoryPoint) => d.load1
const load5Acc  = (d: HistoryPoint) => d.load5
const load15Acc = (d: HistoryPoint) => d.load15

// ── User data ──
const usageUrl = computed(() => `/api/v1/users/${userId.value || 0}/usage`)
const { data: usageData, refresh: rfUsage } = await useFetch(usageUrl, { lazy: true, immediate: false, watch: false })
const usage = computed(() => {
  const raw = usageData.value as any
  return raw?.usage || null
})

function refreshUsage() {
  if (userId.value) rfUsage()
}

// Poll every 5 seconds
useIntervalFn(() => {
  if (isAdmin.value) { rfStats(); rfMetrics(); return }
  refreshUsage()
}, 5000)

watch(userId, () => {
  if (!isAdmin.value) refreshUsage()
}, { immediate: true })

// ── Helpers ──
function fmtBytes(b: number) {
  if (!b) return '0'
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  return (b / 1048576).toFixed(0) + ' MB'
}
function pct(val: number) { return (val || 0).toFixed(1) + '%' }
function sysPct(used: number, total: number) { return total ? pct((used / total) * 100) : '0.0%' }

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

        <!-- Metric cards with sparkline charts -->
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">

          <!-- CPU -->
          <UCard class="overflow-hidden">
            <p class="text-xs text-dimmed uppercase tracking-wider font-semibold">CPU Usage</p>
            <p class="text-2xl font-bold mt-1">{{ pct(system?.cpu_pct) }}</p>
            <ClientOnly>
              <div class="mt-3 -mx-4 -mb-4">
                <VisXYContainer :data="history" :height="64" :y-domain="[0, 100]" :padding="{ top: 4, bottom: 0, left: 0, right: 0 }">
                  <VisArea :x="xAcc" :y="cpuAcc" color="#60a5fa" :opacity="0.2" />
                  <VisLine :x="xAcc" :y="cpuAcc" color="#60a5fa" :line-width="2" />
                </VisXYContainer>
              </div>
            </ClientOnly>
          </UCard>

          <!-- RAM -->
          <UCard class="overflow-hidden">
            <p class="text-xs text-dimmed uppercase tracking-wider font-semibold">RAM</p>
            <p class="text-2xl font-bold mt-1">{{ sysPct(system?.ram_used, system?.ram_total) }}</p>
            <p class="text-xs text-dimmed">{{ fmtBytes(system?.ram_used) }} / {{ fmtBytes(system?.ram_total) }}</p>
            <ClientOnly>
              <div class="mt-3 -mx-4 -mb-4">
                <VisXYContainer :data="history" :height="64" :y-domain="[0, 100]" :padding="{ top: 4, bottom: 0, left: 0, right: 0 }">
                  <VisArea :x="xAcc" :y="ramAcc" color="#a78bfa" :opacity="0.2" />
                  <VisLine :x="xAcc" :y="ramAcc" color="#a78bfa" :line-width="2" />
                </VisXYContainer>
              </div>
            </ClientOnly>
          </UCard>

          <!-- Disk -->
          <UCard class="overflow-hidden">
            <p class="text-xs text-dimmed uppercase tracking-wider font-semibold">Disk</p>
            <p class="text-2xl font-bold mt-1">{{ sysPct(system?.disk_used, system?.disk_total) }}</p>
            <p class="text-xs text-dimmed">{{ fmtBytes(system?.disk_used) }} / {{ fmtBytes(system?.disk_total) }}</p>
            <ClientOnly>
              <div class="mt-3 -mx-4 -mb-4">
                <VisXYContainer :data="history" :height="64" :y-domain="[0, 100]" :padding="{ top: 4, bottom: 0, left: 0, right: 0 }">
                  <VisArea :x="xAcc" :y="diskAcc" color="#fb923c" :opacity="0.2" />
                  <VisLine :x="xAcc" :y="diskAcc" color="#fb923c" :line-width="2" />
                </VisXYContainer>
              </div>
            </ClientOnly>
          </UCard>

          <!-- Load Average -->
          <UCard class="overflow-hidden">
            <p class="text-xs text-dimmed uppercase tracking-wider font-semibold">Load Average</p>
            <p class="text-2xl font-bold mt-1">{{ system?.load_1?.toFixed(2) ?? '—' }}</p>
            <p class="text-xs text-dimmed">5m {{ system?.load_5?.toFixed(2) ?? '—' }} · 15m {{ system?.load_15?.toFixed(2) ?? '—' }}</p>
            <ClientOnly>
              <div class="mt-3 -mx-4 -mb-4">
                <VisXYContainer :data="history" :height="64" :padding="{ top: 4, bottom: 0, left: 0, right: 0 }">
                  <VisArea :x="xAcc" :y="load15Acc" color="#34d399" :opacity="0.08" />
                  <VisLine :x="xAcc" :y="load15Acc" color="#34d399" :line-width="1" />
                  <VisArea :x="xAcc" :y="load5Acc" color="#34d399" :opacity="0.12" />
                  <VisLine :x="xAcc" :y="load5Acc" color="#34d399" :line-width="1.5" />
                  <VisArea :x="xAcc" :y="load1Acc" color="#4ade80" :opacity="0.2" />
                  <VisLine :x="xAcc" :y="load1Acc" color="#4ade80" :line-width="2" />
                </VisXYContainer>
              </div>
            </ClientOnly>
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
