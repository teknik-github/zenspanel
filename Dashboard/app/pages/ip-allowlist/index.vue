<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { Row } from '@tanstack/table-core'
import { getPaginationRowModel } from '@tanstack/table-core'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

// ── Fetch allowlist ──
const { data, status, refresh } = await useFetch('/api/v1/admin/ip-allowlist', { lazy: true })
const entries = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Row actions ──
function getRowItems(row: Row<any>) {
  const item = row.original
  return [
    { type: 'label', label: `IP: ${item.ip}` },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error',
      onSelect: () => showDeleteModal(item)
    }
  ]
}

// ── Delete ──
const deleteTarget = ref<{ id: number; ip: string } | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

function showDeleteModal(item: any) {
  deleteTarget.value = { id: item.id, ip: item.ip }
  deleteOpen.value = true
}
async function handleDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/v1/admin/ip-allowlist/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Removed', description: `${deleteTarget.value.ip} removed from allowlist.`, color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Delete failed', color: 'error' })
  } finally { deleteLoading.value = false }
}

// ── Create ──
const createOpen = ref(false)
const createLoading = ref(false)
const createState = reactive({ ip: '', note: '' })

async function handleCreate() {
  createLoading.value = true
  try {
    await $fetch('/api/v1/admin/ip-allowlist', { method: 'POST', body: createState })
    toast.add({ title: 'Added', description: `${createState.ip} added to allowlist.`, color: 'success' })
    createOpen.value = false
    createState.ip = ''
    createState.note = ''
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Create failed', color: 'error' })
  } finally { createLoading.value = false }
}

// ── Columns ──
const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'ip',
    header: 'IP / CIDR',
    cell: ({ row }: any) => h('code', { class: 'text-sm font-mono text-highlighted' }, row.original.ip)
  },
  {
    accessorKey: 'note',
    header: 'Note',
    cell: ({ row }: any) => row.original.note || '—'
  },
  {
    accessorKey: 'created_at',
    header: 'Created At',
    cell: ({ row }: any) => {
      const d = row.original.created_at
      return d ? new Date(d).toLocaleString() : '—'
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

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="ip-allowlist">
    <template #header>
      <UDashboardNavbar title="IP Allowlist">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <UButton label="Add IP" icon="i-lucide-plus" @click="createOpen = true" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4">
        <div class="text-sm text-muted">
          Restrict <code class="text-xs bg-muted/50 px-1 py-0.5 rounded">/admin/</code> access to specific IPs or CIDR ranges. Empty list means all IPs are allowed.
        </div>

        <UTable
          ref="table"
          v-model:pagination="pagination"
          :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
          class="shrink-0"
          :data="entries"
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
          <div class="text-sm text-muted">{{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} entry(s)</div>
          <UPagination
            :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
            :items-per-page="table?.tableApi?.getState().pagination.pageSize"
            :total="table?.tableApi?.getFilteredRowModel().rows.length"
            @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
          />
        </div>
      </div>

      <!-- Create Modal -->
      <UModal v-model:open="createOpen" title="Add IP to Allowlist">
        <template #body>
          <div class="space-y-4">
            <UFormField label="IP Address or CIDR" required>
              <UInput v-model="createState.ip" :disabled="createLoading" placeholder="203.0.113.0/24" />
            </UFormField>
            <UFormField label="Note (optional)">
              <UInput v-model="createState.note" :disabled="createLoading" placeholder="office" />
            </UFormField>
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="createOpen = false" :disabled="createLoading" />
              <UButton label="Add" color="primary" :loading="createLoading" @click="handleCreate" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Delete confirmation -->
      <UModal v-model:open="deleteOpen" :title="`Remove ${deleteTarget?.ip || ''}`"
        description="This IP range will no longer be allowed. Access will fall back to the next matching rule.">
        <template #body>
          <div class="flex justify-end gap-2">
            <UButton label="Cancel" color="neutral" variant="subtle" @click="deleteOpen = false" :disabled="deleteLoading" />
            <UButton label="Remove" color="error" :loading="deleteLoading" @click="handleDelete" />
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
