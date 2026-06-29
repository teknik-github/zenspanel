<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/backups' })

type NullableString = string | {
  String?: string
  Valid?: boolean
}

type NullableInt = number | {
  Int64?: number
  Valid?: boolean
}

type BackupType = 'db' | 'files' | 'full' | 'domain'

type Backup = {
  id: number
  user_id: number
  type: BackupType
  status: 'pending' | 'running' | 'done' | 'failed' | 'restoring' | 'restore_failed'
  size_bytes?: NullableInt
  error_msg?: NullableString
  created_at?: string
}

type BackupsResponse = {
  data?: Backup[]
}

type ApiError = {
  data?: {
    error?: string
  }
}

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()
const table = useTemplateRef('table')

const createOpen = ref(false)
const createLoading = ref(false)
const backupType = ref<'db' | 'files'>('files')
const selectedDomainIds = ref<number[]>([])
const selectedDbIds = ref<number[]>([])
const downloadingBackupId = ref<number | null>(null)

const { data: domainsData } = await useFetch('/api/v1/domains', { query: { limit: 100 } })
const domainItems = computed(() => {
  const r = domainsData.value as any
  const list = Array.isArray(r?.data) ? r.data : []
  return list.map((d: any) => ({ label: d.domain, value: d.id }))
})

const { data: dbsData } = await useFetch('/api/v1/databases', { query: { limit: 100 } })
const dbItems = computed(() => {
  const r = dbsData.value as any
  const list = Array.isArray(r?.data) ? r.data : []
  return list.map((d: any) => ({ label: d.db_name, value: d.id }))
})

const deleteTarget = ref<Backup | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

const backupTypeOptions = [
  {
    label: 'Website',
    value: 'files',
    icon: 'i-lucide-folder-archive',
    description: 'Backup home directory and website files.'
  },
  {
    label: 'Database',
    value: 'db',
    icon: 'i-lucide-database-backup',
    description: 'Backup all databases owned by this account.'
  }
]

const { data, status, refresh } = await useFetch<Backup[] | BackupsResponse>('/api/v1/backups', {
  lazy: true,
  query: { limit: 100 }
})

const backups = computed<Backup[]>(() => {
  const raw = data.value

  if (!raw) {
    return []
  }

  if (Array.isArray(raw)) {
    return raw
  }

  return Array.isArray(raw.data) ? raw.data : []
})

const runningBackups = computed(() => {
  return backups.value.filter(backup => backup.status === 'pending' || backup.status === 'running').length
})

useIntervalFn(() => {
  if (runningBackups.value > 0) {
    refresh()
  }
}, 5000)

function formatBytes(value?: NullableInt) {
  const bytes = typeof value === 'object' ? value?.Int64 || 0 : Number(value) || 0

  if (!bytes) {
    return '—'
  }

  if (bytes >= 1073741824) {
    return `${(bytes / 1073741824).toFixed(1)} GB`
  }

  return `${(bytes / 1048576).toFixed(0)} MB`
}

function formatBackupType(type: BackupType) {
  const labels: Record<BackupType, string> = {
    db: 'Database',
    files: 'Website',
    full: 'Full',
    domain: 'Domain'
  }

  return labels[type] || type
}

function getTypeColor(type: BackupType) {
  if (type === 'db') {
    return 'warning'
  }

  if (type === 'files') {
    return 'info'
  }

  if (type === 'full') {
    return 'primary'
  }

  return 'neutral'
}

function getStatusColor(statusValue: Backup['status']) {
  if (statusValue === 'done') {
    return 'success'
  }

  if (statusValue === 'running' || statusValue === 'restoring') {
    return 'info'
  }

  if (statusValue === 'pending') {
    return 'warning'
  }

  return 'error'
}

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

function sanitizeDownloadFilename(filename: string) {
  return filename
    .replace(/[\0\r\n/\\]/g, '_')
    .trim()
}

function getDownloadFilename(contentDisposition: string | null, backup: Backup) {
  const encodedFilename = contentDisposition?.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encodedFilename) {
    try {
      return sanitizeDownloadFilename(decodeURIComponent(encodedFilename)) || `backup-${backup.id}-${backup.type}.tar.gz`
    } catch {
      return `backup-${backup.id}-${backup.type}.tar.gz`
    }
  }

  const filename = contentDisposition?.match(/filename="?([^";]+)"?/i)?.[1]
  return filename
    ? sanitizeDownloadFilename(filename) || `backup-${backup.id}-${backup.type}.tar.gz`
    : `backup-${backup.id}-${backup.type}.tar.gz`
}

