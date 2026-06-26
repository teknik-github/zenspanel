<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/php-extensions' })

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

const columnFilters = ref([{ id: 'php_version', value: '' }])

// ── Fetch PHP extensions from API ──
const { data, status, refresh } = await useFetch('/api/v1/admin/php-extensions', { lazy: true })

const phpExtensions = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// Derived unique PHP versions for filter dropdown
const availablePhpVersions = computed(() => {
  const versions = new Set<string>()
  phpExtensions.value.forEach((ext: any) => {
    if (ext.php_version) versions.add(ext.php_version)
  })
  return Array.from(versions).sort()
})

// ── Row actions ──
function getRowItems(row: Row<any>) {
  const ext = row.original
  return [
    { type: 'label', label: ext.name },
    {
      label: ext.enabled ? 'Disable' : 'Enable',
      icon: ext.enabled ? 'i-lucide-toggle-left' : 'i-lucide-toggle-right',
      onSelect: () => handleToggle(ext)
    },
    { type: 'separator' },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error', onSelect: () => showDeleteModal(ext) }
  ]
}

// ── Toggle ──
async function handleToggle(ext: any) {
  try {
    await $fetch(`/api/v1/admin/php-extensions/${ext.id}`, { method: 'PUT', body: { enabled: !ext.enabled } })
    toast.add({ title: ext.enabled ? 'Disabled' : 'Enabled', description: ext.name, color: 'success' })
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  }
}

// ── Delete ──
const deleteTarget = ref<{ id: number; name: string } | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

function showDeleteModal(ext: any) {
  deleteTarget.value = { id: ext.id, name: ext.name }
  deleteOpen.value = true
}
async function handleDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/v1/admin/php-extensions/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', description: `${deleteTarget.value.name} removed.`, color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Delete failed', color: 'error' })
  } finally { deleteLoading.value = false }
}

// ── Create ──
const createOpen = ref(false)
const createLoading = ref(false)
const createState = reactive({ name: '', php_version: '8.3', enabled: true })

function showCreateModal() {
  Object.assign(createState, { name: '', php_version: '8.3', enabled: true })
  createOpen.value = true
}
async function handleCreate() {
  createLoading.value = true
  try {
    await $fetch('/api/v1/admin/php-extensions', { method: 'POST', body: createState })
    toast.add({ title: 'Created', description: `${createState.name} added.`, color: 'success' })
    createOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  } finally { createLoading.value = false }
}

// ── Seed defaults ──
const seedLoading = ref(false)
async function handleSeed() {
  seedLoading.value = true
  try {
    await $fetch('/api/v1/admin/php-extensions/seed', { method: 'POST' })
    toast.add({ title: 'Seeded', description: 'Default extensions populated.', color: 'success' })
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Seed failed', color: 'error' })
  } finally { seedLoading.value = false }
}

// ── Columns ──
const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.name)
  },
  {
    accessorKey: 'php_version',
    header: 'PHP Version',
    filterFn: 'equals',
    cell: ({ row }: any) => h('span', { class: 'text-xs font-mono text-dimmed' }, row.original.php_version || '—')
  },
  {
    accessorKey: 'enabled',
    header: 'Status',
    cell: ({ row }: any) => {
      const color = row.original.enabled ? 'success' : 'neutral'
      const label = row.original.enabled ? 'Enabled' : 'Disabled'
      return h(UBadge, { class: 'capitalize', variant: 'subtle', color }, () => label)
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

const versionFilter = ref('all')
watch(() => versionFilter.value, (newVal) => {
  if (!table?.value?.tableApi) return
  const col = table.value.tableApi.getColumn('php_version')
  if (!col) return
  col.setFilterValue(newVal === 'all' ? undefined : newVal)
})

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="php-extensions">
    <template #header>
      <UDashboardNavbar title="PHP Extensions">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <div class="flex gap-2">
            <UButton label="Seed Defaults" icon="i-lucide-sparkles" color="neutral" variant="subtle" :loading="seedLoading" @click="handleSeed" />
            <UButton label="Add Extension" icon="i-lucide-plus" @click="showCreateModal" />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-1.5">
        <USelect
          v-model="versionFilter"
          :items="[{ label: 'All Versions', value: 'all' }, ...availablePhpVersions.map(v => ({ label: `PHP ${v}`, value: v }))]"
          placeholder="Version" class="min-w-32"
        />
      </div>

      <UTable
        ref="table"
        v-model:column-filters="columnFilters"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="phpExtensions"
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
        <div class="text-sm text-muted">{{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} extension(s)</div>
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>

      <!-- Create Modal -->
      <UModal v-model:open="createOpen" title="Add PHP Extension">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Extension Name" required>
              <UInput v-model="createState.name" :disabled="createLoading" placeholder="redis" />
            </UFormField>
            <UFormField label="PHP Version" required>
              <UInput v-model="createState.php_version" :disabled="createLoading" placeholder="8.3" />
            </UFormField>
            <label class="flex items-center gap-2 text-sm">
              <USwitch v-model="createState.enabled" :disabled="createLoading" />
              Enabled
            </label>
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="createOpen = false" :disabled="createLoading" />
              <UButton label="Add" color="primary" :loading="createLoading" @click="handleCreate" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Delete confirmation -->
      <UModal v-model:open="deleteOpen" :title="`Delete ${deleteTarget?.name || ''}`">
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
