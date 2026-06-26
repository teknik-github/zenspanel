<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/php-versions' })

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

const columnFilters = ref([{ id: 'enabled', value: '' }])

// ── Fetch PHP versions from API ──
const { data, status, refresh } = await useFetch('/api/v1/php-versions', { lazy: true })

const phpVersions = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Row actions ──
function getRowItems(row: Row<any>) {
  const ver = row.original
  const items: any[] = [
    { type: 'label', label: `PHP ${ver.version}` }
  ]
  if (ver.enabled) {
    items.push({ label: 'Disable', icon: 'i-lucide-toggle-left', onSelect: () => handleDisable(ver) })
  } else {
    items.push({ label: 'Enable', icon: 'i-lucide-toggle-right', onSelect: () => handleEnable(ver) })
  }
  items.push(
    { type: 'separator' },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error', onSelect: () => showDeleteModal(ver) }
  )
  return items
}

// ── Enable / Disable ──
async function handleEnable(ver: any) {
  try {
    await $fetch(`/api/v1/php-versions/${ver.id}/enable`, { method: 'PUT' })
    toast.add({ title: `PHP ${ver.version} enabled`, color: 'success' })
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  }
}
async function handleDisable(ver: any) {
  try {
    await $fetch(`/api/v1/php-versions/${ver.id}/disable`, { method: 'PUT' })
    toast.add({ title: `PHP ${ver.version} disabled`, color: 'success' })
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  }
}

// ── Delete ──
const deleteTarget = ref<{ id: number; version: string } | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

function showDeleteModal(ver: any) {
  deleteTarget.value = { id: ver.id, version: ver.version }
  deleteOpen.value = true
}
async function handleDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/v1/php-versions/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', description: `PHP ${deleteTarget.value.version} removed.`, color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Delete failed', color: 'error' })
  } finally { deleteLoading.value = false }
}

// ── Create ──
const createOpen = ref(false)
const createLoading = ref(false)
const createVersion = ref('8.4')

function showCreateModal() {
  createVersion.value = '8.4'
  createOpen.value = true
}
async function handleCreate() {
  createLoading.value = true
  try {
    await $fetch('/api/v1/php-versions', { method: 'POST', body: { version: createVersion.value } })
    toast.add({ title: 'Created', description: `PHP ${createVersion.value} added.`, color: 'success' })
    createOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  } finally { createLoading.value = false }
}

// ── Columns ──
const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'version',
    header: 'Version',
    cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, `PHP ${row.original.version}`)
  },
  { accessorKey: 'fpm_socket', header: 'FPM Socket', cell: ({ row }: any) => h('span', { class: 'text-xs font-mono text-dimmed' }, row.original.fpm_socket || '—') },
  {
    accessorKey: 'enabled',
    header: 'Enabled',
    filterFn: 'equals',
    cell: ({ row }: any) => {
      const color = row.original.enabled ? 'success' : 'neutral'
      const label = row.original.enabled ? 'Enabled' : 'Disabled'
      return h(UBadge, { class: 'capitalize', variant: 'subtle', color }, () => label)
    }
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }: any) => {
      const d = row.original.created_at
      return d ? new Date(d).toLocaleDateString() : '—'
    }
  },
  {
    id: 'actions',
    cell: ({ row }: any) => {
      return h('div', { class: 'text-right' },
        h(UDropdownMenu, { content: { align: 'end' }, items: getRowItems(row) },
          () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' })
        )
      )
    }
  }
]

const enabledFilter = ref('all')
watch(() => enabledFilter.value, (newVal) => {
  if (!table?.value?.tableApi) return
  const col = table.value.tableApi.getColumn('enabled')
  if (!col) return
  if (newVal === 'all') { col.setFilterValue(undefined) }
  else if (newVal === 'true') { col.setFilterValue(true) }
  else { col.setFilterValue(false) }
})

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="php-versions">
    <template #header>
      <UDashboardNavbar title="PHP Versions">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <UButton label="Add Version" icon="i-lucide-plus" @click="showCreateModal" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-1.5">
        <USelect
          v-model="enabledFilter"
          :items="[{ label: 'All', value: 'all' }, { label: 'Enabled', value: 'true' }, { label: 'Disabled', value: 'false' }]"
          placeholder="Status" class="min-w-32"
        />
      </div>

      <UTable
        ref="table"
        v-model:column-filters="columnFilters"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="phpVersions"
        :columns="columns"
        :loading="status === 'pending'"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      />

      <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
        <div class="text-sm text-muted">{{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} version(s)</div>
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>

      <!-- Create Modal -->
      <UModal v-model:open="createOpen" title="Add PHP Version">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Version" required>
              <UInput v-model="createVersion" :disabled="createLoading" placeholder="8.4" />
            </UFormField>
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="createOpen = false" :disabled="createLoading" />
              <UButton label="Add" color="primary" :loading="createLoading" @click="handleCreate" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Delete confirmation -->
      <UModal v-model:open="deleteOpen" :title="`Delete PHP ${deleteTarget?.version || ''}`"
        description="Remove this PHP version from the catalog. This does not uninstall the PHP binary.">
        <template #body>
          <div class="flex justify-end gap-2">
            <UButton label="Cancel" color="neutral" variant="subtle" @click="deleteOpen = false" :disabled="deleteLoading" />
            <UButton label="Delete" color="error" :loading="deleteLoading" @click="handleDelete" />
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
