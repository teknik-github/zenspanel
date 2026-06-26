<script setup lang="ts">
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

type ExtensionGroup = {
  letter: string
  extensions: PhpExtension[]
}

const COLUMN_COUNT = 4

const UBadge = resolveComponent('UBadge')
const toast = useToast()

const selectedPhpVersion = ref<string | undefined>()
const selectedExtensionKeys = ref<string[]>([])
const applyLoading = ref(false)

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

const selectedCount = computed(() => selectedExtensionKeys.value.length)

const changedExtensions = computed(() => {
  return selectedExtensions.value.filter((extension) => {
    if (!extension.admin_enabled) {
      return false
    }

    return selectedExtensionKeys.value.includes(getExtensionKey(extension)) !== extension.user_enabled
  })
})

const groupedExtensions = computed<ExtensionGroup[]>(() => {
  const groups = new Map<string, PhpExtension[]>()

  for (const extension of selectedExtensions.value) {
    const letter = extension.name.charAt(0).toUpperCase().match(/[A-Z]/) ? extension.name.charAt(0).toUpperCase() : '#'
    const items = groups.get(letter) || []
    items.push(extension)
    groups.set(letter, items)
  }

  return Array.from(groups.entries())
    .sort(([letterA], [letterB]) => letterA.localeCompare(letterB))
    .map(([letter, items]) => ({
      letter,
      extensions: items.sort((a, b) => a.name.localeCompare(b.name))
    }))
})

const groupedExtensionColumns = computed<ExtensionGroup[][]>(() => {
  const columns = Array.from({ length: COLUMN_COUNT }, () => [] as ExtensionGroup[])
  const columnWeights = Array.from({ length: COLUMN_COUNT }, () => 0)

  for (const group of groupedExtensions.value) {
    const targetIndex = columnWeights.indexOf(Math.min(...columnWeights))
    const targetColumn = columns[targetIndex]

    if (!targetColumn) {
      continue
    }

    targetColumn.push(group)
    columnWeights[targetIndex] = (columnWeights[targetIndex] || 0) + group.extensions.length + 1
  }

  return columns
})

watch(availablePhpVersions, (versions) => {
  if (!versions.length) {
    selectedPhpVersion.value = undefined
    return
  }

  if (!selectedPhpVersion.value || !versions.includes(selectedPhpVersion.value)) {
    selectedPhpVersion.value = versions[0]
  }
}, { immediate: true })

watch(selectedExtensions, (items) => {
  selectedExtensionKeys.value = items
    .filter(extension => extension.user_enabled)
    .map(extension => getExtensionKey(extension))
}, { immediate: true })

function getExtensionKey(extension: PhpExtension) {
  return `${extension.php_version}:${extension.name}`
}

function getErrorMessage(error: unknown) {
  const apiError = error as ApiError
  return apiError.data?.error || 'Failed to update extension.'
}

function setExtensionSelected(extension: PhpExtension, enabled: boolean) {
  if (!extension.admin_enabled) {
    return
  }

  const key = getExtensionKey(extension)
  const current = new Set(selectedExtensionKeys.value)

  if (enabled) {
    current.add(key)
  } else {
    current.delete(key)
  }

  selectedExtensionKeys.value = Array.from(current)
}

function selectAllAvailable() {
  selectedExtensionKeys.value = selectedExtensions.value
    .filter(extension => extension.admin_enabled)
    .map(extension => getExtensionKey(extension))
}

function clearAvailable() {
  selectedExtensionKeys.value = []
}

async function applyChanges() {
  if (!changedExtensions.value.length) {
    return
  }

  applyLoading.value = true
  try {
    const changes = [...changedExtensions.value]

    for (const extension of changes) {
      await $fetch('/api/v1/php-extensions', {
        method: 'PUT',
        body: {
          name: extension.name,
          php_version: extension.php_version,
          enabled: selectedExtensionKeys.value.includes(getExtensionKey(extension))
        }
      })
    }

    toast.add({
      title: 'PHP extensions updated',
      description: `${changes.length} change(s) applied for PHP ${selectedPhpVersion.value}.`,
      color: 'success'
    })

    await refresh()
  } catch (error: unknown) {
    toast.add({
      title: 'Error',
      description: getErrorMessage(error),
      color: 'error'
    })
  } finally {
    applyLoading.value = false
  }
}
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
              Select available PHP extensions for the selected PHP version, then apply the changes.
            </p>
            <p v-if="selectedPhpVersion" class="text-xs text-muted mt-1">
              Showing {{ selectedExtensions.length }} extension(s) for PHP {{ selectedPhpVersion }}.
              {{ selectedCount }} selected.
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <UBadge v-if="selectedPhpVersion" color="neutral" variant="subtle">
              PHP {{ selectedPhpVersion }}
            </UBadge>
            <UBadge v-if="changedExtensions.length" color="warning" variant="subtle">
              {{ changedExtensions.length }} pending
            </UBadge>
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-default bg-muted/20 px-4 py-3">
          <div class="flex flex-wrap items-center gap-2">
            <UButton
              label="Select all"
              icon="i-lucide-check-check"
              size="sm"
              variant="outline"
              color="neutral"
              :disabled="status === 'pending' || applyLoading || !selectedExtensions.length"
              @click="selectAllAvailable"
            />
            <UButton
              label="Clear"
              icon="i-lucide-eraser"
              size="sm"
              variant="outline"
              color="neutral"
              :disabled="status === 'pending' || applyLoading || !selectedExtensionKeys.length"
              @click="clearAvailable"
            />
          </div>

          <UButton
            label="Apply"
            icon="i-lucide-save"
            color="primary"
            :loading="applyLoading"
            :disabled="status === 'pending' || !changedExtensions.length"
            @click="applyChanges"
          />
        </div>

        <UCard :ui="{ body: 'p-0 sm:p-0' }">
          <div v-if="status === 'pending'" class="flex items-center justify-center py-16 text-sm text-dimmed">
            Loading PHP extensions...
          </div>

          <div v-else-if="!selectedExtensions.length" class="flex items-center justify-center py-16 text-sm text-dimmed">
            No extensions found for this PHP version.
          </div>

          <div v-else class="grid gap-0 md:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="(column, columnIndex) in groupedExtensionColumns"
              :key="columnIndex"
              class="min-w-0 border-default p-4"
              :class="columnIndex < groupedExtensionColumns.length - 1 ? 'xl:border-r' : ''"
            >
              <div class="space-y-5">
                <section
                  v-for="group in column"
                  :key="group.letter"
                  class="space-y-2"
                >
                  <h3 class="border-b border-default pb-1 text-xs font-semibold uppercase tracking-wide text-muted">
                    {{ group.letter }}
                  </h3>

                  <div class="space-y-1">
                    <label
                      v-for="extension in group.extensions"
                      :key="getExtensionKey(extension)"
                      class="flex min-h-8 items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors"
                      :class="extension.admin_enabled ? 'cursor-pointer hover:bg-elevated/60' : 'cursor-not-allowed opacity-45'"
                    >
                      <UCheckbox
                        :model-value="selectedExtensionKeys.includes(getExtensionKey(extension))"
                        :disabled="applyLoading || !extension.admin_enabled"
                        @update:model-value="setExtensionSelected(extension, Boolean($event))"
                      />
                      <span class="min-w-0 truncate font-mono text-highlighted">
                        {{ extension.name }}
                      </span>
                    </label>
                  </div>
                </section>
              </div>
            </div>
          </div>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
