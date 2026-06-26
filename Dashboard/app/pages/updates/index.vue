<script setup lang="ts">
definePageMeta({ alias: '/admin/updates' })

const toast = useToast()

const { data: version } = await useFetch('/api/v1/system/version', { lazy: true })
const ver = computed(() => version.value as any)

const checking = ref(false), checkResult = ref<any>(null)
async function checkUpdates() { checking.value = true; try { checkResult.value = await $fetch('/api/v1/system/update/check') } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { checking.value = false } }

const updating = ref(false), updatePhase = ref(''), updateLog = ref<string[]>([]), updateDone = ref(false), updateError = ref('')
async function startUpdate() { updating.value = true; updatePhase.value = ''; updateLog.value = []; updateDone.value = false; updateError.value = ''; try { await $fetch('/api/v1/system/update/run', { method: 'POST' }); pollUpdate() } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }); updating.value = false } }

async function pollUpdate() {
  try {
    const s: any = await $fetch('/api/v1/system/update/status')
    updatePhase.value = s.phase; updateLog.value = s.log || []; updateDone.value = s.done; updateError.value = s.error || ''
    if (!s.done && !s.error) { setTimeout(pollUpdate, 2000) } else { updating.value = false; if (s.done) toast.add({ title: 'Update complete', color: 'success' }); if (s.error) toast.add({ title: 'Update failed', description: s.error, color: 'error' }) }
  } catch { updating.value = false }
}

const maintLoading = ref(false), maintEnabled = ref(false)
async function toggleMaint() { maintLoading.value = true; try { await $fetch('/api/v1/system/maintenance', { method: 'POST', body: { enabled: maintEnabled.value } }); toast.add({ title: maintEnabled.value ? 'Maintenance mode on' : 'Maintenance mode off', color: 'success' }) } catch (e: any) { toast.add({ title: 'Error', description: e.data?.error, color: 'error' }) } finally { maintLoading.value = false } }
</script>

<template>
<UDashboardPanel id="updates"><template #header><UDashboardNavbar title="System Updates"><template #leading><UDashboardSidebarCollapse /></template></UDashboardNavbar></template>
<template #body>
<div class="max-w-3xl space-y-6">
  <UCard>
    <template #header><h3 class="font-semibold">Current Version</h3></template>
    <div class="grid grid-cols-3 gap-4">
      <div><p class="text-xs text-dimmed">Version</p><p class="font-semibold">{{ ver?.version || '—' }}</p></div>
      <div><p class="text-xs text-dimmed">Branch</p><p class="font-mono text-sm">{{ ver?.branch || '—' }}</p></div>
      <div><p class="text-xs text-dimmed">Commit</p><p class="font-mono text-sm">{{ ver?.current_sha || '—' }}</p></div>
    </div>
    <UButton label="Check for Updates" icon="i-lucide-refresh-cw" :loading="checking" @click="checkUpdates" class="mt-4" />
  </UCard>

  <UCard v-if="checkResult">
    <template #header><h3 class="font-semibold">Update Available</h3></template>
    <div class="space-y-3">
      <div class="grid grid-cols-2 gap-4">
        <div><p class="text-xs text-dimmed">Current</p><p class="font-mono text-sm">{{ checkResult.current_sha }}</p></div>
        <div><p class="text-xs text-dimmed">Latest</p><p class="font-mono text-sm">{{ checkResult.latest_sha }}</p></div>
        <div><p class="text-xs text-dimmed">Behind By</p><p class="font-semibold text-warning">{{ checkResult.behind_by }} commit(s)</p></div>
        <div><p class="text-xs text-dimmed">Release</p><p>{{ checkResult.release_tag || '—' }}</p></div>
      </div>
      <div v-if="checkResult.changelog" class="p-3 bg-elevated rounded-lg text-sm font-mono whitespace-pre-wrap">{{ checkResult.changelog }}</div>
      <UButton label="Run Update" icon="i-lucide-play" color="primary" :loading="updating" @click="startUpdate" />
    </div>
  </UCard>

  <UCard v-if="updating || updateDone || updateError">
    <template #header><h3 class="font-semibold">Update Progress</h3></template>
    <div class="space-y-3">
      <UBadge :color="updateDone?'success':updateError?'error':'info'" variant="subtle">{{ updatePhase }}</UBadge>
      <div v-if="updateLog.length" class="p-3 bg-elevated rounded-lg text-xs font-mono max-h-60 overflow-y-auto">
        <div v-for="(line,i) in updateLog" :key="i">{{ line }}</div>
      </div>
      <div v-if="updateError" class="text-sm text-error">{{ updateError }}</div>
    </div>
  </UCard>

  <UCard>
    <template #header><h3 class="font-semibold">Maintenance Mode</h3></template>
    <div class="flex items-center gap-4">
      <USwitch v-model="maintEnabled" :disabled="maintLoading" />
      <span class="text-sm">{{ maintEnabled ? 'Enabled' : 'Disabled' }}</span>
      <UButton label="Apply" size="xs" :loading="maintLoading" @click="toggleMaint" />
    </div>
    <p class="text-xs text-dimmed mt-2">When enabled, visitors see a maintenance page instead of the panel.</p>
  </UCard>
</div>
</template></UDashboardPanel>
</template>
