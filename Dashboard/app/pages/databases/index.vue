<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

const { data, status, refresh } = await useFetch('/api/v1/databases', { lazy: true, query: { limit: 100 } })

const databases = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

function getRowItems(row: Row<any>) {
  const d = row.original
  return [
    { type: 'label', label: `DB: ${d.db_name}` },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error', onSelect: () => showDelete(d) }
  ]
}

const delTarget = ref<any>(null), delOpen = ref(false), delLoading = ref(false)
function showDelete(d: any) { delTarget.value = d; delOpen.value = true }
async function handleDelete() {
  if (!delTarget.value) return; delLoading.value = true
  try {
    await $fetch(`/api/v1/databases/${delTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', color: 'success' }); delOpen.value = false; refresh()
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { delLoading.value = false }
}

// Create
const createOpen = ref(false), createLoading = ref(false)
const createState = reactive({ db_name: '', db_user: '', db_password: '' })
const createdPwd = ref('')

async function handleCreate() {
  createLoading.value = true
  try {
    const res: any = await $fetch('/api/v1/databases', { method: 'POST', body: createState })
    createdPwd.value = res.db_password || ''
    toast.add({ title: 'Database created', description: 'Save the password — it is not stored.', color: 'success' })
    refresh()
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { createLoading.value = false }
}

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'db_name', header: 'Database', cell: ({ row }: any) => h('span', { class: 'font-medium font-mono text-highlighted' }, row.original.db_name) },
  { accessorKey: 'db_user', header: 'User', cell: ({ row }: any) => h('span', { class: 'font-mono text-sm' }, row.original.db_user) },
  { accessorKey: 'user_id', header: 'Owner ID' },
  { accessorKey: 'created_at', header: 'Created', cell: ({ row }: any) => row.original.created_at?.slice(0, 10) },
  { id: 'actions', cell: ({ row }: any) => h('div', { class: 'text-right' },
      h(UDropdownMenu, { content: { align: 'end' }, items: getRowItems(row) },
        () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' }))) }
]

const search = ref('')
const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
<UDashboardPanel id="databases">
  <template #header>
    <UDashboardNavbar title="Databases">
      <template #leading><UDashboardSidebarCollapse /></template>
      <template #right><UButton label="New Database" icon="i-lucide-plus" @click="createOpen=true" /></template>
    </UDashboardNavbar>
  </template>
  <template #body>
    <UInput v-model="search" class="max-w-sm mb-4" icon="i-lucide-search" placeholder="Filter databases..." />
    <UTable ref="table" v-model:pagination="pagination" :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }" class="shrink-0"
      :data="databases" :columns="columns" :loading="status === 'pending'"
      :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50 [&>tr]:after:content-none', tbody:'[&>tr]:last:[&>td]:border-b-0', th:'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r', td:'border-b border-default', separator:'h-0' }" />
    <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
      <div class="text-sm text-muted">{{ databases.length }} database(s)</div>
      <UPagination :default-page="(table?.tableApi?.getState().pagination.pageIndex||0)+1"
        :items-per-page="table?.tableApi?.getState().pagination.pageSize" :total="databases.length"
        @update:page="(p:number)=>table?.tableApi?.setPageIndex(p-1)" />
    </div>

    <UModal v-model:open="createOpen" title="New Database">
      <template #body>
        <div class="space-y-4">
          <UFormField label="Database Name" required>
            <UInput v-model="createState.db_name" :disabled="createLoading" placeholder="alice_wp" />
          </UFormField>
          <UFormField label="Database User" required>
            <UInput v-model="createState.db_user" :disabled="createLoading" placeholder="alice_wp" />
          </UFormField>
          <UFormField label="Password" required>
            <UInput v-model="createState.db_password" type="password" :disabled="createLoading" placeholder="StrongPass123" />
          </UFormField>
          <div v-if="createdPwd" class="p-3 bg-success/10 border border-success/20 rounded-lg text-sm">
            <p class="font-semibold text-success">Database created!</p>
            <p class="mt-1">Password: <code class="font-mono bg-success/10 px-1 rounded">{{ createdPwd }}</code></p>
            <p class="text-xs text-dimmed mt-1">This password will not be shown again.</p>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <UButton label="Cancel" color="neutral" variant="subtle" @click="createOpen=false;createdPwd=''" :disabled="createLoading" />
            <UButton label="Create" color="primary" :loading="createLoading" @click="handleCreate" />
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="delOpen" :title="`Delete ${delTarget?.db_name||'Database'}`" description="Drops database and MySQL user. This action cannot be undone.">
      <template #body>
        <div class="flex justify-end gap-2">
          <UButton label="Cancel" color="neutral" variant="subtle" @click="delOpen=false" :disabled="delLoading" />
          <UButton label="Delete" color="error" :loading="delLoading" @click="handleDelete" />
        </div>
      </template>
    </UModal>
  </template>
</UDashboardPanel>
</template>
