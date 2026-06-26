<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { FormSubmitEvent } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'
import * as z from 'zod'

definePageMeta({ alias: '/admin/packages' })

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

// ── Fetch packages from API ──
const { data, status, refresh } = await useFetch('/api/v1/packages', { lazy: true })

const packages = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Helpers ──
function formatBytes(bytes: number) {
  if (!bytes || bytes === 0) return '0'
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  return (bytes / 1048576).toFixed(0) + ' MB'
}

function formatMB(bytes: number) {
  if (!bytes || bytes === 0) return '0'
  return (bytes / 1048576).toFixed(0) + ' MB'
}

// ── Row actions ──
function getRowItems(row: Row<any>) {
  const pkg = row.original
  return [
    { type: 'label', label: `Package: ${pkg.name}` },
    { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => showEditModal(pkg) },
    { type: 'separator' },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error',
      onSelect: () => showDeleteModal(pkg)
    }
  ]
}

// ── Delete ──
const deleteTarget = ref<{ id: number; name: string } | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

function showDeleteModal(pkg: any) {
  deleteTarget.value = { id: pkg.id, name: pkg.name }
  deleteOpen.value = true
}
async function handleDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/v1/packages/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', description: `${deleteTarget.value.name} removed.`, color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Delete failed', color: 'error' })
  } finally { deleteLoading.value = false }
}

// ── Create / Edit ──
const editOpen = ref(false)
const editLoading = ref(false)
const editMode = ref<'create' | 'edit'>('create')
const editTarget = ref<any>(null)

const editState = reactive({
  name: '',
  cpu_quota: 100,
  disk_quota_mb: 10240,
  memory_limit_mb: 1024,
  max_domains: 10,
  max_databases: 5,
  max_cron_jobs: 10,
  max_procs: 50,
  max_ftp_accounts: 5,
  io_read_mbps: 0,
  io_write_mbps: 0,
  antivirus_enabled: true,
  terminal_enabled: true,
  backup_enabled: true,
  php_versions_allowed: '8.3,8.2'
})

function showEditModal(pkg?: any) {
  editMode.value = pkg ? 'edit' : 'create'
  editTarget.value = pkg
  if (pkg) {
    Object.assign(editState, {
      name: pkg.name,
      cpu_quota: pkg.cpu_quota,
      disk_quota_mb: pkg.disk_quota_mb || Math.round(pkg.disk_quota / 1048576),
      memory_limit_mb: pkg.memory_limit_mb || Math.round(pkg.memory_limit / 1048576),
      max_domains: pkg.max_domains,
      max_databases: pkg.max_databases,
      max_cron_jobs: pkg.max_cron_jobs,
      max_procs: pkg.max_procs,
      max_ftp_accounts: pkg.max_ftp_accounts,
      io_read_mbps: pkg.io_read_mbps || 0,
      io_write_mbps: pkg.io_write_mbps || 0,
      antivirus_enabled: pkg.antivirus_enabled,
      terminal_enabled: pkg.terminal_enabled,
      backup_enabled: pkg.backup_enabled,
      php_versions_allowed: pkg.php_versions_allowed || '8.3'
    })
  } else {
    Object.assign(editState, {
      name: '', cpu_quota: 100, disk_quota_mb: 10240, memory_limit_mb: 1024,
      max_domains: 10, max_databases: 5, max_cron_jobs: 10, max_procs: 50,
      max_ftp_accounts: 5, io_read_mbps: 0, io_write_mbps: 0,
      antivirus_enabled: true, terminal_enabled: true, backup_enabled: true,
      php_versions_allowed: '8.3,8.2'
    })
  }
  editOpen.value = true
}

async function onPackageSubmit() {
  editLoading.value = true
  try {
    const payload = {
      ...editState,
      php_versions_allowed: editState.php_versions_allowed
        .split(',')
        .map(version => version.trim())
        .filter(Boolean)
    }

    if (editMode.value === 'create') {
      await $fetch('/api/v1/packages', { method: 'POST', body: payload })
      toast.add({ title: 'Created', description: `${editState.name} added.`, color: 'success' })
    } else {
      await $fetch(`/api/v1/packages/${editTarget.value.id}`, { method: 'PUT', body: payload })
      toast.add({ title: 'Updated', description: `${editState.name} saved.`, color: 'success' })
    }
    editOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  } finally { editLoading.value = false }
}

