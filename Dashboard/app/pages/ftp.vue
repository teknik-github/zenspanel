<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
const UButton = resolveComponent('UButton'); const UBadge = resolveComponent('UBadge'); const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()

const { data, refresh } = await useFetch('/api/v1/ftp', { lazy: true })
const accounts = computed(() => { const r = data.value as any; return Array.isArray(r?.data) ? r.data : (Array.isArray(r) ? r : []) })

// Create
const cOpen = ref(false), cLoading = ref(false)
const cState = reactive({ ftp_username: '', password: '', home_dir: '' })
async function handleCreate() { cLoading.value = true; try { await $fetch('/api/v1/ftp', { method: 'POST', body: cState }); toast.add({ title: 'FTP account created', color: 'success' }); cOpen.value = false; refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { cLoading.value = false } }

// Delete
const dTarget = ref<any>(null), dOpen = ref(false), dLoading = ref(false)
async function handleDel() { if (!dTarget.value) return; dLoading.value = true; try { await $fetch(`/api/v1/ftp/${dTarget.value.id}`, { method: 'DELETE' }); toast.add({ title: 'Deleted', color: 'success' }); dOpen.value = false; refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { dLoading.value = false } }

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'ftp_username', header: 'Username', cell: ({ row }: any) => h('span', { class: 'font-medium font-mono text-highlighted' }, row.original.ftp_username) },
  { accessorKey: 'home_dir', header: 'Home Directory', cell: ({ row }: any) => h('code', { class: 'text-xs' }, row.original.home_dir) },
  { accessorKey: 'enabled', header: 'Status', cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: row.original.enabled ? 'success' : 'error' }, () => row.original.enabled ? 'Active' : 'Disabled') },
  { id: 'actions', cell: ({ row }: any) => h('div', { class: 'text-right' },
    h(UDropdownMenu, { content: { align: 'end' }, items: [{ label: 'Delete', icon: 'i-lucide-trash', color: 'error', onSelect: () => { dTarget.value = row.original; dOpen.value = true } }] },
      () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' }))) }
]
</script>

<template>
<UDashboardPanel id="ftp"><template #header><UDashboardNavbar title="FTP Accounts"><template #leading><UDashboardSidebarCollapse /></template><template #right><UButton label="Add FTP Account" icon="i-lucide-plus" @click="cOpen=true;cState.ftp_username='';cState.password='';cState.home_dir=''" /></template></UDashboardNavbar></template>
<template #body>
<UTable :data="accounts" :columns="columns" :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50', td:'border-b border-default' }" />
<UModal v-model:open="cOpen" title="New FTP Account"><template #body>
<div class="space-y-4">
  <UFormField label="Username" required><UInput v-model="cState.ftp_username" :disabled="cLoading" placeholder="alice_ftp" /></UFormField>
  <UFormField label="Password" required><UInput v-model="cState.password" type="password" :disabled="cLoading" /></UFormField>
  <UFormField label="Home Directory"><UInput v-model="cState.home_dir" :disabled="cLoading" placeholder="Defaults to user home" /></UFormField>
  <div class="flex justify-end gap-2"><UButton label="Cancel" color="neutral" variant="subtle" @click="cOpen=false" :disabled="cLoading" /><UButton label="Create" :loading="cLoading" @click="handleCreate" /></div>
</div></template></UModal>
<UModal v-model:open="dOpen" title="Delete FTP Account"><template #body><div class="flex justify-end gap-2"><UButton label="Cancel" color="neutral" variant="subtle" @click="dOpen=false" :disabled="dLoading" /><UButton label="Delete" color="error" :loading="dLoading" @click="handleDel" /></div></template></UModal>
</template></UDashboardPanel>
</template>
