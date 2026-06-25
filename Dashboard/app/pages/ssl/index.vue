<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()
const table = useTemplateRef('table')

const { data, status, refresh } = await useFetch('/api/v1/domains', { lazy: true, query: { limit: 200 } })

const domains = computed(() => { const raw = data.value as any; if (!raw) return []; return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : []) })

function fmtSQL(s: any) { return typeof s === 'object' ? (s.Valid ? s.Time?.slice(0, 10) || '—' : '—') : (s || '—') }
function daysLeft(exp: any) { if (typeof exp === 'object' && exp.Valid) { const d = Math.ceil((new Date(exp.Time).getTime() - Date.now()) / 86400000); return d } return null }

function getRowItems(row: Row<any>) {
  const d = row.original
  const items: any[] = [{ type: 'label', label: d.domain }]
  if (d.ssl_type === 'letsencrypt' || d.ssl_type === 'custom') {
    items.push({ label: 'Remove SSL', icon: 'i-lucide-lock-open', onSelect: async () => { try { await $fetch(`/api/v1/domains/${d.id}/ssl`, { method: 'DELETE' }); toast.add({ title: 'SSL Removed', color: 'success' }); refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } } })
  }
  if (d.ssl_type === 'none' || !d.ssl_type) {
    items.push({ label: 'Issue Let\'s Encrypt', icon: 'i-lucide-shield-check', onSelect: async () => { try { await $fetch(`/api/v1/domains/${d.id}/ssl`, { method: 'POST', body: { type: 'letsencrypt' } }); toast.add({ title: 'SSL Issued', color: 'success' }); refresh() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } } })
  }
  return items
}

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'domain', header: 'Domain', cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.domain) },
  { accessorKey: 'ssl_type', header: 'SSL Type',
    cell: ({ row }: any) => { const t = row.original.ssl_type; return h(UBadge, { variant: 'subtle', color: t === 'letsencrypt' ? 'success' : t === 'custom' ? 'info' : 'neutral', class: 'capitalize' }, () => t || 'none') }
  },
  { accessorKey: 'ssl_expires_at', header: 'Expires',
    cell: ({ row }: any) => {
      const d = daysLeft(row.original.ssl_expires_at)
      if (d === null) return '—'
      const color = d < 0 ? 'error' : d < 30 ? 'warning' : 'success'
      return h(UBadge, { variant: 'subtle', color }, () => `${d}d`)
    }
  },
  { accessorKey: 'status', header: 'Status', cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: row.original.status === 'active' ? 'success' : 'error', class: 'capitalize' }, () => row.original.status) },
  { id: 'actions', cell: ({ row }: any) => h('div', { class: 'text-right' },
      h(UDropdownMenu, { content: { align: 'end' }, items: getRowItems(row) },
        () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' }))) }
]
const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
<UDashboardPanel id="ssl"><template #header><UDashboardNavbar title="SSL Manager"><template #leading><UDashboardSidebarCollapse /></template></UDashboardNavbar></template>
<template #body>
<div class="text-sm text-dimmed mb-4">Manage SSL certificates for all domains. Expiring within 30 days are highlighted.</div>
<UTable ref="table" v-model:pagination="pagination" :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }" class="shrink-0"
  :data="domains" :columns="columns" :loading="status==='pending'"
  :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50 [&>tr]:after:content-none', tbody:'[&>tr]:last:[&>td]:border-b-0', th:'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r', td:'border-b border-default', separator:'h-0' }" />
<UPagination :default-page="1" :items-per-page="20" :total="domains.length" class="mt-4 justify-end" />
</template></UDashboardPanel>
</template>
