<script setup lang="ts">
definePageMeta({ alias: '/admin/settings' })

const auth = useAuth()
const toast = useToast()

const user = computed(() => auth.user.value)

// Profile display — backend doesn't have update-profile endpoint yet
const profile = computed(() => ({
  username: user.value?.username || '',
  email: user.value?.email || '',
  role: user.value?.role || '',
  php_version: user.value?.php_version || '',
  terminal_enabled: user.value?.terminal_enabled || false,
  backup_enabled: user.value?.backup_enabled || false,
  package_id: user.value?.package_id || null,
  totp_enabled: user.value?.totp_enabled || false,
}))
</script>

<template>
  <UDashboardPanel id="settings">
    <template #header>
      <UDashboardNavbar title="Settings">
        <template #leading><UDashboardSidebarCollapse /></template>
      </UDashboardNavbar>
    </template>
    <template #body>
      <div class="max-w-2xl space-y-6">
        <UCard>
          <template #header><h3 class="font-semibold">Account Information</h3></template>
          <div class="space-y-3">
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Username</span>
              <span class="font-medium">{{ profile.username }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Email</span>
              <span class="font-medium">{{ profile.email || '—' }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Role</span>
              <UBadge variant="subtle" :color="profile.role === 'admin' ? 'primary' : 'neutral'">{{ profile.role }}</UBadge>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">PHP Version</span>
              <span class="font-medium">{{ profile.php_version }}</span>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Package ID</span>
              <span class="font-medium">{{ profile.package_id ?? '—' }}</span>
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header><h3 class="font-semibold">Features</h3></template>
          <div class="space-y-3">
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Terminal Access</span>
              <UBadge variant="subtle" :color="profile.terminal_enabled ? 'success' : 'error'">
                {{ profile.terminal_enabled ? 'Enabled' : 'Disabled' }}
              </UBadge>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Backup Access</span>
              <UBadge variant="subtle" :color="profile.backup_enabled ? 'success' : 'error'">
                {{ profile.backup_enabled ? 'Enabled' : 'Disabled' }}
              </UBadge>
            </div>
            <div class="flex justify-between py-2 border-b border-default">
              <span class="text-dimmed">Two-Factor Auth</span>
              <UBadge variant="subtle" :color="profile.totp_enabled ? 'success' : 'error'">
                {{ profile.totp_enabled ? 'Enabled' : 'Disabled' }}
              </UBadge>
            </div>
          </div>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
