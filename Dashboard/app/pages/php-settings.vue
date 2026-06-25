<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'

type PhpExtension = {
  name: string
  php_version: string
  admin_enabled: boolean
  user_enabled: boolean
}

type PhpExtensionsResponse = {
  data?: PhpExtension[]
}

type ApiError = {
  data?: {
    error?: string
  }
}

const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')
const toast = useToast()

const selectedPhpVersion = ref<string | undefined>()

const { data, status, refresh } = await useFetch<PhpExtension[] | PhpExtensionsResponse>('/api/v1/php-extensions', { lazy: true })

const extensions = computed<PhpExtension[]>(() => {
  const raw = data.value

  if (!raw) {
    return []
  }

  if (Array.isArray(raw)) {
    return raw
  }

  return Array.isArray(raw.data) ? raw.data : []
})

const availablePhpVersions = computed(() => {
  return Array.from(new Set(extensions.value.map(ext => ext.php_version).filter(Boolean))).sort()
})

const versionItems = computed(() => {
  return availablePhpVersions.value.map(version => ({
    label: `PHP ${version}`,
    value: version
  }))
})

const selectedExtensions = computed(() => {
  if (!selectedPhpVersion.value) {
    return extensions.value
  }

  return extensions.value.filter(ext => ext.php_version === selectedPhpVersion.value)
})

const enabledCount = computed(() => selectedExtensions.value.filter(ext => ext.user_enabled).length)

watch(availablePhpVersions, (versions) => {
  if (!versions.length) {
    selectedPhpVersion.value = undefined
    return
  }

  if (!selectedPhpVersion.value || !versions.includes(selectedPhpVersion.value)) {
    selectedPhpVersion.value = versions[0]
  }
}, { immediate: true })

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Failed to update extension.'
}

async function toggleExtension(extension: PhpExtension) {
  try {
    await $fetch('/api/v1/php-extensions', {
      method: 'PUT',
      body: {
        name: extension.name,
        php_version: extension.php_version,
        enabled: !extension.user_enabled
      }
    })

    toast.add({
      title: extension.user_enabled ? 'Extension disabled' : 'Extension enabled',
      description: `${extension.name} on PHP ${extension.php_version}`,
      color: 'success'
    })
    refresh()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  }
}

const columns: TableColumn<PhpExtension>[] = [
  {
    accessorKey: 'name',
    header: 'Extension',
    cell: ({ row }) => h('span', { class: 'font-medium text-highlighted' }, row.original.name)
  },
  {
    accessorKey: 'php_version',
    header: 'PHP',
    cell: ({ row }) => h('span', { class: 'text-xs font-mono text-dimmed' }, row.original.php_version)
  },
  {
    accessorKey: 'admin_enabled',
    header: 'Admin Allowed',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: row.original.admin_enabled ? 'success' : 'error'
      },
      () => row.original.admin_enabled ? 'Yes' : 'No'
    )
  },
  {
    accessorKey: 'user_enabled',
    header: 'Your Setting',
    cell: ({ row }) => h(
      UBadge,
      {
        variant: 'subtle',
        color: row.original.user_enabled ? 'success' : 'neutral'
      },
      () => row.original.user_enabled ? 'On' : 'Off'
    )
  },
  {
    id: 'actions',
    cell: ({ row }) => h(
      'div',
      { class: 'text-right' },
      h(UButton, {
        label: row.original.user_enabled ? 'Disable' : 'Enable',
        icon: row.original.user_enabled ? 'i-lucide-toggle-left' : 'i-lucide-toggle-right',
        size: 'xs',
        color: row.original.user_enabled ? 'neutral' : 'primary',
        variant: row.original.user_enabled ? 'outline' : 'solid',
        disabled: !row.original.admin_enabled,
        onClick: () => toggleExtension(row.original)
      })
    )
  }
]
</script>

<template>
  <UDashboardPanel id="php-settings">
    <template #header>
      <UDashboardNavbar title="PHP Settings">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <USelect
            v-model="selectedPhpVersion"
            :items="versionItems"
            :disabled="!versionItems.length"
            icon="i-lucide-code-xml"
            placeholder="PHP Version"
            class="min-w-36"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col gap-4">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-sm text-dimmed">
              Toggle PHP extensions for the selected PHP version. Only extensions enabled by the admin can be activated.
            </p>
            <p v-if="selectedPhpVersion" class="text-xs text-muted mt-1">
              Showing {{ selectedExtensions.length }} extension(s) for PHP {{ selectedPhpVersion }}.
              {{ enabledCount }} currently enabled.
            </p>
          </div>

          <UBadge v-if="selectedPhpVersion" color="neutral" variant="subtle">
            PHP {{ selectedPhpVersion }}
          </UBadge>
        </div>

        <UTable
          :data="selectedExtensions"
          :columns="columns"
          :loading="status === 'pending'"
          :ui="{
            base: 'table-fixed border-separate border-spacing-0',
            thead: '[&>tr]:bg-elevated/50',
            td: 'border-b border-default'
          }"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
