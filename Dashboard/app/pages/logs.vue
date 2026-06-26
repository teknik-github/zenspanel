<script setup lang="ts">
type Domain = {
  id: number
  domain: string
}

type DomainsResponse = {
  data?: Domain[]
}

type LogsResponse = {
  lines?: string[]
}

type ApiError = {
  data?: {
    error?: string
  }
}

type LogType = 'nginx' | 'fpm'

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Request failed.'
}

const toast = useToast()
const domains = ref<Domain[]>([])
const selectedDomainId = ref<number>()
const logType = ref<LogType>('nginx')
const lines = ref(100)
const logLines = ref<string[]>([])
const domainsLoading = ref(false)
const loading = ref(false)

const selectedDomain = computed(() => {
  return domains.value.find(domain => domain.id === selectedDomainId.value) || null
})

const domainOptions = computed(() => {
  return domains.value.map(domain => ({
    label: domain.domain,
    value: domain.id
  }))
})

const logTypeOptions = [
  { label: 'Nginx', value: 'nginx' },
  { label: 'PHP-FPM', value: 'fpm' }
]

const lineOptions = [50, 100, 200, 500]

async function loadDomains() {
  domainsLoading.value = true

  try {
    const response = await $fetch<Domain[] | DomainsResponse>('/api/v1/domains')
    domains.value = Array.isArray(response) ? response : response.data || []

    const firstDomain = domains.value[0]
    if (!selectedDomainId.value && firstDomain) {
      selectedDomainId.value = firstDomain.id
      await fetchLogs()
    }
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    domainsLoading.value = false
  }
}

async function fetchLogs() {
  if (!selectedDomainId.value) {
    logLines.value = []
    return
  }

  loading.value = true
  logLines.value = []

  try {
    const response = await $fetch<LogsResponse>(`/api/v1/domains/${selectedDomainId.value}/logs`, {
      query: {
        type: logType.value,
        lines: lines.value
      }
    })
    logLines.value = response.lines || []
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    loading.value = false
  }
}

function logLineColor(line: string) {
  if (line.includes('[error]') || line.includes('ERROR')) {
    return 'text-red-400'
  }

  if (line.includes('[warn]') || line.includes('WARNING')) {
    return 'text-yellow-400'
  }

  return 'text-gray-300'
}

onMounted(loadDomains)

useIntervalFn(() => {
  if (selectedDomainId.value) {
    fetchLogs()
  }
}, 10000)
</script>

<template>
  <UDashboardPanel id="logs">
    <template #header>
      <UDashboardNavbar title="Error Logs">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <USelect
          v-model="selectedDomainId"
          :items="domainOptions"
          value-key="value"
          label-key="label"
          placeholder="Select domain..."
          class="max-w-xs"
          :loading="domainsLoading"
          @update:model-value="fetchLogs"
        />
        <USelect
          v-model="logType"
          :items="logTypeOptions"
          value-key="value"
          label-key="label"
          @update:model-value="fetchLogs"
        />
        <USelect
          v-model="lines"
          :items="lineOptions"
          @update:model-value="fetchLogs"
        />
      </div>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="font-semibold">
              {{ selectedDomain?.domain || 'No domain selected' }} - {{ logType === 'nginx' ? 'Nginx' : 'PHP-FPM' }}
            </h3>
            <UButton
              icon="i-lucide-refresh-cw"
              size="xs"
              color="neutral"
              variant="ghost"
              :loading="loading"
              :disabled="!selectedDomainId"
              @click="fetchLogs"
            />
          </div>
        </template>

        <div v-if="!selectedDomainId" class="py-8 text-center text-sm text-dimmed">
          Select a domain to view logs
        </div>
        <div v-else-if="loading && !logLines.length" class="py-4 text-sm text-dimmed">
          Loading...
        </div>
        <div v-else-if="!logLines.length" class="py-4 text-sm text-dimmed">
          No log entries
        </div>
        <div v-else class="max-h-96 space-y-0.5 overflow-y-auto rounded-lg bg-black/40 p-3 font-mono text-xs">
          <div
            v-for="(line, index) in logLines"
            :key="index"
            class="whitespace-pre-wrap break-all"
            :class="logLineColor(line)"
          >
            {{ line }}
          </div>
        </div>
      </UCard>
    </template>
  </UDashboardPanel>
</template>
