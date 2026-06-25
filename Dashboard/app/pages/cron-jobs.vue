<script setup lang="ts">
import type { DropdownMenuItem, TableColumn } from '@nuxt/ui'
import type { Row } from '@tanstack/table-core'

type CronJob = {
  id: number
  user_id?: number
  expression: string
  command: string
  enabled: boolean
  created_at?: string
  updated_at?: string
}

type CronJobsResponse = {
  data?: CronJob[]
}

type ApiError = {
  data?: {
    error?: string
  }
}

type SchedulePreset = {
  label: string
  value: string
  description: string
}

const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const toast = useToast()

const schedulePresets: SchedulePreset[] = [
  {
    label: 'Every 5 minutes',
    value: '*/5 * * * *',
    description: 'Runs frequently for lightweight maintenance tasks.'
  },
  {
    label: 'Hourly',
    value: '0 * * * *',
    description: 'Runs once at the start of every hour.'
  },
  {
    label: 'Daily',
    value: '0 0 * * *',
    description: 'Runs every day at midnight.'
  },
  {
    label: 'Weekly',
    value: '0 0 * * 0',
    description: 'Runs every Sunday at midnight.'
  }
]

const createOpen = ref(false)
const createLoading = ref(false)
const selectedPreset = ref(schedulePresets[1]?.value || '0 * * * *')
const toggleLoadingId = ref<number | null>(null)

const deleteOpen = ref(false)
const deleteLoading = ref(false)
const deleteTarget = ref<CronJob | null>(null)

const createState = reactive({
  expression: schedulePresets[1]?.value || '0 * * * *',
  command: '',
  enabled: true
})

const { data, status, refresh } = await useFetch<CronJob[] | CronJobsResponse>('/api/v1/cron-jobs', {
  lazy: true
})

const jobs = computed<CronJob[]>(() => {
  const raw = data.value

  if (!raw) {
    return []
  }

  if (Array.isArray(raw)) {
    return raw
  }

  return Array.isArray(raw.data) ? raw.data : []
})

const activeJobs = computed(() => jobs.value.filter(job => job.enabled).length)
const disabledJobs = computed(() => jobs.value.length - activeJobs.value)

const presetItems = computed(() => {
  return [
    ...schedulePresets,
    {
      label: 'Custom',
      value: 'custom',
      description: 'Use a standard 5-field cron expression.'
    }
  ]
})

watch(selectedPreset, (value) => {
  if (value !== 'custom') {
    createState.expression = value
  }
})

watch(() => createState.expression, (value) => {
  if (!schedulePresets.some(preset => preset.value === value)) {
    selectedPreset.value = 'custom'
  }
})

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

function formatDate(value?: string) {
  if (!value) {
    return '—'
  }

  return value.slice(0, 16).replace('T', ' ')
}

function resetCreateState() {
  selectedPreset.value = schedulePresets[1]?.value || '0 * * * *'
  createState.expression = selectedPreset.value
  createState.command = ''
  createState.enabled = true
}

function showCreate() {
  resetCreateState()
  createOpen.value = true
}

