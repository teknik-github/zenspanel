<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const auth = useAuth()
const open = ref(false)

const isAdmin = computed(() => auth.user.value?.role === 'admin')

const adminLinks = [[{
  label: 'Dashboard',
  icon: 'i-lucide-activity',
  to: '/dashboard',
  onSelect: () => { open.value = false }
}, {
  label: 'Account Management',
  icon: 'i-lucide-id-card',
  type: 'trigger',
  defaultOpen: true,
  children: [{
    label: 'Users',
    icon: 'i-lucide-users',
    to: '/users',
    onSelect: () => { open.value = false }
  }, {
    label: 'Packages',
    icon: 'i-lucide-package',
    to: '/packages',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Hosting Services',
  icon: 'i-lucide-globe',
  type: 'trigger',
  children: [{
    label: 'Domains',
    icon: 'i-lucide-globe',
    to: '/domains',
    onSelect: () => { open.value = false }
  }, {
    label: 'Databases',
    icon: 'i-lucide-database',
    to: '/databases',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'PHP Settings',
  icon: 'i-lucide-code',
  type: 'trigger',
  children: [{
    label: 'PHP Versions',
    icon: 'i-lucide-tags',
    to: '/php-versions',
    onSelect: () => { open.value = false }
  }, {
    label: 'PHP Extensions',
    icon: 'i-lucide-puzzle',
    to: '/php-extensions',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Security',
  icon: 'i-lucide-shield',
  type: 'trigger',
  children: [{
    label: 'SSL Manager',
    icon: 'i-lucide-lock',
    to: '/ssl',
    onSelect: () => { open.value = false }
  }, {
    label: 'Firewall',
    icon: 'i-lucide-flame',
    to: '/firewall',
    onSelect: () => { open.value = false }
  }, {
    label: 'IP Allowlist',
    icon: 'i-lucide-shield-check',
    to: '/ip-allowlist',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Backups',
  icon: 'i-lucide-hard-drive',
  type: 'trigger',
  children: [{
    label: 'Backups',
    icon: 'i-lucide-archive',
    to: '/backups',
    onSelect: () => { open.value = false }
  }, {
    label: 'Backup Targets',
    icon: 'i-lucide-cloud',
    to: '/backup-targets',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Advanced Tools',
  icon: 'i-lucide-wrench',
  type: 'trigger',
  children: [{
    label: 'Audit Logs',
    icon: 'i-lucide-scroll',
    to: '/audit-logs',
    onSelect: () => { open.value = false }
  }, {
    label: 'Terminal',
    icon: 'i-lucide-terminal',
    to: '/terminal',
    onSelect: () => { open.value = false }
  }, {
    label: 'API Keys',
    icon: 'i-lucide-key',
    to: '/api-keys',
    onSelect: () => { open.value = false }
  }, {
    label: 'Updates',
    icon: 'i-lucide-refresh-cw',
    to: '/updates',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Settings',
  icon: 'i-lucide-settings',
  to: '/settings',
  onSelect: () => { open.value = false }
}]] satisfies NavigationMenuItem[][]

const userLinks = [[{
  label: 'Dashboard',
  icon: 'i-lucide-house',
  to: '/dashboard',
  onSelect: () => { open.value = false }
}, {
  label: 'Website Management',
  icon: 'i-lucide-globe',
  type: 'trigger',
  defaultOpen: true,
  children: [{
    label: 'Domains',
    icon: 'i-lucide-globe',
    to: '/domains',
    onSelect: () => { open.value = false }
  }, {
    label: 'SSL Manager',
    icon: 'i-lucide-lock',
    to: '/ssl',
    onSelect: () => { open.value = false }
  }, {
    label: 'PHP Settings',
    icon: 'i-lucide-code',
    to: '/php-settings',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Files & FTP',
  icon: 'i-lucide-folder',
  type: 'trigger',
  children: [{
    label: 'File Manager',
    icon: 'i-lucide-folder-open',
    to: '/file-manager',
    onSelect: () => { open.value = false }
  }, {
    label: 'FTP Accounts',
    icon: 'i-lucide-upload',
    to: '/ftp',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Databases',
  icon: 'i-lucide-database',
  to: '/databases',
  onSelect: () => { open.value = false }
}, {
  label: 'Security & Backups',
  icon: 'i-lucide-shield',
  type: 'trigger',
  children: [{
    label: 'Antivirus',
    icon: 'i-lucide-shield-alert',
    to: '/antivirus',
    onSelect: () => { open.value = false }
  }, {
    label: 'Two-Factor Auth',
    icon: 'i-lucide-fingerprint',
    to: '/two-factor',
    onSelect: () => { open.value = false }
  }, {
    label: 'Backups',
    icon: 'i-lucide-archive',
    to: '/backups',
    onSelect: () => { open.value = false }
  }]
}, {
  label: 'Advanced Tools',
  icon: 'i-lucide-wrench',
  type: 'trigger',
  children: [{
    label: 'Redirects',
    icon: 'i-lucide-arrow-left-right',
    to: '/redirects',
    onSelect: () => { open.value = false }
  }, {
    label: 'Terminal',
    icon: 'i-lucide-terminal',
    to: '/terminal',
    onSelect: () => { open.value = false }
  }, {
    label: 'Cron Jobs',
    icon: 'i-lucide-clock',
    to: '/cron-jobs',
    onSelect: () => { open.value = false }
  }, {
    label: 'Error Logs',
    icon: 'i-lucide-file-warning',
    to: '/logs',
    onSelect: () => { open.value = false }
  }]
}]] satisfies NavigationMenuItem[][]
</script>

<template>
  <UDashboardGroup unit="rem">
    <UDashboardSidebar
      id="default"
      v-model:open="open"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <TeamsMenu :collapsed="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          v-if="isAdmin"
          :collapsed="collapsed"
          :items="adminLinks[0]"
          orientation="vertical"
          tooltip
          popover
        />
        <UNavigationMenu
          v-else
          :collapsed="collapsed"
          :items="userLinks[0]"
          orientation="vertical"
          tooltip
          popover
        />

        <UNavigationMenu
          v-if="isAdmin"
          :collapsed="collapsed"
          :items="adminLinks[1]"
          orientation="vertical"
          tooltip
          popover
          class="mt-auto"
        />
      </template>

      <template #footer="{ collapsed }">
        <UserMenu :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <slot />
  </UDashboardGroup>
</template>
