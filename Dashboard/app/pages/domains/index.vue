<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

const { data, status, refresh } = await useFetch('/api/v1/domains', { lazy: true, query: { limit: 100 } })

const domains = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

function getRowItems(row: Row<any>) {
  const d = row.original
  return [
    { type: 'label', label: `Domain: ${d.domain}` },
    { label: 'Edit', icon: 'i-lucide-pencil', onSelect: () => showEdit(d) },
    { label: d.status === 'active' ? 'Suspend' : 'Unsuspend', icon: 'i-lucide-ban',
      onSelect: async () => {
        const action = d.status === 'active' ? 'suspend' : 'unsuspend'
        try {
          await $fetch(`/api/v1/domains/${d.id}/${action}`, { method: 'POST' })
          toast.add({ title: `Domain ${action}ed`, color: 'success' })
          refresh()
        } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
      }
    },
    { type: 'separator' },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error', onSelect: () => showDelete(d) }
  ]
}

// Delete
const delTarget = ref<any>(null), delOpen = ref(false), delLoading = ref(false)
function showDelete(d: any) { delTarget.value = d; delOpen.value = true }
async function handleDelete() {
  if (!delTarget.value) return; delLoading.value = true
  try {
    await $fetch(`/api/v1/domains/${delTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Deleted', color: 'success' }); delOpen.value = false; refresh()
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { delLoading.value = false }
}

// Edit/Create
const editOpen = ref(false), editLoading = ref(false), editMode = ref<'create'|'edit'>('create')
const editTarget = ref<any>(null)
const editState = reactive({ domain: '', php_version: '8.3', user_id: undefined as number | undefined })

function showEdit(d?: any) {
  editMode.value = d ? 'edit' : 'create'; editTarget.value = d
  if (d) {
    Object.assign(editState, { domain: d.domain, php_version: d.php_version || '8.3', user_id: d.user_id })
  } else {
    Object.assign(editState, { domain: '', php_version: '8.3', user_id: undefined })
  }
  editOpen.value = true
}
async function onSubmit() {
  editLoading.value = true
  try {
    if (editMode.value === 'create') {
      await $fetch('/api/v1/domains', { method: 'POST', body: editState })
    } else {
      await $fetch(`/api/v1/domains/${editTarget.value.id}`, { method: 'PUT', body: { php_version: editState.php_version } })
    }
    toast.add({ title: editMode.value === 'create' ? 'Domain created' : 'Updated', color: 'success' })
    editOpen.value = false; refresh()
  } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) }
  finally { editLoading.value = false }
}

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'domain', header: 'Domain', cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.domain) },
  { accessorKey: 'user_id', header: 'User ID' },
  { accessorKey: 'php_version', header: 'PHP' },
  { accessorKey: 'ssl_type', header: 'SSL',
    cell: ({ row }: any) => {
      const v = row.original.ssl_type
      const color = v === 'letsencrypt' ? 'success' : v === 'custom' ? 'info' : 'neutral'
      return h(UBadge, { variant: 'subtle', color, class: 'capitalize' }, () => v || 'none')
    }
  },
  { accessorKey: 'status', header: 'Status',
    cell: ({ row }: any) => {
      const s = row.original.status
      const color = s === 'active' ? 'success' : s === 'suspended' ? 'error' : 'warning'
      return h(UBadge, { variant: 'subtle', color, class: 'capitalize' }, () => s)
    }
  },
  { accessorKey: 'created_at', header: 'Created', cell: ({ row }: any) => row.original.created_at?.slice(0, 10) },
  { id: 'actions', cell: ({ row }: any) => h('div', { class: 'text-right' },
      h(UDropdownMenu, { content: { align: 'end' }, items: getRowItems(row) },
        () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' }))) }
]

const search = ref('')
const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
<UDashboardPanel id="domains">
  <template #header>
    <UDashboardNavbar title="Domains">
      <template #leading><UDashboardSidebarCollapse /></template>
      <template #right><UButton label="Add Domain" icon="i-lucide-plus" @click="showEdit()" /></template>
    </UDashboardNavbar>
  </template>
  <template #body>
    <div class="flex flex-wrap items-center justify-between gap-1.5 mb-4">
      <UInput v-model="search" class="max-w-sm" icon="i-lucide-search" placeholder="Filter domains..." />
    </div>
    <UTable ref="table" v-model:pagination="pagination" :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }" class="shrink-0"
      :data="domains" :columns="columns" :loading="status === 'pending'"
      :ui="{ base:'table-fixed border-separate border-spacing-0', thead:'[&>tr]:bg-elevated/50 [&>tr]:after:content-none', tbody:'[&>tr]:last:[&>td]:border-b-0', th:'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r', td:'border-b border-default', separator:'h-0' }" />
    <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
      <div class="text-sm text-muted">{{ domains.length }} domain(s)</div>
      <UPagination :default-page="(table?.tableApi?.getState().pagination.pageIndex||0)+1"
        :items-per-page="table?.tableApi?.getState().pagination.pageSize" :total="domains.length"
        @update:page="(p:number)=>table?.tableApi?.setPageIndex(p-1)" />
    </div>

    <UModal v-model:open="editOpen" :title="editMode==='create'?'Add Domain':'Edit Domain'">
      <template #body>
        <div class="space-y-4">
          <UFormField label="Domain" required>
            <UInput v-model="editState.domain" :disabled="editMode==='edit'||editLoading" placeholder="example.com" />
          </UFormField>
          <UFormField v-if="editMode==='create'" label="User ID">
            <UInput v-model.number="editState.user_id" type="number" :disabled="editLoading" placeholder="5" />
          </UFormField>
          <UFormField label="PHP Version">
            <USelect v-model="editState.php_version" :items="['8.3','8.2','8.1','7.4']" :disabled="editLoading" />
          </UFormField>
          <div class="flex justify-end gap-2 pt-2">
            <UButton label="Cancel" color="neutral" variant="subtle" @click="editOpen=false" :disabled="editLoading" />
            <UButton :label="editMode==='create'?'Create':'Save'" color="primary" :loading="editLoading" @click="onSubmit" />
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="delOpen" :title="`Delete ${delTarget?.domain||'Domain'}`" description="Removes nginx vhost. All subdomains under this domain will also be deleted.">
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