// ── Columns ──
const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'name', header: 'Name', cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.name) },
  { accessorKey: 'cpu_quota', header: 'CPU %', cell: ({ row }: any) => row.original.cpu_quota },
  { accessorKey: 'memory_limit', header: 'RAM', cell: ({ row }: any) => formatBytes(row.original.memory_limit) },
  { accessorKey: 'disk_quota', header: 'Disk', cell: ({ row }: any) => formatBytes(row.original.disk_quota) },
  { accessorKey: 'max_domains', header: 'Domains' },
  { accessorKey: 'max_databases', header: 'Databases' },
  { accessorKey: 'max_ftp_accounts', header: 'FTP' },
  { accessorKey: 'php_versions_allowed', header: 'PHP Versions' },
  {
    accessorKey: 'backup_enabled',
    header: 'Backups',
    cell: ({ row }: any) =>
      h(UBadge, { variant: 'subtle', color: row.original.backup_enabled ? 'success' : 'neutral' },
        () => row.original.backup_enabled ? 'Yes' : 'No')
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
  <UDashboardPanel id="packages">
    <template #header>
      <UDashboardNavbar title="Packages">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <UButton label="New Package" icon="i-lucide-plus" @click="showEditModal()" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable
        ref="table"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="packages"
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
        <div class="text-sm text-muted">{{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} package(s)</div>
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>

      <!-- Create/Edit Modal -->
      <UModal v-model:open="editOpen" :title="editMode === 'create' ? 'New Package' : `Edit ${editTarget?.name || 'Package'}`">
        <template #body>
          <div class="space-y-4 max-h-[70vh] overflow-y-auto">
            <UFormField label="Package Name" required>
              <UInput v-model="editState.name" :disabled="editLoading" placeholder="Starter" />
            </UFormField>

            <div class="grid grid-cols-2 gap-4">
              <UFormField label="CPU Quota (%)">
                <UInput v-model.number="editState.cpu_quota" type="number" :disabled="editLoading" />
              </UFormField>
              <UFormField label="RAM (MB)">
                <UInput v-model.number="editState.memory_limit_mb" type="number" :disabled="editLoading" />
              </UFormField>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <UFormField label="Disk Space (MB)">
                <UInput v-model.number="editState.disk_quota_mb" type="number" :disabled="editLoading" />
              </UFormField>
              <UFormField label="Max Domains">
                <UInput v-model.number="editState.max_domains" type="number" :disabled="editLoading" />
              </UFormField>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <UFormField label="Max Databases">
                <UInput v-model.number="editState.max_databases" type="number" :disabled="editLoading" />
              </UFormField>
              <UFormField label="Max Cron Jobs">
                <UInput v-model.number="editState.max_cron_jobs" type="number" :disabled="editLoading" />
              </UFormField>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <UFormField label="Max Processes">
                <UInput v-model.number="editState.max_procs" type="number" :disabled="editLoading" />
              </UFormField>
              <UFormField label="Max FTP Accounts">
                <UInput v-model.number="editState.max_ftp_accounts" type="number" :disabled="editLoading" />
              </UFormField>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <UFormField label="IO Read (MB/s)">
                <UInput v-model.number="editState.io_read_mbps" type="number" :disabled="editLoading" />
              </UFormField>
              <UFormField label="IO Write (MB/s)">
                <UInput v-model.number="editState.io_write_mbps" type="number" :disabled="editLoading" />
              </UFormField>
            </div>

            <UFormField label="Allowed PHP Versions">
              <UInput v-model="editState.php_versions_allowed" :disabled="editLoading" placeholder="8.3,8.2,8.1" />
            </UFormField>

            <div class="flex gap-4">
              <UFormField label="Antivirus">
                <USwitch v-model="editState.antivirus_enabled" :disabled="editLoading" />
              </UFormField>
              <UFormField label="Terminal Access">
                <USwitch v-model="editState.terminal_enabled" :disabled="editLoading" />
              </UFormField>
              <UFormField label="Backups">
                <USwitch v-model="editState.backup_enabled" :disabled="editLoading" />
              </UFormField>
            </div>

            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="editOpen = false" :disabled="editLoading" />
              <UButton :label="editMode === 'create' ? 'Create' : 'Save'" color="primary" :loading="editLoading" @click="onPackageSubmit" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Delete confirmation -->
      <UModal v-model:open="deleteOpen" :title="`Delete ${deleteTarget?.name || 'Package'}`"
        description="This cannot be undone. Users on this package keep their current limits until reassigned.">
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
