<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { FormSubmitEvent } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'
import * as z from 'zod'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const UCheckbox = resolveComponent('UCheckbox')

const toast = useToast()
const table = useTemplateRef('table')

const columnFilters = ref([{ id: 'username', value: '' }])
const columnVisibility = ref()
const rowSelection = ref({})

// ── Fetch users from API ──
const { data, status, refresh } = await useFetch('/api/v1/users', {
  lazy: true,
  query: { limit: 100 }
})

const users = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Row actions ──
function getRowItems(row: Row<any>) {
  const user = row.original
  return [
    { type: 'label', label: `User: ${user.username}` },
    { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => showEditModal(user) },
    {
      label: user.status === 'active' ? 'Suspend' : 'Unsuspend',
      icon: 'i-lucide-ban',
      onSelect: async () => {
        const action = user.status === 'active' ? 'suspend' : 'unsuspend'
        try {
          await $fetch(`/api/v1/users/${user.id}/${action}`, { method: 'PUT' })
          toast.add({ title: `User ${action}ed`, color: 'success' })
          refresh()
        } catch (e: any) {
          toast.add({ title: 'Error', description: e.data?.error, color: 'error' })
        }
      }
    },
    { type: 'separator' },
    { label: 'Delete user', icon: 'i-lucide-trash', color: 'error',
      onSelect: () => showDeleteModal(user)
    }
  ]
}

// ── Delete ──
const deleteTarget = ref<{ id: number; username: string } | null>(null)
const deleteOpen = ref(false)
const deleteLoading = ref(false)

