<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'

// File Manager is FileBrowser, opened in a new tab. The sidebar and
// per-domain "Files" button already do that directly (target=_blank).
// This page exists as a fallback for direct /file-manager navigation
// (back button, bookmark, deep link with ?path=). On mount it opens
// FileBrowser in a new tab and shows a manual launch button if the
// browser blocked the popup.
const route = useRoute()

function fileBrowserURL(): string {
  const p = typeof route.query.path === 'string' ? route.query.path : ''
  return `/filebrowser/files/${p}`
}

function open() {
  window.open(fileBrowserURL(), '_blank', 'noopener')
}

onMounted(() => {
  // Try to auto-open. Modern browsers allow this on direct navigation
  // (the click that brought us here counts as user gesture). If the
  // popup gets blocked the manual button is right there.
  open()
})
</script>

<template>
  <div class="flex items-center justify-center py-16">
    <div class="bg-white border border-gray-200 rounded-xl p-8 max-w-md text-center shadow-sm">
      <svg class="w-12 h-12 mx-auto mb-4 text-indigo-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <h2 class="text-base font-semibold text-gray-800 mb-1">File Manager</h2>
      <p class="text-xs text-gray-500 mb-5">
        File Manager opens in a new tab.
        If the popup was blocked, click the button below.
      </p>
      <button
        @click="open"
        class="bg-indigo-600 text-white text-sm px-4 py-2 rounded-md hover:bg-indigo-700 inline-flex items-center gap-2"
      >
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
          <polyline points="15 3 21 3 21 9"/>
          <line x1="10" y1="14" x2="21" y2="3"/>
        </svg>
        Open File Manager
      </button>
    </div>
  </div>
</template>
