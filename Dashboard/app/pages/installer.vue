<script setup lang="ts">
const toast = useToast()

const { data: appsData } = await useFetch('/api/v1/installer/apps')
const apps = computed(() => (appsData.value as any)?.data ?? [])

const { data: domainsData } = await useFetch('/api/v1/domains', { query: { limit: 100 } })
const domainList = computed(() => {
  const r = domainsData.value as any
  return Array.isArray(r?.data) ? r.data : []
})

// Install modal state
const installOpen = ref(false)
const selectedApp = ref<any>(null)
const installing = ref(false)
const form = reactive({ domain_id: null as number | null, db_name: '', db_user: '', db_pass: '', overwrite: false })

// Progress tracking
const jobID = ref('')
const jobStatus = ref<any>(null)
const pollTimer = ref<ReturnType<typeof setInterval> | null>(null)

function openInstall(app: any) {
  selectedApp.value = app
  form.domain_id = null
  form.db_name = ''
  form.db_user = ''
  form.db_pass = ''
  form.overwrite = false
  jobID.value = ''
  jobStatus.value = null
  installOpen.value = true
}

function stopPolling() {
  if (pollTimer.value) { clearInterval(pollTimer.value); pollTimer.value = null }
}

async function pollStatus() {
  if (!jobID.value) return
  try {
    const res: any = await $fetch(`/api/v1/installer/status/${jobID.value}`)
    jobStatus.value = res
    if (res.done) stopPolling()
  } catch {}
}

async function handleInstall() {
  if (!form.domain_id) { toast.add({ title: 'Select a domain', color: 'warning' }); return }
  installing.value = true
  try {
    const body: any = { app_id: selectedApp.value.id, domain_id: form.domain_id, overwrite: form.overwrite }
    if (selectedApp.value.requires_db) { body.db_name = form.db_name; body.db_user = form.db_user; body.db_pass = form.db_pass }
    const res: any = await $fetch('/api/v1/installer/install', { method: 'POST', body })
    jobID.value = res.job_id
    jobStatus.value = { phase: 'starting', log: [], done: false }
    pollTimer.value = setInterval(pollStatus, 2000)
  } catch (e: any) {
    toast.add({ title: 'Install failed', description: e.data?.error, color: 'error' })
  } finally {
    installing.value = false
  }
}

onUnmounted(stopPolling)

const phaseColor = computed(() => {
  if (!jobStatus.value) return 'neutral'
  if (jobStatus.value.phase === 'done') return 'success'
  if (jobStatus.value.phase === 'failed') return 'error'
  return 'info'
})
</script>

<template>
<UDashboardPanel id="installer">
  <template #header>
    <UDashboardNavbar title="Web Installer">
      <template #leading><UDashboardSidebarCollapse /></template>
    </UDashboardNavbar>
  </template>
  <template #body>
    <div v-if="apps.length === 0" class="text-dimmed text-sm py-12 text-center">No installers are currently available. Contact your administrator.</div>
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <UCard v-for="app in apps" :key="app.id" class="flex flex-col gap-4">
        <div class="flex-1">
          <div class="flex items-center gap-2 mb-1">
            <span class="font-semibold text-highlighted text-lg">{{ app.name }}</span>
            <UBadge variant="subtle" color="neutral" size="xs">v{{ app.version }}</UBadge>
            <UBadge v-if="app.requires_db" variant="subtle" color="info" size="xs">MySQL</UBadge>
          </div>
          <p class="text-sm text-dimmed leading-snug">{{ app.description }}</p>
        </div>
        <UButton label="Install" icon="i-lucide-download" size="sm" block @click="openInstall(app)" />
      </UCard>
    </div>

    <UModal v-model:open="installOpen" :title="`Install ${selectedApp?.name}`" :ui="{ width: 'max-w-lg' }">
      <template #body>
        <div class="space-y-4">
          <!-- Form — hidden once job starts -->
          <template v-if="!jobID">
            <UFormField label="Domain" required>
              <USelect v-model="form.domain_id" :items="domainList" option-attribute="domain" value-attribute="id" placeholder="Select a domain..." />
            </UFormField>
            <template v-if="selectedApp?.requires_db">
              <UFormField label="Database Name" required>
                <UInput v-model="form.db_name" placeholder="myapp_db" :disabled="installing" />
              </UFormField>
              <UFormField label="Database User" required>
                <UInput v-model="form.db_user" placeholder="myapp_user" :disabled="installing" />
              </UFormField>
              <UFormField label="Database Password" required>
                <UInput v-model="form.db_pass" type="password" placeholder="StrongPass123" :disabled="installing" />
              </UFormField>
            </template>
            <UCheckbox v-model="form.overwrite" label="Overwrite existing files in document root" :disabled="installing" />
            <div class="flex justify-end gap-2 pt-2">
              <UButton label="Cancel" color="neutral" variant="subtle" @click="installOpen=false" :disabled="installing" />
              <UButton label="Install" icon="i-lucide-download" :loading="installing" @click="handleInstall" />
            </div>
          </template>

          <!-- Progress -->
          <template v-if="jobID && jobStatus">
            <div class="flex items-center gap-2 mb-2">
              <UBadge :color="phaseColor" variant="subtle">{{ jobStatus.phase }}</UBadge>
              <span v-if="!jobStatus.done" class="text-xs text-dimmed">Installing…</span>
            </div>
            <div class="bg-neutral-950 rounded-lg p-3 h-56 overflow-y-auto font-mono text-xs text-green-400 space-y-0.5">
              <div v-for="(line, i) in jobStatus.log" :key="i">{{ line }}</div>
              <div v-if="!jobStatus.done" class="animate-pulse">▌</div>
            </div>
            <div v-if="jobStatus.error" class="text-sm text-error mt-1">{{ jobStatus.error }}</div>
            <div class="flex justify-end mt-2">
              <UButton label="Close" color="neutral" variant="subtle" :disabled="!jobStatus.done" @click="installOpen=false;stopPolling()" />
            </div>
          </template>
        </div>
      </template>
    </UModal>
  </template>
</UDashboardPanel>
</template>