async function handleCreate() {
  const expression = createState.expression.trim()
  const command = createState.command.trim()

  if (!expression || !command) {
    toast.add({
      title: 'Missing cron job details',
      description: 'Schedule and command are required.',
      color: 'warning'
    })
    return
  }

  createLoading.value = true

  try {
    await $fetch('/api/v1/cron-jobs', {
      method: 'POST',
      body: {
        expression,
        command,
        enabled: createState.enabled
      }
    })

    toast.add({
      title: 'Cron job created',
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

async function toggleJob(job: CronJob) {
  toggleLoadingId.value = job.id

  try {
    await $fetch(`/api/v1/cron-jobs/${job.id}`, {
      method: 'PUT',
      body: { enabled: !job.enabled }
    })

    toast.add({
      title: job.enabled ? 'Cron job disabled' : 'Cron job enabled',
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
    toggleLoadingId.value = null
  }
}

function showDelete(job: CronJob) {
  deleteTarget.value = job
  deleteOpen.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) {
    return
  }

  deleteLoading.value = true

  try {
    await $fetch(`/api/v1/cron-jobs/${deleteTarget.value.id}`, { method: 'DELETE' })
    toast.add({ title: 'Cron job deleted', color: 'success' })
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

function getRowItems(row: Row<CronJob>): DropdownMenuItem[] {
  const job = row.original
  const toggleBusy = toggleLoadingId.value === job.id

  return [
    { type: 'label', label: `Cron Job #${job.id}` },
    {
      label: job.enabled ? 'Disable' : 'Enable',
      icon: job.enabled ? 'i-lucide-pause' : 'i-lucide-play',
      disabled: toggleBusy,
      onSelect: () => toggleJob(job)
    },
    { type: 'separator' },
    {
      label: 'Delete',
      icon: 'i-lucide-trash',
      color: 'error',
      onSelect: () => showDelete(job)
    }
  ]
}

const columns: TableColumn<CronJob>[] = [
  {
    accessorKey: 'expression',
    header: 'Schedule',
    cell: ({ row }) => h(
      'code',
      { class: 'inline-flex rounded bg-elevated px-2 py-1 font-mono text-xs text-highlighted' },
      row.original.expression
    )
  },
  {
    accessorKey: 'command',
    header: 'Command',
    cell: ({ row }) => h(
      'code',
      { class: 'block max-w-xl truncate font-mono text-xs text-muted' },
      row.original.command
    )
  },
  {
    accessorKey: 'enabled',
    header: 'Status',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: row.original.enabled ? 'success' : 'neutral'
      },
      () => row.original.enabled ? 'Active' : 'Paused'
    )
  },
  {
    accessorKey: 'created_at',
    header: 'Created',
    cell: ({ row }) => h('span', { class: 'text-sm text-dimmed' }, formatDate(row.original.created_at))
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
          icon: toggleLoadingId.value === row.original.id ? 'i-lucide-loader-circle' : 'i-lucide-ellipsis-vertical',
          color: 'neutral',
          variant: 'ghost',
          class: 'ml-auto',
          loading: toggleLoadingId.value === row.original.id
        })
      )
    )
  }
]
</script>

<template>
  <UDashboardPanel id="cron-jobs">
    <template #header>
      <UDashboardNavbar title="Cron Jobs">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            label="New Cron Job"
            icon="i-lucide-plus"
            @click="showCreate"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4">
        <div class="grid gap-3 md:grid-cols-3">
          <div class="rounded-lg border border-default bg-muted/30 p-4">
            <p class="text-xs font-medium uppercase text-dimmed">
              Total Jobs
            </p>
            <p class="mt-2 text-2xl font-semibold text-highlighted">
              {{ jobs.length }}
            </p>
          </div>

          <div class="rounded-lg border border-default bg-muted/30 p-4">
            <p class="text-xs font-medium uppercase text-dimmed">
              Active
            </p>
            <p class="mt-2 text-2xl font-semibold text-success">
              {{ activeJobs }}
            </p>
          </div>

          <div class="rounded-lg border border-default bg-muted/30 p-4">
            <p class="text-xs font-medium uppercase text-dimmed">
              Paused
            </p>
            <p class="mt-2 text-2xl font-semibold text-muted">
              {{ disabledJobs }}
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3">
          <p class="text-sm text-dimmed">
            Manage scheduled commands with standard 5-field cron expressions.
          </p>

          <UBadge color="neutral" variant="subtle">
            minute hour day month weekday
          </UBadge>
        </div>

        <UTable
          :data="jobs"
          :columns="columns"
          :loading="status === 'pending'"
          :ui="{
            base: 'table-fixed border-separate border-spacing-0',
            thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
            tbody: '[&>tr]:last:[&>td]:border-b-0',
            th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
            td: 'border-b border-default align-middle',
            separator: 'h-0'
          }"
        />

        <div
          v-if="status !== 'pending' && !jobs.length"
          class="flex flex-col items-center justify-center rounded-lg border border-dashed border-default px-6 py-12 text-center"
        >
          <UIcon name="i-lucide-calendar-clock" class="size-8 text-dimmed" />
          <p class="mt-3 font-medium text-highlighted">
            No cron jobs yet
          </p>
          <p class="mt-1 text-sm text-dimmed">
            Create a scheduled command to run maintenance tasks automatically.
          </p>
        </div>

        <UModal v-model:open="createOpen" title="New Cron Job">
          <template #body>
            <div class="space-y-5">
              <URadioGroup
                v-model="selectedPreset"
                :items="presetItems"
                value-key="value"
                label-key="label"
                description-key="description"
                :disabled="createLoading"
              />

              <UFormField label="Cron Expression" required>
                <UInput
                  v-model="createState.expression"
                  icon="i-lucide-calendar-clock"
                  :disabled="createLoading"
                  placeholder="0 * * * *"
                  class="font-mono"
                />
              </UFormField>

              <UFormField label="Command" required>
                <UTextarea
                  v-model="createState.command"
                  :disabled="createLoading"
                  placeholder="php /home/alice/public_html/example.com/artisan schedule:run"
                  :rows="4"
                />
              </UFormField>

              <div class="flex items-center justify-between gap-4 rounded-lg border border-default p-3">
                <div>
                  <p class="text-sm font-medium text-highlighted">
                    Start enabled
                  </p>
                  <p class="text-xs text-dimmed">
                    Commands cannot use shell chaining characters.
                  </p>
                </div>
                <USwitch v-model="createState.enabled" :disabled="createLoading" />
              </div>

              <div class="flex justify-end gap-2 pt-1">
                <UButton
                  label="Cancel"
                  color="neutral"
                  variant="subtle"
                  :disabled="createLoading"
                  @click="createOpen = false"
                />
                <UButton
                  label="Create"
                  icon="i-lucide-plus"
                  :loading="createLoading"
                  @click="handleCreate"
                />
              </div>
            </div>
          </template>
        </UModal>

        <UModal
          v-model:open="deleteOpen"
          :title="`Delete Cron Job #${deleteTarget?.id || ''}`"
          description="This removes the scheduled command from your account."
        >
          <template #body>
            <div class="space-y-4">
              <code
                v-if="deleteTarget"
                class="block rounded-lg bg-elevated p-3 font-mono text-xs text-muted"
              >
                {{ deleteTarget.command }}
              </code>

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
            </div>
          </template>
        </UModal>
      </div>
    </template>
  </UDashboardPanel>
</template>