async function downloadBackup(backup: Backup) {
  downloadingBackupId.value = backup.id

  try {
    const response = await $fetch.raw<Blob>(`/api/v1/backups/${backup.id}/download`, {
      responseType: 'blob'
    })
    const blob = response._data

    if (!blob) {
      throw new Error('Empty download response')
    }

    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')

    link.href = url
    link.download = getDownloadFilename(response.headers.get('content-disposition'), backup)
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch (error: unknown) {
    toast.add({
      title: 'Download failed',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    downloadingBackupId.value = null
  }
}

function getRowItems(row: Row<Backup>) {
  const backup = row.original
  const items: DropdownMenuItem[] = [
    { type: 'label', label: `Backup #${backup.id}` }
  ]

  if (backup.status === 'done') {
    items.push({
      label: downloadingBackupId.value === backup.id ? 'Downloading...' : 'Download',
      icon: 'i-lucide-download',
      disabled: downloadingBackupId.value === backup.id,
      onSelect: () => downloadBackup(backup)
    })
  }

  items.push(
    { type: 'separator' },
    {
      label: 'Delete',
      icon: 'i-lucide-trash',
      color: 'error',
      onSelect: () => showDelete(backup)
    }
  )

  return items
}

function showCreate() {
  backupType.value = 'files'
  selectedDomainIds.value = []
  selectedDbIds.value = []
  createOpen.value = true
}

async function handleCreate() {
  createLoading.value = true

  try {
    const body: Record<string, unknown> = { type: backupType.value }
    if (backupType.value === 'files' && selectedDomainIds.value.length > 0)
      body.domain_ids = selectedDomainIds.value
    if (backupType.value === 'db' && selectedDbIds.value.length > 0)
      body.db_ids = selectedDbIds.value
    await $fetch('/api/v1/backups', { method: 'POST', body })

    toast.add({
      title: 'Backup started',
      description: `${formatBackupType(backupType.value)} backup is running in the background.`,
      color: 'success'
    })
    createOpen.value = false
    refresh()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    createLoading.value = false
  }
}

function showDelete(backup: Backup) {
  deleteTarget.value = backup
  deleteOpen.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) {
    return
  }

  deleteLoading.value = true

  try {
    await $fetch(`/api/v1/backups/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    deleteLoading.value = false
  }
}

const columns: TableColumn<Backup>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'type',
    header: 'Type',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: getTypeColor(row.original.type)
      },
      () => formatBackupType(row.original.type)
    )
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: getStatusColor(row.original.status),
        class: 'capitalize'
      },
      () => row.original.status
    )
  },
  {
    accessorKey: 'size_bytes',
    header: 'Size',
    cell: ({ row }) => formatBytes(row.original.size_bytes)
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => row.original.created_at?.slice(0, 16) || '—'
  },
  {
    id: 'actions',
    cell: ({ row }) => h(
      'div',
      { class: 'text-right' },
      h(
        UDropdownMenu,
        {
          content: { align: 'end' },
          items: getRowItems(row)
        },
        () => h(UButton, {
          icon: 'i-lucide-ellipsis-vertical',
          color: 'neutral',
          variant: 'ghost',
          class: 'ml-auto'
        })
      )
    )
  }
]

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="backups">
    <template #header>
      <UDashboardNavbar title="Backups">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            label="Create Backup"
            icon="i-lucide-plus"
            @click="showCreate"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <p class="text-sm text-dimmed">
            Create website or database backups, then download completed archives.
          </p>

          <UBadge v-if="runningBackups" color="info" variant="subtle">
            {{ runningBackups }} running
          </UBadge>
        </div>

        <UTable
          ref="table"
          v-model:pagination="pagination"
          :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
          class="shrink-0"
          :data="backups"
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

        <div class="flex justify-end">
          <UPagination
            :default-page="1"
            :items-per-page="20"
            :total="backups.length"
          />
        </div>

        <UModal v-model:open="createOpen" title="Create Backup">
          <template #body>
            <div class="space-y-4">
              <URadioGroup
                v-model="backupType"
                :items="backupTypeOptions"
                value-key="value"
                label-key="label"
                description-key="description"
                :disabled="createLoading"
              />

              <UFormField v-if="backupType === 'files'" label="Domains to backup">
                <USelectMenu
                  v-model="selectedDomainIds"
                  :items="domainItems"
                  value-key="value"
                  label-key="label"
                  multiple
                  placeholder="Leave empty to backup all domains"
                  :disabled="createLoading"
                  class="w-full"
                />
                <template #hint>
                  <span class="text-xs text-dimmed">Leave empty to backup the entire home directory</span>
                </template>
              </UFormField>

              <UFormField v-if="backupType === 'db'" label="Databases to backup">
                <USelectMenu
                  v-model="selectedDbIds"
                  :items="dbItems"
                  value-key="value"
                  label-key="label"
                  multiple
                  placeholder="Leave empty to backup all databases"
                  :disabled="createLoading"
                  class="w-full"
                />
                <template #hint>
                  <span class="text-xs text-dimmed">Leave empty to backup all databases</span>
                </template>
              </UFormField>

              <div class="flex justify-end gap-2 pt-2">
                <UButton
                  label="Cancel"
                  color="neutral"
                  variant="subtle"
                  :disabled="createLoading"
                  @click="createOpen = false"
                />
                <UButton
                  label="Start Backup"
                  icon="i-lucide-play"
                  color="primary"
                  :loading="createLoading"
                  @click="handleCreate"
                />
              </div>
            </div>
          </template>
        </UModal>

        <UModal
          v-model:open="deleteOpen"
          :title="`Delete Backup #${deleteTarget?.id}`"
          description="Removes backup record and archive file."
        >
          <template #body>
            <div class="flex justify-end gap-2">
              <UButton
                label="Cancel"
                color="neutral"
                variant="subtle"
                :disabled="deleteLoading"
                @click="deleteOpen = false"
              />
              <UButton
                label="Delete"
                color="error"
                :loading="deleteLoading"
                @click="handleDelete"
              />
            </div>
          </template>
        </UModal>
      </div>
    </template>
  </UDashboardPanel>
</template>
