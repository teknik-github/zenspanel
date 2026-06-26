<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { Row } from '@tanstack/table-core'
import { getPaginationRowModel } from '@tanstack/table-core'

definePageMeta({ alias: '/admin/firewall' })

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const toast = useToast()
const table = useTemplateRef('table')

const activeTab = ref('blocked')

// ── Blocked IPs ──
const { data: blockedData, status: blockedStatus, refresh: refreshBlocked } = await useFetch('/api/v1/admin/firewall/blocked', { lazy: true })
const blockedIps = computed(() => {
  const raw = blockedData.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Fail2Ban Jails ──
const { data: jailsData, status: jailsStatus, refresh: refreshJails } = await useFetch('/api/v1/admin/firewall/fail2ban/jails', { lazy: true })
const jails = computed(() => {
  const raw = jailsData.value as any
  if (!raw) return []
  return Array.isArray(raw?.data) ? raw.data : (Array.isArray(raw) ? raw : [])
})

// ── Block IP ──
const blockOpen = ref(false)
const blockLoading = ref(false)
const blockState = reactive({ ip: '', reason: '' })

async function handleBlock() {
  blockLoading.value = true
  try {
    await $fetch('/api/v1/admin/firewall/block', { method: 'POST', body: blockState })
    toast.add({ title: 'IP blocked', description: `${blockState.ip} added to firewall.`, color: 'success' })
    blockOpen.value = false
    blockState.ip = ''
    blockState.reason = ''
    refreshBlocked()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Block failed', color: 'error' })
  } finally { blockLoading.value = false }
}

// ── Unblock IP ──
const unblockTarget = ref<string | null>(null)
const unblockOpen = ref(false)
const unblockLoading = ref(false)

function showUnblockModal(ip: string) {
  unblockTarget.value = ip
  unblockOpen.value = true
}
async function handleUnblock() {
  if (!unblockTarget.value) return
  unblockLoading.value = true
  try {
    await $fetch('/api/v1/admin/firewall/unblock', { method: 'POST', body: { ip: unblockTarget.value } })
    toast.add({ title: 'Unblocked', description: `${unblockTarget.value} removed from firewall.`, color: 'success' })
    unblockOpen.value = false
    refreshBlocked()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Unblock failed', color: 'error' })
  } finally { unblockLoading.value = false }
}

// ── Toggle Jail ──
async function toggleJail(name: string, enabled: boolean) {
  try {
    await $fetch(`/api/v1/admin/firewall/fail2ban/jails/${name}`, { method: 'PUT', body: { enabled } })
    toast.add({ title: enabled ? 'Jail enabled' : 'Jail disabled', color: 'success' })
    refreshJails()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error || 'Toggle failed', color: 'error' })
  }
}

// ── Blocked IP row actions ──
function blockedRowItems(row: Row<any>) {
  const item = row.original
  return [
    { type: 'label', label: `IP: ${item.ip}` },
    {
      label: 'Unblock', icon: 'i-lucide-x-circle', color: 'error',
      onSelect: () => showUnblockModal(item.ip)
    }
  ]
}

// ── Blocked IPs columns ──
const blockedColumns: TableColumn<any>[] = [
  {
    accessorKey: 'ip',
    header: 'IP',
    cell: ({ row }: any) => h('code', { class: 'text-sm font-mono text-highlighted' }, row.original.ip)
  },
  { accessorKey: 'reason', header: 'Reason' },
  {
    accessorKey: 'source',
    header: 'Source',
    cell: ({ row }: any) => {
      const color = row.original.source === 'panel' ? 'info' : 'warning'
      return h(UBadge, { variant: 'subtle', color, class: 'capitalize' }, () => row.original.source)
    }
  },
  {
    accessorKey: 'blocked_at',
    header: 'Blocked At',
    cell: ({ row }: any) => {
      const d = row.original.blocked_at
      return d ? new Date(d).toLocaleString() : '—'
    }
  },
  {
    id: 'actions',
    cell: ({ row }: any) => {
      return h('div', { class: 'text-right' },
        h(UDropdownMenu, { content: { align: 'end' }, items: blockedRowItems(row) },
          () => h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' })
        )
      )
    }
  }
]

