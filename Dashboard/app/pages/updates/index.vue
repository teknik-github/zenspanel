<script setup lang="ts">
type VersionInfo = {
  version?: string
  branch?: string
  current_sha?: string
}

type UpdateCheck = {
  current_sha: string
  latest_sha: string
  behind_by: number
  changelog?: string
  current_branch?: string
  download_url?: string
  checksum?: string
  release_tag?: string
}

type UpdateStatus = {
  phase: string
  log?: string[]
  done: boolean
  error?: string
}

type ApiError = {
  data?: {
    error?: string
  }
}

definePageMeta({ alias: '/admin/updates' })

const toast = useToast()

const { data: version } = await useFetch<VersionInfo>('/api/v1/system/version', { lazy: true })

const checking = ref(false)
const checkResult = ref<UpdateCheck | null>(null)
const updating = ref(false)
const updatePhase = ref('')
const updateLog = ref<string[]>([])
const updateDone = ref(false)
const updateError = ref('')
const maintLoading = ref(false)
const maintEnabled = ref(false)

const updateMode = computed(() => {
  return checkResult.value?.download_url ? 'Release package' : 'Build from source'
})

function errorMessage(error: unknown, fallback: string) {
  const apiError = error as ApiError
  return apiError.data?.error || fallback
}

async function checkUpdates() {
  checking.value = true
  try {
    checkResult.value = await $fetch<UpdateCheck>('/api/v1/system/update/check')
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: errorMessage(error, 'Failed to check updates'),
      color: 'error'
    })
  } finally {
    checking.value = false
  }
}

async function startUpdate() {
  updating.value = true
  updatePhase.value = ''
  updateLog.value = []
  updateDone.value = false
  updateError.value = ''

  try {
    await $fetch('/api/v1/system/update/run', {
      method: 'POST',
      body: {
        download_url: checkResult.value?.download_url || '',
        checksum: checkResult.value?.checksum || ''
      }
    })
    pollUpdate()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: errorMessage(error, 'Failed to start update'),
      color: 'error'
    })
    updating.value = false
  }
}

async function pollUpdate() {
  try {
    const status = await $fetch<UpdateStatus>('/api/v1/system/update/status')
    updatePhase.value = status.phase
    updateLog.value = status.log || []
    updateDone.value = status.done
    updateError.value = status.error || ''

    if (!status.done && !status.error) {
      setTimeout(pollUpdate, 2000)
      return
    }

    updating.value = false
    if (status.done && !status.error) {
      toast.add({ title: 'Update complete', color: 'success' })
    }
    if (status.error) {
      toast.add({ title: 'Update failed', description: status.error, color: 'error' })
    }
  } catch {
    updating.value = false
  }
}

async function toggleMaint() {
  maintLoading.value = true
  try {
    await $fetch('/api/v1/system/maintenance', {
      method: 'POST',
      body: { enabled: maintEnabled.value }
    })
    toast.add({
      title: maintEnabled.value ? 'Maintenance mode on' : 'Maintenance mode off',
      color: 'success'
    })
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: errorMessage(error, 'Failed to update maintenance mode'),
      color: 'error'
    })
  } finally {
    maintLoading.value = false
  }
}
</script>

<template>
  <UDashboardPanel id="updates">
    <template #header>
      <UDashboardNavbar title="System Updates">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="max-w-3xl space-y-6">
        <UCard>
          <template #header>
            <h3 class="font-semibold">
              Current Version
            </h3>
          </template>

          <div class="grid gap-4 sm:grid-cols-3">
            <div>
              <p class="text-xs text-dimmed">
                Version
              </p>
              <p class="font-semibold">
                {{ version?.version || '—' }}
              </p>
            </div>
            <div>
              <p class="text-xs text-dimmed">
                Branch
              </p>
              <p class="font-mono text-sm">
                {{ version?.branch || '—' }}
              </p>
            </div>
            <div>
              <p class="text-xs text-dimmed">
                Commit
              </p>
              <p class="font-mono text-sm">
                {{ version?.current_sha || '—' }}
              </p>
            </div>
          </div>

          <UButton
            class="mt-4"
            label="Check for Updates"
            icon="i-lucide-refresh-cw"
            :loading="checking"
            @click="checkUpdates"
          />
        </UCard>

        <UCard v-if="checkResult">
          <template #header>
            <h3 class="font-semibold">
              Update Available
            </h3>
          </template>

          <div class="space-y-3">
            <div class="grid gap-4 sm:grid-cols-2">
              <div>
                <p class="text-xs text-dimmed">
                  Current
                </p>
                <p class="font-mono text-sm">
                  {{ checkResult.current_sha }}
                </p>
              </div>
              <div>
                <p class="text-xs text-dimmed">
                  Latest
                </p>
                <p class="font-mono text-sm">
                  {{ checkResult.latest_sha }}
                </p>
              </div>
              <div>
                <p class="text-xs text-dimmed">
                  Behind By
                </p>
                <p class="font-semibold text-warning">
                  {{ checkResult.behind_by }} commit(s)
                </p>
              </div>
              <div>
                <p class="text-xs text-dimmed">
                  Release
                </p>
                <p>
                  {{ checkResult.release_tag || '—' }}
                </p>
              </div>
              <div>
                <p class="text-xs text-dimmed">
                  Update Mode
                </p>
                <p class="font-semibold">
                  {{ updateMode }}
                </p>
              </div>
              <div>
                <p class="text-xs text-dimmed">
                  Checksum
                </p>
                <p class="font-semibold">
                  {{ checkResult.checksum ? 'SHA-256 verified' : 'Unavailable' }}
                </p>
              </div>
            </div>

            <div
              v-if="checkResult.changelog"
              class="rounded-lg bg-elevated p-3 font-mono text-sm whitespace-pre-wrap"
            >
              {{ checkResult.changelog }}
            </div>

            <UButton
              label="Run Update"
              icon="i-lucide-play"
              color="primary"
              :loading="updating"
              @click="startUpdate"
            />
          </div>
        </UCard>

        <UCard v-if="updating || updateDone || updateError">
          <template #header>
            <h3 class="font-semibold">
              Update Progress
            </h3>
          </template>

          <div class="space-y-3">
            <UBadge
              :color="updateDone ? 'success' : updateError ? 'error' : 'info'"
              variant="subtle"
            >
              {{ updatePhase }}
            </UBadge>

            <div
              v-if="updateLog.length"
              class="max-h-60 overflow-y-auto rounded-lg bg-elevated p-3 font-mono text-xs"
            >
              <div
                v-for="(line, index) in updateLog"
                :key="index"
              >
                {{ line }}
              </div>
            </div>

            <div v-if="updateError" class="text-sm text-error">
              {{ updateError }}
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <h3 class="font-semibold">
              Maintenance Mode
            </h3>
          </template>

          <div class="flex items-center gap-4">
            <USwitch v-model="maintEnabled" :disabled="maintLoading" />
            <span class="text-sm">{{ maintEnabled ? 'Enabled' : 'Disabled' }}</span>
            <UButton
              label="Apply"
              size="xs"
              :loading="maintLoading"
              @click="toggleMaint"
            />
          </div>
          <p class="mt-2 text-xs text-dimmed">
            When enabled, visitors see a maintenance page instead of the panel.
          </p>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
