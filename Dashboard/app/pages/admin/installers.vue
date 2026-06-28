<script setup lang="ts">
const toast = useToast()
const { data, refresh } = await useFetch('/api/v1/admin/installer/apps')
const apps = computed(() => (data.value as any)?.data ?? [])

async function toggle(app: any) {
  try {
    await $fetch(`/api/v1/admin/installer/apps/${app.id}/enabled`, {
      method: 'PUT',
      body: { enabled: !app.enabled }
    })
    toast.add({ title: `${app.name} ${!app.enabled ? 'enabled' : 'disabled'}`, color: 'success' })
    refresh()
  } catch (e: any) {
    toast.add({ title: 'Error', description: e.data?.error, color: 'error' })
  }
}
</script>

<template>
<UDashboardPanel id="admin-installers">
  <template #header>
    <UDashboardNavbar title="Web Installer">
      <template #leading><UDashboardSidebarCollapse /></template>
    </UDashboardNavbar>
  </template>
  <template #body>
    <p class="text-sm text-dimmed mb-6">Enable or disable web application installers globally. Disabled installers are hidden from all users.</p>
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      <UCard v-for="app in apps" :key="app.id" class="flex flex-col gap-3">
        <div class="flex items-start justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-semibold text-highlighted">{{ app.name }}</span>
              <UBadge variant="subtle" color="neutral" size="xs">v{{ app.version }}</UBadge>
              <UBadge v-if="app.requires_db" variant="subtle" color="info" size="xs">MySQL</UBadge>
            </div>
            <p class="text-sm text-dimmed leading-snug">{{ app.description }}</p>
          </div>
          <UToggle :model-value="app.enabled" @update:model-value="toggle(app)" />
        </div>
      </UCard>
    </div>
  </template>
</UDashboardPanel>
</template>
