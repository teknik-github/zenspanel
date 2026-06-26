<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/backup-targets' })

type BackupTarget = {
  id: number
  name: string
  type: string
  bucket: string
  prefix?: string
  access_key: string
  region?: string
  endpoint?: string
  enabled: boolean
}

type BackupTargetsResponse = {
  data?: BackupTarget[]
}

type TestResponse = {
  ok?: boolean
  error?: string
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

const deleteTarget = ref<BackupTarget | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

const editOpen = ref(false)
const editLoading = ref(false)
const editMode = ref<'create' | 'edit'>('create')
const editTarget = ref<BackupTarget | null>(null)
const testingTargetId = ref<number | null>(null)

const editState = reactive({
  name: '',
  type: 's3',
  bucket: '',
  prefix: '',
  access_key: '',
  secret_key: '',
  region: 'us-east-1',
  endpoint: '',
  enabled: true
})

const { data, status, refresh } = await useFetch<BackupTarget[] | BackupTargetsResponse>('/api/v1/admin/backup-targets', {
  lazy: true
})

const targets = computed<BackupTarget[]>(() => {
  const raw = data.value

  if (!raw) {
    return []
  }

  if (Array.isArray(raw)) {
    return raw
  }

  return Array.isArray(raw.data) ? raw.data : []
})

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

function resetEditState(target?: BackupTarget) {
  Object.assign(editState, {
    name: target?.name || '',
    type: target?.type || 's3',
    bucket: target?.bucket || '',
    prefix: target?.prefix || '',
    access_key: target?.access_key || '',
    secret_key: '',
    region: target?.region || 'us-east-1',
    endpoint: target?.endpoint || '',
    enabled: target?.enabled ?? true
  })
}

function showEdit(target?: BackupTarget) {
  editMode.value = target ? 'edit' : 'create'
  editTarget.value = target || null
  resetEditState(target)
  editOpen.value = true
}

function showDelete(target: BackupTarget) {
  deleteTarget.value = target
  deleteOpen.value = true
}

async function testTarget(target: BackupTarget) {
  testingTargetId.value = target.id

  try {
    const result = await $fetch<TestResponse>(`/api/v1/admin/backup-targets/${target.id}/test`, {
      method: 'POST'
    })

    toast.add({
      title: result.ok ? 'Connection OK' : 'Connection failed',
      description: result.ok ? `${target.name} is reachable.` : result.error,
      color: result.ok ? 'success' : 'error'
    })
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    testingTargetId.value = null
  }
}

async function handleDelete() {
  if (!deleteTarget.value) {
    return
  }

  deleteLoading.value = true

  try {
    await $fetch(`/api/v1/admin/backup-targets/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Backup target deleted', color: 'success' })
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

async function handleSubmit() {
  if (!editState.name.trim() || !editState.bucket.trim() || !editState.access_key.trim()) {
    toast.add({
      title: 'Missing backup target details',
      description: 'Name, bucket, and access key are required.',
      color: 'warning'
    })
    return
  }

  if (editMode.value === 'create' && !editState.secret_key.trim()) {
    toast.add({
      title: 'Secret key is required',
      color: 'warning'
    })
    return
  }

  editLoading.value = true

  const payload = {
    name: editState.name.trim(),
    type: editState.type,
    bucket: editState.bucket.trim(),
    prefix: editState.prefix.trim(),
    access_key: editState.access_key.trim(),
    secret_key: editState.secret_key,
    region: editState.region.trim(),
    endpoint: editState.endpoint.trim(),
    enabled: editState.enabled
  }

  try {
    if (editMode.value === 'create') {
      await $fetch('/api/v1/admin/backup-targets', {
        method: 'POST',
        body: payload
      })
    } else if (editTarget.value) {
      await $fetch(`/api/v1/admin/backup-targets/${editTarget.value.id}`, {
        method: 'PUT',
        body: payload
      })
    }

    toast.add({
      title: editMode.value === 'create' ? 'Backup target created' : 'Backup target updated',
      color: 'success'
    })
    editOpen.value = false
    refresh()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    editLoading.value = false
  }
}

function getRowItems(row: Row<BackupTarget>): DropdownMenuItem[] {
  const target = row.original

  return [
    { type: 'label', label: `Target: ${target.name}` },
    {
      label: 'Edit',
      icon: 'i-lucide-pencil',
      onSelect: () => showEdit(target)
    },
    {
      label: testingTargetId.value === target.id ? 'Testing...' : 'Test Connection',
      icon: 'i-lucide-zap',
      disabled: testingTargetId.value === target.id,
      onSelect: () => testTarget(target)
    },
    { type: 'separator' },
    {
      label: 'Delete',
      icon: 'i-lucide-trash',
      color: 'error',
      onSelect: () => showDelete(target)
    }
  ]
}

const columns: TableColumn<BackupTarget>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => h('span', { class: 'font-medium text-highlighted' }, row.original.name)
  },
  {
    accessorKey: 'type',
    header: 'Type',
    cell: ({ row }) => h('span', { class: 'font-mono text-xs uppercase text-dimmed' }, row.original.type)
  },
  { accessorKey: 'bucket', header: 'Bucket' },
  {
    accessorKey: 'region',
    header: 'Region',
    cell: ({ row }) => row.original.region || '-'
  },
  {
    accessorKey: 'enabled',
    header: 'Enabled',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: row.original.enabled ? 'success' : 'neutral'
      },
      () => row.original.enabled ? 'Yes' : 'No'
    )
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
          icon: testingTargetId.value === row.original.id ? 'i-lucide-loader-circle' : 'i-lucide-ellipsis-vertical',
          color: 'neutral',
          variant: 'ghost',
          class: 'ml-auto',
          loading: testingTargetId.value === row.original.id
        })
      )
    )
  }
]

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="backup-targets">
    <template #header>
      <UDashboardNavbar title="Backup Targets">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            label="Add Target"
            icon="i-lucide-plus"
            @click="showEdit()"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable
        ref="table"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="targets"
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

      <UPagination
        :default-page="1"
        :items-per-page="20"
        :total="targets.length"
        class="mt-4 justify-end"
      />

      <UModal
        v-model:open="editOpen"
        :title="editMode === 'create' ? 'Add Backup Target' : 'Edit Backup Target'"
      >
        <template #body>
          <div class="max-h-[72vh] space-y-5 overflow-y-auto pr-1">
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField label="Name" required>
                <UInput
                  v-model="editState.name"
                  icon="i-lucide-hard-drive-upload"
                  :disabled="editLoading"
                  placeholder="Wasabi"
                />
              </UFormField>

              <UFormField label="Type">
                <USelect
                  v-model="editState.type"
                  :items="['s3']"
                  :disabled="editLoading"
                />
              </UFormField>
            </div>

            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField label="Bucket" required>
                <UInput
                  v-model="editState.bucket"
                  :disabled="editLoading"
                  placeholder="my-backups"
                />
              </UFormField>

              <UFormField label="Region">
                <UInput
                  v-model="editState.region"
                  :disabled="editLoading"
                  placeholder="us-east-1"
                />
              </UFormField>
            </div>

            <UFormField label="Prefix">
              <UInput
                v-model="editState.prefix"
                icon="i-lucide-folder"
                :disabled="editLoading"
                placeholder="zenspanel/"
              />
            </UFormField>

            <UFormField label="Endpoint">
              <UInput
                v-model="editState.endpoint"
                icon="i-lucide-link"
                :disabled="editLoading"
                placeholder="https://s3.wasabisys.com"
              />
            </UFormField>

            <div class="rounded-lg border border-default p-4">
              <div class="mb-4 flex items-center gap-2">
                <UIcon name="i-lucide-lock-keyhole" class="size-4 text-dimmed" />
                <p class="text-sm font-medium text-highlighted">
                  Credentials
                </p>
              </div>

              <div class="space-y-4">
                <UFormField label="Access Key" required>
                  <UInput
                    v-model="editState.access_key"
                    :disabled="editLoading"
                    autocomplete="off"
                    placeholder="AKIAIOSFODNN7EXAMPLE"
                  />
                </UFormField>

                <UFormField
                  :label="editMode === 'edit' ? 'Secret Key' : 'Secret Key'"
                  :required="editMode === 'create'"
                >
                  <UInput
                    v-model="editState.secret_key"
                    type="password"
                    :disabled="editLoading"
                    autocomplete="new-password"
                    :placeholder="editMode === 'edit' ? 'Leave blank to keep current secret' : 'Secret access key'"
                  />
                </UFormField>
              </div>
            </div>

            <div class="flex items-center justify-between gap-4 rounded-lg border border-default p-3">
              <div>
                <p class="text-sm font-medium text-highlighted">
                  Enabled
                </p>
                <p class="text-xs text-dimmed">
                  Available as a remote destination for future backups.
                </p>
              </div>
              <USwitch v-model="editState.enabled" :disabled="editLoading" />
            </div>

            <div class="flex justify-end gap-2 pt-1">
              <UButton
                label="Cancel"
                color="neutral"
                variant="subtle"
                :disabled="editLoading"
                @click="editOpen = false"
              />
              <UButton
                :label="editMode === 'create' ? 'Create' : 'Save'"
                icon="i-lucide-save"
                color="primary"
                :loading="editLoading"
                @click="handleSubmit"
              />
            </div>
          </div>
        </template>
      </UModal>

      <UModal
        v-model:open="deleteOpen"
        :title="`Delete ${deleteTarget?.name || 'Target'}`"
        description="This removes the backup destination from admin configuration."
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
    </template>
  </UDashboardPanel>
</template>
