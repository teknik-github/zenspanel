<script setup lang="ts">
const route = useRoute()

const frameKey = ref(0)

function decodePathSegment(segment: string) {
  try {
    return decodeURIComponent(segment)
  } catch {
    return ''
  }
}

const fileBrowserPath = computed(() => {
  const rawPath = typeof route.query.path === 'string' ? route.query.path : ''
  const cleanPath = rawPath.replace(/^\/+/, '').replace(/\/+$/, '')

  if (!cleanPath) {
    return '/filebrowser/files/'
  }

  const encodedPath = cleanPath
    .split('/')
    .map(decodePathSegment)
    .filter(segment => segment && segment !== '.' && segment !== '..')
    .map(segment => encodeURIComponent(segment))
    .join('/')

  return `/filebrowser/files/${encodedPath}/`
})

function refreshFrame() {
  frameKey.value += 1
}

function openFileBrowser() {
  window.open(fileBrowserPath.value, '_blank', 'noopener')
}
</script>

<template>
  <UDashboardPanel id="file-manager">
    <template #header>
      <UDashboardNavbar title="File Manager">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            aria-label="Refresh FileBrowser"
            @click="refreshFrame"
          />
          <UButton
            label="Open"
            icon="i-lucide-external-link"
            color="neutral"
            variant="subtle"
            @click="openFileBrowser"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex h-[calc(100vh-8rem)] min-h-[620px] flex-col overflow-hidden rounded-lg border border-default bg-default">
        <div class="flex items-center justify-between border-b border-default px-4 py-2">
          <div class="flex items-center gap-2 text-sm text-dimmed">
            <UIcon name="i-lucide-folder-open" class="size-4" />
            <code class="font-mono text-xs">
              {{ fileBrowserPath }}
            </code>
          </div>

          <UBadge color="neutral" variant="subtle">
            FileBrowser
          </UBadge>
        </div>

        <iframe
          :key="frameKey"
          :src="fileBrowserPath"
          title="FileBrowser"
          class="min-h-0 flex-1 border-0 bg-white"
          referrerpolicy="same-origin"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
