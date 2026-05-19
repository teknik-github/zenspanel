<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

// FileBrowser is mounted at /filebrowser/ by nginx, which uses
// auth_request to ask the API who the caller is and forwards the
// answer as X-Auth-User. FileBrowser auto-creates the user on first
// hit and scopes them to <homeBase>/<username>/.
//
// Deep links from other pages (the "Files" button on Domains) use
// ?path=<rel> to land the iframe on a subdir. FileBrowser's own URL
// scheme is /filebrowser/files/<rel>, so we translate.
const route = useRoute()
const ready = ref(false)

const src = computed(() => {
  const p = typeof route.query.path === 'string' ? route.query.path : ''
  return p ? `/filebrowser/files/${p}` : '/filebrowser/'
})

onMounted(() => {
  // Tiny delay so the cookie set by Login (zenspanel_token) is
  // committed before the iframe makes its first request — without
  // this, Chrome occasionally fires the iframe load before the
  // SetCookie header from /auth/login has finished writing.
  setTimeout(() => { ready.value = true }, 50)
})
</script>

<template>
  <div class="-mx-6 -mb-6 h-[calc(100vh-72px)]">
    <iframe
      v-if="ready"
      :src="src"
      class="w-full h-full border-0"
      allow="fullscreen; clipboard-read; clipboard-write"
    />
    <div v-else class="flex items-center justify-center h-full text-xs text-gray-400">
      Loading File Manager...
    </div>
  </div>
</template>
