<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
const UButton = resolveComponent('UButton'); const UBadge = resolveComponent('UBadge'); const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()

// We need the user's domains to manage redirects
const { data: domains } = await useFetch('/api/v1/domains')
const domainList = computed(() => { const r = domains.value as any; return Array.isArray(r?.data) ? r.data : [] })

const selectedDomain = ref<any>(null)
const { data, refresh } = await useFetch('/api/v1/domains/0/redirects', { lazy: true, immediate: false })
const redirects = computed(() => { const r = data.value as any; return Array.isArray(r?.data) ? r.data : [] })

watch(selectedDomain, async (d) => {
  if (!d) return
  const res = await $fetch(`/api/v1/domains/${d.id}/redirects`)
  data.value = res
})

// Create
const cOpen = ref(false), cLoading = ref(false)
const cState = reactive({ source_path: '', dest_url: '', type: '301', enabled: true })
async function handleCreate() { cLoading.value = true; try { await $fetch(`/api/v1/domains/${selectedDomain.value.id}/redirects`, { method: 'POST', body: cState }); toast.add({ title: 'Redirect created', color: 'success' }); cOpen.value = false; refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { cLoading.value = false } }

// Delete
async function handleDel(r: any) { if (!confirm('Delete redirect?')) return; try { await $fetch(`/api/v1/domains/${selectedDomain.value.id}/redirects/${r.id}`, { method: 'DELETE' }); toast.add({ title: 'Deleted', color: 'success' }); refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } }

// Toggle
async function toggle(r: any) { try { await $fetch(`/api/v1/domains/${selectedDomain.value.id}/redirects/${r.id}`, { method: 'PUT', body: { enabled: !r.enabled } }); toast.add({ title: r.enabled ? 'Disabled' : 'Enabled', color: 'success' }); refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } }

const columns: TableColumn<any>[] = [
  { accessorKey: 'source_path', header: 'Source Path', cell: ({ row }: any) => h('code', { class: 'font-mono text-sm' }, row.original.source_path) },
  { accessorKey: 'dest_url', header: 'Destination', cell: ({ row }: any) => h('span', { class: 'text-sm break-all' }, row.original.dest_url) },
  { accessorKey: 'type', header: 'Type', cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: row.original.type === '301' ? 'info' : 'warning' }, () => row.original.type) },
  { accessorKey: 'enabled', header: 'Status', cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: row.original.enabled ? 'success' : 'neutral' }, () => row.original.enabled ? 'On' : 'Off') },
  { id: 'actions', cell: ({ row }: any) => h('div', { class: 'text-right flex gap-1 justify-end' }, [
    h(UButton, { icon: row.original.enabled ? 'i-lucide-toggle-left' : 'i-lucide-toggle-right', size: 'xs', color: 'neutral', variant: 'ghost', onClick: () => toggle(row.original) }),
    h(UButton, { icon: 'i-lucide-trash', size: 'xs', color: 'neutral', variant: 'ghost', onClick: () => handleDel(row.original) })]) }
]
</script>

<template>
<UDashboardPanel id="redirects"><template #header><UDashboardNavbar title="Redirect Manager"><template #leading><UDashboardSidebarCollapse /></template><template #right><UButton v-if="selectedDomain" label="Add Redirect" icon="i-lucide-plus" @click="cOpen=true;cState.source_path='';cState.dest_url=''" /></template></UDashboardNavbar></template>
<template #body>
<div class="mb-4">
  <USelect v-model="selectedDomain" :items="domainList" option-attribute="domain" placeholder="Select a domain..." class="max-w-xs" />
</div>
<div v-if="selectedDomain">
  <UTable :data="redirects" :columns="columns" :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50', td:'border-b border-default' }" />
</div>
<div v-else class="text-dimmed text-sm py-8 text-center">Select a domain to manage redirects</div>

<UModal v-model:open="cOpen" title="Add Redirect"><template #body>
<div class="space-y-4">
  <UFormField label="Source Path" required><UInput v-model="cState.source_path" :disabled="cLoading" placeholder="/old-page" /></UFormField>
  <UFormField label="Destination URL" required><UInput v-model="cState.dest_url" :disabled="cLoading" placeholder="https://example.com/new-page" /></UFormField>
  <UFormField label="Type"><USelect v-model="cState.type" :items="['301','302']" :disabled="cLoading" /></UFormField>
  <div class="flex justify-end gap-2"><UButton label="Cancel" color="neutral" variant="subtle" @click="cOpen=false" :disabled="cLoading" /><UButton label="Create" :loading="cLoading" @click="handleCreate" /></div>
</div></template></UModal>
</template></UDashboardPanel>
</template>
