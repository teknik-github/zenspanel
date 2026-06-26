<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/api-keys' })

type SqlTime = string | {
  Time?: string
  String?: string
  Valid?: boolean
}

type ApiKey = {
  id: number
  name: string
  key_prefix: string
  permissions: string
  last_used_at?: SqlTime
  expires_at?: SqlTime
  created_at?: SqlTime
}

type ApiKeysResponse = {
  data?: ApiKey[]
}

type CreateApiKeyResponse = {
  key?: string
}

type ApiError = {
  data?: {
    error?: string
  }
}

const permissionOptions = [
  {
    value: 'read_user',
    label: 'Read users',
    description: 'View users and account usage.'
  },
  {
    value: 'create_user',
    label: 'Create users',
    description: 'Provision new customer accounts.'
  },
  {
    value: 'suspend_user',
    label: 'Suspend users',
    description: 'Suspend and unsuspend accounts.'
  },
  {
    value: 'change_package',
    label: 'Change packages',
    description: 'Move users between hosting packages.'
  },
  {
    value: 'read_package',
    label: 'Read packages',
    description: 'View hosting package metadata.'
  }
]

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()
const table = useTemplateRef('table')
const { copy } = useClipboard()

const createOpen = ref(false)
const createLoading = ref(false)
const createdKey = ref('')
const selectedPermissions = ref<string[]>([])
const createState = reactive({ name: '' })

const deleteTarget = ref<ApiKey | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

const { data, status, refresh } = await useFetch<ApiKey[] | ApiKeysResponse>('/api/v1/api-keys', {
  lazy: true
})

const keys = computed<ApiKey[]>(() => {
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

function formatSqlTime(value?: SqlTime) {
  if (!value) {
    return '-'
  }

  if (typeof value === 'string') {
    return value.slice(0, 10) || '-'
  }

  if (value.Valid === false) {
    return '-'
  }

  return value.Time?.slice(0, 10) || value.String || '-'
}

function permissionLabels(permissions: string) {
  const values = permissions.split(',').map(permission => permission.trim()).filter(Boolean)

  return values.map((value) => {
    return permissionOptions.find(option => option.value === value)?.label || value
  })
}

function togglePermission(permission: string, enabled: boolean) {
  if (enabled && !selectedPermissions.value.includes(permission)) {
    selectedPermissions.value = [...selectedPermissions.value, permission]
    return
  }

  if (!enabled) {
    selectedPermissions.value = selectedPermissions.value.filter(value => value !== permission)
  }
}

function showCreate() {
  createState.name = ''
  selectedPermissions.value = ['read_user', 'read_package']
  createdKey.value = ''
  createOpen.value = true
}

async function handleCreate() {
  if (!createState.name.trim() || !selectedPermissions.value.length) {
    toast.add({
      title: 'Missing API key details',
      description: 'Name and at least one permission are required.',
      color: 'warning'
    })
    return
  }

  createLoading.value = true

  try {
    const res = await $fetch<CreateApiKeyResponse>('/api/v1/api-keys', {
      method: 'POST',
      body: {
        name: createState.name.trim(),
        permissions: selectedPermissions.value.join(',')
      }
    })

    createdKey.value = res.key || ''
    toast.add({
      title: 'API key created',
      description: 'Copy it now. It is shown only once.',
      color: 'success'
    })
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

function showDelete(key: ApiKey) {
  deleteTarget.value = key
  deleteOpen.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) {
    return
  }

  deleteLoading.value = true

  try {
    await $fetch(`/api/v1/api-keys/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'API key revoked', color: 'success' })
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

function copyCreatedKey() {
  copy(createdKey.value)
  toast.add({ title: 'Copied', color: 'success' })
}

function getRowItems(row: Row<ApiKey>): DropdownMenuItem[] {
  return [
    { type: 'label', label: `Key: ${row.original.name}` },
    {
      label: 'Revoke',
      icon: 'i-lucide-trash',
      color: 'error',
      onSelect: () => showDelete(row.original)
    }
  ]
}

const columns: TableColumn<ApiKey>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => h('span', { class: 'font-medium text-highlighted' }, row.original.name)
  },
  {
    accessorKey: 'key_prefix',
    header: 'Prefix',
    cell: ({ row }) => h('code', { class: 'rounded bg-elevated px-1.5 py-0.5 font-mono text-xs' }, row.original.key_prefix)
  },
  {
    accessorKey: 'permissions',
    header: 'Permissions',
    cell: ({ row }) => h(
      'div',
      { class: 'flex flex-wrap gap-1' },
      permissionLabels(row.original.permissions).map(label => h(
        UBadge,
        { color: 'neutral', variant: 'subtle', size: 'sm' },
        () => label
      ))
    )
  },
  {
    accessorKey: 'last_used_at',
    header: 'Last Used',
    cell: ({ row }) => formatSqlTime(row.original.last_used_at)
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
  <UDashboardPanel id="api-keys">
    <template #header>
      <UDashboardNavbar title="API Keys">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            label="Create API Key"
            icon="i-lucide-plus"
            @click="showCreate"
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
        :data="keys"
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
        :total="keys.length"
        class="mt-4 justify-end"
      />

      <UModal v-model:open="createOpen" title="Create API Key">
        <template #body>
          <div class="space-y-5">
            <UFormField label="Name" required>
              <UInput
                v-model="createState.name"
                icon="i-lucide-key-round"
                :disabled="createLoading || !!createdKey"
                placeholder="WHMCS Integration"
              />
            </UFormField>

            <div class="space-y-3">
              <p class="text-sm font-medium text-highlighted">
                Permissions
              </p>

              <div class="grid gap-2 sm:grid-cols-2">
                <label
                  v-for="permission in permissionOptions"
                  :key="permission.value"
                  class="flex cursor-pointer gap-3 rounded-lg border border-default p-3 transition-colors hover:bg-elevated/50"
                  :class="selectedPermissions.includes(permission.value) ? 'bg-primary/5 border-primary/40' : ''"
                >
                  <UCheckbox
                    :model-value="selectedPermissions.includes(permission.value)"
                    :disabled="createLoading || !!createdKey"
                    @update:model-value="togglePermission(permission.value, Boolean($event))"
                  />
                  <span class="min-w-0">
                    <span class="block text-sm font-medium text-highlighted">{{ permission.label }}</span>
                    <span class="block text-xs text-dimmed">{{ permission.description }}</span>
                  </span>
                </label>
              </div>
            </div>

            <div
              v-if="createdKey"
              class="rounded-lg border border-success/30 bg-success/10 p-4"
            >
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="font-medium text-success">
                    API key created
                  </p>
                  <p class="mt-1 text-xs text-dimmed">
                    Store it securely. This value will not be shown again.
                  </p>
                </div>
                <UButton
                  icon="i-lucide-copy"
                  color="neutral"
                  variant="subtle"
                  size="sm"
                  @click="copyCreatedKey"
                />
              </div>

              <code class="mt-3 block rounded bg-elevated p-3 font-mono text-xs break-all">
                {{ createdKey }}
              </code>
            </div>

            <div class="flex justify-end gap-2 pt-1">
              <UButton
                :label="createdKey ? 'Close' : 'Cancel'"
                color="neutral"
                variant="subtle"
                :disabled="createLoading"
                @click="createOpen = false; createdKey = ''"
              />
              <UButton
                v-if="!createdKey"
                label="Create"
                icon="i-lucide-plus"
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
        :title="`Revoke ${deleteTarget?.name || 'Key'}`"
        description="This key will stop working immediately."
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
              label="Revoke"
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
