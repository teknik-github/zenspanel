<script setup lang="ts">
const toast = useToast()
const domains = ref<any[]>([])
const selectedDomain = ref<any>(null)
const logType = ref('nginx')
const lines = ref(100)
const logLines = ref<string[]>([])
const loading = ref(false)

onMounted(async () => {
  try {
    const res: any = await $fetch('/api/v1/domains')
    domains.value = res?.data || []
  } catch {}
})

async function fetchLogs() {
  if (!selectedDomain.value) return
  loading.value = true; logLines.value = []
  try {
    const res: any = await $fetch(`/api/v1/domains/${selectedDomain.value.id}/logs`, { query: { type: logType.value, lines: lines.value } })
    logLines.value = res?.lines || []
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { loading.value = false }
}

function autoRefresh() { fetchLogs() }
useIntervalFn(autoRefresh, 10000)
</script>

<template>
<UDashboardPanel id="logs"><template #header><UDashboardNavbar title="Error Logs"><template #leading><UDashboardSidebarCollapse /></template></UDashboardNavbar></template>
<template #body>
<div class="flex flex-wrap items-center gap-3 mb-4">
  <USelect v-model="selectedDomain" :items="domains" option-attribute="domain" placeholder="Select domain..." class="max-w-xs" @update:model-value="fetchLogs" />
  <USelect v-model="logType" :items="[{label:'Nginx',value:'nginx'},{label:'PHP-FPM',value:'fpm'}]" @update:model-value="fetchLogs" />
  <USelect v-model.number="lines" :items="[50,100,200,500]" @update:model-value="fetchLogs" />
</div>

<UCard>
  <template #header><div class="flex items-center justify-between"><h3 class="font-semibold">{{ selectedDomain?.domain || 'No domain selected' }} — {{ logType === 'nginx' ? 'Nginx' : 'PHP-FPM' }}</h3><UButton icon="i-lucide-refresh-cw" size="xs" color="neutral" variant="ghost" :loading="loading" @click="fetchLogs" /></div></template>
  <div v-if="!selectedDomain" class="text-dimmed text-sm py-8 text-center">Select a domain to view logs</div>
  <div v-else-if="loading && !logLines.length" class="text-dimmed text-sm py-4">Loading...</div>
  <div v-else-if="!logLines.length" class="text-dimmed text-sm py-4">No log entries</div>
  <div v-else class="font-mono text-xs bg-black/40 rounded-lg p-3 max-h-96 overflow-y-auto space-y-0.5">
    <div v-for="(line,i) in logLines" :key="i" class="whitespace-pre-wrap break-all" :class="line.includes('[error]')||line.includes('ERROR')?'text-red-400':line.includes('[warn]')||line.includes('WARNING')?'text-yellow-400':'text-gray-300'">{{ line }}</div>
  </div>
</UCard>
</template></UDashboardPanel>
</template>
