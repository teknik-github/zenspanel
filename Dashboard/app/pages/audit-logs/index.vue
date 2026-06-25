<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')

definePageMeta({ title: 'Audit Logs' })

const toast = useToast()
const table = useTemplateRef('table')

const search = ref('')
const page = ref(1)
const limit = ref(20)
const total = ref(0)

const { data, status, refresh } = await useFetch('/api/v1/audit-logs', {
  lazy: true,
  query: { page, limit, search }
})

const logs = computed(() => {
  const raw = data.value as any
  if (!raw) return []
  total.value = raw.total || 0
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

function formatNullableInt64(val: any) {
  if (!val) return '—'
  return val.Valid ? val.Int64 : '—'
}

function formatNullableString(val: any) {
  if (!val) return '—'
  return val.Valid ? val.String : '—'
}

function onChangePage(pageNum: number) {
  page.value = pageNum
}

function onChangeSearch() {
  page.value = 1
}

const columns: TableColumn<any>[] = [
  { accessorKey: 'id', header: 'ID' },
  {
    accessorKey: 'user_id',
    header: 'User ID',
    cell: ({ row }: any) => formatNullableInt64(row.original.user_id)
  },
  {
    accessorKey: 'action',
    header: 'Action',
    cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.action)
  },
  {
    accessorKey: 'resource',
    header: 'Resource',
    cell: ({ row }: any) => formatNullableString(row.original.resource)
  },
  { accessorKey: 'ip_address', header: 'IP Address' },
  {
    accessorKey: 'created_at',
    header: 'Created At',
    cell: ({ row }: any) => {
      const d = new Date(row.original.created_at)
      return h('span', { class: 'text-xs text-dimmed' }, d.toLocaleString())
    }
  }
]

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="audit-logs">
    <template #header>
      <UDashboardNavbar title="Audit Logs">
        <template #leading><UDashboardSidebarCollapse /></template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-wrap items-center justify-between gap-1.5 mb-4">
        <UInput
          v-model="search"
          class="max-w-sm"
          icon="i-lucide-search"
          placeholder="Search logs by action, IP..."
          @input="onChangeSearch"
        />
        <div class="text-sm text-muted">{{ total }} total entries</div>
      </div>

      <UTable
        ref="table"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="logs"
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
        <div class="text-sm text-muted">{{ logs.length }} entry(s) on this page</div>
        <UPagination
          :default-page="page"
          :items-per-page="limit"
          :total="total"
          @update:page="onChangePage"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
