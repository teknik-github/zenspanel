<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
const UBadge = resolveComponent('UBadge'); const toast = useToast()

const { data: st } = await useFetch('/api/v1/antivirus/status', { lazy: true })
const status = computed(() => st.value as any)

const { data: al, refresh } = await useFetch('/api/v1/antivirus/alerts', { lazy: true })
const alerts = computed(() => { const r = al.value as any; return Array.isArray(r?.data) ? r.data : [] })

const scanning = ref(false); const scanPath = ref(''); const scanResult = ref<any>(null)
async function startScan() { scanning.value = true; try { const r: any = await $fetch('/api/v1/antivirus/scan', { method: 'POST', body: { path: scanPath.value } }); scanResult.value = r; toast.add({ title: 'Scan started', description: `Job: ${r.job_id}`, color: 'success' }) } catch (e:any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { scanning.value = false } }

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'path', header: 'File', cell: ({ row }: any) => h('code', { class: 'text-xs font-mono' }, row.original.path) },
  { accessorKey: 'threat', header: 'Threat', cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: 'error' }, () => row.original.threat) },
  { accessorKey: 'detected_at', header: 'Detected', cell: ({ row }: any) => row.original.detected_at?.slice(0,16) },
]
</script>

<template>
<UDashboardPanel id="antivirus"><template #header><UDashboardNavbar title="Antivirus"><template #leading><UDashboardSidebarCollapse /></template></UDashboardNavbar></template>
<template #body>
<div class="space-y-6 max-w-3xl">
  <UCard><template #header><h3 class="font-semibold">ClamAV Status</h3></template>
    <div class="flex items-center gap-3">
      <UBadge :color="status?.running?'success':'error'" variant="subtle">{{ status?.running ? 'Running' : 'Stopped' }}</UBadge>
      <span class="text-sm text-dimmed" v-if="status?.version">Version: {{ status.version }}</span>
    </div>
  </UCard>

  <UCard><template #header><h3 class="font-semibold">Scan Directory</h3></template>
    <div class="flex items-center gap-3">
      <UInput v-model="scanPath" placeholder="public_html/example.com" :disabled="scanning" class="flex-1" />
      <UButton label="Scan" icon="i-lucide-search" :loading="scanning" @click="startScan" />
    </div>
    <div v-if="scanResult" class="mt-2 text-sm text-dimmed">Job ID: {{ scanResult.job_id }} — polls /antivirus/scan/:job_id for results</div>
  </UCard>

  <UCard><template #header><h3 class="font-semibold">Detected Threats ({{ alerts.length }})</h3></template>
    <UTable :data="alerts" :columns="columns" :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50', td:'border-b border-default' }" />
  </UCard>
</div>
</template></UDashboardPanel>
</template>