function showDeleteModal(user: any) {
  deleteTarget.value = { id: user.id, username: user.username }
  deleteOpen.value = true
}
async function handleDelete() {
  if (!deleteTarget.value) return
  deleteLoading.value = true
  try {
    await $fetch(`/api/v1/users/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', description: `${deleteTarget.value.username} removed.`, color: 'success' })
    deleteOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Delete failed', color: 'error' })
  } finally { deleteLoading.value = false }
}

// ── Edit / Create ──
const editOpen = ref(false)
const editLoading = ref(false)
const editMode = ref<'create' | 'edit'>('create')
const editTarget = ref<any>(null)

const userSchema = z.object({
  username: z.string().min(3).max(32).regex(/^[a-z0-9-]+$/, 'Lowercase letters, digits, hyphens'),
  email: z.string().email(),
  password: z.string().min(6).optional(),
  package_id: z.number().optional(),
  terminal_enabled: z.boolean(),
  backup_enabled: z.boolean(),
  php_version: z.string().default('8.3')
})

const editState = reactive<any>({
  username: '',
  email: '',
  password: '',
  package_id: undefined,
  terminal_enabled: true,
  backup_enabled: false,
  php_version: '8.3'
})

function showEditModal(user?: any) {
  editMode.value = user ? 'edit' : 'create'
  editTarget.value = user
  if (user) {
    Object.assign(editState, {
      username: user.username,
      email: user.email,
      password: '',
      package_id: user.package_id?.Int64 || user.package_id || undefined,
      terminal_enabled: user.terminal_enabled,
      backup_enabled: user.backup_enabled,
      php_version: user.php_version || '8.3'
    })
  } else {
    Object.assign(editState, {
      username: '', email: '', password: '',
      package_id: undefined, terminal_enabled: true,
      backup_enabled: false, php_version: '8.3'
    })
  }
  editOpen.value = true
}

async function onUserSubmit() {
  editLoading.value = true
  try {
    if (editMode.value === 'create') {
      await $fetch('/api/v1/users', { method: 'POST', body: editState })
      toast.add({ title: 'User created', color: 'success' })
    } else {
      const { username, password, ...updateFields } = editState
      await $fetch(`/api/v1/users/${editTarget.value.id}`, { method: 'PUT', body: updateFields })
      toast.add({ title: 'User updated', color: 'success' })
    }
    editOpen.value = false
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Failed', color: 'error' })
  } finally { editLoading.value = false }
}

// ── Columns ──
const columns: TableColumn<any>[] = [
  {
    id: 'select',
    header: ({ table }: any) =>
      h(UCheckbox, {
        modelValue: table.getIsSomePageRowsSelected() ? 'indeterminate' : table.getIsAllPageRowsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') => table.toggleAllPageRowsSelected(!!value),
        ariaLabel: 'Select all'
      }),
    cell: ({ row }: any) =>
      h(UCheckbox, {
        modelValue: row.getIsSelected(),
        'onUpdate:modelValue': (value: boolean | 'indeterminate') => row.toggleSelected(!!value),
        ariaLabel: 'Select row'
      })
  },
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'username',
    header: ({ column }: any) => {
      const isSorted = column.getIsSorted()
      return h(UButton, {
        color: 'neutral', variant: 'ghost', label: 'Username',
        icon: isSorted
          ? isSorted === 'asc' ? 'i-lucide-arrow-up-narrow-wide' : 'i-lucide-arrow-down-wide-narrow'
          : 'i-lucide-arrow-up-down',
        class: '-mx-2.5',
        onClick: () => column.toggleSorting(column.getIsSorted() === 'asc')
      })
    },
    cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.username)
  },
  { accessorKey: 'email', header: 'Email', cell: ({ row }: any) => row.original.email },
  {
    accessorKey: 'status',
    header: 'Status',
    filterFn: 'equals',
    cell: ({ row }: any) => {
      const color = row.original.status === 'active' ? 'success' : 'error'
      return h(UBadge, { class: 'capitalize', variant: 'subtle', color }, () => row.original.status)
    }
  },
  { accessorKey: 'role', header: 'Role', cell: ({ row }: any) => h('span', { class: 'text-xs font-mono text-dimmed' }, row.original.role) },
  { accessorKey: 'php_version', header: 'PHP', cell: ({ row }: any) => row.original.php_version || '—' },
  {
    id: 'terminal_enabled',
    accessorKey: 'terminal_enabled',
    header: 'Terminal',
    cell: ({ row }: any) => h(UBadge, { variant: 'subtle', color: row.original.terminal_enabled ? 'success' : 'neutral' },
      () => row.original.terminal_enabled ? 'On' : 'Off')
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

const search = computed({
  get: (): string => (table.value?.tableApi?.getColumn('username')?.getFilterValue() as string) || '',
  set: (value: string) => table.value?.tableApi?.getColumn('username')?.setFilterValue(value || undefined)
})

const statusFilter = ref('all')
watch(() => statusFilter.value, (newVal) => {
  if (!table?.value?.tableApi) return
  const col = table.value.tableApi.getColumn('status')
  if (!col) return
  col.setFilterValue(newVal === 'all' ? undefined : newVal)
})

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="users">
    <template #header>
      <UDashboardNavbar title="Users">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <UButton label="New User" icon="i-lucide-plus" @click="showEditModal()" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-1.5">
        <UInput v-model="search" class="max-w-sm" icon="i-lucide-search" placeholder="Filter usernames..." />
        <USelect
          v-model="statusFilter"
          :items="[{ label: 'All', value: 'all' }, { label: 'Active', value: 'active' }, { label: 'Suspended', value: 'suspended' }]"
          placeholder="Status" class="min-w-32"
        />
      </div>

      <UTable
        ref="table"
        v-model:column-filters="columnFilters"
        v-model:column-visibility="columnVisibility"
        v-model:row-selection="rowSelection"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="users"
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
        <div class="text-sm text-muted">{{ table?.tableApi?.getFilteredRowModel().rows.length || 0 }} user(s)</div>
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>

      <!-- Create/Edit Modal -->
      <UModal v-model:open="editOpen" :title="editMode === 'create' ? 'New User' : `Edit ${editTarget?.username || ''}`">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Username" required>
              <UInput v-model="editState.username" :disabled="editMode === 'edit' || editLoading" placeholder="alice" />
            </UFormField>
            <UFormField label="Email" required>
              <UInput v-model="editState.email" type="email" :disabled="editLoading" placeholder="alice@example.com" />
            </UFormField>
            <UFormField :label="editMode === 'create' ? 'Password' : 'New password (leave blank to keep)'" :required="editMode === 'create'">
              <UInput v-model="editState.password" type="password" :disabled="editLoading" placeholder="••••••••" />
            </UFormField>
            <div class="grid grid-cols-2 gap-4">
              <UFormField label="Package ID">
                <UInput v-model.number="editState.package_id" type="number" :disabled="editLoading" placeholder="1" />
              </UFormField>
              <UFormField label="PHP Version">
                <USelect
                  v-model="editState.php_version"
                  :items="['8.3', '8.2', '8.1', '7.4']"
                  :disabled="editLoading"
                />
              </UFormField>
            </div>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 text-sm">
                <USwitch v-model="editState.terminal_enabled" :disabled="editLoading" />
                Terminal
              </label>
              <label class="flex items-center gap-2 text-sm">
                <USwitch v-model="editState.backup_enabled" :disabled="editLoading" />
                Backups
              </label>
            </div>
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="editOpen = false" :disabled="editLoading" />
              <UButton :label="editMode === 'create' ? 'Create' : 'Update'" color="primary" :loading="editLoading" @click="onUserSubmit" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Delete confirmation -->
      <UModal v-model:open="deleteOpen" :title="`Delete ${deleteTarget?.username || 'User'}`"
        description="Tears down all resources: nginx vhosts, databases, PHP-FPM pools, cgroup slice, Linux user, home directory.">
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