// ── Fail2Ban jails columns ──
const jailsColumns: TableColumn<any>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }: any) => h('span', { class: 'font-medium text-highlighted' }, row.original.name)
  },
  {
    accessorKey: 'enabled',
    header: 'Enabled',
    cell: ({ row }: any) => {
      return h('div', { class: 'flex items-center' },
        h(resolveComponent('USwitch'), {
          modelValue: row.original.enabled,
          'onUpdate:modelValue': (v: boolean) => toggleJail(row.original.name, v)
        })
      )
    }
  },
  { accessorKey: 'ban_count', header: 'Ban Count' },
  { accessorKey: 'currently_banned', header: 'Currently Banned' }
]

const pagination = ref({ pageIndex: 0, pageSize: 20 })
</script>

<template>
  <UDashboardPanel id="firewall">
    <template #header>
      <UDashboardNavbar title="Firewall">
        <template #leading><UDashboardSidebarCollapse /></template>
        <template #right>
          <UButton label="Block IP" icon="i-lucide-shield-off" @click="blockOpen = true" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-6">
        <div class="text-sm text-muted">
          Manage firewall rules and Fail2Ban intrusion prevention.
        </div>

        <UTabs v-model="activeTab" :items="[
          { label: 'Blocked IPs', value: 'blocked' },
          { label: 'Fail2Ban Jails', value: 'jails' }
        ]" />

        <!-- Blocked IPs table -->
        <UTable
          v-if="activeTab === 'blocked'"
          ref="table"
          v-model:pagination="pagination"
          :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
          class="shrink-0"
          :data="blockedIps"
          :columns="blockedColumns"
          :loading="blockedStatus === 'pending'"
          :ui="{
            base: 'table-fixed border-separate border-spacing-0',
            thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
            tbody: '[&>tr]:last:[&>td]:border-b-0',
            th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
            td: 'border-b border-default',
            separator: 'h-0'
          }"
        />

        <!-- Fail2Ban jails table -->
        <UTable
          v-if="activeTab === 'jails'"
          v-model:pagination="pagination"
          :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
          class="shrink-0"
          :data="jails"
          :columns="jailsColumns"
          :loading="jailsStatus === 'pending'"
          :ui="{
            base: 'table-fixed border-separate border-spacing-0',
            thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
            tbody: '[&>tr]:last:[&>td]:border-b-0',
            th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
            td: 'border-b border-default',
            separator: 'h-0'
          }"
        />
      </div>

      <!-- Block IP Modal -->
      <UModal v-model:open="blockOpen" title="Block IP">
        <template #body>
          <div class="space-y-4">
            <UFormField label="IP Address or CIDR" required>
              <UInput v-model="blockState.ip" :disabled="blockLoading" placeholder="1.2.3.4" />
            </UFormField>
            <UFormField label="Reason">
              <UInput v-model="blockState.reason" :disabled="blockLoading" placeholder="Manual block" />
            </UFormField>
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="blockOpen = false" :disabled="blockLoading" />
              <UButton label="Block" color="error" :loading="blockLoading" @click="handleBlock" />
            </div>
          </div>
        </template>
      </UModal>

      <!-- Unblock confirmation -->
      <UModal v-model:open="unblockOpen" :title="`Unblock ${unblockTarget || ''}`"
        description="This IP will be removed from the firewall blocklist.">
        <template #body>
          <div class="flex justify-end gap-2">
            <UButton label="Cancel" color="neutral" variant="subtle" @click="unblockOpen = false" :disabled="unblockLoading" />
            <UButton label="Unblock" color="primary" :loading="unblockLoading" @click="handleUnblock" />
          </div>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
