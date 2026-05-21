<script setup lang="ts">
import { useToast, useConfirm } from './notify'

const { toasts } = useToast()
const { confirmVisible, confirmOptions, resolve } = useConfirm()
</script>

<template>
  <!-- Toast notifications -->
  <div class="fixed top-4 right-4 z-[9999] space-y-2 pointer-events-none">
    <transition-group name="toast">
      <div v-for="t in toasts" :key="t.id"
        class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-lg shadow-lg text-sm max-w-sm"
        :class="{
          'bg-green-600 text-white': t.type === 'success',
          'bg-red-600 text-white':   t.type === 'error',
          'bg-amber-500 text-white': t.type === 'warning',
          'bg-gray-800 text-white':  t.type === 'info',
        }">
        <svg v-if="t.type === 'success'" class="w-4 h-4 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        <svg v-else-if="t.type === 'error'" class="w-4 h-4 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <svg v-else-if="t.type === 'warning'" class="w-4 h-4 flex-shrink-0 mt-0.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <span>{{ t.message }}</span>
      </div>
    </transition-group>
  </div>

  <!-- Confirm dialog -->
  <div v-if="confirmVisible" class="fixed inset-0 bg-black/40 flex items-center justify-center z-[9998]">
    <div class="bg-white rounded-xl p-6 w-full max-w-sm shadow-xl mx-4">
      <h2 class="font-semibold text-gray-800 mb-2">
        {{ confirmOptions.title || 'Confirm' }}
      </h2>
      <p class="text-sm text-gray-500 mb-5">{{ confirmOptions.message }}</p>
      <div class="flex gap-2">
        <button @click="resolve(false)"
          class="flex-1 border border-gray-200 text-gray-600 rounded-md py-2 text-sm hover:bg-gray-50">
          {{ confirmOptions.cancelLabel || 'Cancel' }}
        </button>
        <button @click="resolve(true)"
          class="flex-1 rounded-md py-2 text-sm text-white"
          :class="confirmOptions.danger ? 'bg-red-600 hover:bg-red-700' : 'bg-indigo-600 hover:bg-indigo-700'">
          {{ confirmOptions.confirmLabel || 'Confirm' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast-enter-active, .toast-leave-active { transition: all 0.3s ease; }
.toast-enter-from { opacity: 0; transform: translateX(100%); }
.toast-leave-to   { opacity: 0; transform: translateX(100%); }
</style>
